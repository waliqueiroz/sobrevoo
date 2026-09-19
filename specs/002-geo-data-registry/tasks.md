---

description: "Task list template for feature implementation"
---

# Tarefas: Registro de Dados Geográficos Locais

**Entrada**: Documentos de design de `/specs/002-geo-data-registry/`

**Pré-requisitos**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/cli.md`, `quickstart.md`

**Testes**: incluídas. Diferente do padrão "testes são opcionais" deste
template, a constituição do projeto (Princípio VI — Testes Automatizados no
Núcleo, Princípio X — Given/When/Then) exige testify + uber-go/mock e
cobertura alta no núcleo (`internal/domain` + `internal/application`) sem
tocar disco, rede ou processo externo. As tarefas de teste abaixo
materializam essa exigência.

**Organização**: as tarefas são agrupadas por história de usuário (P1–P4 de
`spec.md`) para permitir implementação e teste independentes de cada
história. Diferente da etapa 1 (um único comando `inspect` refinado
incrementalmente por todas as histórias), esta etapa expõe quatro comandos
distintos — um por história — então cada subcomando Cobra pertence
inteiramente à fase da própria história, não a um esqueleto compartilhado
estendido ao longo de todas elas.

> **Nota de amendment (pós-implementação)**: as tarefas abaixo (T009–T060)
> descrevem a implementação original — quatro serviços de aplicação
> separados, cada um com um único método `Execute`
> (`RegisterGeoDataService`, `ListGeoDataService`, `RemoveGeoDataService`,
> `CheckCoverageService`). Por pedido explícito do usuário, essa camada foi
> consolidada em um único `GeoDataService` (`Register`/`List`/`Remove`/
> `CheckCoverage`), seguindo o padrão de service layer de
> `waliqueiroz/mystery-gifter-api` — ver `research.md` item 13 e
> `data-model.md`, que são a descrição atual e autoritativa da camada de
> aplicação. As tarefas abaixo permanecem marcadas `[X]` como registro
> histórico do que foi construído; onde mencionam `RegisterGeoDataService`,
> `ListGeoDataService`, `RemoveGeoDataService`, `CheckCoverageService` ou
> `.Execute(...)`, leia como o método correspondente de `GeoDataService`.
> `InspectTrackService.Execute` também foi renomeado para `Inspect` no
> mesmo pedido, por consistência (métodos nomeados pela operação, nunca
> `Execute`).
>
> Uma segunda rodada do mesmo pedido moveu os DTOs de saída
> (`CheckCoverageOutput`→`domain.CoverageReport`, `UncoveredSegment`,
> `CoverageStatus`, `MissingDataType`, `GeoDataSummary`) e o algoritmo de
> cobertura (antes um conjunto de funções soltas em
> `internal/application/check_coverage_service.go`) para
> `internal/domain` (`geo_data_coverage.go` — função pura
> `ComputeCoverage`; `GeoDataSummary` e o construtor `NewGeoDataSource` em
> `geo_data_source.go`), seguindo o mesmo padrão de
> `waliqueiroz/mystery-gifter-api` (DTOs e regra de negócio em
> `internal/domain`; a service layer só orquestra) — ver `research.md`
> item 14. `internal/application/geo_data_service.go` ficou só com
> orquestração de portas.
>
> Uma terceira rodada do mesmo pedido estendeu isso para a etapa 1: T019
> (`InspectTrackService`) passou a expor `Inspect(reader io.Reader,
> simplificationLevel, smoothingLevel domain.Level) (domain.TrackSummary,
> error)` — sem `InspectTrackInput`/`InspectTrackOutput` — e T033 (o
> helper `track_loading.go`) foi eliminado: a composição
> reorder+discard+discard+discard, que ele escondia atrás de uma porta
> `TrackParser` recebida por parâmetro, virou a função pura
> `domain.CleanTrack`, em `internal/domain/cleaning.go`; o que antes era o
> método privado `buildOutput` virou `domain.SummarizeTrack`, em
> `internal/domain/track_summary.go`. `InspectTrackOutputBuilder`
> (`build_application`) virou `build_domain.TrackSummaryBuilder` — ver
> `research.md` item 15.
>
> Uma quarta pergunta do usuário ("dá pra gente passar a chamar registry
> de repository...?") renomeou a porta de persistência de T003/T005/T008
> de `GeoDataRegistry` para `GeoDataRepository` (mock:
> `mock_domain/geo_data_repository.go`; campo/parâmetro `registry`→
> `repository` em `geoDataService`/`NewGeoDataService`) — mesmo
> vocabulário de `GroupRepository`/`UserRepository` em
> `waliqueiroz/mystery-gifter-api`. O conceito de "registro" em si
> (`RegistryPath`, `registry.json`, o texto de ajuda da CLI, o nome desta
> feature) não muda — só o nome do tipo Go da porta e o que dele deriva —
> ver `research.md` item 16.
>
> Uma quinta pergunta do usuário ("a pasta geodatastore precisa mesmo
> existir?") moveu o adapter de T008 de
> `internal/infra/outbound/geodatastore/jsonfile` para
> `internal/infra/outbound/jsonfile` — ver `research.md` item 17. Os
> caminhos antigos nas tarefas T002/T008/T062/T065 abaixo são o registro
> histórico do que foi feito na época.

## Formato: `[ID] [P?] [Story] Descrição`

- **[P]**: pode ser executado em paralelo (arquivos diferentes, sem
  dependência de tarefa incompleta). Tarefas que editam o mesmo arquivo NUNCA
  são marcadas `[P]` entre si, mesmo quando logicamente independentes.
- **[Story]**: a qual história de usuário esta tarefa pertence (US1, US2,
  US3, US4). Tarefas de Setup, Foundational e Polish não têm esse rótulo.
- Toda tarefa inclui o caminho de arquivo exato a criar/editar.

## Convenções de Caminho

Extensão do mesmo projeto único em Go da etapa 1, mesma estrutura
hexagonal definida em `plan.md`: `cmd/sobrevoo/`, `internal/domain/`,
`internal/application/`, `internal/infra/outbound/`,
`internal/infra/inbound/cli/`, `test/helper/`.

---

## Phase 1: Setup (Shared Infrastructure)

**Propósito**: preparar a dependência nova e os diretórios dos adapters
desta etapa (o módulo Go, o `Makefile` e a estrutura raiz já existem desde a
etapa 1).

- [X] T001 [P] Adicionar a dependência `modernc.org/sqlite` via `go get`, atualizando `go.mod`/`go.sum` (research.md item 2)
- [X] T002 [P] Criar o esqueleto de diretórios para os novos pacotes: `internal/infra/outbound/geodatainspector/`, `internal/infra/outbound/geodatastore/jsonfile/`, `internal/infra/outbound/filechecker/`, conforme a árvore em `plan.md`

**Checkpoint**: dependência e diretórios prontos.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Propósito**: a entidade `GeoDataSource`, os erros sentinela, o registro
persistido (`GeoDataRegistry` + adapter JSON) e a configuração do caminho do
registro — a base que TODA história de usuário (US1–US4) precisa, já que
todas leem e/ou escrevem o mesmo registro.

**⚠️ CRÍTICO**: nenhuma tarefa de história de usuário pode começar até que
esta fase esteja completa.

- [X] T003 [P] Criar a entidade `GeoDataSource` (`Name`, `Path`, `Type`, `Format`, `BoundingBox`, `RegisteredAt`), os enums `DataType` (`DataTypeBaseMap`, `DataTypeElevation`) e `DataFormat` (`DataFormatMBTiles`, `DataFormatGeoTIFF`), e a porta `GeoDataRegistry` (`Save(source GeoDataSource) error`, `FindByName(name string) (GeoDataSource, bool, error)`, `List() ([]GeoDataSource, error)`, `Delete(name string) error`) em `internal/domain/geo_data_source.go`, com a diretiva `//go:generate go run go.uber.org/mock/mockgen -destination mock_domain/geo_data_registry.go . GeoDataRegistry` posicionada diretamente acima da interface, conforme `data-model.md`
- [X] T004 [P] Declarar os 5 novos erros sentinela em `internal/domain/errors.go` (estender o arquivo já existente): `ErrDataFileNotFound` (FR-004), `ErrDataFileUnreadable` (FR-005), `ErrUnsupportedDataFormat` (FR-006), `ErrDataSourceNameAlreadyUsed` (FR-007), `ErrDataSourceNotRegistered` (FR-012) — Princípio VII da constituição
- [X] T005 [P] Gerar o mock de `GeoDataRegistry` executando `go generate ./internal/domain/...`, produzindo `internal/domain/mock_domain/geo_data_registry.go` (depende de T003)
- [X] T006 [P] Criar `GeoDataSourceBuilder` em `internal/domain/build_domain/geo_data_source_builder.go`: `NewGeoDataSourceBuilder()` com defaults sensatos, métodos fluentes `WithName`/`WithPath`/`WithType`/`WithFormat`/`WithBoundingBox`/`WithRegisteredAt`, e `Build()` terminal (depende de T003)
- [X] T007 [P] Estender `internal/infra/outbound/config/config.go`: adicionar `RegistryPath string`, resolvido a partir de `os.UserHomeDir()` como `~/.sobrevoo/registry.json`, com teste em `config_test.go` (research.md item 5)
- [X] T008 Implementar o adapter `GeoDataRegistry` em `internal/infra/outbound/geodatastore/jsonfile/jsonfile.go`: `Save`, `FindByName` (devolve `(GeoDataSource{}, false, nil)` quando não encontrado — não é erro), `List`, `Delete`, lendo/escrevendo o arquivo JSON em `RegistryPath` com escrita atômica (arquivo temporário no mesmo diretório + `os.Rename` — research.md item 5); testes em `jsonfile_test.go` cobrindo save/find/list/delete e a ausência do arquivo de registro na primeira execução (depende de T003, T007)

**Checkpoint**: núcleo do registro completo e testável, mas ainda sem
nenhum comando `geodata` executável.

---

## Phase 3: User Story 1 - Registrar um arquivo de dados geográficos (Priority: P1) 🎯 MVP

**Objetivo**: o usuário aponta a CLI para um arquivo MBTiles (mapa base) ou
GeoTIFF em CRS geográfico (relevo) e um nome de sua escolha, e a ferramenta
descobre sozinha o tipo e a área geográfica cobertos, recusando arquivo
inexistente, ilegível, de formato não reconhecido, ou nome já em uso.

**Teste Independente**: rodar `sobrevoo geodata register <arquivo> --name
<nome>` com um MBTiles válido e com um GeoTIFF válido, e verificar que cada
um é registrado com o tipo e a área corretos; repetir com um caminho
inexistente, um conteúdo não reconhecido, e um nome já usado.

### Implementação da História de Usuário 1

- [X] T009 [P] [US1] Estender `internal/domain/geo_data_source.go`: adicionar a struct `InspectedGeoData` (`Format`, `Type`, `BoundingBox`) e a porta `GeoDataInspector` (`Inspect(path string) (InspectedGeoData, error)`), com `//go:generate go run go.uber.org/mock/mockgen -destination mock_domain/geo_data_inspector.go . GeoDataInspector` (depende de T003)
- [X] ~~T010 [P] [US1] Criar a porta `Clock` (`Now() time.Time`) em `internal/domain/clock.go`, com `//go:generate go run go.uber.org/mock/mockgen -destination mock_domain/clock.go . Clock`~~ Feita e depois revertida: `time` é biblioteca padrão, não uma dependência externa no sentido do Princípio II, e `RegisteredAt` nunca é exibido ao usuário — a porta não compensava o custo da abstração. `RegisterGeoDataService` chama `time.Now()` diretamente (research.md item 10.1).
- [X] T011 [P] [US1] Gerar o mock de `GeoDataInspector` executando `go generate ./internal/domain/...`, produzindo `internal/domain/mock_domain/geo_data_inspector.go` (depende de T009)
- [X] T012 [P] [US1] Criar fixtures de MBTiles de teste em `test/helper/mbtiles_fixture.go`: um MBTiles válido com a chave `bounds` na tabela `metadata`, um MBTiles sem `bounds`, e um conteúdo que não é um banco SQLite
- [X] T013 [P] [US1] Criar fixtures de GeoTIFF de teste em `test/helper/geotiff_fixture.go`: um GeoTIFF válido em CRS geográfico (WGS84), um GeoTIFF em CRS projetado, e um conteúdo que não é TIFF
- [X] T014 [P] [US1] Implementar a leitura de MBTiles em `internal/infra/outbound/geodatainspector/mbtiles.go`: abrir o arquivo via `modernc.org/sqlite`, consultar a chave `bounds` da tabela `metadata`, converter `"minLon,minLat,maxLon,maxLat"` para `domain.BoundingBox`; devolver `domain.ErrUnsupportedDataFormat` quando a chave `bounds` estiver ausente; testes em `mbtiles_test.go` usando as fixtures de T012 (research.md itens 1-2) (depende de T012)
- [X] T015 [P] [US1] Implementar a leitura de tags GeoTIFF em `internal/infra/outbound/geodatainspector/geotiff.go`: parser próprio de cabeçalho TIFF + IFD via `encoding/binary`/`io.ReaderAt` (sem biblioteca externa), extraindo `ImageWidth`, `ImageLength`, `ModelPixelScaleTag`, `ModelTiepointTag` (ou `ModelTransformationTag`) e `GeoKeyDirectoryTag`; confirmar CRS geográfico (`GTModelTypeGeoKey = 2`) e computar a `BoundingBox`; devolver `domain.ErrUnsupportedDataFormat` para CRS projetado ou tags de georreferenciamento ausentes; testes em `geotiff_test.go` usando as fixtures de T013 (research.md itens 3-4) (depende de T013)
- [X] T016 [US1] Implementar o adapter dispatcher em `internal/infra/outbound/geodatainspector/geodatainspector.go`: identificar o formato pela assinatura do conteúdo (cabeçalho SQLite → `mbtiles.go`/mapa base; cabeçalho TIFF → `geotiff.go`/relevo), traduzir erros de abertura de arquivo do SO para `domain.ErrDataFileNotFound`/`domain.ErrDataFileUnreadable`, assinatura não reconhecida → `domain.ErrUnsupportedDataFormat`, implementando `domain.GeoDataInspector` (research.md item 8); testes em `geodatainspector_test.go` (depende de T009, T014, T015)
- [X] ~~T017 [P] [US1] Implementar o adapter `Clock` em `internal/infra/outbound/clock/clock.go` usando `time.Now()` (depende de T010)~~ Feita e depois revertida junto com T010 — ver nota acima.
- [X] T018 [US1] Implementar `RegisterGeoDataService` em `internal/application/register_geo_data_service.go`: `RegisterGeoDataInput{Name, Path}` / `RegisterGeoDataOutput{Source}`; recusa com `domain.ErrDataSourceNameAlreadyUsed` quando `GeoDataRegistry.FindByName` já encontra o nome; chama `GeoDataInspector.Inspect`; monta o `GeoDataSource` com `RegisteredAt` vindo de `time.Now()`; chama `GeoDataRegistry.Save`; testes usando os mocks de `GeoDataRegistry` e `GeoDataInspector`, com `RegisteredAt` verificado por uma janela `[antes, depois]` em torno da chamada (depende de T005, T011, T016)
- [X] T019 [P] [US1] Gerar o mock de `RegisterGeoDataService` em `internal/application/mock_application/register_geo_data_service.go` (depende de T018)
- [X] T020 [P] [US1] Implementar o comando pai Cobra `geodata` em `internal/infra/inbound/cli/geodata.go` (mesmo padrão de `root.go`, agrupa os subcomandos desta feature)
- [X] T021 [US1] Implementar o subcomando `register` em `internal/infra/inbound/cli/geodata_register.go`: argumento posicional `<arquivo>` + flag `--name` (obrigatória), chama `RegisterGeoDataService.Execute`, formata a confirmação em inglês descrita em `contracts/cli.md` (nome, tipo, área geográfica), anexa ao comando `geodata` (depende de T018, T020)
- [X] T022 [P] [US1] Estender `internal/infra/inbound/cli/exit_code.go`: mapear `domain.ErrDataFileNotFound`→`5`, `domain.ErrDataFileUnreadable`→`6`, `domain.ErrUnsupportedDataFormat`→`7`, `domain.ErrDataSourceNameAlreadyUsed`→`8` (contracts/cli.md) (depende de T004)
- [X] T023 [US1] Conectar em `cmd/sobrevoo/main.go`: `geodatainspector.New()`, `jsonfile.New(cfg.RegistryPath)`, `application.NewRegisterGeoDataService(...)`, e anexar o comando `geodata` (com o subcomando `register`) ao comando raiz (depende de T008, T014, T015, T016, T018, T020, T021, T022)
- [X] T024 [US1] Teste de contrato em `internal/infra/inbound/cli/geodata_register_test.go`: MBTiles válido → confirmação com tipo "base map" e área corretos, código de saída `0` (depende de T021)
- [X] T025 [US1] Teste de contrato em `geodata_register_test.go`: caminho inexistente → código de saída `5` (depende de T022)
- [X] T026 [US1] Teste de contrato em `geodata_register_test.go`: arquivo existente mas sem permissão de leitura → código de saída `6` (depende de T022)
- [X] T027 [US1] Teste de contrato em `geodata_register_test.go`: conteúdo de formato não reconhecido → código de saída `7` (depende de T022)
- [X] T028 [US1] Teste de contrato em `geodata_register_test.go`: nome já em uso por outro registro → código de saída `8` (depende de T022)

**Checkpoint**: História de Usuário 1 completa, demonstrável de ponta a
ponta (MVP).

---

## Phase 4: User Story 2 - Verificar a cobertura de um trajeto (Priority: P2)

**Objetivo**: o usuário verifica se um trajeto GPX está coberto — mapa base
E relevo, em toda a extensão — pelos dados já registrados, com o veredito
geral e, em caso de cobertura parcial, os subtrechos não cobertos (por
coordenadas de início/fim) e o que falta em cada um.

**Teste Independente**: com registros já existentes (criados via
`register`, História de Usuário 1) e um trajeto de exemplo, rodar `sobrevoo
geodata check <trajeto.gpx>` e verificar que o veredito (total/parcial/nulo)
e os subtrechos não cobertos aparecem corretamente.

### Implementação da História de Usuário 2

- [X] T029 [P] [US2] Estender `internal/domain/bounding_box.go`: adicionar `Contains(lat, lon float64) bool` (tratando `CrossesAntimeridian` corretamente) e `AreaDegrees() float64` (largura × altura em graus, com o mesmo "unwrap" de longitude), com testes em `bounding_box_test.go` cobrindo uma bounding box que cruza o antimeridiano e uma que não cruza (data-model.md)
- [X] T030 [P] [US2] Declarar a porta `FileChecker` (`Exists(path string) bool`) em `internal/domain/file_checker.go`, com `//go:generate go run go.uber.org/mock/mockgen -destination mock_domain/file_checker.go . FileChecker`
- [X] T031 [P] [US2] Gerar o mock de `FileChecker` executando `go generate ./internal/domain/...`, produzindo `internal/domain/mock_domain/file_checker.go` (depende de T030)
- [X] T032 [P] [US2] Implementar o adapter `FileChecker` em `internal/infra/outbound/filechecker/filechecker.go` usando `os.Stat` (depende de T030)
- [X] T033 [US2] Extrair a lógica compartilhada de leitura/limpeza de trajeto (parse via `TrackParser` + `ReorderByTime` + os três `Discard*`) de `internal/application/inspect_track_service.go` para um helper não exportado em `internal/application/track_loading.go`; ajustar `inspect_track_service.go` para chamar esse helper, preservando seu comportamento observável e os testes já existentes (research.md item 9); rodar `go test ./internal/application/...` imediatamente em seguida, antes de prosseguir para T034, para confirmar que nenhum teste existente da etapa 1 quebrou com a extração
- [X] T034 [US2] Implementar `CheckCoverageService` em `internal/application/check_coverage_service.go`: `CheckCoverageInput{Reader}`; enums `CoverageStatus` (`Full`/`Partial`/`None`) e `MissingDataType` (`BaseMap`/`Elevation`/`Both`); `UncoveredSegment{StartLatitude, StartLongitude, EndLatitude, EndLongitude, Missing}`; `CheckCoverageOutput{Status, UncoveredSegments, BaseMapSourcesUsed, ElevationSourcesUsed}`; usa o helper de T033 para obter a rota limpa (sem simplificação/suavização — research.md item 9); lista os registros via `GeoDataRegistry.List`, descartando os que `FileChecker.Exists` reporta como ausentes (FR-017); para cada ponto, escolhe o `GeoDataSource` de mapa base e o de relevo que o cobrem via `BoundingBox.Contains`, desempatando por `BoundingBox.AreaDegrees` (menor área vence) e depois por `RegisteredAt` mais antigo (FR-016, Clarification do spec.md); agrupa pontos consecutivos com o mesmo status de cobertura em `UncoveredSegment` (FR-015, Clarification do spec.md); agrega os conjuntos de fontes usadas; testes usando `TrackParser`, `GeoDataRegistry` e `FileChecker` mockados (depende de T005, T029, T031, T033)
- [X] T035 [P] [US2] Gerar o mock de `CheckCoverageService` em `internal/application/mock_application/check_coverage_service.go` (depende de T034)
- [X] T036 [US2] Implementar o subcomando `check` em `internal/infra/inbound/cli/geodata_check.go`: argumento posicional `<arquivo-de-trajeto>`, abre o arquivo com `os.Open` (mesmo padrão de `inspect.go` na etapa 1 — `CheckCoverageInput.Reader` exige um `io.Reader`, não um caminho), chama `CheckCoverageService.Execute`, formata o relatório em inglês descrito em `contracts/cli.md` (veredito, subtrechos não cobertos, fontes usadas), reaproveita o mapeamento de erro de trajeto já existente em `exit_code.go` (`ErrEmptyFile`→`1`, `ErrUnsupportedFormat`→`2`, `ErrInsufficientPoints`/`ErrInsufficientPointsAfterCleaning`→`3`, erro de I/O do `os.Open`→`4`), anexa ao comando `geodata` (depende de T020, T034)
- [X] T037 [US2] Conectar `CheckCoverageService` e seu subcomando em `cmd/sobrevoo/main.go` (depende de T023, T032, T034, T036)
- [X] T038 [US2] Teste de contrato em `internal/infra/inbound/cli/geodata_check_test.go`: trajeto totalmente coberto por mapa base e relevo registrados → status "full", fontes usadas listadas, código de saída `0` (depende de T036)
- [X] T039 [US2] Teste de contrato em `geodata_check_test.go`: só mapa base registrado (cobrindo toda a extensão), sem relevo cobrindo o trajeto → status "partial" (há cobertura real, só que incompleta — não "none", ver regra em `data-model.md`), relatório indica falta de relevo (depende de T036)
- [X] T040 [US2] Teste de contrato em `geodata_check_test.go`: só relevo registrado (cobrindo toda a extensão), sem mapa base cobrindo o trajeto → status "partial", relatório indica falta de mapa base — caso simétrico a T039 (depende de T036)
- [X] T041 [US2] Teste de contrato em `geodata_check_test.go`: trajeto parcialmente coberto → status "partial", subtrecho não coberto reportado com coordenadas geográficas de início e fim (FR-015) (depende de T036)
- [X] T042 [US2] Teste de contrato em `geodata_check_test.go`: nenhum registro cadastrado → status "none", trajeto inteiro reportado como um único subtrecho não coberto (depende de T036)
- [X] T043 [US2] Teste de contrato em `geodata_check_test.go`: duas fontes de mapa base sobrepostas, uma de área menor contida na maior → a mais específica (menor área) aparece em `BaseMapSourcesUsed` (FR-016, Clarification do spec.md) (depende de T036)
- [X] T044 [US2] Teste de contrato em `geodata_check_test.go`: trajeto cruzando o antimeridiano com registros cobrindo os dois lados → cobertura corretamente reportada, sem tratamento especial (FR-018) (depende de T036)
- [X] T045 [US2] Teste de contrato em `geodata_check_test.go`: registro cujo arquivo não existe mais no caminho registrado → excluído da verificação de cobertura (FR-017) (depende de T036)
- [X] T046 [US2] Teste de contrato em `geodata_check_test.go`: arquivo de trajeto vazio → reaproveita `domain.ErrEmptyFile`, código de saída `1`, igual ao comando `inspect` (Casos Extremos do spec.md) (depende de T036)

**Checkpoint**: Histórias de Usuário 1 e 2 funcionam de forma independente.

---

## Phase 5: User Story 3 - Listar os dados registrados (Priority: P3)

**Objetivo**: o usuário vê nome, tipo, área coberta e disponibilidade de
arquivo de todos os registros, com um registro cujo arquivo foi movido ou
apagado sinalizado em vez de interromper a listagem.

**Teste Independente**: com registros já existentes (um deles com o arquivo
movido/apagado do disco), rodar `sobrevoo geodata list` e verificar que
nome, tipo, área e disponibilidade aparecem corretamente para cada um.

### Implementação da História de Usuário 3

- [X] T047 [US3] Implementar `ListGeoDataService` em `internal/application/list_geo_data_service.go`: `GeoDataSummary{Source, Available}`; `ListGeoDataOutput{Sources}`; `Execute` chama `GeoDataRegistry.List` e preenche `Available` via `FileChecker.Exists` para cada registro; testes usando os mocks de `GeoDataRegistry`/`FileChecker` já existentes (depende de T005, T031)
- [X] T048 [P] [US3] Gerar o mock de `ListGeoDataService` em `internal/application/mock_application/list_geo_data_service.go` (depende de T047)
- [X] T049 [US3] Implementar o subcomando `list` em `internal/infra/inbound/cli/geodata_list.go`: sem argumentos, chama `ListGeoDataService.Execute`, formata uma linha por registro (nome, tipo, área, disponibilidade — sinalizando algo como `(file not found)` quando ausente, FR-009/FR-010) ou uma mensagem clara quando não há nenhum registro, anexa ao comando `geodata` (depende de T020, T047)
- [X] T050 [US3] Conectar `ListGeoDataService` e seu subcomando em `cmd/sobrevoo/main.go` (depende de T037, T047, T049)
- [X] T051 [US3] Teste de contrato em `internal/infra/inbound/cli/geodata_list_test.go`: dois ou mais registros → cada linha mostra nome, tipo, área (depende de T049)
- [X] T052 [US3] Teste de contrato em `geodata_list_test.go`: um registro cujo arquivo foi movido/apagado → sinalizado como indisponível, os demais registros continuam listados normalmente (FR-010) (depende de T049)
- [X] T053 [US3] Teste de contrato em `geodata_list_test.go`: nenhum registro existente → mensagem explícita de que não há dados registrados, código de saída `0` (depende de T049)

**Checkpoint**: Histórias de Usuário 1, 2 e 3 funcionam de forma
independente.

---

## Phase 6: User Story 4 - Remover um registro (Priority: P4)

**Objetivo**: o usuário remove um registro pelo nome sem apagar o arquivo
de dados original, com uma mensagem clara quando o nome não existe.

**Teste Independente**: registrar um arquivo, removê-lo pelo nome, e
confirmar que ele desaparece da listagem enquanto o arquivo original
continua existindo no disco.

### Implementação da História de Usuário 4

- [X] T054 [US4] Implementar `RemoveGeoDataService` em `internal/application/remove_geo_data_service.go`: `RemoveGeoDataInput{Name}`; `Execute` recusa com `domain.ErrDataSourceNotRegistered` quando `GeoDataRegistry.FindByName` não encontra o nome, senão chama `GeoDataRegistry.Delete`; testes usando `GeoDataRegistry` mockado (depende de T005)
- [X] T055 [P] [US4] Gerar o mock de `RemoveGeoDataService` em `internal/application/mock_application/remove_geo_data_service.go` (depende de T054)
- [X] T056 [US4] Implementar o subcomando `remove` em `internal/infra/inbound/cli/geodata_remove.go`: argumento posicional `<nome>`, chama `RemoveGeoDataService.Execute`, imprime confirmação ou mapeia `domain.ErrDataSourceNotRegistered`, anexa ao comando `geodata` (depende de T020, T054)
- [X] T057 [US4] Estender `internal/infra/inbound/cli/exit_code.go`: mapear `domain.ErrDataSourceNotRegistered`→`9` (contracts/cli.md) (depende de T022, T056)
- [X] T058 [US4] Conectar `RemoveGeoDataService` e seu subcomando em `cmd/sobrevoo/main.go` (depende de T050, T054, T056)
- [X] T059 [US4] Teste de contrato em `internal/infra/inbound/cli/geodata_remove_test.go`: registro existente → removido, some da listagem, arquivo de dado original permanece intacto no disco (FR-011) (depende de T056, T057)
- [X] T060 [US4] Teste de contrato em `geodata_remove_test.go`: nome não corresponde a nenhum registro → código de saída `9` (depende de T057)

**Checkpoint**: todas as quatro histórias de usuário funcionam de forma
independente.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Propósito**: qualidade final que atravessa todas as histórias.

- [X] T061 [P] Rodar `go vet ./...` e corrigir qualquer problema encontrado no repositório
- [X] T062 [P] Adicionar comentários de documentação de pacote (`// Package ...`) em `internal/infra/outbound/geodatainspector`, `internal/infra/outbound/geodatastore/jsonfile` e `internal/infra/outbound/filechecker`
- [X] T063 Executar manualmente todos os cenários de `quickstart.md` contra o binário compilado (`make build`) e registrar qualquer divergência encontrada
- [X] T064 [P] Rodar `go test ./... -cover` e confirmar cobertura alta em `internal/domain` e `internal/application` (Princípio VI da constituição), adicionando casos que faltarem
- [X] T065 [P] Atualizar o parágrafo de arquitetura do `CLAUDE.md` (item `internal/infra/inbound/cli`) para refletir que, a partir desta etapa, os adapters de saída `geodatainspector`, `geodatastore/jsonfile` e `filechecker` também tocam o sistema de arquivos — não só a CLI —, conforme justificado em `research.md` item 8

---

## Dependências e Ordem de Execução

### Dependências entre Fases

- **Setup (Phase 1)**: sem dependências — pode começar imediatamente.
- **Foundational (Phase 2)**: depende da conclusão do Setup — BLOQUEIA todas
  as histórias de usuário, já que todas leem e/ou escrevem o mesmo
  `GeoDataRepository` (`GeoDataRegistry` ao tempo desta tarefa — ver
  `research.md` item 16).
- **User Stories (Phase 3–6)**: todas dependem da conclusão da fase
  Foundational.
  - Diferente da etapa 1, as quatro histórias implementam quatro comandos
    Cobra distintos (`register`, `check`, `list`, `remove`), cada um com seu
    próprio serviço de aplicação — não são extensões incrementais do mesmo
    comando. Ainda assim, seguem a ordem de prioridade (P1→P4) porque US2–US4
    reaproveitam peças construídas por histórias anteriores: US2 usa
    `FileChecker` (introduzido nela mesma) e o comando pai `geodata`
    (T020, de US1); US3 reaproveita o `FileChecker` já introduzido por US2;
    US4 estende o mesmo `exit_code.go` já tocado por US1.
  - `internal/application/track_loading.go` (T033, dentro de US2) refatora
    `inspect_track_service.go` da etapa 1 — só a história que precisa da
    rota limpa (verificação de cobertura) faz esse ajuste, mas ele não
    depende de nenhuma outra tarefa desta etapa além da Foundational.
- **Polish (Phase 7)**: depende da conclusão de todas as histórias de
  usuário desejadas.

### Dentro de Cada História de Usuário

- Portas/entidades de domínio antes dos adapters de saída; adapters antes do
  serviço de aplicação; serviço antes do subcomando de CLI; subcomando antes
  dos testes de contrato que o exercitam.
- Testes que compartilham o mesmo arquivo (`geodata_register_test.go`,
  `geodata_check_test.go`, `geodata_list_test.go`, `geodata_remove_test.go`)
  são executados em sequência, mesmo quando cobrem cenários logicamente
  independentes.

### Oportunidades de Paralelização

- Todas as tarefas de Setup marcadas `[P]` (T001, T002).
- Dentro do Foundational: T003, T004, T006, T007 (arquivos independentes);
  T005 pode rodar assim que T003 terminar.
- Dentro de US1: T009–T013 (portas/fixtures em arquivos diferentes, sem
  dependência mútua); depois T014/T015 (implementações independentes de
  MBTiles/GeoTIFF); T017/T019/T020/T022 (adapters/comando/exit codes em
  arquivos diferentes).
- Dentro de US2: T029–T032 (bounding box, porta, mock e adapter de
  `FileChecker` em arquivos diferentes).
- Uma vez concluído o Foundational, US1 pode começar imediatamente; US2, US3
  e US4 têm dependências pontuais em peças de histórias anteriores (ver
  acima), então não são totalmente paralelizáveis entre si apesar de cada
  uma ser independentemente testável ao final de sua fase.

---

## Exemplo de Paralelização: User Story 1

```bash
# Disparar as portas/fixtures independentes juntas:
Task: "Estender geo_data_source.go com InspectedGeoData e GeoDataInspector"
Task: "Criar fixtures de MBTiles em test/helper/mbtiles_fixture.go"
Task: "Criar fixtures de GeoTIFF em test/helper/geotiff_fixture.go"

# Depois, disparar as duas implementações de leitura independentes:
Task: "Implementar leitura de MBTiles em geodatainspector/mbtiles.go"
Task: "Implementar leitura de tags GeoTIFF em geodatainspector/geotiff.go"
```

---

## Estratégia de Implementação

### MVP Primeiro (Somente História de Usuário 1)

1. Completar Phase 1: Setup.
2. Completar Phase 2: Foundational (CRÍTICO — bloqueia todas as histórias).
3. Completar Phase 3: História de Usuário 1.
4. **PARAR E VALIDAR**: rodar os Cenários 1, 3 e 4 de `quickstart.md`.
5. Nesse ponto já existe um `sobrevoo geodata register` funcional (MVP) —
   o usuário já consegue começar a catalogar seus dados locais, mesmo sem
   ainda poder verificar cobertura.

### Entrega Incremental

1. Setup + Foundational → registro pronto, nada executável ainda.
2. + US1 → `register` funcional (MVP) → validar com Cenários 1, 3, 4 de
   `quickstart.md`.
3. + US2 → `check` funcional, incluindo sobreposição e antimeridiano →
   validar com Cenários 6–10.
4. + US3 → `list` funcional, incluindo arquivo movido/apagado → validar com
   Cenário 2.
5. + US4 → `remove` funcional → validar com Cenário 5.
6. Cada história agrega valor sem quebrar as histórias anteriores.

---

## Notas

- `[P]` = arquivos diferentes, sem dependência entre si.
- O rótulo de história (`[US1]`...`[US4]`) mapeia a tarefa para
  rastreabilidade com `spec.md`.
- Cada história de usuário deve ser completável e testável de forma
  independente ao final de sua fase, mesmo quando reaproveita peças
  construídas por uma história anterior (comando pai `geodata`,
  `FileChecker`, `exit_code.go`).
- Faça commit após cada tarefa ou grupo lógico.
- Pare em qualquer checkpoint para validar a história com o `quickstart.md`.
- Evite: tarefas vagas, duas tarefas `[P]` editando o mesmo arquivo,
  dependências que quebrem a independência de teste de uma história.
