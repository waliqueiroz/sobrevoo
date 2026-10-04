# Mudança no Contrato dos Arquivos de Quadro (etapa 5)

**Feature**: `015-overlay-redesign` | **Data**: 2026-10-04

Esta etapa **não** acrescenta nem remove nenhum bloco do PNG que a etapa 5
define (`specs/005-frame-rendering/contracts/frame-files.md`) — a estrutura
do arquivo (`IHDR`, os dois blocos `tEXt`, `IDAT…`, `IEND`) continua
exatamente a mesma. O que muda é **o conteúdo de alguns pixels** (as
sobreposições de tela, quando ligadas) e, por isso, **a versão do
desenho** — a mesma categoria de mudança que a etapa 11 já fez.

## O que muda

- **`RenderVersion` sobe de 4 para 5.**
- A tabela "Cores e padrões" de `frame-files.md`, na parte herdada das
  etapas 9/11/12 que descreve a sobreposição de tela, ganha estas
  entradas novas ou alteradas:

  | Elemento | Antes (até a etapa 12) | Depois (esta etapa) |
  |---|---|---|
  | Fundo de cada bloco numérico | faixa/painel semitransparente (`OverlayPanelColor`/`OverlayPanelOpacity`) atrás do texto | nenhum — o texto é desenhado direto sobre a imagem |
  | Arranjo dos blocos numéricos | empilhados verticalmente no alto do quadro | lado a lado, em colunas de mesma largura, numa faixa horizontal no alto do quadro |
  | Ordem dos blocos | a ordem em que `--overlay-blocks` foi informada (ou a ordem interna de verificação) | ordem fixa, sempre a mesma: velocidade, elevação, distância, ganho, tempo decorrido |
  | Conteúdo de um bloco numérico | uma linha: rótulo abreviado em caixa alta + valor (ex.: `ELEV 120 m   GANHO +45 m`) | até três linhas centralizadas: rótulo por extenso (ex.: `Elevação`), valor em corpo maior, unidade — o ganho sai do bloco de elevação e vira seu próprio bloco (`Ganho`) |
  | Fundo do gráfico de elevação (`profile`) | faixa/painel semitransparente atrás da linha e do marcador | nenhum — a linha e o marcador ganham um contorno escuro (a mesma técnica do texto) para se destacar do terreno |

- Nenhum valor exibido, nenhuma unidade, nenhuma margem de segurança,
  nenhuma fonte e nenhum rasterizador mudam — só a apresentação dos blocos
  numéricos e a moldura do gráfico de elevação (FR-013/FR-018 do
  `spec.md`).

## Versão e compatibilidade

- Como a identificação do conjunto (`frame-set=`) já inclui `RenderVersion`
  (desde a etapa 5) e `OverlayConfig.Fingerprint()` (desde a etapa 9, agora
  com um bit a mais para `gain`), quadros de antes desta etapa tornam-se,
  automaticamente, de **outro conjunto**: sem `--overwrite`, `render all`
  os recusa (`ErrFrameSetConflict`, código já existente); com
  `--overwrite`, redesenha. `fly --keep` redesenha os quadros e o vídeo ao
  notar a mudança (plano e recorte continuam reaproveitados, por não
  dependerem do desenho).
- Não há migração: um quadro se desenha de novo com um comando.
- A igualdade byte a byte do desenho continua (SC-007): o mesmo plano, o
  mesmo recorte, a mesma resolução, a mesma aparência e a mesma
  configuração de sobreposição produzem sempre a mesma imagem, nesta
  versão, em qualquer máquina ou arquitetura.
- Um leitor que só conhece os dois blocos `tEXt` de hoje continua
  funcionando sem nenhuma mudança — nenhum bloco novo foi acrescentado.

## O que muda no código

- `internal/domain/frame_overlay_config.go`: `OverlayBlockGain`;
  `OverlayConfig.Gain`; `overlayBlockOrder` (ordem fixa); `Fingerprint`
  com sétimo segmento.
- `internal/domain/frame_screen_overlay.go`: layout em colunas; `drawBlock`
  (três alturas); textos de bloco separando rótulo/valor/unidade;
  `elevationBlockText`/`gainBlockText` separados; `drawProfile` sem painel,
  com contorno; `stablePanelWidth`/`drawPanel` removidos.
- `internal/domain/frame_scene.go`: `Scene` perde `panelWidth`/
  `panelWidthHeight`/`panelWidthSet`/`numericPanelWidth`.
- `internal/domain/render_tuning.go`: `RenderVersion` 4 → 5;
  `OverlayPanelColor`/`OverlayPanelOpacity` removidas;
  `OverlayLabelHeightRatio`/`OverlayValueHeightRatio` novas.
- `internal/infra/outbound/config/config.go`: `RenderDefaults.OverlayBlocks`
  padrão passa a `distance,elevation,speed,profile`.
- `internal/infra/inbound/cli/overlay.go`: textos de ajuda e
  `overlayBlocksOf` citam `gain`.
- `specs/005-frame-rendering/contracts/frame-files.md` recebe a nota desta
  mudança (tabela de cores/padrões e a nota de versão), do mesmo jeito que
  já recebeu as notas das etapas 6, 8, 9 e 11.
- O hash de referência de `internal/domain/frame_scene_test.go` é
  recalculado (os pixels da sobreposição mudam de propósito).
