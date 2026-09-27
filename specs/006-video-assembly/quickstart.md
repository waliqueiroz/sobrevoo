# Quickstart: validação manual da Montagem do Vídeo do Voo

**Feature**: `006-video-assembly` | **Data**: 2026-09-27

Checklist manual, com o binário real **e o `ffmpeg` real**, para conferir a
etapa de ponta a ponta (não há teste automatizado de ponta a ponta; ver
`plan.md`). Contratos: [`contracts/cli.md`](./contracts/cli.md),
[`contracts/video-file.md`](./contracts/video-file.md) e
[`contracts/frame-files-change.md`](./contracts/frame-files-change.md). Os
cenários usam **dados sintéticos** gerados em código (sem baixar nada), para que
o que se vê seja conhecido; o item 10 repete com os dados reais de
`resources/`. Os valores marcados **(anotar)** são medidos na primeira
execução e escritos aqui; os pontos **(confirmar)** de `research.md` (itens 5 e
6) se fecham aqui.

**Executado por inteiro** com `ffmpeg` 9.0.2 (`libx264` core 165 r3222,
Homebrew, macOS/arm64). Dois enganos da linha de comando original só
apareceram aqui, e foram corrigidos no código e em `research.md` item 6: (1)
`-x264-params info=0` não existe (a opção certa para tirar a versão e as
opções do `x264` de dentro do fluxo não existe em nenhuma versão testada do
`ffmpeg`/`x264` — a saída foi um filtro de bitstream, `-bsf:v
filter_units=remove_types=6`, depois de codificado) e `-movflags +bitexact`
não é mais uma opção válida do muxer MOV (substituída pelo `-bitexact`
global, que também é o que de fato tira a versão da etiqueta `encoder` do
contêiner — `-metadata:s:v:0 encoder=` tira a etiqueta em si); e (2) sem
`-xerror`, o `ffmpeg` **tolera**, por padrão, um quadro cujo `IDAT` não
decodifica e sai com sucesso, o que a spec não permite (achado no item 7,
"Falha do codificador"; corrigido para toda montagem).

## Pré-requisitos

Todos os comandos rodam a partir da **raiz do repositório** (zsh ou bash).

```sh
brew install ffmpeg                               # macOS; outras plataformas: contracts/cli.md, mensagem 46
ffmpeg -version | head -1; ffprobe -version | head -1
ffmpeg -hide_banner -encoders | grep libx264      # o codificador de que a ferramenta precisa

make build                                        # gera bin/sobrevoo
BIN=$PWD/bin/sobrevoo
SVHOME=$(mktemp -d)                               # o registro fica em $HOME/.sobrevoo/registry.json
sv() { HOME=$SVHOME $BIN "$@"; }                  # registro isolado para este roteiro
A=$PWD/specs/005-frame-rendering/amostras         # caminho absoluto: o registro guarda o caminho como dado
G=specs/003-camera-path-planning/amostras
go run ./test/samples --out $A
sv geodata register $A/mapa-imagem-sp.mbtiles --name mapa
sv geodata register $A/relevo-sp.tif --name relevo
```

O padrão do `plan` e do `render` é vídeo **vertical** (`9:16`, `1080x1920`).
Este roteiro usa um plano vertical **curto** (38 s a 10 quadros por segundo: 380
quadros; `pedalada.gpx` pede no mínimo 37 s em 9:16) desenhado em **360×640**,
para os quadros saírem em cerca de um minuto:

```sh
sv plan $G/pedalada.gpx --duration 38 --fps 10 --export /tmp/plano.json --overwrite     # Frames: 380
sv geodata slice /tmp/plano.json --export /tmp/recorte.zip --overwrite
rm -rf /tmp/qv && sv render all /tmp/plano.json /tmp/recorte.zip --output /tmp/qv --resolution 360x640
```

Ver a mídia: `open /tmp/voo.mp4` (macOS) ou qualquer reprodutor. Inspecionar:
`ffprobe` (ver `contracts/video-file.md`, "Como conferir").

## 1. Montar o vídeo (História 1 e 5)

```sh
rm -f /tmp/voo.mp4; sv video /tmp/plano.json /tmp/qv --output /tmp/voo.mp4; echo $?
ffprobe -v error -count_frames -select_streams v:0 \
  -show_entries stream=codec_name,profile,pix_fmt,width,height,r_frame_rate,nb_read_frames,color_space,color_range \
  -show_entries format=duration,format_name -of default=nw=1 /tmp/voo.mp4
ffprobe -v error -select_streams a -show_entries stream=index -of csv=p=0 /tmp/voo.mp4      # nenhuma linha: sem áudio
```

Esperado: código `0`; progresso em `stderr` (uma linha reescrita em terminal);
resumo em `stdout`: `Video written to /tmp/voo.mp4`, `Frames: 380`, `Duration:
00:00:38.000`, `Resolution: 360x640`, `Frame rate: 10 fps`, `Quality: medium`,
`Size: …`, `Encoder: ffmpeg … (libx264)`, `Time: …`. O `ffprobe` confirma:
`codec_name=h264`, `profile=High`, `pix_fmt=yuv420p`, `width=360`, `height=640`,
`r_frame_rate=10/1`, `nb_read_frames=380`, `color_space=bt709`, `color_range=tv`,
`duration=38.0…` (SC-001). **Medido**: `Size: 1.4 MiB`, `Encoder: ffmpeg 9.0.2
(libx264)`, `Time: 00:00:01`.

**Os quadros do vídeo são os da pasta** (SC-002): o primeiro, um do meio e o último
comparados com a imagem original — a diferença é só a perda da compressão:

```sh
ffmpeg -v error -i /tmp/voo.mp4 -framerate 10 -i /tmp/qv/frame_%06d.png \
  -lavfi "psnr=stats_file=/tmp/psnr.log" -f null -
grep -c "^n:" /tmp/psnr.log                                                              # 380 pares comparados
grep -E -o "psnr_avg:[0-9.]+" /tmp/psnr.log | cut -d: -f2 | sort -n | head -1             # o pior quadro
grep -E "^n:(1|190|380) " /tmp/psnr.log | sed -E 's/ mse_avg.*psnr_avg:([0-9.a-z]+).*/ psnr=\1/'
```

Esperado: o pior PSNR acima de **35 dB** no nível `medium` (o quadro `n` do vídeo
com a imagem `n` da pasta; um deslocamento de ordem derrubaria o valor para menos
de 20 dB). **Medido**: 380 pares comparados, pior PSNR **37,54 dB**. O vídeo abre
e mostra o traçado crescendo do começo ao fim, sem texto, sem estatísticas.

**Arquivos alheios** (História 1, cenário 6): a pasta com notas e uma miniatura
não muda o resultado nem é alterada:

```sh
echo nota > /tmp/qv/notas.txt; cp /tmp/qv/frame_000000.png /tmp/qv/capa.png
rm -f /tmp/voo2.mp4; sv video /tmp/plano.json /tmp/qv --output /tmp/voo2.mp4; echo $?
cmp /tmp/voo.mp4 /tmp/voo2.mp4 && echo IDENTICOS; ls /tmp/qv | grep -v frame_
rm /tmp/qv/notas.txt /tmp/qv/capa.png
```

Esperado: `0`, `IDENTICOS`; os dois arquivos alheios continuam lá, iguais.

**Progresso fora do terminal**: `sv video … 2>&1 | cat` imprime no máximo uma linha
a cada 5 s e a última (`Encoding frame 380/380 (100.0%), …`).

## 2. Taxa de quadros não inteira (História 1, cenário 3)

Um plano a 29,97 quadros por segundo; o menor tamanho de quadro basta:

```sh
sv plan $G/pedalada.gpx --duration 38 --fps 29.97 --export /tmp/p2997.json --overwrite | grep -E "Frames|Frame rate"
sv geodata slice /tmp/p2997.json --export /tmp/r2997.zip --overwrite >/dev/null
rm -rf /tmp/q2997 && sv render all /tmp/p2997.json /tmp/r2997.zip --output /tmp/q2997 --resolution 180x320 2>/dev/null | head -1
rm -f /tmp/v2997.mp4; sv video /tmp/p2997.json /tmp/q2997 --output /tmp/v2997.mp4 | sed -n 2,5p
ffprobe -v error -count_frames -select_streams v:0 -show_entries stream=r_frame_rate,nb_read_frames -show_entries format=duration -of default=nw=1 /tmp/v2997.mp4
```

Esperado: `Frames: 1139`, `Duration: 00:00:38.0…` (`1139 ÷ 29.97`), `Frame rate: 29.97
fps`; o `ffprobe` diz `r_frame_rate=2997/100`, `nb_read_frames=1139` e a duração de
`38.00…` (no máximo um quadro de diferença, SC-001).

## 3. Qualidade (História 4)

```sh
for q in low medium high; do
  rm -f /tmp/q-$q.mp4; /usr/bin/time -p $BIN video /tmp/plano.json /tmp/qv --output /tmp/q-$q.mp4 --quality $q 2>&1 >/dev/null | grep real
  ls -l /tmp/q-$q.mp4 | awk '{print $5, $9}'
done
sv video /tmp/plano.json /tmp/qv --output /tmp/x.mp4 --quality ultra; echo $?                  # 2, com os níveis aceitos
```

(O `video` não lê o registro, então o `HOME` isolado não faz diferença.) Esperado: os
tamanhos crescem de `low` para `high` (`low ≤ medium ≤ high`, SC-009) e a
duração, a resolução, a taxa e a quantidade de quadros são iguais nos três
(`ffprobe`). **Medido**: `low` 930 582 B / 0,67 s; `medium` 1 442 368 B / 1,13 s;
`high` 2 357 240 B / 2,08 s. O nível inexistente sai com `2` e
`--quality "ultra": use one of low, medium, high`, sem arquivo criado. `--output
/tmp/x.mkv`: `2`, `--output must end in .mp4`.

## 4. Sempre o mesmo vídeo, sem dado do ambiente (História 7)

```sh
mkdir -p /tmp/outro && rm -f /tmp/A.mp4 /tmp/outro/B.mp4
sv video /tmp/plano.json /tmp/qv --output /tmp/A.mp4 >/dev/null
sleep 3                                                                   # o relógio muda
(cd /tmp/outro && HOME=$SVHOME $BIN video /tmp/plano.json /tmp/qv --output B.mp4 >/dev/null)
cmp /tmp/A.mp4 /tmp/outro/B.mp4 && echo IDENTICOS
ffprobe -v error -show_format -show_streams /tmp/A.mp4 | grep -i -E "creation|encoder|tag:|/tmp|/Users|$(hostname -s)|$USER"
strings -a /tmp/A.mp4 | grep -i -E "x264|Lavf|Lavc|/tmp|/Users|$(hostname -s)|$USER"
```

Esperado (**confirmado**, `research.md` item 6): `IDENTICOS`; o `grep` do
`ffprobe` mostra só as etiquetas constantes (`handler_name`, `major_brand`,
`minor_version`, `compatible_brands`; sem data, sem `encoder`, sem caminho,
sem nome de máquina ou de usuário), e o `strings` não acha `x264`,
`Lavf`/`Lavc`, caminho nem nome. Repetido 5 vezes em sequência: os 5 arquivos
têm o mesmo hash (`md5`).

Na primeira execução, **antes** da correção do item 6 (`-bitexact`
global e `-metadata:s:v:0 encoder=`), o `grep` do `ffprobe` mostrava
`TAG:encoder=Lavc libx264` e o `strings` achava a mensagem SEI completa do
`x264` (`x264 - core 165 r3222 … - options: cabac=1 ref=3 …`) — nada disso
tinha data, caminho ou nome de máquina, mas violava a promessa deste
contrato; corrigido, e refeito com o mesmo resultado limpo descrito acima.

## 5. Conferência dos quadros (História 2)

Cada variação é uma cópia da pasta; o comando recusa **antes de codificar** e nenhum
`.mp4` é criado:

```sh
cp -R /tmp/qv /tmp/q5
rm /tmp/q5/frame_00001[2-5].png;      sv video /tmp/plano.json /tmp/q5 --output /tmp/n.mp4; echo $?   # 41: 4 missing (12-15)
cp /tmp/qv/frame_000000.png /tmp/q5/frame_000012.png; cp /tmp/qv/frame_000001.png /tmp/q5/frame_000380.png
                                       sv video /tmp/plano.json /tmp/q5 --output /tmp/n.mp4; echo $?   # 41: 3 missing (13-15), 1 not in the plan (380)
ls /tmp/n.mp4 2>&1 | head -1                                                                          # No such file
```

Na segunda chamada, a cópia do quadro 0 sobre `frame_000012.png` é um quadro
**nosso**, do mesmo conjunto: preenche o 12 (o número, não o conteúdo, é o que a
conferência olha), e a mensagem passa a dizer `13-15` faltando e o excedente
`380`. Depois, as demais recusas:

```sh
sv video /tmp/plano.json /tmp/nao-existe --output /tmp/n.mp4; echo $?                                 # 40: does not exist
mkdir -p /tmp/q5v && touch /tmp/q5v/frame_000000.png
sv video /tmp/plano.json /tmp/q5v --output /tmp/n.mp4; echo $?                                        # 40: holds no frames (1 file ignored)

# outro plano: um quadro de outro voo na pasta
sv plan $G/pedalada.gpx --duration 40 --fps 10 --export /tmp/outroplano.json --overwrite >/dev/null
sv geodata slice /tmp/outroplano.json --export /tmp/outro.zip --overwrite >/dev/null
mkdir -p /tmp/qo && sv render frame /tmp/outroplano.json /tmp/outro.zip --number 0 --output /tmp/qo/f0.png --resolution 360x640
rm -rf /tmp/q5 && cp -R /tmp/qv /tmp/q5 && cp /tmp/qo/f0.png /tmp/q5/frame_000200.png
sv video /tmp/plano.json /tmp/q5 --output /tmp/n.mp4; echo $?                                         # 43: 1 of 380 carries another plan identification
sv video /tmp/outroplano.json /tmp/qv --output /tmp/n.mp4; echo $?                                    # 43: 380 of 380 (o plano informado é o outro)

# resolução: uma sobrescrita interrompida em outra resolução deixa quadros dos dois
rm -rf /tmp/q5 && cp -R /tmp/qv /tmp/q5
HOME=$SVHOME $BIN render all /tmp/plano.json /tmp/recorte.zip --output /tmp/q5 --resolution 180x320 --overwrite >/dev/null 2>&1 & PID=$!
sleep 4; kill -INT $PID; wait $PID
sv video /tmp/plano.json /tmp/q5 --output /tmp/n.mp4; echo $?                                         # 42: 360x640 is the resolution of N frames, the others are 180x320

# integridade: quadro truncado (a marca está no início, o fim não)
rm -rf /tmp/q5 && cp -R /tmp/qv /tmp/q5 && head -c 5000 /tmp/qv/frame_000007.png > /tmp/q5/frame_000007.png
sv video /tmp/plano.json /tmp/q5 --output /tmp/n.mp4; echo $?                                         # 45: frame_000007.png

# sem a identificação do plano (quadros de antes da versão 2): remove o bloco plan= de uma cópia
python3 - <<'EOF'
import struct, shutil, os
shutil.rmtree('/tmp/q5', ignore_errors=True); shutil.copytree('/tmp/qv', '/tmp/q5')
for name in sorted(os.listdir('/tmp/q5')):
    if not name.startswith('frame_'): continue
    data = open('/tmp/q5/'+name,'rb').read(); out = data[:8]; pos = 8
    while pos < len(data):
        n = struct.unpack('>I', data[pos:pos+4])[0]; chunk = data[pos:pos+12+n]
        if not (chunk[4:8] == b'tEXt' and chunk[8:].startswith(b'Sobrevoo\x00plan=')): out += chunk
        pos += 12 + n
    open('/tmp/q5/'+name,'wb').write(out)
EOF
sv video /tmp/plano.json /tmp/q5 --output /tmp/n.mp4; echo $?                                         # 44: 380 of 380
```

Esperado em cada um: o código do comentário, a mensagem de `contracts/cli.md` dizendo
exatamente o problema (números em faixas, arquivos, resoluções, identificações
abreviadas), e **nenhum** `/tmp/n.mp4` criado. A conferência de 380 quadros leva
menos de 1 s (a de 1140 quadros, recusada por conjunto: **0,40 s medidos**, item 9;
bem abaixo do limite de 2 s de SC-005). Um quadro de outra
ferramenta com nome de quadro (`cp foto.png /tmp/q5/frame_000005.png`) não é
reconhecido: a mensagem de falta diz que foi **ignorado** por não trazer a
identificação da ferramenta.

## 6. Codificador ausente (História 3)

```sh
PATH=/usr/bin:/bin $BIN video /tmp/plano.json /tmp/qv --output /tmp/n.mp4; echo $?                     # 46
ls /tmp/n.mp4 2>&1 | head -1                                                                          # No such file
```

Esperado: código `46`, mensagem `video encoder not available: "ffmpeg" was not found
on the PATH; install it (macOS: brew install ffmpeg; …), then check it with:
ffmpeg -version`; nenhum arquivo criado. A ordem (`contracts/cli.md`): com quadros
também errados **e** sem `ffmpeg`, o erro dos quadros vem primeiro:

```sh
PATH=/usr/bin:/bin $BIN video /tmp/plano.json /tmp/nao-existe --output /tmp/n.mp4; echo $?             # 40, não 46
```

O caso "`ffmpeg` sem `libx264`" é coberto pelos testes do adapter (não há como
simulá-lo com o `ffmpeg` completo instalado).

## 7. Proteção do destino e interrupção (História 6)

```sh
sv video /tmp/plano.json /tmp/qv --output /tmp/voo.mp4; echo $?                                       # 47: use --overwrite
md5 -q /tmp/voo.mp4                                                                                   # (anotar) antes
sv video /tmp/plano.json /tmp/qv --output /tmp/voo.mp4 --quality high --overwrite >/dev/null; echo $?  # 0
sv video /tmp/plano.json /tmp/qv --output /tmp/nada/voo.mp4; echo $?                                  # 48: the folder does not exist
mkdir -p /tmp/dir.mp4 && sv video /tmp/plano.json /tmp/qv --output /tmp/dir.mp4; echo $?              # 48: is a directory
```

Esperado (**confirmado**): `47` (`video destination already exists: …; use
--overwrite to replace it`), o `md5` do arquivo depois igual ao de antes; `48`
para pasta inexistente (`the folder does not exist`) e para um diretório
chamado `dir.mp4` (`is a directory`); com `--overwrite`, o arquivo é
substituído por inteiro. **Interrupção** — o sinal precisa chegar ao
**binário**, não a uma função de shell:

```sh
rm -f /tmp/int.mp4
HOME=$SVHOME $BIN video /tmp/plano.json /tmp/qv --output /tmp/int.mp4 --quality high > /tmp/run.out 2>/tmp/run.err & PID=$!
sleep 0.3; kill -INT $PID; wait $PID; echo "codigo: $?"                                               # 49
cat /tmp/run.out; ls -a /tmp | grep -E "^int.mp4|sobrevoo.*tmp"; pgrep -x ffmpeg                       # nada: sem arquivo, sem temporário, sem ffmpeg órfão
```

Esperado (**confirmado**, `380` quadros em `--quality high`, interrompido a
0,3 s): `Interrupted: 0 of 380 frames were encoded; no video was written, run
the same command again to start over`, código `49`, **nenhum** `int.mp4`,
nenhum temporário `.sobrevoo-*.tmp` e nenhum processo `ffmpeg` sobrando.
Repetido com um conjunto maior (1140 quadros, `--quality high`, interrompido a
1,5 s): `Interrupted: 25 of 1140 frames were encoded; …`, mesmo resultado —
código `49`, sem arquivo, sem processo. Repetir o pedido sem interromper gera
o mesmo vídeo que uma execução ininterrupta (`cmp` com um vídeo `high` de
antes).

**Falha do codificador** — um quadro com o meio corrompido passa pela
conferência (que só olha o cabeçalho e o fim, `research.md` item 7):

```sh
rm -rf /tmp/q5 && cp -R /tmp/qv /tmp/q5
python3 - <<'EOF'
import os
p='/tmp/q5/frame_000100.png'; d=bytearray(open(p,'rb').read()); mid=len(d)//2
d[mid:mid+400]=os.urandom(400); open(p,'wb').write(d)                     # corrompe 400 bytes do meio (dados aleatórios)
EOF
rm -f /tmp/f.mp4; sv video /tmp/plano.json /tmp/q5 --output /tmp/f.mp4; echo $?                       # 50
ls -a /tmp | grep -E "^f.mp4|sobrevoo.*tmp"                                                           # nada
```

Esperado: `video encoding failed: ffmpeg exited with status …: <causa>` (a causa
vem do `ffmpeg`, por exemplo `inflate returned error -3`, `Error submitting
packet to decoder`); código `50`; nenhum arquivo e nenhum temporário.

**Achado nesta validação** (registrado em `research.md` item 6, corrigido no
código): **zerar** bytes do meio do `IDAT` (em vez de substituir por dados
aleatórios) **não** quebra a decodificação do PNG — o `ffmpeg` decodifica algo
(um quadro visualmente errado, mas um `IDAT` válido do ponto de vista do
`deflate`) e segue em frente; é preciso corromper com bytes que quebrem o
fluxo comprimido (`os.urandom`, como acima) para reproduzir uma falha de
decodificação de verdade. E, mais importante: **mesmo com o `IDAT`
genuinamente quebrado, o `ffmpeg` por padrão não falhava** — ele tolera um
erro de decodificação isolado numa sequência de imagens e termina com sucesso
(código `0`), o quadro corrompido silenciosamente descartado ou reaproveitado.
Foi preciso acrescentar `-xerror` à linha de comando (`research.md` item 6)
para qualquer erro de decodificação abortar a codificação — sem ele, o
cenário acima teria terminado com `exit=0` e um `/tmp/f.mp4` gravado, faltando
à garantia da spec.

## 8. Voo que cruza 180° e voo polar (SC-011)

As mesmas propriedades, sem tratamento de região. Os planos de
`antimeridiano.gpx` e `polar.gpx` são montados em vídeo curto (a resolução mínima,
poucos quadros por segundo) e comparados com o do item 1:

```sh
SVAM=$(mktemp -d); sva() { HOME=$SVAM $BIN "$@"; }
sva geodata register $A/mapa-imagem-antimeridiano.mbtiles --name mapa-am
sva geodata register $A/relevo-antimeridiano.tif --name relevo-am
sva plan $G/antimeridiano.gpx --fps 5 --export /tmp/am.json --overwrite | grep -E "Frames"
sva geodata slice /tmp/am.json --export /tmp/am.zip --overwrite >/dev/null
rm -rf /tmp/qam && sva render all /tmp/am.json /tmp/am.zip --output /tmp/qam --resolution 180x320 >/dev/null 2>&1
rm -f /tmp/am.mp4; sva video /tmp/am.json /tmp/qam --output /tmp/am.mp4 | sed -n 2,6p
ffprobe -v error -count_frames -select_streams v:0 -show_entries stream=r_frame_rate,nb_read_frames -show_entries format=duration -of default=nw=1 /tmp/am.mp4
```

Repetir com o par polar (`mapa-imagem-polar`, `relevo-polar`, `polar.gpx`).
Esperado: em cada um, `nb_read_frames` igual a `Frames` do plano, `r_frame_rate=5/1`,
duração `Frames ÷ 5`, mesmas mensagens e mesmos códigos de recusa que no item 5.

**Medido**: antimeridiano — `Frames: 209`, `r_frame_rate=5/1`,
`nb_read_frames=209`, `duration=41.800000` (209 ÷ 5); polar — `Frames: 196`,
`r_frame_rate=5/1`, `nb_read_frames=196`, `duration=39.200000` (196 ÷ 5). Os
dois batem exatamente.

## 9. Desempenho (referência de SC-005 e SC-007)

**Medido**, no mesmo Apple M1 de 8 núcleos das etapas anteriores:

- **Conferência de 380 quadros, com sucesso, mais a codificação** (item 1):
  1,26 s no total (`time` da montagem completa, nível `medium`).
- **Conferência de 1140 quadros, recusada por conjunto** (um quadro de outro
  plano misturado nos quadros reais de `/tmp/qbig`): 0,40 s — bem abaixo do
  limite de 2 s de SC-005, e a recusa acontece antes de qualquer codificação.
- **Vídeo de 1140 quadros reais em 720×1280** (a resolução real do teste, em
  vez dos 180×320 previstos — os quadros de 180×320 do item 8 são poucos
  demais para uma medida de desempenho útil), nos três níveis: `low` 2,9 MB /
  5,04 s; `medium` 4,4 MB / 7,57 s; `high` 6,5 MB / 11,24 s.

A meta de SC-007 (1350 quadros de 1080×1920 em `medium`, até 10 min) é medida no
item 10, com o voo real.

## 10. Dados reais (`resources/`)

O passeio (`passeio_bike_20260912.gpx`), o relevo Copernicus (`dem-S14-W040.tif`
e `dem-S14-W041.tif`) e o mapa em imagem sintético sobre a área do passeio
(`mapa-imagem-passeio.mbtiles`, do `--raster-over`) são os do item 10 da etapa
5. Se os 1020 quadros que aquele item deixou em `/tmp/qrealv` ainda existirem,
são **da versão 1** (sem a identificação do plano): `video` os recusa com `44`, e
é isso que se verifica primeiro; depois, desenhar de novo e montar:

```sh
R=$PWD/resources
SVR=$(mktemp -d); svr() { HOME=$SVR $BIN "$@"; }                                         # registro só com os dados reais
svr geodata register "$R/dem-S14-W040.tif" --name dem-40
svr geodata register "$R/dem-S14-W041.tif" --name dem-41
go run ./test/samples --out $A --raster-over "$R/passeio_bike_20260912.gpx"              # grava mapa-imagem-passeio.mbtiles
svr geodata register $A/mapa-imagem-passeio.mbtiles --name mapa-passeio
svr plan "$R/passeio_bike_20260912.gpx" --export /tmp/realv.json --overwrite             # vertical (padrão)
svr geodata slice /tmp/realv.json --export /tmp/realv.zip --overwrite
ls /tmp/qrealv >/dev/null 2>&1 && svr video /tmp/realv.json /tmp/qrealv --output /tmp/realv.mp4; echo $?   # 44, se os quadros antigos ainda existirem
svr render all /tmp/realv.json /tmp/realv.zip --output /tmp/qrealv --overwrite           # ~16 min: quadros da versão 2
rm -f /tmp/realv.mp4; svr video /tmp/realv.json /tmp/qrealv --output /tmp/realv.mp4
ffprobe -v error -count_frames -select_streams v:0 -show_entries stream=width,height,r_frame_rate,nb_read_frames -show_entries format=duration -of default=nw=1 /tmp/realv.mp4
open /tmp/realv.mp4
```

Esperado: `Frames: 1020`, `Resolution: 1080x1920`, `Frame rate: 30 fps`, `Duration:
00:00:34.000`; o `ffprobe` confere; o vídeo mostra o relevo real (Copernicus), o
traçado do passeio real e o marcador percorrendo o trajeto, na vertical.

**Medido**: os 1020 quadros da versão 1 (sem a identificação do plano, deixados
pela validação da etapa 5) foram recusados com `44`, como esperado; o
`render all --overwrite` levou **16 min 05 s** (0 buracos); a montagem em
`medium` levou **14 s** (6,7 MiB) — bem abaixo dos 10 min de SC-007 (que pede
1350 quadros de 1080×1920; este voo tem 1020, mas na mesma resolução e no
mesmo computador, o resultado é consistente com o limite); em `high`, **20 s**
(9,6 MiB), maior e mais lento que `medium`, como esperado. Sem áudio
(`ffprobe -select_streams a`, nenhuma linha). PSNR: 1020 pares comparados,
pior valor **40,07 dB**. Visualmente: um quadro do meio e um próximo do fim,
extraídos com `ffmpeg -vf select=eq(n\,N)`, mostram o mapa sintético em xadrez
sobre o relevo real, o traçado laranja crescendo (mais longo e mais sinuoso no
quadro tardio) e o marcador vermelho no fim do traçado — igual ao que a etapa 5
já validou nos PNG.

**SC-008**: o vídeo abriu sem erro no **QuickTime Player** e no **Safari**
(player nativo do navegador), os dois disponíveis nesta máquina; o **VLC não
está instalado** aqui, então não foi testado (não é um requisito de instalação
da ferramenta — é só um dos reprodutores de referência do critério). O aceite
por duas plataformas de publicação (subir o arquivo) é manual e do usuário, fora
do alcance desta validação.
