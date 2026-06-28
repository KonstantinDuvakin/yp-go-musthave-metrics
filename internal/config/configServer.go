package config

import (
	"flag"
	"os"
	"strconv"
)

type ServerConfig struct {
	Address         string
	StoreInterval   int
	FileStoragePath string
	Restore         bool
}

func NewConfigServer() *ServerConfig {
	sc := &ServerConfig{}

	flag.StringVar(&sc.Address, "a", "localhost:8080", "The address to listen on for HTTP requests.")
	flag.IntVar(&sc.StoreInterval, "i", 3, "Interval in seconds for savings in storage file.")
	flag.StringVar(&sc.FileStoragePath, "f", "metrics_log.txt", "The file storage path.")
	flag.BoolVar(&sc.Restore, "r", true, "Flag for restoring data from storage file.")

	flag.Parse()

	if envAddress := os.Getenv("ADDRESS"); envAddress != "" {
		sc.Address = envAddress
	}

	if envStoreInterval := os.Getenv("STORE_INTERVAL"); envStoreInterval != "" {
		if interval, err := strconv.Atoi(envStoreInterval); err == nil {
			sc.StoreInterval = interval
		}
	}

	if envFileStoragePath := os.Getenv("FILE_STORAGE_PATH"); envFileStoragePath != "" {
		sc.FileStoragePath = envFileStoragePath
	}

	if envRestore := os.Getenv("RESTORE"); envRestore != "" {
		if restore, err := strconv.ParseBool(envRestore); err == nil {
			sc.Restore = restore
		}
	}

	return sc
}
