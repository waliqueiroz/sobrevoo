# Guia de Validação Rápida: Planejamento do Movimento de Câmera

**Feature**: `003-camera-path-planning` | **Data**: 2026-09-19

Valida, de ponta a ponta e com o binário real, que `sobrevoo plan` cumpre
`spec.md` e os contratos em `contracts/`. Não substitui os testes
automatizados (que isolam cada camada, Princípio X) — é a verificação
manual/exploratória do resultado.

## Pré-requisitos

- Go 1.26 e `jq` instalados.
- Branch `003-camera-path-planning`.
- Nenhuma configuração: 100% offline (Princípio V), sem depender do
  registro da etapa 2.
- Arquivos GPX de teste (reais ou sintéticos), em `./amostras/`:
  `pedalada.gpx` (com horários, ~20 km), `sem-horario.gpx`,
  `com-parada.gpx` (uma parada de ~10 min), `retorno.gpx` (ida e volta
  pelo mesmo caminho), `voltas.gpx` (várias voltas num mesmo lugar),
  `antimeridiano.gpx` (cruza 180°), `polar.gpx` (latitude acima de 80°),
  `curto.gpx` (poucos metros), `enorme.gpx` (abrangência acima de 2 000 km) e, para a
  duração automática, `pequena.gpx` (~1 km), `longa.gpx` (~150 km) e
  `muito-longa.gpx` (abrangência de ~1 500 km).

```bash
go build -o bin/sobrevoo ./cmd/sobrevoo
```

## Cenário 1 — Plano padrão e resumo (História 1 e 4)

```bash
./bin/sobrevoo plan amostras/pedalada.gpx --export /tmp/plano.json
```

**Esperado**: código `0`; resumo com duração automática (entre 20 e 120 s,
marcada `(automatic)`), 30.0 fps, quantidade de quadros igual a duração ×
30, faixas de altitude e de distância, referência `clock` e a lista de
trechos suavizados (ou `Smoothed spans: none`); última linha
`Plan written to /tmp/plano.json`.

```bash
jq '.frames | length, .summary.frame_count, .summary.duration_mode' /tmp/plano.json   # iguais; "automatic"
jq '[.frames[].phase] | unique' /tmp/plano.json                     # opening, following, closing
jq '.frames[0].marker.distance_m, .frames[-1].marker.distance_m' /tmp/plano.json
```

## Cenário 2 — Determinismo (FR-014, SC-003)

```bash
for i in 1 2 3; do ./bin/sobrevoo plan amostras/pedalada.gpx --export /tmp/plano-$i.json --overwrite >/dev/null; done
shasum -a 256 /tmp/plano-1.json /tmp/plano-2.json /tmp/plano-3.json
```

**Esperado**: as três somas são idênticas.

## Cenário 3 — Movimento suave (FR-006/007, SC-002)

Para cada arquivo de `pedalada`, `retorno`, `voltas`, `antimeridiano` e
`polar`, gere o plano e confira o maior passo de rumo entre quadros
consecutivos (com o menor ângulo, tratando a volta 360°→0°):

```bash
./bin/sobrevoo plan amostras/retorno.gpx --export /tmp/retorno.json --overwrite
jq '[.frames[].heading_deg] as $h | [range(1; $h|length) | (($h[.] - $h[.-1] + 540) % 360 - 180 | fabs)] | max' /tmp/retorno.json
```

**Esperado**: o resultado é ≤ `45 / fps` (= 1,5° a 30 fps). O mesmo vale
para `tilt_deg` (≤ `30 / fps`) e para `ln(camera_to_marker_m)` (≤ `1.5 /
fps`). Em `retorno.gpx` e `voltas.gpx`, o rumo gira gradualmente no ponto
de retorno e não oscila a cada volta.

## Cenário 4 — Abertura e fechamento (FR-012)

```bash
jq '.frames[0], .frames[-1]' /tmp/plano.json
```

**Esperado**: `phase` `opening` no primeiro e `closing` no último quadro;
`tilt_deg` 60 e `camera_to_marker_m` grande (o trajeto inteiro enquadrado);
`heading_deg` do primeiro quadro igual ao do primeiro quadro `following`, e
o do último igual ao do último `following` (sem rotação na abertura nem no
fechamento); marcador em `distance_m` 0 no primeiro e no comprimento total
no último.

## Cenário 5 — Parâmetros (História 3)

```bash
./bin/sobrevoo plan amostras/pedalada.gpx --duration 60 --fps 30      # Frames: 1800, "(requested)"
./bin/sobrevoo plan amostras/pedalada.gpx --duration 48 --fps 24      # Frames: 1152
./bin/sobrevoo plan amostras/pedalada.gpx --distance low  --export /tmp/low.json  --overwrite
./bin/sobrevoo plan amostras/pedalada.gpx --distance high --export /tmp/high.json --overwrite
./bin/sobrevoo plan amostras/pedalada.gpx --tilt low  --export /tmp/tlow.json  --overwrite
./bin/sobrevoo plan amostras/pedalada.gpx --tilt high --export /tmp/thigh.json --overwrite
```

**Esperado**: `high` tem `camera_to_marker_m` maior que `low` em todos os
quadros da fase `following`; `--tilt high` tem `tilt_deg` maior (mais
vertical) que `--tilt low`.

## Cenário 5b — Duração automática (FR-003a, SC-011)

```bash
for f in pequena pedalada longa muito-longa; do ./bin/sobrevoo plan amostras/$f.gpx | grep -E 'Duration|Frames'; done
```

**Esperado**: em cada trajeto, `(automatic)`; duração entre 20 s e 120 s
(ou igual ao mínimo, se maior que 120 s), nunca recusada por curta demais;
duração não decrescente com o comprimento, e crescendo menos que o comprimento
(um trajeto 10× maior não gera vídeo 10× mais longo). Com `--duration`
informada, o resumo mostra `(requested)` e o valor exato pedido.

## Cenário 6 — Parada longa e trajeto sem horário (FR-009 a FR-011)

```bash
./bin/sobrevoo plan amostras/com-parada.gpx --export /tmp/parada.json --overwrite
./bin/sobrevoo plan amostras/sem-horario.gpx
```

**Esperado**: em `parada.json`, os quadros com `marker.distance_m`
praticamente constante (a parada) somam no máximo 5% dos quadros; em
`sem-horario.gpx`, o resumo mostra `Time reference: distance`.

## Cenário 7 — Erros (FR-016 a FR-018, FR-021, FR-023)

```bash
./bin/sobrevoo plan amostras/pedalada.gpx --duration 0;   echo $?    # 10
./bin/sobrevoo plan amostras/pedalada.gpx --fps 200;      echo $?    # 11 (informa 1-120)
./bin/sobrevoo plan amostras/pedalada.gpx --duration 5;   echo $?    # 12 (informa o mínimo)
./bin/sobrevoo plan amostras/curto.gpx;                   echo $?    # 13
./bin/sobrevoo plan amostras/enorme.gpx;                  echo $?    # 14
./bin/sobrevoo plan amostras/pedalada.gpx --export /tmp/plano.json; echo $?  # 15 (já existe)
./bin/sobrevoo plan amostras/pedalada.gpx --export /nao/existe/p.json; echo $? # 16
./bin/sobrevoo plan amostras/pedalada.gpx --duration abc; echo $?    # 2
./bin/sobrevoo plan amostras/pedalada.gpx --distance perto; echo $?  # 2 (lista os valores aceitos)
```

**Esperado**: cada mensagem nomeia o parâmetro ou o destino; após o código
`15`, o `/tmp/plano.json` original continua intacto (mesma soma de
verificação); após o `16`, nenhum arquivo parcial em `/nao/existe`. A
duração exatamente igual ao mínimo informado pelo código `12` é aceita
(código `0`).

## Cenário 8 — Antimeridiano e latitudes altas (FR-015, SC-007)

```bash
./bin/sobrevoo plan amostras/antimeridiano.gpx --export /tmp/am.json --overwrite
jq '[.frames[].camera.lon] | [min, max]' /tmp/am.json
```

**Esperado**: `0` de saída; nenhum salto de longitude entre quadros
consecutivos maior que o passo geométrico esperado (nunca ~360°);
mesmos limites de suavidade do cenário 3; resumo comparável ao de um
trajeto equivalente deslocado para outra região (tolerância de 1% em
distâncias).

## Cenário 9 — Desempenho (SC-001)

```bash
time ./bin/sobrevoo plan amostras/dez-mil-pontos.gpx
```

**Esperado**: menos de 5 segundos.
