# Modelo de Dados: Leitura e Tratamento de Trajeto GPS

**Feature**: `001-gps-track-processing` | **Data**: 2026-09-13

Todos os tipos abaixo, salvo indicação contrária, vivem em `internal/domain` e
não importam nenhum pacote de infraestrutura, em conformidade com o Princípio I
da constituição.

## TrackPoint

Um registro individual de um trajeto.

| Campo | Tipo | Obrigatório | Descrição |
|---|---|---|---|
| `Latitude` | `float64` | sim | Graus decimais, intervalo válido `[-90, 90]`. |
| `Longitude` | `float64` | sim | Graus decimais, intervalo válido `[-180, 180]`. |
| `Elevation` | `*float64` | não | Metros acima do nível do mar. `nil` quando o arquivo não traz altitude para este ponto. |
| `Time` | `*time.Time` | não | Instante de captura do ponto. `nil` quando o arquivo não traz tempo para este ponto. |

**Regras de validação** (aplicadas pelas funções de tratamento do domínio, não
no construtor):
- Coordenada impossível: `Latitude` fora de `[-90, 90]` ou `Longitude` fora de
  `[-180, 180]` (FR-008).
- Duplicado consecutivo: `Latitude` e `Longitude` idênticas ao ponto anterior
  na sequência já ordenada (FR-009).
- Salto implausível: velocidade implícita entre este ponto e o anterior (
  distância Haversine dividida pelo intervalo de tempo) acima do limite
  interno de 130 km/h — aplicável apenas quando ambos os pontos têm `Time`
  (FR-010).

## Track

O trajeto bruto, exatamente como veio do arquivo de origem.

| Campo | Tipo | Descrição |
|---|---|---|
| `Format` | `Format` (enum: `FormatGPX`) | Formato identificado pelo `TrackParser` (FR-002, FR-003). Único valor nesta etapa; o tipo enum fica pronto para novos formatos futuros sem alterar as entidades. |
| `Points` | `[]TrackPoint` | Pontos na ordem em que apareceram no arquivo (sem tratamento). |

Produzido exclusivamente pela porta `TrackParser` (ver Portas abaixo).

## Route

O trajeto após todas as etapas de tratamento (reordenação, descarte,
simplificação, suavização) — usado para calcular as estatísticas finais do
resumo.

| Campo | Tipo | Descrição |
|---|---|---|
| `Points` | `[]TrackPoint` | Pontos tratados, na ordem cronológica final. |

## BoundingBox

A área geográfica ocupada por um conjunto de pontos (FR-022, FR-023).

| Campo | Tipo | Descrição |
|---|---|---|
| `MinLatitude` | `float64` | Menor latitude do trajeto. |
| `MaxLatitude` | `float64` | Maior latitude do trajeto. |
| `MinLongitude` | `float64` | Menor longitude do trajeto, já normalizada para `[-180, 180]`. |
| `MaxLongitude` | `float64` | Maior longitude do trajeto, já normalizada para `[-180, 180]`. |
| `CrossesAntimeridian` | `bool` | `true` quando o trajeto cruza a linha de 180° — nesse caso, a área ocupada é a que vai de `MaxLongitude` até `MinLongitude` "pelo lado de fora" (passando por ±180°), não a forma direta entre os dois valores. |

Calculada pela função pura `ComputeBoundingBox` (ver item 7 de `research.md`).

## Level

Nível de intensidade para simplificação ou suavização (FR-014, FR-015,
FR-016). Um único tipo reaproveitado para as duas configurações.

| Valor | Descrição |
|---|---|
| `LevelLow` | Predefinição `low`. |
| `LevelMedium` | Predefinição `medium` — usada como padrão (FR-016) quando o usuário não informa um nível. |
| `LevelHigh` | Predefinição `high`. |

## DiscardStats

Contagem de pontos descartados durante o tratamento, discriminada por motivo
(FR-011).

| Campo | Tipo | Descrição |
|---|---|---|
| `ImpossibleCoordinates` | `int` | Pontos descartados por coordenada impossível. |
| `ConsecutiveDuplicates` | `int` | Pontos descartados por duplicação consecutiva. |
| `ImplausibleJumps` | `int` | Pontos descartados por salto implausível. |

## Portas (interfaces declaradas no domínio)

Não há um arquivo genérico de portas. Cada interface fica no arquivo mais
específico possível: junto da entidade que ela manipula, ou em um arquivo
próprio nomeado pelo conceito, quando não pertence a uma única entidade
(convenção espelhada de `waliqueiroz/mystery-gifter-api`).

### TrackParser (declarada em `track.go`, junto da entidade `Track`)

```go
type TrackParser interface {
    Parse(r io.Reader) (Track, error)
}
```

Implementação em `internal/infra/outbound/trackparser`: um único adapter que
valida se o conteúdo é um GPX reconhecível e delega o parsing a
`tkrajina/gpxgo` (FR-002, FR-003, FR-004). Um segundo formato, se adicionado
no futuro, ganha seu próprio adapter atrás desta mesma porta, sem alterar o
domínio.

### Simplifier (declarada em `simplification.go`, arquivo próprio — não pertence a uma única entidade)

```go
type Simplifier interface {
    Simplify(points []TrackPoint, level Level) []TrackPoint
}
```

Implementação em `internal/infra/outbound/simplifier`, aplicando o algoritmo
de Douglas-Peucker (FR-012, FR-014).

### Smoother (declarada em `smoothing.go`, arquivo próprio — não pertence a uma única entidade)

```go
type Smoother interface {
    Smooth(points []TrackPoint, level Level) []TrackPoint
}
```

Implementação em `internal/infra/outbound/smoother`, aplicando o algoritmo de
Catmull-Rom (FR-013, FR-015).

### Mocks

Gerados com `go.uber.org/mock/mockgen` via `//go:generate` posicionado
diretamente acima de cada interface (não em um arquivo central), com saída em
`internal/domain/mock_domain/` — um arquivo por porta: `track_parser.go`,
`simplifier.go`, `smoother.go`.

## Erros sentinela do domínio

Conforme o Princípio VII da constituição — todos declarados em
`internal/domain`, traduzidos pela CLI em código de saída de processo.

| Erro sentinela | Quando ocorre |
|---|---|
| `ErrEmptyFile` | O arquivo de entrada está vazio (FR-005). |
| `ErrUnsupportedFormat` | O conteúdo não corresponde ao formato GPX (FR-004). |
| `ErrInsufficientPoints` | O arquivo já chega com menos de 2 pontos válidos (FR-006). |
| `ErrInsufficientPointsAfterCleaning` | Restam menos de 2 pontos depois do descarte de pontos problemáticos (FR-006, cenário de aceitação 4 da História de Usuário 2). |

## Funções puras do domínio

Todas operam sobre `[]TrackPoint` ou `Track`/`Route`, sem efeitos colaterais e
sem dependência externa (Princípio I):

| Função | Responsabilidade | Requisito |
|---|---|---|
| `ReorderByTime` | Reordena pontos por `Time` quando todos os pontos possuem tempo; caso contrário, devolve os pontos inalterados. | FR-027 |
| `DiscardImpossibleCoordinates` | Remove pontos com coordenada fora do intervalo válido. | FR-008 |
| `DiscardConsecutiveDuplicates` | Remove repetições consecutivas de coordenada. | FR-009 |
| `DiscardImplausibleJumps` | Remove pontos cuja velocidade implícita em relação ao anterior excede o limite interno. | FR-010 |
| `Haversine` | Distância em metros entre dois pontos. | FR-017 |
| `TotalDistance` | Soma das distâncias Haversine entre pontos consecutivos de uma rota. | FR-017 |
| `ElevationGain` | Soma dos ganhos positivos de altitude; segundo retorno indica se havia dado de altitude suficiente para calcular. | FR-018, FR-019 |
| `Duration` | Diferença entre o tempo do primeiro e do último ponto; segundo retorno indica se havia dado de tempo. | FR-020, FR-021 |
| `ComputeBoundingBox` | Área geográfica ocupada, com tratamento de antimeridiano. | FR-022, FR-023, FR-024 |

## DTOs da camada de aplicação

Vivem em `internal/application`, não em `internal/domain` — são a forma de
entrada/saída do caso de uso, não conceitos de negócio por si só.

### InspectTrackInput

| Campo | Tipo | Descrição |
|---|---|---|
| `Reader` | `io.Reader` | Conteúdo do arquivo de rastreamento a processar. |
| `SimplificationLevel` | `domain.Level` | Nível escolhido pelo usuário (já convertido pelo adapter de entrada). |
| `SmoothingLevel` | `domain.Level` | Nível escolhido pelo usuário (já convertido pelo adapter de entrada). |

### InspectTrackOutput

| Campo | Tipo | Descrição |
|---|---|---|
| `Format` | `domain.Format` | Formato identificado. |
| `PointCountOriginal` | `int` | Quantidade de pontos antes do tratamento. |
| `PointCountTreated` | `int` | Quantidade de pontos após o tratamento. |
| `TotalDistanceMeters` | `float64` | Distância total do trajeto tratado. |
| `ElevationGainMeters` | `*float64` | `nil` quando não havia dado de altitude (FR-019). |
| `Duration` | `*time.Duration` | `nil` quando não havia dado de tempo (FR-021). |
| `BoundingBox` | `domain.BoundingBox` | Área geográfica ocupada pelo trajeto tratado. |
| `Discarded` | `domain.DiscardStats` | Contagem de pontos descartados por motivo. |

Esta estrutura é dado puro — nenhum campo é `io.Writer`, nenhuma formatação de
texto acontece aqui; a apresentação é responsabilidade exclusiva do adapter de
entrada (CLI).
