# -*- coding: utf-8 -*-
"""
04. FullStack Developer Agent
================================
Identificador : agent-fullstack-developer
Responsável   : Análise e Refatoração de Código de Aplicação (Frontend e Backend).
Escopo        : Node.js, Python, Go, Java, C#, React, Vue, Angular,
                REST, GraphQL, gRPC, OWASP, Clean Architecture, SOLID.
"""
from typing import Dict, Any

from guardian_ops.agents.base_agent import BaseAgent, AgentDiagnosis, Severity, DefectLayer


class FullStackDeveloperAgent(BaseAgent):
    """
    Agente especializado em análise e correção de código de aplicação.

    Cobre:
    - Backend: Node.js, Python, Go, Java, C# — exceções, memory leaks, lógica incorreta
    - Frontend: React, Vue, Angular — erros de bundle, CORS, CSP, DOM, estado
    - APIs: REST, GraphQL, gRPC — erros 4xx/5xx, contract violations, rate limiting
    - Segurança: OWASP Top 10 — SQL injection, XSS, IDOR, secrets expostos
    - Performance: N+1 queries, blocking I/O, bundle size, CPU bound
    """
    AGENT_ID   = "agent-fullstack-developer"
    AGENT_NAME = "FullStack Developer Agent"

    _DEV_TRIGGERS = {
        "NULLPOINTEREXCEPTION", "NULLREFERENCE", "TYPEERROR", "ATTRIBUTEERROR",
        "INDEXERROR", "KEYERROR", "VALUEERROR", "RUNTIMEEXCEPTION", "EXCEPTION",
        "500", "503", "401", "403", "CORS", "CSP", "MEMORY LEAK", "HEAP",
        "UNHANDLED PROMISE", "ASYNC", "AWAIT", "IMPORT ERROR", "MODULE NOT FOUND",
        "SEGFAULT", "PANIC", "GOROUTINE", "NPE", "OOM", "BUFFER OVERFLOW",
        "SQL INJECTION", "XSS", "OWASP", "AUTHENTICATION", "AUTHORIZATION",
        "NODE", "PYTHON", "JAVA", "GO", "CSHARP", "REACT", "VUE", "ANGULAR",
        "WEBPACK", "VITE", "BUNDLE", "BUILD FAILED", "COMPILATION",
    }

    def can_handle(self, payload: Dict[str, Any]) -> bool:
        source  = payload.get("source", "").upper()
        failure = payload.get("failure_type", "").upper()
        error   = payload.get("error_message", "").upper()
        combined = f"{source} {failure} {error}"
        return any(t in combined for t in self._DEV_TRIGGERS)

    def analyze(self, payload: Dict[str, Any]) -> AgentDiagnosis:
        failure_type = payload.get("failure_type", "").upper()
        error_msg    = payload.get("error_message", "")
        component    = payload.get("component", payload.get("identifier", "unknown"))
        logs         = payload.get("logs", payload.get("logs_snippet", ""))
        details      = payload.get("details", {})
        language     = self._detect_language(failure_type, error_msg, details)
        layer        = self._detect_layer(failure_type, error_msg, language)

        if "NULLPOINTER" in failure_type or "NULLREFERENCE" in failure_type or "NONE" in failure_type:
            return self._diagnose_null_reference(component, language, layer, error_msg, logs)
        elif "MEMORY LEAK" in failure_type or "HEAP" in failure_type or "OOM" in failure_type:
            return self._diagnose_memory_leak(component, language, layer, error_msg, logs)
        elif "CORS" in failure_type or "CORS" in error_msg.upper():
            return self._diagnose_cors(component, language, error_msg, logs)
        elif "UNHANDLED PROMISE" in failure_type or "ASYNC" in failure_type:
            return self._diagnose_async_error(component, language, error_msg, logs)
        elif "SQL INJECTION" in failure_type or "XSS" in failure_type or "OWASP" in failure_type:
            return self._diagnose_security(component, language, failure_type, error_msg, logs)
        elif "401" in failure_type or "403" in failure_type or "AUTH" in failure_type:
            return self._diagnose_auth(component, language, failure_type, error_msg, logs)
        elif "BUNDLE" in failure_type or "WEBPACK" in failure_type or "VITE" in failure_type:
            return self._diagnose_frontend_build(component, error_msg, logs)
        elif "EXCEPTION" in failure_type or "500" in failure_type:
            return self._diagnose_unhandled_exception(component, language, layer, error_msg, logs)
        else:
            return self._diagnose_generic_code(component, language, layer, failure_type, error_msg, logs)

    def _detect_language(self, failure_type: str, error_msg: str, details: dict) -> str:
        combined = f"{failure_type} {error_msg} {details.get('language', '')}".upper()
        if "NULLPOINTEREXCEPTION" in combined or "JAVA" in combined or "SPRING" in combined:
            return "Java"
        if "PYTHON" in combined or "ATTRIBUTEERROR" in combined or "IMPORTERROR" in combined:
            return "Python"
        if "NODE" in combined or "JAVASCRIPT" in combined or "TYPESCRIPT" in combined or "UNHANDLED PROMISE" in combined:
            return "Node.js/TypeScript"
        if "GO" in combined or "GOROUTINE" in combined or "PANIC" in combined:
            return "Go"
        if "CSHARP" in combined or "C#" in combined or "NULLREFERENCE" in combined or "DOTNET" in combined:
            return "C#/.NET"
        if "REACT" in combined or "VUE" in combined or "ANGULAR" in combined or "BUNDLE" in combined:
            return "Frontend (JS/TS)"
        return "Desconhecido"

    def _detect_layer(self, failure_type: str, error_msg: str, language: str) -> DefectLayer:
        combined = f"{failure_type} {error_msg}".upper()
        if "Frontend" in language or any(k in combined for k in ["DOM", "BROWSER", "CSS", "BUNDLE", "COMPONENT"]):
            return DefectLayer.FRONTEND
        if any(k in combined for k in ["REST", "API", "ENDPOINT", "HTTP", "GRAPHQL", "GRPC", "500", "503"]):
            return DefectLayer.API
        if any(k in combined for k in ["SQL", "DATABASE", "QUERY", "DB"]):
            return DefectLayer.DATABASE
        return DefectLayer.BACKEND

    # -------------------------------------------------------------------------
    # Diagnósticos por tipo de falha de desenvolvimento
    # -------------------------------------------------------------------------

    def _diagnose_null_reference(self, component, language, layer, error_msg, logs) -> AgentDiagnosis:
        if "Java" in language:
            fix = f"""\
// Java — Corrigir NullPointerException em {component}:
// ANTES (código problemático):
// String result = obj.getValue().toString(); // NullPointerException se obj ou getValue() for null

// DEPOIS (código seguro):
// Opção 1: Optional (Java 8+)
String result = Optional.ofNullable(obj)
    .map(MyClass::getValue)
    .map(Object::toString)
    .orElse("valor-padrão");

// Opção 2: Verificação explícita
if (obj != null && obj.getValue() != null) {{
    String result = obj.getValue().toString();
}} else {{
    log.warn("Valor nulo detectado em {component} — usando fallback");
    // tratar o caso nulo adequadamente
}}

// Opção 3: Objects.requireNonNullElse (Java 9+)
String result = Objects.requireNonNullElse(obj.getValue(), "fallback").toString();"""
        elif "Python" in language:
            fix = f"""\
# Python — Corrigir AttributeError/NoneType em {component}:
# ANTES (problemático):
# result = obj.value.upper()  # AttributeError se obj ou value for None

# DEPOIS (defensivo):
# Opção 1: Verificação explícita
result = obj.value.upper() if obj and obj.value else "valor-padrão"

# Opção 2: getattr com fallback
result = getattr(obj, 'value', None)
if result is not None:
    result = result.upper()

# Opção 3: try/except com log
try:
    result = obj.value.upper()
except AttributeError as e:
    logger.warning(f"Valor nulo em {component}: {{e}}")
    result = "fallback" """
        elif "Node" in language:
            fix = f"""\
// Node.js/TypeScript — Corrigir TypeError em {component}:
// ANTES (problemático):
// const result = obj.value.toUpperCase(); // TypeError: Cannot read property of undefined

// DEPOIS (seguro com optional chaining — ES2020+):
const result = obj?.value?.toUpperCase() ?? 'valor-padrão';

// Com TypeScript (tipo explícito):
const result: string = obj?.value?.toUpperCase() ?? 'fallback';

// Em async handler — sempre validar o payload de entrada:
if (!obj || !obj.value) {{
    return res.status(400).json({{ error: 'Campo obrigatório ausente: value' }});
}}"""
        else:
            fix = f"# Adicionar verificação de nulo/undefined antes de acessar propriedades do objeto em {component}."

        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.HIGH,
            defect_layer=layer,
            root_cause=(
                f"NullPointerException / NullReferenceError / TypeError detectado no componente *{component}* "
                f"({language}, camada: {layer.value}). "
                "O código tenta acessar uma propriedade ou chamar um método em um objeto que é null/undefined/None. "
                f"Detalhe do erro: {error_msg[:300] if error_msg else 'Ver stack trace nos logs'}."
            ),
            affected_component=f"{component} ({language} / {layer.value})",
            fix_code=fix,
            fix_description=(
                f"Adicionar verificação defensiva com Optional Chaining (?.), Optional.ofNullable(), "
                "getattr() ou verificação explícita antes de acessar o valor. "
                "Manter a mesma arquitetura e padrões de projeto do repositório."
            ),
            repro_steps=[
                "Identificar a linha exata do stack trace onde o null foi acessado",
                "Verificar qual objeto pode ser null/undefined/None",
                "Reproduzir localmente passando null/undefined no payload",
            ],
            validation_steps=[
                "Código executa sem lançar NullPointerException em qualquer fluxo",
                "Testes unitários cobrem o cenário de entrada nula",
                "Testes de integração/E2E passam sem erros de null",
            ],
        )

    def _diagnose_memory_leak(self, component, language, layer, error_msg, logs) -> AgentDiagnosis:
        if "Node" in language:
            fix = f"""\
// Node.js — Diagnóstico e correção de Memory Leak em {component}:

// 1. Gerar heap snapshot para análise:
const v8 = require('v8');
const fs = require('fs');
const snapshot = v8.writeHeapSnapshot();
console.log(`Heap snapshot: ${{snapshot}}`);

// 2. Usar --inspect para análise com Chrome DevTools:
// node --inspect main.js
// Abrir: chrome://inspect

// 3. Padrões comuns de leak e correção:

// LEAK: Event listener não removido (muito comum):
// ANTES:
// emitter.on('data', handleData);  // nunca removido!
// DEPOIS:
const handler = (data) => handleData(data);
emitter.on('data', handler);
// Limpar quando não mais necessário:
emitter.off('data', handler); // ou emitter.removeListener('data', handler)

// LEAK: Closure retendo referência a objeto grande:
// Garantir que variáveis não usadas saiam do escopo
// Usar WeakMap/WeakRef para referências que podem ser coletadas pelo GC

// LEAK: Interval/Timeout não limpo:
const intervalId = setInterval(task, 1000);
// Sempre limpar ao destruir o componente:
clearInterval(intervalId);"""
        elif "Java" in language:
            fix = f"""\
// Java — Diagnóstico de Memory Leak em {component}:

// 1. Analisar heap com jmap:
// jmap -dump:format=b,file=heap.hprof <PID>
// Abrir com Eclipse MAT ou VisualVM

// 2. Padrões comuns:
// LEAK: Coleção estática crescendo indefinidamente:
// ANTES:
// private static List<Event> events = new ArrayList<>(); // nunca limpa!
// DEPOIS:
// Usar estrutura com tamanho máximo:
// private static final Queue<Event> events = new LinkedBlockingQueue<>(1000);

// 3. Fechar recursos com try-with-resources:
try (Connection conn = dataSource.getConnection();
     PreparedStatement stmt = conn.prepareStatement(sql)) {{
    // usar resources — são fechados automaticamente
}}

// 4. Evitar referências fortes desnecessárias — usar WeakReference:
WeakReference<HeavyObject> ref = new WeakReference<>(heavyObject);"""
        else:
            fix = f"# Analisar heap/memory profile do processo {component} para identificar objetos retidos."

        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.HIGH,
            defect_layer=layer,
            root_cause=(
                f"Memory Leak detectado no componente *{component}* ({language}). "
                "O processo está consumindo memória progressivamente sem liberá-la, "
                "levando eventualmente ao OOMKilled ou degradação severa de performance. "
                "Causas frequentes: event listeners não removidos, closures retendo objetos, "
                "coleções estáticas crescendo sem limite, ou recursos não fechados (conexões, streams)."
            ),
            affected_component=f"{component} ({language} / Memory)",
            fix_code=fix,
            fix_description=(
                "Identificar e corrigir o ponto de vazamento usando heap snapshot ou profiler. "
                "Garantir que event listeners, timers e recursos são sempre limpos ao fim do ciclo de vida."
            ),
            repro_steps=[
                "Monitorar uso de memória do processo ao longo do tempo (crescimento contínuo = leak)",
                "Gerar heap snapshot antes e depois de operações suspeitas",
                "Comparar snapshots para identificar objetos retidos inesperadamente",
            ],
            validation_steps=[
                "Uso de memória estável após 1 hora de execução",
                "GC coletando objetos adequadamente (sem crescimento linear do heap)",
                "Sem eventos OOMKilled nos próximos 24h",
            ],
        )

    def _diagnose_cors(self, component, language, error_msg, logs) -> AgentDiagnosis:
        fix = f"""\
// Corrigir CORS em {component}:

// Node.js (Express):
const cors = require('cors');
app.use(cors({{
  origin: ['https://app.seudominio.com', 'https://admin.seudominio.com'], // domínios permitidos
  methods: ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'OPTIONS'],
  allowedHeaders: ['Content-Type', 'Authorization', 'X-Request-ID'],
  credentials: true, // se usar cookies/sessions
  maxAge: 86400,     // cache do preflight por 24h
}}));

// Python (FastAPI):
from fastapi.middleware.cors import CORSMiddleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["https://app.seudominio.com"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

// Python (Django):
# settings.py:
CORS_ALLOWED_ORIGINS = [
    "https://app.seudominio.com",
]
CORS_ALLOW_CREDENTIALS = True

// NGINX (se necessário no gateway):
# add_header 'Access-Control-Allow-Origin' 'https://app.seudominio.com' always;
# add_header 'Access-Control-Allow-Methods' 'GET, POST, OPTIONS' always;

// AVISO DE SEGURANCA (OWASP):
# NUNCA usar: Access-Control-Allow-Origin: * com credentials: true
# Sempre usar lista explícita de origens em produção"""

        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.MEDIUM,
            defect_layer=DefectLayer.API,
            root_cause=(
                f"Erro de CORS (Cross-Origin Resource Sharing) detectado no componente *{component}*. "
                "O browser está bloqueando a requisição porque o servidor não retornou os headers "
                "Access-Control corretos para a origem do cliente. "
                f"Detalhe: {error_msg[:200] if error_msg else 'CORS policy blocked request'}"
            ),
            affected_component=f"API/{component}/CORS",
            fix_code=fix,
            fix_description=(
                "Configurar o middleware CORS corretamente no servidor com lista explícita de origens permitidas. "
                "Atenção: nunca usar wildcard (*) com credentials em produção (vulnerabilidade OWASP)."
            ),
            repro_steps=[
                "Abrir o DevTools do browser (F12) > Console",
                "Verificar o erro: 'Access to fetch at ... has been blocked by CORS policy'",
                "Checar os headers da resposta da API: falta Access-Control-Allow-Origin",
            ],
            validation_steps=[
                "Requisição cross-origin funciona sem erros de CORS no browser",
                "Header Access-Control-Allow-Origin presente na resposta HTTP",
                "Preflight OPTIONS retorna 204 com headers corretos",
            ],
        )

    def _diagnose_async_error(self, component, language, error_msg, logs) -> AgentDiagnosis:
        fix = f"""\
// Corrigir Unhandled Promise Rejection / async error em {component}:

// Node.js/TypeScript — Tratar erros em async/await:
// ANTES (problemático):
// async function fetchData() {{
//   const data = await api.get('/endpoint'); // lança exceção não tratada!
//   return data;
// }}

// DEPOIS (correto):
async function fetchData(): Promise<Data | null> {{
  try {{
    const data = await api.get('/endpoint');
    return data;
  }} catch (error: unknown) {{
    const err = error as Error;
    logger.error(`[{component}] Falha ao buscar dados: ${{err.message}}`, {{ stack: err.stack }});
    // Decidir: relançar, retornar null, ou retornar valor padrão
    return null; // ou throw new AppError('FETCH_FAILED', err)
  }}
}}

// Global handler para unhandled rejections (adicionar no entry point):
process.on('unhandledRejection', (reason: unknown, promise: Promise<unknown>) => {{
  logger.error('Unhandled Promise Rejection:', {{ reason, promise }});
  // Não derrubar o processo em produção — logar e alertar
}});

// Python — asyncio:
async def fetch_data():
    try:
        result = await api.get('/endpoint')
        return result
    except aiohttp.ClientError as e:
        logger.error(f"[{component}] Falha na requisição async: {{e}}")
        raise  # ou retornar valor padrão"""

        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.HIGH,
            defect_layer=DefectLayer.BACKEND,
            root_cause=(
                f"Unhandled Promise Rejection / erro assíncrono não tratado no componente *{component}* ({language}). "
                "Uma operação assíncrona (async/await, Promise, asyncio) lançou uma exceção que não foi "
                "capturada por um bloco try/catch, resultando em comportamento indefinido ou crash silencioso. "
                f"Detalhe: {error_msg[:300] if error_msg else 'Ver logs do processo'}"
            ),
            affected_component=f"{component} ({language} / async)",
            fix_code=fix,
            fix_description=(
                "Adicionar try/catch em todas as funções async/await. "
                "Adicionar global handler para unhandledRejection para garantir visibilidade dos erros."
            ),
            repro_steps=[
                "Identificar a função async onde a exceção foi lançada",
                "Verificar se há await sem try/catch no código",
                "Verificar se Promise.all() tem tratamento de erro (falha rápida)",
            ],
            validation_steps=[
                "Ausência de 'UnhandledPromiseRejection' nos logs",
                "Todas as funções async com try/catch explícito",
                "Erros de API retornam resposta estruturada ao invés de crash",
            ],
        )

    def _diagnose_security(self, component, language, failure_type, error_msg, logs) -> AgentDiagnosis:
        is_sqli = "SQL INJECTION" in failure_type
        fix = f"""\
// ALERTA DE SEGURANCA (OWASP) — {failure_type} detectado em {component}:

{"// CORRECAO SQL INJECTION:" if is_sqli else "// CORRECAO XSS:"}
{"// NUNCA concatenar input do usuário em queries SQL!" if is_sqli else "// NUNCA inserir input do usuário sem sanitização no HTML!"}

{"// Node.js — usar Prepared Statements (parameterized queries):" if is_sqli else "// Node.js — sanitizar output:"}
{"// ANTES (vulnerável):" if is_sqli else ""}
{"// const query = `SELECT * FROM users WHERE id = ${userId}`; // VULNERAVEL!" if is_sqli else ""}
{"// DEPOIS (seguro — parameterized):" if is_sqli else ""}
{"const query = 'SELECT * FROM users WHERE id = ?';" if is_sqli else "const DOMPurify = require('dompurify');"}
{"await db.execute(query, [userId]); // userId é sanitizado automaticamente" if is_sqli else "const safeHTML = DOMPurify.sanitize(userInput);"}

{"// Python (SQLAlchemy):" if is_sqli else "// Python (Jinja2 — auto-escaping habilitado por padrão):"}
{"# result = db.execute(text('SELECT * FROM users WHERE id = :id'), {'id': user_id})" if is_sqli else "# Garantir: autoescape=True no template engine"}

// ACOES IMEDIATAS RECOMENDADAS:
// 1. Auditar TODOS os pontos de entrada de dados externos no componente {component}
// 2. Ativar WAF (Web Application Firewall) se disponível
// 3. Revisar logs de acesso para verificar se o vetor foi explorado
// 4. Rotacionar credenciais de banco de dados como precaução"""

        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.CRITICAL,
            defect_layer=DefectLayer.BACKEND,
            root_cause=(
                f"VULNERABILIDADE DE SEGURANCA detectada no componente *{component}* ({language}). "
                f"Tipo: *{failure_type}* (OWASP Top 10). "
                "Input de usuário está sendo processado sem sanitização adequada, "
                "criando potencial vetor de ataque. ACAO IMEDIATA NECESSARIA."
            ),
            affected_component=f"Security/{component}/{failure_type}",
            fix_code=fix,
            fix_description=(
                f"Corrigir vulnerabilidade {failure_type} usando Prepared Statements (SQL Injection) "
                "ou sanitização de output (XSS). Auditar todo o código relacionado."
            ),
            repro_steps=[
                "Identificar todos os pontos de entrada de dados externos no componente",
                "Verificar se há concatenação direta de input em queries/HTML",
                "Executar scanner OWASP ZAP ou Semgrep para identificar outros vetores",
            ],
            validation_steps=[
                "Scanner de segurança não detecta mais a vulnerabilidade",
                "Testes de penetração básicos (SQL injection / XSS payload) bloqueados",
                "Code review de segurança aprovado",
            ],
        )

    def _diagnose_auth(self, component, language, failure_type, error_msg, logs) -> AgentDiagnosis:
        is_401 = "401" in failure_type
        fix = f"""\
// Corrigir erro de {'Autenticação (401)' if is_401 else 'Autorização (403)'} em {component}:

// Node.js/Express — Middleware de {'autenticação' if is_401 else 'autorização'} JWT:
{"// Verificar token JWT:" if is_401 else "// Verificar permissão:"}
const authMiddleware = async (req, res, next) => {{
{"  try {" if is_401 else ""}
{"    const token = req.headers.authorization?.split(' ')[1];" if is_401 else "  const user = req.user;"}
{"    if (!token) return res.status(401).json({ error: 'Token ausente' });" if is_401 else "  if (!user.roles.includes('admin')) {"}
{"    const decoded = jwt.verify(token, process.env.JWT_SECRET);" if is_401 else "    return res.status(403).json({ error: 'Permissão negada' });"}
{"    req.user = decoded;" if is_401 else "  }"}
{"    next();" if is_401 else "  next();"}
{"  } catch (e) {" if is_401 else ""}
{"    return res.status(401).json({ error: 'Token inválido ou expirado' });" if is_401 else ""}
{"  }" if is_401 else ""}
}};

// Verificar configurações de CORS + Authorization header:
// O browser pode bloquear o header Authorization em preflight OPTIONS
// Garantir que allowedHeaders inclui 'Authorization' na config CORS"""

        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.HIGH,
            defect_layer=DefectLayer.API,
            root_cause=(
                f"Erro HTTP {'401 Unauthorized' if is_401 else '403 Forbidden'} detectado em *{component}*. "
                f"{'O token de autenticação está ausente, inválido ou expirado.' if is_401 else 'O usuário autenticado não possui permissão para este recurso.'} "
                f"Detalhe: {error_msg[:200] if error_msg else ''}"
            ),
            affected_component=f"API/{component}/auth",
            fix_code=fix,
            fix_description=(
                f"Verificar e corrigir o middleware de {'autenticação' if is_401 else 'autorização'}. "
                "Garantir que tokens JWT são validados e permissões verificadas em cada rota protegida."
            ),
            repro_steps=[
                f"Fazer requisição ao endpoint sem token (401) ou com usuário sem permissão (403)",
                "Verificar o header Authorization na requisição",
                "Verificar as roles/permissões do usuário no sistema",
            ],
            validation_steps=[
                "Endpoint retorna 200 com token/permissão válidos",
                "Endpoint retorna 401/403 corretamente para acessos indevidos",
                "Middleware de auth funcionando para todas as rotas protegidas",
            ],
        )

    def _diagnose_frontend_build(self, component, error_msg, logs) -> AgentDiagnosis:
        fix = f"""\
# Diagnóstico de falha de build Frontend em {component}:

# 1. Limpar cache e node_modules:
rm -rf node_modules .cache dist .next
npm install  # ou yarn install / pnpm install

# 2. Verificar versão do Node.js (compatibilidade):
node --version
nvm use --lts  # usar LTS se divergir

# 3. Verificar dependências com vulnerabilidades ou incompatibilidades:
npm audit
npm outdated

# 4. Executar build com verbose para ver o erro exato:
npm run build -- --verbose 2>&1 | head -100

# 5. Verificar se variáveis de ambiente obrigatórias estão definidas:
# Vite: VITE_API_URL, VITE_*
# Next.js: NEXT_PUBLIC_*, NODE_ENV
# Create React App: REACT_APP_*

# 6. Verificar tamanho do bundle (se warning de bundle size):
# Vite: vite-bundle-visualizer
# Webpack: webpack-bundle-analyzer
npx vite-bundle-visualizer"""

        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.HIGH,
            defect_layer=DefectLayer.FRONTEND,
            root_cause=(
                f"Falha de build do Frontend detectada no componente *{component}*. "
                "Causas frequentes: incompatibilidade de versão de Node.js, dependência com erro de tipagem, "
                "variável de ambiente obrigatória não definida, ou erro de sintaxe em arquivo de configuração "
                "(vite.config.ts, next.config.js, webpack.config.js). "
                f"Detalhe: {error_msg[:300] if error_msg else 'Ver log de build'}"
            ),
            affected_component=f"Frontend/{component}/build",
            fix_code=fix,
            fix_description=(
                "Limpar cache, verificar versão do Node.js, checar variáveis de ambiente e "
                "executar o build com verbose para identificar o erro exato."
            ),
            repro_steps=[
                "Executar npm run build localmente",
                "Verificar o erro exato nas primeiras linhas do stack trace",
                "Verificar se há .env.production configurado",
            ],
            validation_steps=[
                "npm run build completa sem erros ou warnings críticos",
                "Bundle gerado em /dist ou /.next",
                "Aplicação abre corretamente em ambiente de staging",
            ],
        )

    def _diagnose_unhandled_exception(self, component, language, layer, error_msg, logs) -> AgentDiagnosis:
        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.HIGH,
            defect_layer=layer,
            root_cause=(
                f"Exceção não tratada (HTTP 500 / Unhandled Exception) no componente *{component}* "
                f"({language}, camada: {layer.value}). "
                f"A aplicação lançou uma exceção que não foi capturada, resultando em erro 500 ao usuário. "
                f"Stack trace: {error_msg[:400] if error_msg else 'Ver logs do servidor'}"
            ),
            affected_component=f"{component} ({language} / {layer.value})",
            fix_code=(
                f"// Adicionar tratamento de exceção global em {component}:\n\n"
                "// Node.js/Express — Error Handler Global (adicionar ÚLTIMO na cadeia de middlewares):\n"
                "app.use((err: Error, req: Request, res: Response, next: NextFunction) => {\n"
                "  logger.error('[Error Handler]', { message: err.message, stack: err.stack, url: req.url });\n"
                "  res.status(500).json({ error: 'Erro interno do servidor', traceId: req.headers['x-request-id'] });\n"
                "});\n\n"
                "// Python/FastAPI — Exception Handler Global:\n"
                "@app.exception_handler(Exception)\n"
                "async def global_exception_handler(request: Request, exc: Exception):\n"
                "    logger.error(f'Unhandled exception: {exc}', exc_info=True)\n"
                "    return JSONResponse(status_code=500, content={'error': 'Erro interno do servidor'})"
            ),
            fix_description=(
                "Adicionar global exception handler para capturar exceções não tratadas e retornar "
                "resposta HTTP estruturada ao invés de crash. Investigar a causa raiz no stack trace."
            ),
            repro_steps=[
                "Identificar o endpoint que retorna HTTP 500",
                "Verificar o stack trace completo nos logs do servidor",
                "Reproduzir localmente enviando o mesmo payload/request",
            ],
            validation_steps=[
                "Endpoint retorna resposta adequada (não mais 500)",
                "Stack trace não aparece mais nos logs de erro",
                "Global exception handler capturando exceções e retornando 500 estruturado",
            ],
        )

    def _diagnose_generic_code(self, component, language, layer, failure_type, error_msg, logs) -> AgentDiagnosis:
        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.MEDIUM,
            defect_layer=layer,
            root_cause=(
                f"Falha de código detectada no componente *{component}* "
                f"({language}, camada: {layer.value}). Tipo: {failure_type}. "
                f"Erro: {error_msg[:300] if error_msg else 'Ver stack trace nos logs'}."
            ),
            affected_component=f"{component} ({language} / {layer.value})",
            fix_code="# Inspecionar o stack trace completo nos logs para identificar a linha exata do erro.",
            fix_description="Analisar o stack trace e corrigir o código respeitando os padrões do repositório.",
            repro_steps=[
                "Identificar a linha exata do stack trace",
                "Reproduzir o erro localmente",
                "Corrigir mantendo os padrões de código do projeto",
            ],
            validation_steps=["Código corrigido sem o erro", "Testes unitários passando"],
        )
