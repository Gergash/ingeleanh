package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
)

// GenerateECDSAKeypair generates secp256k1 keypair.
func GenerateECDSAKeypair() (*ecdsa.PrivateKey, error) {
	return crypto.GenerateKey()
}

// ECDSAPublicKeyHex returns compressed public key hex.
func ECDSAPublicKeyHex(priv *ecdsa.PrivateKey) string {
	return hex.EncodeToString(crypto.CompressPubkey(&priv.PublicKey))
}

// SignECDSA signs message with private key, returns R||S hex (64 bytes).
func SignECDSA(priv *ecdsa.PrivateKey, message []byte) (string, error) {
	hash := sha256.Sum256(message)
	sig, err := crypto.Sign(hash[:], priv)
	if err != nil {
		return "", err
	}
	if len(sig) == 65 {
		sig = sig[:64]
	}
	return hex.EncodeToString(sig), nil
}

// VerifyECDSA verifies signature against message and compressed pubkey hex.
func VerifyECDSA(pubKeyHex string, message []byte, signatureHex string) bool {
	pubBytes, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return false
	}
	pubKey, err := crypto.DecompressPubkey(pubBytes)
	if err != nil {
		return false
	}
	sig, err := hex.DecodeString(signatureHex)
	if err != nil || len(sig) != 64 {
		return false
	}
	hash := sha256.Sum256(message)
	return crypto.VerifySignature(crypto.CompressPubkey(pubKey), hash[:], sig)
}

// ParseECDSAPublicKey parses compressed secp256k1 pubkey hex.
func ParseECDSAPublicKey(pubKeyHex string) (*ecdsa.PublicKey, error) {
	pubBytes, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return nil, err
	}
	pub, err := crypto.DecompressPubkey(pubBytes)
	if err != nil {
		return nil, err
	}
	return pub, nil
}

// ECDH P-256 session key derivation
func GenerateECDHKeypair() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

// ECDHSharedSecret computes ECDH shared secret (x coordinate of shared point).
func ECDHSharedSecret(priv *ecdsa.PrivateKey, peerPub *ecdsa.PublicKey) ([]byte, error) {
	if peerPub == nil || priv == nil {
		return nil, errors.New("invalid keys")
	}
	x, _ := peerPub.Curve.ScalarMult(peerPub.X, peerPub.Y, priv.D.Bytes())
	return x.Bytes(), nil
}

// DeriveSessionKey HKDF-SHA256 to 32-byte AES key.
func DeriveSessionKey(sharedSecret []byte, info string) ([]byte, error) {
	return HKDFSHA256(sharedSecret, nil, []byte(info), 32)
}

// MarshalECDHPublicKeySPKI returns base64-ready SPKI bytes for P-256 public key.
func MarshalECDHPublicKeySPKI(pub *ecdsa.PublicKey) []byte {
	return elliptic.Marshal(pub.Curve, pub.X, pub.Y)
}

// UnmarshalECDHPublicKeySPKI parses uncompressed point from SPKI-style bytes.
func UnmarshalECDHPublicKeySPKI(curve elliptic.Curve, data []byte) (*ecdsa.PublicKey, error) {
	x, y := elliptic.Unmarshal(curve, data)
	if x == nil {
		return nil, errors.New("invalid ecdh pubkey")
	}
	return &ecdsa.PublicKey{Curve: curve, X: x, Y: y}, nil
}

// SignNonce signs handshake nonce with agent key.
func SignNonce(priv *ecdsa.PrivateKey, nonceHex string) (string, error) {
	return SignECDSA(priv, []byte(nonceHex))
}

// VerifyNonceSignature verifies agent signed the nonce.
func VerifyNonceSignature(pubKeyHex, nonceHex, sigHex string) bool {
	return VerifyECDSA(pubKeyHex, []byte(nonceHex), sigHex)
}

// Low-level secp256k1 sign for tests
func SignHash(priv *ecdsa.PrivateKey, hash []byte) ([]byte, error) {
	return crypto.Sign(hash, priv)
}

func VerifyHash(pub *ecdsa.PublicKey, hash, sig []byte) bool {
	return crypto.VerifySignature(crypto.CompressPubkey(pub), hash, sig)
}

func S256() elliptic.Curve {
	return secp256k1.S256()
}
