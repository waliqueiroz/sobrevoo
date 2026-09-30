# Quickstart: validação manual do Controle do Registro de Dados Geográficos

**Feature**: `010-geo-data-source-control` | **Data**: 2026-09-29

Checklist manual, com o binário real, para conferir as duas capacidades:
limpar o registro inteiro (História 1) e escolher explicitamente uma fonte
(Histórias 2 a 4). Não há teste automatizado de ponta a ponta; ver
`plan.md`. Contratos: [`contracts/registry-clear.md`](./contracts/registry-clear.md),
[`contracts/source-selection-flags.md`](./contracts/source-selection-flags.md).
Os valores marcados **(anotar)** são conferidos na execução.

Duas fontes de mapa base registradas sob nomes diferentes, mesmo arquivo,
bastam para forçar a seleção automática a decidir entre elas e para provar
que a escolha explícita muda qual delas é usada — não é preciso baixar dois
mapas de fato diferentes.

## Pré-requisitos

```sh
make build
BIN=$PWD/bin/sobrevoo
SVHOME=$(mktemp -d)
sv() { HOME=$SVHOME $BIN "$@"; }
A=$PWD/specs/005-frame-rendering/amostras
G=specs/003-camera-path-planning/amostras
go run ./test/samples --out $A
TRACK=$G/pedalada.gpx

rm -rf /tmp/sourcectl && mkdir /tmp/sourcectl
```

## 1. Limpar o registro (História 1)

```sh
sv geodata register $A/mapa-imagem-sp.mbtiles --name mapa-a
sv geodata register $A/mapa-imagem-sp.mbtiles --name mapa-b
sv geodata register $A/relevo-sp.tif --name relevo
sv geodata list   # esperado: 3 entradas

# sem --confirm: recusa, nada é removido
sv geodata clear; echo "código: $?"   # esperado: 57
sv geodata list   # esperado: ainda 3 entradas

# com --confirm: remove tudo
sv geodata clear --confirm; echo "código: $?"   # esperado: 0, "3 entries removed"
sv geodata list   # esperado: 0 entradas

ls $A/mapa-imagem-sp.mbtiles $A/relevo-sp.tif   # esperado: os dois arquivos continuam no disco
```

**(anotar)**: a recusa sem `--confirm` cita "3 entries"; depois de
`--confirm`, `geodata list` mostra o registro vazio e os dois arquivos
originais continuam intactos (SC-001, SC-002).

## 2. Limpar um registro já vazio não é erro (Cenário de Aceitação 3, História 1)

```sh
sv geodata clear --confirm; echo "código: $?"   # esperado: 0, "0 entries removed"
```

## 3. Registrar de novo, para a Parte 2

```sh
sv geodata register $A/mapa-imagem-sp.mbtiles --name mapa-a
sv geodata register $A/mapa-imagem-sp.mbtiles --name mapa-b
sv geodata register $A/relevo-sp.tif --name relevo
```

## 4. `geodata check` reflete a fonte pedida, não a automática (História 2)

```sh
sv geodata check $TRACK   # sem seleção: mostra qual das duas (mapa-a ou mapa-b) a seleção automática escolheu
sv geodata check $TRACK --base-map mapa-b   # "Base map sources used: mapa-b", sempre — mesmo se a automática preferisse mapa-a
sv geodata check $TRACK --base-map mapa-a --elevation relevo
```

**(anotar)**: a linha `Base map sources used:` da segunda chamada mostra
exatamente `mapa-b`, coincida ou não com o que a primeira chamada (sem
flag) mostrou.

## 5. Nome inexistente ou do tipo errado recusa cedo (História 3)

```sh
sv geodata check $TRACK --base-map nao-existe; echo "código: $?"        # esperado: 9
sv geodata check $TRACK --base-map relevo; echo "código: $?"            # esperado: 56 (relevo é elevação, não mapa base)
sv geodata check $TRACK --elevation mapa-a; echo "código: $?"           # esperado: 56 (mapa-a é mapa base, não elevação)
```

## 6. `geodata slice` usa exclusivamente a fonte pedida (História 2)

```sh
sv plan $TRACK --duration 10 --fps 10 --export /tmp/sourcectl/plan.json

sv geodata slice /tmp/sourcectl/plan.json --base-map mapa-a --export /tmp/sourcectl/slice-a.zip
sv geodata slice /tmp/sourcectl/plan.json --base-map mapa-b --export /tmp/sourcectl/slice-b.zip

grep -a "mapa-a" /tmp/sourcectl/slice-a.zip >/dev/null && echo "slice-a usa mapa-a"
grep -a "mapa-b" /tmp/sourcectl/slice-b.zip >/dev/null && echo "slice-b usa mapa-b"
cmp /tmp/sourcectl/slice-a.zip /tmp/sourcectl/slice-b.zip && echo "IGUAIS (inesperado)" || echo "DIFERENTES (esperado — só o nome da fonte muda)"
```

**(anotar)**: os dois arquivos diferem (o nome do registro usado faz parte
do manifesto do recorte, `sources[]`), mesmo vindo do mesmo arquivo de mapa.

## 7. Fonte explícita sem cobertura total recusa como hoje (História 3)

```sh
# elevação inexistente por nome, na geração do recorte
sv geodata slice /tmp/sourcectl/plan.json --elevation nao-existe; echo "código: $?"   # esperado: 9
```

## 8. `fly --keep`: trocar a fonte pedida nunca reaproveita o recorte em silêncio (História 4 / FR-011)

```sh
sv fly $TRACK --output /tmp/sourcectl/voo-a.mp4 --duration 10 --fps 10 \
  --base-map mapa-a --keep /tmp/sourcectl/keep

md5_a=$(md5sum /tmp/sourcectl/keep/slice.zip)

# mesma fonte de novo: reaproveita (recorte não muda, execução mais rápida)
sv fly $TRACK --output /tmp/sourcectl/voo-a2.mp4 --duration 10 --fps 10 \
  --base-map mapa-a --keep /tmp/sourcectl/keep --overwrite
md5 -c <<< "$md5_a" 2>/dev/null || echo "(conferir: slice.zip inalterado — reaproveitado)"

# fonte diferente: NÃO reaproveita, gera um recorte novo
sv fly $TRACK --output /tmp/sourcectl/voo-b.mp4 --duration 10 --fps 10 \
  --base-map mapa-b --keep /tmp/sourcectl/keep --overwrite
md5_b=$(md5sum /tmp/sourcectl/keep/slice.zip)
[ "$md5_a" != "$md5_b" ] && echo "DIFERENTE (esperado — recorte refeito para mapa-b)" || echo "IGUAL (inesperado)"

# de automática para explícita: também não reaproveita
sv fly $TRACK --output /tmp/sourcectl/voo-c.mp4 --duration 10 --fps 10 \
  --keep /tmp/sourcectl/keep2
sv fly $TRACK --output /tmp/sourcectl/voo-d.mp4 --duration 10 --fps 10 \
  --base-map mapa-a --keep /tmp/sourcectl/keep2 --overwrite
```

**(anotar)**: o segundo bloco confirma reaproveitamento (mesmo hash); o
terceiro confirma que pedir `mapa-b` depois de `mapa-a` força um recorte
novo (hash diferente); o quarto confirma que sair da seleção automática
para `--base-map mapa-a` também força um recorte novo, mesmo que a
automática já estivesse escolhendo `mapa-a` por acaso — a comparação é pela
procedência gravada, não por adivinhação.

## 9. Sem nenhuma flag nova, nada muda (SC-005 / FR-009)

```sh
sv geodata check $TRACK   # idêntico ao de antes desta etapa
sv geodata slice /tmp/sourcectl/plan.json --export /tmp/sourcectl/slice-auto.zip   # seleção automática, como sempre
```
