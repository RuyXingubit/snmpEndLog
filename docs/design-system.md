# NMS — Technical Design System

> Guia de referência para a criação de telas e componentes do frontend.
> **Filosofia:** Utilitário, focado em dados, alta densidade e estética estritamente técnica. 

Este projeto *não deve* se parecer com dashboards gerados por IA ou templates comerciais (evite Tailwind padrão, glassmorphism, sombras difusas e bordas muito arredondadas). A inspiração é Grafana, Datadog ou interfaces de equipamentos de rede corporativos.

---

## 1. Princípios Visuais

- **Foco nos Dados:** O conteúdo numérico e os gráficos são os protagonistas.
- **Formas Rígidas:** Zero ou mínimo border-radius (`2px`). Sem cantos "fofos".
- **Sem Sombras Difusas:** O contraste é feito por bordas sólidas de `1px` e diferença de tons de fundo, não por `box-shadow` esfumaçado.
- **Densidade:** Espaçamentos contidos para permitir a exibição de grandes tabelas e muitos logs sem a necessidade de rolagem excessiva.
- **Tipografia:** Uso agressivo de fontes *Monospace* para qualquer dado tabular, métrica, IP ou log.

---

## 2. Paleta de Cores (Dark Mode Nativo)

A paleta evita "tons pastéis" ou neons brilhantes gerados por IA. Usa fundos neutros puros e acentos focados em acessibilidade de leitura.

### Fundos e Bordas
| Uso                  | Cor Hex   | CSS Variable     |
|----------------------|-----------|------------------|
| Fundo Principal      | `#0F1115` | `--bg-primary`   |
| Fundo de Painéis/Cards| `#16191E` | `--bg-panel`     |
| Fundo Secundário (Títulos)| `#1D2127` | `--bg-surface`   |
| Bordas e Divisores   | `#323842` | `--border-subtle`|
| Bordas Ativas / Foco | `#4A5568` | `--border-strong`|

### Textos
| Uso                  | Cor Hex   | CSS Variable     |
|----------------------|-----------|------------------|
| Texto Principal      | `#E2E8F0` | `--text-primary` |
| Texto Secundário     | `#94A3B8` | `--text-secondary`|
| Texto Desativado     | `#475569` | `--text-muted`   |

### Cores Semânticas (Puristas)
As cores de status devem ser sóbrias e legíveis.
- **Up / Success:** `#10B981` (Verde)
- **Down / Critical:** `#EF4444` (Vermelho)
- **Warning / Degraded:** `#F59E0B` (Laranja)
- **Primary / Info:** `#3B82F6` (Azul Utilitário)

---

## 3. Tipografia e Formatação de Dados

- **UI Geral:** `Inter`, `Segoe UI`, `system-ui`.
- **Dados / Tabelas / Logs / Métricas:** `JetBrains Mono`, `Consolas`, monospace.
- Formatação de rede **OBRIGATÓRIA**: Usar `Mbps` e `Gbps` para tráfego (bits), com precisão de 2 casas decimais. Utilize sempre a função `formatBps()` do `app.js`.

---

## 4. Componentes

### Cards & Painéis (`.card`)
```css
.card {
    background-color: var(--bg-panel);
    border: 1px solid var(--border-subtle);
    border-radius: 2px;
}
```

### Tabelas (`.data-table`)
- Cabeçalhos com fundo `--bg-surface`, texto em caixa alta e fonte pequena.
- Células sem bordas verticais, apenas linha horizontal inferior discreta.
- Fontes de dados na tabela devem ser obrigatoriamente Monospace.

### Inputs e Botões
- **Inputs:** Fundo `--bg-primary`, borda sólida. Quando em foco, a borda muda para `--primary` sem "ring" esfumaçado.
- **Botões (`.btn`):** Retangulares. 
  - Primário: Fundo sólido azul utilitário. Hover escurece 10%, sem efeitos de glow.
  - Secundário: Fundo igual ao painel, borda proeminente. Hover muda fundo.

---

## 5. Exemplo de Estrutura

Nenhum elemento flutuante irreal. A sidebar tem cor idêntica ou ligeiramente mais escura que o fundo principal, dividida por uma borda rígida de `1px`.
O header (topbar) deve ser contínuo e sem sombras drop-shadow. O separador é sempre uma `border-bottom`.
