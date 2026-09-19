# Modelo de Dados: Registro de Dados Geográficos Locais

**Feature**: `002-geo-data-registry` | **Data**: 2026-09-13

Todos os tipos abaixo, salvo indicação contrária, vivem em `internal/domain`
e não importam nenhum pacote de infraestrutura, em conformidade com o
Princípio I da constituição. `BoundingBox` já existe (etapa 1); os demais
tipos desta seção são novos.

## GeoDataSource

Um registro de dado geográfico local declarado pelo usuário (Entidade-Chave
"Registro de Dado Geográfico" do `spec.md`).

| Campo | Tipo | Descrição |
|---|---|---|
| `Name` | `string` | Identificador escolhido pelo usuário, único entre todos os registros (FR-001, FR-007). |
| `Path` | `string` | Caminho do arquivo no sistema de arquivos, exatamente como informado no registro. |
| `Type` | `DataType` (enum) | `DataTypeBaseMap` ou `DataTypeElevation`, determinado automaticamente a partir do conteúdo do arquivo (FR-002). |
| `Format` | `DataFormat` (enum) | `DataFormatMBTiles` ou `DataFormatGeoTIFF` — o formato de arquivo concreto identificado. Guardado separadamente de `Type` pelo mesmo motivo que `Track.Format` existe desde a etapa 1: um segundo formato para o mesmo tipo (ex.: um segundo formato de mapa base) pode ser adicionado no futuro sem alterar esta entidade. |
| `BoundingBox` | `BoundingBox` | Área geográfica coberta, determinada automaticamente a partir do conteúdo do arquivo (FR-003). |
| `RegisteredAt` | `time.Time` | Instante em que o registro foi criado, obtido de `time.Now()` diretamente por `NewGeoDataSource` — sem uma porta dedicada (`research.md` item 10.1: `time` é biblioteca padrão, não uma dependência externa no sentido do Princípio II, e o campo nunca é exibido ao usuário). Usado apenas para o desempate determinístico entre fontes sobrepostas (FR-016, ver `research.md` item 10). |

Note-se que a entidade **não** guarda se o arquivo ainda existe: essa é uma
informação dinâmica, recalculada a cada `list`/`check` via a porta
`FileChecker` (FR-010, FR-017), nunca persistida junto do registro.

```go
func NewGeoDataSource(name, path string, inspected InspectedGeoData) GeoDataSource
```

Constrói um `GeoDataSource` a partir do que um `GeoDataInspector`
descobriu, já com `RegisteredAt: time.Now()` — a mesma forma que
`domain.NewGroup` monta `CreatedAt`/`UpdatedAt` em
`waliqueiroz/mystery-gifter-api` (`research.md` item 14). A regra de
negócio "como um `GeoDataSource` nasce" vive aqui, não em
`GeoDataService.Register`, que só orquestra: verifica duplicidade de nome,
chama `GeoDataInspector.Inspect`, chama este construtor, salva.

## GeoDataSummary

Um registro junto da sua disponibilidade de arquivo em tempo real, como
reportado por `GeoDataService.List` (FR-009, FR-010).

| Campo | Tipo | Descrição |
|---|---|---|
| `Source` | `GeoDataSource` | O registro. |
| `Available` | `bool` | `false` quando o arquivo não é mais encontrado no caminho registrado (FR-010). |

## DataType

Tipo de dado geográfico que um `GeoDataSource` representa (FR-002).

| Valor | Descrição |
|---|---|
| `DataTypeBaseMap` | Mapa base. |
| `DataTypeElevation` | Relevo. |

## DataFormat

Formato de arquivo concreto reconhecido por trás de cada `DataType` (ver
`research.md` itens 1 e 3).

| Valor | `DataType` correspondente | Descrição |
|---|---|---|
| `DataFormatMBTiles` | `DataTypeBaseMap` | Container SQLite no formato MBTiles 1.3, identificado pela assinatura de arquivo SQLite. |
| `DataFormatGeoTIFF` | `DataTypeElevation` | TIFF com tags de georreferenciamento GeoTIFF em CRS geográfico (WGS84/EPSG:4326). |

## BoundingBox (estendida)

`BoundingBox` já existe desde a etapa 1 (`internal/domain/bounding_box.go`).
Esta feature adiciona dois métodos, sem alterar seus campos:

| Método | Assinatura | Descrição |
|---|---|---|
| `Contains` | `func (b BoundingBox) Contains(lat, lon float64) bool` | Reporta se o ponto geográfico `(lat, lon)` está dentro da área coberta por `b`, tratando corretamente o caso `CrossesAntimeridian` (FR-018). Usada por `ComputeCoverage` (abaixo) para decidir, ponto a ponto, se um `GeoDataSource` cobre aquele ponto. |
| `AreaDegrees` | `func (b BoundingBox) AreaDegrees() float64` | Área aproximada de `b` em graus quadrados (largura × altura, com a mesma técnica de "unwrap" de longitude usada para o antimeridiano), usada apenas como medida relativa de especificidade no desempate entre fontes sobrepostas (FR-016, ver `research.md` item 10) — não é uma área geodésica real. |

## Cobertura de um trajeto

Os tipos de saída e o algoritmo de verificação de cobertura são objetos de
domínio comuns — não DTOs de `internal/application` — pelo mesmo motivo que
`GroupSummary`/`SearchResult[T]` vivem em `internal/domain` no repositório
de referência do usuário (`waliqueiroz/mystery-gifter-api`; ver
`research.md` item 14). Vivem em `internal/domain/geo_data_coverage.go`.

| Tipo | Campo | Descrição |
|---|---|---|
| `CoverageReport` | `Status CoverageStatus` | Veredito geral: `CoverageStatusFull`, `CoverageStatusPartial` ou `CoverageStatusNone` (SC-003). |
| | `UncoveredSegments []UncoveredSegment` | Subtrechos contínuos não cobertos (vazio quando `Status == CoverageStatusFull`). |
| | `BaseMapSourcesUsed []GeoDataSource` | Conjunto (sem repetição) de registros de mapa base que cobriram pelo menos um ponto do trajeto. |
| | `ElevationSourcesUsed []GeoDataSource` | Conjunto (sem repetição) de registros de relevo que cobriram pelo menos um ponto do trajeto. |
| `UncoveredSegment` | `StartLatitude, StartLongitude float64` | Coordenadas do primeiro ponto do subtrecho não coberto. |
| | `EndLatitude, EndLongitude float64` | Coordenadas do último ponto do subtrecho não coberto. |
| | `Missing MissingDataType` | O que falta nesse subtrecho: `MissingBaseMap`, `MissingElevation` ou `MissingBoth`. |

```go
func ComputeCoverage(route []TrackPoint, baseMaps, elevations []GeoDataSource) CoverageReport
```

Função pura (sem porta, sem I/O) que implementa FR-013 a FR-018: para cada
ponto de `route`, escolhe o `GeoDataSource` de mapa base e o de relevo que
o cobrem (via `BoundingBox.Contains`), desempatando por
`BoundingBox.AreaDegrees` (menor área vence) e depois por `RegisteredAt`
mais antigo (FR-016, Clarification — spec.md); agrupa pontos consecutivos
com o mesmo status de cobertura em `UncoveredSegment` (FR-015,
Clarification — spec.md); e agrega os conjuntos de fontes usadas. Os
helpers privados (`pickCoverageWinner`, `isMoreSpecific`,
`missingDataType`, `sortedSources`) são funções livres no mesmo arquivo —
mesmo padrão de `ComputeBoundingBox`/`Haversine` no domínio, não métodos,
já que nenhum deles precisa de estado além dos parâmetros recebidos.

**Regra de decisão entre `CoverageStatusFull`, `CoverageStatusPartial` e
`CoverageStatusNone`** (resolve a ambiguidade apontada pela análise de
`/speckit-analyze` — ver Cenário de Aceitação 2 da História de Usuário 2 em
`spec.md`, agora alinhado a esta regra): o veredito é calculado a partir do
status de cobertura por ponto (totalmente coberto / falta mapa base / falta
relevo / falta ambos), nunca de uma média ou de "qual tipo falta com mais
frequência":

- **`CoverageStatusFull`**: todo ponto da rota está totalmente coberto (nenhum
  `UncoveredSegment`).
- **`CoverageStatusNone`**: nenhum ponto da rota tem cobertura de nenhum dos
  dois tipos em nenhum lugar do trajeto — equivalente a `BaseMapSourcesUsed`
  e `ElevationSourcesUsed` virem ambos vazios. Inclui o caso de nenhum
  registro cadastrado (Cenário de Aceitação 4).
- **`CoverageStatusPartial`**: todo o restante — ou seja, existe pelo menos
  um ponto totalmente coberto, ou pelo menos um dos dois conjuntos de fontes
  usadas (`BaseMapSourcesUsed`/`ElevationSourcesUsed`) não está vazio, mas
  nem todo ponto está totalmente coberto. Em particular, um trajeto com mapa
  base cobrindo 100% da extensão mas nenhum registro de relevo em lugar
  nenhum é `Partial` (há cobertura real, só que incompleta), não `None`.

`ComputeCoverage` não sabe se um `GeoDataSource` ainda existe em disco:
filtrar as fontes indisponíveis (via `FileChecker.Exists`, FR-017) antes de
chamar esta função é responsabilidade de `GeoDataService.CheckCoverage`
(orquestração, não regra de negócio).

## Erros sentinela do domínio (novos)

Conforme o Princípio VII da constituição — declarados em
`internal/domain/errors.go`, ao lado dos já existentes da etapa 1, e
traduzidos pela CLI em códigos de saída de processo (ver `contracts/cli.md`).

| Erro sentinela | Quando ocorre |
|---|---|
| `ErrDataFileNotFound` | O caminho informado no `register` não existe (FR-004). |
| `ErrDataFileUnreadable` | O arquivo existe mas não pode ser lido/aberto (ex.: permissão negada) (FR-005). |
| `ErrUnsupportedDataFormat` | O conteúdo do arquivo não corresponde a nenhum formato reconhecido (MBTiles ou GeoTIFF em CRS geográfico), incluindo um GeoTIFF em CRS projetado ou um MBTiles sem `bounds` na tabela `metadata` (FR-006). |
| `ErrDataSourceNameAlreadyUsed` | Já existe um registro com o nome informado (FR-007). |
| `ErrDataSourceNotRegistered` | Nenhum registro existe com o nome informado, ao tentar removê-lo (FR-012). |

A verificação de cobertura (`check`) reaproveita os erros sentinela de
trajeto já existentes da etapa 1 (`ErrEmptyFile`, `ErrUnsupportedFormat`,
`ErrInsufficientPoints`, `ErrInsufficientPointsAfterCleaning`) para os
mesmos cenários de trajeto inválido — nenhum erro novo é necessário para
essa parte.

## Portas (interfaces declaradas no domínio)

Nenhum arquivo genérico de portas é criado. Cada interface fica no arquivo
mais específico possível — junto da entidade que produz/manipula, ou em um
arquivo próprio nomeado pelo conceito, quando não pertence a uma única
entidade.

### GeoDataInspector (declarada em `geo_data_source.go`, junto da entidade `GeoDataSource`)

```go
type InspectedGeoData struct {
    Format      DataFormat
    Type        DataType
    BoundingBox BoundingBox
}

type GeoDataInspector interface {
    Inspect(path string) (InspectedGeoData, error)
}
```

Implementação em `internal/infra/outbound/geodatainspector`: identifica o
formato pela assinatura do conteúdo (SQLite → MBTiles/mapa base; TIFF →
GeoTIFF/relevo) e delega a leitura de área para a lógica específica de cada
formato (FR-002, FR-003, FR-006). Erros de sistema de arquivo encontrados ao
abrir `path` são traduzidos aqui para `ErrDataFileNotFound`/
`ErrDataFileUnreadable` (ver `research.md` item 8).

### GeoDataRepository (declarada em `geo_data_source.go`, junto da entidade `GeoDataSource`)

```go
type GeoDataRepository interface {
    Save(source GeoDataSource) error
    FindByName(name string) (GeoDataSource, bool, error)
    List() ([]GeoDataSource, error)
    Delete(name string) error
}
```

Implementação em `internal/infra/outbound/jsonfile`: um único
arquivo JSON em local fixo do SO, com escrita atômica (FR-008, ver
`research.md` item 5). `FindByName` devolve `(GeoDataSource{}, false, nil)`
quando não há registro com aquele nome (não é um erro). Chamada
`GeoDataRepository` — não `GeoDataRegistry` — porque é uma porta de
persistência (`research.md` item 16).

### FileChecker (declarada em `file_checker.go`, arquivo próprio — não pertence a uma única entidade)

```go
type FileChecker interface {
    Exists(path string) bool
}
```

Implementação em `internal/infra/outbound/filechecker`, baseada em
`os.Stat` (FR-010, FR-017, ver `research.md` item 7).

Não há porta `Clock`: `GeoDataService.Register` chama `time.Now()`
diretamente (`research.md` item 10.1) — `time` é biblioteca padrão, não
uma dependência externa no sentido do Princípio II.

### Mocks

Gerados com `go.uber.org/mock/mockgen` via `//go:generate` posicionado
diretamente acima de cada interface, com saída em
`internal/domain/mock_domain/` — um arquivo por porta: `geo_data_inspector.go`,
`geo_data_repository.go`, `file_checker.go`.

## Camada de serviço: `GeoDataService`

Ao contrário do que uma versão anterior deste documento descrevia (quatro
serviços de método único — `RegisterGeoDataService`, `ListGeoDataService`,
`RemoveGeoDataService`, `CheckCoverageService`, cada um com um único
`Execute`), os quatro casos de uso desta etapa são expostos por uma única
interface, `GeoDataService`, com um método nomeado por operação — o mesmo
padrão de service layer já usado em `waliqueiroz/mystery-gifter-api`
(`GroupService` reúne `Create`/`GetByID`/`AddUser`/... numa só interface;
ver `research.md` item 13). Os quatro casos de uso desta etapa operam sobre
o mesmo recurso — o registro de dados geográficos — então cabem
naturalmente juntos:

```go
type GeoDataService interface {
    Register(name, path string) (domain.GeoDataSource, error)
    List() ([]domain.GeoDataSummary, error)
    Remove(name string) error
    CheckCoverage(reader io.Reader) (domain.CoverageReport, error)
}
```

Vive em `internal/application/geo_data_service.go`. Todo tipo que aparece
nas assinaturas acima (`GeoDataSource`, `GeoDataSummary`,
`CoverageReport`) é um tipo de domínio — não um DTO de `internal/application`
— pelo mesmo motivo do item anterior: nenhum campo é `io.Writer`, nenhuma
formatação de texto acontece na camada de serviço nem no domínio; a
apresentação é responsabilidade exclusiva do adapter de entrada, a CLI.

Cada método **só orquestra** portas e delega a regra de negócio para um
construtor ou função de domínio — a mesma divisão de responsabilidade de
`GroupService.AddUser` em `waliqueiroz/mystery-gifter-api` (busca via
repositório, delega para `domain.Group.AddUser`, salva, devolve;
`research.md` item 14):

- **`Register`**: recusa se `name` já existe (`ErrDataSourceNameAlreadyUsed`,
  via `GeoDataRepository.FindByName`); chama `GeoDataInspector.Inspect(path)`;
  delega a construção do registro para `domain.NewGeoDataSource(name, path,
  inspected)`; chama `GeoDataRepository.Save`; devolve o registro criado.
- **`List`**: chama `GeoDataRepository.List`; para cada registro, monta um
  `domain.GeoDataSummary{Source: source, Available: fileChecker.Exists(source.Path)}`
  (FR-009, FR-010) — a única lógica aqui é a combinação de duas portas, não
  uma regra de negócio própria.
- **`Remove`**: recusa com `ErrDataSourceNotRegistered` se `name` não
  existir (via `GeoDataRepository.FindByName`); caso contrário chama
  `GeoDataRepository.Delete`. O arquivo original nunca é tocado (FR-011).
- **`CheckCoverage`**: chama `TrackParser.Parse(reader)`, depois
  `domain.CleanTrack(track.Points, minPoints, maxPlausibleSpeedKmh)` para
  obter a rota limpa — não simplificada/suavizada, e sem os pontos
  descartados (mesma função de domínio que `InspectTrackService.Inspect`
  usa — ver "Limpeza e resumo de um trajeto" abaixo); lista os registros
  via `GeoDataRepository.List`; filtra os que `FileChecker.Exists` reporta
  como ausentes (FR-017, único filtro que exige uma porta, por isso fica no
  serviço e não em `domain.ComputeCoverage`); delega o cálculo de
  cobertura em si para `domain.ComputeCoverage(route, baseMaps, elevations)`
  (ver seção "Cobertura de um trajeto" acima) e devolve o `CoverageReport`
  resultante sem alterá-lo.

## Limpeza e resumo de um trajeto (`InspectTrackService`, etapa 1)

Por pedido do usuário, `InspectTrackService` (etapa 1) foi alinhado ao
mesmo padrão desta etapa (`research.md` item 15) — registrado aqui porque é
o próximo elo da mesma refatoração, embora o serviço em si pertença à
etapa 1.

```go
func CleanTrack(points []TrackPoint, minPoints int, maxPlausibleSpeedKmh float64) ([]TrackPoint, DiscardStats, error)
func SummarizeTrack(track Track, route Route, discarded DiscardStats) TrackSummary
```

Ambas em `internal/domain` (`cleaning.go` e `track_summary.go`, respectivamente):

- `CleanTrack` compõe `ReorderByTime` + os três `Discard*` + a checagem de
  mínimo de pontos (antes e depois, com os dois erros sentinela
  distinguíveis) — a regra de negócio "o que significa limpar um trajeto",
  não apenas suas partes. Usada por **ambos** os serviços que precisam de
  uma rota limpa: `InspectTrackService.Inspect` e
  `GeoDataService.CheckCoverage`.
- `SummarizeTrack` monta um `TrackSummary` (antes `InspectTrackOutput`, um
  DTO de `internal/application`) a partir de `Track` + `Route` +
  `DiscardStats`, chamando `TotalDistance`, `ComputeBoundingBox`,
  `ElevationGain` e `Duration` — a mesma lógica que antes era o método
  privado `buildOutput` do serviço.

`InspectTrackService.Inspect(reader io.Reader, simplificationLevel,
smoothingLevel domain.Level) (domain.TrackSummary, error)` — sem mais
`InspectTrackInput`/`InspectTrackOutput` — chama `TrackParser.Parse`,
`domain.CleanTrack`, `Simplifier.Simplify`, `Smoother.Smooth` (essas duas,
via porta, só fazem sentido para esta etapa — `CheckCoverage` não as usa,
research.md item 9) e `domain.SummarizeTrack`, nessa ordem. Não existe mais
um helper `track_loading.go` compartilhado: a parte que chamava a porta
`TrackParser` (orquestração de verdade) ficou em cada serviço; a parte que
compunha as regras de limpeza virou `domain.CleanTrack`.
