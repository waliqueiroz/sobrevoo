# Pesquisa Técnica: Rótulos em Português e Legibilidade do Texto das Sobreposições

Cada item segue Decisão / Racional / Alternativas consideradas. Item 0
resolve todos os `NEEDS CLARIFICATION` do Contexto Técnico do `plan.md`.

## 0. Contexto técnico — resumo

Nenhum `NEEDS CLARIFICATION` sobrou: a etapa se apoia inteiramente na
stack já existente (Go 1.26, `golang.org/x/image`, já em `go.mod` —
nenhuma dependência nova) e nos mesmos quatro arquivos de domínio que a
etapa anterior (`011-overlay-polish`) já estabeleceu como o ponto de
extensão do acabamento das sobreposições. As decisões abaixo resolvem os
três defeitos de legibilidade e a tradução dos rótulos.

## 1. As quatro palavras em português

**Decisão**: `"DIST"`, `"ELEV"`, `"GANHO"`, `"TEMPO"` — no lugar de
`"DIST"`, `"ELEV"`, `"GAIN"`, `"TIME"`. `"DIST"` e `"ELEV"` não mudam:
já são abreviações válidas em português ("distância", "elevação"), com a
mesma forma que tinham em inglês — trocá-las não teria efeito visível e
arriscaria, sem necessidade, um texto mais longo que o de hoje. `"GAIN"` e
`"TIME"` são as duas palavras genuinamente inglesas do conjunto, e viram
`"GANHO"` e `"TEMPO"`.

**Racional**: mantém o estilo enxuto de hoje (maiúsculas, sem acento,
curto) e muda exatamente o que precisa mudar — as duas palavras que não
fazem sentido em português —, sem inflar a largura dos painéis mais do que
o necessário. `allHaveTime`/`TimeReferenceClock` e os demais nomes de
domínio (código, não texto desenhado) permanecem em inglês, como a
constituição já determina para identificadores.

**Alternativas consideradas**:
- Traduzir "DIST"/"ELEV" também (ex.: para algo mais longo como
  "DISTÂNCIA"/"ELEVAÇÃO"): rejeitada — já são palavras em português na
  forma abreviada, e abreviações mais longas quebrariam o estilo visual
  dos rótulos sem nenhum ganho de clareza.
- Abreviações diferentes (ex.: "ALT" para elevação): rejeitada — "ELEV" já
  é natural em português e trocar sem necessidade só adicionaria risco de
  regressão nos testes que comparam o texto exato.

## 2. O contorno do texto: espessura derivada do corpo da fonte

**Contexto**: hoje (`internal/domain/frame_screen_overlay.go`,
`drawText`), o raio de dilatação do contorno é
`max(OverlayOutlineMinWidth, OverlayOutlineRatio × altura do quadro)` —
em 1080×1920, `0,0025 × 1920 ≈ 4,8 → 5 px`, quase do tamanho do próprio
traço da letra (5–6 px nessa resolução), fechando os vãos internos de
"6"/"8"/"0" e encostando o halo de letras vizinhas. O problema não é o
valor em si, é a base errada: a espessura de um contorno **de uma letra**
deve depender do **tamanho da letra** (`ppem`, "pixels per em" — o mesmo
valor que `drawText` já calcula e usa para rasterizar o glifo), não da
altura do quadro inteiro.

**Decisão**: o raio de dilatação passa a ser
`max(OverlayOutlineMinWidth, OverlayOutlineRatio × ppem)`, com
`OverlayOutlineRatio` recalibrada de `0,0025` para `0,035` (a mesma
constante, reinterpretada e com um novo valor — não uma constante nova).
Em 1080×1920 (`ppem ≈ 54` com `glyphHeightRatio` de hoje), o raio passa a
ser `max(1, round(0,035 × 54)) = max(1, 2) = 2 px` — nitidamente mais fino
que o traço de 5–6 px da letra, só o bastante para destacá-la do fundo.
`OverlayOutlineMinWidth` (piso de 1 px) não muda: continua garantindo que
o contorno nunca desapareça em texto muito pequeno.

**Racional**: a proporção entre a espessura do contorno e o tamanho da
letra passa a ser **constante em qualquer resolução** (já que as duas
grandezas — contorno e `ppem` — escalam juntas, pela mesma altura do
quadro), que é exatamente o que FR-005 pede: "a mesma proporção em
qualquer resolução". Na menor resolução aceita (altura 180, `ppem ≈ 5`), o
cálculo já cairia abaixo de 1 px (`0,035 × 5 ≈ 0,18`) e o piso de 1 px
assume — o mesmo comportamento de piso que a constante já tinha antes,
preservado.

**Alternativas consideradas**:
- Só reduzir o valor de `OverlayOutlineRatio` mantendo a base na altura do
  quadro: rejeitada — resolveria o sintoma numa resolução específica, mas
  a proporção entre contorno e letra continuaria variando com a
  resolução (porque `ppem` e a altura do quadro não crescem na mesma
  razão em todas as resoluções aceitas — na verdade crescem, já que `ppem
  = glyphHeightRatio × altura`, então as duas são proporcionais entre si;
  mas a FR pede explicitamente que a base do cálculo seja o corpo da
  fonte, não a altura, pela mesma razão de clareza de código que já leva
  `TrailWidthRatio`/`MarkerRadiusRatio` a serem fração da altura do quadro
  e não um valor fixo: amanhã, se o tamanho do texto (`glyphHeightRatio`)
  mudar independentemente da altura, o contorno deve acompanhar o texto,
  não a altura).
- Calcular a espessura a partir da espessura real do traço da fonte
  (medindo o `stem width` do glifo): mais preciso, mas exigiria uma nova
  leitura de métrica da fonte sem nenhum ganho prático sobre uma fração
  fixa de `ppem`, já suficiente para o problema relatado.

## 3. A fonte embutida: peso forte, mesma família

**Decisão**: `golang.org/x/image/font/gofont/gobold` — a variante "Go
Bold" da mesma família "Go" (Bigelow & Holmes) já usada desde
`011-overlay-polish`, no lugar de `font/gofont/goregular`. Mesmo módulo
(`golang.org/x/image`, já em `go.mod`), mesma licença (BSD-3-Clause,
coberta pelo mesmo `ttfs/README` do pacote), nenhuma dependência nova.
Única mudança de código: a linha que importa e passa o `[]byte` da fonte a
`sfnt.Parse` em `newVectorFace()` (`internal/domain/vector_font.go`).

**Racional**: cumpre FR-006 ao pé da letra — peso mais forte, mesma
família, embutida, licença permissiva, nenhuma dependência nova — com a
menor mudança de código possível: nenhuma outra parte do rasterizador
(`sfnt`, o achatamento de curvas, a superamostragem) depende do peso da
fonte, só dos bytes que entram em `sfnt.Parse`.

**Alternativas consideradas**:
- Uma fonte de peso forte de outra família: rejeitada pelo próprio pedido
  ("de preferência da mesma família já usada").
- `gofont/gomediumitalic`/`gofont/gobolditalic` ou outras variantes:
  rejeitadas — itálico não resolve o problema de contraste descrito
  (traços finos), só muda a inclinação; a etapa pede peso mais forte, não
  itálico.

## 4. Largura estável dos painéis numéricos — a decisão mais profunda

**Contexto**: hoje, `screenOverlay.draw(plan, index)` monta os três textos
numéricos **do quadro `index`** e chama `numericPanelWidth` só sobre eles
— a largura reflete só o quadro atual, por isso muda de quadro a quadro
conforme o número de dígitos de cada valor varia (ex.: "8.6 km" e
"19.9 km" têm larguras diferentes). FR-007 exige que a largura seja a
mesma do primeiro ao último quadro do voo; FR-008 exige que um quadro
isolado (`render frame`) tenha a mesma largura que teria dentro do voo
inteiro — ou seja, a largura não pode depender só do quadro pedido, tem
que refletir **todo o plano**.

**Decisão**: a largura compartilhada passa a ser calculada a partir de
**todos** os quadros do plano — o maior `textWidth` entre os três textos
(distância; elevação+ganho; tempo decorrido) de **cada** quadro —, uma
única vez por `Scene`, e reaproveitada por toda chamada de `Render`
seguinte. Mecanicamente:

- Uma função pura nova, `stablePanelWidth(face *vectorFace, plan
  CameraPlan, config OverlayConfig, ppem int) int`
  (`frame_screen_overlay.go`), percorre `plan.Frames` uma vez, monta os
  três textos de cada quadro pelas mesmas três funções que `draw` passa a
  usar (`distanceBlockText`/`elevationBlockText`/`timeBlockText` — ver
  item 5) e guarda o maior `face.textWidth(...)` encontrado, pulando um
  bloco que `config`/o plano não mostram (mesmo critério de hoje:
  `config.Elevation && plan.ElevationAvailable`, etc.) — se nenhum bloco
  numérico aparece, `0`.
- `Scene` (`frame_scene.go`) ganha três campos não exportados —
  `panelWidth int`, `panelWidthHeight int`, `panelWidthSet bool` — e um
  método `numericPanelWidth(plan CameraPlan, height, ppem int) int` que
  calcula `stablePanelWidth` só na primeira chamada (ou quando `height`
  muda, ver abaixo) e devolve o valor em cache nas chamadas seguintes.
  `Render` o chama antes de montar `screenOverlay{...}`, passando o
  resultado como um campo novo, `panelWidth`, em vez de deixar
  `screenOverlay.draw` calculá-lo sozinho a cada quadro.

**Por que em `Scene`, não em `screenOverlay`**: `screenOverlay` é
reconstruído a cada `Render` (um valor novo por quadro); `Scene` é o único
tipo que já vive por toda a execução de um `render all`/`fly` — é
exatamente onde `vectorFace` (o cache de glifos da etapa anterior) também
vive, pela mesma razão: evitar refazer, a cada quadro, um trabalho que só
precisa ser feito uma vez por execução.

**Por que isso não é `O(quadros²)`**: sem o cache, calcular a largura a
partir do plano inteiro a cada quadro custaria `O(quadros)` por quadro —
`O(quadros²)` no total, proibitivo em 432 000 quadros. Com o cache em
`Scene`, o custo total do cálculo é `O(quadros)`, pago uma única vez por
execução — a mesma ordem de grandeza que já formatar/escrever o plano uma
vez já custa, desprezível perto do traçado de raios por pixel do terreno.

**A guarda por `height`, não por identidade do plano**: `CameraPlan.ID()`
teria sido a chave "correta" para invalidar o cache se `Scene` fosse
reaproveitada entre planos diferentes — mas `ID()` é, ele mesmo, `O(quadros)`
(percorre e faz hash de cada quadro), então usá-lo como chave de cache
reintroduziria o mesmo custo quadrático que o cache existe para evitar.
Em vez disso, a guarda é só a altura da resolução pedida (`height`, um
inteiro, comparação O(1)) — suficiente para o uso real: nenhum dos dois
call sites (`FrameService.DrawFrame`, `DrawFrames`, em
`internal/application/frame_service.go`) constrói uma `Scene` e a usa para
mais de um plano; `NewScene` é chamado de novo a cada requisição. É a
**mesma garantia** que `vectorFace.cache` já assume desde a etapa anterior
(um `Scene` renderiza um plano, numa resolução, do início ao fim) —
estendida ao cache de largura, com uma verificação extra (a de `height`)
que o cache de glifos não precisava, porque o tamanho de um glifo já é
parte da própria chave dele (`glyphKey{r, ppem}`), e aqui a largura
cacheada não carrega essa chave sozinha.

**Racional**: é o único jeito de cumprir FR-007 (estável no voo inteiro) e
FR-008 (quadro isolado igual ao do voo inteiro) ao mesmo tempo, sem pagar
um custo proibitivo por quadro — e reaproveita, para a largura, exatamente
o mesmo padrão de "cache por `Scene`, um valor por execução" que o
rasterizador de fonte já estabeleceu para os glifos.

**Alternativas consideradas**:
- Calcular a largura máxima teórica (pior caso) sem olhar o plano, com
  base só nos formatos possíveis (ex.: assumir sempre o texto mais longo
  que cada formatador poderia produzir): rejeitada — o próprio `spec.md`
  (Suposições de `011-overlay-polish` e desta etapa) já preferiu medir o
  texto real a assumir um "pior caso" arbitrário; um pior caso teórico
  também produziria painéis desnecessariamente largos na maioria dos
  voos.
- Mover o cálculo para `CameraPlanService`/a geração do plano (gravando a
  largura no arquivo do plano exportado): rejeitada — a largura depende
  da fonte, do `ppem` (logo, da resolução) e do idioma do desenho, nenhum
  dos quais o plano de câmera conhece ou deveria conhecer (o plano é
  independente de aparência/sobreposição, como as etapas 8 e 9 já
  estabeleceram); guardá-la no plano misturaria uma decisão de desenho
  dentro de um artefato que outras etapas (e formatos de arquivo mais
  antigos) não preveem.
- Invalidar o cache por `CameraPlan.ID()` mesmo com o custo: rejeitada
  pelo racional acima (reintroduz o custo quadrático).

## 5. Texto de cada bloco: função só, compartilhada entre o desenho e a medição

**Decisão**: três funções puras novas, uma por bloco numérico —
`distanceBlockText(frame CameraFrame) string`,
`elevationBlockText(frame CameraFrame) string`,
`timeBlockText(frame CameraFrame) string` — em
`internal/domain/frame_screen_overlay.go`, cada uma montando exatamente o
texto que `draw` desenha hoje inline (ex.: `"DIST " +
formatOverlayDistance(frame.MarkerDistance)`, com os rótulos novos do item
1). Tanto `draw` (ao desenhar) quanto `stablePanelWidth` (ao medir, item
4) passam a chamar as mesmas três funções — nunca duas cópias do mesmo
texto escritas em dois lugares.

**Racional**: é a única forma de garantir, por construção, que a largura
calculada bate exatamente com o texto que será desenhado — se as duas
cópias pudessem divergir (por exemplo, um espaçamento diferente entre
"ELEV" e o valor), a largura "estável" deixaria de bater com o texto real,
quebrando a própria garantia que a etapa pede. É o mesmo raciocínio que já
levou outras etapas a extrair uma função só quando duas partes do código
precisam concordar byte a byte (ex.: `CameraPlan.ID()` sendo a única fonte
da identidade de um plano).

**Alternativas consideradas**: manter os textos inline em `draw` e
duplicá-los em `stablePanelWidth` — rejeitada pelo risco de divergência
acima.

## 6. `overlayPpem`: uma função só para o tamanho do glifo

**Decisão**: a conta `max(1, roundHalfUp(glyphHeightRatio × altura))` —
hoje inline dentro de `draw` — vira uma função livre,
`overlayPpem(height int) int`, em `frame_screen_overlay.go`. `draw` a usa
para desenhar; `Scene.Render` a usa, **antes** de montar `screenOverlay`,
para calcular o `ppem` que passa a `numericPanelWidth` (item 4) — o mesmo
`ppem` em que o texto será de fato desenhado.

**Racional**: evita que a fórmula do tamanho do glifo viva em dois
lugares (o mesmo motivo do item 5) — se divergisse, a largura
pré-calculada usaria um `ppem` diferente do que `drawText` usa de verdade,
produzindo painéis com a largura errada para o texto real.

## 7. Versão do desenho

**Decisão**: `domain.RenderVersion` sobe de `3` para `4`. Como
`NewFrameSetID`/`newFrameSetID` (`frame_set.go`) já incluem `RenderVersion`
no hash do `FrameSetID` — sem nenhuma mudança de código além do valor da
constante —, um conjunto de quadros desenhado pela versão anterior passa a
ser, automaticamente, "de outro conjunto": `render all` o recusa sem
`--overwrite` (`ErrFrameSetConflict`) e `fly --keep` redesenha os quadros
e o vídeo ao notar a mudança, pela mesma lógica já comprovada nas etapas
6, 8, 9 e 11.

**Racional**: mesmo mecanismo mínimo de sempre; nenhuma mudança de
formato de arquivo, nenhuma migração, nenhum campo novo precisa existir.

**Alternativas consideradas**: nenhuma — é o mesmo mecanismo que toda
mudança visual anterior já usou.
