package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

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
	chainReader, err := chain.NewReader(cfg.RPCURL, cfg.RegistryAddress, cache)
	if err != nil {
		log.Printf("warning: chain reader: %v", err)
		chainReader, _ = chain.NewReader("", "", cache)
	}
	if cfg.RegistryAddress != "" && chainReader != nil {
		if chainCfg, err := chainReader.GetConfig(context.Background()); err != nil {
			log.Printf("warning: chain getConfig: %v", err)
		} else {
			log.Printf("chain config from registry: version=%d beacon_interval=%ds endpoint_hash=%s",
				chainCfg.Version, chainCfg.BeaconIntervalSec, chainCfg.EndpointHash)
		}
	}
	srv := api.NewServer(cfg, db, redis, chainReader)
	if cfg.DemoMode {
		if err := srv.SeedDemoData(context.Background()); err != nil {
			log.Printf("warning: demo seed: %v", err)
		} else {
			log.Printf("demo mode: IoT simulado + eventos Laureles CSV (data/)")
		}
		go func() {
			ticker := time.NewTicker(45 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				if _, err := srv.ReplayLaurelesAccess(context.Background()); err != nil {
					return
				}
			}
		}()
	}
	handler := srv.Router()
	addr := ":" + cfg.Port
	log.Printf("C2 server listening on %s (portal: http://localhost:%s/portal/)", addr, cfg.Port)
	if cfg.Insecure {
		log.Fatal(http.ListenAndServe(addr, handler))
	}
	if _, err := os.Stat(cfg.TLSCert); err != nil {
		log.Printf("TLS certs not found, starting insecure HTTP (set C2_INSECURE=true explicitly for lab)")
		log.Fatal(http.ListenAndServe(addr, handler))
	}
	log.Fatal(http.ListenAndServeTLS(addr, cfg.TLSCert, cfg.TLSKey, handler))
}
