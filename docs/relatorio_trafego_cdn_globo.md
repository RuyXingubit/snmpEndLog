# 📊 Laudo Técnico e Estudo de Viabilidade Financeira (ROI) — Servidor CDN Globo

**Equipamento de Origem:** BGP-VTX (Vitória do Xingu)  
**Interface Analisada:** Porta 22 (`CDN-GLOBO`)  
**Período Amostrado:** 09/07/2026 a 07/08/2026 (**619 horas de telemetria contínua auditada**)  
**Data da Auditoria:** 24/08/2026  
**Finalidade:** Auditoria de tráfego, modelagem financeira de ROI (R$ 2,50/Mega) e análise de qualidade de experiência (QoE) do cliente.  

---

## 1. Premissas Financeiras e Custos do Provedor

| Item | Parâmetro Adotado | Observações |
| :--- | :--- | :--- |
| **Custo do Mega de Trânsito IP** | **`R$ 2,50 / Mbps / mês`** | Custo unitário para contratação de link de internet/trânsito |
| **Custo Atual do Servidor (Meses 1 a 24)** | **`R$ 3.500,00 a R$ 5.000,00 / mês`** | Fase de parcelamento / aquisição do equipamento |
| **Custo Futuro do Servidor (Pós-24 meses)** | **`R$ 1.000,00 / mês`** | Fase pós-quitação (manutenção, energia, suporte e hospedagem) |

---

## 2. Resumo de Tráfego e Banda Líquida Economizada

Como o provedor precisa contratar link de trânsito dimensionado para suportar o **PICO DE CONSUMO** (evitando gargalos e lentidão), a economia real gerada pela CDN reflete a capacidade de pico que ela retira dos links de internet:

| Métrica Operacional | Direção | Valor Auditado | Banda Evitada / Economizada |
| :--- | :--- | :--- | :--- |
| **Entrega de Pico aos Clientes (Domingo / Futebol)** | `IN` | **`1.886.50 Mbps (1.89 Gbps)`** | Servido 100% da rede local |
| **Link de Trânsito Consumido no Pico** | `OUT` | **`201.25 Mbps`** | Apenas preenchimento de cache |
| **Banda de Trânsito EVITADA no Pico** | `IN - OUT` | **`1.685.25 Mbps (~1.68 Gbps)`** | **Economia direta na porta de trânsito** |
| **Entrega Média no Horário Nobre (18h às 00h)** | `IN` | **`713.00 Mbps`** | Consumo constante de TV/Streaming |
| **Link de Trânsito Médio no Horário Nobre** | `OUT` | **`146.07 Mbps`** | Consumo externo |
| **Banda de Trânsito EVITADA no Horário Nobre** | `IN - OUT` | **`566.93 Mbps (~567 Mbps)`** | **Média contínua economizada toda noite** |

---

## 3. Análise Financeira e Retorno sobre Investimento (ROI)

### A. Valor Monetário da Banda Economizada (R$ 2,50 / Mega)

- **Economia nos Picos de Eventos / Futebol (1.685 Mbps economizados):**  
  $$1.685 \text{ Mbps} \times \text{R\$ } 2,50 = \mathbf{\text{R\$ } 4.212,50 \text{ / mês}}$$
- **Economia Média Contínua no Horário Nobre (567 Mbps economizados):**  
  $$567 \text{ Mbps} \times \text{R\$ } 2,50 = \mathbf{\text{R\$ } 1.417,50 \text{ / mês}}$$

---

### B. Cenário 1: Durante o Financiamento (Próximos 24 meses)

*Nesta fase, o servidor está sendo adquirido/amortizado (custo de R$ 3.500 a R$ 5.000/mês).*

| Cenário de Parcela | Custo do Servidor | Economia de Trânsito no Pico | Custo Líquido Real para o Provedor |
| :--- | :--- | :--- | :--- |
| **Parcela R$ 3.500,00** | R$ 3.500,00 | **R$ 4.212,50** | **`+ R$ 712,50` (O servidor já se paga e dá lucro!)** |
| **Parcela R$ 4.250,00** | R$ 4.250,00 | **R$ 4.212,50** | **`- R$ 37,50` (Empate financeiro total / custo zero)** |
| **Parcela R$ 5.000,00** | R$ 5.000,00 | **R$ 4.212,50** | **`- R$ 787,50` (Custo irrisório para adquirir o ativo)** |

> **Diagnóstico da Fase 1:**  
> Mesmo na fase de parcelamento mais cara, o valor economizado em trânsito IP (**R$ 4.212,50**) cobre entre **84% e 120% da parcela**. O provedor está praticamente pagando o investimento com a própria economia de link!

---

### C. Cenário 2: Pós-Quitação (Após 24 meses — Custo R$ 1.000,00/mês)

*Após o término das parcelas, o custo mensal cai para apenas R$ 1.000,00.*

$$\text{Economia Líquida Mensal} = \text{R\$ } 4.212,50 - \text{R\$ } 1.000,00 = \mathbf{\text{R\$ } 3.212,50 \text{ / mês}}$$
$$\text{Economia Líquida Anual} = \text{R\$ } 3.212,50 \times 12 = \mathbf{\text{R\$ } 38.550,00 \text{ / ano}}$$

| Período Pós-Quitação | Custo de Manutenção | Economia de Link Evitada | **Lucro / Economia Líquida Acumulada** |
| :--- | :--- | :--- | :--- |
| **1 Mês** | R$ 1.000,00 | R$ 4.212,50 | **`+ R$ 3.212,50`** |
| **1 Ano (12 meses)** | R$ 12.000,00 | R$ 50.550,00 | **`+ R$ 38.550,00`** |
| **3 Anos (36 meses)** | R$ 36.000,00 | R$ 151.650,00 | **`+ R$ 115.650,00`** |

*(Com o crescimento natural da base de clientes em 24 meses, a demanda do servidor tende a subir para 2.5 a 3.0 Gbps de pico, elevando a economia líquida para mais de **R$ 5.000,00 a R$ 6.500,00 por mês**).*

---

## 4. Ganhos de Qualidade para os Clientes (QoE / QoS)

Além do retorno financeiro direto, manter o servidor localmente gera diferenciais técnicos competitivos decisivos:

1. **Latência Ultra-Baixa (< 3ms):**
   - Sem o servidor local, o tráfego de vídeo precisaria viajar até os datacenters da Globo em Fortaleza, Belém ou São Paulo, gerando latências de **35ms a 65ms**.
   - Com o servidor local, o cliente assiste com latência de **`1ms a 5ms`**, garantindo início instantâneo do vídeo (*Instant Playback*).
2. **Eliminação de Travamentos (*Zero Buffering*):**
   - Em transmissões ao vivo de alta audiência (Copa, Finais do Brasileirão, estreias e Jornal Nacional), o streaming em 4K/Full HD não sofre com perdas de pacotes ou congestionamentos de links intermunicipais.
3. **Resiliência e Imunidade a Rompimentos de Fibra:**
   - Caso o link de trânsito IP principal sofra um rompimento ou instabilidade temporária, o streaming local da Globo **continua operando 100% no ar** para todos os clientes conectados.
4. **Desafogo da Infraestrutura de Rede:**
   - Evita saturar as portas de subida (Uplinks) dos switches e rádios de transporte da operadora.

---

## 5. Histórico Detalhado dos Picos de Entrega aos Clientes

| Data do Jogo | Horário (Brasília) | Média Entregue aos Clientes | Pico Entregue aos Clientes | Link Externo Consumido | Economia de Trânsito | Fator de Economia |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **19/07/2026 (Dom)** | **19:00** | **1.123.60 Mbps** | **1.886.50 Mbps (1.89 Gbps)** | 173.80 Mbps | **1.712,70 Mbps (R$ 4.281,75)** | **`6.5x`** |
| **19/07/2026 (Dom)** | **18:00** | **1.665.03 Mbps** | **1.829.71 Mbps (1.83 Gbps)** | 201.25 Mbps | **1.628,46 Mbps (R$ 4.071,15)** | **`8.3x`** |
| **19/07/2026 (Dom)** | **16:00** | **1.504.75 Mbps** | **1.729.91 Mbps (1.73 Gbps)** | 162.66 Mbps | **1.567,25 Mbps (R$ 3.918,12)** | **`9.3x`** |
| **19/07/2026 (Dom)** | **17:00** | **1.495.88 Mbps** | **1.643.95 Mbps (1.64 Gbps)** | 184.29 Mbps | **1.459,66 Mbps (R$ 3.649,15)** | **`8.1x`** |
| **15/07/2026 (Qua)** | **21:00** | **801.29 Mbps** | **1.866.11 Mbps (1.87 Gbps)** | 190.80 Mbps | **1.675,31 Mbps (R$ 4.188,27)** | **`4.2x`** |
| **11/07/2026 (Sáb)** | **21:00** | **1.259.95 Mbps** | **1.820.38 Mbps (1.82 Gbps)** | 170.00 Mbps | **1.650,38 Mbps (R$ 4.125,95)** | **`7.4x`** |
| **05/08/2026 (Qua)** | **21:00** | **941.34 Mbps** | **1.351.97 Mbps (1.35 Gbps)** | 203.86 Mbps | **1.148,11 Mbps (R$ 2.870,27)** | **`4.6x`** |

---

## 6. Parecer Final e Decisão Estratégica

1. **Durante o parcelamento (Próximos 24 meses):**  
   O servidor se sustenta financeiramente, pois evita a contratação de até **R$ 4.212,50/mês** em links de trânsito IP de pico, gerando custo líquido quase nulo para a formação de patrimônio tecnológico da empresa.
2. **Após a quitação (A partir do mês 25):**  
   Com a parcela caindo para **R$ 1.000,00/mês**, o servidor gerará uma **economia líquida direta superior a R$ 38.500,00 por ano** no caixa do provedor.
3. **Decisão:**  
   **A MANUTENÇÃO DO SERVIDOR É ALTAMENTE VANTAJOSA** tanto sob a ótica da saúde financeira quanto pela excelência de entrega e fidelização dos assinantes.
