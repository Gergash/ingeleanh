package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DBPath          string
	RedisAddr       string
	Port            string
	TLSCert         string
	TLSKey          string
	MasterKeyHex    string
	JWTSecret       string
	RegistryAddress string
	RPCURL          string
	Insecure        bool
	OperatorUser    string
	OperatorPass    string
}

func Load() Config {
	_ = godotenv.Load()
	insecure := os.Getenv("C2_INSECURE") == "true"
	port := os.Getenv("C2_PORT")
	if port == "" {
		port = "8443"
	}
	return Config{
		DBPath:          envOr("C2_DB_PATH", "./c2.db"),
		RedisAddr:       envOr("C2_REDIS_ADDR", "localhost:6379"),
		Port:            port,
		TLSCert:         envOr("C2_TLS_CERT", "./certs/server.crt"),
		TLSKey:          envOr("C2_TLS_KEY", "./certs/server.key"),
		MasterKeyHex:    os.Getenv("C2_MASTER_KEY"),
		JWTSecret:       envOr("C2_JWT_SECRET", "lab-jwt-secret"),
		RegistryAddress: os.Getenv("C2_REGISTRY_ADDRESS"),
		RPCURL:          os.Getenv("C2_RPC_URL"),
		Insecure:        insecure,
		OperatorUser:    envOr("C2_OPERATOR_USER", "operator"),
		OperatorPass:    envOr("C2_OPERATOR_PASS", "lab"),
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func (c Config) PortInt() int {
	p, _ := strconv.Atoi(c.Port)
	if p == 0 {
		return 8443
	}
	return p
}
