package cli

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	keys := []string{
		"SHARK_DB_BACKEND",
		"SHARK_DB_URL",
		"SHARK_AUTH_TOKEN_FILE",
		"TURSO_AUTH_TOKEN",
	}

	original := make(map[string]*string, len(keys))
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			v := value
			original[key] = &v
		} else {
			original[key] = nil
		}
		_ = os.Unsetenv(key)
	}

	code := m.Run()

	for _, key := range keys {
		if value := original[key]; value != nil {
			_ = os.Setenv(key, *value)
			continue
		}
		_ = os.Unsetenv(key)
	}

	os.Exit(code)
}
