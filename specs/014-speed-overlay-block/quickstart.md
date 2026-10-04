# Quickstart: validação manual do Bloco de Velocidade na Sobreposição

**Feature**: `014-speed-overlay-block` | **Data**: 2026-10-04

Checklist manual, com o binário real, para conferir que o bloco `speed`
funciona como o `spec.md` exige: estável e opt-in (História 1), e nunca
confundido entre planos com velocidades diferentes (História 2). Não há
teste automatizado de ponta a ponta; ver `plan.md`. Contratos:
[`contracts/plan-file-v3.md`](./contracts/plan-file-v3.md),
[`contracts/speed-overlay-block.md`](./contracts/speed-overlay-block.md).
Os valores marcados **(anotar)** são conferidos na execução.

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
NO_CLOCK=$G/sem-horario.gpx    # sem horário em nenhum ponto

rm -rf /tmp/speed && mkdir /tmp/speed
sv plan $TRACK --duration 10 --fps 10 --export /tmp/speed/plan.json
sv geodata slice /tmp/speed/plan.json --export /tmp/speed/slice.zip
PLAN=/tmp/speed/plan.json
SLICE=/tmp/speed/slice.zip
```

## 1. Sem pedir o bloco, nada muda (FR-007 / SC-002)

```sh
sv render frame $PLAN $SLICE --number 50 --output /tmp/speed/padrao.png --resolution 720x1280
```

**(anotar)**: `padrao.png` mostra só os quatro blocos de sempre (distância,
elevação/ganho, tempo decorrido, perfil) — nenhum número de velocidade,
exatamente como antes desta etapa.

## 2. Pedindo `speed`, o bloco aparece e é estável (FR-001/FR-002/FR-006/SC-001)

```sh
sv render frame $PLAN $SLICE --number 50 --output /tmp/speed/com-velocidade.png --resolution 720x1280 \
  --overlay-blocks distance,speed
for n in 48 49 50 51 52; do
  sv render frame $PLAN $SLICE --number $n --output /tmp/speed/v$n.png --resolution 720x1280 \
    --overlay-blocks speed
done
```

**(anotar)**: `com-velocidade.png` mostra distância e velocidade (`VEL
<n,n> km/h`); abrindo `v48.png`..`v52.png` em sequência, o valor muda pouco
de um quadro para o vizinho — nunca um salto de dezenas de km/h entre
quadros consecutivos, o defeito que uma velocidade instantânea teria.

## 3. Trajeto sem horário: o bloco não aparece, sem erro (FR-005/FR-008/SC-004)

```sh
sv plan $NO_CLOCK --duration 10 --fps 10 --export /tmp/speed/plan-sem-horario.json
sv geodata slice /tmp/speed/plan-sem-horario.json --export /tmp/speed/slice-sem-horario.zip
sv render frame /tmp/speed/plan-sem-horario.json /tmp/speed/slice-sem-horario.zip \
  --number 50 --output /tmp/speed/sem-horario.png --resolution 720x1280 \
  --overlay-blocks distance,speed
echo "código: $?"   # esperado: 0
```

**(anotar)**: comando termina com sucesso; `sem-horario.png` mostra a
distância, mas nenhum número de velocidade — o mesmo tratamento que o
bloco de tempo decorrido já recebe para este trajeto.

## 4. Nome de bloco inválido continua recusando (FR-014)

```sh
sv render frame $PLAN $SLICE --number 50 --output /tmp/speed/x.png --overlay-blocks velocidade
echo "código: $?"   # esperado: 55 (ErrInvalidOverlayBlock)
```

**(anotar)**: a mensagem de erro lista `speed` entre os nomes aceitos;
nenhum `x.png` é criado.

## 5. Plano de versão anterior é recusado (FR-012/FR-013/SC-005)

```sh
# Simula um plano da versão 2 (sem speed_mps), trocando format_version e
# removendo o campo novo de um quadro de verdade.
sed -e 's/"format_version": 3/"format_version": 2/' \
    -e 's/,"speed_mps":[0-9.]*}/}/' $PLAN > /tmp/speed/plan-v2.json

sv render frame /tmp/speed/plan-v2.json $SLICE --number 0 --output /tmp/speed/y.png
echo "código: $?"   # esperado: 18 (ErrPlanFormatVersionUnsupported)
```

**(anotar)**: a mensagem de erro cita a versão encontrada (`2`), a aceita
(`3`), e orienta a gerar o plano de novo; nenhum `y.png` é criado.

## 6. A velocidade participa da identidade do plano (FR-011/SC-006)

```sh
sv plan $TRACK --duration 10 --fps 10 --export /tmp/speed/plan-de-novo.json
diff <(jq '.frames[].marker.speed_mps' /tmp/speed/plan.json) \
     <(jq '.frames[].marker.speed_mps' /tmp/speed/plan-de-novo.json)
echo "código do diff: $?"   # esperado: 0 — mesmo trajeto, mesma velocidade por quadro
```

**(anotar)**: os dois planos têm exatamente os mesmos valores de
`speed_mps`, quadro a quadro — confirma que o cálculo é determinístico
(SC-003). A identidade diferente entre planos de velocidades diferentes
(FR-011) é coberta por teste automatizado de `CameraPlan.ID()`, não por
este quickstart (não há como, pela CLI, produzir dois planos do mesmo
trajeto com velocidades diferentes — a velocidade não é uma flag).
