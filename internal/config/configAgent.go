package config

import (
	"flag"
	"os"
)

type ConfigAgent struct {
	Address   string
	PollSec   int
	ReportSec int
}

func NewConfigAgent() *ConfigAgent {
	c := &ConfigAgent{}

	flag.StringVar(&c.Address, "a", "localhost:8080", "The address to listen on for HTTP requests.")
	flag.IntVar(&c.PollSec, "p", 2, "defined poll interval duration")
	flag.IntVar(&c.ReportSec, "r", 10, "defined report interval duration")

	flag.Parse()

	if envAddress := os.Getenv("ADDRESS"); envAddress != "" {
		c.Address = envAddress
	}

	if envPollSec := os.Getenv("POLL_INTERVAL"); envPollSec != "" {
		c.Address = envPollSec
	}

	if envReportSec := os.Getenv("REPORT_INTERVAL"); envReportSec != "" {
		c.Address = envReportSec
	}

	return c
}
