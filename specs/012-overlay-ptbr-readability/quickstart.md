# Quickstart: validação manual dos Rótulos em Português e da Legibilidade do Texto

**Feature**: `012-overlay-ptbr-readability` | **Data**: 2026-10-03

Checklist manual, com o binário real, para conferir que os rótulos saem em
português e que os três defeitos de legibilidade da etapa anterior foram
corrigidos: contorno fino (História 2), fonte de peso forte (História 3) e
largura estável dos painéis (História 4). Não há teste automatizado de
ponta a ponta; ver `plan.md`. Contrato:
[`contracts/frame-files-change.md`](./contracts/frame-files-change.md). Os
itens marcados **(anotar)** são conferidos visualmente, por não haver uma
métrica automática de "legibilidade".

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

rm -rf /tmp/ptbr && mkdir /tmp/ptbr
sv plan $TRACK --duration 45 --fps 10 --export /tmp/ptbr/plan.json
sv geodata slice /tmp/ptbr/plan.json --export /tmp/ptbr/slice.zip
PLAN=/tmp/ptbr/plan.json
SLICE=/tmp/ptbr/slice.zip
```

Para conferir sobre uma imagem de satélite real (não obrigatório, mas é
onde a História 3 importa mais), use um plano/recorte gerados a partir de
dados próprios registrados em `resources/` no lugar das amostras acima.

## 1. Rótulos em português (História 1 / FR-001–FR-004 / SC-001)

```sh
sv render frame $PLAN $SLICE --number 200 --output /tmp/ptbr/rotulos.png --resolution 1080x1920
```

**(anotar)**: abrir `rotulos.png` — os três painéis numéricos mostram
`DIST`, `ELEV ... GANHO`, `TEMPO` (não `GAIN`/`TIME`); os valores e as
unidades (`km`, `m`) são exatamente os de antes desta etapa.

## 2. Contorno fino — vãos do "6"/"8"/"0" abertos (História 2 / FR-005 / SC-002)

```sh
sv render frame $PLAN $SLICE --number 1 --output /tmp/ptbr/contorno.png --resolution 1080x1920
```

**(anotar)**: ampliar `contorno.png` num editor de imagem, sobre um número
que contenha "6", "8" ou "0" (variar `--number` até achar um) — o vão
interno de cada um continua claramente aberto, e o halo de uma letra não
encosta no da vizinha; comparar com um quadro gerado antes desta etapa (se
ainda houver um salvo) para confirmar que o contorno ficou visivelmente
mais fino.

## 3. Fonte de peso forte sobre fundo claro (História 3 / FR-006 / SC-003)

```sh
sv render frame $PLAN $SLICE --number 200 --output /tmp/ptbr/peso.png --resolution 1080x1920 --background-color "#E8E4DA"
```

**(anotar)**: o texto em `peso.png` tem traços visivelmente mais
encorpados que antes desta etapa, e continua legível sobre o fundo claro
sem depender de aumentar a opacidade do painel.

```sh
go list -m all | grep golang.org/x/image   # (anotar) nenhuma dependência nova em relação a antes desta etapa
```

## 4. Largura dos painéis estável do início ao fim do voo (História 4 / FR-007 / FR-008 / SC-004)

```sh
sv render frame $PLAN $SLICE --number 5 --output /tmp/ptbr/inicio.png --resolution 1080x1920
FRAME_COUNT=$(grep -o '"frame_count": [0-9]*' $PLAN | grep -o '[0-9]*')
sv render frame $PLAN $SLICE --number $(( FRAME_COUNT - 5 )) --output /tmp/ptbr/fim.png --resolution 1080x1920

rm -rf /tmp/ptbr/todos
sv render all $PLAN $SLICE --output /tmp/ptbr/todos --resolution 1080x1920
```

**(anotar)**: em `inicio.png` e `fim.png`, os três painéis numéricos têm
exatamente a mesma largura entre si **e a mesma largura nos dois
arquivos**, mesmo com textos de tamanhos diferentes (ex.: a distância
crescendo ao longo do voo).

```sh
cmp /tmp/ptbr/inicio.png /tmp/ptbr/todos/frame_000005.png && echo "IDÊNTICOS" || echo "DIFERENTES (anotar o diff)"
```

**(anotar)**: esperado `IDÊNTICOS` — o quadro isolado (`render frame`) tem
a mesma largura de painel que o mesmo quadro dentro do voo inteiro
(`render all`), FR-008.

## 5. Determinismo byte a byte preservado (FR-011 / SC-006)

```sh
sv render frame $PLAN $SLICE --number 200 --output /tmp/ptbr/det1.png --resolution 1080x1920
sv render frame $PLAN $SLICE --number 200 --output /tmp/ptbr/det2.png --resolution 1080x1920 --overwrite

cmp /tmp/ptbr/det1.png /tmp/ptbr/det2.png && echo "IDÊNTICOS" || echo "DIFERENTES (falha — anotar o diff)"
```

## 6. Quadros de uma versão anterior nunca são reaproveitados (FR-010 / SC-005)

```sh
rm -rf /tmp/ptbr/antigo && mkdir /tmp/ptbr/antigo
sv render all $PLAN $SLICE --output /tmp/ptbr/antigo --resolution 360x640

# Com um binário compilado antes desta etapa (ex.: via `git stash` no
# código-fonte, rebuild, desenhar, `git stash pop`, rebuild de novo),
# desenhar os mesmos quadros em /tmp/ptbr/antigo e então rodar de novo com
# o binário desta etapa:

sv render all $PLAN $SLICE --output /tmp/ptbr/antigo --resolution 360x640; echo "código: $?"   # esperado: 37 (ErrFrameSetConflict)
sv render all $PLAN $SLICE --output /tmp/ptbr/antigo --resolution 360x640 --overwrite             # redesenha todos
```

**(anotar)**: sem `--overwrite`, a execução com o binário desta etapa
sobre quadros de antes dela recusa com o código `37`; com `--overwrite`,
redesenha todos com os rótulos em português e a legibilidade corrigida.

## 7. Nada muda quando as sobreposições estão desligadas

```sh
sv render frame $PLAN $SLICE --number 200 --output /tmp/ptbr/sem.png --resolution 1080x1920 --overlays=false
```

**(anotar)**: `sem.png` não tem nenhum texto, painel ou marcador de perfil
— exatamente como antes desta etapa.
