# Quickstart: validação manual do Recorte de Dados Geográficos

**Feature**: `004-geo-data-slice` | **Data**: 2026-09-20

Checklist manual, com o binário real, para conferir a etapa de ponta a ponta
(não há teste automatizado de ponta a ponta; ver `plan.md`). Contratos:
[`contracts/cli.md`](./contracts/cli.md) e
[`contracts/slice-file.md`](./contracts/slice-file.md). Os cenários usam
**dados de exemplo sintéticos** gerados em código (sem baixar nada), para que
os valores esperados sejam conhecidos; o item final indica como repetir com
dados reais.

## Pré-requisitos

Todos os comandos rodam a partir da **raiz do repositório**.

```sh
make build                      # gera bin/sobrevoo
SVHOME=$(mktemp -d)             # o registro fica em $HOME/.sobrevoo/registry.json
sv() { HOME=$SVHOME ./bin/sobrevoo "$@"; }   # registro isolado para este roteiro
go run ./test/samples --out specs/004-geo-data-slice/amostras
ls specs/004-geo-data-slice/amostras
```

`test/samples` (ferramenta de desenvolvimento, usa `test/helper`) escreve, em
`specs/004-geo-data-slice/amostras/`:

| Arquivo | O que é |
|---|---|
| `mapa-sp.mbtiles` | mapa base, níveis 10 a 16, sobre a área de `pedalada.gpx`, com **3 peças removidas** no nível 16 |
| `relevo-sp.tif` | GeoTIFF `int16`, Deflate, metros, com **valor de "sem dado"** em um bloco conhecido |
| `relevo-pes.tif` | GeoTIFF `float32`, sem compressão, unidade **pé** |
| `relevo-projetado.tif` | GeoTIFF cuja unidade vertical não é metro nem pé |
| `mapa-corrompido.mbtiles` | `metadata` com limites válidos e tabela de peças truncada |
| `mapa-antimeridiano.mbtiles`, `relevo-antimeridiano.tif` | dados que cruzam o meridiano de 180° |
| `mapa-polar.mbtiles`, `relevo-polar.tif` | dados acima de 80° de latitude |

Os trajetos GPX de `specs/003-camera-path-planning/amostras/` são reusados.

## 1. Recorte básico (História 1)

```sh
sv geodata register mapa specs/004-geo-data-slice/amostras/mapa-sp.mbtiles
sv geodata register relevo specs/004-geo-data-slice/amostras/relevo-sp.tif
sv plan specs/003-camera-path-planning/amostras/pedalada.gpx --export /tmp/plano.json
sv geodata slice /tmp/plano.json --export /tmp/recorte.zip; echo $?
```

Esperado: código `0`; resumo com `Area`, `Base map detail`, `Map tiles: … present,
3 missing` (com as três posições), `Elevation samples: … (N without value)`,
`Elevation range`, `Sources` e `Size`; `Slice written to /tmp/recorte.zip`.

```sh
unzip -l /tmp/recorte.zip                     # manifest.json, elevation/, tiles/
unzip -p /tmp/recorte.zip manifest.json | head -40
```

Conferir as garantias 1 a 5 de `slice-file.md` (contagens do `manifest.json`
batem com as entradas do ZIP; `level.chosen` dentro de `[min, max]`).

**Determinismo (SC-002)**:

```sh
sv geodata slice /tmp/plano.json --export /tmp/recorte2.zip
cmp /tmp/recorte.zip /tmp/recorte2.zip && echo IDENTICO
```

## 2. Entradas inválidas (História 1, cenários 5 a 8)

```sh
sv geodata slice /tmp/nao-existe.json; echo $?         # 4
echo '{"a":1}' > /tmp/x.json
sv geodata slice /tmp/x.json; echo $?                   # 17
sed 's/"format_version": 1/"format_version": 2/' /tmp/plano.json > /tmp/v2.json
sv geodata slice /tmp/v2.json; echo $?                  # 18 (found 2, accepted: 1)
sed 's/"frame_count": \([0-9]*\)/"frame_count": 7/' /tmp/plano.json > /tmp/incoerente.json
sv geodata slice /tmp/incoerente.json; echo $?          # 17, aponta frame_count
```

Nenhum deles deve criar arquivo nem imprimir resumo.

## 3. Área não coberta (História 1, cenários 3 e 4)

```sh
sv geodata remove relevo
sv geodata slice /tmp/plano.json; echo $?               # 19: missing elevation
sv geodata register relevo specs/004-geo-data-slice/amostras/relevo-sp.tif
mv specs/004-geo-data-slice/amostras/mapa-sp.mbtiles /tmp/mapa-sp.mbtiles                # arquivo some do caminho registrado
sv geodata slice /tmp/plano.json; echo $?               # 19: missing base map (registro ignorado)
mv /tmp/mapa-sp.mbtiles specs/004-geo-data-slice/amostras/mapa-sp.mbtiles
```

A mensagem lista os subtrechos com o tipo que falta e as coordenadas dos
centros das regiões, no formato de `geodata check`.

## 4. Nível de detalhe (História 2)

```sh
sv plan specs/003-camera-path-planning/amostras/pedalada.gpx --distance low  --export /tmp/perto.json
sv plan specs/003-camera-path-planning/amostras/pedalada.gpx --distance high --export /tmp/longe.json
sv geodata slice /tmp/perto.json | grep "Base map detail"
sv geodata slice /tmp/longe.json | grep "Base map detail"
```

Esperado: o nível de `perto` é maior ou igual ao de `longe`; ambos dentro de
`0-16` do arquivo; cada linha traz o motivo (`above the source's maximum
level`, `within the source's range` ou `below ...`).

## 5. Elevação e "sem valor" (Histórias 3 e 6)

```sh
# célula com valor conhecido, célula sem valor conhecido, ponto fora da cobertura
sv geodata elevation --lat <lat-conhecida> --lon <lon-conhecida>          # Elevation: <valor> m
sv geodata elevation --lat <lat-do-bloco-sem-dado> --lon <lon-do-bloco>   # Elevation: no value ...
sv geodata elevation --lat 0 --lon 0; echo $?                             # 25
sv geodata elevation --lat 91 --lon 0; echo $?                            # 26
sv geodata elevation --lat abc --lon 0; echo $?                           # 2
```

Os valores esperados de cada célula são fixos no gerador de amostras
(`test/samples`, comentários no topo do arquivo). O valor do resumo do
recorte (`Elevation range`) deve incluir o valor consultado; o número de
amostras sem valor no resumo é o do bloco conhecido.

**Consistência consulta × recorte (FR-018, SC-006)**: ler o `.f32` do
recorte na posição da célula consultada (`slice-file.md`) e comparar com a
resposta do comando: são iguais.

**Unidade (FR-008)**:

```sh
sv geodata register relevo-pes specs/004-geo-data-slice/amostras/relevo-pes.tif
sv geodata elevation --lat <lat-pes> --lon <lon-pes>     # valor em metros (pés × 0,3048)
sv geodata register relevo-x specs/004-geo-data-slice/amostras/relevo-projetado.tif
sv geodata elevation --lat <lat-x> --lon <lon-x>; echo $?  # 22
```

## 6. Peças ausentes e registro corrompido (História 5)

Peças ausentes já foram vistas no item 1 (3 peças, código `0`, posições no
resumo e em `missing[]` do manifesto). Registro corrompido
(`mapa-corrompido.mbtiles` tem `metadata` com limites válidos, então é
registrado, mas a tabela de peças está truncada):

```sh
sv geodata remove mapa
sv geodata register ruim specs/004-geo-data-slice/amostras/mapa-corrompido.mbtiles
sv geodata slice /tmp/plano.json; echo $?               # 21, nomeia "ruim"; nenhum recorte parcial
sv geodata remove ruim
sv geodata register mapa specs/004-geo-data-slice/amostras/mapa-sp.mbtiles
```

## 7. Recorte grande demais (História 5, cenários 3 e 4)

```sh
sv plan specs/003-camera-path-planning/amostras/longa.gpx --distance low --export /tmp/longo.json
time sv geodata slice /tmp/longo.json; echo $?          # 20 em < 5 s, sem ler conteúdo
```

A mensagem traz o tamanho estimado, o limite (`256.0 MiB`), o nível efetivo,
a extensão da área e a dica de `--distance` mais alto. O caso "exatamente no
limite é aceito" é coberto por teste de unidade (`SliceTuning.MaxSizeBytes`).

## 8. Exportação (História 4)

```sh
sv geodata slice /tmp/plano.json --export /tmp/recorte.zip; echo $?              # 23 (já existe), arquivo intacto
sv geodata slice /tmp/plano.json --export /tmp/recorte.zip --overwrite; echo $?  # 0
sv geodata slice /tmp/plano.json --export /tmp/nao-existe/recorte.zip; echo $?   # 24
sv geodata slice /tmp/plano.json --overwrite; echo $?                            # 2 (sem --export)
ls /tmp/.sobrevoo-slice-*.tmp 2>/dev/null                                                # nenhum resíduo
```

## 9. Antimeridiano e latitudes altas (FR-016, SC-009)

```sh
sv geodata register mapa-am specs/004-geo-data-slice/amostras/mapa-antimeridiano.mbtiles
sv geodata register relevo-am specs/004-geo-data-slice/amostras/relevo-antimeridiano.tif
sv plan specs/003-camera-path-planning/amostras/antimeridiano.gpx --export /tmp/am.json
sv geodata slice /tmp/am.json          # Area: ... (crosses the antimeridian); peças dos dois lados
sv geodata elevation --lat <lat> --lon 180  ; sv geodata elevation --lat <lat> --lon -180   # mesma resposta

sv geodata register mapa-polar specs/004-geo-data-slice/amostras/mapa-polar.mbtiles
sv geodata register relevo-polar specs/004-geo-data-slice/amostras/relevo-polar.tif
sv plan specs/003-camera-path-planning/amostras/polar.gpx --export /tmp/polar.json
sv geodata slice /tmp/polar.json       # sem erro, área contínua, nível de detalhe coerente
```

## 10. Com dados reais (uma vez)

```sh
sv geodata register meu-mapa  ~/dados/regiao.mbtiles
sv geodata register meu-dem   ~/dados/regiao-dem.tif      # GeoTIFF em coordenadas geográficas
sv plan minha-atividade.gpx --export plano.json
sv geodata slice plano.json --export recorte.zip
sv geodata elevation --lat <um ponto conhecido> --lon <...>   # comparar com um mapa topográfico
```

Esperado: nenhuma conexão de rede (conferir, se quiser, com o firewall ou
`nettop`/`lsof -i` durante a execução) e nenhum arquivo de dado alterado
(`shasum` antes e depois).
