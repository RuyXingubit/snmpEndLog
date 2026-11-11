# Auditoria de Segurança — v3: Decisões Aprovadas e Plano de Execução

**Data:** 11/11/2026  

## Decisões Aprovadas pelo Usuário
1. **Alarmes (`viewer`):** Apenas visualizar alarmes. Somente `admin` pode resolver alarmes (`POST /api/alarms/{id}/resolve`).
2. **Análise com IA (`/ai` e `/api/ai/*`):** Restrito exclusivamente a `admin` (tanto na UI quanto no backend via `RequireAdmin` / `RequireAdminAPI`).
3. **SSH (`22/tcp`) na VM:** Restrito no `UFW` apenas aos blocos IP da Proserv (`143.208.136.0/22`, `177.53.136.0/22`, `2804:192c::/32` + IP da sessão atual por segurança anti-lockout) + `fail2ban` ativo.

## Escopo de Correções de Código + Testes Unitários
1. **C1 — RBAC em `/devices/{id}/edit` e sanitização para `viewer`:**
   - Exigir `admin` em `HandleDeviceEditPage` e omitir `Community` e `WebhookToken` para usuários não-admin em `HandleDevices` e `HandleDeviceDetail`.
2. **C2 — Allowlist de IP em memória no Coletor Syslog (`collector/logs/receiver.py`):**
   - Cache em memória (`ip -> device_id`) atualizado periodicamente a partir de `devices WHERE enabled = TRUE`, descartando imediatamente (`O(1)`) qualquer pacote UDP/TCP de IP não cadastrado sem executar query por pacote.
3. **C3 — Correção de Stored XSS + Validação de `target_ip` e `MaxBytesReader` no Webhook:**
   - Escapar todos os campos dinâmicos em `web/static/js/app.js` e `web/static/js/logs.js`.
   - Validar `target_ip` (IP válido ou hostname seguro até 253 chars) e limitar body (`64 KB`) em `webhooks.go`.
4. **A1 & A2 — Rate Limiter por IP Real + Cookie `Secure` atrás de Proxy (`nms-caddy`):**
   - Extrair IP real sem porta efêmera (`X-Forwarded-For` quando vindo de proxy privado confiável) com teto de memória no `RateLimiter`.
   - Ativar `Secure: true` no cookie JWT quando `r.TLS != nil` ou `X-Forwarded-Proto == "https"`.
5. **A3 — RBAC nas rotas `/ai`, `/api/ai/*`, `/api/status` e `/api/alarms/{id}/resolve`:**
   - `/ai` -> `RequireAdmin`; `/api/ai/*`, `/api/status` e `/api/alarms/` -> `RequireAdminAPI` + exigir `POST` para resolver alarme; ocultar botões de Resolver e Análise IA para `viewer`.
6. **A4 — Validação de Sessão/Role em Tempo Real no Banco:**
   - Confirmar existência do usuário e atualizar `claims.Role` com o valor atual da tabela `users` no middleware de autenticação (revogação imediata ao deletar ou rebaixar conta).
7. **Defesa em Profundidade (M1–M4):**
   - Middleware de Security Headers HTTP (`X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, `Permissions-Policy`) + proteção `Origin`/`Sec-Fetch-Site` em métodos mutáveis.
   - Sanitização de mensagens `err.Error()` nas respostas HTTP/JSON.
   - Proteção contra CSV Formula Injection em `HandleAPILogExport`.
   - Execução com usuário não-root (`nms`) em `web/Dockerfile`.
