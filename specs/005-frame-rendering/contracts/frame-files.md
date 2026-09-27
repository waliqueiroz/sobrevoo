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
  reconhecer quadros da ferramenta e a que conjunto pertencem. Nenhum outro
  bloco auxiliar é gravado.
- **Sem** data, hora, caminho, versão do binário ou resumo dentro do arquivo.

## O que a imagem mostra

Do ponto de vista da câmera do quadro (posição, rumo e inclinação do plano,
altura absoluta somada à elevação do recorte; campo de visão vertical de 45°,
pixels quadrados), em perspectiva:

1. **Terreno** com o relevo das amostras de elevação (superfície contínua,
   com oclusão correta), vestido com as peças do mapa base, **sem iluminação
   nem sombra**.
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
| **Sem imagem de mapa** (peça ausente do recorte, ou terreno onde não há peças, como além de ±85,0511°) | hachura diagonal de 45° com período de 12 px na tela, alternando `#C8C8C8` e `#6E6E6E` |
| **Sem elevação** (célula do relevo sem valor) | xadrez de quadrados de 12 px na tela, alternando `#FF00FF` e `#3A003A` |
| Traçado | linha de largura `max(2, 0,5% da altura da imagem)` px em `#FFB000`, com contorno de 1 px em `#101010` |
| Marcador | disco de raio `max(4, 1,2% da altura)` px em `#E5252A`, com anel branco `#FFFFFF` de `max(1,5, 0,3% da altura)` px |

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
binário — para preservar a igualdade byte a byte. `RenderVersion` (hoje `1`)
sobe sempre que o algoritmo ou as constantes visuais mudam de forma
visível; quadros de outra `RenderVersion` pertencem a outro conjunto e não são
reaproveitados. Consumidores devem ignorar blocos auxiliares que não
conhecem.
