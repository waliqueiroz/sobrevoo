# Modelo de Dados: Iluminação Direcional do Terreno

Só o que muda ou se acrescenta em relação ao modelo já existente (`specs/
005-frame-rendering/data-model.md`, `specs/011-overlay-polish/data-model.md`).
Racional de cada decisão em `research.md`. Nenhuma entidade persistida
(`CameraPlan`, `GeoSlice`, `Appearance`, `OverlayConfig`) muda de forma — só
o desenho, inteiramente dentro de `internal/domain`.

## `domain.RenderVersion` (constante) — `render_tuning.go`

| Antes | Depois |
|---|---|
| `5` | `6` |

Automaticamente propagado para `FrameSetID` por `newFrameSetID`
(`frame_set.go`), sem nenhuma mudança de assinatura — nenhum conjunto de
quadros desta versão é tomado como o de uma versão anterior (FR-010).

## Constantes fixas novas — `render_tuning.go`

| Constante | Valor | Significado |
|---|---|---|
| `TerrainLightAzimuthDegrees` | `315` | Direção da luz, graus no sentido horário a partir do norte — mesma convenção de `CameraFrame.Heading` (research.md item 2). |
| `TerrainLightAltitudeDegrees` | `45` | Altura da luz acima do horizonte, em graus. |
| `TerrainLightMinFactor` | `0.75` | Limite escuro da faixa fixa de brilho (research.md item 8 — revisado de `0.6` após um mapa claro real mostrar saturação relevante do lado claro). |
| `TerrainLightMaxFactor` | `1.15` | Limite claro da faixa fixa de brilho (revisado de `1.4`; o máximo geométrico real da fórmula é `≈1.293`, então `1.4` nunca era alcançado). |

Nenhuma delas entra em `Appearance`, `OverlayConfig` ou `RenderTuning`
(injetada por configuração): são fixas, como `NoMapColors`/
`OverlayTextColor` já são, porque não são escolha do usuário (FR-011).

## Direção da luz (derivada, não persistida) — `frame_terrain_light.go`

```go
// terrainLightDirection is the fixed, unit direction toward the light,
// in the frame plane's basis (x east, y north, z up) — computed once from
// TerrainLightAzimuthDegrees/TerrainLightAltitudeDegrees, never per pixel
// or per frame (research.md item 2).
var terrainLightDirection = newTerrainLightDirection()

func newTerrainLightDirection() [3]float64
```

## `terrainLightFactor` (função livre nova) — `frame_terrain_light.go`

```go
// terrainLightFactor is how much a point of terrain with the given unit
// normal is lightened or darkened: 1 for a flat surface, whatever the
// light's altitude; brighter toward TerrainLightMaxFactor for a slope that
// faces the light, darker toward TerrainLightMinFactor for one that faces
// away (research.md item 1).
func terrainLightFactor(nx, ny, nz float64) float64
```

Matemática sem dono sobre três números — a mesma categoria que `clamp`/
`quantize` já ocupam no domínio (CLAUDE.md, "Onde vive a regra de
negócio").

## `surface` (estendida) — `frame_surface.go`

| Campo novo | Tipo | Significado |
|---|---|---|
| `gradients` | `[]terrainGradient` | A pirâmide de derivadas médias da altura, construída uma vez em `newSurface` a partir das amostras cruas (`grid.values`, nunca `s.heights`) — research.md item 5. Nível 0 na resolução da grade; cada nível seguinte, metade das linhas/colunas, até 1×1. |

```go
// terrainGradient is one level of a surface's precomputed slope pyramid:
// the average height-change per cell, in each grid direction, and how
// much of this level is backed by real elevation samples — never the
// hole-filled heights used for the ray's geometry (research.md item 5).
type terrainGradient struct {
    rows, cols int
    dRow, dCol []float32 // meters of height change per cell, index space
    coverage   []float32 // 0 (no real sample contributed) to 1 (every one did)
}

func newTerrainGradientPyramid(raw []float32, rows, cols int) []terrainGradient
```

`newSurface` chama `newTerrainGradientPyramid(grid.values, s.rows, s.cols)`
e guarda o resultado em `s.gradients`, ao lado de onde já calcula
`s.zmin`/`s.zmax`.

## `placedSurface` (estendida) — `frame_surface.go`

| Método novo | Assinatura | Significado |
|---|---|---|
| `normalAt` | `func (g placedSurface) normalAt(x, y, footprintMeters float64) (nx, ny, nz float64)` | A normal da superfície em `(x, y)` (metros, espaço do quadro), lida da pirâmide no nível proporcional a `footprintMeters` (research.md item 6), subindo um nível quando a cobertura do nível escolhido é zero naquela posição (research.md item 7, caso 3) — sempre devolve uma normal, nunca falha; `(0, 0, 1)` (plana) quando nenhum nível tem cobertura. |

`footprintMeters` é calculado pelo chamador (`Scene.drawPixel`, research.md
item 6), não por `normalAt` — a mesma divisão de responsabilidade que
`sampler.color` já tem com `pixelAngle`/`distance`/`descent` passados pelo
chamador.

## `Scene.drawPixel` (comportamento estendido) — `frame_scene.go`

Sem mudança de assinatura. No ramo `case stateImage` (único ramo afetado,
FR-006):

```
footprint := pixelAngle · hit.t / sqrt(max(|dz|, 0.1))   // mesma correção de rasante do item 6
nx, ny, nz := ground[hit.grid].normalAt(hit.x, hit.y, footprint)
fator := terrainLightFactor(nx, ny, nz)
color := RGB{rounded(float64(color.R)·fator), rounded(float64(color.G)·fator), rounded(float64(color.B)·fator)}
image.Set(x, y, color)
```

Os ramos `stateNoMap` e `stateNoElevation` continuam exatamente como são
hoje — nenhuma leitura da pirâmide, nenhuma multiplicação (FR-006).

## O que **não** muda

- `CameraPlan`/`CameraFrame`, `GeoSlice`, `ElevationGrid`, `Appearance`,
  `OverlayConfig`, `FrameMark`, `FrameSetID` (assinatura),
  `SingleFrameRequest`, `FrameSetRequest`, `FlightRequest`: nenhum campo
  novo, nenhum campo removido (FR-013).
- Nenhum erro sentinela novo: nada que o usuário escolhe pode ficar
  malformado por esta etapa (FR-011).
- Nenhum formato de arquivo (plano `.json`, recorte `.zip`, quadro `.png`):
  só os pixels do terreno dentro do quadro mudam
  (`contracts/frame-files-change.md`).
- `config.RenderDefaults`/`config_mapping.go`: nenhum valor novo a
  configurar — tudo isto é fixo.
- `overlay` (traçado/marcador, `frame_overlay.go`) e `screenOverlay`
  (sobreposições de tela, `frame_screen_overlay.go`): nenhuma mudança —
  continuam desenhados depois, sobre o terreno já iluminado, sem conhecer
  a iluminação (FR-006).
