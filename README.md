# snmpEndLog

**Monitor SNMP + Coletor de Logs Leve** — Alternativa leve ao Zabbix/LibreNMS.

Arquitetura híbrida: **Python** para coleta SNMP e recepção de logs, **Go** para o web server e dashboard.

## Funcionalidades

### Monitor SNMP
- Polling SNMPv2c e SNMPv3 (com autenticação e criptografia)
- Métricas: tráfego de interfaces (32/64-bit), CPU, memória, ping/latência
- Intervalo de polling configurável por equipamento
- Detecção automática de interfaces
- Cálculo de throughput com detecção de counter wrap

### Coletor de Logs
- Receptor Syslog UDP e TCP (porta 514)
- Parser RFC 3164 (BSD) e RFC 5424
- Busca full-text nos logs
- Filtros por host, severidade e período

### Dashboard
- Interface web premium com dark mode
- Gráficos de tráfego, CPU, memória e ping em tempo real (Chart.js)
- Visualizador de logs com busca instantânea
- Gerenciamento de equipamentos (adicionar, editar, remover)
- Autenticação local com JWT (bcrypt, custo 12)

## Requisitos

- Docker e Docker Compose

## Quick Start

```bash
# 1. Clone o repositório
git clone <repo-url> snmpEndLog
cd snmpEndLog

# 2. Configure o ambiente
cp .env.example .env
# Edite .env e defina senhas fortes para:
# - POSTGRES_PASSWORD
# - JWT_SECRET  
# - ADMIN_PASSWORD

# 3. Suba os serviços
docker compose up -d

# 4. Acesse o dashboard
# http://localhost:8080
# Login: admin / (senha definida em ADMIN_PASSWORD)
```

## Arquitetura

```
┌─────────────────┐     ┌─────────────────┐
│  Equipamentos   │     │  Equipamentos   │
│  (SNMP v2c/v3)  │     │  (Syslog)       │
└────────┬────────┘     └────────┬────────┘
         │                       │
         ▼                       ▼
┌─────────────────────────────────────────┐
│         Python Collector                 │
│  ┌──────────┐  ┌────────────────────┐   │
│  │  SNMP    │  │  Syslog Receiver   │   │
│  │  Poller  │  │  (UDP + TCP :514)  │   │
│  └────┬─────┘  └─────────┬──────────┘   │
└───────┼──────────────────┼──────────────┘
        │                  │
        ▼                  ▼
┌─────────────────────────────────────────┐
│  PostgreSQL + TimescaleDB                │
│  ┌─────────────┐  ┌──────────────────┐  │
│  │  Métricas   │  │  Logs (syslog)   │  │
│  │  (hypertbl) │  │  (hypertable)    │  │
│  └─────────────┘  └──────────────────┘  │
└───────────────────┬─────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────┐
│         Go Web Server (:8080)            │
│  ┌──────────┐  ┌────────┐  ┌────────┐  │
│  │Dashboard │  │  API   │  │  Auth  │  │
│  │  (HTML)  │  │ (JSON) │  │ (JWT)  │  │
│  └──────────┘  └────────┘  └────────┘  │
└─────────────────────────────────────────┘
```

## Stack

| Componente | Tecnologia |
|---|---|
| Coleta SNMP | Python 3.12 + pysnmp |
| Coleta Logs | Python 3.12 (asyncio) |
| Banco de dados | PostgreSQL 16 + TimescaleDB |
| Web Server | Go 1.22 (net/http) |
| Dashboard | HTML + Vanilla JS + Chart.js |
| Auth | JWT + bcrypt |
| Deploy | Docker Compose |

## Segurança

- ✅ Senhas com bcrypt (custo ≥ 12)
- ✅ JWT com expiração para sessões web
- ✅ SNMPv3 com autenticação + criptografia
- ✅ Rate limiting no login (5 tentativas/minuto)
- ✅ Prepared statements (proteção SQL injection)
- ✅ Template escaping (proteção XSS)
- ✅ Cookies HttpOnly + SameSite Strict
- ✅ Secrets via variáveis de ambiente (.env gitignored)

## Licença

MIT
