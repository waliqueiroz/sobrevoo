# Contrato dos Arquivos de Quadro

**Feature**: `005-frame-rendering` | **Data**: 2026-09-26

É o contrato de dados que a etapa seguinte (codificar o vídeo) e qualquer
ferramenta externa consomem: as imagens PNG que `sobrevoo render all` grava num
diretório e `sobrevoo render frame` grava num arquivo. O mesmo plano, o mesmo
recorte e a mesma resolução produzem imagens idênticas byte a byte, para o mesmo
binário e plataforma (FR-015, SC-002; `research.md` item 12) — o que exclui data e
hora de geração, caminhos, nome de máquina, tempo gasto e qualquer dado do
ambiente.

## Nome e ordem

`frame_NNNNNN.png`, com `NNNNNN` o **índice do quadro no plano**
(`frames[].index`, a partir de 0), com **6 dígitos** e zeros à esquerda
(`frame_000000.png`, `frame_000300.png`). O plano da etapa 3 tem no máximo 432
000 quadros, que cabem em 6 dígitos. A ordem alfabética dos nomes é a ordem
numérica; um desenho completo tem exatamente `frame_count` arquivos, de 0 a
`frame_count − 1`, sem lacunas nem duplicatas (FR-016). Uma ferramenta de vídeo
lê o diretório com o padrão `frame_%06d.png` a partir de 0.

O quadro isolado (`render frame`) vai para o caminho que o usuário deu, com
qualquer nome; o conteúdo é o do arquivo `frame_NNNNNN.png` do mesmo número.

## Imagem

- **PNG**, profundidade de 8 bits por canal, **RGB** (sem canal alfa), sem
  entrelaçamento, compressão padrão do `image/png` da biblioteca padrão de Go.
- Dimensões exatamente as da resolução pedida (largura × altura).
- Um bloco `tEXt` logo depois do `IHDR`, com a palavra-chave `Sobrevoo` e o
  texto `frame-set=<id>`, em que `<id>` é o identificador do conjunto: SHA-256
  hexadecimal (64 caracteres minúsculos) de `"sobrevoo-frames" |
  RenderVersion | plan.ID() | slice.ContentID | largura | altura |
  RenderTuning.Fingerprint()` (`research.md` item 17). É o que permite
  reconhecer quadros da ferramenta e a que conjunto pertencem.
- Um **segundo** bloco `tEXt`, logo depois do primeiro, com a palavra-chave
  `Sobrevoo` e o texto `plan=<id>`, em que `<id>` é o `CameraPlan.ID()` do plano de
  que o quadro foi desenhado (SHA-256 hexadecimal, 64 caracteres minúsculos): é
  o que a etapa que junta os quadros num vídeo confere com o plano que recebe
  (acrescentado na sexta etapa, versão do desenho 2; ver
  `specs/006-video-assembly/contracts/frame-files-change.md`). Nenhum outro
  bloco auxiliar é gravado.
- **Sem** data, hora, caminho, versão do binário ou resumo dentro do arquivo.

## O que a imagem mostra

Do ponto de vista da câmera do quadro (posição, rumo e inclinação do plano,
altura absoluta somada à elevação do recorte; campo de visão vertical de 45°,
pixels quadrados), em perspectiva:

1. **Terreno** com o relevo das amostras de elevação (superfície contínua,
   com oclusão correta), vestido com as peças do mapa base e iluminado por
   uma luz direcional fixa, conforme a inclinação real da superfície em
   cada ponto (016-terrain-lighting; ver nota da décima sexta etapa).
2. **Traçado** do trajeto: a linha das posições do marcador dos quadros `0` a
   `k` (`k` = o quadro desenhado), sobre o terreno; só o trecho já percorrido,
   terminando no marcador. O primeiro quadro (marcador no início) não tem
   linha.
3. **Marcador** da atividade: um disco na posição do marcador do quadro, sobre
   o terreno, por cima do traçado.
4. **Fundo** onde o raio não encontra terreno.

O traçado e o marcador obedecem à oclusão do terreno.

### Cores e padrões (constantes do domínio; mudam só com `RenderVersion`)

| Elemento | Aparência |
|---|---|
| Fundo (fora do recorte; céu) | cor lisa `#20262E` |
| **Terreno com imagem de mapa** | a cor da peça do mapa, multiplicada por um fator de brilho fixo entre `0,75` e `1,15`, conforme a inclinação real da superfície naquele ponto em relação a uma luz direcional fixa (azimute 315°, altura 45° — 016-terrain-lighting; ver nota da décima sexta etapa) |
| **Sem imagem de mapa** (peça ausente do recorte, ou terreno onde não há peças, como além de ±85,0511°) | hachura diagonal de 45° com período de 12 px na tela, alternando `#C8C8C8` e `#6E6E6E` — **não iluminada** |
| **Sem elevação** (célula do relevo sem valor) | xadrez de quadrados de 12 px na tela, alternando `#FF00FF` e `#3A003A` — **não iluminado** |
| Traçado | linha de largura `max(2, 0,5% da altura da imagem)` px em `#FFB000`, com contorno de 1 px em `#101010` — **não iluminado** |
| Marcador | disco de raio `max(4, 1,2% da altura)` px em `#E5252A`, com anel branco `#FFFFFF` de `max(1,5, 0,3% da altura)` px — **não iluminado** |

Os padrões dependem só da posição do **pixel na tela**, com `x` a coluna e `y`
a linha, a partir de 0 no canto superior esquerdo: na hachura, `#C8C8C8` onde
`(x + y) mod 12 < 6` e `#6E6E6E` no resto; no xadrez, `#FF00FF` onde
`(⌊x/12⌋ + ⌊y/12⌋) mod 2 = 0` e `#3A003A` no resto. Uma célula sem elevação
prevalece sobre a falta de imagem no mesmo pixel. Pixels de peça com
transparência são compostos sobre `#20262E`.

### Contabilização (o que o resumo conta)

Sobre os pixels de **terreno** (raios que acertaram algo), antes do traçado e
do marcador: o quadro tem **buraco de mapa** se algum pixel é "sem imagem de
mapa"; **buraco de elevação** se algum pixel é "sem elevação". O fundo nunca
conta. Um quadro com os dois tipos de pixel conta nas duas colunas.

## Garantias verificáveis (usadas nos testes e no quickstart)

1. Um desenho completo de `N` quadros produz `N` arquivos `frame_000000.png` a
   `frame_{N-1}.png`, todos PNG válidos, todos com a largura e a altura
   pedidas e a marca do mesmo conjunto.
2. Desenhar o quadro `k` sozinho, dentro do conjunto inteiro, ou numa execução
   retomada, produz o mesmo arquivo, byte a byte.
3. O centro do marcador cai a no máximo 1 px da projeção do ponto do marcador
   pela câmera do quadro; e, em duas resoluções de mesma proporção, na mesma
   posição relativa (± 1 px na menor).
4. O quadro `k` tem o traçado do quadro `k − 1` mais o trecho entre os dois;
   nada adiante do marcador.
5. Um pixel com a cor `#FF00FF`/`#3A003A` (xadrez) ou `#C8C8C8`/`#6E6E6E`
   (hachura) só aparece onde há falta de dado; um quadro sem faltas não os
   contém (salvo cor idêntica no mapa real, que os testes evitam).
6. O arquivo de um quadro truncado (sem `IEND` no fim) é reconhecido como
   incompleto e redesenhado na retomada.

## Compatibilidade

O arquivo não referencia data, hora, caminhos, nome de máquina nem versão do
binário — para preservar a igualdade byte a byte. `RenderVersion` (hoje `6`)
sobe sempre que o algoritmo ou as constantes visuais mudam de forma
visível; quadros de outra `RenderVersion` pertencem a outro conjunto e não são
reaproveitados. Consumidores devem ignorar blocos auxiliares que não
conhecem.

**Nota da sexta etapa**: a versão do desenho passou de `1` para `2` quando os
quadros passaram a trazer a identificação do plano dentro deles; os pixels não
mudaram. Quadros da versão `1` são, portanto, de outro conjunto (`render all` os
recusa sem `--overwrite` e os redesenha com ele), e a etapa 6 (`sobrevoo video`)
os recusa por não trazerem a identificação do plano.

**Nota da décima primeira etapa**: a versão do desenho passou de `2` para `3`
— desta vez **os pixels mudam**, no acabamento das sobreposições de tela que a
nona etapa introduziu (texto, painéis numéricos, marcador do perfil de
elevação e margem de segurança; ver `specs/009-frame-overlays/data-model.md`
e `specs/011-overlay-polish/contracts/frame-files-change.md`): o texto passa
de uma fonte bitmap ampliada por fator inteiro para uma fonte vetorial
embutida (`golang.org/x/image/font/gofont/goregular`), rasterizada por um
rasterizador próprio e determinístico — nunca
`golang.org/x/image/vector.Rasterizer`, que tem um caminho em assembly só
para amd64 —, com um contorno escuro fixo; os três painéis numéricos
(distância; elevação e ganho; tempo decorrido) passam a compartilhar a
largura do mais largo presente no quadro; o marcador do perfil de elevação
ganha um raio proporcional à altura do quadro, com piso em pixels; e a
margem de segurança passa de uma única razão igual nas quatro bordas para
três — topo e laterais, e uma maior na base, adequada ao vídeo vertical.
Nenhum valor exibido, bloco, enquadramento, terreno ou traçado muda.
Quadros da versão `2` são, portanto, de outro conjunto, pela mesma regra de
sempre.

**Nota da décima segunda etapa**: a versão do desenho passou de `3` para
`4` (`specs/012-overlay-ptbr-readability/contracts/frame-files-change.md`):
os rótulos das sobreposições passam para português do Brasil, a fonte
embutida ganha um peso mais forte, o contorno do texto passa a ser fino e
proporcional ao tamanho da letra, e a largura dos três painéis numéricos
deixa de pulsar quadro a quadro, sendo computada uma única vez por voo.
Nenhum valor exibido muda. Quadros da versão `3` são de outro conjunto.

**Nota da décima quinta etapa**: a versão do desenho passou de `4` para `5`
(`specs/015-overlay-redesign/contracts/frame-files-change.md`): os blocos
numéricos perdem o painel de fundo e passam a ser desenhados lado a lado,
em colunas de mesma largura, numa faixa horizontal única, em três alturas
de texto (rótulo por extenso, valor, unidade) em vez de uma linha
empilhada; o ganho de elevação acumulado deixa de ser um segundo número
colado ao bloco de elevação e vira um bloco próprio (`gain`); e o gráfico
de elevação no rodapé também perde o painel, destacando-se do terreno por
contorno. Nenhum valor exibido, cálculo, arredondamento, unidade,
enquadramento, terreno ou traçado muda. Quadros da versão `4` são,
portanto, de outro conjunto.

**Nota da décima sexta etapa**: a versão do desenho passou de `5` para `6`
(`specs/016-terrain-lighting/contracts/frame-files-change.md`): o terreno
vestido com imagem de mapa passa a ser iluminado por uma luz direcional
fixa (azimute 315°, altura 45°, nunca escolhida pelo usuário), que clareia
ou escurece a cor do mapa conforme a inclinação real da superfície em cada
ponto, numa faixa fixa entre `0,75` e `1,15` — uma superfície plana
permanece exatamente igual à cor de antes desta etapa, qualquer que seja a
altura da luz. O traçado, o marcador, as sobreposições de tela e os dois
padrões de "sem dado" (hachura e xadrez) continuam sem nenhuma mudança —
só a cor dos pixels de terreno com imagem de mapa muda. Quadros da versão
`5` são, portanto, de outro conjunto.
