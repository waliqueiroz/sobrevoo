# Contrato de CLI: `sobrevoo geodata slice` e `sobrevoo geodata elevation`

**Feature**: `004-geo-data-slice` | **Data**: 2026-09-20

Dois comandos novos, filhos de `geodata`, que expõem
`CameraPlanService.Load`, `GeoSliceService.Generate`/`Export` e
`GeoDataService.ElevationAt` (ver `data-model.md`) através de
`internal/infra/inbound/cli`. Não há API HTTP nem GUI (fora de escopo). Os
contratos de `inspect` (etapa 1), `geodata register|list|remove|check`
(etapa 2) e `plan` (etapa 3) permanecem inalterados. O formato do arquivo
exportado tem contrato próprio em [`slice-file.md`](./slice-file.md); o do
plano lido, em `specs/003-camera-path-planning/contracts/plan-file.md`.

## `sobrevoo geodata slice`

```text
sobrevoo geodata slice <arquivo-de-plano> [--export <caminho>] [--overwrite]
```

- `<arquivo-de-plano>` (posicional, obrigatório): plano de câmera exportado
  por `sobrevoo plan --export` (formato versionado; hoje `format_version` 1).
  O recorte é feito **exatamente** sobre o plano do arquivo; o trajeto GPS
  não é lido nem regerado (Clarificação de 2026-09-20).
- `--export` (opcional): grava o recorte completo neste caminho, um único
  arquivo no formato de `slice-file.md`. Sem esta flag, nada é gravado.
- `--overwrite` (opcional, só com `--export`): permite substituir um destino
  existente. Sem `--export`, é erro de uso (código `2`).
- Usa **somente** os registros de `geodata register`, sem rede e sem
  alterar nenhum arquivo de dados.

### Saída (sucesso)

Resumo legível em inglês, em `stdout`; rótulos e ordem estáveis (FR-012):

```text
Area: lat -23.6100 to -23.4800, lon -46.7200 to -46.5300
Base map detail: level 16 (ideal 19, source offers 0-16; above the source's maximum level)
  nearest camera distance 300.0 m at latitude 23.48, requiring 0.23 m/px per screen pixel
Map tiles: 1204 present, 3 missing
  missing: sp-osm level 16 x=24122 y=36870
  missing: sp-osm level 16 x=24122 y=36871
  missing: sp-osm level 16 x=24123 y=36870
Elevation samples: 486000 (1200 without value)
Elevation range: 712.0 m - 1204.0 m
Sources:
  sp-osm (base map, MBTiles)
  srtm-sp (elevation, GeoTIFF)
Size: 71.3 MiB
```

- `Area` são os limites da área de interesse; quando ela cruza o
  antimeridiano, a linha traz `lon 170.0 to -170.0 (crosses the antimeridian)`.
- `Base map detail` traz, **por registro de mapa base usado**, o nível
  escolhido, o ideal, o intervalo que o arquivo oferece e o motivo da
  limitação (`within the source's range`, `above the source's maximum level`
  ou `below the source's minimum level`); a linha seguinte explica o cálculo
  (distância mínima da câmera, latitude de referência, resolução exigida). Com
  mais de um registro, uma dupla de linhas por registro.
- `Map tiles`: `<n> present, <m> missing`. Com peças ausentes, cada uma é
  listada (`missing: <registro> level <z> x=<x> y=<y>`); acima de 20 peças
  ausentes, as 20 primeiras (em ordem) e uma linha `... and <k> more (all
  listed in the exported file)`. Nenhuma ausente: `Map tiles: <n> present`.
  Zero peças presentes: `Map tiles: 0 present, <m> missing (no imagery in
  this slice)`.
- `Elevation samples` mostra `(<k> without value)` só quando `k > 0`.
- `Elevation range` é calculada só sobre amostras com valor; quando nenhuma
  tem valor, a linha é `Elevation range: none (no sample has a value)`.
  Sempre em metros (FR-008).
- `Sources` lista cada registro usado, ordenado por nome, com tipo e formato.
- `Size` é o tamanho real do recorte (bytes das peças + 4 × amostras), em
  `KiB`/`MiB`/`GiB` com uma casa decimal.
- Com `--export`, uma última linha `Slice written to <caminho>`.

**Código de saída**: `0` (inclusive com peças ausentes ou amostras sem
valor: não são erros).

### Saída (erro)

Mensagem em `stderr`, sem resumo impresso (a exportação, quando pedida, é feita
antes de qualquer saída) e — em qualquer erro — sem arquivo criado ou
alterado no destino.

| Cenário | Erro do domínio | Código |
|---|---|---|
| Plano inexistente ou ilegível (E/S) | erro genérico | `4` |
| Formato de uso: argumento faltando, `--overwrite` sem `--export` | erro de uso da CLI | `2` |
| Arquivo não é um plano (JSON inválido, campos obrigatórios ausentes, truncado, incoerente) | `domain.ErrPlanFileInvalid` | `17` |
| `format_version` desconhecida | `domain.ErrPlanFormatVersionUnsupported` | `18` |
| Área do plano não totalmente coberta | `domain.ErrAreaNotCovered` | `19` |
| Recorte maior que o limite (estimado ou real) | `domain.ErrSliceTooLarge` | `20` |
| Registro ilegível/corrompido, ou com codificação não suportada | `domain.ErrGeoDataContentUnreadable` | `21` |
| Relevo em unidade que não é metro nem pé | `domain.ErrElevationUnitUnsupported` | `22` |
| Destino da exportação já existe (sem `--overwrite`) | `domain.ErrSliceDestinationExists` | `23` |
| Destino da exportação inválido (diretório inexistente, sem permissão) | `domain.ErrSliceDestinationInvalid` | `24` |

Mensagens (SC-010): a de `17` diz o que há de errado (campo ausente, ou a
incoerência: `frame_count is 1260 but duration × frame rate is 1230`); a de
`18`, a versão encontrada e as aceitas (`found 2, accepted: 1`); a de `19`,
um subtrecho por linha no mesmo formato de `geodata check`, com o tipo que
falta (`missing base map`, `missing elevation`, `missing base map and
elevation`) e as coordenadas dos **centros** da primeira e da última região
não coberta do subtrecho (precisão da região, `research.md` item 3); a de
`20`, o tamanho estimado (ou real), o limite (`256.0 MiB`), o nível efetivo,
a extensão da área e a dica `try a higher --distance in the plan, or a
shorter track`; a de `21`, o nome do registro e o motivo; a de `22`, o nome
do registro e a unidade encontrada; a de `23`, o caminho e a dica `use
--overwrite to replace it`.

## `sobrevoo geodata elevation`

```text
sobrevoo geodata elevation --lat <graus> --lon <graus>
```

- `--lat`, `--lon` (obrigatórias): coordenada em graus decimais. São
  **flags** (não posicionais) para aceitar valores negativos sem ambiguidade
  (`--lat -23.5505`). Latitude de -90 a 90; longitude de -180 a 180
  (`180` e `-180` são a mesma posição).
- Usa o relevo registrado escolhido pela mesma regra de todas as etapas
  (menor área; empate: o mais antigo), lendo **a célula da grade que contém a
  coordenada**, sem interpolar (`research.md` item 9).

### Saída (sucesso)

Três formas, todas com código `0`, em `stdout`:

```text
Elevation: 760.0 m
Source: srtm-sp (cell row 412, column 88)
```

```text
Elevation: no value (the file has no data for this point)
Source: srtm-sp (cell row 412, column 88)
```

O primeiro caso traz a elevação em metros, com uma casa decimal (a
conversão de pés, quando houver, é aplicada antes). O segundo é o "sem valor
no arquivo" (FR-009, FR-017): a ferramenta não inventa número. A linha
`Source` traz o registro e a célula (linha e coluna na grade do arquivo)
para conferência com outras ferramentas.

### Saída (erro)

| Cenário | Erro do domínio | Código |
|---|---|---|
| Nenhum relevo registrado cobre a coordenada | `domain.ErrElevationNotCovered` | `25` |
| `--lat`/`--lon` fora do intervalo | `domain.ErrInvalidCoordinate` | `26` |
| `--lat`/`--lon` ausente ou não numérico | erro de uso da CLI | `2` |
| Registro ilegível / codificação não suportada | `domain.ErrGeoDataContentUnreadable` | `21` |
| Relevo em unidade não suportada | `domain.ErrElevationUnitUnsupported` | `22` |

A mensagem de `26` aponta o valor e o intervalo aceito
(`latitude 91 is out of range, must be between -90 and 90`); a de `25` diz
que nenhum relevo registrado cobre o ponto (diferente do "sem valor no
arquivo"), e sugere `geodata list`.

## Exemplos

```bash
# Fluxo completo: plano exportado → recorte exportado
sobrevoo plan pedalada.gpx --export plano.json
sobrevoo geodata slice plano.json --export recorte.zip

# Só o resumo (nada é gravado)
sobrevoo geodata slice plano.json

# Regerar sobre o mesmo destino
sobrevoo geodata slice plano.json --export recorte.zip --overwrite

# Conferir a elevação de um ponto contra outra fonte
sobrevoo geodata elevation --lat -23.5505 --lon -46.6333

# Sobre o antimeridiano (mesma posição)
sobrevoo geodata elevation --lat -17.7 --lon 180
sobrevoo geodata elevation --lat -17.7 --lon -180
```
