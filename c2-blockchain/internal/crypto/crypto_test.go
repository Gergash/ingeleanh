package crypto

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCRYPTO001_AESGCMRoundtrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	plain := []byte(`{"type":"beacon","status":"idle"}`)
	iv, ct, err := EncryptAESGCM(key, plain)
	require.NoError(t, err)
	out, err := DecryptAESGCM(key, iv, ct)
	require.NoError(t, err)
	require.Equal(t, plain, out)
}

func TestCRYPTO002_ECDSAValid(t *testing.T) {
	priv, err := GenerateECDSAKeypair()
	require.NoError(t, err)
	pubHex := ECDSAPublicKeyHex(priv)
	msg := []byte("test-nonce-hex")
	sig, err := SignECDSA(priv, msg)
	require.NoError(t, err)
	require.True(t, VerifyECDSA(pubHex, msg, sig))
}

func TestCRYPTO003_ECDSAInvalid(t *testing.T) {
	priv, err := GenerateECDSAKeypair()
	require.NoError(t, err)
	pubHex := ECDSAPublicKeyHex(priv)
	require.False(t, VerifyECDSA(pubHex, []byte("a"), "deadbeef"))
}

func TestCRYPTO004_HKDFDeterministic(t *testing.T) {
	secret := []byte("shared-secret")
	info := []byte("c2-session-v1")
	k1, err := HKDFSHA256(secret, nil, info, 32)
	require.NoError(t, err)
	k2, err := HKDFSHA256(secret, nil, info, 32)
	require.NoError(t, err)
	require.Equal(t, k1, k2)
	require.Len(t, k1, 32)
}

func TestCRYPTO005_GCMTamperDetect(t *testing.T) {
	key := make([]byte, 32)
	iv, ct, err := EncryptAESGCM(key, []byte("secret"))
	require.NoError(t, err)
	ct[0] ^= 0xff
	_, err = DecryptAESGCM(key, iv, ct)
	require.Error(t, err)
}

func TestCRYPTO006_IVUniqueness(t *testing.T) {
	key := make([]byte, 32)
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		iv, _, err := EncryptAESGCM(key, []byte("x"))
		require.NoError(t, err)
		s := string(iv)
		require.False(t, seen[s])
		seen[s] = true
	}
}

func TestECDHSessionKeyMatch(t *testing.T) {
	aPriv, err := GenerateECDHKeypair()
	require.NoError(t, err)
	bPriv, err := GenerateECDHKeypair()
	require.NoError(t, err)
	secretA, err := ECDHSharedSecret(aPriv, &bPriv.PublicKey)
	require.NoError(t, err)
	secretB, err := ECDHSharedSecret(bPriv, &aPriv.PublicKey)
	require.NoError(t, err)
	kA, err := DeriveSessionKey(secretA, "c2-session-v1")
	require.NoError(t, err)
	kB, err := DeriveSessionKey(secretB, "c2-session-v1")
	require.NoError(t, err)
	require.Equal(t, kA, kB)
}

func TestEnvelopeRoundtrip(t *testing.T) {
	key := make([]byte, 32)
	plain, _ := json.Marshal(map[string]string{"type": "ack"})
	var e Envelope
	require.NoError(t, e.Encrypt(key, plain))
	out, err := e.Decrypt(key)
	require.NoError(t, err)
	require.Equal(t, plain, out)
}

func TestSignNonceAndVerify(t *testing.T) {
	priv, err := GenerateECDSAKeypair()
	require.NoError(t, err)
	pub := ECDSAPublicKeyHex(priv)
	sig, err := SignNonce(priv, "nonce123")
	require.NoError(t, err)
	require.True(t, VerifyNonceSignature(pub, "nonce123", sig))
	require.False(t, VerifyNonceSignature(pub, "other", sig))
}

func TestParseECDSAPublicKeyInvalid(t *testing.T) {
	_, err := ParseECDSAPublicKey("bad")
	require.Error(t, err)
}

func TestSignatureMessage(t *testing.T) {
	msg := SignatureMessage(1719000000, "abc", []byte("body"))
	require.NotEmpty(t, msg)
}

func TestBodyHash(t *testing.T) {
	h := BodyHash([]byte("test"))
	require.Len(t, h, 32)
}
