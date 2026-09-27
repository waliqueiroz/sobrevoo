# Modelo de Dados: Desenho dos Quadros do Voo

**Feature**: `005-frame-rendering` | **Data**: 2026-09-26

Todos os tipos abaixo vivem em `internal/domain` (Princípio IX: DTOs de saída
não triviais e regra de negócio são do domínio, nunca de
`internal/application`). Nomes de campos e tipos em inglês, como no restante
do código. O núcleo não persiste nada: o que é gravado (imagens PNG) sai por
portas. Os tipos que já existem (`CameraPlan`, `CameraFrame`, `GeoSlice`,
`ElevationGrid`, `TileSet`, `BoundingBox`, `SliceTuning`) mudam **só** onde
indicado como "acréscimo".

## Entidades existentes, com acréscimos

### `CameraPlan` (`camera_plan.go`)

| Método novo | Regra |
|---|---|
| `ID() string` | SHA-256 hex (64 caracteres) da codificação canônica do plano: parâmetros efetivos e, por quadro, `index`, `phase` e valores quantizados aos passos do plano, como inteiros de 64 bits em ordem de bytes fixa. Igual para o mesmo conteúdo, venha da memória ou de arquivo (`research.md` item 14). |

### `GeoSlice` (`geo_slice.go`)

| Campo novo | Tipo | Observação |
|---|---|---|
| `PlanID` | `string` | `plan.ID()` do plano de que o recorte veio; gravado no manifesto por `GeoSliceService.Generate`/o exportador e lido de volta pelo leitor. |
| `ContentID` | `string` | SHA-256 hex do arquivo do recorte; preenchido **só** pelo leitor (o recorte recém-gerado não tem arquivo); não é exportado. Identifica "este recorte" no conjunto de quadros. |

| Método novo | Regra |
|---|---|
| `EnsureMatches(plan CameraPlan) error` | `ErrSliceDoesNotMatchPlan` quando `PlanID != plan.ID()`; a mensagem cita as duas identificações abreviadas (12 caracteres). |
| `EnsureCovers(plan CameraPlan, tuning SliceTuning) error` | `ErrSliceDoesNotCoverPlan` quando `Area` não contém `plan.AreaOfInterest(tuning)`; a mensagem compara as duas áreas. |
| `EnsureDrawable() error` | `ErrTileFormatUnsupported` para conjunto de peças com formato fora de `png`/`jpg`/`webp` (mensagem própria para `pbf`/`mvt`: "vector tiles are not drawn yet"); `ErrNoElevationData` quando `SampleCount == NoValueSampleCount`. Cada uma cita o registro e o formato. |

**Porta** (no topo de `geo_slice.go`, ao lado de `GeoSliceExporter`):

```go
type GeoSliceReader interface {
    // Read reads the slice file at path, checks it is coherent with itself and
    // returns it with PlanID and ContentID set. ErrSliceFileInvalid and
    // ErrSliceFormatVersionUnsupported are the sentinels; I/O errors opening
    // or reading the file come back wrapped with no sentinel.
    Read(path string) (GeoSlice, error)
}
```

### `BoundingBox` (`bounding_box.go`)

| Método novo | Regra |
|---|---|
| `ContainsBox(other BoundingBox) bool` | `other` está inteira dentro de `b`, com o desembrulho de longitude de `longitudeSpans` (cobre o antimeridiano). |

## Entidades novas

### `Resolution` (`frame_resolution.go`)

| Campo | Tipo | Regra |
|---|---|---|
| `Width`, `Height` | `int` | pares; cada um de 180 a 3840; `Width × Height ≤ 8 294 400` |

Constantes: `MinFrameSide = 180`, `MaxFrameSide = 3840`, `MaxFramePixels =
8_294_400`. `NewResolution(w, h int) (Resolution, error)` e
`ParseResolution(text string) (Resolution, error)` (`"1080x1920"`, maiúscula ou
minúscula) falham com `ErrInvalidResolution`, dizendo o valor recebido e os
limites. `Pixels() int`. O padrão (1080 × 1920) é dado da configuração.

### `RenderTuning` (`render_tuning.go`)

Constantes de ajuste injetadas (Princípio VIII): `VerticalFOVDegrees`,
`MinCameraClearanceMeters`, `MinTiltForTargetDegrees`, `TrailLiftMeters`,
`DepthBiasMeters`, `DepthBiasRatio`, `TileCacheBytes`, `Workers`
(`research.md` item 22). `Fingerprint() string` devolve uma cadeia canônica dos
campos que afetam a aparência (todos menos `TileCacheBytes` e `Workers`).

Constantes de domínio (o que a imagem significa; `contracts/frame-files.md`):
`RenderVersion = 1`; as cores `BackgroundColor`, `NoMapColors`,
`NoElevationColors`, `TrailColor`, `TrailCasingColor`, `MarkerColor`,
`MarkerRingColor`; o período dos padrões (12 px); as larguras relativas do
traçado e do marcador.

### `TileImage` e a porta `TileDecoder` (`tile_image.go`)

```go
type TileDecoder interface {
    // Decode decodes the image of one tile. format is the tile format of the
    // slice ("png", "jpg", "webp"). Bytes that are not an image fail with an
    // error the caller wraps as ErrSliceFileInvalid, naming the tile.
    Decode(format string, data []byte) (TileImage, error)
}
```

`TileImage`: `Width`, `Height` (a peça pode não ser 256 × 256; o desenho usa o
tamanho real) e pixels `[]uint8` RGBA de 8 bits, linha a linha, de cima para baixo.
Acesso `At(x, y) (r, g, b, a uint8)`.

### `FrameImage` e `FrameStats` (`frame_image.go`)

`FrameImage`: `Resolution`, `Pix []uint8` (RGB de 8 bits, linha a linha, de cima para
baixo; sem alfa). `FrameStats`: `MapHole bool` (algum pixel "sem imagem de
mapa") e `ElevationHole bool` (algum pixel "sem elevação"), calculados sobre os
pixels de terreno (`research.md` item 10).

### `Scene` (`frame_scene.go` e arquivos irmãos)

O terreno e o mapa de um recorte, prontos para desenhar. É construída **uma
vez** por execução por `NewScene(slice GeoSlice, decoder TileDecoder, tuning
RenderTuning) (*Scene, error)` (calcula as superfícies, o preenchimento das
amostras sem valor, os índices de peças e o cache) e é **segura para uso
concorrente** (só o cache de peças muda, e não muda resultado).

| Método | Regra |
|---|---|
| `Render(ctx context.Context, plan CameraPlan, index int, resolution Resolution) (FrameImage, FrameStats, error)` | desenha o quadro `index` do plano: câmera (item 3), raios (5), mapa (7), marcações (8), traçado dos quadros `0…index` e marcador (9), estatísticas (10). Cancela ao fim de uma faixa de linhas quando `ctx` é cancelado (devolve `ctx.Err()`). Peça ilegível: erro que embrulha `ErrSliceFileInvalid` com registro, nível e posição. |

Tipos internos (não exportados, um arquivo cada): `surface` (grade com nós
preenchidos, `zmax`, retângulo no plano; altura bilinear e DDA por quadra),
`imagery` (índice de peças presentes e ausentes por conjunto, cache com
orçamento, mips, filtro trilinear), `cameraFrame`/`camera` (base, projeção de
ponto e raio de pixel), `trail` (poligonal e cápsulas) e `marker`.

Estados de um pixel de terreno: `image`, `noMap`, `noElevation`; e o pixel de
fundo (raio sem acerto): `background`. Prioridade: `noElevation` > `noMap` >
`image`.

### `FrameSetID` e o nome do arquivo (`frame_set.go`)

| Função | Regra |
|---|---|
| `NewFrameSetID(plan CameraPlan, slice GeoSlice, resolution Resolution, tuning RenderTuning) FrameSetID` | SHA-256 hex de `"sobrevoo-frames" \| RenderVersion \| plan.ID() \| slice.ContentID \| largura \| altura \| tuning.Fingerprint()` (`research.md` item 17). |
| `FrameFileName(index int) string` | `frame_%06d.png`. |
| `ParseFrameFileName(name string) (index int, ok bool)` | inverso; `ok` só para `frame_` + 6 dígitos + `.png`. |
| `ParseFrameNumber(text string, frameCount int) (int, error)` | número do quadro de um texto; `ErrFrameOutOfRange` para texto que não é inteiro, negativo ou `≥ frameCount`, citando o valor e a faixa `0 a frameCount-1`. |

### `FrameDirectory` e a decisão do que fazer (`frame_set.go`)

Descrição do que há num diretório de destino, devolvida pela porta
`FrameRepository` (abaixo):

| Tipo | Campos |
|---|---|
| `FrameFile` | `Index int`; `Ours bool` (PNG com a marca `Sobrevoo`); `SetID FrameSetID` (vazio se não é nosso); `Complete bool` (assinatura, `IHDR` com a resolução pedida e `IEND` no fim) |
| `FrameDirectory` | `Files []FrameFile` (só os de nome `frame_NNNNNN.png`, ordenados por `Index`) |

`(FrameDirectory) Plan(id FrameSetID, frameCount int, overwrite bool) (FrameWork, error)`:

| `FrameWork` | O quê |
|---|---|
| `Keep []int` | quadros mantidos (nossos, do mesmo conjunto, completos, dentro do plano) — só sem `overwrite` |
| `Draw []int` | quadros a desenhar, em ordem crescente (todos, com `overwrite`) |
| `Remove []int` | com `overwrite`: arquivos **nossos de outro conjunto** com número fora do plano |

`ErrFrameSetConflict` (sem `overwrite`) quando há arquivo **nosso de outro
conjunto** ou arquivo de nome de quadro **que não é nosso** com número dentro do
plano; a mensagem conta quantos e diz como proceder (`--overwrite` ou outro
diretório). Sem conflito, `Keep ∪ Draw = 0…frameCount-1` e são disjuntos.

**Portas** (uma por papel, em `frame_set.go`, no topo, com `//go:generate`
logo após o `package`):

```go
// FrameRepository persists the frames of a flight in a directory.
type FrameRepository interface {
    // Inspect lists the frame files of dir. A missing dir is an empty
    // FrameDirectory; a path that is not a directory fails with
    // ErrFrameDestinationInvalid.
    Inspect(dir string, resolution Resolution) (FrameDirectory, error)

    // Save publishes the frame as dir/FrameFileName(index), creating dir if it
    // does not exist. It never leaves a partial file; an existing file is
    // replaced (Plan decided that). Failures: ErrFrameDestinationInvalid.
    Save(dir string, index int, id FrameSetID, image FrameImage) error

    // Remove deletes the frame files with the given numbers.
    Remove(dir string, indexes []int) error
}

// FrameExporter writes one frame to a file the user chose.
type FrameExporter interface {
    // Export writes image to path with the set mark id. Unless overwrite is true
    // it refuses an existing path (ErrFrameDestinationExists); it never leaves a
    // partial file. ErrFrameDestinationInvalid for a path that cannot be
    // written.
    Export(image FrameImage, id FrameSetID, path string, overwrite bool) error
}
```

### `RenderSummary` e `RenderProgress` (`render_summary.go`)

| `RenderSummary` | Tipo | Observação |
|---|---|---|
| `Requested` | `int` | quadros pedidos (1 para o quadro isolado; o total do plano) |
| `Drawn`, `Kept` | `int` | desenhados nesta execução; mantidos por já existirem |
| `MapHoleFrames`, `ElevationHoleFrames` | `int` | dos **desenhados**; um quadro com os dois entra nos dois |
| `Resolution` | `Resolution` | |
| `Elapsed` | `time.Duration` | tempo da execução; nunca vai para as imagens |
| `Removed` | `int` | arquivos removidos (sobrescrita) |
| `Interrupted` | `bool` | a execução foi cancelada |

Método `Add(stats FrameStats)`: conta um quadro desenhado. `RenderProgress`:
`Done`, `Total int` e `Elapsed time.Duration`.

## Erros sentinela novos (`errors.go`)

| Erro | Quando |
|---|---|
| `ErrSliceFileInvalid` | recorte não é um recorte, truncado, corrompido, incoerente, sem `plan_id`, ou peça ilegível |
| `ErrSliceFormatVersionUnsupported` | `format_version` do recorte desconhecida |
| `ErrSliceDoesNotMatchPlan` | `plan_id` do recorte ≠ `plan.ID()` |
| `ErrSliceDoesNotCoverPlan` | a área do recorte não contém a que o plano exige |
| `ErrTileFormatUnsupported` | peças vetoriais (`pbf`/`mvt`) ou outro formato que não se desenha |
| `ErrNoElevationData` | nenhuma amostra do recorte tem valor |
| `ErrFrameOutOfRange` | número do quadro inválido |
| `ErrInvalidResolution` | resolução inválida |
| `ErrFrameDestinationInvalid` | diretório ou arquivo de destino que não pode ser usado |
| `ErrFrameDestinationExists` | arquivo do quadro isolado já existe, sem sobrescrita |
| `ErrFrameSetConflict` | diretório com quadros de outro conjunto (ou nomes colidentes) |
| `ErrRenderInterrupted` | a execução foi cancelada pelo usuário |

O erro de um plano ilegível/de versão desconhecida reaproveita
`ErrPlanFileInvalid`/`ErrPlanFormatVersionUnsupported` da etapa 4.

## Serviços (`internal/application`)

`FrameService` (novo, um por recurso: os quadros) — `frame_service.go`:

```go
type FrameService interface {
    // DrawFrame draws frame request.Number of plan into request.Path.
    DrawFrame(ctx context.Context, plan domain.CameraPlan, slice domain.GeoSlice, request domain.SingleFrameRequest) (domain.RenderSummary, error)

    // DrawFrames draws the frames of plan that request.Directory does not have
    // yet (all of them with request.Overwrite), reporting progress after each.
    DrawFrames(ctx context.Context, plan domain.CameraPlan, slice domain.GeoSlice, request domain.FrameSetRequest, progress func(domain.RenderProgress)) (domain.RenderSummary, error)
}
```

`SingleFrameRequest` = `{Number int; Path string; Resolution Resolution;
Overwrite bool}` e `FrameSetRequest` = `{Directory string; Resolution
Resolution; Overwrite bool}`, ambos de domínio (`frame_set.go`).

`DrawFrame` chama, em ordem: `slice.EnsureMatches`, `EnsureCovers`,
`EnsureDrawable` (ordem do item 16), `NewScene`, `Render`, `FrameExporter.Export`.
`DrawFrames` faz o mesmo, mais `FrameRepository.Inspect` →
`FrameDirectory.Plan` → `Remove` → um `Render` + `Save` por quadro pendente, em
ordem, com `ctx` e `progress`. O tempo é medido com `time.Now()` (biblioteca
padrão; Princípio II).

`CameraPlanService` (existente) e `GeoSliceService` (existente, acréscimo):

- `GeoSliceService.Load(path string) (domain.GeoSlice, error)` — chama
  `GeoSliceReader.Read`. Recebe a porta nova no construtor.
- `GeoSliceService.Generate` grava `plan.ID()` em `GeoSlice.PlanID` (uma
  linha), e o exportador o escreve.

## Ciclo de vida de um diretório de quadros

```text
(sem diretório) --render all--> diretório com quadros do conjunto X (parcial ou completo)
   parcial + mesmo comando   --> completa (Keep + Draw); quadros existentes intactos
   completo + mesmo comando  --> nada a desenhar; "all frames already existed"
   conjunto Y + sem --overwrite --> ErrFrameSetConflict; nada alterado
   conjunto Y + --overwrite  --> Remove (nossos de X fora do plano) + Draw de todos; vira conjunto Y
```

Não há arquivo de registro: cada `frame_NNNNNN.png` traz a marca do conjunto.
