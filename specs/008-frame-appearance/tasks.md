---

description: "Task list for feature implementation"
---

# Tarefas: Aparência Ajustável dos Quadros

**Entrada**: Documentos de design de `/specs/008-frame-appearance/`

**Pré-requisitos**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/appearance-flags.md`, `quickstart.md`

**Testes**: incluídos. Como nas etapas anteriores, a constituição do projeto
(Princípio VI — Testes Automatizados no Núcleo; Princípio X —
Given/When/Then, builders e isolamento por camada) exige testify e uber-go/mock
e proíbe testes tabulares: cada cenário é um `t.Run("should ...")` com
`// given`, `// when`, `// then`. As tarefas de teste ficam antes da
implementação correspondente e devem falhar primeiro — com uma ressalva: as
tarefas da Fase 2, Parte B/C mudam a **assinatura** de funções já existentes
(`NewScene`, `newImagery`, `NewFrameImage`, `NewFrameMark`, `NewFrameSetID`),
não só acrescentam método novo; nesses casos, "estender o teste" inclui
atualizar as chamadas já existentes no arquivo de teste para a assinatura
nova (sem isso o pacote nem compila), e o cenário novo é o que fica
vermelho até a tarefa de implementação seguinte.

**Organização**: as tarefas são agrupadas por história de usuário (P1–P4 de
`spec.md`) para permitir implementação e teste independentes de cada
história. A **Phase 2** tem quatro partes: (A) o tipo de domínio `Appearance`,
puro e sem E/S; (B) fazer `Scene`/`overlay`/`imagery`/`FrameImage` desenharem
com uma `Appearance` recebida, em vez dos `var` de pacote fixos de hoje; (C) a
aparência entrando na identidade do conjunto de quadros (`NewFrameMark`/
`NewFrameSetID`); (D) os cinco valores de hoje migrando para
`config.RenderDefaults`. Nenhuma dessas quatro partes é visível para o
usuário — nenhuma flag nova existe até a Phase 3.

**Nenhum mock precisa ser regenerado nesta etapa**: `FrameService`,
`FlightService` e as demais interfaces mockadas não ganham nenhum método novo
nem mudam a assinatura de um método existente (`domain.SingleFrameRequest`,
`domain.FrameSetRequest` e `domain.FlightRequest` continuam sendo os mesmos
tipos, por nome — só ganham um campo). `make generate` não deveria produzir
nenhuma diferença; a verificação final (T050) confirma isso.

**Divisão do trabalho entre histórias** (o mesmo arquivo é estendido por mais
de uma história, nunca em paralelo):

| História | O que entrega | Veículo |
|---|---|---|
| US1 | **MVP**: escolher cor/espessura do traçado, cor/raio do marcador, cor do fundo, e ver o resultado — a capacidade central | `render frame` |
| US2 | o quadro isolado é garantidamente igual ao do voo inteiro (byte a byte) | `render all` ganha as mesmas 5 flags |
| US3 | os mesmos nomes e valores funcionam em qualquer comando, inclusive o único | `fly` ganha as mesmas 5 flags; códigos de saída `52`–`54` |
| US4 | quadros de aparências diferentes nunca são tomados pelo mesmo conjunto | nenhuma lógica nova (já sai da Phase 2C) — só a prova, em teste e manualmente |

Enquanto a US1 não existe, nenhuma flag de aparência existe em lugar nenhum;
enquanto a US2 não existe, `render all` desenha sempre com a aparência padrão;
enquanto a US3 não existe, `fly` idem, e os erros de aparência ainda não têm
código de saída próprio (caem no genérico `4`).

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
`internal/infra/outbound/`, `internal/infra/inbound/cli/`. Builders em
`internal/domain/builddomain`. Regras de estilo: sem `ports.go`; `Appearance`
em arquivo próprio, sem entidade dona, como `AspectRatio` em
`camera_aspect.go`; receivers curtos e consistentes com o tipo (`a`
`Appearance`); `new(x)` do Go 1.26 em vez de um helper `ptr`; nome exportado
nunca repete o pacote; receiver sem uso fica sem nome.

---

## Phase 1: Setup (Shared Infrastructure)

**Propósito**: um ponto de partida próprio (branch) e verde.

- [X] T001 Criar e mudar para o branch `008-frame-appearance` a partir de `main` (`git checkout -b 008-frame-appearance`)
- [X] T002 Confirmar `make build`, `make test`, `make lint` e `make generate` verdes antes de qualquer mudança (linha de base)

**Checkpoint**: repositório pronto para a Phase 2.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Propósito**: infraestrutura que toda história depende. **Nenhuma história começa antes desta fase estar verde.**

### Parte A — o tipo de domínio `Appearance` (research.md itens 1–2)

- [X] T003 [P] Escrever `internal/domain/frame_appearance_test.go`: `ParseColor` aceita `"#FFB000"` e `"#ffb000"` (maiúsculo/minúsculo, mesmo `RGB`); recusa (`ErrInvalidColor`, citando o texto recebido) um texto sem `#`, com menos ou mais de 6 dígitos depois do `#`, com um dígito fora de `0-9a-fA-F`, uma abreviação de 3 dígitos e um valor com canal alfa — um `t.Run` por caso; `NewAppearance` aceita `TrailWidthRatio`/`MarkerRadiusRatio` nos limites inclusivos (`0.0005`/`0.05`/`0.001`/`0.1`); recusa um `TrailWidthRatio` abaixo de `0.0005` ou acima de `0.05` com `ErrInvalidTrailWidth` citando o valor e o intervalo; recusa um `MarkerRadiusRatio` abaixo de `0.001` ou acima de `0.1` com `ErrInvalidMarkerRadius`; `Appearance.Fingerprint()` devolve o mesmo texto para os mesmos cinco valores chamado duas vezes, e um texto diferente quando qualquer um dos cinco campos muda (um `t.Run` por campo)
- [X] T004 Implementar `internal/domain/frame_appearance.go`: `Appearance` (os cinco campos de `data-model.md`), `MinTrailWidthRatio`/`MaxTrailWidthRatio`/`MinMarkerRadiusRatio`/`MaxMarkerRadiusRatio`, `NewAppearance`, `ParseColor`, `Fingerprint()`; `make test` verde para o arquivo novo
- [X] T005 [P] Adicionar `ErrInvalidColor`, `ErrInvalidTrailWidth`, `ErrInvalidMarkerRadius` a `internal/domain/errors.go`
- [X] T006 [P] Adicionar `builddomain.NewAppearanceBuilder()` em `internal/domain/builddomain/appearance_builder.go`: padrão `TrailColor #FFB000`, `TrailWidthRatio 0.005`, `MarkerColor #E5252A`, `MarkerRadiusRatio 0.012`, `BackgroundColor #20262E` (os valores de hoje); `WithTrailColor(domain.RGB)`, `WithTrailWidthRatio(float64)`, `WithMarkerColor(domain.RGB)`, `WithMarkerRadiusRatio(float64)`, `WithBackgroundColor(domain.RGB)`, `Build() domain.Appearance`

### Parte B — `Scene`/`overlay`/`imagery`/`FrameImage` passam a desenhar com a `Appearance` recebida (research.md item 1; data-model.md)

- [X] T007 [P] Escrever `internal/domain/frame_image_test.go` (novo arquivo): `NewFrameImage(resolution, background)` preenche todos os pixels com `background`, para duas cores de fundo diferentes (dois `t.Run`)
- [X] T008 Mudar `NewFrameImage` para `NewFrameImage(resolution Resolution, background RGB) FrameImage` em `internal/domain/frame_image.go`, preenchendo com `background` no lugar do antigo `BackgroundColor` de pacote
- [X] T009 [P] Estender `internal/domain/frame_imagery_test.go`: atualizar as chamadas existentes de `newImagery`/`newTileTexture` para a assinatura com `background RGB`; cenário novo — um pixel parcialmente transparente misturado com dois fundos diferentes produz dois resultados diferentes (mesma conta de mistura de hoje, só com o parâmetro em vez do `var` de pacote)
- [X] T010 Mudar `newImagery` para `newImagery(tileSets []TileSet, decoder TileDecoder, cacheBytes int64, background RGB) *imagery` (campo `background RGB` na struct `imagery`) e `newTileTexture` para receber `background RGB` como parâmetro, em `internal/domain/frame_imagery.go`; `make test` verde
- [X] T011 [P] Estender `internal/domain/frame_overlay_test.go`: atualizar as construções existentes de `overlay{...}` para incluir `appearance`; cenário novo — duas `overlay` com `Appearance` diferente desenham um traçado/marcador de cor e espessura/raio diferentes para os mesmos pontos (dois `t.Run`, um por elemento)
- [X] T012 Adicionar o campo `appearance Appearance` a `overlay` e usar `o.appearance.TrailColor`/`TrailWidthRatio` em `drawTrail` e `o.appearance.MarkerColor`/`MarkerRadiusRatio` em `drawMarker`, em `internal/domain/frame_overlay.go` — `TrailCasingColor`, `MarkerRingColor`, `MarkerRingRatio`, `MarkerRingMin`, `TrailMinWidth`, `MarkerMinRadius` continuam lidos como hoje, sem mudança; `make test` verde
- [X] T013 [P] Estender `internal/domain/frame_scene_test.go` e `internal/domain/frame_scene_bench_test.go`: atualizar as chamadas existentes de `NewScene` para a assinatura com `appearance`; cenário novo — `Render` com duas `Appearance` diferentes (mesmo plano, recorte, quadro) produz duas imagens diferentes; a mesma `Appearance`, chamada duas vezes, produz a mesma imagem byte a byte (reforça SC-003)
- [X] T014 Mudar `NewScene` para `NewScene(slice GeoSlice, decoder TileDecoder, tuning RenderTuning, appearance Appearance) (*Scene, error)` (campo `appearance Appearance` em `Scene`); `Render` usa `s.appearance.BackgroundColor` em `NewFrameImage` e monta `overlay{..., appearance: s.appearance}`, em `internal/domain/frame_scene.go`; `make test` verde
- [X] T015 Remover `BackgroundColor`, `TrailColor`, `MarkerColor`, `TrailWidthRatio`, `MarkerRadiusRatio` de `internal/domain/render_tuning.go` (agora sem nenhum uso) e atualizar o comentário do bloco de `var`/`const` para dizer que só o que resta ali (`NoMapColors`, `NoElevationColors`, `PatternPeriod`, `TrailCasingColor`, `MarkerRingColor`, `TrailMinWidth`, `MarkerMinRadius`, `MarkerRingRatio`, `MarkerRingMin`) é fixo — o que virou ajustável mora em `Appearance` (`frame_appearance.go`); `make build` verde

### Parte C — a aparência na identidade do conjunto de quadros (research.md item 5; data-model.md)

- [X] T016 [P] Estender `internal/domain/frame_set_test.go`: atualizar as chamadas existentes de `NewFrameMark`/`NewFrameSetID` para a assinatura com `appearance`; cenário novo — duas `Appearance` diferentes (mesmo plano, recorte, resolução, tuning) produzem `FrameSetID` diferente; a mesma `Appearance`, duas vezes, produz o mesmo `FrameSetID`
- [X] T017 Adicionar `Appearance Appearance` a `SingleFrameRequest` e a `FrameSetRequest`; mudar `NewFrameMark`/`NewFrameSetID`/`newFrameSetID` para receber `appearance Appearance` e escrever `appearance.Fingerprint()` no hash, depois de `tuning.Fingerprint()`, em `internal/domain/frame_set.go`; `make test` verde

### Parte D — os cinco valores de hoje migram para configuração (research.md item 6)

- [X] T018 [P] Estender `internal/infra/outbound/config/config_test.go` (`Test_Load`): `RenderDefaults` inclui `TrailColor "#FFB000"`, `TrailWidthRatio 0.005`, `MarkerColor "#E5252A"`, `MarkerRadiusRatio 0.012`, `BackgroundColor "#20262E"`
- [X] T019 Adicionar os cinco campos a `RenderDefaults` e preenchê-los em `Load()`, em `internal/infra/outbound/config/config.go`
- [X] T020 [P] Escrever `Test_domainAppearance` em `cmd/sobrevoo/config_mapping_test.go`: `domainAppearance(config.RenderDefaults{...})` devolve o `domain.Appearance` esperado para os cinco valores de hoje; um `TrailColor`/`MarkerColor`/`BackgroundColor` malformado propaga `ErrInvalidColor`
- [X] T021 Implementar `domainAppearance(d config.RenderDefaults) (domain.Appearance, error)` em `cmd/sobrevoo/config_mapping.go`, chamando `domain.ParseColor` três vezes e `domain.NewAppearance` uma vez; `make test` verde

**Checkpoint**: `make build`, `make test`, `make lint`, `make generate` verdes; `Appearance` percorre domínio, aplicação (via os campos novos) e configuração — mas nenhuma flag existe ainda em nenhum comando.

---

## Phase 3: User Story 1 — Escolher a cor e o tamanho do que é desenhado (Priority: P1) 🎯 MVP

**Objetivo**: `sobrevoo render frame` aceita `--trail-color`, `--trail-width`,
`--marker-color`, `--marker-radius` e `--background-color`; sem nenhuma
delas, o resultado é pixel a pixel igual ao de hoje.

**Teste Independente**: `quickstart.md` itens 1, 2 e 5.

- [X] T022 [P] [US1] Estender `internal/application/frame_service_test.go`: `DrawFrame` repassa `request.Appearance` para `domain.NewScene`/`domain.NewFrameMark` sem alteração — duas chamadas de `DrawFrame` com `Appearance` diferente (via `builddomain.NewAppearanceBuilder()...Build()`) produzem uma `FrameImage`/`FrameMark` diferentes no `FrameExporter.Export` mockado
- [X] T023 [US1] Em `internal/application/frame_service.go`, `DrawFrame` e `DrawFrames` passam `request.Appearance` para `domain.NewScene(...)` e `domain.NewFrameMark(...)`; `make test` verde
- [X] T024 [P] [US1] Escrever `internal/infra/inbound/cli/appearance_test.go`: `parseAppearance` devolve os padrões informados quando nenhuma flag foi mudada (`cmd.Flags().Changed` falso); um valor informado em cada uma das cinco flags chega em `Appearance`; `--trail-color`/`--marker-color`/`--background-color` malformado devolve `ErrInvalidColor` diretamente (sem `usageError`); `--trail-width`/`--marker-radius` com texto que não é número devolve `usageError`; com um número fora do intervalo devolve `ErrInvalidTrailWidth`/`ErrInvalidMarkerRadius`
- [X] T025 [US1] Implementar `parseAppearance(cmd *cobra.Command, trailColorFlag, trailWidthFlag, markerColorFlag, markerRadiusFlag, backgroundColorFlag string, defaults domain.Appearance) (domain.Appearance, error)` e as strings de uso das cinco flags em `internal/infra/inbound/cli/appearance.go`, reaproveitando `parseFiniteNumber` (já em `plan.go`) para `--trail-width`/`--marker-radius`
- [X] T026 [P] [US1] Estender `internal/infra/inbound/cli/render_frame_test.go`: as cinco flags novas chegam em `domain.SingleFrameRequest.Appearance` passado a `frameService.DrawFrame`; omitidas, usam o padrão passado ao comando; uma flag inválida recusa antes de `frameService.DrawFrame` ser chamado
- [X] T027 [US1] Adicionar as cinco flags e o parâmetro `defaultAppearance domain.Appearance` a `NewRenderFrameCommand`, chamando `parseAppearance` antes de `runRenderFrame` e preenchendo `SingleFrameRequest.Appearance`, em `internal/infra/inbound/cli/render_frame.go`
- [X] T028 [US1] Em `cmd/sobrevoo/main.go`: `defaultAppearance, err := domainAppearance(cfg.RenderDefaults)` (mesmo tratamento de erro que `defaultResolution` já tem) e passar `defaultAppearance` para `cli.NewRenderFrameCommand(...)`
- [X] T029 [US1] Validação manual: `quickstart.md` itens 1, 2 e 5 (padrão preservado, mudança de cor/tamanho, valor inválido recusado), com `render frame`

**Checkpoint**: `render frame --trail-color ... --trail-width ... --marker-color ... --marker-radius ... --background-color ...` funciona de ponta a ponta; sem flags, o resultado é o de sempre — MVP entregue.

---

## Phase 4: User Story 2 — Ver o efeito num quadro isolado antes do voo inteiro (Priority: P2)

**Objetivo**: `sobrevoo render all` aceita as mesmas cinco flags, com o mesmo
efeito; um quadro desenhado por `render frame` é byte a byte igual ao mesmo
quadro dentro de `render all`, para a mesma aparência.

**Teste Independente**: `quickstart.md` item 3.

- [X] T030 [P] [US2] Estender `internal/infra/inbound/cli/render_all_test.go`: as cinco flags chegam em `domain.FrameSetRequest.Appearance` passado a `frameService.DrawFrames`; omitidas, usam o padrão; flag inválida recusa antes de `DrawFrames` ser chamado
- [X] T031 [US2] Adicionar as cinco flags e o parâmetro `defaultAppearance domain.Appearance` a `NewRenderAllCommand`, chamando `parseAppearance` e preenchendo `FrameSetRequest.Appearance`, em `internal/infra/inbound/cli/render_all.go`
- [X] T032 [US2] Em `cmd/sobrevoo/main.go`: passar o `defaultAppearance` já calculado (T028) para `cli.NewRenderAllCommand(...)`
- [X] T033 [US2] Validação manual: `quickstart.md` item 3 (quadro isolado idêntico byte a byte ao mesmo quadro de `render all`)

**Checkpoint**: `render frame` e `render all`, com a mesma aparência, produzem o mesmo quadro byte a byte.

---

## Phase 5: User Story 3 — Os mesmos ajustes, com os mesmos nomes, em qualquer comando (Priority: P3)

**Objetivo**: `sobrevoo fly` aceita as mesmas cinco flags, com o mesmo nome e
o mesmo efeito que `render frame`/`render all`; os três erros de aparência têm
código de saída próprio (`52`–`54`) em qualquer um dos três comandos.

**Teste Independente**: `quickstart.md` item 4.

- [X] T034 [P] [US3] Adicionar `Appearance Appearance` a `FlightRequest`, em `internal/domain/flight.go`
- [X] T035 [P] [US3] Em `internal/domain/builddomain/flight_request_builder.go`: `NewFlightRequestBuilder()` preenche `Appearance: NewAppearanceBuilder().Build()`; adicionar `WithAppearance(domain.Appearance) *FlightRequestBuilder`
- [X] T036 [P] [US3] Estender `internal/application/flight_service_test.go`: `Fly` repassa `request.Appearance`, sem alteração, para o `domain.FrameSetRequest` que monta para `frameService.DrawFrames`
- [X] T037 [US3] Em `internal/application/flight_service.go`, preencher `Appearance: request.Appearance` no `domain.FrameSetRequest` que `Fly` já monta; `make test` verde
- [X] T038 [P] [US3] Estender `internal/infra/inbound/cli/fly_test.go`: as cinco flags chegam em `domain.FlightRequest.Appearance`; omitidas, usam o padrão; flag inválida recusa antes de `flightService.Fly` ser chamado, com o mesmo erro que `render frame`/`render all` dariam para o mesmo valor
- [X] T039 [US3] Adicionar as cinco flags e o parâmetro `defaultAppearance domain.Appearance` a `NewFlightCommand`, chamando `parseAppearance` e preenchendo `FlightRequest.Appearance`, em `internal/infra/inbound/cli/fly.go`
- [X] T040 [US3] Em `cmd/sobrevoo/main.go`: passar o `defaultAppearance` já calculado (T028) para `cli.NewFlightCommand(...)`
- [X] T041 (adiantada: precisava estar pronta para os testes de US1)[P] [US3] Estender `internal/infra/inbound/cli/exit_code_test.go`: `errors.Is(err, domain.ErrInvalidColor)` → `52`; `errors.Is(err, domain.ErrInvalidTrailWidth)` → `53`; `errors.Is(err, domain.ErrInvalidMarkerRadius)` → `54`
- [X] T042 (adiantada: precisava estar pronta para os testes de US1)[US3] Adicionar os três `case` a `internal/infra/inbound/cli/exit_code.go` (`52`, `53`, `54`, nesta ordem, logo após o `51` existente)
- [X] T043 [US3] Validação manual: `quickstart.md` item 4 (mesmo valor, mesmo nome, mesmo efeito nos três comandos)

**Checkpoint**: as cinco flags funcionam, com o mesmo nome e o mesmo efeito, em `render frame`, `render all` e `fly`; códigos `52`–`54` mapeados.

---

## Phase 6: User Story 4 — Nunca confundir quadros de aparências diferentes (Priority: P4)

**Objetivo**: confirmar — sem nenhuma lógica nova (já sai da Phase 2, Parte C)
— que retomar `render all` ou reaproveitar `fly --keep` com uma aparência
diferente da já presente redesenha (ou recusa sem `--overwrite`), e que a
mesma aparência sempre reaproveita.

**Teste Independente**: `quickstart.md` itens 6 e 7.

- [X] T044 [P] [US4] Estender `internal/application/frame_service_test.go`: `DrawFrames`, com o `FrameRepository` mockado devolvendo um `FrameDirectory` de quadros já lá com um `SetID` de outra `Appearance` (mesmo plano/recorte/resolução), recusa com `ErrFrameSetConflict` sem `Overwrite`; com a mesma `Appearance` de antes, reaproveita (`summary.Kept` bate com o total)
- [X] T045 [P] [US4] Estender `internal/application/flight_service_test.go`: duas chamadas de `Fly` com `Keep` preenchido, mesmo trajeto e mesmos parâmetros, mas `Appearance` diferente — `PlanReused`/`SliceReused` continuam `true` (não dependem de aparência) e `frameService.DrawFrames` é chamado para o conjunto novo (não reaproveita os quadros); com a mesma `Appearance` de antes, os quadros também são reaproveitados
- [X] T046 [US4] Validação manual: `quickstart.md` itens 6 e 7 (retomada de `render all` recusando/reaproveitando por aparência; `fly --keep` refazendo só quadros e vídeo quando só a aparência muda)

**Checkpoint**: nenhuma combinação de execuções mistura, no mesmo diretório ou no mesmo `--keep`, quadros de aparências diferentes.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Propósito**: o caso extremo que não pertence a nenhuma história específica, a documentação e a verificação final.

- [X] T047 [P] Validação manual: `quickstart.md` item 8 (cor coincidente com `NoMapColors`/`NoElevationColors` é aceita normalmente, e o hachurado/xadrez continua com as próprias cores fixas)
- [X] T048 [P] Atualizar `CLAUDE.md`: a oitava etapa, `Appearance`, as cinco flags nos três comandos, a observação de que nenhuma regra de câmera/dados/vídeo muda
- [X] T049 [P] Atualizar `README.md`: as cinco flags novas nas seções de `render frame`, `render all` e `fly` já existentes (mesma estrutura de tabela de flags que as demais)
- [X] T050 Verificação final: `make build`, `make test`, `make lint`, `make generate` verdes (confirmar que `make generate` não produz nenhuma diferença — nenhuma interface mudou de método); `gofmt -l .` limpo; reexecutar `quickstart.md` por inteiro

---

## Dependências e Ordem de Execução

### Dependências entre fases

- **Phase 1 (Setup)**: sem dependências.
- **Phase 2 (Foundational)**: depende da Phase 1; **bloqueia todas as histórias**. Partes A → B → C → D são sequenciais entre si (B usa o tipo que A declara; C usa `Appearance` que A declara; D mapeia para o tipo que A declara) — dentro de cada parte, as tarefas `[P]` são paralelas.
- **Phases 3 a 6 (US1 a US4)**: dependem da Phase 2 concluída; cada uma edita os mesmos arquivos de CLI/aplicação que a anterior estendeu, então seguem a ordem P1 → P4. US2 a US4 dependem da US1 (o MVP e o `appearance.go` compartilhado que ela cria).
- **Phase 7 (Polish)**: depende das histórias desejadas.

### Dependências entre histórias

- **US1 (P1)**: depende só da Foundational. Cria `appearance.go` (`parseAppearance`), usado por todas as demais.
- **US2 (P2)**: depende da US1 (`appearance.go`, `defaultAppearance` em `main.go`).
- **US3 (P3)**: depende da US1; independente de US2 (edita `fly.go`/`flight_service.go`/`flight.go`, não `render_all.go`).
- **US4 (P4)**: depende da US1 (precisa de `Appearance` chegando a `DrawFrames`/`Fly`) e, para o cenário de `fly --keep`, da US3; só acrescenta teste e validação manual, nenhuma implementação nova.

### Dentro de cada história

- Testes antes da implementação (devem falhar primeiro — ver a ressalva do preâmbulo para as tarefas de mudança de assinatura).
- Domínio antes de aplicação; aplicação antes da CLI; CLI antes de `main.go`.
- A história é completa antes da próxima prioridade.

### Oportunidades de paralelismo

```text
# Foundational, Parte A, tudo em paralelo:
T003 frame_appearance_test.go   T005 errors.go   T006 appearance_builder.go
→ T004 frame_appearance.go (implementação que T003 cobra)

# Foundational, Parte B (sequencial: cada arquivo depende do de baixo):
T007/T008 (frame_image) → T009/T010 (frame_imagery, usa RGB de frame_image) →
T011/T012 (frame_overlay) → T013/T014 (frame_scene, monta overlay+imagery) → T015 (limpeza)

# Foundational, Parte C:
T016 frame_set_test.go → T017 frame_set.go

# Foundational, Parte D, em paralelo entre si:
T018 config_test.go → T019 config.go
T020 config_mapping_test.go → T021 config_mapping.go

# US1:
T022 frame_service_test.go → T023 frame_service.go
T024 appearance_test.go → T025 appearance.go (pode começar em paralelo com T022/T023)
T026 render_frame_test.go → T027 render_frame.go → T028 main.go → T029 manual

# US2 a US4: os testes de cada história são [P] entre si (arquivos de teste
# diferentes), mas a implementação de cada história edita main.go em sequência
# (cada uma acrescenta uma linha à chamada de um comando diferente) — nunca em
# paralelo entre histórias.
```

---

## Estratégia de Implementação

### MVP Primeiro (Phase 1 + 2 + User Story 1)

1. Phase 1 (branch e linha de base) e Phase 2 (`Appearance`; `Scene`/`overlay`/`imagery`/`FrameImage` recebendo-a; `FrameSetID` dependendo dela; `config.RenderDefaults` com os cinco valores de hoje).
2. Phase 3 (US1): `parseAppearance` e as cinco flags em `render frame`.
3. **PARE e valide**: `quickstart.md` itens 1, 2 e 5 — padrão preservado, aparência muda o que deveria, valor inválido recusado — antes de acrescentar histórias.

### Entrega Incremental

1. Setup + Foundational → `Appearance` disponível em todo o núcleo, sem nenhuma flag ainda.
2. + US1 → `render frame` com as cinco flags (MVP) → itens 1, 2, 5.
3. + US2 → `render all` com as mesmas flags, byte a byte igual a `render frame` → item 3.
4. + US3 → `fly` com as mesmas flags, códigos `52`–`54` → item 4.
5. + US4 → confirmação de que aparências diferentes nunca se misturam → itens 6, 7.
6. Polish → coincidência de cor permitida, documentação, verificação final → item 8.

Cada história agrega valor sem quebrar as anteriores.

---

## Notas

- `[P]` = arquivos diferentes, sem dependência de tarefa incompleta.
- Nenhuma regra de câmera, de dados geográficos, de codificação ou de
  enquadramento nasce nesta etapa (FR-006): toda tarefa de implementação só
  acrescenta um parâmetro/campo que carrega a aparência escolhida até onde ela
  já era desenhada com valores fixos — se uma tarefa parecer exigir mudar
  onde/como o quadro é enquadrado, o desenho está errado, não a tarefa
  (revisar `research.md` antes de prosseguir).
- Faça commit após cada tarefa ou grupo lógico (só quando o usuário pedir).
- Pare em qualquer checkpoint para validar a história com o `quickstart.md`.
- Evite: tarefas vagas, duas tarefas `[P]` editando o mesmo arquivo,
  dependências que quebrem a independência de teste de uma história.
