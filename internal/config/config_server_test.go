package config

import (
	"encoding/json"
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
			"crypto_key": "/path/to/key.pem"
		}`

		require.NoError(t, json.Unmarshal([]byte(data), sc))

		assert.Equal(t, "localhost:9090", sc.Address)
		assert.False(t, sc.Restore)
		assert.Equal(t, 0.5, sc.StoreInterval)
		assert.Equal(t, "/tmp/metrics.db", sc.StoreFile)
		assert.Equal(t, "postgres://user@localhost/db", sc.DB)
		assert.Equal(t, "/path/to/key.pem", sc.CryptoKey)
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
