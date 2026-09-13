# Sobrevoo

Ferramenta de linha de comando pessoal e open source, em Go, que gera vídeos
de sobrevoo de trajetos a partir de arquivos de GPS.

## Status atual

Só a primeira etapa está implementada: leitura e tratamento de um trajeto
GPX, exposta pelo comando `inspect`. Ainda não há geração de mapa, câmera ou
vídeo.

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
