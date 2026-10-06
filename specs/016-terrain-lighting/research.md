# Pesquisa Técnica: Iluminação Direcional do Terreno

Cada item segue Decisão / Racional / Alternativas consideradas. Item 0
resolve todos os `NEEDS CLARIFICATION` do Contexto Técnico do `plan.md`.

## 0. Contexto técnico — resumo

Nenhum `NEEDS CLARIFICATION` sobrou: a etapa se apoia inteiramente na stack
já existente (Go 1.26, só `math` da biblioteca padrão) e nos dados que o
pipeline de desenho já carrega (a `ElevationGrid` do recorte). A decisão
mais profunda é como ler uma inclinação estável e suave a partir de uma
grade de elevação discreta sem reintroduzir cintilação (item 5) e sem
inventar dado onde não há elevação conhecida (item 7) — os dois cuidados
que o `spec.md` e a sessão de `/speckit-clarify` tornam explícitos.

## 1. O modelo de iluminação (FR-001, FR-002, FR-003)

**Contexto**: o pedido exige "clarear quando a encosta está voltada para a
luz, escurecer quando está voltada para o lado oposto", numa faixa fixa, e
(da sessão de clarificação / caso de borda) "uma superfície praticamente
plana recebe tom neutro, **qualquer que seja** a altura da luz escolhida" —
um sombreamento Lambertiano comum (`fator = max(0, dot(N, L))`) não serve
sozinho: para uma superfície plana (`N = (0,0,1)`), esse fator seria
`sin(altura)`, que só é "neutro" (por acaso) se a altura escolhida for
exatamente 0°, nunca para a altura documentada de fato (item 2).

**Decisão**: um sombreamento **relativo ao plano** — a diferença entre o
produto escalar da normal real com a luz e o produto escalar de uma
superfície plana (`up = (0,0,1)`) com a mesma luz:

```
raw = dot(N, L) - dot(up, L) = dot(N - up, L)
    = N.x·L.x + N.y·L.y + (N.z - 1)·L.z
fator = clamp(1 + raw, TerrainLightMinFactor, TerrainLightMaxFactor)
```

`fator` multiplica cada canal da cor do mapa (via o mesmo `rounded(...)`
que o resto do desenho já usa para arredondar e saturar em [0, 255]).

**Racional**: `N - up` é zero exatamente quando a superfície é plana,
**para qualquer `L`** — então `raw = 0` e `fator = 1` (idêntico ao pixel
sem iluminação) sempre que a inclinação é zero, sem precisar de um caso
especial nem de uma constante ajustada à altura escolhida (resolve o caso
de borda da sessão de clarificação por construção aritmética, não por
`if`). Para uma superfície inclinada, `N - up` tem uma parte horizontal (a
direção para onde a encosta "olha", escalada pela inclinação — ver item 4)
e uma parte vertical ligeiramente negativa (`N.z - 1 ≤ 0`, o quanto a
superfície deixou de apontar para cima): o produto escalar com `L` fica
positivo quando a parte horizontal aponta para a luz (encosta volta
voltada para ela, mais clara) e negativo no lado oposto (mais escura),
exatamente o comportamento pedido por FR-002 — e a parte vertical adiciona
um leve escurecimento a qualquer inclinação, mesmo de perfil à luz, o que é
fisicamente razoável (uma superfície inclinada recebe sempre um pouco menos
de luz direta por área do que uma plana sob o mesmo sol, por
"foreshortening") sem que o `spec.md` exija nem proíba esse detalhe.
`clamp(..., min, max)` é a faixa fixa e documentada de FR-003 — só
comparação e `Min`/`Max`, determinístico.

**Alternativas consideradas**:
- Lambertiano puro, `fator = max(0, dot(N, L))`, remapeado depois para a
  faixa fixa: rejeitado — faria o "neutro" depender da altura da luz
  escolhida (só seria neutro em superfície plana se a altura fosse 0°),
  contrariando o caso de borda resolvido na sessão de clarificação sem
  introduzir um ajuste ad hoc.
- "Half-Lambert" (`0.5 + 0.5·dot(N,L)`), técnica comum em jogos para
  suavizar o lado escuro: mesma objeção — o ponto "neutro" (0,5) não
  corresponde à superfície plana sob uma luz de altura arbitrária.
- Calcular `dot(N, L)` absoluto e normalizar pelo valor mínimo/máximo
  observado na cena (um "auto-exposure" por quadro): rejeitado — o fator de
  um ponto passaria a depender do resto do quadro (e do quadro anterior,
  se suavizado no tempo), quebrando "cada pixel depende só de si mesmo"
  (CLAUDE.md, "O desenho dos quadros") e o determinismo por quadro
  isolado que os testes de hash de referência exigem.

## 2. Direção e altura fixas da luz (FR-001, Suposições)

**Decisão**: azimute 315° (do noroeste) e altura 45° acima do horizonte —
a convenção padrão de mapas de relevo sombreado ("hillshade"), igual ao
padrão de ferramentas como `gdaldem hillshade`. O azimute usa a mesma
convenção de bússola que `CameraFrame.Heading` já usa em
`frame_camera.go` (graus no sentido horário a partir do norte, eixo x para
o leste e y para o norte no plano do quadro), então a direção da luz no
espaço (leste, norte, altura) é:

```
L = (cos(altura)·sin(azimute), cos(altura)·cos(azimute), sin(altura))
```

calculada uma única vez (um `Sin`/`Cos` cada, não por pixel) e guardada
como constante de pacote, nunca recalculada por quadro.

**Racional**: é um valor "documentado" (FR-001) que qualquer pessoa que
já tenha visto um mapa de relevo sombreado reconhece, sem exigir nenhuma
escolha arbitrária nova — e reaproveita a mesma convenção angular que o
resto do domínio (`CameraFrame.Heading`, `newCamera`) já usa, em vez de
inventar uma segunda convenção de ângulo só para a luz. Por ser relativa ao
norte verdadeiro, funciona identicamente em qualquer lugar do mundo
(Princípio IV) — não há "lado" do planeta em que noroeste signifique algo
diferente.

**Alternativas consideradas**: nenhum outro azimute/altura foi
seriamente considerado — o pedido explicitamente delega a escolha exata ao
planejamento (Suposições do `spec.md`), e o padrão de hillshading é a
escolha de menor risco (mais familiar, mais testada por décadas de
cartografia) entre todas as direções possíveis.

## 3. Onde a iluminação entra no pipeline de desenho (FR-002, FR-006)

**Decisão**: só dentro de `Scene.drawPixel` (`frame_scene.go`), no ramo
`case stateImage` — ou seja, só depois que `sampler.color` já resolveu a
cor final do mapa (com toda a mistura de mipmaps que já existe) e só
quando o pixel não é um buraco de mapa nem de elevação. O traçado
(`drawTrail`), o marcador (`drawMarker`) e a sobreposição de tela
(`screenOverlay.draw`) são desenhados depois, em `Scene.Render`, sobre a
imagem já iluminada do terreno — nenhum deles lê nem precisa saber que a
iluminação existe.

**Racional**: é o único ponto do pipeline em que "a cor do mapa neste
pixel" já existe como um valor concreto a modular (FR-002: "a iluminação
modula a cor do mapa, nunca a substitui") — multiplicar ali, uma vez, é
mais simples e mais barato do que iluminar a textura antes de amostrá-la
(que exigiria reconstruir a mistura trilinear de mipmaps para a luz
também) ou iluminar depois de compor o quadro inteiro (que exigiria saber,
por pixel do quadro final, se aquele pixel era terreno ou traçado/
marcador/sobreposição — informação que `drawPixel` já tem de graça, mas
que se perderia depois). Como `stateNoMap`/`stateNoElevation` nunca passam
pelo ramo `stateImage`, FR-006 (os dois padrões de "sem dado" nunca são
iluminados) vale por construção, sem nenhum teste condicional extra.

**Alternativas consideradas**:
- Iluminar dentro de `sampler.color` (a função que já faz a mistura de
  mipmaps da textura): rejeitada — misturaria duas responsabilidades
  (ler uma textura vs. ler uma inclinação) no mesmo tipo (`imagery`/
  `sampler`), que hoje não conhece a geometria da superfície (`surface`)
  nem precisa conhecer.
- Um segundo passe sobre a imagem inteira, depois de `drawTerrain`,
  recalculando quais pixels são terreno: rejeitada — exigiria guardar
  informação adicional por pixel (hoje só `depth` sobrevive ao laço
  principal) só para refazer uma decisão que o próprio laço principal já
  toma uma vez.

## 4. A normal da superfície no ponto do raio

**Contexto**: a superfície já é, hoje, um patch bilinear por célula
(`placedSurface.interpolate`, usado por `heightAt`/`aboveQuad`): dentro de
uma célula, a altura é uma interpolação bilinear dos quatro nós vizinhos.
A normal de um patch bilinear tem uma forma analítica simples a partir das
derivadas parciais nas coordenadas de nó `(a, b)` (a mesma coordenada que
`nodeCoordinates` já calcula) — mas uma derivada local, de uma só célula,
é exatamente o "ponto minúsculo" que o `spec.md` já identifica como fonte
de cintilação ao longe (FR-007): precisa ser combinada com uma vizinhança
maior quando o pixel da tela cobre mais terreno do que uma célula.

**Decisão**: não calcular a derivada do patch bilinear da célula
diretamente; em vez disso, pré-computar uma pirâmide de derivadas médias
por nível (item 5) e buscar nela pelo nível proporcional à distância —
a derivada "de uma célula" é só o nível 0 dessa pirâmide, nunca usada
isoladamente para distâncias maiores que uma célula.

**Racional**: unifica os dois requisitos (ler a inclinação real da
superfície atingida, e fazer isso numa vizinhança proporcional à distância)
num único mecanismo, em vez de dois cálculos separados (um para perto, um
para longe) que precisariam ser costurados sem descontinuidade.

**Alternativas consideradas**: usar só a derivada analítica da célula do
hit, sem pirâmide, aceitando a cintilação ao longe: rejeitada —
contraria FR-007 e a História de Usuário 3 diretamente.

## 5. A pirâmide de gradiente — vizinhança adaptativa por distância (FR-007)

**Decisão**: `surface` (em `frame_surface.go`) ganha, ao lado de
`heights`/`zmin`/`zmax` (já calculados em `newSurface`), uma pirâmide de
níveis — a mesma estrutura de `tileTexture.levels` que `newTileTexture`/
`halved` já constroem para a textura de uma peça do mapa base, só que para
a **derivada da altura** em vez da cor:

```go
// terrainGradient is one level of a surface's precomputed slope pyramid:
// the average height-change per cell, in each grid direction, and how
// much of this level is backed by real elevation samples (never the
// hole-filled heights used for the ray's geometry).
type terrainGradient struct {
    rows, cols   int
    dRow, dCol   []float32 // meters of height change per cell, index space
    coverage     []float32 // 0 (no real sample contributed) to 1 (every one did)
}
```

O nível 0 tem a resolução da própria grade (uma derivada por nó, por
diferença finita com os nós vizinhos — central quando os dois existem,
unilateral quando só um existe, sem valor quando nenhum vizinho nem o
próprio nó têm amostra real); cada nível seguinte tem metade das linhas e
colunas do anterior, com `dRow`/`dCol`/`coverage` a **média ponderada por
cobertura** dos quatro filhos (a mesma conta de `halved`, só com um peso
em vez de uma média simples) — até o nível em que só resta 1×1. Construída
uma vez por `*surface` em `newSurface`, a partir de `grid.values` (as
amostras cruas, com `NaN` para "sem valor"), nunca de `s.heights` (a cópia
com os buracos preenchidos que só a geometria do raio usa).

A posição de busca na pirâmide permanece em coordenadas de nó/célula
(adimensional), não em metros: a conversão para inclinação real por metro
usa `cx`/`cy` de `placedSurface` (que variam por quadro, porque dependem
da latitude da câmera daquele quadro) só no momento da leitura — a
pirâmide em si, como `heights`, é compartilhada por todos os quadros de um
`Scene`.

**Racional**: reaproveita, para um problema análogo (amostrar uma
grandeza contínua numa resolução que varia com a distância, sem
descontinuidade visível), exatamente a técnica que o próprio projeto já
validou para a textura do mapa base — nenhum algoritmo novo é inventado,
só aplicado a um dado diferente (derivada em vez de cor). Separar a
pirâmide (compartilhada entre quadros) da conversão metros-por-célula
(por quadro) espelha a separação que `surface`/`placedSurface` já fazem
para a própria altura.

**Alternativas consideradas**:
- Recalcular a inclinação, a cada pixel, com uma média móvel sobre uma
  janela de tamanho variável (sem pirâmide pré-computada): rejeitada —
  custaria O(tamanho da janela²) por pixel em vez de O(1) numa pirâmide
  pré-computada, e o tamanho da janela cresce com a distância (pode
  cobrir centenas de células ao longe), tornando o desenho arbitrariamente
  mais lento em visões distantes.
- Um único borramento gaussiano da grade de elevação inteira, uma vez,
  numa resolução intermediária fixa: rejeitada — não existe uma única
  resolução "certa" quando a mesma grade é vista de distâncias muito
  diferentes dentro do mesmo voo (de um extremo a outro de um `fly`).

## 6. Escolha do nível da pirâmide e mistura entre níveis (FR-007)

**Decisão**: reaproveitar exatamente a fórmula que `sampler.color` já usa
para escolher o nível de detalhe da textura (`frame_imagery.go`,
`texels`/`lod`), substituindo "texels de uma peça" por "células da grade de
elevação":

```
pegada = pixelAngle · distância / sqrt(max(descida, 0.1))   // mesma correção de rasante já usada
tamanhoDaCélula = sqrt(cx · cy)                               // célula "quadrada" equivalente, em metros
nível = clamp(log2(max(pegada / tamanhoDaCélula, 1)), 0, últimoNível)
```

e misturar entre `floor(nível)` e `ceil(nível)` pela mesma mistura linear
pelo peso fracionário que `sampler.color` já aplica entre dois níveis de
mipmap de textura — cada nível lido por interpolação bilinear entre as
células vizinhas daquele nível (não o nó mais próximo), para a transição
entre pixels vizinhos ser suave (FR-007, "suave entre pixels vizinhos"),
não só estável entre quadros.

**Racional**: `pixelAngle`, `distância` (= `hit.t`, o mesmo valor que já
vai para `sampler.color`) e `descida` (= `|dz|`, já calculado em
`drawPixel`) já existem no ponto exato em que a iluminação é aplicada —
não é preciso calcular nada novo além do tamanho de célula da grade e o
`log2`. Usar a mesma fórmula (em vez de inventar uma segunda heurística de
distância) mantém uma única ideia no código — "quantas unidades da grade
um pixel cobre, a esta distância" — aplicada duas vezes a dados diferentes
(texels de imagem, células de elevação).

**Alternativas consideradas**: um nível de pirâmide fixo, igual para todo
o quadro (baseado só na altitude da câmera, não por pixel): rejeitada —
um único quadro de câmera inclinada mostra terreno próximo e distante ao
mesmo tempo (a parte de baixo do quadro perto, o horizonte longe); um nível
só por quadro deixaria uma das duas grosseira ou a outra cintilante.

## 7. Vizinhança sem elevação suficiente — a regra dos três casos (FR-008)

**Contexto**: a sessão de `/speckit-clarify` corrigiu uma contradição do
primeiro rascunho do `spec.md` e fixou a regra: (1) um ponto que é, ele
próprio, um buraco de elevação continua caindo no padrão "sem elevação"
existente, sem nenhuma iluminação; (2) um ponto com elevação conhecida cuja
vizinhança de amostragem tem buracos usa só as amostras reais disponíveis
nela; (3) se nenhuma amostra da vizinhança tiver elevação conhecida, o
ponto recebe o tom neutro de superfície plana — nunca é tratado como "sem
elevação" por causa da vizinhança.

**Decisão**: o campo `coverage` da pirâmide (item 5) resolve os três casos
sem nenhum `if` dedicado a cada um:

- **Caso 1** nunca chega a `normalAt`: `drawPixel` já decide
  `stateNoElevation` a partir de `ground[hit.grid].isHole(hit.row, hit.col)`
  antes de aplicar qualquer iluminação (item 3) — o próprio ponto sendo um
  buraco é resolvido fora da pirâmide, exatamente como hoje.
- **Caso 2**: `placedSurface.normalAt` lê o nível escolhido (item 6); se a
  cobertura ali, na posição exata, for menor que 1 (alguma amostra que
  contribuiu para aquela média era um buraco), o gradiente já é a média
  **só das amostras reais** — nenhuma invenção, porque a agregação por
  nível (item 5) nunca mistura um valor de "sem dado" como se fosse zero;
  ele simplesmente não entra na média ponderada.
- **Caso 3**: se a cobertura cai a exatamente zero no nível escolhido
  (nenhuma amostra real contribuiu, em nenhum filho), `normalAt` sobe um
  nível (mais grosseiro) da pirâmide e repete, até achar cobertura
  positiva ou esgotar a pirâmide; no topo (nível que cobre a grade
  inteira), a cobertura só é zero se a grade inteira não tiver nenhuma
  amostra real — um caso que `NewScene` já rejeita com
  `ErrNoElevationData` antes de desenhar qualquer quadro (não precisa ser
  tratado aqui de novo). Esgotada a pirâmide sem achar cobertura positiva
  (não deveria ocorrer dada a rejeição de `NewScene`, mas o código
  trata o caso por segurança), `normalAt` devolve a normal plana
  `(0, 0, 1)` — o mesmo resultado que uma inclinação zero produziria,
  então o fator de brilho (item 1) sai exatamente neutro.

**Racional**: nenhum dos três casos precisa de um ramo de código que
"decida que é um destes três casos" — a estrutura de dados (cobertura por
nível, com fallback para o nível acima) já os resolve como consequência
natural de como a média é calculada. Isso evita exatamente o erro que a
sessão de clarificação corrigiu no `spec.md`: tratar "vizinhança
incompleta" como se fosse igual a "o próprio ponto sem dado".

**Alternativas consideradas**:
- Recusar desenhar (erro) um pixel cuja vizinhança não tem cobertura
  total: rejeitada — o `spec.md` já decide explicitamente que esse pixel
  deve mostrar a cor do mapa, só sem iluminação, nunca um erro.
- Propagar `NaN` pela pirâmide e tratar `NaN` como "sem dado" no cálculo
  final do fator: rejeitada — exigiria checagens de `NaN` espalhadas pelo
  caminho crítico (uma por canal, por nível, por pixel), em vez de uma
  única checagem de cobertura no ponto em que o nível é escolhido.

## 8. Faixa fixa de clareamento/escurecimento (FR-003)

**Decisão**: `TerrainLightMinFactor = 0.75` e `TerrainLightMaxFactor = 1.15`
em `render_tuning.go`, ao lado das demais constantes fixas de "o que a
imagem significa" (`NoMapColors`, `OverlayTextColor`, ...).

**Histórico da decisão — por que não ficou em `0,6`–`1,4`**: a primeira
escolha desta pesquisa foi `0,6`–`1,4`, e o item 12 do `quickstart.md`
(mapa claro) só foi conferido visualmente sobre o relevo sintético
`relevo-sp.tif` (cores de xadrez de tom médio) — nunca sobre um mapa
real de fundo claro. Revisão posterior (ver sessão de validação com dados
reais) encontrou dois problemas que esse quickstart não pegava:

1. **O limite `1,4` nunca era alcançado de verdade.** O máximo geométrico
   que `raw = dot(N − cima, luz)` consegue produzir, para a altura fixa de
   45°, é `cos(45°)·(√2 − 1) ≈ 0,293` — atingido exatamente numa encosta de
   45° cujo lado exposto aponta direto para o azimute da luz (conta à mão,
   confirmada em `Test_terrainLightFactor/should actually clamp...`, em
   `frame_terrain_light_test.go`). Isso dá um fator máximo real de
   `≈1,293`, nunca `1,4` — o limite superior de `0,6`–`1,4` era, na
   prática, morto: nunca chegava a valer como teto.
2. **Mesmo o máximo real (`≈1,293`) já estoura um mapa claro.** Um quadro
   real gerado sobre um mapa base claro de verdade (`itaquara_mapa_2.mbtiles`,
   um estilo próximo de um tema claro do OpenStreetMap, fundo por volta de
   `240`–`245`) mostrou **10,4% dos pixels em branco puro** (`255,255,255`)
   e **23,5% em quase-branco** (`≥250`) — o fundo claro e as vias claras
   sobre ele colapsam juntos no branco nas encostas voltadas para a luz,
   exatamente o espelho da legibilidade que SC-002 protege do lado escuro,
   só que do lado claro, que o quickstart original nunca exercitou.

**Racional da faixa revisada**: `255 / 1,15 ≈ 221,7` — um fundo de até
`~221` não estoura mais em nenhum canal no fator mais claro (antes, o
teto efetivo de segurança era `255 / 1,293 ≈ 197`, já que `1,4` nunca era
alcançado mas `1,293` sim); um mapa muito claro (`~242`, o tom típico de
um tema claro do OSM) ainda pode saturar nas encostas mais extremas, mas
o excesso cai de `~58` (`313 − 255`, com `1,4`) para `~23` (`278 − 255`,
com `1,15`) — bem menos severo, e só nas encostas de inclinação mais
extrema exatamente alinhadas com o azimute da luz, não em qualquer
encosta moderada. Eliminar a saturação por completo exigiria um teto
perto de `1,05`, que apagaria quase todo o clareamento também nas
encostas moderadas — fora do que FR-001/FR-002 pedem (volume real e
perceptível). `0,75`–`1,15` é o ponto escolhido entre os dois: ainda dá
volume perceptível (confirmado pelos testes de
`Test_Scene_Render_TerrainLight`, que não dependem do valor exato da
faixa) sem o pior da saturação em mapas claros reais. O lado escuro
(`0,75`, era `0,6`) muda pela mesma razão, por simetria — o mínimo
geométrico real é `≈0,707` (o espelho exato do `1,293`), então `0,6`
também nunca era alcançado como piso de verdade.

**Alternativas consideradas**: manter `0,6`–`1,4` — rejeitada pelos dois
problemas acima, confirmados com dado real, não só sintético; uma faixa
que elimine toda saturação possível (`~0,95`–`1,05`) — rejeitada por
apagar o volume perceptível que é o próprio propósito da etapa
(FR-001/FR-002), trocando um defeito por outro; o item 1 anterior desta
pesquisa já havia cogitado e rejeitado algo próximo de `0,85`–`1,15` por
"dar pouco volume" — a revisão mantém o teto perto dali (`1,15`), mas
agora por uma razão diferente e melhor fundamentada (saturação medida em
dado real, não uma preferência estética a priori).

## 9. `RenderVersion` e reaproveitamento de conjuntos (FR-010)

**Decisão**: `domain.RenderVersion` sobe de `5` para `6`. Como
`NewFrameSetID`/`newFrameSetID` (`frame_set.go`) já incluem `RenderVersion`
no hash do `FrameSetID` — sem nenhuma mudança de código além do valor da
constante —, um conjunto de quadros desenhado pela versão anterior passa a
ser, automaticamente, "de outro conjunto": `render all` o recusa sem
`--overwrite` (`ErrFrameSetConflict`) e `fly --keep` redesenha os quadros e
o vídeo ao notar a mudança (o plano e o recorte continuam reaproveitados,
por não dependerem do desenho).

**Racional**: é o mesmo mecanismo mínimo já usado nas etapas 6, 8, 9, 11,
12 e 15 para qualquer mudança visual — nenhuma mudança de formato de
arquivo, nenhuma migração.

**Alternativas consideradas**: nenhuma — é o mesmo mecanismo que toda
mudança visual anterior já usou.

## 10. Determinismo aritmético (Constitution; CLAUDE.md "O desenho dos quadros")

**Decisão**: toda a aritmética nova — construção da pirâmide, busca por
nível, produto escalar, `clamp` do fator — usa só `+ − × ÷`, `Sqrt`,
`Floor`, `Abs`, `Min`, `Max`, `Log2` e comparação, com todo produto que
entra numa soma envolvido em `float64(...)` (nunca uma fusão
multiplicação-soma), seguindo a mesma disciplina que `frame_camera.go`/
`frame_surface.go`/`frame_imagery.go` já documentam no topo de cada
arquivo. `Sin`/`Cos` são usados só para a direção fixa da luz (item 2),
calculados uma vez (não por pixel, não por quadro) — já na lista de
operadores permitidos, e já usados por quadro em `newCamera` sem quebrar o
hash de referência entre arquiteturas.

**Racional**: preserva a garantia "mesmo plano, mesmo recorte, mesma
resolução, mesma aparência, mesma configuração de sobreposição → mesma
imagem, em qualquer máquina" (FR-009/SC-004) pelo mesmo raciocínio que
toda etapa anterior do renderizador já segue.

**Alternativas consideradas**: nenhuma — é a restrição de projeto, não uma
escolha desta etapa.
