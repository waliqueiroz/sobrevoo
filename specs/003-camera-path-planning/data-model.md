# Modelo de Dados: Planejamento do Movimento de Câmera

**Feature**: `003-camera-path-planning` | **Data**: 2026-09-19

Todos os tipos abaixo vivem em `internal/domain` (Princípio IX: DTOs de
saída não triviais e regra de negócio são do domínio, nunca de
`internal/application`). Nomes de campos e tipos em inglês, como no restante
do código. Nada aqui é persistido pelo núcleo; o único destino externo é o
arquivo exportado (`contracts/plan-file.md`).

## Entidades

### `PlanParameters` (`camera_plan_parameters.go`)

Parâmetros que o usuário escolhe (FR-003, FR-005).

| Campo | Tipo | Regra |
|---|---|---|
| `Duration` | `*time.Duration` | `nil` = automática (FR-003a); se informada, > 0 (`ErrInvalidDuration`) e ≥ duração mínima do trajeto (`ErrDurationTooShort`, verificada em `PlanCamera`) |
| `FrameRate` | `float64` | 1 ≤ valor ≤ 120 (`ErrInvalidFrameRate`); NaN/infinito também inválidos |
| `Distance` | `Level` | `low`, `medium` ou `high` (tipo já existente) |
| `Tilt` | `Level` | idem |

`Validate() error` aplica as regras de intervalo (só à duração, quando
informada). Quantidade de quadros: `FrameCount(duration time.Duration) int`
= `round(duration.Seconds() × FrameRate)` (arredondamento para o inteiro
mais próximo, empates para cima), FR-004; recebe a duração já resolvida,
seja a informada ou a automática.

### `CameraTuning` (`camera_plan_parameters.go`)

Constantes de ajuste do algoritmo, injetadas (research.md item 12). Campos:
`OpeningFraction`, `ClosingFraction`, `StopSpeedMetersPerSecond`,
`StopMinDuration`, `StopCappedDuration`, `StopMaxShareOfMovingTime`,
`MaxHeadingRateDegPerSecond`, `MaxTiltRateDegPerSecond`,
`MaxLogDistanceRatePerSecond`, `MaxTargetSpeedInDistances`,
`GaussianSigmaSeconds`, `OverviewTiltDegrees`, `OverviewVerticalFOVDegrees`,
`OverviewMargin`, `OverviewMinDistanceFactor`, `MinFollowDuration`,
`MinPhaseDuration`, `MinTrackLengthMeters`, `MaxTrackSpanMeters`, os da duração automática
(`AutoDurationBase`, `AutoDurationPerSqrtKm`, `AutoDurationMin`,
`AutoDurationMax`), mais a tabela de níveis:
`BaseDistanceMeters`, `LookAheadSeconds` e `TiltDegrees`, cada um indexado
por `Level`. Valores iniciais em `research.md` itens 3 a 8.

Os padrões dos parâmetros do usuário (taxa 30, distância e inclinação
`medium`) **não** fazem parte de `CameraTuning`: vêm de
`Config.DefaultPlanParameters` (`research.md` item 12).

### `CameraPlan` (`camera_plan.go`)

Resultado completo da etapa (entidade-chave "Plano de Câmera").

| Campo | Tipo | Observação |
|---|---|---|
| `Parameters` | `PlanParameters` | os efetivamente usados; `Duration` sempre preenchida (a informada ou a calculada) |
| `TimeReference` | `TimeReference` | `TimeReferenceClock` ou `TimeReferenceDistance` |
| `TimeFallbackReason` | `string` | vazio quando a referência é o horário; senão, motivo do recuo para distância (ex.: `"no time data"`) |
| `Frames` | `[]CameraFrame` | exatamente `Parameters.FrameCount()` itens, em ordem |
| `Summary` | `PlanSummary` | calculado por construtor a partir dos quadros |

`NewCameraPlan(parameters, durationMode, timeReference, reason, frames, spans)` monta o
plano e calcula o `Summary`; é o único ponto que garante que o resumo
corresponde aos quadros (SC-009).

### `CameraFrame` (`camera_plan.go`)

Um instante do vídeo (entidade-chave "Quadro do Plano").

| Campo | Tipo | Observação |
|---|---|---|
| `Index` | `int` | de 0 a N−1 |
| `Time` | `time.Duration` | `Index / FrameRate` |
| `Phase` | `Phase` | `PhaseOpening`, `PhaseFollowing` ou `PhaseClosing` |
| `CameraLatitude`, `CameraLongitude` | `float64` | graus decimais; longitude em `[-180, 180)` |
| `CameraAltitude` | `float64` | metros, relativa ao ponto observado (research.md item 1) |
| `Heading` | `float64` | graus no sentido horário a partir do norte, em `[0, 360)` |
| `Tilt` | `float64` | graus abaixo do horizonte, em `[0, 90]` |
| `MarkerLatitude`, `MarkerLongitude` | `float64` | posição do marcador |
| `MarkerDistance` | `float64` | metros percorridos desde o início do trajeto |
| `CameraToMarkerDistance` | `float64` | distância em linha reta câmera→marcador, em metros (usada pelo resumo) |

Todos os valores já saem quantizados (research.md item 9).

### `Phase` (`camera_plan.go`)

Enum de três valores: abertura, acompanhamento, fechamento (entidade-chave
"Fase do Vídeo"). Invariantes verificadas por teste de propriedade: a
sequência de fases é `Opening* Following+ Closing*`; `MarkerDistance` é 0
durante toda a abertura e igual ao comprimento total durante todo o
fechamento.

### `TimeReference` (`camera_plan.go`)

Enum: `TimeReferenceClock`, `TimeReferenceDistance` (FR-011).

### `SmoothedSpan` (`camera_plan.go`)

Trecho suavizado (entidade-chave homônima).

| Campo | Tipo | Observação |
|---|---|---|
| `Start`, `End` | `time.Duration` | instantes de vídeo (do primeiro ao último quadro do trecho, inclusive) |
| `Quantity` | `SmoothedQuantity` | `QuantityHeading`, `QuantityTilt`, `QuantityZoom` ou `QuantityTargetSpeed` — qual limite atuou |

### `PlanSummary` (`camera_plan.go`)

Resumo (entidade-chave homônima), calculado a partir dos quadros.

| Campo | Tipo |
|---|---|
| `Duration` | `time.Duration` |
| `DurationMode` | `DurationMode` (`DurationModeAutomatic` ou `DurationModeExplicit`) |
| `FrameRate` | `float64` |
| `FrameCount` | `int` |
| `MinCameraAltitude`, `MaxCameraAltitude` | `float64` (m) |
| `MinCameraDistance`, `MaxCameraDistance` | `float64` (m; câmera→marcador) |
| `TimeReference` | `TimeReference` |
| `SmoothedSpans` | `[]SmoothedSpan` (vazio ⇒ "nenhum trecho suavizado") |

### `CameraPlanExporter` (`camera_plan.go`, no topo do arquivo)

```text
Export(plan CameraPlan, path string, overwrite bool) error
```

Porta de saída (Princípio II). Erros: `ErrPlanDestinationExists`,
`ErrPlanDestinationInvalid`. `//go:generate` logo abaixo de `package`,
mock em `internal/domain/mockdomain/camera_plan_exporter.go`.

### `CleanedTrack` e `TreatedTrack` (`track.go`, refatoração prévia)

DTOs de saída do novo `TrackService` (`research.md` item 13), declarados no
domínio junto de `Track`:

| Tipo | Campos | Produzido por |
|---|---|---|
| `CleanedTrack` | `Track Track`, `Points []TrackPoint`, `Discarded DiscardStats` | `TrackService.Clean` (parse + limpeza; sem simplificar nem suavizar) |
| `TreatedTrack` | `Track Track`, `Route Route`, `Discarded DiscardStats` | `TrackService.Treat` (`Clean` + simplificação + suavização) |

`TreatedTrack` carrega exatamente os três valores que
`domain.SummarizeTrack(track, route, discarded)` já recebe.

### Reuso (sem mudança)

`TrackPoint`, `Track`, `Route`, `Level`, `BoundingBox`, `Haversine`,
`TotalDistance`, `Duration`, `CleanTrack`, `SummarizeTrack`, as portas
`TrackParser`, `Simplifier`, `Smoother` e os sentinelas `ErrEmptyFile`,
`ErrUnsupportedFormat`, `ErrInsufficientPoints[AfterCleaning]` (FR-022).

## Funções puras de domínio

| Função | Arquivo | Responsabilidade |
|---|---|---|
| `PlanCamera(points, parameters, tuning) (CameraPlan, error)` | `camera_planning.go` | orquestra as funções abaixo; único ponto de entrada da regra |
| `MinimumDuration(points, frameRate, distance, tilt Level, tuning) time.Duration` | `camera_planning.go` | FR-017 (research.md item 8) |
| `DefaultDuration(points, frameRate, distance, tilt Level, tuning) time.Duration` | `camera_planning.go` | FR-003a: curva sublinear com piso/teto, nunca abaixo de `MinimumDuration` (research.md item 8.1); chamada por `PlanCamera` quando `Parameters.Duration` é `nil` |
| `NewLocalPlane(points)` / `(l LocalPlane) Project` / `Unproject` | `camera_projection.go` | projeção azimutal equidistante (item 2) |
| `BuildMarkerTimeline(points, plane, tuning) MarkerTimeline` | `camera_timeline.go` | referência de tempo, compressão de paradas, `s(t)` (item 3) |
| `DesiredHeading`, `UnwrapAngles`, `FollowDistance`, `ComputeCameraPose` | `camera_motion.go` | rumo, distância e pose orbital (itens 1, 4, 6) |
| `LimitRate`, `GaussianSmooth`, `DetectSmoothedSpans` | `camera_motion.go` | limitação de taxa, suavização e trechos suavizados (item 5) |
| `OverviewPose`, `BlendPose` | `camera_framing.go` | abertura/fechamento (item 7) |

As funções são **exportadas** para permitir teste isolado a partir do pacote
`domain_test` (`research.md` item 17).

Todas puras: sem I/O, sem relógio, sem aleatoriedade, cada uma com comentário de documentação em inglês.

## Erros sentinela novos (`errors.go`)

`ErrInvalidDuration`, `ErrInvalidFrameRate`, `ErrDurationTooShort`,
`ErrTrackTooShort`, `ErrTrackTooLarge`, `ErrPlanDestinationExists`,
`ErrPlanDestinationInvalid`. Mensagens com dados variáveis usam
`fmt.Errorf("%w: ...", sentinela)`, para que a CLI ainda reconheça o
sentinela com `errors.Is` e o usuário veja, por exemplo, a duração mínima
(SC-008).

## Serviços (`internal/application`)

```text
TrackService (interface; NOVO — substitui InspectTrackService)
  Clean(reader io.Reader) (domain.CleanedTrack, error)
  Treat(reader io.Reader, simplification, smoothing domain.Level) (domain.TreatedTrack, error)
  Inspect(reader io.Reader, simplification, smoothing domain.Level) (domain.TrackSummary, error)

CameraPlanService (interface; NOVO)
  Generate(reader io.Reader, parameters domain.PlanParameters) (domain.CameraPlan, error)
  Export(plan domain.CameraPlan, path string, overwrite bool) error
```

- `trackService` depende de `TrackParser`, `Simplifier`, `Smoother` e dos
  limiares `minPoints`/`maxPlausibleSpeedKmh`. `Clean`: `Parse` →
  `CleanTrack`. `Treat`: `Clean` → `Simplify` → `Smooth`. `Inspect`:
  `Treat` → `domain.SummarizeTrack`.
- `geoDataService.CheckCoverage` passa a chamar `TrackService.Clean` e
  deixa de receber `TrackParser` e os limiares (o construtor encolhe).
- `cameraPlanService` depende de `TrackService`, `CameraPlanExporter`, do
  nível padrão de tratamento e de `domain.CameraTuning`. `Generate`:
  `parameters.Validate()` → `TrackService.Treat` → `domain.PlanCamera`.
  `Export`: delega à porta.

## Transições e ciclo de vida

Não há estado persistente nem transições: o plano é um valor derivado,
recalculável e imutável depois de construído. A única "mutação" externa é
a criação do arquivo exportado.

## Builders de teste (`builddomain`, Princípio X)

`CameraFrameBuilder`, `CameraPlanBuilder` (defaults sensatos: 3 quadros,
distância média) e `PlanParametersBuilder`. Os cenários geométricos usam
o `TrackBuilder` já existente com trajetos sintéticos (linha reta, círculo,
retorno pelo mesmo caminho, curva em U, cruzamento do antimeridiano, alta
latitude).
