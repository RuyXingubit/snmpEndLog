# NMS

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
git clone <repo-url> NMS
cd NMS

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

## Deploy em Produção (SSL via Let's Encrypt + Caddy)

O repositório possui imagens pré-compiladas no Docker Hub (`xingubit`). O sistema pode ser colocado em produção facilmente:

1. Baixe o `docker-compose.prod.yml`:
   ```bash
   wget https://raw.githubusercontent.com/xingubit/snmpendlog/main/docker-compose.prod.yml -O docker-compose.yml
   ```

2. Crie e edite o arquivo `.env`:
   ```bash
   touch .env
   # Adicione e edite as seguintes variáveis:
   # DOMAIN=seu-dominio.com
   # ACME_EMAIL=seu-email@dominio.com (usado para gerar o certificado SSL)
   ```
   
   **Gerando senhas fortes automaticamente:**
   Para garantir a máxima segurança em produção, não invente as senhas. Use os comandos abaixo no terminal do Linux para gerar senhas altamente seguras e copie os resultados para o seu arquivo `.env`:
   
   ```bash
   # Gerar POSTGRES_PASSWORD:
   echo "POSTGRES_PASSWORD=$(openssl rand -hex 32)" >> .env
   
   # Gerar JWT_SECRET:
   echo "JWT_SECRET=$(openssl rand -hex 64)" >> .env
   
   # Gerar ADMIN_PASSWORD (a senha que você usará para entrar no painel web):
   echo "ADMIN_PASSWORD=$(openssl rand -base64 12)" >> .env
   ```

3. Suba o sistema:
   ```bash
   docker-compose up -d
   ```
O Caddy cuidará automaticamente da criação do certificado SSL. Basta acessar `https://seu-dominio.com`.

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
