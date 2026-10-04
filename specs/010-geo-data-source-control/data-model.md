# Modelo de Dados: Controle do Registro de Dados Geográficos

Só o que muda ou se acrescenta em relação ao modelo já existente
(`specs/002-geo-data-registry/data-model.md`,
`specs/004-geo-data-slice/data-model.md`). Racional de cada decisão em
`research.md`.

## `domain.SourceSelection` (novo tipo) — `internal/domain/source_selection.go`

```go
type SourceSelection struct {
    BaseMapName   *string // nil: seleção automática (FR-009)
    ElevationName *string // nil: seleção automática (FR-009)
}

func (s SourceSelection) Resolve(baseMaps, elevations []GeoDataSource) (resolvedBaseMaps, resolvedElevations []GeoDataSource, err error)
```

`Resolve` não faz I/O — opera só sobre as duas listas de candidatos que o
chamador já buscou e já filtrou pelos ainda disponíveis em disco
(`partitionAvailableSources`, já existente). Para cada campo não nulo:

- procura o nome na lista do próprio tipo; achado, a lista desse tipo passa
  a ser só aquele elemento (FR-010: nenhuma mistura é possível depois disso,
  estruturalmente);
- não achado na lista do próprio tipo, procura na do outro tipo; achado ali,
  `ErrDataSourceTypeMismatch` (novo — o nome existe, mas é do tipo errado);
- não achado em nenhuma das duas, `ErrDataSourceNotRegistered` (já
  existente, reaproveitado com uma mensagem que cita o nome e o tipo
  pedido).

Um campo nulo deixa a lista do seu tipo inalterada (FR-008, FR-009) — a
seleção automática (`Route.Coverage`/`SelectSource`) continua decidindo
entre todos os candidatos desse tipo, exatamente como antes desta etapa.

## `GeoDataRepository` (porta estendida) — `internal/domain/geo_data_source.go`

Um método novo:

```go
// Clear removes every registered source at once, keeping no entry — even
// ones whose file still exists on disk. It never touches any data file
// (FR-002).
Clear() error
```

## `GeoDataService` (estendido) — `internal/application/geo_data_service.go`

Um método novo:

```go
// Clear removes every registered source at once (FR-001, FR-002). Without
// confirmed, nothing is removed: it reports, via ErrRegistryClearNotConfirmed,
// how many entries would be removed (FR-003).
Clear(confirmed bool) (removedCount int, err error)
```

`CheckCoverage` ganha um parâmetro:

```go
CheckCoverage(reader io.Reader, selection domain.SourceSelection) (domain.CoverageReport, error)
```

Internamente, resolve `selection` (contra as listas já particionadas por
`partitionAvailableSources`) **antes** de tratar o trajeto (`trackService
.Clean`) — recusa cedo (FR-006), sem gastar o tratamento do trajeto quando o
nome pedido já está errado.

## `GeoSliceService` (estendido) — `internal/application/geo_slice_service.go`

`Generate` ganha um parâmetro:

```go
Generate(plan domain.CameraPlan, selection domain.SourceSelection) (domain.GeoSlice, error)
```

Resolve `selection` logo depois de `partitionAvailableSources`, antes de
`area.Regions`/`route.Coverage` e de qualquer estimativa de tamanho ou
leitura de conteúdo — antes de qualquer outro trabalho (FR-006, FR-007).

## `GeoSlice` (estendido) — `internal/domain/geo_slice.go`

Um método novo, ao lado de `EnsureMatches`/`EnsureCovers` (revisado em
2026-10-04 — `research.md` item 3, "Revisão"; a primeira versão,
`EnsureUsesSelection`, não verificava a seleção automática):

```go
// EnsureUsesSources refuses, with ErrSliceUsesDifferentSource, a slice whose
// recorded provenance (Summary.Sources) is not exactly sources — the same
// registered files, of the same types, no more and no fewer (FR-011).
func (g GeoSlice) EnsureUsesSources(sources []GeoDataSource) error
```

Compara a procedência gravada com o conjunto dado por tipo, nome e caminho do
arquivo, sem depender da ordem; qualquer diferença — uma fonte a mais, a
menos ou trocada — é `ErrSliceUsesDifferentSource`. O conjunto dado vem de
`GeoSliceService.Sources(plan, selection)`: as fontes de que um recorte
gerado agora seria tirado, calculadas só pelos metadados do registro, pelo
mesmo caminho de `Generate` (inclusive `SliceRegions.Sources`, que lista as
fontes das regiões, sem as de valor zero de uma região descoberta).

Nenhum campo novo em `GeoSlice`/`SliceSummary`/`SliceSourceUse`: a
procedência que a verificação usa (`Summary.Sources`) já existe desde a
etapa 4 e já é gravada no arquivo exportado (`sources[]`,
`specs/004-geo-data-slice/contracts/slice-file.md`) — nenhuma mudança de
formato de arquivo nesta etapa.

## `FlightRequest` (estendido) — `internal/domain/flight.go`

Ganha um campo `Selection domain.SourceSelection`, usado em
`GeoSliceService.Generate` e em `GeoSliceService.Sources` dentro de
`FlightService.reuseSlice`; não afeta o plano (`CameraPlanService.Generate`
não lê dados geográficos, Clarifications da spec).

## `FlightService.reuseSlice` (estendido) — `internal/application/flight_service.go`

A condição de reaproveitamento passa a ser:

```go
if existing, err := s.geoSliceService.Load(slicePath); err == nil && existing.EnsureMatches(plan) == nil {
    if sources, err := s.geoSliceService.Sources(plan, selection); err == nil && existing.EnsureUsesSources(sources) == nil {
        // reused
    }
}
```

no lugar de só `existing.EnsureMatches(plan) == nil` (FR-011, História de
Usuário 4). Um nome pedido que não resolve faz `Sources` falhar; o recorte
então não é reaproveitado, e `Generate` recusa o nome como sempre.

## Sentinelas de erro novas — `internal/domain/errors.go`

| Nome | Quando |
|---|---|
| `ErrDataSourceTypeMismatch` | Um nome pedido explicitamente (`--base-map`/`--elevation`) existe no registro, mas é do outro tipo (FR-006). |
| `ErrRegistryClearNotConfirmed` | `geodata clear` sem `--confirm` (FR-003); a mensagem cita quantas entradas seriam removidas. |
| `ErrSliceUsesDifferentSource` | Um recorte guardado por `fly --keep` cuja procedência não usa, para um tipo pedido agora, exatamente a fonte pedida (FR-011) — só comparado dentro de `FlightService.reuseSlice`, nunca devolvido diretamente a um usuário (a mesma forma como `EnsureMatches` já é usado ali: só o `nil`/não-`nil` importa). |

`ErrDataSourceNotRegistered` (já existente, código de saída `9`) é
reaproveitado para um nome pedido que não existe no registro em nenhum tipo
(FR-006) — mesma condição de negócio que `geodata remove` já usa esse
sentinela para, com uma mensagem adaptada ao contexto (cita a flag e o
nome).

## Comando novo: `geodata clear`

Sem entidade de domínio própria — só um caso de uso a mais em
`GeoDataService`. Ver `contracts/cli.md`.

## Flags novas compartilhadas: `--base-map`, `--elevation`

Sem tipo de domínio próprio de CLI: `parseSourceSelection` (novo,
`internal/infra/inbound/cli/source_selection.go`) monta um
`domain.SourceSelection` a partir de duas flags string, usando
`cmd.Flags().Changed(...)` para decidir `nil` vs. um ponteiro — o mesmo
padrão de `parseAppearance`/`parseOverlay`. Não valida nada (não pode: só
`SourceSelection.Resolve`, mais tarde, tem acesso ao registro) — por isso não
devolve `error`. Ver `contracts/source-selection-flags.md`.
