---

description: "Task list for feature implementation"
---

# Tarefas: Sobreposições de Tela nos Quadros

**Entrada**: Documentos de design de `/specs/009-frame-overlays/`

**Pré-requisitos**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/overlay-flags.md`, `contracts/plan-file-v2.md`, `quickstart.md`

**Testes**: incluídos. A constituição do projeto (Princípio VI — Testes
Automatizados no Núcleo; Princípio X — Given/When/Then, builders e
isolamento por camada) exige testify e uber-go/mock e proíbe testes
tabulares: cada cenário é um `t.Run("should ...")` com `// given`, `//
when`, `// then`. As tarefas de teste ficam antes da implementação
correspondente e devem falhar primeiro — com a mesma ressalva de
008-frame-appearance: várias tarefas da Phase 2 mudam a **assinatura** de
funções já existentes (`NewCameraPlan`, `NewScene`, `NewFrameMark`,
`NewFrameSetID`), não só acrescentam algo novo; nesses casos, "estender o
teste" inclui atualizar as chamadas já existentes no arquivo para a
assinatura nova (sem isso o pacote nem compila), e o cenário novo é o que
fica vermelho até a tarefa de implementação seguinte.

**Organização**: as tarefas são agrupadas por história de usuário (P1–P4 de
`spec.md`). A **Phase 2** tem oito partes: (A) interpolação de tempo/
elevação em `Route`; (B) os três campos novos de `CameraFrame` e o novo
`ElevationAvailable`, calculados em `camera_planning.go`; (C) o formato do
plano sobe para a versão 2; (D) `domain.OverlayConfig`; (E) as constantes
fixas de desenho; (F) o desenho da sobreposição de tela em si (a maior
parte — fonte embutida, placa, margem, os quatro blocos, o perfil de
elevação) e sua ligação a `Scene`; (G) a sobreposição entrando na
identidade do conjunto de quadros; (H) os padrões em `config.RenderDefaults`.
Nenhuma dessas oito partes é visível para o usuário — nenhuma flag nova
existe até a Phase 5 (US3); as Phases 3 e 4 (US1, US2) já desenham os quatro
blocos, mas sempre com o padrão (ligados), sem flag para desligá-los.

**Nenhum mock precisa ser regenerado nesta etapa**: nenhuma porta nem
interface de serviço ganha método novo ou muda de assinatura — `domain.
SingleFrameRequest`, `domain.FrameSetRequest`, `domain.FlightRequest`,
`domain.CameraPlanReader`, `domain.CameraPlanExporter` continuam os mesmos
tipos, por nome, só com um campo a mais ou (no caso de `CameraPlanReader.
Read`) o mesmo método lendo um formato de arquivo diferente. `make
generate` não deveria produzir nenhuma diferença — a verificação final
(T072) confirma isso.

**Divisão do trabalho entre histórias** (o mesmo arquivo é estendido por
mais de uma história, nunca em paralelo):

| História | O que entrega | Veículo |
|---|---|---|
| US1 | **MVP**: os quatro blocos aparecem, ligados por padrão, com os valores corretos | `render frame` |
| US2 | o quadro isolado é garantidamente igual ao do voo inteiro (byte a byte) | `render all` ganha o mesmo padrão |
| US3 | desligar tudo, ou escolher quais blocos aparecem — `--overlays`/`--overlay-blocks` | os três comandos ganham as duas flags; `fly` também ganha `Overlay` em `FlightRequest`; código de saída `55` |
| US4 | quadros de configurações de sobreposição diferentes nunca são tomados pelo mesmo conjunto | nenhuma lógica nova (já sai da Phase 2, Parte G) — só a prova, em teste e manualmente |

Enquanto a US1 não existe, nenhum bloco é desenhado em lugar nenhum;
enquanto a US2 não existe, `render all` desenha sempre sem sobreposição;
enquanto a US3 não existe, as sobreposições não podem ser desligadas nem
restritas a alguns blocos, e `fly` ainda não as desenha; `ErrInvalidOverlayBlock`
ainda não tem código de saída próprio (cai no genérico `4`).

## Formato: `[ID] [P?] [Story] Descrição`

- **[P]**: pode ser executado em paralelo (arquivos diferentes, sem
  dependência de tarefa incompleta). Tarefas que editam o mesmo arquivo
  NUNCA são marcadas `[P]` entre si, mesmo quando logicamente independentes.
- **[Story]**: a qual história de usuário esta tarefa pertence (US1 a US4).
  Tarefas de Setup, Foundational e Polish não têm esse rótulo.
- Toda tarefa inclui o caminho de arquivo exato a criar/editar.
- Comentários de código, identificadores, mensagens de commit, flags, saída
  e mensagens de erro em tempo de execução: **inglês**; artefatos do Spec
  Kit: português (constituição, "Idioma dos Artefatos").

## Convenções de Caminho

Mesmo projeto único em Go, mesma estrutura hexagonal de `plan.md`:
`cmd/sobrevoo/`, `internal/domain/`, `internal/application/`,
`internal/infra/outbound/{jsonfile,config}`, `internal/infra/inbound/cli/`.
Builders em `internal/domain/builddomain`. Regras de estilo: sem
`ports.go`; `OverlayConfig` em arquivo próprio, sem entidade dona, como
`Appearance` em `frame_appearance.go`; receivers curtos e consistentes com
o tipo (`o` `OverlayConfig`, `e` `ElevationProfile`, `s` `screenOverlay`);
`new(x)` do Go 1.26 em vez de um helper `ptr`; nome exportado nunca repete
o pacote; receiver sem uso fica sem nome.

---

## Phase 1: Setup (Shared Infrastructure)

**Propósito**: um ponto de partida próprio (branch) e verde.

- [X] T001 Criar e mudar para o branch `009-frame-overlays` a partir de `main` (`git checkout -b 009-frame-overlays`)
- [X] T002 Confirmar `make build`, `make test`, `make lint` e `make generate` verdes antes de qualquer mudança (linha de base)

**Checkpoint**: repositório pronto para a Phase 2.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Propósito**: infraestrutura que toda história depende. **Nenhuma história começa antes desta fase estar verde.**

### Parte A — interpolação de tempo e elevação em `Route` (research.md itens 7–8; data-model.md)

- [X] T003 [P] Estender `internal/domain/duration_test.go`: `Route.TimeAt(distances, at)` interpola linearmente o tempo decorrido desde o primeiro ponto — no meio de um segmento, no início (`at=0` → `0`) e no fim (`at=Distances[last]` → `Duration()` total); devolve `ok=false` quando `!allHaveTime()` (mesmo critério de `Duration()`), um `t.Run` por caso
- [X] T004 Implementar `Route.TimeAt(distances []float64, at float64) (time.Duration, bool)` em `internal/domain/duration.go`, ao lado de `Duration()`, reaproveitando `allHaveTime()`; `make test` verde
- [X] T005 [P] Escrever `internal/domain/elevation_profile_test.go` (novo arquivo): `Route.ElevationProfile(distances)` devolve `ok=false` quando `!allHaveElevation()`; para um trajeto com um perfil não-monotônico (subida-descida-subida), `ElevationProfile.At(distance)` interpola a elevação bruta corretamente em vários pontos, e o ganho acumulado interpolado nunca decresce quando a distância cresce; `.At(0)` devolve elevação/ganho do primeiro ponto (ganho `0`); `.At(Distances[last])` devolve elevação do último ponto e ganho **igual, bit a bit, a `Route.ElevationGain()`** do mesmo trajeto (o caso crítico do research.md item 7 — usar o mesmo trajeto nos dois cálculos e comparar com `assert.Equal`, não `InDelta`)
- [X] T006 Implementar `internal/domain/elevation_profile.go` (novo arquivo): `ElevationProfile{Distances, Elevations, Gains []float64}`, `Route.ElevationProfile(distances []float64) (ElevationProfile, bool)` (pré-computa o ganho acumulado ponto a ponto, mesmo algoritmo de `ElevationGain()` guardando cada soma parcial) e `ElevationProfile.At(distance float64) (elevation, gain float64)` (busca por bracket + interpolação linear, mesma técnica de `PlanarRoute.PointAt`); `make test` verde

### Parte B — `CameraFrame`/`CameraPlan`/`PlanSummary`: os campos novos (data-model.md)

- [X] T007 [P] Estender `internal/domain/camera_plan_test.go`: `NewCameraPlan` ganha o parâmetro `elevationAvailable bool`, atualizar as chamadas já existentes; `CameraPlan.ElevationAvailable` e `PlanSummary.ElevationAvailable` refletem o valor passado (dois `t.Run`, `true` e `false`)
- [X] T008 Em `internal/domain/camera_plan.go`: acrescentar `ActivityElapsed time.Duration`, `TrackElevation float64`, `TrackElevationGain float64` a `CameraFrame`; acrescentar `ElevationAvailable bool` a `CameraPlan` e a `PlanSummary`; mudar `NewCameraPlan` para receber `elevationAvailable bool` e preencher os dois campos; `make test` verde
- [X] T009 [P] Estender `internal/domain/builddomain/camera_frame_builder.go` com `WithActivityElapsed(time.Duration)`, `WithTrackElevation(float64)`, `WithTrackElevationGain(float64)` (padrão: todos zero); estender `internal/domain/builddomain/camera_plan_builder.go` com `WithElevationAvailable(bool)` (padrão: `true`)
- [X] T010 [P] Estender `internal/domain/camera_planning_test.go`: para um trajeto tratado com tempo e elevação completos, `PlanCamera` preenche `ActivityElapsed`/`TrackElevation`/`TrackElevationGain` de cada quadro coerentes com a posição do marcador; para um trajeto sem tempo utilizável (recuo por distância), `ActivityElapsed` é `0` em todo quadro; para um trajeto sem elevação em todos os pontos, `TrackElevation`/`TrackElevationGain` são `0` em todo quadro e `CameraPlan.ElevationAvailable` é `false`; para um trajeto com um perfil de elevação não-monotônico, `TrackElevationGain` do último quadro é **igual, bit a bit**, ao `Route.ElevationGain()` do trajeto tratado (o teste de ponta a ponta do research.md item 7) e nunca decresce entre quadros consecutivos
- [X] T011 Em `internal/domain/camera_planning.go`: `cameraPlanner` ganha os campos `trackRoute Route` e `elevation *ElevationProfile`; `PlanCamera` monta `elevationProfile, elevationAvailable := t.Route.ElevationProfile(route.Distances)` e passa `t.Route`/o ponteiro do perfil ao `cameraPlanner`, e `elevationAvailable` a `NewCameraPlan`; `cameraPlanner.frame` preenche `ActivityElapsed` via `c.trackRoute.TimeAt(c.route.Distances, distance)` e `TrackElevation`/`TrackElevationGain` via `c.elevation.At(distance)` (zero quando o ponteiro é `nil` ou `TimeAt` devolve `ok=false`), quantizados com `lengthStep`/a resolução de segundos que `Time` já usa; `make test` verde
- [X] T012 [P] Estender `internal/domain/camera_plan_test.go` (teste de `CameraPlan.ID()`): dois planos cujos quadros diferem **só** em `ActivityElapsed`/`TrackElevation`/`TrackElevationGain` (e em `ElevationAvailable`) produzem o **mesmo** `ID()` (research.md item 9 — nenhuma mudança de implementação, só a prova de que a decisão de não estender o hash está correta)

### Parte C — o plano exportado sobe para a versão 2 (contracts/plan-file-v2.md)

- [X] T013 [P] Estender `internal/infra/outbound/jsonfile/camera_plan_exporter_test.go`: o JSON produzido tem `"format_version": 2`; cada quadro tem `activity_time_s`, `marker.elevation_m`, `marker.gain_m`; `summary` tem `elevation_available`; os valores batem com os do `domain.CameraPlan` de entrada (mesma checagem, campo a campo, que os campos da versão 1 já recebem)
- [X] T014 Em `internal/infra/outbound/jsonfile/camera_plan_file.go`: `planFormatVersion` de `1` para `2`; `frameFile.ActivityTimeSeconds`, `markerFile.Elevation`/`.Gain`, `summaryFile.ElevationAvailable`; `encodePlan` preenche os quatro a partir do `domain.CameraFrame`/`CameraPlan`; `make test` verde
- [X] T015 [P] Estender `internal/infra/outbound/jsonfile/camera_plan_reader_test.go`: um arquivo com `"format_version": 1` é recusado com `ErrPlanFormatVersionUnsupported`, e a mensagem cita a versão encontrada, a aceita, e orienta a gerar o plano de novo; um arquivo `format_version: 2` sem `frames[].activity_time_s`, sem `marker.elevation_m`, sem `marker.gain_m` ou sem `summary.elevation_available` é `ErrPlanFileInvalid` nomeando o campo ausente (quatro `t.Run`); um arquivo `format_version: 2` válido é lido de volta com os quatro campos exatamente iguais aos escritos (ida e volta com `camera_plan_exporter`)
- [X] T016 Em `internal/infra/outbound/jsonfile/camera_plan_reader.go`: `readFrame.ActivityTimeSeconds *float64`, `readMarker.Elevation`/`.Gain *float64`, `readSummary.ElevationAvailable *bool`; a checagem de versão continua sendo a primeira coisa feita em `Read`; as quatro checagens de ausência, no mesmo padrão de `camera_to_marker_m`; construção de `domain.CameraFrame`/`NewCameraPlan` com os quatro valores; `make test` verde

### Parte D — `domain.OverlayConfig` (research.md itens 2–3; data-model.md)

- [X] T017 [P] Escrever `internal/domain/frame_overlay_config_test.go`: `NewOverlayConfig(true, []OverlayBlock{...})` com os quatro nomes liga os quatro `bool`; com um subconjunto, liga só esses; com `enabled=false`, os quatro `bool` ficam `false` **mesmo com blocos informados**; um nome de bloco desconhecido devolve `ErrInvalidOverlayBlock` citando o nome recebido; `Fingerprint()` devolve o mesmo texto para a mesma configuração chamada duas vezes, e um texto diferente quando `Enabled` ou qualquer um dos quatro blocos muda (um `t.Run` por campo, como `Appearance.Fingerprint()` já testa)
- [X] T018 Implementar `internal/domain/frame_overlay_config.go`: `OverlayBlock` (`OverlayBlockDistance`, `OverlayBlockElevation`, `OverlayBlockTime`, `OverlayBlockProfile`), `OverlayConfig{Enabled, Distance, Elevation, Time, Profile bool}`, `NewOverlayConfig(enabled bool, blocks []OverlayBlock) (OverlayConfig, error)`, `Fingerprint() string`; `make test` verde
- [X] T019 [P] Adicionar `ErrInvalidOverlayBlock` a `internal/domain/errors.go`
- [X] T020 [P] Adicionar `builddomain.NewOverlayConfigBuilder()` em `internal/domain/builddomain/overlay_config_builder.go`: padrão `Enabled true`, os quatro blocos `true`; `WithoutDistance()`, `WithoutElevation()`, `WithoutTime()`, `WithoutProfile()`, `WithDisabled()`, `Build() domain.OverlayConfig`

### Parte E — as constantes fixas de desenho (research.md itens 5–6)

- [X] T021 [P] Adicionar `OverlayMarginRatio` (`0.06`, fração de `min(largura, altura)`), `OverlayPanelColor` (`RGB{0x00,0x00,0x00}`), `OverlayPanelOpacity` (`0.55`), `OverlayTextColor` (`RGB{0xFF,0xFF,0xFF}`) a `internal/domain/render_tuning.go`, junto das demais marcas fixas (`PatternPeriod`, `TrailCasingColor`), com o mesmo comentário de "significado, não estilo" que elas já têm

### Parte F — o desenho da sobreposição de tela (research.md itens 1, 4, 5, 6, 11, 12)

- [X] T022 [P] Escrever `internal/domain/frame_screen_overlay_test.go` (novo arquivo): desenhar um texto curto conhecido (ex.: `"12"`) num `FrameImage` de fundo uniforme, numa posição e escala conhecidas, com `golang.org/x/image/font/inconsolata.Bold8x16`, acende exatamente os pixels do glifo esperado (comparar contra a máscara do glifo replicada pelo fator de escala) e deixa pixels distantes do texto inalterados; desenhar o mesmo texto duas vezes produz a mesma imagem byte a byte (determinismo)
- [X] T023 Implementar, em `internal/domain/frame_screen_overlay.go` (novo arquivo), o tipo `screenOverlay` e a primitiva de texto: busca do glifo em `inconsolata.Bold8x16`, escala por replicação inteira de pixel (`roundHalfUp` para o fator, divisão inteira no laço de pixel — nenhum ponto flutuante por pixel), mistura alfa reaproveitando o padrão `mix`/`rounded` já usado em `overlay.blend`; `make test` verde
- [X] T024 [P] Estender `internal/domain/frame_screen_overlay_test.go`: uma placa de fundo (`OverlayPanelColor`/`OverlayPanelOpacity`) sobre um retângulo conhecido produz, em cada pixel coberto, exatamente `old*(1-0.55) + panelColor*0.55` (arredondado como `rounded()` já arredonda) — valor exato, não só "mudou"; nenhum bloco desenha (texto, placa ou o perfil) mais perto de qualquer borda do quadro do que `OverlayMarginRatio × min(largura, altura)`
- [X] T025 Implementar em `frame_screen_overlay.go`: a primitiva de placa (mesma mistura alfa de `blend`) e o cálculo de layout que mantém todo bloco fora da margem de segurança (`OverlayMarginRatio`); `make test` verde
- [X] T026 [P] Estender `internal/domain/frame_screen_overlay_test.go`: as três funções de formatação (research.md item 12) — distância: `"850 m"` abaixo de 1000 m, `"12.3 km"` a partir de 1000 m (casos na fronteira: `999.9 m`, `1000 m`, `1000.001 m`); elevação/ganho: metros inteiros, com sinal no ganho (`"+567 m"`) e sem sinal na elevação (`"1234 m"`); tempo decorrido: sempre `H:MM:SS`, nunca omitindo a hora (`0:05:03`, `1:00:00`) — um `t.Run` por caso de fronteira
- [X] T027 Implementar em `frame_screen_overlay.go` as três funções de formatação (não exportadas, só usadas pelo desenho); `make test` verde
- [X] T028 [P] Estender `internal/domain/frame_screen_overlay_test.go`: com um `OverlayConfig` que só liga `Distance` e `Time`, só esses dois blocos desenham algo (os outros dois não alteram nenhum pixel fora da margem); com `Enabled=false`, nenhum pixel muda; com um `CameraPlan.TimeReference == TimeReferenceDistance`, o bloco de tempo não desenha nada mesmo pedido; com `CameraPlan.ElevationAvailable == false`, os blocos de elevação e de perfil não desenham nada mesmo pedidos (FR-016)
- [X] T029 Implementar em `frame_screen_overlay.go` a seleção de blocos (honra `OverlayConfig` e, por bloco, `CameraPlan.TimeReference`/`ElevationAvailable`) e o desenho dos três blocos de texto (distância; elevação e ganho; tempo decorrido), cada um sobre sua própria placa; `make test` verde
- [X] T030 [P] Estender `internal/domain/frame_screen_overlay_test.go`: o bloco de perfil de elevação desenha uma linha a partir dos pares `(MarkerDistance, TrackElevation)` dos quadros de `PhaseFollowing` do plano, e um ponto na posição do quadro atual; chamado duas vezes para o mesmo plano e o mesmo índice, produz a mesma imagem byte a byte; para dois índices diferentes, o ponto muda de posição e a linha de fundo não
- [X] T031 Implementar em `frame_screen_overlay.go` o desenho do bloco de perfil (linha recalculada a cada chamada, sem cache entre quadros — mesmo padrão que `overlay.drawTrail` já usa); `make test` verde
- [X] T032 [P] Estender `internal/domain/frame_scene_test.go` e `internal/domain/frame_scene_bench_test.go`: atualizar as chamadas existentes de `NewScene` para a assinatura com `overlayConfig`; cenário novo — `Render` com dois `OverlayConfig` diferentes (mesmo plano, recorte, quadro, aparência) produz duas imagens diferentes; o mesmo `OverlayConfig`, chamado duas vezes, produz a mesma imagem byte a byte (reforça SC-003/FR-013)
- [X] T033 Mudar `NewScene` para `NewScene(slice GeoSlice, decoder TileDecoder, tuning RenderTuning, appearance Appearance, overlay OverlayConfig) (*Scene, error)` (campo `overlayConfig OverlayConfig` em `Scene`); `Render` desenha a sobreposição de tela por último, depois de `over.drawTrail`/`over.drawMarker`, em `internal/domain/frame_scene.go`; `make test` verde

### Parte G — a sobreposição na identidade do conjunto de quadros (data-model.md)

- [X] T034 [P] Estender `internal/domain/frame_set_test.go`: atualizar as chamadas existentes de `NewFrameMark`/`NewFrameSetID` para a assinatura com `overlay`; cenário novo — dois `OverlayConfig` diferentes (mesmo plano, recorte, resolução, tuning, aparência) produzem `FrameSetID` diferente; o mesmo `OverlayConfig`, duas vezes, produz o mesmo `FrameSetID`
- [X] T035 Adicionar `Overlay OverlayConfig` a `SingleFrameRequest` e a `FrameSetRequest`; mudar `NewFrameMark`/`NewFrameSetID`/`newFrameSetID` para receber `overlay OverlayConfig` e escrever `overlay.Fingerprint()` no hash, depois de `appearance.Fingerprint()`, em `internal/domain/frame_set.go`; `make test` verde

### Parte H — os padrões em configuração (research.md; data-model.md)

- [X] T036 [P] Estender `internal/infra/outbound/config/config_test.go` (`Test_Load`): `RenderDefaults.OverlaysEnabled` é `true`; `RenderDefaults.OverlayBlocks` tem os quatro nomes (`"distance"`, `"elevation"`, `"time"`, `"profile"`)
- [X] T037 Adicionar `OverlaysEnabled bool` e `OverlayBlocks []string` a `RenderDefaults` e preenchê-los em `Load()`, em `internal/infra/outbound/config/config.go`
- [X] T038 [P] Escrever `Test_domainOverlayConfig` em `cmd/sobrevoo/config_mapping_test.go`: `domainOverlayConfig(config.RenderDefaults{OverlaysEnabled: true, OverlayBlocks: [...]})` devolve o `domain.OverlayConfig` esperado; um nome em `OverlayBlocks` que não é um dos quatro propaga `ErrInvalidOverlayBlock`
- [X] T039 Implementar `domainOverlayConfig(d config.RenderDefaults) (domain.OverlayConfig, error)` em `cmd/sobrevoo/config_mapping.go`, convertendo `OverlayBlocks` (texto) para `[]domain.OverlayBlock` e chamando `domain.NewOverlayConfig`; `make test` verde

**Checkpoint**: `make build`, `make test`, `make lint`, `make generate` verdes; o plano é versão 2, a sobreposição de tela é desenhada por `Scene.Render` com os quatro blocos e entra na identidade do conjunto — mas nenhuma flag existe ainda em nenhum comando, e nenhum serviço de aplicação repassa `Overlay` ainda.

---

## Phase 3: User Story 1 — Ver as estatísticas da atividade sobre o voo (Priority: P1) 🎯 MVP

**Objetivo**: `sobrevoo render frame` desenha os quatro blocos, ligados por
padrão, com os valores corretos — sem nenhuma flag nova ainda.

**Teste Independente**: `quickstart.md` item 2.

- [X] T040 [P] [US1] Estender `internal/application/frame_service_test.go`: `DrawFrame` e `DrawFrames` repassam `request.Overlay` para `domain.NewScene`/`domain.NewFrameMark` sem alteração — duas chamadas com `Overlay` diferente (via `builddomain.NewOverlayConfigBuilder()...Build()`) produzem uma `FrameImage`/`FrameMark` diferentes no `FrameExporter`/`FrameRepository` mockado
- [X] T041 [US1] Em `internal/application/frame_service.go`, `DrawFrame` e `DrawFrames` passam `request.Overlay` para `domain.NewScene(...)` e `domain.NewFrameMark(...)`; `make test` verde
- [X] T042 [P] [US1] Estender `internal/infra/inbound/cli/render_frame_test.go`: `SingleFrameRequest.Overlay` recebe sempre o `defaultOverlay` passado ao comando (nenhuma flag ainda nesta história)
- [X] T043 [US1] Adicionar o parâmetro `defaultOverlay domain.OverlayConfig` a `NewRenderFrameCommand`, preenchendo `SingleFrameRequest.Overlay: defaultOverlay`, em `internal/infra/inbound/cli/render_frame.go`
- [X] T044 [US1] Em `cmd/sobrevoo/main.go`: `defaultOverlay, err := domainOverlayConfig(cfg.RenderDefaults)` (mesmo tratamento de erro que `defaultAppearance` já tem) e passar `defaultOverlay` para `cli.NewRenderFrameCommand(...)`
- [X] T045 [US1] Validação manual: `quickstart.md` item 2 (sem nenhuma flag, os quatro blocos aparecem com os valores corretos)

**Checkpoint**: `render frame` desenha os quatro blocos por padrão — MVP entregue.

---

## Phase 4: User Story 2 — Conferir a sobreposição num quadro isolado antes do voo inteiro (Priority: P2)

**Objetivo**: `sobrevoo render all` desenha os mesmos quatro blocos; um
quadro de `render frame` é byte a byte igual ao mesmo quadro dentro de
`render all`.

**Teste Independente**: `quickstart.md` item 6.

- [X] T046 [P] [US2] Estender `internal/infra/inbound/cli/render_all_test.go`: `FrameSetRequest.Overlay` recebe sempre o `defaultOverlay` passado ao comando
- [X] T047 [US2] Adicionar o parâmetro `defaultOverlay domain.OverlayConfig` a `NewRenderAllCommand`, preenchendo `FrameSetRequest.Overlay: defaultOverlay`, em `internal/infra/inbound/cli/render_all.go`
- [X] T048 [US2] Em `cmd/sobrevoo/main.go`: passar o `defaultOverlay` já calculado (T044) para `cli.NewRenderAllCommand(...)`
- [X] T049 [US2] Validação manual: `quickstart.md` item 6 (quadro isolado idêntico byte a byte ao mesmo quadro de `render all`)

**Checkpoint**: `render frame` e `render all`, com a mesma configuração de sobreposição (a padrão), produzem o mesmo quadro byte a byte.

---

## Phase 5: User Story 3 — Desligar tudo ou escolher só alguns blocos (Priority: P3)

**Objetivo**: `--overlays`/`--overlay-blocks` funcionam, com o mesmo nome e
o mesmo efeito, em `render frame`, `render all` e `fly`; um bloco inválido
tem código de saída próprio (`55`).

**Teste Independente**: `quickstart.md` itens 3, 4 e 5.

- [X] T050 [P] [US3] Escrever `internal/infra/inbound/cli/overlay_test.go` (novo arquivo): `parseOverlay` devolve os padrões quando nenhuma flag foi mudada; `--overlays=false` desliga tudo mesmo com `--overlay-blocks` informado; `--overlay-blocks` com um subconjunto válido liga só esses; um nome desconhecido em `--overlay-blocks` devolve `ErrInvalidOverlayBlock` diretamente (sem `usageError`); `--overlays` com um texto que não é `true`/`false` devolve um erro de uso
- [X] T051 [US3] Implementar `parseOverlay(cmd *cobra.Command, overlaysFlag bool, overlaysChanged bool, blocksFlag string, defaults domain.OverlayConfig) (domain.OverlayConfig, error)` e as strings de uso das duas flags em `internal/infra/inbound/cli/overlay.go`
- [X] T052 [P] [US3] Estender `internal/infra/inbound/cli/render_frame_test.go`: `--overlays`/`--overlay-blocks` chegam em `domain.SingleFrameRequest.Overlay`; omitidas, usam o `defaultOverlay`; um bloco inválido recusa antes de `frameService.DrawFrame` ser chamado
- [X] T053 [US3] Adicionar as duas flags a `NewRenderFrameCommand`, chamando `parseOverlay` antes de `runRenderFrame`, em `internal/infra/inbound/cli/render_frame.go`
- [X] T054 [P] [US3] Estender `internal/infra/inbound/cli/render_all_test.go`: mesmo comportamento das duas flags para `domain.FrameSetRequest.Overlay`
- [X] T055 [US3] Adicionar as duas flags a `NewRenderAllCommand`, chamando `parseOverlay`, em `internal/infra/inbound/cli/render_all.go`
- [X] T056 [P] [US3] Estender `internal/application/flight_service_test.go`: `Fly` repassa `request.Overlay`, sem alteração, para o `domain.FrameSetRequest` que monta para `frameService.DrawFrames`
- [X] T057 [US3] Em `internal/application/flight_service.go`, preencher `Overlay: request.Overlay` no `domain.FrameSetRequest` que `Fly` já monta; `make test` verde
- [X] T058 [P] [US3] Adicionar `Overlay OverlayConfig` a `FlightRequest`, em `internal/domain/flight.go`
- [X] T059 [P] [US3] Em `internal/domain/builddomain/flight_request_builder.go`: `NewFlightRequestBuilder()` preenche `Overlay: NewOverlayConfigBuilder().Build()`; adicionar `WithOverlay(domain.OverlayConfig) *FlightRequestBuilder`
- [X] T060 [P] [US3] Estender `internal/infra/inbound/cli/fly_test.go`: `--overlays`/`--overlay-blocks` chegam em `domain.FlightRequest.Overlay`; omitidas, usam o `defaultOverlay`; um bloco inválido recusa antes de `flightService.Fly` ser chamado, com o mesmo erro que `render frame`/`render all` dariam para o mesmo valor
- [X] T061 [US3] Adicionar as duas flags a `NewFlightCommand`, chamando `parseOverlay`, em `internal/infra/inbound/cli/fly.go`
- [X] T062 [US3] Em `cmd/sobrevoo/main.go`: passar o `defaultOverlay` já calculado (T044) para `cli.NewFlightCommand(...)`
- [X] T063 [P] [US3] Estender `internal/infra/inbound/cli/exit_code_test.go`: `errors.Is(err, domain.ErrInvalidOverlayBlock)` → `55`
- [X] T064 [US3] Adicionar o `case` a `internal/infra/inbound/cli/exit_code.go` (`55`, logo após o `54` existente)
- [X] T065 [US3] Validação manual: `quickstart.md` itens 3, 4 e 5 (desligar tudo, blocos parciais, bloco inválido recusado)

**Checkpoint**: `--overlays`/`--overlay-blocks` funcionam, com o mesmo nome e o mesmo efeito, em `render frame`, `render all` e `fly`; código `55` mapeado.

---

## Phase 6: User Story 4 — Nunca confundir quadros com sobreposições diferentes (Priority: P4)

**Objetivo**: confirmar — sem nenhuma lógica nova (já sai da Phase 2, Parte
G) — que retomar `render all` ou reaproveitar `fly --keep` com uma
configuração de sobreposição diferente da já presente redesenha (ou recusa
sem `--overwrite`), e que a mesma configuração sempre reaproveita.

**Teste Independente**: `quickstart.md` itens 8 e 9.

- [X] T066 [P] [US4] Estender `internal/application/frame_service_test.go`: `DrawFrames`, com o `FrameRepository` mockado devolvendo um `FrameDirectory` de quadros já lá com um `SetID` de outro `OverlayConfig` (mesmo plano/recorte/resolução/aparência), recusa com `ErrFrameSetConflict` sem `Overwrite`; com o mesmo `OverlayConfig` de antes, reaproveita (`summary.Kept` bate com o total)
- [X] T067 [P] [US4] Estender `internal/application/flight_service_test.go`: duas chamadas de `Fly` com `Keep` preenchido, mesmo trajeto e mesmos parâmetros, mas `Overlay` diferente — `PlanReused`/`SliceReused` continuam `true` (não dependem de sobreposição) e `frameService.DrawFrames` é chamado para o conjunto novo; com o mesmo `Overlay` de antes, os quadros também são reaproveitados
- [X] T068 [US4] Validação manual: `quickstart.md` itens 8 e 9 (retomada de `render all` recusando/reaproveitando por sobreposição; `fly --keep` refazendo só quadros e vídeo quando só a sobreposição muda)

**Checkpoint**: nenhuma combinação de execuções mistura, no mesmo diretório ou no mesmo `--keep`, quadros de configurações de sobreposição diferentes.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Propósito**: os casos que não pertencem a nenhuma história específica, a documentação e a verificação final.

- [X] T069 [P] Validação manual: `quickstart.md` itens 1, 7 e 10 (plano de versão anterior recusado; valores do último quadro batendo com `inspect`; margem de segurança respeitada em vídeo vertical)
- [X] T070 [P] Atualizar `CLAUDE.md`: a nona etapa, o formato do plano na versão 2, `OverlayConfig`, os quatro blocos, a fonte embutida, as duas flags nos três comandos
- [X] T071 [P] Atualizar `README.md`: as duas flags novas nas seções de `render frame`, `render all` e `fly` já existentes (mesma estrutura de tabela de flags que a aparência já tem)
- [X] T072 Verificação final: `make build`, `make test`, `make lint`, `make generate` verdes (confirmar que `make generate` não produz nenhuma diferença); `gofmt -l .` limpo; reexecutar `quickstart.md` por inteiro

---

## Dependências e Ordem de Execução

### Dependências entre fases

- **Phase 1 (Setup)**: sem dependências.
- **Phase 2 (Foundational)**: depende da Phase 1; **bloqueia todas as histórias**. Partes A → B → C são sequenciais (B usa `Route.TimeAt`/`ElevationProfile` de A; C serializa os campos que B declara). Parte D é independente de A–C (pode rodar em paralelo). Parte E é independente de tudo. Parte F depende de B (campos do quadro), D (`OverlayConfig`) e E (constantes) — é a maior parte, com suas próprias sub-dependências internas (texto → placa/margem → formatação → seleção de blocos → perfil → `Scene`, nesta ordem, cada uma usando a anterior). Parte G depende de D. Parte H depende de D.
- **Phases 3 a 6 (US1 a US4)**: dependem da Phase 2 concluída; cada uma edita os mesmos arquivos de aplicação/CLI que a anterior estendeu, então seguem a ordem P1 → P4. US2 a US4 dependem da US1 (o `defaultOverlay` calculado em `main.go` e repassado por `FrameService`).
- **Phase 7 (Polish)**: depende das histórias desejadas.

### Dependências entre histórias

- **US1 (P1)**: depende só da Foundational. Faz `FrameService` repassar `Overlay` e `main.go` calcular `defaultOverlay`, usado por todas as demais.
- **US2 (P2)**: depende da US1 (`defaultOverlay` em `main.go`).
- **US3 (P3)**: depende da US1 e da US2 (edita `render_frame.go`/`render_all.go` que ambas já tocaram) e cria `overlay.go` (`parseOverlay`), usado por `fly.go` também nesta mesma história.
- **US4 (P4)**: depende da US1 (precisa de `Overlay` chegando a `DrawFrames`/`Fly`) e, para o cenário de `fly --keep`, da US3 (`fly` só ganha as flags ali); só acrescenta teste e validação manual, nenhuma implementação nova.

### Dentro de cada história

- Testes antes da implementação (devem falhar primeiro — ver a ressalva do preâmbulo para as tarefas de mudança de assinatura).
- Domínio antes de aplicação; aplicação antes da CLI; CLI antes de `main.go`.
- A história é completa antes da próxima prioridade.

### Oportunidades de paralelismo

```text
# Foundational, Parte A, em paralelo entre si:
T003 duration_test.go   T005 elevation_profile_test.go (novo arquivo)
→ T004 duration.go   → T006 elevation_profile.go

# Foundational, Parte B (sequencial: cada tarefa usa a anterior):
T007/T008 (camera_plan) → T009 (builders) → T010/T011 (camera_planning) → T012 (regressão do ID())

# Foundational, Parte C (sequencial, depende da B):
T013/T014 (exporter) → T015/T016 (reader)

# Foundational, Parte D, em paralelo com A/B/C:
T017 frame_overlay_config_test.go → T018 frame_overlay_config.go
T019 errors.go   T020 overlay_config_builder.go (paralelos entre si e com T017/T018)

# Foundational, Parte E, em paralelo com A–D:
T021 render_tuning.go

# Foundational, Parte F (sequencial, depende de B+D+E):
T022/T023 (texto) → T024/T025 (placa+margem) → T026/T027 (formatação) →
T028/T029 (seleção de blocos + texto) → T030/T031 (perfil) → T032/T033 (Scene)

# Foundational, Parte G (depende de D):
T034 frame_set_test.go → T035 frame_set.go

# Foundational, Parte H (depende de D), em paralelo entre si:
T036 config_test.go → T037 config.go
T038 config_mapping_test.go → T039 config_mapping.go

# US1:
T040 frame_service_test.go → T041 frame_service.go
T042 render_frame_test.go → T043 render_frame.go → T044 main.go → T045 manual

# US3: os testes de cada arquivo são [P] entre si (arquivos de teste
# diferentes), mas T053/T055/T057/T061/T062/T064 editam arquivos em
# sequência (cada um depende do anterior estar compilando) — nunca em
# paralelo entre si.
```

---

## Estratégia de Implementação

### MVP Primeiro (Phase 1 + 2 + User Story 1)

1. Phase 1 (branch e linha de base) e Phase 2 (`Route.TimeAt`/`ElevationProfile`; os três campos do quadro; o plano na versão 2; `OverlayConfig`; o desenho da sobreposição de tela inteiro, ligado a `Scene`; a identidade do conjunto; os padrões em configuração).
2. Phase 3 (US1): `FrameService` repassando `Overlay`, `defaultOverlay` em `main.go`, `render frame` desenhando os quatro blocos por padrão.
3. **PARE e valide**: `quickstart.md` item 2 — os quatro blocos aparecem com os valores corretos — antes de acrescentar histórias.

### Entrega Incremental

1. Setup + Foundational → sobreposição de tela pronta no núcleo, sem nenhum comando desenhando-a ainda.
2. + US1 → `render frame` desenha os quatro blocos por padrão (MVP) → item 2.
3. + US2 → `render all` igual, byte a byte, a `render frame` → item 6.
4. + US3 → `--overlays`/`--overlay-blocks` nos três comandos, código `55` → itens 3, 4, 5.
5. + US4 → confirmação de que configurações diferentes nunca se misturam → itens 8, 9.
6. Polish → plano antigo recusado, valores batendo com `inspect`, margem respeitada, documentação, verificação final → itens 1, 7, 10.

Cada história agrega valor sem quebrar as anteriores.

---

## Notas

- `[P]` = arquivos diferentes, sem dependência de tarefa incompleta.
- Nenhuma regra de câmera, de enquadramento, de terreno, de traçado ou de
  duração do vídeo nasce nesta etapa (FR-015): toda tarefa de implementação
  só acrescenta um jeito de calcular/exibir dado que o plano e o recorte já
  carregam — se uma tarefa parecer exigir reler o trajeto GPS ou os dados
  geográficos registrados durante o desenho, algo está errado (revisar
  FR-005 e research.md item 7 antes de prosseguir).
- O campo `TrackElevationGain` (não uma soma corrente ingênua por quadro) é
  o único jeito, documentado no research.md item 7, de garantir FR-014 —
  não simplificar essa parte para "economizar" um campo.
- Faça commit após cada tarefa ou grupo lógico (só quando o usuário pedir).
- Pare em qualquer checkpoint para validar a história com o `quickstart.md`.
- Evite: tarefas vagas, duas tarefas `[P]` editando o mesmo arquivo,
  dependências que quebrem a independência de teste de uma história.
