# Sobrevoo

Ferramenta de linha de comando pessoal e open source, em Go, que gera vídeos
de sobrevoo de trajetos a partir de arquivos de GPS.

## Status atual

Cinco etapas estão implementadas:

1. Leitura e tratamento de um trajeto GPX, exposta pelo comando `inspect`.
2. Registro local de dados geográficos (mapas base e relevo que você já
   baixou) e verificação de cobertura de um trajeto, expostos pelo grupo de
   comandos `geodata`.
3. Planejamento do movimento de câmera do vídeo de sobrevoo, exposto pelo
   comando `plan`.
4. Leitura do conteúdo dos dados geográficos registrados: o recorte de mapa
   base e relevo que um plano de câmera precisa, exposto por `geodata slice`,
   e a consulta de elevação de uma coordenada, por `geodata elevation`.
5. Desenho dos quadros do voo como imagens: o relevo em perspectiva, vestido
   com as peças do mapa base, o traçado e o marcador, exposto pelo grupo de
   comandos `render`.

Ainda não há geração de vídeo: os quadros desenhados são o que a etapa
seguinte vai juntar.

## Instalação

Requer Go 1.26+.

```sh
go install github.com/waliqueiroz/sobrevoo/cmd/sobrevoo@latest
```

Ou, a partir do código-fonte:

```sh
git clone git@github.com:waliqueiroz/sobrevoo.git
cd sobrevoo
make build   # gera ./bin/sobrevoo
```

## Uso

### `inspect`: tratar e resumir um trajeto

```sh
sobrevoo inspect <arquivo.gpx> [--simplification=low|medium|high] [--smoothing=low|medium|high]
```

Exemplo:

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

O programa lê o arquivo GPX, descarta pontos inválidos (coordenadas
impossíveis, duplicatas consecutivas, saltos fisicamente implausíveis),
reordena por tempo quando necessário, reduz e suaviza o traçado, e imprime um
resumo. Funciona 100% offline, para qualquer trajeto do planeta, inclusive
os que cruzam o meridiano de mudança de data.

| Flag | Valores | Padrão | Descrição |
|---|---|---|---|
| `--simplification` | `low`, `medium`, `high` | `medium` | Nível de redução de pontos do traçado |
| `--smoothing` | `low`, `medium`, `high` | `medium` | Nível de suavização do traçado |

Os códigos de saída de processo (arquivo vazio, formato não reconhecido,
pontos insuficientes, ...) estão documentados em
`specs/001-gps-track-processing/contracts/cli.md`.

### `geodata`: registrar e verificar dados geográficos

O Sobrevoo nunca embute dados de mapa nem de relevo: você baixa os arquivos
e os registra, e ele descobre sozinho o tipo e a área coberta pelo conteúdo
de cada um. Formatos reconhecidos: **MBTiles** (mapa base) e **GeoTIFF em
CRS geográfico, WGS84** (relevo). O registro fica em
`~/.sobrevoo/registry.json` e funciona 100% offline.

```sh
sobrevoo geodata register <arquivo> --name <nome>   # registra um mapa base ou relevo
sobrevoo geodata list                               # lista os registros
sobrevoo geodata remove <nome>                      # remove o registro (nunca o arquivo)
sobrevoo geodata check <arquivo.gpx>                # o trajeto está coberto?
```

Exemplo:

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

Um trajeto só conta como coberto quando há mapa base **e** relevo em toda a
sua extensão; `check` reporta o veredito (`full`, `partial` ou `none`) e os
subtrechos que faltam, com coordenadas. O comando sempre termina com código
`0`, mesmo quando o trajeto não está coberto. Se o arquivo de um registro for
movido ou apagado, `list` o marca com `(file not found)` em vez de
escondê-lo, e `check` deixa de contá-lo.

Os erros de `register` e `remove` têm código de saída próprio (`5` a `9`:
arquivo não encontrado, ilegível, formato não suportado, nome já em uso,
nome não registrado), documentados em
`specs/002-geo-data-registry/contracts/cli.md`.

### `plan`: planejar o movimento de câmera

```sh
sobrevoo plan <arquivo.gpx> [--duration <segundos>] [--fps <n>] \
    [--distance low|medium|high] [--tilt low|medium|high] \
    [--aspect <L:A>] [--export <plano.json>] [--overwrite]
```

A partir de um trajeto (tratado como em `inspect`), calcula o caminho que uma
câmera virtual percorre ao acompanhá-lo do início ao fim: para cada quadro do
futuro vídeo, onde a câmera está (latitude, longitude e altitude), para onde
aponta (direção e inclinação) e em que ponto do trajeto está o marcador da
atividade. O vídeo abre mostrando o trajeto inteiro, passa a acompanhar o
marcador e fecha mostrando o trajeto completo. O movimento é sempre suave,
inclusive em curvas fechadas, retornos e voltas no mesmo lugar; paradas longas
são comprimidas; e o mesmo trajeto com os mesmos parâmetros produz sempre o
mesmo plano, em qualquer lugar do planeta.

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
| `--duration` | segundos (até 3600) | automática | Duração do vídeo. Sem ela, é calculada a partir do comprimento do trajeto (de 20 s a 120 s) |
| `--fps` | 1 a 120 | `30` | Quadros por segundo |
| `--distance` | `low`, `medium`, `high` | `medium` | Quão longe a câmera fica do trajeto |
| `--tilt` | `low`, `medium`, `high` | `medium` | Quão de cima a câmera olha (`high` é quase vertical) |
| `--aspect` | `L:A` (`1:5` a `5:1`) | `9:16` | Proporção do vídeo: a abertura e o fechamento enquadram o trajeto inteiro por ela (`9:16` vertical, `16:9` horizontal) |
| `--export` | caminho | — | Grava o plano completo em JSON (formato em `specs/003-camera-path-planning/contracts/plan-file.md`) |
| `--overwrite` | — | — | Com `--export`, substitui um arquivo que já exista |

Sem `--export`, só o resumo é impresso e nada é gravado em disco. O arquivo
exportado nunca sobrescreve outro sem `--overwrite`. Os erros têm código de
saída próprio (`10` a `16`: duração ou taxa inválida, duração curta demais para
o trajeto, trajeto curto ou grande demais, destino da exportação já existente
ou inválido; e `39`, proporção do vídeo inválida), documentados em
`specs/003-camera-path-planning/contracts/cli.md`.

### `geodata slice` e `geodata elevation`: ler o conteúdo dos dados registrados

```sh
sobrevoo geodata slice <plano.json> [--export <recorte.zip>] [--overwrite]
sobrevoo geodata elevation --lat <graus> --lon <graus>
```

`slice` parte de um plano exportado por `plan --export` e reúne, dos arquivos
que você registrou com `geodata register`, o que aquele voo precisa: as
amostras de elevação do terreno e as peças do mapa base sob a área que a
câmera percorre, no nível de detalhe adequado à distância em que ela voa (o
resumo diz qual nível foi escolhido e por quê). Recusa, dizendo o que falta, se
a área do plano não estiver totalmente coberta por mapa base e relevo, e
recusa um recorte maior que 256 MiB. Peças que faltam num mapa registrado são
listadas e não interrompem o recorte; amostras que o arquivo de relevo não
informa ficam marcadas como sem valor, nunca como zero. Tudo é lido só dos
seus arquivos registrados (MBTiles e GeoTIFF), sem rede e sem alterá-los, e o
mesmo plano com os mesmos registros dá sempre o mesmo recorte, em qualquer
lugar do planeta.

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
```

O recorte exportado é um único arquivo ZIP (formato em
`specs/004-geo-data-slice/contracts/slice-file.md`), que nunca sobrescreve
outro sem `--overwrite`.

`elevation` informa a elevação, em metros, da célula do relevo registrado que
contém a coordenada (`Elevation: 1000.0 m`), ou diz que o arquivo não tem
valor para aquele ponto, para você conferir contra outra fonte. Passe as
coordenadas por `--lat` e `--lon` (aceitam valores negativos).

Os erros têm código de saída próprio (`17` a `26`: plano inválido ou de versão
desconhecida, área não coberta, recorte grande demais, dado ilegível ou em
unidade não suportada, destino da exportação já existente ou inválido,
coordenada sem cobertura ou inválida), documentados em
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
todos, em `frame_000000.png`, `frame_000001.png`, ..., na ordem do plano, com o
progresso na tela e um resumo ao final.

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

- **Sem inventar dado**: onde falta uma peça de mapa, o quadro mostra uma
  hachura cinza; onde o relevo não tem valor, um xadrez magenta; o resumo conta
  os quadros afetados. O fundo liso é o que está fora do recorte.
- **Determinismo**: o mesmo plano, o mesmo recorte e a mesma resolução dão
  sempre as mesmas imagens, byte a byte, seja qual for a ordem ou o número de
  quadros por execução.
- **Retomada e proteção**: `render all` continua de onde parou (é só repetir o
  comando; `Ctrl+C` encerra sem deixar imagem pela metade) e só redesenha os
  quadros que faltam. Um diretório com quadros de **outro** voo ou resolução é
  recusado, para não misturar; `--overwrite` refaz tudo e remove os quadros
  sobrando do voo anterior (só os que a ferramenta desenhou). `--resolution`
  aceita `LxA` par, de 180 a 3840 por lado (padrão `1080x1920`, vídeo vertical, a mesma
  proporção do `plan --aspect 9:16`; se você planejar em `16:9`, desenhe em
  resolução horizontal; com uma resolução mais estreita que o plano, o comando
  avisa em `stderr` que as laterais da abertura e do fechamento podem ser cortadas).
- **Só mapa em imagem**: peças de mapa base em imagem (PNG, JPG, WebP). Peças
  vetoriais (`pbf`) são recusadas com mensagem clara; desenhá-las fica para uma
  etapa futura.

O recorte precisa ter sido exportado por esta versão (ele guarda a
identificação do plano de que veio); um recorte antigo é recusado com a
orientação de gerá-lo de novo. Os erros têm código de saída próprio (`27` a
`38`: recorte inválido, de versão desconhecida, de outro plano, que não cobre o
plano, com peças vetoriais ou sem elevação; número de quadro ou resolução
inválidos; destino inválido, existente ou de outro conjunto; execução
interrompida), documentados em `specs/005-frame-rendering/contracts/cli.md`.

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
