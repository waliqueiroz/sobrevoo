# Modelo de Dados: Voo em um Único Comando

**Feature**: `007-full-flight-pipeline` | **Data**: 2026-09-27

Tipos de `internal/domain` que esta etapa acrescenta, e a extensão pontual de
`VideoService` (`internal/application`). Nenhum tipo de negócio é criado: os
nomes abaixo são só o vocabulário do encadeamento (Requisição, Etapa,
Progresso, Resumo) e a porta mínima que a execução sem `--keep` precisa
(`Workspace`). Regra de negócio continua inteiramente nos tipos já existentes
das etapas 1 a 6 — `CameraPlan`, `GeoSlice`, `FrameDirectory`, `VideoSummary`,
etc. — que esta etapa só chama. Nenhum tipo aqui importa `os`, `os/exec` ou
`internal/infra`.

## Tipos novos (`internal/domain/flight.go`)

### `FlightRequest`

```go
// FlightRequest is everything a single-command run needs beyond the track
// reader: the same parameters plan/render all/video already accept, the
// video destination, and where — if anywhere — to keep the intermediates.
type FlightRequest struct {
    Parameters PlanParameters // duration, frame rate, distance, tilt, aspect
    Resolution Resolution     // of the frames
    Quality    VideoQuality   // of the video
    Output     string         // the video file to write
    Keep       string         // "" means: intermediates in a temporary directory of the run's own, removed at the end
    Overwrite  bool           // for the video destination and for a foreign/stale intermediate under Keep
}
```

Sem validação própria: `Parameters.Validate()`, `Resolution` (checada por
`NewResolution`/`ParseResolution`) e `Quality` já são validados pelos mesmos
caminhos que `plan`/`render all`/`video` já usam, chamados de dentro dos
serviços que `FlightService` invoca. `FlightRequest` é só o envelope.

### `FlightStage`

```go
// FlightStage is one of the five stages a single-command run goes through,
// in order.
type FlightStage int

const (
    StageTrackProcessing FlightStage = iota
    StageCameraPlanning
    StageGeoDataSlicing
    StageFrameRendering
    StageVideoEncoding
)

// String is the stage's label, as FlightProgress reports it: "treating the
// track", "planning the camera", "slicing the geo data", "drawing the
// frames", "encoding the video".
func (s FlightStage) String() string
```

Cinco valores, na ordem fixa em que `FlightService.Fly` sempre os percorre
(FR-004). Enum simples, sem validação (não é entrada do usuário).

### `FlightProgress`

```go
// FlightProgress is reported as a run moves through its five stages: Stage is
// always set; Render and Video carry the same progress the frame-rendering
// and video-encoding stages already report on their own, forwarded as-is —
// nil outside of their stage.
type FlightProgress struct {
    Stage  FlightStage
    Render *RenderProgress // non-nil only while Stage == StageFrameRendering
    Video  *VideoProgress  // non-nil only while Stage == StageVideoEncoding
}
```

Não é uma barra de progresso agregada (item 7 de `research.md`): a CLI decide
como mostrar cada etapa, reaproveitando as mesmas linhas que `render
all`/`video` já escrevem quando `Render`/`Video` vêm preenchidos.

### `FlightSummary`

```go
// FlightSummary is what a run is left with — what completed, what was
// reused, and the summaries of the two stages that already have one of their
// own.
type FlightSummary struct {
    Completed []FlightStage // stages that finished, in order; a partial run ends here on failure or interruption

    PlanReused  bool // the plan in Keep already matched (CameraPlan.ID()); Generate was not called
    SliceReused bool // the slice in Keep already matched (GeoSlice.EnsureMatches); Generate was not called

    Render RenderSummary // from FrameService.DrawFrames — zero value if frame rendering never started
    Video  VideoSummary  // from VideoService.Assemble — zero value if video encoding never started

    Elapsed     time.Duration // the whole run, not just Video.Elapsed
    Interrupted bool
}
```

Nenhum campo duplica o que `RenderSummary`/`VideoSummary` já trazem (quadros
desenhados/mantidos/removidos, buracos, tamanho, codificador, ...) — a CLI
reaproveita `formatFramesSummary`/`formatVideoSummary` para eles e só
acrescenta o que é próprio do encadeamento: quais etapas rodaram, o que foi
reaproveitado, o tempo total.

## Porta nova (`internal/domain/workspace.go`)

```go
//go:generate go run go.uber.org/mock/mockgen -destination mockdomain/workspace.go -package mockdomain . Workspace

// Workspace provides the directory a run without Keep draws its frames into:
// FrameService and VideoService need a real directory on disk (the encoder
// reads the frames from it), but the plan and the slice, passed between
// services as values, never do.
type Workspace interface {
    // NewTemporary creates a fresh, empty directory for this run's exclusive
    // use, and the function that removes it, and everything inside it,
    // afterward. remove MUST be called exactly once, however the run ends.
    NewTemporary() (path string, remove func() error, err error)
}
```

Única porta nova da etapa: o resto da E/S (ler o trajeto, gravar/ler
`plan.json`/`slice.zip`, gravar/ler os quadros, codificar e publicar o vídeo)
já passa pelas portas de `CameraPlanService`, `GeoSliceService`,
`FrameService` e `VideoService`.

## Extensão de `VideoService` (`internal/application/video_service.go`)

```go
type VideoService interface {
    // ... Assemble como hoje ...

    // CheckEncoder probes the video encoder the same way Assemble does before
    // encoding, and returns what it found.
    CheckEncoder(ctx context.Context) (domain.EncoderInfo, error)

    // CheckDestination checks the video destination the same way Assemble
    // does before encoding: refuses an existing file unless overwrite.
    CheckDestination(output string, overwrite bool) error
}
```

`Assemble` passa a chamar os dois no lugar do corpo que hoje fala direto com
`s.encoder.Probe`/`s.exporter.Check` — mesmo comportamento, mesma ordem, sem
nenhuma mudança observável em `video` (item 2 de `research.md`).

## Erro sentinela novo (`internal/domain/errors.go`)

```go
// ErrFlightInterrupted is returned by FlightService.Fly when the run is
// interrupted, in place of whichever stage-specific interruption error
// (ErrRenderInterrupted, ErrVideoInterrupted) the stage that was running
// would have returned on its own (FR-011 — the single command always exits
// with its own code for an interruption, never the stage's).
var ErrFlightInterrupted = errors.New("flight interrupted")
```

## `FlightService` (`internal/application/flight_service.go`)

```go
type FlightService interface {
    // Fly runs the whole flight: it treats reader's track, plans the camera,
    // gathers the geo data slice, draws the frames and encodes the video
    // described by request, reusing whatever of the plan/slice/frames under
    // request.Keep is still valid for the same track and the same values.
    // progress is called as the run enters each stage and as the frame
    // rendering and video encoding stages report their own progress (may be
    // nil). The summary says what was done even when the run stopped early,
    // with ErrFlightInterrupted or another error.
    Fly(ctx context.Context, reader io.Reader, request domain.FlightRequest, progress func(domain.FlightProgress)) (domain.FlightSummary, error)
}
```

Construtor: `NewFlightService(cameraPlanService CameraPlanService,
geoSliceService GeoSliceService, frameService FrameService, videoService
VideoService, workspace domain.Workspace) FlightService`.

## Fluxo de `Fly` (orquestração; nenhuma regra de negócio própria)

1. `videoService.CheckDestination(request.Output, request.Overwrite)`, depois `videoService.CheckEncoder(ctx)` — antes de tocar no trajeto (FR-005, FR-007), na mesma ordem que `Assemble` já confere as duas (destino antes do codificador).
2. `cameraPlanService.Generate(reader, request.Parameters)` — trata o trajeto e planeja a câmera (etapas 1-2, uma chamada; `Completed` ganha as duas).
3. Se `request.Keep != ""`: tenta reaproveitar `<Keep>/plan.json` (item 4 de `research.md`); senão `cameraPlanService.Export(...)`.
4. Se `request.Keep != ""`: tenta reaproveitar `<Keep>/slice.zip`; senão `geoSliceService.Generate(plan)` (a cobertura é conferida aqui, item 3) e, se `Keep`, `geoSliceService.Export(...)`.
5. Resolve o diretório de quadros: `<Keep>/frames` ou `workspace.NewTemporary()` (com a remoção adiada).
6. `frameService.DrawFrames(ctx, plan, slice, ..., progress)` — mesma chamada de `render all` (item 5).
7. `videoService.Assemble(ctx, plan, ..., progress)` — mesma chamada de `video`.
8. Remove o diretório temporário, se um foi criado.

Uma falha em qualquer passo devolve o erro tal como a etapa o deu (FR-008),
exceto quando é uma interrupção, sempre traduzida para `ErrFlightInterrupted`
(item 8). `FlightSummary` é preenchido incrementalmente, então um erro no meio
ainda deixa em `Completed`/`Render`/`Video` o que já havia acontecido.
