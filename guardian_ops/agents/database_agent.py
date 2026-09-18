# -*- coding: utf-8 -*-
"""
02. Data & Database Specialist Agent
======================================
Identificador : agent-database-specialist
Responsável   : Diagnóstico de falhas, otimização de performance e manutenção
                em múltiplos bancos de dados.
Escopo        : PostgreSQL, MySQL, SQL Server, MongoDB, Redis, Elasticsearch,
                Migrações (Flyway, Liquibase, Prisma, TypeORM, Alembic).
"""
from typing import Dict, Any

from guardian_ops.agents.base_agent import BaseAgent, AgentDiagnosis, Severity, DefectLayer


class DatabaseSpecialistAgent(BaseAgent):
    """
    Agente especializado em falhas e performance de bancos de dados.

    Cobre:
    - PostgreSQL: deadlocks, slow queries, connection pool exhausted, replication lag
    - MySQL/MariaDB: deadlocks, lock wait timeout, replication errors, max connections
    - SQL Server: blocking queries, transaction log full, tempdb issues
    - MongoDB: replica set issues, cursor timeout, index missing, WiredTiger cache
    - Redis: OOM, maxmemory policy, keyspace miss, cluster down
    - Elasticsearch: shard unassigned, disk watermark, mapping conflict
    - Migrações: Flyway, Liquibase, Alembic, Prisma, TypeORM failures
    """
    AGENT_ID   = "agent-database-specialist"
    AGENT_NAME = "Data & Database Specialist Agent"

    _DB_TRIGGERS = {
        "POSTGRES", "POSTGRESQL", "MYSQL", "MARIADB", "SQLSERVER", "MSSQL",
        "MONGODB", "MONGO", "REDIS", "ELASTICSEARCH", "ELASTIC", "DATABASE",
        "CONNECTION", "DEADLOCK", "SLOW QUERY", "MIGRATION", "FLYWAY",
        "LIQUIBASE", "ALEMBIC", "PRISMA", "TYPEORM", "SEQUELIZE",
        "CONNECTION REFUSED", "TOO MANY CONNECTIONS", "LOCK WAIT TIMEOUT",
        "REPLICATION", "REPLICA", "SHARD", "INDEX", "QUERY TIMEOUT",
    }

    def can_handle(self, payload: Dict[str, Any]) -> bool:
        source = payload.get("source", "").upper()
        failure = payload.get("failure_type", "").upper()
        error = payload.get("error_message", "").upper()
        combined = f"{source} {failure} {error}"
        return any(t in combined for t in self._DB_TRIGGERS)

    def analyze(self, payload: Dict[str, Any]) -> AgentDiagnosis:
        failure_type = payload.get("failure_type", "").upper()
        error_msg    = payload.get("error_message", "").upper()
        component    = payload.get("component", payload.get("identifier", "unknown-db"))
        logs         = payload.get("logs", payload.get("logs_snippet", ""))
        details      = payload.get("details", {})
        db_type      = self._detect_db_type(failure_type, error_msg, details)

        # Roteamento por tipo de falha
        if "DEADLOCK" in failure_type or "DEADLOCK" in error_msg:
            return self._diagnose_deadlock(component, db_type, logs)
        elif "CONNECTION" in failure_type or "CONN" in error_msg or "TOO MANY" in error_msg:
            return self._diagnose_connection_pool(component, db_type, logs)
        elif "SLOW" in failure_type or "TIMEOUT" in failure_type or "SLOW" in error_msg:
            return self._diagnose_slow_query(component, db_type, logs)
        elif "MIGRATION" in failure_type or any(t in failure_type for t in ["FLYWAY", "LIQUIBASE", "ALEMBIC", "PRISMA"]):
            return self._diagnose_migration(component, db_type, logs)
        elif "REPLICATION" in failure_type or "REPLICA" in failure_type:
            return self._diagnose_replication(component, db_type, logs)
        elif "REDIS" in failure_type or "REDIS" in error_msg or db_type == "Redis":
            return self._diagnose_redis(component, logs)
        elif "ELASTIC" in failure_type or "SHARD" in failure_type:
            return self._diagnose_elasticsearch(component, logs)
        elif "MONGO" in failure_type or db_type == "MongoDB":
            return self._diagnose_mongodb(component, logs)
        else:
            return self._diagnose_generic_db(component, db_type, failure_type, logs)

    def _detect_db_type(self, failure_type: str, error_msg: str, details: dict) -> str:
        """Identifica o tipo de banco de dados pelo contexto."""
        combined = f"{failure_type} {error_msg} {details.get('database_type', '')}".upper()
        if "POSTGRES" in combined or "PG" in combined:
            return "PostgreSQL"
        if "MYSQL" in combined or "MARIADB" in combined:
            return "MySQL/MariaDB"
        if "SQLSERVER" in combined or "MSSQL" in combined or "SQL SERVER" in combined:
            return "SQL Server"
        if "MONGO" in combined:
            return "MongoDB"
        if "REDIS" in combined:
            return "Redis"
        if "ELASTIC" in combined:
            return "Elasticsearch"
        return "Relacional (genérico)"

    # -------------------------------------------------------------------------
    # Diagnósticos por tipo de falha
    # -------------------------------------------------------------------------

    def _diagnose_deadlock(self, component, db_type, logs) -> AgentDiagnosis:
        if "PostgreSQL" in db_type:
            fix = """\
-- PostgreSQL: Identificar e matar processos em deadlock
SELECT pid, now() - pg_stat_activity.query_start AS duration, query, state
FROM pg_stat_activity
WHERE state = 'active' AND now() - pg_stat_activity.query_start > interval '5 minutes'
ORDER BY duration DESC;

-- Ver locks ativos:
SELECT blocked_locks.pid     AS blocked_pid,
       blocking_locks.pid    AS blocking_pid,
       blocked_activity.query AS blocked_statement
FROM pg_catalog.pg_locks blocked_locks
JOIN pg_catalog.pg_locks blocking_locks
  ON blocking_locks.locktype = blocked_locks.locktype
 AND blocking_locks.pid != blocked_locks.pid
JOIN pg_catalog.pg_stat_activity blocked_activity ON blocked_activity.pid = blocked_locks.pid
WHERE NOT blocked_locks.granted;

-- Encerrar processo bloqueante (substituir pelo PID real):
SELECT pg_terminate_backend(<blocking_pid>);"""
        elif "MySQL" in db_type:
            fix = """\
-- MySQL: Identificar deadlocks
SHOW ENGINE INNODB STATUS;
SHOW FULL PROCESSLIST;

-- Ver transações aguardando lock:
SELECT * FROM information_schema.INNODB_TRX;
SELECT * FROM information_schema.INNODB_LOCK_WAITS;

-- Matar o processo bloqueante:
KILL <process_id>;

-- Configurar timeout de lock para evitar esperas longas:
SET GLOBAL innodb_lock_wait_timeout = 10;"""
        else:
            fix = """\
-- Verificar processos e locks ativos no banco de dados.
-- Consulte a documentação do SGBD específico para comandos de diagnóstico de deadlock."""

        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.HIGH,
            defect_layer=DefectLayer.DATABASE,
            root_cause=(
                f"Deadlock detectado no banco de dados *{db_type}* do componente *{component}*. "
                "Duas ou mais transações estão aguardando recursos que a outra detém, criando um ciclo "
                "de dependência que o SGBD não pode resolver automaticamente. "
                "Causas típicas: múltiplas threads atualizando as mesmas linhas em ordem diferente, "
                "transactions de longa duração segurando locks, ausência de índices causando table-level locks."
            ),
            affected_component=f"Database/{db_type}/{component}",
            fix_code=fix,
            fix_description=(
                "Identificar os processos em deadlock, encerrar a transação bloqueante e revisar "
                "a lógica de acesso ao banco para garantir ordem consistente de bloqueio de recursos."
            ),
            repro_steps=[
                "Verificar logs do banco de dados no momento do erro",
                "Identificar as queries envolvidas no deadlock",
                "Executar queries de diagnóstico de locks (ver fix acima)",
            ],
            validation_steps=[
                "Ausência de erros de deadlock nos próximos 10 minutos",
                "Tempo de resposta das queries voltando ao normal",
                "Métricas de connection pool estabilizadas",
            ],
        )

    def _diagnose_connection_pool(self, component, db_type, logs) -> AgentDiagnosis:
        fix = f"""\
-- Diagnóstico de Connection Pool Esgotado ({db_type}):

-- PostgreSQL:
SELECT count(*), state FROM pg_stat_activity GROUP BY state;
SELECT max_conn, used FROM (
  SELECT count(*) used FROM pg_stat_activity
) t, (SELECT setting::int max_conn FROM pg_settings WHERE name='max_connections') t2;

-- MySQL:
SHOW STATUS LIKE 'Threads_connected';
SHOW VARIABLES LIKE 'max_connections';
SET GLOBAL max_connections = 500; -- Aumentar temporariamente

-- Verificar configuração do pool na aplicação (exemplo Spring Boot / HikariCP):
# spring.datasource.hikari.maximum-pool-size=20
# spring.datasource.hikari.minimum-idle=5
# spring.datasource.hikari.connection-timeout=30000

-- MongoDB:
db.serverStatus().connections

-- Redis:
redis-cli INFO clients"""

        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.CRITICAL,
            defect_layer=DefectLayer.DATABASE,
            root_cause=(
                f"Esgotamento do Connection Pool no banco de dados *{db_type}* do componente *{component}*. "
                "A aplicação está tentando abrir mais conexões do que o banco aceita ou do que o pool permite. "
                "Causas: conexões não liberadas (connection leak), aumento repentino de tráfego, "
                "pool subdimensionado para a carga atual, ou queries de longa duração segurando conexões."
            ),
            affected_component=f"Database/{db_type}/{component}/connection-pool",
            fix_code=fix,
            fix_description=(
                "Aumentar o max_connections no SGBD e o pool size na aplicação, "
                "investigar connection leaks no código e otimizar queries de longa duração."
            ),
            repro_steps=[
                "Verificar métricas de connections ativas no banco",
                "Checar logs da aplicação por erros 'too many connections' ou 'connection timeout'",
                "Identificar queries com duração > 30 segundos no SGBD",
            ],
            validation_steps=[
                "Número de conexões abaixo de 80% do max_connections",
                "Tempo de resposta das queries normalizado",
                "Ausência de erros de connection timeout na aplicação",
            ],
        )

    def _diagnose_slow_query(self, component, db_type, logs) -> AgentDiagnosis:
        fix = """\
-- Identificar Slow Queries:

-- PostgreSQL (habilitar pg_stat_statements):
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
SELECT query, calls, total_time/calls AS avg_ms, rows
FROM pg_stat_statements
ORDER BY avg_ms DESC LIMIT 10;

-- MySQL (analisar slow query log):
SHOW VARIABLES LIKE 'slow_query_log';
SET GLOBAL slow_query_log = 'ON';
SET GLOBAL long_query_time = 1;
-- Ver queries lentas:
SELECT * FROM mysql.slow_log ORDER BY query_time DESC LIMIT 10;

-- Analisar plano de execução de query específica:
EXPLAIN ANALYZE SELECT ... ;  -- PostgreSQL
EXPLAIN FORMAT=JSON SELECT ...;  -- MySQL

-- Criar índice para query frequente (exemplo):
CREATE INDEX CONCURRENTLY idx_tabela_coluna ON tabela(coluna);  -- PostgreSQL
CREATE INDEX idx_tabela_coluna ON tabela(coluna);  -- MySQL"""

        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.MEDIUM,
            defect_layer=DefectLayer.DATABASE,
            root_cause=(
                f"Slow queries detectadas no banco *{db_type}* do componente *{component}*. "
                "Queries com tempo de execução elevado degradam a performance geral do sistema. "
                "Causas frequentes: ausência de índices em colunas de filtro/join, "
                "tabelas sem VACUUM/ANALYZE (PostgreSQL), estatísticas desatualizadas, "
                "N+1 queries geradas por ORM, ou queries com full table scan."
            ),
            affected_component=f"Database/{db_type}/{component}/slow-query",
            fix_code=fix,
            fix_description=(
                "Identificar as queries mais lentas via pg_stat_statements ou slow query log, "
                "analisar o plano de execução com EXPLAIN ANALYZE e criar índices nas colunas críticas."
            ),
            repro_steps=[
                "Habilitar slow query log no SGBD",
                "Executar EXPLAIN ANALYZE na query mais lenta identificada",
                "Verificar se há full table scan no plano de execução",
            ],
            validation_steps=[
                "avg_ms das top queries reduzido em > 50% após criação de índices",
                "Ausência de sequential scan em tabelas grandes no EXPLAIN",
                "Tempo de resposta da API dependente normalizado",
            ],
        )

    def _diagnose_migration(self, component, db_type, logs) -> AgentDiagnosis:
        fix = """\
-- Falha em Migration de Banco de Dados:

-- Flyway:
# Ver status das migrations:
./mvnw flyway:info
# Reparar migration quebrada (marcar como falha):
./mvnw flyway:repair
# Executar migration pendente:
./mvnw flyway:migrate

-- Alembic (Python):
alembic current        # Ver versão atual
alembic history        # Ver histórico
alembic upgrade head   # Aplicar migrations pendentes
alembic downgrade -1   # Reverter última migration

-- Prisma:
npx prisma migrate status
npx prisma migrate deploy   # Aplicar em produção
npx prisma migrate reset    # Resetar (apenas dev!)

-- TypeORM:
npx typeorm migration:run
npx typeorm migration:revert

-- Verificar o erro exato da migration:
# Rolar back da migration quebrada e corrigir o SQL antes de re-aplicar."""

        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.HIGH,
            defect_layer=DefectLayer.DATABASE,
            root_cause=(
                f"Falha na execução de migration de banco de dados no componente *{component}* ({db_type}). "
                "Causas frequentes: sintaxe SQL inválida na migration, tentativa de alterar coluna com dados "
                "incompatíveis (ex: NOT NULL sem valor default em tabela populada), conflito de versão entre "
                "migrations, ou migration aplicada parcialmente deixando o schema em estado inconsistente."
            ),
            affected_component=f"Database/{db_type}/{component}/migration",
            fix_code=fix,
            fix_description=(
                "Verificar o estado atual da migration via CLI da ferramenta, identificar o SQL que falhou, "
                "corrigir o arquivo de migration e re-aplicar após um rollback seguro."
            ),
            repro_steps=[
                "Verificar o log de erro exato da ferramenta de migration",
                "Verificar o estado atual com flyway:info / alembic current / prisma migrate status",
                "Identificar a migration que falhou e o SQL problemático",
            ],
            validation_steps=[
                "Status da migration mostra Applied/Up para todas as versões",
                "Schema do banco correto após a migration",
                "Aplicação inicializa sem erros de schema incompatível",
            ],
        )

    def _diagnose_replication(self, component, db_type, logs) -> AgentDiagnosis:
        fix = """\
-- Diagnóstico de Replicação:

-- PostgreSQL Streaming Replication:
-- Na réplica:
SELECT now() - pg_last_xact_replay_timestamp() AS replication_delay;
SELECT * FROM pg_stat_replication;  -- No primário

-- MySQL/MariaDB Replication:
SHOW SLAVE STATUS\\G
-- Verificar Seconds_Behind_Master e Last_Error
-- Reiniciar replicação se parou:
STOP SLAVE; START SLAVE;
-- Ou pular erro pontual:
STOP SLAVE; SET GLOBAL SQL_SLAVE_SKIP_COUNTER=1; START SLAVE;

-- MongoDB Replica Set:
rs.status()
rs.printSlaveReplicationInfo()"""

        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.HIGH,
            defect_layer=DefectLayer.DATABASE,
            root_cause=(
                f"Problema de replicação detectado no banco *{db_type}* do componente *{component}*. "
                "Causas típicas: lag de replicação excessivo por carga elevada no primário, "
                "erro de aplicação de binlog/WAL na réplica, ou desconexão da réplica por timeout de rede."
            ),
            affected_component=f"Database/{db_type}/{component}/replication",
            fix_code=fix,
            fix_description="Verificar status de replicação e lag. Reiniciar o processo de replicação se necessário.",
            repro_steps=["Verificar lag de replicação via SHOW SLAVE STATUS ou pg_stat_replication"],
            validation_steps=[
                "Seconds_Behind_Master < 5 segundos",
                "Réplica em sincronia com o primário",
            ],
        )

    def _diagnose_redis(self, component, logs) -> AgentDiagnosis:
        fix = """\
-- Diagnóstico Redis:

# Status geral:
redis-cli INFO server
redis-cli INFO memory
redis-cli INFO stats

# Verificar uso de memória:
redis-cli INFO memory | grep used_memory_human

# Ver keys mais grandes:
redis-cli --bigkeys

# Verificar configuração de maxmemory e política de eviction:
redis-cli CONFIG GET maxmemory
redis-cli CONFIG GET maxmemory-policy

# Ajustar política de evição (recomendado para cache):
redis-cli CONFIG SET maxmemory-policy allkeys-lru

# Verificar clientes conectados:
redis-cli CLIENT LIST | wc -l

# Verificar latência:
redis-cli --latency"""

        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.HIGH,
            defect_layer=DefectLayer.DATABASE,
            root_cause=(
                f"Problema detectado no Redis do componente *{component}*. "
                "Causas frequentes: OOM (maxmemory atingido sem política de eviction configurada), "
                "alta latência por keyspace muito grande, cluster com shards indisponíveis, "
                "ou número excessivo de conexões abertas."
            ),
            affected_component=f"Cache/Redis/{component}",
            fix_code=fix,
            fix_description="Verificar uso de memória, configurar política de eviction e analisar keys grandes.",
            repro_steps=[
                "redis-cli INFO memory",
                "redis-cli --latency-history",
                "redis-cli INFO stats | grep evicted_keys",
            ],
            validation_steps=[
                "Uso de memória < 80% do maxmemory configurado",
                "Latência de comandos < 10ms (p99)",
                "Taxa de eviction zerada ou controlada",
            ],
        )

    def _diagnose_elasticsearch(self, component, logs) -> AgentDiagnosis:
        fix = """\
-- Diagnóstico Elasticsearch:

# Status do cluster:
curl -XGET 'http://localhost:9200/_cluster/health?pretty'

# Ver shards não alocados:
curl -XGET 'http://localhost:9200/_cat/shards?h=index,shard,prirep,state,unassigned.reason'

# Explicar por que um shard não foi alocado:
curl -XGET 'http://localhost:9200/_cluster/allocation/explain?pretty'

# Verificar uso de disco nos nós:
curl -XGET 'http://localhost:9200/_cat/nodes?v&h=name,diskUsed,diskAvail'

# Liberar shard (se disk watermark atingido):
curl -XPUT 'http://localhost:9200/_cluster/settings' \\
  -H 'Content-Type: application/json' \\
  -d '{"persistent":{"cluster.routing.allocation.disk.watermark.high":"90%"}}'"""

        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.HIGH,
            defect_layer=DefectLayer.DATABASE,
            root_cause=(
                f"Problema detectado no Elasticsearch do componente *{component}*. "
                "Causas comuns: shards não alocados por disk watermark atingido, "
                "nó saindo do cluster por OOM ou timeout, conflito de mapeamento de campo."
            ),
            affected_component=f"Search/Elasticsearch/{component}",
            fix_code=fix,
            fix_description="Verificar saúde do cluster, shards não alocados e uso de disco nos nós.",
            repro_steps=["curl _cluster/health", "curl _cat/shards?v"],
            validation_steps=[
                "cluster_status = green",
                "Todos os shards alocados (unassigned = 0)",
                "Disco < 75% nos nós",
            ],
        )

    def _diagnose_mongodb(self, component, logs) -> AgentDiagnosis:
        fix = """\
-- Diagnóstico MongoDB:

# Status do servidor:
db.serverStatus()

# Ver operações lentas:
db.currentOp({"secs_running": {$gt: 5}})

# Status do Replica Set:
rs.status()

# Ver índices de uma coleção:
db.<collection>.getIndexes()

# Criar índice para query frequente:
db.<collection>.createIndex({campo: 1}, {background: true})

# Verificar uso de WiredTiger cache:
db.serverStatus().wiredTiger.cache"""

        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.MEDIUM,
            defect_layer=DefectLayer.DATABASE,
            root_cause=(
                f"Problema detectado no MongoDB do componente *{component}*. "
                "Causas comuns: cursor timeout, operações lentas sem índice, "
                "WiredTiger cache cheio, ou problema no Replica Set."
            ),
            affected_component=f"Database/MongoDB/{component}",
            fix_code=fix,
            fix_description="Verificar operações lentas, índices e status do Replica Set.",
            repro_steps=["db.currentOp()", "rs.status()", "db.collection.explain().find(query)"],
            validation_steps=[
                "Operações lentas (> 5s) zeradas",
                "Replica Set em PRIMARY/SECONDARY estáveis",
            ],
        )

    def _diagnose_generic_db(self, component, db_type, failure_type, logs) -> AgentDiagnosis:
        return AgentDiagnosis(
            agent_id=self.AGENT_ID,
            agent_name=self.AGENT_NAME,
            severity=Severity.MEDIUM,
            defect_layer=DefectLayer.DATABASE,
            root_cause=(
                f"Falha de banco de dados detectada: *{db_type}* no componente *{component}*. "
                f"Tipo de falha: {failure_type}. Inspecionar logs do SGBD para diagnóstico detalhado."
            ),
            affected_component=f"Database/{db_type}/{component}",
            fix_code="# Verificar logs do banco de dados e status das conexões.",
            fix_description="Analisar logs do SGBD e métricas de performance para diagnóstico.",
            repro_steps=["Verificar logs do banco no momento do erro"],
            validation_steps=["Banco de dados respondendo normalmente às queries"],
        )
