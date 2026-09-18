package actions

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

type Deduplicator struct {
	mu              sync.RWMutex
	cacheFilePath   string
	cooldownDuration time.Duration
	cache           map[string]float64 // fingerprint -> unix timestamp
}

func NewDeduplicator(cacheFilePath string, cooldownMinutes int) *Deduplicator {
	d := &Deduplicator{
		cacheFilePath:   cacheFilePath,
		cooldownDuration: time.Duration(cooldownMinutes) * time.Minute,
		cache:           make(map[string]float64),
	}
	d.load()
	return d
}

func (d *Deduplicator) load() {
	d.mu.Lock()
	defer d.mu.Unlock()

	data, err := os.ReadFile(d.cacheFilePath)
	if err != nil {
		return
	}

	if err := json.Unmarshal(data, &d.cache); err != nil {
		log.Printf("[Deduplicator] Aviso: Falha ao carregar cache JSON (%v). Iniciando vazio.", err)
		d.cache = make(map[string]float64)
	}
}

func (d *Deduplicator) save() {
	now := float64(time.Now().Unix())
	cooldownSec := d.cooldownDuration.Seconds()

	// Limpar expirados
	clean := make(map[string]float64)
	for k, ts := range d.cache {
		if now-ts < cooldownSec {
			clean[k] = ts
		}
	}
	d.cache = clean

	data, err := json.MarshalIndent(d.cache, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(d.cacheFilePath, data, 0644)
}

func (d *Deduplicator) IsInCooldown(fingerprint string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	ts, ok := d.cache[fingerprint]
	if !ok {
		return false
	}

	now := float64(time.Now().Unix())
	elapsed := now - ts
	if elapsed < d.cooldownDuration.Seconds() {
		remainMin := int((d.cooldownDuration.Seconds() - elapsed) / 60)
		log.Printf("[Deduplicator] ⏳ Cooldown ativo para fingerprint %s (restam ~%d min). Chamado duplicado suprimido.", fingerprint, remainMin)
		return true
	}
	return false
}

func (d *Deduplicator) Record(fingerprint string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.cache[fingerprint] = float64(time.Now().Unix())
	d.save()
	fmt.Printf("[Deduplicator] [✓] Fingerprint %s registrado na janela de cooldown.\n", fingerprint)
}
