package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger"
	"go.uber.org/zap"
)

// ServerConfig — настройки сервера сбора метрик.
type ServerConfig struct {
	Address       string  `json:"address"`        // адрес прослушивания HTTP, host:port
	StoreInterval float64 `json:"store_interval"` // интервал сохранения в файл, секунд (0 — синхронно)
	StoreFile     string  `json:"store_file"`     // путь к файлу хранения метрик
	ConfigPath    string  `json:"config_path"`    // путь к файлу конфигурации
	Restore       bool    `json:"restore"`        // восстанавливать ли метрики из файла при старте
	DB            string  `json:"database_dsn"`   // DSN для подключения к БД (пусто — in-memory)
	Key           string  `json:"key"`            // ключ для проверки/подписи запросов
	AuditFile     string  `json:"audit_file"`     // путь к файлу аудита (пусто — выключен)
	AuditURL      string  `json:"audit_url"`      // URL для отправки аудита (пусто — выключен)
	CryptoKey     string  `json:"crypto_key"`     // приватный ключ для расшифровки сообщений от клиента
}

// NewConfigServer создаёт [ServerConfig] и регистрирует флаги командной
// строки со значениями по умолчанию.
//
// Флаги ещё не разобраны: после вызова нужно выполнить flag.Parse(), а
// затем [ServerConfig.ApplyEnv].
func NewConfigServer() *ServerConfig {
	sc := &ServerConfig{}

	flag.BoolVar(&sc.Restore, "r", true, "flag for restoring data from storage file.")
	flag.Float64Var(&sc.StoreInterval, "i", 3, "interval in seconds for savings in storage file.")
	flag.StringVar(&sc.Address, "a", "localhost:8080", "the address to listen on for HTTP requests.")
	flag.StringVar(&sc.ConfigPath, "c", "", "shorthand for flag -config.")
	flag.StringVar(&sc.ConfigPath, "config", "", "the config file path.")
	flag.StringVar(&sc.StoreFile, "f", "metrics_log.txt", "the file storage path.")
	flag.StringVar(&sc.DB, "d", "", "flag for database address.")
	flag.StringVar(&sc.Key, "k", "", "key for hash.")
	flag.StringVar(&sc.CryptoKey, "crypto-key", "", "private key for decrypt messages from agent.")
	flag.StringVar(&sc.AuditFile, "audit-file", "", "path to audit file.")
	flag.StringVar(&sc.AuditURL, "audit-url", "", "url for audit.")

	return sc
}

// ApplyEnv переопределяет значения конфигурации переменными окружения
// ADDRESS, STORE_INTERVAL, FILE_STORAGE_PATH, RESTORE, DATABASE_DSN, KEY,
// AUDIT_FILE, AUDIT_URL, CONFIG, CRYPTO_KEY. Переменные имеют приоритет
// над флагами командной строки.
//
// Учитываются все заданные переменные, в том числе с пустым значением:
// например, DATABASE_DSN="" переключает сервер на хранение в памяти, даже
// если DSN задан флагом. Невалидные значения STORE_INTERVAL (не число или
// меньше нуля) и RESTORE (не bool) не применяются: поле сохраняет прежнее
// значение.
func (sc *ServerConfig) ApplyEnv() {
	if envAddress, ok := os.LookupEnv("ADDRESS"); ok {
		sc.Address = envAddress
	}

	if envStoreInterval, ok := os.LookupEnv("STORE_INTERVAL"); ok {
		if interval, err := strconv.ParseFloat(envStoreInterval, 64); err == nil && interval >= 0 {
			sc.StoreInterval = interval
		} else {
			logger.Log.Warn("Invalid STORE_INTERVAL value", zap.String("STORE_INTERVAL", envStoreInterval))
		}
	}

	if envFileStoragePath, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		sc.StoreFile = envFileStoragePath
	}

	if envConfigPath, ok := os.LookupEnv("CONFIG"); ok {
		sc.ConfigPath = envConfigPath
	}

	if envRestore, ok := os.LookupEnv("RESTORE"); ok {
		if restore, err := strconv.ParseBool(envRestore); err == nil {
			sc.Restore = restore
		}
	}

	if envDatabaseAddress, ok := os.LookupEnv("DATABASE_DSN"); ok {
		sc.DB = envDatabaseAddress
	}

	if envKey, ok := os.LookupEnv("KEY"); ok {
		sc.Key = envKey
	}

	if envAuditFile, ok := os.LookupEnv("AUDIT_FILE"); ok {
		sc.AuditFile = envAuditFile
	}

	if envAuditURL, ok := os.LookupEnv("AUDIT_URL"); ok {
		sc.AuditURL = envAuditURL
	}

	if envPrivateKey, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		sc.CryptoKey = envPrivateKey
	}
}

// Validate проверяет итоговые значения конфигурации после применения
// файла, флагов и переменных окружения. StoreInterval не может быть
// отрицательным: 0 означает синхронную запись.
func (sc *ServerConfig) Validate() error {
	if sc.StoreInterval < 0 {
		return fmt.Errorf("store interval can't be below zero value: %v", sc.StoreInterval)
	}

	return nil
}

// UnmarshalJSON заполняет конфигурацию из JSON. store_interval задаётся
// строкой длительности ("1s", "500ms") и переводится в секунды.
// Поля, которых нет в JSON, сохраняют текущие значения.
func (sc *ServerConfig) UnmarshalJSON(b []byte) error {
	type scAlias ServerConfig

	alias := &struct {
		*scAlias
		StoreInterval string `json:"store_interval"`
	}{
		scAlias: (*scAlias)(sc),
	}

	if err := json.Unmarshal(b, alias); err != nil {
		return err
	}

	if alias.StoreInterval != "" {
		seconds, err := intervalToSeconds(alias.StoreInterval, "store_interval")
		if err != nil {
			return err
		}

		sc.StoreInterval = seconds
	}

	return nil
}
