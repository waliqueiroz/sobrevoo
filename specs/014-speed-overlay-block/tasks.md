---

description: "Task list for feature implementation"
---

# Tarefas: Bloco de Velocidade na Sobreposição

**Entrada**: Documentos de design de `/specs/014-speed-overlay-block/`

**Pré-requisitos**: `plan.md`, `spec.md`, `research.md`, `data-model.md`,
`contracts/plan-file-v3.md`, `contracts/speed-overlay-block.md`,
`quickstart.md`

**Testes**: incluídos. A constituição do projeto (Princípio VI — Testes
Automatizados no Núcleo; Princípio X — Given/When/Then, builders e
isolamento por camada) exige testify e uber-go/mock e proíbe testes
tabulares: cada cenário é um `t.Run("should ...")` com `// given`, `//
when`, `// then`. As tarefas de teste ficam antes da implementação
correspondente e devem falhar primeiro. Uma ressalva, como nas etapas 9/13:
as tarefas da Phase 2 mudam **assinaturas e constantes** já existentes
(`planFormatVersion`, `CameraTuning`, `CameraPlan.ID()`), não só acrescentam
comportamento novo; nesses casos, "estender o teste" inclui atualizar as
chamadas já existentes no arquivo (sem isso o pacote nem compila).

**Organização**: as tarefas são agrupadas por história de usuário (P1–P2 de
`spec.md`). A **Phase 2 (Foundational)** carrega todo o mecanismo que torna
a velocidade um dado do plano — a janela de tempo (`CameraTuning.
SpeedWindow`), o cálculo por quadro (`Route.DistanceAt` + `cameraPlanner.
frame`), e a persistência no arquivo exportado (`format_version` 2 → 3,
`frames[].marker.speed_mps`) — porque sem isso nenhuma das duas histórias
tem o que mostrar ou verificar. US1 (P1) acrescenta o bloco visível
(`OverlayBlockSpeed`, o desenho, a flag de CLI). US2 (P2) acrescenta a
única regra de negócio que a Phase 2 ainda não cobre — a velocidade
participar da identidade do plano (`CameraPlan.ID()`/`.Validate()`) — e
prova, sem nenhum código de produção adicional, que a subida de versão já
recusa um plano antigo (a mesma checagem de `format_version` que a Phase 2
já exercita).

**Nenhum mock precisa ser regenerado nesta etapa**: nenhuma interface de
porta ou de serviço de aplicação ganha método novo nem muda de assinatura —
`CameraPlanExporter.Export`/`CameraPlanReader.Read`,
`application.CameraPlanService`/`FrameService`/`FlightService` continuam
exatamente como são; só o conteúdo que cada implementação concreta
escreve/lê muda. A verificação final (T033) confirma que `make generate`
não produz diferença.

**Divisão do trabalho entre histórias**:

| História | O que entrega | Veículo |
|---|---|---|
| US1 (P1) | ver a velocidade estável da atividade no vídeo — a capacidade central | `OverlayBlockSpeed` ("speed") em `--overlay-blocks`, em `render frame`, `render all` e `fly` |
| US2 (P2) | nunca confundir, nem reaproveitar por engano, um plano cuja velocidade calculada é diferente | `CameraPlan.ID()`/`.Validate()` + a recusa (já automática) de um plano `format_version` anterior |

Enquanto a Phase 2 não existe, nada deste arquivo compila. Enquanto a US1
não existe, a velocidade existe no plano e no arquivo, mas nenhum comando
a desenha. Enquanto a US2 não é verificada (mas a Phase 2 já existe), a
identidade do plano ainda não distingue velocidades diferentes — só falta
a prova e as duas linhas que a tornam real.

## Formato: `[ID] [P?] [Story] Descrição`

- **[P]**: pode ser executado em paralelo (arquivos diferentes, sem
  dependência de tarefa incompleta). Tarefas que editam o mesmo arquivo
  NUNCA são marcadas `[P]` entre si, mesmo quando logicamente independentes.
- **[Story]**: a qual história de usuário esta tarefa pertence (US1 ou
  US2). Tarefas de Setup, Foundational e Polish não têm esse rótulo.
- Toda tarefa inclui o caminho de arquivo exato a criar/editar.
- Comentários de código, identificadores, mensagens de commit, flags, saída
  e mensagens de erro em tempo de execução: **inglês**; artefatos do Spec
  Kit: português (constituição, "Idioma dos Artefatos").

## Convenções de Caminho

Mesmo projeto único em Go, mesma estrutura hexagonal de `plan.md`:
`cmd/sobrevoo/`, `internal/domain/`, `internal/infra/outbound/{jsonfile,
config}`, `internal/infra/inbound/cli/`. Builders em
`internal/domain/builddomain`. Receivers curtos e consistentes com o tipo
(`r` `Route`, `c` `CameraPlan`, `s` `screenOverlay`); nome exportado nunca
repete o pacote.

---

## Phase 1: Setup (Shared Infrastructure)

**Propósito**: um ponto de partida próprio (branch) e verde.

- [X] T001 Confirmar que o branch `014-speed-overlay-block` está ativo (já criado por `/speckit-specify`); se não estiver, `git checkout 014-speed-overlay-block`
- [X] T002 Confirmar `make build`, `make test`, `make lint` e `make generate` verdes antes de qualquer mudança (linha de base)

**Checkpoint**: repositório pronto para a Phase 2.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Propósito**: a velocidade passa a existir, ser calculada e sobreviver ao
arquivo de plano exportado. **Nenhuma história começa antes desta fase
estar verde.**

### Parte A — a janela de tempo (`CameraTuning.SpeedWindow`) (research.md item 3; data-model.md)

- [X] T003 [P] Adicionar `SpeedWindow time.Duration` a `domain.CameraTuning`, ao lado de `GaussianSigmaSeconds`, em `internal/domain/camera_plan_parameters.go` — só o campo, sem validação nova (é um limiar interno, não uma entrada do usuário, como os demais campos de `CameraTuning`)
- [X] T004 [P] Em `internal/domain/builddomain/camera_tuning_builder.go`: `NewCameraTuningBuilder()` passa a preencher `SpeedWindow: 30 * time.Second` no literal padrão; adicionar `WithSpeedWindow(window time.Duration) *CameraTuningBuilder` (depende de T003)
- [X] T005 [P] Estender `cmd/sobrevoo/config_mapping_test.go` (`Test_domainCameraTuning`): esperar `tuning.SpeedWindow == 30*time.Second` quando `config.CameraTuning.SpeedWindowSeconds == 30` — deve falhar até T006/T007 (coberto pelo cenário de igualdade completa já existente, "should map the configuration's camera tuning...", que compara a struct inteira contra `builddomain.NewCameraTuningBuilder().Build()`; confirmado vermelho antes de T006/T007)
- [X] T006 [P] Em `internal/infra/outbound/config/config.go`: adicionar `SpeedWindowSeconds float64` a `CameraTuning` (ao lado de `GaussianSigmaSeconds`) e `SpeedWindowSeconds: 30` à função `cameraTuning()`; `make test` verde (depende de T005)
- [X] T007 Em `cmd/sobrevoo/config_mapping.go` (`domainCameraTuning`): acrescentar `SpeedWindow: time.Duration(t.SpeedWindowSeconds * float64(time.Second))`; `make test` verde (depende de T003, T006)

### Parte B — o espelho de `Route.TimeAt` (research.md item 1; data-model.md)

- [X] T008 [P] Estender `internal/domain/duration_test.go`: `Test_Route_DistanceAt` (nova função, ao lado de `Test_Route_TimeAt`), espelhando exatamente os cenários de `Test_Route_TimeAt` com os papéis de tempo e distância trocados — "should return the distance travelled at the given elapsed time", "should clamp to the first point's distance for an elapsed time before it", "should clamp to the last point's distance for an elapsed time after it", "should report ok=false when the route has no time data" — deve falhar até T009
- [X] T009 Em `internal/domain/duration.go`: adicionar `func (r Route) DistanceAt(distances []float64, at time.Duration) (float64, bool)`, buscando por `at` sobre `r.Points[i].Time.Sub(*r.Points[0].Time)` (ao contrário de `TimeAt`, que busca por distância) e interpolando a distância com a mesma forma seguro `(1-t)*a + t*b`; `make test` verde (depende de T008)

### Parte C — o cálculo por quadro (research.md itens 1, 2; data-model.md)

- [X] T010 [P] Adicionar `MarkerSpeed float64` a `domain.CameraFrame`, ao lado de `CameraToMarkerDistance`, em `internal/domain/camera_plan.go` (metros por segundo; comentário citando a janela de tempo e o critério de disponibilidade — sem checagem em `Validate()`/`ID()` ainda, isso é US2)
- [X] T011 [P] Em `internal/domain/builddomain/camera_frame_builder.go`: adicionar `WithMarkerSpeed(speed float64) *CameraFrameBuilder` (mesmo padrão de `WithCameraToMarkerDistance`) (depende de T010)
- [X] T012 [P] Estender `internal/domain/camera_planning_test.go` (`Test_PlanCamera`), usando `builddomain.NewSyntheticRouteBuilder().WithLine(...).WithConstantSpeed(n)`:
  - "should compute marker speed as the average over the full window, away from both ends of a constant-speed route" — rota longa o bastante para a janela completa (`defaultTuning().SpeedWindow`) caber nos dois lados de um quadro do meio; `assert.InDelta(t, n, frame.MarkerSpeed, 0.01)`
  - "should shorten the window at the first frame instead of leaving the speed at zero or jumping" — usando `builddomain.NewCameraTuningBuilder().WithSpeedWindow(...).Build()` com uma janela pequena sobre uma rota curta de velocidade constante; o primeiro quadro (`ActivityElapsed == 0`) já tem `MarkerSpeed` igual à velocidade constante da rota (a janela encurtada, só para a frente, ainda mede exatamente essa velocidade)
  - "should shorten the window at the last frame the same way" — o espelho do cenário anterior no último quadro
  - "should be zero when the track has no time data" — rota construída com `.WithoutTime()`; todo quadro tem `MarkerSpeed == 0`
  deve falhar até T013
- [X] T013 Em `internal/domain/camera_planning.go` (`cameraPlanner.frame`): calcular `markerSpeed` usando o mesmo `ok` de `ActivityElapsed` (`c.trackRoute.TimeAt`), `c.trackRoute.Duration()` para os limites, `DistanceAt` (T009) para as duas pontas da janela (`c.tuning.SpeedWindow`, via o novo método `cameraPlanner.markerSpeedAt`), dividindo a distância pelo tempo realmente coberto (zero quando esse tempo é zero); gravar `MarkerSpeed: quantize(markerSpeed, speedStep)` no `CameraFrame` retornado; acrescentar a constante `speedStep = 1e-3` ao lado de `coordinateStep`/`lengthStep`/`angleStep`; `make test` verde (depende de T009, T010, T012)

### Parte D — persistência no arquivo de plano exportado (research.md itens 4, 5; contracts/plan-file-v3.md)

- [X] T014 [P] Estender `internal/infra/outbound/jsonfile/camera_plan_exporter_test.go`: na fixture de quadro, acrescentar `MarkerSpeed: n`; esperar `format_version` igual a `3` no JSON decodificado; na struct de decodificação bruta do quadro, acrescentar `Marker.SpeedMPS float64` (tag `"speed_mps"`) e `assert.Equal(t, n, decoded.Frames[i].Marker.SpeedMPS)` — deve falhar até T015
- [X] T015 Em `internal/infra/outbound/jsonfile/camera_plan_file.go`: `planFormatVersion`: `2` → `3` (atualizar o comentário citando por que — um campo obrigatório novo, como a transição `1` → `2` já documentou); `markerFile`/`frameFile`: acrescentar `Speed number `json:"speed_mps"``; `encodePlan`: preencher com `measure(frame.MarkerSpeed)`; `make test` verde (depende de T010, T014)
- [X] T016 [P] Estender `internal/infra/outbound/jsonfile/camera_plan_reader_test.go` e `test/helper/camera_plan_fixture.go` (fixture compartilhada, `format_version` 2 → 3, `MarkerSpeed`/`speed_mps` acrescentados):
  - um cenário de ida e volta lendo `marker.speed_mps` de volta exatamente (como os já existentes para `distance_m`/`elevation_m`/`gain_m`)
  - "should name a marker missing speed_mps" — `ErrPlanFileInvalid`, citando o campo, mesmo padrão de `camera_to_marker_m`/`activity_time_s` ausentes
  - os dois cenários já existentes de versão desconhecida/anterior atualizados para os números pós-subida (`4`/`3` e `2`/`3`), mais um cenário novo "should refuse a plan from two format versions back the same way" (`1`/`3`)
  deve falhar até T017
- [X] T017 Em `internal/infra/outbound/jsonfile/camera_plan_reader.go`: `readMarker`: acrescentar `Speed *float64 `json:"speed_mps"``; `Read()`: checar `f.Marker.Speed == nil` → `invalidPlanFile("frames[%d].marker.speed_mps is missing", i)`, na mesma lista de `switch` dos demais campos obrigatórios por quadro; `CameraFrame{...}`: acrescentar `MarkerSpeed: *f.Marker.Speed`; `acceptedPlanFormatVersions` continua `strconv.Itoa(planFormatVersion)` (nenhuma mudança de código, só o valor que já segue a constante); `make test` verde (depende de T015, T016)

**Checkpoint**: `make build`, `make test`, `make lint` verdes; a velocidade é calculada, sobrevive ao arquivo de plano exportado e volta intacta na leitura — mas nenhum comando a desenha, e a identidade do plano ainda não a distingue.

---

## Phase 3: User Story 1 — Ver a velocidade estável da atividade sobre o voo (Priority: P1) 🎯 MVP

**Objetivo**: pedir o bloco `speed` em `--overlay-blocks` (`render frame`,
`render all`, `fly`) desenha, em posição fixa, a velocidade média da
atividade — estável, nunca a velocidade instantânea entre dois pontos do
GPS; sem pedir, nada muda.

**Teste Independente**: `quickstart.md` itens 1 a 4.

### Parte A — `OverlayBlockSpeed` e `OverlayConfig.Speed` (data-model.md)

- [X] T018 [P] [US1] Estender `internal/domain/frame_overlay_config_test.go`:
  - `Test_NewOverlayConfig`: "should turn on the speed block when named, even though it is not among the four default names" — `NewOverlayConfig(true, []OverlayBlock{OverlayBlockSpeed})` → `config.Speed == true`; estender a mensagem de erro esperada em "should refuse an unknown block name" não muda (continua `"altitude"`, que continua inválido)
  - `Test_OverlayConfig_Fingerprint`: "should change when Speed changes" (mesmo padrão dos outros quatro cenários)
  deve falhar até T019
- [X] T019 [US1] Em `internal/domain/frame_overlay_config.go`: adicionar `OverlayBlockSpeed OverlayBlock = "speed"`; `OverlayConfig`: campo `Speed bool`; `NewOverlayConfig`: `case OverlayBlockSpeed: config.Speed = true`, e a mensagem de erro do `default` passa a citar `"...profile, speed"`; `Fingerprint()`: acrescentar `"|" + flag(o.Speed)` ao final; `make test` verde (depende de T018)

### Parte B — o desenho do bloco (contracts/speed-overlay-block.md)

- [X] T020 [P] [US1] Estender `internal/domain/frame_screen_overlay_test.go`:
  - `Test_FormatOverlaySpeed` (nova função, ao lado de `Test_FormatOverlayElapsed`): "should show km/h to one decimal, converted from meters per second" — `formatOverlaySpeed(10.0/3.6)` ≈ `"10.0 km/h"`
  - `Test_BlockText`: "should write the speed label in Portuguese" — `speedBlockText(frame)` == `"VEL " + formatOverlaySpeed(frame.MarkerSpeed)`
  - `Test_StablePanelWidth`: um novo cenário "should include the speed block among the widest candidates when requested" cobrindo a branch de `speed` em `stablePanelWidth` (mesmo padrão dos três blocos já cobertos)
  - `Test_ScreenOverlay_Draw`: "should draw the speed block when requested and the plan has a clock reference" e "should draw nothing for the speed block when the plan has no clock reference" (espelhando os dois cenários equivalentes já existentes para o bloco `time`)
  deve falhar até T021
- [X] T021 [US1] Em `internal/domain/frame_screen_overlay.go`:
  - `speedBlockText(frame CameraFrame) string { return "VEL " + formatOverlaySpeed(frame.MarkerSpeed) }` e `formatOverlaySpeed(mps float64) string { return fmt.Sprintf("%.1f km/h", mps*3.6) }`, ao lado das demais funções de formatação
  - `draw()`: `showSpeed := s.config.Speed && plan.TimeReference == TimeReferenceClock`; depois do bloco `showTime` (que hoje é o último e não incrementa `y`), acrescentar `y += lineHeight + pad` ali e então, se `showSpeed`, `s.drawLine(marginSide, y, ppem, pad, s.panelWidth, speedBlockText(frame))` — `speed` passa a ser o último painel empilhado
  - `stablePanelWidth()`: acrescentar `showSpeed := config.Speed && plan.TimeReference == TimeReferenceClock` e a branch correspondente no laço que mede `speedBlockText`
  - `make test` verde (depende de T019, T020)

### Parte C — a flag de CLI (contracts/speed-overlay-block.md)

- [X] T022 [P] [US1] Estender `internal/infra/inbound/cli/render_frame_test.go` (`Test_RenderFrameCommand_Overlay`): "should turn on the speed block when named in --overlay-blocks, even though it is not in the default list" — `--overlay-blocks=distance,speed` produz `domain.OverlayConfig{Enabled: true, Distance: true, Speed: true}` passado a `frameService.DrawFrame` (passou de imediato — `parseOverlay`/`NewOverlayConfig` já aceitavam `speed` desde T019 — confirmando que a CLI não precisava de nenhuma lógica própria de validação)
- [X] T023 [US1] Em `internal/infra/inbound/cli/overlay.go`: `overlaysUsage`/`overlayBlocksUsage` passam a mencionar `speed` e deixar explícito que ele não é um dos padrão (ex.: `"...distance, elevation, time, profile, speed (speed is not on by default)"`); `overlayBlocksOf`: acrescentar `if config.Speed { blocks = append(blocks, domain.OverlayBlockSpeed) }`, ao final da lista; `make test` verde (depende de T019, T022)
- [X] T024 [US1] Comprovar, sem nenhum teste redundante, que `render all` e `fly` já aceitam `speed` do mesmo jeito: seus testes existentes (`Test_RenderAllCommand_Overlay`/`Test_FlightCommand_Overlay`, "should turn on only the blocks named in --overlay-blocks") já provam que `parseOverlay` repassa qualquer nome válido por construção — a validade de `speed` como nome já está provada em T018/T022; nenhuma tarefa de código ou de teste nova é necessária aqui (mesmo padrão da comprovação T024 da etapa 13)
- [X] T025 [US1] Validação manual: `quickstart.md` itens 1 a 4 (sem pedir, nada muda; pedindo, o bloco aparece e é estável entre quadros vizinhos; trajeto sem horário não desenha o bloco, sem erro; nome de bloco inválido continua recusando)

**Checkpoint**: pedir `--overlay-blocks ...,speed` em qualquer um dos três comandos desenha um bloco de velocidade estável — MVP entregue.

---

## Phase 4: User Story 2 — Nunca confundir planos com velocidades diferentes (Priority: P2)

**Objetivo**: dois planos iguais em tudo menos na velocidade calculada por
quadro são sempre planos diferentes (nunca reaproveitados um pelo outro por
`fly --keep`); um plano com uma velocidade inválida é recusado; um plano de
antes desta etapa (`format_version` 2) já é recusado pela Phase 2, sem
nenhum código novo — só falta provar isso explicitamente.

**Teste Independente**: `quickstart.md` itens 5 e 6.

- [X] T026 [P] [US2] Estender `internal/domain/camera_plan_test.go` (`Test_CameraPlan_Validate`): "should refuse a frame with a negative marker speed" e "should refuse a frame with a non-finite marker speed" (mesmo padrão dos cenários já existentes para `CameraToMarkerDistance`) — deve falhar até T027
- [X] T027 Em `internal/domain/camera_plan.go` (`CameraPlan.Validate()`): acrescentar, na mesma lista de `switch` por quadro, `case math.IsNaN(f.MarkerSpeed) || math.IsInf(f.MarkerSpeed, 0) || f.MarkerSpeed < 0: return invalid("frames[%d].marker_speed_mps is %g, must be a finite number of at least 0", i, f.MarkerSpeed)`; `make test` verde (depende de T026)
- [X] T028 [P] [US2] Estender `internal/domain/camera_plan_test.go` (`Test_CameraPlan_ID`): acrescentar `MarkerSpeed` aos cenários "should be the same for values that differ by less than the step" (ruído de `1e-5`) e "should change when one value of one frame changes by one step" (`1e-3`), espelhando exatamente como `CameraToMarkerDistance` já é tratado nos dois — deve falhar até T029
- [X] T029 Em `internal/domain/camera_plan.go` (`CameraPlan.ID()`): acrescentar `write(quantized(f.MarkerSpeed, speedStep))` imediatamente depois da escrita já existente de `CameraToMarkerDistance`, dentro do laço por quadro; `make test` verde (depende de T028)
- [X] T030 [US2] Comprovar, sem nenhum teste redundante, que um plano `format_version: 2` já é recusado: o cenário "should refuse a plan of the previous format version" (T016, Phase 2 Parte D) já exercita exatamente essa recusa — a mesma checagem genérica de versão que a etapa 9 já estabeleceu, agora comparando contra `3`; nenhum teste novo aqui, só o registro de que FR-012/FR-013 do `spec.md` já estão cobertos desde a Phase 2
- [X] T031 [US2] Validação manual: `quickstart.md` itens 5 e 6 — confirmado com o binário real: um plano `format_version: 2` é recusado com código de saída `18` e a mensagem "found 2, accepted: 3; generate the plan again..."; dois `plan` independentes sobre o mesmo trajeto produzem exatamente a mesma sequência de `speed_mps` por quadro (determinismo); a diferença de identidade entre planos de velocidades diferentes é coberta por T028, não pela CLI

**Checkpoint**: a velocidade participa da identidade do plano; um valor inválido é recusado; um plano antigo já era recusado desde a Phase 2.

---

## Phase 5: Polish & Cross-Cutting Concerns

**Propósito**: documentação e verificação final.

- [X] T032 [P] Atualizar `CLAUDE.md`: a décima quarta etapa (resumo do projeto, no mesmo estilo das demais), citando o bloco `speed` em `--overlay-blocks`, o cálculo por janela de tempo fixa, a subida de `format_version` para `3` e a participação na identidade do plano — mais uma seção dedicada `### A velocidade da atividade na sobreposição (etapa 14)`, no mesmo padrão das etapas 9/10/11/13
- [X] T033 [P] Atualizar `README.md` (seção "Sobreposições de tela", linhas ~273-293): mencionar o quinto bloco opcional (`speed`, `VEL <n> km/h`), deixando explícito que ele não vem ligado por padrão como os outros quatro, e acrescentar `speed` à lista de valores aceitos na linha da tabela de `--overlay-blocks` (linha ~287)
- [X] T034 Verificação final: `make build`, `make test`, `make lint`, `make generate` verdes (confirmar que `make generate` não produz nenhuma diferença — nenhuma interface mudou de método); `gofmt -l .` limpo; reexecutar `quickstart.md` por inteiro

---

## Dependências e Ordem de Execução

### Dependências entre Fases

- **Phase 1 (Setup)**: sem dependências.
- **Phase 2 (Foundational)**: depende da Phase 1; **bloqueia US1 e US2**. Partes A, B, C e D têm uma ordem real entre si (A alimenta C; B alimenta C; C alimenta D), mas as tarefas `[P]` já indicadas dentro de cada parte podem ser trabalhadas em paralelo.
- **Phase 3 (US1)**: depende da Phase 2 completa.
- **Phase 4 (US2)**: depende da Phase 2 completa; não depende de US1 em código (edita `camera_plan.go`/`camera_plan_test.go`, nunca `frame_screen_overlay.go`/CLI), mas faz mais sentido depois dela por ordem de prioridade.
- **Phase 5 (Polish)**: depende das histórias desejadas.

### Dentro de Cada História de Usuário

- Testes são escritos (ou estendidos) e devem falhar antes da tarefa de implementação correspondente.
- Domínio antes de infraestrutura de saída (`jsonfile`/`config`); infraestrutura antes de CLI; CLI antes de composition root.

### Oportunidades de Paralelização

- Todas as tarefas de Setup marcadas com `[P]` podem ser executadas em paralelo.
- Dentro da Phase 2: Parte A (T003-T007) e Parte B (T008-T009) não dependem uma da outra e podem ser trabalhadas em paralelo; Parte C (T010-T013) depende de ambas; Parte D (T014-T017) depende de Parte C.
- Dentro da Phase 3: Parte A (T018-T019) e Parte C (T022-T024) não dependem uma da outra; Parte B (T020-T021) depende de Parte A.
- US2 (Phase 4) pode ser trabalhada em paralelo com US1 (Phase 3) por outra pessoa, já que edita arquivos diferentes — ambas só dependem da Phase 2.

---

## Estratégia de Implementação

### MVP Primeiro (Somente História de Usuário 1)

1. Completar Phase 1: Setup.
2. Completar Phase 2: Foundational (CRÍTICO — bloqueia US1 e US2).
3. Completar Phase 3: História de Usuário 1 (o bloco `speed` desenhado e estável).
4. **PARAR E VALIDAR**: `quickstart.md` itens 1–4.
5. Implantar/demonstrar se estiver pronta.

### Entrega Incremental

1. Setup + Foundational → velocidade calculada e persistida, nada visível ainda.
2. US1 (o bloco, MVP) → validar → demonstrar.
3. US2 (identidade do plano nunca confunde velocidades diferentes) → validar.
4. Polish.

Cada história agrega valor sem quebrar as anteriores; US2 é a mais barata
das duas (duas linhas de produção — `Validate()`, `ID()` — e uma
comprovação sem código novo para a recusa de versão), exatamente como a
US3 da etapa 13 e a US3/US4 da etapa 10 já foram as mais baratas das suas
respectivas features.
