---

description: "Task list for feature implementation"
---

# Tarefas: Acabamento das Sobreposições de Tela

**Entrada**: Documentos de design de `/specs/011-overlay-polish/`

**Pré-requisitos**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/frame-files-change.md`, `quickstart.md`

**Testes**: incluídos. A constituição do projeto (Princípio VI — Testes
Automatizados no Núcleo; Princípio X — Given/When/Then, builders e
isolamento por camada) exige testify e proíbe testes tabulares: cada
cenário é um `t.Run("should ...")` com `// given`, `// when`, `// then`.
**Ressalva**: a Phase 2 substitui a fonte de bitmap por uma fonte vetorial
dentro de `internal/domain/frame_screen_overlay.go` e
`internal/domain/frame_scene.go` — isto não acrescenta nada a um arquivo
vazio, **reescreve** testes existentes que hoje comparam pixels contra a
máscara de `inconsolata` (`Test_DrawText`, em
`frame_screen_overlay_test.go`); nesses casos, "escrever o teste" significa
substituir o cenário antigo pelo equivalente novo, não apenas acrescentar.

**Organização**: as tarefas são agrupadas por história de usuário (P1–P4
de `spec.md`). Diferente de etapas anteriores, **toda** a mudança fica
dentro de `internal/domain` — nenhuma porta, serviço, flag de CLI ou
arquivo de configuração é tocado (FR-010/FR-011 do `spec.md`): nada aqui é
uma escolha nova do usuário. A Phase 2 (Foundational) é por isso a maior
parte do trabalho: o rasterizador de fonte vetorial determinístico em si
(Parte A), as constantes fixas novas (Parte B) e a ligação dele ao desenho
já existente (Parte C) — o que, por si só, já entrega texto suave em todos
os blocos (FR-001/002/003), mas ainda sem contorno, sem largura
compartilhada, sem marcador do perfil maior e sem margem por borda. Cada
história de usuário (P1–P4) acrescenta exatamente um desses quatro
acabamentos, de forma independente.

**Divisão do trabalho entre histórias** (o mesmo arquivo,
`frame_screen_overlay.go`, é estendido por mais de uma história — nunca em
paralelo):

| História | O que entrega | FR |
|---|---|---|
| US1 (P1) | contorno escuro no texto — legível sobre qualquer fundo sem aumentar a opacidade do painel | FR-004 |
| US2 (P2) | os três painéis numéricos (distância; elevação+ganho; tempo decorrido) passam a ter a mesma largura | FR-005 |
| US3 (P3) | o marcador do perfil de elevação fica proporcional à altura do quadro, com piso em pixels | FR-006 |
| US4 (P4) | a margem de segurança passa a ser maior na borda inferior que nas demais | FR-007 |

Enquanto nenhuma história existe, o texto já sai suavizado (Foundational),
mas sem contorno, com painéis de larguras diferentes, com o marcador do
perfil do tamanho de hoje, e com a mesma margem única nas quatro bordas.

**Nenhum mock precisa ser regenerado nesta etapa**: nenhuma porta nem
interface de serviço é tocada — tudo fica em tipos e funções não exportados
de `internal/domain`. `make generate` não deveria produzir nenhuma
diferença (verificado em T026).

## Formato: `[ID] [P?] [Story] Descrição`

- **[P]**: pode ser executado em paralelo (arquivos diferentes, sem
  dependência de tarefa incompleta). Tarefas que editam o mesmo arquivo
  NUNCA são marcadas `[P]` entre si, mesmo quando logicamente independentes.
- **[Story]**: a qual história de usuário esta tarefa pertence (US1 a US4).
  Tarefas de Setup, Foundational e Polish não têm esse rótulo.
- Toda tarefa inclui o caminho de arquivo exato a criar/editar.
- Comentários de código, identificadores, mensagens de commit, e mensagens
  de erro em tempo de execução: **inglês**; artefatos do Spec Kit:
  português (constituição, "Idioma dos Artefatos").

## Convenções de Caminho

Mesmo projeto único em Go, mesma estrutura hexagonal de `plan.md`: toda a
mudança fica em `internal/domain/`. Regras de estilo: receivers curtos e
consistentes com o tipo (`s` `screenOverlay`, `f` `vectorFace`); `new(x)`
do Go 1.26 em vez de um helper `ptr`; nome exportado nunca repete o pacote;
receiver sem uso fica sem nome; `//go:generate` não se aplica aqui (nenhuma
interface nova).

---

## Phase 1: Setup (Shared Infrastructure)

**Propósito**: um ponto de partida próprio (branch) e verde.

- [X] T001 Criar e mudar para o branch `011-overlay-polish` a partir de `main` (`git checkout -b 011-overlay-polish`)
- [X] T002 Confirmar `make build`, `make test`, `make lint` e `make generate` verdes antes de qualquer mudança (linha de base)

**Checkpoint**: repositório pronto para a Phase 2.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Propósito**: o rasterizador de fonte vetorial determinístico e sua
ligação ao desenho da sobreposição já existente. **Nenhuma história
começa antes desta fase estar verde.**

### Parte A — o rasterizador vetorial determinístico (research.md itens 1, 3, 4; data-model.md)

- [X] T003 [P] Escrever `internal/domain/vector_font_test.go` (novo arquivo): achatar um segmento `QuadTo`/`CubeTo` sintético (pontos de controle conhecidos) produz exatamente o número fixo de subdivisões documentado (8 para quadrática, 12 para cúbica), com os pontos extremos coincidindo com o início/fim da curva original; rasterizar um contorno sintético (ex.: um retângulo) sobre uma grade de pixels dá cobertura máxima num pixel inteiramente dentro, zero num pixel inteiramente fora, e uma fração conhecida (calculada a mão para a grade de superamostragem 4×4 documentada) num pixel que uma aresta reta atravessa pela metade; rasterizar o mesmo contorno duas vezes produz exatamente a mesma cobertura (determinismo); `glyph(r, ppem)` devolve `ok=false` para um rune fora da fonte, sem pânico, e máscaras de tamanhos diferentes para `ppem` diferentes do mesmo rune; uma segunda chamada de `glyph` com o mesmo `(rune, ppem)` não aumenta o cache (`len(face.cache)` continua `1`); `textWidth` soma o avanço de cada rune no `ppem` pedido
- [X] T004 Implementar `internal/domain/vector_font.go` (novo arquivo): `vectorFace{font *sfnt.Font, buf sfnt.Buffer, cache map[glyphKey]glyphMask}`, `newVectorFace()` (`sfnt.Parse(goregular.TTF)` uma vez; `panic` se falhar — dado estático embutido, uma falha seria bug de build, nunca situação de runtime a tratar); `glyph(r rune, ppem int) (glyphMask, bool)` — contornos via `(*sfnt.Font).LoadGlyph`, achatamento fixo de curvas e rasterização por superamostragem 4×4 com regra `nonzero winding` (só `+ − × ÷`, `Min`, `Max`, `Abs`, comparação — nenhuma função transcendental, nenhuma assembly por arquitetura), cache por `(rune, ppem)`; `textWidth(text string, ppem int) int`; `make test` verde

### Parte B — constantes fixas novas e `RenderVersion` (data-model.md)

- [X] T005 [P] Em `internal/domain/render_tuning.go`: `RenderVersion` de `2` para `3`; adicionar, junto das demais marcas fixas ("significado, não estilo"), sem remover `OverlayMarginRatio` ainda (continua em uso até a US4 trocar os chamadores): `OverlayTopMarginRatio = 0.06`, `OverlaySideMarginRatio = 0.06`, `OverlayBottomMarginRatio = 0.14`, `OverlayTextOutlineColor = RGB{0x10,0x10,0x10}`, `OverlayOutlineRatio`, `OverlayOutlineMinWidth`, `ProfileMarkerRadiusRatio`, `ProfileMarkerMinRadius` (valores por razão de altura/largura com piso em pixels, mesma categoria de `TrailMinWidth`/`MarkerMinRadius`); atualizar `internal/domain/frame_set_test.go` (`assert.Equal(t, 2, domain.RenderVersion)` → `3`, único lugar que fixa o valor hoje); `make test` verde

### Parte C — ligar o rasterizador ao desenho da sobreposição (research.md itens 1, 3, 4, 5; data-model.md)

- [X] T006 [P] Estender `internal/domain/frame_screen_overlay_test.go`: substituir os cenários de `Test_DrawText` que comparam contra a máscara de `inconsolata` ("should light up exactly the glyph's own mask...", "should replicate each glyph pixel into a scale x scale block") por equivalentes sobre a fonte vetorial — um texto curto desenhado sobre um fundo uniforme só altera pixels dentro do retângulo esperado do texto, nunca os "far from the text"; desenhar o mesmo texto duas vezes produz a mesma imagem byte a byte; um rune fora da fonte não desenha nada, sem pânico; remover o helper `alphaAt` e a dependência de `golang.org/x/image/font/inconsolata`/`basicfont`; atualizar todos os literais `screenOverlay{...}` existentes em `Test_ScreenOverlay_Draw` para incluir um `face *vectorFace` real (`newVectorFace()`, reaproveitado entre subtestes) — sem ele, `face: nil` causaria pânico em qualquer configuração ligada que tente medir ou desenhar texto
- [X] T007 Implementar em `internal/domain/frame_screen_overlay.go`: `screenOverlay` ganha o campo `face *vectorFace`; `drawText` deixa de ser função livre presa à variável de pacote `overlayFace` e passa a desenhar a partir de `s.face.glyph(r, ppem)` (mistura alfa pela cobertura, reaproveitando `blendPixel`); o tamanho do glifo em pixels (`ppem`) é calculado a partir de `glyphHeightRatio * altura` (sem mais o fator de escala inteiro `textScale`); a medição de largura e de altura de linha usa `s.face.textWidth`/`s.face.lineHeight` no lugar da conta de monoespaçamento; como `scale` deixa de existir, o raio do marcador do perfil e a espessura de sua linha em `drawProfile`/`drawDot` passam, só nesta tarefa, por uma fórmula provisória equivalente em `ppem` (`ppem/16` no lugar do antigo `scale`, preservando o tamanho de hoje) — a fórmula final por razão/piso (`ProfileMarkerRadiusRatio`/`ProfileMarkerMinRadius`) é da US3 (T017), não desta tarefa; remover `overlayFace`, `textScale`, `glyphOffset` e os imports de `basicfont`/`inconsolata`; `make test` verde
- [X] T008 [P] Estender `internal/domain/frame_scene_test.go`: um quadro com sobreposição ligada, desenhado duas vezes a partir da mesma `Scene`, produz a mesma imagem byte a byte (reforça o determinismo com o cache de glifos reaproveitado entre quadros); confirmar que o cenário "should draw the image the reference says" continua com a sobreposição desligada (`roughScene`) e por isso não precisa de nova constante de hash
- [X] T009 Implementar em `internal/domain/frame_scene.go`: `Scene` ganha o campo `face *vectorFace`, preenchido em `NewScene` (`face: newVectorFace()`, sem parâmetro novo na assinatura — não é escolha do usuário); `Render` passa `s.face` ao construir `screenOverlay{...}`; `make test` verde

**Checkpoint**: `make build`, `make test`, `make lint`, `make generate`
verdes; o texto das sobreposições já sai suavizado em qualquer bloco
(FR-001/002/003), mas ainda sem contorno, sem largura compartilhada, com o
marcador do perfil do tamanho de antes e com a mesma margem única nas
quatro bordas — nenhuma história começou ainda.

---

## Phase 3: User Story 1 — Ler o texto sem serrilhado e sobre qualquer fundo (Priority: P1) 🎯 MVP

**Objetivo**: o texto ganha um contorno escuro fixo, legível sobre
qualquer fundo sem depender da opacidade do painel (FR-004) — a
suavização em si (FR-001/002/003) já saiu pronta da Foundational.

**Teste Independente**: `quickstart.md` item 1.

- [X] T010 [P] [US1] Estender `internal/domain/frame_screen_overlay_test.go`: desenhar um texto sobre um fundo exatamente da cor `OverlayTextColor` ainda produz, ao redor de cada glifo, pixels do contorno visivelmente diferentes de `OverlayTextColor` (prova de que o contorno nunca depende da cor por baixo); nenhum pixel do contorno fica a mais de `max(OverlayOutlineMinWidth, OverlayOutlineRatio*altura)` pixels do glifo original; desenhar o mesmo texto duas vezes produz a mesma imagem byte a byte
- [X] T011 [US1] Implementar em `internal/domain/frame_screen_overlay.go`: `drawText` desenha, para cada glifo, a máscara dilatada (máximo de cobertura numa vizinhança de raio `max(OverlayOutlineMinWidth, OverlayOutlineRatio*altura)`, só `Max` e comparação) em `OverlayTextOutlineColor`, **antes** da máscara original em `OverlayTextColor` por cima — mesma técnica de `TrailCasingColor`/`MarkerRingColor` (research.md item 6); `make test` verde
- [X] T012 [US1] Validação manual: `quickstart.md` item 1 (texto suave e legível sobre fundo claro e escuro) — confirmado com o binário real: texto vetorial suave, com contorno visível, legível sobre o terreno nos dois fundos

**Checkpoint**: texto suave, com contorno, legível sobre qualquer fundo — MVP entregue.

---

## Phase 4: User Story 2 — Ver uma pilha de blocos numéricos alinhada (Priority: P2)

**Objetivo**: os três painéis numéricos presentes num quadro (distância;
elevação+ganho; tempo decorrido) compartilham a largura do mais largo
(FR-005), mantendo ordem e espaçamento de hoje.

**Teste Independente**: `quickstart.md` item 2.

- [X] T013 [P] [US2] Estender `internal/domain/frame_screen_overlay_test.go`: com os três blocos numéricos presentes e textos de larguras bem diferentes entre si (ex.: forçar valores de distância/tempo com dígitos a mais), os três painéis desenhados (medidos pela extensão da placa, não do texto) têm exatamente a mesma largura — igual à do bloco que precisaria da maior largura; com só um ou dois blocos numéricos presentes (o terceiro desligado ou sem dado), os presentes continuam com a mesma largura entre si, sem referência ao ausente; a ordem e o espaçamento vertical entre blocos (`y += lineHeight + pad`) não mudam
- [X] T014 [US2] Implementar em `internal/domain/frame_screen_overlay.go`: `draw` mede a largura de cada bloco numérico presente via `s.face.textWidth` antes de desenhar qualquer painel, toma o maior valor, e passa essa largura a `drawLine` para os três painéis presentes (research.md item 7); `make test` verde
- [X] T015 [US2] Validação manual: `quickstart.md` item 2 (painéis numéricos alinhados, mesmo quando o texto cresce ao longo do voo) — confirmado com o binário real: os três painéis terminam exatamente na mesma borda direita, no início e perto do fim do voo

**Checkpoint**: os painéis numéricos formam uma pilha de largura uniforme.

---

## Phase 5: User Story 3 — Enxergar o marcador do perfil de elevação num celular (Priority: P3)

**Objetivo**: o marcador do perfil de elevação passa a ter raio
proporcional à altura do quadro, com piso em pixels (FR-006), em vez de
ligado à escala do texto.

**Teste Independente**: `quickstart.md` item 3.

- [X] T016 [P] [US3] Estender `internal/domain/frame_screen_overlay_test.go`: numa resolução grande, o raio do marcador do perfil é `ProfileMarkerRadiusRatio * altura` (acima do piso); numa resolução muito pequena (onde essa conta ficaria abaixo do piso), o raio nunca é menor que `ProfileMarkerMinRadius`; em duas resoluções de mesma proporção (uma acima do piso nas duas), o raio escala na mesma fração visual nas duas
- [X] T017 [US3] Implementar em `internal/domain/frame_screen_overlay.go`: `drawProfile`/`drawDot` usam `max(ProfileMarkerMinRadius, ProfileMarkerRadiusRatio * altura)` no lugar da fórmula provisória baseada em `ppem` que a Foundational usou só para manter a compilação depois de `scale` deixar de existir (research.md item 8); `make test` verde
- [X] T018 [US3] Validação manual: `quickstart.md` item 3 (marcador do perfil visível na menor resolução aceita) — confirmado com o binário real em 180x320 e 1080x1920

**Checkpoint**: o marcador do perfil é visível em qualquer resolução aceita.

---

## Phase 6: User Story 4 — Publicar o vídeo numa rede social sem perder sobreposições na borda (Priority: P4)

**Objetivo**: a margem de segurança passa a ser maior na borda inferior
que nas demais (FR-007), em frações da altura e da largura.

**Teste Independente**: `quickstart.md` item 4.

- [X] T019 [P] [US4] Estender `internal/domain/frame_screen_overlay_test.go`: nenhum bloco (texto, painel ou perfil) desenha a menos de `OverlayBottomMarginRatio * altura` da borda inferior; nenhum desenha a menos de `OverlayTopMarginRatio * altura` do topo, nem a menos de `OverlaySideMarginRatio * largura` de qualquer lateral; a margem inferior resultante é estritamente maior que as outras três, nas mesmas dimensões de quadro
- [X] T020 [US4] Implementar em `internal/domain/frame_screen_overlay.go`: `draw`/`drawProfile` calculam margens separadas por borda (`OverlayTopMarginRatio`, `OverlaySideMarginRatio`, `OverlayBottomMarginRatio`) no lugar do único `margin` de hoje; remover `OverlayMarginRatio` de `internal/domain/render_tuning.go`, já sem nenhum chamador (research.md item 9); `make test` verde
- [X] T021 [US4] Validação manual: `quickstart.md` item 4 (margem maior na base em vídeo vertical) — confirmado com o binário real: a margem inferior do painel de perfil é visivelmente maior que a margem superior/lateral dos demais blocos

**Checkpoint**: nenhuma sobreposição invade a margem de segurança documentada, maior na base, em nenhuma resolução aceita.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Propósito**: os casos que não pertencem a nenhuma história específica, a documentação e a verificação final.

- [X] T022 [P] Validação manual: `quickstart.md` itens 5, 6, 7 e 8 (determinismo byte a byte preservado entre execuções/arquiteturas; `--overlays=false` continua inalterado; um conjunto de quadros de antes desta etapa é recusado como de outro conjunto, por `render all` e por `fly --keep`)
- [X] T023 [P] Atualizar `specs/005-frame-rendering/contracts/frame-files.md` com a nota desta etapa (tabela "Cores e padrões" da sobreposição de tela; `RenderVersion` de 2 para 3) — mesmo padrão que a etapa 6 já aplicou para a mudança dela (`contracts/frame-files-change.md` desta etapa já descreve o conteúdo exato da nota)
- [X] T024 [P] Atualizar `CLAUDE.md`: a décima primeira etapa (resumo do projeto), a fonte vetorial embutida e o rasterizador próprio, o contorno do texto, a largura compartilhada dos painéis, o marcador do perfil proporcional, a margem por borda, `RenderVersion` 3
- [X] T025 Verificação final: `make build`, `make test`, `make lint`, `make generate` verdes (confirmado: `make generate` não produziu nenhuma diferença de mock); `gofmt -l .` limpo; `go mod tidy` sem diferença; `quickstart.md` reexecutado item a item ao longo da implementação (T012, T015, T018, T021, T022), todos confirmados com o binário real

---

## Dependências e Ordem de Execução

### Dependências entre fases

- **Phase 1 (Setup)**: sem dependências.
- **Phase 2 (Foundational)**: depende da Phase 1; **bloqueia todas as histórias**. Parte A (o rasterizador em si, sem nenhuma ligação ao desenho) é independente das demais. Parte B (constantes) é independente de A. Parte C depende de A (usa `vectorFace`) e de B (usa as constantes de margem/contorno/marcador só a partir da história que as liga — mas a troca de fonte em si só precisa de A).
- **Phases 3 a 6 (US1 a US4)**: dependem da Phase 2 concluída; todas editam `frame_screen_overlay.go`, então as edições de implementação (T011, T014, T017, T020) seguem a ordem P1 → P4, cada uma depois da anterior estar compilando. As quatro são, por outro lado, **independentes entre si** quanto à lógica: nenhuma lê o resultado de outra (contorno, largura compartilhada, raio do marcador e margem são quatro cálculos ortogonais dentro do mesmo arquivo).
- **Phase 7 (Polish)**: depende das histórias desejadas.

### Dependências entre histórias

- **US1 (P1)**: depende só da Foundational.
- **US2 (P2)**: depende só da Foundational (não do US1 — o contorno e a largura compartilhada não interagem); editar depois do US1 só por estarem no mesmo arquivo.
- **US3 (P3)**: depende só da Foundational, pela mesma razão.
- **US4 (P4)**: depende só da Foundational, pela mesma razão.

### Dentro de cada história

- Teste antes da implementação (deve falhar primeiro, exceto onde a ressalva do preâmbulo se aplica).
- A história é completa antes da próxima prioridade (por disciplina de revisão, não por dependência lógica real).

### Oportunidades de paralelismo

```text
# Foundational, Partes A e B, em paralelo entre si:
T003 vector_font_test.go (novo arquivo)   T005 render_tuning.go + frame_set_test.go
→ T004 vector_font.go

# Foundational, Parte C (sequencial, depende de A completa; B só importa a partir das histórias):
T006 frame_screen_overlay_test.go → T007 frame_screen_overlay.go
T008 frame_scene_test.go → T009 frame_scene.go (pode rodar em paralelo com T006/T007 — arquivos diferentes — mas T009 depende de T007 se screenOverlay{...} mudar de forma; na prática, fazer T006–T007 antes de T008–T009 evita retrabalho)

# US1 a US4: cada teste [P] entre si (mesmo arquivo de teste, mas tarefas
# diferentes — na prática, rodar uma história por vez, pela ordem de
# prioridade, evita conflito de edição concorrente no mesmo arquivo)
T010 → T011 → T012
T013 → T014 → T015
T016 → T017 → T018
T019 → T020 → T021
```

---

## Estratégia de Implementação

### MVP Primeiro (Phase 1 + 2 + User Story 1)

1. Phase 1 (branch e linha de base) e Phase 2 (o rasterizador vetorial determinístico; as constantes fixas novas e `RenderVersion` 3; a ligação ao desenho já existente — texto suave em todo bloco).
2. Phase 3 (US1): contorno escuro no texto.
3. **PARE e valide**: `quickstart.md` item 1 — texto suave e legível sobre qualquer fundo — antes de acrescentar as demais histórias.

### Entrega Incremental

1. Setup + Foundational → texto suavizado em todo bloco, sem nenhum dos quatro acabamentos de história ainda.
2. + US1 → texto com contorno, legível sobre qualquer fundo (MVP) → item 1.
3. + US2 → painéis numéricos de largura uniforme → item 2.
4. + US3 → marcador do perfil visível em qualquer resolução → item 3.
5. + US4 → margem maior na base, adequada ao vídeo vertical → item 4.
6. Polish → determinismo confirmado, versões antigas recusadas, documentação, verificação final → itens 5–8.

Cada história agrega valor sem quebrar as anteriores; como as quatro são
ortogonais entre si (nenhuma depende do resultado de outra), a ordem de
entrega poderia, em princípio, ser qualquer uma — a ordem P1→P4 é a de
prioridade do `spec.md`, não uma dependência técnica.

---

## Notas

- `[P]` = arquivos diferentes, sem dependência de tarefa incompleta.
- Nenhum valor exibido, nenhum bloco, nenhuma configuração de sobreposição,
  nenhum enquadramento, terreno, traçado ou duração de vídeo nasce ou muda
  nesta etapa (FR-010): toda tarefa de implementação só muda **como** algo
  já desenhado é desenhado — se uma tarefa parecer exigir ler de novo o
  trajeto GPS, os dados geográficos registrados, ou mudar um valor exibido,
  algo está errado.
- Nunca usar `golang.org/x/image/vector.Rasterizer` nem nenhuma outra
  função com implementação em assembly por arquitetura (research.md item
  1) — é o motivo de existir o rasterizador próprio em `vector_font.go`.
- Faça commit após cada tarefa ou grupo lógico (só quando o usuário pedir).
- Pare em qualquer checkpoint para validar a história com o `quickstart.md`.
- Evite: tarefas vagas, duas tarefas `[P]` editando o mesmo arquivo,
  pular a verificação de `make test` entre uma tarefa de teste e a de
  implementação que a segue.
