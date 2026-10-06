# Quickstart: validação manual da Iluminação Direcional do Terreno

**Feature**: `016-terrain-lighting` | **Data**: 2026-10-05

Checklist manual, com o binário real, para conferir que a iluminação do
terreno funciona como o `spec.md` exige: volume real no relevo (História
1), mapa ainda legível na encosta mais escura (História 2), estabilidade
sem cintilação (História 3), nenhum efeito sobre traçado/marcador/
sobreposições/padrões de "sem dado", e o não-reaproveitamento de quadros
antigos. Não há teste automatizado de ponta a ponta; ver `plan.md`.
Contrato: [`contracts/frame-files-change.md`](./contracts/frame-files-change.md).
Os itens marcados **(observar)** são conferidos abrindo a imagem — não há
métrica automática de "volume" ou "legibilidade".

## Pré-requisitos

```sh
make build
BIN=$PWD/bin/sobrevoo
SVHOME=$(mktemp -d)
sv() { HOME=$SVHOME $BIN "$@"; }
A=$PWD/specs/005-frame-rendering/amostras
G=specs/003-camera-path-planning/amostras
go run ./test/samples --out $A
sv geodata register $A/mapa-imagem-sp.mbtiles --name mapa
sv geodata register $A/relevo-sp.tif --name relevo
TRACK=$G/pedalada.gpx

rm -rf /tmp/terrain-lighting && mkdir /tmp/terrain-lighting
sv plan $TRACK --duration 10 --fps 10 --export /tmp/terrain-lighting/plan.json
sv geodata slice /tmp/terrain-lighting/plan.json --export /tmp/terrain-lighting/slice.zip
PLAN=/tmp/terrain-lighting/plan.json
SLICE=/tmp/terrain-lighting/slice.zip
```

`relevo-sp.tif` tem uma superfície em dente de serra (altura = 700 + (3·linha
+ 2·coluna) mod 400 metros): cada faixa diagonal tem uma inclinação
diferente, o que dá, num único quadro, encostas voltadas em várias direções
— útil para ver a variação de brilho sem precisar de um relevo real.

## 1. Volume real: encostas com tons diferentes na mesma cor de mapa (História 1 / FR-001/FR-002/FR-005 / SC-001)

```sh
sv render frame $PLAN $SLICE --number 150 --output /tmp/terrain-lighting/voo-amplo.png \
  --resolution 1080x1920
```

**(observar)** `voo-amplo.png`: o terreno não é mais um tom uniforme por
quadradinho do xadrez do mapa — dentro de cada quadradinho (que antes desta
etapa tinha uma única cor sólida), agora há uma variação de claro/escuro
que acompanha as faixas diagonais da inclinação do relevo sintético; olhando
o quadro de lado a lado, algumas faixas estão visivelmente mais claras e
outras mais escuras que o tom "base" do mapa.

## 2. Mapa continua legível na parte mais escura (História 2 / FR-003/FR-004 / SC-002)

```sh
sv render frame $PLAN $SLICE --number 150 --output /tmp/terrain-lighting/legibilidade.png \
  --resolution 1080x1920 --background-color "#101418"
```

**(observar)** `legibilidade.png`: mesmo na faixa mais escurecida pela
iluminação, o padrão de xadrez do mapa base (`mapa-imagem-sp.mbtiles`)
continua distinguível — dois quadradinhos vizinhos do xadrez, ambos na
mesma faixa escura do relevo, ainda têm tons diferentes entre si (a
iluminação escureceu os dois pela mesma proporção, sem apagar a diferença
entre eles); nenhuma área vira preto ou branco uniforme.

## 3. Nenhum efeito sobre traçado, marcador, sobreposições e padrões de "sem dado" (FR-006 / SC-006)

```sh
sv render frame $PLAN $SLICE --number 150 --output /tmp/terrain-lighting/elementos.png \
  --resolution 1080x1920 --trail-color "#00FF00" --marker-color "#FF8800" \
  --overlay-blocks distance,elevation,speed,gain,time
```

**(observar)** `elementos.png`: o traçado aparece exatamente em
`#00FF00` e o marcador em `#FF8800` (a mesma cor pedida, nunca clareada ou
escurecida por estarem sobre uma encosta clara ou escura); o texto e os
números das sobreposições de tela mantêm a cor fixa de sempre
(`OverlayTextColor`/contorno), sem nenhuma variação de brilho ligada ao
relevo por baixo. A faixa de "sem mapa" (fora da área coberta por
`mapa-imagem-sp.mbtiles`, se visível no ângulo) continua no hachurado
cinza de sempre, sem nenhuma tonalidade nova.

```sh
sv render frame $PLAN $SLICE --number 150 --output /tmp/terrain-lighting/sem-elevacao.png \
  --resolution 1080x1920
```

**(observar)**: se o quadro cruzar a área sem dado de `relevo-sp.tif`
(linhas 240–249, colunas 300–309), o xadrez magenta/roxo de "sem elevação"
continua exatamente nas duas cores fixas de sempre (`NoElevationColors`),
sem nenhuma variação de brilho.

## 4. Vizinhança incompleta perto de um buraco de elevação não inventa nem apaga dado (FR-008)

```sh
sv geodata register $A/relevo-buraco.tif --name relevo-com-buraco
rm -rf /tmp/terrain-lighting/buraco && mkdir /tmp/terrain-lighting/buraco
sv plan $TRACK --duration 10 --fps 10 --export /tmp/terrain-lighting/buraco/plan.json
sv geodata slice /tmp/terrain-lighting/buraco/plan.json --export /tmp/terrain-lighting/buraco/slice.zip \
  --elevation relevo-com-buraco
sv render frame /tmp/terrain-lighting/buraco/plan.json /tmp/terrain-lighting/buraco/slice.zip \
  --number 5 --output /tmp/terrain-lighting/buraco/perto-do-buraco.png --resolution 1080x1920
```

**(observar)** `perto-do-buraco.png`: o bloco sem valor (linhas 345–359,
colunas 360–379, perto do início de `pedalada.gpx`) continua no xadrez
magenta/roxo fixo; os pontos com elevação válida ao redor dele continuam
mostrando a cor do mapa, com uma iluminação que pode ficar um pouco mais
próxima do tom neutro perto da borda do buraco (porque a vizinhança de
amostragem ali tem menos amostras reais), mas nunca viram xadrez nem ficam
pretos — nenhuma área de mapa válido é substituída pelo padrão de "sem
elevação" por estar perto do buraco.

## 5. Estabilidade entre quadros a distâncias diferentes (História 3 / FR-007 / SC-003)

```sh
rm -rf /tmp/terrain-lighting/longe /tmp/terrain-lighting/perto
mkdir /tmp/terrain-lighting/longe /tmp/terrain-lighting/perto
sv plan $TRACK --duration 10 --fps 10 --distance high --export /tmp/terrain-lighting/longe/plan.json
sv geodata slice /tmp/terrain-lighting/longe/plan.json --export /tmp/terrain-lighting/longe/slice.zip
sv plan $TRACK --duration 10 --fps 10 --distance low --export /tmp/terrain-lighting/perto/plan.json
sv geodata slice /tmp/terrain-lighting/perto/plan.json --export /tmp/terrain-lighting/perto/slice.zip

sv render frame /tmp/terrain-lighting/longe/plan.json /tmp/terrain-lighting/longe/slice.zip \
  --number 50 --output /tmp/terrain-lighting/longe/quadro.png --resolution 1080x1920
sv render frame /tmp/terrain-lighting/perto/plan.json /tmp/terrain-lighting/perto/slice.zip \
  --number 50 --output /tmp/terrain-lighting/perto/quadro.png --resolution 1080x1920
```

**(observar)** `longe/quadro.png` (câmera a `--distance high`, terreno
visto de mais longe) **vs.** `perto/quadro.png` (`--distance low`, terreno
visto de perto): no quadro de longe, onde um pixel cobre uma faixa maior
do relevo em dente de serra, a variação de claro/escuro aparece suave, sem
um padrão granulado pixel a pixel, mesmo cobrindo várias faixas finas do
relevo sintético de uma vez; no quadro de perto, a variação acompanha de
perto cada faixa individual. Em nenhum dos dois a iluminação parece
"ruído" (pixels isolados claros/escuros sem relação com os vizinhos) —
confirma a leitura adaptativa pela distância (research.md item 6).

## 6. Quadros de antes desta etapa nunca se juntam aos de depois (FR-010)

```sh
rm -rf /tmp/terrain-lighting/frames
sv render all $PLAN $SLICE --output /tmp/terrain-lighting/frames --resolution 480x854
sv render all $PLAN $SLICE --output /tmp/terrain-lighting/frames --resolution 480x854
echo "código: $?"   # esperado: 0 — retomada do mesmo conjunto, nada a redesenhar
```

**(observar)** a segunda chamada termina com sucesso, sem recusar nada
(mesmo `RenderVersion` 6, mesmo `FrameSetID`); um diretório de quadros
desenhado por um binário anterior a esta etapa (`RenderVersion` 5) é
recusado com `ErrFrameSetConflict` sem `--overwrite`, e redesenhado com
`--overwrite` ou por `fly --keep`.

## 7. Determinismo byte a byte preservado (FR-009 / SC-004)

```sh
sv render frame $PLAN $SLICE --number 150 --output /tmp/terrain-lighting/det1.png --resolution 1080x1920
sv render frame $PLAN $SLICE --number 150 --output /tmp/terrain-lighting/det2.png --resolution 1080x1920 --overwrite

cmp /tmp/terrain-lighting/det1.png /tmp/terrain-lighting/det2.png && echo "IDÊNTICOS" || echo "DIFERENTES (falha — anotar o diff)"
```

**(observar)** esperado `IDÊNTICOS`. Se o binário foi compilado e executado
em outra arquitetura (amd64 vs. arm64) para o mesmo plano/recorte/
resolução/aparência/sobreposição, repetir e comparar os dois arquivos
também devem ser idênticos — a garantia central desta etapa
(`research.md` item 10).
