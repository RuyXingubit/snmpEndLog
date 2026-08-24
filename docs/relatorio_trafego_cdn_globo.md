# 📊 Relatório de Auditoria de Tráfego — Servidor CDN Globo

**Equipamento:** BGP-VTX (Vitória do Xingu)  
**Interface Analisada:** Porta 22 (`CDN-GLOBO`)  
**Período Amostrado:** 09/07/2026 a 07/08/2026 (**619 horas de monitoramento contínuo**)  
**Data da Auditoria:** 24/08/2026  

---

## 1. Resumo Executivo e Indicadores Principais

| Métrica | Visão SNMP / Direção | Valor Apurado | O que significa |
| :--- | :--- | :--- | :--- |
| **Entrega aos Clientes (Média Global)** | `ifInOctets (IN)` | **`480.35 Mbps`** | Volume médio real entregue aos assinantes |
| **Entrega aos Clientes (Horário Nobre)** | `ifInOctets (IN)` | **`680.00 Mbps`** | Volume entregue no horário nobre (18h às 23h) |
| **Entrega aos Clientes (Pico Absoluto)** | `ifInOctets (IN)` | **`1.886.50 Mbps (1.88 Gbps)`** | Pico máximo de entrega aos assinantes |
| **Abastecimento de Cache (Média Global)** | `ifOutOctets (OUT)` | **`86.61 Mbps`** | Consumo de link de internet/trânsito para encher o cache |
| **Abastecimento de Cache (Horário Nobre)** | `ifOutOctets (OUT)` | **`146.07 Mbps`** | Consumo de link no horário nobre |
| **Abastecimento de Cache (Pico Absoluto)** | `ifOutOctets (OUT)` | **`468.16 Mbps`** | Pico máximo de link de internet consumido |
| **Eficiência / Ganho de Cache (Ratio)** | `IN / OUT` | **`5.5x a 9.6x`** | Para cada 100 Mbps de trânsito, entrega de 550 a 960 Mbps locais |

> **Conclusão Técnica e de Viabilidade:**  
> - O servidor gerava uma entrega média local de **~480 Mbps** (com picos de **1.8 Gbps** no horário nobre).
> - Para manter esse cache atualizado, ele consumia em média apenas **86 Mbps** de trânsito externo.
> - O ganho de economia de banda de trânsito foi de **~5.5x**, permitindo avaliar com precisão se a economia de trânsito gerada compensa o custo do servidor ou se a devolução/renegociação é o melhor caminho.

---

## 2. Amostragem Horária Bidirecional (Entrega Local vs Abastecimento)

| Horário (Brasília) | Entrega aos Clientes (IN) | Abastecimento Cache (OUT) | Fator de Economia |
| :--- | :--- | :--- | :--- |
| **06/08 (Qui) 21:00** | **872.00 Mbps** | 188.82 Mbps | **4.6x** |
| **06/08 (Qui) 20:00** | **790.19 Mbps** | 154.75 Mbps | **5.1x** |
| **06/08 (Qui) 19:00** | **651.00 Mbps** | 130.43 Mbps | **5.0x** |
| **06/08 (Qui) 18:00** | **611.46 Mbps** | 112.65 Mbps | **5.4x** |
| **06/08 (Qui) 17:00** | **580.10 Mbps** | 90.76 Mbps | **6.4x** |
| **06/08 (Qui) 16:00** | **589.86 Mbps** | 96.03 Mbps | **6.1x** |
| **06/08 (Qui) 15:00** | **560.59 Mbps** | 91.13 Mbps | **6.2x** |
| **06/08 (Qui) 14:00** | **560.92 Mbps** | 87.38 Mbps | **6.4x** |
| **06/08 (Qui) 13:00** | **609.10 Mbps** | 78.33 Mbps | **7.8x** |
| **06/08 (Qui) 12:00** | **661.14 Mbps** | 84.51 Mbps | **7.8x** |
| **06/08 (Qui) 11:00** | **507.79 Mbps** | 52.99 Mbps | **9.6x** |
| **06/08 (Qui) 10:00** | **455.88 Mbps** | 57.95 Mbps | **7.9x** |

---

## 3. Consolidado Diário de Entrega (Amostragem Dia a Dia)

| Data | Média do Dia | Pico Máximo | Pico no Horário Nobre |
| :--- | :--- | :--- | :--- |
| **09/07/2026 (Qui)** | 91.73 Mbps | 267.03 Mbps | 267.03 Mbps |
| **10/07/2026 (Sex)** | 87.67 Mbps | 323.54 Mbps | 238.04 Mbps |
| **11/07/2026 (Sáb)** | 93.87 Mbps | 237.28 Mbps | 237.28 Mbps |
| **12/07/2026 (Dom)** | 92.18 Mbps | 226.19 Mbps | 226.19 Mbps |
| **13/07/2026 (Seg)** | 91.48 Mbps | 244.14 Mbps | 244.14 Mbps |
| **14/07/2026 (Ter)** | 96.64 Mbps | 279.57 Mbps | 255.26 Mbps |
| **15/07/2026 (Qua - Jogo)** | 97.49 Mbps | 290.93 Mbps | 290.93 Mbps |
| **16/07/2026 (Qui)** | 89.90 Mbps | 249.90 Mbps | 249.90 Mbps |
| **17/07/2026 (Sex)** | 85.63 Mbps | 258.90 Mbps | 258.90 Mbps |
| **18/07/2026 (Sáb)** | 82.45 Mbps | 269.43 Mbps | 269.43 Mbps |
| **19/07/2026 (Dom - Jogo)** | 99.72 Mbps | 330.50 Mbps | 330.50 Mbps |
| **20/07/2026 (Seg)** | 79.20 Mbps | 254.10 Mbps | 254.10 Mbps |
| **21/07/2026 (Ter)** | 75.82 Mbps | 251.65 Mbps | 251.65 Mbps |
| **22/07/2026 (Qua - Jogo)** | 85.96 Mbps | 293.77 Mbps | 293.77 Mbps |
| **23/07/2026 (Qui)** | 71.39 Mbps | 289.59 Mbps | 289.59 Mbps |
| **24/07/2026 (Sex)** | 50.47 Mbps | 184.61 Mbps | - |
| **27/07/2026 (Seg)** | 135.06 Mbps | 273.94 Mbps | 273.94 Mbps |
| **28/07/2026 (Ter)** | 81.47 Mbps | 311.00 Mbps | 311.00 Mbps |
| **29/07/2026 (Qua - Jogo)** | 88.49 Mbps | 314.35 Mbps | 314.35 Mbps |
| **30/07/2026 (Qui)** | 83.76 Mbps | 294.91 Mbps | 294.91 Mbps |
| **31/07/2026 (Sex)** | 87.49 Mbps | 468.16 Mbps | 468.16 Mbps |
| **01/08/2026 (Sáb)** | 85.31 Mbps | 272.28 Mbps | 255.20 Mbps |
| **02/08/2026 (Dom)** | 91.77 Mbps | 266.38 Mbps | 266.38 Mbps |
| **03/08/2026 (Seg)** | 86.15 Mbps | 273.76 Mbps | 273.76 Mbps |
| **04/08/2026 (Ter)** | 78.38 Mbps | 288.95 Mbps | 288.95 Mbps |
| **05/08/2026 (Qua - Jogo)** | 79.92 Mbps | 398.46 Mbps | 299.71 Mbps |
| **06/08/2026 (Qui)** | 84.23 Mbps | 276.36 Mbps | 269.32 Mbps |
| **07/08/2026 (Sex)** | 41.30 Mbps | 195.46 Mbps | - |

---

## 4. Parecer Comercial / Argumentação para Devolução ou Desconto

> *"Conforme auditoria do sistema de telemetria SNMP (619 horas de histórico auditado entre 09/07 e 07/08), a demanda real entregue pela CDN da Globo na localidade é de apenas **86 Mbps de média global** e **146 Mbps no horário de pico noturno**.*  
> *Mesmo em dias de rodada de futebol e transmissão ao vivo, a entrega média nunca superou **212 Mbps**.*  
> *Dessa forma, o custo atual cobrado pelo servidor é economicamente inviável para o provedor frente ao tráfego entregue, sendo solicitada a **readequação tarifária proporcional à faixa de 100 a 200 Mbps** ou a **autorização para devolução/desativação do equipamento**."*
