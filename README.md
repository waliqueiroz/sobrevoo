# Sobrevoo

Sobrevoo é uma ferramenta de linha de comando, em Go, que transforma um
trajeto GPS (GPX) num vídeo de sobrevoo, no estilo Relive/Strava: uma câmera
virtual acompanha o trajeto por cima do relevo real do terreno, vestido com o
mapa base escolhido por você, enquanto um marcador percorre o caminho.

Não há nenhum serviço externo envolvido: o mapa base e o relevo são arquivos
que você mesmo baixa (MBTiles e GeoTIFF) e registra localmente uma vez; a
partir daí, todo o processamento — leitura do trajeto, planejamento de
câmera, recorte dos dados, desenho dos quadros e codificação do vídeo — roda
offline, no seu computador, e produz sempre o mesmo resultado, byte a byte,
para as mesmas entradas. A única dependência externa é o
[`ffmpeg`](https://ffmpeg.org), usado como processo externo para codificar o
vídeo final.

Ainda não há sobreposição de texto ou de estatísticas no vídeo, nem áudio.

## Instalação

Requer Go 1.26+. Para gerar o vídeo (comandos `video` e `fly`), requer também
o `ffmpeg` com o codificador `libx264` instalado e no `PATH`
(`brew install ffmpeg` no macOS, `sudo apt install ffmpeg` no Debian e no
Ubuntu, `winget install Gyan.FFmpeg` no Windows); o Sobrevoo não o traz nem o
baixa, e diz o que instalar se ele faltar.

```sh
go install github.com/waliqueiroz/sobrevoo/cmd/sobrevoo@latest
```

Ou, a partir do código-fonte:

```sh
git clone git@github.com:waliqueiroz/sobrevoo.git
cd sobrevoo
make build   # gera ./bin/sobrevoo
```

Todo comando aceita `--help` para a lista de flags.

## Uso

### Visão geral: `fly`

O jeito mais direto de usar o Sobrevoo é o comando `fly`: informe o trajeto e
o destino do vídeo, e ele faz o percurso inteiro sozinho — tratamento do
trajeto, planejamento de câmera, recorte dos dados geográficos já
registrados, desenho dos quadros e montagem do vídeo —, usando os mapas e o
relevo que você já registrou com `geodata register` (veja mais abaixo).

```sh
sobrevoo fly <trajeto.gpx> --output <voo.mp4>
             [--duration <segundos>] [--fps <n>] [--distance low|medium|high] [--tilt low|medium|high]
             [--aspect <L:A>] [--resolution <LxA>] [--quality low|medium|high]
             [--keep <diretório>] [--overwrite]
```

```console
$ sobrevoo fly pedalada.gpx --output pedalada.mp4
Stage 1/5: treating the track
Stage 2/5: planning the camera
Stage 3/5: slicing the geo data
Stage 4/5: drawing the frames
Stage 5/5: encoding the video
Frames: 1260 requested, 1260 drawn, 0 kept (already in the destination)
Resolution: 1080x1920
Time: 00:31:07
Holes (in the frames drawn now): none
Destination: /tmp/sobrevoo-fly-3f9a2c/frames (frame_000000.png to frame_001259.png)
Video written to pedalada.mp4
Frames: 1260
Duration: 00:00:42.000
Resolution: 1080x1920
Frame rate: 30 fps
Quality: medium
Size: 18.4 MiB
Encoder: ffmpeg 7.1 (libx264)
Time: 00:01:52
Total time: 00:33:00
```

| Flag | Valores | Padrão | Descrição |
|---|---|---|---|
| `--output` | caminho terminado em `.mp4` | — | Destino do vídeo (obrigatória) |
| `--duration` | segundos (até 3600) | automática | Duração do vídeo; sem ela, calculada a partir do comprimento do trajeto |
| `--fps` | 1 a 120 | `30` | Quadros por segundo |
| `--distance` | `low`, `medium`, `high` | `medium` | Quão longe a câmera fica do trajeto |
| `--tilt` | `low`, `medium`, `high` | `medium` | Quão de cima a câmera olha (`high` é quase vertical) |
| `--aspect` | `L:A` (`1:5` a `5:1`) | `9:16` | Proporção do vídeo (a abertura e o fechamento enquadram o trajeto inteiro por ela) |
| `--resolution` | `LxA`, ambos pares, 180 a 3840 por lado, no máximo 3.840×2.160 no total | `1080x1920` | Resolução dos quadros e do vídeo |
| `--quality` | `low`, `medium`, `high` | `medium` | Qualidade da codificação: `low` (rápido e pequeno, para conferir), `medium` (para publicar), `high` (para guardar) |
| `--keep` | diretório | — | Guarda `plan.json`, `slice.zip` e `frames/` nesse diretório em vez de num temporário, e reaproveita o que ainda vale numa execução seguinte |
| `--overwrite` | — | — | Substitui o vídeo de destino, e qualquer intermediário desatualizado sob `--keep`, se já existirem |

Todas essas flags são exatamente as de `plan`, `render all` e `video`
(mesmos nomes, valores aceitos, padrões e mensagens de erro) — o resultado é
idêntico, byte a byte, ao de rodar os seis comandos individuais na mão com
os mesmos valores.

- **Recusa cedo**: a disponibilidade do `ffmpeg` e o destino do vídeo (com a
  mesma regra do `--output` de `video`, inclusive a extensão `.mp4`) são
  conferidos antes de tratar o trajeto; a cobertura dos dados registrados,
  logo após planejar a câmera — bem antes de gastar tempo desenhando
  quadros.
- **Sem `--keep`**: nada fica para trás. O plano e o recorte nunca tocam
  disco, e os quadros vivem num diretório temporário, sempre apagado ao
  final — inclusive em caso de falha ou interrupção.
- **Com `--keep <diretório>`**: o plano, o recorte e os quadros ficam nesse
  diretório, nos mesmos formatos que `plan --export`/`geodata slice
  --export`/`render all --output` já produzem — abríveis com as ferramentas
  de sempre. Uma execução seguinte, com o mesmo trajeto e os mesmos valores,
  reaproveita o que ainda vale em vez de refazer. Atenção: repetir o mesmo
  `--output` numa segunda execução exige `--overwrite` para o vídeo — e
  `--overwrite` também refaz os quadros do zero, mesmo que o conjunto já
  bata (é a mesma regra de `render all --overwrite`, sem exceção para
  `fly`); para aproveitar de fato o plano/recorte/quadros já prontos, use um
  `--output` que ainda não existe.
- **Interrupção** (`Ctrl+C`): encerra de forma ordenada assim que a etapa em
  curso permitir, diz o que já havia sido concluído e sai com código próprio
  (`51`) — nunca o de uma etapa; nenhum vídeo parcial fica.
- **Erros**: cada etapa falha com exatamente o mesmo erro, a mesma mensagem
  e o mesmo código de saída que o comando individual dessa etapa já usa,
  documentados em `specs/007-full-flight-pipeline/contracts/cli.md`.

Por baixo do capô, `fly` encadeia seis comandos independentes — cada um
também disponível sozinho, para inspecionar um resultado intermediário,
reaproveitar um recorte já pronto, ou rodar só uma parte do processo. As
seções abaixo documentam cada um por completo, na ordem em que `fly` os
executa.

### `inspect`: tratar e resumir um trajeto

```sh
sobrevoo inspect <trajeto.gpx> [--simplification=low|medium|high] [--smoothing=low|medium|high]
```

Lê o arquivo GPX, descarta pontos inválidos (coordenadas impossíveis,
duplicatas consecutivas, saltos fisicamente implausíveis), reordena por tempo
quando necessário, reduz e suaviza o traçado, e imprime um resumo. Funciona
100% offline, para qualquer trajeto do planeta, inclusive os que cruzam o
meridiano de mudança de data.

```console
$ sobrevoo inspect atividade.gpx
Format: GPX
Points: 3 -> 2 (original -> treated)
Distance: 0.28 km
Elevation gain: 10.0 m
Duration: 1m0s
Bounding box: lat [40.000000, 40.002000], lon [-3.002000, -3.000000]
Discarded points: 0 (impossible coordinates: 0, consecutive duplicates: 0, implausible jumps: 0)
```

| Flag | Valores | Padrão | Descrição |
|---|---|---|---|
| `--simplification` | `low`, `medium`, `high` | `medium` | Nível de redução de pontos do traçado |
| `--smoothing` | `low`, `medium`, `high` | `medium` | Nível de suavização do traçado |

Códigos de saída: `1` arquivo vazio, `2` formato não reconhecido (ou uso
inválido da linha de comando), `3` pontos insuficientes; documentados em
`specs/001-gps-track-processing/contracts/cli.md`.

### `geodata`: registrar e verificar dados geográficos

O Sobrevoo nunca embute dados de mapa nem de relevo: você baixa os arquivos
e os registra, e ele descobre sozinho o tipo e a área coberta pelo conteúdo
de cada um. Formatos reconhecidos: **MBTiles** (mapa base, em imagem — PNG,
JPG ou WebP; peças vetoriais são recusadas na hora de desenhar, não no
registro) e **GeoTIFF em CRS geográfico, WGS84** (relevo). O registro fica em
`~/.sobrevoo/registry.json` e funciona 100% offline.

```sh
sobrevoo geodata register <arquivo> --name <nome>   # registra um mapa base ou relevo
sobrevoo geodata list                               # lista os registros
sobrevoo geodata remove <nome>                      # remove o registro (nunca o arquivo)
sobrevoo geodata check <trajeto.gpx>                # o trajeto está coberto?
```

```console
$ sobrevoo geodata register mapa-regiao.mbtiles --name europa-mapa
Registered "europa-mapa" as base map, covering lat [40.000000, 50.000000], lon [10.000000, 20.000000]
$ sobrevoo geodata register relevo-regiao.tif --name europa-relevo
Registered "europa-relevo" as elevation, covering lat [50.000000, 51.000000], lon [10.000000, 11.000000]
$ sobrevoo geodata list
europa-mapa (base map): lat [40.000000, 50.000000], lon [10.000000, 20.000000]
europa-relevo (elevation): lat [50.000000, 51.000000], lon [10.000000, 11.000000]
$ sobrevoo geodata check atividade.gpx
Coverage: partial
Uncovered segments:
  - lat [50.416800, 50.417500], lon [10.703800, 10.702000]: missing base map
Base map sources used: (none)
Elevation sources used: europa-relevo
```

| Flag | Valores | Padrão | Descrição |
|---|---|---|---|
| `register --name` | texto | — | Nome do registro (obrigatória, precisa ser única) |

`register` não tem outras flags: o tipo (mapa base ou relevo) e a área
coberta são descobertos a partir do conteúdo do arquivo, nunca informados
pelo usuário. `list` e `remove` também não têm flags.

Um trajeto só conta como coberto quando há mapa base **e** relevo em toda a
sua extensão; `check` reporta o veredito (`full`, `partial` ou `none`) e os
subtrechos que faltam, com coordenadas, e sempre termina com código `0`
quando consegue processar o trajeto — mesmo quando ele não está coberto (os
erros de leitura do próprio arquivo de trajeto, se ele for inválido, usam os
mesmos códigos de `inspect`: `1` a `3`). Se o arquivo de um registro for
movido ou apagado, `list` o marca com `(file not found)` em vez de
escondê-lo, e `check` deixa de contá-lo.

Códigos de saída de `register`/`remove`: `5` arquivo não encontrado, `6`
ilegível, `7` formato não suportado, `8` nome já em uso, `9` nome não
registrado; documentados em `specs/002-geo-data-registry/contracts/cli.md`.

### `plan`: planejar o movimento de câmera

```sh
sobrevoo plan <trajeto.gpx> [--duration <segundos>] [--fps <n>] \
    [--distance low|medium|high] [--tilt low|medium|high] \
    [--aspect <L:A>] [--export <plano.json>] [--overwrite]
```

A partir de um trajeto (tratado como em `inspect`), calcula o caminho que uma
câmera virtual percorre ao acompanhá-lo do início ao fim: para cada quadro do
futuro vídeo, onde a câmera está (latitude, longitude e altitude), para onde
aponta (direção e inclinação) e em que ponto do trajeto está o marcador da
atividade. O vídeo abre mostrando o trajeto inteiro, passa a acompanhar o
marcador e fecha mostrando o trajeto completo. O movimento é sempre suave,
inclusive em curvas fechadas, retornos e voltas no mesmo lugar; paradas
longas são comprimidas; e o mesmo trajeto com os mesmos parâmetros produz
sempre o mesmo plano, em qualquer lugar do planeta.

```console
$ sobrevoo plan atividade.gpx --export plano.json
Duration: 42.0 s (automatic)
Frame rate: 30.0 fps
Aspect ratio: 9:16
Frames: 1260
Camera altitude: 2098.6 m - 11711.6 m
Camera distance: 2780.9 m - 15982.9 m
Time reference: clock
Smoothed spans: none
Plan written to plano.json
```

| Flag | Valores | Padrão | Descrição |
|---|---|---|---|
| `--duration` | segundos (`> 0`, até 3600) | automática | Duração do vídeo. Sem ela, é calculada a partir do comprimento do trajeto (de 20 s a 120 s) |
| `--fps` | 1 a 120 | `30` | Quadros por segundo |
| `--distance` | `low`, `medium`, `high` | `medium` | Quão longe a câmera fica do trajeto |
| `--tilt` | `low`, `medium`, `high` | `medium` | Quão de cima a câmera olha (`high` é quase vertical) |
| `--aspect` | `L:A`, inteiros de 1 a 1000, razão de `1:5` a `5:1` | `9:16` | Proporção do vídeo: a abertura e o fechamento enquadram o trajeto inteiro por ela (`9:16` vertical, `16:9` horizontal) |
| `--export` | caminho | — | Grava o plano completo em JSON (formato em `specs/003-camera-path-planning/contracts/plan-file.md`) |
| `--overwrite` | — | — | Com `--export`, substitui um arquivo que já exista (recusa com uso inválido se usada sem `--export`) |

Sem `--export`, só o resumo é impresso e nada é gravado em disco. O arquivo
exportado nunca sobrescreve outro sem `--overwrite`.

Códigos de saída: `10` duração inválida, `11` taxa de quadros inválida, `12`
duração curta demais para o trajeto, `13` trajeto curto demais, `14` trajeto
grande demais, `15` destino da exportação já existe, `16` destino da
exportação inválido, `39` proporção (`--aspect`) inválida; documentados em
`specs/003-camera-path-planning/contracts/cli.md`.

### `geodata slice` e `geodata elevation`: ler o conteúdo dos dados registrados

```sh
sobrevoo geodata slice <plano.json> [--export <recorte.zip>] [--overwrite]
sobrevoo geodata elevation --lat <graus> --lon <graus>
```

`slice` parte de um plano exportado por `plan --export` e reúne, dos
arquivos que você registrou com `geodata register`, o que aquele voo
precisa: as amostras de elevação do terreno e as peças do mapa base sob a
área que a câmera percorre, no nível de detalhe adequado à distância em que
ela voa (o resumo diz qual nível foi escolhido e por quê). Recusa, dizendo o
que falta, se a área do plano não estiver totalmente coberta por mapa base e
relevo, e recusa um recorte maior que 256 MiB. Peças que faltam num mapa
registrado são listadas e não interrompem o recorte; amostras que o arquivo
de relevo não informa ficam marcadas como sem valor, nunca como zero. Tudo é
lido só dos seus arquivos registrados (MBTiles e GeoTIFF), sem rede e sem
alterá-los, e o mesmo plano com os mesmos registros dá sempre o mesmo
recorte, em qualquer lugar do planeta.

```console
$ sobrevoo plan atividade.gpx --export plano.json
$ sobrevoo geodata slice plano.json --export recorte.zip
Area: lat -23.7390 to -23.4359, lon -46.7584 to -46.4233
Base map detail (mapa): level 16 (ideal 16, source offers 10-16; within the source's range)
  nearest camera distance 2780.9 m, area closest to the equator at latitude 23.44, tiles of at most 4.27 m/px
Map tiles: 3779 present, 3 missing
  missing: mapa level 16 x=24278 y=37181
  ...
Elevation samples: 101505 (100 without value)
Elevation range: 700.0 m - 1099.0 m
Sources:
  mapa (base map, MBTiles)
  relevo (elevation, GeoTIFF)
Size: 466.6 KiB
Slice written to recorte.zip
$ sobrevoo geodata elevation --lat -23.5505 --lon -46.6333
Elevation: 760.0 m
Source: relevo (cell row 1203, column 884)
```

| Flag | Valores | Padrão | Descrição |
|---|---|---|---|
| `slice --export` | caminho | — | Grava o recorte completo num ZIP (formato em `specs/004-geo-data-slice/contracts/slice-file.md`) |
| `slice --overwrite` | — | — | Com `--export`, substitui um arquivo que já exista |
| `elevation --lat` | graus decimais, -90 a 90 | — | Latitude da coordenada (obrigatória) |
| `elevation --lon` | graus decimais, -180 a 180 | — | Longitude da coordenada (obrigatória) |

`elevation` informa a elevação, em metros, da célula do relevo registrado
que contém a coordenada, ou diz que o arquivo não tem valor para aquele
ponto, para você conferir contra outra fonte; sempre mostra qual registro e
qual célula (linha e coluna) foram usados.

Códigos de saída: `17` plano inválido, `18` plano de versão desconhecida,
`19` área não coberta, `20` recorte grande demais, `21` dado geográfico
ilegível, `22` unidade de elevação não suportada, `23` destino da exportação
já existe, `24` destino da exportação inválido, `25` coordenada sem
cobertura de relevo, `26` coordenada inválida; documentados em
`specs/004-geo-data-slice/contracts/cli.md`.

### `render frame` e `render all`: desenhar os quadros do voo

```sh
sobrevoo render frame <plano.json> <recorte.zip> --number <n> --output <quadro.png> [--resolution LxA] [--overwrite]
sobrevoo render all   <plano.json> <recorte.zip> --output <diretório>              [--resolution LxA] [--overwrite]
```

A partir do plano exportado por `plan --export` e do recorte exportado por
`geodata slice --export` **desse mesmo plano**, desenha o que a câmera vê em
cada quadro: o relevo do terreno em perspectiva, vestido com as peças do mapa
base, o traçado do trajeto até o ponto em que o marcador está e o marcador da
atividade. `render frame` desenha um só, pelo número, para conferir o
enquadramento antes de gastar tempo com o voo inteiro; `render all` desenha
todos, em `frame_000000.png`, `frame_000001.png`, ..., na ordem do plano, com
o progresso na tela e um resumo ao final.

```console
$ sobrevoo render frame plano.json recorte.zip --number 300 --output conferir.png
Frame 300 of 1260 drawn to conferir.png
Resolution: 1080x1920
Time: 00:00:01
Holes: map tiles missing: no, elevation without value: no
$ sobrevoo render all plano.json recorte.zip --output quadros/
Frames: 1260 requested, 1260 drawn, 0 kept (already in the destination)
Resolution: 1080x1920
Time: 00:14:52
Holes (in the frames drawn now): 87 with missing map tiles, 4 with elevation without value
Destination: quadros/ (frame_000000.png to frame_001259.png)
```

| Flag | Valores | Padrão | Descrição |
|---|---|---|---|
| `frame --number` | inteiro, 0 até o total de quadros do plano - 1 | — | Número do quadro a desenhar (obrigatória) |
| `frame --output` | caminho | — | Arquivo PNG de destino (obrigatória) |
| `all --output` | diretório | — | Diretório de destino dos quadros (obrigatória); criado se não existir |
| `--resolution` | `LxA`, ambos pares, 180 a 3840 por lado, no máximo 3.840×2.160 no total | `1080x1920` | Resolução das imagens; se mais estreita que a proporção do plano, avisa em `stderr` que a abertura/o fechamento podem ser cortados nas laterais |
| `--overwrite` | — | — | `frame`: substitui o arquivo se já existir. `all`: redesenha todos os quadros, mesmo os já prontos, e remove os de um voo anterior que sobrarem |

- **Sem inventar dado**: onde falta uma peça de mapa, o quadro mostra uma
  hachura cinza; onde o relevo não tem valor, um xadrez magenta; o resumo
  conta os quadros afetados. O fundo liso é o que está fora do recorte.
- **Determinismo**: o mesmo plano, o mesmo recorte e a mesma resolução dão
  sempre as mesmas imagens, byte a byte, seja qual for a ordem ou o número de
  quadros por execução, em qualquer processador.
- **Retomada e proteção**: `render all` continua de onde parou (é só repetir
  o comando; `Ctrl+C` encerra sem deixar imagem pela metade) e só redesenha
  os quadros que faltam. Um diretório com quadros de **outro** voo ou
  resolução é recusado, para não misturar; `--overwrite` refaz tudo.
- **Só mapa em imagem**: peças de mapa base em imagem (PNG, JPG, WebP). Peças
  vetoriais (`pbf`) são recusadas com mensagem clara; desenhá-las fica para
  uma etapa futura.

O recorte precisa ter sido exportado por esta versão (ele guarda a
identificação do plano de que veio); um recorte antigo é recusado com a
orientação de gerá-lo de novo.

Códigos de saída: `27` recorte inválido, `28` recorte de versão desconhecida,
`29` recorte de outro plano, `30` recorte que não cobre o plano, `31` peças
vetoriais no recorte, `32` sem dado de elevação, `33` número de quadro fora
do intervalo, `34` resolução inválida, `35` destino inválido, `36` destino
já existe, `37` diretório com quadros de outro conjunto, `38` execução
interrompida; documentados em
`specs/005-frame-rendering/contracts/cli.md`.

### `video`: montar o vídeo do voo

```sh
sobrevoo video <plano.json> <diretório-de-quadros> --output <voo.mp4> [--quality low|medium|high] [--overwrite]
```

Junta os quadros que `render all` desenhou num único arquivo MP4 (H.264, sem
áudio), na ordem e na taxa de quadros do **mesmo plano** de que eles vieram.
Antes de começar, confere que os quadros são do plano, que estão todos ali
(sem lacuna nem sobra), que têm a mesma resolução e que estão inteiros, e diz
exatamente o que falta ou destoa. Mostra o progresso na tela e, ao final, um
resumo.

```console
$ sobrevoo video plano.json quadros/ --output pedalada.mp4
Video written to pedalada.mp4
Frames: 1260
Duration: 00:00:42.000
Resolution: 1080x1920
Frame rate: 30 fps
Quality: medium
Size: 18.4 MiB
Encoder: ffmpeg 7.1 (libx264)
Time: 00:01:52
```

| Flag | Valores | Padrão | Descrição |
|---|---|---|---|
| `--output` | caminho terminado em `.mp4` | — | Destino do vídeo (obrigatória) |
| `--quality` | `low`, `medium`, `high` | `medium` | `low` (rápido e pequeno, para conferir), `medium` (para publicar), `high` (para guardar); um nível mais alto dá um arquivo maior ou igual, sem mudar duração, resolução nem ordem dos quadros |
| `--overwrite` | — | — | Substitui o vídeo de destino se já existir |

- **Sempre o mesmo vídeo**: o mesmo plano, os mesmos quadros, a mesma
  qualidade e o mesmo `ffmpeg` (o resumo diz qual foi) dão o mesmo arquivo,
  byte a byte; o arquivo não leva data, caminho, nome de máquina nem outro
  dado do ambiente.
- **Proteção**: o destino deve terminar em `.mp4` e, por padrão, não pode
  existir. O vídeo só aparece pronto: `Ctrl+C`, falta de espaço ou uma falha
  do codificador não deixam arquivo pela metade nem temporário. Não há
  retomada: repetir o comando recomeça.
- **Quadros de versões anteriores**: quadros desenhados antes de esta etapa
  existir (sem a identificação do plano dentro deles) são recusados com a
  orientação de desenhá-los de novo (`render all --overwrite`).

Códigos de saída: `40` diretório de quadros inválido, `41` quadros que
faltam/sobram/se repetem, `42` resoluções diferentes ou ímpares, `43`
quadros de outro plano, `44` quadros sem identificação de plano, `45`
quadro truncado, `46` codificador ausente, `47` destino já existe, `48`
destino inválido, `49` montagem interrompida, `50` falha do codificador;
documentados em `specs/006-video-assembly/contracts/cli.md`.

## Desenvolvimento

```sh
make build      # compila o binário em ./bin/sobrevoo
make test       # roda a suíte de testes com cobertura
make lint       # go vet
make generate   # regenera os mocks (go.uber.org/mock)
```

O projeto segue arquitetura hexagonal, com as convenções de código e de
teste documentadas em [`CLAUDE.md`](CLAUDE.md) e definidas de forma
vinculante pela [constituição do projeto](.specify/memory/constitution.md).
O desenvolvimento de novas features usa o fluxo
[Spec Kit](https://github.com/github/spec-kit) — cada feature vive em
`specs/<NNN-nome-da-feature>/`.

## Licença

[MIT](LICENSE)
