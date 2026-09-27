# Pesquisa: Desenho dos Quadros do Voo

**Feature**: `005-frame-rendering` | **Data**: 2026-09-26

Cada item registra a decisão, o motivo e as alternativas descartadas. Onde a
decisão vem de código já existente, o arquivo é citado. Os valores numéricos
são os iniciais; os que são heurísticas vão para `RenderTuning` (item 22) e
podem ser ajustados sem mudar o comportamento descrito na spec.

## 1. Como desenhar: lançamento de raios sobre a grade de elevação, em Go puro, no domínio

**Decisão**: cada quadro é desenhado por **lançamento de raios** (ray casting)
sobre a superfície de terreno formada pelas grades de elevação do recorte: para
cada pixel sai um raio da câmera, o primeiro ponto em que ele encontra o
terreno dá a posição (e, por ela, a cor do mapa base), e nada mais. É código
puro Go, sem biblioteca gráfica, em `internal/domain`: projeção, oclusão,
texturização e composição são regra de negócio calculada sobre dados em memória
(Princípio IX). O que é E/S — decodificar as peças de imagem, codificar e gravar
PNG, ler o recorte — fica em adapters, atrás de portas (itens 7, 15 e 18).

**Motivo**:
- **Oclusão exata e sem costuras**: o raio acha o ponto mais próximo por
  construção; não há malha, cortes de visibilidade, nem frestas entre blocos
  ou entre grades de resolução diferente.
- **Marcar a falta de dado é trivial**: o pixel sabe em que célula da grade e
  em que peça caiu; "sem elevação" e "sem imagem" são uma consulta, e a
  contagem de quadros com buraco sai dos mesmos pixels que aparecem na imagem
  (FR-014).
- **Determinismo**: cada pixel só depende de si mesmo. Dividir o quadro em
  faixas de linhas entre goroutines não muda nenhum byte, e a ordem dos quadros
  também não (FR-015).
- **Custo previsível**: proporcional ao número de pixels e às células que cada
  raio atravessa, não ao tamanho do recorte (itens 5 e 23).

**Alternativas descartadas**:
- *Malha de triângulos + z-buffer*: exige decimação por distância (LOD) e
  cuidado com frestas entre blocos de níveis diferentes; a falta de dado por
  célula vira atributo de triângulo. Mais código para o mesmo resultado.
- *GPU, WebGL, navegador sem cabeça, three.js, ffmpeg*: violam o Princípio I
  (a regra sairia do núcleo), dificultam o Princípio V e tornam o
  byte-a-byte dependente de driver.
- *Biblioteca Go de rasterização 3D*: dependência nova sem ganho; o controle
  de cada operação em ponto flutuante (item 12) é o que garante o
  determinismo.
- *Supersampling ou MSAA nesta etapa*: multiplica o custo por 4 ou mais; a
  suavização de bordas do terreno fica para depois de o usuário ver os
  quadros (item 25, risco R4). O traçado e o marcador têm borda suavizada
  (item 9).

## 2. Geometria: plano local por quadro, sem curvatura, idêntico em qualquer lugar

**Decisão**: cada quadro é desenhado num **plano local** com origem no ponto
onde está a câmera daquele quadro, eixos `x` a leste e `y` a norte, em metros,
e altura `z`. A conversão entre graus e metros é a **equiretangular em torno
da latitude da câmera** (`φ₀`):

```text
x = (Δλ normalizado em [-180, 180)) · (π/180) · R · max(cos φ₀, 1e-6)
y = (φ − φ₀)                         · (π/180) · R
```

com `R` = `earthRadiusMeters` (o mesmo de `camera_projection.go`). É uma
transformação **afim**: numa grade de graus, as células são retângulos
alinhados aos eixos; procurar uma célula ou uma peça a partir de `(x, y)` é
somar e multiplicar, sem trigonometria por passo do raio.

**Motivo**:
- A etapa 3 planejou a câmera num plano local plano, sem curvatura, e o campo
  de visão que ela supôs é o mesmo aqui; usar o mesmo modelo evita uma segunda
  fonte de discrepância. O erro desta aproximação em relação à esfera é
  sub-percentual para o alcance de um quadro (a 60° de latitude e 11 km de
  distância a leste-oeste, 0,3%; a 80° e 50 km, 4,5% na borda) e igual para
  câmera, marcador, traçado e terreno: o quadro é **autoconsistente**, o que é
  o que se vê.
- **Neutralidade geográfica (Princípio IV)**: a fórmula é a mesma em qualquer
  hemisfério; a longitude é sempre relativa à da câmera, normalizada em
  `[-180, 180)`, então cruzar o meridiano de 180° não tem caso especial: uma
  grade que passa de 180° ganha `x` contínuo, sem salto (FR-028, SC-009).
  O piso `cos φ₀ ≥ 1e-6` é o mesmo que a etapa 4 usa (`minimumCosLatitude`)
  e mantém a conta finita nos polos.
- A curvatura da Terra (queda de ~200 m a 50 km) é ignorada, como na etapa 3.
  Um voo "de baixo alcance" não a percebe; documentar como limitação (R5).

**Alternativas descartadas**:
- *`LocalPlane` da etapa 3 (azimutal equidistante) por passo do raio*: exato,
  mas custa `sin`/`cos`/`asin` por passo (centenas de milhões por quadro).
- *Um plano único para o voo inteiro*: o erro cresce com a extensão do
  trajeto (até 2000 km é o teto da etapa 3); um plano por quadro limita o
  erro ao alcance daquele quadro.
- *Esfera com raio curvo*: precisaria de intersecção raio-esfera mais grade
  esférica; complexidade sem ganho visível no alcance de um quadro.

## 3. A câmera do quadro: pose, campo de visão, alvo e altura absoluta

**Decisão**:

- **Base da câmera** (`heading` `h`, `tilt` `t`, em radianos, do quadro do
  plano): `frente f = (sin h·cos t, cos h·cos t, −sin t)`,
  `direita r = (cos h, −sin h, 0)`, `cima u = (sin h·sin t, cos h·sin t, cos t)`.
  Sem rotação em torno do eixo de visão (o horizonte fica horizontal).
- **Campo de visão**: **vertical, fixo, 45°** — o mesmo valor que a etapa 3
  supôs para enquadrar a abertura (`CameraTuning.OverviewVerticalFOVDegrees`)
  e a etapa 4 para o nível de detalhe. É **uma constante só** em `config`,
  lida pelos três (evita divergir). O horizontal segue da proporção: 
  `tan(FOVh/2) = tan(FOVv/2) · largura/altura`. Pixels quadrados; distância
  focal em pixels `f_px = (altura/2) / tan(FOVv/2)`. Por isso o enquadramento
  vertical é o mesmo em qualquer resolução (FR-019) e uma imagem mais larga só
  mostra mais dos lados.
- **Raio do pixel `(i, j)`** (centro em `+0,5`):
  `d ∝ f + ((i+0,5−W/2)/f_px)·r − ((j+0,5−H/2)/f_px)·u`, normalizado. A
  projeção de um ponto para a tela usa a mesma câmera, o mesmo `f_px` e o mesmo
  `+0,5` — é o que garante o marcador a ≤ 1 pixel de onde a projeção o põe
  (SC-004).
- **Alvo do quadro**: a etapa 3 põe a câmera atrás do alvo, ao longo do
  rumo, e mede a altura **em relação ao ponto observado** (o alvo, que é o
  marcador suavizado, não o marcador). O plano não guarda o alvo, mas ele se
  recupera do próprio quadro: `alvo = câmera + (altitude / tan t)·(sin h,
  cos h)` no plano local. Com `t < 1°` (`MinTiltForTargetDegrees`; os
  níveis da etapa 3 vão de 25° a 65° e a suavização só interpola entre
  eles e a visão geral de 60°) o alvo é o marcador.
- **Altura absoluta da câmera**: `H = terreno(alvo) + altitude_do_quadro`,
  e, para nunca ficar sob o terreno que enxerga (FR-009):
  `H = max(H, terreno(câmera) + MinCameraClearanceMeters)` (2 m). O terreno
  usado é a superfície do item 4, com o preenchimento do item 6 quando a
  amostra ali não tem valor. Se a posição estiver fora de todas as grades
  (só ocorre na meia célula da borda), usa-se a grade mais próxima com o ponto
  puxado para dentro dela.

**Motivo**: as fórmulas repetem `CameraView.Pose` ao contrário e não pedem
nada além do que o quadro do plano já traz. Uma constante única de campo de
visão é o que mantém as três etapas consistentes.

**Alternativas descartadas**:
- *Usar o marcador como "chão" de referência*: o alvo e o marcador diferem
  até `MaxTargetSpeedInDistances × distância` (uma distância inteira, no
  pior caso); o alvo é o que a etapa 3 usou.
- *Gravar o alvo no plano (formato v2)*: muda o contrato da etapa 3 sem
  necessidade — o valor sai do próprio quadro.
- *Sem "folga" sobre o terreno*: com o chão sob a câmera mais alto que o
  chão sob o alvo (câmera atrás de uma subida), a câmera ficaria dentro do
  morro e o quadro seria inútil.

## 4. A superfície do terreno: bilinear entre os centros das células, esticada até as bordas

**Decisão**: cada grade do recorte é uma **superfície bilinear** cujos nós são
os centros das células (a mesma convenção do contrato do recorte:
`lat = north_lat − (r+0,5)·cell_lat`). Fora do retângulo dos centros, até a
borda da célula, os índices são **presos** ao primeiro/último nó (extensão
plana): assim cada grade cobre exatamente o seu retângulo de células
(`north_lat`, `west_lon`, `rows × cell_lat`, `cols × cell_lon`) e duas grades
vizinhas (regiões disjuntas do recorte) se tocam sem fresta horizontal. Um
degrau de altura entre grades na emenda não é desenhado (R6).

Em coordenadas de nó, `a = (x − x₀)/cx − 0,5` (coluna), `b = (y₀ − y)/cy − 0,5`
(linha) e a altura é a interpolação bilinear dos quatro nós vizinhos, com
`N(r, c)` preso a `[0, rows−1] × [0, cols−1]`.

**Motivo**: a etapa 4 lê a célula que contém o ponto (sem interpolar), o que é
certo para *consultar* um valor e para *não inventar* dado; para *desenhar*, a
interpolação entre amostras dá um relevo contínuo em vez de degraus do
tamanho da célula. É uma escolha de desenho, não de dado: nenhuma amostra
nova aparece, e o que se marca como "sem elevação" continua sendo a célula
original (item 6).

**Alternativas descartadas**:
- *Terreno em degraus (célula constante)*: fiel à consulta, mas visualmente
  áspero e sem inclinação para o raio "escorregar".
- *Interpolação bicúbica*: overshoot invisível em dado ruidoso e custo maior,
  sem ganho para 30 m de resolução.

## 5. Como o raio encontra o terreno: travessia de células (DDA) com poda por altura

**Decisão**: por grade, o raio é recortado ao retângulo da grade (método das
lajes, em `x` e `y`); dentro dele, percorre-se **quadra a quadra** (a quadra é o
espaço entre quatro nós vizinhos, no espaço `(a, b)` do item 4) por DDA
(Amanatides-Woo). Em cada quadra, com `f(t) = z_raio(t) − z_terreno(t)`:

1. se `f(t_entrada) ≤ 0`, o raio já entrou no terreno: acerto em `t_entrada`
   (é o que acontece ao entrar numa grade vizinha mais alta);
2. senão, se `f(t_saída) ≤ 0`, há acerto na quadra e `t` é refinado por
   **16 bisseções** (a altura do terreno na quadra é quadrática em `t`);
3. senão, segue para a quadra vizinha.

**Poda por altura**, sem estrutura extra: cada grade guarda `zmax` (maior
altura, já preenchida). Se o raio está acima de `zmax` e sobe ou fica
horizontal, ele não encontra a grade; se está acima e desce, a travessia
começa no `t` em que o raio atinge `zmax`. Essa poda só pula onde o raio
certamente está acima de todo o terreno, então **não muda nenhum resultado**;
qualquer aceleração futura (por exemplo, pirâmide de máximos) tem de manter
essa propriedade e é verificada por teste de igualdade byte a byte.

**Grades múltiplas** (regiões de fontes diferentes): as grades cujo retângulo
o raio cruza são percorridas na ordem do `t` de entrada (empate: índice da
grade); vale o primeiro acerto. Como as regiões são disjuntas, os intervalos
não se sobrepõem.

**Motivo**: o passo por quadra não perde feição mais fina que a grade
(diferente de passos fixos ou proporcionais à distância) e o custo é o número
de quadras atravessadas: ~20 a 50 num quadro típico (câmera a 600 m,
células de 30 m), algumas centenas nos raios quase horizontais.

**Alternativas descartadas**:
- *Passo fixo ou proporcional à distância, com bisseção*: subamostra
  cumes finos ao longe (a 0,5% do alcance o passo passa de 7 pixels) e
  cintila entre quadros.
- *Pirâmide de máximos (relief mapping hierárquico)*: mais rápida, mais
  código; adiada para depois da medição (R1). A poda por `zmax` é o ganho
  barato.

## 6. Amostras sem valor: geometria preenchida, marcação pela célula original

**Decisão**: a marcação de "sem elevação" é decidida pela **célula original**
(onde o valor é `NaN`): se o ponto de acerto cai numa célula sem valor, o
pixel é pintado com a marcação de "sem elevação" (item 8) e conta como buraco
de elevação. A **geometria** sob essas células precisa de uma altura para que
o raio e a oclusão funcionem, e a altura de referência da câmera pode cair
sobre elas; a regra, determinística e documentada, é: **a altura da amostra
com valor mais próxima**, em distância de quarteirão (L1, em células), por
propagação em **duas passadas** (ida e volta) na grade, que é exata para L1 e
linear no tamanho da grade; empate resolvido pela ordem das passadas (linha
antes de coluna, do noroeste ao sudeste e de volta). Uma grade **inteira** sem
valor recebe a menor elevação com valor do recorte (que existe: um recorte
sem nenhuma é recusado, item 16). Só se aloca a cópia preenchida para grades
que têm buracos.

**Motivo**: é a regra mais simples que não usa zero, não inventa relevo
(ficam as alturas vizinhas, marcadas como "sem dado") e cabe em
`O(linhas × colunas)`. O quadro nunca mostra a altura preenchida como se fosse
dado, porque a marcação cobre a região.

**Alternativas descartadas**:
- *Deixar buraco sem superfície (o raio atravessa)*: o raio "vaza" e mostra o
  terreno de trás ou o fundo: confunde com "fora do recorte" e quebra a
  garantia de que o fundo neutro nunca é buraco (FR-013).
- *Média dos vizinhos ou interpolação*: inventa mais do que "repetir o
  vizinho mais próximo" e custa mais.
- *Preenchimento com o mínimo global*: afunda um vazio de montanha (SRTM) ao
  nível do mar; o vizinho mais próximo mantém a paisagem.

## 7. Imagem do mapa: peça pelo Web Mercator, mip por peça e decodificação atrás de porta

**Decisão**:

- **Peça do pixel**: o ponto do terreno `(lat, lon)` vira coordenadas Web
  Mercator normalizadas `u = (lon + 180)/360`, `v = 1/2 − ln(tan(π/4 + φ/2))/(2π)`
  (só `log` e `tan`, ver item 12); com o nível `z` do conjunto (`level.chosen`),
  a peça é `(z, ⌊u·2^z⌋, ⌊v·2^z⌋)` e o texel `((u·2^z − x)·256, (v·2^z − y)·256)`.
  O esquema é XYZ, como o recorte (a conversão de TMS foi feita na etapa 4).
- **Qual conjunto de peças**: percorrem-se os conjuntos do recorte na ordem
  de `base_map[]`; vale a peça **presente** do primeiro conjunto que a tem;
  se nenhum tem, e um conjunto a lista como **ausente**, é "peça ausente";
  se nenhum a conhece (fora do intervalo de peças pedidas, ou além de
  ±85,0511° onde não existem peças), também é "sem imagem de mapa" — o mesmo
  sinal e a mesma contagem (FR-012, FR-013).
- **Filtro**: **trilinear** — bilinear em dois níveis de uma pirâmide de mips
  por peça (caixa 2×2 com aritmética de inteiros: `(a+b+c+d+2)>>2`), com
  `λ = log₂(max(ρ, 1))`, `ρ = passo_angular_do_pixel · t / (m/texel do nível na
  latitude do ponto) / √max(|d_z|, 0,1)`. As amostras que caem numa peça
  vizinha ausente usam o texel mais próximo da peça do centro. O mip do nível
  0 é a própria peça; sem o filtro, o mapa cintila em movimento (o mapa foi
  dimensionado para o pixel na menor distância; ao longe cada pixel cobre
  muitos texels).
- **Decodificação**: uma porta nova, **`TileDecoder`**, no domínio
  (`Decode(format string, data []byte) (TileImage, error)`), implementada por
  `tiledecoder.NewRaster()` sobre `image/png`, `image/jpeg` e
  `golang.org/x/image/webp` (a dependência `x/image` já está no `go.mod` desde a
  etapa 4). Cada peça é decodificada **uma vez**, na primeira vez que um raio
  a alcança, e guardada com seus mips num cache com orçamento de memória
  (256 MiB; descarta as mais antigas). O cache muda tempo, nunca resultado.
- **Transparência**: pixels com alfa < 255 são compostos sobre a cor do fundo
  (item 8): peça transparente parece "sem mapa", não uma cor inventada.
- **Peça ilegível** (bytes que não são imagem): `ErrSliceFileInvalid`
  identificando registro, nível e posição; os quadros já publicados
  continuam válidos (História 7, cenário 10).

**Motivo**: decodificar formato de imagem é E/S de formato de arquivo, como
o parser de GPX: fica atrás de porta (Princípio II). O domínio recebe pixels.

**Alternativas descartadas**:
- *Decodificar todas as peças de uma vez, na leitura do recorte*: milhares de
  peças de 60 KB viram gigabytes de pixels; o recorte pode ter 256 MiB.
- *Amostragem por vizinho mais próximo, sem mips*: cintilação forte.
- *Mips a partir do nível mais fino do arquivo original*: o recorte só
  carrega o nível escolhido.

## 8. Marcações de falta, fundo, traçado e marcador: cores fixas, padrões em tela

**Decisão**: as cores e padrões são **fatos do formato da imagem**, constantes
do domínio documentadas em `contracts/frame-files.md`, não ajuste:

| Elemento | Aparência |
|---|---|
| Fundo (fora do recorte; céu) | cor lisa `#20262E` |
| "Sem imagem de mapa" | hachura diagonal de 45°, período de 12 px na tela, alternando `#C8C8C8` e `#6E6E6E` |
| "Sem elevação" | xadrez de quadrados de 12 px na tela, alternando `#FF00FF` e `#3A003A` |
| Traçado | linha de largura `max(2, 0,5% da altura)` px em `#FFB000`, com contorno de 1 px em `#101010` |
| Marcador | disco de raio `max(4, 1,2% da altura)` px em `#E5252A`, com anel branco de `max(1,5, 0,3% da altura)` px |

Os padrões são funções da posição do **pixel na tela** (não do mundo): não
cintilam com a distância e não dependem de projeção. Quando um pixel cai numa
célula sem elevação *e* numa peça ausente, vale "sem elevação" (a geometria é a
que falta), e o quadro entra nas duas contagens só se os dois tipos de pixel
aparecerem (item 10).

**Motivo**: precisam ser inconfundíveis entre si e com qualquer imagem de mapa
real. Magenta em xadrez e cinza hachurado quase nunca aparecem num mapa; o fundo
é escuro e liso, distinto de ambos.

**Alternativas descartadas**: *cor lisa para a falta* (confunde com água ou
mata num mapa real), *padrão no mundo* (aliasing ao longe), *degradê de céu e
horizonte* (não é dado, só decoração; pode vir com a iluminação depois).

## 9. Traçado e marcador: projeção, oclusão e borda suave

**Decisão**:

- **Traçado do quadro `k`**: a poligonal das posições do marcador dos quadros
  `0…k` do plano (Clarificação de 2026-09-26), cada vértice na altura do
  terreno sob ele (item 4) mais 0,3 m. Cada segmento é projetado pela mesma
  câmera; os que cruzam o plano de aproximação (`0,1 m` à frente da câmera)
  são recortados nele; os que estão inteiros atrás dela, descartados.
- **Desenho**: para cada segmento, cápsula com borda suave (cobertura =
  `clamp(meia_largura + 0,5 − distância_ao_segmento, 0, 1)`), em dois passes:
  contorno e miolo.
- **Oclusão**: a profundidade do pixel de terreno (distância `t` do raio) é
  guardada num buffer; o traçado só é desenhado onde sua distância à câmera é
  `≤ t_terreno + folga`, com `folga = 1 m + 0,2% de t`. A folga absorve o
  levantamento de 0,3 m e o erro de interpolação. O **marcador** decide sua
  visibilidade **uma vez só, no pixel do centro** (o disco é um símbolo de
  tamanho fixo na tela, e o chão sob a metade de baixo dele está mais perto da
  câmera que o ponto que ele marca: testar pixel a pixel cortava o disco ao
  meio — visto no primeiro quadro real); visível, é desenhado inteiro.
- **Marcador**: na posição do marcador **do quadro**, na altura do terreno
  sob ele (mais 0,3 m), desenhado por último, sobre o traçado.
- Quando o alvo do desenho está atrás da câmera ou fora do campo de visão, não
  há marcador na imagem; não é erro (Casos Extremos da spec).

**Motivo**: o traçado é "colado" no terreno pela altura, e a oclusão é a
mesma do terreno (FR-011). A borda suave é barata e, sem ela, a linha pisca
entre quadros.

**Alternativas descartadas**: *desenhar o traçado na textura do mapa*
(depende de resolução da peça, embaça ao perto), *marcador sempre por cima*
(perde a noção de profundidade e mostra o marcador atrás de um morro).

## 10. Contabilização: a partir dos pixels de terreno que aparecem

**Decisão**: durante o desenho, cada pixel cujo raio acertou terreno é
classificado em **imagem**, **sem imagem de mapa** ou **sem elevação**, e o
quadro guarda dois booleanos: `MapHole` (pelo menos um pixel "sem imagem de
mapa") e `ElevationHole` (pelo menos um pixel "sem elevação"). A contagem é
feita sobre o terreno, **antes** do traçado e do marcador, então um
marcador sobre a região não a esconde da contagem. O fundo neutro (raio que
não acertou nada) não é buraco. O resumo soma os booleanos dos quadros
desenhados **na execução** (Clarificação e Suposição da spec).

**Motivo**: é literalmente "a parte visível do terreno inclui" (FR-014): a
contagem e a imagem vêm do mesmo cálculo, então não podem discordar (SC-005).
Um único pixel na borda do horizonte conta; é uma regra simples e testável.

**Alternativas descartadas**: *contar por interseção de retângulos
(quadro × peça ausente)* — divergiria da imagem quando a peça está oculta
por um morro; *limiar mínimo de pixels* — arbitrário.

## 11. Resolução: `LARGURAxALTURA`, limites documentados

**Decisão**: a resolução é `Resolution{Width, Height}` de domínio, validada por
`NewResolution` (e por `ParseResolution` para o texto `1080x1920`, que a CLI
recebe numa flag `--resolution`): largura e altura **pares**, cada uma de
**180 a 3840**, com no máximo **8 294 400 pixels** (o de 3840 × 2160). Padrão
**1080 × 1920**. Isso reduz, no planejamento, os limites de referência da spec
(320–7680 × 180–4320): um desenhista de software com 8K por quadro passaria
de 130 MB por buffer e minutos por quadro; 4K é o teto sensato agora, e é uma
constante fácil de subir. Retrato (1080 × 1920) e ultralargo (3840 × 1080) são
aceitos.

**Erro**: `ErrInvalidResolution`, com o valor recebido e os limites; texto que
não é `NxM` de inteiros também.

**Motivo**: uma flag só (mais simples de validar e de documentar que duas);
dimensões pares porque a etapa de vídeo seguinte usa codecs que exigem.

## 12. Determinismo: aritmética sem fusão, funções matemáticas escolhidas e paralelismo por pixel

**Decisão**:

1. **Sem fusão multiplicação-soma (FMA)**: o compilador Go pode fundir
   `x*y + z` em arm64 e não em amd64, mudando os últimos bits. Todo produto
   que entra numa soma é envolvido em `float64(...)`, como o código já faz
   em `camera_projection.go`. Cobre-se com um teste que renderiza um quadro
   pequeno e compara os bytes com uma referência gravada (`golden`).
2. **Funções matemáticas**: só `+ − × ÷`, `math.Sqrt`, `Floor`, `Abs`, `Min`,
   `Max`, `Sin`, `Cos`, `Tan`, `Atan2`, `Log`, `Log2` (implementadas em Go
   puro em amd64 e arm64) — **nunca `Exp`, `Pow` nem `Sinh`**, que têm
   assembly por arquitetura. Onde precisar de uma potência de dois, `math.Ldexp`.
3. **Sem estado compartilhado que influencie valores**: o cache de peças e o
   buffer de profundidade não mudam nenhum valor; a ordem de iteração de
   `map` nunca é usada (índices ordenados).
4. **Paralelismo**: o quadro é dividido em faixas de 16 linhas
   distribuídas a `Workers` goroutines; cada pixel depende só de si mesmo,
   e os dois booleanos do quadro são um OU. O número de goroutines não
   altera nenhum byte (teste: 1 × 8 goroutines).
5. **PNG**: `image/png` com `DefaultCompression` e o mesmo encoder é
   determinístico **no mesmo binário**; entre versões do Go o `compress/flate`
   pode mudar os bytes comprimidos (não os pixels). A igualdade byte a byte é
   garantida, como nas etapas 3 e 4, para o mesmo binário e plataforma; entre
   arquiteturas, os pixels devem ser iguais (itens 1 e 2) e a igualdade dos
   bytes vale enquanto o `compress/flate` for o mesmo.

**Motivo**: FR-015 e SC-002. As regras 1 e 2 são baratas e evitam a única
fonte de divergência entre arquiteturas que o código controla.

## 13. Sequência de quadros, cancelamento e o que fica em paralelo

**Decisão**: os quadros pendentes são desenhados **em ordem crescente do
número**, um de cada vez, cada um usando todos os núcleos (item 12.4). O
progresso é simples (`concluídos/total`) e a interrupção termina no fim de
uma faixa de linhas do quadro em curso (o quadro parcial é descartado, nunca
publicado). O cancelamento chega por `context.Context` (biblioteca padrão, não
é E/S: Princípio II) criado pela CLI a partir de `SIGINT`/`SIGTERM`.

Não há *pipeline* entre desenhar o quadro `k+1` e codificar/gravar o `k`
(a codificação PNG de 1080p leva ~50 ms contra ~1 s de desenho); se a medição
mostrar que vale, é uma otimização localizada no serviço, sem mudar resultado.

**Alternativas descartadas**: *vários quadros em paralelo* (memória
multiplica por quadros e o progresso deixa de ser contíguo, complicando a
retomada), *processos separados por faixa de quadros* (o usuário faz isso
sozinho, rodando o comando de novo: a retomada trata).

## 14. Identificação do plano: hash do conteúdo, calculado no domínio (Clarificação 1)

**Decisão**: `CameraPlan.ID() string` — SHA-256, em hexadecimal minúsculo
(64 caracteres), de uma codificação canônica do plano: os parâmetros efetivos
(duração em nanossegundos, taxa de quadros como `Float64bits`, níveis de
distância e inclinação) e, para cada quadro em ordem, `index`, `phase` e todos
os valores **quantizados aos passos do plano** (coordenadas em `1e-7`°,
comprimentos e ângulos em `1e-3`, como inteiros `int64` em ordem de bytes
fixa). O resumo não entra: é derivado dos quadros.

`GeoSliceService.Generate` grava `plan.ID()` em `GeoSlice.PlanID` e o
exportador o escreve no manifesto (`plan_id`, item 15 e
`contracts/slice-file-change.md`). Esta etapa compara o `plan_id` do recorte
com `plan.ID()` do plano informado.

**Motivo**: é derivada do conteúdo **decodificado**, então não depende de
espaços, ordem dos campos ou do arquivo: um plano gerado na memória e o
mesmo plano relido de arquivo têm a mesma identificação, e reexportar o
plano com a mesma ferramenta não muda nada. Fica no domínio porque é regra
(o que é "o mesmo plano"), e `crypto/sha256` e `encoding/binary` são
biblioteca padrão pura (Princípio II).

**Alternativas descartadas**: *hash dos bytes do arquivo do plano* (dependeria
do adaptador e mudaria com formatação; e o plano em memória não teria
identificação), *comparar só a área* (a alternativa A da clarificação,
descartada pelo usuário).

**Consequência (a única mudança de comportamento anterior)**: `geodata slice
--export` passa a gravar o campo; recortes exportados antes não o têm e são
recusados por esta etapa (`ErrSliceFileInvalid`, "the slice has no plan
identification; generate it again"). `format_version` continua 1
(o campo é acrescentado, não muda o significado de nenhum existente, como o
contrato já prevê).

## 15. Leitura do recorte: adaptador ZIP, validação completa antes de desenhar

**Decisão**: nova porta `GeoSliceReader` em `geo_slice.go` (junto de
`GeoSliceExporter`), implementada por `zipfile.NewGeoSliceReader()`, que lê o
ZIP com `archive/zip` (acesso aleatório, sem extrair para o disco) e devolve
um `domain.GeoSlice` completo, reconstruído por `NewGeoSlice` (que reordena e
recalcula o resumo). Validações (todas `ErrSliceFileInvalid`, com a causa):

- não é um ZIP, ou está truncado ou corrompido (`zip.ErrChecksum` na leitura
  de uma entrada);
- entrada com método diferente de *store*, ou nome duplicado;
- `manifest.json` ausente, JSON inválido, campo obrigatório ausente,
  `plan_id` ausente (item 14) ou que não é hexadecimal de 64 caracteres;
- `format_version` diferente de 1 → `ErrSliceFormatVersionUnsupported`
  ("found 2, accepted: 1");
- uma peça ou grade citada no manifesto que não existe no ZIP, ou cujo
  tamanho difere (`bytes`; `rows × cols × 4`);
- entradas no ZIP que o manifesto não cita;
- o resumo do manifesto (`tile_count`, `missing_tile_count`, `sample_count`,
  `no_value_sample_count`, faixa de elevação e `size_bytes`) diferente do
  recalculado a partir do conteúdo (as garantias 1 a 5 de `slice-file.md`);
- `level.chosen` fora de `[min, max]`; `source` fora de `sources`.

O leitor também calcula o **`ContentID`**: SHA-256 do arquivo inteiro (lido em
sequência), guardado em `GeoSlice.ContentID` (não é exportado; identifica
"este recorte" para o conjunto de quadros, item 17). O arquivo é limitado a
256 MiB + 16 MiB de manifesto, o teto do recorte, para que um ZIP forjado não
esgote a memória; um recorte maior é `ErrSliceFileInvalid`.

**E/S**: arquivo inexistente ou sem permissão é um erro de E/S comum (sem
sentinela, código 4), como para o plano na etapa 4.

**Alternativas descartadas**: *desempacotar num diretório temporário* (mais
E/S, um estado a limpar), *carregar peças sob demanda* (o recorte cabe em
memória por construção — 256 MiB — e a leitura sequencial é rápida; fica como
otimização se o teto subir).

## 16. Correspondência, cobertura e formato: ordem das verificações

**Decisão**: a ordem de recusa, do mais barato ao mais caro, é (a) resolução,
(b) plano (`Load`, códigos 17/18), (c) número do quadro, (d) leitura do
recorte (27/28), (e) **correspondência** (`slice.PlanID != plan.ID()`,
29), (f) **cobertura** (a área do recorte contém `plan.AreaOfInterest(tuning)`,
30; é uma proteção para um recorte do mesmo plano gerado com outra margem),
(g) **formato de peça** (31), (h) **elevação** (32), (i) destino
(35 a 37). Cada uma é um método de domínio (`GeoSlice.EnsureMatches`,
`EnsureCovers`, `EnsureDrawable`), chamado em ordem pelo serviço.

**Formato de peça** (31): só `png`, `jpg` e `webp` (o que `TileDecoder`
sabe); `pbf`/`mvt` são recusados com a mensagem de peça vetorial; qualquer
outro, com "format not supported for drawing". A decisão vem do manifesto
(`tile_format` de cada `base_map[]`), antes de qualquer peça ser decodificada.

**Elevação** (32): recorte em que `SampleCount == NoValueSampleCount` (nenhuma
amostra com valor) é recusado.

`BoundingBox.ContainsBox` (novo) compara caixas com o desembrulho de
longitude que `BoundingBox` já usa (`longitudeSpans`), então cobre o
antimeridiano.

## 17. Conjuntos de quadros: cada imagem carrega a identificação do conjunto

**Decisão (muda a redação da spec, ver Suposições)**: em vez de um arquivo de
registro no diretório de destino (que o FR-022 previa como possibilidade), a
identificação do conjunto **vai dentro de cada imagem PNG**, num bloco de texto
`tEXt` com a palavra-chave `Sobrevoo` e o valor `frame-set=<id>`, logo após o
cabeçalho `IHDR`. O `id` (`FrameSetID`) é o SHA-256 (hex) de:

```text
"sobrevoo-frames" | RenderVersion | plan.ID() | slice.ContentID | largura | altura | RenderTuning.Fingerprint()
```

com `RenderVersion` uma constante inteira que sobe sempre que o algoritmo,
as cores ou os padrões mudam de forma visível, e `RenderTuning.Fingerprint()`
as constantes de ajuste que afetam a aparência (campo de visão, folga da
câmera, etc.). Assim, **mudar o plano, o recorte, a resolução, a versão do
desenho ou o ajuste muda o conjunto** e um diretório de outro conjunto é
reconhecido.

**Por que não um arquivo de registro**: (1) não há estado que possa
discordar das imagens (um registro atualizado no meio e uma interrupção
gerariam dúvida sobre quais quadros são do conjunto novo); (2) o
reconhecimento do que é "quadro seu" é exato: a marca está na imagem, não
no nome; (3) o diretório fica só com imagens, que o passo seguinte lê por
ordem de nome.

**Classificação do diretório** (`FrameDirectory.Plan`, regra de domínio, dada
a listagem que o repositório devolve — `pngfile.FrameRepository.Inspect`):

- Considera-se "arquivo de quadro" o que casa com `frame_NNNNNN.png` (6
  dígitos: o plano da etapa 3 tem no máximo 432 000 quadros).
- Para cada um, o repositório informa: número, se é **nosso** (PNG com o
  bloco `Sobrevoo`), o `id` do conjunto, e se está **completo** (assinatura,
  `IHDR` com a largura e a altura do conjunto e o final `IEND` no fim do
  arquivo — detecta truncamento).
- **Mantidos**: nossos, do mesmo conjunto, completos, com número dentro do
  plano. **A desenhar**: os que faltam, ou nossos/mesmo conjunto/incompletos.
- **Conflito (`ErrFrameSetConflict`), sem sobrescrita**: existe algum arquivo
  nosso de **outro** conjunto (qualquer número), ou um arquivo com nome de
  quadro que **não** é nosso e que cairia num número do plano. A mensagem
  conta quantos e diz como proceder.
- **Com sobrescrita**: todos os quadros pedidos são redesenhados e
  substituídos; arquivos **nossos** de outro conjunto com número **fora** do
  plano são **removidos** (é a garantia de que a etapa seguinte não junta
  quadros de outro voo); arquivos que não são nossos nunca são removidos
  (os de nome colidente dentro do plano são substituídos, como qualquer
  sobrescrita).
- Arquivos que não casam com o padrão de nome são invisíveis à ferramenta.

**Ordem das operações com sobrescrita**: primeiro remove os nossos de outro
conjunto **fora** do plano; depois desenha e substitui em ordem. Uma
interrupção no meio deixa um diretório em que cada quadro é ou do conjunto novo
ou do antigo, e a próxima execução (sem sobrescrita) recusa por conflito: o
usuário repete com sobrescrita. Não há estado escondido.

**Quadro isolado**: recebe a **mesma** marca (para ser idêntico byte a
byte ao quadro do conjunto, FR-018a), mas nunca consulta nem altera um
diretório de quadros.

**Alternativas descartadas**: *arquivo `frames.json` no diretório* (estado
duplicado, risco de o registro e as imagens divergirem, e a remoção de
"sobras" dependeria de um arquivo), *marcar pelo nome do arquivo* (não
distingue os quadros de outro voo, nem os de outra ferramenta com o mesmo
padrão).

## 18. Gravação atômica de cada imagem: reuso de `atomicfile`, com `fsync`

**Decisão**: cada PNG é gravado por `atomicfile.Publish` (a etapa 4 já o
extraiu para o plano e o recorte): arquivo temporário no mesmo diretório,
publicado por `rename`, ou por `link` exclusivo quando não se pode
sobrescrever. **Novo**: `Publish` passa a chamar `Sync` no arquivo temporário
antes de fechar, para que uma queda de energia depois do `rename` não deixe
um arquivo vazio com o nome do quadro. Vale para os três usos (plano, recorte,
quadros) e não muda comportamento observável; o custo (poucos ms por arquivo)
é irrelevante diante do desenho. Os temporários órfãos de uma interrupção
(`.sobrevoo-*.tmp`) não casam com o padrão de nome de quadro e são
ignorados; o próximo `Publish` no diretório não os toca (a limpeza de órfãos
não é escopo).

A **verificação de completude** do item 17 (assinatura, `IHDR`, `IEND` no
fim) cobre o resto: um quadro publicado mas truncado por falha externa é
redesenhado na retomada.

## 19. Progresso, interrupção e códigos de saída

**Decisão**:

- **Progresso**: o serviço chama um `func(RenderProgress)` (`Done`, `Total`,
  `Elapsed`) a cada quadro concluído (FR-020). A CLI o mostra em `stderr`:
  numa linha que se reescreve (`\r`) quando `stderr` é um terminal, e uma
  linha nova a cada 10 quadros (e no último) quando não é, para não poluir
  um log.
- **Interrupção**: a CLI cria o contexto com `signal.NotifyContext(SIGINT,
  SIGTERM)`. Ao ser cancelado, o serviço interrompe o quadro em curso,
  devolve o `RenderSummary` do que ficou pronto com `Interrupted = true` e o
  erro `ErrRenderInterrupted`; a CLI imprime o resumo e sai com o código 38.
- **Resumo em falha no meio**: se um quadro falha por E/S (disco cheio), o
  serviço devolve o resumo parcial e o erro; a CLI imprime o resumo antes da
  mensagem quando algum quadro foi concluído.
- **Códigos** 27 a 38 (`contracts/cli.md`), a continuar os 1 a 26.

## 20. A CLI: `render frame` e `render all`

**Decisão**: um comando pai `render` (como `geodata`) com dois filhos, cujas
formas são diferentes o bastante para não misturarem `--output`:

```text
sobrevoo render frame <plano> <recorte> --number <n> --output <arquivo.png> [--resolution LxA] [--overwrite]
sobrevoo render all   <plano> <recorte> --output <diretório>              [--resolution LxA] [--overwrite]
```

`--number` é lido como **texto** e validado por `ParseFrameNumber` (domínio),
para que "3,5" ou "abc" sejam `ErrFrameOutOfRange` (código 33), e não um erro
de uso do Cobra, como pede a spec. O mesmo vale para `--resolution` e
`ErrInvalidResolution`. Ausência das flags obrigatórias e argumentos faltando
continuam sendo erro de uso (código 2). O resumo vai para `stdout`; o progresso,
para `stderr` (item 19).

**Alternativas descartadas**: *uma flag `--frame` no mesmo comando de todos os
quadros* (o `--output` significaria arquivo ou diretório conforme a flag),
*posicional para o número* (mais opaco que uma flag nomeada).

## 21. Arquivo e nome de imagem

**Decisão**: PNG, RGB de 8 bits por canal, sem canal alfa, sem entrelaçamento,
nome `frame_NNNNNN.png` (número do plano, 6 dígitos com zeros à esquerda: a
ordem alfabética é a numérica, FR-016). O quadro isolado vai para o caminho que
o usuário deu, com o mesmo conteúdo. Detalhes em `contracts/frame-files.md`.

## 22. Constantes de ajuste (`RenderTuning`) e o que vai para a configuração

**Decisão**: `domain.RenderTuning`, injetada (Princípio VIII), com tipos
próprios em `config.RenderTuning` (que **não importa o domínio**) e mapeada em
`cmd/sobrevoo/config_mapping.go`:

| Campo | Valor inicial | O quê |
|---|---|---|
| `VerticalFOVDegrees` | 45 | campo de visão vertical (a mesma constante de `CameraTuning.OverviewVerticalFOVDegrees`) |
| `MinCameraClearanceMeters` | 2 | folga mínima da câmera sobre o terreno (item 3) |
| `MinTiltForTargetDegrees` | 1 | abaixo disso o alvo é o marcador (item 3) |
| `TrailLiftMeters` | 0,3 | quanto o traçado e o marcador levantam do chão (item 9) |
| `DepthBiasMeters`, `DepthBiasRatio` | 1, 0,002 | folga do teste de profundidade (item 9) |
| `TileCacheBytes` | 256 MiB | orçamento do cache de peças decodificadas (item 7) |
| `Workers` | `runtime.NumCPU()` | goroutines por quadro (lido na configuração, não no domínio: Princípio VIII) |

`RenderVersion`, as cores, os padrões, os tamanhos relativos do traçado e do
marcador e os limites de resolução são **constantes do domínio**
(`contracts/frame-files.md`): definem o que a imagem significa, e não são
ajuste. O padrão de resolução (1080 × 1920) é dado da configuração
(`config.RenderDefaults`), como os padrões do plano.

## 23. Desempenho e memória: orçamento e o que medir

**Metas** (spec, SC-001, SC-011 e SC-012): um quadro isolado em ≤ 10 s; nenhum
quadro além de 15 s; 1350 quadros em ≤ 45 min (média de 2,0 s por quadro),
em 1080p, em computador pessoal comum.

**Orçamento**: ~2 milhões de raios; ~35 quadras por raio em média (poda
por `zmax` e acerto cedo) → ~70 milhões de passos de DDA por quadro, ~10 ns
cada, → ~0,7 s de CPU, dividido por `Workers`. Amostragem trilinear:
2 milhões × 8 texels. Traçado: até 1350 cápsulas. Codificação PNG: ~50 ms. A
soma cabe na meta com folga; **a medição real é a primeira tarefa de
implementação depois do DDA** (R1) e decide se a pirâmide de máximos
(item 5) entra.

**Memória** por execução: o recorte (≤ 256 MiB) + cópias preenchidas das
grades com buracos (≤ o tamanho das amostras do recorte, item 6) + cache de
peças (256 MiB) + buffers do quadro (`largura × altura × (3 + 4)` bytes,
~58 MB em 1080p; ~116 MB em 4K).

## 24. Dados de teste e validação com dados reais

**Decisão**:

- **Fixtures em código** (`test/helper`), sem arquivos binários versionados:
  `slice_fixture.go` (grava um recorte ZIP a partir de um `GeoSlice`, para o
  teste do leitor; e recortes inválidos, truncados, de versão 2, sem
  `plan_id`, com contagens erradas), `png_fixture.go` (peças PNG de cor lisa
  ou padrão), e o `mbtiles_fixture.go` ganha `TileFormat` já existente.
- **Testes do domínio** sobre cenas pequenas (`Scene` com poucas células e
  resolução 64 × 36): terreno plano e inclinado (a altura do acerto bate com
  a fórmula), pirâmide (oclusão), degrau entre grades, marcador projetado a
  ≤ 1 px, contagem de buracos, traçado que só cresce, igualdade byte a byte
  com 1 e com 8 goroutines, e **propriedade de deslocamento** (SC-009: o mesmo
  voo em coordenadas de outras regiões, inclusive `lon` em torno de 180 e `lat`
  > 80, dá imagens com as mesmas propriedades).
- **`test/samples`** (ferramenta de desenvolvimento, ver quickstart) ganha
  dois recursos: peças de mapa **em imagem** (PNG, com padrão reconhecível:
  xadrez de tom por peça e borda) para o MBTiles sintético, e a geração de um
  MBTiles raster **sobre a área de um GPX real**, menor que o mapa vetorial
  registrado (para vencer a regra "menor área"), usado com o relevo e o
  passeio reais de `resources/`.
- **Dados reais**: o MBTiles do BBBike em `resources/` é vetorial (`pbf`); a
  validação real exercita a **recusa** (código 31) com ele e o **desenho** com
  o raster sintético sobre o relevo Copernicus e o passeio reais.

## 25. Riscos e alternativas rejeitadas (resumo)

- **R1 — desempenho**: se a medição real passar da meta, entra a pirâmide de
  máximos (item 5), sem mudar resultado.
- **R2 — memória** com recortes de 256 MiB e grades com buracos (item 6 e
  23): documentada; o preenchimento pode ser feito por trechos se necessário.
- **R3 — degrau entre grades** de fontes diferentes na emenda: sem fresta
  horizontal, mas sem parede vertical. Raro (exige duas fontes de relevo
  no mesmo voo).
- **R4 — bordas do terreno sem suavização**: serrilhado no horizonte e nas
  cristas; o traçado e o marcador têm borda suave. Se incomodar, o custo de
  supersampling 2×2 é conhecido (4×).
- **R5 — sem curvatura** (itens 2 e 3): coerente com a etapa 3.
- **R6 — determinismo entre arquiteturas**: os pixels devem coincidir (itens 12.1
  e 12.2); os bytes do PNG só se o `compress/flate` coincidir.
- **R7 — sem iluminação nem sombra** (fora de escopo): o mapa aparece "chapado"
  sobre o relevo; a profundidade vem do paralaxe e do traçado. Uma etapa
  futura pode acrescentar sombreamento sem mudar o formato.
