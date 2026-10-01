package analyzer

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"guardian/internal/domain"
)

// AnalysisResult contém a análise prévia de causa raiz e sugestões acionáveis para o operador SRE.
type AnalysisResult struct {
	Category          string   `json:"category"`           // Categoria do incidente
	Summary           string   `json:"summary"`            // Resumo conciso em 1 frase
	RootCause         string   `json:"root_cause"`         // Análise prévia detalhada da provável causa raiz
	SuggestedFix      string   `json:"suggested_fix"`      // Explicação da correção/ajuste recomendado
	ActionSteps       []string `json:"action_steps"`       // Passos ordenados que o operador deve executar
	SuggestedCommands []string `json:"suggested_commands"` // Comandos prontos para execução em terminal
}

// AnalyzeIncident avalia o evento de incidente, logs e contexto de execução e gera uma análise técnica
// preliminar com sugestões de ajuste manual.
func AnalyzeIncident(event *domain.IncidentEvent) *AnalysisResult {
	if event == nil {
		return &AnalysisResult{
			Category:     "DESCONHECIDO",
			Summary:      "Evento não especificado",
			RootCause:    "Não foi possível extrair dados do evento.",
			SuggestedFix: "Verifique os logs do cluster manualmente.",
		}
	}

	// 0. ANÁLISE POR IA VIA ANTIGRAVITY LOCAL (Modo Estritamente Leitura / RCA)
	if IsAIAnalysisEnabled() && strings.TrimSpace(event.Logs) != "" {
		if aiRes, err := AnalyzeWithAntigravity(context.Background(), event); err == nil && aiRes != nil {
			return aiRes
		} else if err != nil {
			log.Printf("[Analyzer] ℹ️ Análise via Antigravity indisponível (%v). Prosseguindo com regras heurísticas de fallback.", err)
		}
	}

	reasonUpper := strings.ToUpper(strings.TrimSpace(event.Reason))
	logsLower := strings.ToLower(event.Logs)
	isK8s := event.Type == domain.EnvKubernetes
	scope := event.Scope
	if scope == "" {
		scope = "default"
	}
	entity := event.EntityName

	// 1. POD EVICTED (Pressão no nó K8s tem prioridade mesmo se o exit code for 137)
	if reasonUpper == "EVICTED" || strings.Contains(logsLower, "evicted") || strings.Contains(logsLower, "eviction manager") {
		return analyzeEvicted(scope, entity, event)
	}

	// 2. OOMKILLED / OUT OF MEMORY
	if reasonUpper == "OOMKILLED" || event.ExitCode == 137 || matchAny(logsLower, "out of memory", "oom-killer", "cannot allocate memory", "memory cgroup out of memory", "java.lang.outofmemoryerror") {
		return analyzeOOM(isK8s, scope, entity, event)
	}

	// 3. FALHA DE MIGRAÇÃO / SCHEMA DE BANCO DE DADOS (Flyway / Liquibase / DDL)
	if matchAny(logsLower, "flywaysqlscriptexception", "migration.*failed", "relation.*does not exist", "table.*does not exist", "column.*does not exist", "flyway") {
		return analyzeDatabaseMigrationFailure(isK8s, scope, entity, logsLower)
	}

	// 4. FALHA DE CONEXÃO COM BANCO DE DADOS
	if matchAny(logsLower,
		"connection refused", "dial tcp.*5432", "dial tcp.*3306", "dial tcp.*1521", "dial tcp.*27017", "dial tcp.*1433",
		"cannot connect to postgres", "cannot connect to mysql", "could not connect to server: connection refused",
		"too many connections", "max_connections", "connection pool exhausted", "deadlock detected",
	) {
		return analyzeDatabaseFailure(isK8s, scope, entity, logsLower)
	}

	// 4. FALHA DE CACHE / REDIS
	if matchAny(logsLower, "dial tcp.*6379", "redis.*connection refused", "cannot connect to redis", "redis.*loading", "max number of clients reached") {
		return analyzeRedisFailure(isK8s, scope, entity)
	}

	// 5. FALHA DE DNS OU REDE
	if matchAny(logsLower, "no such host", "lookup.*failed", "name resolution", "temporary failure in name resolution", "dial tcp: lookup") {
		return analyzeDNSFailure(isK8s, scope, entity)
	}

	// 6. AUTENTICAÇÃO / AUTORIZAÇÃO / PERMISSÃO
	if matchAny(logsLower, "permission denied", "access denied", "unauthorized", "401 unauthorized", "403 forbidden", "token expired", "invalid jwt") {
		return analyzeAuthOrPermission(isK8s, scope, entity, logsLower)
	}

	// 7. CONFIGURAÇÃO / VARIÁVEIS DE AMBIENTE / ARQUIVOS FALTANTES
	if matchAny(logsLower, "no such file or directory", "config.*missing", "required environment variable", "env var not set", "cannot find configuration") {
		return analyzeConfigFailure(isK8s, scope, entity, logsLower)
	}

	// 8. CONFLITO DE PORTA (EADDRINUSE)
	if matchAny(logsLower, "address already in use", "eaddrinuse", "bind: address already in use") {
		return analyzePortConflict(isK8s, scope, entity)
	}

	// 9. PANIC / EXCEÇÃO CRÍTICA NÃO TRATADA
	if matchAny(logsLower, "panic:", "nil pointer dereference", "fatal error:", "segmentation fault", "sigsegv", "nullpointerexception", "modulenotfounderror", "traceback (most recent call last)") {
		return analyzePanicOrException(isK8s, scope, entity, logsLower)
	}

	// 10. CRASH / FALHA GENÉRICA DE EXECUÇÃO
	return analyzeGenericCrash(isK8s, scope, entity, event)
}

func analyzeOOM(isK8s bool, scope, entity string, event *domain.IncidentEvent) *AnalysisResult {
	if isK8s {
		return &AnalysisResult{
			Category:     "MEMÓRIA (OOMKilled / Limite de RAM Excedido)",
			Summary:      "Container abortado pelo Linux OOM-Killer por ultrapassar a cota de memória configurada.",
			RootCause:    fmt.Sprintf("O processo em execução no Pod '%s' (namespace '%s') consumiu mais memória RAM do que o teto estabelecido em 'resources.limits.memory', ou o nó sofreu pressão severa de RAM. O kernel enviou o sinal SIGKILL (137).", entity, scope),
			SuggestedFix: "Aumente a cota de memória no manifesto do workload (Deployment/StatefulSet) ou investigue vazamento de memória (memory leak) na aplicação. Em apps Java/JVM, revise as flags '-Xmx' ou '-XX:MaxRAMPercentage=75'.",
			ActionSteps: []string{
				fmt.Sprintf("1. Verifique os limites atuais e o último estado com: kubectl describe pod %s -n %s", entity, scope),
				fmt.Sprintf("2. Analise o consumo de recursos recente via métricas: kubectl top pod %s -n %s --containers", entity, scope),
				"3. Se o consumo for legítimo devido à carga, edite o manifesto aumentando o 'limits.memory' (ex: de 512Mi para 1Gi ou 2Gi).",
				"4. Se o consumo cresce indefinidamente mesmo com carga constante, acione a equipe de desenvolvimento para investigar vazamento de memória ou threads acumuladas.",
			},
			SuggestedCommands: []string{
				fmt.Sprintf("kubectl describe pod %s -n %s | grep -A 12 -B 2 'Limits:'", entity, scope),
				fmt.Sprintf("kubectl top pod %s -n %s --containers", entity, scope),
				fmt.Sprintf("kubectl logs %s -n %s --previous --tail=50", entity, scope),
			},
		}
	}

	return &AnalysisResult{
		Category:     "MEMÓRIA (Docker OOM / Memória Insuficiente)",
		Summary:      "Container Docker finalizado com código 137 por esgotamento de memória no host ou limite do cgroup.",
		RootCause:    fmt.Sprintf("O container '%s' excedeu o limite de memória configurado no Docker Compose/daemon ou consumiu toda a RAM livre do host, sendo terminado pelo kernel.", entity),
		SuggestedFix: "Ajuste o limite de memória nas configurações do container (ex: 'mem_limit' no docker-compose.yml) ou aumente a capacidade de memória da máquina hospedeira.",
		ActionSteps: []string{
			fmt.Sprintf("1. Verifique o limite configurado: docker inspect %s --format '{{.HostConfig.Memory}}'", entity),
			"2. Verifique o consumo geral de memória do host: free -h",
			"3. Ajuste o parâmetro 'mem_limit' no compose file e recrie o container.",
		},
		SuggestedCommands: []string{
			fmt.Sprintf("docker inspect %s --format 'Nome: {{.Name}} | MemoryLimit: {{.HostConfig.Memory}} | ExitCode: {{.State.ExitCode}}'", entity),
			fmt.Sprintf("docker logs --tail 50 %s", entity),
			"free -h && vmstat 1 5",
		},
	}
}

func analyzeEvicted(scope, entity string, event *domain.IncidentEvent) *AnalysisResult {
	return &AnalysisResult{
		Category:     "INFRAESTRUTURA (Pod Evicted / Pressão de Recursos no Nó)",
		Summary:      "Pod expulso pelo Kubelet Eviction Manager para proteger a estabilidade do nó Kubernetes.",
		RootCause:    fmt.Sprintf("O nó onde o Pod '%s' estava alocado atingiu um gatilho de pressão de recursos (DiskPressure, MemoryPressure ou PIDPressure). O Kubelet expulsa pods que não têm QoS Guaranteed ou que utilizam mais ephemeral-storage do que o permitido.", entity),
		SuggestedFix: "Limpar pods evictados antigos para desobstruir a API, inspecionar a partição de disco dos nós (/var/lib/kubelet e /var/lib/docker) e configurar 'ephemeral-storage' requests e limits no Pod.",
		ActionSteps: []string{
			fmt.Sprintf("1. Identifique o nó do pod e o motivo da expulsão: kubectl describe pod %s -n %s | grep -E 'Node:|Reason:|Message:'", entity, scope),
			"2. Inspecione as condições dos nós do cluster buscando por DiskPressure ou MemoryPressure: kubectl get nodes",
			fmt.Sprintf("3. Exclua os pods evicted que permanecem em estado Failed: kubectl delete pod %s -n %s", entity, scope),
			"4. Se a causa for disco cheio, execute a limpeza de imagens órfãs ou expanda o disco dos workers do cluster.",
		},
		SuggestedCommands: []string{
			fmt.Sprintf("kubectl describe pod %s -n %s | grep -E 'Node:|Reason:|Message:'", entity, scope),
			"kubectl get nodes -o wide",
			fmt.Sprintf("kubectl delete pod %s -n %s", entity, scope),
			fmt.Sprintf("kubectl get pods -n %s --field-selector=status.phase=Failed", scope),
		},
	}
}

func analyzeDatabaseMigrationFailure(isK8s bool, scope, entity, logsLower string) *AnalysisResult {
	framework := "Migration / DDL"
	if strings.Contains(logsLower, "flyway") {
		framework = "Flyway Migration"
	} else if strings.Contains(logsLower, "liquibase") {
		framework = "Liquibase Migration"
	}

	missingRelation := "tabela ou coluna"
	if strings.Contains(logsLower, "relation") && strings.Contains(logsLower, "does not exist") {
		missingRelation = "tabela inexistente (relation does not exist)"
	} else if strings.Contains(logsLower, "column") && strings.Contains(logsLower, "does not exist") {
		missingRelation = "coluna inexistente (column does not exist)"
	}

	if isK8s {
		return &AnalysisResult{
			Category:     fmt.Sprintf("BANCO DE DADOS (%s / %s)", framework, missingRelation),
			Summary:      "Aplicação falhou durante a inicialização ao executar migrações de esquema de banco de dados.",
			RootCause:    fmt.Sprintf("O processo de migração (%s) falhou no Pod '%s' (namespace '%s'). Um script SQL de migração tentou operar em um objeto de banco que não existe no esquema configurado ou possui dependência ausente.", framework, entity, scope),
			SuggestedFix: "Inspecione os logs com '--previous' para identificar o script SQL da falha. Verifique se as migrações anteriores foram executadas, garanta a criação prévia da tabela/coluna no banco e execute 'flyway repair' se necessário para desobstruir o baseline.",
			ActionSteps: []string{
				fmt.Sprintf("1. Extraia o stacktrace do script SQL com falha: kubectl logs %s -n %s --previous --tail=100", entity, scope),
				"2. Identifique a versão do arquivo de migração (ex: V...__nome.sql) e a instrução SQL que falhou.",
				"3. Valide no banco de dados se a tabela/esquema prévio existe e se o usuário possui permissão de DDL.",
				"4. Caso o histórico de migrações esteja marcado como falho, avalie executar 'repair' no banco ou gerar hotfix na pipeline.",
			},
			SuggestedCommands: []string{
				fmt.Sprintf("kubectl logs %s -n %s --previous --tail=100 | grep -i -C 5 'migration'", entity, scope),
				fmt.Sprintf("kubectl describe pod %s -n %s | grep -i 'Image:'", entity, scope),
			},
		}
	}

	return &AnalysisResult{
		Category:     fmt.Sprintf("BANCO DE DADOS (%s / %s)", framework, missingRelation),
		Summary:      "Container falhou ao executar migração de banco de dados no startup.",
		RootCause:    fmt.Sprintf("O container '%s' encerrou com erro ao aplicar scripts de migração (%s).", entity, framework),
		SuggestedFix: "Verifique o histórico de migrações no banco de dados e confirme a integridade dos scripts SQL montados no container.",
		ActionSteps: []string{
			fmt.Sprintf("1. Analise os logs do container: docker logs --tail 80 %s", entity),
			"2. Valide o esquema do banco de dados referenciado.",
		},
		SuggestedCommands: []string{
			fmt.Sprintf("docker logs --tail 80 %s", entity),
		},
	}
}

func analyzeDatabaseFailure(isK8s bool, scope, entity, logsLower string) *AnalysisResult {
	dbType := "Banco de Dados"
	if strings.Contains(logsLower, "postgres") || strings.Contains(logsLower, "5432") {
		dbType = "PostgreSQL"
	} else if strings.Contains(logsLower, "mysql") || strings.Contains(logsLower, "3306") {
		dbType = "MySQL / MariaDB"
	} else if strings.Contains(logsLower, "oracle") || strings.Contains(logsLower, "1521") {
		dbType = "Oracle DB"
	} else if strings.Contains(logsLower, "mongo") || strings.Contains(logsLower, "27017") {
		dbType = "MongoDB"
	}

	if isK8s {
		return &AnalysisResult{
			Category:     fmt.Sprintf("BANCO DE DADOS (%s / Conexão Recusada ou Timeout)", dbType),
			Summary:      fmt.Sprintf("Aplicação não consegue se comunicar com a instância do %s.", dbType),
			RootCause:    fmt.Sprintf("Os logs indicam falha ao estabelecer handshake TCP/TLS com o %s. Causas comuns: serviço de banco parado, credenciais incorretas nas Secrets/ConfigMaps, bloqueio por NetworkPolicy ou hostname incorreto.", dbType),
			SuggestedFix: fmt.Sprintf("Verifique se o serviço do %s está UP e acessível a partir do namespace '%s', valide as variáveis de conexão (DB_HOST, DB_PORT) e teste a rota de rede.", dbType, scope),
			ActionSteps: []string{
				fmt.Sprintf("1. Verifique as variáveis de ambiente e secrets de banco montadas no pod: kubectl describe pod %s -n %s | grep -A 8 'Environment:'", entity, scope),
				"2. Confirme se o serviço do banco de dados está online e respondendo.",
				fmt.Sprintf("3. Teste a conectividade de rede a partir do namespace: kubectl run netshoot-test --rm -i --tty --image=nicolaka/netshoot -n %s -- bash", scope),
				"4. Verifique se há NetworkPolicies bloqueando a porta no namespace.",
			},
			SuggestedCommands: []string{
				fmt.Sprintf("kubectl describe pod %s -n %s | grep -A 10 'Environment:'", entity, scope),
				fmt.Sprintf("kubectl logs %s -n %s --previous --tail=50 | grep -i -E 'database|postgres|mysql|oracle|connection|refused'", entity, scope),
				fmt.Sprintf("kubectl get svc,endpoints -n %s", scope),
			},
		}
	}

	return &AnalysisResult{
		Category:     fmt.Sprintf("BANCO DE DADOS (%s / Conexão Recusada)", dbType),
		Summary:      fmt.Sprintf("Container não consegue alcançar o %s.", dbType),
		RootCause:    fmt.Sprintf("O container '%s' tentou se conectar ao %s mas recebeu 'Connection Refused' ou timeout.", entity, dbType),
		SuggestedFix: "Certifique-se de que o container do banco está em execução e na mesma Docker Network que a aplicação.",
		ActionSteps: []string{
			"1. Verifique se o container do banco está ativo: docker ps",
			"2. Verifique as redes Docker conectadas: docker network ls",
			fmt.Sprintf("3. Inspecione as variáveis de conexão: docker inspect %s --format '{{range .Config.Env}}{{println .}}{{end}}'", entity),
		},
		SuggestedCommands: []string{
			fmt.Sprintf("docker logs --tail 50 %s | grep -i -E 'database|postgres|mysql|connection'", entity),
			"docker ps -a",
		},
	}
}

func analyzeRedisFailure(isK8s bool, scope, entity string) *AnalysisResult {
	return &AnalysisResult{
		Category:     "CACHE / REDIS (Indisponibilidade ou Timeout)",
		Summary:      "Aplicação falhou ao conectar ao servidor de cache Redis na porta 6379.",
		RootCause:    fmt.Sprintf("O workload '%s' depende de uma instância Redis que está inativa, reiniciando ou com conexões esgotadas.", entity),
		SuggestedFix: "Verifique o status do serviço/pod do Redis, valide as credenciais de autenticação (REDIS_PASSWORD) e confirme a acessibilidade de rede.",
		ActionSteps: []string{
			fmt.Sprintf("1. Verifique se o Redis está em execução no namespace: kubectl get pods,svc -n %s | grep redis", scope),
			fmt.Sprintf("2. Valide os logs do pod da aplicação buscando o hostname de conexão: kubectl logs %s -n %s --previous --tail=40", entity, scope),
			"3. Verifique a cota de memória da instância Redis para descartar eviction de chaves ou OOM.",
		},
		SuggestedCommands: []string{
			fmt.Sprintf("kubectl get pods,svc -A | grep -i redis"),
			fmt.Sprintf("kubectl logs %s -n %s --previous --tail=50 | grep -i redis", entity, scope),
		},
	}
}

func analyzeDNSFailure(isK8s bool, scope, entity string) *AnalysisResult {
	return &AnalysisResult{
		Category:     "REDE / DNS (Falha na Resolução de Nomes de Host)",
		Summary:      "Container não conseguiu resolver o hostname de um serviço interno ou externo.",
		RootCause:    fmt.Sprintf("Ocorreu falha de resolução DNS ('no such host' ou 'lookup i/o timeout') dentro do container '%s'. Possíveis causas: CoreDNS instável, endpoint digitado incorretamente nas variáveis de ambiente ou problema no resolv.conf.", entity),
		SuggestedFix: "Verifique a integridade dos pods do CoreDNS no cluster e valide o formato dos hostnames (serviços internos devem usar o padrão <servico>.<namespace>.svc.cluster.local).",
		ActionSteps: []string{
			"1. Inspecione os pods do CoreDNS no cluster: kubectl get pods -n kube-system -l k8s-app=kube-dns",
			"2. Verifique se os logs do CoreDNS reportam erros: kubectl logs -n kube-system -l k8s-app=kube-dns --tail=50",
			fmt.Sprintf("3. Revise as URLs configuradas nas variáveis de ambiente do Pod '%s'.", entity),
		},
		SuggestedCommands: []string{
			"kubectl get pods -n kube-system -l k8s-app=kube-dns",
			"kubectl logs -n kube-system -l k8s-app=kube-dns --tail=50",
			fmt.Sprintf("kubectl describe pod %s -n %s | grep -A 8 'Environment:'", entity, scope),
		},
	}
}

func analyzeAuthOrPermission(isK8s bool, scope, entity, logsLower string) *AnalysisResult {
	isFS := strings.Contains(logsLower, "permission denied")
	if isFS {
		return &AnalysisResult{
			Category:     "PERMISSÃO / SISTEMA DE ARQUIVOS (Permission Denied)",
			Summary:      "Processo dentro do container não possui permissão para ler ou escrever em arquivo/volume.",
			RootCause:    fmt.Sprintf("O usuário sob o qual o processo executa (UID/GID) não tem permissões suficientes no diretório ou volume montado no Pod '%s'.", entity),
			SuggestedFix: "Ajuste o 'securityContext.fsGroup' ou 'securityContext.runAsUser' no manifesto para conceder acesso ao UID correto, ou altere as permissões do PersistentVolume.",
			ActionSteps: []string{
				fmt.Sprintf("1. Verifique as configurações de SecurityContext: kubectl describe pod %s -n %s | grep -A 6 'Security Context'", entity, scope),
				fmt.Sprintf("2. Identifique qual arquivo/diretório gerou Permission Denied nos logs anteriores: kubectl logs %s -n %s --previous --tail=50", entity, scope),
				"3. Se utilizar PV/PVC, adicione o 'fsGroup' correspondente na spec do pod.",
			},
			SuggestedCommands: []string{
				fmt.Sprintf("kubectl logs %s -n %s --previous --tail=50 | grep -i 'permission denied'", entity, scope),
				fmt.Sprintf("kubectl describe pod %s -n %s | grep -A 10 'Mounts:'", entity, scope),
			},
		}
	}

	return &AnalysisResult{
		Category:     "AUTENTICAÇÃO (401 Unauthorized / Token Expirado)",
		Summary:      "Aplicação rejeitada ao se comunicar com API externa ou serviço de autenticação.",
		RootCause:    fmt.Sprintf("Credenciais, chaves de API ou tokens JWT configurados no Pod '%s' são inválidos ou expiraram.", entity),
		SuggestedFix: "Atualize os Secrets do Kubernetes com credenciais ou tokens válidos e realize o rollout do deployment.",
		ActionSteps: []string{
			fmt.Sprintf("1. Verifique os Secrets referenciados pelo Pod: kubectl describe pod %s -n %s | grep -A 5 'Secret'", entity, scope),
			"2. Renove o token ou credencial expirada no Secret correspondente.",
			fmt.Sprintf("3. Reinicie o deployment para carregar as novas credenciais: kubectl rollout restart deployment/<nome> -n %s", scope),
		},
		SuggestedCommands: []string{
			fmt.Sprintf("kubectl get secrets -n %s", scope),
			fmt.Sprintf("kubectl logs %s -n %s --previous --tail=50 | grep -i -E 'unauthorized|forbidden|401|403|token'", entity, scope),
		},
	}
}

func analyzeConfigFailure(isK8s bool, scope, entity, logsLower string) *AnalysisResult {
	return &AnalysisResult{
		Category:     "CONFIGURAÇÃO (Variável de Ambiente ou Arquivo Ausente)",
		Summary:      "Aplicação abortou durante a inicialização por falta de variável obrigatória ou arquivo de configuração.",
		RootCause:    fmt.Sprintf("O processo em '%s' iniciou mas não localizou um parâmetro obrigatório (env var ou config file) necessário para subir.", entity),
		SuggestedFix: "Verifique os ConfigMaps e Secrets associados e garanta que todas as variáveis requeridas pelo framework estejam preenchidas.",
		ActionSteps: []string{
			fmt.Sprintf("1. Localize nos logs a variável ou arquivo faltante: kubectl logs %s -n %s --previous --tail=40", entity, scope),
			fmt.Sprintf("2. Inspecione os ConfigMaps do namespace: kubectl get cm -n %s", scope),
			"3. Adicione a variável faltante no ConfigMap ou na spec do deployment.",
		},
		SuggestedCommands: []string{
			fmt.Sprintf("kubectl logs %s -n %s --previous --tail=50 | grep -i -E 'not set|missing|required|config'", entity, scope),
			fmt.Sprintf("kubectl get cm,secret -n %s", scope),
		},
	}
}

func analyzePortConflict(isK8s bool, scope, entity string) *AnalysisResult {
	return &AnalysisResult{
		Category:     "REDE (Porta já em uso / EADDRINUSE)",
		Summary:      "Falha de binding de porta — porta de rede já está ocupada por outro processo.",
		RootCause:    fmt.Sprintf("A aplicação no container '%s' tentou escutar em uma porta que já estava alocada no mesmo namespace de rede.", entity),
		SuggestedFix: "Altere a porta de escuta da aplicação nas configurações ou remova processos zumbis que possam ter permanecido ativos.",
		ActionSteps: []string{
			fmt.Sprintf("1. Identifique qual porta gerou conflito: kubectl logs %s -n %s --previous --tail=30 | grep -i 'address already in use'", entity, scope),
			"2. Verifique se outro container no mesmo Pod compartilha a mesma porta.",
			"3. Ajuste a variável de porta (PORT, SERVER_PORT, etc.) para uma porta livre.",
		},
		SuggestedCommands: []string{
			fmt.Sprintf("kubectl logs %s -n %s --previous --tail=40 | grep -i -E 'bind|port|address'", entity, scope),
		},
	}
}

func analyzePanicOrException(isK8s bool, scope, entity, logsLower string) *AnalysisResult {
	return &AnalysisResult{
		Category:     "APLICAÇÃO (Panic / Exceção Não Tratada em Runtime)",
		Summary:      "Encerramento abrupto devido a erro de código ou crash interno da aplicação.",
		RootCause:    fmt.Sprintf("O processo no workload '%s' disparou um panic ou exceção não tratada (como NullPointer, NilPointer ou Segfault) que interrompeu o runtime.", entity),
		SuggestedFix: "Analise o stacktrace capturado nos logs e encaminhe para a equipe de desenvolvimento com a versão da imagem e o ponto de falha no código.",
		ActionSteps: []string{
			fmt.Sprintf("1. Extraia o stacktrace completo dos logs: kubectl logs %s -n %s --previous --tail=80", entity, scope),
			fmt.Sprintf("2. Verifique a tag/commit exata da imagem: kubectl describe pod %s -n %s | grep -i 'Image:'", entity, scope),
			"3. Se a falha ocorreu após um deploy recente, avalie realizar rollback para a versão estável anterior.",
		},
		SuggestedCommands: []string{
			fmt.Sprintf("kubectl logs %s -n %s --previous --tail=100", entity, scope),
			fmt.Sprintf("kubectl describe pod %s -n %s | grep -i 'Image:'", entity, scope),
		},
	}
}

func analyzeGenericCrash(isK8s bool, scope, entity string, event *domain.IncidentEvent) *AnalysisResult {
	if isK8s {
		return &AnalysisResult{
			Category:     fmt.Sprintf("ESTABILIDADE (CrashLoopBackOff / Código de Saída %d)", event.ExitCode),
			Summary:      fmt.Sprintf("Pod em loop de reinicialização contínua com exit code %d.", event.ExitCode),
			RootCause:    fmt.Sprintf("O processo no Pod '%s' (namespace '%s') encerrou com código %d. O Kubernetes aplicou política de backoff para reiniciar o container.", entity, scope, event.ExitCode),
			SuggestedFix: "Analise os logs da execução anterior com '--previous', valide as probes de Liveness/Readiness e confira as variáveis de ambiente.",
			ActionSteps: []string{
				fmt.Sprintf("1. Inspecione os eventos do Pod para verificar erros de probe ou volume: kubectl describe pod %s -n %s", entity, scope),
				fmt.Sprintf("2. Visualize a saída de logs do container que falhou: kubectl logs %s -n %s --previous --tail=50", entity, scope),
				"3. Se houver falha de liveness probe, avalie aumentar o 'initialDelaySeconds' ou 'timeoutSeconds'.",
			},
			SuggestedCommands: []string{
				fmt.Sprintf("kubectl describe pod %s -n %s", entity, scope),
				fmt.Sprintf("kubectl logs %s -n %s --previous --tail=50", entity, scope),
			},
		}
	}

	return &AnalysisResult{
		Category:     fmt.Sprintf("DOCKER (Container Parado / Código de Saída %d)", event.ExitCode),
		Summary:      fmt.Sprintf("Container Docker finalizou inesperadamente com código %d.", event.ExitCode),
		RootCause:    fmt.Sprintf("O processo principal no container '%s' encerrou com código de saída %d.", entity, event.ExitCode),
		SuggestedFix: "Verifique os logs de inicialização e confira os argumentos de entrada (entrypoint e cmd).",
		ActionSteps: []string{
			fmt.Sprintf("1. Analise os últimos logs do container: docker logs --tail 50 %s", entity),
			fmt.Sprintf("2. Inspecione o status detalhado: docker inspect %s", entity),
		},
		SuggestedCommands: []string{
			fmt.Sprintf("docker logs --tail 60 %s", entity),
			fmt.Sprintf("docker inspect %s --format 'ExitCode: {{.State.ExitCode}} | Error: {{.State.Error}}'", entity),
		},
	}
}

func matchAny(text string, patterns ...string) bool {
	for _, p := range patterns {
		if strings.Contains(p, ".*") {
			if matched, _ := regexp.MatchString("(?i)"+p, text); matched {
				return true
			}
		} else {
			if strings.Contains(text, strings.ToLower(p)) {
				return true
			}
		}
	}
	return false
}
