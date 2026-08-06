package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

type AgentConfig struct {
	Address   string
	PollSec   int
	ReportSec int
	Key       string
	RateLimit int
}

func NewConfigAgent() *AgentConfig {
	ac := &AgentConfig{}

	flag.StringVar(&ac.Address, "a", "localhost:8080", "The address to listen on for HTTP requests.")
	flag.IntVar(&ac.PollSec, "p", 2, "defined poll interval duration")
	flag.IntVar(&ac.ReportSec, "r", 10, "defined report interval duration")
	flag.StringVar(&ac.Key, "k", "", "key for hash")
	flag.IntVar(&ac.RateLimit, "l", 5, "rate limit for outcoming requests")

	return ac
}

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
		if v, err := strconv.Atoi(envReportSec); err == nil {
			ac.ReportSec = v
		}
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
