package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"io"
)

// HKDFSHA256 extracts key material using HKDF-SHA256.
func HKDFSHA256(secret, salt, info []byte, length int) ([]byte, error) {
	if length <= 0 || length > 255*sha256.Size {
		return nil, io.ErrShortBuffer
	}
	if salt == nil {
		salt = make([]byte, sha256.Size)
	}
	prk := hmacSHA256(salt, secret)
	out := make([]byte, 0, length)
	var block []byte
	counter := byte(1)
	for len(out) < length {
		h := hmac.New(sha256.New, prk)
		if len(block) > 0 {
			h.Write(block)
		}
		h.Write(info)
		h.Write([]byte{counter})
		block = h.Sum(nil)
		out = append(out, block...)
		counter++
	}
	return out[:length], nil
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// BodyHash SHA256 of body for signature input.
func BodyHash(body []byte) []byte {
	h := sha256.Sum256(body)
	return h[:]
}

// SignatureMessage builds timestamp||nonce||bodyHash for beacon signing.
func SignatureMessage(timestamp int64, nonce string, body []byte) []byte {
	h := sha256.Sum256(body)
	s := fmt.Sprintf("%d%s", timestamp, nonce)
	buf := make([]byte, 0, len(s)+len(h))
	buf = append(buf, s...)
	buf = append(buf, h[:]...)
	return buf
}
