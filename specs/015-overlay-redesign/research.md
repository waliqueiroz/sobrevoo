# Pesquisa Técnica: Redesenho das Sobreposições de Tela em Colunas

Cada item segue Decisão / Racional / Alternativas consideradas. Item 0
resolve o Contexto Técnico do `plan.md` — nenhum `NEEDS CLARIFICATION`
sobrou.

## 0. Contexto técnico — resumo

A etapa não introduz nenhuma dependência nova nem decisão de stack: tudo se
apoia no que as etapas 9/11/12/14 já deixaram pronto (`vectorFace`, a
técnica de casca-antes-do-núcleo, `OverlayConfig`/`NewOverlayConfig`/
`Fingerprint`, `RenderVersion` dentro de `FrameSetID`). As decisões reais
são todas de **layout e apresentação**, dentro de `frame_screen_overlay.go`
— itens 1 a 6 — e uma de **gating** (o bloco `gain` precisa do mesmo dado
que `elevation` já exige) — item 4.

## 1. Da pilha vertical com largura compartilhada para colunas horizontais

**Contexto**: hoje `screenOverlay.draw` empilha verticalmente os blocos
presentes (`y += lineHeight + pad` a cada um), e `Scene.numericPanelWidth`/
`stablePanelWidth` percorrem **todos os quadros do plano** para achar o
texto mais largo que qualquer bloco numérico atinge em qualquer quadro, e
usam esse valor como a largura compartilhada dos painéis (etapa 12,
FR-007/FR-008) — um mecanismo que existe só porque cada painel precisava de
uma largura de fundo.

**Decisão**: substituir a pilha vertical por uma faixa horizontal: os
blocos presentes (numa ordem fixa, item 2) dividem a largura útil
(`largura do quadro − 2×margem lateral`, a mesma `OverlaySideMarginRatio`
de sempre) em colunas de mesma largura — `colWidth = usableWidth /
len(present)` — cada uma com seu centro em `marginSide + colWidth×(i+0.5)`.
Como não há mais painel, não há mais "largura compartilhada" a calcular:
cada bloco centraliza seu próprio texto (medido fresco, só daquele quadro)
no centro da própria coluna. Isso elimina por completo
`Scene.numericPanelWidth`/`stablePanelWidth` e os campos `panelWidth`/
`panelWidthHeight`/`panelWidthSet` de `Scene` — nenhum substituto precisa
escanear `plan.Frames`, porque o conjunto de blocos presentes (o único dado
que varia por voo, não por quadro) já é conhecido sem olhar nenhum texto.

**Racional**: cumpre FR-002/FR-003 com a estrutura mais simples possível —
menos código do que o que existia antes desta etapa, não mais. A
"estabilidade por voo" que FR-015 exige é automática: o número de colunas e
sua posição dependem só de `OverlayConfig` e da disponibilidade de dado no
plano (`ElevationAvailable`, `TimeReference`), que não mudam quadro a
quadro dentro do mesmo voo — recalcular a cada quadro (barato, O(1)) dá
sempre o mesmo resultado, sem precisar de nenhum cache explícito.

**Alternativas consideradas**:
- Manter um cache explícito do layout em `Scene` (como `panelWidth` hoje):
  rejeitada — o cálculo é tão barato (contagem de bools + uma divisão) que
  cachear só acrescentaria estado para manter sincronizado, sem ganho de
  desempenho mensurável.
- Colunas de largura proporcional ao texto que cada bloco tende a ocupar
  (ex.: tempo decorrido sempre mais largo que velocidade): rejeitada — o
  pedido original exige explicitamente colunas de **mesma largura**
  (FR-003), não proporcionais ao conteúdo.

## 2. Onde a ordem fixa dos blocos vive

**Decisão**: uma lista de pacote, não exportada, em
`frame_overlay_config.go` (ao lado de `OverlayBlock`, por ser o mesmo
conceito):

```go
var overlayBlockOrder = []OverlayBlock{
    OverlayBlockSpeed,
    OverlayBlockElevation,
    OverlayBlockDistance,
    OverlayBlockGain,
    OverlayBlockTime,
}
```

`screenOverlay.draw` percorre `overlayBlockOrder`, inclui cada bloco cuja
combinação config+disponibilidade o mostra, e divide a largura útil pelo
total incluído — nunca pela ordem em que `--overlay-blocks` foi escrito
(FR-004, Clarifications). `profile` fica fora dessa lista: continua sendo
tratado à parte, no rodapé, como hoje.

**Racional**: um único ponto de verdade para a ordem, no mesmo arquivo que
já declara `OverlayBlock`/`OverlayConfig` — consistente com a convenção do
projeto de manter porta/entidade/comportamento relacionados no mesmo
arquivo (Princípio IX). Reaproveita o padrão que `draw` já seguia (uma
sequência fixa de `if`s, hoje distância→elevação→tempo→velocidade): só
passa a ser guiado por uma lista de dados, em vez de `if`s escritos na
ordem, porque agora a posição de cada bloco (sua coluna) depende de
*quantos* estão presentes, não só de desenhar um após o outro.

**Alternativas consideradas**: manter a sequência como `if`s na ordem
literal (sem a lista): rejeitada — calcular a coluna de cada bloco (índice
entre os presentes) é mais direto iterando uma lista do que replicando a
mesma ordem em condicionais soltos; a lista também documenta a ordem num
só lugar, para o comentário do código e para `data-model.md` apontarem.

## 3. Texto em três alturas — rótulo, valor, unidade

**Decisão**: duas novas constantes de corpo em `render_tuning.go`, ao lado
de `OverlayOutlineRatio` etc. — fração da altura do quadro, sem piso em
pixels novo (o texto da sobreposição já não tem piso hoje):

- `OverlayLabelHeightRatio` (rótulo e unidade, corpo pequeno) — `0.020`
- `OverlayValueHeightRatio` (valor, corpo bem maior) — `0.040`, o dobro do
  rótulo, para que o valor seja "o que a pessoa enxerga de relance" (FR-005)
  sem exigir um fator arbitrário maior.

Um bloco desenha, de cima para baixo, até três linhas centralizadas no
centro da própria coluna: rótulo (`OverlayLabelHeightRatio`), valor
(`OverlayValueHeightRatio`), unidade (`OverlayLabelHeightRatio`) — pulando
a terceira quando o bloco não tem unidade (tempo decorrido), sem reservar
o espaço que ela ocuparia (FR-006). O espaçamento vertical entre as linhas
reaproveita `overlayLinePadding` (a mesma fração de `lineHeight` que já
separa blocos hoje), aplicado entre cada par de linhas presentes.

**Racional**: duas constantes bastam para os três papéis (rótulo e
unidade compartilham o mesmo corpo pequeno, como o pedido descreve — "de
novo num corpo pequeno"); a proporção 1:2 entre rótulo e valor é uma
escolha de planejamento técnico (a especificação só exige "bem maior"),
documentada aqui para ficar explícita e revisável sem precisar de nova
clarificação. Nenhuma das duas é exposta por flag (FR-018).

**Alternativas consideradas**: uma terceira constante só para a unidade,
diferente do rótulo: rejeitada — o pedido original agrupa rótulo e
unidade como "de novo num corpo pequeno", isto é, o mesmo corpo; introduzir
uma terceira constante sem motivo do pedido só acrescentaria uma
superfície de ajuste que ninguém pediu.

## 4. O bloco `gain`, independente de `elevation`

**Decisão**: `OverlayBlockGain = "gain"` (nova constante, ao lado das
cinco existentes); `OverlayConfig.Gain bool` (novo campo);
`NewOverlayConfig` ganha um `case OverlayBlockGain: config.Gain = true`, e a
mensagem de erro de nome inválido passa a listar os seis nomes
(`distance, elevation, gain, time, profile, speed`, a mesma ordem de
declaração dos `case`s, não a ordem de exibição do item 2).
`elevationBlockText` para de concatenar o ganho (`"ELEV ... GANHO ..."`) e
passa a devolver só a altitude; uma nova `gainBlockText` devolve só o
ganho. O gate de exibição do bloco `gain` é idêntico ao de `elevation`:
`config.Gain && plan.ElevationAvailable` — o ganho acumulado não existe
sem elevação disponível, pelo mesmo motivo que a elevação em si não existe.

**Racional**: cumpre FR-008/FR-009/FR-010 com a menor mudança possível —
`elevation` e `gain` passam a ser dois blocos irmãos, cada um com seu
próprio texto e seu próprio gate, no lugar de um bloco que escondia dois
valores. Usar o mesmo gate de disponibilidade que `elevation` já tem é a
leitura mais direta do requisito (nenhum caso em que o ganho exista sem a
elevação do plano estar disponível).

**Alternativas consideradas**: um gate próprio para `gain`, independente
de `ElevationAvailable` (ex.: checar só se `TrackElevationGain` é
calculável): rejeitada — `TrackElevationGain` é derivado exatamente da
mesma `Route.ElevationProfile` que alimenta `TrackElevation`
(`CLAUDE.md`, etapa 9); não existe um plano em que um esteja disponível e o
outro não, então um segundo critério só duplicaria o mesmo sinal sem
nenhum caso real que o distinguisse.

## 5. Remoção do painel: `drawPanel`, `OverlayPanelColor`, `OverlayPanelOpacity`

**Decisão**: remover a função `drawPanel` e as duas constantes
(`OverlayPanelColor`, `OverlayPanelOpacity`) de `render_tuning.go` — depois
desta etapa, nada mais as chama: os blocos numéricos passam a desenhar só
texto (item 3) e o gráfico de elevação passa a usar contorno em vez de
painel (item 6).

**Racional**: código morto não é deixado no domínio só porque "pode servir
depois" — o próprio pedido (FR-001/FR-012) elimina o único uso de ambos.
Mantê-los sem nenhum chamador violaria a mesma disciplina que já levou a
etapa 11 a remover `overlayFace`/`textScale`/`glyphOffset` quando a fonte
bitmap saiu de uso.

**Alternativas consideradas**: manter as constantes, sem uso, para um
eventual retorno do painel: rejeitada — não há indicação de que o painel
volte, e reintroduzi-lo no futuro custaria o mesmo escrever a constante de
novo; manter código morto só adiciona ruído de manutenção agora.

## 6. Contorno para a linha e o marcador do gráfico de elevação

**Decisão**: `drawProfile` deixa de chamar `drawPanel` sobre a área do
gráfico. A linha do perfil (`drawSegment`/`plotSquare`) e o marcador
(`drawDot`) passam a usar a mesma técnica de casca-antes-do-núcleo que
`TrailCasingColor` já usa para o traçado sobre o terreno
(`frame_overlay.go`, `drawTrail`): cada segmento é desenhado primeiro
`thickness+2` pixels de espessura em `OverlayTextOutlineColor`, depois
`thickness` pixels em `OverlayTextColor`, por cima; o marcador, primeiro
com `radius+2` em `OverlayTextOutlineColor`, depois com `radius` na cor
escolhida (`appearance.MarkerColor`), por cima — o mesmo `+2` fixo que
`TrailCasingColor` aplica como `core+1` sobre o raio em pixels (aqui `+2`,
não `+1`, porque a linha do perfil é mais fina que o traçado do terreno e
precisa de uma casca proporcionalmente mais visível para não desaparecer
num pixel).

**Racional**: cumpre FR-012 reaproveitando uma técnica já estabelecida no
projeto (a mesma categoria que o contorno do texto já é, só aplicada a uma
linha/disco em vez de a um glifo) — nenhuma técnica nova, nenhuma
constante de proporção nova: um incremento fixo em pixels, like
`TrailCasingColor` já usa (`core+1`), e não uma fração da altura do quadro,
porque a linha do perfil e seu marcador já têm sua própria espessura/raio
calculados (`thickness`, `profileMarkerRadius`) — a casca só precisa somar
alguns pixels a eles, não reintroduzir uma proporção nova.

**Alternativas consideradas**:
- Reusar literalmente `drawGlyphOutline` (a dilatação de máscara de
  cobertura do texto): rejeitada — opera sobre uma `glyphMask` (um
  retângulo de cobertura por pixel vindo da rasterização de um glifo);
  a linha e o marcador do gráfico não são glifos, e adaptar a mesma função
  para uma forma geométrica arbitrária seria mais código do que desenhar a
  forma duas vezes (casca, depois núcleo) como o traçado já faz.
- Um contorno proporcional (`OverlayOutlineRatio` aplicado a `thickness`/
  `radius`): rejeitada como exigência — o pedido só pede "o mesmo recurso
  de contorno", não uma fórmula nova; um incremento fixo já é visível em
  qualquer resolução suportada e é mais simples de justificar.

## 7. Rótulos por extenso, em português, capitalização normal

**Decisão**: os cinco rótulos passam de abreviados em caixa alta para por
extenso, capitalização normal (só a primeira letra maiúscula):

| Bloco | Rótulo (antes) | Rótulo (depois) |
|---|---|---|
| `distance` | `DIST` | `Distância` |
| `elevation` | `ELEV` | `Elevação` |
| `gain` | `GANHO` (preso a `elevation`) | `Ganho` (bloco próprio) |
| `time` | `TEMPO` | `Tempo decorrido` |
| `speed` | `VEL` | `Velocidade` |

As abreviações de unidade (`m`, `km`, `km/h`) e o conteúdo/arredondamento
de cada valor não mudam (FR-013) — só a forma do rótulo e o fato de a
unidade passar a ser desenhada como sua própria linha, em vez de
concatenada ao valor na mesma string.

**Racional**: leitura direta de FR-005 ("por extenso e com capitalização
normal") e da Suposição do `spec.md` ("a mesma convenção... só a forma do
rótulo muda"); os nomes escolhidos são a forma por extenso mais direta de
cada rótulo abreviado já existente, sem introduzir nenhum termo novo.

**Alternativas consideradas**: capitalização total em maiúsculas por
extenso (ex.: "DISTÂNCIA"): rejeitada — o próprio FR-005 pede
explicitamente capitalização normal "em vez de abreviado em caixa alta",
então manter caixa alta (mesmo por extenso) não atenderia ao requisito.

## 8. Nova escolha padrão de blocos

**Decisão**: `config.RenderDefaults.OverlayBlocks` muda de
`[]string{"distance", "elevation", "time", "profile"}` para
`[]string{"distance", "elevation", "speed", "profile"}` — `time` sai,
`speed` entra; `gain` nunca esteve no padrão e continua fora. A ordem
**dentro dessa lista não importa para o desenho** (a ordem de exibição é
sempre `overlayBlockOrder`, item 2) — ela só precisa conter exatamente os
quatro nomes certos, na mesma lógica que `parseOverlay`/`NewOverlayConfig`
já tratam qualquer lista informada.

**Racional**: cumpre FR-011/SC-004 diretamente — é a mesma leitura
genérica de string→`OverlayBlock` que `domainOverlayConfig`
(`config_mapping.go`) já faz para a lista de hoje; nenhuma mudança de
assinatura ou de mapeamento é necessária, só o valor do slice padrão em
`config.go`.

**Alternativas consideradas**: nenhuma — é a troca de um valor de
configuração, sem decisão técnica alternativa real (o conjunto exato já
vem definido pelo `spec.md`/Clarifications).

## 9. Versão do desenho e impressão digital da configuração de sobreposição

**Decisão**: `domain.RenderVersion` sobe de `4` para `5`.
`OverlayConfig.Fingerprint()` ganha um sétimo campo (`flag(o.Gain)`, ao
lado dos seis já existentes) — como `NewFrameSetID` já inclui
`OverlayConfig.Fingerprint()` e `RenderVersion` no hash do `FrameSetID`
(etapas 9/11), as duas mudanças juntas bastam, sem nenhum código novo em
`frame_set.go`: um conjunto de quadros desenhado antes desta etapa (painel,
pilha vertical, sem `gain`) nunca é tomado como o de um conjunto desenhado
depois — `render all` recusa sem `--overwrite` (`ErrFrameSetConflict`,
código já existente), `fly --keep` redesenha ao notar a mudança.

**Racional**: é o mesmo mecanismo comprovado nas etapas 6, 8, 9 e 11 — bump
de versão e/ou de impressão digital, zero código novo de comparação.
Subir a versão é necessário porque os pixels de qualquer quadro com
sobreposição ligada mudam de verdade (painel sai, colunas mudam de
posição); a impressão digital precisa do bit de `gain` porque dois
`OverlayConfig` que diferem só nesse campo já desenhavam exatamente os
mesmos pixels antes desta etapa (quando `gain` não existia) e passam a
desenhar pixels diferentes agora.

**Alternativas consideradas**: nenhuma — inventar um mecanismo diferente
de invalidação de conjunto de quadros seria uma regressão de consistência
sem nenhum benefício sobre o que já existe.

## 10. Builders e texto de ajuda da CLI afetados

**Decisão**: `builddomain.OverlayConfigBuilder` ganha `WithGain()`, mesmo
padrão de `WithSpeed()` (etapa 14) — liga só o bit de `gain`, sem mudar o
padrão construído por `NewOverlayConfigBuilder()`. `overlay.go`
(`overlaysUsage`/`overlayBlocksUsage`) passa a mencionar `gain` entre os
nomes aceitos e a nova composição do padrão; `overlayBlocksOf` ganha o
ramo `if config.Gain`, na mesma função que já lista os blocos ligados para
`--help` (a ordem ali não precisa ser `overlayBlockOrder` — é só a lista
default mostrada em `--help`, não a ordem de desenho).

**Racional**: mantém os dois pontos de extensão já estabelecidos pelas
etapas anteriores (builder de teste, texto de ajuda) consistentes com o
padrão que `WithSpeed()`/a menção a `speed` em `overlay.go` já seguiram.

**Alternativas consideradas**: nenhuma — é a extensão direta do padrão já
existente, sem variação a avaliar.
