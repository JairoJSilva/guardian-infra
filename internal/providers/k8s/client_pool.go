package k8s

import (
	"fmt"
	"sync"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type ClientPool struct {
	mu             sync.RWMutex
	kubeConfigPath string
	clients        map[string]*kubernetes.Clientset
}

func NewClientPool(kubeConfigPath string) *ClientPool {
	return &ClientPool{
		kubeConfigPath: kubeConfigPath,
		clients:        make(map[string]*kubernetes.Clientset),
	}
}

func (p *ClientPool) GetClientForContext(contextName string) (*kubernetes.Clientset, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if client, ok := p.clients[contextName]; ok {
		return client, nil
	}

	loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: p.kubeConfigPath}
	configOverrides := &clientcmd.ConfigOverrides{}
	if contextName != "" {
		configOverrides.CurrentContext = contextName
	}

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	restConfig, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("falha ao carregar restConfig para contexto %s: %w", contextName, err)
	}

	// Timeout de requisição seguro
	restConfig.Timeout = 10 * 1e9 // 10s

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("falha ao criar clientset para contexto %s: %w", contextName, err)
	}

	p.clients[contextName] = clientset
	return clientset, nil
}
