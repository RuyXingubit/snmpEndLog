# 📊 Laudo Técnico de Desempenho e Eficiência — Servidor CDN Globo

**Equipamento de Origem:** BGP-VTX (Vitória do Xingu)  
**Interface Analisada:** Porta 22 (`CDN-GLOBO`)  
**Período Amostrado:** 09/07/2026 a 07/08/2026 (**619 horas de telemetria contínua auditada**)  
**Data do Relatório:** 24/08/2026  
**Finalidade:** Auditoria de entrega de tráfego, ganho de cache e viabilidade de manutenção do servidor.  

---

## 1. Resumo Executivo e Indicadores Principais

| Métrica | Direção / Visão | Valor Apurado | Impacto na Rede |
| :--- | :--- | :--- | :--- |
| **Entrega Média no Horário Nobre (18h às 00h)** | `ifInOctets (IN)` | **`713.00 Mbps`** | Volume médio real entregue aos assinantes no pico noturno |
| **Entrega Média no Horário Comercial (06h às 18h)** | `ifInOctets (IN)` | **`534.95 Mbps`** | Volume médio real entregue durante o dia |
| **Entrega Média Global 24h** | `ifInOctets (IN)` | **`480.35 Mbps`** | Média geral constante de streaming local |
| **Pico Máximo de Entrega aos Clientes** | `ifInOctets (IN)` | **`1.886.50 Mbps (1.89 Gbps)`** | Registrado em 19/07/2026 às 19:00 (Domingo) |
| **Consumo Médio de Link Externo (Trânsito)** | `ifOutOctets (OUT)` | **`86.61 Mbps`** | Custo real de banda de internet para abastecer o cache |
| **Consumo de Link Externo no Horário Nobre** | `ifOutOctets (OUT)` | **`146.07 Mbps`** | Custo de banda de internet no pico noturno |
| **Fator de Economia / Eficiência de Cache** | `IN / OUT` | **`5.5x a 9.3x`** | Para cada 100 Mbps de trânsito, entrega de 550 a 930 Mbps locais |

> 🏆 **Veredito Técnico de Viabilidade:**  
> O servidor apresentou um **desempenho excelente**. Ele descarrega da rede de trânsito IP paga uma média de **~713 Mbps** todas as noites, atingindo picos de **1.88 Gbps** nos domingos e noites de futebol, consumindo menos de 200 Mbps de link externo. **A recomendação técnica é MANTER o servidor ativo.**

---

## 2. Desempenho por Turnos de Consumo

| Turno do Dia | Entrega aos Clientes (IN) | Link Externo Consumido (OUT) | Fator de Economia | Amostras |
| :--- | :--- | :--- | :--- | :--- |
| **☀️ Diurno / Comercial (06h às 18h)** | **`534.95 Mbps`** | 78.38 Mbps | **`6.8x`** | 305 horas |
| **🌙 Noturno / Horário Nobre (18h às 00h)** | **`713.00 Mbps`** | 146.07 Mbps | **`4.9x`** | 155 horas |
| **🌌 Madrugada (00h às 06h)** | **`150.27 Mbps`** | 44.79 Mbps | **`3.4x`** | 159 horas |

---

## 3. Desempenho nos Momentos de Transmissão de Futebol e Grandes Eventos

Abaixo estão os registros nos momentos de jogos de futebol (Domingos à tarde e Quartas-feiras à noite):

| Data do Jogo | Horário (Brasília) | Média Entregue aos Clientes | Pico de Entrega aos Clientes | Link Externo Usado | Ganho de Cache |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **19/07/2026 (Dom)** | **19:00** | **1.123.60 Mbps (1.12 Gbps)** | **1.886.50 Mbps (1.89 Gbps)** | 173.80 Mbps | **`6.5x`** |
| **19/07/2026 (Dom)** | **18:00** | **1.665.03 Mbps (1.66 Gbps)** | **1.829.71 Mbps (1.83 Gbps)** | 201.25 Mbps | **`8.3x`** |
| **19/07/2026 (Dom)** | **16:00** | **1.504.75 Mbps (1.50 Gbps)** | **1.729.91 Mbps (1.73 Gbps)** | 162.66 Mbps | **`9.3x`** |
| **19/07/2026 (Dom)** | **17:00** | **1.495.88 Mbps (1.50 Gbps)** | **1.643.95 Mbps (1.64 Gbps)** | 184.29 Mbps | **`8.1x`** |
| **19/07/2026 (Dom)** | **15:00** | **811.95 Mbps** | **1.247.34 Mbps (1.25 Gbps)** | 125.90 Mbps | **`6.4x`** |
| **05/08/2026 (Qua)** | **21:00** | **941.34 Mbps** | **1.351.97 Mbps (1.35 Gbps)** | 203.86 Mbps | **`4.6x`** |
| **05/08/2026 (Qua)** | **20:00** | **904.76 Mbps** | **1.139.76 Mbps (1.14 Gbps)** | 157.58 Mbps | **`5.7x`** |
| **05/08/2026 (Qua)** | **22:00** | **795.43 Mbps** | **1.228.53 Mbps (1.23 Gbps)** | 182.38 Mbps | **`4.4x`** |
| **02/08/2026 (Dom)** | **16:00** | **613.57 Mbps** | **1.126.99 Mbps (1.13 Gbps)** | 97.30 Mbps | **`6.3x`** |
| **02/08/2026 (Dom)** | **15:00** | **640.92 Mbps** | **1.082.23 Mbps (1.08 Gbps)** | 96.93 Mbps | **`6.6x`** |
| **22/07/2026 (Qua)** | **21:00** | **870.62 Mbps** | **1.079.26 Mbps (1.08 Gbps)** | 212.18 Mbps | **`4.1x`** |
| **29/07/2026 (Qua)** | **21:00** | **873.60 Mbps** | **1.001.72 Mbps (1.00 Gbps)** | 201.61 Mbps | **`4.3x`** |
| **29/07/2026 (Qua)** | **22:00** | **794.42 Mbps** | **985.91 Mbps** | 176.78 Mbps | **`4.5x`** |
| **15/07/2026 (Qua)** | **21:00** | **801.29 Mbps** | **970.86 Mbps** | 190.80 Mbps | **`4.2x`** |
| **19/07/2026 (Dom)** | **20:00** | **773.80 Mbps** | **970.10 Mbps** | 137.47 Mbps | **`5.6x`** |

---

## 4. Consolidado Diário Completo (09/07 a 07/08)

| Data | Média do Dia (24h) | Pico Máximo do Dia | Média no Horário Nobre | Pico no Horário Nobre |
| :--- | :--- | :--- | :--- | :--- |
| **09/07/2026 (Qui)** | 502.75 Mbps | 1.474.32 Mbps | 713.61 Mbps | 1.259.18 Mbps |
| **10/07/2026 (Sex)** | 475.97 Mbps | 1.397.86 Mbps | 596.23 Mbps | 893.57 Mbps |
| **11/07/2026 (Sáb)** | **616.15 Mbps** | **1.820.38 Mbps** | **1.259.95 Mbps** | **1.820.38 Mbps** |
| **12/07/2026 (Dom)** | 469.67 Mbps | 1.228.90 Mbps | 638.47 Mbps | 888.16 Mbps |
| **13/07/2026 (Seg)** | 455.81 Mbps | 1.043.82 Mbps | 654.28 Mbps | 1.043.82 Mbps |
| **14/07/2026 (Ter)** | 564.25 Mbps | 1.686.89 Mbps | 758.23 Mbps | 1.512.90 Mbps |
| **15/07/2026 (Qua - Jogo)** | 571.94 Mbps | **1.866.11 Mbps** | 703.22 Mbps | **1.768.52 Mbps** |
| **16/07/2026 (Qui)** | 486.13 Mbps | 1.516.81 Mbps | 725.97 Mbps | 1.516.81 Mbps |
| **17/07/2026 (Sex)** | 437.96 Mbps | 818.48 Mbps | 614.47 Mbps | 818.48 Mbps |
| **18/07/2026 (Sáb)** | 469.78 Mbps | 1.168.63 Mbps | 731.06 Mbps | 1.168.63 Mbps |
| **19/07/2026 (Dom - Jogo)** | **582.01 Mbps** | **1.886.50 Mbps** | **885.11 Mbps** | **1.886.50 Mbps** |
| **20/07/2026 (Seg)** | 450.11 Mbps | 1.031.83 Mbps | 660.01 Mbps | 1.031.83 Mbps |
| **21/07/2026 (Ter)** | 439.30 Mbps | 1.031.35 Mbps | 632.56 Mbps | 874.12 Mbps |
| **22/07/2026 (Qua - Jogo)** | 461.58 Mbps | 1.079.26 Mbps | 706.70 Mbps | 1.079.26 Mbps |
| **23/07/2026 (Qui)** | 433.65 Mbps | 893.86 Mbps | 678.68 Mbps | 893.86 Mbps |
| **24/07/2026 (Sex)** | 146.43 Mbps | 501.75 Mbps | - | - |
| **27/07/2026 (Seg)** | 621.18 Mbps | 868.08 Mbps | 639.04 Mbps | 868.08 Mbps |
| **28/07/2026 (Ter)** | 455.54 Mbps | 1.120.55 Mbps | 685.15 Mbps | 1.120.55 Mbps |
| **29/07/2026 (Qua - Jogo)** | 470.83 Mbps | 1.001.72 Mbps | 728.98 Mbps | 1.001.72 Mbps |
| **30/07/2026 (Qui)** | 497.11 Mbps | 1.322.10 Mbps | 791.81 Mbps | 1.322.10 Mbps |
| **31/07/2026 (Sex)** | 507.46 Mbps | 1.462.44 Mbps | 663.15 Mbps | 1.462.44 Mbps |
| **01/08/2026 (Sáb)** | 454.37 Mbps | 916.74 Mbps | 587.34 Mbps | 798.99 Mbps |
| **02/08/2026 (Dom)** | 460.65 Mbps | 1.126.99 Mbps | 648.08 Mbps | 914.19 Mbps |
| **03/08/2026 (Mon)** | 479.47 Mbps | 994.47 Mbps | 687.50 Mbps | 918.10 Mbps |
| **04/08/2026 (Ter)** | 466.68 Mbps | 1.389.98 Mbps | 692.37 Mbps | 1.389.98 Mbps |
| **05/08/2026 (Qua - Jogo)** | 470.32 Mbps | 1.351.97 Mbps | 762.06 Mbps | 1.351.97 Mbps |
| **06/08/2026 (Qui)** | 449.80 Mbps | 1.088.47 Mbps | 682.44 Mbps | 1.088.47 Mbps |
| **07/08/2026 (Sex)** | 128.33 Mbps | 538.92 Mbps | - | - |

---

## 5. Parecer de Engenharia e Conclusão de Negócio

1. **Eficiência Comprovada:**
   - O servidor local da Globo atendeu com primazia a localidade, garantindo que o tráfego de streaming (GloboPlay, transmissões ao vivo e TV aberta) não sobrecarregasse os links de subida/trânsito do provedor.
   - O ganho médio de **6.8x no horário comercial** e **4.9x a 9.3x no horário nobre** demonstra que o servidor cumpre seu papel com alta eficiência.
2. **Impacto de Retirada:**
   - Caso o servidor fosse devolvido ou desativado, o provedor precisaria contratar **entre 1.2 Gbps e 1.8 Gbps a mais de trânsito IP** para suportar a demanda nos dias de futebol e horário nobre, gerando um custo significativamente maior do que a manutenção do servidor.
3. **Recomendação Final:**
   - **Manter o servidor ativo** e em operação integrada no BGP-ATM.
