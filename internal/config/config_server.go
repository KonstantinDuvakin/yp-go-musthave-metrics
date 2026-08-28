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
	DB              string
	Key             string
	AuditFile       string
	AuditUrl        string
}

func NewConfigServer() *ServerConfig {
	sc := &ServerConfig{}

	flag.StringVar(&sc.Address, "a", "localhost:8080", "The address to listen on for HTTP requests.")
	flag.IntVar(&sc.StoreInterval, "i", 3, "Interval in seconds for savings in storage file.")
	flag.StringVar(&sc.FileStoragePath, "f", "metrics_log.txt", "The file storage path.")
	flag.BoolVar(&sc.Restore, "r", true, "Flag for restoring data from storage file.")
	flag.StringVar(&sc.DB, "d", "", "Flag for database address.")
	flag.StringVar(&sc.Key, "k", "", "key for hash.")
	flag.StringVar(&sc.AuditFile, "audit-file", "", "path to audit file.")
	flag.StringVar(&sc.AuditUrl, "audit-url", "", "url for audit.")

	return sc
}

func (sc *ServerConfig) ApplyEnv() {
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

	if envDatabaseAddress := os.Getenv("DATABASE_DSN"); envDatabaseAddress != "" {
		sc.DB = envDatabaseAddress
	}

	if envKey := os.Getenv("KEY"); envKey != "" {
		sc.Key = envKey
	}

	if envAuditFile := os.Getenv("AUDIT_FILE"); envAuditFile != "" {
		sc.AuditFile = envAuditFile
	}

	if envAuditUrl := os.Getenv("AUDIT_URL"); envAuditUrl != "" {
		sc.AuditUrl = envAuditUrl
	}
}
