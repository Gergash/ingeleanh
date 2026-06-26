package handshake

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/ingeleanh/c2-blockchain/internal/crypto"
)

const NonceTTL = 30 * time.Second
const NonceHexLen = 64

var (
	ErrNonceExpired    = errors.New("NONCE_EXPIRED")
	ErrNonceReused     = errors.New("NONCE_REUSED")
	ErrSignatureInvalid = errors.New("SIGNATURE_INVALID")
	ErrInvalidStep     = errors.New("INVALID_STEP")
)

type NonceStore interface {
	Set(nonce string, ttl time.Duration) error
	Exists(nonce string) (bool, error)
	Delete(nonce string) error
}

type MemoryNonceStore struct {
	mu    sync.Mutex
	items map[string]time.Time
}

func NewMemoryNonceStore() *MemoryNonceStore {
	return &MemoryNonceStore{items: make(map[string]time.Time)}
}

func (m *MemoryNonceStore) Set(nonce string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[nonce] = time.Now().Add(ttl)
	return nil
}

func (m *MemoryNonceStore) Exists(nonce string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	exp, ok := m.items[nonce]
	if !ok {
		return false, nil
	}
	if time.Now().After(exp) {
		delete(m.items, nonce)
		return false, nil
	}
	return true, nil
}

func (m *MemoryNonceStore) Delete(nonce string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, nonce)
	return nil
}

type Service struct {
	nonces NonceStore
}

func NewService(nonces NonceStore) *Service {
	return &Service{nonces: nonces}
}

func GenerateNonce() (string, error) {
	b := make([]byte, 32)
	if _, err := crypto.ReadRand(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

type Challenge struct {
	Nonce          string
	ServerECDHPub  string
	ExpiresAt      int64
	ServerECDHPriv *ecdsa.PrivateKey
}

func (s *Service) CreateChallenge() (*Challenge, error) {
	nonce, err := GenerateNonce()
	if err != nil {
		return nil, err
	}
	if err := s.nonces.Set(nonce, NonceTTL); err != nil {
		return nil, err
	}
	priv, err := crypto.GenerateECDHKeypair()
	if err != nil {
		return nil, err
	}
	pubBytes := crypto.MarshalECDHPublicKeySPKI(&priv.PublicKey)
	return &Challenge{
		Nonce:          nonce,
		ServerECDHPub:  base64.StdEncoding.EncodeToString(pubBytes),
		ExpiresAt:      time.Now().Add(NonceTTL).Unix(),
		ServerECDHPriv: priv,
	}, nil
}

type ChallengeResponse struct {
	Nonce         string
	AgentECDSAPub string
	AgentECDHPub  string
	Signature     string
	Hostname      string
	OS            string
	Timestamp     int64
}

type SessionResult struct {
	SessionKey []byte
}

func (s *Service) CompleteChallenge(ch *Challenge, resp *ChallengeResponse) (*SessionResult, error) {
	if resp.Timestamp > 0 {
		age := time.Now().Unix() - resp.Timestamp
		if age > int64(NonceTTL.Seconds()) || age < -5 {
			return nil, ErrNonceExpired
		}
	}
	exists, err := s.nonces.Exists(resp.Nonce)
	if err != nil {
		return nil, err
	}
	if !exists || resp.Nonce != ch.Nonce {
		return nil, ErrNonceExpired
	}
	if !crypto.VerifyNonceSignature(resp.AgentECDSAPub, resp.Nonce, resp.Signature) {
		return nil, ErrSignatureInvalid
	}
	if _, err := crypto.ParseECDSAPublicKey(resp.AgentECDSAPub); err != nil {
		return nil, ErrSignatureInvalid
	}
	agentPubBytes, err := base64.StdEncoding.DecodeString(resp.AgentECDHPub)
	if err != nil {
		return nil, ErrSignatureInvalid
	}
	agentPub, err := crypto.UnmarshalECDHPublicKeySPKI(elliptic.P256(), agentPubBytes)
	if err != nil {
		return nil, ErrSignatureInvalid
	}
	shared, err := crypto.ECDHSharedSecret(ch.ServerECDHPriv, agentPub)
	if err != nil {
		return nil, err
	}
	key, err := crypto.DeriveSessionKey(shared, "c2-session-v1")
	if err != nil {
		return nil, err
	}
	if err := s.nonces.Delete(resp.Nonce); err != nil {
		return nil, err
	}
	return &SessionResult{SessionKey: key}, nil
}

func (s *Service) CheckReplay(nonce string) error {
	exists, err := s.nonces.Exists(nonce)
	if err != nil {
		return err
	}
	if exists {
		return ErrNonceReused
	}
	return s.nonces.Set(nonce, NonceTTL)
}
