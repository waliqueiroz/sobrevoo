---

description: "Task list template for feature implementation"
---

# Tarefas: Leitura e Tratamento de Trajeto GPS

**Entrada**: Documentos de design de `/specs/001-gps-track-processing/`

**Pré-requisitos**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/cli.md`, `quickstart.md`

**Testes**: incluídas. Diferente do padrão "testes são opcionais" deste
template, a constituição do projeto (Princípio VI — Testes Automatizados no
Núcleo) exige testify + uber-go/mock e cobertura alta no núcleo
(`internal/domain` + `internal/application`) sem tocar disco, rede ou
processo externo. As tarefas de teste abaixo materializam essa exigência.

**Organização**: as tarefas são agrupadas por história de usuário (P1–P4 de
`spec.md`) para permitir implementação e teste independentes de cada
história.

## Formato: `[ID] [P?] [Story] Descrição`

- **[P]**: pode ser executado em paralelo (arquivos diferentes, sem
  dependência de tarefa incompleta). Tarefas que editam o mesmo arquivo NUNCA
  são marcadas `[P]` entre si, mesmo quando logicamente independentes.
- **[Story]**: a qual história de usuário esta tarefa pertence (US1, US2,
  US3, US4). Tarefas de Setup, Foundational e Polish não têm esse rótulo.
- Toda tarefa inclui o caminho de arquivo exato a criar/editar.

## Convenções de Caminho

Projeto único em Go, estrutura hexagonal definida em `plan.md`:
`cmd/sobrevoo/`, `internal/domain/`, `internal/application/`,
`internal/infra/outbound/`, `internal/infra/inbound/cli/`, `pkg/`,
`test/helper/`.

---

## Phase 1: Setup (Shared Infrastructure)

**Propósito**: inicialização do módulo Go e da estrutura de pastas.

- [X] T001 Inicializar o módulo Go `github.com/waliqueiroz/sobrevoo` (Go 1.26) em `go.mod` na raiz do repositório
- [X] T002 Adicionar as dependências `github.com/spf13/cobra`, `github.com/tkrajina/gpxgo`, `github.com/stretchr/testify` e `go.uber.org/mock` via `go get`, atualizando `go.mod`/`go.sum` (depende de T001)
- [X] T003 [P] Criar o esqueleto de diretórios e arquivos de pacote (`package X` vazios) para `cmd/sobrevoo/`, `internal/domain/`, `internal/application/`, `internal/infra/outbound/{config,trackparser,simplifier/douglaspeucker,smoother/catmullrom}/`, `internal/infra/inbound/cli/`, `internal/domain/mock_domain/`, `pkg/ptr/`, `test/helper/`, conforme a árvore em `plan.md` (depende de T001)
- [X] T004 [P] Criar `Makefile` na raiz com os alvos `build` (`go build -o bin/sobrevoo ./cmd/sobrevoo`), `test` (`go test ./... -cover`), `generate` (`go generate ./...`) e `lint` (`go vet ./...`) (depende de T001)

**Checkpoint**: módulo Go pronto, com dependências e esqueleto de pastas.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Propósito**: entidades, portas e funções puras do domínio, o adapter GPX, a
configuração interna e o esqueleto do caso de uso `InspectTrack` — a base que
TODA história de usuário (US1–US4) precisa para funcionar, mesmo que ainda
sem nenhum entrypoint executável.

**⚠️ CRÍTICO**: nenhuma tarefa de história de usuário pode começar até que
esta fase esteja completa.

- [X] T005 [P] Criar a entidade `TrackPoint` em `internal/domain/track_point.go` com os campos `Latitude float64`, `Longitude float64`, `Elevation *float64` (nil quando o arquivo não traz altitude) e `Time *time.Time` (nil quando o arquivo não traz tempo), conforme `data-model.md`
- [X] T006 [P] Criar o enum `Level` em `internal/domain/level.go` com os valores `LevelLow`, `LevelMedium`, `LevelHigh` (data-model.md)
- [X] T007 [P] Criar `DiscardStats` em `internal/domain/discard_stats.go` com os campos inteiros `ImpossibleCoordinates`, `ConsecutiveDuplicates` e `ImplausibleJumps` (data-model.md, FR-011)
- [X] T008 [P] Declarar os erros sentinela em `internal/domain/errors.go`: `ErrEmptyFile` (FR-005), `ErrUnsupportedFormat` (FR-004), `ErrInsufficientPoints` (FR-006) e `ErrInsufficientPointsAfterCleaning` (FR-006) — Princípio VII da constituição
- [X] T009 Criar em `internal/domain/track.go`: o enum `Format` (valor `FormatGPX`), a entidade `Track` (`Format Format`, `Points []TrackPoint`) e a porta `TrackParser` (`Parse(r io.Reader) (Track, error)`) com a diretiva `//go:generate go run go.uber.org/mock/mockgen -destination mock_domain/track_parser.go . TrackParser` posicionada diretamente acima da interface (depende de T005)
- [X] T010 [P] Criar a entidade `Route` (`Points []TrackPoint`) em `internal/domain/route.go` (depende de T005)
- [X] T011 [P] Implementar `Haversine` (distância em metros entre dois pontos) e `TotalDistance` (soma ao longo da rota) em `internal/domain/distance.go`, com testes de tabela em `internal/domain/distance_test.go` cobrindo cruzamento de antimeridiano e de equador (FR-017, research.md item 6) (depende de T005)
- [X] T012 [P] Implementar `ElevationGain` (soma dos deltas positivos de altitude; segundo retorno `bool` indica se havia dado de altitude) em `internal/domain/elevation.go`, com testes em `internal/domain/elevation_test.go` incluindo o caso de nenhum ponto ter altitude (FR-018, FR-019) (depende de T005)
- [X] T013 [P] Implementar `Duration` (diferença entre o tempo do primeiro e do último ponto; segundo retorno `bool` indica se havia dado de tempo) em `internal/domain/duration.go` — arquivo adicional ao listado em `plan.md`, seguindo o mesmo padrão de `distance.go`/`elevation.go` —, com testes em `internal/domain/duration_test.go` incluindo o caso de nenhum ponto ter tempo (FR-020, FR-021) (depende de T005)
- [X] T014 [P] Implementar a entidade `BoundingBox` e `ComputeBoundingBox` em `internal/domain/bounding_box.go`, aplicando o algoritmo de "unwrap" de longitude descrito em `research.md` item 6 (campo `CrossesAntimeridian`), com testes em `internal/domain/bounding_box_test.go` cobrindo: trajeto que cruza o antimeridiano, trajeto que cruza o equador, e trajeto que não cruza nenhum dos dois (FR-022, FR-023, FR-024) (depende de T005)
- [X] T015 [P] Gerar o mock de `TrackParser` executando `go generate ./internal/domain/...`, produzindo `internal/domain/mock_domain/track_parser.go` (depende de T009)
- [X] T016 [P] Criar fixtures de GPX de teste em `test/helper/gpx_fixture.go`: um GPX válido com altitude e tempo em todos os pontos, um GPX válido sem altitude, um GPX válido sem tempo, um conteúdo que não é GPX, e um XML malformado
- [X] T017 Implementar o adapter `TrackParser` em `internal/infra/outbound/trackparser/gpx.go`: ler todo o conteúdo, verificar se o primeiro `xml.StartElement` tem nome local `gpx` (senão retornar `domain.ErrUnsupportedFormat`), e delegar o parsing a `tkrajina/gpxgo`, com testes em `internal/infra/outbound/trackparser/gpx_test.go` usando as fixtures de T016 (FR-002, FR-003, FR-004, research.md itens 2–3) (depende de T009, T016)
- [X] T018 [P] Implementar o adapter de configuração em `internal/infra/outbound/config/config.go`: `Config` com `MinPoints int` (2), `MaxPlausibleSpeedKmh float64` (130) e `DefaultLevel domain.Level` (`domain.LevelMedium`), e uma função `Load() Config` que devolve esses valores fixos (Princípio VIII, research.md itens 7 e 9) (depende de T006)
- [X] T019 Implementar o esqueleto do caso de uso `InspectTrack` em `internal/application/inspect_track.go`: DTOs `InspectTrackInput` (`Reader io.Reader`, `SimplificationLevel domain.Level`, `SmoothingLevel domain.Level`) e `InspectTrackOutput` (todos os campos de `data-model.md`); construtor recebendo `domain.TrackParser` e `config.Config`; `Execute` fazendo parse via a porta, construindo uma `Route` diretamente a partir de `Track.Points` (sem tratamento ainda) e calculando `BoundingBox`/`TotalDistance`/`ElevationGain`/`Duration`; testes em `internal/application/inspect_track_test.go` usando o mock de T015 (depende de T009, T010, T011, T012, T013, T014, T015, T018)

**Checkpoint**: núcleo completo e testável (parser, entidades, funções puras,
caso de uso), mas ainda sem nenhum entrypoint executável.

---

## Phase 3: User Story 1 - Resumo de um trajeto válido (Priority: P1) 🎯 MVP

**Objetivo**: o usuário aponta a CLI para um arquivo GPX válido e recebe um
resumo em `stdout` com formato, contagem de pontos, distância, elevação,
duração e área geográfica — incluindo os casos de dado ausente e de trajeto
cruzando o antimeridiano.

**Teste Independente**: rodar `sobrevoo inspect <arquivo.gpx>` com um GPX
válido e completo e verificar que todos os campos do resumo aparecem com
valores coerentes; repetir com GPX sem altitude/sem tempo e com um GPX que
cruza o antimeridiano.

### Implementação da História de Usuário 1

- [X] T020 [P] [US1] Criar o helper genérico `Of[T](v T) *T` em `pkg/ptr/ptr.go`, com teste em `pkg/ptr/ptr_test.go`
- [X] T021 [P] [US1] Implementar o comando raiz Cobra `sobrevoo` em `internal/infra/inbound/cli/root.go` (depende de T002)
- [X] T022 [US1] Implementar o comando `inspect` em `internal/infra/inbound/cli/inspect.go`: argumento posicional `<arquivo>`, abre o arquivo com `os.Open`, chama `InspectTrack.Execute`, formata `InspectTrackOutput` no texto em inglês descrito em `contracts/cli.md` (formato, pontos original/tratado, distância em km, elevação ou indicação de ausência, duração ou indicação de ausência, área geográfica) e escreve em `cmd.OutOrStdout()` (FR-025) (depende de T019, T021)
- [X] T023 [US1] Implementar `cmd/sobrevoo/main.go`: montar `config.Load()`, o adapter GPX (T017), o caso de uso `InspectTrack` (T019) e o comando raiz (T022), então chamar `Execute()` (depende de T017, T018, T019, T022)
- [X] T024 [US1] Teste de contrato em `internal/infra/inbound/cli/inspect_test.go`: GPX válido com altitude e tempo em todos os pontos → todos os campos de FR-025 aparecem em `stdout` e código de saída `0` (depende de T023)
- [X] T025 [US1] Teste de integração em `internal/infra/inbound/cli/inspect_test.go`: GPX sem altitude → resumo indica explicitamente a ausência de ganho de elevação, nunca um valor calculado (FR-019) (depende de T023)
- [X] T026 [US1] Teste de integração em `internal/infra/inbound/cli/inspect_test.go`: GPX sem tempo → resumo indica explicitamente a ausência de duração (FR-021) (depende de T023)
- [X] T027 [US1] Teste de integração em `internal/infra/inbound/cli/inspect_test.go`: GPX com pontos alternando entre longitudes próximas de `+179.9` e `-179.9` → distância total coerente e área geográfica reportada como atravessando o antimeridiano (FR-023) (depende de T023)

**Checkpoint**: História de Usuário 1 completa, demonstrável de ponta a ponta
(MVP).

---

## Phase 4: User Story 2 - Recusa clara de arquivo inválido (Priority: P2)

**Objetivo**: arquivo vazio, em formato não reconhecido, ou com pontos
insuficientes (antes ou depois da limpeza) é recusado com mensagem clara e
código de saída específico, sem nenhum resumo parcial.

**Teste Independente**: rodar `sobrevoo inspect` com um arquivo vazio, um
arquivo de formato não reconhecido, um arquivo com um único ponto, e um
arquivo cujos pontos válidos somem para menos de 2 após a limpeza — cada caso
deve produzir uma mensagem de erro específica em `stderr` e um código de saída
distinto, sem resumo.

### Implementação da História de Usuário 2

- [X] T028 [US2] Implementar em `internal/domain/cleaning.go` as funções puras `ReorderByTime` (reordena por `Time` apenas quando TODOS os pontos o possuem; caso contrário devolve os pontos inalterados — FR-027, research.md item 8), `DiscardImpossibleCoordinates` (FR-008), `DiscardConsecutiveDuplicates` (FR-009) e `DiscardImplausibleJumps` (usa `Haversine` e o `MaxPlausibleSpeedKmh` recebido por parâmetro — FR-010), com testes de tabela em `internal/domain/cleaning_test.go` cobrindo o caso básico de cada função (depende de T005, T007, T011)
- [X] T029 [US2] Estender `InspectTrack.Execute` em `internal/application/inspect_track.go`: retornar `domain.ErrEmptyFile` quando `Track.Points` estiver vazio logo após o parse; retornar `domain.ErrInsufficientPoints` quando houver menos de `Config.MinPoints` pontos antes da limpeza; rodar `ReorderByTime` e as três funções de descarte (usando `Config.MaxPlausibleSpeedKmh`); retornar `domain.ErrInsufficientPointsAfterCleaning` quando restarem menos de `Config.MinPoints` pontos após a limpeza (depende de T019, T028)
- [X] T030 [US2] Estender o comando `inspect` em `internal/infra/inbound/cli/inspect.go` para mapear `domain.ErrEmptyFile`→código 1, `domain.ErrUnsupportedFormat`→código 2, `domain.ErrInsufficientPoints` e `domain.ErrInsufficientPointsAfterCleaning`→código 3, e erro de I/O ao abrir o arquivo→código 4, escrevendo a mensagem em inglês em `stderr` (nenhum resumo impresso) — conforme a tabela de `contracts/cli.md` (depende de T022, T029)
- [X] T031 [US2] Teste de contrato em `internal/infra/inbound/cli/inspect_test.go`: arquivo vazio → mensagem em `stderr` e código de saída `1` (depende de T030)
- [X] T032 [US2] Teste de contrato em `internal/infra/inbound/cli/inspect_test.go`: conteúdo que não corresponde a GPX → mensagem em `stderr` e código de saída `2` (depende de T030)
- [X] T033 [US2] Teste de contrato em `internal/infra/inbound/cli/inspect_test.go`: arquivo original com menos de 2 pontos válidos → mensagem em `stderr` e código de saída `3` (depende de T030)
- [X] T034 [US2] Teste de contrato em `internal/infra/inbound/cli/inspect_test.go`: arquivo com pontos originais suficientes mas que ficam abaixo de 2 após a limpeza → mensagem em `stderr` menciona explicitamente que a insuficiência ocorreu após a limpeza, código de saída `3`, nenhum resumo parcial (depende de T030)

**Checkpoint**: Histórias de Usuário 1 e 2 funcionam de forma independente.

---

## Phase 5: User Story 3 - Tratamento de pontos problemáticos (Priority: P3)

**Objetivo**: o resumo reporta quantos pontos foram descartados por cada
motivo (coordenada impossível, duplicado consecutivo, salto implausível), a
reordenação por tempo acontece antes da detecção de duplicados/saltos, e a
distância total não é afetada pelos pontos descartados.

**Teste Independente**: processar um arquivo com pontos deliberadamente
problemáticos dos três tipos e verificar que o resumo relata a contagem
correta por motivo e que a distância total exclui os pontos descartados.

### Implementação da História de Usuário 3

- [X] T035 [US3] Estender `InspectTrack.Execute` em `internal/application/inspect_track.go` para acumular um `domain.DiscardStats` enquanto executa cada função de descarte de T028, atribuindo o resultado a `InspectTrackOutput.Discarded` (FR-011) (depende de T029)
- [X] T036 [US3] Estender a formatação de saída do comando `inspect` em `internal/infra/inbound/cli/inspect.go` para imprimir a contagem de pontos descartados por motivo (depende de T035)
- [X] T037 [US3] Adicionar em `internal/domain/cleaning_test.go` testes cobrindo múltiplos pontos por motivo: várias coordenadas impossíveis na mesma rota, uma sequência de mais de dois duplicados consecutivos, e um salto implausível calculado via `Haversine` + intervalo de tempo entre os pontos (depende de T028)
- [X] T038 [US3] Teste de integração em `internal/infra/inbound/cli/inspect_test.go`: GPX com pontos fora de ordem cronológica → pontos são reordenados antes da detecção de duplicados/saltos e o resumo reflete a rota já reordenada (FR-027) (depende de T036)
- [X] T039 [US3] Teste de integração em `internal/infra/inbound/cli/inspect_test.go`: GPX combinando os três tipos de ponto problemático → resumo relata a contagem correta por motivo e a distância total exclui os pontos descartados (depende de T036)

**Checkpoint**: Histórias de Usuário 1, 2 e 3 funcionam de forma
independente.

---

## Phase 6: User Story 4 - Redução e suavização do traçado (Priority: P4)

**Objetivo**: o usuário ajusta os níveis de simplificação e suavização
(`low`/`medium`/`high`, padrão `medium`) via flags da CLI, e o traçado tratado
reflete o nível escolhido, preservando o formato geral do trajeto.

**Teste Independente**: processar o mesmo arquivo com `--simplification=low`
vs. `--simplification=high` (e o equivalente para `--smoothing`) e verificar
que o nível mais alto produz, respectivamente, menos pontos e menos variação
brusca ponto a ponto.

### Implementação da História de Usuário 4

- [X] T040 [P] [US4] Declarar a porta `Simplifier` (`Simplify(points []TrackPoint, level Level) []TrackPoint`) em `internal/domain/simplification.go`, com a diretiva `//go:generate go run go.uber.org/mock/mockgen -destination mock_domain/simplifier.go . Simplifier` (FR-012, FR-014)
- [X] T041 [P] [US4] Declarar a porta `Smoother` (`Smooth(points []TrackPoint, level Level) []TrackPoint`) em `internal/domain/smoothing.go`, com a diretiva `//go:generate go run go.uber.org/mock/mockgen -destination mock_domain/smoother.go . Smoother` (FR-013, FR-015)
- [X] T042 [US4] Gerar os mocks de `Simplifier` e `Smoother` executando `go generate ./internal/domain/...`, produzindo `internal/domain/mock_domain/simplifier.go` e `internal/domain/mock_domain/smoother.go` (depende de T040, T041)
- [X] T043 [P] [US4] Implementar o adapter Douglas-Peucker para `Simplifier` em `internal/infra/outbound/simplifier/douglaspeucker/douglaspeucker.go` (nível → tolerância), com testes em `douglaspeucker_test.go` verificando que a quantidade de pontos diminui à medida que o nível aumenta e que os pontos inicial/final são preservados (depende de T040)
- [X] T044 [P] [US4] Implementar o adapter Catmull-Rom para `Smoother` em `internal/infra/outbound/smoother/catmullrom/catmullrom.go` (nível → intensidade de suavização), com testes em `catmullrom_test.go` verificando que a variação ponto a ponto diminui à medida que o nível aumenta (depende de T041)
- [X] T045 [US4] Estender `InspectTrack` (construtor + `Execute`) em `internal/application/inspect_track.go` para receber `Simplifier` e `Smoother`, aplicar `Simplify` e depois `Smooth` após a limpeza (usando `InspectTrackInput.SimplificationLevel`/`SmoothingLevel`, com `domain.LevelMedium` como padrão quando não informado — FR-016) antes de calcular as estatísticas finais; testes usando os mocks de T042 (depende de T035, T042)
- [X] T046 [US4] Adicionar as flags `--simplification` e `--smoothing` (aceitando `low`/`medium`/`high`, padrão `medium`) ao comando `inspect` em `internal/infra/inbound/cli/inspect.go`, convertendo o valor da flag para `domain.Level` (depende de T030)
- [X] T047 [US4] Conectar os adapters Douglas-Peucker e Catmull-Rom em `cmd/sobrevoo/main.go` (depende de T043, T044, T045)
- [X] T048 [US4] Teste de contrato em `internal/infra/inbound/cli/inspect_test.go`: valor de flag fora de `low`/`medium`/`high` → erro de uso do Cobra, código de saída `2` (depende de T046)
- [X] T049 [US4] Teste de integração em `internal/infra/inbound/cli/inspect_test.go`: mesmo arquivo processado com `--simplification=low` e com `--simplification=high` → `high` produz menos pontos que `low` (SC-006) (depende de T047)
- [X] T050 [US4] Teste de integração em `internal/infra/inbound/cli/inspect_test.go`: mesmo arquivo processado com `--smoothing=low` e com `--smoothing=high` → `high` produz visivelmente menos variação brusca ponto a ponto (SC-007) (depende de T047)

**Checkpoint**: todas as quatro histórias de usuário funcionam de forma
independente.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Propósito**: qualidade final que atravessa todas as histórias.

- [X] T051 [P] Rodar `go vet ./...` e corrigir qualquer problema encontrado no repositório
- [X] T052 [P] Adicionar comentários de documentação de pacote (`// Package ...`) em `internal/domain`, `internal/application`, cada pacote de `internal/infra/outbound/*` e `internal/infra/inbound/cli`
- [X] T053 Executar manualmente todos os cenários de `quickstart.md` contra o binário compilado (`make build`) e registrar qualquer divergência encontrada
- [X] T054 [P] Rodar `go test ./... -cover` e confirmar cobertura alta em `internal/domain` e `internal/application` (Princípio VI da constituição), adicionando casos de tabela que faltarem

---

## Dependências e Ordem de Execução

### Dependências entre Fases

- **Setup (Phase 1)**: sem dependências — pode começar imediatamente.
- **Foundational (Phase 2)**: depende da conclusão do Setup — BLOQUEIA todas
  as histórias de usuário.
- **User Stories (Phase 3–6)**: todas dependem da conclusão da fase
  Foundational.
  - US1 → US2 → US3 → US4 seguem essa ordem de prioridade (P1→P4) porque cada
    uma estende os mesmos arquivos (`inspect_track.go`, `inspect.go`,
    `main.go`) construídos pela anterior — não são paralelizáveis entre si
    apesar de cada uma ser independentemente testável ao final de sua fase.
  - Particularmente: US2 (T028) introduz as funções de limpeza porque sua
    própria história exige detectar "pontos insuficientes após a limpeza"
    (FR-006); US3 reaproveita essas funções para expor a contagem detalhada
    por motivo (FR-011) exigida por sua história.
- **Polish (Phase 7)**: depende da conclusão de todas as histórias de usuário
  desejadas.

### Dentro de Cada História de Usuário

- Funções/entidades de domínio antes do caso de uso; caso de uso antes do
  adapter de CLI; adapter de CLI antes dos testes de contrato/integração que
  o exercitam.
- Testes que compartilham o mesmo arquivo (`inspect_test.go`,
  `cleaning_test.go`) são executados em sequência, mesmo quando cobrem
  cenários logicamente independentes.

### Oportunidades de Paralelização

- Todas as tarefas de Setup marcadas `[P]` (T003, T004).
- Dentro do Foundational: T005–T008 (arquivos de entidade/valor
  independentes); depois T010–T014 (cada uma depende só de T005, arquivos
  diferentes); T015 e T016 podem rodar em paralelo com T018.
- Dentro de US1: T020 e T021 (arquivos diferentes, sem dependência mútua).
- Dentro de US4: T040/T041 (portas em arquivos diferentes); depois T043/T044
  (adapters em arquivos diferentes).

---

## Exemplo de Paralelização: Foundational

```bash
# Disparar as entidades/valores independentes juntos:
Task: "Criar TrackPoint em internal/domain/track_point.go"
Task: "Criar enum Level em internal/domain/level.go"
Task: "Criar DiscardStats em internal/domain/discard_stats.go"
Task: "Declarar erros sentinela em internal/domain/errors.go"

# Depois, disparar as funções puras que só dependem de TrackPoint:
Task: "Implementar Haversine/TotalDistance em internal/domain/distance.go"
Task: "Implementar ElevationGain em internal/domain/elevation.go"
Task: "Implementar Duration em internal/domain/duration.go"
Task: "Implementar BoundingBox/ComputeBoundingBox em internal/domain/bounding_box.go"
```

---

## Estratégia de Implementação

### MVP Primeiro (Somente História de Usuário 1)

1. Completar Phase 1: Setup.
2. Completar Phase 2: Foundational (CRÍTICO — bloqueia todas as histórias).
3. Completar Phase 3: História de Usuário 1.
4. **PARAR E VALIDAR**: rodar os Cenários 1–3 de `quickstart.md`.
5. Nesse ponto já existe um binário `sobrevoo inspect` funcional (MVP).

### Entrega Incremental

1. Setup + Foundational → núcleo pronto, nada executável ainda.
2. + US1 → CLI funcional para o caso feliz, com dados ausentes e
   antimeridiano (MVP) → validar com Cenários 1–3 de `quickstart.md`.
3. + US2 → recusa robusta de entradas ruins → validar com Cenários 4–6.
4. + US3 → relatório de descartes confiável → validar com Cenário 7.
5. + US4 → simplificação/suavização ajustáveis → validar com Cenário 8.
6. Cada história agrega valor sem quebrar as histórias anteriores.

---

## Notas

- `[P]` = arquivos diferentes, sem dependência entre si.
- O rótulo de história (`[US1]`...`[US4]`) mapeia a tarefa para rastreabilidade
  com `spec.md`.
- Cada história de usuário deve ser completável e testável de forma
  independente ao final de sua fase, mesmo quando reaproveita arquivos
  estendidos por uma história anterior.
- Faça commit após cada tarefa ou grupo lógico.
- Pare em qualquer checkpoint para validar a história com o `quickstart.md`.
- Evite: tarefas vagas, duas tarefas `[P]` editando o mesmo arquivo,
  dependências que quebrem a independência de teste de uma história.
