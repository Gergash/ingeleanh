package store

import (
	"context"
	"time"

	"github.com/ingeleanh/c2-blockchain/internal/handshake"
)

type RedisNonceStore struct {
	r   *RedisStore
	ctx context.Context
}

func NewRedisNonceStore(r *RedisStore) handshake.NonceStore {
	return &RedisNonceStore{r: r, ctx: context.Background()}
}

func (n *RedisNonceStore) Set(nonce string, ttl time.Duration) error {
	return n.r.SetNonce(n.ctx, nonce, ttl)
}

func (n *RedisNonceStore) Exists(nonce string) (bool, error) {
	return n.r.HasNonce(n.ctx, nonce)
}

func (n *RedisNonceStore) Delete(nonce string) error {
	return n.r.DelNonce(n.ctx, nonce)
}
