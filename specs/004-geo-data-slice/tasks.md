---

description: "Task list for feature implementation"
---

# Tarefas: Recorte de Dados Geográficos para o Voo

**Entrada**: Documentos de design de `/specs/004-geo-data-slice/`

**Pré-requisitos**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/cli.md`, `contracts/slice-file.md`, `quickstart.md`

**Testes**: incluídos. Como nas etapas anteriores, a constituição do projeto
(Princípio VI — Testes Automatizados no Núcleo; Princípio X —
Given/When/Then, builders e isolamento por camada) exige testify e uber-go/mock
e proíbe testes tabulares: cada cenário é um `t.Run("should ...")` com
`// given`, `// when`, `// then`. As tarefas de teste abaixo materializam
essa exigência; ficam antes da implementação correspondente em cada fase.

**Organização**: as tarefas são agrupadas por história de usuário (P1–P6 de
`spec.md`) para permitir implementação e teste independentes de cada
história. A **Phase 2** tem três partes: (A) a extração prévia de
`atomicfile` do exportador da etapa 3 (`research.md` item 13), que precisa
estar concluída e verde antes de qualquer código novo; (B) a base
compartilhada (erros, códigos de saída, tipos de domínio e portas,
configuração, mocks, builders); e (C) as fixtures com conteúdo em
`test/helper`.

**Divisão do trabalho entre histórias** (o mesmo arquivo é estendido por mais
de uma história, nunca em paralelo):

| História | O que entrega |
|---|---|
| US1 | ler e validar o plano; área de interesse; regiões e cobertura (recusa `ErrAreaNotCovered`); leitores de MBTiles e GeoTIFF (conteúdo); nível de detalhe (`Ideal`/`Chosen`, sem texto de motivo); `GeoSliceService.Generate`; comando `geodata slice <plano>` com saída mínima |
| US2 | motivo explicável do nível de detalhe, propriedades (monotonicidade, determinismo, latitude de referência), linhas `Base map detail` |
| US3 | resumo completo (`NewGeoSlice` calcula `SliceSummary`; formatação com faixa de elevação, sem valor, peças ausentes, registros, tamanho) |
| US4 | exportação em ZIP atômica (`zipfile`), `--export` e `--overwrite` |
| US5 | recorte grande demais (estimativa + guarda), registro corrompido, unidade não suportada, peças ausentes tratadas de ponta a ponta |
| US6 | `geodata elevation --lat --lon` (`GeoDataService.ElevationAt`, `Coordinate`, três respostas) |

## Formato: `[ID] [P?] [Story] Descrição`

- **[P]**: pode ser executado em paralelo (arquivos diferentes, sem
  dependência de tarefa incompleta). Tarefas que editam o mesmo arquivo NUNCA
  são marcadas `[P]` entre si, mesmo quando logicamente independentes.
- **[Story]**: a qual história de usuário esta tarefa pertence (US1 a US6).
  Tarefas de Setup, Foundational e Polish não têm esse rótulo.
- Toda tarefa inclui o caminho de arquivo exato a criar/editar.
- Comentários de código, identificadores, mensagens de commit, flags, saída
  e mensagens de erro em tempo de execução: **inglês**; artefatos do
  Spec Kit: português (constituição, "Idioma dos Artefatos").

## Convenções de Caminho

Mesmo projeto único em Go, mesma estrutura hexagonal de `plan.md`:
`cmd/sobrevoo/`, `internal/domain/`, `internal/application/`,
`internal/infra/outbound/`, `internal/infra/inbound/cli/`. Mocks em
`internal/domain/mockdomain` e `internal/application/mockapplication`;
builders em `internal/domain/builddomain`; fixtures em `test/helper`. Regras
de estilo: sem `ports.go`; porta no arquivo da entidade, no topo, com
`//go:generate` logo após `package`; receivers curtos e consistentes com o
tipo (`c` `CameraPlan`, `b` `BoundingBox`, `t` `SliceTuning`, `g`
`ElevationGrid`, `s` os `*Service`, `r` readers, `e` exporters); `new(x)` do
Go 1.26 em vez de um helper `ptr`; nome exportado nunca repete o pacote.

---

## Phase 1: Setup (Shared Infrastructure)

**Propósito**: garantir um ponto de partida verde.

- [ ] T001 Se ainda estiver em `main`, criar e mudar para o branch `004-geo-data-slice` (`git switch -c 004-geo-data-slice`); rodar `make test`, `make lint` e `make generate` e confirmar que os três passam e que `make generate` não produz diff (`git status` limpo fora de `specs/004-geo-data-slice/`); registrar qualquer falha pré-existente antes de continuar

**Checkpoint**: baseline verde.

---

## Phase 2: Foundational (Blocking Prerequisites)

**⚠️ CRÍTICO**: nenhuma tarefa de história de usuário pode começar até que
esta fase esteja completa.

### Parte A — Extração prévia: `atomicfile` (research.md item 13)

**Propósito**: mover a publicação atômica de arquivo (hoje em `jsonfile`,
`publishExclusive` e a sequência temporário → chmod → rename) para um pacote
utilitário compartilhado com o futuro `zipfile`. **Nenhum comportamento da
etapa 3 muda**: `plan --export` e os testes existentes continuam válidos.

- [ ] T002 [P] Criar `internal/infra/outbound/atomicfile/atomic_file_test.go` (pacote `atomicfile_test`, e um arquivo interno `atomic_file_fallback_test.go` para o fallback): migrar os 3 cenários de `internal/infra/outbound/jsonfile/camera_plan_exporter_fallback_test.go` (criar o destino diretamente quando o `link` não é possível; ainda recusar destino existente; reportar destino inválido quando também não pode criar) e acrescentar: publica um arquivo novo com o conteúdo escrito pelo callback; sem `overwrite`, destino existente → `errors.Is(err, atomicfile.ErrExists)` e o arquivo existente intacto; com `overwrite`, o conteúdo antigo é substituído integralmente; diretório inexistente → `errors.Is(err, atomicfile.ErrInvalid)` com a causa preservada; erro devolvido pelo callback → nenhum arquivo no destino e nenhum `.sobrevoo-*.tmp` deixado no diretório; permissão final `0o644`
- [ ] T003 Criar `internal/infra/outbound/atomicfile/atomic_file.go`: `Publish(path string, overwrite bool, write func(io.Writer) error) error` e os sentinelas `ErrExists` e `ErrInvalid`; escreve num temporário `.sobrevoo-*.tmp` no mesmo diretório do destino (`os.CreateTemp`), chama `write`, fecha, `Chmod 0o644`; com `overwrite`, `os.Rename`; sem, publica com `os.Link` (atômico) e, se o `link` falhar por outro motivo que "já existe", cai para `os.OpenFile(O_WRONLY|O_CREATE|O_EXCL)` copiando o conteúdo do temporário, removendo o destino se a cópia falhar; remove o temporário sempre (`defer`); mesma lógica de `publishExclusive` em `jsonfile/camera_plan_exporter.go` (depende de T002)
- [ ] T004 Editar `internal/infra/outbound/jsonfile/camera_plan_exporter.go`: `Export` passa a codificar o plano e chamar `atomicfile.Publish(path, overwrite, ...)`, traduzindo `atomicfile.ErrExists` para `domain.ErrPlanDestinationExists` (mesma mensagem: `"%w: %s (use --overwrite to replace it)"`) e `atomicfile.ErrInvalid` para `domain.ErrPlanDestinationInvalid` (mesmo `invalidDestination`); remover `publishExclusive`; remover `camera_plan_exporter_fallback_test.go` (migrado em T002); `camera_plan_exporter_test.go` NÃO muda (depende de T003)
- [ ] T005 Rodar `make test`, `make lint` e `make generate` (sem diff) e conferir que `sobrevoo plan <gpx> --export` mantém o mesmo comportamento e os mesmos códigos de saída (depende de T004)

**Checkpoint A**: `make test`, `make lint` e `make generate` verdes;
`plan --export` inalterado. Nenhum código da etapa 4 existe ainda além do
`atomicfile`.

### Parte B — Base compartilhada

- [ ] T006 [P] Estender `internal/domain/errors.go` com os 10 sentinelas novos (cada um com comentário em inglês; Princípio VII): `ErrPlanFileInvalid`, `ErrPlanFormatVersionUnsupported`, `ErrAreaNotCovered`, `ErrSliceTooLarge`, `ErrGeoDataContentUnreadable`, `ErrElevationUnitUnsupported`, `ErrSliceDestinationExists`, `ErrSliceDestinationInvalid`, `ErrElevationNotCovered`, `ErrInvalidCoordinate`; e o tipo `AreaNotCoveredError` (campo `Report CoverageReport`; `Error()` formata um subtrecho por linha como `geodata check` mostra — tipo que falta (`MissingDataType`) e coordenadas de início/fim —, prefixado por `area is not fully covered by the registered geo data`; `Is(target error) bool` devolve `target == ErrAreaNotCovered`) (`data-model.md`, "AreaNotCoveredError")
- [ ] T007 [P] Editar `internal/infra/inbound/cli/exit_code.go` mapeando com `errors.Is`: `ErrPlanFileInvalid`→17, `ErrPlanFormatVersionUnsupported`→18, `ErrAreaNotCovered`→19, `ErrSliceTooLarge`→20, `ErrGeoDataContentUnreadable`→21, `ErrElevationUnitUnsupported`→22, `ErrSliceDestinationExists`→23, `ErrSliceDestinationInvalid`→24, `ErrElevationNotCovered`→25, `ErrInvalidCoordinate`→26; estender `internal/infra/inbound/cli/exit_code_test.go` (um `t.Run` por sentinela, um com o erro embrulhado por `fmt.Errorf("%w: ...")`, um com `AreaNotCoveredError` por `errors.Is`, e um que confirma que os códigos 1–16 existentes não mudaram) (depende de T006)
- [ ] T008 [P] Criar `internal/domain/tile.go`: diretiva `//go:generate go run go.uber.org/mock/mockgen -destination mockdomain/base_map_reader.go -package mockdomain . BaseMapReader` logo após `package`; a interface `BaseMapReader` no topo, após os imports (`Levels(path string) (LevelRange, error)`; `ReadTiles(path string, level int, ids []TileID) (TileRead, error)`); depois os tipos de `data-model.md`: `LevelRange{Min, Max int}`, `TileID{Level, X, Y int}` (esquema XYZ, `Y` cresce para o sul), `TileRange{Level, MinX, MaxX, MinY, MaxY int}`, `Tile{ID TileID; Data []byte}`, `TileRead{Format string; Tiles []Tile; Missing []TileID}`, `DetailLevel{Ideal, Chosen, Min, Max int; Reason, Explanation string}` (`Reason` = o motivo curto da limitação; `Explanation` = a explicação do cálculo) e `TileSet{Source GeoDataSource; Detail DetailLevel; Format string; Tiles []Tile; Missing []TileID}`; só declarações, sem comportamento (o comportamento entra nas histórias)
- [ ] T009 [P] Criar `internal/domain/elevation_grid.go`: diretiva `//go:generate ... -destination mockdomain/elevation_reader.go -package mockdomain . ElevationReader` logo após `package`; a interface `ElevationReader` no topo (`Describe(path string) (ElevationGridInfo, error)`; `ReadWindow(path string, window GridWindow) (ElevationWindow, error)`); depois `ElevationGridInfo{Rows, Cols int; NorthLatitude, WestLongitude, CellLatitude, CellLongitude, UnitToMeters float64}`, `GridWindow{FirstRow, FirstCol, Rows, Cols int}`, `ElevationWindow{Values []float32}` ("sem valor" = NaN, só dentro do tipo), `ElevationGrid` (campos `Source GeoDataSource`, `Window GridWindow`, `NorthLatitude`, `WestLongitude`, `CellLatitude`, `CellLongitude float64` e `values []float32` **privado**; construtor `NewElevationGrid(source, window, info, values []float32) ElevationGrid`) e `ElevationReading{Latitude, Longitude, Meters float64; HasValue bool; Source GeoDataSource; Row, Col int}`; só declarações e o construtor
- [ ] T010 [P] Criar `internal/domain/geo_slice.go`: diretiva `//go:generate ... -destination mockdomain/geo_slice_exporter.go -package mockdomain . GeoSliceExporter` logo após `package`; a interface `GeoSliceExporter` no topo (`Export(slice GeoSlice, path string, overwrite bool) error`, com o comentário do contrato: sem `overwrite` recusa destino existente com `ErrSliceDestinationExists`; nunca deixa resultado parcial); `SliceTuning` (`MarginFactor float64`, `ReferenceHeightPixels float64`, `TexelScreenRatio float64`, `EstimatedTileBytes int64`, `MaxSizeBytes int64`), as constantes de domínio `TilePixels = 256`, `BytesPerElevationSample = 4`, `MaxMercatorLatitude = 85.0511287798`, `MetersPerDegree = 111320`, `EquatorResolution = 156543.03392`, `SliceRegion{Box BoundingBox; BaseMap, Elevation GeoDataSource}`, `SliceSourceUse{Source GeoDataSource; Detail *DetailLevel}`, `SliceSummary` (campos de `data-model.md`: `Area`, `TileCount`, `MissingTileCount`, `SampleCount`, `NoValueSampleCount`, `MinElevation`, `MaxElevation`, `HasElevationRange`, `Sources`, `SizeBytes`), `GeoSlice{Area BoundingBox; TileSets []TileSet; Elevation []ElevationGrid; Summary SliceSummary}` e `NewGeoSlice(area BoundingBox, tileSets []TileSet, grids []ElevationGrid) GeoSlice` que por ora só monta a struct (o resumo entra na US3)
- [ ] T011 [P] Editar `internal/domain/camera_plan.go`: acrescentar, logo após a diretiva `//go:generate` existente, `//go:generate go run go.uber.org/mock/mockgen -destination mockdomain/camera_plan_reader.go -package mockdomain . CameraPlanReader`, e declarar a interface `CameraPlanReader` (`Read(path string) (CameraPlan, error)`, com comentário: devolve `ErrPlanFileInvalid`/`ErrPlanFormatVersionUnsupported`; um erro de E/S do arquivo sobe embrulhado, sem sentinela) logo abaixo de `CameraPlanExporter`, antes de `Phase`
- [ ] T012 Rodar `make generate` para gerar `internal/domain/mockdomain/{base_map_reader,elevation_reader,geo_slice_exporter,camera_plan_reader}.go` (depende de T008, T009, T010, T011)
- [ ] T013 [P] Editar `internal/infra/outbound/config/config.go`: tipo `SliceTuning` próprio do pacote (mesmos campos de `domain.SliceTuning`, sem importar o domínio), campo `Config.SliceTuning` preenchido em `Load()` com os valores iniciais de `research.md` item 15 (`MarginFactor` 1,0; `ReferenceHeightPixels` 1080; `TexelScreenRatio` 2,0; `EstimatedTileBytes` 65536; `MaxSizeBytes` 268435456); estender `internal/infra/outbound/config/config_test.go` com um cenário que confere esses valores
- [ ] T014 Editar `cmd/sobrevoo/config_mapping.go`: `domainSliceTuning(t config.SliceTuning) domain.SliceTuning` (depende de T010, T013)
- [ ] T015 [P] Criar `internal/domain/builddomain/slice_tuning_builder.go`: `NewSliceTuningBuilder()` com os mesmos valores iniciais de T013 e `WithMarginFactor`, `WithReferenceHeightPixels`, `WithTexelScreenRatio`, `WithEstimatedTileBytes`, `WithMaxSizeBytes`, `Build()`; receiver `b`; um cenário em `config_test.go` (mesmo padrão de `camera_tuning_builder`) garante que builder e configuração coincidem (depende de T010, T013)
- [ ] T016 [P] Criar `internal/domain/builddomain/elevation_grid_builder.go` (`NewElevationGridBuilder()`: grade 3×3 em metros, fonte `GeoDataSourceBuilder` de relevo, canto noroeste `(-23, -47)`, célula 0,001°; `WithSource`, `WithWindow`, `WithOrigin`, `WithCellSize`, `WithValues(...float32)`, `WithNoValueAt(row, col)`, `Build()`), `internal/domain/builddomain/tile_set_builder.go` (`NewTileSetBuilder()`: um registro de mapa base, nível 16, três peças `png`; `WithSource`, `WithLevel`, `WithTiles`, `WithMissing`, `Build()`), `internal/domain/builddomain/geo_slice_builder.go` (`NewGeoSliceBuilder()`: um `TileSet` e uma `ElevationGrid` padrão; `WithArea`, `WithTileSets`, `WithElevation`, `Build()` via `NewGeoSlice`) e `internal/domain/builddomain/elevation_reading_builder.go` (`NewElevationReadingBuilder()`: `WithMeters`, `WithoutValue`, `WithSource`, `WithCell`, `Build()`); todos com receiver `b` (depende de T008, T009, T010)

**Checkpoint B**: erros, códigos de saída, tipos, portas, mocks, configuração
e builders prontos; nenhum comando novo ainda.

### Parte C — Fixtures com conteúdo (`test/helper`)

**Propósito**: os fixtures atuais só têm metadados; os leitores precisam de
conteúdo. Tudo gerado em código, sem binário versionado (mesmo estilo dos
fixtures existentes).

- [ ] T017 [P] Estender `test/helper/mbtiles_fixture.go`: `MBTilesWithTiles(spec MBTilesSpec) []byte` (`MBTilesSpec`: `Bounds` (minLon, minLat, maxLon, maxLat), `MinZoom`/`MaxZoom` opcionais (`*int`; ausentes ⇒ só a tabela `tiles` os revela), `Format` (`png` por padrão; `jpg`, `webp`, `pbf`), `Tiles []MBTile{Z, X, Y int; Data []byte}` em **XYZ** — o fixture grava `tile_row = 2^z − 1 − y` (TMS) — e `TileData(z, x, y)` que gera um `[]byte` determinístico e distinto por peça) e `CorruptMBTiles() []byte` (`metadata` com `bounds` válidos, mas tabela `tiles` inexistente/truncada, de modo que `Inspect` a aceite e a leitura de conteúdo falhe); manter `ValidMBTiles`, `MBTilesWithoutBounds` e `NotSQLiteContent` inalterados
- [ ] T018 [P] Estender `test/helper/geotiff_fixture.go`: `GeoTIFFWithSamples(spec GeoTIFFSpec) []byte` (`GeoTIFFSpec`: `Width`, `Height`, `OriginLon`, `OriginLat`, `ScaleX`, `ScaleY`, `SampleType` (`Uint8`, `Int16`, `Uint16`, `Int32`, `Float32`, `Float64`), `Values [][]float64` (linhas de norte a sul), `Compression` (`None`, `Deflate` (código 8), `AdobeDeflate` (32946), `LZW`, e `JPEG` só para o teste de codificação não suportada), `Predictor` (1, 2, 3), `Layout` (faixas com `RowsPerStrip`, ou peças `TileWidth × TileLength`), `ByteOrder` (`II`/`MM`), `NoData *string` (tag `GDAL_NODATA` 42113, ASCII), `VerticalUnit *uint16` (`VerticalUnitsGeoKey` 4099: 9001, 9002, 9003 ou outro), `PixelIsPoint bool` (`GTRasterTypeGeoKey` 1025) e `ProjectedCRS bool`), codificado à mão como o `buildGeoTIFF` atual (sem biblioteca de TIFF), com um **codificador LZW mínimo no próprio fixture** (MSB, "early change", o pacote `golang.org/x/image/tiff/lzw` só decodifica) e `zlib` da biblioteca padrão para Deflate; `Truncated(data []byte) []byte` para simular arquivo cortado; manter as funções atuais inalteradas
- [ ] T019 [P] Criar `test/helper/camera_plan_fixture.go`: `ValidPlanFile(spec PlanFileSpec) []byte` gerando JSON no formato de `specs/003-camera-path-planning/contracts/plan-file.md` (`format_version` 1, `parameters`, `summary`, `frames` compactos, um por linha, ordem de campos fixa; sem importar `jsonfile`), com `PlanFileSpec` (`Frames []PlanFrameSpec` — câmera, marcador, distância câmera–marcador, fase —, `DurationSeconds`, `FrameRate`), e as variações `PlanFileWithVersion(v int)`, `PlanFileWithoutFrames()`, `PlanFileWithFrameCountMismatch()`, `PlanFileWithLatitudeOutOfRange()`, `PlanFileWithoutMarker()`, `PlanFileWithUnknownField()` (campo extra que deve ser ignorado), `TruncatedPlanFile()` e `NotJSONContent()`

**Checkpoint C**: fixtures de MBTiles, GeoTIFF e plano prontos e reutilizáveis.

---

## Phase 3: User Story 1 - Obter o recorte de dados de um plano de câmera (Priority: P1) 🎯 MVP

**Objetivo**: `sobrevoo geodata slice <plano.json>` lê o plano exportado,
verifica a cobertura pelo registro, lê as amostras de elevação e as peças de
mapa base sob a área do voo e imprime um resumo mínimo. Recusa área não
coberta com o relatório de cobertura, e plano inválido ou de versão
desconhecida.

**Escopo desta fase**: sem texto de motivo do nível de detalhe (US2), sem
resumo completo (US3), sem exportação (US4), sem limite de tamanho, guarda e
tratamento de registro corrompido/unidade (US5), sem `elevation` (US6). A
elevação já sai em metros e "sem valor" já é preservado pelo leitor
(FR-008, FR-009) porque o leitor precisa disso para produzir amostras
corretas.

**Teste independente**: `sobrevoo geodata register` de um mapa e de um relevo
que cobrem o trajeto, `sobrevoo plan <gpx> --export plano.json` e
`sobrevoo geodata slice plano.json` imprime área, quantidade de peças e de
amostras; a saída é idêntica em execuções repetidas; sem cobertura, recusa
com código 19.

### Testes da User Story 1

- [ ] T020 [P] [US1] Estender `internal/domain/camera_plan_test.go` com `Test_CameraPlan_Validate` (pacote `domain_test`; usar `builddomain.CameraPlanBuilder`): plano coerente é válido; `frames` vazio → `ErrPlanFileInvalid`; `Summary.FrameCount` ≠ `len(Frames)` ou ≠ `round(duration × frameRate)` → `ErrPlanFileInvalid` com a mensagem `frame_count is 1260 but duration × frame rate is 1230`; `Frames[i].Index != i`; fase fora de `opening`/`following`/`closing`; latitude fora de [-90, 90] e longitude fora de [-180, 180], tanto na câmera quanto no marcador; distância câmera–marcador negativa, NaN ou infinita → `ErrPlanFileInvalid`; a mensagem sempre nomeia o campo e o valor
- [ ] T021 [P] [US1] Criar `internal/domain/camera_plan_area_test.go` (pacote `domain_test`): `Test_CameraPlan_AreaOfInterest` — a área contém todas as posições da câmera e do marcador de todos os quadros (teste de propriedade sobre planos sintéticos: linha, círculo, retorno); com `MarginFactor` 1,0, um plano de um quadro com marcador em `(0,0)` e `camera_to_marker_m = 1000` dá `Δlat ≈ 1000/111320` e `Δlon ≈ 1000/111320`; margem cresce com a distância (quadros de abertura alargam a área); plano cruzando o antimeridiano (marcadores em 179,99° e −179,99°) dá `CrossesAntimeridian == true` e largura pequena (nunca ~360°); latitude alta (85°) não explode a longitude (`cos φ` limitado a `1e-6`; largura total limitada a 360°); latitudes limitadas a [-90, 90]; coordenadas arredondadas a 1e-7°; **SC-009**: o mesmo plano sintético deslocado para o equador, para 60° e para o antimeridiano dá áreas com a mesma altura e larguras compatíveis em metros (tolerância 1%); determinístico (100 chamadas, `assert.Equal`)
- [ ] T022 [P] [US1] Criar `internal/domain/bounding_box_regions_test.go` (pacote `domain_test`): `Test_BoundingBox_Regions` — um registro de mapa base e um de relevo cobrindo tudo ⇒ **uma** região com os dois vencedores; dois relevos adjacentes ⇒ regiões ordenadas de sul a norte e, na mesma linha, de oeste a leste, cada uma com o vencedor certo; dois relevos sobrepostos ⇒ vence o de menor `AreaDegrees` e, em empate, o `RegisteredAt` mais antigo (FR-016 da etapa 2); o `Route` devolvido tem um ponto por região (o centro) e `Route.Coverage` sobre ele reporta `CoverageStatusFull` quando tudo é coberto; com um buraco (relevo cobre só metade), `Route.Coverage` reporta `UncoveredSegments` com `MissingElevation` e as coordenadas dos centros das regiões não cobertas; registros que não intersectam a área não geram regiões extras; área e registros que cruzam o antimeridiano (áreas em 175°–−175°) ⇒ regiões contínuas, ordenadas no espaço de longitude desembrulhado; teste de propriedade: as áreas das regiões somam a área da caixa (partição sem sobreposição) e os centros de todas as regiões estão na área; determinístico
- [ ] T023 [P] [US1] Estender `internal/domain/bounding_box_test.go` com `Test_BoundingBox_Intersects` (caixas disjuntas, tocando na borda, contidas; cruzando o antimeridiano contra uma que não cruza; duas que cruzam) e `Test_BoundingBox_TileRange` (nível 0 ⇒ uma peça `x=0,y=0`; `(lat 0, lon 0)` no nível 1 ⇒ `x=1, y=1`; a fórmula de `research.md` item 6: `x = floor((lon + 180)/360 · 2^z)`, `y = floor((1 − asinh(tan φ)/π)/2 · 2^z)`; latitude acima de 85,0511287798° limitada a essa latitude e índices limitados a `[0, 2^z − 1]`; caixa que cruza o antimeridiano ⇒ **dois** intervalos de `x` (`[xmin..2^z−1]` e `[0..xmax]`); caixa de um único ponto ⇒ uma peça; a quantidade de peças cresce 4× a cada nível para uma caixa grande)
- [ ] T024 [P] [US1] Estender `internal/domain/geo_data_coverage_test.go` com `Test_SelectSource` (o mesmo vencedor que `Route.Coverage` usa — menor `AreaDegrees`, empate pelo mais antigo —; `false` quando nenhum cobre; ponto no meridiano de 180° dentro de registro que cruza o antimeridiano) e um cenário de regressão confirmando que `Route.Coverage` mantém exatamente o comportamento anterior (os testes existentes continuam passando sem alteração)
- [ ] T025 [P] [US1] Criar `internal/domain/elevation_grid_test.go` (pacote `domain_test`; usar `builddomain.ElevationGridBuilder`): `Test_ElevationGridInfo_CellAt` (regra do item 9 de `research.md`: `linha = floor((norte − lat)/Δlat)`, `coluna = floor((lon − oeste)/Δlon)` medida no sentido leste módulo 360; ponto exatamente sobre o limite entre duas células pertence à célula ao **sul/leste**; borda sul/leste externa limitada à última linha/coluna; grade cruzando o antimeridiano), `Test_ElevationGridInfo_Window` (as células cujo **centro** cai na caixa; janela vazia é válida; caixa que cruza o antimeridiano com grade que cruza ⇒ duas janelas retangulares, nunca uma que cruze; borda semiaberta: sul/oeste inclusive, norte/leste exclusive, exceto a borda externa da área que é inclusiva), `Test_ElevationGrid_At` (valor em metros; "sem valor" ⇒ `hasValue == false`, **nunca** zero), `Test_ElevationGrid_NoValueCount`, `Test_ElevationGrid_Range` (mínimo e máximo **só** sobre amostras com valor; `ok == false` quando nenhuma tem valor) e propriedade: para regiões que particionam a área, cada célula da grade cai na janela de exatamente uma região
- [ ] T026 [P] [US1] Criar `internal/domain/tile_test.go` (pacote `domain_test`): `Test_SliceTuning_DetailLevel` — `Ideal` é o menor `z` com `EquatorResolution · cos φ_ref / 2^z ≤ TexelScreenRatio · 2 · d_min · tan(FOV/2) / ReferenceHeightPixels`: `d_min=600 m`, `FOV=45°`, `φ_ref=0` ⇒ `Ideal 18`; mesmo com `φ_ref=60°` ⇒ `Ideal 17`; `Chosen = clamp(Ideal, Min, Max)` (acima do máximo ⇒ `Max`; abaixo do mínimo ⇒ `Min`; dentro ⇒ `Ideal`); `φ_ref` é o menor `|lat|` da área (área cruzando o equador ⇒ 0; área toda no hemisfério sul usa o `|lat|` mais próximo do equador); teste de propriedade: `d_min` menor nunca dá `Ideal` menor; determinístico (100 chamadas)
- [ ] T027 [P] [US1] Criar `internal/infra/outbound/jsonfile/camera_plan_reader_test.go` (pacote `jsonfile_test`; arquivos em `t.TempDir()` a partir de `test/helper/camera_plan_fixture.go`): lê um plano válido e devolve `domain.CameraPlan` com os mesmos quadros (câmera, marcador, distância, fase, tempo), parâmetros e resumo (`DurationMode`, `TimeReference`, `TimeFallbackReason`, trechos suavizados); **round trip**: `CameraPlanExporter.Export` de um plano e `Read` dele devolvem valores iguais dentro da precisão do arquivo (1e-7° e 1e-3); ignora campos desconhecidos; `format_version` ≠ 1 ⇒ `ErrPlanFormatVersionUnsupported` com `found 2, accepted: 1`; `format_version` ausente, JSON inválido, arquivo truncado, `frames[].marker` ausente, `frames[].camera` ausente, `frames[].camera_to_marker_m` ausente, `parameters` ausente ou `summary.frame_count` ausente ⇒ `ErrPlanFileInvalid` com a causa; arquivo inexistente ⇒ erro **sem** sentinela de domínio (E/S, exit 4)
- [ ] T028 [P] [US1] Criar `internal/infra/outbound/basemapreader/mbtiles_base_map_reader_test.go` (pacote `basemapreader_test`; `helper.MBTilesWithTiles`): `Levels` lê `minzoom`/`maxzoom` do `metadata`; sem eles, usa `MIN/MAX(zoom_level)` da tabela `tiles`; `ReadTiles` devolve os bytes exatos das peças pedidas com `ID` em **XYZ** (o arquivo guarda TMS: `tile_row = 2^z − 1 − y`), o `Format` do `metadata` (`png` quando ausente; `jpg`, `webp`, `pbf`), e as peças que o arquivo não contém em `Missing` — peça ausente **não** é erro; nenhuma peça presente ⇒ `Tiles` vazio e `Missing` com todas; o arquivo não é modificado (abrir em modo somente leitura; comparar o hash antes/depois)
- [ ] T029 [P] [US1] Criar `internal/infra/outbound/elevationreader/geotiff_elevation_reader_describe_test.go` (pacote `elevationreader_test`; `helper.GeoTIFFWithSamples`): `Describe` devolve linhas, colunas, canto noroeste, tamanho de célula e `UnitToMeters` a partir **só das tags** (o teste passa um arquivo cujos dados de amostra foram truncados e `Describe` ainda funciona); `PixelIsPoint` desloca a grade de meia célula para noroeste (`PixelIsArea` não desloca); unidade vertical ausente ou 9001 ⇒ `UnitToMeters 1`; 9002 ⇒ `0.3048`; 9003 ⇒ `1200/3937`; qualquer outro valor ⇒ `ErrElevationUnitUnsupported`; CRS projetado, tags faltando ou incoerentes ⇒ `ErrGeoDataContentUnreadable`
- [ ] T030 [P] [US1] Criar `internal/infra/outbound/elevationreader/geotiff_elevation_reader_window_test.go` (pacote `elevationreader_test`): `ReadWindow` sem compressão, em **faixas** e em **peças**, `II` e `MM`, para `uint8`, `int16`, `uint16`, `int32`, `float32` e `float64` — os valores devolvidos em `float32` coincidem com os do fixture, na janela pedida (uma janela no meio da grade, uma que toca a borda, uma de 1×1, uma que atravessa a fronteira entre faixas/peças); só as faixas/peças que intersectam a janela são lidas (o teste conta leituras com um `io.ReaderAt` instrumentado ou compara com um arquivo em que as demais faixas foram corrompidas); janela vazia ⇒ `Values` vazio
- [ ] T031 [P] [US1] Criar `internal/infra/outbound/elevationreader/geotiff_elevation_reader_encoding_test.go` (pacote `elevationreader_test`): `ReadWindow` com compressão Deflate (8 e 32946) e LZW (5), com predictor 1, 2 (horizontal, inteiros) e 3 (ponto flutuante), em faixas e em peças; e, para `GDAL_NODATA` e unidade (FR-008, FR-009): amostra igual ao `GDAL_NODATA` ⇒ "sem valor" (comparado no tipo original, **antes** da conversão de unidade); `float32` NaN ⇒ "sem valor"; sem a tag `GDAL_NODATA`, nenhum valor é "sem valor" (exceto NaN); valor em pés ⇒ multiplicado por `0.3048` (e `1200/3937` para pés US survey); **nunca** um "sem valor" vira zero; compressão JPEG/PackBits, multibanda (`SamplesPerPixel > 1`) e `PlanarConfiguration` separada ⇒ `ErrGeoDataContentUnreadable` com `unsupported encoding: <o quê>`
- [ ] T032 [P] [US1] Estender `internal/application/camera_plan_service_test.go` com `Test_cameraPlanService_Load` (`mockdomain.NewMockCameraPlanReader`): devolve o plano lido e validado; propaga o erro do reader inalterado; chama `plan.Validate()` e propaga `ErrPlanFileInvalid` de um plano incoerente; o construtor passa a receber o `CameraPlanReader`
- [ ] T033 [P] [US1] Criar `internal/application/geo_slice_service_test.go` (pacote `application_test`; mocks de `mockdomain`: `GeoDataRepository`, `FileChecker`, `BaseMapReader`, `ElevationReader`; builders): `Test_geoSliceService_Generate` — plano válido com um mapa base e um relevo cobrindo tudo ⇒ chama `Levels` e `Describe` de cada registro, calcula o nível, chama `ReadTiles` (com as `TileID` de `BoundingBox.TileRange` no nível escolhido) e `ReadWindow` (com a janela da região) e devolve `GeoSlice` com um `TileSet` e uma `ElevationGrid`; registro de mapa base cujo arquivo não existe (`FileChecker`) é ignorado; área sem relevo ⇒ devolve `AreaNotCoveredError` (`errors.Is(err, ErrAreaNotCovered)`, `Report` com o subtrecho e `MissingElevation`) e o gomock **não** vê chamada alguma a `Levels`/`Describe`/`ReadTiles`/`ReadWindow`; dois relevos adjacentes ⇒ duas `ElevationGrid`, uma por região, na ordem das regiões; dois mapas base ⇒ um `TileSet` por registro, cada um com o nível limitado ao **seu** `LevelRange`; determinístico (`Generate` 100 vezes com os mesmos mocks ⇒ 100 recortes `assert.Equal`, SC-002); propaga erro de `List`, de `Levels`, de `Describe`, de `ReadTiles` e de `ReadWindow` sem alterar
- [ ] T034 [P] [US1] Criar `internal/infra/inbound/cli/geodata_slice_test.go` (pacote `cli_test`; `mockapplication.NewMockCameraPlanService` e `NewMockGeoSliceService`): sem argumento ou com 2 ⇒ erro de uso (exit 2); `--overwrite` sem `--export` ⇒ erro de uso (exit 2); caminho ⇒ chama `Load(path)` e depois `Generate(plan)`, e imprime `Area:`, `Map tiles: <n> present` e `Elevation samples: <n>` (formato completo na US3); erro de `Load` sobe inalterado; `AreaNotCoveredError` de `Generate` sobe e `ExitCode` devolve 19; `ErrPlanFileInvalid` ⇒ 17 e `ErrPlanFormatVersionUnsupported` ⇒ 18; nenhum resumo é impresso quando há erro

### Implementação da User Story 1

- [ ] T035 [US1] Editar `internal/domain/camera_plan.go`: `func (c CameraPlan) Validate() error` (receiver `c`; regras de T020, mensagens `fmt.Errorf("%w: ...", ErrPlanFileInvalid)` que nomeiam o campo e o valor) e `func (c CameraPlan) AreaOfInterest(tuning SliceTuning) BoundingBox` (receiver `c`; `research.md` item 2: por quadro, quadrado de meio-lado `tuning.MarginFactor × CameraToMarkerDistance` em torno do marcador; `Δlat = m / MetersPerDegree`, `Δlon = m / (MetersPerDegree · max(cos φ, 1e-6))`; longitudes dos marcadores desembrulhadas somando deltas consecutivos, como `Route.BoundingBox`; largura total limitada a 360°; latitudes limitadas a [-90, 90]; resultado normalizado com `CrossesAntimeridian` e arredondado a 1e-7°) (depende de T020, T021)
- [ ] T036 [US1] Editar `internal/domain/bounding_box.go` (receiver `b`): `Intersects(other BoundingBox) bool` (mesmo tratamento de antimeridiano de `Contains`), `TileRange(level int) []TileRange` (fórmula e limites de `research.md` item 6; dois intervalos de `x` quando cruza o antimeridiano) e `Regions(baseMaps, elevations []GeoDataSource) ([]SliceRegion, Route)` (decomposição por compressão de coordenadas, `research.md` item 3: limites da área e das caixas dos candidatos que intersectam, no espaço de longitude desembrulhado a partir do limite oeste; regiões de sul a norte e de oeste a leste; vencedores por `SelectSource`; devolve também o `Route` com o centro de cada região, na mesma ordem); editar `internal/domain/geo_data_coverage.go` exportando `SelectSource(sources []GeoDataSource, lat, lon float64) (GeoDataSource, bool)` como envoltório de `pickCoverageWinner` **sem alterar a lógica** (`Route.Coverage` passa a usá-la) (depende de T022, T023, T024)
- [ ] T037 [US1] Editar `internal/domain/elevation_grid.go` (receivers `i` para `ElevationGridInfo`, `g` para `ElevationGrid`): `ElevationGridInfo.CellAt(lat, lon float64) (row, col int)` e `Window(box BoundingBox) []GridWindow` (regras de T025; devolve uma janela, ou duas quando a caixa e a grade cruzam o antimeridiano), `ElevationGrid.At(row, col int) (meters float64, hasValue bool)`, `Rows()`, `Cols()`, `NoValueCount() int` e `Range() (min, max float64, ok bool)`; "sem valor" é NaN internamente e nunca escapa do tipo (depende de T025)
- [ ] T038 [US1] Editar `internal/domain/tile.go` (receiver `t` para `SliceTuning`, definido em `geo_slice.go`): `SliceTuning.DetailLevel(minCameraDistance, verticalFOVDegrees float64, area BoundingBox, offered LevelRange) DetailLevel` — `Ideal` = menor `z` com `EquatorResolution · cos φ_ref / 2^z ≤ TexelScreenRatio · 2 · minCameraDistance · tan(FOV/2) / ReferenceHeightPixels`, `φ_ref` = menor `|lat|` dentro da área (0 se a área cruza o equador), `Chosen = clamp(Ideal, offered.Min, offered.Max)`, `Min`/`Max` = `offered`; **`Reason` e `Explanation` ficam vazios** nesta fase (US2) (depende de T026)
- [ ] T039 [US1] Criar `internal/infra/outbound/jsonfile/camera_plan_reader.go`: `CameraPlanReader` e `NewCameraPlanReader()` (nome da porta; pacote de tecnologia; receiver `r`, `Read(path string) (domain.CameraPlan, error)`); decodifica `format_version`, `parameters`, `summary` e `frames` do formato de `plan-file.md`, ignora campos desconhecidos, exige os campos obrigatórios de T027, recusa `format_version` ≠ 1 com `ErrPlanFormatVersionUnsupported` (mensagem `found <n>, accepted: 1`) e devolve `ErrPlanFileInvalid` embrulhado para JSON inválido/campos ausentes; erro de abrir/ler o arquivo sobe embrulhado **sem** sentinela; monta o plano com `domain.NewCameraPlan(...)` (resumo recalculado a partir dos quadros; `DurationMode`, `TimeReference` e `TimeFallbackReason` vêm do arquivo) (depende de T027, T011)
- [ ] T040 [US1] Criar `internal/infra/outbound/basemapreader/mbtiles_base_map_reader.go`: `MBTiles` e `NewMBTiles()` (arquivo `mbtiles_base_map_reader.go`; receiver `r`); `Levels` e `ReadTiles` conforme T028 (`modernc.org/sqlite` em modo somente leitura `?mode=ro`, uma abertura por chamada, uma consulta preparada por peça, conversão XYZ↔TMS somente aqui, bytes crus sem decodificar imagem, `Format` de `metadata.format`, `png` por padrão); qualquer falha de SQLite/tabela ausente/arquivo truncado ⇒ `domain.ErrGeoDataContentUnreadable` embrulhado com a causa (depende de T028, T008)
- [ ] T041 [US1] Criar `internal/infra/outbound/elevationreader/geotiff_elevation_reader.go`: `GeoTIFF` e `NewGeoTIFF()` (arquivo `geotiff_elevation_reader.go`; receiver `r`); `Describe` (só tags: largura, altura, escala de pixel, ponto de amarração, `BitsPerSample`, `SampleFormat`, `Compression`, `Predictor`, layout de faixas/peças, `GDAL_NODATA`, `VerticalUnitsGeoKey`, `GTRasterTypeGeoKey`) e `ReadWindow` **sem compressão** para faixas e peças, `II`/`MM`, inteiros 8/16/32 e ponto flutuante 32/64, decodificando só as faixas/peças que intersectam a janela; reaproveita a leitura de IFD de `geodatainspector/geotiff.go` **copiando** o mínimo necessário (o inspetor não é alterado nesta etapa; `research.md`, riscos); erros ⇒ `ErrGeoDataContentUnreadable` (depende de T029, T030, T009)
- [ ] T042 [US1] Editar `internal/infra/outbound/elevationreader/geotiff_elevation_reader.go`: compressão Deflate (8 e 32946, `compress/zlib`) e LZW (5, `golang.org/x/image/tiff/lzw`; rodar `go get golang.org/x/image@latest` e `go mod tidy`), predictors 1, 2 e 3 (inverso do predictor horizontal para inteiros e do predictor de ponto flutuante) e rejeição de qualquer outra combinação com `unsupported encoding: <o quê>` (depende de T031, T041)
- [ ] T043 [US1] Editar `internal/infra/outbound/elevationreader/geotiff_elevation_reader.go`: valor de "sem dado" (`GDAL_NODATA` comparado no tipo original antes da conversão; NaN em ponto flutuante ⇒ NaN interno de `float32`), conversão de unidade para metros (`UnitToMeters` de `Describe`; `9001`/ausente = 1, `9002` = 0,3048, `9003` = 1200/3937; outro código ⇒ `domain.ErrElevationUnitUnsupported` **em `Describe`**, antes de ler amostras) e `PixelIsPoint` (deslocamento de meia célula), conforme T029 e T031 (depende de T042)
- [ ] T044 [US1] Editar `internal/application/camera_plan_service.go`: acrescentar `Load(path string) (domain.CameraPlan, error)` à interface `CameraPlanService` (com comentário) e o campo `reader domain.CameraPlanReader`; `NewCameraPlanService(trackService, exporter, reader, defaultLevel, tuning)`; `Load` chama `reader.Read(path)`, depois `plan.Validate()`, devolvendo o erro de qualquer um sem alterar (depende de T032, T035, T011)
- [ ] T045 [US1] Criar `internal/application/geo_slice_service.go`: diretiva `//go:generate go run go.uber.org/mock/mockgen -destination mockapplication/geo_slice_service.go -package mockapplication . GeoSliceService` logo acima da interface `GeoSliceService` (`Generate(plan domain.CameraPlan) (domain.GeoSlice, error)`; `Export` entra na US4), struct `geoSliceService` (receiver `s`; campos `repository domain.GeoDataRepository`, `fileChecker domain.FileChecker`, `baseMapReader`, `elevationReader`, `sliceTuning domain.SliceTuning`, `cameraTuning domain.CameraTuning`) e `NewGeoSliceService(...)`; `Generate` só orquestra: `area := plan.AreaOfInterest(tuning)` → `repository.List()` → separa os registros disponíveis por tipo com `fileChecker.Exists` (mesma lógica de `GeoDataService.partitionAvailableSources` — extrair para um método de domínio compartilhado **ou** repetir o laço mínimo, sem regra nova) → `area.Regions(baseMaps, elevations)` → `Route.Coverage`; se `Status != CoverageStatusFull` ⇒ `&domain.AreaNotCoveredError{Report: report}`; senão, por registro de mapa base usado: `Levels`, `SliceTuning.DetailLevel(plan.Summary.MinCameraDistance, cameraTuning.OverviewVerticalFOVDegrees, area, levels)`, `ReadTiles`; por região com relevo: `Describe`, `info.Window(region.Box)`, `ReadWindow`, `domain.NewElevationGrid(...)`; ordena `TileSets` por `(nome do registro, nível)`; `domain.NewGeoSlice(area, tileSets, grids)`; nenhuma regra de negócio na camada (depende de T033, T035–T038, T040–T043)
- [ ] T046 [US1] Rodar `make generate` para gerar `internal/application/mockapplication/geo_slice_service.go` e regenerar `mockapplication/camera_plan_service.go` (novo método `Load`); nenhum outro diff (depende de T044, T045)
- [ ] T047 [US1] Criar `internal/infra/inbound/cli/geodata_slice.go`: `NewGeoDataSliceCommand(cameraPlanService application.CameraPlanService, geoSliceService application.GeoSliceService)` — `Use: "slice <plan-file>"`, `SilenceErrors`/`SilenceUsage` como os demais, `Args` exigindo exatamente um argumento (erro de uso via `newUsageError`), `SetFlagErrorFunc` para erros de flag; `RunE`: `Load` → `Generate`; imprime, por ora, `Area: lat <min> to <max>, lon <min> to <max>` (com `(crosses the antimeridian)` quando aplicável), `Map tiles: <n> present` e `Elevation samples: <n>` (formato completo na US3); flags `--export` e `--overwrite` só entram na US4 (depende de T034, T046)
- [ ] T048 [US1] Editar `cmd/sobrevoo/main.go`: criar `basemapreader.NewMBTiles()`, `elevationreader.NewGeoTIFF()`, `jsonfile.NewCameraPlanReader()`; passar o reader a `application.NewCameraPlanService(trackService, cameraPlanExporter, cameraPlanReader, ...)`; `application.NewGeoSliceService(geoDataRepository, geoDataFileChecker, baseMapReader, elevationReader, domainSliceTuning(cfg.SliceTuning), domainCameraTuning(cfg.CameraTuning))`; `geoDataCommand.AddCommand(cli.NewGeoDataSliceCommand(cameraPlanService, geoSliceService))`; rodar `make build`, `make test` e `make lint` (depende de T014, T044, T045, T047)

**Checkpoint US1**: `make test`, `make lint` e `make generate` verdes;
`sobrevoo geodata slice plano.json` funciona sobre dados sintéticos
registrados e recusa área não coberta (19), plano inválido (17) e versão
desconhecida (18). MVP.

---

## Phase 4: User Story 2 - Nível de detalhe do mapa base explicável (Priority: P2)

**Objetivo**: o nível de detalhe é determinístico, monotônico na distância,
respeita o intervalo da fonte e vem com um motivo legível no resumo.

**Teste independente**: planos do mesmo trajeto com `--distance low` e
`--distance high` sobre o mesmo mapa base dão níveis com `low ≥ high`, dentro
do intervalo do arquivo, e cada `Base map detail` traz o motivo.

### Testes da User Story 2

- [ ] T049 [P] [US2] Estender `internal/domain/tile_test.go` com `Test_SliceTuning_DetailLevel_Reason`: `Reason` diz exatamente `within the source's range` quando `Chosen == Ideal`, `above the source's maximum level` quando `Ideal > Max` e `below the source's minimum level` quando `Ideal < Min`; e `Explanation` traz a distância mínima da câmera, a latitude de referência e a resolução exigida em metros por pixel (`nearest camera distance 300.0 m at latitude 23.48, requiring 0.23 m/px per screen pixel`, formato de `contracts/cli.md`); propriedade de monotonicidade sobre uma varredura de distâncias (300 a 6000 m): `Ideal` nunca aumenta quando a distância aumenta (SC-005); mesmo `DetailLevel` em 100 chamadas; dois registros com faixas diferentes dão `Chosen` diferentes para o mesmo `Ideal`
- [ ] T050 [P] [US2] Estender `internal/infra/inbound/cli/geodata_slice_test.go` com o formato de `Base map detail`: uma linha `Base map detail: level <chosen> (ideal <ideal>, source offers <min>-<max>; <Reason>)` mais a linha `<Explanation>` indentada por registro de mapa base usado, em ordem de nome; com dois registros, duas duplas de linhas; o texto exato de `contracts/cli.md`

### Implementação da User Story 2

- [ ] T051 [US2] Editar `internal/domain/tile.go`: `SliceTuning.DetailLevel` preenche `DetailLevel.Reason` e `DetailLevel.Explanation` (textos de T049) (depende de T049)
- [ ] T052 [US2] Editar `internal/infra/inbound/cli/geodata_slice.go`: imprimir as linhas `Base map detail` por `TileSet` (depende de T050, T051)

**Checkpoint US2**: níveis explicados e monotônicos; `quickstart.md` item 4.

---

## Phase 5: User Story 3 - Ver o resumo do recorte (Priority: P3)

**Objetivo**: o resumo completo do recorte, calculado pelo domínio a partir do
conteúdo e impresso no formato de `contracts/cli.md`.

**Teste independente**: o resumo de um recorte conhecido coincide com o que
se calcula a partir do conteúdo (contagens, faixa de elevação, tamanho).

### Testes da User Story 3

- [ ] T053 [P] [US3] Criar `internal/domain/geo_slice_test.go` (pacote `domain_test`; usar os builders): `Test_NewGeoSlice_Summary` — `TileCount` e `MissingTileCount` somam sobre todos os `TileSet`; `SampleCount` = soma de `rows × cols`; `NoValueSampleCount` = soma de `NoValueCount`; `MinElevation`/`MaxElevation` calculados **só** sobre amostras com valor e `HasElevationRange == true`; nenhuma amostra com valor ⇒ `HasElevationRange == false` (sem faixa inventada); `SizeBytes` = soma de `len(Tile.Data)` + `BytesPerElevationSample × SampleCount` (`4 × amostras`); `Sources` lista cada registro usado **uma vez**, ordenado por nome, com o `DetailLevel` só para mapa base; `Area` copiada; zero peças presentes e várias ausentes é um recorte válido; o resumo sempre coincide com o conteúdo (propriedade: construir recortes aleatórios pelos builders e recalcular por fora)
- [ ] T054 [P] [US3] Estender `internal/infra/inbound/cli/geodata_slice_test.go` com o formato completo de `contracts/cli.md` (`mockapplication`): linhas `Area`, `Base map detail` (US2), `Map tiles: <n> present, <m> missing` e, com ausentes, uma linha `missing: <registro> level <z> x=<x> y=<y>` por peça, **limitadas a 20** seguidas de `... and <k> more (all listed in the exported file)`; sem ausentes `Map tiles: <n> present`; zero presentes `Map tiles: 0 present, <m> missing (no imagery in this slice)`; `Elevation samples: <n>` com `(<k> without value)` **só** quando `k > 0`; `Elevation range: <min> m - <max> m` e, sem faixa, `Elevation range: none (no sample has a value)`; `Sources:` com cada registro (`<nome> (<tipo>, <formato>)`) ordenado por nome; `Size:` em `KiB`/`MiB`/`GiB` com uma casa decimal (`512 B` abaixo de 1 KiB); código de saída 0 mesmo com ausentes ou sem valor

### Implementação da User Story 3

- [ ] T055 [US3] Editar `internal/domain/geo_slice.go`: `NewGeoSlice` passa a calcular `SliceSummary` a partir do conteúdo, como `NewCameraPlan` (T053); receiver `g` nos métodos auxiliares; registros ordenados por nome (depende de T053)
- [ ] T056 [US3] Editar `internal/infra/inbound/cli/geodata_slice.go`: `formatSliceSummary(slice domain.GeoSlice) string` com o formato completo de `contracts/cli.md` (helper `formatBytes` para `B/KiB/MiB/GiB`; limite de 20 ausentes; `Elevation range` em metros com uma casa) substituindo a saída mínima de T047 (depende de T054, T055)

**Checkpoint US3**: `quickstart.md` item 1 (resumo completo).

---

## Phase 6: User Story 4 - Exportar o recorte (Priority: P4)

**Objetivo**: `--export` grava o recorte num único arquivo ZIP
determinístico (`contracts/slice-file.md`), sem sobrescrita por padrão e sem
resultado parcial.

**Teste independente**: exportar, reler o ZIP e conferir que peças,
amostras, ausências e resumo coincidem; duas exportações são idênticas byte a
byte.

### Testes da User Story 4

- [ ] T057 [P] [US4] Criar `internal/infra/outbound/zipfile/geo_slice_exporter_test.go` (pacote `zipfile_test`; `archive/zip` para reler; `builddomain.GeoSliceBuilder`): escreve `manifest.json` como **primeira** entrada, depois `elevation/NNN.f32` em ordem numérica e `tiles/<índice-do-registro>/<z>/<x>/<y>.<ext>` em ordem `(registro, z, x, y)`; todas as entradas em método *store*, com data de modificação fixa (`1980-01-01T00:00:00Z`) e sem comentário; o `manifest.json` traz os campos, a ordem e os formatos de `contracts/slice-file.md` (`format_version` 1; `area`; `summary` com `elevation_m` `null` quando não há faixa; `sources[]` ordenados por nome com `path` como registrado; `base_map[]` com `level {ideal, chosen, min, max, reason}`, `tiles[]` compactos com `bytes`, `missing[]` sempre presente e `[]` quando vazio; `elevation[]` com `north_lat`, `west_lon`, `cell_lat`, `cell_lon`, `no_value_count`); os bytes das peças no ZIP são **idênticos** aos de `Tile.Data`; cada `.f32` tem `rows × cols × 4` bytes, `float32` little-endian, linhas de norte a sul, "sem valor" = `0x7FC00000` e mais nenhum NaN; garantias 1 a 5 de `slice-file.md` verificadas; **round trip (SC-012)**: reler o ZIP reconstrói peças, amostras, ausências e resumo iguais aos do recorte exportado; **determinismo (SC-002)**: duas exportações do mesmo recorte ⇒ arquivos idênticos byte a byte; sem `overwrite`, destino existente ⇒ `errors.Is(err, domain.ErrSliceDestinationExists)` (mensagem com `use --overwrite to replace it`) e arquivo intacto; com `overwrite`, o destino passa a conter integralmente o novo ZIP; diretório inexistente ⇒ `ErrSliceDestinationInvalid`; nenhum `.sobrevoo-*.tmp` sobra em nenhum erro
- [ ] T058 [P] [US4] Estender `internal/application/geo_slice_service_test.go` com `Test_geoSliceService_Export` (`mockdomain.NewMockGeoSliceExporter`): delega `slice`, `path` e `overwrite` ao exporter e devolve o erro dele sem alterar
- [ ] T059 [P] [US4] Estender `internal/infra/inbound/cli/geodata_slice_test.go` com `--export`/`--overwrite`: `--export <caminho>` chama `Export(slice, caminho, false)` **antes** de imprimir qualquer coisa e termina com `Slice written to <caminho>`; `--overwrite` passa `true`; erro de `Export` sobe sem imprimir resumo (`ExitCode` 23 para `ErrSliceDestinationExists`, 24 para `ErrSliceDestinationInvalid`); `--overwrite` sem `--export` continua erro de uso (exit 2)

### Implementação da User Story 4

- [ ] T060 [US4] Criar `internal/infra/outbound/zipfile/geo_slice_exporter.go`: `GeoSliceExporter` e `NewGeoSliceExporter()` (pacote de tecnologia: construtor nomeado pela porta; receiver `e`); estruturas de manifesto com números em precisão fixa e sem zeros à direita (mesma ideia do tipo `number` de `jsonfile/camera_plan_file.go` — copiar o mínimo, sem importar `jsonfile`); escreve o ZIP com `archive/zip` (`CreateHeader` com `Method: zip.Store`, `Modified` fixo, sem `Extra`) dentro de `atomicfile.Publish`, traduzindo `atomicfile.ErrExists` para `domain.ErrSliceDestinationExists` (`"%w: %s (use --overwrite to replace it)"`) e `atomicfile.ErrInvalid` para `domain.ErrSliceDestinationInvalid`; ordem das entradas e formato de `.f32` conforme T057 (depende de T057, T003, T010)
- [ ] T061 [US4] Editar `internal/application/geo_slice_service.go`: acrescentar `Export(slice domain.GeoSlice, path string, overwrite bool) error` à interface (com comentário) e o campo `exporter domain.GeoSliceExporter` (`NewGeoSliceService` passa a recebê-lo); rodar `make generate` para regenerar `mockapplication/geo_slice_service.go` (depende de T058, T060)
- [ ] T062 [US4] Editar `internal/infra/inbound/cli/geodata_slice.go`: flags `--export` e `--overwrite` (textos de ajuda de `contracts/cli.md`), exportar antes de imprimir e acrescentar `Slice written to <path>`; editar `cmd/sobrevoo/main.go` criando `zipfile.NewGeoSliceExporter()` e passando-o ao `NewGeoSliceService` (depende de T059, T061)

**Checkpoint US4**: `quickstart.md` itens 1 (determinismo) e 8.

---

## Phase 7: User Story 5 - Tolerar peça ausente e recusar recorte grande demais (Priority: P5)

**Objetivo**: peça ausente não interrompe o recorte; recorte acima do limite é
recusado **antes** de ler conteúdo; registro corrompido e unidade não
suportada são recusados com mensagem clara que nomeia o registro.

**Teste independente**: mapa base com peças removidas ⇒ recorte concluído
com as posições faltantes; plano cuja área exigiria mais que o limite ⇒
código 20 em menos de 5 s, sem `ReadTiles`/`ReadWindow`; limite exato aceito,
`+1` byte recusado.

### Testes da User Story 5

- [ ] T063 [P] [US5] Estender `internal/domain/geo_slice_test.go` com `Test_SliceTuning_Estimate` e `Test_SliceTuning_EnsureFits` (`SliceTuning` definido em T010): `Estimate(tileCount, sampleCount int64) int64 = tileCount × EstimatedTileBytes + sampleCount × BytesPerElevationSample`; exatamente `MaxSizeBytes` ⇒ aceito; `MaxSizeBytes + 1` ⇒ `errors.Is(err, ErrSliceTooLarge)` (comparação `>` estrita); a mensagem traz o tamanho, o limite (`256.0 MiB`) e a dica `try a higher --distance in the plan, or a shorter track`; o mesmo `EnsureFits` serve à guarda de bytes reais
- [ ] T064 [P] [US5] Estender `internal/application/geo_slice_service_test.go` com cenários de robustez: peças ausentes (`ReadTiles` devolve `Missing`) ⇒ o recorte é produzido com `TileSet.Missing` preenchido e nenhum erro; **todas** as peças ausentes ⇒ ainda um recorte válido; estimativa acima do limite (mock de `Describe` com grade enorme) ⇒ `ErrSliceTooLarge` e o gomock **não** vê `ReadTiles` nem `ReadWindow`; a estimativa usa a contagem de `TileRange` no nível **escolhido** e o número de amostras das janelas; guarda durante a leitura: `ReadTiles` devolvendo bytes que somados passam do limite ⇒ `ErrSliceTooLarge`; `Levels`/`ReadTiles`/`Describe`/`ReadWindow` devolvendo `ErrGeoDataContentUnreadable` ⇒ sobe com o **nome do registro** na mensagem (`errors.Is` preservado) e nenhum recorte parcial; `Describe` devolvendo `ErrElevationUnitUnsupported` ⇒ sobe com o nome do registro
- [ ] T065 [P] [US5] Estender `internal/infra/outbound/basemapreader/mbtiles_base_map_reader_test.go` e `internal/infra/outbound/elevationreader/geotiff_elevation_reader_window_test.go` com corrupção: `helper.CorruptMBTiles()` ⇒ `Levels` e `ReadTiles` devolvem `ErrGeoDataContentUnreadable`; GeoTIFF truncado (`helper.Truncated`) ⇒ `ReadWindow` devolve `ErrGeoDataContentUnreadable`, nunca um valor parcial; arquivo que desapareceu depois da checagem de disponibilidade ⇒ `ErrGeoDataContentUnreadable` (não `ErrDataFileNotFound`)

### Implementação da User Story 5

- [ ] T066 [US5] Editar `internal/domain/geo_slice.go` (receiver `t`): `SliceTuning.Estimate(tileCount, sampleCount int64) int64` e `SliceTuning.EnsureFits(bytes int64) error` (mensagem em inglês com tamanho, limite em `KiB/MiB` e a dica; `fmt.Errorf("%w: ...", ErrSliceTooLarge)`) (depende de T063)
- [ ] T067 [US5] Editar `internal/application/geo_slice_service.go`: antes de qualquer `ReadTiles`/`ReadWindow`, calcular a contagem de peças (`len` de `TileRange` por registro no nível escolhido) e de amostras (soma das janelas) e chamar `SliceTuning.EnsureFits(Estimate(...))`; durante a leitura, acumular `len(Tile.Data) + 4 × amostras` e chamar `EnsureFits` a cada registro/região lido; embrulhar erros dos readers com o nome do registro (`fmt.Errorf("source %q: %w", source.Name, err)`, preservando `errors.Is`); a mensagem de `ErrSliceTooLarge` inclui o nível efetivo e a extensão da área (depende de T064, T066, T045)
- [ ] T068 [US5] Ajustar `internal/infra/outbound/basemapreader/mbtiles_base_map_reader.go` e `internal/infra/outbound/elevationreader/geotiff_elevation_reader.go` até T065 passar: qualquer falha de E/S, SQLite ou decodificação depois da abertura vira `ErrGeoDataContentUnreadable` embrulhado com a causa; nenhum `panic` por índice fora do intervalo em arquivo truncado (validar tamanhos e deslocamentos antes de ler) (depende de T065)

**Checkpoint US5**: `quickstart.md` itens 6 e 7.

---

## Phase 8: User Story 6 - Consultar a elevação de uma coordenada (Priority: P6)

**Objetivo**: `sobrevoo geodata elevation --lat --lon` informa a elevação em
metros, "sem valor" ou "sem cobertura", pela mesma regra do recorte.

**Teste independente**: relevo de exemplo com valores conhecidos; consulta de
uma célula com valor, uma sem valor e um ponto fora da cobertura.

### Testes da User Story 6

- [ ] T069 [P] [US6] Criar `internal/domain/coordinate_test.go` (pacote `domain_test`): `Test_NewCoordinate` — latitude fora de -90 a 90 e longitude fora de -180 a 180 ⇒ `errors.Is(err, ErrInvalidCoordinate)` com mensagem `latitude 91 is out of range, must be between -90 and 90` (e o equivalente para longitude); NaN e infinito ⇒ `ErrInvalidCoordinate`; -90, 90, -180 e 180 são aceitos; longitude `180` e `-180` normalizam para a **mesma** posição (`-180`, faixa `[-180, 180)`); coordenada válida preservada
- [ ] T070 [P] [US6] Estender `internal/application/geo_data_service_test.go` (o construtor passa a receber `ElevationReader`; ajustar todos os cenários existentes ao novo construtor, sem mudar expectativas) com `Test_geoDataService_ElevationAt` (`mockdomain`: `GeoDataRepository`, `FileChecker`, `ElevationReader`): escolhe o relevo vencedor entre os **disponíveis** (menor área; empate: o mais antigo; arquivo ausente ignorado); `Describe` + `CellAt` + `ReadWindow` de **1×1** ⇒ `ElevationReading` com `Meters`, `Source`, `Row` e `Col` da grade de origem; célula "sem valor" ⇒ `HasValue == false` (sem erro, sem zero); nenhum relevo cobre ⇒ `ErrElevationNotCovered`; `--lon 180` e `-180` ⇒ a mesma `ElevationReading`; **consistência (FR-018, SC-006)**: para o mesmo ponto, o valor de `ElevationAt` coincide com o valor de `ElevationGrid.At` de um `GeoSliceService.Generate` sobre um plano cujo recorte contém a célula (mesmos mocks de `ElevationReader`); erros de `Describe`/`ReadWindow` sobem com o nome do registro; unidade em pés ⇒ `Meters` já convertido pela porta
- [ ] T071 [P] [US6] Criar `internal/infra/inbound/cli/geodata_elevation_test.go` (pacote `cli_test`; `mockapplication.NewMockGeoDataService`): `--lat` e `--lon` obrigatórios (ausente ⇒ erro de uso, exit 2), não numérico ou NaN/inf ⇒ erro de uso (exit 2), valores negativos aceitos (`--lat -23.5505`); chama `ElevationAt(lat, lon)`; saída com valor: `Elevation: 760.0 m` e `Source: srtm-sp (cell row 412, column 88)`; sem valor: `Elevation: no value (the file has no data for this point)` e a linha `Source`; `ErrElevationNotCovered` ⇒ 25 com sugestão de `geodata list`; `ErrInvalidCoordinate` ⇒ 26; `ErrGeoDataContentUnreadable` ⇒ 21; `ErrElevationUnitUnsupported` ⇒ 22; nada é impresso em `stdout` quando há erro

### Implementação da User Story 6

- [ ] T072 [US6] Criar `internal/domain/coordinate.go`: `Coordinate{Latitude, Longitude float64}` e `NewCoordinate(latitude, longitude float64) (Coordinate, error)` (validação de T069; longitude normalizada para `[-180, 180)`; `fmt.Errorf("%w: ...", ErrInvalidCoordinate)`); arquivo acrescentado ao layout de `plan.md` (depende de T069)
- [ ] T073 [US6] Editar `internal/application/geo_data_service.go`: `ElevationAt(latitude, longitude float64) (domain.ElevationReading, error)` na interface (com comentário) e o campo `elevationReader domain.ElevationReader` (`NewGeoDataService(repository, inspector, fileChecker, trackService, elevationReader)`); implementação: `domain.NewCoordinate` → `repository.List()` → só relevos com arquivo presente → `domain.SelectSource` (`ErrElevationNotCovered` se não houver) → `Describe` → `info.CellAt` → `ReadWindow` 1×1 → `ElevationReading`; reaproveitar `partitionAvailableSources`; rodar `make generate` para regenerar `mockapplication/geo_data_service.go` (depende de T070, T072, T036, T037)
- [ ] T074 [US6] Criar `internal/infra/inbound/cli/geodata_elevation.go`: `NewGeoDataElevationCommand(geoDataService application.GeoDataService)` — `Use: "elevation"`, flags `--lat` e `--lon` (texto, parseadas com o mesmo `parseFiniteNumber` de `plan.go`; obrigatórias), formato de saída de `contracts/cli.md` (uma casa decimal em metros; `Source: <nome> (cell row <r>, column <c>)`); erros por `newUsageError`/sentinelas (depende de T071, T073)
- [ ] T075 [US6] Editar `cmd/sobrevoo/main.go`: passar `elevationReader` a `NewGeoDataService` e `geoDataCommand.AddCommand(cli.NewGeoDataElevationCommand(geoDataService))`; `make build`, `make test`, `make lint` (depende de T074)

**Checkpoint US6**: `quickstart.md` item 5 e 9 (antimeridiano).

---

## Phase 9: Polish & Cross-Cutting Concerns

- [ ] T076 [P] Criar `test/samples/main.go` (ferramenta de desenvolvimento: `go run ./test/samples --out <dir>`): usando `test/helper`, grava os arquivos de `quickstart.md` ("Pré-requisitos"): `mapa-sp.mbtiles` (níveis 10–16 sobre a área de `specs/003-camera-path-planning/amostras/pedalada.gpx`, com **3 peças removidas** no nível 16), `relevo-sp.tif` (`int16`, Deflate, metros, com um bloco de "sem dado" de posição conhecida), `relevo-pes.tif` (`float32`, sem compressão, pés), `relevo-projetado.tif` (unidade vertical 9999), `mapa-corrompido.mbtiles` (`CorruptMBTiles` com `bounds` sobre a mesma área), `mapa-antimeridiano.mbtiles` e `relevo-antimeridiano.tif` (cruzando 180°), `mapa-polar.mbtiles` e `relevo-polar.tif` (acima de 80°); comentário no topo listando, para cada arquivo, as coordenadas e os valores esperados de células conhecidas (usados em `quickstart.md` item 5); os binários **não** são versionados (adicionar `specs/004-geo-data-slice/amostras/` ao `.gitignore`)
- [ ] T077 Editar `specs/004-geo-data-slice/quickstart.md`: no item 8 trocar o padrão do resíduo para `/tmp/.sobrevoo-*.tmp` (o `atomicfile` usa `.sobrevoo-*.tmp`) e preencher, com os valores reais de `test/samples`, os marcadores `<lat-conhecida>`, `<lon-conhecida>`, `<valor>` e demais placeholders dos itens 5 e 9 (depende de T076)
- [ ] T078 [P] Atualizar `CLAUDE.md` (prosa em português; blocos de código e identificadores em inglês): "Projeto" (quarta etapa: `geodata slice` e `geodata elevation`), "Comandos" (exemplos de `slice`/`elevation`), "Arquitetura" (novas entidades de domínio, portas `CameraPlanReader`/`BaseMapReader`/`ElevationReader`/`GeoSliceExporter`, serviço `GeoSliceService`, métodos novos `CameraPlanService.Load` e `GeoDataService.ElevationAt`, adapters `basemapreader`, `elevationreader`, `zipfile`, `atomicfile`, `jsonfile.CameraPlanReader`, `SliceTuning` em `config`) e a lista de contratos de CLI (`specs/004-geo-data-slice/contracts/cli.md`)
- [ ] T079 [P] Atualizar `README.md` com a quarta etapa (o que faz, comandos, o formato do recorte e o limite de 256 MiB)
- [ ] T080 Percorrer `quickstart.md` de ponta a ponta com o binário real (`make build`; `go run ./test/samples`), registrando qualquer desvio; medir SC-001 (`time sobrevoo geodata slice` de um voo típico, < 30 s) e SC-010 (recusa por tamanho em < 5 s); repetir uma vez com dados reais (item 10) — MBTiles e DEM GeoTIFF do próprio usuário — e confirmar que nenhum arquivo de dado mudou (`shasum` antes e depois) e que não houve conexão de rede
- [ ] T081 Rodar `gofmt -l .` (sem saída), `make generate` (sem diff), `make lint` e `make test` (`go test ./... -cover`); conferir que os pacotes novos têm cobertura comparável aos existentes e que `go.mod` só ganhou `golang.org/x/image` (`git diff go.mod go.sum`); rodar `/speckit-analyze` e resolver qualquer inconsistência entre `spec.md`, `plan.md` e `tasks.md`

---

## Dependências e Ordem de Execução

### Dependências entre fases

- **Phase 1** → **Phase 2 Parte A** → **Parte B** → **Parte C** → histórias.
- **Parte A** (T002–T005) é sequencial e precisa estar verde antes de
  qualquer outra coisa.
- **Parte B**: T006–T011 e T013 são independentes entre si (`[P]`); T012
  depende de T008–T011; T014 depende de T010 e T013; T015 e T016 dependem dos
  tipos (T008–T010, T013).
- **Parte C**: T017, T018 e T019 são independentes (`[P]`) e só dependem de
  T001.
- **US1** depende de Foundational completo. **US2**, **US3**, **US4** e
  **US5** dependem de US1. **US6** depende só de Foundational e de T036/T037
  (`SelectSource`, `CellAt`) e dos leitores de US1 (T041–T043), podendo ser
  adiantada em relação a US2–US5.
- **US3** estende `geodata_slice.go` e `geo_slice.go` (nunca em paralelo com
  US2/US4/US5 nos mesmos arquivos); **US4** e **US5** estendem
  `geo_slice_service.go` (sequenciais entre si).
- **Polish** depende de todas as histórias desejadas.

### Dentro de cada história

- Testes (T020–T034 etc.) antes da implementação correspondente e devem
  falhar primeiro.
- Domínio (T035–T038) → adapters (T039–T043) → serviços (T044–T046) → CLI e
  composition root (T047–T048).

### Oportunidades de paralelismo

- Fundação: T006, T007, T008, T009, T010, T011, T013 em paralelo; depois
  T014/T015/T016; T017–T019 em paralelo com a Parte B.
- US1, testes: T020–T031 são arquivos diferentes ⇒ todos `[P]`; T032–T034
  (serviços e CLI) também.
- US1, implementação: T035–T038 (domínio) sequenciais entre si só onde o
  mesmo arquivo é editado (`bounding_box.go`, `elevation_grid.go`, `tile.go`,
  `camera_plan.go` são arquivos distintos ⇒ T035, T036, T037 e T038 podem
  rodar em paralelo, embora T045 dependa de todos); T039, T040 e T041
  (adapters distintos) em paralelo; T042 e T043 estendem o mesmo arquivo de
  T041 (sequenciais).
- US2 (T049, T050), US3 (T053, T054), US4 (T057–T059), US5 (T063–T065) e US6
  (T069–T071): os testes de cada história em paralelo entre si.
- Polish: T076, T078 e T079 em paralelo.

---

## Exemplo de Paralelização: User Story 1

```text
# Todos os testes de US1 em paralelo (arquivos diferentes):
T020 camera_plan_test.go            T021 camera_plan_area_test.go
T022 bounding_box_regions_test.go   T023 bounding_box_test.go
T024 geo_data_coverage_test.go      T025 elevation_grid_test.go
T026 tile_test.go                   T027 camera_plan_reader_test.go
T028 mbtiles_base_map_reader_test.go
T029 ..._describe_test.go  T030 ..._window_test.go  T031 ..._encoding_test.go
T032 camera_plan_service_test.go    T033 geo_slice_service_test.go
T034 geodata_slice_test.go

# Depois, o domínio e os três adapters em paralelo:
T035 camera_plan.go   T036 bounding_box.go   T037 elevation_grid.go   T038 tile.go
T039 camera_plan_reader.go   T040 mbtiles_base_map_reader.go   T041 geotiff_elevation_reader.go
```

---

## Estratégia de Implementação

### MVP Primeiro (Phase 1 + 2 + User Story 1)

1. Phase 1 (baseline) e Phase 2 (atomicfile extraído; base; fixtures).
2. Phase 3 (US1): ler o plano, verificar cobertura por região e ler o
   conteúdo → `geodata slice plano.json` com saída mínima.
3. **PARE e valide**: `quickstart.md` itens 1 (sem `--export`), 2 e 3.

### Entrega Incremental

1. Setup + Foundational → `atomicfile`, erros, portas, configuração,
   fixtures prontos.
2. + US1 → recorte funcional e recusas de cobertura/plano (MVP).
3. + US2 → nível de detalhe explicado → item 4.
4. + US3 → resumo completo → item 1 completo.
5. + US4 → exportação em ZIP → itens 1 (determinismo) e 8.
6. + US5 → robustez: peças ausentes, tamanho, registro corrompido → itens 6 e 7.
7. + US6 → consulta de elevação → itens 5 e 9 (pode ser adiantada).
8. Polish → amostras, documentação, dados reais, verificação final.

Cada história agrega valor sem quebrar as anteriores.

---

## Notas

- `[P]` = arquivos diferentes, sem dependência de tarefa incompleta.
- Os valores numéricos de `SliceTuning` são iniciais e ficam concentrados em
  `config.Load()`; ajustá-los é uma edição num só lugar (`research.md`
  item 15).
- `internal/domain/coordinate.go` (T072) não estava na árvore de `plan.md`;
  é o construtor de regra de `ErrInvalidCoordinate` (Princípio IX: construtor
  no domínio, não helper na camada de aplicação) e será refletido em
  `plan.md` ao final, se a implementação o mantiver.
- O `geodatainspector` da etapa 2 **não** é alterado nesta etapa (risco
  `PixelIsPoint` registrado em `plan.md`).
- Faça commit após cada tarefa ou grupo lógico. A extração `atomicfile`
  (T002–T005) merece um commit próprio, separado do código novo.
- Pare em qualquer checkpoint para validar a história com o `quickstart.md`.
- Evite: tarefas vagas, duas tarefas `[P]` editando o mesmo arquivo,
  dependências que quebrem a independência de teste de uma história.
