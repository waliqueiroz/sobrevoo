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
  `BoundingBox`, `Level`, `DiscardStats`, `GeoDataSource`, `GeoDataSummary`,
  `CoverageReport`, `TrackSummary`), construtores que carregam regra de
  negócio (`NewGeoDataSource`, `SummarizeTrack`), funções puras (distância
  de Haversine, ganho de elevação, duração, cálculo de bounding box —
  incluindo o "unwrap" de longitude no antimeridiano —, `ReorderByTime` e
  os `Discard*` compostos em `CleanTrack`, e o algoritmo de verificação de
  cobertura, `ComputeCoverage`), erros sentinela (`ErrEmptyFile`,
  `ErrUnsupportedFormat`, `ErrInsufficientPoints[AfterCleaning]`), e as
  portas `TrackParser`, `Simplifier`, `Smoother`, `GeoDataInspector`,
  `GeoDataRegistry`, `FileChecker`. Qualquer DTO de saída que não seja um
  valor trivial (ex.: `GeoDataSummary`, `CoverageReport`, `TrackSummary`)
  também é um tipo de domínio comum — não um DTO de `internal/application`
  — e qualquer lógica não trivial (construir uma entidade, calcular algo a
  partir de uma coleção) é construtor ou função pura de domínio, nunca um
  helper solto na camada de aplicação; ver "Onde vive a regra de negócio"
  abaixo.
- **`internal/application`** — a *service layer*. Orquestra portas e
  construtores/funções puras do domínio; não conhece Cobra, arquivo, nem
  código de saída, e não decide nenhuma regra de negócio por conta própria
  — só decide qual porta/função de domínio chamar, e em qual ordem.
- **`internal/infra/outbound/*`** — adapters que implementam as portas do
  domínio: `trackparser` (GPX via `tkrajina/gpxgo`),
  `simplifier/douglaspeucker`, `smoother/catmullrom`, `config` (limiares
  internos fixos: mínimo de pontos, velocidade máxima plausível, nível
  padrão — ainda sem fonte de configuração externa, mas o ponto de extensão
  já existe, conforme o Princípio VIII da constituição).
- **`internal/infra/inbound/cli`** — o(s) comando(s) Cobra, e o lugar que
  traduz erros sentinela do domínio em códigos de saída de processo
  (`exit_code.go`); ver `specs/001-gps-track-processing/contracts/cli.md` e
  `specs/002-geo-data-registry/contracts/cli.md` para o mapeamento exato.
  Na etapa 1, era também o único lugar que tocava o filesystem (`os.Open`,
  para obter o `io.Reader` que `TrackParser` espera). A partir da etapa 2
  isso não é mais universal: adapters de saída que precisam de acesso
  posicional a um arquivo — `geodatainspector` (lê SQLite/TIFF por
  caminho), `geodatastore/jsonfile` (lê/escreve o registro) e `filechecker`
  (`os.Stat`) — abrem o arquivo eles mesmos, dado apenas o caminho; a CLI
  continua sendo quem abre o arquivo só quando o método do serviço exige um
  `io.Reader` (`register` não abre nada, pois passa um caminho;
  `check` abre, pois `GeoDataService.CheckCoverage` exige um `Reader`,
  igual a `inspect`). Ambos os padrões respeitam os Princípios I e II da
  constituição — é só uma questão de qual adapter concreto faz a chamada de
  I/O real (`specs/002-geo-data-registry/research.md`, item 8).
- **`cmd/sobrevoo/main.go`** — composition root; o único lugar que conecta
  todos os adapters concretos entre si.

### Portas e nomenclatura da service layer (Princípios I, II e IX da constituição)

- Nada de `ports.go`/`interfaces.go`. Uma porta ligada a uma única entidade
  fica no arquivo dessa entidade (`TrackParser` é declarada em `track.go`,
  já que produz `Track`). Uma porta sem entidade dona ganha seu próprio
  arquivo, nomeado pelo conceito que representa (`Simplifier` em
  `simplification.go`, `Smoother` em `smoothing.go`).
- `XService` (interface exportada) / `xService` (struct não exportada) /
  `NewXService(...)` (construtor) é **um serviço por recurso/agregado, não
  um serviço por caso de uso**: `X` nomeia o que o serviço gerencia (ex.:
  `GeoDataService`), e cada caso de uso vira um método nomeado pela
  operação (`Register`, `List`, `Remove`, `CheckCoverage` — nunca um
  `Execute` genérico, nunca uma interface por método). Vários casos de uso
  que operam sobre o mesmo recurso pertencem à mesma interface e à mesma
  struct — padrão espelhado de `waliqueiroz/mystery-gifter-api`
  (`internal/application/group_service.go`: `GroupService` reúne
  `Create`/`GetByID`/`Search`/`AddUser`/`RemoveUser`/`GenerateMatches`/
  `Reopen`/`Archive`/`GetUserMatch`). Um serviço pode depender de outro
  serviço de aplicação (não só de portas do domínio) quando isso faz
  sentido — ver `GroupInviteService` dependendo de `UserService` no mesmo
  repositório de referência. Adapters de entrada dependem só da interface,
  nunca da struct concreta. (`InspectTrackService`/`GeoDataService`, em
  `internal/application`, seguem esse padrão; ver
  `specs/002-geo-data-registry/research.md` item 13 para o histórico dessa
  decisão.)
- **Onde vive a regra de negócio**: DTOs de saída não triviais e qualquer
  lógica que não seja "chamar uma porta na ordem certa" vivem em
  `internal/domain`, nunca em `internal/application` — nem como DTO
  próprio da camada de aplicação, nem como função solta no pacote
  `application`. Um método de serviço busca/checa via porta, delega a
  regra para um construtor (`domain.NewGeoDataSource`) ou função pura de
  domínio (`domain.ComputeCoverage`), e devolve o resultado — a mesma
  divisão de `GroupService.AddUser` (busca via repositório, delega para
  `domain.Group.AddUser`) em `waliqueiroz/mystery-gifter-api`. Dentro do
  domínio, tanto faz a lógica virar método de uma entidade
  (`Group.AddUser`, `Group.GenerateMatches`) quanto função livre operando
  sobre coleções (`ComputeBoundingBox`, `Haversine`, `ComputeCoverage`) —
  o segundo já é o padrão deste projeto desde a etapa 1, e o repositório de
  referência do usuário usa os dois conforme o caso. Ver
  `specs/002-geo-data-registry/research.md` item 14 para o histórico dessa
  decisão.
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
