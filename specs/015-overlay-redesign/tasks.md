---

description: "Task list for feature implementation"
---

# Tarefas: Redesenho das Sobreposições de Tela em Colunas

**Entrada**: Documentos de design de `/specs/015-overlay-redesign/`

**Pré-requisitos**: `plan.md`, `spec.md`, `research.md`, `data-model.md`,
`contracts/overlay-blocks-update.md`, `contracts/frame-files-change.md`,
`quickstart.md`

**Testes**: incluídos. A constituição do projeto (Princípio VI — Testes
Automatizados no Núcleo; Princípio X — Given/When/Then, builders e
isolamento por camada) exige testify e proíbe testes tabulares: cada
cenário é um `t.Run("should ...")` com `// given`, `// when`, `// then`. As
tarefas de teste ficam antes da implementação correspondente e devem
falhar primeiro. Como nas etapas 9/11/12/14: várias tarefas desta etapa
**mudam ou removem** comportamento e assinaturas já existentes
(`stablePanelWidth`, `screenOverlay.panelWidth`, `drawPanel`,
`OverlayPanelColor`/`Opacity`, as funções `formatOverlay*`), não só
acrescentam — "estender o teste" aqui inclui apagar ou reescrever cenários
cujo mecanismo deixou de existir, sem os quais o pacote nem compila.

**Organização**: as tarefas são agrupadas por história de usuário (P1–P4
de `spec.md`). A **Phase 2 (Foundational)** carrega só o que `gain`
precisa existir como configuração válida (constante, campo, caso no
`switch`, impressão digital, builder) e a ordem fixa dos blocos — nenhum
pixel muda ainda. **US1 (P1)** é o grosso da etapa: troca a pilha vertical
e o painel compartilhado por colunas de mesma largura, três alturas de
texto e rótulos por extenso — e, por exigência mecânica do próprio formato
de três alturas (um bloco não cabe dois números), é aqui que o ganho sai
do texto de `elevation` e ganha sua própria função de texto, mesmo sendo
US2 quem valida e expõe isso de ponta a ponta pela CLI. **US2 (P2)**
termina a exposição de `gain` (texto de ajuda, listagem) e prova a
independência. **US3 (P3)** troca só o valor padrão de configuração. **US4
(P4)** remove o painel do gráfico de elevação, a última coisa que ainda
usa `drawPanel` — só depois dela `drawPanel`/`OverlayPanelColor`/
`OverlayPanelOpacity` podem ser apagados, por não terem mais nenhum
chamador.

**Divisão do trabalho entre histórias**:

| História | O que entrega | Veículo |
|---|---|---|
| US1 (P1) | a limpeza visual central: sem painel, em colunas, três alturas, ordem fixa | `screenOverlay.draw`/`drawBlock` reescritos em `frame_screen_overlay.go` |
| US2 (P2) | `gain` escolhível de forma totalmente independente de `elevation`, descoberto pelo `--help` | `overlay.go` (`overlayBlocksOf`, textos de ajuda) + prova de independência |
| US3 (P3) | o conjunto padrão que faz sentido num vídeo (velocidade, elevação, distância, perfil) | `config.RenderDefaults.OverlayBlocks` |
| US4 (P4) | o gráfico de elevação também sem painel, com contorno | `drawProfile` reescrito; `drawPanel` e as duas constantes removidos |

Enquanto a Phase 2 não existe, nada deste arquivo compila (o `switch` de
`overlayBlockOrder`, usado desde a US1, referencia `OverlayBlockGain`).
Enquanto a US1 não existe, nenhum comando desenha em colunas. Enquanto a
US4 não existe, o gráfico de elevação ainda usa o painel — `drawPanel`
continua tendo um chamador até lá.

## Formato: `[ID] [P?] [Story] Descrição`

- **[P]**: pode ser executado em paralelo com as demais tarefas `[P]` da
  mesma seção (arquivos diferentes). Tarefas que editam o mesmo arquivo
  NUNCA são marcadas `[P]` entre si, mesmo quando logicamente
  independentes.
- **[Story]**: a qual história de usuário esta tarefa pertence (US1–US4).
  Tarefas de Setup, Foundational e Polish não têm esse rótulo.
- Toda tarefa inclui o caminho de arquivo exato a criar/editar.
- Comentários de código, identificadores, mensagens de commit, flags,
  saída e mensagens de erro em tempo de execução: **inglês**; artefatos do
  Spec Kit: português (constituição, "Idioma dos Artefatos").

## Convenções de Caminho

Mesmo projeto único em Go, mesma estrutura hexagonal de `plan.md`:
`internal/domain/`, `internal/infra/outbound/config/`,
`internal/infra/inbound/cli/`. Builders em `internal/domain/builddomain`.
Receivers curtos e consistentes com o tipo (`s` `screenOverlay`/`Scene`,
`o` `OverlayConfig`); nome exportado nunca repete o pacote.

---

## Phase 1: Setup (Shared Infrastructure)

**Propósito**: um ponto de partida próprio (branch) e verde.

- [X] T001 Criar/trocar para o branch `015-overlay-redesign` (`git checkout -b 015-overlay-redesign` a partir de `main`, se ainda não existir)
- [X] T002 Confirmar `make build`, `make test`, `make lint` e `make generate` verdes antes de qualquer mudança (linha de base)

**Checkpoint**: repositório pronto para a Phase 2.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Propósito**: `gain` passa a existir como nome de bloco e campo de
configuração válidos, com sua própria impressão digital, e a ordem fixa
dos cinco blocos numéricos passa a existir como dado — **nenhum pixel
muda ainda**. Nenhuma história de usuário começa antes desta fase estar
verde.

- [X] T003 [P] Estender `internal/domain/frame_overlay_config_test.go`:
  - `Test_NewOverlayConfig`: "should turn on the gain block when named, independent of elevation" — `NewOverlayConfig(true, []OverlayBlock{OverlayBlockGain})` → `config.Gain == true`, `config.Elevation == false`; atualizar "should refuse an unknown block name" para esperar a lista de seis nomes na mensagem (`"...distance, elevation, gain, time, profile, speed"`)
  - `Test_OverlayConfig_Fingerprint`: "should change when Gain changes" (mesmo padrão dos outros seis cenários)
  deve falhar até T004
- [X] T004 Em `internal/domain/frame_overlay_config.go`:
  - `OverlayBlockGain OverlayBlock = "gain"` (nova constante, comentário citando que, ao contrário dos quatro originais, nasce fora do padrão — como `OverlayBlockSpeed` já é)
  - `OverlayConfig`: campo `Gain bool`
  - `NewOverlayConfig`: `case OverlayBlockGain: config.Gain = true`; mensagem do `default` passa a citar os seis nomes, nessa ordem de declaração dos `case`s
  - `Fingerprint()`: acrescentar `"|" + flag(o.Gain)` ao final
  - novo `var overlayBlockOrder = []OverlayBlock{OverlayBlockSpeed, OverlayBlockElevation, OverlayBlockDistance, OverlayBlockGain, OverlayBlockTime}` (comentário citando 015-overlay-redesign Clarifications, Session 2026-10-04 — ordem fixa, independente da ordem pedida em `--overlay-blocks`; ainda sem nenhum consumidor nesta fase)
  - `make test` verde (depende de T003)
- [X] T005 Em `internal/domain/builddomain/overlay_config_builder.go`: adicionar `WithGain() *OverlayConfigBuilder` (liga `config.Gain = true`, mesmo padrão de `WithSpeed()`) (depende de T004)

**Checkpoint**: `gain` é uma configuração válida, com impressão digital
própria; a ordem fixa existe como dado; nenhum comando desenha nada
diferente ainda.

---

## Phase 3: User Story 1 — Ver os números direto sobre a imagem, sem faixas escuras (Priority: P1) 🎯 MVP

**Objetivo**: os blocos numéricos presentes (em qualquer combinação)
aparecem lado a lado, em colunas de mesma largura, sem nenhum painel de
fundo, numa ordem fixa, cada um em até três linhas (rótulo por extenso,
valor maior, unidade) — nunca empilhados, nunca na ordem em que foram
pedidos.

**Teste Independente**: `quickstart.md` itens 1, 2, 3 e 7.

### Parte A — tamanhos de corpo e versão do desenho (`render_tuning.go`; research.md item 3, 9)

- [X] T006 [P] [US1] Em `internal/domain/render_tuning.go`: adicionar `OverlayLabelHeightRatio = 0.020` (rótulo e unidade) e `OverlayValueHeightRatio = 0.040` (valor — o dobro do rótulo), ao lado de `OverlayOutlineRatio`; `RenderVersion`: `4` → `5`, estendendo o comentário histórico com a nota desta etapa (colunas, três alturas, sem painel nos blocos numéricos, bloco `gain`)
- [X] T007 [P] [US1] Em `internal/domain/frame_set_test.go` linha ~290: atualizar `assert.Equal(t, 4, domain.RenderVersion)` para `5` (depende de T006)

### Parte B — valor e unidade separados (`research.md` item 7; `data-model.md`)

- [X] T008 [P] [US1] Estender `internal/domain/frame_screen_overlay_test.go`: `Test_FormatOverlayDistance`/`Test_FormatOverlayElevation`/`Test_FormatOverlayGain`/`Test_FormatOverlaySpeed` passam a esperar `(value, unit string)` em vez de uma string só, mantendo exatamente os mesmos números e arredondamentos já cobertos (ex.: `formatOverlayDistance(850)` → `("850", "m")`; `formatOverlayDistance(12345)` → `("12.3", "km")`; `formatOverlayGain(567.4)` → `("+567", "m")`; `formatOverlaySpeed(10.0/3.6)` → `("10.0", "km/h")`); `Test_FormatOverlayElapsed` não muda (continua sem unidade) — deve falhar até T009
- [X] T009 [US1] Em `internal/domain/frame_screen_overlay.go`: `formatOverlayDistance`/`formatOverlayElevation`/`formatOverlayGain`/`formatOverlaySpeed` passam a devolver `(value, unit string)` — mesmo cálculo e arredondamento de hoje (FR-013), só a forma do retorno muda; `formatOverlayElapsed` não muda; `make test` verde (depende de T008)

### Parte C — rótulos por extenso; `elevation` e `gain` com texto próprio (`research.md` itens 4, 7; `data-model.md`)

- [X] T010 [US1] Reescrever `Test_BlockText` em `internal/domain/frame_screen_overlay_test.go` em torno de um novo tipo `overlayBlockText{label, value, unit string}`:
  - "should write the distance block with its full Portuguese label" — `distanceBlockText(frame).label == "Distância"`, `.value`/`.unit` vindos de `formatOverlayDistance`
  - "should write the elevation block with only the altitude, no gain" — `elevationBlockText(frame).label == "Elevação"`, valor/unidade de `formatOverlayElevation`, nenhuma referência a ganho
  - "should write the gain block on its own" (bloco novo) — `gainBlockText(frame).label == "Ganho"`, valor/unidade de `formatOverlayGain`
  - "should write the time block without a unit" — `timeBlockText(frame).label == "Tempo decorrido"`, `.unit == ""`
  - "should write the speed block" — `speedBlockText(frame).label == "Velocidade"`
  - "should never write the old abbreviated, upper-case labels" — nenhum dos cinco rótulos contém `"DIST"`, `"ELEV"`, `"GANHO"`, `"TEMPO"` ou `"VEL"`
  deve falhar até T011 (depende de T009)
- [X] T011 [US1] Em `internal/domain/frame_screen_overlay.go`: declarar `type overlayBlockText struct { label, value, unit string }`; reescrever `distanceBlockText`/`elevationBlockText`/`timeBlockText`/`speedBlockText` para devolver `overlayBlockText` com rótulo por extenso, capitalização normal (`Distância`, `Elevação`, `Tempo decorrido`, `Velocidade`); `elevationBlockText` não inclui mais o ganho; nova `gainBlockText(frame CameraFrame) overlayBlockText` (`label: "Ganho"`); `make test` verde (depende de T010)

### Parte D — layout em colunas, sem painel compartilhado (`research.md` itens 1, 2; `data-model.md`)

- [X] T012 [US1] Em `internal/domain/frame_screen_overlay_test.go`: remover por inteiro `Test_StablePanelWidth` (a função deixa de existir); em `Test_ScreenOverlay_Draw`, remover o campo `panelWidth` de todo literal `screenOverlay{...}` e os dois cenários que dependiam da largura compartilhada ("should give the distance panel the width the longer elevation text needs..." e "...only its own width when it is the only numeric block shown") — o conceito de painel compartilhado deixou de existir; adicionar:
  - "should draw blocks side by side, in columns, not stacked" — com dois blocos (`distance`, `time`) ligados, a região abaixo da margem superior na segunda metade da largura útil já muda (prova de que há conteúdo na segunda coluna, não só na primeira linha empilhada)
  - "should draw the same pixels whatever order the blocks were named in NewOverlayConfig" — dois `OverlayConfig` montados com os mesmos blocos em ordens diferentes produzem exatamente os mesmos bools (e, portanto, o mesmo desenho) — a ordem de desenho nunca lê a ordem de `blocks`, só os campos booleanos
  - "should not leave a gap for a block that is off, in the middle of the fixed order" — comparar o desenho de `{Distance, Speed}` contra `{Distance, Elevation: false, Speed}` (elevação desligada, no meio da ordem fixa `overlayBlockOrder`) — as colunas de `Distance` e `Speed` caem exatamente nas mesmas posições nos dois casos
  deve falhar até T013
- [X] T013 [US1] Reescrever `draw()` em `internal/domain/frame_screen_overlay.go`: percorrer `overlayBlockOrder`, incluindo cada bloco cujo gate de hoje permanece (`Distance` sempre; `Elevation`/`Gain` && `plan.ElevationAvailable`; `Time`/`Speed` && `plan.TimeReference == TimeReferenceClock`) numa lista de `overlayBlockText`; dividir `width - 2*marginSide` por `len(present)` (divisão inteira — determinística por construção, sem ponto flutuante) em colunas de mesma largura; para cada bloco presente, chamar o novo `drawBlock` no centro da própria coluna (`marginSide + colWidth*i + colWidth/2`); remover o campo `panelWidth` de `screenOverlay` e a função `stablePanelWidth`; `make test` verde (depende de T004, T011, T012)
- [X] T014 [P] [US1] Em `internal/domain/frame_screen_overlay_test.go`: adicionar `Test_DrawBlock` (nova função): "should draw up to three lines horizontally centered on centerX" (rótulo, valor e unidade, cada um centralizado por sua própria largura em `face.textWidth`, nos corpos `OverlayLabelHeightRatio`/`OverlayValueHeightRatio`); "should draw only two lines, with no gap, when unit is empty" (tempo decorrido) — deve falhar até T015
- [X] T015 [US1] Implementar `drawBlock(centerX, topY, labelPpem, valuePpem int, text overlayBlockText)` em `internal/domain/frame_screen_overlay.go`: desenha rótulo (`labelPpem`), valor (`valuePpem`) e, só quando `text.unit != ""`, a unidade (`labelPpem`), cada linha centralizada em `centerX` por uma função auxiliar `drawCenteredLine(centerX, y, ppem int, line string)` (`x := centerX - face.textWidth(line, ppem)/2`, chamando o `drawText` já existente); sem terceira linha nem avanço de `y` reservado quando não há unidade (FR-006); `make test` verde (depende de T013, T014)

### Parte E — `Scene` para de medir o texto mais largo do plano (`research.md` item 1; `data-model.md`)

- [X] T016 [US1] Em `internal/domain/frame_scene_test.go`: remover o cenário "should give a frame a wider numeric panel when another frame of the same plan has longer text (012-overlay-ptbr-readability)" de `Test_Scene_Render_ScreenOverlay` (o mecanismo que ele prova — largura compartilhada medida do plano inteiro — deixou de existir nesta etapa); confirmar que nenhum outro teste do arquivo monta `screenOverlay{...}` com `panelWidth` nem chama `Scene.numericPanelWidth`
- [X] T017 [US1] Em `internal/domain/frame_scene.go`: remover os campos `panelWidth`/`panelWidthHeight`/`panelWidthSet` de `Scene` e o método `numericPanelWidth`; `Render()` para de chamá-lo e de passar `panelWidth` no literal `screenOverlay{...}`; `make build`/`make test` verdes (depende de T013, T016)

### Parte F — hash de referência e validação (`CLAUDE.md`, "O desenho dos quadros")

- [X] T018 [US1] Recalcular o hash de `Test_Scene_Render_Determinism`/"should draw the image the reference says, on any machine" (`internal/domain/frame_scene_test.go`) (depende de T006, T013, T015, T017) — **achado**: `roughScene` (usada só por esse cenário) desenha com `domain.OverlayConfig{}` (sobreposições desligadas), e `screenOverlay.draw` retorna cedo quando `!Enabled`; como nada desta etapa toca o terreno/traçado/marcador, o hash existente já passa sem nenhuma mudança — confirmado rodando o teste (verde)
- [X] T019 [US1] Validação manual: `quickstart.md` itens 1, 2, 3 e 7 (sem painel nos blocos numéricos; lado a lado em colunas, mesma ordem independente da pedida, confirmado por `diff`; três alturas, com e sem unidade; determinismo byte a byte)

**Checkpoint**: `render frame`, `render all` e `fly` desenham os blocos
numéricos em colunas, sem painel, com três alturas — a limpeza visual
central está entregue (MVP).

---

## Phase 4: User Story 2 — Escolher a elevação e o ganho acumulado de forma independente (Priority: P2)

**Objetivo**: `gain` é descoberto pelo `--help` como os demais blocos e
comprovadamente independente de `elevation` — pedir um nunca traz nem
esconde o outro.

**Teste Independente**: `quickstart.md` item 4.

- [X] T020 [P] [US2] Estender `internal/infra/inbound/cli/render_frame_test.go` (`Test_RenderFrameCommand_Overlay`): "should turn on the gain block when named in --overlay-blocks, independent of elevation" — `--overlay-blocks=gain` produz `domain.OverlayConfig{Enabled: true, Gain: true}` (sem `Elevation`) passado a `frameService.DrawFrame` — deve falhar até T021 (a validação do nome já passa desde T004; o que falta é a descoberta pelo `--help`, não o parsing)
- [X] T021 [US2] Em `internal/infra/inbound/cli/overlay.go`: `overlayBlocksOf`: acrescentar `if config.Gain { blocks = append(blocks, domain.OverlayBlockGain) }`; `overlaysUsage`/`overlayBlocksUsage`: passam a citar `gain` entre os nomes aceitos; `make test` verde (depende de T004, T020)
- [X] T022 [US2] Em `internal/domain/frame_screen_overlay_test.go` (`Test_ScreenOverlay_Draw`): adicionar "should draw the elevation block without gain, and the gain block without elevation, when each is requested alone" e "should draw both as distinct blocks when elevation and gain are requested together" — ambos já devem passar sem nenhuma mudança de produção (T011/T013 da US1 já tornam os dois blocos independentes); registrar isso como a prova explícita de FR-008/FR-009, sem código novo (mesmo padrão da comprovação T024 da etapa 14) (depende de T013) — **comprovado**: os dois cenários já foram escritos junto de T012/T013 (Parte D da US1, por já estarem naturalmente cobertos pela reescrita genérica de `draw()`) e passam sem nenhuma mudança adicional
- [X] T023 [US2] Validação manual: `quickstart.md` item 4 (elevação sozinha, ganho sozinho, os dois juntos; nome de bloco inválido lista os seis nomes aceitos)

**Checkpoint**: `gain` aparece no `--help` e funciona de forma totalmente
independente de `elevation`, nos três comandos.

---

## Phase 5: User Story 3 — Obter, sem pedir nada, o conjunto de blocos que faz sentido num vídeo (Priority: P3)

**Objetivo**: sem `--overlay-blocks`, o vídeo mostra exatamente
velocidade, elevação e distância no alto, e o perfil de elevação no
rodapé — nunca tempo decorrido nem ganho.

**Teste Independente**: `quickstart.md` item 5.

- [X] T024 [P] [US3] Em `internal/infra/outbound/config/config_test.go` linha ~143: atualizar a asserção que hoje espera `[]string{"distance", "elevation", "time", "profile"}` para `[]string{"distance", "elevation", "speed", "profile"}` — deve falhar até T025
- [X] T025 [US3] Em `internal/infra/outbound/config/config.go` linha ~233: trocar o valor padrão de `RenderDefaults.OverlayBlocks` para `[]string{"distance", "elevation", "speed", "profile"}`; `make test` verde (depende de T024)
- [X] T026 [US3] Validação manual: `quickstart.md` item 5 (sem `--overlay-blocks`, aparecem velocidade/elevação/distância/perfil; pedindo `time`/`gain` explicitamente, eles aparecem também)

**Checkpoint**: a escolha padrão é a nova, em `render frame`,
`render all` e `fly`.

---

## Phase 6: User Story 4 — Destacar o gráfico de elevação do terreno sem um painel (Priority: P4)

**Objetivo**: o gráfico de elevação no rodapé perde o painel escuro e
passa a se destacar do terreno pelo mesmo contorno que o texto já usa —
depois desta história, `drawPanel` não tem mais nenhum chamador.

**Teste Independente**: `quickstart.md` item 1 (parte do gráfico de
elevação).

- [X] T027 [US4] Em `internal/domain/frame_screen_overlay_test.go` (`Test_ScreenOverlay_Draw`): adicionar "should draw the profile's line and marker with an outline instead of a panel" — nenhum pixel misturado com `OverlayPanelColor`/`OverlayPanelOpacity` aparece na área do gráfico; a linha e o marcador têm uma borda de `OverlayTextOutlineColor` em torno de si, do mesmo jeito que o texto já tem — deve falhar até T028
- [X] T028 [US4] Em `internal/domain/frame_screen_overlay.go` (`drawProfile`): remover a chamada a `drawPanel`; em `drawSegment`/`plotSquare` e `drawDot`, desenhar primeiro uma versão da mesma forma, `+2` pixels de espessura/raio, em `OverlayTextOutlineColor`, depois a versão normal (`thickness`/`radius`) em `OverlayTextColor`/`appearance.MarkerColor` por cima — a mesma técnica de casca-antes-do-núcleo que `TrailCasingColor` já usa (research.md item 6); `make test` verde (depende de T027)
- [X] T029 [US4] Em `internal/domain/frame_screen_overlay_test.go`: remover `Test_DrawPanel` por inteiro (a função deixa de ter chamador); em `internal/domain/frame_screen_overlay.go`/`internal/domain/render_tuning.go`: remover a função `drawPanel` e as constantes `OverlayPanelColor`/`OverlayPanelOpacity`; `make build`/`make test` verdes (depende de T028)
- [X] T030 [US4] Recalcular novamente o hash de `Test_Scene_Render_Determinism` (`internal/domain/frame_scene_test.go`) — a remoção do painel do gráfico de elevação muda pixels além dos já alterados em T018, quando o bloco `profile` está ligado (depende de T018, T028) — **achado**: mesma razão de T018 (`roughScene` desenha com as sobreposições desligadas); o hash existente já passa sem nenhuma mudança
- [X] T031 [US4] Validação manual: `quickstart.md` item 1 (o gráfico de elevação, sem painel, com contorno visível sobre o terreno)

**Checkpoint**: nenhum bloco — numérico ou gráfico — desenha mais sobre um
painel; `drawPanel` não existe mais no código.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Propósito**: documentação e verificação final.

- [X] T032 [P] Atualizar `CLAUDE.md`: a décima quinta etapa (resumo do projeto, no mesmo estilo das demais), citando a troca da pilha vertical por colunas de mesma largura, as três alturas de texto (rótulo por extenso, valor, unidade), o bloco `gain` independente, o novo conjunto padrão e a remoção do painel do gráfico de elevação — mais uma seção dedicada `### O redesenho das sobreposições de tela em colunas (etapa 15)`, no mesmo padrão das etapas 9/11/12/14
- [X] T033 [P] Atualizar `README.md` (seção "Sobreposições de tela"): refletir o novo arranjo em colunas sem painel, os rótulos por extenso, o bloco `gain` e o novo conjunto padrão (`distance,elevation,speed,profile`)
- [X] T034 [P] Acrescentar, em `specs/009-frame-overlays/contracts/overlay-flags.md` e `specs/005-frame-rendering/contracts/frame-files.md`, a nota desta etapa (apontando para `specs/015-overlay-redesign/contracts/*`), do mesmo jeito que as etapas 6, 8, 9, 11 e 12 já fizeram
- [X] T035 Verificação final: `make build`, `make test`, `make lint`, `make generate` verdes (confirmar que `make generate` não produz nenhuma diferença — nenhuma interface de porta ou de serviço mudou de método nesta etapa); `gofmt -l .` limpo; reexecutar `quickstart.md` por inteiro, do item 1 ao 7

---

## Dependências e Ordem de Execução

### Dependências entre Fases

- **Phase 1 (Setup)**: sem dependências.
- **Phase 2 (Foundational)**: depende da Phase 1; **bloqueia US1–US4** (o `switch` de `overlayBlockOrder`, consumido desde a US1, referencia `OverlayBlockGain`).
- **Phase 3 (US1)**: depende da Phase 2 completa. É o maior bloco de trabalho — Partes A a F têm uma ordem real entre si (A e B/C independem uma da outra; D depende de B/C; E depende de D; F depende de A+D+E).
- **Phase 4 (US2)**: depende da Phase 3 completa (edita `frame_screen_overlay.go`/`overlay.go`, e a prova de independência em T022 depende do `draw()` já reescrito em T013).
- **Phase 5 (US3)**: depende só da Phase 2 (edita só `config.go`) — não depende de US1/US2 em código, mas faz mais sentido depois delas por ordem de prioridade.
- **Phase 6 (US4)**: depende da Phase 3 completa (edita `drawProfile`, que já precisa existir no novo formato); é a única história que remove `drawPanel`/`OverlayPanelColor`/`OverlayPanelOpacity` — só pode fazê-lo depois que a Phase 3 já removeu o uso deles pelos blocos numéricos.
- **Phase 7 (Polish)**: depende das histórias desejadas.

### Dentro de Cada História de Usuário

- Testes são escritos (ou estendidos/reescritos) e devem falhar antes da tarefa de implementação correspondente.
- Dentro da Phase 3: tamanhos/versão (Parte A) e formatação (Parte B/C) não dependem uma da outra; layout em colunas (Parte D) depende de ambas; `Scene` (Parte E) depende de D; hash de referência (Parte F) depende de tudo.
- Domínio antes de infraestrutura de saída (`config`); infraestrutura antes de CLI.

### Oportunidades de Paralelização

- Dentro da Phase 3: T006/T007 (Parte A) podem ser trabalhados em paralelo com T008/T009/T010/T011 (Partes B/C) — arquivos diferentes, sem dependência mútua.
- Phase 5 (US3) pode ser trabalhada em paralelo com a Phase 3 (US1) ou a Phase 4 (US2) por outra pessoa, já que edita só `config.go`/`config_test.go` — só depende da Phase 2.
- Phase 7: T032/T033/T034 (documentação) podem ser feitas em paralelo entre si.

---

## Estratégia de Implementação

### MVP Primeiro (Somente História de Usuário 1)

1. Completar Phase 1: Setup.
2. Completar Phase 2: Foundational (CRÍTICO — bloqueia as quatro histórias).
3. Completar Phase 3: História de Usuário 1 (colunas, sem painel, três alturas).
4. **PARAR E VALIDAR**: `quickstart.md` itens 1, 2, 3 e 7.
5. Implantar/demonstrar se estiver pronta.

### Entrega Incremental

1. Setup + Foundational → `gain` existe como configuração, nada visível ainda.
2. US1 (o redesenho visual, MVP) → validar → demonstrar.
3. US2 (`gain` descoberto e comprovadamente independente) → validar.
4. US3 (novo padrão) → validar — pode ser feita em paralelo com US1/US2.
5. US4 (perfil sem painel; `drawPanel` removido) → validar.
6. Polish.

Cada história agrega valor sem quebrar as anteriores; US3 é a mais barata
das quatro (uma troca de valor de configuração, sem nenhuma decisão
técnica nova), exatamente como a US3 da etapa 13 e a US2 da etapa 14 já
foram as mais baratas das suas respectivas features.
