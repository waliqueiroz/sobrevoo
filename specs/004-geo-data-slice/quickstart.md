# Quickstart: validação manual do Recorte de Dados Geográficos

**Feature**: `004-geo-data-slice` | **Data**: 2026-09-20

Checklist manual, com o binário real, para conferir a etapa de ponta a ponta
(não há teste automatizado de ponta a ponta; ver `plan.md`). Contratos:
[`contracts/cli.md`](./contracts/cli.md) e
[`contracts/slice-file.md`](./contracts/slice-file.md). Os cenários usam
**dados de exemplo sintéticos** gerados em código (sem baixar nada), para que
os valores esperados sejam conhecidos; o item final indica como repetir com
dados reais. Todos os resultados abaixo foram conferidos com o binário real.

## Pré-requisitos

Todos os comandos rodam a partir da **raiz do repositório** (zsh ou bash).

```sh
make build                                        # gera bin/sobrevoo
SVHOME=$(mktemp -d)                               # o registro fica em $HOME/.sobrevoo/registry.json
sv() { HOME=$SVHOME ./bin/sobrevoo "$@"; }        # registro isolado para este roteiro
A=$PWD/specs/004-geo-data-slice/amostras          # caminho absoluto: o registro guarda o caminho como dado
G=specs/003-camera-path-planning/amostras
go run ./test/samples --out $A                    # ~2 s; imprime as células conhecidas
```

`test/samples` (ferramenta de desenvolvimento, usa `test/helper`) escreve, em
`amostras/` (não versionada):

| Arquivo | O que é |
|---|---|
| `mapa-sp.mbtiles` | mapa base, níveis 10 a 16, sobre a área de `pedalada.gpx`, com **3 peças removidas** no nível 16 (a do início do trajeto e as a leste e a sul dela) |
| `relevo-sp.tif` | GeoTIFF `int16`, Deflate, metros; célula (linha r, coluna c) = `700 + (3r + 2c) mod 400`, com um bloco de "sem dado" nas linhas 240–249 e colunas 300–309 |
| `relevo-pes.tif` | GeoTIFF `float32`, sem compressão, unidade **pé**: toda célula vale 1000 pés (304,8 m) |
| `relevo-projetado.tif` | GeoTIFF cuja unidade vertical não é metro nem pé |
| `mapa-corrompido.mbtiles` | `metadata` com limites válidos e tabela de peças inutilizável |
| `plano-enorme.json` | plano sobre São Paulo cuja câmera chega a 300 m do chão e a 30 km do marcador: o recorte seria grande demais |
| `mapa-antimeridiano.mbtiles`, `relevo-antimeridiano.tif` | dados que cruzam o meridiano de 180° |
| `mapa-polar.mbtiles`, `relevo-polar.tif` | dados acima de 80° de latitude |

Os trajetos GPX de `specs/003-camera-path-planning/amostras/` são reusados.

## 1. Recorte básico (História 1)

```sh
sv geodata register $A/mapa-sp.mbtiles --name mapa
sv geodata register $A/relevo-sp.tif --name relevo
sv plan $G/pedalada.gpx --export /tmp/plano.json --overwrite
sv geodata slice /tmp/plano.json --export /tmp/recorte.zip --overwrite; echo $?
```

Esperado: código `0`; resumo com `Area`, `Base map detail (mapa)`, `Map tiles:
3779 present, 3 missing` (com as três posições: `level 16 x=24278 y=37181`,
`x=24278 y=37182` e `x=24279 y=37181`), `Elevation samples: 101505 (100
without value)`, `Elevation range: 700.0 m - 1099.0 m`, `Sources` e `Size:
466.6 KiB`; `Slice written to /tmp/recorte.zip`.

```sh
unzip -l /tmp/recorte.zip                     # manifest.json, elevation/000.f32, tiles/000/...
unzip -p /tmp/recorte.zip manifest.json | head -40
```

Conferir as garantias 1 a 5 de `slice-file.md` (contagens do `manifest.json`
batem com as entradas do ZIP; `level.chosen` dentro de `[min, max]`).

**Determinismo (SC-002)**:

```sh
rm -f /tmp/r1.zip /tmp/r2.zip
sv geodata slice /tmp/plano.json --export /tmp/r1.zip >/dev/null
sv geodata slice /tmp/plano.json --export /tmp/r2.zip >/dev/null
cmp /tmp/r1.zip /tmp/r2.zip && echo IDENTICO
```

## 2. Entradas inválidas (História 1, cenários 5 a 8)

```sh
sv geodata slice /tmp/nao-existe.json; echo $?          # 4  (reading the plan file: ... no such file)
echo '{"a":1}' > /tmp/x.json
sv geodata slice /tmp/x.json; echo $?                   # 17 (format_version is missing)
sed 's/"format_version": 1/"format_version": 2/' /tmp/plano.json > /tmp/v2.json
sv geodata slice /tmp/v2.json; echo $?                  # 18 (found 2, accepted: 1)
sed 's/"frame_count": [0-9]*/"frame_count": 7/' /tmp/plano.json > /tmp/incoerente.json
sv geodata slice /tmp/incoerente.json; echo $?          # 17 (summary.frame_count is 7 but the file lists 1260 frames)
```

Nenhum deles deve criar arquivo nem imprimir resumo.

## 3. Área não coberta (História 1, cenários 3 e 4)

```sh
sv geodata remove relevo
sv geodata slice /tmp/plano.json; echo $?               # 19: missing elevation from (...) to (...)
sv geodata register $A/relevo-sp.tif --name relevo
mv $A/mapa-sp.mbtiles /tmp/mapa-sp.mbtiles              # o arquivo some do caminho registrado
sv geodata slice /tmp/plano.json; echo $?               # 19: missing base map (registro ignorado)
mv /tmp/mapa-sp.mbtiles $A/mapa-sp.mbtiles
```

A mensagem lista os subtrechos com o tipo que falta e as coordenadas dos
**centros** das regiões não cobertas, no formato de `geodata check`.

## 4. Nível de detalhe (História 2)

```sh
sv plan $G/pedalada.gpx --distance low  --export /tmp/perto.json --overwrite >/dev/null
sv plan $G/pedalada.gpx --distance high --export /tmp/longe.json --overwrite >/dev/null
sv geodata slice /tmp/perto.json | grep -A1 "Base map detail"
sv geodata slice /tmp/longe.json | grep -A1 "Base map detail"
```

Esperado: `perto` → `level 16 (ideal 17, source offers 10-16; above the
source's maximum level)`; `longe` → `level 14 (ideal 14, source offers 10-16;
within the source's range)`. Cada uma traz a linha de explicação (distância
mínima da câmera, latitude de referência, resolução exigida).

## 5. Elevação e "sem valor" (Histórias 3 e 6)

```sh
sv geodata elevation --lat -23.3005 --lon -46.7995    # Elevation: 1000.0 m   (cell row 100, column 200)
sv geodata elevation --lat -23.6005 --lon -46.4995    # Elevation: 900.0 m    (cell row 400, column 500)
sv geodata elevation --lat -23.4455 --lon -46.6945    # Elevation: no value ... (cell row 245, column 305), código 0
sv geodata elevation --lat 0 --lon 0; echo $?         # 25
sv geodata elevation --lat 91 --lon 0; echo $?        # 26
sv geodata elevation --lat abc --lon 0; echo $?       # 2
```

O `Elevation range` do resumo do recorte (item 1) vai de 700 a 1099 m e o
número de amostras sem valor (100) é o do bloco 10 × 10 conhecido.

**Consistência consulta × recorte (FR-018, SC-006)**: com Python, ler o
`elevation/000.f32` de `/tmp/recorte.zip` na célula da consulta (a grade do
recorte tem `north_lat`, `west_lon` e `cell_lat`/`cell_lon` no manifesto) e
comparar com a resposta do comando: são iguais.

**Unidade (FR-008)**:

```sh
sv geodata register $A/relevo-pes.tif --name pes
sv geodata elevation --lat -23.5505 --lon -46.6333     # Elevation: 304.8 m (1000 pés × 0,3048), fonte "pes" (a menor área vence)
sv geodata register $A/relevo-projetado.tif --name x
sv geodata elevation --lat -23.5505 --lon -46.6333; echo $?   # 22 (nomeia "x")
sv geodata remove x; sv geodata remove pes
```

## 6. Peças ausentes e registro corrompido (História 5)

Peças ausentes já foram vistas no item 1 (3 peças, código `0`, posições no
resumo e em `missing[]` do manifesto). Registro corrompido
(`mapa-corrompido.mbtiles` tem `metadata` com limites válidos, então é
registrado, mas a tabela de peças é inutilizável):

```sh
sv geodata remove mapa
sv geodata register $A/mapa-corrompido.mbtiles --name ruim
sv geodata slice /tmp/plano.json; echo $?               # 21, nomeia "ruim"; nenhum recorte parcial
sv geodata remove ruim
sv geodata register $A/mapa-sp.mbtiles --name mapa
```

## 7. Recorte grande demais (História 5, cenários 3 e 4)

```sh
time sv geodata slice $A/plano-enorme.json; echo $?     # 20 em < 1 s, sem ler conteúdo
```

A mensagem traz o tamanho estimado (`730.2 MiB`), o limite (`256.0 MiB`), o
nível (`level 16`), a extensão da área (`60.0 km × 60.0 km`) e a dica de
`--distance` mais alto. O caso "exatamente no limite é aceito" é coberto por
teste de unidade (`SliceTuning.EnsureFits` e `Test_geoSliceService_Generate_Robustness`).

## 8. Exportação (História 4)

```sh
sv geodata slice /tmp/plano.json --export /tmp/recorte.zip; echo $?              # 23 (já existe), arquivo intacto
sv geodata slice /tmp/plano.json --export /tmp/recorte.zip --overwrite; echo $?  # 0
sv geodata slice /tmp/plano.json --export /tmp/nao-existe/recorte.zip; echo $?   # 24
sv geodata slice /tmp/plano.json --overwrite; echo $?                            # 2 (--overwrite requires --export)
ls /tmp/.sobrevoo-*.tmp 2>/dev/null                                              # nenhum resíduo
```

## 9. Antimeridiano e latitudes altas (FR-016, SC-009)

```sh
sv geodata register $A/mapa-antimeridiano.mbtiles --name mapa-am
sv geodata register $A/relevo-antimeridiano.tif --name relevo-am
sv plan $G/antimeridiano.gpx --export /tmp/am.json --overwrite >/dev/null
sv geodata slice /tmp/am.json          # Area: lat -16.7362 to -16.2638, lon 179.7509 to -179.6830 (crosses the antimeridian); 168 peças; nível 13
sv geodata elevation --lat -16.5 --lon 180; sv geodata elevation --lat -16.5 --lon -180   # 250.0 m, mesma célula (linha 250, coluna 500)

sv geodata register $A/mapa-polar.mbtiles --name mapa-polar
sv geodata register $A/relevo-polar.tif --name relevo-polar
sv plan $G/polar.gpx --export /tmp/polar.json --overwrite >/dev/null
sv geodata slice /tmp/polar.json       # Area: lat 81.8667 to 82.2345, lon 14.0422 to 16.9843; 288 peças; sem erro
```

## 10. Com dados reais (`resources/`)

A pasta `resources/` (não versionada) guarda o passeio real do autor e os dois
arquivos de dados reais; sempre que existirem, este item deve ser rodado:

```sh
R=$PWD/resources
sv geodata register "$R/planet_-40.036,-13.661_-38.086,-12.56.mbtiles" --name bbbike   # MBTiles vetorial (pbf) do BBBike
sv geodata register $R/dem-S14-W040.tif --name cop                                     # Copernicus DEM GLO-30 (float32, Deflate, predictor 3)
sv geodata check $R/passeio_bike_20260912.gpx                                          # Coverage: full
sv plan $R/passeio_bike_20260912.gpx --export /tmp/bike.json --overwrite
time sv geodata slice /tmp/bike.json --export /tmp/bike.zip --overwrite
sv geodata elevation --lat -13.4527 --lon -39.9438                                     # Elevation: 594.5 m (cell row 1630, column 202)
```

Esperado (conferido em 2026-09-20): 30 peças `pbf` no nível 14 (o máximo do
arquivo; o ideal seria 17), 113 886 amostras de elevação de 527,4 a 863,3 m,
0 sem valor, 470,8 KiB; o valor da consulta é o da mesma célula no
`elevation/000.f32` do recorte. Nenhuma conexão de rede e nenhum arquivo de
dado alterado (`shasum` antes e depois).

Achados que só os dados reais mostraram: o Copernicus grava a largura e a
altura do raster como `SHORT` (o inspetor da etapa 2 exigia `LONG`; corrigido);
e o `bounds` do BBBike vem quebrado (`-40.036,-13.661,0,0`); o inspetor agora
recorta os `bounds` declarados pela área onde há peças no nível mais detalhado,
e o registro passa a declarar lat [-13.661, -12.469], lon [-40.036, -38.057]
(a área real, no grão de uma peça).
