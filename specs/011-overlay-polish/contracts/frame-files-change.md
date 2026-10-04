# Mudança no Contrato dos Arquivos de Quadro (etapa 5)

**Feature**: `011-overlay-polish` | **Data**: 2026-10-02

Esta etapa **não** acrescenta nem remove nenhum bloco do PNG que a etapa 5
define (`specs/005-frame-rendering/contracts/frame-files.md`) — a estrutura
do arquivo (`IHDR`, os dois blocos `tEXt`, `IDAT…`, `IEND`) continua exatamente
a mesma. O que muda é **o conteúdo de alguns pixels** (as sobreposições de
tela) e, por isso, **a versão do desenho**. Na implementação, o contrato da
etapa 5 recebe uma nota desta mudança, como as etapas 6, 8 e 9 já fizeram
antes dela.

## O que muda

- **`RenderVersion` sobe de 2 para 3.** Diferente da mudança da etapa 6
  (versão subiu, pixels não mudaram), **os pixels mudam de propósito**: é
  a primeira vez, desde a introdução das sobreposições de tela (etapa 9),
  que o próprio desenho delas é corrigido, não um dado novo exibido.
- A tabela "Cores e padrões" de `frame-files.md` ganha, para a parte que
  descreve a sobreposição de tela (herdada da etapa 9, nunca antes
  detalhada nesse contrato em si — ver `specs/009-frame-overlays/
  data-model.md`), estas entradas novas ou alteradas:

  | Elemento | Antes (etapa 9) | Depois (esta etapa) |
  |---|---|---|
  | Fonte do texto | bitmap `inconsolata.Bold8x16`, ampliada por fator inteiro (replicação de pixel) | fonte vetorial embutida (`golang.org/x/image/font/gofont/goregular`), rasterizada com suavização por um rasterizador próprio (nunca `golang.org/x/image/vector`) |
  | Contorno do texto | nenhum (só a placa escurecida por baixo) | contorno escuro fixo (`OverlayTextOutlineColor`) ao redor de cada glifo |
  | Largura dos painéis numéricos | cada painel do tamanho do próprio texto | os três painéis numéricos (distância; elevação+ganho; tempo decorrido) compartilham a largura do mais largo presente no quadro |
  | Raio do marcador do perfil | ligado à escala do texto (`max(2, scale)`) | fração fixa da altura do quadro, com piso em pixels (`ProfileMarkerRadiusRatio`/`ProfileMarkerMinRadius`) |
  | Margem de segurança | uma só razão, igual nas quatro bordas (`OverlayMarginRatio`, fração do lado menor) | três razões — topo e laterais (fração da altura/largura, respectivamente) e base, maior que as outras duas (`OverlayTopMarginRatio`/`OverlaySideMarginRatio`/`OverlayBottomMarginRatio`) |

- Nenhum valor exibido, nenhum bloco, nenhuma posição de bloco na tela e
  nenhuma cor de texto/painel/perfil mudam — só a forma como são
  desenhados (FR-010/FR-011 do `spec.md`).

## Versão e compatibilidade

- Como a identificação do conjunto (`frame-set=`) já inclui `RenderVersion`
  (desde a etapa 5), quadros de uma versão anterior a esta tornam-se,
  automaticamente, de **outro conjunto**: sem `--overwrite`, `render all`
  os recusa (`ErrFrameSetConflict`, código `37`, sem mudança de código);
  com `--overwrite`, redesenha. `fly --keep` redesenha os quadros e o vídeo
  ao notar a mudança (plano e recorte continuam reaproveitados, por não
  dependerem do desenho).
- Não há migração: um quadro se desenha de novo com um comando.
- A igualdade byte a byte do desenho continua (SC-007): o mesmo plano, o
  mesmo recorte, a mesma resolução, a mesma aparência e a mesma
  configuração de sobreposição produzem sempre a mesma imagem, nesta
  versão, em qualquer máquina ou arquitetura.
- Um leitor que só conhece os dois blocos `tEXt` de hoje continua
  funcionando sem nenhuma mudança — nenhum bloco novo foi acrescentado.

## O que muda no código da etapa 5/9

- `internal/domain/vector_font.go` (novo): parsing dos contornos
  (`golang.org/x/image/font/sfnt` + `font/gofont/goregular`) e o
  rasterizador próprio, determinístico (`research.md` itens 1, 3, 4).
- `internal/domain/frame_screen_overlay.go`: fonte vetorial no lugar do
  bitmap; largura compartilhada dos três painéis numéricos; contorno do
  texto; raio do marcador do perfil pelas novas constantes; margem por
  borda.
- `internal/domain/frame_scene.go`: `Scene` ganha o campo `face
  *vectorFace`, construído uma vez em `NewScene`.
- `internal/domain/render_tuning.go`: `RenderVersion` 2 → 3; `
  OverlayMarginRatio` removida, substituída por `OverlayTopMarginRatio`/
  `OverlaySideMarginRatio`/`OverlayBottomMarginRatio`; `
  OverlayTextOutlineColor`, `OverlayOutlineRatio`, `OverlayOutlineMinWidth`,
  `ProfileMarkerRadiusRatio`, `ProfileMarkerMinRadius` novas.
- `specs/005-frame-rendering/contracts/frame-files.md` recebe a nota desta
  mudança (tabela de cores/padrões e a nota de versão), do mesmo jeito que
  já recebeu a nota da etapa 6.
- O hash de referência de `internal/domain/frame_scene_test.go` é
  recalculado (os pixels da sobreposição mudam de propósito).
