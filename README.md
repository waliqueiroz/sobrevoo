# Sobrevoo

Ferramenta de linha de comando pessoal e open source, em Go, que gera vídeos
de sobrevoo de trajetos a partir de arquivos de GPS.

## Status atual

Duas etapas estão implementadas:

1. Leitura e tratamento de um trajeto GPX, exposta pelo comando `inspect`.
2. Registro local de dados geográficos (mapas base e relevo que você já
   baixou) e verificação de cobertura de um trajeto, expostos pelo grupo de
   comandos `geodata`.

Ainda não há geração de mapa, câmera ou vídeo.

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
