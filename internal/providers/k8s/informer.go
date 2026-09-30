package k8s

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"guardian/internal/domain"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type K8sWatcher struct {
	pool    *ClientPool
	target  *domain.Target
	seeding bool // quando true, o scan popula o dedup sem disparar eventos
}

func NewK8sWatcher(pool *ClientPool, target *domain.Target) *K8sWatcher {
	return &K8sWatcher{
		pool:   pool,
		target: target,
	}
}

func (w *K8sWatcher) Watch(ctx context.Context, out chan<- *domain.IncidentEvent) error {
	interval := time.Duration(w.target.Rules.PollIntervalSeconds) * time.Second
	if interval < 5*time.Second {
		interval = 15 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("[K8sWatcher] [Target: %s] Iniciando monitoramento K8s no contexto '%s' para os namespaces %v...", w.target.Name, w.target.Endpoint, w.target.Scopes)

	// Scan inicial em modo seeding: registra o estado atual do cluster como baseline
	// sem disparar eventos — evita flood de alertas de pods já evicted/failed no momento
	// em que o target é criado.
	w.seeding = true
	w.scan(ctx, out)
	w.seeding = false
	log.Printf("[K8sWatcher] [Target: %s] Baseline do cluster registrado. Monitoramento ativo iniciado.", w.target.Name)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[K8sWatcher] [Target: %s] Monitoramento encerrado via cancelamento de contexto.", w.target.Name)
			return nil
		case <-ticker.C:
			w.scan(ctx, out)
		}
	}
}

func (w *K8sWatcher) scan(ctx context.Context, out chan<- *domain.IncidentEvent) {
	clientset, err := w.pool.GetClientForContext(w.target.Endpoint)
	if err != nil {
		log.Printf("[K8sWatcher] ⚠️ [Target: %s] Falha ao obter client K8s para contexto '%s': %v", w.target.Name, w.target.Endpoint, err)
		return
	}

	for _, ns := range w.target.Scopes {
		pods, err := clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			log.Printf("[K8sWatcher] ⚠️ [Target: %s] Falha ao listar pods no namespace '%s': %v", w.target.Name, ns, err)
			continue
		}

		for _, pod := range pods.Items {
			w.inspectPod(ctx, clientset, &pod, ns, out)
		}
	}
}

func (w *K8sWatcher) inspectPod(ctx context.Context, clientset *kubernetes.Clientset, pod *corev1.Pod, ns string, out chan<- *domain.IncidentEvent) {
	// Se o pod inteiro estiver em status Failed (inclui Evicted)
	if pod.Status.Phase == corev1.PodFailed {
		reason := "PodFailed"
		if pod.Status.Reason != "" {
			reason = pod.Status.Reason
		}

		// Gera o fingerprint do evento para registro no deduplicador
		fp := domain.FingerprintRaw(string(domain.EnvKubernetes), w.target.Endpoint, ns, pod.Name, reason)

		// Modo seeding: apenas registra o fingerprint como baseline, sem emitir evento.
		// Evita flood de 200+ alertas de pods já evicted quando o target é criado.
		if w.seeding {
			log.Printf("[K8sWatcher] [Baseline] Pod já evicted/failed ignorado (seed): %s/%s (%s)", ns, pod.Name, reason)
			out <- &domain.IncidentEvent{
				ID:         fmt.Sprintf("evt-seed-%s", fp),
				Type:       domain.EnvKubernetes,
				TargetID:   w.target.ID,
				Environment: w.target.Endpoint,
				Scope:      ns,
				EntityName: pod.Name,
				Reason:     reason,
				ExitCode:   1,
				Timestamp:  time.Now(),
				Severity:   "SEED", // marcador especial para o supervisor ignorar
			}
			return
		}

		event := &domain.IncidentEvent{
			ID:          fmt.Sprintf("evt-k8s-%d", time.Now().UnixNano()),
			Type:        domain.EnvKubernetes,
			TargetID:    w.target.ID,
			Environment: w.target.Endpoint,
			Scope:       ns,
			EntityName:  pod.Name,
			Image:       pod.Spec.Containers[0].Image,
			Reason:      reason,
			ExitCode:    1,
			Logs:        w.fetchPodLogs(ctx, clientset, pod.Name, ns, pod.Spec.Containers[0].Name),
			Timestamp:   time.Now(),
			Severity:    "CRITICAL",
		}
		out <- event
		return
	}

	// Analisa cada container do Pod
	for _, cs := range pod.Status.ContainerStatuses {
		isFailed := false
		reason := ""
		exitCode := 0
		severity := "WARNING"

		// 1. Container aguardando com erro
		if cs.State.Waiting != nil {
			wReason := cs.State.Waiting.Reason
			if wReason == "CrashLoopBackOff" || wReason == "Error" || wReason == "ImagePullBackOff" || wReason == "ErrImagePull" || wReason == "CreateContainerConfigError" || wReason == "CreateContainerError" {
				isFailed = true
				reason = wReason
				if cs.LastTerminationState.Terminated != nil {
					exitCode = int(cs.LastTerminationState.Terminated.ExitCode)
					if cs.LastTerminationState.Terminated.Reason == "OOMKilled" || exitCode == 137 {
						reason = "OOMKilled"
						severity = "CRITICAL"
					}
				}
			}
		}

		// 2. Container terminado com erro
		if cs.State.Terminated != nil && cs.State.Terminated.ExitCode != 0 {
			isFailed = true
			exitCode = int(cs.State.Terminated.ExitCode)
			reason = cs.State.Terminated.Reason
			if reason == "" {
				reason = fmt.Sprintf("Error (ExitCode %d)", exitCode)
			}
			if reason == "OOMKilled" || exitCode == 137 {
				severity = "CRITICAL"
			}
		}

		// 3. Reinicializações repetidas
		if cs.RestartCount > 3 && !cs.Ready && !isFailed {
			isFailed = true
			reason = fmt.Sprintf("CrashLoopRestarting (%d restarts)", cs.RestartCount)
			severity = "WARNING"
		}

		if !isFailed {
			continue
		}

		// Modo seeding: emite evento marcado como SEED para o supervisor registrar
		// o fingerprint como baseline, sem broadcast e sem abertura de chamado.
		if w.seeding {
			fp := domain.FingerprintRaw(string(domain.EnvKubernetes), w.target.Endpoint, ns, pod.Name, reason)
			log.Printf("[K8sWatcher] [Baseline] Container com falha ignorado (seed): %s/%s container=%s (%s)", ns, pod.Name, cs.Name, reason)
			select {
			case out <- &domain.IncidentEvent{
				ID:          fmt.Sprintf("evt-seed-%s", fp),
				Type:        domain.EnvKubernetes,
				TargetID:    w.target.ID,
				Environment: w.target.Endpoint,
				Scope:       ns,
				EntityName:  pod.Name,
				Reason:      reason,
				ExitCode:    exitCode,
				Timestamp:   time.Now(),
				Severity:    "SEED",
			}:
			default:
			}
			continue
		}

		// Coleta logs do pod
		logs := w.fetchPodLogs(ctx, clientset, pod.Name, ns, cs.Name)

		event := &domain.IncidentEvent{
			ID:          fmt.Sprintf("evt-k8s-%d", time.Now().UnixNano()),
			Type:        domain.EnvKubernetes,
			TargetID:    w.target.ID,
			Environment: w.target.Endpoint,
			Scope:       ns,
			EntityName:  pod.Name,
			Image:       cs.Image,
			Reason:      reason,
			ExitCode:    exitCode,
			Logs:        logs,
			Timestamp:   time.Now(),
			Severity:    severity,
			Details: map[string]interface{}{
				"container":     cs.Name,
				"restart_count": cs.RestartCount,
				"pod_phase":     string(pod.Status.Phase),
			},
		}

		select {
		case out <- event:
		default:
		}
	}
}

func (w *K8sWatcher) fetchPodLogs(ctx context.Context, clientset *kubernetes.Clientset, podName, ns, containerName string) string {
	tailLines := int64(50)
	opts := &corev1.PodLogOptions{
		Container: containerName,
		TailLines: &tailLines,
		Previous:  true, // Tenta coletar do container anterior que crashou
	}

	req := clientset.CoreV1().Pods(ns).GetLogs(podName, opts)
	stream, err := req.Stream(ctx)
	if err != nil {
		// Se previous falhar, tenta logs do container atual
		opts.Previous = false
		req = clientset.CoreV1().Pods(ns).GetLogs(podName, opts)
		stream, err = req.Stream(ctx)
		if err != nil {
			return ""
		}
	}
	defer stream.Close()

	buf := new(bytes.Buffer)
	_, _ = io.Copy(buf, stream)
	return buf.String()
}
