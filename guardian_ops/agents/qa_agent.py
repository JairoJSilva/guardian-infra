# -*- coding: utf-8 -*-
"""
03. Senior QA Analyst Agent
==============================
Identificador : agent-senior-qa
Responsável   : Garantia de Qualidade, Triagem de Erros de Runtime e
                Análise de Cobertura de Testes.
Escopo        : Cypress, Playwright, Jest, PyTest, JUnit, BDD/Gherkin,
                Testes Unitários, de Integração e E2E.
"""
from typing import Dict, Any, List

from guardian_ops.agents.base_agent import BaseAgent, AgentDiagnosis, Severity, DefectLayer


class SeniorQAAgent(BaseAgent):
    """
    Agente especializado em qualidade de software e análise de falhas de testes.

    Cobre:
    - Análise de stack traces e relatórios de execução de testes
    - Determinação da camada exata da falha (Frontend, API, DB, Integração)
    - Classificação de defeito: regressão, erro de contrato, validação de payload
    - Geração de cenários de teste automatizados (Gherkin/BDD)
    - Identificação de testes flaky (intermitentes)
    - Root Cause Analysis (RCA) orientada a qualidade
    """
    AGENT_ID   = "agent-senior-qa"
    AGENT_NAME = "Senior QA Analyst Agent"

    _QA_TRIGGERS = {
        "ASSERTION", "TEST FAILED", "EXCEPTION", "UNHANDLED", "500", "503",
        "404", "TIMEOUT", "ASSERTION ERROR", "STACK TRACE", "NULLPOINTER",
        "CYPRESS", "PLAYWRIGHT", "JEST", "PYTEST", "JUNIT", "SELENIUM",
        "API ERROR", "CONTRACT", "PAYLOAD", "VALIDATION", "REGRESSION",
        "BUG", "DEFECT", "FLAKY", "COVERAGE", "TEST SUITE",
    }

    def can_handle(self, payload: Dict[str, Any]) -> bool:
        source  = payload.get("source", "").upper()
        failure = payload.get("failure_type", "").upper()
        error   = payload.get("error_message", "").upper()
        combined = f"{source} {failure} {error}"
        return any(t in combined for t in self._QA_TRIGGERS)

    def analyze(self, payload: Dict[str, Any]) -> AgentDiagnosis:
        failure_type = payload.get("failure_type", "").upper()
        error_msg    = payload.get("error_message", "")
        component    = payload.get("component", payload.get("identifier", "unknown"))
        logs         = payload.get("logs", payload.get("logs_snippet", ""))
        framework    = self._detect_test_framework(payload)
        layer        = self._detect_defect_layer(failure_type, error_msg)

        if "REGRESSION" in failure_type or "REGRESS" in failure_type:
            return self._diagnose_regression(component, framework, layer, error_msg, logs)
        elif "CONTRACT" in failure_type or "SCHEMA" in failure_type:
            return self._diagnose_api_contract(component, framework, error_msg, logs)
        elif "FLAKY" in failure_type or "INTERMITTENT" in failure_type:
            return self._diagnose_flaky_test(component, framework, error_msg, logs)
        elif "500" in failure_type or "503" in failure_type or "EXCEPTION" in failure_type:
            return self._diagnose_runtime_exception(component, framework, layer, error_msg, logs)
        elif "ASSERTION" in failure_type or "ASSERTION" in error_msg.upper():
            return self._diagnose_assertion_failure(component, framework, layer, error_msg, logs)
        elif "TIMEOUT" in failure_type:
            return self._diagnose_test_timeout(component, framework, error_msg, logs)
        else:
            return self._diagnose_generic_quality(component, framework, layer, failure_type, error_msg, logs)

    def _detect_test_framework(self, payload: Dict[str, Any]) -> str:
        combined = f"{payload.get('failure_type','')} {payload.get('error_message','')} {payload.get('logs','')}".upper()
        if "CYPRESS" in combined:
            return "Cypress"
        if "PLAYWRIGHT" in combined:
            return "Playwright"
        if "JEST" in combined:
            return "Jest"
        if "PYTEST" in combined:
            return "PyTest"
        if "JUNIT" in combined or "JAVA" in combined:
            return "JUnit"
        if "SELENIUM" in combined:
            return "Selenium"
        return "Desconhecido"

    def _detect_defect_layer(self, failure_type: str, error_msg: str) -> DefectLayer:
        combined = f"{failure_type} {error_msg}".upper()
        if any(k in combined for k in ["API", "REST", "GRAPHQL", "GRPC", "ENDPOINT", "HTTP", "500", "503", "404"]):
            return DefectLayer.API
        if any(k in combined for k in ["SQL", "DATABASE", "QUERY", "CONNECTION", "DB"]):
            return DefectLayer.DATABASE
        if any(k in combined for k in ["FRONTEND", "REACT", "VUE", "ANGULAR", "DOM", "BROWSER", "CSS", "UI"]):
            return DefectLayer.FRONTEND
        return DefectLayer.BACKEND

    def _generate_gherkin_scenario(self, component: str, failure_type: str, expected: str, actual: str) -> str:
        return f"""\
# Cenário de Teste Automatizado — Regressão para {component}
# Framework: BDD/Gherkin (compatível com Cucumber/Behave/SpecFlow)

Feature: Validação de comportamento de {component}

  @regressao @guardianops @automacao
  Scenario: Detectar falha de {failure_type} em {component}
    Given o ambiente está operacional e {component} está disponível
    When uma requisição é feita ao endpoint afetado
    Then o sistema deve retornar resposta com status 200
    And o payload de resposta deve conter os campos obrigatórios
    But NÃO deve ocorrer: {failure_type}

  # Comportamento Esperado:
  # {expected}

  # Comportamento Observado (BUG):
  # {actual}"""

    # -------------------------------------------------------------------------
    # Diagnósticos por tipo de falha de qualidade
    # -------------------------------------------------------------------------

    def _diagnose_regression(self, component, framework, layer, error_msg, logs) -> AgentDiagnosis:
        test_code = self._generate_gherkin_scenario(
            component, "Regressão",
            expected="Funcionalidade executada com sucesso como nas versões anteriores",
            actual=error_msg[:200] if error_msg else "Comportamento inesperado detectado"
        )

        if framework == "Jest":
            test_code += f"""\n\n// Teste Jest para regressão em {component}:
describe('{component} - Teste de Regressão', () => {{
  it('deve executar sem erros após o fix', async () => {{
    const response = await request(app).get('/endpoint-afetado');
    expect(response.status).toBe(200);
    expect(response.body).toHaveProperty('data');
  }});
}});"""
        elif framework == "PyTest":
            test_code += f"""\n\n# Teste PyTest para regressão em {component}:
def test_{component.lower().replace('-','_')}_regression(client):
    \"\"\"Valida que a regressão em {component} foi corrigida.\"\"\"
    response = client.get('/endpoint-afetado')
    assert response.status_code == 200
    assert 'data' in response.json()"""

        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.HIGH,
            defect_layer=layer,
            root_cause=(
                f"Regressão detectada no componente *{component}*. "
                "Uma funcionalidade que funcionava corretamente em versões anteriores "
                "passou a falhar após uma mudança recente de código, configuração ou dependência. "
                f"Erro observado: {error_msg[:300] if error_msg else 'Ver logs'}."
            ),
            affected_component=f"{component} (Camada: {layer.value})",
            fix_code=test_code,
            fix_description=(
                f"Cenário de teste automatizado gerado ({framework}) para garantir que a regressão "
                "seja detectada em futuras execuções de CI/CD. Execute este teste após aplicar o fix."
            ),
            repro_steps=[
                f"Identificar o commit que introduziu a regressão via `git bisect`",
                f"Executar o suite de testes de {framework} no branch afetado",
                "Comparar comportamento com a última versão estável (tag anterior)",
            ],
            validation_steps=[
                f"Suite de testes de {framework} passa sem falhas",
                f"Nenhum erro de regressão nos últimos 3 builds de CI/CD",
                "Coverage de testes mantida ou aumentada após o fix",
            ],
        )

    def _diagnose_api_contract(self, component, framework, error_msg, logs) -> AgentDiagnosis:
        test_code = f"""\
// Teste de Contrato de API — {component}
// Usar com: Jest + Supertest | PyTest + httpx | Pact.io

// Jest + Supertest:
describe('Contrato de API: {component}', () => {{
  it('deve retornar o schema correto no endpoint afetado', async () => {{
    const response = await request(app).get('/api/endpoint');
    
    // Validar schema da resposta:
    expect(response.status).toBe(200);
    expect(response.body).toMatchObject({{
      id: expect.any(Number),
      name: expect.any(String),
      // Adicionar campos obrigatórios conforme OpenAPI/Swagger spec
    }});
  }});

  it('deve retornar 400 para payload inválido', async () => {{
    const response = await request(app).post('/api/endpoint').send({{}});
    expect(response.status).toBe(400);
    expect(response.body).toHaveProperty('errors');
  }});
}});

# PyTest + httpx:
def test_api_contract_{component.lower().replace('-','_')}(client):
    response = client.get("/api/endpoint")
    assert response.status_code == 200
    data = response.json()
    assert "id" in data
    assert "name" in data
    assert isinstance(data["id"], int)"""

        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.HIGH,
            defect_layer=DefectLayer.API,
            root_cause=(
                f"Erro de contrato de API detectado no componente *{component}*. "
                "A API está retornando um payload com estrutura diferente do esperado pelo contrato "
                "(schema, tipos de dados, campos obrigatórios ausentes ou renomeados). "
                f"Detalhe: {error_msg[:300] if error_msg else 'Verificar schema da API'}"
            ),
            affected_component=f"API/{component}",
            fix_code=test_code,
            fix_description=(
                "Testes de contrato gerados para validar o schema da API. "
                "Integrar no CI/CD para detectar quebras de contrato antes do deploy em produção."
            ),
            repro_steps=[
                "Verificar a documentação OpenAPI/Swagger do endpoint afetado",
                "Comparar o payload retornado com o schema esperado",
                "Identificar o campo ausente, renomeado ou com tipo incorreto",
            ],
            validation_steps=[
                "Todos os testes de contrato passam",
                "Payload da API valida contra o schema OpenAPI sem erros",
                "Clientes (frontend/outros serviços) sem erros de parsing",
            ],
        )

    def _diagnose_flaky_test(self, component, framework, error_msg, logs) -> AgentDiagnosis:
        fix = f"""\
# Estratégias para eliminar Testes Flaky em {component}:

# 1. Isolamento: garantir que o teste não depende de estado compartilhado
# Usar beforeEach/afterEach para reset de estado:

# Jest:
beforeEach(async () => {{
  await db.migrate.rollback();
  await db.migrate.latest();
  await db.seed.run();
}});

# PyTest:
@pytest.fixture(autouse=True)
def reset_state(db):
    db.rollback()
    yield
    db.rollback()

# 2. Aguardar elementos assíncronos explicitamente (E2E):

# Cypress:
cy.get('[data-testid="submit-btn"]', {{ timeout: 10000 }}).should('be.visible').click();

# Playwright:
await page.waitForSelector('[data-testid="result"]', {{ state: 'visible' }});

# 3. Rodar o teste em loop para confirmar flakiness:
# pytest --count=5 test_flaky.py  (com pytest-repeat)
# npx cypress run --spec "flaky.cy.js" --env RETRIES=3"""

        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.MEDIUM,
            defect_layer=self._detect_defect_layer("", error_msg),
            root_cause=(
                f"Teste intermitente (flaky) detectado no componente *{component}* "
                f"com framework *{framework}*. "
                "Testes flaky falham de forma não determinística, sem mudança de código. "
                "Causas: dependência de timing/race condition, estado compartilhado entre testes, "
                "chamadas HTTP reais sem mock, ou seletores de UI instáveis."
            ),
            affected_component=f"TestSuite/{component}/{framework}",
            fix_code=fix,
            fix_description=(
                "Isolar o teste com reset de estado no beforeEach, mockar dependências externas "
                "e usar waitFor explícito para operações assíncronas."
            ),
            repro_steps=[
                f"Executar o teste em loop 10 vezes: pytest --count=10 / cypress run --env RETRIES=3",
                "Verificar se a falha ocorre apenas em CI ou também localmente",
                "Identificar se há shared state entre testes",
            ],
            validation_steps=[
                "Teste passa 10 execuções consecutivas sem falha",
                "Teste isolado (sem dependência de outros testes ou estado externo)",
            ],
        )

    def _diagnose_runtime_exception(self, component, framework, layer, error_msg, logs) -> AgentDiagnosis:
        gherkin = self._generate_gherkin_scenario(
            component, "Runtime Exception",
            expected="Requisição processada com sucesso (HTTP 200)",
            actual=f"HTTP 500 / Unhandled Exception: {error_msg[:150] if error_msg else 'Ver logs'}"
        )
        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.HIGH,
            defect_layer=layer,
            root_cause=(
                f"Exceção de runtime detectada no componente *{component}* (camada: {layer.value}). "
                f"Erro: {error_msg[:400] if error_msg else 'Ver stack trace nos logs'}. "
                "A exceção não foi tratada (unhandled exception), resultando em falha visível ao usuário final."
            ),
            affected_component=f"{component} (camada: {layer.value})",
            fix_code=gherkin,
            fix_description=(
                "Cenário de teste de regressão gerado para cobrir a exceção. "
                "O desenvolvedor deve implementar tratamento de exceção (try/catch + resposta HTTP adequada) "
                "e o QA deve validar com este cenário de teste."
            ),
            repro_steps=[
                "Reproduzir a requisição que causou a exceção",
                f"Verificar o stack trace completo nos logs da aplicação: {component}",
                "Identificar a linha exata onde a exceção foi lançada",
            ],
            validation_steps=[
                "Endpoint retorna HTTP 200 (ou código correto) sem exceção",
                "Stack trace não aparece mais nos logs de erro",
                "Teste de regressão passa no CI/CD",
            ],
        )

    def _diagnose_assertion_failure(self, component, framework, layer, error_msg, logs) -> AgentDiagnosis:
        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.MEDIUM,
            defect_layer=layer,
            root_cause=(
                f"Falha de assertion em teste automatizado do componente *{component}* "
                f"({framework}). O valor retornado pela aplicação diverge do valor esperado pelo teste. "
                f"Detalhe: {error_msg[:300] if error_msg else 'Ver relatório de testes'}."
            ),
            affected_component=f"TestSuite/{component}/{framework}",
            fix_code=(
                f"# Assertion failure em {framework}:\n"
                "# 1. Verificar se o comportamento esperado ainda é válido (o teste pode estar desatualizado)\n"
                "# 2. Se o comportamento mudou intencionalmente: atualizar o teste\n"
                "# 3. Se é um bug: corrigir a aplicação e manter o teste\n"
                "# 4. Verificar se há data/estado mutável causando o comportamento diferente"
            ),
            fix_description="Verificar se o teste está desatualizado ou se é um bug real na aplicação.",
            repro_steps=[
                f"Executar: {framework.lower()} --verbose {component}",
                "Verificar o valor esperado vs valor recebido no relatório",
                "Verificar se houve mudança intencional no comportamento",
            ],
            validation_steps=["Teste passa sem assertion failure", "Comportamento da aplicação correto e documentado"],
        )

    def _diagnose_test_timeout(self, component, framework, error_msg, logs) -> AgentDiagnosis:
        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.MEDIUM,
            defect_layer=DefectLayer.API,
            root_cause=(
                f"Timeout detectado em teste do componente *{component}* ({framework}). "
                "A operação testada demorou mais do que o threshold configurado. "
                "Causas: chamada HTTP real sem mock (lenta em CI), banco de dados sem índice, "
                "operação bloqueante no thread principal, ou threshold muito conservador."
            ),
            affected_component=f"TestSuite/{component}/{framework}/timeout",
            fix_code=(
                f"# Aumentar timeout e mockar dependências externas em {framework}:\n\n"
                "# Jest:\n"
                "jest.setTimeout(30000); // aumentar timeout global\n"
                "jest.mock('./api-service'); // mockar chamadas HTTP\n\n"
                "# PyTest:\n"
                "@pytest.mark.timeout(30)\n"
                "def test_slow_operation():\n    ...\n\n"
                "# Cypress:\n"
                "cy.get('selector', {timeout: 15000}).should('exist');"
            ),
            fix_description="Aumentar timeout de testes e substituir chamadas HTTP reais por mocks em ambiente de CI.",
            repro_steps=[
                "Verificar o tempo médio de execução do teste",
                "Identificar se há chamadas HTTP reais ou operações de I/O lentas",
            ],
            validation_steps=["Teste completa dentro do timeout configurado", "Pipeline de CI não trava mais neste step"],
        )

    def _diagnose_generic_quality(self, component, framework, layer, failure_type, error_msg, logs) -> AgentDiagnosis:
        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.MEDIUM,
            defect_layer=layer,
            root_cause=(
                f"Falha de qualidade detectada no componente *{component}* "
                f"(Framework: {framework}, Camada: {layer.value}). "
                f"Tipo: {failure_type}. Erro: {error_msg[:200] if error_msg else 'Ver logs'}."
            ),
            affected_component=f"{component}/{framework}",
            fix_code="# Analisar o relatório de testes completo para identificar a causa exata.",
            fix_description="Inspecionar o relatório de execução dos testes para diagnóstico detalhado.",
            repro_steps=["Executar a suite de testes e analisar o relatório de falhas"],
            validation_steps=["Suite de testes passa sem falhas", "Cobertura de código mantida"],
        )
