# Modelo de Dados: Redesenho das Sobreposições de Tela em Colunas

Só o que muda ou se acrescenta em relação ao modelo já existente (`specs/
009-frame-overlays/data-model.md`, `specs/011-overlay-polish/data-model.md`,
`specs/014-speed-overlay-block/data-model.md`). Racional de cada decisão em
`research.md`. Nenhuma entidade persistida (plano, recorte) muda de forma —
só a configuração de sobreposição e o desenho, inteiramente dentro de
`internal/domain`.

## `domain.RenderVersion` (constante) — `render_tuning.go`

| Antes | Depois |
|---|---|
| `4` | `5` |

Automaticamente propagado para `FrameSetID` por `newFrameSetID`
(`frame_set.go`), sem nenhuma mudança de assinatura — nenhum conjunto de
quadros desta versão é tomado como o de uma versão anterior (FR-016).

## `domain.OverlayBlock` (enum estendido) — `frame_overlay_config.go`

| Constante | Valor | Novo? |
|---|---|---|
| `OverlayBlockDistance` | `"distance"` | não |
| `OverlayBlockElevation` | `"elevation"` | não (passa a mostrar só altitude) |
| `OverlayBlockGain` | `"gain"` | **sim** (015-overlay-redesign) |
| `OverlayBlockTime` | `"time"` | não |
| `OverlayBlockProfile` | `"profile"` | não |
| `OverlayBlockSpeed` | `"speed"` | não |

## `domain.OverlayConfig` (campo novo) — `frame_overlay_config.go`

| Campo novo | Tipo | Significado |
|---|---|---|
| `Gain` | `bool` | Liga o bloco de ganho acumulado, independente de `Elevation` (FR-008/FR-009). |

`NewOverlayConfig` ganha `case OverlayBlockGain: config.Gain = true`; a
mensagem de erro de nome inválido passa a citar os seis nomes aceitos
(FR-010). `Fingerprint()` ganha um sétimo segmento
(`flag(o.Gain)`), ao lado dos seis campos já existentes (`Enabled`,
`Distance`, `Elevation`, `Time`, `Profile`, `Speed`), nessa ordem — a
posição exata dentro da string não importa, só que exista e distinga
configurações que diferem apenas em `Gain` (FR-016/SC-008).

## `overlayBlockOrder` (novo, não exportado) — `frame_overlay_config.go`

```go
// overlayBlockOrder é a ordem fixa e documentada em que os blocos
// numéricos presentes são desenhados na faixa do alto — nunca a ordem em
// que o usuário os escreveu em --overlay-blocks (015-overlay-redesign
// Clarifications, Session 2026-10-04).
var overlayBlockOrder = []OverlayBlock{
    OverlayBlockSpeed,
    OverlayBlockElevation,
    OverlayBlockDistance,
    OverlayBlockGain,
    OverlayBlockTime,
}
```

`OverlayBlockProfile` fica fora dessa lista: continua tratado à parte, no
rodapé, sem fazer parte da faixa de colunas.

## `screenOverlay` (comportamento reescrito) — `frame_screen_overlay.go`

Sem campo novo na struct (`image`, `config`, `appearance`, `face` — o
campo `panelWidth` é **removido**, item abaixo). O que muda:

- **`draw`**: monta, a partir de `overlayBlockOrder`, a lista de blocos
  presentes (`config.<Bloco> && <disponibilidade>` — a mesma condição que
  cada bloco já tinha); divide a largura útil (`width − 2×marginSide`)
  pelo total presente em colunas de mesma largura; para cada bloco,
  chama o novo `drawBlock` no centro da própria coluna (FR-002/FR-003/
  FR-004/FR-014/FR-015).
- **`drawBlock`** (novo): desenha até três linhas centralizadas
  horizontalmente no centro da coluna — rótulo (`OverlayLabelHeightRatio`),
  valor (`OverlayValueHeightRatio`), unidade (`OverlayLabelHeightRatio`,
  omitida sem deixar vão quando o bloco não tem unidade) — sem nenhum
  painel atrás (FR-001/FR-005/FR-006/FR-007).
- **Textos dos blocos**: `elevationBlockText` devolve só a altitude;
  `gainBlockText` (novo) devolve só o ganho; `distanceBlockText`/
  `timeBlockText`/`speedBlockText` continuam calculando o mesmo valor de
  sempre, mas cada `formatOverlay*` passa a devolver rótulo/valor/unidade
  como partes separadas (uma unidade vazia para o tempo decorrido), não
  mais uma única string concatenada (FR-013: o cálculo/formatação/
  arredondamento do número em si não muda).
- **`drawProfile`**: não chama mais `drawPanel`; a linha
  (`drawSegment`/`plotSquare`) e o marcador (`drawDot`) desenham primeiro
  uma versão maior de si mesmos em `OverlayTextOutlineColor`, depois a
  versão normal em `OverlayTextColor`/`appearance.MarkerColor`, por cima —
  a mesma técnica de casca-antes-do-núcleo que `TrailCasingColor` já usa
  (FR-012).
- **Removidos**: `stablePanelWidth`, `drawPanel` (sem mais nenhum
  chamador).

## `Scene` (campos removidos) — `frame_scene.go`

| Campo removido | Motivo |
|---|---|
| `panelWidth`, `panelWidthHeight`, `panelWidthSet` | O layout por colunas não depende de medir o texto mais largo de `plan.Frames` — é uma função O(1) de `OverlayConfig` e da disponibilidade de dado no plano (research.md item 1). |

`numericPanelWidth` (método) é removido; `Render` para de chamá-lo e passa
`panelWidth` a `screenOverlay` — o campo também sai da struct literal de
`screenOverlay{...}` em `Render`, já que não existe mais.

## `render_tuning.go` (constantes novas ou removidas)

| Constante | Antes | Depois | Significado |
|---|---|---|---|
| `OverlayPanelColor` | `RGB{0x00,0x00,0x00}` | **removida** | Painel removido (FR-001/FR-012). |
| `OverlayPanelOpacity` | `0.55` | **removida** | Idem. |
| `OverlayLabelHeightRatio` | — | `0.020` (nova) | Corpo do rótulo e da unidade — fração da altura do quadro. |
| `OverlayValueHeightRatio` | — | `0.040` (nova) | Corpo do valor — o dobro do rótulo (FR-005). |

Nenhuma dessas entra em `Appearance` nem em `OverlayConfig` — continuam
fixas, pelo mesmo motivo que `OverlayOutlineRatio`/`ProfileMarkerRadiusRatio`
já são fixas (FR-018): são o acabamento da sobreposição, não uma escolha do
usuário. `OverlayTextColor`, `OverlayTextOutlineColor`,
`OverlayOutlineRatio`/`OverlayOutlineMinWidth`,
`ProfileMarkerRadiusRatio`/`ProfileMarkerMinRadius`,
`OverlayTopMarginRatio`/`OverlaySideMarginRatio`/`OverlayBottomMarginRatio`
não mudam.

## `config.RenderDefaults.OverlayBlocks` (valor padrão) — `config.go`

| Antes | Depois |
|---|---|
| `[]string{"distance", "elevation", "time", "profile"}` | `[]string{"distance", "elevation", "speed", "profile"}` |

Mapeado para `domain.OverlayConfig` por `domainOverlayConfig`
(`cmd/sobrevoo/config_mapping.go`), sem nenhuma mudança nessa função — ela
já converte qualquer lista de strings genericamente (FR-011).

## `builddomain.OverlayConfigBuilder` (método novo) — `overlay_config_builder.go`

| Método novo | Efeito |
|---|---|
| `WithGain()` | Liga `config.Gain = true`, mesmo padrão de `WithSpeed()` (etapa 14). |

## `internal/infra/inbound/cli/overlay.go` (textos e listagem)

- `overlaysUsage`/`overlayBlocksUsage`: passam a citar `gain` entre os
  nomes aceitos e a nova composição do padrão (`distance, elevation,
  speed, profile`).
- `overlayBlocksOf`: ganha `if config.Gain { blocks = append(blocks,
  domain.OverlayBlockGain) }` — usada só para montar o texto de `--help`
  (`formatOverlayBlocks`), não a ordem de desenho (`overlayBlockOrder`).

## O que **não** muda

- `CameraPlan`/`CameraFrame`, `GeoSlice`, `FrameMark`, `FrameSetID`
  (assinatura), `SingleFrameRequest`, `FrameSetRequest`, `FlightRequest`:
  nenhum campo novo, nenhum campo removido — a mudança é só na
  configuração de sobreposição e no desenho.
- Nenhum erro sentinela novo (`gain` reaproveita `ErrInvalidOverlayBlock`).
- Nenhum formato de arquivo (plano `.json`, recorte `.zip`): a
  configuração de sobreposição nunca fez parte deles.
- Os valores calculados por cada bloco, sua precisão e seu arredondamento
  (FR-013) — só a apresentação (rótulo por extenso, três alturas, sem
  painel) muda.
