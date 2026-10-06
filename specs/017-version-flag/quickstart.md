# Quickstart: validação manual da Informação de Versão da CLI

**Feature**: `017-version-flag` | **Data**: 2026-10-05

Checklist manual, com o binário real, para conferir as três fontes de
versão e sua precedência (FR-005 a FR-008), que a saída é uma única linha
sem texto extra (FR-002), e que nenhum arquivo, registro ou rede é tocado
(FR-003). Não há teste automatizado de ponta a ponta para a instalação via
tag real (passo 3) — ver `plan.md`. Contrato:
[`contracts/version-flag.md`](./contracts/version-flag.md). Os valores
marcados **(anotar)** são conferidos na execução.

## Pré-requisitos

```sh
cd /Users/waliqueiroz/Documents/projetos/sobrevoo
MOD=github.com/waliqueiroz/sobrevoo
```

## 1. Compilação local, sem tag: indica honestamente "development" (História 3 / FR-006)

```sh
make build
./bin/sobrevoo --version
```

**(anotar)**: esperado algo como
`sobrevoo v0.0.0-20261006015100-c169b2b38ac9+dirty` — uma pseudo-versão
derivada do commit atual (o "VCS stamping" que o Go já faz por padrão
desde a 1.18, dentro de um checkout git), nunca um número de release
inventado; `+dirty` aparece porque a árvore de trabalho tem mudanças não
commitadas. Sem VCS disponível (`-buildvcs=false`, abaixo), o resultado é
o literal `(devel)` — os dois são igualmente honestos:

```sh
go build -buildvcs=false -o /tmp/sobrevoo-no-vcs ./cmd/sobrevoo
/tmp/sobrevoo-no-vcs --version   # esperado: sobrevoo (devel)
```

## 2. Uma única linha, sem banner nem texto extra (FR-002 / SC-005)

```sh
./bin/sobrevoo --version | wc -l    # esperado: 1
./bin/sobrevoo --version; echo "código: $?"   # esperado: código: 0
./bin/sobrevoo --version | awk '{print $1, $2}'   # esperado: igual à linha inteira (nada sobrou de fora)
```

## 3. Versão fixada no build tem precedência (História 4 / FR-007 / FR-008)

```sh
go build -ldflags "-X main.version=v9.9.9-quickstart" -o /tmp/sobrevoo-release ./cmd/sobrevoo
/tmp/sobrevoo-release --version
```

**(anotar)**: esperado `sobrevoo v9.9.9-quickstart` — mesma árvore de
trabalho sem tag do passo 1, mas a versão fixada no build prevalece sobre
a pseudo-versão (ou `(devel)`) que a ausência de tag produziria sozinha.

## 4. Nenhum arquivo, registro ou rede é tocado (FR-003)

```sh
SVHOME=$(mktemp -d)   # HOME vazio, sem ~/.sobrevoo/registry.json
HOME=$SVHOME ./bin/sobrevoo --version
ls $SVHOME/.sobrevoo 2>&1   # esperado: diretório não existe
```

**(anotar)**: esperado a mesma linha do passo 1, e nenhum `~/.sobrevoo`
criado — confirma que `--version` não tenta ler nem criar o registro de
dados geográficos.

## 5. Instalado a partir de uma tag real: informa exatamente essa tag (História 1 / FR-005)

Depende de uma tag já publicada neste repositório — rode depois do merge
desta feature, quando a próxima tag existir:

```sh
go install $MOD/cmd/sobrevoo@vX.Y.Z   # substitua vX.Y.Z pela tag publicada
sobrevoo --version
```

**(anotar)**: esperado `sobrevoo vX.Y.Z`, exatamente a tag usada no
`go install` — sem nenhuma flag de build envolvida neste passo.

## 6. Nenhum outro comando reconhece a flag (FR-009)

```sh
./bin/sobrevoo plan --version 2>&1; echo "código: $?"
```

**(anotar)**: comportamento fora de escopo desta etapa (Casos Extremos de
`spec.md`) — qualquer resultado aqui é aceitável, desde que nenhum outro
comando existente tenha mudado de comportamento para quem não usa
`--version`.

## 7. Testes automatizados dos três cenários de precedência

```sh
go test ./cmd/sobrevoo/... ./internal/infra/inbound/cli/... -run 'pickVersion|NewRootCommand' -v
```

**(anotar)**: os três cenários de `pickVersion` (versão de módulo, nenhuma
versão disponível, versão fixada no build com precedência) passam, junto
com o cenário novo de `root_test.go` que confere o texto de `--version`.
