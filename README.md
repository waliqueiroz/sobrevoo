# Sobrevoo

Sobrevoo é uma ferramenta de linha de comando, em Go, que transforma um
trajeto GPS num vídeo de sobrevoo, no estilo Relive/Strava: uma câmera
virtual acompanha o caminho que você fez por cima do relevo real do terreno,
vestido com um mapa à sua escolha, enquanto um marcador percorre o traçado e
a distância, a elevação e o tempo da atividade aparecem na tela.

Tudo roda no seu computador, sem conta, sem servidor e sem internet: o mapa e
o relevo são arquivos que você mesmo baixa e apresenta ao Sobrevoo uma vez. A
única dependência externa é o [`ffmpeg`](https://ffmpeg.org), que codifica o
vídeo final. As mesmas entradas dão sempre o mesmo resultado, byte a byte (o
vídeo, desde que com o mesmo `ffmpeg`).

O vídeo ainda não tem áudio.

- [Instalação](#instalação)
- [Seu primeiro voo](#seu-primeiro-voo)
- [Ajustando o vídeo](#ajustando-o-vídeo)
- [Quando algo dá errado](#quando-algo-dá-errado)
- [Além do `fly`: casos específicos](#além-do-fly-casos-específicos)
- [Referência dos comandos](#referência-dos-comandos)
- [Códigos de saída](#códigos-de-saída)

## Instalação

Você precisa do Go 1.26 ou mais novo e do `ffmpeg` com o codificador
`libx264`, no `PATH`:

```sh
brew install ffmpeg            # macOS
sudo apt install ffmpeg        # Debian e Ubuntu
winget install Gyan.FFmpeg     # Windows
```

O Sobrevoo não traz nem baixa o `ffmpeg`; se ele faltar, diz o que instalar
— no `fly`, antes de começar o voo.

```sh
go install github.com/waliqueiroz/sobrevoo/cmd/sobrevoo@latest
```

Ou, a partir do código-fonte:

```sh
git clone git@github.com:waliqueiroz/sobrevoo.git
cd sobrevoo
make build   # gera ./bin/sobrevoo
```

Todo comando aceita `--help`, que lista as flags com seus padrões. As
mensagens do programa são em inglês.

## Seu primeiro voo

São quatro passos: ter o trajeto, conseguir um mapa e um relevo da região,
registrá-los e voar. Os três primeiros você faz uma vez só; depois disso,
cada vídeo novo da mesma região é um único comando.

### 1. O trajeto

O **trajeto** é o registro GPS da sua atividade — a pedalada, a corrida, a
caminhada —, uma sequência de pontos com latitude, longitude e, normalmente,
altitude e horário. O Sobrevoo lê arquivos **GPX**, o formato que Strava,
Garmin Connect, Komoot e a maioria dos aplicativos de atividade física
exportam.

Não é preciso limpar o arquivo antes: o Sobrevoo descarta sozinho pontos com
coordenadas impossíveis, pontos repetidos em sequência e saltos fisicamente
implausíveis (acima de 130 km/h entre dois pontos), reordena os pontos por
horário quando necessário e suaviza o traçado. Um trajeto sem horários ou sem
altitude também funciona, com o que se perde descrito em
[Ajustando o vídeo](#sobreposições-de-tela).

### 2. O mapa base e o relevo

O Sobrevoo nunca embute nem baixa dados geográficos. Para desenhar o voo, ele
precisa de dois arquivos que cubram a região do trajeto:

- O **relevo** é a altitude do terreno, ponto a ponto — o que dá forma às
  montanhas e aos vales do vídeo. Formato: **GeoTIFF** de uma banda, em
  coordenadas geográficas (latitude e longitude, como o WGS84), sem
  compressão ou com compressão Deflate ou LZW, em metros ou pés. O
  [Copernicus DEM GLO-30](https://dataspace.copernicus.eu) e o SRTM, ambos
  gratuitos e distribuídos por fontes como o
  [OpenTopography](https://opentopography.org), já vêm assim. Se o seu
  arquivo estiver em outra projeção (UTM, por exemplo), o GDAL converte:
  `gdalwarp -t_srs EPSG:4326 -co COMPRESS=DEFLATE entrada.tif relevo.tif`.
- O **mapa base** é a imagem que veste o relevo — ruas, imagem de satélite,
  um mapa topográfico. Formato: **MBTiles de imagem**, com as peças em PNG,
  JPG ou WebP e a área coberta declarada nos metadados (`bounds`). Uma forma
  de gerar um é o algoritmo *Generate XYZ tiles (MBTiles)* do
  [QGIS](https://qgis.org), a partir de qualquer camada de mapa aberta nele;
  respeite os termos de uso de quem fornece o mapa. **MBTiles vetorial** (com
  peças `pbf`, como os do OpenMapTiles ou do BBBike) **não serve**: é aceito
  no registro, mas recusado na hora de desenhar.

Os dois arquivos precisam cobrir com folga o **entorno** do trajeto, não só
o traçado: a câmera voa afastada e enxerga longe, sobretudo na abertura e no
fechamento do vídeo, quando mostra o trajeto inteiro de cima. Uma área de
vários quilômetros em volta é o normal; num passeio de 3 km de lado, a área
vista pode passar de 15 km de lado.

### 3. O registro

O **registro** é a lista dos mapas e relevos que você apresentou ao
Sobrevoo, cada um com um nome escolhido por você. Ele só guarda o caminho do
arquivo e a área que ele cobre — o arquivo continua onde está e não pode ser
movido depois (se for, remova o registro com `geodata remove` e registre o
arquivo de novo). O registro fica em
`~/.sobrevoo/registry.json`.

```console
$ sobrevoo geodata register ~/mapas/sp-satelite.mbtiles --name sp-mapa
Registered "sp-mapa" as base map, covering lat [-23.800000, -23.300000], lon [-46.900000, -46.300000]
$ sobrevoo geodata register ~/mapas/sp-relevo.tif --name sp-relevo
Registered "sp-relevo" as elevation, covering lat [-24.000000, -23.000000], lon [-47.000000, -46.000000]
```

Você não diz se o arquivo é mapa ou relevo, nem a área: o Sobrevoo descobre
as duas coisas pelo conteúdo. Para conferir se o trajeto está dentro do que
foi registrado:

```console
$ sobrevoo geodata check pedalada.gpx
Coverage: full
Base map sources used: sp-mapa
Elevation sources used: sp-relevo
```

`check` confere os pontos do trajeto, não o entorno que a câmera vê: um
`full` aqui é um bom sinal, mas o voo ainda pode recusar uma área de borda
(veja [Quando algo dá errado](#quando-algo-dá-errado)).

### 4. O voo

```console
$ sobrevoo fly pedalada.gpx --output pedalada.mp4
Stage 1/5: treating the track
Stage 2/5: planning the camera
Stage 3/5: slicing the geo data
Stage 4/5: drawing the frames
Drawing frame 1020/1020 (100.0%), elapsed 00:23:41
Stage 5/5: encoding the video
Encoding frame 1020/1020 (100.0%), elapsed 00:01:20
Frames: 1020 requested, 1020 drawn, 0 kept (already in the destination)
Resolution: 1080x1920
Time: 00:23:41
Holes (in the frames drawn now): none
Video written to pedalada.mp4
Frames: 1020
Duration: 00:00:34.000
Resolution: 1080x1920
Frame rate: 30 fps
Quality: medium
Size: 15.7 MiB
Encoder: ffmpeg 7.1 (libx264)
Time: 00:01:20
Total time: 00:25:02
```

Pronto: `pedalada.mp4` é um vídeo vertical (1080×1920, para celular e redes
sociais) que abre mostrando o trajeto inteiro de cima, desce para acompanhar
o marcador do início ao fim e fecha mostrando o trajeto completo de novo. A
duração, quando você não escolhe uma, cresce com o comprimento do trajeto,
entre 20 s e 120 s.

A etapa demorada é o desenho dos quadros: dezenas de minutos na resolução
padrão. Para um teste rápido antes do vídeo definitivo, use uma resolução
menor e a qualidade mais baixa — `--resolution 540x960 --quality low` leva
uma fração do tempo.

#### O que acontece durante o voo

Antes de qualquer etapa, o `fly` confere o que pode recusar sem ler o trajeto:
os valores das flags (inclusive se `--output` termina em `.mp4`) e se o
arquivo do trajeto abre, depois o destino do vídeo (que, sem `--overwrite`,
ainda não pode existir) e por fim se o `ffmpeg` está disponível. Só então
passa pelas cinco etapas que a saída numera:

1. **Tratamento do trajeto** — lê o GPX, descarta os pontos inválidos e
   reduz e suaviza o traçado (veja `--simplification` e `--smoothing`).
2. **Planejamento da câmera** — calcula o **plano de câmera**: para cada
   quadro do vídeo, onde a câmera está, para onde aponta e em que ponto do
   trajeto está o marcador. O movimento sai sempre suave, mesmo em curvas
   fechadas e retornos, e paradas longas são comprimidas.
3. **Recorte dos dados geográficos** — reúne o **recorte**: só as peças do
   mapa e as amostras de relevo que a câmera vai de fato enxergar, no nível
   de detalhe adequado à distância em que ela voa. É aqui que o Sobrevoo
   confere se o mapa e o relevo registrados cobrem toda a área vista, e
   recusa, dizendo o que falta, se não cobrirem — antes de gastar tempo
   desenhando.
4. **Desenho dos quadros** — desenha os **quadros**, as imagens que formam o
   vídeo, uma por instante: o relevo em perspectiva vestido com o mapa, o
   traçado até o marcador, o marcador e as sobreposições de tela.
5. **Codificação do vídeo** — o `ffmpeg` junta os quadros num MP4 (H.264,
   sem áudio).

Sem `--keep`, nada disso fica para trás: o plano e o recorte nunca tocam o
disco, e os quadros vivem num diretório temporário, apagado ao final mesmo
em caso de falha ou interrupção. `Ctrl+C` encerra de forma ordenada, assim
que a etapa em curso permitir, diz quais etapas já tinham terminado e nunca
deixa um vídeo pela metade.

Cada uma dessas etapas existe também como comando próprio, para quem quiser
examinar ou reaproveitar um resultado intermediário — são cinco etapas, mas
**quatro comandos**, porque o tratamento do trajeto não é um comando à parte:
ele acontece dentro do `plan`, que recebe o GPX e já o trata antes de
planejar.

| Etapas do `fly` | Comando equivalente | Produz |
|---|---|---|
| 1 e 2 | `sobrevoo plan <trajeto.gpx> --export plano.json` | o plano de câmera |
| 3 | `sobrevoo geodata slice plano.json --export recorte.zip` | o recorte |
| 4 | `sobrevoo render all plano.json recorte.zip --output quadros/` | os quadros |
| 5 | `sobrevoo video plano.json quadros/ --output voo.mp4` | o vídeo |

Rodar esses quatro comandos na mão, com os mesmos valores, dá o mesmo vídeo,
byte a byte, que o `fly` — é literalmente o mesmo código. Ficam de fora o
`geodata register`, que é preparação, feita uma vez antes de qualquer voo, e
o `inspect`, que só mostra o resultado do tratamento do trajeto, sem fazer
parte do voo. Veja [Além do `fly`](#além-do-fly-casos-específicos).

## Ajustando o vídeo

Todas as flags desta seção são do `fly`. Os comandos individuais aceitam as
que dizem respeito à sua etapa, com o mesmo nome, os mesmos valores, os
mesmos padrões e as mesmas mensagens de erro — a
[referência](#referência-dos-comandos) diz qual comando aceita qual.

### Câmera, duração e formato

| Flag | Valores | Padrão | O que faz |
|---|---|---|---|
| `--duration` | segundos, maior que 0 e até 3600 | automática | Duração do vídeo. Sem ela, é calculada a partir do comprimento do trajeto (de 20 s a 120 s); um valor curto demais para o trajeto é recusado, dizendo o mínimo |
| `--fps` | 1 a 120 | `30` | Quadros por segundo |
| `--distance` | `low`, `medium`, `high` | `medium` | Quão longe a câmera voa do marcador |
| `--tilt` | `low`, `medium`, `high` | `medium` | Quão de cima a câmera olha: `low` perto do horizonte, `high` quase na vertical |
| `--aspect` | `L:A`, inteiros de 1 a 1000, razão de `1:5` a `5:1` | `9:16` | Proporção do vídeo, usada para enquadrar o trajeto inteiro na abertura e no fechamento |
| `--resolution` | `LxA` em pixels, ambos pares, de 180 a 3840 por lado, no máximo 3840×2160 no total | `1080x1920` | Tamanho dos quadros e do vídeo |

`--aspect` e `--resolution` andam juntos: para um vídeo horizontal, use
`--aspect 16:9 --resolution 1920x1080`. Uma resolução mais estreita que a
proporção do plano é aceita, mas gera um aviso, porque a abertura e o
fechamento podem sair cortados nas laterais.

### Tratamento do traçado

| Flag | Valores | Padrão | O que faz |
|---|---|---|---|
| `--simplification` | `low`, `medium`, `high` | `medium` | Quanto o traçado é reduzido: quanto mais alto, menos pontos e menos detalhe nas curvas pequenas |
| `--smoothing` | `low`, `medium`, `high` | `medium` | Quanto o traçado é suavizado: quanto mais alto, mais arredondadas as curvas |

Os dois níveis mudam o traçado desenhado e o caminho que a câmera segue.

### Aparência

| Flag | Valores | Padrão | O que faz |
|---|---|---|---|
| `--trail-color` | `#RRGGBB` | `#FFB000` | Cor do traçado |
| `--trail-width` | proporção da altura do quadro, de `0.0005` a `0.05` | `0.005` | Espessura do traçado |
| `--marker-color` | `#RRGGBB` | `#E5252A` | Cor do marcador |
| `--marker-radius` | proporção da altura do quadro, de `0.001` a `0.1` | `0.012` | Raio do marcador |
| `--background-color` | `#RRGGBB` | `#20262E` | Cor do fundo: acima do horizonte, fora da área do recorte e por baixo de uma peça de mapa parcialmente transparente |

A espessura e o raio são proporções da altura do quadro, para o desenho
manter o mesmo aspecto em qualquer resolução; numa resolução muito pequena,
um mínimo em pixels impede que sumam. Duas marcações não são ajustáveis,
porque são o significado da imagem e não estilo: onde falta uma peça de mapa,
o quadro mostra uma **hachura cinza**; onde o relevo não tem valor, um
**xadrez magenta**. O Sobrevoo nunca inventa um dado que não tem.

### Sobreposições de tela

Fixos na tela, por cima do terreno, o vídeo mostra quatro blocos, todos
ligados por padrão: a distância percorrida até o marcador (`DIST`), a
elevação no ponto do marcador e o ganho acumulado até ali (`ELEV` e
`GANHO`), o tempo decorrido da atividade (`TEMPO`) e, na parte de baixo, um
perfil de elevação do trajeto inteiro com um marcador que avança com o voo.
Os valores vêm só do trajeto, nunca do relevo registrado; o ganho acumulado
do último quadro é exatamente o ganho de elevação que o `inspect` relata para
o mesmo trajeto, com os mesmos níveis de tratamento.

| Flag | Valores | Padrão | O que faz |
|---|---|---|---|
| `--overlays` | `true`, `false` | `true` | Liga ou desliga todos os blocos de uma vez |
| `--overlay-blocks` | lista separada por vírgula de `distance`, `elevation`, `time`, `profile` | os quatro | Quais blocos aparecem, quando `--overlays` não é `false` |

Por ser uma flag booleana, `--overlays` precisa do sinal de igual para ser
desligada: `--overlays=false`. Um trajeto sem altitude em todos os pontos
fica sem a elevação e sem o perfil; um sem horários (ou com horários
inconsistentes) fica sem o tempo — e o marcador avança pela distância, não
pelo relógio.

### Escolha da fonte de dados

Com mais de um mapa ou relevo registrado, o Sobrevoo escolhe sozinho, ponto a
ponto, a fonte mais específica entre as que cobrem aquele ponto (a de menor
área; em caso de empate, a registrada primeiro) — e pode, assim, combinar
fontes diferentes ao longo de um mesmo trajeto. Para usar uma fonte
específica:

| Flag | Valores | Padrão | O que faz |
|---|---|---|---|
| `--base-map` | nome já registrado como mapa base | seleção automática | Usa exclusivamente esse mapa base |
| `--elevation` | nome já registrado como relevo | seleção automática | Usa exclusivamente esse relevo |

A escolha explícita nunca mistura fontes: se a fonte pedida não cobrir toda
a área, o voo é recusado, em vez de ser completado em silêncio por outra
fonte registrada.

### Saída

| Flag | Valores | Padrão | O que faz |
|---|---|---|---|
| `--output` | caminho terminado em `.mp4` | — | Destino do vídeo (obrigatória) |
| `--quality` | `low`, `medium`, `high` | `medium` | `low` é rápido e pequeno, para conferir; `medium`, para publicar; `high`, para guardar. Um nível mais alto dá um arquivo maior ou igual, sem mudar a duração, a resolução nem a ordem dos quadros |
| `--overwrite` | — | — | Substitui o vídeo se ele já existir (e, com `--keep`, os intermediários desatualizados) |
| `--keep` | diretório | — | Guarda o plano, o recorte e os quadros para reaproveitá-los numa execução seguinte; veja [Guardar e reaproveitar](#guardar-e-reaproveitar-entre-execuções---keep) |

## Quando algo dá errado

As mensagens de erro do Sobrevoo dizem o que aconteceu e, quase sempre, o que
fazer; o código de saída de cada uma está em
[Códigos de saída](#códigos-de-saída). Os tropeços mais comuns num primeiro
voo:

- **`area is not fully covered by the registered geo data`** (código `19`): o
  mapa ou o relevo não cobrem toda a área que a câmera vê. A mensagem lista os
  trechos e o que falta em cada um (mapa base, relevo ou os dois). Registre
  um arquivo que cubra mais área em volta do trajeto.
- **`geo data slice is too large`** (código `20`): o recorte passaria de
  256 MiB. Acontece com mapas de muito detalhe (níveis de zoom 17 e 18) e a
  câmera perto do chão. Afaste a câmera com `--distance high`, use um mapa
  com nível máximo de detalhe menor, ou um trajeto mais curto.
- **`tile format is not supported for drawing`** (código `31`): o mapa base
  registrado é um MBTiles vetorial. Use um MBTiles de imagem.
- **`unsupported geo data file format`** (código `7`, no `register`): o
  arquivo não é um MBTiles nem um GeoTIFF que o Sobrevoo entenda — por
  exemplo, um GeoTIFF em projeção UTM, que precisa ser convertido para
  latitude e longitude (veja [O mapa base e o relevo](#2-o-mapa-base-e-o-relevo)).
- **`video encoder not available`** (código `46`): o `ffmpeg`, ou o
  `libx264` dentro dele, não foi encontrado. Instale como em
  [Instalação](#instalação).
- **`video destination already exists`** (código `47`): o vídeo de destino já
  existe. Escolha outro `--output` ou passe `--overwrite`.

## Além do `fly`: casos específicos

### Guardar e reaproveitar entre execuções (`--keep`)

Com `--keep <diretório>`, o `fly` guarda os três resultados intermediários
nesse diretório (criado se ainda não existir), em vez de descartá-los:

```text
<diretório>/
├── plan.json     # o plano de câmera, como o de "plan --export"
├── slice.zip     # o recorte, como o de "geodata slice --export"
└── frames/       # os quadros, como os de "render all --output"
```

Numa execução seguinte com o mesmo `--keep`, o `fly` reaproveita o que ainda
vale para o trajeto e os valores informados agora, e avisa numa linha logo
abaixo da etapa:

```console
Stage 2/5: planning the camera
  unchanged since the last run under --keep, reusing plan.json
Stage 3/5: slicing the geo data
  unchanged, reusing slice.zip
```

O plano é sempre recalculado — é rápido — e só comparado com o guardado; o
ganho de tempo está no recorte e, sobretudo, nos quadros, que não são
redesenhados. O que vale depende do que mudou:

| O que mudou desde a execução anterior | Plano | Recorte | Quadros |
|---|---|---|---|
| nada (só um `--output` novo), ou a execução anterior foi interrompida | reaproveitado | reaproveitado | reaproveitados; só os que faltam são desenhados |
| só `--quality`, com um `--output` novo | reaproveitado | reaproveitado | reaproveitados (só o vídeo é refeito) |
| `--resolution`, uma flag de aparência ou de sobreposição | reaproveitado | reaproveitado | redesenhados |
| `--base-map` ou `--elevation`, ou o registro, de um jeito que muda as fontes de que o recorte seria tirado agora (inclusive passar de seleção automática para explícita, ou o contrário) | reaproveitado | refeito | redesenhados |
| `--duration`, `--fps`, `--distance`, `--tilt`, `--aspect`, `--simplification`, `--smoothing` ou o próprio trajeto | refeito | refeito | redesenhados |

Duas regras de proteção valem aqui, iguais às dos comandos individuais:

- Um intermediário desatualizado só é substituído com `--overwrite`. Sem
  ela, o `fly` recusa (o plano com o código `15`, o recorte com o `23`, os
  quadros de outro conjunto com o `37`) em vez de apagar um resultado
  anterior.
- `--overwrite` também redesenha todos os quadros, mesmo os que ainda valem
  (a mesma regra do `render all --overwrite`). Por isso, para aproveitar os
  quadros prontos e só refazer o vídeo — numa troca de `--quality`, por
  exemplo —, use um `--output` que ainda não exista, em vez de
  `--overwrite`.

Arquivos seus dentro do diretório que não sejam esses três nunca são lidos,
alterados nem removidos.

### Conferir um quadro antes do voo inteiro

O desenho de todos os quadros é demorado; para testar uma cor, um
enquadramento ou as sobreposições, desenhe um só:

```sh
sobrevoo plan pedalada.gpx --export plano.json
sobrevoo geodata slice plano.json --export recorte.zip
sobrevoo render frame plano.json recorte.zip --number 300 --output conferir.png --trail-color "#00FF00"
```

O quadro sai idêntico, pixel a pixel, ao mesmo quadro dentro do voo inteiro
com os mesmos valores. Depois de satisfeito, `render all` e `video`
completam o voo a partir do mesmo plano e do mesmo recorte — ou o `fly`, com
as mesmas flags.

### Inspecionar o trajeto

`sobrevoo inspect pedalada.gpx` mostra o que o tratamento do trajeto fez com
o arquivo — quantos pontos foram descartados e por quê, quantos sobraram
depois da redução — e o resumo da atividade: distância, ganho de elevação,
duração e a área coberta. Útil para entender um GPX estranho, ou para ver o
efeito de `--simplification` e `--smoothing` antes de voar.

### Consultar a elevação de um ponto

`sobrevoo geodata elevation --lat <graus> --lon <graus>` informa a elevação,
em metros, que o relevo registrado dá para uma coordenada, e de qual arquivo
e célula ela veio — para conferir o relevo contra outra fonte.

### Rodar uma etapa de cada vez

Os quatro comandos da tabela de
[O que acontece durante o voo](#o-que-acontece-durante-o-voo) servem para
ver ou guardar um resultado intermediário, gerar um recorte uma vez e
desenhar vários voos a partir dele, ou repetir só a etapa que mudou. Cada
arquivo intermediário é verificado ao ser lido: um recorte de outro plano,
ou quadros de outro plano, são recusados em vez de misturados.

## Referência dos comandos

As flags compartilhadas — de câmera, de tratamento, de aparência, de
sobreposição, de escolha de fonte e de qualidade — estão descritas uma única
vez, em [Ajustando o vídeo](#ajustando-o-vídeo). Esta seção lista quais cada
comando aceita e descreve só as que são próprias dele. Toda saída de resumo
vai para a saída padrão; o progresso e os avisos, para a saída de erro.

Uma regra vale para todo comando que grava um arquivo: o destino nunca é
sobrescrito sem `--overwrite`, e o arquivo só aparece quando está completo —
uma interrupção, falta de espaço ou falha nunca deixa um arquivo pela metade.

### `fly`

```sh
sobrevoo fly <trajeto.gpx> --output <voo.mp4> [flags]
```

Aceita todas as flags de [Ajustando o vídeo](#ajustando-o-vídeo).

### `inspect`

```sh
sobrevoo inspect <trajeto.gpx> [--simplification low|medium|high] [--smoothing low|medium|high]
```

```console
$ sobrevoo inspect pedalada.gpx
Format: GPX
Points: 3633 -> 82 (original -> treated)
Distance: 9.52 km
Elevation gain: 183.4 m
Duration: 1h4m6s
Bounding box: lat [-23.571204, -23.549254], lon [-46.672310, -46.647837]
Discarded points: 318 (impossible coordinates: 0, consecutive duplicates: 318, implausible jumps: 0)
```

`Points` compara os pontos do arquivo com os que sobram depois da limpeza,
da redução e da suavização. Sem altitude ou sem horários no arquivo, a linha
correspondente diz `not available`; um trajeto que cruza o meridiano de 180°
é marcado como tal em `Bounding box`.

### `geodata`

```sh
sobrevoo geodata register <arquivo> --name <nome>
sobrevoo geodata list
sobrevoo geodata remove <nome>
sobrevoo geodata clear --confirm
sobrevoo geodata check <trajeto.gpx> [--base-map <nome>] [--elevation <nome>]
sobrevoo geodata slice <plano.json> [--export <recorte.zip>] [--overwrite] [--base-map <nome>] [--elevation <nome>]
sobrevoo geodata elevation --lat <graus> --lon <graus>
```

- **`register`** registra um mapa base (MBTiles) ou um relevo (GeoTIFF) sob
  `--name` (obrigatória, única no registro). O tipo e a área vêm do conteúdo
  do arquivo.
- **`list`** lista os registros, com o tipo e a área de cada um. Um registro
  cujo arquivo foi movido ou apagado continua listado, marcado com
  `(file not found)`, e deixa de ser usado pelos demais comandos.
- **`remove`** remove um registro pelo nome. Nunca apaga o arquivo.
- **`clear`** remove todos os registros de uma vez — nunca os arquivos. Sem
  `--confirm`, não remove nada e diz quantos registros seriam removidos; não
  há pergunta interativa.
- **`check`** diz se os pontos do trajeto estão cobertos por mapa base **e**
  relevo: `full`, `partial` ou `none`, com os trechos descobertos (dados pelas
  coordenadas de onde cada um começa e termina) e o que falta em cada um. Sai
  com código `0` sempre que consegue ler o trajeto, mesmo sem cobertura.
  Aceita `--base-map` e `--elevation`; com elas, o relatório reflete só a
  fonte pedida, e o que ela não cobre aparece como lacuna.
- **`slice`** é a etapa 3 do voo: lê um plano exportado por `plan --export` e
  reúne o recorte. Aceita `--base-map` e `--elevation`. Sem `--export`, só
  mostra o resumo; com `--export <recorte.zip>`, grava o recorte num ZIP
  (`--overwrite`, só junto com `--export`, substitui um arquivo existente).
  Peças que faltam dentro de um mapa registrado são listadas e não
  interrompem o recorte — viram a hachura cinza nos quadros; amostras sem
  valor no relevo ficam marcadas como sem valor, nunca como zero.
- **`elevation`** lê a elevação de uma coordenada (`--lat` de -90 a 90 e
  `--lon` de -180 a 180, ambas obrigatórias, em graus decimais), ou diz que o
  arquivo não tem valor para aquele ponto.

```console
$ sobrevoo geodata list
sp-mapa (base map): lat [-23.800000, -23.300000], lon [-46.900000, -46.300000]
sp-relevo (elevation): lat [-24.000000, -23.000000], lon [-47.000000, -46.000000]
$ sobrevoo geodata slice plano.json --export recorte.zip
Area: lat -23.6612 to -23.4593, lon -46.7637 to -46.5561
Base map detail (sp-mapa): level 16 (ideal 16, source offers 10-16; within the source's range)
  nearest camera distance 2780.9 m, area closest to the equator at latitude 23.46, tiles of at most 4.27 m/px
Map tiles: 3779 present, 3 missing
  missing: sp-mapa level 16 x=24278 y=37181
  missing: sp-mapa level 16 x=24279 y=37181
  missing: sp-mapa level 16 x=24280 y=37181
Elevation samples: 101505 (100 without value)
Elevation range: 700.0 m - 1099.0 m
Sources:
  sp-mapa (base map, MBTiles)
  sp-relevo (elevation, GeoTIFF)
Size: 46.6 MiB
Slice written to recorte.zip
$ sobrevoo geodata elevation --lat -23.5505 --lon -46.6333
Elevation: 760.0 m
Source: sp-relevo (cell row 1203, column 884)
```

O resumo do `slice` explica o nível de detalhe escolhido para o mapa: o
ideal para a menor distância em que a câmera passa, limitado ao que o arquivo
oferece.

### `plan`

```sh
sobrevoo plan <trajeto.gpx> [--duration <s>] [--fps <n>] [--distance <nível>] [--tilt <nível>]
              [--simplification <nível>] [--smoothing <nível>] [--aspect <L:A>]
              [--export <plano.json>] [--overwrite]
```

Corresponde às etapas 1 e 2 do voo: trata o trajeto e calcula o plano de
câmera. Aceita as flags de câmera, duração, formato e tratamento. Sem
`--export`, só mostra o resumo; com `--export <plano.json>`, grava o plano
completo, em JSON (`--overwrite`, só junto com `--export`, substitui um
arquivo existente).

```console
$ sobrevoo plan pedalada.gpx --export plano.json
Duration: 34.0 s (automatic)
Frame rate: 30.0 fps
Aspect ratio: 9:16
Frames: 1020
Camera altitude: 1802.4 m - 8029.4 m
Camera distance: 2780.9 m - 9133.1 m
Time reference: clock
Smoothed spans: none
Plan written to plano.json
```

`Time reference` diz o que move o marcador: `clock`, os horários do GPX, ou
`distance`, a distância percorrida, quando o arquivo não tem horários
utilizáveis (o motivo vem entre parênteses). `Smoothed spans` lista os
trechos do vídeo em que o movimento da câmera precisou ser suavizado além do
normal — numa curva fechada, por exemplo.

O plano exportado é um JSON legível: os parâmetros usados e, para cada
quadro, o instante no vídeo e na atividade, a fase (abertura, acompanhamento
ou fechamento), a posição e a altitude da câmera, a direção e a inclinação
para onde ela aponta e a posição, a distância percorrida, a elevação e o
ganho acumulado do marcador.

### `render frame` e `render all`

```sh
sobrevoo render frame <plano.json> <recorte.zip> --number <n> --output <quadro.png> [flags] [--overwrite]
sobrevoo render all   <plano.json> <recorte.zip> --output <diretório> [flags] [--overwrite]
```

É a etapa 4 do voo: desenha quadros a partir de um plano e de um recorte
**feito desse mesmo plano**. Os dois aceitam `--resolution` e as flags de
aparência e de sobreposição.

- **`render frame`** desenha um quadro só: `--number` (obrigatória, de 0 ao
  total de quadros do plano menos 1) num PNG em `--output` (obrigatória).
- **`render all`** desenha todos, como `frame_000000.png`,
  `frame_000001.png`, ..., no diretório `--output` (obrigatória; criado se
  não existir), com o progresso na tela.

```console
$ sobrevoo render frame plano.json recorte.zip --number 300 --output conferir.png
Frame 300 of 1020 drawn to conferir.png
Resolution: 1080x1920
Time: 00:00:01
Holes: map tiles missing: no, elevation without value: no
$ sobrevoo render all plano.json recorte.zip --output quadros/
Frames: 1020 requested, 1020 drawn, 0 kept (already in the destination)
Resolution: 1080x1920
Time: 00:23:41
Holes (in the frames drawn now): 87 with missing map tiles, 4 with elevation without value
Destination: quadros/ (frame_000000.png to frame_001019.png)
```

`Holes` conta os quadros com hachura de mapa ou xadrez de relevo.

O `render all` **retoma** de onde parou: interrompido com `Ctrl+C`, basta
repetir o mesmo comando, e só os quadros que faltam são desenhados. Cada
quadro carrega, dentro do próprio PNG, a identificação do plano, do recorte,
da resolução, da aparência e das sobreposições com que foi desenhado — o seu
**conjunto**. Um diretório com quadros de outro conjunto, ou com outras
imagens no lugar dos quadros, é recusado, para nunca misturar voos;
`--overwrite` redesenha todos os quadros e remove os que sobrarem de um voo
anterior mais longo.

### `video`

```sh
sobrevoo video <plano.json> <diretório-de-quadros> --output <voo.mp4> [--quality <nível>] [--overwrite]
```

É a etapa 5 do voo: junta os quadros num MP4 (H.264, sem áudio), na ordem e
na taxa de quadros do plano. `--output` é obrigatória e precisa terminar em
`.mp4`; aceita `--quality` e `--overwrite`.

```console
$ sobrevoo video plano.json quadros/ --output pedalada.mp4
Video written to pedalada.mp4
Frames: 1020
Duration: 00:00:34.000
Resolution: 1080x1920
Frame rate: 30 fps
Quality: medium
Size: 15.7 MiB
Encoder: ffmpeg 7.1 (libx264)
Time: 00:01:20
```

Antes de codificar, o `video` confere, nesta ordem, que o diretório tem
quadros do Sobrevoo, que eles foram desenhados a partir do plano informado,
que têm todos a mesma resolução, que são de um só conjunto, que estão todos
ali — numerados de 0 ao último, sem lacuna nem sobra — e que nenhum está
truncado; e diz exatamente o que falta ou destoa. Só depois confere o
destino e o `ffmpeg`. Não há retomada: uma montagem interrompida recomeça do
início.

O arquivo não leva data, caminho, nome de máquina nem outro dado do
ambiente: o mesmo plano, os mesmos quadros, a mesma qualidade e o mesmo
`ffmpeg` (o resumo diz qual) dão o mesmo vídeo, byte a byte.

## Códigos de saída

Cada erro tem um código de saída próprio, útil em scripts. O `fly` sai com o
mesmo código que o comando da etapa em que falhou, com uma única exceção: a
interrupção, que no `fly` tem código próprio (`51`).

| Código | Significado | Comandos |
|---|---|---|
| `0` | sucesso | todos |
| `1` | arquivo de trajeto vazio | `inspect`, `geodata check`, `plan`, `fly` |
| `2` | formato de trajeto não reconhecido, ou uso inválido da linha de comando (argumento ou flag obrigatória faltando, valor fora da lista aceita, `--output` sem `.mp4`, `--overwrite` sem `--export`) | todos |
| `3` | trajeto com pontos insuficientes, antes ou depois da limpeza | `inspect`, `geodata check`, `plan`, `fly` |
| `4` | falha não classificada (por exemplo, arquivo de trajeto inexistente ou ilegível) | todos |
| `5` | arquivo de dado geográfico não encontrado | `geodata register` |
| `6` | arquivo de dado geográfico ilegível | `geodata register` |
| `7` | formato de dado geográfico não suportado | `geodata register` |
| `8` | nome já em uso no registro | `geodata register` |
| `9` | nome não registrado | `geodata remove`, e `--base-map`/`--elevation` |
| `10` | duração inválida | `plan`, `fly` |
| `11` | taxa de quadros inválida | `plan`, `fly` |
| `12` | duração curta demais para o trajeto | `plan`, `fly` |
| `13` | trajeto curto demais (menos de 50 m) | `plan`, `fly` |
| `14` | trajeto grande demais (mais de 2.000 km de extensão) | `plan`, `fly` |
| `15` | destino do plano já existe | `plan --export`, `fly --keep` |
| `16` | destino do plano inválido | `plan --export`, `fly --keep` |
| `17` | arquivo de plano inválido | `geodata slice`, `render`, `video` |
| `18` | arquivo de plano de uma versão não suportada (gere o plano de novo) | `geodata slice`, `render`, `video` |
| `19` | área não coberta pelos dados registrados | `geodata slice`, `fly` |
| `20` | recorte grande demais (mais de 256 MiB) | `geodata slice`, `fly` |
| `21` | conteúdo do dado geográfico ilegível | `geodata slice`, `geodata elevation`, `fly` |
| `22` | unidade de elevação não suportada | `geodata slice`, `geodata elevation`, `fly` |
| `23` | destino do recorte já existe | `geodata slice --export`, `fly --keep` |
| `24` | destino do recorte inválido | `geodata slice --export`, `fly --keep` |
| `25` | nenhum relevo registrado cobre a coordenada | `geodata elevation` |
| `26` | coordenada inválida | `geodata elevation` |
| `27` | arquivo de recorte inválido | `render` |
| `28` | arquivo de recorte de uma versão não suportada (gere o recorte de novo) | `render` |
| `29` | recorte feito de outro plano | `render` |
| `30` | recorte que não cobre o plano | `render` |
| `31` | mapa base com peças vetoriais | `render`, `fly` |
| `32` | recorte sem nenhum dado de elevação | `render`, `fly` |
| `33` | número de quadro fora do intervalo | `render frame` |
| `34` | resolução inválida | `render`, `fly` |
| `35` | destino dos quadros inválido | `render`, `fly` |
| `36` | destino do quadro já existe | `render frame` |
| `37` | diretório com quadros de outro conjunto | `render all`, `fly --keep` |
| `38` | desenho interrompido | `render` |
| `39` | proporção (`--aspect`) inválida | `plan`, `fly` |
| `40` | diretório de quadros inválido | `video` |
| `41` | quadros que faltam, sobram ou se repetem | `video` |
| `42` | quadros de resoluções diferentes, ou de dimensão ímpar | `video` |
| `43` | quadros de outro plano, ou de mais de um conjunto misturados | `video` |
| `44` | quadros sem a identificação do plano (desenhados por uma versão antiga: redesenhe com `render all --overwrite`) | `video` |
| `45` | quadro truncado | `video` |
| `46` | `ffmpeg` ou `libx264` indisponível | `video`, `fly` |
| `47` | destino do vídeo já existe | `video`, `fly` |
| `48` | destino do vídeo inválido | `video`, `fly` |
| `49` | montagem do vídeo interrompida | `video` |
| `50` | falha do `ffmpeg` durante a codificação | `video`, `fly` |
| `51` | voo interrompido | `fly` |
| `52` | cor mal formada | `render`, `fly` |
| `53` | espessura do traçado fora do intervalo | `render`, `fly` |
| `54` | raio do marcador fora do intervalo | `render`, `fly` |
| `55` | bloco de sobreposição desconhecido | `render`, `fly` |
| `56` | nome registrado como o outro tipo (um relevo pedido como mapa base, ou o contrário) | `--base-map`/`--elevation` |
| `57` | `geodata clear` sem `--confirm` | `geodata clear` |

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
[Spec Kit](https://github.com/github/spec-kit).

## Licença

[MIT](LICENSE)
