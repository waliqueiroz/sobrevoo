# Modelo de Dados: Rótulos em Português e Legibilidade do Texto das Sobreposições

Só o que muda ou se acrescenta em relação ao modelo já existente (`specs/
011-overlay-polish/data-model.md`). Racional de cada decisão em
`research.md`. Nenhuma entidade persistida (plano, recorte, `OverlayConfig`,
`Appearance`) muda de forma — só o desenho, inteiramente dentro de
`internal/domain`.

## `domain.RenderVersion` (constante) — `render_tuning.go`

| Antes | Depois |
|---|---|
| `3` | `4` |

Automaticamente propagado para `FrameSetID` por `newFrameSetID`
(`frame_set.go`), sem nenhuma mudança de assinatura.

## Fonte embutida — `vector_font.go`

| Antes | Depois |
|---|---|
| `golang.org/x/image/font/gofont/goregular` (peso regular) | `golang.org/x/image/font/gofont/gobold` (peso forte) |

Só a linha que passa o `[]byte` da fonte a `sfnt.Parse`, dentro de
`newVectorFace()`, muda; `vectorFace`, `glyphMask`, `glyphKey`, o
achatamento de curvas e o rasterizador (`rasterizeOutline`,
`windingNumber`) não mudam.

## `render_tuning.go` (constante redefinida)

| Constante | Antes | Depois | Significado |
|---|---|---|---|
| `OverlayOutlineRatio` | `0.0025` (fração da **altura do quadro**) | `0.035` (fração do **`ppem`**, o tamanho do glifo) | Multiplicador do raio de dilatação do contorno do texto — agora proporcional ao tamanho da letra, não ao quadro inteiro (FR-005). |
| `OverlayOutlineMinWidth` | `1.0` | sem mudança | Piso em pixels do raio de dilatação — continua garantindo que o contorno nunca desapareça. |

## `internal/domain/frame_screen_overlay.go` (estendido)

### Funções novas — texto de cada bloco numérico

```go
func distanceBlockText(frame CameraFrame) string
func elevationBlockText(frame CameraFrame) string
func timeBlockText(frame CameraFrame) string
```

Cada uma monta o texto de um bloco — rótulo em português (research.md
item 1) + o(s) valor(es) formatado(s) pelas funções `formatOverlay*` já
existentes, sem mudança de formatação. `draw` e `stablePanelWidth` (abaixo)
são os dois únicos chamadores — nunca o texto escrito duas vezes.

### Função nova — `overlayPpem`

```go
func overlayPpem(height int) int
```

O tamanho do glifo (`ppem`) para um quadro de `height` pixels de altura —
antes calculado inline dentro de `draw`, agora também usado por
`Scene.Render` (via `Scene.numericPanelWidth`) antes de `draw` existir
para aquele quadro.

### Função nova — `stablePanelWidth`

```go
func stablePanelWidth(face *vectorFace, plan CameraPlan, config OverlayConfig, ppem int) int
```

A largura compartilhada dos três painéis numéricos, calculada a partir de
**todos** os quadros de `plan` — o maior `face.textWidth(...)` entre os
três textos de bloco (via `distanceBlockText`/`elevationBlockText`/
`timeBlockText`) de cada quadro, pulando um bloco que `config`/`plan` não
mostram (mesmo critério de `draw`: `config.Elevation &&
plan.ElevationAvailable`, `config.Time && plan.TimeReference ==
TimeReferenceClock`); `0` quando nenhum bloco numérico aparece
(research.md item 4).

### `numericPanelWidth` (removida) — substituída por `stablePanelWidth` + o cache de `Scene`

A função antiga, que calculava a largura só a partir dos textos de **um**
quadro, deixa de existir — seu papel passa para `stablePanelWidth` (todo o
plano) mais o cache de `Scene` (abaixo), que juntos garantem a
estabilidade que FR-007/FR-008 exigem.

### `screenOverlay` (estendida)

| Campo novo | Tipo | Significado |
|---|---|---|
| `panelWidth` | `int` | A largura compartilhada dos painéis numéricos, já calculada por `Scene` (não mais calculada por `draw`). |

`draw` deixa de chamar `numericPanelWidth`/`stablePanelWidth` — usa
`s.panelWidth` diretamente nas três chamadas de `drawLine`.

### `drawText` (comportamento alterado, mesma assinatura)

O raio de dilatação do contorno passa de
`max(OverlayOutlineMinWidth, OverlayOutlineRatio × altura do quadro)` para
`max(OverlayOutlineMinWidth, OverlayOutlineRatio × ppem)` — usa o `ppem`
que o próprio `drawText` já recebe como parâmetro, não mais
`s.image.Resolution.Height` (research.md item 2).

## `internal/domain/frame_scene.go` (estendido)

### `Scene` (estendida) — campos novos, não exportados

| Campo novo | Tipo | Significado |
|---|---|---|
| `panelWidth` | `int` | A largura compartilhada em cache, válida quando `panelWidthSet` é `true` e `panelWidthHeight` bate com a altura pedida. |
| `panelWidthHeight` | `int` | A altura de resolução para a qual `panelWidth` foi calculada — recalcula se uma `Render` seguinte pedir uma altura diferente. |
| `panelWidthSet` | `bool` | Se `panelWidth`/`panelWidthHeight` já foram calculados ao menos uma vez. |

### Método novo — `Scene.numericPanelWidth`

```go
func (s *Scene) numericPanelWidth(plan CameraPlan, height, ppem int) int
```

Devolve `stablePanelWidth(s.face, plan, s.overlayConfig, ppem)`, calculado
só na primeira chamada (ou quando `height` muda desde a última); nas
chamadas seguintes com a mesma altura, devolve o valor em cache
(research.md item 4) — o mesmo padrão de "calcular uma vez por `Scene`,
reaproveitar depois" que `vectorFace` já estabelece para o cache de
glifos, com uma guarda extra por `height` (barata, O(1)) porque, ao
contrário do cache de glifos, a largura cacheada não carrega sozinha a
chave de que depende.

### `Scene.Render` (comportamento alterado, mesma assinatura)

Antes de montar `screenOverlay{...}`, calcula `ppem := overlayPpem(resolution.Height)`
e `panelWidth := s.numericPanelWidth(plan, resolution.Height, ppem)`,
passando `panelWidth: panelWidth` no literal de `screenOverlay`.

## O que **não** muda

- `OverlayConfig`, `Appearance`, `CameraPlan`/`CameraFrame`, `GeoSlice`,
  `FrameMark`, `FrameSetID` (assinatura), `SingleFrameRequest`,
  `FrameSetRequest`, `FlightRequest`: nenhum campo novo, nenhum campo
  removido.
- `NewScene` (assinatura): sem parâmetro novo — o cache de largura, como o
  de glifos, é construído e preenchido internamente, não injetado (não há
  escolha de usuário envolvida).
- Nenhum erro sentinela novo: nada que o usuário escolhe pode ficar
  malformado por esta etapa.
- Nenhum formato de arquivo (plano `.json`, recorte `.zip`, quadro `.png`):
  o bloco `tEXt` do `frame-set=<id>` já absorve o `RenderVersion` novo sem
  mudar de estrutura (`contracts/frame-files-change.md`).
- `config.RenderDefaults`/`config_mapping.go`: nenhum valor novo a
  configurar — tudo isto é fixo.
- O rasterizador (`rasterizeOutline`, `windingNumber`, `flattenQuad`/
  `flattenCube`) e a extração de contornos (`sfnt.LoadGlyph`) da etapa
  anterior: inalterados (FR-012).
