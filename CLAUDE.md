# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Projeto

Sobrevoo é uma ferramenta de linha de comando pessoal e open source, em Go,
que vai gerar vídeos de sobrevoo a partir de trajetos GPS (no estilo
Relive/Strava). A única feature implementada até agora
(`specs/001-gps-track-processing/`) lê um trajeto GPX, trata ele (descarta
pontos inválidos, reordena por tempo), reduz/suaviza o traçado, e imprime um
resumo — ainda sem mapa, câmera ou renderização de vídeo.

**A constituição do projeto (`.specify/memory/constitution.md`) é
vinculante.** Ela é curta — leia antes de fazer mudanças estruturais. As
seções abaixo são um resumo de consulta rápida dela, não um substituto.

## Comandos

```sh
make build      # go build -o bin/sobrevoo ./cmd/sobrevoo
make test       # go test ./... -cover
make generate   # go generate ./...  (regenera os mocks; precisa do mockgen, declarado como tool no go.mod)
make lint       # go vet ./...

# Rodar um teste específico (por pacote + nome do teste/subteste):
go test ./internal/domain/... -run Test_Haversine -v
go test ./internal/infra/inbound/cli/... -run 'Test_InspectCommand_Execute/should_map_ErrEmptyFile' -v

# Rodar a CLI direto, sem compilar um binário:
go run ./cmd/sobrevoo inspect path/to/track.gpx --simplification=low --smoothing=high
```

## Arquitetura

Hexagonal / portas e adapters. O núcleo (`internal/domain` +
`internal/application`) nunca importa nada de `internal/infra` — toda
dependência externa (parsing de GPX, algoritmos de simplificação/suavização,
configuração, terminal) é acessada por uma porta implementada por um
adapter.

- **`internal/domain`** — entidades (`TrackPoint`, `Track`, `Route`,
  `BoundingBox`, `Level`, `DiscardStats`), funções puras (distância de
  Haversine, ganho de elevação, duração, cálculo de bounding box —
  incluindo o "unwrap" de longitude no antimeridiano — e as funções de
  limpeza de pontos), erros sentinela (`ErrEmptyFile`,
  `ErrUnsupportedFormat`, `ErrInsufficientPoints[AfterCleaning]`), e as
  portas `TrackParser`, `Simplifier`, `Smoother`.
- **`internal/application`** — a *service layer*. Uma interface exportada
  por caso de uso (`InspectTrackService`), implementada por uma struct não
  exportada (`inspectTrackService`), construída por
  `NewInspectTrackService(...)`. Orquestra portas e funções puras do
  domínio; não conhece Cobra, arquivo, nem código de saída.
- **`internal/infra/outbound/*`** — adapters que implementam as portas do
  domínio: `trackparser` (GPX via `tkrajina/gpxgo`),
  `simplifier/douglaspeucker`, `smoother/catmullrom`, `config` (limiares
  internos fixos: mínimo de pontos, velocidade máxima plausível, nível
  padrão — ainda sem fonte de configuração externa, mas o ponto de extensão
  já existe, conforme o Princípio VIII da constituição).
- **`internal/infra/inbound/cli`** — o(s) comando(s) Cobra. O único lugar
  que toca o filesystem (`os.Open`) e traduz erros sentinela do domínio em
  códigos de saída de processo (`exit_code.go`); ver
  `specs/001-gps-track-processing/contracts/cli.md` para o mapeamento
  exato.
- **`cmd/sobrevoo/main.go`** — composition root; o único lugar que conecta
  todos os adapters concretos entre si.

### Portas e nomenclatura da service layer (Princípios I, II e IX da constituição)

- Nada de `ports.go`/`interfaces.go`. Uma porta ligada a uma única entidade
  fica no arquivo dessa entidade (`TrackParser` é declarada em `track.go`,
  já que produz `Track`). Uma porta sem entidade dona ganha seu próprio
  arquivo, nomeado pelo conceito que representa (`Simplifier` em
  `simplification.go`, `Smoother` em `smoothing.go`).
- Casos de uso são `XService` (interface exportada) / `xService` (struct
  não exportada) / `NewXService(...)` (construtor). Adapters de entrada
  dependem só da interface, nunca da struct concreta.
- Mocks são gerados com `go.uber.org/mock/mockgen` via diretiva
  `//go:generate` posicionada diretamente acima da interface que ela
  mocka — nunca em um arquivo central. A saída vai para um subpacote
  irmão `mock_<pacote>` (`internal/domain/mock_domain`,
  `internal/application/mock_application`), um arquivo gerado por
  interface. Rode `make generate` depois de adicionar ou alterar uma
  porta/interface.

### Testes (Princípio X da constituição)

- given/when/then: todo teste é
  `t.Run("should ...", func(t *testing.T) { // given ... // when ... // then ... })`.
- Nada de testes tabulares (`[]struct{...}` + `for`) — um cenário, um
  `t.Run`, mesmo que isso repita configuração.
- Test data builders vivem em subpacotes `build_<pacote>`
  (`internal/domain/build_domain`, `internal/application/build_application`):
  `NewXBuilder()` com defaults sensatos, `WithCampo(...)`/`WithoutCampo()`
  fluentes, `Build()` terminal. Use um sempre que um literal de struct
  repetido ou grande demais deixaria o teste poluído.
- Cada camada é testada isolada, com o que ela depende mockado: os testes
  de domain/application mockam as portas do domínio (`mock_domain`); os
  testes de `internal/infra/inbound/cli` mockam
  `application.InspectTrackService` (`mock_application`) e nunca conectam
  um serviço ou adapter de saída real. Não existe teste automatizado de
  ponta a ponta — `specs/<feature>/quickstart.md` é o checklist manual, com
  o binário real, pra isso.

### Detalhe do Go 1.26

Este módulo usa Go 1.26, que estendeu o builtin `new` para aceitar uma
expressão, não só um tipo: `new(x)` devolve um `*T` apontando pra uma cópia
de `x`. O código usa isso diretamente (ex.: `Elevation: new(50.0)` em
testes, ou `Time: new(someTime)`) em vez de um pacote helper do tipo
`ptr.Of[T]` escrito à mão — não reintroduza um.

### Fluxo do Spec Kit

O desenvolvimento de features passa por `specs/<NNN-feature-name>/`
(`spec.md`, `plan.md`, `tasks.md`, `research.md`, `data-model.md`,
`contracts/`, `quickstart.md`), conduzido pelos slash commands `/speckit-*`
(`/speckit-specify`, `/speckit-clarify`, `/speckit-plan`, `/speckit-tasks`,
`/speckit-implement`, ...). Todos os artefatos do Spec Kit e toda a
comunicação durante esse fluxo DEVEM ser em português do Brasil
(constituição: "Stack Tecnológica e Idioma dos Artefatos") — código-fonte,
identificadores, nomes de pacote/arquivo, nomes de branch, mensagens de
commit e comentários de código permanecem em inglês; termos técnicos já
consagrados não são traduzidos em nenhuma direção.
