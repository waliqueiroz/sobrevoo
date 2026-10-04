# Pesquisa Técnica: Acabamento das Sobreposições de Tela

Cada item segue Decisão / Racional / Alternativas consideradas. Item 0
resolve todos os `NEEDS CLARIFICATION` do Contexto Técnico do `plan.md`.

## 0. Contexto técnico — resumo

Nenhum `NEEDS CLARIFICATION` sobrou: a etapa se apoia inteiramente na stack
já existente (Go 1.26, `golang.org/x/image`, já em `go.mod` — nenhuma
dependência nova). A única decisão realmente profunda é o item 1: como
desenhar texto vetorial suavizado sem reintroduzir a divergência entre
arquiteturas que a etapa 9 explicitamente evitou ao escolher uma fonte
bitmap (`specs/009-frame-overlays/research.md`, item 4).

## 1. Por que não usar `golang.org/x/image/vector.Rasterizer`

**Contexto**: o caminho "natural" para rasterizar uma fonte TrueType/OpenType
em Go, sem depender de `cgo`/FreeType, é `golang.org/x/image/font/sfnt`
(parsing dos contornos) + `golang.org/x/image/vector.Rasterizer`
(rasterização com antialiasing) — o mesmo par que `font/opentype.Face` usa
internamente. A etapa 9 já havia rejeitado essa combinação, citando "o
principal risco de não-determinismo entre arquiteturas" sem detalhar; esta
pesquisa confirma e documenta exatamente qual é o risco, porque esta etapa
não pode mais evitar fonte vetorial (é o próprio pedido).

**O risco, confirmado no código do módulo** (`golang.org/x/image@v0.46.0/
vector/`): o pacote tem duas implementações da etapa de acumulação de
cobertura (antialiasing) — `acc_amd64.go`/`acc_amd64.s`, em assembly SIMD
(SSE4.1), usada quando `GOARCH=amd64` e a CPU tem o recurso; e
`acc_other.go`, em Go puro, usada em qualquer outra arquitetura (inclusive
arm64) ou quando o SIMD não está disponível. As duas são matematicamente
equivalentes "em tese", mas somam números em ponto flutuante numa ordem
diferente — exatamente o tipo de divergência que já levou este projeto a
proibir `Pow`/`Exp`/`Sinh` no desenho do terreno (CLAUDE.md, "O desenho dos
quadros"), por também terem implementação em assembly por arquitetura.
Usar `vector.Rasterizer` arriscaria quebrar o hash de referência de
`frame_scene_test.go` entre amd64 e arm64 — o que a nota do `CLAUDE.md` pede
para resolver corrigindo a aritmética, nunca a constante.

**Decisão**: escrever um rasterizador de glifos próprio, em
`internal/domain/vector_font.go`, usando só os operadores que o resto do
desenho de quadros já usa (`+ − × ÷`, `Floor`, `Min`, `Max`, `Abs`,
comparação) — nunca uma função com implementação por arquitetura. O
`sfnt.Font.LoadGlyph` continua sendo usado (item 3) porque só *extrai* os
contornos do arquivo de fonte (segmentos `MoveTo`/`LineTo`/`QuadTo`/
`CubeTo` em ponto fixo 26.6) — uma leitura determinística de dado, sem
nenhuma soma de ponto flutuante dependente de ordem, igual em qualquer
arquitetura, análoga a ler a tabela do `inconsolata.Bold8x16.Mask` que a
etapa 9 já faz.

**Racional**: mantém a garantia "mesmo plano, mesmo recorte, mesma
resolução, mesma aparência, mesma configuração de sobreposição → mesma
imagem, em qualquer máquina" (FR-002/FR-009/SC-007), a restrição mais
apertada desta etapa, sem abrir mão da suavização (FR-003) que motiva a
mudança. O racional do item 4 detalha o algoritmo escolhido.

**Alternativas consideradas**:
- Usar `vector.Rasterizer` e aceitar builds sempre com a tag `noasm`:
  rejeitada — exigiria toda build do binário (inclusive a que o usuário
  compila com `go build`/`make build`, sem nenhuma tag especial) forçar o
  caminho portátil, o que não é garantível fora do controle do projeto
  (ninguém força `-tags noasm` por padrão); e mesmo a implementação
  portátil (`acc_other.go`) mistura ponto fixo e ponto flutuante conforme o
  tamanho do glifo (`floatingPointMathThreshold`), outra fonte de
  comportamento não inteiramente coberto pela disciplina aritmética deste
  projeto.
- Depender de uma biblioteca de rasterização de terceiros fora do módulo
  `golang.org/x/image` (ex.: `github.com/golang/freetype`): rejeitada — nova
  dependência em `go.mod`, licença a revisar, e o mesmo risco de assembly
  por arquitetura não eliminado, só deslocado para outro pacote.
- Continuar com a fonte bitmap, só aumentando a resolução do glifo-fonte
  (ex.: uma fonte bitmap "maior"): rejeitada — ainda seria serrilhada ao
  ser escalada por fator inteiro (o problema que o pedido do usuário
  identifica), só com um serrilhado mais fino; não resolve FR-003.

## 2. A fonte vetorial escolhida

**Decisão**: `golang.org/x/image/font/gofont/goregular` — a fonte "Go
Regular" (desenhada pela fundição Bigelow & Holmes especificamente para o
projeto Go), distribuída como bytes TrueType embutidos em código Go
(`goregular.TTF`), dentro do mesmo módulo `golang.org/x/image` que já
fornece `font/inconsolata` hoje. **Nenhuma dependência nova em `go.mod`**:
é o mesmo `require golang.org/x/image v0.46.0` já presente, só importando
mais um subpacote dele.

**Licença**: BSD-3-Clause (Bigelow & Holmes Inc., 2016; ver
`$(go env GOMODCACHE)/golang.org/x/image@.../font/gofont/ttfs/README` e o
`LICENSE` do módulo) — permissiva, compatível com a licença MIT do
Sobrevoo (`LICENSE`, raiz do repositório), satisfazendo FR-001. A fonte não
é vendorizada pelo projeto: ela chega pela própria dependência de módulo já
declarada, do mesmo jeito que `inconsolata` chega hoje.

**Racional**: zero custo de dependência nova, zero arquivo binário
adicional no repositório, licença já resolvida pela equipe do Go, e um
design proporcional (sans-serif) legível em texto pequeno — o mesmo
critério que levou a etapa 9 a escolher `inconsolata.Bold8x16` entre as
fontes prontas do módulo.

**Alternativas consideradas**:
- Vendorizar um arquivo `.ttf`/`.otf` de terceiros (ex.: uma fonte do
  Google Fonts) via `go:embed`: rejeitada — mais um artefato binário no
  repositório, mais uma licença a documentar e revisar, sem nenhum ganho
  sobre uma fonte já disponível na dependência existente.
- `golang.org/x/image/font/gofont/gomono` (variante monoespaçada da mesma
  família): considerada para preservar a largura fixa por caractere que a
  fonte bitmap tinha; rejeitada porque a largura compartilhada dos painéis
  (FR-005/item 7) já resolve a irregularidade visual sem precisar de
  monoespaçamento, e uma fonte proporcional é mais legível em texto curto.

## 3. Extração dos contornos do glifo

**Decisão**: `golang.org/x/image/font/sfnt.Font` (`sfnt.Parse(goregular.TTF)`,
uma vez, no carregamento) + `(*sfnt.Font).LoadGlyph(buffer, index, ppem,
nil)`, que devolve `sfnt.Segments` — uma sequência de `MoveTo`/`LineTo`/
`QuadTo`/`CubeTo` já escalados para o tamanho em pixels pedido (`ppem`,
"pixels per em"), em `fixed.Point26_6` (ponto fixo de 26 bits inteiros e 6
fracionários, sempre a mesma representação, qualquer arquitetura).

**Racional**: é puramente extração de dado — o arquivo da fonte já contém
as curvas de Bézier do desenho de cada glifo; `LoadGlyph` só decodifica a
tabela `glyf`/`CFF` e aplica a escala pedida, sem nenhuma soma de ponto
flutuante dependente de ordem de operação (escala em ponto fixo, inteiro).
`sfnt.Buffer` é um buffer de reuso não seguro para uso concorrente — como
`screenOverlay.draw` já roda sempre sequencialmente dentro de um mesmo
`Scene` (nunca duas chamadas de `Render` em paralelo sobre a mesma
instância; `FrameService.DrawFrames` desenha quadros um a um, item 5), um
único `sfnt.Buffer` por `Scene` é suficiente e seguro, sem precisar de uma
exclusão mútua nova.

**Alternativas consideradas**: nenhuma — é a única forma, no módulo já
usado, de obter os contornos vetoriais de um TrueType sem escrever um
parser de fonte do zero (fora de escopo: escolher/entender formatos de
fonte não é o problema que esta etapa resolve).

## 4. O algoritmo de rasterização determinístico

**Decisão**: para cada glifo (rune, tamanho em pixels), uma vez:

1. Achatar cada `QuadTo`/`CubeTo` dos segmentos do glifo em um número fixo
   de trechos de reta — 8 subdivisões para uma curva quadrática, 12 para
   uma cúbica —, por avaliação paramétrica uniforme (`t = i/N`, interpolação
   linear/De Casteljau com só `+ − × ÷`). O número de subdivisões é fixo,
   nunca adaptativo por erro estimado (um critério adaptativo dependeria de
   comparações de ponto flutuante que poderiam, em tese, variar por
   arredondamento — fixo é mais simples e já suficiente no tamanho de texto
   desta sobreposição).
2. Rasterizar o polígono resultante (só retas) por superamostragem numa
   grade fixa de subpixels por pixel do retângulo do glifo (4×4 = 16
   amostras), contando, para cada subamostra, se o seu centro está dentro
   do contorno pela regra do número de voltas (`nonzero winding`, somando o
   sinal do produto vetorial de cada aresta — só `+ − × ÷` e comparação).
   A cobertura do pixel é a fração de subamostras dentro, de 0,0 a 1,0.
3. O resultado é uma máscara de cobertura (um `float64`/inteiro de 0 a 255
   por pixel do retângulo do glifo), guardada no cache do item 5 — a mesma
   forma de uso que a máscara `image.Alpha` do `inconsolata` já tem hoje
   (`frame_screen_overlay.go`, `drawText`), só que calculada em vez de
   lida de uma tabela fixa.

**Racional**: cada etapa usa só os operadores que `frame_scene.go` e
`frame_overlay.go` já usam (`+ − × ÷`, `Min`, `Max`, `Abs`, comparação,
nenhuma função transcendental, nenhuma assembly por arquitetura) — o mesmo
"cada pixel depende só de si mesmo" que o resto do renderizador já garante,
estendido ao desenho de texto. Uma grade fixa de subpixels (em vez de
cobertura analítica por área exata) é mais simples de implementar e de
testar (dado um conjunto fixo de arestas, o resultado de cada subamostra é
uma comparação determinística), e suficiente na escala de texto desta
sobreposição (poucos pixels de altura por glifo); o custo (16 testes de
ponto por pixel do glifo) é desprezível ao lado do custo de uma única
rajada de lançamento de raio por pixel de terreno que `Scene.drawTerrain`
já faz.

**Alternativas consideradas**:
- Cobertura analítica por área (como `raster_fixed.go`/`raster_floating.go`
  do próprio `golang.org/x/image/vector` fazem, sem a etapa de acumulação
  SIMD): tecnicamente mais precisa, mas reaproveitar qualquer parte interna
  não exportada desse pacote não é possível (API privada), e reescrevê-la à
  mão só para a parte "segura" seria mais código e mais risco de divergir
  sutilmente do original sem o ganho de precisão importar no tamanho de
  texto desta sobreposição.
- Supersampling com uma grade maior (ex.: 8×8): mais suave, mas 4×4 já
  elimina o serrilhado em bloco que motivou a etapa (FR-003/SC-001); o
  valor exato é um parâmetro de implementação, ajustável sem mudar nenhuma
  garantia desta pesquisa.

## 5. Cache de glifos por `Scene`

**Decisão**: o rasterizador (contornos do item 3 + algoritmo do item 4)
vive atrás de um tipo não exportado, guardado como campo novo de `Scene`
(construído uma vez em `NewScene`, ao lado de `imagery`), com um
`map[runeAndSize]glyphMask` que rasteriza cada glifo a primeira vez que é
pedido naquele tamanho e reaproveita depois. O tamanho em pixels do glifo
não muda entre quadros de uma mesma execução (é função só da resolução,
que é a mesma do início ao fim de um `render all`/`fly`), então o cache
nunca precisa de uma segunda entrada por tamanho na prática — mas é
indexado por tamanho de qualquer forma, para `render frame` e `render all`
poderem compartilhar o mesmo tipo sem exigir isso como invariante.

**Racional**: evita rasterizar o mesmo glifo centenas ou milhares de vezes
(uma vez por quadro em que aparece); é seguro sem exclusão mútua porque
`screenOverlay.draw` roda sempre sequencialmente dentro do mesmo `Scene`
(confirmado no item 3) — o mesmo raciocínio que já justifica o cache de
peças decodificadas em `imagery` não precisar de lock para a parte que só
`drawTerrain` usa internamente (ali, sim, concorrente entre workers, mas
com sua própria sincronização já existente, não tocada por esta etapa).

**Alternativas consideradas**:
- Rasterizar a cada chamada, sem cache: rejeitada por custo — desperdiça
  trabalho idêntico a cada quadro sem necessidade.
- Pré-rasterizar todos os glifos do alfabeto usado de uma vez, em
  `NewScene`: rejeitada como exigência — adiciona complexidade (teria que
  enumerar o alfabeto de antemão) sem benefício sobre o cache "on demand"
  descrito acima, que já é O(alfabeto) no total.

## 6. O contorno escuro do texto (FR-004)

**Decisão**: para cada glifo, desenhar duas vezes, na ordem: primeiro uma
versão **dilatada** da máscara de cobertura do item 4 — o valor de cada
pixel do contorno é o máximo da cobertura original numa vizinhança de raio
fixo em pixels ao redor dele (`OverlayOutlineRatio` de fração da altura do
quadro, com piso `OverlayOutlineMinWidth`, mesmo padrão de
`TrailMinWidth`/`MarkerMinRadius`) — pintada em `OverlayTextOutlineColor`
(uma cor escura fixa, nova constante ao lado de `OverlayTextColor`); depois
o glifo original, na cobertura do item 4, em `OverlayTextColor`, por cima.
O resultado é um contorno visível só onde a dilatação alcança além do
próprio glifo — exatamente a mesma técnica que `TrailCasingColor` (uma
linha mais larga, escura, desenhada antes do `TrailColor` por cima) e
`MarkerRingColor` (um anel desenhado antes do disco do marcador) já usam
hoje para o traçado e o marcador.

**Racional**: reaproveita um padrão de desenho que já existe no projeto
(casca/anel antes do núcleo), em vez de introduzir uma técnica nova; a
dilatação por máximo numa vizinhança usa só `Max` e comparação — já na
lista de operadores permitidos —, sem nenhuma dependência do pixel de
fundo (preserva o racional da etapa 9, item 5: nunca amostrar o que está
por baixo para decidir contraste). Funciona sobre fundo claro e escuro por
construção (FR-004/SC-002), porque o contorno é sempre escuro e o texto
sempre claro, sem relação com o fundo real.

**Alternativas consideradas**:
- Aumentar a opacidade do painel: explicitamente rejeitada pelo próprio
  pedido do usuário (taparia mais imagem).
- Desenhar o contorno só nas quatro direções cardeais (técnica clássica de
  "stroke" de texto em jogos 2D, 4 ou 8 cópias deslocadas do glifo): mais
  barata, mas produz um contorno anguloso/irregular em diagonais; a
  dilatação por máximo numa vizinhança circular (mesmo raio em todas as
  direções) dá um contorno mais uniforme, ao custo de mais operações por
  pixel — custo ainda desprezível no tamanho do texto desta sobreposição.

## 7. Largura compartilhada dos três painéis numéricos (FR-005)

**Decisão**: `screenOverlay.draw` passa a medir, antes de desenhar
qualquer painel, a largura do texto de cada um dos blocos de distância,
elevação+ganho e tempo decorrido presentes naquele quadro (via uma função
de medição sobre a fonte vetorial — a soma dos avanços de cada glifo mais o
espaçamento entre eles, a mesma conta que o desenho em si precisa para
posicionar cada glifo), toma o maior desses valores, e usa-o como a largura
de todos os painéis presentes — preenchendo com o mesmo preenchimento
(`overlayLinePadding`) de hoje, texto alinhado à esquerda dentro do painel.
Um bloco ausente (desligado ou sem dado) não entra no cálculo do máximo.

**Racional**: cumpre FR-005 com a mínima mudança de estrutura —
`drawLine` já calculava a largura do próprio texto para dimensionar seu
painel; passa a receber a largura já decidida (o máximo) em vez de
calculá-la sozinho. A ordem e o espaçamento vertical entre blocos (`y +=
lineHeight + pad`) não mudam.

**Alternativas consideradas**:
- Uma largura fixa, calculada uma vez para o texto mais largo
  teoricamente possível em qualquer execução (ex.: elevação de 5 dígitos):
  rejeitada — exigiria decidir um "pior caso" arbitrário, e produziria
  painéis maiores que o necessário na maioria dos vídeos; medir por quadro
  o texto real, como o pedido descreve ("definida pelo bloco mais largo"),
  é mais simples e sempre exato.

## 8. Dimensionamento do marcador do perfil de elevação (FR-006)

**Decisão**: duas constantes novas em `render_tuning.go`, fixas (não
ajustáveis pelo usuário, fora de `Appearance` — o mesmo motivo que já
mantém `MarkerRingRatio`/`MarkerRingMin` fora dela): `ProfileMarkerRadiusRatio`
(fração da altura do quadro) e `ProfileMarkerMinRadius` (piso em pixels).
`drawProfile` troca o raio de hoje (`max(2, scale)`, ligado à escala do
texto bitmap, que deixa de existir) por
`max(ProfileMarkerMinRadius, ProfileMarkerRadiusRatio * altura)`.

**Racional**: mesma fórmula que `MarkerMinRadius`/`MarkerRadiusRatio` já
usa para o marcador sobre o terreno (FR-006 pede explicitamente "como o
marcador do mapa já tem") — mas com sua própria constante, não a mesma
`Appearance.MarkerRadiusRatio` que o usuário escolhe para o marcador do
terreno: o marcador do perfil não é uma escolha de aparência (FR-011), e
reusar o valor do usuário deixaria o marcador do perfil minúsculo sempre
que alguém pedir um marcador de terreno pequeno, contrariando o próprio
propósito desta etapa (SC-004).

**Alternativas consideradas**: ligar o raio à `Appearance.MarkerRadiusRatio`
do usuário (rejeitada pelo racional acima); um raio em pixels absolutos
(rejeitada, mesma razão de toda medida de desenho da ferramenta ser uma
razão da altura, não um valor fixo de tela).

## 9. Margem de segurança por borda (FR-007)

**Decisão**: a única constante de hoje (`OverlayMarginRatio`, fração do
lado menor, igual nas quatro bordas) é substituída por três:
`OverlayTopMarginRatio` e `OverlaySideMarginRatio` (frações da altura e da
largura, respectivamente, ~0,06 — o valor de hoje, mantido onde já
funcionava) e `OverlayBottomMarginRatio` (fração da altura, ~0,14 — pouco
mais que o dobro do topo). `screenOverlay.draw`/`drawProfile` calculam a
margem de cada lado separadamente (`marginTop = OverlayTopMarginRatio *
altura`, `marginLeft = marginRight = OverlaySideMarginRatio * largura`,
`marginBottom = OverlayBottomMarginRatio * altura`) em vez de um único
`margin` aplicado a tudo.

**Racional**: FR-007 pede frações da altura **e** da largura, com a base
maior — expressar a margem lateral como fração da largura e a
vertical como fração da altura (em vez de ambas como fração do lado menor,
como hoje) é a leitura literal do requisito, e é como guias de "área
segura" de redes sociais verticais (Reels, Stories, Shorts) costumam
documentar a faixa cobertas por legenda/botões — tipicamente próxima de
12–15% da altura na base, bem mais que o topo ou as laterais. ~0,14 cobre
essa faixa com folga; ~0,06 (igual ao valor de hoje) já funcionava para as
outras três bordas.

**Alternativas consideradas**: manter uma única razão, só aumentando seu
valor (ex.: 0,10 nas quatro bordas): rejeitada — não é "maior na borda
inferior do que nas demais" como FR-007 exige, e desperdiçaria espaço útil
nas outras três bordas, que não têm o mesmo problema de interface de rede
social.

## 10. Versão do desenho e teste de hash de referência

**Decisão**: `domain.RenderVersion` sobe de `2` para `3`. Como
`NewFrameSetID`/`newFrameSetID` (`frame_set.go`) já incluem `RenderVersion`
no hash do `FrameSetID` — sem precisar de nenhuma mudança de código além do
valor da constante —, um conjunto de quadros desenhado pela versão anterior
passa a ser, automaticamente, "de outro conjunto": `render all` o recusa
sem `--overwrite` (`ErrFrameSetConflict`) e `fly --keep` redesenha os
quadros e o vídeo ao notar a mudança, pela mesma lógica que já existe desde
a etapa 8. O hash de referência de pixels de `frame_scene_test.go` precisa
de uma nova constante, calculada depois da implementação, exatamente como a
nota do `CLAUDE.md` já prevê.

**Racional**: é o mecanismo mínimo e já comprovado (usado nas etapas 6, 8 e
9) para garantir que quadros de antes e depois desta etapa nunca se
misturem; nenhuma mudança de formato de arquivo, nenhuma migração, nenhum
campo novo precisa existir.

**Alternativas consideradas**: nenhuma — é o mesmo mecanismo que toda
mudança visual anterior já usou; inventar outro seria uma regressão de
consistência sem benefício.
