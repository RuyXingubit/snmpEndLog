# Auditoria de Segurança — v5: Validação Estrita de Origem e Identidade Host nos Logs Syslog

**Data:** 16/11/2026

## 1. Diagnóstico do Incidente Reportado

Na tela `/logs`, o filtro `"Todos os hosts"` exibiu quatro IPs externos desconhecidos (`188.240.59.45`, `213.166.84.59`, `45.133.173.197`, `5.252.83.214`) ao lado dos equipamentos legítimos (`BNG-PROSERV`, `SW-CORE-S6730`).

### Causas Raiz Identificadas
1. **Registros legados pré-`v1.14.1` na janela de 7 dias (`GET /api/logs/hosts`):**
   - Antes do hardening de 11/11 (`v1.14.1`), o coletor Syslog aceitava pacotes na porta `5514` de qualquer origem, gravando `device_id = NULL` e `host = source_ip`.
   - Em 16/11 (5 dias depois), `HandleAPILogHosts` (`web/internal/handlers/logs.go`) continuou executando `SELECT DISTINCT host FROM logs WHERE time > NOW() - INTERVAL '7 days'` sem filtrar `device_id IS NOT NULL` nem checar vínculo com `devices` ativos (`enabled = TRUE`).
   - O mesmo ocorria nas demais queries de `logs` em `logs.go` (`/api/logs`, `/api/logs/stats`, `/api/logs/export`, `/api/logs/export/txt`), `ai.go` (`handleAddContext`), `dashboard.go` (`HandleDashboard`) e `status.go` (`HandleAPIStatus`).
2. **Vínculo de `host` no parser e guarda de inserção no banco (`collector/`):**
   - Mesmo após a allowlist por IP de origem na `v1.14.1`, `parse_syslog_message` (`collector/logs/parser.py`) aceitava cegamente o token de `HOSTNAME` do payload Syslog. Se um dispositivo cadastrado enviasse mensagem sem hostname começando com um IP externo ou com hostname adulterado, esse valor entrava na coluna `logs.host`.
   - `db.insert_logs` (`collector/db.py`) não possuía trava defensiva impedindo inserção de linhas com `device_id IS NULL`.

---

## 2. Defesa em 3 Camadas Implementada

1. **Coletor Syslog (`collector/db.py`, `collector/logs/parser.py`, `collector/logs/receiver.py`):**
   - Novo método `db.get_allowed_devices()` retorna mapa `IP -> {id, hostname, sys_name}` somente de dispositivos com `enabled = TRUE AND ip_address IS NOT NULL`.
   - `apply_registered_device_identity()` em `collector/logs/parser.py` vincula o log ao `device_id` autenticado e valida o campo `host` contra `{hostname, sys_name}` do equipamento cadastrado. Se o token capturado for um IP estrangeiro (ou tag app tipo `sshd[123]:`), ele é preservado no corpo `message` (ou `app_name`) e `host` é normalizado para `sys_name` / `hostname` cadastrado.
   - `db.insert_logs()` descarta defensivamente qualquer linha com `device_id is None`.

2. **Dashboard e API Web (`web/internal/handlers/logs.go`, `ai.go`, `dashboard.go`, `status.go`):**
   - Helper compartilhado `buildLogsFilterWhere` + constante `registeredDeviceLogsWhere`:
     `device_id IS NOT NULL AND EXISTS (SELECT 1 FROM devices d WHERE d.id = logs.device_id AND d.enabled = TRUE)`
   - Aplicado uniformemente em `/api/logs/hosts`, `/api/logs`, `/api/logs/stats`, `/api/logs/export`, `/api/logs/export/txt`, contexto IA (`/api/ai/sessions/{id}/context`), `Dashboard` e `/api/status`.

3. **Regra de Não-Destruição de Dados:**
   - Nenhuma migration destrutiva (`DELETE FROM logs`) foi criada automaticamente; a limpeza de linhas antigas (`device_id IS NULL`) fica condicionada à aprovação explícita do usuário na VPS.
