package config

import (
	"flag"
	"os"
)

type ConfigServer struct {
	Address string
}

func NewConfigServer() *ConfigServer {
	c := &ConfigServer{}

	flag.StringVar(&c.Address, "a", "localhost:8080", "The address to listen on for HTTP requests.")

	flag.Parse()

	if envAddress := os.Getenv("ADDRESS"); envAddress != "" {
		c.Address = envAddress
	}

	return c
}
