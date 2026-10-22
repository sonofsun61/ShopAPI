package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sonofsun61/APIFromSpec/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var envKeys = []string{"DATABASE_USER", "DATABASE_PASSWORD", "DATABASE_PORT", "DATABASE_NAME", "AUTH_SERVICE_ADDR"}

func inTempDir(t *testing.T, envContent string) {
	t.Helper()
	dir := t.TempDir()
	if envContent != "" {
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte(envContent), 0o600))
	}
	t.Chdir(dir)
	for _, key := range envKeys {
		t.Setenv(key, "")
		require.NoError(t, os.Unsetenv(key))
	}
}

func TestMustLoadConfig_Success(t *testing.T) {
	inTempDir(t, "DATABASE_USER=user\n"+
		"DATABASE_PASSWORD=secret\n"+
		"DATABASE_PORT=5432\n"+
		"DATABASE_NAME=shop\n"+
		"AUTH_SERVICE_ADDR=localhost:50051\n")

	cfg := config.MustLoadConfig()

	require.NotNil(t, cfg)
	assert.Equal(t, "postgres://user:secret@localhost:5432/shop?sslmode=disable", cfg.ConnString)
	assert.Equal(t, "localhost:50051", cfg.AuthServiceAddr)
}

func TestMustLoadConfig_NoEnvFile(t *testing.T) {
	inTempDir(t, "")

	assert.PanicsWithValue(t, "Could not load .env file", func() {
		config.MustLoadConfig()
	})
}

func TestMustLoadConfig_NoAuthServiceAddr(t *testing.T) {
	inTempDir(t, "DATABASE_USER=user\n")

	assert.PanicsWithValue(t, "AUTH_SERVICE_ADDR is not set", func() {
		config.MustLoadConfig()
	})
}
