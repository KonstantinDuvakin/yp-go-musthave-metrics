package config

import (
	"encoding/json"
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerConfig_Validate(t *testing.T) {
	tests := []struct {
		name          string
		storeInterval float64
		wantErr       bool
	}{
		{name: "positive", storeInterval: 3},
		{name: "fractional", storeInterval: 0.5},
		{name: "zero means sync", storeInterval: 0},
		{name: "negative", storeInterval: -1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &ServerConfig{StoreInterval: tt.storeInterval}

			err := sc.Validate()

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestServerConfig_UnmarshalJSON(t *testing.T) {
	// Значения по умолчанию, как после регистрации флагов.
	defaults := func() *ServerConfig {
		return &ServerConfig{
			Address:       "localhost:8080",
			StoreInterval: 3,
			StoreFile:     "metrics_log.txt",
			Restore:       true,
		}
	}

	t.Run("full file", func(t *testing.T) {
		sc := defaults()
		data := `{
			"address": "localhost:9090",
			"restore": false,
			"store_interval": "500ms",
			"store_file": "/tmp/metrics.db",
			"database_dsn": "postgres://user@localhost/db",
			"crypto_key": "/path/to/key.pem",
			"trusted_subnet": "192.168.1.0/24",
			"grpc_address": "localhost:3200"
		}`

		require.NoError(t, json.Unmarshal([]byte(data), sc))

		assert.Equal(t, "localhost:9090", sc.Address)
		assert.False(t, sc.Restore)
		assert.Equal(t, 0.5, sc.StoreInterval)
		assert.Equal(t, "/tmp/metrics.db", sc.StoreFile)
		assert.Equal(t, "postgres://user@localhost/db", sc.DB)
		assert.Equal(t, "/path/to/key.pem", sc.CryptoKey)
		assert.Equal(t, "192.168.1.0/24", sc.TrustedSubnet)
		assert.Equal(t, "localhost:3200", sc.GRPCAddress)
	})

	t.Run("missing fields keep defaults", func(t *testing.T) {
		sc := defaults()

		require.NoError(t, json.Unmarshal([]byte(`{"address": "localhost:9090"}`), sc))

		assert.Equal(t, "localhost:9090", sc.Address)
		assert.Equal(t, float64(3), sc.StoreInterval)
		assert.Equal(t, "metrics_log.txt", sc.StoreFile)
		assert.True(t, sc.Restore)
	})

	errorCases := []struct {
		name string
		data string
	}{
		{name: "invalid duration", data: `{"store_interval": "abc"}`},
		{name: "number instead of string", data: `{"store_interval": 10}`},
		{name: "invalid json", data: `{"address": `},
	}

	for _, tt := range errorCases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Error(t, json.Unmarshal([]byte(tt.data), defaults()))
		})
	}
}

func TestServerConfig_ApplyEnv(t *testing.T) {
	envNames := []string{
		"ADDRESS", "STORE_INTERVAL", "FILE_STORAGE_PATH", "CONFIG", "RESTORE",
		"DATABASE_DSN", "KEY", "AUDIT_FILE", "AUDIT_URL", "CRYPTO_KEY",
		"TRUSTED_SUBNET", "GRPC_ADDRESS",
	}

	base := func() ServerConfig {
		return ServerConfig{
			Address:       "localhost:8080",
			StoreInterval: 300,
			StoreFile:     "metrics_log.txt",
			ConfigPath:    "/flag/config.json",
			Restore:       true,
			DB:            "postgres://flag",
			Key:           "flag-key",
			AuditFile:     "/flag/audit.log",
			AuditURL:      "http://flag/audit",
			CryptoKey:     "/flag/private.pem",
			TrustedSubnet: "10.0.0.0/8",
			GRPCAddress:   "flag:3200",
		}
	}

	tests := []struct {
		name string
		env  map[string]string
		want func(c *ServerConfig)
	}{
		{
			name: "no env keeps values",
			env:  map[string]string{},
			want: func(c *ServerConfig) {},
		},
		{
			name: "valid values override",
			env: map[string]string{
				"ADDRESS":           "0.0.0.0:9090",
				"STORE_INTERVAL":    "1.5",
				"FILE_STORAGE_PATH": "/env/metrics.json",
				"CONFIG":            "/env/config.json",
				"RESTORE":           "false",
				"DATABASE_DSN":      "postgres://env",
				"KEY":               "env-key",
				"AUDIT_FILE":        "/env/audit.log",
				"AUDIT_URL":         "http://env/audit",
				"CRYPTO_KEY":        "/env/private.pem",
				"TRUSTED_SUBNET":    "192.168.1.0/24",
				"GRPC_ADDRESS":      "env:3200",
			},
			want: func(c *ServerConfig) {
				c.Address = "0.0.0.0:9090"
				c.StoreInterval = 1.5
				c.StoreFile = "/env/metrics.json"
				c.ConfigPath = "/env/config.json"
				c.Restore = false
				c.DB = "postgres://env"
				c.Key = "env-key"
				c.AuditFile = "/env/audit.log"
				c.AuditURL = "http://env/audit"
				c.CryptoKey = "/env/private.pem"
				c.TrustedSubnet = "192.168.1.0/24"
				c.GRPCAddress = "env:3200"
			},
		},
		{
			name: "empty TRUSTED_SUBNET disables subnet check",
			env:  map[string]string{"TRUSTED_SUBNET": ""},
			want: func(c *ServerConfig) { c.TrustedSubnet = "" },
		},
		{
			name: "zero STORE_INTERVAL enables sync mode",
			env:  map[string]string{"STORE_INTERVAL": "0"},
			want: func(c *ServerConfig) { c.StoreInterval = 0 },
		},
		{
			name: "empty DATABASE_DSN switches to memory",
			env:  map[string]string{"DATABASE_DSN": ""},
			want: func(c *ServerConfig) { c.DB = "" },
		},
		{
			name: "invalid STORE_INTERVAL keeps previous value",
			env:  map[string]string{"STORE_INTERVAL": "abc"},
			want: func(c *ServerConfig) {},
		},
		{
			name: "negative STORE_INTERVAL keeps previous value",
			env:  map[string]string{"STORE_INTERVAL": "-1"},
			want: func(c *ServerConfig) {},
		},
		{
			name: "invalid RESTORE keeps previous value",
			env:  map[string]string{"RESTORE": "maybe"},
			want: func(c *ServerConfig) {},
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

func TestNewConfigServer_GRPCAddressFlag(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "default is empty (gRPC disabled)", args: nil, want: ""},
		{name: "flag sets address", args: []string{"-grpc-address", ":3200"}, want: ":3200"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// NewConfigServer регистрирует флаги в глобальном flag.CommandLine,
			// поэтому на время теста подменяем его чистым набором.
			orig := flag.CommandLine
			t.Cleanup(func() { flag.CommandLine = orig })
			flag.CommandLine = flag.NewFlagSet("server", flag.ContinueOnError)

			sc := NewConfigServer()
			require.NoError(t, flag.CommandLine.Parse(tt.args))

			assert.Equal(t, tt.want, sc.GRPCAddress)
		})
	}
}
