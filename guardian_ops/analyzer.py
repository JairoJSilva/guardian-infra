from typing import Optional
from guardian_ops.models import IncidentEvent, AnalysisResult, FailureSource, IncidentPriority
from guardian_ops.templates import build_jira_description

class GuardianAnalyzer:
    """
    Motor Analítico do GuardianOps:
    Interpreta eventos de falhas em Pods do Kubernetes e Pipelines de CI/CD,
    determina a causa raiz provável e gera recomendações estruturadas para o operador humano.
    MODO DE OPERAÇÃO: 100% Analítico e Observacional (Sem intervenção automática).
    """

    def analyze(self, event: IncidentEvent, reference_code: str = "OPS-GUARDIAN") -> AnalysisResult:
        if event.source == FailureSource.KUBERNETES_POD:
            return self._analyze_pod_failure(event, reference_code)
        elif event.source == FailureSource.PIPELINE_CI_CD:
            return self._analyze_pipeline_failure(event, reference_code)
        else:
            return self._analyze_generic_failure(event, reference_code)

    def _analyze_pod_failure(self, event: IncidentEvent, reference_code: str) -> AnalysisResult:
        pod_name = event.identifier
        namespace = event.namespace_or_project
        container = event.container_or_stage or "app"
        failure_type = event.failure_type.upper()
        restarts = event.restart_count or 1
        exit_code = event.exit_code

        # 1. Caso OOMKilled (Out Of Memory)
        if "OOMKILLED" in failure_type or exit_code == 137:
            priority = IncidentPriority.ALTA
            title = f"Incidente: Pod '{pod_name}' encerrado por OOMKilled (Namespace: {namespace})"
            category = "Kubernetes Workload / Limites de Recursos (Memory)"
            root_cause = (
                f"O container *{container}* no pod *{pod_name}* excedeu o limite de memória alocado "
                f"(`resources.limits.memory`) ou sofreu um vazamento de memória (_memory leak_). "
                f"O kernel do nó disparou o processo Linux OOM-Killer finalizando o container com Exit Code 137. "
                f"Contagem de reinicializações detectada: *{restarts}*."
            )
            remediation = f"""1. Inspecionar o consumo de memória histórico e atual do pod:
{{code:bash}}
kubectl top pod {pod_name} -n {namespace} --containers
{{code}}
2. Verificar os limites configurados no Deployment:
{{code:bash}}
kubectl get deployment -n {namespace} -l app={container} -o jsonpath='{{.items[*].spec.template.spec.containers[*].resources}}'
{{code}}
3. Sugestão de Ajuste Declarativo: Aumentar o `resources.limits.memory` e `resources.requests.memory` em pelo menos 25% a 50% no manifesto do repositório Git (GitOps).
4. Analisar os logs da aplicação antes do encerramento para identificar consultas pesadas ou rotinas em batch consumindo RAM excessiva."""

            acceptance = (
                f"* Pod *{pod_name}* estabilizado em status {{Running}} com {{1/1 Ready}}.\n"
                f"* Consumo de memória dentro da faixa segura (< 80% do novo limite estipulado)."
            )
            rollback = (
                "Caso o aumento de memória cause pressão no nó (_Node Memory Pressure_), reverta o manifesto no Git "
                f"ou execute: {{code:bash}}kubectl rollout undo deployment/<deployment-name> -n {namespace}{{code}}"
            )

        # 2. Caso CrashLoopBackOff
        elif "CRASHLOOP" in failure_type:
            priority = IncidentPriority.ALTA
            title = f"Incidente: Pod '{pod_name}' em CrashLoopBackOff (Namespace: {namespace})"
            category = "Kubernetes Workload / Estabilidade de Aplicação"
            root_cause = (
                f"O processo principal do container *{container}* está falhando imediatamente após a inicialização "
                f"(Exit Code: {exit_code if exit_code is not None else 'Desconhecido'}). "
                f"O Kubelet tentou reiniciar o container repetidamente ({restarts} reinicializações), "
                "ativando a política de espera exponencial (_back-off delay_). "
                "Causas frequentes: Variáveis de ambiente/Secrets ausentes, falha de conexão com banco de dados/cache ou exceção fatal não tratada."
            )
            remediation = f"""1. Visualizar os logs da última execução do container antes de cair:
{{code:bash}}
kubectl logs {pod_name} -n {namespace} -c {container} --previous --tail=100
{{code}}
2. Inspecionar os eventos recentes do ciclo de vida do Pod:
{{code:bash}}
kubectl describe pod {pod_name} -n {namespace}
{{code}}
3. Validar se os Secrets e ConfigMaps referenciados pelo Deployment existem no namespace:
{{code:bash}}
kubectl get configmaps,secrets -n {namespace}
{{code}}
4. Corrigir a causa-raiz identificada nos logs (ex: string de conexão, credencial expirada) via repositório de código/infraestrutura."""

            acceptance = (
                f"* Pod *{pod_name}* atinge status {{Running}} e passa em todas as checagens de liveness/readiness.\n"
                f"* Contador de reinicializações cessa o incremento."
            )
            rollback = (
                f"Reverter a última alteração no repositório de configuração ou aplicar rollback do Deployment:\n"
                f"{{code:bash}}kubectl rollout undo deployment/<deployment-name> -n {namespace}{{code}}"
            )

        # 3. Caso ImagePullBackOff / ErrImagePull
        elif "IMAGE" in failure_type:
            priority = IncidentPriority.MEDIA
            title = f"Falha de Imagem: Pod '{pod_name}' em ImagePullBackOff (Namespace: {namespace})"
            category = "Kubernetes Workload / Container Registry"
            root_cause = (
                f"O Kubelet não conseguiu baixar a imagem de container definida para o pod *{pod_name}*. "
                "Causas frequentes: A tag da imagem não existe no Registry (ex: erro de digitação no pipeline), "
                "falha de autenticação do Kubernetes com o Container Registry (`imagePullSecrets` ausente ou expirado), "
                "ou bloqueio de rede/firewall entre os nós e o registry."
            )
            remediation = f"""1. Obter a imagem exata e o motivo da recusa:
{{code:bash}}
kubectl get pod {pod_name} -n {namespace} -o jsonpath='{{.spec.containers[*].image}}'
kubectl describe pod {pod_name} -n {namespace} | grep -A 5 -i "Failed to pull image"
{{code}}
2. Validar se a imagem e a tag correspondente foram publicadas no Container Registry (GitLab/Harbor/ECR/DockerHub).
3. Verificar a validade do segredo de pull no namespace:
{{code:bash}}
kubectl get secret regcred -n {namespace}
{{code}}
4. Ajustar o manifesto com a tag de imagem correta e aplicar novamente."""

            acceptance = (
                f"* Imagem baixada com sucesso pelo nó.\n"
                f"* Pod *{pod_name}* inicia o container normalmente em status {{Running}}."
            )
            rollback = (
                f"Ajustar o manifesto para apontar para a última tag de imagem estável e testada:\n"
                f"{{code:bash}}kubectl rollout undo deployment/<deployment-name> -n {namespace}{{code}}"
            )

        # 4. Caso Falha de Probes (Liveness/Readiness)
        elif "PROBE" in failure_type:
            priority = IncidentPriority.MEDIA
            title = f"Falha de Probe: Pod '{pod_name}' não responde à checagem de saúde (Namespace: {namespace})"
            category = "Kubernetes Workload / Health Probes"
            root_cause = (
                f"O container *{container}* falhou reiteradamente nas checagens de saúde configuradas (Liveness ou Readiness Probe). "
                "O endpoint de saúde da aplicação pode estar respondendo com erro HTTP (5xx), sofrendo timeout devido a alto processamento, "
                "ou a aplicação demorou mais para inicializar do que o tempo definido em `initialDelaySeconds`."
            )
            remediation = f"""1. Consultar os eventos do Pod para verificar qual probe falhou e a resposta HTTP/erro retornado:
{{code:bash}}
kubectl describe pod {pod_name} -n {namespace} | grep -E "(Liveness|Readiness|Unhealthy)"
{{code}}
2. Testar o endpoint de saúde de dentro do cluster:
{{code:bash}}
kubectl exec -ti {pod_name} -n {namespace} -c {container} -- curl -Iv http://localhost:8080/health
{{code}}
3. Caso a aplicação demore para subir, ajustar o `initialDelaySeconds` ou `periodSeconds` no manifesto."""

            acceptance = (
                f"* Probes respondendo com código HTTP 200 dentro do intervalo esperado.\n"
                f"* Endpoints do Service associado apontando para o Pod como {{Ready}}."
            )
            rollback = (
                f"Reverter alterações recentes na configuração de probes:\n"
                f"{{code:bash}}kubectl rollout undo deployment/<deployment-name> -n {namespace}{{code}}"
            )

        # 5. Caso Genérico de Pod
        else:
            priority = IncidentPriority.MEDIA
            title = f"Alerta de Workload: Falha no Pod '{pod_name}' ({event.failure_type})"
            category = "Kubernetes Workload / Diagnóstico Geral"
            root_cause = (
                f"Detectada anomalia no ciclo de vida do pod *{pod_name}* no namespace *{namespace}*. "
                f"Status/Motivo: *{event.failure_type}*. Detalhes: {event.error_message or 'Verificar logs anexos'}."
            )
            remediation = f"""1. Inspecionar o estado do pod e eventos do nó:
{{code:bash}}
kubectl describe pod {pod_name} -n {namespace}
kubectl logs {pod_name} -n {namespace} --tail=80
{{code}}
2. Avaliar o status do nó onde o pod está agendado:
{{code:bash}}
kubectl get nodes -o wide
{{code}}"""
            acceptance = f"* Pod *{pod_name}* restaurado para operação regular."
            rollback = "Reverter para revisão de deployment anterior se aplicável."

        # Monta tabela de ativos impactados
        impacted_table = f"""|| Namespace || Nome do Pod || Container || Motivo da Falha || Reinicializações ||
| {namespace} | {{{{ {pod_name} }}}} | {{{{ {container} }}}} | *{event.failure_type}* | {restarts} |"""

        # Procedimento com aviso institucional de leitura estrita
        procedure_text = f"""> *AVISO DE GOVERNANÇA:* O *GuardianOps* atua exclusivamente em modo de *SOMENTE-LEITURA* (Read-Only). Nenhuma modificação automática foi realizada no cluster. A equipe responsável deve executar o roteiro abaixo manualmente:

{remediation}"""

        description = build_jira_description(
            reference=reference_code,
            date_str=event.timestamp,
            environment=event.environment,
            title=title,
            issue_type="Solicitação de serviço",
            priority=priority.value,
            category=category,
            root_cause_explanation=root_cause,
            evidence_snippet=event.logs_snippet or event.error_message,
            impacted_assets_table=impacted_table,
            remediation_procedure=procedure_text,
            acceptance_criteria=acceptance,
            rollback_plan=rollback
        )

        return AnalysisResult(
            summary=title,
            priority=priority,
            category=category,
            issue_type="Solicitação de serviço",
            root_cause=root_cause,
            impacted_asset_summary=f"Pod {pod_name} no namespace {namespace}",
            remediation_procedure=procedure_text,
            acceptance_criteria=acceptance,
            rollback_plan=rollback,
            jira_description=description,
            labels=["gerado-por-ia", "guardian-ops", "k8s-failure"]
        )

    def _analyze_pipeline_failure(self, event: IncidentEvent, reference_code: str) -> AnalysisResult:
        pipeline_id = event.identifier
        project = event.namespace_or_project
        stage = event.container_or_stage or "build"
        failure_type = event.failure_type

        priority = IncidentPriority.ALTA
        title = f"Falha de Pipeline CI/CD: Projeto '{project}' no estágio '{stage}' (Pipeline #{pipeline_id})"
        category = "CI-CD / Esteira de Integração Contínua"

        root_cause = (
            f"A execução da esteira de CI/CD para o projeto *{project}* falhou durante a execução do job/estágio *{stage}*. "
            f"Erro reportado: *{event.error_message or failure_type}*. "
            "A falha bloqueou o fluxo de integração/entrega contínua, impedindo a validação ou implantação das alterações."
        )

        remediation = f"""> *AVISO DE GOVERNANÇA:* O *GuardianOps* atua exclusivamente em modo de *SOMENTE-LEITURA*. O job da pipeline foi interrompido e requer correção manual do desenvolvedor/engenheiro de DevOps responsável:

1. Acessar os detalhes e log completo da execução no servidor de CI/CD:
   * Projeto: *{project}*
   * Estágio com falha: *{stage}*
   * Identificador da Pipeline: *#{pipeline_id}*
2. Analisar o trecho do log anexado neste chamado na seção de evidências.
3. Se for falha de compilação/testes: reproduzir o comando localmente no ambiente de desenvolvimento antes de subir novo commit.
4. Se for falha de conectividade ou credencial (ex: runner sem acesso ao Docker daemon ou Kubernetes): acionar a equipe de Cloud/DevOps.
5. Após o ajuste, submeter um novo commit ou reexecutar o job com falha (_Retry_)."""

        impacted_table = f"""|| Projeto / Repositório || Pipeline ID || Estágio / Job || Motivo ||
| {project} | {{{{ #{pipeline_id} }}}} | {{{{ {stage} }}}} | *{failure_type}* |"""

        acceptance = (
            f"* Job *{stage}* reexecutado com código de retorno 0 (Sucesso).\n"
            f"* Pipeline do projeto *{project}* concluída integralmente (Status: {{Passed}})."
        )

        rollback = (
            "Caso a alteração introduzida na branch tenha quebrado o fluxo principal, execute `git revert` do commit faltoso na branch de trabalho."
        )

        description = build_jira_description(
            reference=reference_code,
            date_str=event.timestamp,
            environment=event.environment,
            title=title,
            issue_type="Solicitação de serviço",
            priority=priority.value,
            category=category,
            root_cause_explanation=root_cause,
            evidence_snippet=event.logs_snippet or event.error_message,
            impacted_assets_table=impacted_table,
            remediation_procedure=remediation,
            acceptance_criteria=acceptance,
            rollback_plan=rollback
        )

        return AnalysisResult(
            summary=title,
            priority=priority,
            category=category,
            issue_type="Solicitação de serviço",
            root_cause=root_cause,
            impacted_asset_summary=f"Pipeline #{pipeline_id} no projeto {project}",
            remediation_procedure=remediation,
            acceptance_criteria=acceptance,
            rollback_plan=rollback,
            jira_description=description,
            labels=["gerado-por-ia", "guardian-ops", "pipeline-failure"]
        )

    def _analyze_generic_failure(self, event: IncidentEvent, reference_code: str) -> AnalysisResult:
        priority = IncidentPriority.MEDIA
        title = f"Incidente Operacional: Falha em '{event.identifier}'"
        category = "Operações / Infraestrutura"
        root_cause = f"Falha detectada no componente *{event.identifier}*: {event.error_message}"
        remediation = f"Inspecionar logs e status de {event.identifier}."
        impacted_table = f"|| Alvo || Namespace/Origem || Falha ||\n| {event.identifier} | {event.namespace_or_project} | {event.failure_type} |"
        acceptance = "Componente operando normalmente."
        rollback = "Reverter alteração anterior."

        description = build_jira_description(
            reference=reference_code,
            date_str=event.timestamp,
            environment=event.environment,
            title=title,
            issue_type="Solicitação de serviço",
            priority=priority.value,
            category=category,
            root_cause_explanation=root_cause,
            evidence_snippet=event.logs_snippet or event.error_message,
            impacted_assets_table=impacted_table,
            remediation_procedure=remediation,
            acceptance_criteria=acceptance,
            rollback_plan=rollback
        )

        return AnalysisResult(
            summary=title,
            priority=priority,
            category=category,
            issue_type="Solicitação de serviço",
            root_cause=root_cause,
            impacted_asset_summary=event.identifier,
            remediation_procedure=remediation,
            acceptance_criteria=acceptance,
            rollback_plan=rollback,
            jira_description=description,
            labels=["gerado-por-ia", "guardian-ops"]
        )
