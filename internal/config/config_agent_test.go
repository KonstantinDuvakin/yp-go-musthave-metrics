package config

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentConfig_Validate(t *testing.T) {
	tests := []struct {
		name           string
		pollInterval   float64
		reportInterval float64
		wantErr        bool
	}{
		{name: "valid", pollInterval: 2, reportInterval: 10},
		{name: "fractional", pollInterval: 0.5, reportInterval: 0.5},
		{name: "zero poll", pollInterval: 0, reportInterval: 10, wantErr: true},
		{name: "negative poll", pollInterval: -1, reportInterval: 10, wantErr: true},
		{name: "zero report", pollInterval: 2, reportInterval: 0, wantErr: true},
		{name: "negative report", pollInterval: 2, reportInterval: -1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ac := &AgentConfig{PollInterval: tt.pollInterval, ReportInterval: tt.reportInterval}

			err := ac.Validate()

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestAgentConfig_UnmarshalJSON(t *testing.T) {
	// Значения по умолчанию, как после регистрации флагов.
	defaults := func() *AgentConfig {
		return &AgentConfig{
			Address:        "localhost:8080",
			PollInterval:   2,
			ReportInterval: 10,
			RateLimit:      5,
		}
	}

	t.Run("full file", func(t *testing.T) {
		ac := defaults()
		data := `{
			"address": "localhost:9090",
			"poll_interval": "500ms",
			"report_interval": "1m",
			"rate_limit": 3,
			"crypto_key": "/path/to/key.pem"
		}`

		require.NoError(t, json.Unmarshal([]byte(data), ac))

		assert.Equal(t, "localhost:9090", ac.Address)
		assert.Equal(t, 0.5, ac.PollInterval)
		assert.Equal(t, float64(60), ac.ReportInterval)
		assert.Equal(t, 3, ac.RateLimit)
		assert.Equal(t, "/path/to/key.pem", ac.CryptoKey)
	})

	t.Run("missing fields keep defaults", func(t *testing.T) {
		ac := defaults()

		require.NoError(t, json.Unmarshal([]byte(`{"address": "localhost:9090"}`), ac))

		assert.Equal(t, "localhost:9090", ac.Address)
		assert.Equal(t, float64(2), ac.PollInterval)
		assert.Equal(t, float64(10), ac.ReportInterval)
		assert.Equal(t, 5, ac.RateLimit)
	})

	errorCases := []struct {
		name string
		data string
	}{
		{name: "invalid poll_interval", data: `{"poll_interval": "abc"}`},
		{name: "invalid report_interval", data: `{"report_interval": "abc"}`},
		{name: "number instead of string", data: `{"poll_interval": 2}`},
	}

	for _, tt := range errorCases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Error(t, json.Unmarshal([]byte(tt.data), defaults()))
		})
	}
}

// clearEnv удаляет переменные окружения names на время теста и
// восстанавливает их прежние значения после него.
func clearEnv(t *testing.T, names ...string) {
	t.Helper()
	for _, name := range names {
		t.Setenv(name, "") // регистрирует восстановление прежнего значения
		require.NoError(t, os.Unsetenv(name))
	}
}

func TestAgentConfig_ApplyEnv(t *testing.T) {
	envNames := []string{"ADDRESS", "POLL_INTERVAL", "REPORT_INTERVAL", "KEY", "RATE_LIMIT", "CRYPTO_KEY", "CONFIG"}

	base := func() AgentConfig {
		return AgentConfig{
			RateLimit:      5,
			PollInterval:   2,
			ReportInterval: 10,
			Address:        "localhost:8080",
			Key:            "flag-key",
			CryptoKey:      "/flag/public.pem",
			ConfigPath:     "/flag/config.json",
		}
	}

	tests := []struct {
		name string
		env  map[string]string
		want func(c *AgentConfig)
	}{
		{
			name: "no env keeps values",
			env:  map[string]string{},
			want: func(c *AgentConfig) {},
		},
		{
			name: "valid values override",
			env: map[string]string{
				"ADDRESS":         "example.com:9090",
				"POLL_INTERVAL":   "0.5",
				"REPORT_INTERVAL": "3",
				"KEY":             "env-key",
				"RATE_LIMIT":      "8",
				"CRYPTO_KEY":      "/env/public.pem",
				"CONFIG":          "/env/config.json",
			},
			want: func(c *AgentConfig) {
				c.Address = "example.com:9090"
				c.PollInterval = 0.5
				c.ReportInterval = 3
				c.Key = "env-key"
				c.RateLimit = 8
				c.CryptoKey = "/env/public.pem"
				c.ConfigPath = "/env/config.json"
			},
		},
		{
			name: "empty values override strings",
			env:  map[string]string{"KEY": "", "CRYPTO_KEY": ""},
			want: func(c *AgentConfig) {
				c.Key = ""
				c.CryptoKey = ""
			},
		},
		{
			name: "invalid RATE_LIMIT keeps previous value",
			env:  map[string]string{"RATE_LIMIT": "abc"},
			want: func(c *AgentConfig) {},
		},
		{
			name: "empty RATE_LIMIT keeps previous value",
			env:  map[string]string{"RATE_LIMIT": ""},
			want: func(c *AgentConfig) {},
		},
		{
			name: "non-positive RATE_LIMIT becomes 1",
			env:  map[string]string{"RATE_LIMIT": "0"},
			want: func(c *AgentConfig) { c.RateLimit = 1 },
		},
		{
			name: "invalid intervals keep previous values",
			env:  map[string]string{"POLL_INTERVAL": "abc", "REPORT_INTERVAL": "-1"},
			want: func(c *AgentConfig) {},
		},
		{
			name: "zero interval keeps previous value",
			env:  map[string]string{"POLL_INTERVAL": "0"},
			want: func(c *AgentConfig) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t, envNames...)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			got := base()
			got.ApplyEnv()

			want := base()
			tt.want(&want)
			assert.Equal(t, want, got)
		})
	}
}
