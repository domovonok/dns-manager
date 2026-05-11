package config

import (
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Server struct {
	GRPCAddr                string        `env:"GRPC_ADDR" envDefault:":8080"`
	ResolveConfPath         string        `env:"RESOLV_CONF_PATH" envDefault:"/etc/resolv.conf"`
	GracefulShutdownTimeout time.Duration `env:"GRACEFUL_SHUTDOWN_TIMEOUT" envDefault:"10s"`
}

type Client struct {
	DefaultAddr    string        `env:"DEFAULT_ADDR" envDefault:"127.0.0.1:8080"`
	RequestTimeout time.Duration `env:"REQUEST_TIMEOUT" envDefault:"5s"`
}

func NewServer() (*Server, error) {
	return parseConfig[Server]()
}

func NewClient() (*Client, error) {
	return parseConfig[Client]()
}

func parseConfig[T any]() (*T, error) {
	_ = godotenv.Load()
	var cfg T
	err := env.Parse(&cfg)
	return &cfg, err
}
