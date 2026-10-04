---

description: "Task list for feature implementation"
---

# Tarefas: Níveis de tratamento em `plan` e `fly`

**Entrada**: Documentos de design de `/specs/013-treatment-level-flags/`

**Pré-requisitos**: `plan.md`, `spec.md`, `research.md`, `data-model.md`,
`contracts/treatment-level-flags.md`, `contracts/plan-file-addendum.md`,
`quickstart.md`

**Testes**: incluídos. A constituição do projeto (Princípio VI — Testes
Automatizados no Núcleo; Princípio X — Given/When/Then, builders e
isolamento por camada) exige testify e uber-go/mock e proíbe testes
tabulares: cada cenário é um `t.Run("should ...")` com `// given`, `//
when`, `// then`. As tarefas de teste ficam antes da implementação
correspondente e devem falhar primeiro — com uma ressalva, como na etapa 8:
as tarefas da Phase 2 mudam a **assinatura** de `NewCameraPlanService` e o
conteúdo de `parametersFile`/`readParameters`, não só acrescentam
comportamento novo; nesses casos, "estender o teste" inclui atualizar as
chamadas já existentes no arquivo (sem isso o pacote nem compila), e o
cenário novo é o que fica vermelho até a tarefa de implementação seguinte.

**Organização**: as tarefas são agrupadas por história de usuário (P1–P3 de
`spec.md`). A **Phase 2 (Foundational)** carrega todo o mecanismo que US1 e
US2 compartilham — os dois campos novos em `PlanParameters`, a identidade do
plano (`CameraPlan.ID()`), `CameraPlanService` sem `defaultLevel` próprio, o
arquivo de plano exportado e a configuração — porque `plan` (US1) já
precisa de tudo isso para funcionar, e `fly` (US2) só acrescenta duas flags
de CLI sobre o mesmo mecanismo. US3 (`fly --keep` respeitar os níveis, e
tratar um `--keep` sem nível registrado como padrão) não acrescenta nenhuma
lógica de produção nova — a Phase 2 já garante os dois efeitos de graça
(`research.md`, itens 3 e 6); US3 só prova isso em teste e manualmente,
como a US3/US4 da etapa 10 e a US4 da etapa 8 já fizeram.

**Nenhum mock precisa ser regenerado nesta etapa**: a interface
`application.CameraPlanService` não ganha nenhum método nem muda a
assinatura de `Generate`/`Export`/`Load` — só o **construtor concreto**
`NewCameraPlanService` perde um parâmetro (`defaultLevel`), e construtores
não são mocks. `make generate` não deveria produzir nenhuma diferença; a
verificação final (T027) confirma isso.

**Divisão do trabalho entre histórias**:

| História | O que entrega | Veículo |
|---|---|---|
| US1 (P1) | escolher os dois níveis ao planejar a câmera — a capacidade central | `--simplification`/`--smoothing` em `plan` |
| US2 (P2) | o mesmo, de ponta a ponta, no comando único | `--simplification`/`--smoothing` em `fly` |
| US3 (P3) | `fly --keep` nunca confunde dois níveis, e nunca recalcula à toa por causa de um `--keep` antigo | nenhuma lógica nova (já sai da Phase 2) — só a prova, em teste e manualmente |

Enquanto a Phase 2 não existe, nada deste arquivo compila. Enquanto a US1
não existe, nenhuma flag existe em lugar nenhum. Enquanto a US2 não existe,
`fly` continua sempre no nível padrão. Enquanto a US3 não é verificada
(mas a Phase 2 já existe), o comportamento correto já está presente na
prática — só falta a prova.

## Formato: `[ID] [P?] [Story] Descrição`

- **[P]**: pode ser executado em paralelo (arquivos diferentes, sem
  dependência de tarefa incompleta). Tarefas que editam o mesmo arquivo
  NUNCA são marcadas `[P]` entre si, mesmo quando logicamente independentes.
- **[Story]**: a qual história de usuário esta tarefa pertence (US1 a US3).
  Tarefas de Setup, Foundational e Polish não têm esse rótulo.
- Toda tarefa inclui o caminho de arquivo exato a criar/editar.
- Comentários de código, identificadores, mensagens de commit, flags, saída
  e mensagens de erro em tempo de execução: **inglês**; artefatos do Spec
  Kit: português (constituição, "Idioma dos Artefatos").

## Convenções de Caminho

Mesmo projeto único em Go, mesma estrutura hexagonal de `plan.md`:
`cmd/sobrevoo/`, `internal/domain/`, `internal/application/`,
`internal/infra/outbound/jsonfile`, `internal/infra/inbound/cli/`.
Builders em `internal/domain/builddomain`. Receivers curtos e consistentes
com o tipo (`s` `*cameraPlanService`, `c` `CameraPlan`, `p`
`PlanParameters`); nome exportado nunca repete o pacote; `new(x)` do Go
1.26 em vez de um helper `ptr`.

---

## Phase 1: Setup (Shared Infrastructure)

**Propósito**: um ponto de partida próprio (branch) e verde.

- [X] T001 Criar e mudar para o branch `013-treatment-level-flags` a partir de `main` (`git checkout -b 013-treatment-level-flags`)
- [X] T002 Confirmar `make build`, `make test`, `make lint` e `make generate` verdes antes de qualquer mudança (linha de base)

**Checkpoint**: repositório pronto para a Phase 2.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Propósito**: infraestrutura que US1 e US2 compartilham. **Nenhuma
história começa antes desta fase estar verde.**

### Parte A — `domain.PlanParameters` e a identidade do plano (research.md itens 1, 3; data-model.md)

- [X] T003 [P] Adicionar `Simplification Level` e `Smoothing Level` a `domain.PlanParameters`, ao lado de `Distance`/`Tilt`, em `internal/domain/camera_plan_parameters.go` (sem mudança em `Validate()` — os dois campos são um enum fechado de três valores já resolvido pela CLI, como `Distance`/`Tilt` já são)
- [X] T004 [P] Em `internal/domain/builddomain/plan_parameters_builder.go`: `NewPlanParametersBuilder()` passa a preencher `Simplification: domain.LevelMedium, Smoothing: domain.LevelMedium` (os mesmos valores padrão de `Distance`/`Tilt`); adicionar `WithSimplification(domain.Level) *PlanParametersBuilder` e `WithSmoothing(domain.Level) *PlanParametersBuilder` (depende de T003)
- [X] T005 [P] Estender `internal/domain/camera_plan_test.go` (`Test_CameraPlan_ID`, cenário `"should change when a parameter changes"`): acrescentar `assert.NotEqual(t, base, with(builder().WithSimplification(domain.LevelHigh).Build()), "simplification level")` e o equivalente para `WithSmoothing(domain.LevelLow)` — deve falhar até T006 (depende de T004)
- [X] T006 Em `internal/domain/camera_plan.go` (`CameraPlan.ID()`): acrescentar `write(int64(c.Parameters.Simplification))` e `write(int64(c.Parameters.Smoothing))` imediatamente depois das escritas já existentes de `Distance`/`Tilt`; `make test` verde (depende de T003, T005)

### Parte B — `CameraPlanService` sem nível padrão próprio (research.md item 2; data-model.md)

- [X] T007 [P] Em `internal/application/camera_plan_service_test.go`: substituir o cenário `"should treat the track with the default level for both simplification and smoothing"` por `"should treat the track with the parameters' simplification and smoothing levels"`, construindo `parameters` com `Simplification: domain.LevelHigh, Smoothing: domain.LevelLow` (via `builddomain.NewPlanParametersBuilder()...Build()`) e esperando `trackService.EXPECT().Treat(gomock.Any(), domain.LevelHigh, domain.LevelLow)`; atualizar **todas** as demais chamadas de `application.NewCameraPlanService(...)` neste arquivo para a assinatura sem o parâmetro `defaultLevel` (quarto argumento removido) — deve falhar até T008
- [X] T008 Em `internal/application/camera_plan_service.go`: remover o campo `defaultLevel domain.Level` de `cameraPlanService` e o parâmetro correspondente de `NewCameraPlanService`; `Generate` passa a chamar `s.trackService.Treat(reader, parameters.Simplification, parameters.Smoothing)`; `make test` verde (depende de T003, T007)

### Parte C — o arquivo de plano exportado (research.md item 6; contracts/plan-file-addendum.md)

- [X] T009 [P] Estender `internal/infra/outbound/jsonfile/camera_plan_exporter_test.go`: na fixture de parâmetros, acrescentar `.WithSimplification(domain.LevelHigh).WithSmoothing(domain.LevelLow)`; na struct de decodificação bruta do JSON, acrescentar `Simplification string` / `Smoothing string` (tags `"simplification"`/`"smoothing"`) em `parameters`, e `assert.Equal(t, "high", decoded.Parameters.Simplification)` / `assert.Equal(t, "low", decoded.Parameters.Smoothing)` — deve falhar até T010
- [X] T010 Em `internal/infra/outbound/jsonfile/camera_plan_file.go`: acrescentar `Simplification string `json:"simplification"`` e `Smoothing string `json:"smoothing"`` a `parametersFile`, preenchidos em `encodePlan` com `levelText(plan.Parameters.Simplification)`/`levelText(plan.Parameters.Smoothing)` — a mesma função já usada para `distance`/`tilt`; `make test` verde (depende de T003, T009)
- [X] T011 [P] Estender `internal/infra/outbound/jsonfile/camera_plan_reader_test.go`: um cenário de ida e volta lendo `parameters.simplification`/`.smoothing` de volta exatamente (como o já existente para `distance`/`tilt`); um cenário novo — um arquivo cujo JSON de `parameters` **não tem** as chaves `simplification`/`smoothing` é lido com `plan.Parameters.Simplification == domain.LevelMedium` e `.Smoothing == domain.LevelMedium` (a decisão da sessão de `/speckit-clarify`: ausência de registro = nível padrão) — deve falhar até T012
- [X] T012 Em `internal/infra/outbound/jsonfile/camera_plan_reader.go`: acrescentar `Simplification string `json:"simplification"`` e `Smoothing string `json:"smoothing"`` a `readParameters` (campos `string`, não ponteiro — uma chave ausente desserializa como `""`), preenchendo `parameters.Simplification`/`.Smoothing` em `Read()` com `parseLevel(file.Parameters.Simplification)`/`parseLevel(file.Parameters.Smoothing)` — a mesma função já usada para `distance`/`tilt`, que já lê texto desconhecido ou vazio como `medium`; `make test` verde (depende de T003, T011)

### Parte D — configuração e composition root (research.md item 5)

- [X] T013 [P] Estender `cmd/sobrevoo/config_mapping_test.go` (`Test_domainPlanParameters`): chamar `domainPlanParameters(defaults, config.LevelHigh)` e esperar `parameters.Simplification == domain.LevelHigh` e `parameters.Smoothing == domain.LevelHigh` — deve falhar até T014
- [X] T014 Em `cmd/sobrevoo/config_mapping.go`: `domainPlanParameters` ganha um segundo parâmetro, `defaultLevel config.Level`, e preenche `Simplification`/`Smoothing` com `domainLevel(defaultLevel)` (a mesma função de mapeamento já existente); `make test` verde (depende de T003, T013)
- [X] T015 Em `cmd/sobrevoo/main.go`: os dois call sites de `domainPlanParameters(cfg.PlanDefaults)` (para `plan` e para `fly`) passam a `domainPlanParameters(cfg.PlanDefaults, cfg.DefaultLevel)`; o call site de `application.NewCameraPlanService(...)` perde o argumento `domainLevel(cfg.DefaultLevel)`; `make build` verde (depende de T008, T014)

**Checkpoint**: `make build`, `make test`, `make lint` verdes; `Simplification`/`Smoothing` percorrem domínio, aplicação, o arquivo de plano e a configuração — mas nenhuma flag existe ainda em `plan` nem em `fly`.

---

## Phase 3: User Story 1 — Planejar a câmera com o nível conferido no `inspect` (Priority: P1) 🎯 MVP

**Objetivo**: `sobrevoo plan` aceita `--simplification`/`--smoothing`, com o
mesmo nome, os mesmos valores e o mesmo padrão que `inspect` já aceita; sem
nenhuma das duas, o resultado é idêntico ao de antes desta etapa.

**Teste Independente**: `quickstart.md` itens 1 a 4.

- [X] T016 [P] [US1] Estender `internal/infra/inbound/cli/plan_test.go`: a função `defaultParameters()` (ou equivalente) passa a incluir `Simplification: domain.LevelMedium, Smoothing: domain.LevelMedium`; estender o cenário `"should translate the distance and tilt levels"` (ou um novo ao lado dele) para também passar `--simplification high --smoothing low` e esperar `want.Simplification = domain.LevelHigh, want.Smoothing = domain.LevelLow`; estender o mapa de `Test_PlanCommand_UsageErrors` com `"an unknown simplification": {"--simplification", "extreme"}` e `"an unknown smoothing": {"--smoothing", "extreme"}` — deve falhar até T017
- [X] T017 [US1] Em `internal/infra/inbound/cli/plan.go`: `parsePlanParameters` ganha os parâmetros `simplificationFlag, smoothingFlag string` e preenche `parameters.Simplification`/`.Smoothing` via `parseLevel` (mesmo padrão de `Distance`/`Tilt`, erro envolto em `newUsageError` com o prefixo `--simplification:`/`--smoothing:`); `NewPlanCommand` registra as duas flags novas com `cmd.Flags().StringVar(&simplificationFlag, "simplification", levelName(defaults.Simplification), "Simplification level applied to the treated route: low, medium, or high")` e o equivalente para `smoothing`, repassando os dois novos parâmetros na chamada de `parsePlanParameters`; `make test` verde (depende de T003, T016)
- [X] T018 [US1] Validação manual: `quickstart.md` itens 1 a 4 (o traçado usado por `plan --simplification=X --smoothing=Y` bate com o de `inspect` para os mesmos valores; sem flags, resultado idêntico ao de antes; o arquivo exportado registra os dois níveis; dois planos com níveis diferentes nunca têm o mesmo conteúdo)

**Checkpoint**: `plan --simplification --smoothing` funciona de ponta a ponta — MVP entregue.

---

## Phase 4: User Story 2 — Voar sobre o trajeto com o tratamento escolhido (Priority: P2)

**Objetivo**: `sobrevoo fly` aceita as mesmas duas flags, com o mesmo nome e
o mesmo efeito que `plan`, aplicadas em toda a tubulação.

**Teste Independente**: `quickstart.md` item 5.

- [X] T019 [P] [US2] Estender `internal/infra/inbound/cli/fly_test.go`: a fixture de parâmetros padrão passa a incluir `Simplification: domain.LevelMedium, Smoothing: domain.LevelMedium`; estender `Test_FlightCommand_Parameters` (`"should pass the duration, fps, distance, tilt and aspect flags..."`) para também cobrir `--simplification`/`--smoothing`; acrescentar um cenário ao lado de `"should refuse an invalid distance with the same usage error plan already gives, without flying anything"` para um `--simplification`/`--smoothing` inválido, confirmando que `flightService.Fly` nunca é chamado — deve falhar até T020
- [X] T020 [US2] Em `internal/infra/inbound/cli/fly.go`: `NewFlightCommand` registra `--simplification`/`--smoothing` com o mesmo padrão de `plan.go` (`levelName(defaults.Simplification)`/`levelName(defaults.Smoothing)`), repassando-as à mesma chamada de `parsePlanParameters` que já existe; `make test` verde (depende de T017, T019)
- [X] T021 [US2] Validação manual: `quickstart.md` item 5 e 6 (`fly` aplica o tratamento pedido do início ao fim da tubulação; valor inválido recusa antes de qualquer etapa)

**Checkpoint**: `plan` e `fly` aceitam exatamente as mesmas duas flags, com o mesmo efeito.

---

## Phase 5: User Story 3 — Reaproveitar `--keep` com confiança quando o tratamento muda (Priority: P3)

**Objetivo**: confirmar — sem nenhuma lógica nova (já garantido pela Phase
2) — que `fly --keep` recalcula plano/recorte/quadros quando um dos dois
níveis muda entre execuções, reaproveita quando os níveis efetivos são
iguais, e trata um `--keep` sem nível registrado (de antes desta etapa)
como se tivesse usado o nível padrão.

**Teste Independente**: `quickstart.md` itens 7 e 8.

- [X] T022 [P] [US3] Estender `internal/application/flight_service_test.go` (os cenários de `Fly` com `request.Keep` preenchido): duas chamadas sobre o mesmo trajeto e o mesmo `--keep`, a segunda com `Parameters.Simplification` (ou `.Smoothing`) diferente da primeira, resultam em `summary.PlanReused == false` (o `cameraPlanService.Exporter`/`Export` mockado é chamado de novo, sobrescrevendo o plano guardado); as mesmas duas chamadas, mas com os mesmos níveis efetivos na segunda (explícitos ou não — desde que resolvam para o mesmo `Level`), resultam em `summary.PlanReused == true` e `Export` **não** é chamado
- [X] T023 [P] [US3] Comprovar a decisão da sessão de `/speckit-clarify` ("ausência de nível registrado = padrão") sem nenhum teste redundante: como `FlightService` mocka `CameraPlanService` por inteiro (Princípio X — isolamento por camada), ela nunca vê JSON bruto, então a prova de que um arquivo sem os campos lê como `medium` já é — e só pode ser — a de `camera_plan_reader_test.go` (T011); o comentário do cenário `"should reuse the plan under the kept directory when it already matches..."` em `flight_service_test.go` foi estendido para registrar explicitamente que ele já exercita o caso "ambos os lados em `medium`" que esse arquivo produziria, fechando a cadeia T011 (leitor → `medium`) + este cenário (mesmo nível → reaproveita) + T022 (nível diferente → recalcula) sem duplicar cobertura
- [X] T024 [US3] Validação manual: `quickstart.md` itens 7 e 8 (mudar um nível entre execuções de `fly --keep` sempre recalcula; os mesmos níveis reaproveitam; um `--keep` sem nível registrado é tratado como padrão)

**Checkpoint**: `fly --keep` nunca confunde dois níveis diferentes, e nunca recalcula à toa só por causa de um `--keep` de antes desta etapa.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Propósito**: documentação e verificação final.

- [X] T025 [P] Atualizar `CLAUDE.md`: a décima terceira etapa (resumo do projeto, no mesmo estilo das demais), citando `--simplification`/`--smoothing` em `plan` e `fly`, a identidade do plano estendida e a ausência de mudança em `inspect`
- [X] T026 [P] Atualizar `README.md`: acrescentar `--simplification`/`--smoothing` à linha de uso e à tabela de flags das seções de `plan` (linhas ~53, ~91-92) e de `fly` (linhas ~258, ~289-290), no mesmo formato das flags já documentadas ali
- [X] T027 Verificação final: `make build`, `make test`, `make lint`, `make generate` verdes (confirmar que `make generate` não produz nenhuma diferença — nenhuma interface mudou de método); `gofmt -l .` limpo; reexecutar `quickstart.md` por inteiro

---

## Dependências e Ordem de Execução

### Dependências entre Fases

- **Phase 1 (Setup)**: sem dependências.
- **Phase 2 (Foundational)**: depende da Phase 1; **bloqueia US1 e US2**. Partes A → B → C → D podem ser trabalhadas em paralelo entre si depois que a Parte A (T003) existir — B, C e D só dependem de T003, não umas das outras; dentro de cada parte, as tarefas `[P]` já indicadas são paralelas.
- **Phase 3 (US1)**: depende da Phase 2 completa.
- **Phase 4 (US2)**: depende da Phase 2 completa e de T017 (US1) — reaproveita a mesma `parsePlanParameters` que a US1 estende.
- **Phase 5 (US3)**: depende da Phase 2 completa; não depende de US1/US2 em código (usa `FlightRequest.Parameters` diretamente, não a CLI), mas faz mais sentido depois delas por ordem de prioridade.
- **Phase 6 (Polish)**: depende das histórias desejadas.

### Dentro de Cada História de Usuário

- Testes são escritos (ou estendidos) e devem falhar antes da tarefa de implementação correspondente.
- Domínio antes de aplicação; aplicação antes de infraestrutura de saída (`jsonfile`); infraestrutura antes de CLI; CLI antes de composition root (`main.go`).

### Oportunidades de Paralelização

- Todas as tarefas de Setup marcadas com `[P]` podem ser executadas em paralelo.
- Dentro da Phase 2, as Partes B, C e D podem ser trabalhadas em paralelo por pessoas diferentes depois que a Parte A (T003) estiver pronta — cada uma edita arquivos próprios, sem sobreposição.
- US1 e US2 compartilham `parsePlanParameters` (`plan.go`) — US2 depende da US1 terminar, não podem ser paralelas entre si.
- US3 pode ser trabalhada em paralelo com US1/US2 (arquivos diferentes: `flight_service_test.go` em vez de `plan.go`/`fly.go`), mas só depois da Phase 2.

---

## Estratégia de Implementação

### MVP Primeiro (Somente História de Usuário 1)

1. Completar Phase 1: Setup.
2. Completar Phase 2: Foundational (CRÍTICO — bloqueia US1 e US2).
3. Completar Phase 3: História de Usuário 1 (`plan --simplification --smoothing`).
4. **PARAR E VALIDAR**: `quickstart.md` itens 1–4.
5. Implantar/demonstrar se estiver pronta.

### Entrega Incremental

1. Setup + Foundational → fundação pronta.
2. US1 (`plan`, MVP) → validar → demonstrar.
3. US2 (`fly`) → validar → demonstrar.
4. US3 (prova de que `--keep` nunca confunde níveis, nem recalcula à toa) → validar.
5. Polish.

Cada história agrega valor sem quebrar as anteriores; US3 é a mais barata
de todas (nenhuma produção nova, só a prova em teste e manualmente) —
exatamente como a US4 da etapa 8 e a US3 da etapa 10 já foram.
