package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigFlagValue(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    string
		wantErr bool
	}{
		{name: "no args", args: nil, want: ""},
		{name: "no config flag", args: []string{"-a", ":9090"}, want: ""},
		{name: "separate value", args: []string{"-c", "cfg.json"}, want: "cfg.json"},
		{name: "equals value", args: []string{"-c=cfg.json"}, want: "cfg.json"},
		{name: "double dash separate", args: []string{"--c", "cfg.json"}, want: "cfg.json"},
		{name: "double dash equals", args: []string{"--c=cfg.json"}, want: "cfg.json"},
		{name: "equals as last arg", args: []string{"-a", ":9090", "-c=cfg.json"}, want: "cfg.json"},
		{name: "among other flags", args: []string{"-a", ":9090", "-c", "cfg.json", "-r=false"}, want: "cfg.json"},
		{name: "equals sign inside value", args: []string{"-c=a=b.json"}, want: "a=b.json"},
		{name: "empty value after equals", args: []string{"-c=", "-a", ":9090"}, want: ""},
		{name: "last one wins", args: []string{"-c", "a.json", "-c=b.json"}, want: "b.json"},
		{name: "similar flag is ignored", args: []string{"-crypto-key", "key.pem"}, want: ""},
		{name: "missing value", args: []string{"-a", ":9090", "-c"}, wantErr: true},

		{name: "long separate value", args: []string{"-config", "cfg.json"}, want: "cfg.json"},
		{name: "long equals value", args: []string{"-config=cfg.json"}, want: "cfg.json"},
		{name: "long double dash separate", args: []string{"--config", "cfg.json"}, want: "cfg.json"},
		{name: "long double dash equals", args: []string{"--config=cfg.json"}, want: "cfg.json"},
		{name: "long empty value after equals", args: []string{"-config=", "-a", ":9090"}, want: ""},
		{name: "long missing value", args: []string{"-a", ":9090", "-config"}, wantErr: true},
		{name: "short then long, last wins", args: []string{"-c", "a.json", "-config", "b.json"}, want: "b.json"},
		{name: "long then short, last wins", args: []string{"-config=a.json", "-c=b.json"}, want: "b.json"},
		{name: "long prefix is ignored", args: []string{"-configfoo", "x.json"}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := configFlagValue(tt.args)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// setArgs подменяет os.Args на время теста.
func setArgs(t *testing.T, args ...string) {
	t.Helper()

	orig := os.Args
	os.Args = append([]string{"test"}, args...)
	t.Cleanup(func() { os.Args = orig })
}

// writeConfig записывает content во временный файл и возвращает путь к нему.
func writeConfig(t *testing.T, name, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	return path
}

func TestParseFile(t *testing.T) {
	t.Run("no path leaves config untouched", func(t *testing.T) {
		setArgs(t)
		t.Setenv("CONFIG", "")
		sc := &ServerConfig{Address: "localhost:8080"}

		require.NoError(t, ParseFile(sc))

		assert.Equal(t, "localhost:8080", sc.Address)
	})

	t.Run("path from flag", func(t *testing.T) {
		path := writeConfig(t, "flag.json", `{"address": "from-flag:1"}`)
		setArgs(t, "-c", path)
		t.Setenv("CONFIG", "")
		sc := &ServerConfig{}

		require.NoError(t, ParseFile(sc))

		assert.Equal(t, "from-flag:1", sc.Address)
	})

	t.Run("path from long flag", func(t *testing.T) {
		path := writeConfig(t, "flag.json", `{"address": "from-long-flag:1"}`)
		setArgs(t, "-config", path)
		t.Setenv("CONFIG", "")
		sc := &ServerConfig{}

		require.NoError(t, ParseFile(sc))

		assert.Equal(t, "from-long-flag:1", sc.Address)
	})

	t.Run("path from env", func(t *testing.T) {
		path := writeConfig(t, "env.json", `{"address": "from-env:1"}`)
		setArgs(t)
		t.Setenv("CONFIG", path)
		sc := &ServerConfig{}

		require.NoError(t, ParseFile(sc))

		assert.Equal(t, "from-env:1", sc.Address)
	})

	t.Run("env wins over flag", func(t *testing.T) {
		flagPath := writeConfig(t, "flag.json", `{"address": "from-flag:1"}`)
		envPath := writeConfig(t, "env.json", `{"address": "from-env:1"}`)
		setArgs(t, "-c", flagPath)
		t.Setenv("CONFIG", envPath)
		sc := &ServerConfig{}

		require.NoError(t, ParseFile(sc))

		assert.Equal(t, "from-env:1", sc.Address)
	})

	t.Run("works for agent config", func(t *testing.T) {
		path := writeConfig(t, "agent.json", `{"poll_interval": "500ms"}`)
		setArgs(t, "-c="+path)
		t.Setenv("CONFIG", "")
		ac := &AgentConfig{}

		require.NoError(t, ParseFile(ac))

		assert.Equal(t, 0.5, ac.PollInterval)
	})

	t.Run("missing flag value", func(t *testing.T) {
		setArgs(t, "-c")
		t.Setenv("CONFIG", "")

		assert.Error(t, ParseFile(&ServerConfig{}))
	})

	t.Run("file not found", func(t *testing.T) {
		setArgs(t, "-c", filepath.Join(t.TempDir(), "missing.json"))
		t.Setenv("CONFIG", "")

		assert.Error(t, ParseFile(&ServerConfig{}))
	})

	t.Run("invalid json", func(t *testing.T) {
		path := writeConfig(t, "bad.json", `{"address": "x", // comment
		}`)
		setArgs(t, "-c", path)
		t.Setenv("CONFIG", "")

		assert.Error(t, ParseFile(&ServerConfig{}))
	})
}
