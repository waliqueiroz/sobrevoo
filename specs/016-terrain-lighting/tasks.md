---

description: "Task list for feature implementation"
---

# Tarefas: Iluminação Direcional do Terreno

**Entrada**: Documentos de design de `/specs/016-terrain-lighting/`

**Pré-requisitos**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/frame-files-change.md`, `quickstart.md`

**Testes**: incluídos. A constituição do projeto (Princípio VI — Testes
Automatizados no Núcleo; Princípio X — Given/When/Then, builders e
isolamento por camada) exige testify e proíbe testes tabulares: cada
cenário é um `t.Run("should ...")` com `// given`, `// when`, `// then`.

**Organização**: as tarefas são agrupadas por história de usuário (P1–P3
de `spec.md`). Diferente de uma feature com histórias ortogonais, as três
histórias aqui formam **camadas sucessivas do mesmo mecanismo** — a própria
`spec.md` já descreve a História 3 como "um refinamento de qualidade sobre
o resultado da História 1". A Foundational entrega só a parte inerte (a
direção fixa da luz e a fórmula pura do fator de brilho, sem nada que a
chame ainda — nenhuma mudança visual). **US1** liga essa fórmula a uma
normal lida só do nível mais fino da grade (a própria célula do raio) —
já entrega volume real, com a ressalva conhecida de poder cintilar ao
longe. **US2** não muda o mecanismo; confirma (ou ajusta) a faixa fixa de
brilho para nunca apagar a diferença entre cores do mapa. **US3** estende
a mesma grade de US1 numa pirâmide de níveis e troca a leitura "só nível
0" por uma escolhida pela distância — corrigindo a cintilação sem mudar a
fórmula de brilho nem o ponto em que ela é aplicada.

**Divisão do trabalho entre histórias**:

| História | O que entrega | FR |
|---|---|---|
| US1 (P1) | iluminação direcional básica, lida da inclinação do nível mais fino da grade, aplicada só à cor do mapa | FR-001, FR-002, FR-005, FR-006, FR-008 (caso base: ponto isolado sem nenhum vizinho real) |
| US2 (P2) | a faixa fixa de brilho confirmada (ou ajustada) para nunca apagar a diferença entre cores comuns do mapa | FR-003, FR-004 |
| US3 (P3) | a vizinhança de amostragem passa a ser uma pirâmide de níveis, escolhida pela distância da câmera, estável e suave, subindo de nível perto de buracos | FR-007, FR-008 (subida entre níveis) |

**Atenção a arquivos compartilhados entre histórias**: `internal/domain/
frame_surface.go` e `internal/domain/frame_scene.go` são criados/editados
pelo US1 (T008, T010) e **estendidos de novo** pelo US3 (T016, T017) —
nunca em paralelo entre si, mesmo que a tarefa de teste correspondente
apareça marcada `[P]` dentro de cada fase. US2 não toca nenhum dos dois:
só `internal/domain/frame_terrain_light.go`/`render_tuning.go`.

**Nenhum mock precisa ser regenerado nesta etapa**: nenhuma porta nem
interface de serviço é tocada — tudo fica em tipos e funções não
exportados de `internal/domain`. `make generate` não deveria produzir
nenhuma diferença (verificado em T023).

## Formato: `[ID] [P?] [Story] Descrição`

- **[P]**: pode ser executado em paralelo (arquivos diferentes, sem
  dependência de tarefa incompleta). Uma tarefa de teste é marcada `[P]`
  quando pode ser escrita independentemente de outra tarefa em andamento;
  a tarefa de implementação que a segue imediatamente nunca é marcada
  `[P]`, por depender do próprio teste já escrito.
- **[Story]**: a qual história de usuário esta tarefa pertence (US1 a
  US3). Tarefas de Setup, Foundational e Polish não têm esse rótulo.
- Toda tarefa inclui o caminho de arquivo exato a criar/editar.
- Comentários de código, identificadores, mensagens de commit, e mensagens
  de erro em tempo de execução: **inglês**; artefatos do Spec Kit:
  português (constituição, "Idioma dos Artefatos").

## Convenções de Caminho

Mesmo projeto único em Go, mesma estrutura hexagonal de `plan.md`: toda a
mudança fica em `internal/domain/`. Regras de estilo: receivers curtos e
consistentes com o tipo (`s` `surface`/`screenOverlay`, `g` `placedSurface`);
nome exportado nunca repete o pacote; receiver sem uso fica sem nome;
`//go:generate` não se aplica aqui (nenhuma interface nova); toda aritmética
nova usa só os operadores já permitidos pelo desenho de quadros (`+ − × ÷`,
`Sqrt`, `Floor`, `Abs`, `Min`, `Max`, `Sin`, `Cos`, `Log2`), com todo produto
que entra numa soma envolvido em `float64(...)`.

---

## Phase 1: Setup (Shared Infrastructure)

**Propósito**: um ponto de partida próprio (branch) e verde.

- [X] T001 Criar e mudar para o branch `016-terrain-lighting` a partir de `main` (`git checkout -b 016-terrain-lighting`)
- [X] T002 Confirmar `make build`, `make test`, `make lint` e `make generate` verdes antes de qualquer mudança (linha de base)

**Checkpoint**: repositório pronto para a Phase 2.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Propósito**: a direção fixa da luz, a faixa fixa de brilho e a fórmula
pura que mapeia uma normal num fator — nada que ainda produza mudança
visual, porque nada no renderizador a chama ainda. **Nenhuma história
começa antes desta fase estar verde.**

### Parte A — constantes fixas e `RenderVersion` (data-model.md)

- [X] T003 [P] Atualizar `internal/domain/frame_set_test.go`: `assert.Equal(t, 5, domain.RenderVersion)` → `assert.Equal(t, 6, domain.RenderVersion)` (linha 290, único lugar que fixa o valor hoje)
- [X] T004 Implementar em `internal/domain/render_tuning.go`: `RenderVersion` de `5` para `6` (com comentário descrevendo a etapa, no mesmo padrão das versões anteriores já documentadas ali); acrescentar, junto das demais marcas fixas ("significado, não estilo"): `TerrainLightAzimuthDegrees = 315.0`, `TerrainLightAltitudeDegrees = 45.0`, `TerrainLightMinFactor = 0.6`, `TerrainLightMaxFactor = 1.4` (research.md itens 2, 8); `make test` verde

### Parte B — a fórmula pura do fator de brilho (research.md itens 1, 2; data-model.md)

- [X] T005 [P] Escrever `internal/domain/frame_terrain_light_test.go` (novo arquivo): `terrainLightDirection` é um vetor unitário (soma dos quadrados das três componentes igual a 1, com tolerância de ponto flutuante); sua componente vertical (`z`) é `sin(45°)` (a altura fixa documentada); `terrainLightFactor(0, 0, 1)` (normal de uma superfície plana) é exatamente `1.0`, **qualquer que seja** a altura/azimute escolhidos — testar isso construindo o fator com a fórmula direta `dot(N-up, L)` para confirmar que o termo é zero por construção quando `N = up`, não só para os valores atuais; uma normal inclinada cuja componente horizontal aponta exatamente na direção da luz dá um fator maior que `1.0`; uma inclinada na direção exatamente oposta dá um fator menor que `1.0`; para uma normal quase vertical (encosta extrema, `nz` próximo de `0`) nas duas direções, o resultado nunca sai de `[TerrainLightMinFactor, TerrainLightMaxFactor]`
- [X] T006 Implementar `internal/domain/frame_terrain_light.go` (novo arquivo): `terrainLightDirection = newTerrainLightDirection()` (`var` de pacote, calculado uma única vez a partir de `TerrainLightAzimuthDegrees`/`TerrainLightAltitudeDegrees` com `Sin`/`Cos`, nunca por pixel); `terrainLightFactor(nx, ny, nz float64) float64`, implementando `clamp(1 + (nx·Lx + ny·Ly + (nz-1)·Lz), TerrainLightMinFactor, TerrainLightMaxFactor)` com todo produto envolvido em `float64(...)`; `make test` verde

**Checkpoint**: `make build`, `make test`, `make lint`, `make generate`
verdes; a direção da luz e a fórmula do fator já existem e estão testadas
isoladamente, mas nenhum quadro desenhado muda — nenhuma história começou
ainda.

---

## Phase 3: User Story 1 — Relevo com volume real, sem nenhuma flag nova (Priority: P1) 🎯 MVP

**Objetivo**: ligar a fórmula da Foundational a uma normal lida da própria
grade de elevação (nível mais fino, sem ainda nenhuma adaptação por
distância) e aplicá-la só à cor do mapa, nunca ao traçado, ao marcador, às
sobreposições ou aos dois padrões de "sem dado" (FR-001, FR-002, FR-005,
FR-006; o caso base de FR-008).

**Teste Independente**: `quickstart.md` itens 1 e 3.

- [X] T007 [P] [US1] Escrever/estender `internal/domain/frame_surface_test.go`: uma grade sintética 3×3 construída com `builddomain.NewElevationGridBuilder().WithValues(...)` cuja altura cresce uniformemente numa direção conhecida produz, no nó central, uma derivada (`dRow`/`dCol` do nível 0) igual à diferença central esperada entre os vizinhos opostos, com `coverage = 1`; marcando um vizinho com `WithNoValueAt`, a derivada naquela direção passa a vir só do lado que ainda tem valor (diferença unilateral), nunca `NaN`; marcando os dois vizinhos opostos de um nó sem valor, a cobertura daquele nó cai a `0` e sua derivada não entra em nenhuma conta (valor irrelevante, nunca lido); `placedSurface.normalAt(x, y)` sobre essa grade inclinada devolve uma normal cuja componente horizontal aponta na direção esperada (downhill, oposta ao crescimento da altura) e cuja magnitude cresce com a inclinação da grade; sobre uma grade totalmente plana, `normalAt` devolve exatamente `(0, 0, 1)` em qualquer ponto
- [X] T008 [US1] Implementar em `internal/domain/frame_surface.go`: tipo `terrainGradient{rows, cols int; dRow, dCol, coverage []float32}`; `surface` ganha o campo `gradients []terrainGradient` (uma pirâmide de um só nível por enquanto — o campo já no formato que o US3 vai estender, para não precisar mudar a forma do dado depois), populado em `newSurface` a partir de `grid.values` (nunca de `s.heights`): por nó, diferença central entre os dois vizinhos na mesma direção quando os dois têm valor real, diferença unilateral quando só um tem, cobertura `0` (derivada sem sentido, nunca usada) quando nenhum vizinho nem o próprio nó tem valor; `placedSurface.normalAt(x, y float64) (nx, ny, nz float64)` lê o nível 0 por interpolação bilinear entre os quatro nós vizinhos de `(x, y)` (mesma coordenada `nodeCoordinates` que `heightAt` já usa), cai para a normal plana `(0, 0, 1)` onde a cobertura ali é `0`, converte a derivada (metros por célula) em inclinação real usando `cx`/`cy` do `placedSurface`, e normaliza o vetor `(-inclinaçãoX, -inclinaçãoY, 1)`; `make test` verde
- [X] T009 [P] [US1] Estender `internal/domain/frame_scene_test.go`: um novo `Test_Scene_Render_TerrainLight`, com uma `Scene` sobre uma grade inclinada numa direção conhecida (reaproveitando `sceneSlice`) e um mapa de cor sólida (`solidDecoder`): o pixel de terreno na encosta voltada para a direção da luz fixa (`TerrainLightAzimuthDegrees`/`TerrainLightAltitudeDegrees`) sai mais claro que `mapColor`; o da encosta oposta sai mais escuro; sobre um trecho **plano** da mesma cena (reaproveitando `flat(40, 100)`), o pixel de terreno sai **exatamente** igual a `mapColor` (nenhuma mudança, prova de que superfície plana é neutra); reaproveitando `holeSlice`/`withHoleBlock(flat(40, 100))`, o hachurado de "sem mapa" e o xadrez de "sem elevação" continuam exatamente `NoMapColors`/`NoElevationColors`, pixel a pixel, iguais aos de antes desta etapa
- [X] T010 [US1] Implementar em `internal/domain/frame_scene.go`: em `drawPixel`, no ramo `case stateImage` (único ramo afetado), depois de obter `color` de `sampler.color`, calcular `nx, ny, nz := ground[hit.grid].normalAt(hit.x, hit.y)`, `fator := terrainLightFactor(nx, ny, nz)`, e substituir `color` por `RGB{rounded(float64(color.R)*fator), rounded(float64(color.G)*fator), rounded(float64(color.B)*fator)}` antes de `image.Set`; os ramos `stateNoMap`/`stateNoElevation` continuam sem nenhuma mudança; `make test` verde
- [X] T011 [US1] Validação manual: `quickstart.md` itens 1 e 3 (volume real na mesma cor de mapa; traçado/marcador/sobreposições/padrões de "sem dado" inalterados) — confirmar com o binário real

**Checkpoint**: `render frame`/`render all`/`fly` já mostram volume real
no relevo, sem nenhuma flag nova — MVP entregue. A iluminação pode ainda
cintilar ao longe (História 3 resolve isso depois).

---

## Phase 4: User Story 2 — Mapa continua legível na encosta mais escura (Priority: P2)

**Objetivo**: confirmar (ou, se necessário, ajustar) que a faixa fixa de
brilho escolhida na Foundational nunca apaga a diferença entre cores
comuns do mapa, nem na encosta mais clara nem na mais escura (FR-003,
FR-004).

**Teste Independente**: `quickstart.md` item 2.

- [X] T012 [P] [US2] Estender `internal/domain/frame_terrain_light_test.go`: duas cores de mapa distinguíveis e típicas (nenhuma delas já no extremo 0/255 de nenhum canal — ex.: `RGB{40, 80, 120}` e `RGB{60, 100, 140}`), multiplicadas pela mesma conta que `drawPixel` usa (`rounded(float64(canal) * fator)`) em `TerrainLightMinFactor` e em `TerrainLightMaxFactor`, continuam diferentes uma da outra nos dois casos (a faixa fixa escurece/clareia as duas pela mesma proporção, sem apagar a diferença entre elas); a mesma conta sobre uma cor já próxima de um extremo (ex.: `RGB{245, 245, 245}`) em `TerrainLightMaxFactor` ainda produz uma saturação documentada como esperada (não é uma falha desta tarefa — só confirma o limite conhecido)
- [X] T013 [US2] Se os cenários de T012 revelarem que `TerrainLightMinFactor`/`TerrainLightMaxFactor` (0,6/1,4, escolhidos no `research.md`) apagam a diferença entre cores comuns do mapa, ajustar os dois valores em `internal/domain/render_tuning.go` para o menor intervalo que ainda cumpre T012; se os valores do `research.md` já passam sem ajuste, esta tarefa só registra essa confirmação (sem mudança de código) — `make test` verde em qualquer dos dois casos. **Revisado após validação manual com dado real** (fora do escopo original de T012, que só testava cores sintéticas típicas, nunca a cor de fundo de um mapa claro real): um quadro gerado sobre `itaquara_mapa_2.mbtiles` (tema claro de estilo OSM) mostrou >10% dos pixels saturando em branco puro — o máximo geométrico real da fórmula (`≈1,293`) nunca batia no teto de `1,4`, que por isso nunca protegia nada; `TerrainLightMinFactor`/`TerrainLightMaxFactor` ajustados para `0,75`/`1,15` (`render_tuning.go`, `research.md` item 8), `frame_terrain_light_test.go` ganhou um teste que confirma o novo teto realmente limita (`should actually clamp...`) e um que documenta o limite residual sobre um fundo claro típico; hash de referência de `frame_scene_test.go` recalculado; `make test` verde.
- [X] T014 [US2] Validação manual: `quickstart.md` item 2 (mapa continua legível na parte mais escura, com um fundo escuro) — confirmar com o binário real

**Checkpoint**: a faixa fixa dá volume perceptível sem nunca apagar a
diferença entre duas cores comuns do mapa base.

---

## Phase 5: User Story 3 — Iluminação estável ao longo do voo (Priority: P3)

**Objetivo**: trocar a leitura "só nível 0" de `normalAt` por uma pirâmide
de níveis escolhida pela distância da câmera ao ponto — a mesma técnica de
mipmap que `imagery.go` já usa para a textura —, estável entre quadros
vizinhos e suave entre pixels vizinhos, subindo de nível quando a
vizinhança mais fina não tem nenhuma amostra real (FR-007; a parte de
FR-008 que a História 1 deixou de lado: subir de nível em vez de cair
direto no tom neutro).

**Teste Independente**: `quickstart.md` itens 4 e 5.

- [X] T015 [P] [US3] Estender `internal/domain/frame_surface_test.go`: construindo a pirâmide inteira de uma grade `n×n` (potência de dois, para simplificar o cálculo esperado à mão), `len(gradients)` é `log2(n) + 1`, terminando num nível `1×1`; cada nível, a partir do primeiro, tem `dRow`/`dCol`/`coverage` iguais à média dos quatro filhos **ponderada pela cobertura de cada um** (cenário com um bloco de buracos num canto: o nível acima daquele canto tem cobertura menor que `1`, mas ainda reflete só a derivada dos filhos com valor real, nunca um valor inventado a partir do buraco); `normalAt(x, y, footprintPequeno)` dá, a menos de tolerância de ponto flutuante, o mesmo resultado que a leitura de só nível 0 já dava antes desta tarefa; sobre uma grade com inclinações alternadas em faixas finas (como o relevo sintético do `quickstart.md`), `normalAt(x, y, footprintGrande)` dá uma normal visivelmente mais próxima da plana que `normalAt(x, y, footprintPequeno)` no mesmo ponto — prova de que o nível mais grosseiro suaviza a leitura; um ponto cuja vizinhança de nível 0 não tem nenhuma cobertura (isolado dentro de um bloco de buracos), mas cujo nível seguinte já tem cobertura positiva, devolve a normal desse nível seguinte em vez de cair direto no plano (a subida de nível de FR-008)
- [X] T016 [US3] Implementar em `internal/domain/frame_surface.go`: `newTerrainGradientPyramid` constrói, a partir do nível 0 (já construído pelo US1), cada nível seguinte como a média dos quatro filhos ponderada por `coverage` (mesmo padrão de agregação que `halved`, de `frame_imagery.go`, usa para a textura, só que com peso em vez de média simples), até sobrar `1×1`; `surface.gradients` passa a guardar a pirâmide inteira; `placedSurface.normalAt` ganha o parâmetro `footprintMeters`: escolhe o nível por `clamp(log2(max(footprintMeters/sqrt(cx*cy), 1)), 0, últimoNível)`, mistura entre `floor`/`ceil` do nível pela mesma fração que `sampler.color` já usa entre mipmaps de textura, e, quando a cobertura do nível escolhido naquela posição é `0`, sobe um nível e repete (até achar cobertura positiva ou esgotar a pirâmide, caindo então no plano); `make test` verde
- [X] T017 [US3] Implementar em `internal/domain/frame_scene.go`: em `drawPixel`, calcular `footprint := pixelAngle * hit.t / sqrt(max(|dz|, 0.1))` (mesma correção de rasante que `sampler.color` já aplica) e passá-lo a `ground[hit.grid].normalAt(hit.x, hit.y, footprint)` no lugar da chamada de dois argumentos do US1; expor `pixelAngle` do `sampler` ao chamador (campo já existente em `sampler`, hoje só usado internamente) para esta conta; `make test` verde
- [X] T018 [P] [US3] Estender `internal/domain/frame_scene_test.go`: sobre um terreno com inclinações alternadas em faixas finas, duas `Scene`s do mesmo recorte renderizadas a distâncias de câmera bem diferentes (uma próxima, uma distante) têm, no quadro distante, uma variação de brilho pixel a pixel (diferença entre vizinhos horizontais) menor, em média, que no quadro próximo — prova indireta de suavização sem exigir uma métrica perceptual; recalcular e atualizar a constante de hash de `Test_Scene_Render_Determinism`/"should draw the image the reference says" (`roughScene`) com o valor real produzido depois da implementação (CLAUDE.md: a constante só muda junto com `RenderVersion`, que já mudou na Foundational). **Nota**: a constante de hash já havia sido recalculada em T010 (os pixels do terreno já mudam desde o US1, não só a partir desta tarefa); com a pirâmide/`footprintMeters` do US3, `roughScene` continua dentro do nível 0 (câmera próxima, quadro pequeno) e o hash permanece exatamente o mesmo valor — confirmado, sem necessidade de um novo valor.
- [X] T019 [US3] Validação manual: `quickstart.md` itens 4 e 5 (vizinhança incompleta perto de um buraco de elevação; estabilidade entre quadros a distâncias diferentes) — confirmar com o binário real

**Checkpoint**: a iluminação é lida numa vizinhança proporcional à
distância, estável ao longo do voo e suave entre pixels vizinhos, sem
nunca inventar dado perto de um buraco de elevação.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Propósito**: os casos que não pertencem a nenhuma história específica, a documentação e a verificação final.

- [X] T020 [P] Validação manual: `quickstart.md` itens 6 e 7 (quadros de antes desta etapa nunca se juntam aos de depois, por `render all` e por `fly --keep`; determinismo byte a byte preservado entre execuções/arquiteturas)
- [X] T021 [P] Atualizar `specs/005-frame-rendering/contracts/frame-files.md` com a nota desta etapa (tabela "Cores e padrões": a cor do terreno passa a ser modulada por um fator de brilho fixo; `RenderVersion` de 5 para 6) — mesmo padrão que as etapas 6, 8, 9, 11, 12 e 15 já aplicaram (`contracts/frame-files-change.md` desta etapa já descreve o conteúdo exato da nota)
- [X] T022 [P] Atualizar `CLAUDE.md`: acrescentar o parágrafo da etapa 016 ao resumo do projeto e a seção "A iluminação direcional do terreno (etapa 16)" (luz fixa documentada, pirâmide de gradiente, faixa fixa de brilho, `RenderVersion` 6), seguindo o mesmo nível de detalhe das seções de etapas anteriores já presentes no arquivo
- [X] T023 Verificação final: `make build`, `make test`, `make lint`, `make generate` verdes (confirmar que `make generate` não produz nenhuma diferença — nenhuma porta nova); `gofmt -l .` limpo; `go mod tidy` sem diferença; `quickstart.md` reexecutado item a item ao longo da implementação (T011, T014, T019, T020), todos confirmados com o binário real

---

## Dependências e Ordem de Execução

### Dependências entre fases

- **Phase 1 (Setup)**: sem dependências.
- **Phase 2 (Foundational)**: depende da Phase 1; **bloqueia todas as
  histórias**. Parte A (constantes/`RenderVersion`) e Parte B (fórmula do
  fator) são independentes entre si (arquivos diferentes).
- **Phase 3 (US1)**: depende da Phase 2 concluída (usa `terrainLightFactor`
  e as constantes). T007→T008 e T009→T010 são, cada par, sequenciais
  (teste antes da implementação que depende dele); T008 e a dupla T009/
  T010 podem avançar em paralelo até T010 precisar de `normalAt` (T008)
  pronto — na prática, concluir T008 antes de começar T010 evita
  retrabalho.
- **Phase 4 (US2)**: depende só da Phase 2 (não lê nada que o US1
  construiu — `normalAt`/`drawPixel` não são tocados); pode, em
  princípio, ser feita antes ou em paralelo com a Phase 3, mas só faz
  sentido como validação depois que a Phase 3 já produz uma imagem para
  observar no item de quickstart correspondente (T014).
- **Phase 5 (US3)**: depende da Phase 3 concluída — **estende** o nível 0
  que o US1 construiu (`gradients`) numa pirâmide, e troca a assinatura de
  `normalAt`/a chamada em `drawPixel` que o US1 deixou pronta. Não pode
  ser feita em paralelo com a Phase 3 (mesmos arquivos, mesmo tipo).
- **Phase 6 (Polish)**: depende das histórias desejadas (idealmente todas).

### Dependências entre histórias

- **US1 (P1)**: depende só da Foundational.
- **US2 (P2)**: depende só da Foundational — independente de US1 em
  lógica (não toca `frame_surface.go`/`frame_scene.go`), mas sua
  validação manual (T014) é mais útil depois que US1 já existe para gerar
  uma imagem iluminada a inspecionar.
- **US3 (P3)**: depende de **US1 concluído** — estende diretamente o que
  US1 construiu (`terrainGradient`/`normalAt`/a chamada em `drawPixel`),
  nos mesmos arquivos. Não é independente de US1 do jeito que US2 é.

### Dentro de cada história

- Teste antes da implementação que depende dele (deve falhar primeiro).
- A história é completa antes da próxima prioridade (US3, em particular,
  só começa com US1 já mesclado/completo, por depender dele de verdade).

### Oportunidades de paralelismo

```text
# Foundational, Partes A e B, em paralelo entre si:
T003 frame_set_test.go           T005 frame_terrain_light_test.go (novo arquivo)
→ T004 render_tuning.go          → T006 frame_terrain_light.go (novo arquivo)

# US1: os dois pares de teste podem ser escritos em paralelo; as
# implementações, na prática, em sequência (T010 usa o normalAt de T008):
T007 frame_surface_test.go       T009 frame_scene_test.go
→ T008 frame_surface.go          → (aguarda T008) → T010 frame_scene.go

# US2: independente de US1/US3, mas sem nada para paralelizar dentro de si
T012 frame_terrain_light_test.go → T013 (se necessário) render_tuning.go

# US3: mesmo padrão de US1, estendendo os mesmos arquivos (nunca em
# paralelo com as tarefas de implementação de US1, só depois delas):
T015 frame_surface_test.go       T018 frame_scene_test.go
→ T016 frame_surface.go          → (aguarda T016/T017)
                     T017 frame_scene.go
```

---

## Estratégia de Implementação

### MVP Primeiro (Phase 1 + 2 + User Story 1)

1. Phase 1 (branch e linha de base) e Phase 2 (direção fixa da luz, faixa
   fixa de brilho, fórmula pura — nenhuma mudança visual ainda).
2. Phase 3 (US1): a normal do nível mais fino ligada à fórmula, aplicada
   só à cor do mapa.
3. **PARE e valide**: `quickstart.md` itens 1 e 3 — volume real, nenhum
   efeito sobre traçado/marcador/sobreposições/padrões de "sem dado" —
   antes de acrescentar as demais histórias.

### Entrega Incremental

1. Setup + Foundational → nenhuma mudança visual ainda.
2. + US1 → volume real no relevo em qualquer comando que desenha quadros
   (MVP!) → itens 1, 3 — com a ressalva conhecida de poder cintilar ao
   longe.
3. + US2 → confirmado (ou ajustado) que a faixa fixa nunca apaga a
   diferença entre cores do mapa → item 2.
4. + US3 → iluminação estável e suave em qualquer distância de câmera,
   sem inventar dado perto de buracos → itens 4, 5.
5. Polish → quadros antigos nunca reaproveitados, determinismo byte a
   byte confirmado, documentação, verificação final → itens 6, 7.

Diferente de uma feature com histórias ortogonais, a ordem aqui **é** uma
dependência técnica a partir de US3 (que estende US1), não só uma ordem de
prioridade — ver "Dependências entre histórias" acima.

---

## Notas

- `[P]` = arquivos diferentes, sem dependência de tarefa incompleta (ou,
  para o par teste/implementação, a tarefa de teste em si — nunca a
  implementação que a segue).
- Nenhum valor exibido, bloco, flag, formato de arquivo ou entidade
  persistida (plano, recorte, aparência, sobreposição) nasce ou muda
  nesta etapa (FR-011, FR-013): toda tarefa de implementação só muda
  **como** o terreno já desenhado é colorido — se uma tarefa parecer
  exigir uma porta nova, uma flag nova, ou uma leitura nova do trajeto
  GPS/dados geográficos registrados, algo está errado.
- Nunca usar `Pow`/`Exp`/`Sinh`, nem iterar um `map` para produzir um
  valor que entra no desenho — a mesma disciplina aritmética que o
  resto do desenho de quadros já segue (research.md item 10).
- Faça commit após cada tarefa ou grupo lógico (só quando o usuário pedir).
- Pare em qualquer checkpoint para validar a história com o `quickstart.md`.
- Evite: tarefas vagas, duas tarefas `[P]` editando o mesmo arquivo,
  pular a verificação de `make test` entre uma tarefa de teste e a de
  implementação que a segue, começar a Phase 5 (US3) antes da Phase 3
  (US1) estar de fato concluída.
