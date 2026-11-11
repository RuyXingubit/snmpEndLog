# Auditoria de Segurança — v2: Diagnóstico Real de Portas da VM e Regras de Firewall (UFW)

**Data:** 11/11/2026  
**Referência:** Continuação de `v1_diagnostico_completo_e_fail2ban.md` após coleta de `ufw status verbose` e `ss -tulnp` na VM `143.208.136.39`.

---

## 1. Retrato Real das Portas Abertas na VM (`ss -tulnp`)

| Protocolo | Endereço : Porta | Processo | Diagnóstico de Exposição |
| :--- | :--- | :--- | :--- |
| `TCP` | `127.0.0.1:5432` | `docker-proxy` (`nms-db`) | **Seguro** — escutando exclusivamente em loopback local |
| `TCP` | `0.0.0.0:80` / `[::]:80` | `docker-proxy` (`nms-caddy`) | **Esperado** — redirecionamento HTTP → HTTPS e validação ACME |
| `TCP` | `0.0.0.0:443` / `[::]:443` | `docker-proxy` (`nms-caddy`) | **Esperado** — Dashboard HTTPS e endpoint `/api/webhooks/link-quality` |
| `TCP` | `0.0.0.0:22` / `[::]:22` | `sshd` | **Exposto** — UFW inativo; alvo ativo de brute-force externo |
| `UDP` | `0.0.0.0:5514` | `python` (`nms-collector`) | **Crítico / Exposto** — Syslog UDP aberto para toda a internet |
| `TCP` | `0.0.0.0:5514` | `python` (`nms-collector`) | **Crítico / Exposto** — Syslog TCP aberto para toda a internet |
| `UDP` | `0.0.0.0:54360` | `python` (`nms-collector`) | Socket cliente efêmero do `pysnmp` para consultas de saída |

- **Status atual do UFW:** `Status: inactive`
- **Chave SSH administrativa (`id_ed25529.pub`):** Cadastrada em `~/.ssh/authorized_keys` do usuário `ruy`.

---

## 2. Estratégia de Blindagem de Rede (UFW)

1. **Política padrão:** `deny incoming` / `allow outgoing` (mantém todo o polling SNMP `UDP/161` e ICMP `ping` funcionando via estados `ESTABLISHED/RELATED` do `conntrack`).
2. **Web (`80/tcp` e `443/tcp`):** Liberado para acesso HTTPS, Let's Encrypt e webhooks dos roteadores.
3. **Syslog (`5514/udp` e `5514/tcp`):** Restrito exclusivamente aos blocos da rede Proserv e sub-redes internas de gerência (cobrindo todos os equipamentos ativos observados no coletor: `143.208.136.0/22`, `177.53.136.0/22`, `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, `100.64.0.0/10` e `2804:192c::/32`).
4. **SSH (`22/tcp`):** Protegido com `ufw limit 22/tcp` + `fail2ban` (`backend = systemd`, `mode = aggressive`), aguardando definição do usuário sobre desativar senha (`PasswordAuthentication no`) e/ou restringir origem no UFW.

---

## 3. Decisões de Regra de Permissão Pendentes para o Hardening do Código

1. Perfil `viewer` em **Alarmes**: apenas leitura vs permissão de resolver alarmes.
2. Perfil `viewer` em **Análise IA (`/ai`)**: restrito a `admin` vs compartilhado/isolado.
3. Política final de **SSH (`22/tcp`)** na VM.
