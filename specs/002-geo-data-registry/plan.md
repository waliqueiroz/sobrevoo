# Plano de Implementação: Registro de Dados Geográficos Locais

**Branch**: `002-geo-data-registry` | **Data**: 2026-09-13 | **Especificação**: [spec.md](./spec.md)

**Entrada**: Especificação de funcionalidade de `/specs/002-geo-data-registry/spec.md`

**Nota**: Este template é preenchido pelo comando `/speckit-plan`; sua definição descreve o fluxo de execução.

## Resumo

Segunda etapa do Sobrevoo: uma extensão da mesma CLI em Go que permite ao
usuário registrar arquivos locais de mapa base (MBTiles) e de relevo
(GeoTIFF), com tipo e área geográfica coberta descobertos automaticamente a
partir do conteúdo do arquivo; listar, remover e usar esse registro para
verificar se um trajeto GPS já processado pela etapa 1 está coberto — para
mapa base e relevo, em toda a sua extensão — pelos dados já disponíveis
localmente, relatando com precisão qual trecho ficou de fora quando não
estiver. A abordagem técnica mantém a arquitetura hexagonal já estabelecida:
o núcleo ganha uma nova entidade (`GeoDataSource`) e três novas portas
(`GeoDataInspector`, `GeoDataRegistry`, `FileChecker`, mais um pequeno
`Clock`); toda leitura de arquivo de dados geográfico, toda persistência do
registro e toda checagem de disponibilidade de arquivo acontece em adapters
de saída dedicados, nunca no núcleo. Quatro novos casos de uso
(`RegisterGeoDataService`, `ListGeoDataService`, `RemoveGeoDataService`,
`CheckCoverageService`) são expostos por um novo comando Cobra `geodata`
(com subcomandos `register`, `list`, `remove`, `check`), reaproveitando o
`TrackParser` e as funções puras de limpeza de trajeto já existentes da
etapa 1 em vez de duplicá-las.

## Contexto Técnico

**Linguagem/Versão**: Go 1.26 (sem mudança em relação à etapa 1)

**Dependências Principais**: Cobra (CLI, já existente); `modernc.org/sqlite`
(driver SQLite puro em Go, sem cgo, usado apenas para ler a tabela
`metadata` de arquivos MBTiles — ver `research.md` itens 1-2); parsing de
GeoTIFF feito à mão com a biblioteca padrão (`encoding/binary`, `io`), sem
biblioteca externa (ver `research.md` item 4); `encoding/json` da biblioteca
padrão para persistir o registro; `github.com/stretchr/testify` e
`go.uber.org/mock` (já existentes, mesmo padrão de teste)

**Armazenamento**: um único arquivo JSON local, em um caminho fixo e igual
em qualquer sistema operacional (`~/.sobrevoo/registry.json`, ver
`research.md` item 5), resolvido por um adapter de configuração e escrito
de forma atômica — não há banco de dados nem serviço externo (Princípio V
da constituição)

**Testes**: mesmo padrão já estabelecido — `go test` com `testify`, formato
given/when/then (`// given`/`// when`/`// then`, sem tabelas de casos),
builders em `build_domain`/`build_application` para as novas entidades/DTOs,
mocks gerados com `go.uber.org/mock` para as quatro novas portas do domínio
(`GeoDataInspector`, `GeoDataRegistry`, `FileChecker`, `Clock`) e para os
quatro novos serviços de aplicação; os testes de `internal/infra/inbound/cli`
continuam mockando exclusivamente a service layer, nunca os adapters de
saída reais (nenhum teste automatizado abre um MBTiles/GeoTIFF real ou toca
o arquivo de registro de verdade)

**Plataforma-Alvo**: o mesmo binário de linha de comando multiplataforma
(Linux, macOS, Windows) da etapa 1 — `modernc.org/sqlite` foi escolhido
justamente por ser livre de cgo, preservando a compilação cruzada sem
dependência de bibliotecas de sistema

**Tipo de Projeto**: CLI (mesmo projeto único em Go da etapa 1, sem
frontend/backend separados)

**Metas de Desempenho**: a especificação não define uma meta numérica
formal (ferramenta pessoal, sem limite superior de registros definido —
Suposições do spec). Como diretriz de engenharia: `list` e `check` nunca
reabrem nem reprocessam o conteúdo de um arquivo MBTiles/GeoTIFF — usam
apenas o tipo e a área geográfica já persistidos no registro no momento do
`register` (ver `research.md` item 6) — de modo que ambos os comandos
permaneçam rápidos (sublinear ao tamanho dos arquivos de dados) mesmo com
centenas de registros e trajetos de até 20.000 pontos (mesma ordem de
grandeza da etapa 1)

**Restrições**: funcionamento 100% offline, sem rede nem chave de API
(Princípio V); núcleo (`internal/domain` + `internal/application`)
continua sem acesso a disco, rede, ou processo externo — toda leitura de
arquivo de dados geográfico, leitura/escrita do registro e checagem de
existência de arquivo passa por uma porta implementada em
`internal/infra/outbound` (Princípios I e II); apenas GeoTIFF em CRS
geográfico (WGS84/EPSG:4326) é suportado nesta etapa (ver `research.md`
item 4)

**Escala/Escopo**: registro pessoal de dados geográficos, tipicamente
dezenas a poucas centenas de fontes (Suposições do spec: sem limite
superior definido); verificação de cobertura sobre um trajeto de até
20.000 pontos (mesma escala da etapa 1, já que reaproveita o mesmo
`TrackParser`)

## Verificação da Constituição

*PORTÃO: Deve passar antes da Fase 0 de pesquisa. Reverificar após o design da Fase 1.*

| Princípio | Avaliação | Como o design atende |
|---|---|---|
| I. Arquitetura Hexagonal | PASSA | A nova entidade `GeoDataSource` e as novas portas (`GeoDataInspector`, `GeoDataRegistry`, `FileChecker`, `Clock`) vivem em `internal/domain`; nenhum desses arquivos importa `modernc.org/sqlite`, `encoding/json`/`os` para I/O real, ou qualquer pacote de `internal/infra`. Os quatro novos serviços de `internal/application` orquestram essas portas e as funções puras já existentes, sem tocar arquivo, rede, ou terminal diretamente. |
| II. Portas para Toda Dependência Externa | PASSA | Leitura de MBTiles/GeoTIFF, persistência do registro e checagem de existência de arquivo são acessadas exclusivamente via `GeoDataInspector`, `GeoDataRegistry` e `FileChecker`, declaradas no domínio e implementadas em `internal/infra/outbound/geodatainspector`, `internal/infra/outbound/geodatastore/jsonfile` e `internal/infra/outbound/filechecker`. |
| III. Entrypoints Descartáveis | PASSA | Os quatro novos serviços (`RegisterGeoDataService`, `ListGeoDataService`, `RemoveGeoDataService`, `CheckCoverageService`) recebem/devolvem apenas DTOs de dados; a CLI (`internal/infra/inbound/cli`, comando `geodata` e seus subcomandos) é o único adapter de entrada desta etapa, mas nenhum deles conhece Cobra, flags, ou formatação de texto — prontos para um futuro adapter HTTP sem alteração. |
| IV. Neutralidade Geográfica | PASSA | Nenhum dado de mapa/relevo/coordenada embutido; tipo e área de cada registro vêm exclusivamente do conteúdo do arquivo que o próprio usuário indicou. A comparação de área para desempate (`BoundingBox.AreaDegrees`) e a checagem de contenção (`BoundingBox.Contains`) reaproveitam o mesmo tratamento de antimeridiano já usado por `ComputeBoundingBox` na etapa 1 — nenhuma região recebe tratamento especial. |
| V. Funcionamento Offline | PASSA | `modernc.org/sqlite` e o parser de GeoTIFF operam inteiramente sobre arquivos locais já baixados pelo usuário; o registro é um arquivo local; nenhuma chamada de rede em nenhum adapter desta etapa. |
| VI. Testes Automatizados no Núcleo | PASSA | `internal/domain` (novas funções puras de `BoundingBox`) testado sem I/O; `internal/application` testado com `GeoDataInspector`, `GeoDataRegistry`, `FileChecker`, `Clock` e `TrackParser` mockados via `go.uber.org/mock`; nenhum teste do núcleo abre um arquivo MBTiles/GeoTIFF real nem toca o arquivo de registro. |
| VII. Erros Sentinela no Domínio | PASSA | `ErrDataFileNotFound`, `ErrDataFileUnreadable`, `ErrUnsupportedDataFormat`, `ErrDataSourceNameAlreadyUsed`, `ErrDataSourceNotRegistered` declarados em `internal/domain/errors.go`, traduzidos pela CLI em novos códigos de saída de processo (ver `contracts/cli.md`); a verificação de cobertura reaproveita os sentinelas de trajeto já existentes (`ErrEmptyFile`, `ErrUnsupportedFormat`, `ErrInsufficientPoints[AfterCleaning]`) para os mesmos erros de trajeto inválido. |
| VIII. Configuração Injetada | PASSA | O caminho do arquivo de registro é resolvido por `internal/infra/outbound/config` (estendido) e injetado na construção do adapter `geodatastore/jsonfile` em `cmd/sobrevoo`; o núcleo nunca lê `os.UserHomeDir()`, variável de ambiente, ou flag diretamente. |
| IX. Organização de Portas, Service Layer e Mocks | PASSA | Nenhum `ports.go`/`interfaces.go` novo: `GeoDataInspector` e `GeoDataRegistry` — que produzem/manipulam `GeoDataSource` — ficam em `geo_data_source.go`, junto da entidade; `FileChecker` e `Clock`, sem entidade dona, ganham arquivo próprio (`file_checker.go`, `clock.go`). Os quatro novos casos de uso seguem `XService`/`xService`/`NewXService(...)`, cada um em seu próprio arquivo em `internal/application`. Mocks gerados com `go.uber.org/mock/mockgen` via `//go:generate` acima de cada interface, em `mock_domain`/`mock_application`. |
| X. Testes: Given/When/Then, Builders e Isolamento por Camada | PASSA | Todo teste novo usa `t.Run("should ...")` com comentários `// given`/`// when`/`// then`, sem tabelas de casos. Um novo builder `GeoDataSourceBuilder` em `build_domain` evita literais de struct repetidos nos testes das quatro novas portas/serviços. Cada camada continua isolada: domínio/serviço com portas mockadas, CLI com os quatro novos serviços mockados (`mock_application`) — nunca com a implementação real nem com os adapters de saída reais. |
| Idioma dos Artefatos | PASSA | Este plano, `research.md`, `data-model.md`, `contracts/cli.md` e `quickstart.md` estão em português do Brasil; identificadores, nomes de pacote/arquivo e comentários de código permanecem em inglês. Seguindo a mesma decisão já registrada na etapa 1 (`research.md` item 11), todo o I/O em tempo de execução dos novos comandos (nomes de flag, texto de saída, mensagens de erro) também é em inglês. |

Nenhuma violação identificada. A seção de Rastreamento de Complexidade não se
aplica a este plano.

## Estrutura do Projeto

### Documentação (desta funcionalidade)

```text
specs/002-geo-data-registry/
├── plan.md              # Este arquivo (saída do comando /speckit-plan)
├── research.md          # Saída da Fase 0 (comando /speckit-plan)
├── data-model.md        # Saída da Fase 1 (comando /speckit-plan)
├── quickstart.md        # Saída da Fase 1 (comando /speckit-plan)
├── contracts/
│   └── cli.md           # Saída da Fase 1 (comando /speckit-plan)
└── tasks.md             # Saída da Fase 2 (comando /speckit-tasks - NÃO criado pelo /speckit-plan)
```

### Código-Fonte (raiz do repositório)

```text
cmd/
└── sobrevoo/
    └── main.go                                   # (estendido) monta os novos adapters/serviços e o comando "geodata"

internal/
├── domain/
│   ├── geo_data_source.go                        # (novo) entidade GeoDataSource + DataType + DataFormat + portas GeoDataInspector e GeoDataRegistry (produzem/manipulam GeoDataSource)
│   ├── file_checker.go                           # (novo) porta FileChecker (sem entidade dona)
│   ├── clock.go                                  # (novo) porta Clock (sem entidade dona)
│   ├── bounding_box.go                           # (estendido) + Contains(lat, lon float64) bool, + AreaDegrees() float64
│   ├── errors.go                                 # (estendido) + 5 novos erros sentinela
│   ├── build_domain/
│   │   └── geo_data_source_builder.go            # (novo) test data builder
│   └── mock_domain/                              # (estendido) + geo_data_inspector.go, geo_data_registry.go, file_checker.go, clock.go
│
├── application/
│   ├── register_geo_data_service.go              # (novo) RegisterGeoDataService
│   ├── list_geo_data_service.go                  # (novo) ListGeoDataService
│   ├── remove_geo_data_service.go                # (novo) RemoveGeoDataService
│   ├── check_coverage_service.go                 # (novo) CheckCoverageService
│   ├── track_loading.go                          # (novo) helper interno compartilhado (parse + reordenação + descarte), extraído de inspect_track_service.go
│   ├── inspect_track_service.go                  # (ajustado) passa a chamar o helper de track_loading.go em vez de repetir a lógica
│   ├── build_application/                        # (estendido) builders para os novos DTOs de saída, se necessário
│   └── mock_application/                         # (estendido) + um mock por novo serviço
│
└── infra/
    ├── outbound/
    │   ├── config/
    │   │   └── config.go                         # (estendido) + RegistryPath, resolvido via os.UserHomeDir() (~/.sobrevoo)
    │   ├── geodatainspector/
    │   │   ├── geodatainspector.go                # (novo) adapter GeoDataInspector: identifica o formato pela assinatura do arquivo e delega
    │   │   ├── mbtiles.go                         # (novo) leitura de tipo/área de um MBTiles (mapa base) via modernc.org/sqlite
    │   │   └── geotiff.go                         # (novo) leitura de tipo/área de um GeoTIFF (relevo) via parser de tags próprio
    │   ├── geodatastore/
    │   │   └── jsonfile/
    │   │       └── jsonfile.go                    # (novo) adapter GeoDataRegistry: registro persistido em JSON, escrita atômica
    │   ├── filechecker/
    │   │   └── filechecker.go                     # (novo) adapter FileChecker via os.Stat
    │   └── clock/
    │       └── clock.go                           # (novo) adapter Clock via time.Now()
    │
    └── inbound/
        └── cli/
            ├── geodata.go                          # (novo) comando pai "geodata", agrupa os subcomandos abaixo
            ├── geodata_register.go                 # (novo) subcomando "register"
            ├── geodata_list.go                     # (novo) subcomando "list"
            ├── geodata_remove.go                    # (novo) subcomando "remove"
            ├── geodata_check.go                     # (novo) subcomando "check"
            └── exit_code.go                         # (estendido) + novos casos de erro sentinela

test/
└── helper/
    ├── gpx_fixture.go                              # (já existente)
    ├── mbtiles_fixture.go                           # (novo) geração de conteúdo MBTiles de teste (válido/inválido)
    └── geotiff_fixture.go                           # (novo) geração de conteúdo GeoTIFF de teste (válido/inválido/CRS projetado)
```

**Decisão de Estrutura**: extensão do mesmo projeto único em Go da etapa 1,
sem novo módulo nem repositório separado. Os novos adapters de saída ganham
seus próprios subdiretórios em `internal/infra/outbound` (um por
dependência externa concreta: `geodatainspector`, `geodatastore/jsonfile`,
`filechecker`, `clock`), seguindo o mesmo padrão de um pacote por adapter já
usado por `trackparser`, `simplifier/douglaspeucker` e
`smoother/catmullrom`. Dentro de `geodatainspector`, MBTiles e GeoTIFF ficam
no mesmo pacote (não em subpacotes por formato) porque, assim como
`trackparser` na etapa 1, é um único adapter que primeiro identifica o
formato pela assinatura do conteúdo e depois delega — não dois adapters
concorrendo pela mesma porta.

Nenhum arquivo genérico `ports.go`/`interfaces.go` é criado: `GeoDataInspector`
e `GeoDataRegistry` — que produzem/manipulam a entidade `GeoDataSource` —
ficam no arquivo dessa entidade (`geo_data_source.go`), assim como
`TrackParser` fica em `track.go` na etapa 1. `FileChecker` e `Clock`, sem
entidade dona, ganham cada um seu próprio arquivo nomeado pelo conceito que
representam, assim como `Simplifier`/`Smoother`.

A camada de `internal/application` ganha quatro novos casos de uso, cada um
modelado como `XService`/`xService`/`NewXService(...)`, no mesmo padrão de
`InspectTrackService`. Para não duplicar a lógica de leitura e limpeza de
trajeto (parse + reordenação por tempo + descarte de pontos problemáticos)
entre `InspectTrackService` e o novo `CheckCoverageService`, essa lógica é
extraída para uma função interna não exportada em `track_loading.go`,
chamada por ambos — um ajuste pontual em `inspect_track_service.go` para
remover a duplicação, sem alterar seu comportamento observável nem sua
interface pública.

Testes de unidade continuam no formato given/when/then, sem tabelas de
casos, com builders em `build_domain`/`build_application` para as novas
entidades/DTOs sempre que um literal de struct repetido prejudicaria a
legibilidade — mesma convenção já em vigor.

## Rastreamento de Complexidade

*Não aplicável — nenhuma violação da Verificação da Constituição foi identificada.*
