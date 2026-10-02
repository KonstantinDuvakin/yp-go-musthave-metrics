// Package config описывает конфигурацию агента и сервера и её загрузку из
// флагов командной строки и переменных окружения.
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

// AgentConfig — настройки агента сбора метрик.
type AgentConfig struct {
	RateLimit      int     `json:"rate_limit"`      // максимум одновременных исходящих запросов
	PollInterval   float64 `json:"poll_interval"`   // интервал опроса метрик, секунд
	ReportInterval float64 `json:"report_interval"` // интервал отправки метрик на сервер, секунд
	Address        string  `json:"address"`         // адрес сервера в формате host:port
	Key            string  `json:"key"`             // ключ для подписи запросов (пусто — без подписи)
	CryptoKey      string  `json:"crypto_key"`      // публичный ключ для шифрования сообщений
	ConfigPath     string  `json:"config_path"`     // путь к файлу конфигурации
}

// NewConfigAgent создаёт [AgentConfig] и регистрирует флаги командной
// строки со значениями по умолчанию.
//
// Флаги ещё не разобраны: после вызова нужно выполнить flag.Parse(), а
// затем [AgentConfig.ApplyEnv], чтобы переменные окружения переопределили
// значения флагов.
func NewConfigAgent() *AgentConfig {
	ac := &AgentConfig{}

	flag.IntVar(&ac.RateLimit, "l", 5, "rate limit for outcoming requests.")
	flag.Float64Var(&ac.PollInterval, "p", 2, "defined poll interval duration.")
	flag.Float64Var(&ac.ReportInterval, "r", 10, "defined report interval duration.")
	flag.StringVar(&ac.Address, "a", "localhost:8080", "The address to listen on for HTTP requests.")
	flag.StringVar(&ac.Key, "k", "", "key for hash.")
	flag.StringVar(&ac.CryptoKey, "crypto-key", "", "public key for encrypt agent messages to server.")
	flag.StringVar(&ac.ConfigPath, "c", "", "shorthand for flag -config.")
	flag.StringVar(&ac.ConfigPath, "config", "", "the config file path.")

	return ac
}

// ApplyEnv переопределяет значения конфигурации переменными окружения
// ADDRESS, POLL_INTERVAL, REPORT_INTERVAL, KEY, RATE_LIMIT, CONFIG,
// CRYPTO_KEY. Переменные имеют приоритет над флагами командной строки.
//
// Учитываются все заданные переменные, в том числе с пустым значением:
// например, KEY="" отключает подпись, даже если ключ задан флагом.
// Невалидные числовые значения не применяются: поле сохраняет прежнее
// значение, а в лог пишется предупреждение. POLL_INTERVAL и
// REPORT_INTERVAL должны быть больше нуля; RATE_LIMIT меньше единицы
// заменяется на 1.
func (ac *AgentConfig) ApplyEnv() {
	if envAddress, ok := os.LookupEnv("ADDRESS"); ok {
		ac.Address = envAddress
	}

	if envPollInterval, ok := os.LookupEnv("POLL_INTERVAL"); ok {
		if v, err := strconv.ParseFloat(envPollInterval, 64); err == nil && v > 0 {
			ac.PollInterval = v
		} else {
			logger.Log.Warn("Invalid POLL_INTERVAL value", zap.String("POLL_INTERVAL", envPollInterval))
		}
	}

	if envReportInterval, ok := os.LookupEnv("REPORT_INTERVAL"); ok {
		if v, err := strconv.ParseFloat(envReportInterval, 64); err == nil && v > 0 {
			ac.ReportInterval = v
		} else {
			logger.Log.Warn("Invalid REPORT_INTERVAL value", zap.String("REPORT_INTERVAL", envReportInterval))
		}
	}

	if envKey, ok := os.LookupEnv("KEY"); ok {
		ac.Key = envKey
	}

	if envRateLimit, ok := os.LookupEnv("RATE_LIMIT"); ok {
		v, err := strconv.Atoi(envRateLimit)
		if err != nil {
			logger.Log.Warn("Invalid RATE_LIMIT value", zap.String("RATE_LIMIT", envRateLimit))
		} else {
			if v <= 0 {
				v = 1
			}
			ac.RateLimit = v
		}
	}

	if envCryptoKey, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		ac.CryptoKey = envCryptoKey
	}

	if envConfigPath, ok := os.LookupEnv("CONFIG"); ok {
		ac.ConfigPath = envConfigPath
	}
}

// Validate проверяет итоговые значения конфигурации после применения
// файла, флагов и переменных окружения. PollInterval и ReportInterval
// должны быть строго больше нуля.
func (ac *AgentConfig) Validate() error {
	if ac.PollInterval <= 0 {
		return fmt.Errorf("poll interval can't be equal or below zero value: %v", ac.PollInterval)
	}

	if ac.ReportInterval <= 0 {
		return fmt.Errorf("report interval can't be equal or below zero value: %v", ac.ReportInterval)
	}

	return nil
}

// UnmarshalJSON заполняет конфигурацию из JSON. poll_interval и
// report_interval задаются строками длительности ("1s", "500ms") и
// переводятся в секунды. Поля, которых нет в JSON, сохраняют текущие
// значения.
func (ac *AgentConfig) UnmarshalJSON(b []byte) error {
	type acAlias AgentConfig

	alias := &struct {
		*acAlias
		PollSec   string `json:"poll_interval"`
		ReportSec string `json:"report_interval"`
	}{
		acAlias: (*acAlias)(ac),
	}

	if err := json.Unmarshal(b, alias); err != nil {
		return err
	}

	if alias.PollSec != "" {
		seconds, err := intervalToSeconds(alias.PollSec, "poll_interval")
		if err != nil {
			return err
		}

		ac.PollInterval = seconds
	}

	if alias.ReportSec != "" {
		seconds, err := intervalToSeconds(alias.ReportSec, "report_interval")
		if err != nil {
			return err
		}

		ac.ReportInterval = seconds
	}

	return nil
}
