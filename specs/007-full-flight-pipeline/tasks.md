---

description: "Task list for feature implementation"
---

# Tarefas: Voo em um Único Comando

**Entrada**: Documentos de design de `/specs/007-full-flight-pipeline/`

**Pré-requisitos**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/cli.md`, `contracts/intermediates-directory.md`, `quickstart.md`

**Testes**: incluídos. Como nas etapas anteriores, a constituição do projeto
(Princípio VI — Testes Automatizados no Núcleo; Princípio X —
Given/When/Then, builders e isolamento por camada) exige testify e uber-go/mock
e proíbe testes tabulares: cada cenário é um `t.Run("should ...")` com
`// given`, `// when`, `// then`. As tarefas de teste abaixo ficam antes da
implementação correspondente em cada fase e devem falhar primeiro.

**Organização**: as tarefas são agrupadas por história de usuário (P1–P4 de
`spec.md`) para permitir implementação e teste independentes de cada
história. A **Phase 2** tem três partes: (A) `VideoService` ganha
`CheckEncoder`/`CheckDestination` (`research.md` item 2), pequena e isolada,
sem mudar nenhum comportamento observável de `video`; (B) o domínio novo da
etapa 7, puro e sem E/S (`Workspace`, `FlightRequest`, `FlightStage`,
`FlightProgress`, `FlightSummary`, `ErrFlightInterrupted`); (C) o adapter
`workingdir` e o código de saída `51`.

**Divisão do trabalho entre histórias** (o mesmo arquivo é estendido por mais
de uma história, nunca em paralelo):

| História | O que entrega |
|---|---|
| US1 | **MVP**: o comando `fly` de ponta a ponta, sem `--keep` e sem progresso por etapa — trata o trajeto, planeja, recorta, desenha e monta, byte a byte igual ao fluxo manual |
| US2 | os parâmetros ajustáveis: as sete flags de sempre (`--duration`, `--fps`, `--distance`, `--tilt`, `--aspect`, `--resolution`, `--quality`), reaproveitando os mesmos analisadores de `plan`/`render all`/`video` |
| US3 | o progresso por etapa: os anúncios "Stage N/5: ..." e o encaminhamento do progresso que `render all`/`video` já relatam |
| US4 | `--keep`: grava plano, recorte e quadros previsíveis; reaproveita o que já é válido (`CameraPlan.ID()`, `GeoSlice.EnsureMatches`, a regra de conjunto dos quadros) em vez de refazer |

Enquanto a US2 não existe, `fly` só aceita os padrões de configuração;
enquanto a US3 não existe, a execução roda sem nenhuma linha de progresso (só
o resumo final); enquanto a US4 não existe, não há flag `--keep` e o plano e o
recorte nunca tocam disco.

## Formato: `[ID] [P?] [Story] Descrição`

- **[P]**: pode ser executado em paralelo (arquivos diferentes, sem
  dependência de tarefa incompleta). Tarefas que editam o mesmo arquivo NUNCA
  são marcadas `[P]` entre si, mesmo quando logicamente independentes.
- **[Story]**: a qual história de usuário esta tarefa pertence (US1 a US4).
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
builders em `internal/domain/builddomain`. Regras de estilo: sem `ports.go`;
porta sem entidade dona em arquivo próprio, no topo, com `//go:generate` logo
após `package` (`Workspace` em `workspace.go`, como `Simplifier`); receivers
curtos e consistentes com o tipo (`s` `flightService`/`FlightSummary`/
`FlightStage`, `r` `FlightRequest`, `e` `OS` do adapter `workingdir`); `new(x)`
do Go 1.26 em vez de um helper `ptr`; nome exportado nunca repete o pacote;
receiver sem uso fica sem nome.

---

## Phase 1: Setup (Shared Infrastructure)

**Propósito**: um ponto de partida próprio (branch) e verde.

- [X] T001 Criar e mudar para o branch `007-full-flight-pipeline` a partir de `main` (`git checkout -b 007-full-flight-pipeline`)
- [X] T002 Confirmar `make build`, `make test`, `make lint` e `make generate` verdes antes de qualquer mudança (linha de base)

**Checkpoint**: repositório pronto para a Phase 2.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Propósito**: infraestrutura que toda história depende. **Nenhuma história começa antes desta fase estar verde.**

### Parte A — `VideoService.CheckEncoder`/`CheckDestination` (research.md item 2)

- [X] T003 [P] Estender `internal/application/video_service_test.go`: `CheckEncoder` chama `encoder.Probe(ctx)` e devolve `EncoderInfo`/erro (inclusive interrupção); `CheckDestination` chama `exporter.Check(output, overwrite)` e devolve o erro tal como veio; `Assemble` continua se comportando exatamente como antes (mesmos testes existentes, sem alteração de expectativa)
- [X] T004 Adicionar `CheckEncoder(ctx) (domain.EncoderInfo, error)` e `CheckDestination(output string, overwrite bool) error` à interface `VideoService` e à struct `videoService` em `internal/application/video_service.go`; refatorar `Assemble` para chamá-los no lugar do corpo que fala direto com `s.encoder`/`s.exporter`; `make test` verde (nenhuma mudança de comportamento em `video`)
- [X] T005 [P] Regenerar o mock de `VideoService` (`make generate`) → `internal/application/mockapplication/video_service.go` com os dois métodos novos

### Parte B — domínio novo da etapa 7

- [X] T006 [P] Adicionar a porta `Workspace` a `internal/domain/workspace.go` (interface no topo, `//go:generate` logo após `package`, conforme `data-model.md`)
- [X] T007 [P] Adicionar `ErrFlightInterrupted` a `internal/domain/errors.go`
- [X] T008 [P] Escrever `internal/domain/flight_test.go`: `FlightStage.String()` para os cinco valores (`TrackProcessing` → "treating the track", ..., `VideoEncoding` → "encoding the video"), um `t.Run` por valor
- [X] T009 Adicionar `FlightRequest`, `FlightStage` (+ `String()`), `FlightProgress`, `FlightSummary` a `internal/domain/flight.go`, exatamente como `data-model.md` descreve; `make test` verde
- [X] T010 [P] Adicionar `builddomain.NewFlightRequestBuilder()` em `internal/domain/builddomain/flight_request_builder.go` (defaults sensatos: `Parameters` de um `PlanParameters` válido, `Resolution` 1080×1920, `Quality` medium, `Output` um caminho `.mp4`, `Keep` vazio, `Overwrite` falso; `WithKeep(dir)`, `WithOverwrite()`, etc.)
- [X] T011 [P] Regenerar o mock de `Workspace` (`make generate`) → `internal/domain/mockdomain/workspace.go`

### Parte C — adapter `workingdir` e código de saída

- [X] T012 [P] Escrever `internal/infra/outbound/workingdir/os_workspace_test.go`: `NewTemporary()` cria um diretório vazio que existe e é gravável; duas chamadas devolvem diretórios diferentes; a função `remove` devolvida apaga o diretório e tudo dentro dele; `remove` chamada duas vezes não é erro (`os.RemoveAll` já é idempotente)
- [X] T013 Implementar `workingdir.NewOS()` (`Workspace`) em `internal/infra/outbound/workingdir/os_workspace.go`, usando `os.MkdirTemp`/`os.RemoveAll`; `make test` verde
- [X] T014 [P] Estender `internal/infra/inbound/cli/exit_code_test.go` e `exit_code.go`: `errors.Is(err, domain.ErrFlightInterrupted)` → código `51`

**Checkpoint**: `make test`, `make lint`, `make generate` verdes; `VideoService`, `Workspace` e os tipos de domínio da etapa 7 prontos para `FlightService`.

---

## Phase 3: User Story 1 — Do trajeto ao vídeo num único comando (Priority: P1) 🎯 MVP

**Objetivo**: `sobrevoo fly <trajeto> --output <vídeo.mp4>` produz, sem
`--keep` e com todos os padrões, um vídeo idêntico byte a byte ao do fluxo
manual de seis comandos.

**Teste Independente**: `quickstart.md` item 1.

- [X] T015 [P] [US1] Escrever `internal/application/flight_service_test.go` (com `CameraPlanService`, `GeoSliceService`, `FrameService`, `VideoService` e `Workspace` mockados): a ordem das chamadas (`CheckEncoder` → `CheckDestination` → `CameraPlanService.Generate` → `GeoSliceService.Generate` → `Workspace.NewTemporary` → `FrameService.DrawFrames` (com o diretório temporário) → `VideoService.Assemble` (mesmo diretório) → `remove`); uma falha em qualquer chamada propaga o erro tal como veio, sem tradução, e ainda assim chama `remove`; uma interrupção (`ErrRenderInterrupted`/`ErrVideoInterrupted`/erro de contexto cancelado durante `CheckEncoder`) devolve `ErrFlightInterrupted` com `summary.Interrupted = true`; o resumo final traz `Completed` com as cinco etapas, `Elapsed > 0`
- [X] T016 [US1] Implementar `FlightService`/`flightService`/`NewFlightService` (`Fly`) em `internal/application/flight_service.go` para o caminho sem `Keep`: as pré-checagens antes de abrir o `reader`; gera o plano; gera o recorte; `workspace.NewTemporary()` com `defer remove()`; desenha os quadros; monta o vídeo; a função `flightInterruption` (como `videoInterruption`/`interruption` já existem) traduzindo a interrupção de qualquer etapa em `ErrFlightInterrupted`; `make test` verde
- [X] T017 [P] [US1] Regenerar o mock de `FlightService` (`make generate`) → `internal/application/mockapplication/flight_service.go`
- [X] T018 [P] [US1] Escrever `internal/infra/inbound/cli/fly_test.go` (com `FlightService` mockado): argumento posicional obrigatório; `--output` obrigatória; `--output` deve terminar em `.mp4` (mesma mensagem de `video`); sucesso imprime o resumo de quadros (`formatFramesSummary`) seguido do de vídeo (`formatVideoSummary`); erro de negócio propagado sem alteração; `ErrFlightInterrupted` imprime a linha de interrompido com o código `51`
- [X] T019 [US1] Implementar `NewFlightCommand` em `internal/infra/inbound/cli/fly.go`: comando `fly <trajeto> --output <vídeo.mp4>`, sem nenhuma outra flag ainda (valores fixos correspondentes aos padrões); abre o trajeto, `interruptContext`, chama `flightService.Fly`, imprime os dois resumos reaproveitando `formatFramesSummary`/`formatVideoSummary`
- [X] T020 [US1] Ligar `workingdir.NewOS()`, `application.NewFlightService(...)` e `cli.NewFlightCommand(...)` em `cmd/sobrevoo/main.go`
- [X] T021 [US1] Validação manual: `quickstart.md` item 1 (comando único vs. fluxo manual, byte a byte), com os dados de amostra; anotar o resultado

**Checkpoint**: `make build`/`make test`/`make lint` verdes; `sobrevoo fly
trajeto.gpx --output voo.mp4` funciona e é idêntico ao fluxo manual — MVP
entregue.

---

## Phase 4: User Story 2 — Ajustar os mesmos parâmetros de sempre (Priority: P2)

**Objetivo**: as sete flags que já existem em `plan`/`render all`/`video`
funcionam em `fly`, com os mesmos nomes, valores aceitos, validação e
padrões.

**Teste Independente**: `quickstart.md` item 2.

- [X] T022 [P] [US2] Estender `internal/infra/inbound/cli/fly_test.go`: cada uma das sete flags (`--duration`, `--fps`, `--distance`, `--tilt`, `--aspect`, `--resolution`, `--quality`) chega em `domain.FlightRequest` com o valor informado; omitida, usa o padrão passado ao comando; um valor inválido em qualquer uma produz o mesmo erro de uso (código `2`) e a mesma mensagem que `plan`/`render all`/`video` já dão para essa flag, sem nenhuma etapa ter rodado
- [X] T023 [US2] Adicionar as sete flags a `NewFlightCommand` em `internal/infra/inbound/cli/fly.go`, reaproveitando `parsePlanParameters` (duração/fps/distância/inclinação/proporção), `domain.ParseResolution` e `domain.ParseVideoQuality` — as mesmas funções que `plan`/`render all`/`video` já usam, nenhuma reimplementada; preenche `domain.FlightRequest.Parameters`/`Resolution`/`Quality`; chama `warnIfFramingCut`
- [X] T024 [US2] Atualizar a ligação em `cmd/sobrevoo/main.go`: `cli.NewFlightCommand` recebe os mesmos `domainPlanParameters(cfg.PlanDefaults)`, `defaultResolution` e `domainVideoQuality(cfg.VideoDefaults.Quality)` que `plan`/`render all`/`video` já recebem
- [X] T025 [US2] Validação manual: `quickstart.md` item 2 (parâmetros ajustáveis e valor inválido)

**Checkpoint**: todas as sete flags funcionam e validam como nos comandos
individuais.

---

## Phase 5: User Story 3 — Acompanhar o progresso por etapa (Priority: P3)

**Objetivo**: a saída identifica a etapa em curso e, nas duas mais
demoradas, mostra o mesmo progresso que `render all`/`video` já mostram.

**Teste Independente**: `quickstart.md` item 3.

- [X] T026 [P] [US3] Estender `internal/application/flight_service_test.go`: `progress` é chamado com `FlightProgress{Stage: ...}` ao entrar em cada uma das cinco etapas, na ordem; durante o desenho dos quadros, `progress` também é chamado com `Render` preenchido (o mesmo `RenderProgress` que `FrameService.DrawFrames` relatou); durante a montagem, com `Video` preenchido; `progress == nil` não quebra nada (mesmo teste de sucesso sem callback)
- [X] T027 [US3] Implementar os anúncios de etapa e o encaminhamento de progresso em `Fly` (`internal/application/flight_service.go`): chama `progress(domain.FlightProgress{Stage: ...})` ao entrar em cada etapa; embrulha os callbacks passados a `DrawFrames`/`Assemble` para encaminhar `Render`/`Video`
- [X] T028 [P] [US3] Escrever `internal/infra/inbound/cli/fly_progress_test.go`: as linhas "Stage N/5: ..." (uma por etapa, na ordem, com o rótulo de `FlightStage.String()`) e que as linhas de progresso de quadro/codificação encaminhadas têm exatamente o formato que `render all`/`video` já escrevem (terminal: linha reescrita; log: a cada N)
- [X] T029 [US3] Implementar o impressor de etapa em `internal/infra/inbound/cli/fly_progress.go`, ligado a `runFly` (`fly.go`)
- [X] T030 [US3] Validação manual: `quickstart.md` item 3 (progresso por etapa)

**Checkpoint**: a saída de `fly` mostra as cinco etapas e o progresso detalhado
das duas mais lentas.

---

## Phase 6: User Story 4 — Guardar e reaproveitar os arquivos intermediários (Priority: P4)

**Objetivo**: `--keep <diretório>` grava `plan.json`/`slice.zip`/`frames/`
previsíveis; uma execução seguinte reaproveita o que ainda é válido para o
mesmo trajeto e os mesmos valores, e recusa por padrão (com `--overwrite`
substituindo) o que não é.

**Teste Independente**: `quickstart.md` itens 4 a 7.

- [X] T031 [P] [US4] Estender `internal/application/flight_service_test.go` com `Keep` preenchido: `plan.json` existente e de `ID()` igual ao plano recém-calculado → `PlanReused = true`, `CameraPlanService.Export` **não** é chamado; ausente ou de outro plano → `Export(fresh, planPath, request.Overwrite)` é chamado; mesmos dois cenários para `slice.zip` com `GeoSlice.EnsureMatches`; o diretório de quadros passado a `DrawFrames`/`Assemble` é `<Keep>/frames`, e `Workspace.NewTemporary` **não** é chamado
- [X] T032 [US4] Implementar o ramo `Keep != ""` em `Fly` (`internal/application/flight_service.go`): `filepath.Join(Keep, "plan.json"/"slice.zip"/"frames")`; a decisão de reaproveitar-ou-exportar para o plano e o recorte (`research.md` item 4); pular `workspace.NewTemporary` quando `Keep != ""`
- [X] T033 [P] [US4] Estender `internal/infra/inbound/cli/fly_test.go` (flag `--keep`) e `fly_progress_test.go` (linha de anúncio "unchanged... reusing ..." quando `PlanReused`/`SliceReused`)
- [X] T034 [US4] Adicionar a flag `--keep` a `NewFlightCommand` (`fly.go`) e as linhas de reaproveitamento a `fly_progress.go`
- [X] T035 [US4] Validação manual: `quickstart.md` itens 4 a 7 (grava os três, reaproveitamento total, mudança parcial de parâmetro, conteúdo de outro conjunto com e sem `--overwrite`)

**Checkpoint**: `--keep` grava, inspeciona e reaproveita como
`contracts/intermediates-directory.md` descreve.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Propósito**: os casos extremos que não pertencem a nenhuma história
específica, a documentação e a verificação final.

- [X] T036 [P] Validação manual: `quickstart.md` itens 8 (recusa cedo: destino já existe, codificador ausente, cobertura ausente) e 9 (interrupção com e sem `--keep`), anotando os valores medidos
- [X] T037 [P] Atualizar `CLAUDE.md`: a sétima etapa, o comando `fly`, `FlightService`, a porta `Workspace`, a observação de que nenhuma regra de negócio nasce aqui
- [X] T038 [P] Atualizar `README.md`: seção do comando `fly`, com exemplos (a mesma estrutura das seções de `plan`/`render`/`video` já existentes)
- [X] T039 Verificação final: `make build`, `make test`, `make lint`, `make generate` verdes; reexecutar `quickstart.md` por inteiro com o `ffmpeg` real

---

## Dependências e Ordem de Execução

### Dependências entre fases

- **Phase 1 (Setup)**: sem dependências.
- **Phase 2 (Foundational)**: depende da Phase 1; **bloqueia todas as histórias**. Partes A, B e C são sequenciais entre si (A e B não compartilham arquivo, mas B usa o `errors.go`/`flight.go` que a Parte C referencia em `exit_code.go`); dentro de cada parte, as tarefas `[P]` são paralelas.
- **Phases 3 a 6 (US1 a US4)**: dependem da Phase 2 concluída; cada uma **estende os arquivos** da anterior (`flight_service.go`, `fly.go`, `fly_progress.go`), então seguem a ordem P1 → P4, e só dentro de uma história as tarefas `[P]` são paralelas. US2 a US4 dependem da US1 (o MVP).
- **Phase 7 (Polish)**: depende das histórias desejadas.

### Dependências entre histórias

- **US1 (P1)**: depende só da Foundational. Ao terminar, existe o comando `fly`.
- **US2 (P2)**: depende da US1 (`fly.go`, `flight_service.go` já existem).
- **US3 (P3)**: depende da US1; independente de US2 (edita `flight_service.go`/`fly.go` em sequência, não em paralelo).
- **US4 (P4)**: depende da US1; independente de US2 e US3 (mesma observação).

### Dentro de cada história

- Testes antes da implementação (devem falhar primeiro).
- Domínio/portas antes de serviço; serviço antes da CLI; CLI antes de `main.go`.
- A história é completa antes da próxima prioridade.

### Oportunidades de paralelismo

```text
# Foundational, Parte A:
T003 video_service_test.go → T004 video_service.go → T005 mock

# Foundational, Parte B, tudo em paralelo:
T006 workspace.go   T007 errors.go   T008 flight_test.go
→ T009 flight.go (depende de T008) → T010 builder   T011 mock

# Foundational, Parte C:
T012 os_workspace_test.go → T013 os_workspace.go
T014 exit_code (independente, [P])

# US1:
T015 flight_service_test.go → T016 flight_service.go → T017 mock
T018 fly_test.go (pode começar em paralelo com T015/T016) → T019 fly.go → T020 main.go

# US2 a US4: os testes de cada história são [P] entre si (arquivos de teste
# diferentes), mas a implementação de cada história edita flight_service.go,
# fly.go e fly_progress.go, um de cada vez — nunca [P] entre histórias.
```

---

## Estratégia de Implementação

### MVP Primeiro (Phase 1 + 2 + User Story 1)

1. Phase 1 (branch e linha de base) e Phase 2 (`CheckEncoder`/`CheckDestination`; `Workspace`; `FlightRequest`/`FlightStage`/`FlightProgress`/`FlightSummary`; `ErrFlightInterrupted`; adapter `workingdir`; código `51`).
2. Phase 3 (US1): `FlightService.Fly` sem `Keep`, comando `fly` com só `--output`.
3. **PARE e valide**: `quickstart.md` item 1 — o vídeo de `fly` é idêntico byte a byte ao do fluxo manual, antes de acrescentar histórias.

### Entrega Incremental

1. Setup + Foundational → `VideoService` estendido; domínio e adapter da etapa 7 prontos.
2. + US1 → o comando `fly` de ponta a ponta (MVP) → item 1.
3. + US2 → as sete flags ajustáveis → item 2.
4. + US3 → progresso por etapa → item 3.
5. + US4 → `--keep` e reaproveitamento → itens 4 a 7.
6. Polish → recusa cedo, interrupção, documentação, verificação final → itens 8 e 9.

Cada história agrega valor sem quebrar as anteriores.

---

## Notas

- `[P]` = arquivos diferentes, sem dependência de tarefa incompleta.
- Nenhuma regra de negócio nova nasce nesta etapa (FR-002): toda tarefa de
  implementação **chama** um serviço/porta já existente das etapas 1 a 6, nunca
  reimplementa a lógica dele — se uma tarefa parecer exigir isso, o desenho
  está errado, não a tarefa (revisar `research.md` antes de prosseguir).
- Faça commit após cada tarefa ou grupo lógico (só quando o usuário pedir).
- Pare em qualquer checkpoint para validar a história com o `quickstart.md`.
- Evite: tarefas vagas, duas tarefas `[P]` editando o mesmo arquivo,
  dependências que quebrem a independência de teste de uma história.

---

## Notas de implementação (desvios do plano original, registrados ao executar)

`quickstart.md` foi validado **por inteiro** (itens 1 a 9) com o `ffmpeg` real
e dados sintéticos, e achou dois problemas reais que nenhum teste automatizado
pega sozinho (os mocks não reproduzem o comportamento real de `Load`/`Export`
nem do sistema de arquivos):

1. **`--keep` para um diretório que ainda não existe falhava**: o plano
   supunha que `CameraPlanService.Export` bastava para o primeiro `--keep`,
   mas ele só cria o arquivo — a pasta-mãe precisa existir (mesma regra de
   `plan --export`). Corrigido com um método novo na porta `Workspace`
   (`EnsureDirectory`, `os.MkdirAll`), chamado antes de qualquer tentativa de
   reaproveitar ou gravar o plano.
2. **Quadros da primeira execução com `--keep` nunca batiam com uma segunda
   execução que reaproveitava o recorte**: `FrameMark.SetID` inclui
   `GeoSlice.ContentID`, que só a leitura de um recorte já gravado preenche
   (um recorte recém-gerado não tem arquivo, logo não tem `ContentID` — contrato
   do próprio tipo). A primeira execução desenhava com o recorte recém-gerado
   (`ContentID` vazio); a segunda, ao reaproveitar o recorte pelo arquivo
   (`ContentID` preenchido), calculava uma identidade de quadro diferente para
   o **mesmo** recorte — `ErrFrameSetConflict` bem no caso mais comum (gerar e
   guardar, depois reaproveitar), inclusive depois de uma interrupção seguida
   de retomada. Corrigido: depois de gerar e exportar um recorte novo,
   `FlightService.reuseSlice` o **relê** do arquivo antes de desenhar, para os
   quadros carregarem a identidade que uma execução futura, reaproveitando o
   recorte, vai calcular. Confirmado com uma interrupção real no meio do
   desenho (17 de 380 quadros) e uma retomada: antes da correção, "un frames
   são de outro conjunto"; depois, `17 kept`, só os 363 restantes desenhados.

Os dois viraram tarefas de teste antes da correção (TDD sobre o achado), e
ficam documentados em `plan.md` (R4 e R5) e em `contracts/`/`CLAUDE.md`. Um
terceiro ponto, **não** corrigido por ser o comportamento já existente de
`render all --overwrite` (redesenha tudo, mesmo um conjunto que já bate):
pedir `--keep` com o **mesmo** destino de vídeo de uma execução anterior
precisa de `--overwrite` só por causa do vídeo, e essa mesma flag também
força o redesenho dos quadros à toa — documentado em `plan.md` (R6) e em
`quickstart.md` item 5 (que usa um destino diferente para medir o
reaproveitamento de verdade: 31 s → 1 s).

Medidas reais (Apple M1, dados sintéticos, plano de 380 quadros a 360×640):
execução do zero, 31 s; repetição com tudo reaproveitado (destino novo, sem
`--overwrite`), 1 s — a montagem do vídeo é o único trabalho de verdade
(SC-004). `sobrevoo fly` produziu um vídeo **idêntico, byte a byte**
(`cmp`), ao de rodar os seis comandos na mão com os mesmos valores (SC-002).
