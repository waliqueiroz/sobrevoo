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
| `RegisteredAt` | `time.Time` | Instante em que o registro foi criado, obtido de `time.Now()` diretamente por `RegisterGeoDataService` — sem uma porta dedicada (`research.md` item 10.1: `time` é biblioteca padrão, não uma dependência externa no sentido do Princípio II, e o campo nunca é exibido ao usuário). Usado apenas para o desempate determinístico entre fontes sobrepostas (FR-016, ver `research.md` item 10). |

Note-se que a entidade **não** guarda se o arquivo ainda existe: essa é uma
informação dinâmica, recalculada a cada `list`/`check` via a porta
`FileChecker` (FR-010, FR-017), nunca persistida junto do registro.

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
| `Contains` | `func (b BoundingBox) Contains(lat, lon float64) bool` | Reporta se o ponto geográfico `(lat, lon)` está dentro da área coberta por `b`, tratando corretamente o caso `CrossesAntimeridian` (FR-018). Usada por `CheckCoverageService` para decidir, ponto a ponto, se um `GeoDataSource` cobre aquele ponto. |
| `AreaDegrees` | `func (b BoundingBox) AreaDegrees() float64` | Área aproximada de `b` em graus quadrados (largura × altura, com a mesma técnica de "unwrap" de longitude usada para o antimeridiano), usada apenas como medida relativa de especificidade no desempate entre fontes sobrepostas (FR-016, ver `research.md` item 10) — não é uma área geodésica real. |

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

### GeoDataRegistry (declarada em `geo_data_source.go`, junto da entidade `GeoDataSource`)

```go
type GeoDataRegistry interface {
    Save(source GeoDataSource) error
    FindByName(name string) (GeoDataSource, bool, error)
    List() ([]GeoDataSource, error)
    Delete(name string) error
}
```

Implementação em `internal/infra/outbound/geodatastore/jsonfile`: um único
arquivo JSON em local fixo do SO, com escrita atômica (FR-008, ver
`research.md` item 5). `FindByName` devolve `(GeoDataSource{}, false, nil)`
quando não há registro com aquele nome (não é um erro).

### FileChecker (declarada em `file_checker.go`, arquivo próprio — não pertence a uma única entidade)

```go
type FileChecker interface {
    Exists(path string) bool
}
```

Implementação em `internal/infra/outbound/filechecker`, baseada em
`os.Stat` (FR-010, FR-017, ver `research.md` item 7).

Não há porta `Clock`: `RegisterGeoDataService` chama `time.Now()`
diretamente (`research.md` item 10.1) — `time` é biblioteca padrão, não
uma dependência externa no sentido do Princípio II.

### Mocks

Gerados com `go.uber.org/mock/mockgen` via `//go:generate` posicionado
diretamente acima de cada interface, com saída em
`internal/domain/mock_domain/` — um arquivo por porta: `geo_data_inspector.go`,
`geo_data_registry.go`, `file_checker.go`.

## DTOs da camada de serviço

Vivem em `internal/application` — a forma de entrada/saída de cada novo
serviço, não conceitos de negócio por si só.

### RegisterGeoDataService

| Tipo | Campo | Descrição |
|---|---|---|
| `RegisterGeoDataInput` | `Name string` | Nome escolhido pelo usuário. |
| | `Path string` | Caminho do arquivo a registrar. |
| `RegisterGeoDataOutput` | `Source domain.GeoDataSource` | O registro criado, com tipo, formato e área já determinados. |

`Execute`: recusa se `Name` já existe (`ErrDataSourceNameAlreadyUsed`),
chama `GeoDataInspector.Inspect(Path)`, monta o `GeoDataSource` (com
`RegisteredAt` vindo de `time.Now()`) e chama `GeoDataRegistry.Save`.

### ListGeoDataService

| Tipo | Campo | Descrição |
|---|---|---|
| `GeoDataSummary` | `Source domain.GeoDataSource` | O registro. |
| | `Available bool` | `false` quando o arquivo não é mais encontrado no caminho registrado (FR-010). |
| `ListGeoDataOutput` | `Sources []GeoDataSummary` | Todos os registros, na ordem devolvida por `GeoDataRegistry.List`. |

`Execute`: sem entrada; para cada registro de `GeoDataRegistry.List()`,
preenche `Available` via `FileChecker.Exists`.

### RemoveGeoDataService

| Tipo | Campo | Descrição |
|---|---|---|
| `RemoveGeoDataInput` | `Name string` | Nome do registro a remover. |

`Execute`: recusa com `ErrDataSourceNotRegistered` se não existir; caso
contrário chama `GeoDataRegistry.Delete`. Não há tipo de saída além do
erro — o arquivo original nunca é tocado (FR-011).

### CheckCoverageService

| Tipo | Campo | Descrição |
|---|---|---|
| `CheckCoverageInput` | `Reader io.Reader` | Conteúdo do arquivo de trajeto a verificar (mesmo formato de entrada de `InspectTrackInput`). |
| `CheckCoverageOutput` | `Status CoverageStatus` | Veredito geral: `CoverageStatusFull`, `CoverageStatusPartial` ou `CoverageStatusNone` (SC-003). |
| | `UncoveredSegments []UncoveredSegment` | Subtrechos contínuos não cobertos (vazio quando `Status == CoverageStatusFull`). |
| | `BaseMapSourcesUsed []domain.GeoDataSource` | Conjunto (sem repetição) de registros de mapa base que cobriram pelo menos um ponto do trajeto. |
| | `ElevationSourcesUsed []domain.GeoDataSource` | Conjunto (sem repetição) de registros de relevo que cobriram pelo menos um ponto do trajeto. |

| Tipo auxiliar | Campo | Descrição |
|---|---|---|
| `UncoveredSegment` | `StartLatitude, StartLongitude float64` | Coordenadas do primeiro ponto do subtrecho não coberto. |
| | `EndLatitude, EndLongitude float64` | Coordenadas do último ponto do subtrecho não coberto. |
| | `Missing MissingDataType` | O que falta nesse subtrecho: `MissingBaseMap`, `MissingElevation` ou `MissingBoth`. |

`Execute`: obtém a rota limpa (não simplificada/suavizada) via o helper
compartilhado de `track_loading.go` (mesmo `TrackParser` e mesmas funções
puras de limpeza da etapa 1); lista os registros via `GeoDataRegistry.List`,
descartando os que `FileChecker.Exists` reporta como ausentes (FR-017);
para cada ponto da rota, determina o `GeoDataSource` de mapa base e o de
relevo que o cobrem (usando `BoundingBox.Contains`), escolhendo entre
candidatos do mesmo tipo pelo critério de desempate de `BoundingBox.AreaDegrees`
(menor área vence; empate por `RegisteredAt` mais antigo — FR-016); agrupa
pontos consecutivos com o mesmo status de cobertura em `UncoveredSegment`
(FR-015); e agrega os conjuntos de fontes usadas.

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

Assim como `InspectTrackOutput`, todos os DTOs acima são dado puro — nenhum
campo é `io.Writer`, nenhuma formatação de texto acontece na camada de
serviço; a apresentação é responsabilidade exclusiva do adapter de entrada
(CLI).
