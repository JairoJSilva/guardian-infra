package actions

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"guardian/internal/domain"
)

func TestDeduplicator(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.json")

	dedup := NewDeduplicator(cachePath, 60)

	evt := &domain.IncidentEvent{
		Type:        domain.EnvKubernetes,
		Environment: "aks-prod-brazil",
		Scope:       "billing",
		EntityName:  "payment-pod-1",
		Reason:      "CrashLoopBackOff",
		Timestamp:   time.Now(),
	}

	fp := evt.Fingerprint()
	if fp == "" {
		t.Fatal("fingerprint não pode ser vazio")
	}

	// Inicialmente não deve estar em cooldown
	if dedup.IsInCooldown(fp) {
		t.Errorf("fingerprint %s não deveria estar em cooldown antes do registro", fp)
	}

	// Registra incidente
	dedup.Record(fp)

	// Agora deve estar em cooldown
	if !dedup.IsInCooldown(fp) {
		t.Errorf("fingerprint %s deveria estar em cooldown após registro", fp)
	}

	// Verifica se persistiu em disco
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Errorf("arquivo de cache %s não foi criado", cachePath)
	}
}
