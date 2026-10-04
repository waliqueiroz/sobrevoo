# Quickstart: validação manual do Redesenho das Sobreposições de Tela

**Feature**: `015-overlay-redesign` | **Data**: 2026-10-04

Checklist manual, com o binário real, para conferir que o novo arranjo em
colunas, o bloco `gain` independente e a nova escolha padrão funcionam como
o `spec.md` exige. Não há teste automatizado de ponta a ponta; ver
`plan.md`. Contratos: [`contracts/overlay-blocks-update.md`](./contracts/overlay-blocks-update.md),
[`contracts/frame-files-change.md`](./contracts/frame-files-change.md). Os
quadros gerados precisam ser **abertos visualmente** — nenhum dos itens
abaixo é verificável só pelo código de saída. Os valores marcados
**(observar)** são conferidos olhando a imagem.

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
TRACK=$G/pedalada.gpx          # tem horário e elevação em todo ponto

rm -rf /tmp/overlay-redesign && mkdir /tmp/overlay-redesign
sv plan $TRACK --duration 10 --fps 10 --export /tmp/overlay-redesign/plan.json
sv geodata slice /tmp/overlay-redesign/plan.json --export /tmp/overlay-redesign/slice.zip
PLAN=/tmp/overlay-redesign/plan.json
SLICE=/tmp/overlay-redesign/slice.zip
```

## 1. Sem painel: números e gráfico direto sobre a imagem (FR-001/FR-012/SC-001)

```sh
sv render frame $PLAN $SLICE --number 50 --output /tmp/overlay-redesign/sem-painel.png \
  --resolution 720x1280 --overlay-blocks distance,elevation,gain,time,speed,profile
```

**(observar)** `sem-painel.png`: nenhum bloco numérico e nenhuma parte do
gráfico de elevação tem um retângulo ou faixa de fundo — o terreno/mapa
aparece inteiro por baixo de cada número; a linha e o marcador do gráfico
de elevação se destacam do terreno por um contorno escuro, não por um
painel.

## 2. Lado a lado, em colunas, ordem fixa (FR-002/FR-003/FR-004/SC-002)

```sh
sv render frame $PLAN $SLICE --number 50 --output /tmp/overlay-redesign/ordem-a.png \
  --resolution 720x1280 --overlay-blocks time,distance,speed
sv render frame $PLAN $SLICE --number 50 --output /tmp/overlay-redesign/ordem-b.png \
  --resolution 720x1280 --overlay-blocks speed,distance,time
diff /tmp/overlay-redesign/ordem-a.png /tmp/overlay-redesign/ordem-b.png
echo "código do diff: $?"   # esperado: 0 — pixels idênticos
```

**(observar)** `ordem-a.png`/`ordem-b.png`: os três blocos aparecem lado a
lado numa única faixa no alto do quadro (nunca empilhados), cada um numa
coluna da mesma largura, sempre na ordem velocidade → distância → tempo
decorrido — a mesma em ambos os arquivos, apesar da ordem diferente
pedida em `--overlay-blocks`. O `diff` confirma isso sem depender do olho.

## 3. Três alturas — rótulo, valor, unidade (FR-005/FR-006)

```sh
sv render frame $PLAN $SLICE --number 50 --output /tmp/overlay-redesign/alturas.png \
  --resolution 720x1280 --overlay-blocks distance,time
```

**(observar)** `alturas.png`: o bloco de distância mostra, de cima para
baixo, "Distância" (pequeno), o número (bem maior), e a unidade (`m` ou
`km`, pequeno) — três linhas. O bloco de tempo decorrido mostra só
"Tempo decorrido" e o valor (`H:MM:SS`) — duas linhas, sem nenhuma terceira
linha vazia ou espaço sobrando onde ela estaria.

## 4. `gain` independente de `elevation` (FR-008/FR-009/FR-010/SC-003)

```sh
sv render frame $PLAN $SLICE --number 50 --output /tmp/overlay-redesign/so-elevacao.png \
  --resolution 720x1280 --overlay-blocks elevation
sv render frame $PLAN $SLICE --number 50 --output /tmp/overlay-redesign/so-ganho.png \
  --resolution 720x1280 --overlay-blocks gain
sv render frame $PLAN $SLICE --number 50 --output /tmp/overlay-redesign/os-dois.png \
  --resolution 720x1280 --overlay-blocks elevation,gain
```

**(observar)** `so-elevacao.png`: bloco "Elevação" com a altitude, nenhum
número de ganho. `so-ganho.png`: bloco "Ganho" com o ganho acumulado,
nenhuma altitude. `os-dois.png`: os dois aparecem, cada um na sua coluna.

```sh
sv render frame $PLAN $SLICE --number 50 --output /tmp/overlay-redesign/x.png \
  --overlay-blocks velocidade
echo "código: $?"   # esperado: 55 (ErrInvalidOverlayBlock)
```

**(observar)** a mensagem de erro lista os seis nomes aceitos, incluindo
`gain`; nenhum `x.png` é criado.

## 5. Nova escolha padrão (FR-011/SC-004)

```sh
sv render frame $PLAN $SLICE --number 50 --output /tmp/overlay-redesign/padrao.png \
  --resolution 720x1280
```

**(observar)** `padrao.png` mostra exatamente velocidade, elevação e
distância na faixa do alto, e o gráfico de elevação no rodapé — nenhum
bloco de tempo decorrido nem de ganho, sem pedir nada além de desenhar o
quadro (as sobreposições já ligam por padrão).

## 6. Quadros de antes desta etapa nunca se juntam aos de depois (FR-016/SC-008)

```sh
rm -rf /tmp/overlay-redesign/frames
sv render all $PLAN $SLICE --output /tmp/overlay-redesign/frames --resolution 480x854
# simula um quadro "antigo" apagando um e trocando a versão esperada não é
# possível sem recompilar; na prática, qualquer binário anterior a esta
# etapa que já tenha desenhado em /tmp/overlay-redesign/frames é suficiente
# para reproduzir a recusa abaixo. Quando não houver um binário anterior à
# mão, confirme ao menos que o diretório atual é aceito sem --overwrite na
# segunda chamada (mesma versão, mesmo conjunto):
sv render all $PLAN $SLICE --output /tmp/overlay-redesign/frames --resolution 480x854
echo "código: $?"   # esperado: 0 — retomada do mesmo conjunto, nada a redesenhar
```

**(observar)** a segunda chamada termina com sucesso, sem recusar nada
(mesmo `RenderVersion`, mesmo `OverlayConfig`); se houver, à mão, um
diretório de quadros desenhado por um binário anterior a esta etapa,
rodar o comando acima sobre ele recusa com `ErrFrameSetConflict` sem
`--overwrite`.

## 7. Determinismo byte a byte (SC-007)

```sh
sv render frame $PLAN $SLICE --number 120 --output /tmp/overlay-redesign/det-a.png --resolution 720x1280
sv render frame $PLAN $SLICE --number 120 --output /tmp/overlay-redesign/det-b.png --resolution 720x1280
diff /tmp/overlay-redesign/det-a.png /tmp/overlay-redesign/det-b.png
echo "código do diff: $?"   # esperado: 0
```

**(observar)** os dois arquivos são idênticos — confirma que o mesmo
plano, recorte, resolução, aparência e configuração de sobreposição
sempre produzem a mesma imagem.
