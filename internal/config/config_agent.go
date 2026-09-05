// Package config описывает конфигурацию агента и сервера и её загрузку из
// флагов командной строки и переменных окружения.
package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

// AgentConfig — настройки агента сбора метрик.
type AgentConfig struct {
	Address   string // адрес сервера в формате host:port
	PollSec   int    // интервал опроса метрик, секунд
	ReportSec int    // интервал отправки метрик на сервер, секунд
	Key       string // ключ для подписи запросов (пусто — без подписи)
	RateLimit int    // максимум одновременных исходящих запросов
}

// NewConfigAgent создаёт [AgentConfig] и регистрирует флаги командной
// строки со значениями по умолчанию.
//
// Флаги ещё не разобраны: после вызова нужно выполнить flag.Parse(), а
// затем [AgentConfig.ApplyEnv], чтобы переменные окружения переопределили
// значения флагов.
func NewConfigAgent() *AgentConfig {
	ac := &AgentConfig{}

	flag.StringVar(&ac.Address, "a", "localhost:8080", "The address to listen on for HTTP requests.")
	flag.IntVar(&ac.PollSec, "p", 2, "defined poll interval duration")
	flag.IntVar(&ac.ReportSec, "r", 10, "defined report interval duration")
	flag.StringVar(&ac.Key, "k", "", "key for hash")
	flag.IntVar(&ac.RateLimit, "l", 5, "rate limit for outcoming requests")

	return ac
}

// ApplyEnv переопределяет значения конфигурации переменными окружения,
// если они заданы: ADDRESS, POLL_INTERVAL, REPORT_INTERVAL, KEY,
// RATE_LIMIT. Переменные имеют приоритет над флагами командной строки.
func (ac *AgentConfig) ApplyEnv() {
	if envAddress := os.Getenv("ADDRESS"); envAddress != "" {
		ac.Address = envAddress
	}

	if envPollSec := os.Getenv("POLL_INTERVAL"); envPollSec != "" {
		if v, err := strconv.Atoi(envPollSec); err == nil {
			ac.PollSec = v
		}
	}

	if envReportSec := os.Getenv("REPORT_INTERVAL"); envReportSec != "" {
		v, err := strconv.Atoi(envReportSec)
		if err != nil {
			fmt.Printf("Invalid REPORT_INTERVAL value: %s\n", envReportSec)
		}
		ac.ReportSec = v
	}

	if envKey := os.Getenv("KEY"); envKey != "" {
		ac.Key = envKey
	}

	if envRateLimit := os.Getenv("RATE_LIMIT"); envRateLimit != "" {
		v, err := strconv.Atoi(envRateLimit)
		if err != nil {
			fmt.Printf("Invalid RATE_LIMIT value: %s\n", envRateLimit)
		}

		if v <= 0 {
			v = 1
		}
		ac.RateLimit = v
	}
}
