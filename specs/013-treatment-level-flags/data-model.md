# Modelo de Dados: Níveis de tratamento em `plan` e `fly`

Só o que muda ou se acrescenta em relação ao modelo já existente
(`specs/003-camera-path-planning/data-model.md`). Racional de cada decisão
em `research.md`.

## `domain.PlanParameters` (estendido) — `internal/domain/camera_plan_parameters.go`

Dois campos novos, ao lado de `Distance`/`Tilt`:

```go
type PlanParameters struct {
    Duration  *time.Duration
    FrameRate float64
    Distance  Level
    Tilt      Level

    // Simplification and Smoothing choose how the track is treated before
    // planning the camera — the same two choices "inspect" already lets the
    // user make, now effective for the camera plan too (013-treatment-
    // level-flags). Both are resolved to a concrete Level by the caller
    // before Generate is called; there is no "use the configured default"
    // behavior left inside the service.
    Simplification Level
    Smoothing      Level

    Aspect AspectRatio
}
```

Nenhuma mudança em `Validate()`: os dois campos são um enum fechado de três
valores já resolvido pela CLI antes de chegar ao domínio (como `Distance`/
`Tilt` já são) — não há faixa numérica para checar.

## `CameraPlan.ID()` (estendido) — `internal/domain/camera_plan.go`

O hash de identidade do plano passa a incluir os dois campos novos, na
mesma posição que `Distance`/`Tilt` já ocupam:

```go
write(int64(c.Parameters.Distance))
write(int64(c.Parameters.Tilt))
write(int64(c.Parameters.Simplification)) // novo
write(int64(c.Parameters.Smoothing))      // novo
```

Nenhuma outra mudança em `ID()`: a versão do hash (`"sobrevoo-plan-v1"`) não
muda — acrescentar um campo ao que já é hasheado não é a mesma coisa que
mudar o significado de um campo existente, e o comentário de `ID()` já
descreve o hash como cobrindo "os parâmetros efetivos", que os dois níveis
sempre foram, só que não gravados.

## `CameraPlanService` (reduzido) — `internal/application/camera_plan_service.go`

```go
func NewCameraPlanService(
    trackService TrackService,
    exporter domain.CameraPlanExporter,
    reader domain.CameraPlanReader,
    tuning domain.CameraTuning,
) CameraPlanService
```

O parâmetro `defaultLevel domain.Level` deixa de existir — o serviço não
resolve mais nenhum "padrão" por conta própria. `Generate` passa a chamar:

```go
treated, err := s.trackService.Treat(reader, parameters.Simplification, parameters.Smoothing)
```

no lugar de `s.trackService.Treat(reader, s.defaultLevel, s.defaultLevel)`.
A assinatura pública de `Generate`/`Export`/`Load` não muda — só o que
`Generate` faz internamente, e como o serviço é construído.

## Arquivo de plano exportado (estendido) — `internal/infra/outbound/jsonfile`

Dois campos de texto novos em `parameters`, lidos/escritos como
`distance`/`tilt` já são:

```json
{
  "format_version": 2,
  "parameters": {
    "duration_s": 42.0,
    "frame_rate": 30,
    "distance": "medium",
    "tilt": "medium",
    "simplification": "high",
    "smoothing": "low",
    "aspect_ratio": "9:16"
  }
}
```

| Campo | Tipo | Semântica |
|---|---|---|
| `parameters.simplification` | texto | `low`, `medium` ou `high` — o nível de simplificação efetivamente usado para tratar o trajeto antes de planejar a câmera. |
| `parameters.smoothing` | texto | `low`, `medium` ou `high` — idem, para a suavização. |

Ambos opcionais na leitura: ausentes (plano de antes desta etapa), lêem como
`medium` — o mesmo valor que `parseLevel` já devolve para qualquer texto
desconhecido ou vazio (`camera_plan_reader.go`, função já existente,
reaproveitada sem mudança). `format_version` continua `2`: ver
`contracts/plan-file-addendum.md`.

## Sem sentinelas de erro novas, sem exit codes novos

Um valor de `--simplification`/`--smoothing` fora de `low`/`medium`/`high`,
em `plan` ou em `fly`, continua sendo um erro de uso da CLI (código `2`) —
não um erro sentinela do domínio. Ver `contracts/treatment-level-flags.md`.

## `FlightRequest` — `internal/domain/flight.go`

Nenhum campo novo: `Parameters domain.PlanParameters` já existe e já é
exatamente o que a CLI preenche com os dois níveis resolvidos antes de
montar o `FlightRequest` (como já faz para `Distance`/`Tilt`). Só o
comentário do campo (hoje "duration, frame rate, distance, tilt and aspect
ratio") passa a citar os dois níveis também.

## `FlightService.reusePlan` — `internal/application/flight_service.go`

Nenhuma mudança de código: a condição já existente,
`existing.ID() == plan.ID()`, passa a comparar os dois níveis novos porque
`ID()` (acima) passou a incluí-los — exatamente o mesmo efeito "de graça"
que a etapa 8 (aparência) e a etapa 9 (sobreposição) já tiveram sobre
`NewFrameSetID`/o reaproveitamento de `render all`/`fly --keep`.

## `cmd/sobrevoo/config_mapping.domainPlanParameters` (estendida)

```go
func domainPlanParameters(defaults config.PlanDefaults, defaultLevel config.Level) domain.PlanParameters {
    return domain.PlanParameters{
        FrameRate:      defaults.FrameRate,
        Distance:       domainLevel(defaults.Distance),
        Tilt:           domainLevel(defaults.Tilt),
        Simplification: domainLevel(defaultLevel),
        Smoothing:      domainLevel(defaultLevel),
        Aspect:         domain.AspectRatio{Width: defaults.AspectWidth, Height: defaults.AspectHeight},
    }
}
```

Os dois call sites em `main.go` (para `plan` e para `fly`) passam
`cfg.DefaultLevel` — o mesmo valor que já é passado para `inspect` via
`domainLevel(cfg.DefaultLevel)`. Nenhum campo novo em `config.go`.

## Flags novas compartilhadas: `--simplification`, `--smoothing`

Sem tipo de domínio próprio de CLI, e sem parser novo: `parsePlanParameters`
(`internal/infra/inbound/cli/plan.go`, já existente, já compartilhada por
`plan.go` e `fly.go`) ganha dois parâmetros de flag e usa `parseLevel`/
`levelName` — já genéricas no pacote `cli` — exatamente como já faz para
`--distance`/`--tilt`. Ver `contracts/treatment-level-flags.md`.

## `builddomain.PlanParametersBuilder` (estendido) — `internal/domain/builddomain/plan_parameters_builder.go`

```go
func NewPlanParametersBuilder() *PlanParametersBuilder {
    return &PlanParametersBuilder{
        parameters: domain.PlanParameters{
            Duration:       new(60 * time.Second),
            FrameRate:      30,
            Distance:       domain.LevelMedium,
            Tilt:           domain.LevelMedium,
            Simplification: domain.LevelMedium, // novo
            Smoothing:      domain.LevelMedium, // novo
            Aspect:         domain.LandscapeAspectRatio,
        },
    }
}

func (b *PlanParametersBuilder) WithSimplification(level domain.Level) *PlanParametersBuilder
func (b *PlanParametersBuilder) WithSmoothing(level domain.Level) *PlanParametersBuilder
```

Mesmo padrão que `WithDistance`/`WithTilt` já seguem.
