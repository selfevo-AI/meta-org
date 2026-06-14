package config

import "testing"

func TestLoadUses32ByteDefaultModelSecretKey(t *testing.T) {
	t.Setenv("MODEL_SECRET_KEY", "")

	cfg := Load()

	if len(cfg.ModelSecretKey) != 32 {
		t.Fatalf("ModelSecretKey length = %d, want 32", len(cfg.ModelSecretKey))
	}
}
