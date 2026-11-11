# Auditoria e Hardening de Segurança — NMS (`snmpEndLog`)

**Data de Consolidação:** 11/11/2026  
**Histórico Incremental:** `docs/brainstorming/auditoria_seguranca/v1..v4`

---

## 1. Políticas de Acesso e Infraestrutura Definidas

1. **Alarmes (`viewer` vs `admin`):**
   - Usuários `viewer` possuem permissão **somente leitura** sobre alarmes.
   - A resolução manual (`POST /api/alarms/{id}/resolve`) é exclusiva do perfil `admin` (botão oculto na UI e bloqueado com `403` no backend).
2. **Análise com IA (`/ai` e `/api/ai/*`):**
   - Restrita exclusivamente ao perfil `admin` na navegação, telas e endpoints de API (`RequireAdmin` / `RequireAdminAPI`), protegendo sessões e cota de `GEMINI_API_KEY`.
3. **Firewall (`UFW`) + `fail2ban` na VM de Produção (`143.208.136.39`):**
   - **SSH (`22/tcp`):** restrito aos blocos ASN Proserv (`143.208.136.0/22`, `177.53.136.0/22`, `2804:192c::/32`) + `fail2ban` em modo agressivo (`bantime = 24h`, `maxretry = 5`, `backend = systemd`).
   - **Syslog (`5514/udp` e `5514/tcp`):** restrito aos blocos da Proserv e faixas privadas/CGNAT de equipamentos (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, `100.64.0.0/10`).
   - **Web (`80/tcp`, `443/tcp`):** abertas via `nms-caddy`.
   - **PostgreSQL/TimescaleDB (`5432/tcp`):** isolado em `127.0.0.1`.

---

## 2. Vulnerabilidades Corrigidas na Aplicação

| ID | Severidade | Vulnerabilidade | Solução Implementada |
|---|---|---|---|
| **C1** | Crítica | Bypass de RBAC em `/devices/{id}/edit` expondo `Community` SNMP e `WebhookToken` para `viewer` | Verificação `isAdminUser` em `HandleDeviceDetail`, `HandleDeviceEditPage` e ações `POST`, além de `sanitizeDeviceForRole` removendo segredos para não-admins. |
| **C2** | Crítica | Coletor Syslog (`5514`) aceitando pacotes de IPs não cadastrados e executando query síncrona por pacote | Cache em memória `_allowed_ips` (`O(1)`) atualizado assincronamente a cada `30s` (`db.get_allowed_device_ips()`), descartando IPs desconhecidos em UDP e encerrando conexões TCP imediatamente. |
| **C3** | Alta | Stored XSS no modal global de alarmes (`app.js`) e `target_ip` sem validação no webhook | `escapeHtml` aplicado em todos os campos dinâmicos (`app.js`, `logs.js`) + `IsValidWebhookTarget`, validação de `packet_loss` (`0..100`) e `MaxBytesReader` (`64 KB`) em `webhooks.go`. |
| **A1** | Alta | Rate Limiter de login inoperante por usar `r.RemoteAddr` com porta efêmera e IP do container Caddy | `middleware.ClientIP(r)` extrai IP real sem porta (lendo `X-Forwarded-For` da direita para a esquerda apenas de proxies privados confiáveis) e limita o mapa a `10.000` chaves. |
| **A2** | Alta | Cookie JWT sem flag `Secure` atrás de proxy reverso TLS (`nms-caddy`) | `middleware.IsSecureRequest(r)` valida `r.TLS != nil` ou `X-Forwarded-Proto: https` de proxy confiável, ativando `Secure: true` e `Strict-Transport-Security`. |
| **A3** | Alta | Ausência de RBAC/método em `/ai`, `/api/ai/*`, `/api/status` e `/api/alarms/{id}/resolve` | Rotas protegidas por `RequireAdmin` / `RequireAdminAPI`, validação `POST` em `HandleAPIAlarmResolve` e ocultação condicional nos templates/JS. |
| **A4** | Alta | Sessões JWT permanecendo válidas por até 24h após exclusão de usuário ou rebaixamento de role | `RequireAuth` e `RequireAPI` consultam `auth.LookupActiveUser` no banco, revogando imediatamente contas removidas e atualizando `claims.Role` em tempo real. |
| **M1** | Média | Vazamento de `err.Error()` interno do PostgreSQL/APIs em respostas JSON/HTTP | Logs detalhados mantidos no servidor (`log.Printf`) e mensagens genéricas seguras retornadas ao cliente. |
| **M2** | Média | Ausência de Security Headers HTTP, verificação CSRF `Origin`/`Sec-Fetch-Site` e logout via `GET` | Middlewares globais `SecurityHeaders` e `CSRFOriginCheck`, limites `http.MaxBytesReader` e `/logout` restrito a `POST`. |
| **M3** | Média | CSV Formula Injection na exportação de logs (`/api/logs/export`) | `SanitizeCSVField` neutraliza células iniciadas por `=`, `+`, `-`, `@`, `\t` e `\r`. |
| **M4** | Baixa | Processo `nms-web` executando como `root` no container | `web/Dockerfile` atualizado para executar como usuário não-privilegiado `nms` (`UID 10001`). |

---

## 3. Cobertura de Testes Automatizados
- `web/internal/middleware/middleware_test.go`: `ClientIP`, `IsSecureRequest`, `RateLimiter`, `SecurityHeaders`, `CSRFOriginCheck` e revogação/sincronização de sessão em `RequireAuth`/`RequireAPI`.
- `web/internal/handlers/security_test.go`: RBAC em equipamentos, alarmes, IA e status, sanitização de segredos, `/logout` `POST`-only, validação de webhook (`IsValidWebhookTarget` + limite de payload), `SanitizeCSVField` e parse de todos os templates HTML.
- `collector/tests/test_receiver.py`: normalização IPv4/IPv6, descarte silencioso UDP/TCP de IPs não autorizados e atualização assíncrona de allowlist.
