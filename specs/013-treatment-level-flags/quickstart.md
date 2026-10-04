# Quickstart: validação manual dos Níveis de Tratamento em `plan` e `fly`

**Feature**: `013-treatment-level-flags` | **Data**: 2026-10-03

Checklist manual, com o binário real, para conferir que `--simplification`/
`--smoothing` em `plan` e em `fly` têm o mesmo efeito que em `inspect`
(História 1 e 2), que o arquivo de plano exportado registra os dois níveis
(FR-007), que `fly --keep` recalcula quando um dos dois níveis muda e
reaproveita quando não muda (História 3), e que o padrão de hoje é
preservado byte a byte para quem não informa nada (SC-002). Não há teste
automatizado de ponta a ponta; ver `plan.md`. Contratos:
[`contracts/treatment-level-flags.md`](./contracts/treatment-level-flags.md),
[`contracts/plan-file-addendum.md`](./contracts/plan-file-addendum.md). Os
valores marcados **(anotar)** são conferidos na execução.

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

rm -rf /tmp/treatment && mkdir /tmp/treatment
```

## 1. `plan` reflete o mesmo traçado que `inspect` já mostra (História 1 / SC-001)

```sh
sv inspect $TRACK --simplification=high --smoothing=low
sv plan $TRACK --duration 10 --fps 10 --simplification=high --smoothing=low
sv plan $TRACK --duration 10 --fps 10   # nível padrão (medium/medium) nos dois
```

**(anotar)**: a contagem de pontos tratados de `inspect --simplification=high
--smoothing=low` e a geometria implícita no resumo do `plan` equivalente
(duração automática — se usada —, distâncias de câmera) condizem entre si;
o segundo `plan` (sem flags) produz um resumo diferente do primeiro sempre
que o trajeto tiver pontos suficientes para a simplificação `high` reduzir o
traçado de forma perceptível.

## 2. Sem nenhuma flag, o resultado é o de sempre (FR-004 / SC-002)

```sh
sv plan $TRACK --duration 10 --fps 10 --export /tmp/treatment/plan-padrao.json
sv plan $TRACK --duration 10 --fps 10 --simplification=medium --smoothing=medium --export /tmp/treatment/plan-explicito.json
cmp /tmp/treatment/plan-padrao.json /tmp/treatment/plan-explicito.json && echo "IDÊNTICOS" || echo "DIFERENTES (inesperado)"
```

**(anotar)**: os dois arquivos são idênticos byte a byte — informar o próprio
valor padrão não é diferente de omitir a flag.

## 3. O arquivo exportado registra os dois níveis (FR-007)

```sh
sv plan $TRACK --duration 10 --fps 10 --simplification=high --smoothing=low --export /tmp/treatment/plan-high-low.json
grep -o '"simplification": *"[a-z]*"' /tmp/treatment/plan-high-low.json
grep -o '"smoothing": *"[a-z]*"' /tmp/treatment/plan-high-low.json
```

**(anotar)**: esperado `"simplification": "high"` e `"smoothing": "low"`.

## 4. Dois planos com níveis diferentes nunca se confundem (FR-006)

```sh
cmp /tmp/treatment/plan-padrao.json /tmp/treatment/plan-high-low.json && echo "IGUAIS (inesperado)" || echo "DIFERENTES (esperado)"
```

## 5. `fly` aplica o tratamento pedido de ponta a ponta (História 2)

```sh
sv fly $TRACK --output /tmp/treatment/voo-high-low.mp4 --duration 10 --fps 10 --resolution 360x640 \
  --simplification=high --smoothing=low --keep /tmp/treatment/keep-a

grep -o '"simplification": *"[a-z]*"' /tmp/treatment/keep-a/plan.json
grep -o '"smoothing": *"[a-z]*"' /tmp/treatment/keep-a/plan.json
ls /tmp/treatment/voo-high-low.mp4
```

**(anotar)**: o plano guardado em `keep-a/plan.json` registra `high`/`low`;
`voo-high-low.mp4` existe.

## 6. Valor inválido recusa antes de qualquer etapa (FR-003 / SC-004)

```sh
sv plan $TRACK --simplification=extreme; echo "código: $?"                                          # esperado: 2, nenhuma saída de resumo
sv fly $TRACK --output /tmp/treatment/x.mp4 --smoothing=extreme; echo "código: $?"                   # esperado: 2, sem x.mp4
ls /tmp/treatment/x.mp4 2>&1                                                                          # esperado: não existe
```

## 7. `fly --keep`: mudar um nível recalcula; manter reaproveita (História 3)

```sh
sv fly $TRACK --output /tmp/treatment/voo1.mp4 --duration 10 --fps 10 --resolution 360x640 \
  --keep /tmp/treatment/keep-b   # níveis padrão

md5_plan_1=$(md5sum /tmp/treatment/keep-b/plan.json)
md5_slice_1=$(md5sum /tmp/treatment/keep-b/slice.zip)
md5_frame_1=$(md5sum /tmp/treatment/keep-b/frames/frame_000000.png)

# mesmos níveis de novo (explícitos, mas iguais ao padrão): reaproveita tudo
sv fly $TRACK --output /tmp/treatment/voo2.mp4 --duration 10 --fps 10 --resolution 360x640 \
  --simplification=medium --smoothing=medium --keep /tmp/treatment/keep-b --overwrite

md5sum -c <<< "$md5_plan_1" 2>/dev/null; md5sum -c <<< "$md5_slice_1" 2>/dev/null; md5sum -c <<< "$md5_frame_1" 2>/dev/null
# esperado: os três continuam batendo — plano, recorte e quadros reaproveitados

# nível diferente: NÃO reaproveita nada
sv fly $TRACK --output /tmp/treatment/voo3.mp4 --duration 10 --fps 10 --resolution 360x640 \
  --simplification=high --keep /tmp/treatment/keep-b --overwrite

md5_plan_2=$(md5sum /tmp/treatment/keep-b/plan.json)
[ "$md5_plan_1" != "$md5_plan_2" ] && echo "DIFERENTE (esperado — plano recalculado)" || echo "IGUAL (inesperado)"
```

**(anotar)**: o segundo bloco confirma reaproveitamento total (os três
`md5sum -c` batem); o terceiro confirma que só mudar `--simplification`
força um plano novo — e, por consequência (`EnsureMatches` já exige o mesmo
plano), um recorte e quadros novos também, mesmo sem nenhuma flag de
aparência, de sobreposição ou de seleção de fonte ter mudado.

## 8. `--keep` de antes desta etapa: tratado como nível padrão (Clarification de `spec.md`)

Simula um `plan.json` de antes desta etapa editando manualmente os dois
campos novos para fora do arquivo:

```sh
rm -rf /tmp/treatment/keep-c && mkdir -p /tmp/treatment/keep-c
sv fly $TRACK --output /tmp/treatment/voo4.mp4 --duration 10 --fps 10 --resolution 360x640 \
  --keep /tmp/treatment/keep-c   # níveis padrão, grava simplification/smoothing: "medium"

# remove os dois campos, simulando um arquivo anterior a esta etapa
python3 -c "
import json
with open('/tmp/treatment/keep-c/plan.json') as f:
    plan = json.load(f)
del plan['parameters']['simplification']
del plan['parameters']['smoothing']
with open('/tmp/treatment/keep-c/plan.json', 'w') as f:
    json.dump(plan, f)
"

md5_plan_old=$(md5sum /tmp/treatment/keep-c/plan.json)

# pedir o padrão de novo: deve reaproveitar (ausência == medium == padrão pedido)
sv fly $TRACK --output /tmp/treatment/voo5.mp4 --duration 10 --fps 10 --resolution 360x640 \
  --keep /tmp/treatment/keep-c --overwrite
md5_plan_reused=$(md5sum /tmp/treatment/keep-c/plan.json)
[ "$md5_plan_old" = "$md5_plan_reused" ] && echo "REAPROVEITADO (esperado)" || echo "RECALCULADO (inesperado)"
```

**(anotar)**: como o arquivo editado não tem o formato exato que `plan`
escreveria de volta (indentação, por exemplo), compare o conteúdo relevante
em vez do arquivo bruto se o `md5sum` não bater por motivo de formatação —
o que importa é que nenhuma etapa foi refeita (sem demora perceptível, sem
"Frames: N drawn" na saída). Repita pedindo `--simplification=high`: aí sim
deve recalcular.

## 9. `render frame`, `render all`, `video` e `geodata check` ficam fora (Suposições de `spec.md`)

```sh
sv geodata check $TRACK --simplification=high; echo "código: $?"   # esperado: 2 (flag desconhecida, erro de uso do Cobra)
```

**(anotar)**: `geodata check` nem reconhece a flag — confirma que ela não
foi acrescentada ali.
