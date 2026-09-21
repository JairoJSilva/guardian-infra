package supervisor

import (
	"context"
	"log"

	"guardian/internal/domain"
	"guardian/internal/providers/docker"
	"guardian/internal/providers/k8s"
)

type TargetWorker struct {
	target     *domain.Target
	ctx        context.Context
	cancel     context.CancelFunc
	k8sPool    *k8s.ClientPool
	dockerPool *docker.DockerPool
	outChan    chan<- *domain.IncidentEvent
}

func NewTargetWorker(
	target *domain.Target,
	parentCtx context.Context,
	k8sPool *k8s.ClientPool,
	dockerPool *docker.DockerPool,
	outChan chan<- *domain.IncidentEvent,
) *TargetWorker {
	ctx, cancel := context.WithCancel(parentCtx)
	return &TargetWorker{
		target:     target,
		ctx:        ctx,
		cancel:     cancel,
		k8sPool:    k8sPool,
		dockerPool: dockerPool,
		outChan:    outChan,
	}
}

func (w *TargetWorker) Start() {
	go func() {
		log.Printf("[Supervisor Worker] Iniciando worker para Target '%s' (Tipo: %s, Endpoint: %s)...", w.target.Name, w.target.Type, w.target.Endpoint)

		if w.target.Type == domain.EnvKubernetes {
			watcher := k8s.NewK8sWatcher(w.k8sPool, w.target)
			if err := watcher.Watch(w.ctx, w.outChan); err != nil {
				log.Printf("[Supervisor Worker] Erro no watcher K8s para %s: %v", w.target.Name, err)
			}
		} else if w.target.Type == domain.EnvDocker {
			client := w.dockerPool.GetClient(w.target.Endpoint)
			watcher := docker.NewDockerWatcher(client, w.target)
			if err := watcher.Watch(w.ctx, w.outChan); err != nil {
				log.Printf("[Supervisor Worker] Erro no watcher Docker para %s (%s): %v", w.target.Name, w.target.Endpoint, err)
			}
		} else {
			log.Printf("[Supervisor Worker] Tipo de ambiente desconhecido para target %s: %s", w.target.ID, w.target.Type)
		}
	}()
}

func (w *TargetWorker) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
}
