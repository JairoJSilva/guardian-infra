package k8s

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"guardian/internal/domain"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type K8sDiscovery struct {
	pool           *ClientPool
	kubeConfigPath string
}

func NewK8sDiscovery(pool *ClientPool, kubeConfigPath string) *K8sDiscovery {
	return &K8sDiscovery{
		pool:           pool,
		kubeConfigPath: kubeConfigPath,
	}
}

func (d *K8sDiscovery) loadKubeConfig() (*clientcmdapi.Config, string, error) {
	paths := []string{d.kubeConfigPath}

	if env := os.Getenv("KUBECONFIG"); env != "" {
		paths = append([]string{env}, paths...)
	}
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		paths = append(paths, filepath.Join("/home", sudoUser, ".kube", "config"))
	}
	if homeDir, err := os.UserHomeDir(); err == nil && homeDir != "" {
		paths = append(paths, filepath.Join(homeDir, ".kube", "config"))
	}
	if matches, _ := filepath.Glob("/home/*/.kube/config"); len(matches) > 0 {
		paths = append(paths, matches...)
	}

	for _, p := range paths {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			cfg, err := clientcmd.LoadFromFile(p)
			if err == nil && len(cfg.Contexts) > 0 {
				return cfg, p, nil
			}
		}
	}

	// Tentativa final com o path original
	cfg, err := clientcmd.LoadFromFile(d.kubeConfigPath)
	return cfg, d.kubeConfigPath, err
}

func (d *K8sDiscovery) DiscoverEnvironments() []domain.EnvironmentInfo {
	cfg, actualPath, err := d.loadKubeConfig()
	if err != nil || cfg == nil || len(cfg.Contexts) == 0 {
		return []domain.EnvironmentInfo{
			{
				Name:        "k8s-local",
				Type:        domain.EnvKubernetes,
				Endpoint:    "k8s-local",
				Status:      "UNAVAILABLE",
				Description: fmt.Sprintf("Nenhum cluster/contexto encontrado no kubeconfig (%s)", d.kubeConfigPath),
				Scopes:      []string{"default"},
			},
		}
	}

	// Atualiza o caminho no pool se um arquivo alternativo funcional foi encontrado
	if actualPath != d.kubeConfigPath {
		d.kubeConfigPath = actualPath
		d.pool.kubeConfigPath = actualPath
	}

	type ctxResult struct {
		info      domain.EnvironmentInfo
		isCurrent bool
	}

	ctxNames := make([]string, 0, len(cfg.Contexts))
	for name := range cfg.Contexts {
		ctxNames = append(ctxNames, name)
	}

	results := make([]ctxResult, len(ctxNames))
	var wg sync.WaitGroup

	for i, name := range ctxNames {
		wg.Add(1)
		go func(idx int, ctxName string) {
			defer wg.Done()
			ctxObj := cfg.Contexts[ctxName]
			clusterName := ctxName
			serverURL := ""
			if ctxObj != nil {
				if ctxObj.Cluster != "" {
					clusterName = ctxObj.Cluster
				}
				if clusterObj, ok := cfg.Clusters[ctxObj.Cluster]; ok && clusterObj != nil {
					serverURL = clusterObj.Server
				}
			}

			isCurrent := (cfg.CurrentContext == ctxName)

			// Tenta listar namespaces com timeout rápido e não-bloqueante
			namespaces := d.discoverNamespacesForContext(ctxName)
			status := "READY"
			if len(namespaces) == 0 {
				status = "CONFIGURED"
				if ctxObj != nil && ctxObj.Namespace != "" {
					namespaces = []string{ctxObj.Namespace, "default", "kube-system"}
				} else {
					namespaces = []string{"default", "kube-system"}
				}
			}

			desc := fmt.Sprintf("Cluster: %s", clusterName)
			if serverURL != "" {
				desc += fmt.Sprintf(" (%s)", serverURL)
			}
			if isCurrent {
				desc += " ★ [Contexto Ativo]"
			}

			results[idx] = ctxResult{
				info: domain.EnvironmentInfo{
					Name:        ctxName,
					Type:        domain.EnvKubernetes,
					Endpoint:    ctxName,
					Status:      status,
					Description: desc,
					Scopes:      namespaces,
				},
				isCurrent: isCurrent,
			}
		}(i, name)
	}

	wg.Wait()

	// Ordena colocando o contexto ativo em primeiro lugar e o restante em ordem alfabética
	sort.Slice(results, func(i, j int) bool {
		if results[i].isCurrent != results[j].isCurrent {
			return results[i].isCurrent // true primeiro
		}
		return results[i].info.Name < results[j].info.Name
	})

	envs := make([]domain.EnvironmentInfo, len(results))
	for i, r := range results {
		envs[i] = r.info
	}

	return envs
}

func (d *K8sDiscovery) discoverNamespacesForContext(ctxName string) []string {
	clientset, err := d.pool.GetClientForContext(ctxName)
	if err != nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	nsList, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil
	}

	var names []string
	for _, ns := range nsList.Items {
		names = append(names, ns.Name)
	}
	return names
}
