---

description: "Task list for feature implementation"
---

# Tarefas: Desenho dos Quadros do Voo

**Entrada**: Documentos de design de `/specs/005-frame-rendering/`

**Pré-requisitos**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/cli.md`, `contracts/frame-files.md`, `contracts/slice-file-change.md`, `quickstart.md`

**Testes**: incluídos. Como nas etapas anteriores, a constituição do projeto
(Princípio VI — Testes Automatizados no Núcleo; Princípio X —
Given/When/Then, builders e isolamento por camada) exige testify e uber-go/mock
e proíbe testes tabulares: cada cenário é um `t.Run("should ...")` com
`// given`, `// when`, `// then`. As tarefas de teste abaixo materializam
essa exigência; ficam antes da implementação correspondente em cada fase e
devem falhar primeiro.

**Organização**: as tarefas são agrupadas por história de usuário (P1–P7 de
`spec.md`) para permitir implementação e teste independentes de cada
história. A **Phase 2** tem quatro partes: (A) o `fsync` em `atomicfile`
(`research.md` item 18), pequeno e isolado; (B) a **mudança na etapa 4**
(`plan_id`, `research.md` item 14), que precisa estar verde antes de qualquer
código novo, para isolar uma regressão de uma etapa já entregue; (C) a base
compartilhada (erros, códigos de saída, tipos de domínio e portas,
configuração, mocks, builders); e (D) as fixtures em `test/helper`.

**Divisão do trabalho entre histórias** (o mesmo arquivo é estendido por mais
de uma história, nunca em paralelo):

| História | O que entrega |
|---|---|
| US1 | **MVP**: um quadro isolado de ponta a ponta — leitor do recorte (caminho feliz), decodificação das peças, o renderizador completo (câmera, superfície, DDA, mapa, traçado, marcador, cena), gravação atômica de um PNG com a marca do conjunto, `FrameService.DrawFrame`, `GeoSliceService.Load` e o comando `render frame` (resolução padrão) |
| US2 | `render all`: `FrameRepository.Save`, `FrameService.DrawFrames` num destino vazio, resumo e comando |
| US3 | falta de dado explícita: marcações "sem imagem de mapa" e "sem elevação", contagem de quadros com buraco, resumo |
| US4 | resolução: flag `--resolution`, invariância do enquadramento, retrato e ultralargo |
| US5 | progresso, interrupção e retomada: `FrameRepository.Inspect`, `FrameDirectory.Plan` (manter × desenhar), `context`, sinais, código 38 |
| US6 | proteção do destino: conflito de conjunto, `--overwrite`, remoção das sobras, gravação atômica sob falha, quadro isolado existente |
| US7 | recusas: `EnsureMatches`, `EnsureCovers`, `EnsureDrawable`, validação completa do recorte, ordem das verificações, mensagens |

Enquanto a US3 não existe, um pixel sem imagem de mapa ou sem elevação sai na
cor de fundo (é o que a US1 implementa); enquanto a US7 não existe, o serviço
não chama `Ensure*` (a US7 os insere no serviço).

## Formato: `[ID] [P?] [Story] Descrição`

- **[P]**: pode ser executado em paralelo (arquivos diferentes, sem
  dependência de tarefa incompleta). Tarefas que editam o mesmo arquivo NUNCA
  são marcadas `[P]` entre si, mesmo quando logicamente independentes.
- **[Story]**: a qual história de usuário esta tarefa pertence (US1 a US7).
  Tarefas de Setup, Foundational e Polish não têm esse rótulo.
- Toda tarefa inclui o caminho de arquivo exato a criar/editar.
- Comentários de código, identificadores, mensagens de commit, flags, saída
  e mensagens de erro em tempo de execução: **inglês**; artefatos do
  Spec Kit: português (constituição, "Idioma dos Artefatos").

## Convenções de Caminho

Mesmo projeto único em Go, mesma estrutura hexagonal de `plan.md`:
`cmd/sobrevoo/`, `internal/domain/`, `internal/application/`,
`internal/infra/outbound/`, `internal/infra/inbound/cli/`. Mocks em
`internal/domain/mockdomain` e `internal/application/mockapplication`;
builders em `internal/domain/builddomain`; fixtures em `test/helper`. Regras
de estilo: sem `ports.go`; porta no arquivo da entidade, no topo, com
`//go:generate` logo após `package`; receivers curtos e consistentes com o
tipo (`s` `Scene` e os `*Service`, `c` `CameraPlan`/`camera`, `g` `GeoSlice`,
`b` `BoundingBox` e os builders, `r` `Resolution` e readers/repositories, `d`
`FrameDirectory`, `t` `RenderTuning`, `e` exporters); `new(x)` do Go 1.26 em vez
de um helper `ptr`; nome exportado nunca repete o pacote.

**Regras de determinismo do desenho (`research.md` item 12), válidas em todo
código de `frame_*.go`**: todo produto que entra numa soma é envolvido em
`float64(...)` (sem fusão multiplicação-soma); só `+ − × ÷`, `math.Sqrt`,
`Floor`, `Abs`, `Min`, `Max`, `Sin`, `Cos`, `Tan`, `Atan2`, `Log`, `Log2` e
`Ldexp` — **nunca `Exp`, `Pow` nem `Sinh`**; nunca iterar `map` para produzir
valor; nada de `time`, `rand` ou ambiente no cálculo de um pixel.

---

## Phase 1: Setup (Shared Infrastructure)

**Propósito**: garantir um ponto de partida verde.

- [X] T001 Conferir que o branch atual é `005-frame-rendering` (criado por `/speckit-specify`); rodar `make test`, `make lint` e `make generate` e confirmar que os três passam e que `make generate` não produz diff (`git status` limpo fora de `specs/005-frame-rendering/`); registrar qualquer falha pré-existente antes de continuar

**Checkpoint**: baseline verde.

---

## Phase 2: Foundational (Blocking Prerequisites)

**⚠️ CRÍTICO**: nenhuma tarefa de história de usuário pode começar até que
esta fase esteja completa.

### Parte A — `fsync` em `atomicfile` (research.md item 18)

- [X] T002 [P] Estender `internal/infra/outbound/atomicfile/atomic_file_test.go`: acrescentar a variável de teste `syncFile` (mesmo padrão da variável `link`, ver `atomic_file_fallback_test.go`) e os cenários: `Publish` chama `Sync` no arquivo temporário antes de publicar (o teste troca `syncFile` por um contador e confere 1 chamada); um `Sync` que falha → `errors.Is(err, atomicfile.ErrInvalid)`, nenhum arquivo no destino e nenhum `.sobrevoo-*.tmp` deixado; os cenários existentes continuam passando
- [X] T003 Editar `internal/infra/outbound/atomicfile/atomic_file.go`: declarar `var syncFile = func(f *os.File) error { return f.Sync() }` e chamá-la em `Publish` depois de `write` e antes de `Close`, traduzindo a falha por `invalid(...)`; nenhuma outra mudança de comportamento (depende de T002)
- [X] T004 Rodar `make test`, `make lint` e `make generate` (sem diff) e conferir que `plan --export` e `geodata slice --export` mantêm o comportamento e os códigos de saída (depende de T003)

### Parte B — Mudança na etapa 4: `plan_id` (research.md item 14; contracts/slice-file-change.md)

**Propósito**: acrescentar a identificação do plano ao recorte exportado.
`format_version` continua 1; o único efeito visível é uma linha a mais no
`manifest.json`.

- [X] T005 [P] Estender `internal/domain/camera_plan_test.go` com os cenários de `CameraPlan.ID()`: devolve 64 caracteres hexadecimais minúsculos; dois planos com o mesmo conteúdo têm o mesmo `ID` (dois `CameraPlanBuilder` iguais); um plano relido de arquivo (valores decodificados de JSON com as casas do contrato) tem o mesmo `ID` do plano de memória de que o arquivo veio; mudar **um** valor de **um** quadro no menor passo do plano (latitude da câmera em `1e-7`, altitude em `1e-3`, `heading` em `1e-3`, distância do marcador em `1e-3`) muda o `ID`; mudar `index` ou `phase` de um quadro muda o `ID`; mudar a taxa de quadros, a duração, o nível de distância ou o de inclinação muda o `ID`; acrescentar um quadro muda o `ID`; mudar só `Summary` (que é derivado) **não** muda o `ID`; o `ID` não depende de a lista `SmoothedSpans` estar vazia ou não
- [X] T006 Editar `internal/domain/camera_plan.go`: `func (c CameraPlan) ID() string` — SHA-256 (`crypto/sha256`) em hexadecimal minúsculo de uma codificação canônica escrita com `encoding/binary` (ordem de bytes fixa, `big-endian`): um prefixo `"sobrevoo-plan-v1"`; os parâmetros efetivos (`Duration` em nanossegundos como `int64`, `FrameRate` como `math.Float64bits`, os níveis `Distance` e `Tilt` como texto com comprimento); e, por quadro em ordem, `Index` (`int64`), `Phase` (texto com comprimento) e os valores **quantizados aos passos do plano** como `int64` (`math.Round(v / passo)`): latitude/longitude da câmera e do marcador em `1e-7`; altitude, distância do marcador e distância câmera–marcador em `1e-3`; `Heading` e `Tilt` em `1e-3`; `Time` não entra (deriva de `Index` e da taxa); `Summary` e `TimeReference` também não entram (depende de T005)
- [X] T007 [P] Estender `internal/application/geo_slice_service_test.go` (cenário: `Generate` grava `plan.ID()` em `GeoSlice.PlanID`), `internal/infra/outbound/zipfile/geo_slice_exporter_test.go` (o `manifest.json` traz, entre `format_version` e `area`, a linha `  "plan_id": "<64 hex>",` com o valor de `slice.PlanID`; duas exportações do mesmo recorte continuam idênticas byte a byte; os cenários que comparam o manifesto passam a esperar a linha) e `internal/domain/geo_slice_test.go` (`GeoSlice` tem os campos `PlanID` e `ContentID`, ambos vazios num recorte montado por `NewGeoSlice`)
- [X] T008 Editar `internal/domain/geo_slice.go` (campos `PlanID string` e `ContentID string` em `GeoSlice`, com o comentário do contrato: `ContentID` só o leitor preenche e não é exportado), `internal/application/geo_slice_service.go` (`Generate` atribui `slice.PlanID = plan.ID()` antes de devolver) e `internal/infra/outbound/zipfile/geo_slice_exporter.go` (escrever `plan_id` no manifesto logo após `format_version`, com `%q`) (depende de T006, T007)
- [X] T009 [P] Editar `specs/004-geo-data-slice/contracts/slice-file.md`: acrescentar o campo `plan_id` ao exemplo de `manifest.json` e uma linha na tabela de campos, e uma nota em "Compatibilidade" apontando para `specs/005-frame-rendering/contracts/slice-file-change.md` (a mesma prática da nota das duas correções do `register` em `specs/002-geo-data-registry/contracts/cli.md`)
- [X] T010 Rodar `make test`, `make lint` e `make generate` (sem diff) e, com o binário, `geodata slice <plano> --export /tmp/r.zip` duas vezes conferindo `cmp` e `unzip -p /tmp/r.zip manifest.json | head -4` (depende de T008, T009)

**Checkpoint B**: a etapa 4 verde, com o campo novo; nenhum código novo da
etapa 5 além de `CameraPlan.ID`, `PlanID`/`ContentID` e o `fsync`.

### Parte C — Base compartilhada

- [X] T011 [P] Estender `internal/domain/errors.go` com os 12 sentinelas novos (cada um com comentário em inglês; Princípio VII): `ErrSliceFileInvalid`, `ErrSliceFormatVersionUnsupported`, `ErrSliceDoesNotMatchPlan`, `ErrSliceDoesNotCoverPlan`, `ErrTileFormatUnsupported`, `ErrNoElevationData`, `ErrFrameOutOfRange`, `ErrInvalidResolution`, `ErrFrameDestinationInvalid`, `ErrFrameDestinationExists`, `ErrFrameSetConflict`, `ErrRenderInterrupted` (significados em `data-model.md`, "Erros sentinela novos")
- [X] T012 [P] Editar `internal/infra/inbound/cli/exit_code.go` mapeando com `errors.Is`: `ErrSliceFileInvalid`→27, `ErrSliceFormatVersionUnsupported`→28, `ErrSliceDoesNotMatchPlan`→29, `ErrSliceDoesNotCoverPlan`→30, `ErrTileFormatUnsupported`→31, `ErrNoElevationData`→32, `ErrFrameOutOfRange`→33, `ErrInvalidResolution`→34, `ErrFrameDestinationInvalid`→35, `ErrFrameDestinationExists`→36, `ErrFrameSetConflict`→37, `ErrRenderInterrupted`→38; estender `internal/infra/inbound/cli/exit_code_test.go` (um `t.Run` por sentinela, um com o erro embrulhado por `fmt.Errorf("%w: ...")`, e um que confirma que os códigos 1–26 existentes não mudaram) (depende de T011)
- [X] T013 [P] Criar `internal/domain/tile_image.go`: diretiva `//go:generate go run go.uber.org/mock/mockgen -destination mockdomain/tile_decoder.go -package mockdomain . TileDecoder` logo após `package`; a interface `TileDecoder` no topo, após os imports (`Decode(format string, data []byte) (TileImage, error)`, comentário do contrato: `format` é `png`, `jpg` ou `webp`; bytes que não são uma imagem legível falham com um erro que o chamador embrulha em `ErrSliceFileInvalid`); depois `TileImage{Width, Height int; Pix []uint8}` (RGBA de 8 bits, linha a linha, de cima para baixo) com `At(x, y int) (r, g, b, a uint8)` e `NewTileImage(width, height int, pix []uint8) TileImage`
- [X] T014 [P] Criar `internal/domain/frame_resolution_test.go`: `NewResolution` aceita 1920×1080, 180×180, 3840×2160, 2160×3840 (retrato) e 3840×1080 (ultralargo) e recusa (com `ErrInvalidResolution` citando o valor recebido e os limites) largura ou altura ímpar, zero, negativa, menor que 180, maior que 3840, e um produto acima de 8 294 400 (por exemplo 3840×2200); `ParseResolution("1920x1080")` e `"1920X1080"` funcionam; `""`, `"abc"`, `"1920"`, `"1920x"`, `"x1080"`, `"1920x1080x2"`, `"19.5x1080"`, `"-1920x1080"` e `"1920 x 1080"` falham com `ErrInvalidResolution`; `Pixels()` é `Width*Height`
- [X] T015 [P] Criar `internal/domain/frame_resolution.go`: `Resolution{Width, Height int}`; constantes `MinFrameSide = 180`, `MaxFrameSide = 3840`, `MaxFramePixels = 8_294_400`; `NewResolution(width, height int) (Resolution, error)` — **largura e altura pares, cada uma de 180 a 3840, com `Width × Height ≤ 8 294 400`** — e `ParseResolution(text string) (Resolution, error)` (`LARGURAxALTURA`, `x` ou `X`, só dígitos, sem espaços); ambos falham com `ErrInvalidResolution` (mensagem: o valor recebido e os limites); `(r Resolution) Pixels() int` (depende de T011, T014)
- [X] T016 [P] Criar `internal/domain/render_tuning_test.go`: `Fingerprint()` é o mesmo para dois `RenderTuning` iguais; muda quando muda `VerticalFOVDegrees`, `MinCameraClearanceMeters`, `MinTiltForTargetDegrees`, `TrailLiftMeters`, `DepthBiasMeters` ou `DepthBiasRatio`; **não** muda quando muda `TileCacheBytes` ou `Workers`; a cadeia é estável (um valor fixo esperado para os valores iniciais de `research.md` item 22)
- [X] T017 [P] Criar `internal/domain/render_tuning.go`: `RenderTuning{VerticalFOVDegrees, MinCameraClearanceMeters, MinTiltForTargetDegrees, TrailLiftMeters, DepthBiasMeters, DepthBiasRatio float64; TileCacheBytes int64; Workers int}` (comentário por campo, `research.md` item 22); `(t RenderTuning) Fingerprint() string` (cadeia canônica com `strconv.FormatFloat(v, 'g', -1, 64)` dos seis campos que afetam a aparência, em ordem fixa); e as **constantes de domínio** de `contracts/frame-files.md`: `RenderVersion = 1`; cores `BackgroundColor` `#20262E`, `NoMapColors` (`#C8C8C8`, `#6E6E6E`), `NoElevationColors` (`#FF00FF`, `#3A003A`), `TrailColor` `#FFB000`, `TrailCasingColor` `#101010`, `MarkerColor` `#E5252A`, `MarkerRingColor` `#FFFFFF` (tipo `RGB{R, G, B uint8}`); `PatternPeriod = 12`; larguras relativas do traçado (`TrailWidthRatio = 0.005`, mínimo 2 px), do marcador (`MarkerRadiusRatio = 0.012`, mínimo 4 px) e do anel (`MarkerRingRatio = 0.003`, mínimo 1,5 px) (depende de T016)
- [X] T018 [P] Criar `internal/domain/frame_image.go`: `FrameImage{Resolution Resolution; Pix []uint8}` (RGB de 8 bits, linha a linha, de cima para baixo, sem alfa) com `NewFrameImage(resolution Resolution) FrameImage` (preenchida com `BackgroundColor`), `Set(x, y int, c RGB)` e `At(x, y int) RGB`; e `FrameStats{MapHole, ElevationHole bool}` (depende de T017)
- [X] T019 [P] Criar `internal/domain/render_summary_test.go`: `RenderSummary.Add(stats)` incrementa `Drawn`, e `MapHoleFrames` só se `stats.MapHole`, `ElevationHoleFrames` só se `stats.ElevationHole`; um quadro com os dois incrementa os dois; `Requested`, `Kept`, `Removed`, `Resolution`, `Elapsed` e `Interrupted` são campos simples
- [X] T020 [P] Criar `internal/domain/render_summary.go`: `RenderSummary{Requested, Drawn, Kept, MapHoleFrames, ElevationHoleFrames, Removed int; Resolution Resolution; Elapsed time.Duration; Interrupted bool}` com `(s *RenderSummary) Add(stats FrameStats)`, e `RenderProgress{Done, Total int; Elapsed time.Duration}` (depende de T018, T015, T019)
- [X] T021 [P] Criar `internal/domain/frame_set_test.go`: `FrameFileName(0)` = `frame_000000.png`, `FrameFileName(300)` = `frame_000300.png`, `FrameFileName(431999)` = `frame_431999.png`; a ordem alfabética dos nomes de 0 a 1259 é a numérica; `ParseFrameFileName` devolve `(300, true)` para `frame_000300.png` e `ok == false` para `frame_1.png`, `frame_0003000.png`, `frame_000300.jpg`, `Frame_000300.png`, `frame_00030a.png` e `""`; `ParseFrameNumber("300", 1260)` = 300, `("0", 1260)` = 0, `("1259", 1260)` = 1259; `ParseFrameNumber` devolve `ErrFrameOutOfRange` para `"1260"` (mensagem cita o valor e `frames 0 to 1259`), `"-1"`, `"3.5"`, `"abc"`, `""`, `" 3"` e `"99999999999999999999"` (`"3.5"` e `"abc"`: mensagem diz que não é um número inteiro); `NewFrameSetID` é uma cadeia de 64 hexadecimais minúsculos, igual para as mesmas entradas e **diferente** quando muda o plano (`ID`), o recorte (`ContentID`), a largura, a altura, o `RenderTuning.Fingerprint()` ou a constante `RenderVersion` (este último testado por uma variável de teste que sobrescreve a versão)
- [X] T022 Criar `internal/domain/frame_set.go`: diretivas `//go:generate ... -destination mockdomain/frame_repository.go -package mockdomain . FrameRepository` e `... mockdomain/frame_exporter.go ... FrameExporter` logo após `package`; no topo, as interfaces `FrameRepository` (`Inspect(dir string, resolution Resolution) (FrameDirectory, error)`; `Save(dir string, index int, id FrameSetID, image FrameImage) error`; `Remove(dir string, indexes []int) error`) e `FrameExporter` (`Export(image FrameImage, id FrameSetID, path string, overwrite bool) error`), com os comentários de contrato de `data-model.md`; depois `FrameSetID` (`string`), `NewFrameSetID(plan CameraPlan, slice GeoSlice, resolution Resolution, tuning RenderTuning) FrameSetID` (SHA-256 hex de `"sobrevoo-frames" | RenderVersion | plan.ID() | slice.ContentID | largura | altura | tuning.Fingerprint()`, cada parte com o comprimento à frente para evitar ambiguidade), `FrameFileName`, `ParseFrameFileName`, `ParseFrameNumber(text string, frameCount int) (int, error)`, `SingleFrameRequest{Number int; Path string; Resolution Resolution; Overwrite bool}`, `FrameSetRequest{Directory string; Resolution Resolution; Overwrite bool}`, e **só os tipos** `FrameFile{Index int; Ours bool; SetID FrameSetID; Complete bool}`, `FrameDirectory{Files []FrameFile}` e `FrameWork{Keep, Draw, Remove []int}` (o método `Plan` entra na US5/US6) (depende de T011, T017, T015, T018, T006, T008, T021)
- [X] T023 Editar `internal/domain/geo_slice.go`: acrescentar, logo após a diretiva `//go:generate` existente, `//go:generate go run go.uber.org/mock/mockgen -destination mockdomain/geo_slice_reader.go -package mockdomain . GeoSliceReader`, e a interface `GeoSliceReader` (`Read(path string) (GeoSlice, error)`, comentário do contrato de `data-model.md`: `ErrSliceFileInvalid`, `ErrSliceFormatVersionUnsupported`; erro de E/S sobe embrulhado sem sentinela; devolve o recorte com `PlanID` e `ContentID` preenchidos) logo abaixo de `GeoSliceExporter`, antes das constantes (depende de T008)
- [X] T024 Rodar `make generate` para gerar `internal/domain/mockdomain/{tile_decoder,frame_repository,frame_exporter,geo_slice_reader}.go` (depende de T013, T022, T023)
- [X] T025 [P] Editar `internal/infra/outbound/config/config.go`: tipos próprios do pacote (sem importar o domínio) `RenderTuning` (mesmos campos de `domain.RenderTuning`) e `RenderDefaults{Width, Height int}`; campos `Config.RenderTuning` e `Config.RenderDefaults` preenchidos em `Load()` com os valores de `research.md` item 22 (`VerticalFOVDegrees` 45 **lido da mesma constante de pacote** que `CameraTuning.OverviewVerticalFOVDegrees`, agora nomeada, para que os dois nunca divirjam; `MinCameraClearanceMeters` 2; `MinTiltForTargetDegrees` 1; `TrailLiftMeters` 0,3; `DepthBiasMeters` 1; `DepthBiasRatio` 0,002; `TileCacheBytes` 268435456; `Workers` = `runtime.NumCPU()`) e `RenderDefaults{1920, 1080}`; estender `config_test.go` (os valores; e que `RenderTuning.VerticalFOVDegrees == CameraTuning.OverviewVerticalFOVDegrees`; `Workers >= 1`)
- [X] T026 Editar `cmd/sobrevoo/config_mapping.go`: `domainRenderTuning(t config.RenderTuning) domain.RenderTuning` e `domainRenderResolution(d config.RenderDefaults) (domain.Resolution, error)` (usa `domain.NewResolution`); estender o teste de mapeamento existente em `cmd/sobrevoo/` (ou criar `config_mapping_test.go` no padrão dos testes vizinhos) conferindo campo a campo (depende de T017, T015, T025)
- [X] T027 [P] Criar `internal/domain/builddomain/render_tuning_builder.go` (`NewRenderTuningBuilder()` com os valores de T025 e `WithVerticalFOVDegrees`, `WithMinCameraClearanceMeters`, `WithMinTiltForTargetDegrees`, `WithTrailLiftMeters`, `WithDepthBias`, `WithTileCacheBytes`, `WithWorkers`, `Build()`) e `internal/domain/builddomain/frame_directory_builder.go` (`NewFrameDirectoryBuilder()`: diretório vazio; `WithOursFrame(index int, setID FrameSetID)`, `WithIncompleteFrame(index, setID)`, `WithForeignFrame(index)`, `Build()`); estender `internal/domain/builddomain/geo_slice_builder.go` com `WithPlanID(string)` e `WithContentID(string)`; todos com receiver `b`; um cenário em `config_test.go` garante que `NewRenderTuningBuilder().Build()` coincide com a configuração (depende de T017, T022, T025)

**Checkpoint C**: erros, códigos de saída, tipos, portas, mocks, configuração e
builders prontos; nenhum comando novo ainda; `make test`, `make lint` e `make
generate` verdes.

### Parte D — Fixtures (`test/helper`)

**Propósito**: os leitores e o desenho precisam de imagens e de arquivos de
recorte reais. Tudo gerado em código, sem binário versionado.

- [X] T028 [P] Criar `test/helper/png_fixture.go`: `PNGTile(color RGB)` (peça 256×256 de cor lisa, codificada com `image/png`), `CheckerPNGTile(a, b RGB, square int)` (xadrez), `PositionPNGTile(z, x, y int)` (xadrez de dois tons cuja matiz e cuja borda de 4 px dependem de `z`, `x`, `y`, para reconhecer a peça numa imagem), `TransparentPNGTile()` (metade dos pixels com alfa 0), `JPEGTile(color RGB)` (`image/jpeg`, qualidade 100), `WebPTile()` (o literal base64 mínimo `UklGRhoAAABXRUJQVlA4TA0AAAAvAAAAEAcQERGIiP4HAA==`, um WebP sem perdas de 1×1 que `golang.org/x/image/webp` decodifica) e `NotAnImage()` (bytes que não são imagem); nenhuma depende de arquivo em disco
- [X] T029 [P] Criar `test/helper/slice_fixture.go`: `SliceFileSpec` (`PlanID string`, `Area` (`domain.BoundingBox`), `Sources`, `TileSets` — cada um com formato, nível `Ideal/Chosen/Min/Max`, peças presentes (`Tiles []MBTile`) e ausentes —, `Grids` — cada uma com `Rows`, `Cols`, canto e célula, e `Values []float32` com NaN para "sem valor"), `DefaultSliceFileSpec()` (uma fonte de mapa e uma de relevo, 3 peças PNG de nível 16, uma ausente, uma grade 4×4 com uma célula sem valor) e `ValidSliceFile(spec) []byte` que escreve um ZIP **na mesma estrutura de `contracts/slice-file.md`** (`archive/zip`, método *store*, `manifest.json` com `format_version` 1, `plan_id`, `area`, `summary` **recalculado a partir do conteúdo**, `sources`, `base_map`, `elevation`; `elevation/NNN.f32` em `float32` little-endian com o NaN `0x7FC00000`; `tiles/III/Z/X/Y.ext`) sem usar o exportador (o teste do leitor não pode depender do exportador); variações nomeadas: `TruncatedSliceFile`, `SliceFileWithVersion(n)`, `SliceFileWithoutPlanID`, `SliceFileWithBadPlanID` (`plan_id` que não é hexadecimal de 64), `SliceFileWithWrongCounts(field)` (altera um campo do `summary`), `SliceFileWithoutEntry(path)`, `SliceFileWithExtraEntry`, `SliceFileWithWrongTileSize`, `SliceFileWithWrongGridSize`, `SliceFileWithCompressedEntry` (método *deflate*), `SliceFileWithChosenOutOfRange`, `SliceFileWithBadSourceIndex`, `SliceFileWithoutManifest`, `SliceFileWithCorruptEntry` (CRC errado numa entrada), `NotZipContent()`
- [X] T030 Rodar `make test`, `make lint` e `make generate` (sem diff) (depende de T027, T026, T028, T029, T012, T024)

**Checkpoint**: base pronta; as histórias podem começar.

---

## Phase 3: User Story 1 - Desenhar um quadro isolado pelo número (Priority: P1) 🎯 MVP

**Goal**: com um plano e um recorte que correspondem, `render frame` grava um
PNG do que a câmera vê naquele quadro: relevo em perspectiva vestido com as
peças do mapa base, traçado até o marcador e marcador, idêntico a cada repetição.

**Independent Test**: `quickstart.md` itens 1 e 2: registrar o mapa em imagem
e o relevo sintéticos, exportar plano e recorte, `render frame --number 300`;
abrir a imagem (terreno com o xadrez, traçado até o marcador, marcador
vermelho); repetir e comparar com `cmp`.

### Testes da User Story 1 (escrever primeiro; devem falhar)

- [X] T031 [P] [US1] Criar `internal/domain/frame_camera_test.go`: **plano local** — o ponto da própria câmera é `(0, 0)`; `x` cresce a leste e `y` a norte, em metros, com `x = Δλ·(π/180)·R·max(cos φ₀, 1e-6)` e `y = Δφ·(π/180)·R` (`R` = `earthRadiusMeters`); ida e volta graus→plano→graus; uma câmera em `lon = 179,99` e um ponto em `lon = -179,99` ficam a `x ≈ +2·0,01°` sem salto (antimeridiano); `cos φ₀` preso em `1e-6` perto do polo (`lat = 89,9999999`) mantém tudo finito; **base da câmera** — `frente = (sin h·cos t, cos h·cos t, −sin t)`, `direita = (cos h, −sin h, 0)`, `cima = (sin h·sin t, cos h·sin t, cos t)` são ortonormais para `h` em 0, 90, 217,5 e `t` em 0, 25, 45, 90; com `h = 0` a direita é o leste; com `t = 90` o "cima" da tela é o rumo; **raio do pixel** — o pixel central (`(W/2−0,5, H/2−0,5)` em resolução par, média dos quatro centrais) aponta para a frente com erro < 1e-9; o pixel do canto superior esquerdo desvia `atan(tan(FOVv/2)·W/H)` para a esquerda e `FOVv/2` para cima; `f_px = (H/2)/tan(FOVv/2)`; **projeção** — projetar um ponto sobre o raio de um pixel `(i, j)` devolve `(i+0,5, j+0,5)` com erro < 1e-9 (ida e volta), e um ponto atrás da câmera é rejeitado (`ok == false`); **alvo** — `alvo = câmera + (altitude/tan t)·(sin h, cos h)`; com `t < MinTiltForTargetDegrees` (0,5°, 0°) o alvo é o marcador; **altura absoluta** — `H = terreno(alvo) + altitude`; com o terreno sob a câmera mais alto que `H − clearance`, `H = terreno(câmera) + MinCameraClearanceMeters`; com terreno plano `H = terreno + altitude`
- [X] T032 [P] [US1] Criar `internal/domain/frame_surface_test.go`: a superfície interpola bilinearmente entre os **centros** das células (`N(r,c)` no ponto `(x₀ + (c+0,5)·cx, y₀ − (r+0,5)·cy)`): a altura num nó é o valor da célula; no ponto médio entre dois nós é a média; no centro de quatro nós é a média dos quatro; **extensão até a borda** — entre o canto da grade e o primeiro centro a altura é a do nó mais próximo (índices presos), então a grade cobre exatamente `rows×cy` por `cols×cx`; **amostras sem valor** — uma grade com um `NaN` no meio recebe, na geometria, o valor da amostra com valor mais próxima em distância de quarteirão (L1), com o empate resolvido pela ordem das duas passadas (linha antes de coluna, do noroeste ao sudeste e de volta); um bloco 3×3 sem valor no meio de uma rampa toma os valores do vizinho mais próximo de cada célula; uma grade **inteira** sem valor recebe a **menor** elevação com valor do recorte; uma grade sem buracos não aloca a cópia preenchida (o teste confere que `filled == nil`); a marca "célula sem valor" continua consultável pela célula original (`hole(r, c) == true` só onde era `NaN`); **posição no plano** — o retângulo de uma grade cujo `west_lon` é `179,9` e cujas colunas passam de 180° tem `x` contínuo em relação a uma câmera em `lon = -179,95`; `zmax` é a maior altura já preenchida
- [X] T033 [P] [US1] Criar `internal/domain/frame_surface_dda_test.go`: raios sobre superfícies de fórmula conhecida — **plano horizontal** a 100 m: o raio de uma câmera a 300 m descendo a 45° acerta a `t = 200·√2` (± 1e-6); **rampa** de 10% na direção norte: o acerto coincide com a interseção analítica (± 1e-6, com as 16 bisseções); **pirâmide** (nós que sobem e descem): o raio que passa por trás do cume **não** acerta o vale (oclusão) e o que passa acima do cume acerta o vale; **raio para o céu** (sobe, acima de `zmax`) não acerta; raio horizontal acima de `zmax` não acerta; **raio que começa abaixo da superfície** (câmera dentro do morro) acerta em `t` do início da grade; **raio que entra na grade vizinha já abaixo do terreno** (degrau de 50 m entre duas grades) acerta em `t_entrada` da segunda; duas grades vizinhas sem fresta (raio rasante cruzando a emenda acerta uma das duas); grade que cruza o antimeridiano (raio de uma câmera em `lon = 179,999` para leste acerta a parte de `lon = -179,9`); raio fora do retângulo da grade não acerta; o acerto devolve também a célula `(linha, coluna)` que contém o ponto (`floor`, mesma convenção de `ElevationGridInfo.CellAt`) e a posição `(x, y)`; **poda por altura** — desenhar um conjunto de 200 raios com e sem a poda por `zmax` (uma variável de teste a desliga) dá exatamente os mesmos `t`, células e acertos
- [X] T034 [P] [US1] Criar `internal/domain/frame_imagery_test.go` (com o `TileDecoder` mockado por `mockdomain`): **Web Mercator** — `(lon 0, lat 0)` → `(u, v) = (0,5; 0,5)`; `(lon −180, lat 85,0511287798)` → `(0; ~0)`; `(lon 180)` → `u = 1`; a peça de `(z, u, v)` é `(z, ⌊u·2^z⌋, ⌊v·2^z⌋)` e o texel é `((u·2^z − x)·256, (v·2^z − y)·256)` (o tamanho real da peça decodificada substitui 256); **escolha da peça entre conjuntos** — uma peça presente no primeiro conjunto vence a presente no segundo; presente em qualquer conjunto vence "ausente" de outro; ausente num conjunto e desconhecida nos outros → "peça ausente"; desconhecida em todos → "sem peça" (o mesmo sinal, para a US3); **mips** — o nível 1 de uma peça 4×4 é a média `(a+b+c+d+2)>>2` por canal de cada bloco 2×2; o nível 0 é a própria peça; **filtro trilinear** — numa peça de cor lisa qualquer `λ` devolve a mesma cor; numa peça em xadrez, `λ` grande converge para o tom médio; `λ = log2(max(ρ, 1))` com `ρ = passo_angular_do_pixel · t / (m/texel do nível na latitude do ponto) / √max(|d_z|, 0,1)` cresce com `t`; amostras que caem numa peça vizinha **ausente** usam o texel mais próximo da peça do centro; **transparência** — pixel com alfa 0 sai na cor `BackgroundColor`, alfa 128 é a mistura; **cache** — cada peça é decodificada **uma vez** (o mock de `Decode` é chamado 1 vez por peça mesmo com 8 goroutines pedindo a mesma peça), e com `TileCacheBytes` pequeno o cache descarta as mais antigas **sem mudar a cor devolvida**; **erro** — `Decode` que falha vira um erro que `errors.Is(err, ErrSliceFileInvalid)` e cuja mensagem cita o registro, o nível e a posição da peça
- [X] T035 [P] [US1] Criar `internal/domain/frame_overlay_test.go`: **traçado** — a poligonal do quadro `k` tem os vértices dos marcadores dos quadros `0…k` (o quadro 0 não tem segmento visível); cada vértice fica na altura do terreno mais `TrailLiftMeters`; o traçado do quadro `k` contém, pixel a pixel, o do quadro `k−1` (cobertura ≥) quando só o segmento novo é acrescentado; segmentos inteiros atrás da câmera são descartados e os que cruzam o plano de aproximação (0,1 m) são recortados sem exceção; **cápsula** — cobertura `clamp(meia_largura + 0,5 − distância, 0, 1)`: 1 no eixo, 0 além da meia largura + 1 px, valores intermediários na borda; largura `max(2, 0,005·altura)` px; **marcador** — o centro do disco cai a ≤ 1 px da projeção do ponto do marcador (terreno plano, várias câmeras); raio `max(4, 0,012·altura)` px; anel de `max(1,5, 0,003·altura)` px; **oclusão** — traçado e marcador só aparecem onde `distância ≤ t_terreno + (DepthBiasMeters + DepthBiasRatio·t_terreno)`; um marcador atrás de um morro (t_terreno menor) não é desenhado; um traçado rente ao chão (levantado 0,3 m) aparece por inteiro sobre terreno plano; o traçado é desenhado antes do marcador (o marcador cobre o traçado no centro)
- [X] T036 [P] [US1] Criar `internal/domain/frame_scene_test.go` (cenas pequenas: grade 40×40 de células de 0,0005°, resolução 64×36 e 128×72, `TileDecoder` mockado devolvendo cor lisa): `NewScene` sobre um recorte sem nenhuma amostra com valor não é testado aqui (a recusa é da US7); `Render` devolve uma imagem com as dimensões pedidas e `FrameStats` zerados numa cena completa; a metade **acima do horizonte** (câmera com inclinação de 20°) sai na cor de fundo `#20262E` e a de baixo na cor da peça; a câmera do quadro é a do plano (rumo e inclinação): girar o rumo em 180° espelha uma cena simétrica; **determinismo** — `Render` com `Workers = 1` e com `Workers = 8` dá `Pix` idêntico byte a byte; renderizar os quadros `[3, 1, 2]` em qualquer ordem dá a mesma imagem de cada quadro que renderizá-los isolados; duas chamadas iguais dão o mesmo resultado; **hash de referência** — o SHA-256 de `Pix` de um quadro pequeno é uma constante gravada no teste (protege contra fusão multiplicação-soma e mudanças de arquitetura; a constante é registrada na primeira execução verde e só muda com `RenderVersion`); **cancelamento** — um `context` já cancelado devolve `context.Canceled` sem imagem, e um cancelado durante o desenho devolve `context.Canceled` (o teste cancela num `TileDecoder` mockado que bloqueia); **campo de visão** — o mesmo quadro em 64×36 e em 128×72 tem o horizonte na mesma fração da altura (± 1 linha da menor); **peça ilegível** — um `Decode` que falha faz `Render` devolver `ErrSliceFileInvalid` sem imagem
- [X] T037 [P] [US1] Criar `internal/infra/outbound/zipfile/geo_slice_reader_test.go` (pacote `zipfile_test`, com `test/helper` — nunca com o exportador): `Read` de `ValidSliceFile(DefaultSliceFileSpec())` devolve um `GeoSlice` com a `Area` (inclusive `CrossesAntimeridian`), os `TileSets` (fonte, formato `png`, `Detail` com `Ideal/Chosen/Min/Max/Reason`, peças com `ID{Level: chosen, X, Y}` e os **bytes exatos**, peças ausentes), as `Elevation` (fonte, `NorthLatitude`, `WestLongitude`, `CellLatitude`, `CellLongitude`, linhas, colunas e os valores, com o NaN preservado como "sem valor" — `At` devolve `hasValue == false` e `NoValueCount` bate) e o `Summary` idêntico ao do manifesto; `PlanID` igual ao do manifesto; `ContentID` é o SHA-256 hex do arquivo inteiro e muda quando muda um byte de uma peça; um recorte com duas fontes de mapa e duas grades; uma área que cruza o antimeridiano; um arquivo inexistente devolve um erro de E/S (`errors.Is(err, fs.ErrNotExist)`, sem sentinela do domínio)
- [X] T038 [P] [US1] Criar `internal/infra/outbound/tiledecoder/raster_tile_decoder_test.go`: decodifica `PNGTile`, `CheckerPNGTile`, `JPEGTile` (cor lisa dentro de ±2 por causa do JPEG) e `WebPTile` (1×1) para `TileImage` RGBA com as dimensões e os pixels esperados; o alfa de `TransparentPNGTile` é preservado; um formato desconhecido (`"gif"`, `"pbf"`, `""`) e `NotAnImage()` falham com erro (o decodificador **não** conhece sentinelas: quem embrulha é o domínio); um PNG truncado falha; duas decodificações dão o mesmo `Pix`
- [X] T039 [P] [US1] Criar `internal/infra/outbound/pngfile/png_mark_test.go`: `encodeFrame(image, id)` gera um PNG que `image/png` decodifica para os mesmos pixels RGB; o arquivo traz **um** bloco `tEXt` `Sobrevoo` = `frame-set=<id>` **logo depois do `IHDR`** e nenhum outro bloco auxiliar; a assinatura, o `IHDR` (largura, altura, profundidade 8, tipo de cor RGB, sem entrelaçamento) e o `IEND` no fim estão corretos; o CRC do bloco `tEXt` confere; duas codificações iguais dão os mesmos bytes; ids diferentes dão arquivos diferentes; `readMark(bytes)` devolve `(id, true)` para um quadro da ferramenta e `("", false)` para um PNG comum, para um não-PNG e para bytes truncados no meio do cabeçalho
- [X] T040 [P] [US1] Criar `internal/infra/outbound/pngfile/frame_exporter_test.go` (diretório temporário): `Export` grava um PNG válido com as dimensões e a marca do conjunto; o arquivo gravado é idêntico ao que `encodeFrame` produz (o mesmo que o repositório grava, para a garantia byte a byte do FR-018a); nenhum `.sobrevoo-*.tmp` fica no diretório (os cenários de existente/sobrescrita/inválido entram na US6)
- [X] T041 [P] [US1] Estender `internal/application/geo_slice_service_test.go`: `Load(path)` chama `GeoSliceReader.Read` (mock) e devolve o recorte; propaga `ErrSliceFileInvalid`, `ErrSliceFormatVersionUnsupported` e um erro de E/S sem alterá-los (`errors.Is` preservado); o construtor recebe o leitor
- [X] T042 [P] [US1] Criar `internal/application/frame_service_test.go` (só `DrawFrame`; `TileDecoder` e `FrameExporter` mockados, cena real pequena): desenha o quadro `Number`, chama `FrameExporter.Export` **uma vez** com a imagem na resolução pedida, o `FrameSetID` de `NewFrameSetID(plan, slice, resolution, tuning)`, o caminho e o `Overwrite` do pedido; devolve `RenderSummary{Requested: 1, Drawn: 1, Resolution: ...}` com `Elapsed > 0`; um erro de `Export` sobe sem alteração e sem contar como desenhado; um `context` cancelado devolve `ErrRenderInterrupted` (com `Interrupted: true` no resumo) sem chamar `Export`; o número do pedido é o índice do plano (`plan.Frames[Number]`)
- [X] T043 [P] [US1] Criar `internal/infra/inbound/cli/render_frame_test.go` (`FrameService`, `CameraPlanService` e `GeoSliceService` mockados): `render frame <plano> <recorte> --number 300 --output /tmp/q.png` chama `CameraPlanService.Load(plano)`, depois `ParseFrameNumber("300", plan frame count)`, depois `GeoSliceService.Load(recorte)`, depois `FrameService.DrawFrame` com a resolução padrão (a de `RenderDefaults`, injetada no construtor) e imprime o resumo do contrato (`Frame 300 of 1260 drawn to /tmp/q.png`, `Resolution: 1920x1080`, `Time: HH:MM:SS`, `Holes: ...`) em `stdout`; `--number` e `--output` obrigatórias (ausência → `usageError`, código 2); argumentos posicionais diferentes de 2 → `usageError`; `--number 3.5` → `ErrFrameOutOfRange` (código 33) **sem** carregar o recorte; um erro de cada serviço sobe com o código certo (17, 18, 27–33, 4); nada é impresso em `stdout` quando há erro

### Implementação da User Story 1

- [X] T044 [US1] Criar `internal/domain/frame_camera.go`: o plano local do quadro (`newFramePlane(latitude, longitude float64)`: `toPlane(lat, lon) (x, y)` com o desembrulho de longitude em `[-180, 180)` e `cos φ₀ ≥ 1e-6`; `toGeo(x, y) (lat, lon)`, afim), a `camera` do quadro (`newCamera(frame CameraFrame, absoluteHeight float64, resolution Resolution, fovDegrees float64)`: posição `(0, 0, H)`, base `frente/direita/cima` de `research.md` item 3, `f_px`, `rayFor(i, j int) (dx, dy, dz float64)` normalizado com o centro do pixel em `+0,5`, `project(x, y, z float64) (px, py, depth float64, ok bool)`), `targetOf(frame, tuning)` (alvo recuperado do quadro) e `referenceHeight(surface, frame, tuning)` (`H = terreno(alvo) + altitude`, com a folga mínima sobre o terreno da câmera); todo produto que entra numa soma em `float64(...)` (depende de T031)
- [X] T045 [US1] Criar `internal/domain/frame_surface.go`: `surface` (uma por `ElevationGrid` do recorte): retângulo no plano do quadro (`x₀`, `y₀`, `cx`, `cy` a partir de `WestLongitude` normalizada em relação à longitude do quadro, `NorthLatitude` e as células), nós preenchidos (`filled []float32` só quando há buraco; preenchimento por duas passadas L1 de `research.md` item 6, com a regra da grade inteira sem valor), `hole(r, c)`, `zmax`, `heightAt(x, y)` (bilinear entre centros, índices presos) e `cellAt(x, y) (row, col)`; a construção do preenchimento é feita **uma vez** em `NewScene` (a superfície guarda os nós em coordenadas de grade, e o retângulo no plano do quadro é calculado por quadro, barato) (depende de T032)
- [X] T046 [US1] Editar `internal/domain/frame_surface.go`: `(s *surface) trace(ox, oy, oz, dx, dy, dz float64) (hit surfaceHit, ok bool)` — recorte do raio ao retângulo pelo método das lajes, poda por `zmax` (raio acima de `zmax` que sobe ou fica horizontal não acerta; que desce começa no `t` em que atinge `zmax`), DDA de Amanatides-Woo por quadra no espaço `(a, b)` dos nós (quadras `−1…rows−1` × `−1…cols−1`), teste `f(t_entrada) ≤ 0` (acerto em `t_entrada`), `f(t_saída) ≤ 0` (16 bisseções entre `t_entrada` e `t_saída`), `surfaceHit{T, X, Y, Row, Col}`; uma variável de pacote de teste desliga a poda por `zmax`; e `traceScene(surfaces []*surface, ...)` que percorre as grades cujo retângulo o raio cruza, **na ordem do `t` de entrada** (empate: índice), e devolve o primeiro acerto (depende de T045, T033)
- [X] T047 [US1] Criar `internal/domain/frame_imagery.go`: `imagery` (construída em `NewScene` a partir dos `TileSet`s: índice de peças presentes e ausentes por conjunto, sem iterar `map` para produzir valor) com `mercator(lat, lon) (u, v float64)` (`v = 1/2 − ln(tan(π/4 + φ/2))/(2π)` com `math.Log` e `math.Tan`), `lookup(lat, lon)` (regra de `research.md` item 7: primeira presente na ordem dos conjuntos; senão ausente; senão "sem peça"), a pirâmide de mips por peça (caixa 2×2 com `(a+b+c+d+2)>>2`), o filtro trilinear (`λ = log2(max(ρ, 1))`) com cruzamento de peças vizinhas presentes e o texel mais próximo da peça do centro quando a vizinha está ausente, a transparência composta sobre `BackgroundColor`, e o cache de peças decodificadas com orçamento `TileCacheBytes` (descarta as mais antigas; **uma** decodificação por peça mesmo sob concorrência, com um mutex e uma entrada `sync.Once` por peça); `TileDecoder.Decode` chamado sob demanda; o erro da peça vira `fmt.Errorf("%w: base map %q level %d x=%d y=%d: %w", ErrSliceFileInvalid, ...)` (depende de T013, T034)
- [X] T048 [US1] Criar `internal/domain/frame_overlay.go`: a poligonal do traçado do quadro `k` (vértices dos marcadores `0…k` no plano do quadro, na altura `surface + TrailLiftMeters`), o recorte pelo plano de aproximação (0,1 m), a cápsula com cobertura suave em dois passes (contorno `TrailCasingColor` e miolo `TrailColor`), o marcador (disco `MarkerColor` com anel `MarkerRingColor`, uma profundidade só — a do centro), o teste de profundidade `d ≤ t_terreno + DepthBiasMeters + DepthBiasRatio·t_terreno` contra o buffer de profundidade do terreno, e a mistura com aritmética de `float64` e arredondamento `math.Floor(v + 0.5)` para inteiros de 8 bits (depende de T044, T035)
- [X] T049 [US1] Criar `internal/domain/frame_scene.go`: `Scene`, `NewScene(slice GeoSlice, decoder TileDecoder, tuning RenderTuning) (*Scene, error)` (constrói `surface`s, `imagery` e os índices; segura para uso concorrente) e `(s *Scene) Render(ctx context.Context, plan CameraPlan, index int, resolution Resolution) (FrameImage, FrameStats, error)`: câmera do quadro (T044), buffer de profundidade `float32` do tamanho da imagem, faixas de 16 linhas distribuídas a `Workers` goroutines (contador atômico), por pixel: raio → `traceScene` → cor do mapa (`imagery`) ou fundo, sem marcações (pixels sem peça ou sobre célula sem valor saem na cor de fundo até a US3); `ctx` verificado ao começar cada faixa (cancelamento devolve `ctx.Err()`); depois o traçado e o marcador (T048) na ordem; `FrameStats` zerado por ora; um erro de peça de qualquer goroutine cancela as demais e é devolvido (depende de T045, T046, T047, T048, T036, T018)
- [X] T050 [US1] Criar `internal/domain/frame_scene_bench_test.go`: `BenchmarkScene_Render_1080p` (grade sintética de 2000×2000 células de 30 m sobre relevo ondulado, 1 fonte de mapa com peças PNG de cor lisa em mosaico, câmera a 600 m com inclinação de 45°, resolução 1920×1080, `Workers = runtime.NumCPU()`) e `BenchmarkScene_Render_1080p_LowTilt` (inclinação de 25°, o pior caso de raios longos); rodar `go test ./internal/domain -run '^$' -bench Scene -benchtime 3x` e **registrar os tempos em "Notas de implementação" ao fim deste arquivo**; se um quadro de 1080p passar de **2 s** a meta de `plan.md` (R1) está em risco: abrir a pirâmide de máximos de `research.md` item 5 como tarefa nova, mantendo o teste de igualdade da poda (depende de T049)
- [X] T051 [US1] Criar `internal/infra/outbound/zipfile/geo_slice_reader.go`: `GeoSliceReader` e `NewGeoSliceReader()` (construtor pela porta, tecnologia ZIP); `Read(path)` abre com `archive/zip` (acesso aleatório), lê `manifest.json` (`encoding/json`, campos desconhecidos ignorados), monta as fontes (`GeoDataSource{Name, Type, Format, Path}`), `TileSet`s (`Detail{Ideal, Chosen, Min, Max, Reason}`, peças com os bytes lidos da entrada, ausentes) e `ElevationGrid`s (`NewElevationGrid(source, GridWindow{0, 0, rows, cols}, ElevationGridInfo{Rows, Cols, NorthLatitude, WestLongitude, CellLatitude, CellLongitude, 1}, values)` com os `float32` little-endian lidos e o NaN preservado), chama `domain.NewGeoSlice` e preenche `PlanID` (do manifesto) e `ContentID` (SHA-256 hex do arquivo lido em sequência); erro de E/S sobe embrulhado sem sentinela; **a validação completa (contagens, tamanhos, versão, entradas sobrando etc.) entra na US7** — aqui só o que o caminho feliz exige e `ErrSliceFileInvalid` para "não é um ZIP" e "sem `manifest.json`" (depende de T037, T023)
- [X] T052 [P] [US1] Criar `internal/infra/outbound/tiledecoder/raster_tile_decoder.go`: `Raster` e `NewRaster()` (pacote leva o nome da porta; tipo/construtor só a estratégia); `Decode(format, data)` usa `image/png`, `image/jpeg` e `golang.org/x/image/webp` conforme `format`, converte para RGBA com `draw.Draw` num `image.RGBA` (alfa preservado, sem pré-multiplicação: usar `image.NRGBA` → RGBA não pré-multiplicado, ver o teste de transparência) e devolve `domain.TileImage`; formato desconhecido e bytes ilegíveis → erros comuns (sem sentinela) (depende de T038, T013)
- [X] T053 [P] [US1] Criar `internal/infra/outbound/pngfile/png_mark.go`: `encodeFrame(image domain.FrameImage, id domain.FrameSetID) ([]byte, error)` — copia os pixels para um `*image.NRGBA` opaco (o `image/png` escolhe RGB de 8 bits, sem alfa, quando todos os pixels são opacos), codifica com `png.Encoder{CompressionLevel: png.DefaultCompression}` (entrelaçamento nenhum) e **insere** o bloco `tEXt` `Sobrevoo\0frame-set=<id>` (tipo + dados com CRC-32 IEEE) imediatamente depois do `IHDR`; `readMark(header []byte) (id domain.FrameSetID, ok bool)` (confere a assinatura e o `IHDR` e percorre os blocos até o primeiro `IDAT` procurando o `tEXt` `Sobrevoo`); `readInfo(head, tail []byte) (width, height int, complete bool)` (largura e altura do `IHDR` e se os últimos 12 bytes são o `IEND`); só estas funções conhecem o formato PNG do arquivo (depende de T039, T018, T022)
- [X] T054 [US1] Criar `internal/infra/outbound/pngfile/frame_exporter.go`: `FrameExporter` e `NewFrameExporter()` (construtor pela porta); `Export(image, id, path, overwrite)` codifica com `encodeFrame` e publica com `atomicfile.Publish(path, overwrite, ...)`, traduzindo `atomicfile.ErrExists` → `domain.ErrFrameDestinationExists` (`"%w: %s (use --overwrite to replace it)"`) e `atomicfile.ErrInvalid` → `domain.ErrFrameDestinationInvalid` (`"%w: %s: %w"`), como `zipfile.GeoSliceExporter.Export` (depende de T053, T040, T003)
- [X] T055 [US1] Editar `internal/application/geo_slice_service.go`: o construtor `NewGeoSliceService` recebe também `domain.GeoSliceReader` (campo `reader`), e o método `Load(path string) (domain.GeoSlice, error)` na interface `GeoSliceService` (comentário: lê o arquivo de recorte que `Export` gravou e o valida) chama `s.reader.Read(path)`; rodar `make generate` (regera `mockapplication/geo_slice_service.go`); ajustar todos os chamadores do construtor (`main.go` em T058; testes de `geo_slice_service_test.go`) (depende de T041, T023)
- [X] T056 [US1] Criar `internal/application/frame_service.go`: diretiva `//go:generate ... -destination mockapplication/frame_service.go -package mockapplication . FrameService` (no mesmo lugar em que `geo_slice_service.go` a declara); interface `FrameService` (`DrawFrame(ctx context.Context, plan domain.CameraPlan, slice domain.GeoSlice, request domain.SingleFrameRequest) (domain.RenderSummary, error)`; `DrawFrames(ctx, plan, slice, request domain.FrameSetRequest, progress func(domain.RenderProgress)) (domain.RenderSummary, error)`, com o comentário de `data-model.md`), struct `frameService` e `NewFrameService(decoder domain.TileDecoder, repository domain.FrameRepository, exporter domain.FrameExporter, renderTuning domain.RenderTuning, sliceTuning domain.SliceTuning) FrameService`; `DrawFrame`: `id := domain.NewFrameSetID(...)`, `scene, err := domain.NewScene(slice, s.decoder, s.renderTuning)`, `image, stats, err := scene.Render(ctx, plan, request.Number, request.Resolution)`, `s.exporter.Export(image, id, request.Path, request.Overwrite)`, `summary.Add(stats)`; mede `Elapsed` com `time.Now()`/`time.Since`; um `ctx` cancelado devolve `ErrRenderInterrupted` com `Interrupted: true`; `DrawFrames` devolve, até a US2, `errors.New("DrawFrames is not implemented yet")`; rodar `make generate` (depende de T055, T049, T054, T042)
- [X] T057 [US1] Criar `internal/infra/inbound/cli/render.go` (`NewRenderCommand()`: comando pai `render`, sem ação própria, como `geodata.go`) e `internal/infra/inbound/cli/render_frame.go` (`NewRenderFrameCommand(cameraPlanService, geoSliceService, frameService, defaultResolution domain.Resolution)`): `Use: "render frame <plan-file> <slice-file>"`, `SilenceErrors`/`SilenceUsage`, `Args` exatamente 2 (`newUsageError`), flags `--number` (texto, obrigatória), `--output` (obrigatória) e `--overwrite`; `runRenderFrame`: `Load` do plano, `ParseFrameNumber`, `Load` do recorte, `DrawFrame`, e o resumo do contrato de `contracts/cli.md` (`formatFrameSummary(number, total, path, summary)`: rótulos e ordem estáveis; `Time` em `HH:MM:SS`; `Holes: map tiles missing: yes|no, elevation without value: yes|no`); nada em `stdout` quando há erro (depende de T056, T043, T012)
- [X] T058 [US1] Editar `cmd/sobrevoo/main.go`: criar `zipfile.NewGeoSliceReader()`, `tiledecoder.NewRaster()` e `pngfile.NewFrameExporter()`; passar o leitor a `NewGeoSliceService`; criar `application.NewFrameService(decoder, nil, exporter, domainRenderTuning(cfg.RenderTuning), domainSliceTuning(cfg.SliceTuning))` (o repositório é `nil` até a US2; `DrawFrame` não o usa); resolver `domainRenderResolution(cfg.RenderDefaults)` (erro → `stderr` e código 4, como `config.Load`); montar `renderCommand := cli.NewRenderCommand()`, `renderCommand.AddCommand(cli.NewRenderFrameCommand(...))` e `root.AddCommand(renderCommand)`; conferir que `make build` compila (depende de T057, T026)

**Checkpoint US1**: `make test`, `make lint` verdes; `bin/sobrevoo render frame`
funciona; `quickstart.md` itens 1 e 2 (a partir de `test/samples`, que o
Polish estende; até lá, com uma fixture gravada por um programa de teste
temporário ou com os dados sintéticos da etapa 4 trocando o mapa por um
gerado por `helper.PositionPNGTile`) — o **MVP** está entregue.

---

## Phase 4: User Story 2 - Desenhar todos os quadros do voo (Priority: P2)

**Goal**: `render all` desenha todos os quadros do plano num diretório, com a
numeração e a ordem do plano, e imprime o resumo.

**Independent Test**: `quickstart.md` item 4: plano curto (6 s a 10 fps),
`render all --output /tmp/quadros --resolution 960x540` (a flag só chega na
US4; até lá, a resolução padrão) → 60 arquivos `frame_000000.png` a
`frame_000059.png`, cada quadro idêntico ao isolado.

### Testes da User Story 2

- [X] T059 [P] [US2] Criar `internal/infra/outbound/pngfile/frame_repository_test.go` (diretório temporário; só `Save`): `Save(dir, 7, id, image)` grava `dir/frame_000007.png` com o mesmo conteúdo que `encodeFrame(image, id)`; cria o diretório quando não existe (a pasta-mãe existe); um caminho que existe e **não** é diretório → `ErrFrameDestinationInvalid`; uma pasta-mãe inexistente → `ErrFrameDestinationInvalid`; sem permissão de escrita (`0o500`) → `ErrFrameDestinationInvalid` (pulado como root/Windows com `t.Skip`); nenhum `.sobrevoo-*.tmp` fica; `Save` **substitui** um arquivo existente (a decisão de sobrescrever é do domínio)
- [X] T060 [P] [US2] Estender `internal/application/frame_service_test.go` (só o caminho de destino vazio; `FrameRepository` e `TileDecoder` mockados, cena real pequena): `DrawFrames` desenha todos os quadros do plano **em ordem crescente**, chamando `Save(dir, i, id, image)` uma vez por quadro (a ordem das chamadas é `0…N−1`); devolve `RenderSummary{Requested: N, Drawn: N, Kept: 0, Resolution}`; um erro de `Save` interrompe, devolve o resumo parcial (quadros já salvos contados em `Drawn`) e o erro sem alteração; um `context` cancelado antes do primeiro quadro devolve `ErrRenderInterrupted` sem chamar `Save` (a retomada e o progresso entram na US5); o teste que esperava o erro "not implemented" da US1 é removido
- [X] T061 [P] [US2] Criar `internal/infra/inbound/cli/render_all_test.go` (serviços mockados): `render all <plano> <recorte> --output /tmp/q` chama `Load` do plano, `Load` do recorte e `DrawFrames` com `FrameSetRequest{Directory, Resolution: padrão}`, e imprime o resumo do contrato (`Frames: 60 requested, 60 drawn, 0 kept (already in the destination)`, `Resolution: 1920x1080`, `Time`, `Holes (in the frames drawn now): ...`, `Destination: /tmp/q (frame_000000.png to frame_000059.png)`); `--output` obrigatória; argumentos ≠ 2 → uso (código 2); erros dos serviços sobem com os códigos certos e nada em `stdout`

### Implementação da User Story 2

- [X] T062 [US2] Criar `internal/infra/outbound/pngfile/frame_repository.go`: `FrameRepository` e `NewFrameRepository()` (construtor pela porta); `Save(dir, index, id, image)`: `os.MkdirAll` só do último nível (a pasta-mãe precisa existir: use `os.Mkdir` quando o diretório não existe e traduza a falha), `os.Stat` que recusa um caminho que não é diretório, `atomicfile.Publish(filepath.Join(dir, domain.FrameFileName(index)), true, ...)` com `encodeFrame`; falhas → `ErrFrameDestinationInvalid` com a causa; `Inspect` e `Remove` devolvem `errors.New("not implemented")` até a US5 e a US6 (os testes correspondentes só entram nelas) (depende de T059, T053)
- [X] T063 [US2] Editar `internal/application/frame_service.go`: `DrawFrames`: `id := NewFrameSetID(...)`, `scene := NewScene(...)`, para `i := 0…len(plan.Frames)−1`: `ctx` cancelado → `ErrRenderInterrupted` (resumo `Interrupted`), `scene.Render`, `s.repository.Save(request.Directory, i, id, image)`, `summary.Add(stats)`; resumo com `Requested = len(plan.Frames)`, `Elapsed`; erro de `Render` ou `Save` devolve o resumo parcial e o erro; o argumento `progress` é aceito e ainda não é chamado (US5) (depende de T060, T062)
- [X] T064 [US2] Criar `internal/infra/inbound/cli/render_all.go` (`NewRenderAllCommand(cameraPlanService, geoSliceService, frameService, defaultResolution)`): `Use: "render all <plan-file> <slice-file>"`, flags `--output` (obrigatória) e `--overwrite` (aceita e repassada em `FrameSetRequest.Overwrite`; o comportamento é da US6), `formatFramesSummary(total, dir, summary)` do contrato; passa `nil` como `progress` por ora (US5) (depende de T063, T061)
- [X] T065 [US2] Editar `cmd/sobrevoo/main.go`: substituir o `nil` do repositório por `pngfile.NewFrameRepository()` e acrescentar `renderCommand.AddCommand(cli.NewRenderAllCommand(...))`; `make build` (depende de T064)

**Checkpoint US2**: `render all` desenha o voo num destino vazio; `quickstart.md`
item 4 (sem `--resolution`).

---

## Phase 5: User Story 3 - Ver a falta de dado de forma explícita e contabilizada (Priority: P3)

**Goal**: pixels sem imagem de mapa e sobre células sem elevação aparecem com
as marcações do contrato, e o resumo conta os quadros afetados por causa.

**Independent Test**: `quickstart.md` item 3: o quadro 0 de `pedalada.gpx` com
o mapa em imagem (3 peças ausentes) e o relevo (bloco sem dado): hachura cinza
e xadrez magenta nos lugares certos; `Holes: ... yes`; o quadro do meio sem
nenhuma.

### Testes da User Story 3

- [X] T066 [P] [US3] Estender `internal/domain/frame_scene_test.go` (cenas pequenas): **peça ausente** — pixels de terreno cuja peça está em `Missing` saem na hachura (`#C8C8C8` onde `(x + y) mod 12 < 6`, `#6E6E6E` no resto, com `x` e `y` os do pixel na imagem), o relevo continua (a posição do horizonte e o acerto não mudam) e `FrameStats.MapHole == true`; **peça desconhecida** (fora dos intervalos pedidos, ou latitude além de ±85,0511°) → a mesma hachura e `MapHole`; **célula sem valor** — pixels cujo acerto cai numa célula `NaN` saem no xadrez (`#FF00FF` onde `(⌊x/12⌋ + ⌊y/12⌋) mod 2 == 0`, `#3A003A` no resto) e `ElevationHole == true`; **prevalência** — um pixel sobre célula sem valor **e** peça ausente é xadrez, e o quadro só tem `MapHole` se **algum outro** pixel for de peça ausente; **quadro sem faltas** → os dois `false` e **nenhum** pixel com as cores de hachura ou xadrez (as peças de teste usam outras cores); **fundo** — o raio que não acerta terreno é `#20262E` e **nunca** conta como buraco (câmera olhando para o horizonte com o recorte terminando antes); **todas as peças ausentes** → todo pixel de terreno é hachura e `MapHole`; **traçado e marcador sobre uma região de falta** não escondem a contagem (a contagem é feita antes da sobreposição); o `Render` continua idêntico com 1 e com 8 goroutines com marcações; um recorte sem peças mas com relevo (`TileSets` vazio) → tudo hachura
- [X] T067 [P] [US3] Estender `internal/application/frame_service_test.go`: `DrawFrame` e `DrawFrames` somam `MapHoleFrames`/`ElevationHoleFrames` do `FrameStats` de cada quadro **desenhado** (uma cena com buraco de mapa só nos quadros 0–1 e de elevação só no quadro 2 dá `MapHoleFrames == 2`, `ElevationHoleFrames == 1`, e um quadro com as duas causas entra nas duas contagens)
- [X] T068 [P] [US3] Estender `render_frame_test.go` e `render_all_test.go`: `Holes: map tiles missing: yes, elevation without value: no` no quadro isolado; `Holes (in the frames drawn now): 87 with missing map tiles, 4 with elevation without value` em `render all`, e `none drawn now` quando `Drawn == 0`

### Implementação da User Story 3

- [X] T069 [US3] Editar `internal/domain/frame_imagery.go` (`lookup` devolve o estado `image`/`noMap`, com a regra de `research.md` item 7) e `internal/domain/frame_scene.go`: cada pixel de terreno é classificado (`noElevation` se `surface.hole(row, col)` na célula do acerto, senão `noMap` se `imagery.lookup` não achou peça presente, senão `image`), pintado com a hachura, o xadrez ou a cor do mapa (a posição do pixel decide o tom), e os booleanos `MapHole`/`ElevationHole` de `FrameStats` são o OU dos pixels de cada faixa (sem estado compartilhado que dependa da ordem); o fundo não é classificado; a contagem é feita **antes** do traçado e do marcador (depende de T066)
- [X] T070 [US3] Editar `render_frame.go` e `render_all.go`: a linha `Holes` dos resumos do contrato (`formatFrameSummary`, `formatFramesSummary`); os testes de T067 passam com o que T020 e T056/T063 já fazem (`summary.Add(stats)`) assim que T069 preenche `FrameStats` (depende de T069, T068)

**Checkpoint US3**: `quickstart.md` item 3.

---

## Phase 6: User Story 4 - Escolher a resolução da imagem (Priority: P4)

**Goal**: `--resolution LxA` nos dois comandos, com o enquadramento vertical
igual em qualquer resolução.

**Independent Test**: `quickstart.md` item 8.

### Testes da User Story 4

- [X] T071 [P] [US4] Estender `internal/domain/frame_scene_test.go`: o mesmo quadro em 64×36 e 128×72 tem o marcador na mesma posição relativa da imagem (diferença ≤ 1 px na menor) e a linha do horizonte na mesma fração da altura (± 1 linha da menor); em retrato (36×64) e ultralargo (128×36) o `Render` funciona, a altura do enquadramento vertical é a mesma (mesmo `f_px/altura`) e só muda quanto se vê dos lados; a resolução padrão 1920×1080 renderiza sem erro numa cena pequena (com `Workers = 2`); uma resolução de 8 294 400 pixels aloca o buffer do tamanho certo (`len(Pix) == 3·W·H`)
- [X] T072 [P] [US4] Estender `render_frame_test.go` e `render_all_test.go`: `--resolution 960x540` chega em `SingleFrameRequest.Resolution`/`FrameSetRequest.Resolution` como `Resolution{960, 540}` e aparece no resumo; sem a flag, a resolução é a padrão; `--resolution 1921x1080`, `8000x4500`, `abc`, `0x0`, `-960x540` e `960` → `ErrInvalidResolution` (código 34) **antes** de qualquer `Load` (nenhum serviço é chamado)

### Implementação da User Story 4

- [X] T073 [US4] Editar `render_frame.go` e `render_all.go`: a flag `--resolution` (texto; padrão `<L>x<A>` da resolução padrão injetada; a ajuda cita os limites: pares, 180 a 3840 por lado, até 8 294 400 pixels), com `domain.ParseResolution` chamado **primeiro**, antes de qualquer `Load`; e, se T071 falhar, corrigir `frame_camera.go` e `frame_scene.go` para que só a proporção use a largura (o campo de visão vertical é fixo) (depende de T072, T071, T049)

**Checkpoint US4**: `quickstart.md` item 8.

---

## Phase 7: User Story 5 - Acompanhar o progresso e retomar de onde parou (Priority: P5)

**Goal**: o progresso aparece a cada quadro; a execução pode ser interrompida
sem imagem parcial; o mesmo comando retoma de onde parou, sem redesenhar o que
já está pronto e é do mesmo conjunto.

**Independent Test**: `quickstart.md` item 5.

### Testes da User Story 5

- [X] T074 [P] [US5] Estender `internal/domain/frame_set_test.go` (com `FrameDirectoryBuilder`): `FrameDirectory.Plan(id, frameCount, false)` — diretório vazio → `Draw = 0…N−1`, `Keep` vazio; todos os quadros nossos, do mesmo conjunto e completos → `Keep = 0…N−1`, `Draw` vazio; parte deles → `Keep` os presentes, `Draw` os que faltam, em ordem crescente, disjuntos e a união é `0…N−1`; um quadro nosso do mesmo conjunto **incompleto** vai para `Draw`; um quadro nosso do mesmo conjunto com número **fora** do plano é ignorado (nem `Keep` nem `Remove`; a US6 trata o de outro conjunto)
- [X] T075 [P] [US5] Estender `internal/infra/outbound/pngfile/frame_repository_test.go`: `Inspect(dir, resolution)` — diretório inexistente → `FrameDirectory` vazio sem erro; um caminho que não é diretório → `ErrFrameDestinationInvalid`; lista só os nomes `frame_NNNNNN.png` (ignora `notas.txt`, `frame_1.png`, `.sobrevoo-x.tmp`), **ordenados por número**; para cada um, `Ours` e `SetID` vêm da marca (`readMark`), e `Complete` só é `true` se o `IHDR` tem a largura e a altura da resolução pedida e o `IEND` está no fim; um quadro gravado por `Save` é `Ours`, com o `SetID` certo e `Complete`; um truncado (`head -c 1000`) é `Ours` (a marca vem antes) mas `!Complete`; um PNG comum com nome de quadro é `!Ours`; um arquivo vazio com nome de quadro é `!Ours` e `!Complete`; outra resolução → `!Complete`
- [X] T076 [P] [US5] Estender `internal/application/frame_service_test.go`: `DrawFrames` chama `Inspect(dir, resolution)` **antes** de desenhar; com `Keep = {0, 1}` (mock de `Inspect`) desenha só `2…N−1`, não chama `Save` para os mantidos, e `RenderSummary{Requested: N, Drawn: N−2, Kept: 2}`; com todos mantidos, não cria `Scene` nem chama `Render` (o mock de `TileDecoder` não é chamado) e devolve `Drawn: 0, Kept: N`; o callback `progress` é chamado **uma vez por quadro desenhado**, com `RenderProgress{Done: mantidos + desenhados até ali, Total: N, Elapsed}` crescente; um `ctx` cancelado no meio (cancelado pelo callback de progresso do quadro 3) devolve `ErrRenderInterrupted`, `Interrupted: true`, `Drawn: 3` e nenhum `Save` além do último completo; um `Inspect` que falha sobe sem desenhar nada
- [X] T077 [P] [US5] Estender `render_all_test.go` e criar `internal/infra/inbound/cli/render_progress_test.go`: o progresso vai para `stderr`: com `stderr` não terminal, uma linha `Drawing frame 10/60 (16.7%), elapsed 00:00:04` a cada 10 quadros e no último; com `stderr` terminal (um `io.Writer` que declara ser terminal, via uma função injetada no construtor), uma linha reescrita com `\r`; o contexto do comando é cancelado por `SIGINT`/`SIGTERM` (o teste envia o sinal a um `signal.NotifyContext` injetado, ou cancela o contexto passado a `ExecuteContext`) e, ao voltar `ErrRenderInterrupted` com o resumo, o comando imprime `Interrupted: 412 of 1260 frames are ready; run the same command again to continue`, o resumo e devolve `ErrRenderInterrupted` (código 38); `Kept` aparece no resumo (`12 kept (already in the destination)`) e `Holes ... none drawn now` quando nada foi desenhado

### Implementação da User Story 5

- [X] T078 [US5] Editar `internal/domain/frame_set.go`: `(d FrameDirectory) Plan(id FrameSetID, frameCount int, overwrite bool) (FrameWork, error)` — por ora **só o caminho sem conflito**: `Keep` = nossos do mesmo conjunto, completos e dentro do plano; `Draw` = o resto de `0…frameCount−1`, em ordem crescente; a US6 acrescenta o conflito e a sobrescrita (depende de T074)
- [X] T079 [US5] Editar `internal/infra/outbound/pngfile/frame_repository.go`: `Inspect(dir, resolution)` (lê o diretório, filtra com `domain.ParseFrameFileName`, e para cada arquivo lê os primeiros bytes (até o primeiro `IDAT`, no máximo 4 KiB) com `readMark` e os 12 últimos com `readInfo`) e devolve `FrameDirectory` ordenado; um diretório inexistente é `FrameDirectory{}`; não lê o arquivo inteiro (depende de T075, T053)
- [X] T080 [US5] Editar `internal/application/frame_service.go`: `DrawFrames` chama `Inspect`, `FrameDirectory.Plan(id, N, request.Overwrite)`, mede o tempo, só cria a `Scene` se houver o que desenhar, desenha `work.Draw` em ordem, chama `progress` após cada quadro (`Done = len(work.Keep) + desenhados`), verifica `ctx` antes de cada quadro e converte cancelamento em `ErrRenderInterrupted` com `Interrupted: true`; o resumo tem `Kept = len(work.Keep)` (depende de T076, T078, T079, T063)
- [X] T081 [US5] Editar `render_all.go`: criar o contexto com `signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)` (com `defer stop()`), passar o callback de progresso que escreve em `stderr` (`\r` num terminal — detectado com `os.ModeCharDevice` de `os.Stderr.Stat()`, injetável para o teste —, uma linha a cada 10 quadros e no último caso contrário), imprimir a linha `Interrupted: ...` e o resumo quando o erro é `ErrRenderInterrupted`, e imprimir o resumo parcial também para outros erros depois de algum quadro concluído (`summary.Drawn + summary.Kept > 0`), sempre devolvendo o erro; a mesma verificação de contexto em `render_frame.go` (o quadro isolado também cancela) (depende de T077, T080)

**Checkpoint US5**: `quickstart.md` item 5 (interrupção, retomada, `diff -r`).

---

## Phase 8: User Story 6 - Proteger as imagens já existentes no destino (Priority: P6)

**Goal**: nunca sobrescrever sem pedir, nunca misturar quadros de outro
conjunto, remover só as sobras da própria ferramenta, e nunca deixar imagem
parcial.

**Independent Test**: `quickstart.md` item 6.

### Testes da User Story 6

- [X] T082 [P] [US6] Estender `internal/domain/frame_set_test.go` (com `FrameDirectoryBuilder`): sem `overwrite`, um quadro **nosso de outro conjunto** (qualquer número, dentro ou fora do plano) → `ErrFrameSetConflict` (mensagem: quantos são de outro conjunto e `use --overwrite to replace them, or another --output`), sem `FrameWork`; um arquivo com nome de quadro que **não é nosso** com número **dentro** do plano → `ErrFrameSetConflict` (a mensagem conta os arquivos que não são da ferramenta); um que não é nosso com número **fora** do plano é ignorado; um `frame_1.png` (nome fora do padrão) nunca aparece; com `overwrite`: `Draw = 0…N−1` (todos), `Keep` vazio, `Remove` = os quadros **nossos de outro conjunto** com número **fora** do plano (ordenados), e nunca um que não é nosso; um quadro nosso do **mesmo** conjunto com número fora do plano (não ocorre na prática, pois o mesmo id implica o mesmo total) é ignorado, nunca removido
- [X] T083 [P] [US6] Estender `internal/infra/outbound/pngfile/frame_repository_test.go`: `Remove(dir, [3, 9])` apaga `frame_000003.png` e `frame_000009.png` e **nada mais** (nem `notas.txt`, nem `frame_000004.png`); um número que não existe é ignorado sem erro; um diretório sem permissão → `ErrFrameDestinationInvalid`; `Save` sobre um arquivo existente o substitui por inteiro; `Save` que falha (diretório sem permissão de escrita, `t.Skip` como root) **preserva** o arquivo anterior intacto e não deixa `.sobrevoo-*.tmp`; com o `syncFile` de `atomicfile` trocado por um que falha, o quadro anterior permanece
- [X] T084 [P] [US6] Estender `internal/infra/outbound/pngfile/frame_exporter_test.go`: `Export` num caminho que existe, sem `overwrite` → `ErrFrameDestinationExists` (mensagem com o caminho e `use --overwrite to replace it`) e o arquivo existente **intacto**; com `overwrite`, o arquivo é substituído por inteiro; uma pasta inexistente ou sem permissão → `ErrFrameDestinationInvalid`; uma falha na gravação (`syncFile` falhando) preserva o arquivo anterior e não deixa temporário
- [X] T085 [P] [US6] Estender `internal/application/frame_service_test.go`: com `Overwrite: true`, `DrawFrames` chama `Inspect`, `Plan`, `Remove(dir, work.Remove)` **antes** do primeiro `Save`, depois desenha todos; `RenderSummary.Removed = len(work.Remove)`; um `Remove` que falha impede qualquer `Save` e sobe o erro; sem `Overwrite` e com conflito, `Plan` devolve `ErrFrameSetConflict` e **nem `Remove` nem `Save` nem `Render`** são chamados (o `TileDecoder` não é tocado); `DrawFrame` repassa `Overwrite` ao `FrameExporter` (e não consulta o `FrameRepository`)
- [X] T086 [P] [US6] Estender `render_frame_test.go` e `render_all_test.go`: `--overwrite` chega em `SingleFrameRequest.Overwrite` e `FrameSetRequest.Overwrite`; `ErrFrameDestinationExists` → código 36 com a dica; `ErrFrameSetConflict` → 37; `ErrFrameDestinationInvalid` → 35; uma linha `Removed: 20 frames from a previous set` no resumo de `render all` quando `Removed > 0`

### Implementação da User Story 6

- [X] T087 [US6] Editar `internal/domain/frame_set.go`: completar `FrameDirectory.Plan` com o conflito (`ErrFrameSetConflict` sem `overwrite`, contando quantos arquivos nossos de outro conjunto e quantos que não são nossos com número dentro do plano) e a sobrescrita (`Draw` = todos; `Remove` = nossos de outro conjunto com número fora do plano, em ordem crescente; nunca um que não é nosso) (depende de T082, T078)
- [X] T088 [US6] Editar `internal/infra/outbound/pngfile/frame_repository.go`: `Remove(dir, indexes)` — `os.Remove` de `dir/FrameFileName(i)` para cada `i`, ignorando `fs.ErrNotExist`, com as demais falhas como `ErrFrameDestinationInvalid`; só apaga nomes gerados por `FrameFileName` (depende de T083)
- [X] T089 [US6] Editar `internal/application/frame_service.go`: `DrawFrames` chama `Remove` (só se `len(work.Remove) > 0`) antes do primeiro `Save` e preenche `summary.Removed`; um erro de `Remove` é devolvido sem desenhar (depende de T085, T087, T088, T080)
- [X] T090 [US6] Editar `render_all.go`: a linha `Removed: <n> frames from a previous set` (só quando `> 0`) e a ajuda da flag `--overwrite` (`replace existing frames; also removes the frames of a previous set that this plan does not have`); em `render_frame.go`, a ajuda `replace the file if it already exists` (depende de T086, T089)

**Checkpoint US6**: `quickstart.md` item 6.

---

## Phase 9: User Story 7 - Recusar entradas que não servem, com mensagem clara (Priority: P7)

**Goal**: plano ou recorte inválido, de versão desconhecida, de outro plano ou
que não cobre, mapa vetorial, recorte sem elevação, peça ilegível — todos com
mensagem, código próprio e nenhuma imagem criada.

**Independent Test**: `quickstart.md` item 7.

### Testes da User Story 7

- [X] T091 [P] [US7] Estender `internal/domain/geo_slice_test.go`: `EnsureMatches(plan)` — `PlanID == plan.ID()` → nil; diferente → `ErrSliceDoesNotMatchPlan` com as duas identificações **abreviadas em 12 caracteres** na mensagem; `PlanID` vazio → `ErrSliceDoesNotMatchPlan`; `EnsureCovers(plan, tuning)` — a área do recorte igual à do plano ou maior → nil; menor em qualquer lado → `ErrSliceDoesNotCoverPlan` com a área exigida e a do recorte no formato `lat -23.6100 to -23.4800, lon -46.7200 to -46.5300` (o da etapa 4); com antimeridiano (recorte e plano cruzando, e recorte que não cruza para um plano que cruza); `EnsureDrawable()` — conjuntos `png`, `jpg`, `webp` → nil; `pbf` → `ErrTileFormatUnsupported` com a mensagem `base map "bbbike" has vector tiles (pbf); drawing vector tiles is not supported yet, use a base map of image tiles (PNG, JPG or WebP)`; `mvt` idem; `gif` e vazio → `ErrTileFormatUnsupported` (`format "gif" is not supported for drawing`); um recorte com `pbf` num conjunto e `png` em outro → recusa (cita o `pbf`); recorte sem conjuntos de peças → nil; recorte com `SampleCount == NoValueSampleCount` (inclusive 0 amostras) → `ErrNoElevationData`; com uma amostra com valor → nil; a ordem da verificação: formato de peça antes de elevação
- [X] T092 [P] [US7] Estender `internal/domain/bounding_box_test.go`: `ContainsBox` — igual, menor dentro, maior, tocando a borda (contido), deslocado só em latitude/longitude (não contido); com antimeridiano: `[170, −170]` contém `[175, −175]` e `[179, 179.5]` e `[−179.5, −178]`, não contém `[160, 175]`; uma caixa do mundo todo (`−180 a 180`) contém qualquer uma; uma que cruza não contém uma que a cobre por fora
- [X] T093 [P] [US7] Estender `internal/infra/outbound/zipfile/geo_slice_reader_test.go` (um `t.Run` por fixture, todos → `ErrSliceFileInvalid` com a causa na mensagem, salvo a versão): `NotZipContent()` ("not a ZIP file"); `TruncatedSliceFile()` (truncado/`zip: not a valid zip file`); `SliceFileWithoutManifest()`; manifesto que não é JSON; campos obrigatórios ausentes (`area`, `summary`, `sources`, `base_map`, `elevation`, `plan_id`; a mensagem cita o campo); `SliceFileWithoutPlanID()` (mensagem `the slice has no plan identification; generate it again with "geodata slice --export"`); `SliceFileWithBadPlanID()`; `SliceFileWithVersion(2)` → `ErrSliceFormatVersionUnsupported` com `found 2, accepted: 1` e `format_version` ausente ou não inteiro → `ErrSliceFileInvalid`; `SliceFileWithWrongCounts` para cada campo (`tile_count`, `missing_tile_count`, `sample_count`, `no_value_sample_count`, `elevation_m`, `size_bytes`; mensagem `summary.tile_count is 7 but the file has 3`); `SliceFileWithoutEntry` (peça e grade citadas mas ausentes); `SliceFileWithExtraEntry` (entrada que o manifesto não cita); `SliceFileWithWrongTileSize` (`bytes` diferente da entrada); `SliceFileWithWrongGridSize` (`rows × cols × 4` diferente do tamanho); `SliceFileWithCompressedEntry` (só *store* é aceito); `SliceFileWithChosenOutOfRange`; `SliceFileWithBadSourceIndex`; `SliceFileWithCorruptEntry` (CRC errado → erro de checksum como `ErrSliceFileInvalid`); nomes de entrada duplicados; caminho de entrada com `..` ou absoluto; um arquivo maior que 256 MiB + 16 MiB (com um leitor de teste que declara o tamanho) → `ErrSliceFileInvalid`
- [X] T094 [P] [US7] Estender `internal/application/frame_service_test.go`: `DrawFrame` e `DrawFrames` chamam, **nesta ordem e antes de criar a `Scene`**, `slice.EnsureMatches(plan)`, `EnsureCovers(plan, sliceTuning)`, `EnsureDrawable()`; cada uma que falha devolve o erro **sem** chamar `TileDecoder`, `FrameRepository` nem `FrameExporter` (mocks sem expectativas); um recorte `pbf` → `ErrTileFormatUnsupported`; uma peça ilegível no meio do voo (o `Decode` do mock falha na terceira peça pedida) → `ErrSliceFileInvalid` com registro, nível e posição, o resumo mostra os quadros já salvos em `Drawn` e **nenhum** `Save` do quadro que falhou
- [X] T095 [P] [US7] Estender `render_frame_test.go` e `render_all_test.go`: cada erro de domínio de `contracts/cli.md` chega ao código certo (17, 18, 27–33, 35–37) com a mensagem do serviço em `stderr`, sem `stdout`, e **`DrawFrame`/`DrawFrames` não é chamado** quando um `Load` falha; `ErrFrameOutOfRange` acontece antes de carregar o recorte; `ErrInvalidResolution` antes de carregar o plano

### Implementação da User Story 7

- [X] T096 [US7] Editar `internal/domain/bounding_box.go`: `(b BoundingBox) ContainsBox(other BoundingBox) bool` (latitudes contidas; longitudes pelos `longitudeSpans` de `b` e de `other`: cada intervalo de `other` precisa estar dentro de algum de `b`, com o caso do mundo todo) (depende de T092)
- [X] T097 [US7] Editar `internal/domain/geo_slice.go`: `EnsureMatches(plan CameraPlan) error`, `EnsureCovers(plan CameraPlan, tuning SliceTuning) error` (usa `plan.AreaOfInterest(tuning)` e `ContainsBox`; formata as áreas como o resumo da etapa 4, `Area: lat ... to ..., lon ... to ...`) e `EnsureDrawable() error` (formatos e mensagens de `data-model.md`; peças `png`/`jpg`/`webp`; `pbf`/`mvt` com a mensagem de peça vetorial; qualquer outro com `format %q is not supported for drawing`; depois `ErrNoElevationData`) (depende de T091, T096)
- [X] T098 [US7] Editar `internal/infra/outbound/zipfile/geo_slice_reader.go`: toda a validação de `research.md` item 15 (as causas dos testes de T093): estrutura do ZIP (só *store*, nomes únicos e seguros, sem entradas que o manifesto não cite, tamanho total ≤ 256 MiB + 16 MiB), manifesto (obrigatórios; `format_version` inteiro e igual a 1, senão `ErrSliceFormatVersionUnsupported` com `found N, accepted: 1`; `plan_id` de 64 hexadecimais minúsculos e com a mensagem de gerar de novo quando ausente), coerência (tamanhos das peças e das grades, `level.chosen ∈ [min, max]`, índices de fonte, contagens do `summary` **recalculadas** com `domain.NewGeoSlice` e comparadas, com a mensagem `summary.<campo> is X but the file has Y`), e a tradução de `zip.ErrChecksum` e de qualquer falha de formato em `ErrSliceFileInvalid` (`"%w: %s"`) (depende de T093, T051)
- [X] T099 [US7] Editar `internal/application/frame_service.go`: `DrawFrame`/`DrawFrames` chamam `EnsureMatches`, `EnsureCovers` e `EnsureDrawable` nessa ordem, antes de `NewScene` e de qualquer E/S de destino; rodar `make generate` (depende de T094, T097)
- [X] T100 [US7] Conferir em `render_frame.go` e `render_all.go` a **ordem das verificações** de `contracts/cli.md` (resolução → plano → número → recorte → serviço, que faz correspondência, cobertura, formato, elevação e destino) e ajustar o que faltar (depende de T095, T099)

**Checkpoint US7**: `quickstart.md` item 7 (todas as recusas; a de cobertura só
pelos testes automatizados).

---

## Phase 10: Polish & Cross-Cutting Concerns

- [X] T101 [P] Estender `test/samples/main.go` (e o comentário de cabeçalho) para gravar em `specs/005-frame-rendering/amostras/` os arquivos de `quickstart.md` ("Pré-requisitos"): `mapa-imagem-sp.mbtiles` (PNG, níveis 10 a 16, `PositionPNGTile`, com as 3 peças removidas de nível 16 da etapa 4), `mapa-vetorial-sp.mbtiles` (`Format: "pbf"`, dados quaisquer), `relevo-sem-dado.tif` (nenhuma célula com valor), `mapa-imagem-antimeridiano.mbtiles` e `mapa-imagem-polar.mbtiles` (PNG, mesma área dos de etapa 4), e a flag `--raster-over <gpx>` que lê o GPX (usar `trackparser` só nesta ferramenta de desenvolvimento) e grava `mapa-imagem-passeio.mbtiles`, PNG, níveis 10 a 17, **menor** que o `bounds` do mapa vetorial real (caixa do GPX + 20%), para vencer a regra "menor área"; imprimir o que cada arquivo tem
- [X] T102 [P] Editar `.gitignore`: acrescentar `specs/005-frame-rendering/amostras/` (com a linha anterior terminando em nova linha; conferir com `git status` que nenhum binário entra)
- [X] T103 [P] Atualizar `CLAUDE.md` e `README.md` para a quinta etapa: o parágrafo de projeto (cinco features), os comandos (`go run ./cmd/sobrevoo render frame ...` e `render all ...`, `go run ./test/samples --out specs/005-frame-rendering/amostras`), as entidades e portas de domínio novas (`Scene`, `Resolution`, `FrameSetID`, `FrameDirectory`, `RenderSummary`, `RenderTuning`; `GeoSliceReader`, `TileDecoder`, `FrameRepository`, `FrameExporter`), os erros, `FrameService` e `GeoSliceService.Load`, os adapters (`zipfile.NewGeoSliceReader`, `tiledecoder.NewRaster`, `pngfile`), o `atomicfile.Sync`, o `config.RenderTuning`, e a regra de determinismo do desenho (sem FMA, funções de Go puro); manter a prosa em português e os identificadores em inglês
- [X] T104 Rodar `quickstart.md` de ponta a ponta com o binário real (`make build`, `go run ./test/samples --out $A`, itens 1 a 9), conferir cada resultado esperado, **preencher os valores marcados "(anotar)"** (tempos, contagens de `Holes`), corrigir o que o binário mostrar diferente e registrar desvios em "Notas de implementação" (depende de todas as histórias, T101)
- [X] T105 Validar com os dados reais de `resources/` (item 10 do `quickstart.md`): registrar o DEM e o mapa do BBBike, gerar plano e recorte do passeio, conferir a **recusa 31** com o `pbf`, gerar o raster sintético com `--raster-over`, desenhar `render frame` e `render all --resolution 1280x720`, abrir alguns quadros e conferir visualmente relevo, traçado e marcador; conferir com o GDAL (`gdallocationinfo` no DEM) a elevação sob o marcador de um quadro contra `geodata elevation --lat --lon`; registrar as observações (tempo por quadro, aparência, buracos reais) em "Notas de implementação" (depende de T104)
- [X] T106 Medir o tempo real de `render all` do plano de `pedalada.gpx` (padrão, 1080p) e de um plano de ~1350 quadros, e comparar com as metas de `plan.md` (SC-001, SC-011, SC-012); registrar em "Notas de implementação" e, se passarem, abrir a tarefa da pirâmide de máximos (`research.md` item 5) (depende de T104)
- [X] T107 [P] Editar `specs/005-frame-rendering/spec.md`: `Status: Implementada`; e `plan.md` se algo do desenho mudou na implementação
- [X] T108 Rodar `make test`, `make lint`, `make generate` (sem diff) e `go vet ./...`; conferir com `git status` que só arquivos esperados mudaram e que `resources/` e `amostras/` não foram adicionados; conferir cobertura do domínio com `go test ./internal/domain/... -cover` (depende de todas as tarefas anteriores)

---

## Dependências e Ordem de Execução

### Dependências entre fases

- **Phase 1** → **Phase 2 Parte A** → **Parte B** → **Parte C** → **Parte D** → histórias.
- **Parte A** (T002–T004) e **Parte B** (T005–T010) são
  sequenciais e precisam estar verdes antes de qualquer outra coisa; a Parte B
  depende da A só por `make test` verde, não por código.
- **Parte C**: T011, T012, T013, T015*, T017*,
  T018, T020*, T025 e T027 são quase todos independentes
  (`[P]`); T022 depende de T011, T017, T015, T018 e
  da Parte B; T023 de T008; T024 depende de
  T013, T022 e T023; T026 depende de T017,
  T015 e T025; T027 de T017, T022 e T025.
- **Parte D**: T028 e T029 são independentes (`[P]`); T030
  fecha a fundação.
- **US1** depende de Foundational completo. **US2** depende de US1 (estende o
  serviço e o `main.go`). **US3** depende de US1 (estende `frame_imagery.go` e
  `frame_scene.go`). **US4** depende de US1 (podendo ser feita depois de US2
  para a flag em `render all`). **US5** depende de US2. **US6** depende de US5
  (`Plan` e o repositório). **US7** depende de US1 e, para os testes de
  serviço, de US2/US5.
- **US3**, **US4** e **US7** não dependem entre si e podem ser feitas em
  qualquer ordem depois de US2 (mas nunca em paralelo nos mesmos arquivos:
  `frame_service.go`, `frame_scene.go`, `render_all.go`, `render_frame.go`).
- **Polish** depende de todas as histórias desejadas.

### Dentro de cada história

- Testes antes da implementação correspondente e devem falhar primeiro.
- Domínio → adapters → serviços → CLI e composition root.
- Dentro da US1: T044 → T045 → T046 → T047 → T048 → T049 →
  T050 (medir); T051, T052, T053 e T054 em paralelo com o
  domínio (arquivos diferentes); depois os serviços, a CLI e o `main.go`.

### Oportunidades de paralelismo

- Fundação: T002, T005, T011, T012, T013,
  T014, T015, T016, T017, T018,
  T019, T020, T021, T025, T028 e
  T029 em paralelo (respeitando as dependências acima).
- US1, testes: T031, T032, T033, T034, T035,
  T036, T037, T038, T039, T040,
  T041, T042 e T043 são arquivos diferentes ⇒ todos `[P]`.
- US1, implementação: T051, T052, T053 e T054 (adapters
  distintos) em paralelo com T044/T045 (domínio); T045, T046
  editam o mesmo arquivo (sequenciais); T047 e T048 são arquivos
  distintos, mas dependem de T044.
- Demais histórias: os testes de cada uma em paralelo entre si; as tarefas
  que editam o mesmo arquivo (`frame_service.go`, `frame_scene.go`,
  `render_all.go`, `render_frame.go`, `frame_repository.go`, `frame_set.go`,
  `geo_slice.go`) são sequenciais.
- Polish: T101, T102, T103 e T107 em paralelo.

---

## Exemplo de Paralelização: User Story 1

```text
# Todos os testes de US1 em paralelo (arquivos diferentes):
T031 frame_camera_test.go        T032 frame_surface_test.go
T033 frame_surface_dda_test.go      T034 frame_imagery_test.go
T035 frame_overlay_test.go      T036 frame_scene_test.go
T037 geo_slice_reader_test.go T038 raster_tile_decoder_test.go
T039 png_mark_test.go           T040 frame_exporter_test.go
T041  T042  T043

# Depois, o domínio do desenho e os adapters em paralelo:
T044 frame_camera.go  →  T045 + T046 frame_surface.go  →  T047  T048  →  T049
T051 geo_slice_reader.go   T052 raster_tile_decoder.go   T053 png_mark.go → T054
```

---

## Estratégia de Implementação

### MVP Primeiro (Phase 1 + 2 + User Story 1)

1. Phase 1 (baseline) e Phase 2 (`fsync`; `plan_id` na etapa 4; base;
   fixtures).
2. Phase 3 (US1): ler o recorte, desenhar um quadro e gravá-lo → `render frame`.
3. **PARE e valide**: `quickstart.md` itens 1 e 2, e leia T050: o tempo por
   quadro decide se a pirâmide de máximos entra antes de seguir.

### Entrega Incremental

1. Setup + Foundational → etapa 4 com `plan_id`, portas, erros, configuração e
   fixtures prontos.
2. + US1 → quadro isolado (MVP) → itens 1 e 2.
3. + US2 → `render all` → item 4.
4. + US3 → falta de dado explícita e contada → item 3.
5. + US4 → resolução → item 8.
6. + US5 → progresso, interrupção e retomada → item 5.
7. + US6 → proteção do destino → item 6.
8. + US7 → recusas → item 7.
9. Polish → amostras, documentação, antimeridiano e polar (item 9), dados reais
   (item 10), desempenho e verificação final.

Cada história agrega valor sem quebrar as anteriores.

---

## Notas

- `[P]` = arquivos diferentes, sem dependência de tarefa incompleta.
- Os valores numéricos de `RenderTuning` são iniciais e ficam concentrados em
  `config.Load()`; ajustá-los é uma edição num só lugar (`research.md`
  item 22). Mudar um deles muda o `Fingerprint()` e portanto o conjunto de
  quadros; mudar cores, padrões ou o algoritmo exige subir `RenderVersion`.
- O hash de referência de T036 protege o determinismo entre arquiteturas
  (sem FMA): se ele falhar em outra máquina, **corrija a aritmética**, não a
  constante.
- Faça commit após cada tarefa ou grupo lógico (só quando o usuário pedir). A
  Parte A e a Parte B da Phase 2 merecem commits próprios, separados do
  código novo, por tocarem etapas já entregues.
- Pare em qualquer checkpoint para validar a história com o `quickstart.md`.
- Evite: tarefas vagas, duas tarefas `[P]` editando o mesmo arquivo,
  dependências que quebrem a independência de teste de uma história.

---

## Notas de implementação (desvios do plano original, registrados ao executar)

- **Desempenho (T050, T106)**, Apple M1, 8 núcleos: `BenchmarkScene_Render_1080p`
  0,63 s/quadro (inclinação 45°) e `..._LowTilt` 0,98 s (25°), abaixo da meta de 2 s;
  a pirâmide de máximos (`research.md` item 5) **não** foi necessária. Passeio real
  (1020 quadros, 1920×1080): 17 min 15 s, sem buracos; quadro 4K: ~4 s;
  quadro 1080×1920: ~0,9 s.
- **Determinismo**: o hash de referência de um quadro é igual em arm64 e amd64
  (Rosetta), graças aos produtos em soma envolvidos em `float64(...)` (sem FMA) e
  ao uso só de funções puras de Go.
- **Marcador**: a primeira versão testava profundidade por pixel e desenhava
  meio disco; a visibilidade passou a ser decidida no pixel do centro
  (`research.md` item 9), com o disco inteiro.
- **Roteiro**: `pedalada.gpx` exige ao menos 31,2 s, então o quickstart usa
  `--duration 32`; a interrupção é enviada ao binário direto (não à subshell).
- **Resolução padrão**: 1080×1920 (vídeo vertical), pedido do usuário durante a
  implementação; `RenderDefaults` em `config` e os documentos foram ajustados.
- **Enquadramento em vertical (etapa 3)**: com o FOV vertical fixo de 45°, o
  horizontal em 9:16 é ~26°, e a abertura e o fechamento do plano (que só
  consideravam o vertical) cortavam as laterais (o quadro 1019 do passeio real).
  Resolvido com `plan --aspect L:A` (padrão `9:16`, gravado em
  `parameters.aspect_ratio` do plano; plano sem o campo é lido como `16:9`;
  erro `39`): a abertura e o fechamento ficam ~1,8× mais afastados em 9:16 e o
  trajeto inteiro cabe (conferido num quadro de fechamento em 1080×1920). Efeito
  colateral: a área que o recorte precisa cresce, e o passeio real deixou de caber
  no único ladrilho Copernicus disponível (falta elevação a oeste de -40°); é
  preciso registrar o ladrilho vizinho. Baixado (`dem-S14-W041.tif`, Copernicus
  GLO-30, 41 MB, em `resources/`): com os dois ladrilhos o recorte do passeio em
  9:16 sai (256 peças, nível 15, 358 737 amostras, 1,6 MiB) e o quadro 1019 mostra
  o traçado inteiro.
- **Detalhe do recorte em vertical**: `SliceTuning.ReferenceHeightPixels` passou de
  1080 para 1920 (a altura do vídeo padrão); o nível ideal sobe 1 em relação ao
  antigo à mesma distância (o teste de `tile_test.go` passou a fixar 1080 onde o
  valor entra na conta, e um teste novo cobre a diferença).
- **Voo real em vertical**: `render all` do plano 9:16 do passeio (1020 quadros,
  1080×1920, dois ladrilhos Copernicus e o mapa sintético) levou 16 min 01 s (cerca
  de 0,9 s por quadro; a máquina ficou parada por períodos) e terminou sem
  buracos.
- **Aviso de proporção**: `render frame` e `render all` avisam em `stderr` (sem
  mudar o código de saída) quando a resolução é mais estreita que o vídeo do plano
  (`Resolution.NarrowerThan`, tolerância de 1%).
- **Amostras extras** em `test/samples`: mapas em imagem, vetorial, relevo sem
  dado, `relevo-buraco.tif` e `--raster-over <gpx>`.
