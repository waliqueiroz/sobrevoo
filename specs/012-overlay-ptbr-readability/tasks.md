---

description: "Task list for feature implementation"
---

# Tarefas: Rótulos em Português e Legibilidade do Texto das Sobreposições

**Entrada**: Documentos de design de `/specs/012-overlay-ptbr-readability/`

**Pré-requisitos**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/frame-files-change.md`, `quickstart.md`

**Testes**: incluídos. A constituição do projeto (Princípio VI — Testes
Automatizados no Núcleo; Princípio X — Given/When/Then, builders e
isolamento por camada) exige testify e proíbe testes tabulares: cada
cenário é um `t.Run("should ...")` com `// given`, `// when`, `// then`.

**Organização**: as tarefas são agrupadas por história de usuário (P1–P4
de `spec.md`). Diferente de `011-overlay-polish`, as quatro histórias
**não** são inteiramente ortogonais: a US4 (largura estável) reaproveita
`distanceBlockText`/`elevationBlockText`/`timeBlockText`, que a US1 extrai
com os rótulos já em português — por isso a US4 depende da US1, além da
ordem de prioridade. As US2 e US3 continuam independentes de todas as
outras (uma mexe só em `drawText`/`outlineRadius`, a outra só na fonte
embutida em `vector_font.go`).

| História | O que entrega | FR |
|---|---|---|
| US1 (P1) | rótulos em português (`DIST`/`ELEV`/`GANHO`/`TEMPO`) | FR-001–FR-004 |
| US2 (P2) | contorno do texto fino, proporcional ao tamanho da letra | FR-005 |
| US3 (P3) | fonte embutida de peso forte (`Go Bold`) | FR-006 |
| US4 (P4) | largura dos painéis numéricos estável do início ao fim do voo | FR-007/FR-008 |

**Nenhum mock precisa ser regenerado nesta etapa**: nenhuma porta nem
interface de serviço é tocada — tudo fica em tipos e funções não
exportados de `internal/domain`. `make generate` não deveria produzir
nenhuma diferença (verificado em T021).

## Formato: `[ID] [P?] [Story] Descrição`

- **[P]**: pode ser executado em paralelo (arquivos diferentes, sem
  dependência de tarefa incompleta). As tarefas de teste de cada história
  ficam em `frame_screen_overlay_test.go` (exceto a da US3, em
  `vector_font_test.go`) e são marcadas `[P]` entre si por conveniência de
  leitura — na prática, rode uma história por vez, pela ordem de
  prioridade, para não editar o mesmo arquivo de teste em paralelo de
  verdade. Tarefas de implementação que tocam o mesmo arquivo NUNCA são
  marcadas `[P]` entre si.
- **[Story]**: a qual história de usuário esta tarefa pertence (US1 a US4).
  Tarefas de Setup, Foundational e Polish não têm esse rótulo.
- Toda tarefa inclui o caminho de arquivo exato a criar/editar.
- Comentários de código, identificadores, mensagens de commit, e mensagens
  de erro em tempo de execução: **inglês**; artefatos do Spec Kit:
  português (constituição, "Idioma dos Artefatos"). Os **rótulos
  desenhados no quadro** (`"DIST "`, `"ELEV "`, `"GANHO "`, `"TEMPO "`) são
  a única exceção: são dado em português do Brasil, por definição desta
  etapa (FR-001), mesmo dentro de código-fonte em inglês.

## Convenções de Caminho

Mesmo projeto único em Go, mesma estrutura hexagonal de `plan.md`: toda a
mudança fica em `internal/domain/`. Regras de estilo: receivers curtos e
consistentes com o tipo (`s` `screenOverlay`/`Scene`, `f` `vectorFace`);
nome exportado nunca repete o pacote; receiver sem uso fica sem nome.

---

## Phase 1: Setup (Shared Infrastructure)

**Propósito**: um ponto de partida próprio (branch) e verde.

- [X] T001 Criar e mudar para o branch `012-overlay-ptbr-readability` (a partir do estado atual, que ainda inclui o trabalho não commitado de `011-overlay-polish` — nenhum commit foi pedido) (`git checkout -b 012-overlay-ptbr-readability`)
- [X] T002 Confirmar `make build`, `make test`, `make lint` e `make generate` verdes antes de qualquer mudança (linha de base)

**Checkpoint**: repositório pronto para a Phase 2.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Propósito**: a única mudança que toda história pressupõe — a versão do
desenho já subir, para que cada história, sozinha, já produza um conjunto
de quadros que o restante do pipeline nunca confunde com o de antes dela.
**Nenhuma história começa antes desta fase estar verde.**

- [X] T003 [P] Em `internal/domain/render_tuning.go`: `RenderVersion` de `3` para `4` (comentário atualizado, research.md item 7); atualizar `internal/domain/frame_set_test.go` (o teste que hoje fixa `assert.Equal(t, 3, domain.RenderVersion)` → `4`, único lugar que fixa o valor); `make test` verde

**Checkpoint**: `make build`, `make test`, `make lint`, `make generate`
verdes; nenhum pixel mudou ainda — nenhuma história começou.

---

## Phase 3: User Story 1 — Ler os rótulos das sobreposições em português (Priority: P1) 🎯 MVP

**Objetivo**: os rótulos desenhados passam a ser `"DIST "`, `"ELEV "`,
`"GANHO "`, `"TEMPO "` (research.md item 1), sem mudar nenhum valor nem
unidade.

**Teste Independente**: `quickstart.md` item 1.

- [X] T004 [P] [US1] Escrever em `internal/domain/frame_screen_overlay_test.go`: `distanceBlockText(frame)` devolve `"DIST " + formatOverlayDistance(frame.MarkerDistance)` para alguns quadros conhecidos; `elevationBlockText(frame)` devolve `"ELEV " + formatOverlayElevation(...) + "   GANHO " + formatOverlayGain(...)`; `timeBlockText(frame)` devolve `"TEMPO " + formatOverlayElapsed(...)`; e um cenário de integração — desenhar um quadro com os três blocos ligados e confirmar que o resultado **difere** de uma referência construída à mão com os rótulos antigos (`"GAIN "`/`"TIME "`, via chamadas diretas a `screenOverlay.drawLine`/`drawText` com esses literais) na mesma posição e largura — prova de que o inglês não aparece mais
- [X] T005 [US1] Implementar em `internal/domain/frame_screen_overlay.go`: extrair `distanceBlockText(frame CameraFrame) string`, `elevationBlockText(frame CameraFrame) string`, `timeBlockText(frame CameraFrame) string`, com os rótulos em português; `draw` passa a chamar as três funções no lugar de montar os textos inline (a chamada a `numericPanelWidth` com os três textos retornados continua igual — a largura, por si, só muda na US4); `make test` verde
- [X] T006 [US1] Validação manual: `quickstart.md` item 1 (rótulos em português, valores e unidades inalterados) — confirmado com o binário real: `DIST`, `ELEV ... GANHO`, `TEMPO`, mesmos valores e unidades de antes

**Checkpoint**: nenhuma palavra em inglês é desenhada — MVP entregue.

---

## Phase 4: User Story 2 — Ler o texto sem as letras se fecharem por trás do contorno (Priority: P2)

**Objetivo**: o raio de dilatação do contorno passa a ser uma fração do
tamanho da letra (`ppem`), não da altura do quadro (research.md item 2).

**Teste Independente**: `quickstart.md` item 2.

- [X] T007 [P] [US2] Escrever `Test_OutlineRadius` em `internal/domain/frame_screen_overlay_test.go`: `outlineRadius(ppem)` é `OverlayOutlineMinWidth` quando `OverlayOutlineRatio × ppem` arredonda abaixo do piso; é `round(OverlayOutlineRatio × ppem)` quando isso fica acima do piso; e escala proporcionalmente entre dois `ppem` diferentes, ambos acima do piso (`outlineRadius(2×ppem) == 2×outlineRadius(ppem)`, nos mesmos moldes de `Test_ProfileMarkerRadius` da etapa anterior)
- [X] T008 [US2] Implementar `outlineRadius(ppem int) int` em `internal/domain/frame_screen_overlay.go` (`max(int(OverlayOutlineMinWidth), roundHalfUp(OverlayOutlineRatio*float64(ppem)))`); `drawText` o chama com o `ppem` que já recebe como parâmetro, no lugar do cálculo a partir de `s.image.Resolution.Height`; em `internal/domain/render_tuning.go`, `OverlayOutlineRatio` de `0.0025` para `0.035` e o comentário atualizado (fração de `ppem`, não mais da altura do quadro); `make test` verde
- [X] T009 [US2] Validação manual: `quickstart.md` item 2 (vãos do "6"/"8"/"0" abertos, halos de letras vizinhas sem se tocar) — confirmado com o binário real: o vão do "0" em "DIST 0 m"/"TEMPO 0:00:00" fica claramente aberto

**Checkpoint**: o contorno é fino o bastante para só destacar a letra do fundo, em qualquer resolução.

---

## Phase 5: User Story 3 — Ler o texto com contraste sobre uma imagem de satélite clara (Priority: P3)

**Objetivo**: a fonte embutida passa de peso regular (`Go Regular`) para
peso forte (`Go Bold`), mesma família, mesma licença, nenhuma dependência
nova (research.md item 3).

**Teste Independente**: `quickstart.md` item 3.

- [X] T010 [P] [US3] Escrever em `internal/domain/vector_font_test.go`: a soma de cobertura do glifo `'0'` num `ppem` conhecido, rasterizado por `newVectorFace()` (a fonte de produção), é **maior** que a soma de cobertura do mesmo glifo rasterizado por uma `vectorFace` construída no próprio teste a partir de `golang.org/x/image/font/gofont/goregular.TTF` (importado só no teste) — prova de que a fonte embutida é mais encorpada que o peso regular, sem comparar bytes de fonte diretamente
- [X] T011 [US3] Implementar em `internal/domain/vector_font.go`: `newVectorFace` passa a chamar `sfnt.Parse(gobold.TTF)` (`golang.org/x/image/font/gofont/gobold`) no lugar de `sfnt.Parse(goregular.TTF)`; atualizar o comentário do tipo `vectorFace`; `make test` verde
- [X] T012 [US3] Validação manual: `quickstart.md` item 3 (traços mais encorpados sobre fundo claro; nenhuma dependência nova em `go list -m all`) — confirmado com o binário real: texto visivelmente mais encorpado; `golang.org/x/image v0.46.0` continua a mesma versão

**Checkpoint**: o texto tem peso suficiente para se destacar sobre uma imagem de satélite clara.

---

## Phase 6: User Story 4 — Ver os painéis numéricos com largura estável do início ao fim do voo (Priority: P4)

**Objetivo**: a largura compartilhada dos painéis numéricos passa a ser
calculada uma vez por `Scene`, a partir de todos os quadros do plano, não
mais recalculada a cada quadro (research.md item 4).

**Teste Independente**: `quickstart.md` item 4.

- [X] T013 [P] [US4] Escrever em `internal/domain/frame_screen_overlay_test.go`: `stablePanelWidth(face, plan, config, ppem)` devolve a largura do texto mais largo entre `distanceBlockText`/`elevationBlockText`/`timeBlockText` de **qualquer** quadro do plano — construir um plano com um quadro de texto curto (ex.: distância `0`) e outro de texto bem mais longo (ex.: distância de 5 dígitos), e confirmar que o resultado é a largura calculada sobre o quadro mais longo, mesmo chamando a função só com o índice do quadro curto em mente (a função não recebe índice — sempre varre tudo); devolve `0` quando nenhum bloco numérico aparece (`config` desligado ou sem dado disponível); ignora um bloco que `config`/o plano não mostram, sem deixar que ele influencie o resultado
- [X] T014 [US4] Implementar `stablePanelWidth(face *vectorFace, plan CameraPlan, config OverlayConfig, ppem int) int` e `overlayPpem(height int) int` (extraída do cálculo que `draw` já fazia inline) em `internal/domain/frame_screen_overlay.go`; `screenOverlay` ganha o campo `panelWidth int`; `draw` usa `s.panelWidth` diretamente nas três chamadas de `drawLine`, sem calcular largura nenhuma; remover a função antiga `numericPanelWidth` (a que calculava só a partir de um quadro); `make test` verde
- [X] T015 [P] [US4] Escrever em `internal/domain/frame_scene_test.go` (ajustado durante a implementação: como o arquivo é `package domain_test`, não dá para chamar `Scene.numericPanelWidth` nem inspecionar o cache diretamente — o teste prova o mesmo comportamento de fora, por `Render`: o quadro curto desenhado sozinho em seu próprio plano produz um painel mais estreito que o mesmo quadro desenhado num plano que também tem um quadro de texto bem mais longo, isolado do bloco de perfil — que já depende do plano inteiro desde a etapa 9 por outro motivo, e confundiria a comparação): `Scene.numericPanelWidth(plan, height, ppem)` calcula a largura só na primeira chamada e devolve o valor em cache numa segunda chamada com a mesma `height` (sem depender de medir tempo — conferir que o resultado bate e, se possível, que uma segunda chamada com um `plan` diferente mas a mesma `height` ainda devolve o valor cacheado, documentando esse limite conhecido); recalcula quando `height` muda; um quadro de texto curto e um de texto longo do mesmo plano, desenhados pela mesma `Scene`, têm painéis da mesma largura (medição de pixel, mesma técnica das etapas anteriores — localizar a borda do painel a partir de `OverlayTopMarginRatio`/`OverlaySideMarginRatio`)
- [X] T016 [US4] Implementar em `internal/domain/frame_scene.go`: `Scene` ganha os campos `panelWidth int`, `panelWidthHeight int`, `panelWidthSet bool`, e o método `numericPanelWidth(plan CameraPlan, height, ppem int) int` (calcula `stablePanelWidth` só quando `!panelWidthSet || panelWidthHeight != height`); `Render` calcula `ppem := overlayPpem(resolution.Height)` e `panelWidth := s.numericPanelWidth(plan, resolution.Height, ppem)` antes de montar `screenOverlay{...}`, passando `panelWidth: panelWidth`; `make test` verde
- [X] T017 [US4] Validação manual: `quickstart.md` item 4 (largura idêntica entre o primeiro e o último quadro do voo; quadro isolado idêntico, byte a byte, ao mesmo quadro dentro do voo inteiro) — confirmado com o binário real: painel `ELEV...GANHO` termina na mesma coluna em `DIST 0 m` e em `DIST 19.9 km`; `cmp` confirmou IDÊNTICOS entre `render frame` e `render all`

**Checkpoint**: a largura dos painéis numéricos nunca muda dentro de um mesmo voo, nem entre `render frame` e `render all`.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Propósito**: os casos que não pertencem a nenhuma história específica, a documentação e a verificação final.

- [ ] T018 [P] Validação manual: `quickstart.md` itens 5, 6 e 7 (determinismo byte a byte preservado entre execuções; conjunto de quadros de antes desta etapa recusado por `render all` sem `--overwrite`; `--overlays=false` continua sem desenhar nada)
- [ ] T019 [P] Atualizar `specs/005-frame-rendering/contracts/frame-files.md` com a nota desta etapa (tabela "Cores e padrões" da sobreposição de tela — idioma, peso da fonte, espessura do contorno, estabilidade da largura; `RenderVersion` de 3 para 4) — mesmo padrão que as etapas 6 e 11 já aplicaram (`contracts/frame-files-change.md` desta etapa já descreve o conteúdo exato da nota)
- [ ] T020 [P] Atualizar `CLAUDE.md`: a décima segunda etapa (resumo do projeto, logo após a nota da décima primeira), os rótulos em português, o contorno derivado de `ppem`, a fonte `Go Bold`, a largura estável dos painéis calculada por `Scene`, `RenderVersion` 4
- [ ] T021 Verificação final: `make build`, `make test`, `make lint`, `make generate` verdes (confirmar que `make generate` não produz nenhuma diferença); `gofmt -l .` limpo; `go mod tidy` sem diferença; reexecutar `quickstart.md` por inteiro

---

## Dependências e Ordem de Execução

### Dependências entre fases

- **Phase 1 (Setup)**: sem dependências.
- **Phase 2 (Foundational)**: depende da Phase 1; **bloqueia todas as histórias** (a versão do desenho precisa já estar em 4 antes de qualquer pixel mudar).
- **Phases 3 a 6 (US1 a US4)**: dependem da Phase 2 concluída. US1, US2 e US3 são independentes entre si (tocam partes diferentes dos mesmos arquivos). **US4 depende da US1** — reaproveita `distanceBlockText`/`elevationBlockText`/`timeBlockText`, que só existem a partir dela. A ordem de implementação segue a prioridade (P1 → P4) tanto por disciplina de revisão quanto, no caso de US4, por essa dependência real.
- **Phase 7 (Polish)**: depende das histórias desejadas.

### Dependências entre histórias

- **US1 (P1)**: depende só da Foundational.
- **US2 (P2)**: depende só da Foundational — não lê nem escreve nada que a US1 toca (`outlineRadius`/`drawText` vs. os textos dos blocos).
- **US3 (P3)**: depende só da Foundational — arquivo totalmente diferente (`vector_font.go`).
- **US4 (P4)**: depende da Foundational **e da US1** (usa as três funções de texto que a US1 extrai, já com os rótulos em português).

### Dentro de cada história

- Teste antes da implementação (deve falhar primeiro).
- A história é completa antes da próxima prioridade.

### Oportunidades de paralelismo

```text
# Foundational:
T003 render_tuning.go + frame_set_test.go

# US1:
T004 frame_screen_overlay_test.go → T005 frame_screen_overlay.go → T006 manual

# US2 (pode rodar em paralelo com US1 — arquivos de implementação diferentes
# dentro do mesmo frame_screen_overlay.go, mas funções que não se tocam):
T007 frame_screen_overlay_test.go → T008 frame_screen_overlay.go + render_tuning.go → T009 manual

# US3 (genuinamente paralelo a US1/US2 — arquivo diferente):
T010 vector_font_test.go → T011 vector_font.go → T012 manual

# US4 (depende de T005 estar pronto):
T013 frame_screen_overlay_test.go → T014 frame_screen_overlay.go
T015 frame_scene_test.go → T016 frame_scene.go
→ T017 manual
```

---

## Estratégia de Implementação

### MVP Primeiro (Phase 1 + 2 + User Story 1)

1. Phase 1 (branch e linha de base) e Phase 2 (`RenderVersion` 4).
2. Phase 3 (US1): rótulos em português.
3. **PARE e valide**: `quickstart.md` item 1 — nenhuma palavra em inglês — antes de acrescentar as demais histórias.

### Entrega Incremental

1. Setup + Foundational → versão do desenho pronta para a mudança, nenhum pixel alterado ainda.
2. + US1 → rótulos em português (MVP) → item 1.
3. + US2 → contorno fino, proporcional à letra → item 2.
4. + US3 → fonte de peso forte → item 3.
5. + US4 → largura dos painéis estável → item 4.
6. Polish → determinismo confirmado, versões antigas recusadas, `--overlays=false` inalterado, documentação, verificação final → itens 5–7.

Cada história agrega valor sem quebrar as anteriores; US2 e US3 poderiam,
em princípio, ser entregues em qualquer ordem entre si (são ortogonais),
mas US4 só faz sentido depois da US1.

---

## Notas

- `[P]` = arquivos diferentes, sem dependência de tarefa incompleta (ver a
  ressalva sobre os arquivos de teste compartilhados no preâmbulo).
- Nenhum valor exibido, nenhum bloco, nenhuma configuração de sobreposição,
  nenhum enquadramento, terreno, traçado, margem de segurança ou tamanho
  do marcador do perfil nasce ou muda nesta etapa (FR-009): toda tarefa de
  implementação só muda **como** o texto é escrito e dimensionado — se uma
  tarefa parecer exigir mudar um valor exibido, um formato de número ou
  uma unidade, algo está errado.
- O rasterizador (`rasterizeOutline`, `windingNumber`, `flattenQuad`/
  `flattenCube`) e a extração de contornos (`sfnt.LoadGlyph`) não mudam
  nesta etapa (FR-012) — só os bytes da fonte que entram em `sfnt.Parse`
  (US3) e os parâmetros que `drawText`/`stablePanelWidth` calculam antes
  de chamar o rasterizador.
- Faça commit após cada tarefa ou grupo lógico (só quando o usuário pedir).
- Pare em qualquer checkpoint para validar a história com o `quickstart.md`.
- Evite: tarefas vagas, duas tarefas `[P]` de implementação editando o
  mesmo arquivo, pular a verificação de `make test` entre uma tarefa de
  teste e a de implementação que a segue.
