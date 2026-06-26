package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	rdb *redis.Client
}

func NewRedis(addr string) *RedisStore {
	return &RedisStore{rdb: redis.NewClient(&redis.Options{Addr: addr})}
}

func (r *RedisStore) Client() *redis.Client { return r.rdb }

func (r *RedisStore) Ping(ctx context.Context) error {
	return r.rdb.Ping(ctx).Err()
}

func (r *RedisStore) SetNonce(ctx context.Context, nonce string, ttl time.Duration) error {
	return r.rdb.Set(ctx, "handshake:nonce:"+nonce, "1", ttl).Err()
}

func (r *RedisStore) HasNonce(ctx context.Context, nonce string) (bool, error) {
	n, err := r.rdb.Exists(ctx, "handshake:nonce:"+nonce).Result()
	return n > 0, err
}

func (r *RedisStore) DelNonce(ctx context.Context, nonce string) error {
	return r.rdb.Del(ctx, "handshake:nonce:"+nonce).Err()
}

func (r *RedisStore) PushTask(ctx context.Context, agentID string, task map[string]interface{}) error {
	b, _ := json.Marshal(task)
	return r.rdb.RPush(ctx, "beacon:pending:"+agentID, b).Err()
}

func (r *RedisStore) PopTask(ctx context.Context, agentID string) (map[string]interface{}, error) {
	res, err := r.rdb.LPop(ctx, "beacon:pending:"+agentID).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	json.Unmarshal([]byte(res), &m)
	return m, nil
}

func (r *RedisStore) IncrRate(ctx context.Context, key string, window time.Duration) (int64, error) {
	pipe := r.rdb.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}
	return incr.Val(), nil
}

func (r *RedisStore) SetLockState(ctx context.Context, deviceID, state string) error {
	return r.rdb.Set(ctx, "iot:lock:"+deviceID, state, 0).Err()
}

func (r *RedisStore) GetLockState(ctx context.Context, deviceID string) (string, error) {
	return r.rdb.Get(ctx, "iot:lock:"+deviceID).Result()
}
