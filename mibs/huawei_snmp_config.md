# Configuração SNMP - Huawei NetEngine 8000 M4

Este documento lista os comandos necessários para visualizar as configurações atuais e configurar uma nova community SNMPv2c no seu roteador Huawei NetEngine 8000 M4.

## 1. Visualizar a configuração atual

Para checar como o SNMP está configurado atualmente, acesse o terminal do roteador e utilize os seguintes comandos na view normal ou de sistema:

```bash
# Visualizar o status e versão do SNMP rodando no equipamento
display snmp-agent sys-info

# Visualizar as communities SNMP configuradas (caso não estejam ocultadas por cipher)
display snmp-agent community

# Visualizar toda a configuração atual filtrando apenas por snmp
display current-configuration | include snmp
```

---

## 2. Criar uma nova Community SNMPv2c

Para o nosso sistema `snmpEndLog` conseguir ler os dados (como tráfego, PPPoE online, CPU, Memória, Voltagem), precisamos de uma community com permissão de leitura (`read-only`). Siga os passos abaixo:

```bash
# 1. Entre no modo de configuração do sistema
system-view

# 2. Habilite a versão v2c (caso o roteador esteja restrito apenas a v3)
snmp-agent sys-info version v2c

# 3. Crie a nova community de leitura (substitua SUA_COMMUNITY_AQUI pelo nome desejado)
# O parâmetro 'cipher' criptografa a senha no arquivo de configuração (recomendado).
snmp-agent community read cipher SUA_COMMUNITY_AQUI

# 4. (Opcional, mas Altamente Recomendado) Restringir o acesso via ACL
# Isso garante que apenas o IP do seu servidor snmpEndLog possa fazer as consultas
acl number 2000
 rule 5 permit source IP_DO_SEU_SERVIDOR 0
 quit
snmp-agent community read cipher SUA_COMMUNITY_AQUI acl 2000

# 5. Salve as configurações
return
save
```

## 3. Configurar no snmpEndLog
Após criar a community no roteador, basta ir na interface Web do **snmpEndLog**, clicar em **Equipamentos > Cadastrar (ou Configurações do Huawei)** e inserir:

- **Versão SNMP:** `v2c`
- **Community:** O nome que você definiu no passo 3.
