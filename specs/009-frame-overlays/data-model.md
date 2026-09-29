# Modelo de Dados: Sobreposições de Tela nos Quadros

Só o que muda ou se acrescenta em relação ao modelo já existente (`specs/
003-camera-path-planning/data-model.md`, `specs/008-frame-appearance/`).
Racional de cada decisão em `research.md`.

## `CameraFrame` (estendido) — `internal/domain/camera_plan.go`

Três campos novos, ao lado dos já existentes:

| Campo | Tipo | Significado |
|---|---|---|
| `ActivityElapsed` | `time.Duration` | Tempo real decorrido da atividade, desde o primeiro ponto do trajeto, até o instante em que o marcador está neste quadro. Sem sentido (valor zero) quando `CameraPlan.TimeReference == TimeReferenceDistance` — mesma condição que já governa `Time` vs. tempo real hoje. |
| `TrackElevation` | `float64` | Elevação do trajeto (metros), interpolada linearmente, no ponto do marcador deste quadro. Sem sentido (valor zero) quando `CameraPlan.ElevationAvailable == false`. |
| `TrackElevationGain` | `float64` | Ganho de elevação acumulado (metros) do início do trajeto até o ponto do marcador deste quadro — não-decrescente ao longo dos quadros; no último quadro, igual a `Route.ElevationGain()` do trajeto tratado. Sem sentido (valor zero) quando `CameraPlan.ElevationAvailable == false`. |

Quantização: os dois campos em metros usam `lengthStep` (1e-3), como
`MarkerDistance`/`CameraToMarkerDistance` já usam; `ActivityElapsed` é
guardado com a mesma resolução de segundos que `Time` já usa no arquivo
(9 casas decimais, function `seconds()`).

Nenhum campo existente de `CameraFrame` muda de tipo ou de significado.

## `CameraPlan` / `PlanSummary` (estendidos)

| Campo novo | Tipo | Onde | Significado |
|---|---|---|---|
| `ElevationAvailable` | `bool` | `CameraPlan` e `PlanSummary` | Se o trajeto tratado tinha elevação em todos os pontos (mesmo critério de `Route.ElevationGain()`'s segundo retorno). Replicado em `PlanSummary` do mesmo jeito que `TimeReference` já é. |

`CameraPlan.ID()` **não muda** (research.md item 9) — os campos novos são
funções determinísticas dos campos já hasheados.

## `domain.Route` (estendido) — novos métodos

| Método | Arquivo | Assinatura | Comportamento |
|---|---|---|---|
| `TimeAt` | `duration.go` | `(r Route) TimeAt(distances []float64, at float64) (time.Duration, bool)` | Interpola o tempo real decorrido desde o primeiro ponto, na distância `at`; `false` quando `!r.allHaveTime()`. |
| `ElevationProfile` | `elevation_profile.go` (novo) | `(r Route) ElevationProfile(distances []float64) (ElevationProfile, bool)` | Pré-computa elevação e ganho acumulado em cada ponto; `false` quando `!r.allHaveElevation()`. |

## `domain.ElevationProfile` (novo tipo) — `elevation_profile.go`

```go
type ElevationProfile struct {
    Distances  []float64 // igual ao de PlanarRoute.Distances, mesma rota
    Elevations []float64
    Gains      []float64 // não-decrescente
}

func (e ElevationProfile) At(distance float64) (elevation, gain float64)
```

Mesma técnica de busca por bracket + interpolação linear que
`PlanarRoute.PointAt` já usa, sobre o mesmo array `Distances`.

## `domain.OverlayConfig` (novo) — `frame_overlay_config.go`

```go
type OverlayBlock string

const (
    OverlayBlockDistance  OverlayBlock = "distance"
    OverlayBlockElevation OverlayBlock = "elevation"
    OverlayBlockTime      OverlayBlock = "time"
    OverlayBlockProfile   OverlayBlock = "profile"
)

type OverlayConfig struct {
    Enabled  bool
    Distance bool
    Elevation bool
    Time     bool
    Profile  bool
}

func NewOverlayConfig(enabled bool, blocks []OverlayBlock) (OverlayConfig, error)
func (o OverlayConfig) Fingerprint() string
```

`NewOverlayConfig` valida cada elemento de `blocks` contra as quatro
constantes; um nome desconhecido é `ErrInvalidOverlayBlock`. `Fingerprint()`
segue o mesmo formato canônico de `Appearance.Fingerprint()`/`RenderTuning.
Fingerprint()` (texto determinístico, uma linha, campos separados por `|`).

## `RenderTuning` (estendido) — constantes fixas novas, `render_tuning.go`

Não ajustáveis (mesma categoria de `TrailCasingColor`/`NoMapColors`/
`PatternPeriod` — significado, não estilo):

| Constante | Valor inicial | Significado |
|---|---|---|
| `OverlayMarginRatio` | `0.06` | Margem de segurança, como fração de `min(largura, altura)` do quadro — nenhum bloco desenha mais perto de qualquer borda que isso. |
| `OverlayPanelColor` | `RGB{0x00,0x00,0x00}` | Cor da placa de fundo semitransparente de cada bloco. |
| `OverlayPanelOpacity` | `0.55` | Opacidade da placa. |
| `OverlayTextColor` | `RGB{0xFF,0xFF,0xFF}` | Cor do texto e do traço do perfil de elevação. |

## `FrameSetID` (estendido) — `frame_set.go`

`NewFrameSetID`/`newFrameSetID` ganham um parâmetro `overlay
domain.OverlayConfig`, e o hash inclui `overlay.Fingerprint()` depois de
`appearance.Fingerprint()`. `NewFrameMark` ganha o mesmo parâmetro novo.
`RenderVersion` **não muda** — o mesmo raciocínio de 008-frame-appearance
se aplica: estender o hash já garante que uma configuração de sobreposição
diferente (inclusive "nenhuma", que nunca escreveu esse segmento antes)
nunca bate com a de agora.

## `SingleFrameRequest` / `FrameSetRequest` (estendidos) — `frame_set.go`

Ambos ganham um campo `Overlay domain.OverlayConfig`, ao lado de
`Appearance`.

## `FlightRequest` (estendido) — `flight_service.go` (domínio)

Ganha um campo `Overlay domain.OverlayConfig`, usado nas etapas de desenho
de quadros e vídeo do comando único; não muda nada nas etapas de plano ou
recorte (FR-015).

## Formato do arquivo do plano (`jsonfile`) — ver `contracts/plan-file-v2.md`

`format_version` sobe de `1` para `2`. Campos novos: `frames[].
activity_time_s`, `frames[].marker.elevation_m`, `frames[].marker.gain_m`,
`summary.elevation_available`.

## Sentinela de erro novo — `internal/domain/errors.go`

| Nome | Quando |
|---|---|
| `ErrInvalidOverlayBlock` | Um nome de bloco de sobreposição que não é `distance`, `elevation`, `time` ou `profile` (FR-017). |

`ErrPlanFileInvalid` e `ErrPlanFormatVersionUnsupported` (já existentes)
cobrem, respectivamente, um plano v2 malformado e um plano de versão
anterior (FR-007) — nenhum sentinela novo para isso.
