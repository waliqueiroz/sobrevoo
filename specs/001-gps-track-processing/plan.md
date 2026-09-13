# Plano de Implementação: Leitura e Tratamento de Trajeto GPS

**Branch**: `001-gps-track-processing` | **Data**: 2026-09-13 | **Especificação**: [spec.md](./spec.md)

**Entrada**: Especificação de funcionalidade de `/specs/001-gps-track-processing/spec.md`

**Nota**: Este template é preenchido pelo comando `/speckit-plan`; sua definição descreve o fluxo de execução.

## Resumo

Primeira etapa do Sobrevoo: uma CLI em Go que lê um arquivo de rastreamento GPS
em formato GPX (conteúdo validado antes do parsing), aplica um tratamento em
quatro etapas — reordenação por tempo, descarte de pontos problemáticos,
simplificação (Douglas-Peucker) e suavização (Catmull-Rom) — e imprime um
resumo com
contagem de pontos, distância, ganho de elevação, duração e área geográfica
ocupada, funcionando de forma idêntica em qualquer região do planeta. A
abordagem técnica segue arquitetura hexagonal: o núcleo (`internal/domain` +
`internal/application`) contém apenas entidades, portas e funções puras; toda
biblioteca de parsing/algoritmo concreta vive em adapters de saída
(`internal/infra/outbound`), e a CLI (`internal/infra/inbound/cli`, via
Cobra) é o único adapter de entrada desta etapa, convertendo flags em um DTO
de entrada e formatando o DTO de saída do serviço `InspectTrackService` — sem
nenhuma regra de negócio duplicada nele.

## Contexto Técnico

**Linguagem/Versão**: Go 1.26

**Dependências Principais**: Cobra (CLI); `github.com/tkrajina/gpxgo` (parsing
de GPX); `github.com/stretchr/testify` (asserções de teste); `go.uber.org/mock`
(mocks de porta gerados via `mockgen`)

**Armazenamento**: N/A — a ferramenta lê um arquivo local por execução e não
persiste nenhum estado

**Testes**: `go test` com `testify` (assert/require) para todo o núcleo e os
adapters, no formato given/when/then (comentários `// given`/`// when`/
`// then`, sem tabelas de casos), com builders de dados de teste
(`build_domain`, `build_application`) onde isso deixa o teste mais legível;
casos explícitos de antimeridiano e equador nas funções puras do domínio;
mocks gerados com `go.uber.org/mock` para as portas `TrackParser`,
`Simplifier` e `Smoother` nos testes de `internal/application`, e para o
serviço `InspectTrackService` nos testes de `internal/infra/inbound/cli`
(a CLI nunca é testada contra a implementação real do serviço); testes do
comando Cobra via `cmd.SetArgs`/`SetOut`

**Plataforma-Alvo**: binário de linha de comando multiplataforma (Linux,
macOS, Windows) — sem dependência de sistema operacional específico

**Tipo de Projeto**: CLI (projeto único em Go, sem frontend/backend
separados)

**Metas de Desempenho**: resumo de um trajeto de até 20.000 pontos apresentado
em menos de 5 segundos (SC-005)

**Restrições**: funcionamento 100% offline, sem rede nem chave de API
(Princípio V da constituição); núcleo (`internal/domain` +
`internal/application`) sem acesso a disco, rede, ou processo externo, e sem
`os.Exit`, `fmt.Println` ou qualquer `io.Writer` de terminal (Princípios I e
VI, e requisito explícito do usuário para permitir reuso futuro por um adapter
HTTP)

**Escala/Escopo**: um arquivo de rastreamento de uma atividade individual
(corrida, pedalada ou caminhada) por execução, na casa de milhares a dezenas
de milhares de pontos

## Verificação da Constituição

*PORTÃO: Deve passar antes da Fase 0 de pesquisa. Reverificar após o design da Fase 1.*

| Princípio | Avaliação | Como o design atende |
|---|---|---|
| I. Arquitetura Hexagonal | PASSA | `internal/domain` contém só entidades (`TrackPoint`, `Track`, `Route`, `BoundingBox`), portas (`TrackParser`, `Simplifier`, `Smoother`) e funções puras (Haversine, elevação, bounding box, descarte). Nenhum desses arquivos importa `gpxgo`, `encoding/xml` para parsing real, ou qualquer pacote de `internal/infra`. |
| II. Portas para Toda Dependência Externa | PASSA | Parsing de GPX e os algoritmos de simplificação/suavização são acessados via `TrackParser`, `Simplifier` e `Smoother`, declaradas no domínio e implementadas em `internal/infra/outbound/*`. |
| III. Entrypoints Descartáveis | PASSA | `InspectTrackService` (em `internal/application`) recebe/devolve apenas DTOs de dados (`InspectTrackInput`/`Output`); a CLI depende apenas da interface e é o único adapter que existe hoje, mas o serviço não conhece Cobra, flags, nem formatação de texto — pronto para um adapter HTTP futuro sem alteração. |
| IV. Neutralidade Geográfica | PASSA | Nenhum dado de mapa/relevo/coordenada embutido; o cálculo de bounding box trata o antimeridiano algebricamente (unwrap de longitude), sem tabela ou suposição de região (ver `research.md` item 6). |
| V. Funcionamento Offline | PASSA | `gpxgo` opera sobre bytes já lidos do arquivo local; nenhuma chamada de rede em nenhum adapter desta etapa. |
| VI. Testes Automatizados no Núcleo | PASSA | Domínio testado por tabelas puras (sem I/O); `internal/application` testado com mocks de `TrackParser`/`Simplifier`/`Smoother` gerados por `go.uber.org/mock`; meta de cobertura alta no núcleo, com casos explícitos de antimeridiano e equador. |
| VII. Erros Sentinela no Domínio | PASSA | `ErrEmptyFile`, `ErrUnsupportedFormat`, `ErrInsufficientPoints`, `ErrInsufficientPointsAfterCleaning` declarados em `internal/domain`; a CLI os traduz em códigos de saída de processo (ver `contracts/cli.md`); a tradução para status HTTP fica pronta para um adapter futuro, sem exigir mudança no domínio. |
| VIII. Configuração Injetada | PASSA | Os limiares internos (mínimo de pontos, velocidade máxima plausível, nível padrão) são resolvidos por `internal/infra/outbound/config` e injetados no serviço por quem monta a CLI (`cmd/sobrevoo`); o núcleo nunca lê variável de ambiente, arquivo, ou flag diretamente. |
| IX. Organização de Portas, Service Layer e Mocks | PASSA | Nenhum `ports.go`/`interfaces.go`: `TrackParser` está em `track.go` (produz `Track`); `Simplifier` e `Smoother`, sem entidade dona, têm arquivo próprio (`simplification.go`, `smoothing.go`). `internal/application` expõe `InspectTrackService` (interface) implementada por `inspectTrackService`, construída por `NewInspectTrackService(...)`; a CLI depende só da interface. Mocks gerados com `go.uber.org/mock/mockgen` via `//go:generate` acima de cada interface, em `mock_domain`/`mock_application`. |
| X. Testes: Given/When/Then, Builders e Isolamento por Camada | PASSA | Todo teste usa `t.Run("should ...")` com comentários `// given`/`// when`/`// then`, sem tabelas de casos. Builders fluentes em `build_domain`/`build_application` constroem entidades e DTOs de teste. Cada camada é testada isolada: domínio/serviço com portas mockadas (`mock_domain`), CLI com `InspectTrackService` mockado (`mock_application`) — nunca com implementação real. |
| Idioma dos Artefatos | PASSA | Este plano, o `research.md`, `data-model.md`, `contracts/cli.md` e `quickstart.md` estão em português do Brasil; identificadores, nomes de pacote/arquivo e comentários de código (a serem escritos na fase de implementação) permanecerão em inglês. Adicionalmente, por decisão do usuário, todo o I/O em tempo de execução da própria ferramenta (valores de flag como `low`/`medium`/`high`, texto do resumo, mensagens de erro) também é em inglês — ver `research.md` item 11 e a nota de escopo em `spec.md`. |

Nenhuma violação identificada. A seção de Rastreamento de Complexidade não se
aplica a este plano.

## Estrutura do Projeto

### Documentação (desta funcionalidade)

```text
specs/001-gps-track-processing/
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
    └── main.go                              # ponto de entrada; monta config, adapters e o comando raiz Cobra

internal/
├── domain/
│   ├── track_point.go                       # entidade TrackPoint
│   ├── track.go                             # entidade Track (bruto) + Format + porta TrackParser (produz Track)
│   ├── route.go                             # entidade Route (tratado)
│   ├── bounding_box.go                      # entidade BoundingBox + ComputeBoundingBox
│   ├── level.go                             # enum Level (low/medium/high)
│   ├── discard_stats.go                     # DiscardStats
│   ├── cleaning.go                          # ReorderByTime, Discard* (funções puras)
│   ├── distance.go                          # Haversine, TotalDistance
│   ├── duration.go                          # Duration
│   ├── elevation.go                         # ElevationGain
│   ├── simplification.go                    # porta Simplifier (não pertence a uma única entidade)
│   ├── smoothing.go                         # porta Smoother (não pertence a uma única entidade)
│   ├── errors.go                            # erros sentinela (ErrEmptyFile, ErrUnsupportedFormat, ...)
│   ├── build_domain/                        # test data builders (ex.: TrackPointBuilder, TrackBuilder)
│   └── mock_domain/                         # mocks gerados (go.uber.org/mock), um arquivo por porta
│
├── application/
│   ├── inspect_track_service.go             # InspectTrackService (interface) + inspectTrackService + InspectTrackInput/Output
│   ├── build_application/                   # test data builders (ex.: InspectTrackOutputBuilder)
│   └── mock_application/                    # mock gerado (go.uber.org/mock) de InspectTrackService
│
└── infra/
    ├── outbound/
    │   ├── config/
    │   │   └── config.go                    # adapter que resolve os limiares internos (Princípio VIII)
    │   ├── trackparser/
    │   │   └── gpx.go                       # adapter TrackParser: valida conteúdo GPX e delega a tkrajina/gpxgo
    │   ├── simplifier/
    │   │   └── douglaspeucker/douglaspeucker.go  # adapter Simplifier
    │   └── smoother/
    │       └── catmullrom/catmullrom.go     # adapter Smoother
    │
    └── inbound/
        └── cli/
            ├── root.go                       # comando raiz Cobra
            └── inspect.go                    # comando `inspect`: flags → InspectTrackInput, formata InspectTrackOutput

test/
└── helper/
    └── gpx_fixture.go                        # geração de conteúdo GPX de teste (válido/inválido)
```

**Decisão de Estrutura**: projeto único em Go, seguindo a árvore hexagonal
descrita explicitamente pelo usuário (mesma organização de alto nível de
`waliqueiroz/mystery-gifter-api`, com simplificações). Não há necessidade de
frontend/backend separados nem de estrutura mobile — esta etapa é
exclusivamente uma CLI. Nomes de pacote, arquivo e identificadores em inglês,
conforme a política de idioma da constituição; os arquivos deste diretório de
especificação, em português.

Dentro de `internal/domain`, não existe nenhum arquivo com nome genérico como
`ports.go` ou `interfaces.go` — convenção também espelhada de
`waliqueiroz/mystery-gifter-api`: uma interface que manipula uma entidade
específica fica no arquivo dessa entidade (`TrackParser` em `track.go`, pois
produz `Track`); uma interface sem entidade dona ganha um arquivo próprio
nomeado pelo conceito que representa (`Simplifier` em `simplification.go`,
`Smoother` em `smoothing.go`). Os mocks seguem o mesmo padrão do repositório
de referência: gerados em `<pacote>/mock_<pacote>/`, um arquivo por
porta/serviço, via `//go:generate` posicionado junto da própria interface.

A camada de `internal/application` é a *service layer* do projeto — cada
caso de uso é modelado como uma interface exportada (`InspectTrackService`)
implementada por uma struct não exportada (`inspectTrackService`), construída
por `NewInspectTrackService(...)`, também espelhando
`waliqueiroz/mystery-gifter-api`. Isso permite que adapters de entrada (como
a CLI) dependam apenas da interface, e a substituam por um mock nos próprios
testes de unidade — os testes de `internal/infra/inbound/cli` nunca
executam a implementação real do serviço nem dos adapters de saída.

Testes de unidade seguem o formato given/when/then (comentários `// given`,
`// when`, `// then` dentro de cada `t.Run`), sem tabelas de casos
(`[]struct{...}`) — um `t.Run` por cenário, ainda que isso repita alguma
configuração entre cenários. Builders de dados de teste (`build_domain`,
`build_application`) substituem literais de struct repetidos quando isso
deixa o teste mais legível.

## Rastreamento de Complexidade

*Não aplicável — nenhuma violação da Verificação da Constituição foi identificada.*
