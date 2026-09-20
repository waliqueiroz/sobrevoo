# Modelo de Dados: Recorte de Dados Geográficos para o Voo

**Feature**: `004-geo-data-slice` | **Data**: 2026-09-20

Todos os tipos abaixo vivem em `internal/domain` (Princípio IX: DTOs de saída
não triviais e regra de negócio são do domínio, nunca de
`internal/application`). Nomes de campos e tipos em inglês, como no restante
do código. O núcleo não persiste nada; o único destino externo é o arquivo
exportado (`contracts/slice-file.md`). Os tipos que já existem
(`GeoDataSource`, `BoundingBox`, `CoverageReport`, `CameraPlan`, `Route`)
**não mudam** — esta etapa só acrescenta.

## Entidades

### `SliceTuning` (`geo_slice.go`)

Constantes de ajuste, injetadas (Princípio VIII; `research.md` item 15).

| Campo | Tipo | Valor inicial |
|---|---|---|
| `MarginFactor` | `float64` | 1,0 |
| `ReferenceHeightPixels` | `float64` | 1080 |
| `TexelScreenRatio` | `float64` | 2,0 |
| `EstimatedTileBytes` | `int64` | 65 536 |
| `MaxSizeBytes` | `int64` | 268 435 456 |

Constantes de domínio (fatos da grade, não ajuste): `TilePixels = 256`,
`BytesPerElevationSample = 4`, `MaxMercatorLatitude = 85.0511287798`,
`MetersPerDegree = 111 320`, `EquatorResolution = 156 543.03392`.

### `CameraPlan` (existente) — acréscimos

| Método | Regra |
|---|---|
| `Validate() error` | coerência do plano lido (`research.md` item 1); `ErrPlanFileInvalid` |
| `AreaOfInterest(tuning SliceTuning) BoundingBox` | área que o voo precisa ver (item 2) |

### `BoundingBox` (existente) — acréscimos

| Método | Regra |
|---|---|
| `Regions(baseMaps, elevations []GeoDataSource) ([]SliceRegion, Route)` | decomposição por compressão de coordenadas + `Route` com os centros (item 3) |
| `TileRange(level int) []TileRange` | peças XYZ que cobrem a caixa no nível; dois intervalos de `x` quando cruza o antimeridiano (item 6) |
| `Intersects(other BoundingBox) bool` | usado para escolher os candidatos de `Regions`; mesmo tratamento de antimeridiano de `Contains` |

### `SliceRegion` (`geo_slice.go`)

Retângulo da área em que o vencedor de cada tipo é uniforme.

| Campo | Tipo | Observação |
|---|---|---|
| `Box` | `BoundingBox` | limites da região |
| `BaseMap` | `GeoDataSource` | vencedor de mapa base (menor área, empate: o mais antigo) |
| `Elevation` | `GeoDataSource` | vencedor de relevo |

Quando `Route.Coverage` acusa área não coberta, o serviço **não** chega a
usar as regiões para ler: devolve `AreaNotCoveredError` (abaixo).

### `DetailLevel` (`tile.go`)

O nível de detalhe escolhido para **um** registro de mapa base (FR-006).

| Campo | Tipo | Observação |
|---|---|---|
| `Ideal` | `int` | `z*` calculado (`research.md` item 5) |
| `Chosen` | `int` | `clamp(Ideal, Min, Max)` |
| `Min`, `Max` | `int` | intervalo que o arquivo oferece (`BaseMapReader.Levels`) |
| `Reason` | `string` | texto explicativo: distância mínima da câmera, latitude de referência, resolução exigida, e se houve limitação ("within the source's range", "above ... maximum", "below ... minimum") |

Construtor de regra: `SliceTuning.DetailLevel(minCameraDistance, fov float64,
area BoundingBox, offered LevelRange) DetailLevel` (método de
`SliceTuning`, que é o dono das constantes).

### `TileID` e `Tile` (`tile.go`)

Porta `BaseMapReader` no topo do arquivo (item 7); depois:

| Tipo | Campos | Observação |
|---|---|---|
| `LevelRange` | `Min`, `Max int` | níveis oferecidos |
| `TileID` | `Level, X, Y int` | esquema XYZ (`Y` cresce para o sul); TMS só existe dentro do adapter |
| `TileRange` | `Level, MinX, MaxX, MinY, MaxY int` | retângulo de peças |
| `Tile` | `ID TileID`, `Data []byte` | bytes originais da peça, sem decodificar |
| `TileRead` | `Format string`, `Tiles []Tile`, `Missing []TileID` | o que `ReadTiles` devolveu: formato (`png`, `jpg`, ...), peças presentes e ausentes |

`TileSet` — as peças de um registro de mapa base num nível:

| Campo | Tipo | Observação |
|---|---|---|
| `Source` | `GeoDataSource` | de qual registro vieram |
| `Detail` | `DetailLevel` | nível escolhido e motivo |
| `Format` | `string` | formato de imagem das peças |
| `Tiles` | `[]Tile` | ordenadas por `(Level, X, Y)` |
| `Missing` | `[]TileID` | peças requeridas que o registro não contém; ordenadas |

Regra de pertencimento (item 4): uma peça requerida pertence à região que
contém o centro de `peça ∩ área`; `TileSet.Source` é o vencedor de mapa
base dessa região.

### `ElevationGridInfo`, `GridWindow`, `ElevationGrid` (`elevation_grid.go`)

Porta `ElevationReader` no topo do arquivo (item 8); depois:

`ElevationGridInfo` — geometria de uma grade de relevo, lida só das tags:

| Campo | Tipo | Observação |
|---|---|---|
| `Rows`, `Cols` | `int` | tamanho da grade |
| `NorthLatitude`, `WestLongitude` | `float64` | canto noroeste da célula (0,0) (já deslocado de meia célula se `PixelIsPoint`) |
| `CellLatitude`, `CellLongitude` | `float64` | tamanho da célula em graus (positivos) |
| `UnitToMeters` | `float64` | 1 (metro), 0,3048 (pé), 1200/3937 (pé US survey) |

Métodos: `Window(box BoundingBox) GridWindow` (as células cujo **centro** cai
em `box`, com a regra do intervalo semiaberto do item 4; janela vazia é
válida), `CellAt(lat, lon float64) (row, col int)` (regra do item 9).

`GridWindow`: `FirstRow, FirstCol, Rows, Cols int` — sempre retangular, em
linhas de norte a sul e colunas de oeste a leste; nunca cruza o
antimeridiano (uma janela que o cruzaria vira duas).

`ElevationGrid` — as amostras de uma região:

| Campo | Tipo | Observação |
|---|---|---|
| `Source` | `GeoDataSource` | registro de relevo de origem |
| `Window` | `GridWindow` | posição na grade de origem |
| `NorthLatitude`, `WestLongitude` | `float64` | canto noroeste da célula `(0,0)` **da janela** |
| `CellLatitude`, `CellLongitude` | `float64` | tamanho da célula |
| `Values` | `[]float32` (privado) | `Rows × Cols`, linha a linha, em **metros**; "sem valor" = NaN, nunca visto fora do tipo |

Métodos: `At(row, col int) (meters float64, hasValue bool)`, `Rows()`,
`Cols()`, `NoValueCount() int`, `Range() (min, max float64, ok bool)` (só
sobre amostras com valor; `ok == false` quando não há nenhuma).
Construtor `NewElevationGrid(...)` recebe valores em metros com o "sem valor"
já marcado.

`ElevationWindow` (retorno de `ReadWindow`): `Values []float32` (mesma
convenção) — o construtor de `ElevationGrid` o consome.

### `ElevationReading` (`elevation_grid.go`)

Resposta da consulta isolada (FR-017):

| Campo | Tipo | Observação |
|---|---|---|
| `Latitude`, `Longitude` | `float64` | coordenada consultada, longitude normalizada para [-180, 180) |
| `Meters` | `float64` | válido só se `HasValue` |
| `HasValue` | `bool` | `false` = o arquivo não informa valor para a célula |
| `Source` | `GeoDataSource` | relevo usado |
| `Row`, `Col` | `int` | célula da grade de origem, para conferência com outras ferramentas |

"Nenhum relevo cobre o ponto" **não** é um `ElevationReading`: é o erro
`ErrElevationNotCovered`.

### `GeoSlice` e `SliceSummary` (`geo_slice.go`)

Porta `GeoSliceExporter` no topo do arquivo; depois:

`GeoSlice` — o recorte completo (entidade-chave "Recorte de Dados
Geográficos"):

| Campo | Tipo | Observação |
|---|---|---|
| `Area` | `BoundingBox` | área de interesse |
| `TileSets` | `[]TileSet` | ordenados por `(nome do registro, nível)` |
| `Elevation` | `[]ElevationGrid` | uma por região com amostras, na ordem das regiões |
| `Summary` | `SliceSummary` | calculado por construtor |

`NewGeoSlice(area, tileSets, grids) GeoSlice` calcula o resumo a partir do
conteúdo, para que ele nunca discorde do recorte (mesmo padrão de
`NewCameraPlan`).

`SliceSummary`:

| Campo | Significado (FR-012) |
|---|---|
| `Area` | área coberta |
| `TileCount`, `MissingTileCount` | peças presentes / ausentes |
| `SampleCount`, `NoValueSampleCount` | amostras de elevação / sem valor |
| `MinElevation`, `MaxElevation`, `HasElevationRange` | faixa em metros, só sobre amostras com valor |
| `Sources` | `[]SliceSourceUse` — cada registro usado (nome, tipo, formato, caminho) |
| `SizeBytes` | bytes reais do recorte: soma dos bytes das peças e `4 × amostras` |

`SliceSourceUse`: `Source GeoDataSource` e, para mapa base, o
`DetailLevel`.

### `AreaNotCoveredError` (`errors.go`)

Erro tipado para a recusa de cobertura (FR-005): campo `Report
CoverageReport` (`Status`, `UncoveredSegments`, fontes usadas); `Error()`
formata os subtrechos como o comando `geodata check` os mostra; `Is(target)`
casa `ErrAreaNotCovered`. É o único erro novo com dados estruturados; os
demais são sentinelas embrulhados com `fmt.Errorf("%w: ...")`.

## Portas novas (resumo)

| Porta | Arquivo | Implementação | Métodos |
|---|---|---|---|
| `CameraPlanReader` | `camera_plan.go` | `jsonfile.NewCameraPlanReader()` | `Read(path) (CameraPlan, error)` |
| `BaseMapReader` | `tile.go` | `basemapreader.NewMBTiles()` | `Levels(path)`, `ReadTiles(path, level, ids)` |
| `ElevationReader` | `elevation_grid.go` | `elevationreader.NewGeoTIFF()` | `Describe(path)`, `ReadWindow(path, window)` |
| `GeoSliceExporter` | `geo_slice.go` | `zipfile.NewGeoSliceExporter()` | `Export(slice, path, overwrite)` |

## Funções de domínio de seleção

`SelectSource(sources []GeoDataSource, lat, lon float64) (GeoDataSource,
bool)` — exporta a regra hoje interna a `pickCoverageWinner` (menor
`AreaDegrees`, empate pelo `RegisteredAt` mais antigo) sem alterar a
lógica; é o que `GeoDataService.ElevationAt` e `BoundingBox.Regions` usam,
para que consulta, cobertura e recorte concordem (FR-018).

## Estados e transições

Nenhuma entidade tem ciclo de vida: cada uma é o resultado de uma chamada.
O fluxo de `GeoSliceService.Generate` é linear e cada passo pode terminar
em recusa **antes** de qualquer leitura de conteúdo, exceto o último:

```text
plano válido? ──não──▶ ErrPlanFileInvalid            (em Load/Validate)
   │sim
área de interesse
   │
registros disponíveis (FileChecker) → regiões + Route.Coverage
   │                                     └─ incompleta ─▶ AreaNotCoveredError
   │completa
Levels + Describe (só metadados) ─────┬─ ilegível ──────▶ ErrGeoDataContentUnreadable
   │                                     └─ unidade ───────▶ ErrElevationUnitUnsupported
nível de detalhe + estimativa
   │                                     └─ > limite ──▶ ErrSliceTooLarge
   │ok
ReadTiles / ReadWindow por região        └─ arquivo ilegível ▶ ErrGeoDataContentUnreadable
   │                                     └─ bytes > limite ▶ ErrSliceTooLarge
NewGeoSlice → GeoSlice (com peças ausentes e amostras sem valor registradas)
```
