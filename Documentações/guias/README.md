# 🧭 Guias e Manuais Operacionais do Guardian

Este diretório reúne a documentação prática e passo a passo para instalação, configuração, operação diária e resolução de problemas no **Guardian**.

---

## 📚 Índice de Manuais

| Guia | Objetivo | Público-Alvo |
|:---|:---|:---|
| 💻 [**GUIA-EXECUCAO-LOCAL-E-DOCKER.md**](GUIA-EXECUCAO-LOCAL-E-DOCKER.md) | Como iniciar e rodar o Guardian localmente (Go nativo, `./guardian.sh`, Docker Compose ou Python). | Operadores, SREs e Desenvolvedores |
| 🎫 [**GUIA-INTEGRACAO-JIRA.md**](GUIA-INTEGRACAO-JIRA.md) | Configuração das credenciais Jira, mapeamento de contratos, custom fields e validação de abertura de chamados. | Administradores Jira e DevOps |
| 🎯 [**GUIA-SUPERVISOR-E-TARGETS.md**](GUIA-SUPERVISOR-E-TARGETS.md) | Como gerenciar targets dinamicamente (K8s Namespaces e Docker Stacks) pela UI e API REST. | Operadores e Engenheiros de Plataforma |
| 🤖 [**GUIA-SISTEMA-MULTI-AGENTE-RCA.md**](GUIA-SISTEMA-MULTI-AGENTE-RCA.md) | Funcionamento e regras heurísticas dos 5 agentes especialistas de análise de causa raiz (RCA). | Engenheiros de Software e QA |

---

## ⚡ Comandos Mais Utilizados

```bash
# Iniciar o sistema completo
./guardian.sh start

# Ver o status do supervisor
./guardian.sh status

# Parar a execução
./guardian.sh stop

# Rodar os testes de integração do Jira
python3 test_jira.py
```
