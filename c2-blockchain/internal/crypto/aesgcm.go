package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

const GCMIVSize = 12
const GCMTagSize = 16

var ErrTampered = errors.New("gcm: authentication failed")

// EncryptAESGCM encrypts plaintext with AES-256-GCM. Returns iv, ciphertext+tag combined.
func EncryptAESGCM(key, plaintext []byte) (iv, ciphertext []byte, err error) {
	if len(key) != 32 {
		return nil, nil, errors.New("key must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	iv = make([]byte, GCMIVSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, nil, err
	}
	ct := gcm.Seal(nil, iv, plaintext, nil)
	return iv, ct, nil
}

// DecryptAESGCM decrypts ciphertext produced by EncryptAESGCM (includes tag).
func DecryptAESGCM(key, iv, ciphertext []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("key must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plain, err := gcm.Open(nil, iv, ciphertext, nil)
	if err != nil {
		return nil, ErrTampered
	}
	return plain, nil
}

// EncryptEnvelope builds envelope fields (iv, ct, tag separated).
func EncryptEnvelope(key, plaintext []byte) (iv, ct, tag []byte, err error) {
	ivRaw, sealed, err := EncryptAESGCM(key, plaintext)
	if err != nil {
		return nil, nil, nil, err
	}
	if len(sealed) < GCMTagSize {
		return nil, nil, nil, errors.New("invalid sealed length")
	}
	ct = sealed[:len(sealed)-GCMTagSize]
	tag = sealed[len(sealed)-GCMTagSize:]
	return ivRaw, ct, tag, nil
}

// DecryptEnvelope decrypts from separated iv, ct, tag.
func DecryptEnvelope(key, iv, ct, tag []byte) ([]byte, error) {
	sealed := append(ct, tag...)
	return DecryptAESGCM(key, iv, sealed)
}

// Envelope is the JSON wire format for encrypted payloads.
type Envelope struct {
	V   int    `json:"v"`
	Alg string `json:"alg"`
	IV  string `json:"iv"`
	Ct  string `json:"ct"`
	Tag string `json:"tag"`
}

func (e *Envelope) Encrypt(key, plaintext []byte) error {
	iv, ct, tag, err := EncryptEnvelope(key, plaintext)
	if err != nil {
		return err
	}
	e.V = 1
	e.Alg = "AES-256-GCM"
	e.IV = base64.StdEncoding.EncodeToString(iv)
	e.Ct = base64.StdEncoding.EncodeToString(ct)
	e.Tag = base64.StdEncoding.EncodeToString(tag)
	return nil
}

func (e *Envelope) Decrypt(key []byte) ([]byte, error) {
	iv, err := base64.StdEncoding.DecodeString(e.IV)
	if err != nil {
		return nil, err
	}
	ct, err := base64.StdEncoding.DecodeString(e.Ct)
	if err != nil {
		return nil, err
	}
	tag, err := base64.StdEncoding.DecodeString(e.Tag)
	if err != nil {
		return nil, err
	}
	return DecryptEnvelope(key, iv, ct, tag)
}
