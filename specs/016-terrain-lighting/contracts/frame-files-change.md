# Mudança no Contrato dos Arquivos de Quadro (etapa 5)

**Feature**: `016-terrain-lighting` | **Data**: 2026-10-05

Esta etapa **não** acrescenta nem remove nenhum bloco do PNG que a etapa 5
define (`specs/005-frame-rendering/contracts/frame-files.md`) — a estrutura
do arquivo (`IHDR`, os dois blocos `tEXt`, `IDAT…`, `IEND`) continua
exatamente a mesma. O que muda é **a cor de alguns pixels do terreno** e,
por isso, **a versão do desenho**, como já aconteceu nas etapas 6, 8, 9, 11,
12 e 15.

## O que muda

- **`RenderVersion` sobe de 5 para 6.**
- A tabela "Cores e padrões" de `frame-files.md` ganha, para a parte do
  terreno vestido com o mapa base, esta entrada nova:

  | Elemento | Antes (etapa 5) | Depois (esta etapa) |
  |---|---|---|
  | Cor do terreno (pixel com imagem de mapa) | a cor da peça do mapa base, sem nenhuma modulação | a mesma cor, multiplicada por um fator de brilho fixo entre `TerrainLightMinFactor` (0,75) e `TerrainLightMaxFactor` (1,15), conforme a inclinação da superfície naquele ponto em relação a uma luz direcional fixa (azimute 315°, altura 45°) |

- Nenhuma outra cor ou padrão muda: o hachurado de "sem mapa"
  (`NoMapColors`), o xadrez de "sem elevação" (`NoElevationColors`), a
  casca e o núcleo do traçado, o anel e o disco do marcador, e as
  sobreposições de tela (texto, painéis, perfil de elevação) continuam
  exatamente como a etapa 5/9/11/12/15 já definem — a luz atinge só o
  terreno vestido com imagem de mapa (FR-006 do `spec.md`).

## Versão e compatibilidade

- Como a identificação do conjunto (`frame-set=`) já inclui `RenderVersion`
  (desde a etapa 5), quadros de uma versão anterior a esta tornam-se,
  automaticamente, de **outro conjunto**: sem `--overwrite`, `render all`
  os recusa (`ErrFrameSetConflict`, sem mudança de código); com
  `--overwrite`, redesenha. `fly --keep` redesenha os quadros e o vídeo ao
  notar a mudança (plano e recorte continuam reaproveitados, por não
  dependerem do desenho do terreno).
- Não há migração: um quadro se desenha de novo com um comando.
- A igualdade byte a byte do desenho continua (SC-004 do `spec.md`): o
  mesmo plano, o mesmo recorte, a mesma resolução, a mesma aparência e a
  mesma configuração de sobreposição produzem sempre a mesma imagem, nesta
  versão, em qualquer máquina, processo ou arquitetura.
- Um leitor que só conhece os dois blocos `tEXt` de hoje continua
  funcionando sem nenhuma mudança — nenhum bloco novo foi acrescentado.

## O que muda no código das etapas 5/11

- `internal/domain/frame_terrain_light.go` (novo): a direção fixa da luz,
  a faixa fixa de brilho e `terrainLightFactor` (research.md itens 1, 2, 8).
- `internal/domain/frame_surface.go`: `surface` ganha a pirâmide de
  gradiente (`gradients`), construída em `newSurface`; `placedSurface`
  ganha `normalAt` (research.md itens 4, 5, 6, 7).
- `internal/domain/frame_scene.go`: `drawPixel` multiplica a cor do mapa
  pelo fator de brilho no ramo `stateImage` (research.md item 3).
- `internal/domain/render_tuning.go`: `RenderVersion` 5 → 6;
  `TerrainLightAzimuthDegrees`, `TerrainLightAltitudeDegrees`,
  `TerrainLightMinFactor`, `TerrainLightMaxFactor` novas.
- `specs/005-frame-rendering/contracts/frame-files.md` recebe a nota desta
  mudança (tabela de cores/padrões e a nota de versão), do mesmo jeito que
  já recebeu a nota das etapas 6, 8, 9, 11, 12 e 15.
- O hash de referência de `internal/domain/frame_scene_test.go` é
  recalculado (os pixels do terreno mudam de propósito).
