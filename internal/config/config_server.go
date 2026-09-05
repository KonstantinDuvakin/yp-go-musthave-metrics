package config

import (
	"flag"
	"os"
	"strconv"
)

// ServerConfig — настройки сервера сбора метрик.
type ServerConfig struct {
	Address         string // адрес прослушивания HTTP, host:port
	StoreInterval   int    // интервал сохранения в файл, секунд (0 — синхронно)
	FileStoragePath string // путь к файлу хранения метрик
	Restore         bool   // восстанавливать ли метрики из файла при старте
	DB              string // DSN для подключения к БД (пусто — in-memory)
	Key             string // ключ для проверки/подписи запросов
	AuditFile       string // путь к файлу аудита (пусто — выключен)
	AuditURL        string // URL для отправки аудита (пусто — выключен)
}

// NewConfigServer создаёт [ServerConfig] и регистрирует флаги командной
// строки со значениями по умолчанию.
//
// Флаги ещё не разобраны: после вызова нужно выполнить flag.Parse(), а
// затем [ServerConfig.ApplyEnv].
func NewConfigServer() *ServerConfig {
	sc := &ServerConfig{}

	flag.StringVar(&sc.Address, "a", "localhost:8080", "The address to listen on for HTTP requests.")
	flag.IntVar(&sc.StoreInterval, "i", 3, "Interval in seconds for savings in storage file.")
	flag.StringVar(&sc.FileStoragePath, "f", "metrics_log.txt", "The file storage path.")
	flag.BoolVar(&sc.Restore, "r", true, "Flag for restoring data from storage file.")
	flag.StringVar(&sc.DB, "d", "", "Flag for database address.")
	flag.StringVar(&sc.Key, "k", "", "key for hash.")
	flag.StringVar(&sc.AuditFile, "audit-file", "", "path to audit file.")
	flag.StringVar(&sc.AuditURL, "audit-url", "", "url for audit.")

	return sc
}

// ApplyEnv переопределяет значения конфигурации переменными окружения,
// если они заданы: ADDRESS, STORE_INTERVAL, FILE_STORAGE_PATH, RESTORE,
// DATABASE_DSN, KEY, AUDIT_FILE, AUDIT_URL. Переменные имеют приоритет
// над флагами командной строки.
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

	if envAuditURL := os.Getenv("AUDIT_URL"); envAuditURL != "" {
		sc.AuditURL = envAuditURL
	}
}
