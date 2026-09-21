package config

import (
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/joho/godotenv"
)

type JiraContract struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

type Config struct {
	mu sync.RWMutex

	JiraBaseURL               string
	JiraUser                  string
	JiraPassword              string
	JiraProjectKey            string
	JiraIssueType             string
	JiraContratoDefault       string
	JiraCustomfieldContratoID string

	HTTPPort         int
	CooldownMinutes  int
	DryRun           bool
	KubeConfigPath   string
	DockerSocketPath string
	StoragePath      string
	CachePath        string
}

func (c *Config) IsDryRun() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.DryRun
}

func (c *Config) SetDryRun(val bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.DryRun = val
}

func (c *Config) ToggleDryRun() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.DryRun = !c.DryRun
	return c.DryRun
}

var ContratoMap = map[string]JiraContract{
	"INTERNO":                     {ID: "22514", Value: "INTERNO"},
	"DENTALIS":                    {ID: "22300", Value: "DENTALIS"},
	"DENTALIS (SNOW FLAKE)":       {ID: "22500", Value: "DENTALIS (SNOW FLAKE)"},
	"FARMACIA DIGITAL (AWS)":      {ID: "22502", Value: "FARMACIA DIGITAL (AWS)"},
	"GIF":                         {ID: "22517", Value: "GIF"},
	"MAIDA (DIGITAL OCEAN AKAMAI)":{ID: "22303", Value: "MAIDA (DIGITAL OCEAN AKAMAI)"},
	"MAIDA (GCP)":                 {ID: "22301", Value: "MAIDA (GCP)"},
	"MAIDA (LEGADO AWS)":          {ID: "22302", Value: "MAIDA (LEGADO AWS)"},
	"MAIDA BI CLOUDOPS (AZURE/OCI)":{ID: "22501", Value: "MAIDA BI CLOUDOPS (AZURE/OCI)"},
}

func CleanURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return strings.TrimRight(raw, "/")
	}
	return u.Scheme + "://" + u.Host
}

func getEffectiveUserHome() string {
	if snapHome := os.Getenv("SNAP_REAL_HOME"); snapHome != "" {
		return snapHome
	}
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" && sudoUser != "root" {
		return filepath.Join("/home", sudoUser)
	}
	if h, err := os.UserHomeDir(); err == nil && h != "" {
		if strings.Contains(h, "/snap/") {
			return strings.Split(h, "/snap/")[0]
		}
		return h
	}
	return ""
}

func loadEnvFiles() {
	// 1. .env no diretório atual de trabalho
	_ = godotenv.Load(".env")

	// 2. Arquivo customizado definido por variável
	if custom := os.Getenv("GUARDIAN_CONFIG_FILE"); custom != "" {
		_ = godotenv.Load(custom)
	}

	// 3. Padrão XDG Config no Home do Usuário (~/.config/guardian/guardian.env)
	home := getEffectiveUserHome()
	if home != "" {
		xdgEnv := filepath.Join(home, ".config", "guardian", "guardian.env")
		_ = godotenv.Load(xdgEnv)
		xdgDotEnv := filepath.Join(home, ".config", "guardian", ".env")
		_ = godotenv.Load(xdgDotEnv)
	}

	// 4. Configuração global do sistema (/etc/guardian/guardian.env)
	_ = godotenv.Load("/etc/guardian/guardian.env")

	// 5. Variáveis de ambiente herdadas
	_ = godotenv.Load()
}

func resolveStoragePath() string {
	if envPath := os.Getenv("GUARDIAN_STORAGE_FILE"); envPath != "" {
		return envPath
	}
	// Se existe targets.json no diretório local atual, priorizar para desenvolvimento
	if _, err := os.Stat("targets.json"); err == nil {
		return "targets.json"
	}
	// Padrão XDG Data Home (~/.local/share/guardian/targets.json)
	home := getEffectiveUserHome()
	if home != "" {
		dir := filepath.Join(home, ".local", "share", "guardian")
		_ = os.MkdirAll(dir, 0755)
		return filepath.Join(dir, "targets.json")
	}
	return "targets.json"
}

func resolveCachePath() string {
	if envPath := os.Getenv("GUARDIAN_CACHE_FILE"); envPath != "" {
		return envPath
	}
	if _, err := os.Stat("guardian_cache.json"); err == nil {
		return "guardian_cache.json"
	}
	home := getEffectiveUserHome()
	if home != "" {
		dir := filepath.Join(home, ".local", "share", "guardian")
		_ = os.MkdirAll(dir, 0755)
		return filepath.Join(dir, "guardian_cache.json")
	}
	return "guardian_cache.json"
}

func LoadConfig() *Config {
	loadEnvFiles()

	defaultKube := resolveKubeConfigPath()

	cooldown := 60
	if cdStr := os.Getenv("GUARDIAN_COOLDOWN_MINUTES"); cdStr != "" {
		if v, err := strconv.Atoi(cdStr); err == nil && v > 0 {
			cooldown = v
		}
	}

	port := 8092
	if pStr := os.Getenv("PORT"); pStr != "" {
		if v, err := strconv.Atoi(pStr); err == nil && v > 0 {
			port = v
		}
	}

	dryRun := false
	if dr := strings.ToLower(os.Getenv("GUARDIAN_DRY_RUN")); dr == "true" || dr == "1" || dr == "sim" {
		dryRun = true
	}

	dockerSock := os.Getenv("DOCKER_HOST")
	if dockerSock == "" {
		dockerSock = "/var/run/docker.sock"
	}

	jiraBase := CleanURL(os.Getenv("JIRA_BASE_URL"))
	if jiraBase == "" {
		jiraBase = "https://jira.example.com"
	}

	return &Config{
		JiraBaseURL:               jiraBase,
		JiraUser:                  strings.TrimSpace(os.Getenv("JIRA_USER")),
		JiraPassword:              strings.TrimSpace(os.Getenv("JIRA_PASSWORD")),
		JiraProjectKey:            getEnvDefault("JIRA_PROJECT_KEY", "OPS"),
		JiraIssueType:             getEnvDefault("JIRA_ISSUE_TYPE", "Bug"),
		JiraContratoDefault:       getEnvDefault("JIRA_CONTRATO_DEFAULT", ""),
		JiraCustomfieldContratoID: getEnvDefault("JIRA_CUSTOMFIELD_CONTRATO_ID", ""),
		HTTPPort:                  port,
		CooldownMinutes:           cooldown,
		DryRun:                    dryRun,
		KubeConfigPath:            defaultKube,
		DockerSocketPath:          dockerSock,
		StoragePath:               resolveStoragePath(),
		CachePath:                 resolveCachePath(),
	}
}

func getEnvDefault(key, defVal string) string {
	if val := os.Getenv(key); strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val)
	}
	return defVal
}

func (c *Config) ResolveContract(hint string) JiraContract {
	defContract, ok := ContratoMap[c.JiraContratoDefault]
	if !ok {
		defContract = ContratoMap["INTERNO"]
	}
	if strings.TrimSpace(hint) == "" {
		return defContract
	}

	upper := strings.ToUpper(strings.TrimSpace(hint))

	if exact, exists := ContratoMap[upper]; exists {
		return exact
	}

	if strings.Contains(upper, "SNOW") || strings.Contains(upper, "FLAKE") {
		return ContratoMap["DENTALIS (SNOW FLAKE)"]
	}
	if strings.Contains(upper, "DENTALIS") {
		return ContratoMap["DENTALIS"]
	}
	if strings.Contains(upper, "FARMACIA") || strings.Contains(upper, "DROGARIA") {
		return ContratoMap["FARMACIA DIGITAL (AWS)"]
	}
	if strings.Contains(upper, "GIF") {
		return ContratoMap["GIF"]
	}
	if strings.Contains(upper, "MAIDA") {
		if strings.Contains(upper, "GCP") {
			return ContratoMap["MAIDA (GCP)"]
		}
		if strings.Contains(upper, "AKAMAI") || strings.Contains(upper, "OCEAN") || strings.Contains(upper, "DO") {
			return ContratoMap["MAIDA (DIGITAL OCEAN AKAMAI)"]
		}
		if strings.Contains(upper, "BI") || strings.Contains(upper, "AZURE") || strings.Contains(upper, "OCI") {
			return ContratoMap["MAIDA BI CLOUDOPS (AZURE/OCI)"]
		}
		return ContratoMap["MAIDA (GCP)"]
	}

	for _, term := range []string{"INTERNO", "INFRA", "KUBE", "MONITOR", "DEFAULT", "CORE", "SYSTEM", "DEV", "STAGING"} {
		if strings.Contains(upper, term) {
			return ContratoMap["INTERNO"]
		}
	}

	return defContract
}

func resolveKubeConfigPath() string {
	// 1. Variável de ambiente KUBECONFIG explícita
	if envKube := os.Getenv("KUBECONFIG"); strings.TrimSpace(envKube) != "" {
		return strings.TrimSpace(envKube)
	}

	// 2. Se executado com sudo, tentar primeiro o kubeconfig do usuário real que invocou o sudo
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		sudoKube := filepath.Join("/home", sudoUser, ".kube", "config")
		if _, err := os.Stat(sudoKube); err == nil || !os.IsNotExist(err) {
			return sudoKube
		}
	}

	// 3. Diretório home do usuário atual (ou real se em ambiente Snap)
	homeDir, err := os.UserHomeDir()
	if err == nil && homeDir != "" {
		if strings.Contains(homeDir, "/snap/") {
			realHome := strings.Split(homeDir, "/snap/")[0]
			realKube := filepath.Join(realHome, ".kube", "config")
			if _, err := os.Stat(realKube); err == nil || !os.IsNotExist(err) {
				return realKube
			}
		}
		userKube := filepath.Join(homeDir, ".kube", "config")
		if _, err := os.Stat(userKube); err == nil || !os.IsNotExist(err) {
			return userKube
		}
	}

	// 4. Se o home atual for /root ou não tiver config, buscar em /home/*/.kube/config ou Documentos
	if matches, _ := filepath.Glob("/home/*/.kube/config"); len(matches) > 0 {
		for _, m := range matches {
			if _, err := os.Stat(m); err == nil || !os.IsNotExist(err) {
				return m
			}
		}
	}
	if docMatches, _ := filepath.Glob("/home/*/Documentos/kube-config*"); len(docMatches) > 0 {
		for _, m := range docMatches {
			if _, err := os.Stat(m); err == nil || !os.IsNotExist(err) {
				return m
			}
		}
	}

	// 5. Fallback padrão
	if homeDir != "" {
		return filepath.Join(homeDir, ".kube", "config")
	}
	return "/root/.kube/config"
}

