# Modelo de Dados: Acabamento das Sobreposições de Tela

Só o que muda ou se acrescenta em relação ao modelo já existente (`specs/
009-frame-overlays/data-model.md`, `specs/008-frame-appearance/
data-model.md`). Racional de cada decisão em `research.md`. Nenhuma
entidade persistida (plano, recorte, `OverlayConfig`, `Appearance`) muda de
forma — só o desenho, inteiramente dentro de `internal/domain`.

## `domain.RenderVersion` (constante) — `render_tuning.go`

| Antes | Depois |
|---|---|
| `2` | `3` |

Automaticamente propagado para `FrameSetID` por `newFrameSetID`
(`frame_set.go`), sem nenhuma mudança de assinatura — nenhum conjunto de
quadros desta versão é tomado como o de uma versão anterior (FR-008).

## Fonte vetorial e rasterizador (novo) — `vector_font.go`

Tipos e funções não exportados, internos ao domínio — nenhum deles cruza
para `internal/application` ou para fora do pacote `domain`:

```go
// vectorFace carrega os contornos de uma fonte TrueType embutida (Go
// Regular) e rasteriza, com cache, cada glifo que algum bloco de
// sobreposição precisa, num tamanho em pixels fixo para toda a execução.
type vectorFace struct {
    font  *sfnt.Font
    buf   sfnt.Buffer // reuso: nunca usado por duas goroutines à vez (research.md item 3)
    cache map[glyphKey]glyphMask
}

type glyphKey struct {
    r    rune
    ppem int // tamanho do glifo em pixels, nesta execução
}

// glyphMask é a cobertura (0 a 255) de cada pixel do retângulo do glifo,
// calculada pelo rasterizador próprio (research.md item 4) — nunca por
// golang.org/x/image/vector.Rasterizer.
type glyphMask struct {
    width, height int
    advance       int // avanço horizontal até o próximo glifo, em pixels
    coverage      []uint8
}

func newVectorFace() *vectorFace
func (f *vectorFace) glyph(r rune, ppem int) (glyphMask, bool) // rasteriza ou lê do cache
func (f *vectorFace) textWidth(text string, ppem int) int      // soma de advance, para medir um painel
```

O algoritmo de `glyph` (achatamento de curvas + superamostragem 4×4,
nonzero winding) está documentado em `research.md` item 4; não é repetido
aqui porque não é modelo de dado, é comportamento.

## `Scene` (estendida) — `frame_scene.go`

| Campo novo | Tipo | Significado |
|---|---|---|
| `face` | `*vectorFace` | Construído uma vez em `NewScene`, reaproveitado por todo `Render` da mesma instância — o cache de glifos vale para toda a execução de `render all`/`fly` (research.md item 5). |

`NewScene` não ganha nenhum parâmetro novo: `face` é construído
internamente, não injetado (não há escolha de usuário envolvida).

## `screenOverlay` (estendida) — `frame_screen_overlay.go`

| Campo novo | Tipo | Significado |
|---|---|---|
| `face` | `*vectorFace` | Repassado de `Scene.face` a cada `Render`, no lugar do antigo `overlayFace` de pacote (`inconsolata.Bold8x16`, removido). |

Comportamento que muda (sem mudar assinatura pública de `draw`):

- **Medição antes do desenho**: `draw` passa a medir a largura de cada bloco
  numérico presente (distância; elevação+ganho; tempo decorrido) e usar o
  maior valor como a largura dos três painéis (FR-005, research.md item 7).
- **Contorno do texto**: `drawLine`/`drawText` passam a desenhar, para cada
  glifo, a máscara dilatada em `OverlayTextOutlineColor` antes da máscara
  original em `OverlayTextColor` (FR-004, research.md item 6).
- **Raio do marcador do perfil**: `drawDot`, chamado por `drawProfile`, usa
  `max(ProfileMarkerMinRadius, ProfileMarkerRadiusRatio * altura)` no lugar
  de `max(2, scale)` (FR-006, research.md item 8).
- **Margem por borda**: `margin` (um único valor) é substituído por três —
  topo, laterais, base — calculados separadamente (FR-007, research.md
  item 9).

Removido: `overlayFace` (variável de pacote), `textScale`, `glyphOffset`,
e a dependência de `golang.org/x/image/font/basicfont`/`font/inconsolata` —
nenhum deles é mais necessário com a fonte vetorial.

## `render_tuning.go` (constantes fixas — novas, renomeadas ou removidas)

| Constante | Antes | Depois | Significado |
|---|---|---|---|
| `OverlayMarginRatio` | `0.06` (fração do lado menor, 4 bordas iguais) | **removida** | Substituída pelas três abaixo (FR-007). |
| `OverlayTopMarginRatio` | — | `0.06` (nova) | Margem do topo, fração da **altura**. |
| `OverlaySideMarginRatio` | — | `0.06` (nova) | Margem das laterais, fração da **largura**. |
| `OverlayBottomMarginRatio` | — | `0.14` (nova) | Margem da base, fração da **altura** — maior que as outras três. |
| `OverlayTextColor` | `RGB{0xFF,0xFF,0xFF}` | sem mudança | Cor do texto e do traço/marcador do perfil. |
| `OverlayTextOutlineColor` | — | `RGB{0x10,0x10,0x10}` (nova) | Cor do contorno do texto — mesmo tom escuro de `TrailCasingColor` (FR-004). |
| `OverlayOutlineRatio` | — | nova (fração da altura) | Raio de dilatação do contorno do texto. |
| `OverlayOutlineMinWidth` | — | nova (piso em pixels) | Piso do raio de dilatação, mesma categoria de `TrailMinWidth`. |
| `ProfileMarkerRadiusRatio` | — | nova (fração da altura) | Raio do marcador do perfil de elevação (FR-006). |
| `ProfileMarkerMinRadius` | — | nova (piso em pixels) | Piso do raio do marcador do perfil, mesma categoria de `MarkerMinRadius`. |

Nenhuma dessas constantes entra em `Appearance` nem em `OverlayConfig` —
continuam fixas, pelo mesmo motivo que `NoMapColors`/`TrailCasingColor`/
`OverlayPanelColor` já são fixas (FR-011): são o acabamento da sobreposição,
não uma escolha do usuário.

## O que **não** muda

- `OverlayConfig`, `Appearance`, `CameraPlan`/`CameraFrame`, `GeoSlice`,
  `FrameMark`, `FrameSetID` (assinatura), `SingleFrameRequest`,
  `FrameSetRequest`, `FlightRequest`: nenhum campo novo, nenhum campo
  removido.
- Nenhum erro sentinela novo: nada que o usuário escolhe pode ficar
  malformado por esta etapa (FR-010/FR-011).
- Nenhum formato de arquivo (plano `.json`, recorte `.zip`, quadro `.png`):
  o bloco `tEXt` do `frame-set=<id>` já absorve o `RenderVersion` novo sem
  mudar de estrutura (`contracts/frame-files-change.md`).
- `config.RenderDefaults`/`config_mapping.go`: nenhum valor novo a
  configurar — tudo isto é fixo.
