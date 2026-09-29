# Quickstart: validação manual de Sobreposições de Tela nos Quadros

**Feature**: `009-frame-overlays` | **Data**: 2026-09-28

Checklist manual, com o binário real, para conferir que as sobreposições
funcionam como o `spec.md` exige: ligadas por padrão (História 1), quadro
isolado idêntico ao voo inteiro (História 2), desligar tudo ou escolher
blocos (História 3), retomada e `fly --keep` nunca misturando configurações
(História 4), plano antigo recusado, e os valores acumulados batendo com
`inspect`. Não há teste automatizado de ponta a ponta; ver `plan.md`.
Contratos: [`contracts/overlay-flags.md`](./contracts/overlay-flags.md),
[`contracts/plan-file-v2.md`](./contracts/plan-file-v2.md). Os valores
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

rm -rf /tmp/overlays && mkdir /tmp/overlays
sv inspect $TRACK > /tmp/overlays/inspect.txt
cat /tmp/overlays/inspect.txt   # (anotar) distância total, ganho de elevação, duração
sv plan $TRACK --duration 10 --fps 10 --export /tmp/overlays/plan.json
sv geodata slice /tmp/overlays/plan.json --export /tmp/overlays/slice.zip
PLAN=/tmp/overlays/plan.json
SLICE=/tmp/overlays/slice.zip
```

## 1. Plano de versão anterior é recusado (FR-007 / SC-004)

```sh
# Simula um plano da versão 1 (sem os campos desta etapa), trocando só
# format_version por 1 num plano de verdade.
sed 's/"format_version": 2/"format_version": 1/' $PLAN > /tmp/overlays/plan-v1.json

sv render frame /tmp/overlays/plan-v1.json $SLICE --number 0 --output /tmp/overlays/x.png; echo "código: $?"   # esperado: 18 (ErrPlanFormatVersionUnsupported)
```

**(anotar)**: a mensagem de erro cita a versão encontrada, a aceita, e
orienta a gerar o plano de novo; nenhum `x.png` é criado.

## 2. Sem nenhuma flag, as sobreposições vêm ligadas (FR-002)

```sh
sv render frame $PLAN $SLICE --number 100 --output /tmp/overlays/padrao.png --resolution 720x1280
```

**(anotar)**: abrir `padrao.png` — os quatro blocos aparecem em posição
fixa: distância percorrida, elevação e ganho, tempo decorrido, e o perfil de
elevação com o marcador; tudo legível sobre o terreno.

## 3. Desligar tudo (História 3 / FR-003)

```sh
sv render frame $PLAN $SLICE --number 100 --output /tmp/overlays/sem.png --resolution 720x1280 --overlays=false
```

**(anotar)**: `sem.png` não tem nenhum texto nem gráfico por cima — só voo,
traçado e marcador, como antes desta etapa.

## 4. Escolher só alguns blocos (História 3 / FR-004)

```sh
sv render frame $PLAN $SLICE --number 100 --output /tmp/overlays/parcial.png --resolution 720x1280 \
  --overlay-blocks distance,time
```

**(anotar)**: `parcial.png` mostra só distância e tempo decorrido; sem
elevação/ganho nem perfil.

## 5. Bloco inválido recusa antes de desenhar (FR-017)

```sh
sv render frame $PLAN $SLICE --number 0 --output /tmp/overlays/x2.png --overlay-blocks distance,altura; echo "código: $?"   # esperado: 55
ls /tmp/overlays/x2.png 2>&1   # esperado: não existe
```

## 6. Quadro isolado idêntico ao voo inteiro (História 2 / SC-008)

```sh
sv render frame $PLAN $SLICE --number 100 --output /tmp/overlays/isolado.png --resolution 720x1280

rm -rf /tmp/overlays/todos
sv render all $PLAN $SLICE --output /tmp/overlays/todos --resolution 720x1280

cmp /tmp/overlays/isolado.png /tmp/overlays/todos/frame_000100.png && echo "IDÊNTICOS" || echo "DIFERENTES (anotar o diff)"
```

## 7. Os valores do último quadro coincidem com `inspect` (FR-014 / SC-005)

```sh
FRAME_COUNT=$(grep -o '"frame_count": [0-9]*' $PLAN | grep -o '[0-9]*')
sv render frame $PLAN $SLICE --number $(( FRAME_COUNT - 1 )) --output /tmp/overlays/ultimo.png --resolution 720x1280
```

**(anotar)**: abrir `ultimo.png` e comparar, a olho, a distância percorrida,
o ganho de elevação e o tempo decorrido mostrados com os três números
anotados de `inspect.txt` no início — devem ser exatamente os mesmos
(mesmas unidades, mesmo arredondamento de exibição).

## 8. Retomada com configuração diferente recusa; com a mesma, reaproveita (História 4)

```sh
rm -rf /tmp/overlays/retomada
sv render all $PLAN $SLICE --output /tmp/overlays/retomada --resolution 360x640

# configuração diferente: recusa
sv render all $PLAN $SLICE --output /tmp/overlays/retomada --resolution 360x640 --overlays=false; echo "código: $?"   # esperado: 37

# mesma configuração: reaproveita (nenhum quadro redesenhado)
sv render all $PLAN $SLICE --output /tmp/overlays/retomada --resolution 360x640   # "N kept" == total
```

## 9. `fly --keep`: só sobreposição muda → quadros e vídeo refeitos, plano e recorte reaproveitados

```sh
sv fly $TRACK --output /tmp/overlays/voo2.mp4 --duration 10 --fps 10 --resolution 360x640 \
  --keep /tmp/overlays/keep2

md5=$(md5sum /tmp/overlays/keep2/plan.json /tmp/overlays/keep2/slice.zip)

sv fly $TRACK --output /tmp/overlays/voo3.mp4 --duration 10 --fps 10 --resolution 360x640 \
  --overlays=false --keep /tmp/overlays/keep2 --overwrite

md5 -c <<< "$md5" 2>/dev/null || echo "(conferir manualmente: plan.json e slice.zip inalterados; voo3.mp4 diferente de voo2.mp4)"
```

**(anotar)**: `plan.json` e `slice.zip` continuam com o mesmo conteúdo antes
e depois; `voo2.mp4` tem sobreposições, `voo3.mp4` não; a segunda execução
leva sensivelmente menos tempo que a primeira (pula planejamento e recorte).

## 10. Margem de segurança em vídeo vertical (FR-009 / SC-006)

```sh
sv render frame $PLAN $SLICE --number 100 --output /tmp/overlays/vertical.png --resolution 1080x1920
```

**(anotar)**: nenhum texto ou traço do perfil toca ou ultrapassa a margem
documentada nas quatro bordas de `vertical.png` — simular o corte de borda
comum de uma rede social (recortar ~10% de cada lado) e conferir que nada
essencial é cortado.
