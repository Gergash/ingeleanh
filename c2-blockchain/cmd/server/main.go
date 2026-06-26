package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ingeleanh/c2-blockchain/internal/api"
	"github.com/ingeleanh/c2-blockchain/internal/chain"
	"github.com/ingeleanh/c2-blockchain/internal/config"
	"github.com/ingeleanh/c2-blockchain/internal/store"
)

func main() {
	cfg := config.Load()
	db, err := store.OpenSQLite(cfg.DBPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	migration := filepath.Join("migrations", "001_initial.sql")
	if err := db.Migrate(migration); err != nil {
		log.Fatal(err)
	}
	redis := store.NewRedis(cfg.RedisAddr)
	if err := redis.Ping(context.Background()); err != nil {
		log.Printf("warning: redis not available: %v", err)
	}
	cache := chain.NewCache()
	if cfg.RegistryAddress != "" {
		hash := chain.ParseEndpointHash(cfg.RegistryAddress)
		endpoints := chain.EndpointsJSON("https://localhost:" + cfg.Port)
		db.UpsertChainConfig(context.Background(), hash, endpoints, 30, 1, 0)
		cache.Update(hash, 30, 1, 0)
	}
	chainReader, _ := chain.NewReader(cfg.RPCURL, cfg.RegistryAddress, cache)
	srv := api.NewServer(cfg, db, redis, chainReader)
	handler := srv.Router()
	addr := ":" + cfg.Port
	log.Printf("C2 server listening on %s", addr)
	if cfg.Insecure {
		log.Fatal(http.ListenAndServe(addr, handler))
	}
	if _, err := os.Stat(cfg.TLSCert); err != nil {
		log.Printf("TLS certs not found, starting insecure HTTP (set C2_INSECURE=true explicitly for lab)")
		log.Fatal(http.ListenAndServe(addr, handler))
	}
	log.Fatal(http.ListenAndServeTLS(addr, cfg.TLSCert, cfg.TLSKey, handler))
}
