# Auditoria Completa de Segurança — NMS (`snmpEndLog`) & Infraestrutura VM (v1)

**Data:** 11/11/2026  
**Escopo:** Infraestrutura da VM (`143.208.136.39`), Backend Go (`web/`), Coletor Python (`collector/`), Banco TimescaleDB (`db/`), Frontend JS/Templates e Containers Docker.

---

## 1. Infraestrutura da VM (`143.208.136.39`)

### Achados
1. **SSH (`22/tcp`) com autenticação por senha exposta à internet:**
   - Sofrendo ataques contínuos de força bruta (`79.72.59.54`, `2.27.32.182`, etc.) contra usuários comuns (`root`, `admin`, `oracle`, `ubuntu`).
   - Chave pública administrativa (`id_ed25529.pub`) ainda não cadastrada em `~/.ssh/authorized_keys` na VM (`Permission denied (publickey,password)` em modo não-interativo).
2. **Syslog (`5514/udp` e `5514/tcp`) exposto em `0.0.0.0` (`network_mode: host`):**
   - Em `docker-compose.prod.yml`, o `nms-collector` usa a pilha de rede do host para enxergar o IP real de origem dos pacotes UDP Syslog. Sem filtro UFW/Proxmox Firewall, qualquer IP da internet alcança a porta `5514`.
3. **Ausência de `fail2ban`:**
   - No Ubuntu 24.04, o `fail2ban` precisa usar `backend = systemd` para ler diretamente do `systemd-journald` e deve incluir `ignoreip` com os blocos próprios da Proserv (`143.208.136.0/22`, `177.53.136.0/22`, `2804:192c::/32`) para evitar auto-bloqueio.

---

## 2. Vulnerabilidades na Aplicação (Código-Fonte)

### 🔴 Críticas

#### C1. Bypass de RBAC em `/devices/{id}/edit` — Vazamento de Credenciais SNMP e `WebhookToken`
- **Onde:** `web/main.go:64`, `web/internal/handlers/devices.go:112-124` e `devices.go:378-416`.
- **Causa:** A rota `/devices/` está registrada no `authMux` (acessível por `viewer`). Dentro de `HandleDeviceDetail`, requisições para `/devices/{id}/edit` são desviadas para `HandleDeviceEditPage(w, r)` sem checar `claims.Role == "admin"`.
- **Impacto:** Qualquer usuário `viewer` que digite `/devices/1/edit` na URL visualiza a **Community SNMP em texto claro** e o **`WebhookToken` Bearer** do roteador.
- **Correção:** Exigir `claims.Role == "admin"` em `HandleDeviceEditPage` (e na rota) e omitir `Community` e `WebhookToken` das queries/structs quando o usuário não for `admin`.

#### C2. Coletor Syslog sem Allowlist Real + Risco de DoS no Banco e Disco
- **Onde:** `collector/logs/receiver.py:3-7`, `receiver.py:55-60`, `receiver.py:95-100`.
- **Causa:** O docstring afirma possuir *"IP allowlist: only accepts syslog from IPs registered as devices"*, porém o código apenas chama `device_id = db.resolve_device_id_by_ip(source_ip)` e **nunca descarta** quando `device_id is None`. Além disso, executa um `SELECT` síncrono no PostgreSQL a cada pacote UDP recebido na thread do loop `asyncio`.
- **Impacto:** Qualquer host externo pode inundar a porta `5514` com logs falsos (Log Spoofing / Disk Fill) e esgotar o pool de conexões do PostgreSQL.
- **Correção:** Manter um cache em memória (`dict[ip, device_id]`) atualizado a cada 30–60s com os IPs de `devices` ativos (`enabled = TRUE`) e descartar imediatamente em memória (`O(1)`, sem query por pacote) qualquer pacote vindo de IP não autorizado.

#### C3. Stored XSS no Modal Global de Alarmes + Falta de Validação em `webhooks.go`
- **Onde:** `web/static/js/app.js:156-173`, `web/static/js/logs.js:58-61` e `web/internal/handlers/webhooks.go:63-66`.
- **Causa:**
  1. Em `app.js`, `renderAlarmsTable` interpola `match[1]`, `a.name` e `a.entity_id` diretamente em `tbody.innerHTML` **sem `escapeHtml()`**.
  2. Em `webhooks.go`, `payload.TargetIP` aceita qualquer string não vazia (sem validar IP/hostname) e grava em `alarms.entity_id`. Um token vazado ou descrição de interface SNMP maliciosa injeta JavaScript persistente na sessão do `admin`.
- **Correção:** Aplicar `escapeHtml()` em todos os campos interpolados em `app.js` e `logs.js`, e validar formato estrito de IP/hostname + tamanho máximo em `webhooks.go`.

---

### 🟠 Altas

#### A1. Rate Limiter de Login Inoperante atrás do Proxy Reverso (`nms-caddy`)
- **Onde:** `web/internal/handlers/auth.go:210-216`.
- **Causa:** Usa `ip := r.RemoteAddr`, que:
  1. Inclui a porta TCP efêmera (`ip:porta`), gerando uma chave diferente a cada tentativa de login (nunca bloqueia força bruta no `/login` e consome memória no mapa `attempts`);
  2. Enxerga o IP interno do container `nms-caddy` (`172.18.0.x`) em vez do IP real do cliente (`X-Forwarded-For`).
- **Correção:** Extrair o IP real com `net.SplitHostPort(r.RemoteAddr)` e, quando a conexão vier de IP privado confiável (rede Docker do Caddy), ler o último/cliente de `X-Forwarded-For` validado com `net.ParseIP`, além de limitar o tamanho máximo da tabela do `RateLimiter`.

#### A2. Cookie JWT sem Flag `Secure` em Produção HTTPS
- **Onde:** `web/internal/handlers/auth.go:231-239`.
- **Causa:** Define `Secure: r.TLS != nil`. Como o TLS termina no `nms-caddy` e chega em HTTP interno na porta `8080` do `nms-web`, `r.TLS` é sempre `nil` em produção, deixando o cookie de sessão sem `Secure`.
- **Correção:** Definir `Secure: r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")`.

#### A3. Endpoints `/api/*` sem Segregação de Role (`admin` vs `viewer`) e sem Checagem de Método HTTP
- **Onde:** `web/main.go:95-112`, `web/internal/handlers/api_alarms.go:108`, `status.go:19`, `ai.go:27-158`.
- **Causa:**
  1. `/api/status` está liberado para `viewer`, embora `/status` seja restrito a `RequireAdmin`.
  2. `/api/alarms/{id}/resolve` não valida `r.Method == http.MethodPost` (aceita `GET`) e permite `viewer` resolver alarmes.
  3. `/api/ai/sessions/*` permite que `viewer` leia, apague sessões de admin (`DELETE`) e consuma cota da `GEMINI_API_KEY`.
- **Correção:** Proteger `/api/status` com `RequireAdminAPI`, exigir `POST` em `/api/alarms/{id}/resolve`, e alinhar com o produto a permissão de `viewer` para resolver alarmes e acessar IA.

#### A4. Sessões JWT Não Revogadas ao Excluir ou Rebaixar Usuário
- **Onde:** `web/internal/auth/auth.go:144-160`, `web/internal/middleware/middleware.go:26-73`.
- **Causa:** O JWT vale por 24h apenas pela assinatura HMAC. Se um usuário for removido ou alterado de `admin` para `viewer`, o token antigo continua válido como `admin` até expirar.
- **Correção:** Consultar (ou validar via cache curto / lookup leve em `users` por `claims.UserID`) se o usuário ainda existe e usar a `role` atualizada do banco de dados em cada requisição autenticada.

---

### 🟡 Médias / Defesa em Profundidade

1. **Exposição de Erros Internos (`err.Error()`) nas Respostas HTTP/JSON:**
   - **Onde:** `logs.go`, `metrics.go`, `ai.go`, `api_alarms.go`, `devices.go`.
   - **Correção:** Registrar o erro detalhado em `log.Printf` no servidor e devolver mensagem genérica padronizada ao cliente.
2. **Falta de Limite de Payload (`http.MaxBytesReader`) e Headers de Segurança HTTP:**
   - **Correção:** Limitar corpo das requisições (ex.: `64 KB` para webhooks/forms, `1 MB` geral) para impedir OOM na VM de 2 GB RAM, e adicionar middleware de headers (`X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: strict-origin-when-cross-origin`).
3. **CSV Formula Injection na Exportação de Logs (`logs.go:269-281`):**
   - **Correção:** Prefixar células iniciadas por `=`, `+`, `-`, `@`, `\t`, `\r` com apóstrofo `'` na exportação CSV.
4. **Comparação de `webhook_token` e Proteção de Logout/CSRF:**
   - Exigir `POST` em `/logout` (evita logout forçado via tag `<img>`) e validar `Origin` / `Sec-Fetch-Site` em mutações `POST`/`DELETE`.
5. **Execução Não-Root no Container Web (`web/Dockerfile`):**
   - Criar usuário dedicado sem privilégios (`adduser -D -u 10001 nms`) no Alpine para rodar `./nms-web`.
