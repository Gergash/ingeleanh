package chain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Config struct {
	EndpointHash      string
	BeaconIntervalSec uint32
	Version           uint64
}

type Cache struct {
	mu      sync.RWMutex
	current Config
	block   uint64
}

func NewCache() *Cache {
	return &Cache{}
}

func (c *Cache) Update(hash string, beaconSec uint32, version uint64, block uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.current = Config{EndpointHash: hash, BeaconIntervalSec: beaconSec, Version: version}
	c.block = block
}

func (c *Cache) Get() (Config, uint64) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.current, c.block
}

// IndexConfigUpdated processes ConfigUpdated event fields (CHAIN-001).
func (c *Cache) IndexConfigUpdated(endpointHash string, beaconSec uint32, version uint64, block uint64) {
	c.Update(endpointHash, beaconSec, version, block)
}

type Reader struct {
	client   *ethclient.Client
	registry common.Address
	cache    *Cache
}

func NewReader(rpcURL, registryAddr string, cache *Cache) (*Reader, error) {
	if rpcURL == "" || registryAddr == "" {
		return &Reader{cache: cache}, nil
	}
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, err
	}
	return &Reader{
		client:   client,
		registry: common.HexToAddress(registryAddr),
		cache:    cache,
	}, nil
}

func getConfigSelectorHex() string {
	return "c3f909d4"
}

// GetConfig eth_call (CHAIN-002).
func (r *Reader) GetConfig(ctx context.Context) (Config, error) {
	if r.client == nil {
		cfg, _ := r.cache.Get()
		return cfg, nil
	}
	data := common.Hex2Bytes(getConfigSelectorHex())
	msg := ethereum.CallMsg{To: &r.registry, Data: data}
	out, err := r.client.CallContract(ctx, msg, nil)
	if err != nil {
		cfg, _ := r.cache.Get()
		return cfg, err
	}
	if len(out) < 96 {
		cfg, _ := r.cache.Get()
		return cfg, nil
	}
	hash := hex.EncodeToString(out[0:32])
	beacon := uint32(new(big.Int).SetBytes(out[32:64]).Uint64())
	version := new(big.Int).SetBytes(out[64:96]).Uint64()
	cfg := Config{EndpointHash: "0x" + hash, BeaconIntervalSec: beacon, Version: version}
	r.cache.Update(cfg.EndpointHash, cfg.BeaconIntervalSec, cfg.Version, 0)
	return cfg, nil
}

func (r *Reader) IndexConfigUpdated(endpointHash string, beaconSec uint32, version uint64, block uint64) {
	if r.cache != nil {
		r.cache.IndexConfigUpdated(endpointHash, beaconSec, version, block)
	}
}

func (r *Reader) Status() map[string]interface{} {
	cfg, block := r.cache.Get()
	endpoints := []string{}
	if cfg.EndpointHash != "" {
		endpoints = []string{cfg.EndpointHash}
	}
	return map[string]interface{}{
		"contract_address":   r.registry.Hex(),
		"network":            "polygon-amoy",
		"chain_id":           80002,
		"config_version":     cfg.Version,
		"beacon_interval_sec": cfg.BeaconIntervalSec,
		"endpoint_hash":      cfg.EndpointHash,
		"last_indexed_block": block,
		"indexer_lag_blocks": 0,
		"endpoints_json":     endpoints,
	}
}

// ParseEndpointHash matches ethers.sha256(utf8(url)) used in deploy.js.
func ParseEndpointHash(url string) string {
	sum := sha256.Sum256([]byte(url))
	return "0x" + hex.EncodeToString(sum[:])
}

func EndpointsJSON(primaryURL string) string {
	b, _ := json.Marshal([]string{primaryURL})
	return string(b)
}

func NormalizeHash(h string) string {
	if strings.HasPrefix(h, "0x") {
		return h
	}
	return "0x" + h
}
