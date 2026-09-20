---

description: "Task list for feature implementation"
---

# Tarefas: Planejamento do Movimento de Câmera

**Entrada**: Documentos de design de `/specs/003-camera-path-planning/`

**Pré-requisitos**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/cli.md`, `contracts/plan-file.md`, `quickstart.md`

**Testes**: incluídos. Como nas etapas anteriores, a constituição do projeto
(Princípio VI — Testes Automatizados no Núcleo; Princípio X —
Given/When/Then, builders e isolamento por camada) exige testify e uber-go/mock
e proíbe testes tabulares: cada cenário é um `t.Run("should ...")` com
`// given`, `// when`, `// then`. As tarefas de teste abaixo materializam
essa exigência; ficam antes da implementação correspondente em cada fase.

**Organização**: as tarefas são agrupadas por história de usuário (P1–P5 de
`spec.md`) para permitir implementação e teste independentes de cada
história. A **Phase 2** tem duas partes: (A) a refatoração prévia das etapas
1 e 2 (`TrackService`, `research.md` item 13), que precisa estar concluída e
verde antes de qualquer código de câmera; e (B) a base compartilhada de
câmera (tipos, erros, builders, configuração, códigos de saída).

**Divisão do algoritmo entre histórias** (o mesmo arquivo de domínio é
estendido por mais de uma história, nunca em paralelo):

| História | O que entrega no algoritmo |
|---|---|
| US1 | projeção local (funções exportadas para teste isolado, `research.md` item 17), linha do tempo linear (horário ou distância), trilhas de rumo/inclinação/distância, `PlanCamera` só com a fase de acompanhamento, duração automática pela curva (sem o piso do mínimo), comando `plan <arquivo>` |
| US2 | limitação de taxa + suavização gaussiana + trechos suavizados, abertura e fechamento (10% / 80% / 10%, visão geral com o rumo do acompanhamento), testes com duração explícita de 120 s |
| US3 | flags `--duration/--fps/--distance/--tilt`, validações (FR-016 a FR-018, FR-017a, FR-017b), compressão de paradas, `MinimumDuration`, piso da duração automática pelo mínimo |
| US4 | resumo completo (faixas, trechos suavizados, referência de tempo, modo da duração) |
| US5 | exportação JSON atômica, `--export` e `--overwrite` |

## Formato: `[ID] [P?] [Story] Descrição`

- **[P]**: pode ser executado em paralelo (arquivos diferentes, sem
  dependência de tarefa incompleta). Tarefas que editam o mesmo arquivo NUNCA
  são marcadas `[P]` entre si, mesmo quando logicamente independentes.
- **[Story]**: a qual história de usuário esta tarefa pertence (US1 a US5).
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
porta no arquivo da entidade, no topo, com `//go:generate` logo após
`package`; receivers curtos e consistentes com o tipo; `new(x)` do Go 1.26
em vez de um helper `ptr`.

---

## Phase 1: Setup (Shared Infrastructure)

**Propósito**: garantir um ponto de partida verde. Nenhuma dependência nova
(`plan.md`: somente biblioteca padrão) e nenhum diretório novo precisa ser
criado à mão.

- [X] T001 Rodar `make test`, `make lint` e `make generate` na branch `003-camera-path-planning` e confirmar que os três passam e que `make generate` não produz diff (`git status` limpo fora de `specs/003-camera-path-planning/`); registrar qualquer falha pré-existente antes de continuar

**Checkpoint**: baseline verde.

---

## Phase 2: Foundational (Blocking Prerequisites)

**⚠️ CRÍTICO**: nenhuma tarefa de história de usuário pode começar até que
esta fase esteja completa.

### Parte A — Refatoração prévia: `TrackService` (research.md item 13)

**Propósito**: eliminar a duplicação de `Parse → CleanTrack → Simplify →
Smooth` entre `InspectTrackService`, `GeoDataService` e o futuro
`CameraPlanService`, e absorver `InspectTrackService` (serviço de método
único) num serviço por recurso. **Nenhum comportamento das etapas 1 e 2
muda**: os cenários de teste existentes são migrados com as mesmas
expectativas.

- [X] T002 [P] Declarar em `internal/domain/track.go` os DTOs `CleanedTrack` (`Track Track`, `Points []TrackPoint`, `Discarded DiscardStats`) e `TreatedTrack` (`Track Track`, `Route Route`, `Discarded DiscardStats`), abaixo de `Track` e sem alterar a ordem existente do arquivo (`data-model.md`, "CleanedTrack e TreatedTrack")
- [X] T003 Criar `internal/application/track_service.go` com a diretiva `//go:generate go run go.uber.org/mock/mockgen -destination mockapplication/track_service.go -package mockapplication . TrackService` logo acima da interface `TrackService` (`Clean(reader io.Reader) (domain.CleanedTrack, error)`, `Treat(reader io.Reader, simplificationLevel, smoothingLevel domain.Level) (domain.TreatedTrack, error)`, `Inspect(reader io.Reader, simplificationLevel, smoothingLevel domain.Level) (domain.TrackSummary, error)`), a struct `trackService` (receiver `s`) e `NewTrackService(parser domain.TrackParser, simplifier domain.Simplifier, smoother domain.Smoother, minPoints int, maxPlausibleSpeedKmh float64) TrackService`. `Clean` = `Parse` → `domain.CleanTrack`; `Treat` = `Clean` → `Simplify` → `Smooth`; `Inspect` = `Treat` → `domain.SummarizeTrack`. Mover a lógica e os comentários de `inspect_track_service.go` sem mudar o comportamento Ao final, rodar `make generate` para gerar `internal/application/mockapplication/track_service.go`, necessário para os testes de T006 e T007 (depende de T002)
- [X] T004 Criar `internal/application/track_service_test.go` migrando **todos** os cenários de `inspect_track_service_test.go` para `Test_trackService_Inspect` (mesmas expectativas; reaproveitar `passthroughSimplifier`/`passthroughSmoother`) e acrescentar `Test_trackService_Clean` (propaga erro do parser; propaga erro de `CleanTrack`; devolve pontos limpos e `DiscardStats` sem chamar `Simplifier`/`Smoother`) e `Test_trackService_Treat` (chama `Simplify` e `Smooth` na ordem, com os níveis recebidos; propaga erro de `Clean`). Mocks de `mockdomain` (depende de T003)
- [X] T005 Editar `internal/application/geo_data_service.go`: remover os campos e parâmetros `parser domain.TrackParser`, `minPoints` e `maxPlausibleSpeedKmh`; adicionar `trackService TrackService`; `NewGeoDataService(repository domain.GeoDataRepository, inspector domain.GeoDataInspector, fileChecker domain.FileChecker, trackService TrackService) GeoDataService`; `CheckCoverage` passa a chamar `s.trackService.Clean(reader)` e usar `CleanedTrack.Points` (mantendo o comentário de que a rota de cobertura é limpa mas não simplificada/suavizada, research.md item 9 da etapa 2). Atualizar o comentário do tipo que cita `InspectTrackService` (depende de T003)
- [X] T006 Editar `internal/application/geo_data_service_test.go` para o novo construtor: os cenários de `CheckCoverage` passam a usar `mockapplication.NewMockTrackService` (`Clean` devolvendo `domain.CleanedTrack`) em vez de `TrackParser` + limiares; remover as constantes `testMinPoints`/`testMaxPlausibleSpeedKmh` se ficarem sem uso; acrescentar "should propagate TrackService.Clean's error unchanged". Demais cenários (Register/List/Remove) só mudam a chamada do construtor (depende de T005)
- [X] T007 Editar `internal/infra/inbound/cli/inspect.go` e `internal/infra/inbound/cli/inspect_test.go`: `NewInspectCommand(trackService application.TrackService, defaultLevel domain.Level)` chamando `trackService.Inspect(...)`; o teste usa `mockapplication.NewMockTrackService`, com os mesmos cenários e expectativas. Atualizar também os comentários que citam `InspectTrackService` em `internal/infra/inbound/cli/geodata_check.go`, `internal/domain/cleaning.go` e `internal/domain/track_summary.go` (depende de T003)
- [X] T008 Editar `cmd/sobrevoo/main.go`: criar `trackService := application.NewTrackService(parser, douglasPeucker, catmullRom, cfg.MinPoints, cfg.MaxPlausibleSpeedKmh)`; `application.NewGeoDataService(geoDataRepository, geoDataInspector, geoDataFileChecker, trackService)`; `cli.NewInspectCommand(trackService, cfg.DefaultLevel)`; remover `inspectTrackService` (depende de T005, T007)
- [X] T009 Remover `internal/application/inspect_track_service.go`, `internal/application/inspect_track_service_test.go` e `internal/application/mockapplication/inspect_track_service.go`; rodar `make generate` (não deve haver diff além da remoção), `make test` e `make lint` (depende de T004, T006, T007, T008)

**Checkpoint A**: `make test`, `make lint` e `make generate` verdes; `sobrevoo
inspect` e `sobrevoo geodata check` com o mesmo comportamento de antes
(mesmos testes, mesmos códigos de saída). Nenhum código de câmera existe
ainda.

### Parte B — Base compartilhada de câmera

- [X] T010 [P] Declarar os 7 erros sentinela novos em `internal/domain/errors.go` (estendendo o arquivo existente, cada um com comentário em inglês): `ErrInvalidDuration`, `ErrInvalidFrameRate`, `ErrDurationTooShort`, `ErrTrackTooShort`, `ErrTrackTooLarge`, `ErrPlanDestinationExists`, `ErrPlanDestinationInvalid` (`data-model.md`, "Erros sentinela novos"; Princípio VII)
- [X] T011 [P] Criar `internal/domain/camera_plan_parameters.go` com `PlanParameters` (`Duration *time.Duration` — `nil` = automática; se informada, "> 0" e "≤ 3 600 s" (`ErrInvalidDuration`, mensagem com o máximo); `FrameRate float64` — "1 ≤ valor ≤ 120" (`ErrInvalidFrameRate`), NaN e infinito também inválidos; `Distance Level`; `Tilt Level`), `Validate() error` (receiver `p`; a taxa é sempre validada e a duração só é validada quando não for `nil`; mensagens `fmt.Errorf("%w: ...", sentinela)` nomeando o parâmetro e, para a taxa, o intervalo `1 to 120`), `FrameCount(duration time.Duration) int` (`round(duration.Seconds() × FrameRate)`, arredondamento para o inteiro mais próximo, empates para cima) e `CameraTuning` com todos os campos de `data-model.md` (abertura/fechamento, parada, limites de taxa, sigma, visão geral, mínimos, duração automática, tabelas por `Level`: `BaseDistanceMeters`, `LookAheadSeconds`, `TiltDegrees`, mais `OverviewMinDistanceFactor` (piso da distância da visão geral) e `MaxTrackSpanMeters` (abrangência máxima))
- [X] T012 Criar `internal/domain/camera_plan_parameters_test.go`: `Test_PlanParameters_Validate` (duração `nil` é válida; duração 0, negativa e acima de 3 600 s → `ErrInvalidDuration`; 3 600 s exatos válidos; taxa 0, negativa, 0.999, 120.001, NaN, +Inf → `ErrInvalidFrameRate`; taxa 1 e 120 válidas; mensagem cita o intervalo) e `Test_PlanParameters_FrameCount` (60 s × 30 → 1800; 45.5 s × 29.97 arredondado; empate `.5` para cima) (depende de T011)
- [X] T013 [P] Criar `internal/domain/camera_plan.go` com a diretiva `//go:generate go run go.uber.org/mock/mockgen -destination mockdomain/camera_plan_exporter.go -package mockdomain . CameraPlanExporter` logo após `package`, a interface `CameraPlanExporter` (`Export(plan CameraPlan, path string, overwrite bool) error`) no topo (após os imports), e depois `Phase` (`PhaseOpening`, `PhaseFollowing`, `PhaseClosing`), `TimeReference` (`TimeReferenceClock`, `TimeReferenceDistance`), `DurationMode` (`DurationModeAutomatic`, `DurationModeExplicit`), `SmoothedQuantity` (`QuantityHeading`, `QuantityTilt`, `QuantityZoom`, `QuantityTargetSpeed`), `SmoothedSpan` (`Start`, `End time.Duration`, `Quantity`), `CameraFrame` (campos e unidades de `data-model.md`; longitude "em `[-180, 180)`", rumo "em `[0, 360)`", inclinação "em `[0, 90]`"), `PlanSummary` e `CameraPlan` (`Parameters` com `Duration` sempre preenchida, `TimeReference`, `TimeFallbackReason`, `Frames`, `Summary`). Criar também `NewCameraPlan(parameters PlanParameters, durationMode DurationMode, timeReference TimeReference, fallbackReason string, frames []CameraFrame, spans []SmoothedSpan) CameraPlan`, por ora preenchendo no `Summary` só `Duration`, `DurationMode`, `FrameRate`, `FrameCount`, `TimeReference` e `SmoothedSpans` (as faixas de altitude/distância vêm na US4) (depende de T011, pois `CameraPlan.Parameters` usa `PlanParameters`)
- [X] T014 Rodar `make generate` para gerar `internal/domain/mockdomain/camera_plan_exporter.go` (depende de T013)
- [X] T015 [P] Criar `internal/domain/builddomain/plan_parameters_builder.go`: `NewPlanParametersBuilder()` (defaults: duração `new(60 * time.Second)`, taxa 30, `LevelMedium`/`LevelMedium`), `WithDuration`, `WithoutDuration`, `WithFrameRate`, `WithDistance`, `WithTilt`, `Build()`; receiver `b` (depende de T011)
- [X] T016 [P] Criar `internal/domain/builddomain/camera_frame_builder.go` (`NewCameraFrameBuilder()` com defaults sensatos, `WithIndex`, `WithPhase`, `WithCameraAltitude`, `WithHeading`, `WithTilt`, `WithMarkerDistance`, `WithCameraToMarkerDistance`, `Build()`) e `internal/domain/builddomain/camera_plan_builder.go` (`NewCameraPlanBuilder()` com 3 quadros, distância média, `WithFrames`, `WithSummary`, `WithTimeReference`, `Build()`) (depende de T013)
- [X] T017 [P] Criar `internal/domain/builddomain/synthetic_route_builder.go`: `NewSyntheticRouteBuilder()` (receiver `b`) que produz `[]domain.TrackPoint` sintéticos para os testes geométricos, com métodos fluentes no padrão `WithCampo(...)`: `WithOrigin(lat, lon float64)`, `WithLine(lengthMeters, bearingDegrees float64)`, `WithCircle(radiusMeters float64, laps int)`, `WithOutAndBack(lengthMeters float64)`, `WithUTurn(lengthMeters float64)`, `WithPointCount(n int)`, `WithConstantSpeed(metersPerSecond float64)` (preenche `Time`), `WithoutTime()` e `Build()` terminal; a geometria usa deslocamento geodésico esférico (destino a partir de origem, rumo e distância), sem projeção plana, para valer em antimeridiano e latitudes altas. Acompanhar de `synthetic_route_builder_test.go` verificando que `TotalDistance` do resultado bate com o comprimento pedido (tolerância de 0,1%) numa origem no equador, em `lon = 179.9` e em `lat = 85` (sem dependências: usa só `domain.TotalDistance`, já existente)
- [X] T018 [P] Editar `internal/infra/outbound/config/config.go`: acrescentar a `Config` os campos `CameraTuning domain.CameraTuning` e `DefaultPlanParameters domain.PlanParameters`, preenchidos em `Load()`. `CameraTuning` recebe os valores iniciais de `research.md` (abertura 0,10 e fechamento 0,10; parada: 0,5 m/s, mín. 30 s, teto 2 s, máx. 5% do tempo em movimento; taxas: 45 °/s de rumo, 30 °/s de inclinação, 1,5 /s de `ln D`, alvo 1,0 D/s; sigma 0,5 s; visão geral: inclinação 60°, FOV vertical 45°, margem 1,2, `OverviewMinDistanceFactor` 2; `MinFollowDuration` 5 s, `MinPhaseDuration` 2 s; `MinTrackLengthMeters` 50, `MaxTrackSpanMeters` 2 000 000; duração automática: base 15 s, 6 s por √km, piso 20 s, teto 120 s; níveis `low/medium/high`: distância base 300/600/1200 m, antecipação 2/4/8 s, inclinação 25/45/65°). `DefaultPlanParameters` recebe `Duration: nil` (automática), `FrameRate: 30`, `Distance: LevelMedium`, `Tilt: LevelMedium` (`research.md` item 12; distinto de `DefaultLevel`, que é só o nível de tratamento do trajeto); estender `config_test.go` conferindo esses valores (depende de T011)
- [X] T019 [P] Editar `internal/infra/inbound/cli/exit_code.go` mapeando `ErrInvalidDuration`→10, `ErrInvalidFrameRate`→11, `ErrDurationTooShort`→12, `ErrTrackTooShort`→13, `ErrTrackTooLarge`→14, `ErrPlanDestinationExists`→15, `ErrPlanDestinationInvalid`→16 (com `errors.Is`, para reconhecer erros embrulhados); criar ou estender o teste do mapeamento em `internal/infra/inbound/cli/exit_code_test.go` (um `t.Run` por sentinela, incluindo o erro embrulhado com `fmt.Errorf("%w: ...")`, e um que confirma que os códigos 1–9 existentes não mudaram) (depende de T010)

**Checkpoint B**: tipos de câmera, erros, mocks, builders, configuração e
códigos de saída prontos e testados; ainda nenhum comando `plan` executável.

---

## Phase 3: User Story 1 - Gerar o plano de câmera de um trajeto (Priority: P1) 🎯 MVP

**Objetivo**: `sobrevoo plan <arquivo>` gera, com parâmetros padrão, um plano
determinístico com um quadro por instante do vídeo (posição da câmera,
direção, inclinação, marcador), no plano tangente local, e imprime um resumo
básico.

**Escopo desta fase**: só a fase de acompanhamento (sem abertura/fechamento,
sem limitação de taxa: US2), sem flags de parâmetros (US3), sem exportação
(US5), resumo básico (US4).

**Teste independente**: `go run ./cmd/sobrevoo plan amostras/pedalada.gpx`
imprime duração automática, taxa, quantidade de quadros; a saída é idêntica
em execuções repetidas.

### Testes da User Story 1

- [X] T020 [P] [US1] Criar `internal/domain/camera_projection_test.go` (pacote `domain_test`): `Test_NewLocalPlane` (centroide por média de vetores unitários: trajeto em `lon = 179.9`/`-179.9` tem centroide perto de 180 e não de 0; trajeto a `lat = 85` não colapsa), `Test_LocalPlane_ProjectUnproject` (ida e volta com erro < 1 mm de distância em equador, antimeridiano e latitudes 85 e −85; longitude devolvida normalizada em `[-180, 180)`), `Test_LocalPlane_Project` (distância no plano ≈ `Haversine` com tolerância de 0,1% nos mesmos três casos — base de FR-015/SC-007)
- [X] T021 [P] [US1] Criar `internal/domain/camera_timeline_test.go`: `Test_BuildMarkerTimeline` (pacote `domain_test`) — usa horário quando todos os pontos têm horário, duração total positiva e nenhum segmento com `dt ≤ 0` e deslocamento maior que 1 m; recua para distância (com motivo `"no time data"` ou `"time data is inconsistent"`) quando falta horário em algum ponto, quando a duração total é zero, ou quando há segmento com `dt ≤ 0` e deslocamento > 1 m; `s(t)` é monotônica não decrescente; começa em 0 e termina no comprimento total; velocidade constante no horário ⇒ avanço linear no tempo de vídeo
- [X] T022 [P] [US1] Criar `internal/domain/camera_motion_test.go` (pacote `domain_test`): `Test_DesiredHeading` (trajeto reto para leste ⇒ rumo 90°; para norte ⇒ 0°; vetores que se cancelam num retorno ⇒ mantém o rumo do quadro anterior), `Test_UnwrapAngles` (menor arco; desempate anti-horário para diferença de exatamente 180°; nunca salta 360°), `Test_FollowDistance` (`D = BaseDistanceMeters[level] + LookAheadSeconds[level] × velocidade do marcador no vídeo`; `high` > `low` para a mesma velocidade), `Test_ComputeCameraPose` (câmera atrás do alvo: `posição = T − D·cos θ·(sen ψ, cos ψ)`, `altitude = D·sen θ`)
- [X] T023 [P] [US1] Criar `internal/domain/camera_planning_test.go` com `Test_PlanCamera` (só acompanhamento): número de quadros == `FrameCount(duração)`; todos os quadros na fase `following`; `MarkerDistance` começa em 0, termina no comprimento total e nunca decresce; determinístico (100 chamadas com a mesma entrada ⇒ 100 planos `assert.Equal`, SC-003); valores quantizados ("latitude/longitude 1e-7°, alturas e distâncias 1e-3 m, ângulos 1e-3°"); longitude em `[-180, 180)` em trajeto que cruza 180°; sem `NaN` nem infinito; trajeto sem horário gera plano com `TimeReferenceDistance`; e `Test_DefaultDuration` (curva `clamp(15 s + 6 s·√(km), 20 s, 120 s)` arredondada ao segundo: 1 km → 21 s, 5 km → 28 s, 20 km → 42 s, 50 km → 57 s, 200 km → 100 s, 400 km → 120 s; monotônica não decrescente; sublinear: dobrar o comprimento não dobra a duração; determinística; assinatura `DefaultDuration(points, frameRate, distance, tilt, tuning)`)
- [X] T024 [P] [US1] Criar `internal/application/camera_plan_service_test.go` com `Test_cameraPlanService_Generate`: propaga o erro de `TrackService.Treat` sem alteração; chama `Treat` com o nível padrão de tratamento nos dois níveis; delega a `domain.PlanCamera` com os pontos de `TreatedTrack.Route`; devolve o plano; usa `mockapplication.NewMockTrackService`
- [X] T025 [P] [US1] Criar `internal/infra/inbound/cli/plan_test.go` com `Test_PlanCommand_Execute`: construído com `NewPlanCommand(service, defaults)`; exige exatamente um argumento (erro de uso, código 2); com `CameraPlanService` mockado devolvendo um `CameraPlan` do `CameraPlanBuilder`, imprime `Duration:`, `Frame rate:` e `Frames:` conforme `contracts/cli.md`; propaga erros do serviço sem mudar (o mapeamento para códigos é do `ExitCode`); erro ao abrir o arquivo é devolvido (código 4); nunca instancia serviço ou adapter real

### Implementação da User Story 1

- [X] T026 [US1] Criar `internal/domain/camera_projection.go`: `LocalPlane` (centro) com `NewLocalPlane(points []TrackPoint) LocalPlane` calculando o centroide pela média de vetores unitários 3D normalizada, `(l LocalPlane) Project(lat, lon float64) (x, y float64)` e `Unproject(x, y float64) (lat, lon float64)` pela projeção azimutal equidistante (`research.md` item 2), longitude devolvida normalizada em `[-180, 180)`; tipos e funções exportados, cada um com comentário de documentação em inglês (`research.md` item 17); usar `earthRadiusMeters` de `distance.go`; conversões explícitas `float64(x*y) + z` nos pontos sensíveis a FMA (`research.md` item 9) (depende de T020)
- [X] T027 [US1] Criar `internal/domain/camera_timeline.go`: `BuildMarkerTimeline(points []TrackPoint, plane LocalPlane, tuning CameraTuning) MarkerTimeline` (tipo exportado) devolvendo a distância acumulada por ponto, a `TimeReference`, o motivo do recuo e uma função `s(fraction)` monotônica; regra de uso do horário e motivos conforme `research.md` item 3 (passos 1 e 3–4; a compressão de paradas, passo 2, é da US3); ordem de soma fixa, sem `map`; documentação em inglês (depende de T021, T026)
- [X] T028 [US1] Criar `internal/domain/camera_motion.go` com as funções exportadas `DesiredHeading` (soma ponderada gaussiana das tangentes numa janela `L = 2·D` de comprimento de arco; mantém o valor anterior se a norma cai abaixo do limiar), `UnwrapAngles` (menor arco, desempate anti-horário), `FollowDistance` (`BaseDistanceMeters[level] + LookAheadSeconds[level] × velocidade do marcador no vídeo`, suavizada), `ComputeCameraPose` (devolve o tipo exportado `CameraPose`: alvo + deslocamento orbital, altitude `D·sen θ`, `θ = TiltDegrees[level]`), e a quantização final não exportada (lat/lon 1e-7°, alturas e distâncias 1e-3 m, ângulos 1e-3°), coberta via `PlanCamera`; documentação em inglês (depende de T022, T026)
- [X] T029 [US1] Criar `internal/domain/camera_planning.go` com `PlanCamera(points []TrackPoint, parameters PlanParameters, tuning CameraTuning) (CameraPlan, error)` — por ora todos os quadros em `PhaseFollowing`, `DurationMode` conforme `parameters.Duration == nil` — e `DefaultDuration(points []TrackPoint, frameRate float64, distance, tilt Level, tuning CameraTuning) time.Duration` (curva `clamp(AutoDurationBase + AutoDurationPerSqrtKm·√(km), AutoDurationMin, AutoDurationMax)` arredondada ao segundo, comprimento = `TotalDistance`; os parâmetros `frameRate`, `distance` e `tilt` só passam a ser usados na US3, quando o `max` com a duração mínima entra). Preencher `CameraPlan` via `NewCameraPlan` (depende de T023, T027, T028)
- [X] T030 [US1] Criar `internal/application/camera_plan_service.go` com a diretiva `//go:generate go run go.uber.org/mock/mockgen -destination mockapplication/camera_plan_service.go -package mockapplication . CameraPlanService` logo acima da interface `CameraPlanService` (por ora só `Generate(reader io.Reader, parameters domain.PlanParameters) (domain.CameraPlan, error)`), a struct `cameraPlanService` (receiver `s`) e `NewCameraPlanService(trackService TrackService, exporter domain.CameraPlanExporter, defaultLevel domain.Level, tuning domain.CameraTuning) CameraPlanService`. `Generate`: `s.trackService.Treat(reader, s.defaultLevel, s.defaultLevel)` → `domain.PlanCamera(treated.Route.Points, parameters, s.tuning)`. O `Validate` das flags entra na US3. Rodar `make generate` (depende de T024, T029)
- [X] T031 [US1] Criar `internal/infra/inbound/cli/plan.go`: `NewPlanCommand(cameraPlanService application.CameraPlanService, defaults domain.PlanParameters) *cobra.Command` com `Use: "plan <file>"`, `SilenceErrors`/`SilenceUsage`, `Args` exigindo um argumento (erro de uso via `newUsageError`), abre o arquivo com `os.Open` e passa o `io.Reader` ao serviço (como `inspect`), usa `defaults` (de `Config.DefaultPlanParameters`; `Duration` `nil` ⇒ automática) como parâmetros do plano, sem nenhum valor fixo no código do comando, e imprime `Duration: %.1f s`, `Frame rate: %.1f fps`, `Frames: %d` (formato completo do resumo na US4) (depende de T025, T030)
- [X] T032 [US1] Editar `cmd/sobrevoo/main.go`: criar `cameraPlanService := application.NewCameraPlanService(trackService, nil, cfg.DefaultLevel, cfg.CameraTuning)` (o exporter real entra na US5) e `root.AddCommand(cli.NewPlanCommand(cameraPlanService, cfg.DefaultPlanParameters))` (a `cfg.DefaultLevel` passada ao serviço é o nível de tratamento do trajeto) (depende de T031, T018)

**Checkpoint US1**: `make test` verde; quickstart Cenário 1 (parcial: duração
automática, quantidade de quadros) e Cenário 2 (determinismo, comparando a
saída do resumo em execuções repetidas). MVP entregável: o plano já é
calculado; falta suavidade (US2), parâmetros (US3), resumo completo (US4) e
arquivo (US5).

---

## Phase 4: User Story 2 - Movimento suave e vídeo com abertura e fechamento (Priority: P2)

**Objetivo**: o voo respeita limites de suavidade em curvas fechadas,
retornos e voltas; o vídeo abre mostrando o trajeto inteiro e fecha
mostrando o trajeto completo; trechos com limitação forçada são registrados.

**Teste independente**: gerar planos de trajetos sintéticos (U, ida e
volta, voltas, antimeridiano, latitude 85) e verificar, para todos os pares
de quadros consecutivos, os limites de `research.md` item 5, a ordem das fases e o
enquadramento no primeiro/último quadro.

### Testes da User Story 2

- [X] T033 [P] [US2] Estender `internal/domain/camera_motion_test.go` com `Test_LimitRate` (passe direto e inverso: nenhum par consecutivo excede `limite/fps`; sinal já dentro do limite permanece inalterado; extremos mantêm o valor quando viável), `Test_GaussianSmooth` (não aumenta a inclinação máxima de um sinal Lipschitz; janela truncada em ±3σ; bordas seguradas), `Test_DetectSmoothedSpans` (trecho contínuo em que o clamp alterou o valor além da tolerância — "0,05° para ângulos; 0,001 para `ln D`; 1% para a velocidade relativa do alvo"; início e fim em tempo de vídeo; trechos separados não se fundem; suavização gaussiana comum não conta) (edita o mesmo arquivo de T022; depende da conclusão de T022)
- [X] T034 [P] [US2] Criar `internal/domain/camera_framing_test.go` (pacote `domain_test`): `Test_OverviewPose` (inclinação "60°"; rumo igual ao rumo de acompanhamento recebido (sem rotação); distância `max(margem · R / tan(FOV/2), OverviewMinDistanceFactor · D₀)` com R = raio do menor círculo que contém o trajeto projetado, margem 1,2, FOV vertical 45°, fator 2 e `D₀` do nível; todos os pontos do trajeto ficam dentro do enquadramento; trajeto pequeno (100 m) resulta em distância `2·D₀`, maior que a de acompanhamento), `Test_BlendPose` (smoothstep `3u² − 2u³`: pose exata nas pontas, derivada nula nas pontas, monotônico; rumo pelo menor arco)
- [X] T035 [P] [US2] Criar `internal/domain/camera_plan_assertions_test.go` (helpers de teste do pacote `domain_test`, sem `Test_`): `assertSmooth(t, plan, tuning)` varrendo todos os pares consecutivos — variação de rumo ≤ `MaxHeadingRateDegPerSecond/fps`, de inclinação ≤ `MaxTiltRateDegPerSecond/fps`, de `ln(CameraToMarkerDistance)` ≤ `MaxLogDistanceRatePerSecond/fps` e deslocamento do alvo ≤ `MaxTargetSpeedInDistances × distância / fps`; `assertPhaseOrder(t, plan)` (`Opening* Following+ Closing*`); `assertMarkerMonotonic(t, plan)`
- [X] T036 [US2] Estender `internal/domain/camera_planning_test.go` com `Test_PlanCamera_Smoothness` (um `t.Run` por forma, com duração explícita e confortável de 120 s (a duração mínima só passa a existir na US3), usando `SyntheticRouteBuilder`: reta, curva em U (>150°), ida e volta pelo mesmo caminho, círculo com 5 voltas, cruzamento do antimeridiano, latitude 85, latitude −85; cada um chama `assertSmooth`, `assertPhaseOrder`, `assertMarkerMonotonic`), `Test_PlanCamera_OpeningAndClosing` (abertura `round(N × OpeningFraction)` quadros, fechamento `round(N × ClosingFraction)`, acompanhamento o restante (`N − abertura − fechamento`); marcador em 0 durante toda a abertura e no comprimento total durante todo o fechamento; primeiro e último quadros com inclinação 60° e o trajeto inteiro enquadrado, com o rumo do primeiro igual ao do primeiro quadro de acompanhamento e o do último igual ao do último de acompanhamento (sem rotação na abertura e no fechamento); distância câmera–marcador do primeiro quadro maior que a do meio do acompanhamento), `Test_PlanCamera_RegionEquivalence` (SC-007: a mesma forma sintética gerada na origem `(0, 0)`, em `lon = 179.9` e em `lat = 85` produz planos com o mesmo número de quadros, mesmas propriedades de suavidade e resumos — faixas de altitude e de distância — iguais dentro de "1%"), `Test_PlanCamera_SmoothedSpans` (trajeto que exige limitação forçada gera trecho registrado com `Start ≤ End` e `Quantity` correta; trajeto reto não gera nenhum) (edita o mesmo arquivo de T023; depende da conclusão de T023)

### Implementação da User Story 2

- [X] T037 [US2] Estender `internal/domain/camera_motion.go` com `LimitRate` (passe direto e depois inverso de clamp de inclinação máxima por quadro), `GaussianSmooth` (σ = `GaussianSigmaSeconds`, janela ±3σ, bordas seguradas) e `DetectSmoothedSpans` (`SmoothedSpan` por grandeza limitada, tolerâncias de `research.md` item 5), todas exportadas e documentadas em inglês; aplicar a cada trilha (`ψ`, `θ`, `ln D`, alvo relativo a `D`) na ordem "clamp → gaussiana" (depende de T033, T028)
- [X] T038 [US2] Criar `internal/domain/camera_framing.go`: `OverviewPose` (centro do trajeto projetado, `OverviewTiltDegrees`, rumo recebido por parâmetro — o do primeiro quadro de acompanhamento na abertura, o do último no fechamento —, `D_geral = max(OverviewMargin · R / tan(OverviewVerticalFOVDegrees/2), OverviewMinDistanceFactor · D₀)`) e `BlendPose` (smoothstep sobre alvo, rumo pelo menor arco, inclinação e `ln D`), ambas exportadas e documentadas em inglês (depende de T034, T026)
- [X] T039 [US2] Editar `internal/domain/camera_planning.go`: `PlanCamera` passa a dividir os quadros em abertura (`round(N × OpeningFraction)` quadros), acompanhamento e fechamento (`round(N × ClosingFraction)` quadros), o acompanhamento recebendo o restante — o marcador fica em 0 na abertura e no comprimento total no fechamento, o tempo do trajeto ocupa só o acompanhamento —, interpola as poses de abertura/fechamento com `BlendPose` entre `OverviewPose` e a pose de acompanhamento, passa todas as trilhas por `LimitRate`/`GaussianSmooth` (T037) e entrega os `SmoothedSpan` a `NewCameraPlan`; quantização por último (depende de T036, T037, T038)

**Checkpoint US2**: `make test` verde, incluindo as propriedades de
suavidade; quickstart Cenários 3 e 4 (rumo ≤ 45/fps entre quadros; abertura
e fechamento enquadrando o trajeto inteiro). US1 continua funcionando.

---

## Phase 5: User Story 3 - Ajustar duração, taxa de quadros, distância e inclinação (Priority: P3)

**Objetivo**: o usuário escolhe duração, taxa, distância e inclinação; o
tempo do vídeo segue o tempo real com paradas longas comprimidas; parâmetros
e trajetos inválidos são recusados com mensagem clara; a duração
automática nunca é recusada.

**Teste independente**: gerar planos do mesmo trajeto com valores
diferentes de cada parâmetro e conferir quadros, faixas de distância e
inclinação; usar um trajeto com parada de 10 min; verificar cada recusa e
seu código de saída.

### Testes da User Story 3

- [X] T040 [P] [US3] Estender `internal/domain/camera_timeline_test.go` com `Test_BuildMarkerTimeline_LongStops` (segmento com velocidade média "abaixo de 0,5 m/s" durante mais de 30 s é parada longa, com duração efetiva `min(dt real, 2 s, 5% do tempo em movimento)`; parada de 25 s não é alterada; parada de 10 min ocupa no máximo 5% do vídeo; razão vídeo/real constante fora das paradas, com variação ≤ 5% entre trechos; marcador sem saltos durante a parada; trajeto sem horário não tem compressão) (edita o mesmo arquivo de T021; depende da conclusão de T021)
- [X] T041 [P] [US3] Criar `internal/domain/camera_minimum_duration_test.go` (pacote `domain_test`): `Test_MinimumDuration` (assinatura `MinimumDuration(points, frameRate, distance, tilt, tuning)`; `abertura_mín = max(2 s, 1,5·|θ_geral−θ|/30°/s, 1,5·|ln(D_geral/D₀)|/1,5/s)` sem termo de rumo, `fechamento_mín` idêntico, `acomp_mín = 5 s`, `duração_mín = max(abertura_mín/0,10, fechamento_mín/0,10, acomp_mín/0,80)` arredondada para cima ao próximo quadro; trajeto de poucos quilômetros ⇒ 20 s; 20 km ⇒ ≈ 39 s; 100 km ⇒ ≈ 55 s; abrangência de 2 000 km ⇒ menor que 120 s; cresce com o tamanho do trajeto; independe da duração escolhida), `Test_PlanCamera_Duration` (duração igual à mínima é aceita e gera plano suave; um quadro abaixo → `ErrDurationTooShort` com a duração mínima na mensagem; duração informada nunca passa pela curva automática), `Test_DefaultDuration_AtLeastMinimum` (com um `CameraTuning` **sintético** de limites de taxa bem mais restritivos — por exemplo rumo/zoom 10× menores —, para o qual o mínimo excede 120 s, a duração automática é o mínimo e nunca produz `ErrDurationTooShort`; com o `CameraTuning` inicial o mínimo fica sempre abaixo do teto, o que o teste também confirma; monotônica em comprimento) e `Test_PlanCamera_SmoothnessWithAutomaticDuration` (repete, com duração automática em vez de explícita, as formas sintéticas de `Test_PlanCamera_Smoothness` da US2 e chama `assertSmooth`, `assertPhaseOrder`, `assertMarkerMonotonic`)
- [X] T042 [P] [US3] Criar `internal/domain/camera_track_limits_test.go`: `Test_PlanCamera_TrackTooShort` (comprimento abaixo de "50 m" → `ErrTrackTooShort`, mensagem com o comprimento encontrado e o mínimo; 50 m exatos são aceitos), `Test_PlanCamera_TrackTooLarge` (abrangência > "2 000 km" → `ErrTrackTooLarge`, mensagem com a abrangência encontrada e o máximo aceito; 2 000 km exatos aceitos; um trajeto de 3 000 km percorrido de ida e volta no mesmo lugar, com abrangência pequena, é aceito por este critério)
- [X] T043 [P] [US3] Criar `internal/domain/camera_levels_test.go`: `Test_PlanCamera_DistanceLevels` (para o mesmo trajeto, `CameraToMarkerDistance` de `high` maior que a de `low` em todos os quadros da fase de acompanhamento), `Test_PlanCamera_TiltLevels` (`Tilt` de `high` > `medium` > `low`, respectivamente 65°/45°/25°, constantes na fase de acompanhamento)
- [X] T044 [P] [US3] Estender `internal/application/camera_plan_service_test.go` com cenários de `Generate`: `Validate` falho devolve o erro sem chamar `TrackService`; parâmetros válidos seguem o fluxo; propaga `ErrDurationTooShort`, `ErrTrackTooShort` e `ErrTrackTooLarge` sem alterar (edita o mesmo arquivo de T024; depende da conclusão de T024)
- [X] T045 [P] [US3] Estender `internal/infra/inbound/cli/plan_test.go` com: o comando é construído com `defaults` de teste (`FrameRate: 30`, `Distance`/`Tilt` `medium`); `--duration 45.5` chega ao serviço como `new(45500 * time.Millisecond)`; sem `--duration` chega `nil`; sem `--fps`, `--distance` e `--tilt` chegam os valores de `defaults`; `--fps 29.97`; `--distance`/`--tilt` aceitam `low|medium|high` e traduzem para `domain.Level`; `--duration abc`, `--duration nan`, `--duration inf`, `--fps abc`, `--distance perto` e `--tilt perto` são erro de uso (código 2), e a mensagem de nível desconhecido lista os valores aceitos; `--duration 0` e `--fps 200` chegam ao serviço (a regra é do domínio, códigos 10 e 11 pelo `ExitCode`) (edita o mesmo arquivo de T025/T031; depende da conclusão de T025)

### Implementação da User Story 3

- [X] T046 [US3] Estender `internal/domain/camera_timeline.go` com a compressão de paradas longas conforme `research.md` item 3, passo 2: velocidade média do segmento "abaixo de 0,5 m/s", sequência contínua com duração real "acima de 30 s", `dt` efetivo `min(dt real, 2 s, 5% do tempo em movimento)`, paradas ≤ 30 s inalteradas; o tempo efetivo total é distribuído linearmente sobre o acompanhamento (depende de T040, T027)
- [X] T047 [US3] Estender `internal/domain/camera_planning.go`: `MinimumDuration(points []TrackPoint, frameRate float64, distance, tilt Level, tuning CameraTuning) time.Duration` (fórmula de `research.md` item 8, sem termo de rumo, com as cotas conservadoras independentes da duração escolhida, arredondada para cima ao próximo quadro); `PlanCamera` valida, nesta ordem, o comprimento mínimo e a abrangência máxima (`ErrTrackTooShort`, `ErrTrackTooLarge`; comprimento = distância acumulada, deve ser ≥ `MinTrackLengthMeters`; abrangência = 2 × maior distância do centro da projeção (`LocalPlane`) a um ponto do trajeto tratado, deve ser ≤ `MaxTrackSpanMeters`) e a duração informada ≥ `MinimumDuration` (`ErrDurationTooShort`, mensagem com a duração mínima); `DefaultDuration` passa a devolver `max(curva, MinimumDuration)`; `PlanCamera` resolve a duração (informada ou automática), preenchendo `Parameters.Duration` e `DurationMode`; níveis de distância e inclinação já vêm das tabelas de `CameraTuning` (depende de T041, T042, T043, T039, T046)
- [X] T048 [US3] Editar `internal/application/camera_plan_service.go`: `Generate` chama `parameters.Validate()` antes de tratar o trajeto e devolve o erro (depende de T044, T047)
- [X] T049 [US3] Editar `internal/infra/inbound/cli/plan.go`: acrescentar as flags `--duration` (sem padrão fixo; `float64` em segundos, convertido com `time.Duration(seconds * float64(time.Second))`; ausente ⇒ `nil`), `--fps` (padrão `defaults.FrameRate`, isto é, 30), `--distance` e `--tilt` (padrão `levelName(defaults.Distance)` e `levelName(defaults.Tilt)`, texto `low|medium|high`, reaproveitando `parseLevel`/`levelName` de `inspect.go`); valor não numérico, NaN, infinito ou nível desconhecido vira `newUsageError` com mensagem `--duration: ...` etc., listando os valores aceitos; textos de ajuda conforme `contracts/cli.md` (depende de T045, T048)

**Checkpoint US3**: `make test` verde; quickstart Cenários 5, 5b, 6 e 7
(parte de parâmetros: códigos 10 a 14 e 2) e o critério "duração exatamente
igual à mínima é aceita".

---

## Phase 6: User Story 4 - Ver o resumo do plano (Priority: P4)

**Objetivo**: o resumo mostra duração (automática ou informada), quadros,
faixas de altitude e distância, referência de tempo e trechos suavizados.

**Teste independente**: gerar o plano de um trajeto conhecido e conferir que
cada valor do resumo bate com o recalculado a partir dos quadros.

### Testes da User Story 4

- [X] T050 [P] [US4] Criar `internal/domain/camera_plan_test.go` com `Test_NewCameraPlan` (resumo recalculável dos quadros: `MinCameraAltitude`/`MaxCameraAltitude`, `MinCameraDistance`/`MaxCameraDistance`, `FrameCount == len(Frames)`, `Duration`, `FrameRate`, `DurationMode`, `TimeReference`; `SmoothedSpans` vazio quando não há trecho; trechos copiados na ordem recebida) usando `CameraFrameBuilder`
- [X] T051 [P] [US4] Estender `internal/infra/inbound/cli/plan_test.go` com `Test_formatPlanSummary`/cenários do comando: saída exatamente no formato de `contracts/cli.md` (`Duration: 42.0 s (automatic)` ou `(requested)`, `Frame rate`, `Frames`, `Camera altitude: 127.3 m - 912.8 m`, `Camera distance: 300.0 m - 1204.5 m`, `Time reference: clock`, `Time reference: distance (time data is inconsistent)` quando há motivo, `Smoothed spans: 2` seguido de uma linha `  0.00 s - 1.20 s (heading)` por trecho, `Smoothed spans: none` sem lista) (edita o mesmo arquivo de T045; depende da conclusão de T045)

### Implementação da User Story 4

- [X] T052 [US4] Estender `NewCameraPlan` em `internal/domain/camera_plan.go` para calcular no `Summary` as faixas mínima/máxima de altitude da câmera e de distância câmera→marcador a partir dos quadros (plano sem quadros ⇒ zeros) (depende de T050)
- [X] T053 [US4] Editar `internal/infra/inbound/cli/plan.go`: substituir a saída básica pelo `formatPlanSummary(plan domain.CameraPlan) string` completo de `contracts/cli.md` (rótulos e ordem estáveis; motivo do recuo após `Time reference: distance`) (depende de T051, T052, T049)

**Checkpoint US4**: quickstart Cenário 1 completo e Cenário 6 (`Time
reference: distance`); SC-009 verificável lendo apenas o resumo.

---

## Phase 7: User Story 5 - Exportar o plano completo (Priority: P5)

**Objetivo**: `--export <caminho>` grava o plano completo em JSON
determinístico e atômico; sem `--overwrite`, recusa destino existente.

**Teste independente**: exportar, reler o arquivo e conferir que todos os
quadros e campos coincidem com o plano; exportar duas vezes e comparar os
bytes; verificar as recusas.

### Testes da User Story 5

- [X] T054 [P] [US5] Criar `internal/infra/outbound/jsonfile/camera_plan_exporter_test.go` em diretório temporário (`t.TempDir()`): `format_version` `1` e ordem de campos de `contracts/plan-file.md` (`parameters`, `summary`, `frames`); cada quadro numa única linha compacta; casas decimais fixas (lat/lon 7; alturas, distâncias e ângulos 3); `duration_mode` `automatic`/`explicit`; `time_fallback_reason` e `smoothed_spans` sempre presentes; 100 exportações do mesmo plano ⇒ 100 arquivos idênticos byte a byte (SC-003); releitura via `encoding/json` bate com o plano; destino existente sem `overwrite` → `ErrPlanDestinationExists` e arquivo original intacto (mesmo conteúdo); com `overwrite` substitui integralmente; diretório inexistente e sem permissão → `ErrPlanDestinationInvalid`; nenhum arquivo temporário ou parcial resta após qualquer falha
- [X] T055 [P] [US5] Estender `internal/application/camera_plan_service_test.go` com `Test_cameraPlanService_Export`: delega à porta com `plan`, `path` e `overwrite` exatos; propaga `ErrPlanDestinationExists` e `ErrPlanDestinationInvalid` sem alterar; usa `mockdomain.NewMockCameraPlanExporter` (edita o mesmo arquivo de T024/T044; depende da conclusão de T044)
- [X] T056 [P] [US5] Estender `internal/infra/inbound/cli/plan_test.go` com: `--export path` chama `Export(plan, path, false)` e imprime `Plan written to <path>` como última linha; `--export path --overwrite` chama com `overwrite = true`; `--overwrite` sem `--export` é erro de uso (código 2) e não chama o serviço; erros de exportação são devolvidos sem alteração (edita o mesmo arquivo de T051; depende da conclusão de T051)

### Implementação da User Story 5

- [X] T057 [US5] Criar `internal/infra/outbound/jsonfile/camera_plan_file.go` com as structs de DTO do arquivo (`cameraPlanFile`, `parametersFile`, `summaryFile`, `frameFile`, etc., sempre structs e nunca `map`, para ordem estável) e a função de mapeamento de `domain.CameraPlan` para o DTO, conforme `contracts/plan-file.md` (nomes JSON `duration_s`, `frame_rate`, `duration_mode`, `camera_altitude_m`, `time_reference`, `time_fallback_reason`, `smoothed_spans[].start_s|end_s|quantity`, `frames[].index|time_s|phase|camera.lat|lon|altitude_m|heading_deg|tilt_deg|marker.lat|lon|distance_m|camera_to_marker_m`), com formatação numérica de casas fixas sem zeros à direita desnecessários (depende de T013)
- [X] T058 [US5] Criar `internal/infra/outbound/jsonfile/camera_plan_exporter.go` com `CameraPlanExporter` (receiver `e`) e `NewCameraPlanExporter() CameraPlanExporter` (nome exportado sem repetir o pacote): serializa top-level indentado com 2 espaços e cada quadro compacto numa linha; grava num arquivo temporário no mesmo diretório do destino; sem `overwrite` publica com `os.Link` (falha atômica se existir → `domain.ErrPlanDestinationExists`; recorre a `O_CREATE|O_EXCL` se o sistema de arquivos não suportar hard link; esse recurso fica isolado numa função pequena e comentada, coberta só por revisão de código, `research.md` item 11); com `overwrite` publica com `os.Rename`; remove sempre o temporário; traduz erros do `os` (`fs.ErrExist`, diretório inexistente, permissão) para `ErrPlanDestinationExists`/`ErrPlanDestinationInvalid` (`research.md` item 11) (depende de T054, T057)
- [X] T059 [US5] Editar `internal/application/camera_plan_service.go`: acrescentar `Export(plan domain.CameraPlan, path string, overwrite bool) error` à interface `CameraPlanService` e implementá-lo delegando a `s.exporter.Export(plan, path, overwrite)`; rodar `make generate` para regenerar `mockapplication/camera_plan_service.go` (depende de T055, T048)
- [X] T060 [US5] Editar `internal/infra/inbound/cli/plan.go`: flags `--export` (texto) e `--overwrite` (bool); `--overwrite` sem `--export` → `newUsageError`; com `--export`, chama `Export` depois de gerar o plano e imprime `Plan written to <path>` como última linha (depende de T056, T059, T053)
- [X] T061 [US5] Editar `cmd/sobrevoo/main.go`: `cameraPlanExporter := jsonfile.NewCameraPlanExporter()` e passá-lo a `application.NewCameraPlanService(...)` no lugar do `nil` de T032 (depende de T058, T060)

**Checkpoint US5**: `make test` verde; quickstart Cenários 1 (export), 2
(soma de verificação idêntica) e 7 (códigos 15 e 16, arquivo original intacto).

---

## Phase 8: Polish & Cross-Cutting Concerns

**Propósito**: documentação, amostras do quickstart e verificação final.

- [X] T062 [P] Criar os GPX sintéticos do `quickstart.md` em `specs/003-camera-path-planning/amostras/` — `pedalada.gpx` (~20 km, com horários), `sem-horario.gpx`, `com-parada.gpx` (parada de ~10 min), `retorno.gpx`, `voltas.gpx`, `antimeridiano.gpx`, `polar.gpx` (latitude > 80°), `curto.gpx` (poucos metros), `enorme.gpx` (> 2 000 km), `pequena.gpx` (~1 km), `longa.gpx` (~150 km), `muito-longa.gpx` (~1 500 km), `dez-mil-pontos.gpx` — gerados por um programa descartável no diretório de scratchpad da sessão (não versionar o gerador); versionar só os `.gpx`
- [X] T063 [P] Atualizar `CLAUDE.md`: descrição do projeto (três etapas, comando `plan`), a lista de entidades e portas do domínio (`CameraPlan`, `CameraFrame`, `PlanParameters`, `CameraTuning`, `CameraPlanExporter`, `CleanedTrack`, `TreatedTrack`), a seção de `internal/application` (`TrackService` no lugar de `InspectTrackService`, `CameraPlanService`; serviço dependendo de serviço), a de adapters de saída (`jsonfile.NewCameraPlanExporter`), a de testes (a CLI mocka `application.TrackService`, `application.GeoDataService` e `application.CameraPlanService`), as referências a `specs/003-camera-path-planning/contracts/` e o exemplo de `go run ./cmd/sobrevoo plan ...`; remover toda menção a `InspectTrackService` (prosa em português; identificadores e comandos em inglês)
- [X] T064 [P] Atualizar `README.md`: seção do comando `plan` com flags, duração automática, exemplo de saída do resumo e exemplo de `--export`, no mesmo estilo das seções de `inspect` e `geodata`
- [X] T065 Revisão final de terminologia e nomes nos artefatos: conferir com `grep` que `spec.md`, `plan.md`, `research.md`, `data-model.md`, `contracts/` e `quickstart.md` usam "comprimento" (distância percorrida) e "abrangência" (2 × maior distância do centro a um ponto) sem a palavra ambígua "extensão" nesses sentidos, e os nomes exportados de `research.md` item 17 (`NewLocalPlane`, `BuildMarkerTimeline`, ...) em vez das versões não exportadas; corrigir qualquer resto e reconciliar com o que foi de fato implementado (depende de T047)
- [X] T066 Rodar todos os cenários de `quickstart.md` com o binário real (`go build -o bin/sobrevoo ./cmd/sobrevoo`) sobre as amostras de T062 e registrar qualquer divergência; corrigir o código, não o guia, salvo erro do guia (depende de T061, T062)
- [X] T067 Verificar SC-001 (menos de 5 s para 10 000 pontos com parâmetros padrão) com `time ./bin/sobrevoo plan specs/003-camera-path-planning/amostras/dez-mil-pontos.gpx` e, se exceder, otimizar a janela gaussiana (ex.: kernel pré-calculado) sem alterar resultados (depende de T066)
- [X] T068 Passada final: `gofmt -l .` vazio, `make lint`, `make generate` sem diff, `make test` verde com cobertura de `internal/domain` e `internal/application` não menor que a da branch `main` no início (T001); revisar que não sobrou nenhuma referência a `InspectTrackService` fora de `specs/001-*` e `specs/002-*` (documentos históricos) (depende de T063, T064, T067)

---

## Dependências e Ordem de Execução

### Dependências entre fases

- **Phase 1 (Setup)**: sem dependências.
- **Phase 2 Parte A (refatoração `TrackService`)**: depende de T001; bloqueia
  US1 (o serviço de câmera consome `TrackService`). É concluída e verde
  (Checkpoint A) antes de qualquer arquivo `camera_*.go`.
- **Phase 2 Parte B (base de câmera)**: pode começar em paralelo com a
  Parte A nos arquivos que não se tocam (T010, T011, T017, T018 não
  dependem de `TrackService`; T013 só depois de T011), mas **T009 deve estar concluída antes de
  T030** (primeiro uso de `TrackService` pelo novo serviço). Bloqueia todas
  as histórias.
- **US1 (P1)**: depende da Phase 2 completa. É o MVP.
- **US2 (P2)**: depende de US1 (estende `camera_motion.go`,
  `camera_planning.go`).
- **US3 (P3)**: depende de US2 (a duração mínima depende de abertura e
  fechamento) e de US1 (estende serviço e CLI).
- **US4 (P4)**: depende de US1 (comando e `NewCameraPlan`); independente de
  US2/US3 para o cálculo do resumo, mas o teste de trechos suavizados só faz
  sentido após US2. Recomendado após US3, pois T053 estende `plan.go` depois
  de T049.
- **US5 (P5)**: depende de US1 (comando/serviço) e de T013/T014 (porta e
  mock); recomendado após US4 pelo mesmo motivo de `plan.go`.
- **Polish**: depende das histórias concluídas.

### Dentro de cada história

- Testes antes da implementação correspondente.
- Domínio (funções puras) → serviço → CLI → `main.go`.
- Tarefas que editam o mesmo arquivo (`camera_planning.go`,
  `camera_motion.go`, `camera_timeline.go`, `camera_plan_service.go`,
  `plan.go`, `plan_test.go`, `camera_planning_test.go`) são sequenciais entre
  histórias e nunca `[P]` entre si.

### Oportunidades de paralelismo

- Phase 2: T002 ‖ T010 ‖ T011 ‖ T017; depois T013 ‖ T015 ‖ T018 (após T011) e T019 (após T010); depois T014 ‖ T016.
  T004 ‖ T006 ‖ T007 após T003/T005.
- US1: T020 ‖ T021 ‖ T022 ‖ T023 ‖ T024 ‖ T025 (arquivos de teste
  diferentes); depois T026 → T027 → T028 sequenciais (cada um usa o
  anterior), com T030 ‖ T031 possíveis após T029.
- US2: T033 ‖ T034 ‖ T035; US3: T040 ‖ T041 ‖ T042 ‖ T043 ‖ T044 ‖ T045;
  US5: T054 ‖ T055 ‖ T056.
- Polish: T062 ‖ T063 ‖ T064.

---

## Exemplo de Paralelização: User Story 1

```bash
# Escrever os testes de US1 em arquivos diferentes, juntos:
Task: "Criar camera_projection_test.go (projeção, antimeridiano, latitudes altas)"
Task: "Criar camera_timeline_test.go (referência de tempo, s(t) monotônica)"
Task: "Criar camera_motion_test.go (rumo, unwrap, distância, pose)"
Task: "Criar camera_planning_test.go (PlanCamera, DefaultDuration)"
Task: "Criar camera_plan_service_test.go (Generate)"
Task: "Criar plan_test.go (comando plan)"

# Depois, o núcleo em ordem, e serviço/CLI em paralelo:
Task: "camera_projection.go" → "camera_timeline.go" → "camera_motion.go" → "camera_planning.go"
Task: "camera_plan_service.go"  ‖  "plan.go"
```

---

## Estratégia de Implementação

### MVP Primeiro (Phase 1 + 2 + User Story 1)

1. Phase 1 e Phase 2 (a refatoração `TrackService` primeiro, com o
   Checkpoint A verde).
2. User Story 1: `sobrevoo plan <arquivo>` gera o plano com duração
   automática e resumo básico.
3. **PARAR E VALIDAR**: Cenários 1 (parcial) e 2 do `quickstart.md`.
   O plano ainda não é agradável de assistir (sem suavização, abertura e
   fechamento), mas a espinha (projeção, tempo, câmera, determinismo,
   serviço, CLI) está de pé e testada.

### Entrega Incremental

1. Setup + Foundational → etapas 1 e 2 refatoradas e verdes; base de câmera
   pronta.
2. + US1 → plano determinístico (MVP).
3. + US2 → voo suave, com abertura e fechamento → Cenários 3 e 4.
4. + US3 → parâmetros, paradas comprimidas, recusas → Cenários 5, 5b, 6, 7.
5. + US4 → resumo completo → Cenário 1 completo.
6. + US5 → exportação → Cenários 2 (export) e 7 (15 e 16).
7. Polish → documentação, amostras, desempenho e verificação final.

Cada história agrega valor sem quebrar as anteriores.

---

## Notas

- `[P]` = arquivos diferentes, sem dependência de tarefa incompleta.
- Os valores numéricos (`CameraTuning`) são iniciais e ficam concentrados em
  `config.Load()`; ajustá-los é uma edição num só lugar (`research.md` item 12).
- Constantes de precisão (quantização) e limites de suavidade vêm de
  `research.md` itens 5 e 9; os testes verificam *limites*, não estética.
- Faça commit após cada tarefa ou grupo lógico. A refatoração
  (T002–T009) merece um commit próprio, separado do código de câmera.
- Pare em qualquer checkpoint para validar a história com o `quickstart.md`.
- Evite: tarefas vagas, duas tarefas `[P]` editando o mesmo arquivo,
  dependências que quebrem a independência de teste de uma história.

---

## Notas de implementação (desvios do plano original, registrados ao executar)

- **Duração informada com teto** (análise N1): FR-016 e SC-008 da spec ganharam
  o teto de 3 600 s para `--duration`; `PlanParameters.Validate` o aplica e a
  CLI rejeita NaN e infinito como erro de uso. Tarefas T011, T012, T045 e T049
  já foram reescritas com isso.
- **Ritmo do marcador pelos pontos limpos**: ao executar o quickstart com uma
  parada real de 10 min, a simplificação da etapa 1 escondia a parada.
  `TreatedTrack` passou a carregar `CleanedPoints`; `PlanCamera` recebe um
  `TreatedTrack` (não mais só os pontos da rota) e constrói a linha do tempo
  sobre os pontos limpos, com o jitter de GPS das paradas longas descontado.
  `BuildMarkerTimeline(cleanedPoints, tuning)` não recebe mais o plano local.
  Detalhes em `research.md` item 3.
- **Exportação antes do resumo**: `plan --export` exporta primeiro e só então
  imprime o resumo e `Plan written to ...`, para que um erro de exportação não
  venha acompanhado de um resumo (contrato: erro = sem plano).
- **`CameraTuningBuilder`** (`builddomain/camera_tuning_builder.go`), não listado
  nas tarefas: dá aos testes de domínio os valores iniciais de `CameraTuning`
  sem importar a configuração; `config_test.go` garante que os dois coincidem.
- **Visão geral com o rumo do acompanhamento e piso `2·D_a`** (`research.md`
  item 7): `OverviewView` recebe o rumo e a distância mínima por parâmetro.
- **Testes do fallback `O_EXCL`** em `camera_plan_exporter_fallback_test.go`
  (pacote interno), forçando a falha do hard link com um arquivo temporário
  inexistente — o que `research.md` item 11 dava como não testável.
