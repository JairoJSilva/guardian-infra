# -*- coding: utf-8 -*-
"""
01. DevOps & Cloud Native Agent
================================
Identificador : agent-devops-cloudnative
Responsável   : Infraestrutura como Código (IaC), Containers, Orquestração e Pipelines.
Escopo        : Terraform, OpenTofu, Docker, Docker Compose, Kubernetes (Helm, Manifests),
                GitHub Actions, GitLab CI, CrashLoopBackOff, OOMKilled, ImagePullBackOff.
"""
from typing import Dict, Any

from guardian_ops.agents.base_agent import BaseAgent, AgentDiagnosis, Severity, DefectLayer


class DevOpsCloudNativeAgent(BaseAgent):
    """
    Agente especializado em falhas de infraestrutura, containers e orquestração.

    Cobre:
    - Kubernetes: CrashLoopBackOff, OOMKilled, ImagePullBackOff, Probe failures,
                  PVC Pending, Secret/ConfigMap ausentes, DNS/rede
    - Docker: Healthcheck failed, container exit codes, volume mounts
    - IaC: Terraform state lock, plan/apply errors, Ansible playbook failures
    - CI/CD: Pipeline stage failures, Docker build errors, registry auth issues
    """
    AGENT_ID   = "agent-devops-cloudnative"
    AGENT_NAME = "DevOps & Cloud Native Agent"

    # Palavras-chave que indicam que este agente deve ser acionado
    _K8S_TRIGGERS   = {"CRASHLOOP", "OOMKILLED", "IMAGEPULL", "PROBE", "PENDING", "EVICTED",
                        "KUBECTL", "KUBERNETES", "K8S", "POD", "NAMESPACE", "HELM", "INGRESS"}
    _DOCKER_TRIGGERS = {"DOCKER", "CONTAINER", "HEALTHCHECK", "COMPOSE", "REGISTRY", "DOCKERFILE", "EXITCODE", "STOPPED"}
    _IAC_TRIGGERS    = {"TERRAFORM", "OPENTOFU", "ANSIBLE", "PULUMI", "STATE LOCK", "PLAN",
                        "APPLY", "DESTROY", "PLAYBOOK", "INVENTORY"}
    _CICD_TRIGGERS   = {"PIPELINE", "GITHUB ACTIONS", "GITLAB CI", "JENKINS", "BUILD", "RUNNER",
                        "WORKFLOW", "DEPLOY", "RELEASE", "ARTIFACT"}

    def can_handle(self, payload: Dict[str, Any]) -> bool:
        source = payload.get("source", "").upper()
        failure = payload.get("failure_type", "").upper()
        combined = f"{source} {failure}"
        all_triggers = (
            self._K8S_TRIGGERS | self._DOCKER_TRIGGERS |
            self._IAC_TRIGGERS | self._CICD_TRIGGERS
        )
        return any(t in combined for t in all_triggers)

    def analyze(self, payload: Dict[str, Any]) -> AgentDiagnosis:
        failure_type = payload.get("failure_type", "").upper()
        component    = payload.get("component", payload.get("identifier", "unknown"))
        namespace    = payload.get("namespace", payload.get("namespace_or_project", "default"))
        container    = payload.get("container", payload.get("container_or_stage", "app"))
        logs         = payload.get("logs", payload.get("logs_snippet", ""))
        restarts     = int(payload.get("restart_count", 0) or 0)
        exit_code    = payload.get("exit_code")
        image        = payload.get("details", {}).get("image", "")

        # Roteamento para o diagnóstico especializado
        if "OOMKILLED" in failure_type or exit_code == 137:
            return self._diagnose_oomkilled(component, namespace, container, restarts, logs)
        elif "CRASHLOOP" in failure_type:
            return self._diagnose_crashloop(component, namespace, container, restarts, exit_code, logs)
        elif "IMAGE" in failure_type:
            return self._diagnose_imagepull(component, namespace, image, logs)
        elif "PROBE" in failure_type or "LIVENESS" in failure_type or "READINESS" in failure_type:
            return self._diagnose_probe(component, namespace, container, logs)
        elif "TERRAFORM" in failure_type or "TOFU" in failure_type:
            return self._diagnose_iac_terraform(component, logs)
        elif "ANSIBLE" in failure_type or "PLAYBOOK" in failure_type:
            return self._diagnose_iac_ansible(component, logs)
        elif "PIPELINE" in failure_type or "BUILD" in failure_type:
            return self._diagnose_pipeline(component, container, logs)
        elif "DOCKER" in failure_type or "HEALTH" in failure_type or "EXIT" in failure_type or "STOP" in failure_type:
            return self._diagnose_docker(component, container, exit_code, restarts, logs)
        else:
            return self._diagnose_generic_infra(component, namespace, failure_type, logs)

    # -------------------------------------------------------------------------
    # Diagnósticos Kubernetes
    # -------------------------------------------------------------------------

    def _diagnose_oomkilled(self, pod, namespace, container, restarts, logs) -> AgentDiagnosis:
        fix = f"""\
# 1. Verificar consumo atual de memória:
kubectl top pod {pod} -n {namespace} --containers

# 2. Inspecionar os limites configurados no Deployment:
kubectl get deployment -n {namespace} -o jsonpath='{{.items[*].spec.template.spec.containers[*].resources}}'

# 3. Patch declarativo — aumentar limits.memory em 50%:
kubectl patch deployment <deployment-name> -n {namespace} --type='json' \\
  -p='[{{"op":"replace","path":"/spec/template/spec/containers/0/resources/limits/memory","value":"512Mi"}}]'

# 4. Verificar via GitOps no manifesto (recomendado):
# resources:
#   requests:
#     memory: "256Mi"
#   limits:
#     memory: "512Mi"  # <- aumentar este valor"""

        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.CRITICAL if restarts >= 5 else Severity.HIGH,
            defect_layer=DefectLayer.ORCHESTRATION,
            root_cause=(
                f"O container *{container}* no pod *{pod}* (namespace: {namespace}) excedeu o limite de "
                f"memória configurado em `resources.limits.memory`. O kernel Linux acionou o OOM-Killer "
                f"encerrando o processo com Exit Code 137. Reinicializações: {restarts}. "
                "Causas típicas: memory leak na aplicação, carga de dados excessiva sem paginação, "
                "ou limite de memória subdimensionado para o workload atual."
            ),
            affected_component=f"Pod/{pod} no namespace {namespace}",
            fix_code=fix,
            fix_description=(
                "Aumentar resources.limits.memory e resources.requests.memory no manifesto do Deployment. "
                "Recomenda-se aumentar em no mínimo 50% e monitorar com kubectl top após o ajuste."
            ),
            repro_steps=[
                f"kubectl describe pod {pod} -n {namespace}",
                f"kubectl logs {pod} -n {namespace} --previous --tail=100",
                f"kubectl top pod {pod} -n {namespace} --containers",
            ],
            validation_steps=[
                f"Pod {pod} estável em status Running com 1/1 Ready por > 10 minutos",
                f"Consumo de memória < 80% do novo limite após o ajuste",
                "Ausência de eventos OOMKilled nos próximos 30 minutos",
            ],
        )

    def _diagnose_crashloop(self, pod, namespace, container, restarts, exit_code, logs) -> AgentDiagnosis:
        fix = f"""\
# 1. Logs da última execução antes de cair:
kubectl logs {pod} -n {namespace} -c {container} --previous --tail=100

# 2. Eventos do ciclo de vida do Pod:
kubectl describe pod {pod} -n {namespace}

# 3. Validar Secrets e ConfigMaps no namespace:
kubectl get configmaps,secrets -n {namespace}

# 4. Verificar variáveis de ambiente do container:
kubectl exec {pod} -n {namespace} -- env | grep -v PASSWORD | grep -v SECRET

# 5. Rollback do Deployment (se o erro surgiu após um deploy):
kubectl rollout undo deployment/<deployment-name> -n {namespace}"""

        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.CRITICAL if restarts >= 8 else Severity.HIGH,
            defect_layer=DefectLayer.ORCHESTRATION,
            root_cause=(
                f"O processo principal do container *{container}* no pod *{pod}* falha imediatamente "
                f"após a inicialização (Exit Code: {exit_code if exit_code is not None else 'Desconhecido'}). "
                f"O Kubelet aplicou a política de back-off exponencial após {restarts} reinicializações. "
                "Causas frequentes: Secrets/ConfigMaps ausentes, string de conexão incorreta com banco de dados, "
                "exceção fatal não tratada no startup da aplicação, ou incompatibilidade de versão de dependência."
            ),
            affected_component=f"Pod/{pod} container/{container} namespace/{namespace}",
            fix_code=fix,
            fix_description=(
                "Inspecionar logs da última execução com --previous flag para capturar o stack trace antes "
                "da reinicialização. Corrigir a causa raiz identificada (Secret, ConfigMap, conexão) "
                "via repositório Git e re-aplicar o manifesto."
            ),
            repro_steps=[
                f"kubectl get pod {pod} -n {namespace}",
                f"kubectl logs {pod} -n {namespace} --previous --tail=100",
                f"kubectl describe pod {pod} -n {namespace} | grep -A10 'Last State'",
            ],
            validation_steps=[
                f"Pod {pod} atinge STATUS=Running sem reinicialização por > 5 minutos",
                "Restart count cessa o incremento",
                "Liveness e Readiness probes respondem com sucesso",
            ],
        )

    def _diagnose_imagepull(self, pod, namespace, image, logs) -> AgentDiagnosis:
        fix = f"""\
# 1. Obter a imagem exata referenciada pelo pod:
kubectl get pod {pod} -n {namespace} -o jsonpath='{{.spec.containers[*].image}}'

# 2. Verificar o motivo da recusa pelo Kubelet:
kubectl describe pod {pod} -n {namespace} | grep -A5 "Failed to pull image"

# 3. Verificar validade do imagePullSecret:
kubectl get secret regcred -n {namespace}
kubectl get secret regcred -n {namespace} -o jsonpath='{{.data.\\.dockerconfigjson}}' | base64 -d

# 4. Recriar o secret de pull (se expirado):
kubectl create secret docker-registry regcred \\
  --docker-server=<registry-url> \\
  --docker-username=<user> \\
  --docker-password=<token> \\
  -n {namespace} --dry-run=client -o yaml | kubectl apply -f -

# 5. Corrigir a tag da imagem no manifesto e re-aplicar:
kubectl set image deployment/<deployment-name> {pod}={image}:latest -n {namespace}"""

        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.HIGH,
            defect_layer=DefectLayer.ORCHESTRATION,
            root_cause=(
                f"O Kubelet não conseguiu fazer pull da imagem `{image or 'referenciada'}` para o pod *{pod}*. "
                "Causas possíveis: tag de imagem inexistente no registry (erro de typo no pipeline de CI/CD), "
                "falha de autenticação (`imagePullSecrets` ausente, expirado ou com permissão revogada), "
                "bloqueio de rede/firewall entre os nós e o Container Registry."
            ),
            affected_component=f"Pod/{pod} image/{image or 'unknown'} namespace/{namespace}",
            fix_code=fix,
            fix_description=(
                "Verificar se a tag da imagem foi publicada no registry, renovar o imagePullSecret se necessário "
                "e corrigir a referência de imagem no manifesto do Deployment."
            ),
            repro_steps=[
                f"kubectl describe pod {pod} -n {namespace}",
                f"kubectl get events -n {namespace} --sort-by='.lastTimestamp' | tail -20",
            ],
            validation_steps=[
                "Imagem baixada com sucesso (sem evento ErrImagePull)",
                f"Pod {pod} inicia o container normalmente em status Running",
            ],
        )

    def _diagnose_probe(self, pod, namespace, container, logs) -> AgentDiagnosis:
        fix = f"""\
# 1. Identificar qual probe falhou e o erro retornado:
kubectl describe pod {pod} -n {namespace} | grep -E "(Liveness|Readiness|Unhealthy|probe)"

# 2. Testar o endpoint de saúde de dentro do cluster:
kubectl exec -ti {pod} -n {namespace} -c {container} -- curl -sv http://localhost:8080/health

# 3. Verificar se a aplicação responde na porta correta:
kubectl exec -ti {pod} -n {namespace} -- ss -tlnp

# 4. Ajustar initialDelaySeconds se aplicação demora para subir (patch):
kubectl patch deployment <deployment-name> -n {namespace} --type='json' \\
  -p='[{{"op":"replace","path":"/spec/template/spec/containers/0/livenessProbe/initialDelaySeconds","value":60}}]'"""

        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.HIGH,
            defect_layer=DefectLayer.ORCHESTRATION,
            root_cause=(
                f"O container *{container}* no pod *{pod}* falhou nas checagens de saúde configuradas "
                "(Liveness ou Readiness Probe). O endpoint de saúde pode estar respondendo com erro HTTP (5xx), "
                "sofrendo timeout por alta carga, ou a aplicação demora mais para inicializar do que "
                "o valor configurado em `initialDelaySeconds`."
            ),
            affected_component=f"Pod/{pod} container/{container} probe/health namespace/{namespace}",
            fix_code=fix,
            fix_description=(
                "Aumentar initialDelaySeconds para dar tempo à aplicação de inicializar, "
                "ou corrigir o endpoint de health check na aplicação para retornar HTTP 200."
            ),
            repro_steps=[
                f"kubectl describe pod {pod} -n {namespace} | grep -E '(Liveness|Readiness|Unhealthy)'",
                f"kubectl get events -n {namespace} | grep {pod}",
            ],
            validation_steps=[
                "Probe retorna HTTP 200 dentro do periodSeconds configurado",
                f"Endpoints do Service apontam para o Pod {pod} como Ready",
            ],
        )

    # -------------------------------------------------------------------------
    # Diagnósticos IaC
    # -------------------------------------------------------------------------

    def _diagnose_iac_terraform(self, component, logs) -> AgentDiagnosis:
        state_lock = "LOCK" in (logs or "").upper()
        if state_lock:
            fix = """\
# Terraform State Lock detectado — verificar e forçar unlock se necessário:

# 1. Verificar o estado do lock:
terraform force-unlock -force <LOCK_ID>

# 2. Verificar o backend do state (S3/GCS/Azure Blob/Terraform Cloud):
terraform state list

# 3. Se o lock foi causado por processo interrompido, verificar se o apply está realmente parado:
ps aux | grep terraform

# 4. Limpar o lock somente se o processo anterior foi confirmadamente encerrado:
# AWS S3: aws dynamodb delete-item --table-name <lock-table> --key '{"LockID":{"S":"<lock-id>"}}'"""
            root_cause = (
                f"O Terraform detectou um state lock ativo no componente *{component}*. "
                "Isso ocorre quando um `terraform apply` ou `terraform plan` anterior foi interrompido "
                "abruptamente (kill do processo, timeout de CI/CD, queda de rede), deixando o lock no backend. "
                "Nenhum novo apply pode ser executado até que o lock seja liberado."
            )
        else:
            fix = """\
# Terraform Plan/Apply com erro — procedimento de diagnóstico:

# 1. Revalidar a sintaxe dos arquivos .tf:
terraform validate

# 2. Formatar os arquivos:
terraform fmt -check -recursive

# 3. Verificar o plano antes de aplicar:
terraform plan -out=tfplan.bin

# 4. Inspecionar o plano gerado:
terraform show tfplan.bin

# 5. Aplicar apenas se o plano estiver correto:
terraform apply tfplan.bin"""
            root_cause = (
                f"Erro detectado na execução do Terraform para o componente *{component}*. "
                "Causas típicas: sintaxe inválida em arquivo .tf, provider desatualizado, "
                "recurso em estado inconsistente ou permissão IAM insuficiente para o apply."
            )

        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.HIGH,
            defect_layer=DefectLayer.INFRASTRUCTURE,
            root_cause=root_cause,
            affected_component=f"IaC/Terraform/{component}",
            fix_code=fix,
            fix_description="Procedimento para liberar state lock ou corrigir erro de plan/apply no Terraform.",
            repro_steps=[
                "terraform init",
                "terraform validate",
                "terraform plan (observar o erro exato)",
            ],
            validation_steps=[
                "terraform plan retorna sem erros",
                "terraform apply completa com sucesso",
                "State backend sem locks pendentes",
            ],
        )

    def _diagnose_iac_ansible(self, component, logs) -> AgentDiagnosis:
        fix = """\
# Ansible Playbook com falha — procedimento de diagnóstico:

# 1. Re-executar com verbose para ver o erro completo:
ansible-playbook site.yml -i inventory.ini -vvv

# 2. Testar conectividade com os hosts:
ansible all -i inventory.ini -m ping

# 3. Verificar sintaxe do playbook:
ansible-playbook site.yml --syntax-check

# 4. Executar apenas as tasks que falharam com tags específicas:
ansible-playbook site.yml -i inventory.ini --tags "deploy" --limit <host>

# 5. Verificar permissões do usuário de execução:
ansible all -i inventory.ini -m shell -a "whoami" --become"""

        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.MEDIUM,
            defect_layer=DefectLayer.INFRASTRUCTURE,
            root_cause=(
                f"Falha na execução do Ansible Playbook para o componente *{component}*. "
                "Causas comuns: falha de conectividade SSH com o host alvo, permissão sudo insuficiente, "
                "variável de inventário indefinida, módulo Ansible desatualizado ou handler com erro."
            ),
            affected_component=f"IaC/Ansible/{component}",
            fix_code=fix,
            fix_description="Executar com verbose (-vvv) para capturar o erro exato e verificar conectividade e permissões.",
            repro_steps=[
                "ansible all -i inventory.ini -m ping",
                "ansible-playbook site.yml --syntax-check",
                "ansible-playbook site.yml -vvv 2>&1 | tail -50",
            ],
            validation_steps=[
                "ansible-playbook retorna PLAY RECAP sem failed=",
                "Serviço no host alvo funcionando após o playbook",
            ],
        )

    def _diagnose_pipeline(self, project, stage, logs) -> AgentDiagnosis:
        fix = f"""\
# Pipeline CI/CD com falha no estágio '{stage}':

# 1. Reproduzir o build localmente:
docker build -t {project}:local .

# 2. Verificar variáveis de ambiente/secrets no runner:
# GitLab: Settings > CI/CD > Variables
# GitHub Actions: Settings > Secrets and variables > Actions

# 3. Verificar conectividade do runner com o registry:
docker login <registry-url>

# 4. Limpar cache do runner e reexecutar:
# GitLab: Jobs > Retry  |  GitHub Actions: Re-run failed jobs

# 5. Verificar o .gitlab-ci.yml / .github/workflows/*.yml:
# - Validador GitLab CI: https://gitlab.com/-/ci/lint
# - GitHub Actions: act (https://github.com/nektos/act)"""

        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.HIGH,
            defect_layer=DefectLayer.CI_CD,
            root_cause=(
                f"A esteira de CI/CD do projeto *{project}* falhou no estágio *{stage}*. "
                "Causas comuns: falha de autenticação com Container Registry, Dockerfile com erro de sintaxe, "
                "dependência de build não disponível, secret não configurado no runner, "
                "ou timeout de rede durante o pull de imagem base."
            ),
            affected_component=f"Pipeline/{project}/stage/{stage}",
            fix_code=fix,
            fix_description=(
                "Reproduzir o build localmente com docker build para isolar o erro, "
                "verificar secrets do runner e corrigir o arquivo de pipeline."
            ),
            repro_steps=[
                f"Acessar o log completo do job {stage} no servidor de CI/CD",
                "Identificar a linha exata do erro no log",
                "Reproduzir localmente com docker build ou executando o comando do stage manualmente",
            ],
            validation_steps=[
                f"Job {stage} conclui com exit code 0",
                "Pipeline completa com status Passed/Success",
                "Imagem publicada no registry com a tag esperada",
            ],
        )

    def _diagnose_docker(self, container, stage, exit_code, restarts, logs) -> AgentDiagnosis:
        fix = f"""\
# Container Docker com falha:

# 1. Inspecionar logs do container:
docker logs --tail=100 {container}

# 2. Verificar status e configurações:
docker inspect {container}
docker stats --no-stream {container}

# 3. Verificar healthcheck interno:
docker inspect --format '{{{{range .State.Health.Log}}}}{{{{.Output}}}}{{{{end}}}}' {container}

# 4. Recriar o container após correção:
docker-compose up -d --build --force-recreate {container}"""

        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=self._detect_severity("DOCKER_FAILURE", restarts),
            defect_layer=DefectLayer.CONTAINER,
            root_cause=(
                f"Container Docker *{container}* em estado anômalo "
                f"(Exit Code: {exit_code if exit_code is not None else 'N/A'}, Reinicializações: {restarts}). "
                "Causas típicas: falha de healthcheck, erro de aplicação no startup, "
                "volume montado com permissão incorreta ou dependência de serviço não disponível."
            ),
            affected_component=f"Container/{container}",
            fix_code=fix,
            fix_description="Inspecionar logs e recriar o container após identificar e corrigir a causa raiz.",
            repro_steps=[
                f"docker logs --tail=100 {container}",
                f"docker inspect {container} | grep -A5 Status",
            ],
            validation_steps=[
                f"docker ps | grep {container} mostra STATUS=Up (healthy)",
                "Aplicação dentro do container responde às requisições",
            ],
        )

    def _diagnose_generic_infra(self, component, namespace, failure_type, logs) -> AgentDiagnosis:
        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.MEDIUM,
            defect_layer=DefectLayer.INFRASTRUCTURE,
            root_cause=(
                f"Falha de infraestrutura detectada no componente *{component}* "
                f"(namespace/projeto: {namespace}). Tipo: {failure_type}. "
                "Inspecionar logs e eventos do ambiente para determinar a causa raiz."
            ),
            affected_component=f"{component} ({namespace})",
            fix_code=(
                f"kubectl describe pod {component} -n {namespace}\n"
                f"kubectl get events -n {namespace} --sort-by='.lastTimestamp'"
            ),
            fix_description="Coletar eventos e logs para diagnóstico detalhado.",
            repro_steps=[
                f"kubectl describe pod {component} -n {namespace}",
                f"kubectl logs {component} -n {namespace} --tail=80",
            ],
            validation_steps=[f"Componente {component} restaurado para operação normal."],
        )
