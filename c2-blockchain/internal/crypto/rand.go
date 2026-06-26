package crypto

import (
	"crypto/rand"
	"io"
)

// ReadRand fills b with random bytes.
func ReadRand(b []byte) (int, error) {
	return io.ReadFull(rand.Reader, b)
}
