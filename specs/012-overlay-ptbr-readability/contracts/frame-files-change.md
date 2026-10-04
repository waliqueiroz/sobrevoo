# Mudança no Contrato dos Arquivos de Quadro (etapa 5)

**Feature**: `012-overlay-ptbr-readability` | **Data**: 2026-10-03

Esta etapa **não** acrescenta nem remove nenhum bloco do PNG que a etapa 5
define (`specs/005-frame-rendering/contracts/frame-files.md`) — a estrutura
do arquivo continua exatamente a mesma. O que muda é **o conteúdo de alguns
pixels** (as sobreposições de tela) e, por isso, **a versão do desenho**.
Na implementação, o contrato da etapa 5 recebe uma nota desta mudança, como
as etapas 6, 8, 9 e 11 já fizeram antes dela.

## O que muda

- **`RenderVersion` sobe de 3 para 4.** Os pixels mudam de propósito outra
  vez: não é um dado novo exibido, é o acabamento de um dado que já
  existia.
- A tabela "Cores e padrões" de `frame-files.md`, na parte herdada da
  etapa 11 (sobreposição de tela), ganha estas entradas alteradas:

  | Elemento | Antes (etapa 11) | Depois (esta etapa) |
  |---|---|---|
  | Idioma do texto | rótulos em inglês (`DIST`, `ELEV`, `GAIN`, `TIME`) | rótulos em português do Brasil (`DIST`, `ELEV`, `GANHO`, `TEMPO`); abreviações de unidade (`km`, `m`) inalteradas |
  | Peso da fonte | regular (`golang.org/x/image/font/gofont/goregular`) | forte (`golang.org/x/image/font/gofont/gobold`), mesma família |
  | Espessura do contorno do texto | fração da altura do quadro (`OverlayOutlineRatio × altura`) | fração do tamanho do glifo (`OverlayOutlineRatio × ppem`) |
  | Largura dos painéis numéricos | recalculada a cada quadro, a partir do texto daquele quadro | calculada uma vez por execução, a partir do texto mais largo de **todo** o plano — estável do primeiro ao último quadro |

- Nenhum valor exibido, nenhum bloco, nenhuma posição de bloco na tela,
  nenhuma cor, nenhuma margem de segurança e nenhum tamanho do marcador do
  perfil mudam (FR-009 do `spec.md`).

## Versão e compatibilidade

- Como a identificação do conjunto (`frame-set=`) já inclui `RenderVersion`
  (desde a etapa 5), quadros de uma versão anterior a esta tornam-se,
  automaticamente, de **outro conjunto**: sem `--overwrite`, `render all`
  os recusa (`ErrFrameSetConflict`, código `37`, sem mudança de código);
  com `--overwrite`, redesenha. `fly --keep` redesenha os quadros e o
  vídeo ao notar a mudança (plano e recorte continuam reaproveitados).
- Não há migração: um quadro se desenha de novo com um comando.
- A igualdade byte a byte do desenho continua (SC-006): o mesmo plano, o
  mesmo recorte, a mesma resolução, a mesma aparência e a mesma
  configuração de sobreposição produzem sempre a mesma imagem, nesta
  versão, em qualquer máquina ou arquitetura — e, agora, também em
  qualquer ordem de desenho dos quadros (a largura do painel não depende
  de qual quadro é pedido primeiro).
- Um leitor que só conhece os blocos `tEXt` de hoje continua funcionando
  sem nenhuma mudança — nenhum bloco novo foi acrescentado.

## O que muda no código das etapas 5/9/11

- `internal/domain/vector_font.go`: `newVectorFace` passa a usar
  `golang.org/x/image/font/gofont/gobold` no lugar de `.../goregular`.
- `internal/domain/frame_screen_overlay.go`: rótulos em português;
  `distanceBlockText`/`elevationBlockText`/`timeBlockText` (novas,
  compartilhadas entre o desenho e o cálculo da largura estável);
  `overlayPpem` (nova, extraída de `draw`); raio de dilatação do contorno
  em `drawText` baseado em `ppem`; `screenOverlay` ganha o campo
  `panelWidth`; `numericPanelWidth` (a antiga, por quadro) é substituída
  por `stablePanelWidth` (todo o plano).
- `internal/domain/frame_scene.go`: `Scene` ganha o cache da largura
  estável (`panelWidth`/`panelWidthHeight`/`panelWidthSet`) e o método
  `numericPanelWidth(plan, height, ppem)`; `Render` o chama antes de
  montar `screenOverlay`.
- `internal/domain/render_tuning.go`: `RenderVersion` 3 → 4;
  `OverlayOutlineRatio` redefinida como fração de `ppem` (valor ajustado
  de `0.0025` para `0.035`); `OverlayOutlineMinWidth` inalterada.
- `specs/005-frame-rendering/contracts/frame-files.md` recebe a nota desta
  mudança (tabela de cores/padrões e a nota de versão), do mesmo jeito que
  já recebeu as notas das etapas 6 e 11.
