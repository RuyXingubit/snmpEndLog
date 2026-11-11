# Auditoria de Segurança — v4: Implementação Concluída, Cobertura de Testes e Hardening da VM

**Data:** 11/11/2026  

## 1. Correções Implementadas no Código (Aguardando Aprovação para Commit/Deploy)

1. **C1 — Correção de RBAC em `/devices/{id}/edit` e Sanitização de Segredos (`web/internal/handlers/devices.go`):**
   - Bloqueio com `403 Forbidden` para usuários não-admin tanto no roteamento de `HandleDeviceDetail` quanto diretamente em `HandleDeviceEditPage` e endpoints `POST` (`HandleDeviceAdd`, `HandleDeviceEdit`, `HandleDeviceDelete`, `HandleDeviceGenerateToken`).
   - `sanitizeDeviceForRole` remove `Community` e `WebhookToken` da struct `Device` sempre que `Role != "admin"`.
   - Modais administrativos em `web/templates/devices.html` restritos a `{{if eq .Role "admin"}}`.

2. **C2 — Allowlist de IP em Memória `O(1)` no Coletor Syslog (`collector/db.py`, `collector/logs/receiver.py`):**
   - Novo método `db.get_allowed_device_ips()` busca `SELECT id, host(ip_address) FROM devices WHERE enabled = TRUE`.
   - `SyslogReceiver` carrega e atualiza periodicamente (a cada 30s via `run_in_executor`, sem travar o loop `asyncio`) o mapa `_allowed_ips` normalizado (`IPv4`, `IPv6` e `IPv4-mapped IPv6`).
   - Pacotes UDP e conexões TCP de IPs não cadastrados são descartados/fechados imediatamente em `O(1)` antes do parse e sem disparar queries síncronas por pacote.

3. **C3 — Correção de Stored XSS + Validação Estrita no Webhook (`web/static/js/app.js`, `logs.js`, `web/internal/handlers/webhooks.go`):**
   - Escapamento HTML completo (`escapeHtml`) em `renderAlarmsTable` (`app.js`) e `loadLogStats` (`logs.js`).
   - `IsValidWebhookTarget` valida estritamente `target_ip` (IP válido ou hostname RFC-1123 até 253 caracteres), valida `packet_loss` entre `0` e `100`, e limita o body a `64 KB` (`http.MaxBytesReader`).

4. **A1 & A2 — Rate Limiter por IP Real + Flag `Secure` Atrás do Caddy (`web/internal/middleware/middleware.go`, `web/internal/handlers/auth.go`):**
   - `ClientIP(r)` remove a porta efêmera (`net.SplitHostPort`) e extrai o IP real de `X-Forwarded-For` (da direita para a esquerda, impedindo spoofing) apenas quando o peer TCP imediato é proxy privado/loopback confiável (`nms-caddy`).
   - Teto de memória `MaxRateLimiterKeys = 10000` no `RateLimiter`.
   - `IsSecureRequest(r)` habilita `Secure: true` no cookie JWT e HSTS quando `r.TLS != nil` ou `X-Forwarded-Proto: https` via proxy confiável.

5. **A3 — RBAC Estrito em Alarmes, IA e Status (`web/main.go`, `api_alarms.go`, `ai.go`, `status.go`, `layout.html`, `logs.html`, `app.js`, `alarms.js`):**
   - `POST /api/alarms/{id}/resolve`: exige método `POST` (`405`), ID numérico positivo (`400`) e perfil `admin` (`403`); botão `✅ Resolver` oculto para `viewer`.
   - `/ai` e `/api/ai/*`: restritos exclusivamente a `admin` no roteador (`RequireAdmin` / `RequireAdminAPI`), nos handlers e nos templates (`layout.html`, `logs.html`).
   - `/api/status`: restrito a `admin` (`RequireAdminAPI` + defesa em profundidade no handler).

6. **A4 — Revogação Imediata de Sessão/Role no Banco (`web/internal/auth/auth.go`, `web/internal/middleware/middleware.go`):**
   - `RequireAuth` e `RequireAPI` consultam `auth.LookupActiveUser` (`SELECT username, role FROM users WHERE id = $1`), invalidando imediatamente cookies de usuários removidos e sincronizando rebaixamentos de `admin` -> `viewer` em tempo real.

7. **M1, M2, M3 & M4 — Defesa em Profundidade:**
   - Sanitização de `err.Error()` em `logs.go`, `metrics.go`, `ai.go`, `api_alarms.go` e `devices.go`.
   - Middlewares globais `SecurityHeaders` (`X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, `Permissions-Policy`, `Content-Security-Policy`, `Strict-Transport-Security`) e `CSRFOriginCheck` (`Sec-Fetch-Site` + `Origin`), além de `/logout` convertido para `POST`.
   - `SanitizeCSVField` contra CSV Formula Injection (`=`, `+`, `-`, `@`, `\t`, `\r`) em `logs.go`.
   - Container `nms-web` configurado para rodar sob usuário sem privilégios `nms` (`UID 10001`) em `web/Dockerfile`.

## 2. Resultado das Suítes de Testes Unitários
- **Go (`web/`):** `go vet ./...`, `go build .` e `go test -v ./...` — **100% PASS** (`handlers` + `middleware`, incluindo parsing de todos os 10 templates HTML).
- **Python (`collector/`):** `python3 -m unittest tests/test_receiver.py -v` — **100% PASS** (4 cenários cobrindo UDP drop/allow, TCP close/allow, normalização IPv4/IPv6 e refresh assíncrono).
