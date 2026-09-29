# Pesquisa Técnica: Sobreposições de Tela nos Quadros

Cada item segue Decisão / Racional / Alternativas consideradas. Item 0 resolve
todos os `NEEDS CLARIFICATION` do Contexto Técnico do `plan.md`.

## 0. Contexto técnico — resumo

Nenhum `NEEDS CLARIFICATION` sobrou: a etapa se apoia inteiramente na stack
já existente (Go 1.26, sem nova dependência de terceiros — `golang.org/x/
image`, já presente em `go.mod`, cobre a fonte embutida). As decisões abaixo
resolvem as questões técnicas que o `spec.md` deixou para o planejamento
(números exatos de margem, formatação, mecanismo de contraste, e — a mais
profunda — como o ganho de elevação acumulado pode ser exibido byte a byte
correto sem reler o trajeto).

## 1. Onde a sobreposição de tela entra no pipeline de desenho

**Decisão**: um novo passo de desenho, chamado por último dentro de
`Scene.Render` (depois de `overlay.drawTrail`/`overlay.drawMarker`), roda
sobre a imagem já pronta (terreno + mapa + traçado + marcador) e nunca é
afetado pelo teste de profundidade (`overlay.visible`) — está sempre por
cima, por definição, já que não é "colado no terreno". `Scene` ganha um
campo novo, `overlayConfig domain.OverlayConfig`, e `NewScene` um parâmetro
novo para recebê-lo; `Scene.Render` mantém a assinatura atual.

**Racional**: preserva a garantia de "mesmo plano, mesmo recorte, mesma
resolução, mesma aparência e mesma configuração de sobreposição → mesma
imagem" (FR-013) porque a sobreposição de tela é só mais um passo
determinístico no fim do mesmo `Render`, com o mesmo modelo de "cada pixel
depende só de si mesmo" — na verdade cada pixel do texto/gráfico depende só
do índice do quadro e do `CameraPlan`, nunca da câmera ou do terreno, então
nem precisa entrar na banda de goroutines que desenha o terreno (roda uma
vez, sequencial, depois que as bandas terminam — desenhar texto é muito mais
barato que traçar raios, não precisa de paralelismo).

**Alternativas consideradas**:
- Um novo tipo de "camada" combinável fora de `Scene` (ex.: `FrameService`
  desenha o terreno e depois chama um segundo desenhador) — rejeitada
  porque duplicaria a abertura/fechamento de `FrameImage` e complicaria o
  determinismo (dois passos que precisam concordar em não se sobrepor de
  forma dependente de ordem) sem nenhum ganho: nada physically impede que
  seja o mesmo `Scene.Render`.
- Desenhar a sobreposição dentro do laço de bandas (por pixel, junto do
  terreno) — rejeitada: a sobreposição de tela não depende de raio nem de
  profundidade, então misturá-la ali só acoplaria dois conceitos
  independentes sem necessidade.

## 2. Nome do novo conceito de domínio (evitar colisão com `overlay`)

**Decisão**: o tipo não exportado que já existe em `frame_overlay.go`
(traçado + marcador, colados no terreno) mantém o nome `overlay` — é Go, e
`overlay` (minúsculo) e o novo `domain.OverlayConfig`/`domain.OverlayBlock`
(exportados) não colidem. O código de desenho da sobreposição de tela em si
vive num tipo não exportado novo, `screenOverlay`, em `frame_screen_overlay.go`,
para não reusar o nome `overlay` como identificador dentro do mesmo pacote
(evita confundir os dois na leitura, mesmo sem colisão de compilador). A
configuração (o que o usuário escolhe) é `domain.OverlayConfig`, em
`frame_overlay_config.go` — nomeada como o `spec.md` já nomeia o conceito
("Configuração de Sobreposição"), não como "HUD" ou outro termo que o pedido
original não usa.

**Racional**: Go permite os dois nomes convivendo (case-sensitive), e usar o
vocabulário do próprio `spec.md` mantém código e especificação alinhados —
constitution Princípio IX pede nomes que descrevem o papel, não um termo
emprestado de outro domínio (HUD é jargão de jogos, não do texto do
pedido).

## 3. Os quatro blocos e sua configuração

**Decisão**: `domain.OverlayBlock` é um tipo `string` com quatro constantes
— `OverlayBlockDistance`, `OverlayBlockElevation`, `OverlayBlockTime`,
`OverlayBlockProfile` — confirmadas na sessão de clarificação. `domain.
OverlayConfig` é um struct com um `Enabled bool` e um `bool` por bloco
(`Distance`, `Elevation`, `Time`, `Profile`), não um `map[OverlayBlock]bool`:
mais simples de comparar, de testar (given/when/then sem laço) e de
impressão em `Fingerprint()`. `NewOverlayConfig(enabled bool, blocks
[]OverlayBlock) (OverlayConfig, error)` valida cada nome de bloco
(`ErrInvalidOverlayBlock` para um nome desconhecido, FR-017) e monta os
quatro `bool`; quando `enabled` é falso, os blocos informados não importam
(desligar tudo já é desligar tudo — a CLI nem chega a interpretar
`--overlay-blocks` nesse caso, ver item 8).

**Racional**: espelha exatamente `domain.Appearance` (cinco campos `bool`/
`RGB`/`float64` simples, `Fingerprint()` de string canônica) — mesma forma,
mesmo lugar no pipeline (identidade do conjunto de quadros), mesmo padrão de
teste. `PatternPeriod`/`RGB`-style enums do projeto já usam `string` para
enums pequenos (`Phase`, `TimeReference`), então `OverlayBlock` segue o
mesmo estilo.

## 4. Fonte embutida

**Decisão**: `golang.org/x/image/font/inconsolata.Bold8x16` — uma fonte
bitmap (8×16 pixels por glifo), pré-renderizada e embutida como dados Go
(`data.go` gerado, sem carregar arquivo em tempo de execução), cobrindo o
ASCII imprimível. **Nenhuma dependência nova**: `golang.org/x/image` já é
exigida por `go.mod` (hoje indireta, via `tiledecoder`/`elevationreader`);
importar `font/inconsolata` só move a mesma versão (v0.46.0) para direta.

**Racional**: satisfaz FR-008 (fonte embutida, sem depender do sistema) sem
tocar em `go.mod`/licenciamento novo. Quanto ao determinismo (byte a byte,
research.md da etapa 5, item 9 do CLAUDE.md): os glifos são uma máscara
`image.Alpha` de dados **fixos**, gerados uma vez e congelados no código-fonte
do pacote `inconsolata` — não há nenhum cálculo de antialiasing em tempo de
execução, então usá-los não introduz nenhuma operação de ponto flutuante nova
além de leitura de tabela; a composição do glifo sobre a imagem (mistura
alfa) usa só `+ − × ÷` — a mesma função `mix`/`blend` que `overlay.go`(o
existente) já usa para o traçado e o marcador (`rounded(old*(1-coverage) +
painted*coverage)`), reaproveitada tal e qual. O aumento de escala do glifo
nativo (8×16) para o tamanho final é feito por replicação de pixel
("nearest-neighbor" por um fator **inteiro**), nunca por interpolação: cada
pixel de destino mapeia para `srcX = dstX / fator` (divisão inteira), sem
ponto flutuante algum no laço interno — o único ponto flutuante do texto
inteiro é a escolha do fator de escala em si (uma conta por quadro, não por
pixel), feita com `roundHalfUp` (já existe em `camera_planning.go`, mesma
função reaproveitada). Isso é mais simples e mais seguro contra divergência
entre arquiteturas do que rasterizar um TTF (`sfnt`/`vector` do mesmo
módulo) em tempo de execução, que envolveria curvas de Bézier e
antialiasing calculados a cada renderização.

**Alternativas consideradas**:
- `basicfont.Face7x13` (a outra fonte pronta do mesmo módulo): mais
  compacta, mas 7×13 é pouco legível já escalada; `inconsolata.Bold8x16` lê
  melhor em negrito, útil sobre um fundo variado (FR-010).
- Rasterizar uma fonte TTF externa embutida via `go:embed` +
  `golang.org/x/image/font/sfnt` + `vector.Rasterizer`: rejeitada — exigiria
  escolher e vendorizar um arquivo de fonte (mais um artefato binário no
  repositório, mais uma licença a documentar) e reintroduziria antialiasing
  calculado em tempo real, o principal risco de não-determinismo entre
  arquiteturas que a etapa 5 documentou evitar.
- Desenhar dígitos com um "segmento" vetorial próprio (como um mostrador de
  relógio digital): rejeitada por não cobrir letras/rótulos e por
  reinventar o que uma fonte pronta já resolve.

## 5. Legibilidade sobre qualquer fundo (FR-010)

**Decisão**: cada bloco desenha sobre uma placa de fundo semitransparente,
cor fixa quase-preta (`OverlayPanelColor`) a uma opacidade fixa
(`OverlayPanelOpacity`, ~55%), e o texto em si é sempre branco
(`OverlayTextColor`, fixo). Nenhum dos três é ajustável — são o significado
da sobreposição, não estilo, o mesmo raciocínio que já mantém
`TrailCasingColor`/`MarkerRingColor`/`NoMapColors`/`NoElevationColors` fora
de `Appearance` (008-frame-appearance FR-005, repetido aqui). Ficam junto
deles em `render_tuning.go`.

**Racional**: uma placa semitransparente garante contraste mínimo
**sem amostrar o pixel de fundo em tempo de desenho** — não precisa ler a
cor do terreno para decidir a cor do texto (o que quebraria o determinismo
por depender de mais estado, e tornaria a etapa acoplada ao conteúdo do
recorte de dados). É o mesmo raciocínio que já resolveu o problema análogo
do traçado/marcador (uma casca/anel de cor fixa, não uma decisão baseada no
pixel embaixo).

**Alternativas consideradas**:
- Contorno (stroke) preto ao redor do texto, sem placa: mais parecido com
  legendas de vídeo, mas pouco confiável para glifos bitmap pequenos sobre
  um fundo muito ruidoso (o próprio terreno com textura de mapa); a placa
  cobre esse caso com uma margem de segurança maior.
- Amostrar a luminância média do fundo sob cada bloco e escolher texto claro
  ou escuro: rejeitada — introduz uma dependência do conteúdo da cena na
  sobreposição de tela (contradiz "não colada no terreno") e uma nova fonte
  de nuance de arredondamento a documentar; a placa fixa resolve o mesmo
  problema com uma regra mais simples e sempre determinística por
  construção.

## 6. Margem de segurança

**Decisão**: `OverlayMarginRatio` (constante fixa, não ajustável, ~6%),
medida como fração do **menor lado** do quadro (`min(largura, altura)`) —
não da altura sozinha como `TrailWidthRatio`/`MarkerRadiusRatio` — porque a
preocupação central (FR-009) é o corte de borda em vídeo vertical, onde a
largura é o lado mais apertado; usar o menor lado garante a mesma margem
relativa não importa a proporção escolhida (`--aspect`). Fica em
`render_tuning.go`, junto de `PatternPeriod`.

**Racional**: um valor documentado, único, cobre as quatro bordas
igualmente (FR-009 pede margem "nas quatro bordas", não uma por lado).
Nenhum bloco desenha nada — texto, placa ou o gráfico do perfil — mais perto
de qualquer borda do que essa margem.

**Alternativas consideradas**: margem em pixels absolutos — rejeitada,
seria a única medida de desenho da ferramenta que não escala com a
resolução (toda medida análoga hoje é uma razão: `TrailWidthRatio`,
`MarkerRadiusRatio`, `MarkerRingRatio`).

## 7. O ganho de elevação acumulado — a decisão mais delicada

**Contexto**: o pedido original nomeia dois campos novos no plano — "o
instante real da atividade" e "a elevação do trajeto no ponto do marcador"
— e o bloco (b) da sobreposição mostra **dois números**: a elevação atual e
o ganho acumulado (`spec.md` FR-001, FR-004). FR-014 exige que o ganho
acumulado **ao final do voo** coincida exatamente com o que `inspect`
relata (`Route.ElevationGain()`).

**O problema**: se o ganho acumulado fosse calculado, a cada quadro, por uma
soma corrente dos deltas positivos **entre quadros consecutivos** (usando só
a elevação instantânea já gravada por quadro), o resultado seria, em geral,
**menor ou igual** ao ganho verdadeiro — nunca maior — porque reamostrar um
perfil de elevação num espaçamento diferente do original (menos denso perto
dos pontos do trajeto, mais denso longe deles, conforme o ritmo do marcador)
só pode "esconder" subidas-e-descidas pequenas entre dois quadros vizinhos,
nunca inventar uma nova. Isso violaria FR-014 sempre que o trajeto tiver
qualquer trecho de sobe-desce que cair inteiro entre dois quadros
consecutivos — um caso comum, não uma borda rara.

**Decisão**: o campo novo de elevação do plano não é a elevação bruta
instantânea, e sim o **ganho de elevação acumulado até o ponto do
marcador** (`TrackElevationGain`), pré-computado **uma vez**, na geração do
plano, sobre os pontos do trajeto tratado (não reamostrado por quadro) — o
mesmo algoritmo que `Route.ElevationGain()` já usa, mas guardando a soma
parcial em cada ponto em vez de só o total — e então **interpolado
linearmente** na distância exata do marcador de cada quadro, do mesmo jeito
que `PlanarRoute.PointAt` já interpola X/Y. Como o último quadro está
sempre exatamente na distância `route.Length()` (o último ponto do
trajeto), a interpolação ali cai exatamente sobre o ponto final — sem erro
de interpolação —, garantindo `TrackElevationGain` do último quadro ==
`Route.ElevationGain()` bit a bit. Para não perder o número de **elevação
atual** que o bloco (b) também mostra, o plano guarda **um terceiro campo**,
`TrackElevation` (a elevação bruta, interpolada da mesma forma, sem a
exigência de exatidão — `inspect` não relata "elevação atual", então FR-014
não a cobre).

Ou seja: o formato do plano ganha **três** campos novos por quadro — não
dois — mais um novo campo de nível de plano, `ElevationAvailable bool`
(mirror do já existente `TimeReference`, que já sinaliza a ausência de
dado de tempo do mesmo jeito). O `spec.md` nomeia a necessidade em termos de
usuário ("o instante real" + "a elevação do trajeto no ponto do marcador");
esta pesquisa documenta que, para cumprir FR-014 com exatidão byte a byte,
"a elevação do trajeto" se desdobra em dois números que precisam ser
gravados separadamente. Isso não muda nenhum requisito do `spec.md` nem
nenhum critério de aceite — só precisa mais que dois campos JSON para
cumpri-los.

**Racional**: é a única forma de garantir exatidão no último quadro sem
reler o trajeto GPS no momento do desenho (FR-005) — a soma parcial correta
só existe com acesso aos pontos originais do trajeto tratado, disponíveis
apenas na geração do plano (`TreatedTrack.PlanCamera`), nunca depois. Como
efeito colateral, o ganho mostrado em quadros intermediários também fica
sempre não-decrescente (a soma parcial pré-computada é monotônica; interpolar
entre dois valores monotônicos numa distância crescente nunca decresce) —
um contador de "subida" que não anda para trás bate com a expectativa
natural do usuário.

**Alternativas consideradas**:
- Soma corrente ingênua entre quadros consecutivos (rejeitada acima —
  subestima o total verdadeiro em geral).
- Guardar só o ganho acumulado (sem a elevação bruta), e mostrar no bloco
  (b) só o ganho, sem "elevação atual": rejeitada porque o `spec.md`
  aprovado (FR-001, Entidades-Chave) já lista os dois números como parte do
  bloco — reduzir o bloco agora seria mudar o escopo aprovado sem o
  usuário pedir.
- Recalcular o ganho acumulado em tempo de desenho a partir da grade de
  elevação do **recorte** (o relevo/DEM, não o trajeto): rejeitada —
  FR-005 e o próprio `spec.md` (Suposições) dizem explicitamente que a
  elevação exibida vem do **trajeto**, não do relevo, e é essa escolha que
  permite bater com `inspect` (que também usa a elevação do trajeto); usar
  o DEM daria um número plausível mas diferente do de `inspect`.

## 8. Onde e como o instante real e a elevação são calculados

**Decisão**: dois métodos novos em `domain.Route` (mesma entidade dona do
dado, Princípio IX):
- `Route.TimeAt(distances []float64, at float64) (elapsed time.Duration, ok bool)`
  em `duration.go`, ao lado de `Duration()`: interpola linearmente o tempo
  real decorrido desde o primeiro ponto (`ok` falso quando `!allHaveTime()`,
  mesmo critério de `Duration()`).
- Um novo tipo `Route.ElevationProfile(distances []float64) (ElevationProfile, bool)`
  em `elevation_profile.go`, ao lado de `elevation.go`: pré-computa, uma vez,
  a elevação e o ganho acumulado em cada ponto (`ok` falso quando
  `!allHaveElevation()`, mesmo critério de `ElevationGain()`), devolvendo um
  `ElevationProfile{ Distances, Elevations, Gains []float64 }` com um método
  `At(distance float64) (elevation, gain float64)` que faz a mesma busca por
  bracket + interpolação que `PlanarRoute.PointAt` já faz — evita recalcular
  o perfil inteiro a cada quadro (o) (n) pontos × frameCount quadros seria
  desperdício; pré-computar uma vez e reusar por busca binária é o mesmo
  padrão que `PlanarRoute.Distances` já emprega).

Em `distances` os dois recebem o mesmo `[]float64` que `PlanarRoute.
Distances` já guarda (distância acumulada de cada ponto do trajeto
tratado) — não recomputam `Route.Distances()`.

`cameraPlanner` (camera_planning.go) ganha dois campos novos —
`trackRoute Route` (o `t.Route` original, para `TimeAt`) e `elevation
*ElevationProfile` (nil quando o trajeto não tem elevação) —, preenchidos
uma vez em `PlanCamera` antes do laço de quadros, e `cameraPlanner.frame`
passa a preencher `ActivityElapsed`, `TrackElevation` e `TrackElevationGain`
de cada `CameraFrame`, quantizados como os demais campos (`lengthStep` para
metros, um passo equivalente em segundos para o tempo). `NewCameraPlan`
ganha um parâmetro `elevationAvailable bool`, guardado em
`CameraPlan.ElevationAvailable` e replicado em `PlanSummary.
ElevationAvailable` (o mesmo padrão de `TimeReference`).

**Racional**: mantém a regra de negócio na entidade dona do dado (`Route`),
como o Princípio IX exige, e reaproveita o padrão de "pré-computa um array,
interpola por busca" que `PlanarRoute` já estabeleceu — nenhum conceito
novo de arquitetura, só uma extensão simétrica.

## 9. `CameraPlan.ID()` não muda

**Decisão**: os três campos novos de `CameraFrame` e o novo
`ElevationAvailable` **não** entram no hash de `CameraPlan.ID()`
(`camera_planning.go`) nem no texto `"sobrevoo-plan-v1"` que o abre.

**Racional**: os três valores são funções puras e determinísticas dos
campos que **já** entram no hash — `MarkerDistance` (já presente) mais os
dados do trajeto tratado, que por sua vez é function do trajeto de entrada e
dos parâmetros de simplificação/suavização já cobertos pelo restante do
plano. Dois planos com o mesmo `ID()` (mesmos quadros nos campos já
hasheados) têm, por construção, sempre o mesmo `ActivityElapsed`/
`TrackElevation`/`TrackElevationGain` — incluí-los no hash não mudaria
nenhuma decisão de identidade, só forçaria uma migração de `ID()` (e,
por tabela, invalidaria todo reaproveitamento de `fly --keep` existente,
mesmo para quem não usa sobreposição) sem nenhum ganho de correção. O
próprio código já trata `Time` (o instante de vídeo) da mesma forma — não
entra no hash, por ser função pura de `Index`/`FrameRate`.

**Alternativas consideradas**: incluir os campos novos no hash "por
paranoia" — rejeitada pelo racional acima; o efeito prático seria só mais
trabalho e mais risco de regressão, sem nenhuma correção real.

## 10. Versão do arquivo do plano

**Decisão**: `planFormatVersion` (jsonfile) sobe de 1 para 2. Um arquivo
`format_version: 1` é recusado **antes** de qualquer outra checagem, com
`ErrPlanFormatVersionUnsupported` e uma mensagem que nomeia a versão
encontrada, a aceita, e instrui a gerar o plano de novo (FR-007). Os campos
novos (`activity_time_s` no quadro; `elevation_m` e `gain_m` dentro de
`marker`; `elevation_available` no resumo) são, ainda assim, obrigatórios
na leitura (ponteiro nulo → `ErrPlanFileInvalid`, mesmo padrão que
`camera_to_marker_m` já segue) — defesa a mais contra um arquivo versão 2
corrompido à mão, não o mecanismo principal de recusa.

**Racional**: a interface `CameraPlanReader` já documenta que
`ErrPlanFormatVersionUnsupported` é exatamente para "um formato que o leitor
não conhece" — um plano v1 é um arquivo v1 válido, só que de uma versão que
esta etapa não aceita mais; é semanticamente diferente de "v2 malformado"
(`ErrPlanFileInvalid`), e dá ao usuário uma mensagem mais direta ("gerado
por uma versão antiga; gere de novo") do que "campo ausente". Precedente
oposto (campo novo **sem** subir a versão): `parameters.aspect_ratio`
(etapa 3→posterior) foi acrescentado como opcional, com um padrão razoável
(`LandscapeAspectRatio`) para planos antigos — não se aplica aqui porque não
existe um padrão razoável para "instante da atividade"/"elevação" de um
plano antigo: são exatamente os dados que faltam.

## 11. O que aparece no perfil de elevação (bloco d)

**Decisão**: o gráfico usa as amostras (`MarkerDistance`, `TrackElevation`)
de todos os quadros da fase `PhaseFollowing` (os únicos em que a distância
varia; abertura e fechamento ficam parados em 0/`route.Length()`),
recalculadas a cada `Render()` — sem cache entre quadros —, seguindo o
mesmo padrão que `Scene.Render` já usa para o traçado (`over.drawTrail`
reconstrói a trilha inteira, de nnnn até o quadro atual, a cada chamada).
Um ponto (o quadro atual) marca a posição corrente sobre a linha.

**Racional**: consistente com a escolha de estilo já feita na etapa 5
(recomputar em vez de cachear entre quadros — mais simples de manter
correto, e o custo é desprezível perto do de traçar raios por pixel do
terreno). Não exige nenhuma estrutura de dado nova além dos três campos por
quadro que o item 7/8 já introduz.

## 12. Formatação dos valores exibidos

**Decisão** (documentada aqui porque `spec.md` deixou os números exatos
para o planejamento):
- Distância: metros inteiros abaixo de 1 km, quilômetros com 1 casa decimal
  a partir de 1 km (`"850 m"`, `"12.3 km"`).
- Elevação e ganho: metros inteiros (`"1234 m"`, `"+567 m"`).
- Tempo decorrido: sempre `H:MM:SS` (nunca omite a hora), para a largura do
  bloco não variar quadro a quadro — o texto é bitmap monoespaçado, então
  uma largura fixa evita o bloco "tremer" horizontalmente.

**Racional**: unidades métricas documentadas (FR-014 do `spec.md`); largura
fixa por bloco é uma decisão de desenho puramente estética, sem efeito em
nenhum requisito, mas evita um efeito visual ruim (texto reancorado a cada
quadro) que o `spec.md` não previu explicitamente mas que a margem de
segurança e a posição fixa (FR-001) implicam.
