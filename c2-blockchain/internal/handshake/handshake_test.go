package handshake

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/ingeleanh/c2-blockchain/internal/crypto"
	"github.com/stretchr/testify/require"
)

func TestHS001_NonceUnique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		n, err := GenerateNonce()
		require.NoError(t, err)
		require.Len(t, n, 64)
		require.False(t, seen[n])
		seen[n] = true
	}
}

func TestHS002_RejectExpiredTimestamp(t *testing.T) {
	svc := NewService(NewMemoryNonceStore())
	ch, err := svc.CreateChallenge()
	require.NoError(t, err)
	priv, err := crypto.GenerateECDSAKeypair()
	require.NoError(t, err)
	agentECDH, err := crypto.GenerateECDHKeypair()
	require.NoError(t, err)
	sig, err := crypto.SignNonce(priv, ch.Nonce)
	require.NoError(t, err)
	pubBytes := crypto.MarshalECDHPublicKeySPKI(&agentECDH.PublicKey)
	resp := &ChallengeResponse{
		Nonce:         ch.Nonce,
		AgentECDSAPub: crypto.ECDSAPublicKeyHex(priv),
		AgentECDHPub:  base64.StdEncoding.EncodeToString(pubBytes),
		Signature:     sig,
		Hostname:      "lab",
		OS:            "linux-amd64",
		Timestamp:     time.Now().Add(-31 * time.Second).Unix(),
	}
	_, err = svc.CompleteChallenge(ch, resp)
	require.ErrorIs(t, err, ErrNonceExpired)
}

func TestHS003_RejectReplayNonce(t *testing.T) {
	svc := NewService(NewMemoryNonceStore())
	nonce, err := GenerateNonce()
	require.NoError(t, err)
	require.NoError(t, svc.CheckReplay(nonce))
	err = svc.CheckReplay(nonce)
	require.ErrorIs(t, err, ErrNonceReused)
}

func TestHS004_ECDHSessionKeyMatch(t *testing.T) {
	svc := NewService(NewMemoryNonceStore())
	ch, err := svc.CreateChallenge()
	require.NoError(t, err)
	agentECDSA, err := crypto.GenerateECDSAKeypair()
	require.NoError(t, err)
	agentECDH, err := crypto.GenerateECDHKeypair()
	require.NoError(t, err)
	sig, err := crypto.SignNonce(agentECDSA, ch.Nonce)
	require.NoError(t, err)
	pubBytes := crypto.MarshalECDHPublicKeySPKI(&agentECDH.PublicKey)
	resp := &ChallengeResponse{
		Nonce:         ch.Nonce,
		AgentECDSAPub: crypto.ECDSAPublicKeyHex(agentECDSA),
		AgentECDHPub:  base64.StdEncoding.EncodeToString(pubBytes),
		Signature:     sig,
		Hostname:      "lab-vm",
		OS:            "linux-amd64",
		Timestamp:     time.Now().Unix(),
	}
	result, err := svc.CompleteChallenge(ch, resp)
	require.NoError(t, err)

	agentPub, err := crypto.UnmarshalECDHPublicKeySPKI(ch.ServerECDHPriv.Curve, mustDecode(ch.ServerECDHPub))
	require.NoError(t, err)
	shared, err := crypto.ECDHSharedSecret(agentECDH, agentPub)
	require.NoError(t, err)
	agentKey, err := crypto.DeriveSessionKey(shared, "c2-session-v1")
	require.NoError(t, err)
	require.Equal(t, result.SessionKey, agentKey)
}

func TestHS005_RejectMalformedPubkey(t *testing.T) {
	svc := NewService(NewMemoryNonceStore())
	ch, err := svc.CreateChallenge()
	require.NoError(t, err)
	resp := &ChallengeResponse{
		Nonce:         ch.Nonce,
		AgentECDSAPub: "not-hex",
		AgentECDHPub:  base64.StdEncoding.EncodeToString([]byte("bad")),
		Signature:     "aa",
		Timestamp:     time.Now().Unix(),
	}
	_, err = svc.CompleteChallenge(ch, resp)
	require.ErrorIs(t, err, ErrSignatureInvalid)
}

func mustDecode(s string) []byte {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}
