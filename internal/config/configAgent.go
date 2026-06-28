package config

import (
	"flag"
	"os"
	"strconv"
)

type AgentConfig struct {
	Address   string
	PollSec   int
	ReportSec int
}

func NewConfigAgent() *AgentConfig {
	ac := &AgentConfig{}

	flag.StringVar(&ac.Address, "a", "localhost:8080", "The address to listen on for HTTP requests.")
	flag.IntVar(&ac.PollSec, "p", 2, "defined poll interval duration")
	flag.IntVar(&ac.ReportSec, "r", 10, "defined report interval duration")

	flag.Parse()

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

	return ac
}
