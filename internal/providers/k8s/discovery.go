package k8s

import (
	"context"
	"time"

	"guardian/internal/domain"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
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

func (d *K8sDiscovery) DiscoverEnvironments() []domain.EnvironmentInfo {
	cfg, err := clientcmd.LoadFromFile(d.kubeConfigPath)
	if err != nil {
		// Retorna mock documentado na especificação
		return []domain.EnvironmentInfo{
			{
				Name:        "aks-prod-brazil",
				Type:        domain.EnvKubernetes,
				Endpoint:    "aks-prod-brazil",
				Status:      "READY",
				Description: "Azure Kubernetes Service (Cluster Produção Brasil)",
				Scopes:      []string{"billing", "checkout", "orders", "notifications"},
			},
			{
				Name:        "eks-us-east-1",
				Type:        domain.EnvKubernetes,
				Endpoint:    "eks-us-east-1",
				Status:      "PAUSED",
				Description: "AWS EKS Cluster (Autenticação e Usuários)",
				Scopes:      []string{"auth", "keycloak-prod", "user-profile"},
			},
		}
	}

	var envs []domain.EnvironmentInfo
	for ctxName := range cfg.Contexts {
		namespaces := d.discoverNamespacesForContext(ctxName)
		status := "READY"
		if len(namespaces) == 0 {
			namespaces = []string{"default", "kube-system"}
		}

		envs = append(envs, domain.EnvironmentInfo{
			Name:        ctxName,
			Type:        domain.EnvKubernetes,
			Endpoint:    ctxName,
			Status:      status,
			Description: "Kubernetes Context: " + ctxName,
			Scopes:      namespaces,
		})
	}

	if len(envs) == 0 {
		return []domain.EnvironmentInfo{
			{
				Name:        "aks-prod-brazil",
				Type:        domain.EnvKubernetes,
				Endpoint:    "aks-prod-brazil",
				Status:      "READY",
				Description: "Azure Kubernetes Service (Cluster Produção Brasil)",
				Scopes:      []string{"billing", "checkout", "orders", "notifications"},
			},
			{
				Name:        "eks-us-east-1",
				Type:        domain.EnvKubernetes,
				Endpoint:    "eks-us-east-1",
				Status:      "PAUSED",
				Description: "AWS EKS Cluster (Autenticação e Usuários)",
				Scopes:      []string{"auth", "keycloak-prod", "user-profile"},
			},
		}
	}

	return envs
}

func (d *K8sDiscovery) discoverNamespacesForContext(ctxName string) []string {
	clientset, err := d.pool.GetClientForContext(ctxName)
	if err != nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
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
