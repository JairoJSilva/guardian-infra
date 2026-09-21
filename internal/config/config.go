package config

import (
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type JiraContract struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

type Config struct {
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
		return "https://jira.mv.com.br"
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return strings.TrimRight(raw, "/")
	}
	return u.Scheme + "://" + u.Host
}

func LoadConfig() *Config {
	_ = godotenv.Load(".env")
	_ = godotenv.Load()

	defaultKube := resolveKubeConfigPath()

	cooldown := 60
	if cdStr := os.Getenv("GUARDIAN_COOLDOWN_MINUTES"); cdStr != "" {
		if v, err := strconv.Atoi(cdStr); err == nil && v > 0 {
			cooldown = v
		}
	}

	port := 8080
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

	return &Config{
		JiraBaseURL:               CleanURL(os.Getenv("JIRA_BASE_URL")),
		JiraUser:                  strings.TrimSpace(os.Getenv("JIRA_USER")),
		JiraPassword:              strings.TrimSpace(os.Getenv("JIRA_PASSWORD")),
		JiraProjectKey:            getEnvDefault("JIRA_PROJECT_KEY", "OPS"),
		JiraIssueType:             getEnvDefault("JIRA_ISSUE_TYPE", "Solicitação de serviço"),
		JiraContratoDefault:       getEnvDefault("JIRA_CONTRATO_DEFAULT", "INTERNO"),
		JiraCustomfieldContratoID: getEnvDefault("JIRA_CUSTOMFIELD_CONTRATO_ID", "customfield_30118"),
		HTTPPort:                  port,
		CooldownMinutes:           cooldown,
		DryRun:                    dryRun,
		KubeConfigPath:            defaultKube,
		DockerSocketPath:          dockerSock,
		StoragePath:               getEnvDefault("GUARDIAN_STORAGE_FILE", "targets.json"),
		CachePath:                 getEnvDefault("GUARDIAN_CACHE_FILE", "guardian_cache.json"),
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

	// 4. Se o home atual for /root ou não tiver config, buscar em /home/*/.kube/config
	if matches, _ := filepath.Glob("/home/*/.kube/config"); len(matches) > 0 {
		for _, m := range matches {
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

