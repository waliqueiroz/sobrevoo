# Modelo de Dados: Aparência Ajustável dos Quadros

**Feature**: `008-frame-appearance` | **Data**: 2026-09-28

Um tipo de domínio novo (`Appearance`) e mudanças de assinatura em seis tipos e
funções já existentes, para que a aparência escolhida percorra o mesmo caminho
que `Resolution` e `Overwrite` já percorrem hoje, do flag da CLI até a
identidade do conjunto de quadros. Nenhum tipo existente muda de significado;
só ganha um campo ou um parâmetro novo.

## `Appearance` (novo — `internal/domain/frame_appearance.go`)

O conjunto dos cinco valores ajustáveis que um quadro desenha por cima do
terreno: a cor e a espessura do traçado, a cor e o raio do marcador, e a cor
do fundo (`spec.md` → Entidades-Chave). Não inclui `NoMapColors`,
`NoElevationColors`, `PatternPeriod`, `TrailCasingColor`, `MarkerRingColor`,
`MarkerRingRatio`, `MarkerRingMin`, `TrailMinWidth` nem `MarkerMinRadius`, que
continuam fixos em `render_tuning.go` (FR-005).

| Campo | Tipo | Significado |
|---|---|---|
| `TrailColor` | `RGB` | Cor do núcleo do traçado (a casca, `TrailCasingColor`, é fixa). |
| `TrailWidthRatio` | `float64` | Espessura do traçado, como proporção da altura do quadro. |
| `MarkerColor` | `RGB` | Cor do preenchimento do marcador (o anel, `MarkerRingColor`, é fixo). |
| `MarkerRadiusRatio` | `float64` | Raio do marcador, como proporção da altura do quadro. |
| `BackgroundColor` | `RGB` | Cor onde um raio não encontra nada (fora do recorte, acima do horizonte, e por baixo de pixels parcialmente transparentes do mapa base). |

**Regras de validação** (`NewAppearance`, erro por campo — FR-004):

- `TrailWidthRatio`: de `MinTrailWidthRatio` (0.0005) a `MaxTrailWidthRatio`
  (0.05), inclusive; fora disso, `ErrInvalidTrailWidth`.
- `MarkerRadiusRatio`: de `MinMarkerRadiusRatio` (0.001) a
  `MaxMarkerRadiusRatio` (0.1), inclusive; fora disso, `ErrInvalidMarkerRadius`.
- As três cores não têm regra própria em `NewAppearance` — já chegam validadas
  por `ParseColor` antes de `NewAppearance` ser chamado (nenhuma combinação de
  RGB é inválida por si).

**`ParseColor(text string) (RGB, error)`** (novo, em `frame_appearance.go`):
lê `text` como `#RRGGBB` — `#` seguido de exatamente 6 dígitos hexadecimais
(`0-9`, `a-f`, `A-F`); qualquer outra coisa (tamanho errado, sem `#`, dígito
fora do alfabeto hexadecimal, nome de cor, canal alfa) falha com
`ErrInvalidColor`, citando o texto recebido e o formato esperado (o mesmo
estilo de `ParseAspectRatio`: `%w: %q, expected #RRGGBB, for example #FFB000`).

**`Appearance.Fingerprint() string`** (novo): texto canônico dos cinco campos
(`RGB` como `#RRGGBB`, os dois `float64` como `strconv.FormatFloat(..., 'g',
-1, 64)`, na mesma ordem da tabela acima), separados por `|` — mesmo padrão de
`RenderTuning.Fingerprint()`. Usado só para entrar no hash de `NewFrameSetID`
(FR-007); não é exposto em nenhum arquivo nem impresso para o usuário.

## `RGB` (existente — `internal/domain/render_tuning.go`, sem mudança de forma)

Continua `struct { R, G, B uint8 }`. Nenhum campo novo (sem alfa — decisão do
`/speckit-clarify`, Q2).

## Erros sentinela novos (`internal/domain/errors.go`)

| Sentinela | Quando |
|---|---|
| `ErrInvalidColor` | `--trail-color`, `--marker-color` ou `--background-color` não é `#RRGGBB`. |
| `ErrInvalidTrailWidth` | `--trail-width` é um número fora de `[0.0005, 0.05]`. |
| `ErrInvalidMarkerRadius` | `--marker-radius` é um número fora de `[0.001, 0.1]`. |

Três sentinelas — não um genérico `ErrInvalidAppearance` — seguindo o
precedente de `ErrInvalidDuration`/`ErrInvalidFrameRate` (dois valores da
mesma origem, dois erros distintos, cada um citando o campo que falhou).

## `SingleFrameRequest` e `FrameSetRequest` (existentes — `internal/domain/frame_set.go`, ganham um campo)

Ambos ganham `Appearance Appearance`, ao lado de `Resolution`/`Overwrite` que
já tinham — o mesmo padrão de "o que este desenho usa desta vez", por chamada.

## `FlightRequest` (existente — `internal/domain/flight.go`, ganha um campo)

Ganha `Appearance Appearance`, ao lado de `Resolution` que já tinha; repassado
sem alteração ao `FrameSetRequest` que `FlightService.Fly` monta internamente
(pesquisa, item 7).

## `NewFrameMark` / `NewFrameSetID` (existentes — `internal/domain/frame_set.go`, ganham um parâmetro)

```go
func NewFrameMark(plan CameraPlan, slice GeoSlice, resolution Resolution, tuning RenderTuning, appearance Appearance) FrameMark
func NewFrameSetID(plan CameraPlan, slice GeoSlice, resolution Resolution, tuning RenderTuning, appearance Appearance) FrameSetID
```

`newFrameSetID` (privada) escreve `appearance.Fingerprint()` no hash, depois de
`tuning.Fingerprint()` — ver pesquisa, item 5. `FrameMark`/`FrameSetID` em si
não mudam de forma (continuam só `SetID`/`PlanID`): a aparência participa só
pelo hash, nunca é lida de volta de um `FrameSetID`.

## `NewScene` / `Scene` / `overlay` (existentes — `internal/domain/frame_scene.go`, `frame_overlay.go`, ganham um parâmetro)

```go
func NewScene(slice GeoSlice, decoder TileDecoder, tuning RenderTuning, appearance Appearance) (*Scene, error)
```

`Scene` ganha o campo `appearance Appearance`; `Scene.Render` usa
`s.appearance.BackgroundColor` para `NewFrameImage` e monta `overlay{...,
appearance: s.appearance}`; `overlay.drawTrail` usa
`o.appearance.TrailColor`/`TrailWidthRatio` no lugar dos antigos
`TrailColor`/`TrailWidthRatio` de pacote; `overlay.drawMarker`, o mesmo para
`MarkerColor`/`MarkerRadiusRatio`. `TrailCasingColor`, `MarkerRingColor`,
`MarkerRingRatio`, `MarkerRingMin`, `TrailMinWidth`, `MarkerMinRadius`
continuam lidos como constantes/`var` de pacote, sem mudança.

## `NewFrameImage` (existente — `internal/domain/frame_image.go`, ganha um parâmetro)

```go
func NewFrameImage(resolution Resolution, background RGB) FrameImage
```

Preenche a imagem com `background` no lugar do antigo `BackgroundColor` de
pacote.

## `imagery` / `newImagery` (existentes — `internal/domain/frame_imagery.go`, ganham um parâmetro)

```go
func newImagery(tileSets []TileSet, decoder TileDecoder, cacheBytes int64, background RGB) *imagery
```

`imagery` ganha o campo `background RGB`, repassado para `newTileTexture` (que
passa a receber `background RGB` como parâmetro, no lugar do antigo
`BackgroundColor` de pacote) sempre que decodifica e prepara a textura de uma
peça — a mistura de um pixel parcialmente transparente com o fundo escolhido
(pesquisa, item 6).

## `render_tuning.go` — o que sai, o que fica

**Sai** (migra para `config.RenderDefaults` + `Appearance`, pesquisa item 6):
`BackgroundColor`, `TrailColor`, `MarkerColor`, `TrailWidthRatio`,
`MarkerRadiusRatio`.

**Fica** (constantes/`var` fixos, não ajustáveis — FR-005/FR-006):
`NoMapColors`, `NoElevationColors`, `PatternPeriod`, `TrailCasingColor`,
`MarkerRingColor`, `TrailMinWidth`, `MarkerMinRadius`, `MarkerRingRatio`,
`MarkerRingMin`, `RenderVersion`, `RenderTuning` (sem mudança de campos).

## `config.RenderDefaults` (existente — `internal/infra/outbound/config/config.go`, ganha cinco campos)

```go
type RenderDefaults struct {
	Width, Height int

	TrailColor        string  // "#FFB000"
	TrailWidthRatio    float64 // 0.005
	MarkerColor        string  // "#E5252A"
	MarkerRadiusRatio  float64 // 0.012
	BackgroundColor    string  // "#20262E"
}
```

Os cinco valores são exatamente os que `render_tuning.go` tinha como `var` de
pacote — só o lugar muda. `config.Load()` os preenche junto com
`Width`/`Height`.

## `domainAppearance` (novo — `cmd/sobrevoo/config_mapping.go`)

```go
func domainAppearance(d config.RenderDefaults) (domain.Appearance, error)
```

Chama `domain.ParseColor` três vezes (uma por cor) e `domain.NewAppearance`
uma vez, devolvendo o primeiro erro que encontrar — mesmo padrão de
`domainRenderResolution`, que já mapeia `Width`/`Height` para
`domain.NewResolution`. Uma falha aqui só pode vir de um literal errado em
`config.Load()` (erro de programação, não de entrada do usuário): `main.go` a
trata como as demais falhas de composição do processo (interrompe a
inicialização com a mensagem de erro).

## `parseAppearance` (novo — `internal/infra/inbound/cli/appearance.go`)

```go
func parseAppearance(cmd *cobra.Command, trailColorFlag, trailWidthFlag, markerColorFlag, markerRadiusFlag, backgroundColorFlag string, defaults domain.Appearance) (domain.Appearance, error)
```

Um parser só, chamado por `render frame`, `render all` e `fly` (pesquisa, item
4) — garante FR-002 por construção. Erros:

- `--trail-color`/`--marker-color`/`--background-color` inválidos:
  `ErrInvalidColor` (via `domain.ParseColor`), sem embrulho de uso.
- `--trail-width`/`--marker-radius` que não são um número finito
  (`parseFiniteNumber`, o mesmo helper de `--fps`/`--duration`): `usageError`
  (código de saída `2`).
- `--trail-width`/`--marker-radius` numéricos mas fora do intervalo:
  `ErrInvalidTrailWidth`/`ErrInvalidMarkerRadius` (via `domain.NewAppearance`),
  sem embrulho de uso.

## `internal/domain/builddomain` (dois arquivos afetados)

- **Novo** `appearance_builder.go`: `NewAppearanceBuilder()` com os cinco
  valores de hoje como padrão (`#FFB000`/0.005/`#E5252A`/0.012/`#20262E`),
  `WithTrailColor(RGB)`, `WithTrailWidthRatio(float64)`, `WithMarkerColor(RGB)`,
  `WithMarkerRadiusRatio(float64)`, `WithBackgroundColor(RGB)`, `Build()
  Appearance`.
- **Alterado** `flight_request_builder.go`: `NewFlightRequestBuilder()` passa a
  preencher `Appearance: NewAppearanceBuilder().Build()`; ganha
  `WithAppearance(domain.Appearance) *FlightRequestBuilder`.

## Diagrama de dependência (quem chama quem, de cima para baixo)

```text
CLI (render_frame.go / render_all.go / fly.go)
  └─ appearance.go: parseAppearance(...) → domain.Appearance
       ├─ domain.ParseColor (×3)
       └─ domain.NewAppearance

domain.SingleFrameRequest.Appearance ─┐
domain.FrameSetRequest.Appearance ────┼─→ application.FrameService.DrawFrame/DrawFrames
domain.FlightRequest.Appearance ──────┘        (repassado por application.FlightService.Fly)
                                            │
                                            ├─→ domain.NewScene(..., appearance)
                                            │      ├─ Scene.appearance → overlay.appearance (traçado, marcador)
                                            │      └─ newImagery(..., appearance.BackgroundColor) → newTileTexture
                                            │
                                            └─→ domain.NewFrameMark(..., appearance)
                                                   └─ NewFrameSetID(..., appearance)
                                                        └─ hash += appearance.Fingerprint()
```
