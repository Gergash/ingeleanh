package api

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/ingeleanh/c2-blockchain/internal/chain"
	"github.com/ingeleanh/c2-blockchain/internal/config"
	"github.com/ingeleanh/c2-blockchain/internal/store"
	"github.com/stretchr/testify/require"
)

func TestSeedDemoData(t *testing.T) {
	mr := miniredis.RunT(t)
	db, err := store.OpenSQLite(t.TempDir() + "/demo.db")
	require.NoError(t, err)
	defer db.Close()
	require.NoError(t, db.Migrate("../../migrations/001_initial.sql"))
	redis := store.NewRedis(mr.Addr())
	cache := chain.NewCache()
	reader, _ := chain.NewReader("", "", cache)
	cfg := config.Config{DemoMode: true, JWTSecret: "x", OperatorUser: "op", OperatorPass: "p"}
	srv := NewServer(cfg, db, redis, reader)
	require.NoError(t, srv.SeedDemoData(context.Background()))
	devs, err := db.ListIoTDevices(context.Background())
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(devs), 4)
}
