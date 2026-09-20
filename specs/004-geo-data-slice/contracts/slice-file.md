# Contrato do Arquivo de Recorte Exportado

**Feature**: `004-geo-data-slice` | **Data**: 2026-09-20

É o contrato de dados que as etapas seguintes (desenho do terreno e do mapa,
renderização) e qualquer ferramenta externa consomem (FR-013). É **um único
arquivo ZIP**, gerado por `sobrevoo geodata slice --export`. O mesmo plano
com os mesmos registros (mesmo conteúdo de arquivo) produz um arquivo
idêntico byte a byte (SC-002) — o que exclui data/hora de geração, versão do
binário e qualquer dado do ambiente, ao contrário da procedência, que é dos
registros.

## Estrutura do ZIP

- Método de compressão *store* (sem compressão) em todas as entradas — as
  peças de mapa já são comprimidas e o método garante os mesmos bytes entre
  versões do compilador.
- Ordem das entradas fixa: `manifest.json`, depois `elevation/*` em ordem
  numérica, depois `tiles/*` em ordem `(registro, z, x, y)`.
- Data de modificação de todas as entradas fixa em `1980-01-01T00:00:00Z`;
  nenhum campo extra, nenhum comentário.

```text
recorte.zip
├── manifest.json
├── elevation/
│   ├── 000.f32
│   └── 001.f32
└── tiles/
    └── 000/                 # índice do registro em manifest.base_map[]
        └── 16/              # nível (z)
            └── 24121/       # x
                └── 36869.png   # y; extensão = manifest.base_map[].tile_format
```

## `manifest.json`

UTF-8, campos em ordem fixa, indentado com 2 espaços; cada item de
`tiles`, `missing` e `elevation` compacto, em uma linha.

```json
{
  "format_version": 1,
  "area": {
    "min_lat": -23.61, "max_lat": -23.48,
    "min_lon": -46.72, "max_lon": -46.53,
    "crosses_antimeridian": false
  },
  "summary": {
    "tile_count": 1204,
    "missing_tile_count": 3,
    "sample_count": 486000,
    "no_value_sample_count": 1200,
    "elevation_m": { "min": 712, "max": 1204 },
    "size_bytes": 74760192
  },
  "sources": [
    { "name": "sp-osm", "type": "base map", "format": "MBTiles", "path": "/dados/sp.mbtiles" },
    { "name": "srtm-sp", "type": "elevation", "format": "GeoTIFF", "path": "/dados/srtm.tif" }
  ],
  "base_map": [
    {
      "source": 0,
      "tile_format": "png",
      "level": { "ideal": 19, "chosen": 16, "min": 0, "max": 16, "reason": "above the source's maximum level" },
      "tiles": [
        {"x":24121,"y":36869,"path":"tiles/000/16/24121/36869.png","bytes":18234}
      ],
      "missing": [
        {"x":24122,"y":36870}
      ]
    }
  ],
  "elevation": [
    {"source":1,"file":"elevation/000.f32","rows":600,"cols":810,
     "north_lat":-23.48,"west_lon":-46.72,"cell_lat":0.000208333,"cell_lon":0.000208333,
     "no_value_count":1200}
  ]
}
```

| Campo | Tipo | Semântica |
|---|---|---|
| `format_version` | inteiro | Muda **somente** quando um campo é removido ou muda de significado; acrescentar campos não muda a versão. Esta etapa emite `1`. |
| `area.*` | número | Área de interesse, graus decimais, 7 casas. Com `crosses_antimeridian: true`, `min_lon > max_lon` e a área vai de `min_lon` até 180° e de −180° até `max_lon` (mesma convenção de `BoundingBox`). |
| `summary.*` | — | Mesmos valores do resumo impresso (`GeoSlice.Summary`), recalculáveis a partir do conteúdo. `elevation_m` é `null` quando nenhuma amostra tem valor. `size_bytes` é a soma dos bytes das peças mais `4 × sample_count`. |
| `sources[]` | lista | Procedência: cada registro usado, ordenado por `name` (índice = posição). Traz `path` exatamente como registrado. |
| `base_map[].source` | inteiro | Índice em `sources`. |
| `base_map[].tile_format` | texto | Extensão/formato das peças (`png`, `jpg`, `webp`, `pbf`). Os bytes das peças são os do arquivo de origem, sem reprocessar. |
| `base_map[].level` | objeto | `ideal` (calculado), `chosen` (usado), `min`/`max` (o que o registro oferece) e `reason` (texto). |
| `base_map[].tiles[]` | lista | Peças presentes, ordenadas por `(z, x, y)`, esquema **XYZ** (`y` cresce para o sul, origem no noroeste; a conversão do TMS do MBTiles já foi feita). `bytes` = tamanho da entrada. |
| `base_map[].missing[]` | lista | Peças requeridas que o registro **não** contém, ordenadas; sempre presente, `[]` quando não houve. Aqui não há imagem: a etapa seguinte sabe que ali não há. |
| `elevation[]` | lista | Uma grade por região com amostras, na ordem das regiões. `north_lat`/`west_lon` são o canto noroeste da célula `(0,0)` da grade; `cell_lat`/`cell_lon`, o tamanho da célula em graus (positivos). `no_value_count` = amostras sem valor. |

## `elevation/NNN.f32`

`rows × cols` amostras `float32` IEEE 754 **little-endian**, linha a linha
de **norte para sul** e, dentro da linha, de **oeste para leste**. A amostra
`(linha r, coluna c)` tem centro em
`lat = north_lat − (r + 0.5) · cell_lat` e
`lon = west_lon + (c + 0.5) · cell_lon` (normalizada para [−180, 180) — a
grade pode cruzar o antimeridiano, e então `lon` passa de 180 e volta a
−180). Valores em **metros**. **"Sem valor"** é o NaN silencioso
`0x7FC00000`; nenhum outro NaN aparece, nunca há zero no lugar de "sem
valor", e o consumidor deve testar NaN.

## Garantias verificáveis (usadas nos testes e no quickstart)

1. `summary.tile_count` = soma de `len(base_map[].tiles)` = número de
   entradas em `tiles/`; `summary.missing_tile_count` = soma de
   `len(base_map[].missing)`.
2. `summary.sample_count` = soma de `rows × cols` = soma de
   `tamanho(elevation/*.f32) / 4`; `no_value_sample_count` = soma de
   `no_value_count` = número de NaN nos `.f32`.
3. Toda amostra e toda peça aparece exatamente uma vez (regiões disjuntas).
4. `size_bytes` ≤ 268 435 456 (256 MiB).
5. `level.chosen ∈ [level.min, level.max]`.
6. O arquivo é idêntico byte a byte em duas execuções com as mesmas
   entradas.

## Compatibilidade

O arquivo não referencia data/hora de geração, nome de máquina nem versão
do binário — para preservar a igualdade byte a byte (FR-013). `sources[].path`
reflete o registro do usuário; se o registro mudar de caminho, o arquivo
muda, o que é o comportamento esperado ("mesmos registros"). Consumidores
devem ignorar campos desconhecidos.
