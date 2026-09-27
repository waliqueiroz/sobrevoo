# Quickstart: validação manual do Desenho dos Quadros do Voo

**Feature**: `005-frame-rendering` | **Data**: 2026-09-26

Checklist manual, com o binário real, para conferir a etapa de ponta a ponta
(não há teste automatizado de ponta a ponta; ver `plan.md`). Contratos:
[`contracts/cli.md`](./contracts/cli.md),
[`contracts/frame-files.md`](./contracts/frame-files.md) e
[`contracts/slice-file-change.md`](./contracts/slice-file-change.md). Os
cenários usam **dados sintéticos** gerados em código (sem baixar nada), para que
o que se vê seja conhecido; o item 10 repete com os dados reais de
`resources/`. Todos os resultados abaixo foram conferidos com o binário real
(em um Apple M1, 8 núcleos); os tempos variam com a máquina.

## Pré-requisitos

Todos os comandos rodam a partir da **raiz do repositório** (zsh ou bash).

```sh
make build                                        # gera bin/sobrevoo
SVHOME=$(mktemp -d)                               # o registro fica em $HOME/.sobrevoo/registry.json
sv() { HOME=$SVHOME ./bin/sobrevoo "$@"; }        # registro isolado para este roteiro
A=$PWD/specs/005-frame-rendering/amostras         # caminho absoluto: o registro guarda o caminho como dado
G=specs/003-camera-path-planning/amostras
go run ./test/samples --out $A                    # imprime as células e peças conhecidas
```

O padrão do `plan` e do `render` é vídeo **vertical** (`--aspect 9:16` e
`1080x1920`). Os roteiros abaixo desenham em resoluções horizontais, então os
planos são gerados com `--aspect 16:9` (um plano 9:16 enquadra a abertura e o
fechamento mais longe, e o mínimo de duração de `pedalada.gpx` sobe de 31,2 s
para 37 s); o item 8 desenha também em retrato, e o item 10 mostra o voo em
vertical.

`test/samples` (ferramenta de desenvolvimento, usa `test/helper`) passa a gravar,
em `amostras/` (não versionada), além dos arquivos da etapa 4:

| Arquivo | O que é |
|---|---|
| `mapa-imagem-sp.mbtiles` | mapa base **em imagem (PNG)**, níveis 10 a 14, sobre a área de `pedalada.gpx`: cada peça é um xadrez de dois tons, com o tom e uma borda que dependem da posição da peça, para se reconhecer onde cada peça cai; **3 peças removidas** no nível 14, no início do trajeto |
| `mapa-vetorial-sp.mbtiles` | mesma área, com peças `pbf` (para a recusa) |
| `relevo-sp.tif` | o da etapa 4: rampa em dentes de serra entre 700 e 1099 m, com um bloco de "sem dado" (linhas 240–249, colunas 300–309), longe do trajeto |
| `relevo-buraco.tif` | o mesmo, com mais um bloco de "sem dado" de 15 × 20 células (linhas 345–359, colunas 360–379) **sob o início do trajeto** de `pedalada.gpx` |
| `relevo-sem-dado.tif` | GeoTIFF em que nenhuma célula tem valor |
| `mapa-imagem-antimeridiano.mbtiles`, `relevo-antimeridiano.tif` | em imagem, em torno de (−16,5; 180) |
| `mapa-imagem-polar.mbtiles`, `relevo-polar.tif` | em imagem, em torno de (82; 15) |

Ver as imagens: `open quadro.png` (macOS) ou qualquer visualizador. Ver a
resolução: `sips -g pixelWidth -g pixelHeight quadro.png` (macOS) ou `file
quadro.png`.

## 1. Um quadro isolado (História 1)

```sh
sv geodata register $A/mapa-imagem-sp.mbtiles --name mapa
sv geodata register $A/relevo-sp.tif --name relevo
sv plan $G/pedalada.gpx --aspect 16:9 --export /tmp/plano.json --overwrite          # 1260 quadros
sv geodata slice /tmp/plano.json --export /tmp/recorte.zip --overwrite
rm -f /tmp/quadro.png; sv render frame /tmp/plano.json /tmp/recorte.zip --number 300 --output /tmp/quadro.png; echo $?
```

Esperado: código `0`; resumo com `Frame 300 of … drawn to /tmp/quadro.png`,
`Resolution: 1080x1920` e `Time: 00:00:01` (a meta é < 10 s). Abrir
`/tmp/quadro.png`: o terreno com o relevo em perspectiva, vestido com o
xadrez das peças; a linha do traçado, do começo até o marcador; o disco vermelho
do marcador. Repetir com `--number 0` (o marcador no início: **sem** traçado) e
com o último quadro (o traçado completo, terminando no marcador).

## 2. Determinismo (SC-002)

```sh
rm -f /tmp/q1.png /tmp/q2.png /tmp/g1.png /tmp/g8.png
sv render frame /tmp/plano.json /tmp/recorte.zip --number 300 --output /tmp/q1.png
sv render frame /tmp/plano.json /tmp/recorte.zip --number 300 --output /tmp/q2.png
cmp /tmp/q1.png /tmp/q2.png && echo IDENTICO
cmp /tmp/q1.png /tmp/quadro.png && echo IDENTICO-AO-PRIMEIRO
```

E com um número de processadores diferente: `GOMAXPROCS=1` e `GOMAXPROCS=8`
produzem o mesmo arquivo:

```sh
GOMAXPROCS=1 sv render frame /tmp/plano.json /tmp/recorte.zip --number 300 --output /tmp/g1.png
GOMAXPROCS=8 sv render frame /tmp/plano.json /tmp/recorte.zip --number 300 --output /tmp/g8.png
cmp /tmp/g1.png /tmp/g8.png && echo IDENTICO
```

## 3. Falta de dado, explícita e contabilizada (História 3)

O recorte de `mapa-imagem-sp` tem 3 peças ausentes (nível 14), no início do
trajeto, e `relevo-buraco.tif` tem um bloco sem dado sob o mesmo lugar. Usa-se um
plano curto (32 s a 2 quadros por segundo, 64 quadros; `pedalada.gpx` pede no
mínimo 31,2 s):

```sh
sv geodata remove relevo && sv geodata register $A/relevo-buraco.tif --name relevo
sv plan $G/pedalada.gpx --duration 32 --fps 2 --aspect 16:9 --export /tmp/curto.json --overwrite
sv geodata slice /tmp/curto.json --export /tmp/buraco.zip --overwrite          # Elevation samples: 100868 (400 without value)
rm -f /tmp/h12.png /tmp/h30.png
sv render frame /tmp/curto.json /tmp/buraco.zip --number 12 --output /tmp/h12.png
sv render frame /tmp/curto.json /tmp/buraco.zip --number 30 --output /tmp/h30.png
```

Esperado: `h12.png` traz, onde caem as peças ausentes, a **hachura cinza** em
diagonal, sobre o terreno, que continua com o seu relevo; onde cai o bloco sem
dado, o **xadrez magenta** (que prevalece sobre a hachura onde os dois faltam);
o marcador e o traçado sobre elas. O resumo diz `Holes: map tiles missing: yes,
elevation without value: yes`. `h30.png`, no meio do trajeto, só enxerga
terreno completo: `no`/`no`, e não contém magenta nem hachura. A contagem de
todos os quadros está no item 4.

## 4. Todos os quadros (História 2)

O mesmo plano curto (64 quadros), com o relevo com buraco do item 3 e depois com
o `relevo-sp.tif` original:

```sh
rm -rf /tmp/quadros && sv render all /tmp/curto.json /tmp/buraco.zip --output /tmp/quadros; echo $?
ls /tmp/quadros | head -3; ls /tmp/quadros | wc -l                    # frame_000000.png …; 64
```

Esperado: código `0`; progresso em `stderr`; 64 arquivos `frame_000000.png` a
`frame_000063.png`, todos 1080 × 1920 (`sips`); resumo `Frames: 64 requested, 64
drawn, 0 kept`, `Resolution: 1080x1920`, `Time` de cerca de 40 s (≈ 0,6 s por
quadro) e `Holes (in the frames drawn now): 24 with missing map tiles, 19 with
elevation without value`. Com o `relevo-sp.tif` (sem o buraco sob o trajeto), a
contagem de elevação é `0`. O quadro isolado é igual ao do conjunto:

```sh
rm -f /tmp/q30.png; sv render frame /tmp/curto.json /tmp/buraco.zip --number 30 --output /tmp/q30.png
cmp /tmp/q30.png /tmp/quadros/frame_000030.png && echo IDENTICO
```

Ordem para a etapa seguinte: `ls /tmp/quadros` está em ordem numérica. Conferir o
traçado crescendo: abrir `frame_000000.png`, `frame_000030.png` e
`frame_000063.png`.

## 5. Interrupção e retomada (História 5)

O sinal precisa chegar ao **binário**, não a uma função de shell: rode-o
diretamente (com o registro isolado em `HOME`).

```sh
rm -rf /tmp/qA /tmp/qB
sv render all /tmp/curto.json /tmp/buraco.zip --output /tmp/qA >/dev/null 2>&1              # referência, ininterrupta
HOME=$SVHOME ./bin/sobrevoo render all /tmp/curto.json /tmp/buraco.zip --output /tmp/qB > /tmp/run.out 2>/tmp/run.err & PID=$!
sleep 9; kill -INT $PID; wait $PID; echo "codigo: $?"                                       # 38
cat /tmp/run.out; ls /tmp/qB | wc -l; ls -a /tmp/qB | grep -c tmp                           # menos de 64; 0 temporários
```

Esperado: `Interrupted: 14 of 64 frames are ready; run the same command again to
continue`, o resumo parcial e o código `38`; nenhuma imagem pela metade. Ajuste
o `sleep` para cair no meio. Continuar, e comparar com a referência:

```sh
sv render all /tmp/curto.json /tmp/buraco.zip --output /tmp/qB 2>/dev/null | head -4          # 50 drawn, 14 kept
diff -r /tmp/qA /tmp/qB && echo IDENTICOS
```

Esperado: a segunda execução diz `14 kept (already in the destination)`, desenha
só os 50 que faltam, sai com `0`, e o conjunto é idêntico ao da execução
ininterrupta. Uma terceira execução: `0 drawn, 64 kept`, `Holes … none drawn now`,
nada muda.

## 6. Proteção do destino (História 6)

```sh
sv render all /tmp/curto.json /tmp/buraco.zip --output /tmp/qB --resolution 1280x720; echo $?     # 37: outra resolução
ls /tmp/qB | wc -l                                                                                # continua 64, intactos
sv plan $G/pedalada.gpx --duration 32 --fps 1 --aspect 16:9 --export /tmp/menor.json --overwrite                # 32 quadros
sv geodata slice /tmp/menor.json --export /tmp/menor.zip --overwrite >/dev/null
sv render all /tmp/menor.json /tmp/menor.zip --output /tmp/qB --resolution 960x540 --overwrite | head -6
ls /tmp/qB | wc -l                                                                                # 32
echo "nota" > /tmp/qB/leia-me.txt; touch /tmp/qB/frame_1.png                                        # arquivos que não são quadros da ferramenta
sv render all /tmp/menor.json /tmp/menor.zip --output /tmp/qB --resolution 960x540 | head -2      # 32 kept; leia-me.txt e frame_1.png intactos
```

Esperado: a primeira recusa com `frame destination holds frames of another set: 64
frames are of another set; use --overwrite to replace them, or another --output`
e código `37`; a segunda, com `--overwrite`, desenha os 32 quadros do plano novo,
imprime `Removed: 32 frames from a previous set` (os quadros 32 a 63 do voo anterior)
e deixa o diretório com 32 quadros; a terceira mantém tudo. Nada que não seja
`frame_NNNNNN.png` da ferramenta é tocado.

Quadro isolado sobre um arquivo existente:

```sh
sv render frame /tmp/plano.json /tmp/recorte.zip --number 300 --output /tmp/quadro.png; echo $?               # 36 (use --overwrite)
sv render frame /tmp/plano.json /tmp/recorte.zip --number 300 --output /tmp/quadro.png --overwrite | head -1   # Frame 300 of 1260 drawn to …
sv render frame /tmp/plano.json /tmp/recorte.zip --number 300 --output /tmp/nao-existe/q.png; echo $?         # 35
```

Truncar um quadro e retomar (a completude é verificada):

```sh
head -c 1000 /tmp/qA/frame_000010.png > /tmp/qA/x && mv /tmp/qA/x /tmp/qA/frame_000010.png
sv render all /tmp/curto.json /tmp/buraco.zip --output /tmp/qA | head -1                        # 1 drawn, 63 kept
```

## 7. Recusas (História 7)

```sh
sv render frame /tmp/plano.json /tmp/recorte.zip --number 999999 --output /tmp/x.png; echo $?      # 33 (faixa válida)
sv render frame /tmp/plano.json /tmp/recorte.zip --number 3.5 --output /tmp/x.png; echo $?         # 33 (não é inteiro)
sv render all /tmp/plano.json /tmp/recorte.zip --output /tmp/x --resolution 1921x1080; echo $?     # 34 (ímpar)
sv render all /tmp/plano.json /tmp/recorte.zip --output /tmp/x --resolution 8000x4500; echo $?     # 34 (fora dos limites)
sv render all /tmp/plano.json /tmp/recorte.zip --output /tmp/x --resolution abc; echo $?           # 34
sv render all /tmp/nao-existe.json /tmp/recorte.zip --output /tmp/x; echo $?                       # 4
sv render all /tmp/plano.json /tmp/nao-existe.zip --output /tmp/x; echo $?                         # 4
echo '{"a":1}' > /tmp/x.json; sv render all /tmp/x.json /tmp/recorte.zip --output /tmp/x; echo $?   # 17
head -c 5000 /tmp/recorte.zip > /tmp/truncado.zip
sv render all /tmp/plano.json /tmp/truncado.zip --output /tmp/x; echo $?                           # 27 (truncado)
sv render all /tmp/plano.json /tmp/plano.json --output /tmp/x; echo $?                             # 27 (não é um recorte)
sv render all /tmp/curto.json /tmp/recorte.zip --output /tmp/x; echo $?                            # 29 (recorte de outro plano)
ls /tmp/x 2>&1                                                                                     # nenhum destes criou imagem nem diretório
```

Versão desconhecida do recorte (código 28):

```sh
rm -rf /tmp/z && mkdir /tmp/z && cd /tmp/z && unzip -q /tmp/recorte.zip
sed 's/"format_version": 1/"format_version": 2/' manifest.json > m && mv m manifest.json
zip -q -0 -X ../v2.zip manifest.json $(unzip -Z1 /tmp/recorte.zip | grep -v manifest.json) ; cd - >/dev/null
sv render all /tmp/plano.json /tmp/v2.zip --output /tmp/x; echo $?                                 # 28 (found 2, accepted: 1)
```

Peças vetoriais e sem elevação:

```sh
sv geodata remove mapa && sv geodata register $A/mapa-vetorial-sp.mbtiles --name mapa-pbf
sv geodata slice /tmp/plano.json --export /tmp/recorte-pbf.zip --overwrite
time sv render all /tmp/plano.json /tmp/recorte-pbf.zip --output /tmp/x; echo $?                   # 31, em < 5 s, nenhuma imagem
sv geodata remove mapa-pbf && sv geodata remove relevo && sv geodata register $A/relevo-sem-dado.tif --name relevo-vazio
sv geodata register $A/mapa-imagem-sp.mbtiles --name mapa
sv geodata slice /tmp/plano.json --export /tmp/recorte-vazio.zip --overwrite
sv render all /tmp/plano.json /tmp/recorte-vazio.zip --output /tmp/x; echo $?                      # 32
```

A recusa por cobertura (30) exige um recorte do mesmo plano gerado com outra
margem: só é exercitada pelos testes automatizados do domínio.

## 8. Resolução (História 4)

```sh
sv render frame /tmp/plano.json /tmp/recorte.zip --number 300 --resolution 640x360 --output /tmp/r1.png --overwrite
sv render frame /tmp/plano.json /tmp/recorte.zip --number 300 --resolution 1280x720 --output /tmp/r2.png --overwrite
sv render frame /tmp/plano.json /tmp/recorte.zip --number 300 --resolution 1080x1920 --output /tmp/r3.png --overwrite   # retrato
sv render frame /tmp/plano.json /tmp/recorte.zip --number 300 --resolution 3840x2160 --output /tmp/r4.png --overwrite
```

Esperado: cada arquivo com exatamente a resolução pedida; `r1` e `r2` com o
mesmo enquadramento vertical e o marcador na mesma posição relativa; o retrato
mostra menos dos lados; em resolução alta o mapa fica menos nítido (o recorte
foi dimensionado para 1080 pixels de altura), sem ser erro. O 4K leva cerca de 4 s por quadro
no M1 de 8 núcleos (medido no passeio real).

## 9. Antimeridiano e latitudes altas (SC-009)

O plano de `antimeridiano.gpx` tem 1140 quadros; desenhá-los todos leva minutos, e
basta olhar os quadros em volta da junção. O marcador cruza 180° entre os quadros
567 e 568:

```sh
rm -rf /tmp/qam && mkdir /tmp/qam
SVAM=$(mktemp -d); sva() { HOME=$SVAM ./bin/sobrevoo "$@"; }
sva geodata register $A/mapa-imagem-antimeridiano.mbtiles --name mapa-am
sva geodata register $A/relevo-antimeridiano.tif --name relevo-am
sva plan $G/antimeridiano.gpx --aspect 16:9 --export /tmp/am.json --overwrite | grep -E "Frames"          # Frames: 1140
sva geodata slice /tmp/am.json --export /tmp/am.zip --overwrite | head -2
for n in 560 567 568 575; do sva render frame /tmp/am.json /tmp/am.zip --number $n --resolution 960x540 --output /tmp/qam/f$n.png | tail -1; done
```

Esperado: `Holes: map tiles missing: no, elevation without value: no` nos quatro; os
quadros 567 e 568 (um de cada lado do meridiano) mostram o terreno, o xadrez e o
traçado **contínuos, sem emenda, salto nem espelhamento** — as duas imagens são
quase idênticas, e a faixa escura que os cruza é a borda de uma peça, que cai
sobre o meridiano. Repetir com o par polar (`mapa-imagem-polar`, `relevo-polar`,
`polar.gpx`): formas e traçado sem distorção pela latitude; qualquer terreno
além de 85,0511° aparece com a hachura de "sem imagem de mapa" (os testes do
domínio cobrem esse caso, pois os dados de exemplo ficam abaixo desse limite).

## 10. Dados reais (`resources/`)

O passeio (`passeio_bike_20260912.gpx`), o relevo Copernicus
(`dem-S14-W040.tif`) e o mapa vetorial do BBBike (`planet_…mbtiles`) estão em
`resources/` (não versionada). O mapa do BBBike é **vetorial (`pbf`)**:

```sh
R=$PWD/resources
sv geodata register "$R/dem-S14-W040.tif" --name dem-real
sv geodata register "$R/planet_-40.036,-13.661_-38.086,-12.56.mbtiles" --name bbbike
sv plan "$R/passeio_bike_20260912.gpx" --aspect 16:9 --export /tmp/real.json --overwrite
sv geodata slice /tmp/real.json --export /tmp/real.zip --overwrite
sv render frame /tmp/real.json /tmp/real.zip --number 300 --output /tmp/real.png; echo $?           # 31: peças vetoriais
```

Esperado: código `31`, com a mensagem que identifica `bbbike` e o formato `pbf`, e
nenhuma imagem. Para ver o **voo real**, um mapa em imagem **menor** que o do
BBBike (para vencer a regra "menor área") sobre o passeio: gerado sinteticamente
sobre a área do GPX:

```sh
go run ./test/samples --out $A --raster-over "$R/passeio_bike_20260912.gpx"      # grava mapa-imagem-passeio.mbtiles
sv geodata register $A/mapa-imagem-passeio.mbtiles --name mapa-passeio
sv geodata slice /tmp/real.json --export /tmp/real.zip --overwrite
sv render frame /tmp/real.json /tmp/real.zip --number 300 --output /tmp/real.png; echo $?          # 0
sv render all /tmp/real.json /tmp/real.zip --output /tmp/qreal --resolution 1920x1080
```

Esperado: o relevo é o **real** (Copernicus) e o traçado, o **passeio real**; o
mapa é o xadrez sintético. Conferir visualmente que o relevo bate com o que se
sabe do local, que o marcador percorre o trajeto e que o resumo conta os
buracos que o relevo e o mapa realmente têm.
Medido: nível de detalhe 15, 100 peças, sem buracos de mapa nem de
elevação; 1020 quadros em 1920×1080 levaram 17 min 15 s (parte com a máquina
disputada; cerca de 0,9 s por quadro sem disputa) e um quadro no padrão
1080×1920 leva cerca de 0,9 s. Para conferir o
valor do relevo sob o marcador de um quadro contra o GDAL, vale o roteiro da
etapa 4 (`geodata elevation --lat --lon`).
