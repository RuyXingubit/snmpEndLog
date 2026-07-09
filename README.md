# NMS

**Monitor SNMP + Coletor de Logs + Análise com IA** — Alternativa leve ao Zabbix/LibreNMS.

Arquitetura híbrida: **Python** para coleta SNMP e recepção de logs, **Go** para o web server e dashboard, **Gemini AI** para análise inteligente de logs.

## Funcionalidades

### Monitor SNMP
- Polling SNMPv2c e SNMPv3 (com autenticação e criptografia)
- Métricas: tráfego de interfaces (32/64-bit), CPU, memória, ping/latência
- Intervalo de polling configurável por equipamento
- Detecção automática de interfaces
- Cálculo de throughput com detecção de counter wrap

### Coletor de Logs
- Receptor Syslog UDP e TCP (porta 514)
- Parser RFC 3164 (BSD) e RFC 5424 (com suporte a Huawei VRP)
- Busca full-text nos logs
- Filtros por host, severidade e período
- Exportação CSV

### Análise com IA (Gemini)
- Página dedicada de análise com sessões persistentes
- Contexto acumulável de múltiplos dispositivos e períodos
- Chat com IA para perguntas sobre os logs
- Respostas renderizadas em Markdown
- Chave de API segura no backend

### Dashboard
- Interface web premium com dark mode
- Gráficos de tráfego, CPU, memória e ping em tempo real (Chart.js)
- Visualizador de logs com busca instantânea e scroll horizontal
- Gerenciamento de equipamentos (adicionar, editar, remover)
- Página de status com uso de disco e saúde do sistema
- Autenticação local com JWT (bcrypt, custo 12)

## Requisitos

- Docker e Docker Compose
- Chave de API do [Google AI Studio](https://aistudio.google.com/apikey) (opcional, para análise IA)

---

## Instalação com Docker (Recomendado)

As imagens pré-compiladas estão no Docker Hub (`xingubit`). Não é necessário baixar o código fonte.

### 1. Crie um diretório e baixe o docker-compose

```bash
mkdir ~/nms && cd ~/nms

# Download do docker-compose de produção (com Caddy + SSL)
wget -O docker-compose.yml \
  https://raw.githubusercontent.com/RuyXingubit/snmpEndLog/main/docker-compose.prod.yml
```

### 2. Configure o `.env`

```bash
# Domínio e email para SSL
echo "DOMAIN=nms.seudominio.com" >> .env
echo "ACME_EMAIL=seu-email@dominio.com" >> .env

# Senhas (geradas automaticamente)
echo "POSTGRES_PASSWORD=$(openssl rand -hex 32)" >> .env
echo "JWT_SECRET=$(openssl rand -hex 64)" >> .env
echo "ADMIN_PASSWORD=$(openssl rand -base64 12)" >> .env

# Opcional: Gemini AI para análise de logs
echo "GEMINI_API_KEY=sua_chave_aqui" >> .env
```

> **⚠️ Importante:** Anote a senha do admin antes de continuar:
> ```bash
> grep ADMIN_PASSWORD .env
> ```

### 3. Suba os serviços

```bash
docker compose up -d
```

### 4. Acesse

- `https://nms.seudominio.com`
- Login: `admin` / (senha do `ADMIN_PASSWORD`)

O Caddy cuida automaticamente do certificado SSL via Let's Encrypt.

### Atualizando

```bash
docker compose pull
docker compose down && docker compose up -d
```

---

## Instalação sem SSL (localhost/rede local)

Se não precisa de SSL (ex: acesso via IP local):

```bash
mkdir ~/nms && cd ~/nms

# Download do docker-compose de desenvolvimento
wget -O docker-compose.yml \
  https://raw.githubusercontent.com/RuyXingubit/snmpEndLog/main/docker-compose.prod.yml
```

Remova o serviço `caddy` do `docker-compose.yml` e adicione ao serviço `web`:

```yaml
    ports:
      - "8080:8080"
```

Acesse via `http://IP_DO_SERVIDOR:8080`.

---

## Configuração Syslog

### MikroTik

```
/system logging action set nms target=remote remote=IP_DO_NMS remote-port=5514 bsd-syslog=yes
/system logging add topics=critical action=nms
/system logging add topics=error action=nms
/system logging add topics=warning action=nms
/system logging add topics=info action=nms
```

### Huawei VRP (ME60 / S6730)

```
system-view
info-center loghost IP_DO_NMS port 5514 facility local7
info-center source default channel loghost log level informational
```

---

## Build do Código Fonte (Desenvolvimento)

Para quem deseja compilar do código fonte:

```bash
# 1. Clone o repositório
git clone https://github.com/RuyXingubit/snmpEndLog.git nms
cd nms

# 2. Configure o ambiente
cp .env.example .env
# Edite .env e defina senhas fortes

# 3. Build e suba
docker compose up -d --build

# 4. Acesse: http://localhost:8080
```

---

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
│  │Dashboard │  │  API   │  │Gemini  │  │
│  │  (HTML)  │  │ (JSON) │  │  AI    │  │
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
| Análise IA | Gemini Flash (Google AI) |
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
- ✅ API key do Gemini no backend (nunca exposta ao frontend)

## Licença

MIT
