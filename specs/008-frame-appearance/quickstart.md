# Quickstart: validação manual de Aparência Ajustável dos Quadros

**Feature**: `008-frame-appearance` | **Data**: 2026-09-28

Checklist manual, com o binário real, para conferir que as cinco flags de
aparência funcionam como a spec exige: mesmo efeito nos três comandos
(História 3), quadro isolado idêntico ao voo inteiro (História 2), retomada e
`fly --keep` nunca misturando aparências (História 4), e o padrão de hoje
preservado byte a byte para quem não informa nada (FR-003). Não há teste
automatizado de ponta a ponta; ver `plan.md`. Contrato:
[`contracts/appearance-flags.md`](./contracts/appearance-flags.md). Os valores
marcados **(anotar)** são conferidos na execução.

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

rm -rf /tmp/appearance && mkdir /tmp/appearance
sv plan $TRACK --duration 10 --fps 10 --export /tmp/appearance/plan.json
sv geodata slice /tmp/appearance/plan.json --export /tmp/appearance/slice.zip
PLAN=/tmp/appearance/plan.json
SLICE=/tmp/appearance/slice.zip
```

## 1. Sem nenhuma flag, o resultado é o de sempre (FR-003 / SC-002)

```sh
sv render frame $PLAN $SLICE --number 0 --output /tmp/appearance/padrao.png --resolution 360x640
```

**(anotar)**: abrir `/tmp/appearance/padrao.png` — traçado laranja, marcador
vermelho com anel branco, fundo escuro, exatamente como antes desta etapa.

## 2. Mudar cor e tamanho muda só o que foi pedido (História 1)

```sh
sv render frame $PLAN $SLICE --number 50 --output /tmp/appearance/traco.png --resolution 360x640 \
  --trail-color "#00FF00" --trail-width 0.02

sv render frame $PLAN $SLICE --number 50 --output /tmp/appearance/marcador.png --resolution 360x640 \
  --marker-color "#0000FF" --marker-radius 0.05

sv render frame $PLAN $SLICE --number 50 --output /tmp/appearance/fundo.png --resolution 360x640 \
  --background-color "#FFFFFF"
```

**(anotar)**: `traco.png` tem um traçado verde e visivelmente mais grosso, sem
nada mais mudado; `marcador.png` tem um marcador azul maior; `fundo.png` tem
qualquer área de fundo (fora do recorte, acima do horizonte) branca.

## 3. Quadro isolado idêntico ao voo inteiro (História 2 / SC-005)

```sh
sv render frame $PLAN $SLICE --number 50 --output /tmp/appearance/isolado.png --resolution 360x640 \
  --trail-color "#00FF00" --trail-width 0.02 --marker-color "#0000FF" --marker-radius 0.05 --background-color "#FFFFFF"

rm -rf /tmp/appearance/todos
sv render all $PLAN $SLICE --output /tmp/appearance/todos --resolution 360x640 \
  --trail-color "#00FF00" --trail-width 0.02 --marker-color "#0000FF" --marker-radius 0.05 --background-color "#FFFFFF"

cmp /tmp/appearance/isolado.png /tmp/appearance/todos/frame_000050.png && echo "IDÊNTICOS" || echo "DIFERENTES (anotar o diff)"
```

## 4. Mesmo valor, mesmo nome, mesmo efeito nos três comandos (História 3)

```sh
sv fly $TRACK --output /tmp/appearance/voo.mp4 --duration 10 --fps 10 --resolution 360x640 \
  --trail-color "#00FF00" --trail-width 0.02 --marker-color "#0000FF" --marker-radius 0.05 --background-color "#FFFFFF" \
  --keep /tmp/appearance/keep

cmp /tmp/appearance/isolado.png /tmp/appearance/keep/frames/frame_000050.png && echo "IDÊNTICOS" || echo "DIFERENTES (anotar o diff)"
```

## 5. Valor inválido recusa antes de desenhar (FR-004 / SC-004)

```sh
sv render frame $PLAN $SLICE --number 0 --output /tmp/appearance/x.png --trail-color "laranja"; echo "código: $?"        # esperado: 52, sem x.png
sv render frame $PLAN $SLICE --number 0 --output /tmp/appearance/x.png --trail-width 0.5; echo "código: $?"              # esperado: 53, sem x.png
sv render frame $PLAN $SLICE --number 0 --output /tmp/appearance/x.png --marker-radius -1; echo "código: $?"             # esperado: 54, sem x.png
sv render frame $PLAN $SLICE --number 0 --output /tmp/appearance/x.png --trail-width abc; echo "código: $?"              # esperado: 2 (erro de uso)
ls /tmp/appearance/x.png 2>&1                                                                                            # esperado: não existe
```

## 6. Retomada com aparência diferente recusa; com a mesma, reaproveita (História 4)

```sh
rm -rf /tmp/appearance/retomada
sv render all $PLAN $SLICE --output /tmp/appearance/retomada --resolution 360x640 --trail-color "#FF0000"

# aparência diferente: recusa
sv render all $PLAN $SLICE --output /tmp/appearance/retomada --resolution 360x640 --trail-color "#00FF00"; echo "código: $?"   # esperado: 37

# mesma aparência: reaproveita (nenhum quadro redesenhado)
sv render all $PLAN $SLICE --output /tmp/appearance/retomada --resolution 360x640 --trail-color "#FF0000"   # "N kept" == total
```

## 7. `fly --keep`: só aparência muda → quadros e vídeo refeitos, plano e recorte reaproveitados (FR-009)

```sh
sv fly $TRACK --output /tmp/appearance/voo2.mp4 --duration 10 --fps 10 --resolution 360x640 \
  --trail-color "#123456" --keep /tmp/appearance/keep2

md5=$(md5sum /tmp/appearance/keep2/plan.json /tmp/appearance/keep2/slice.zip)

sv fly $TRACK --output /tmp/appearance/voo3.mp4 --duration 10 --fps 10 --resolution 360x640 \
  --trail-color "#654321" --keep /tmp/appearance/keep2 --overwrite

md5 -c <<< "$md5" 2>/dev/null || echo "(conferir manualmente: plan.json e slice.zip inalterados; voo3.mp4 diferente de voo2.mp4; quadros redesenhados)"
```

**(anotar)**: `plan.json` e `slice.zip` continuam com o mesmo conteúdo antes e
depois (mesmo `md5`); `voo2.mp4` e `voo3.mp4` diferem (traçado de cor
diferente); a segunda execução leva sensivelmente menos tempo que a primeira
(pula planejamento e recorte).

## 8. Coincidência de cor com as marcações de falta é permitida (Caso Extremo)

```sh
sv render frame $PLAN $SLICE --number 0 --output /tmp/appearance/coincide.png --resolution 360x640 --marker-color "#C8C8C8"; echo "código: $?"   # esperado: 0
```

**(anotar)**: comando aceita normalmente; o hachurado de "sem mapa" continua
sendo desenhado nas mesmas duas tonalidades fixas de sempre, mesmo coincidindo
com a cor do marcador.
