# Modelo de Dados: Montagem do Vídeo do Voo

**Feature**: `006-video-assembly` | **Data**: 2026-09-27

Tipos de `internal/domain` (e o serviço de `internal/application`) que esta
etapa acrescenta ou estende. Os nomes seguem o que a spec chama de: Plano de
Câmera Exportado (`CameraPlan`), Diretório de Quadros (`FrameDirectory`),
Quadro (`FrameFile`), Conferência dos Quadros (`FrameDirectory.Verify`), Nível de
Qualidade (`VideoQuality`), Codificador de Vídeo (`VideoEncoder`), Vídeo (o
arquivo de `VideoExporter`) e Resumo da Montagem (`VideoSummary`). Regra de
negócio fica no domínio; o serviço só chama portas na ordem certa (Princípio
IX). Nenhum tipo aqui importa `os`, `os/exec`, `image/png` ou `internal/infra`.

## Entidades existentes, com acréscimos

### `FrameMark` no lugar do `FrameSetID` solto (`frame_set.go`) — Clarificação de 2026-09-27

```go
// FrameMark is what a frame says about itself, inside its image: the set it
// belongs to and the plan it was drawn from.
type FrameMark struct {
    SetID  FrameSetID
    PlanID string // CameraPlan.ID()
}

// NewFrameMark is the mark of the frames of plan drawn from slice at resolution.
func NewFrameMark(plan CameraPlan, slice GeoSlice, resolution Resolution, tuning RenderTuning) FrameMark
```

- `FrameSetID` e `NewFrameSetID` **não mudam** (a identificação do conjunto já
  inclui `plan.ID()`), só ganham a versão nova: `RenderVersion` = **2**
  (`research.md` item 8). O teste que compara a versão atual com a próxima
  continua valendo; o hash dos pixels de um quadro não muda.
- As portas passam a receber a marca:

  ```go
  Save(dir string, index int, mark FrameMark, image FrameImage) error   // era: id FrameSetID
  Export(image FrameImage, mark FrameMark, path string, overwrite bool) error
  ```

### `FrameFile` e a porta `FrameRepository` (`frame_set.go`)

`FrameFile` ganha o que a etapa 6 lê, sem tirar nada:

| Campo | Tipo | Semântica |
|---|---|---|
| `Index` | int | o número no nome do arquivo (como hoje) |
| `Ours`, `SetID` | | como hoje: PNG com a marca do conjunto e o conjunto |
| `PlanID` | texto | **novo**: a identificação do plano gravada no quadro; vazia se o quadro é nosso mas não a traz (de antes da versão 2) |
| `Width`, `Height` | int | **novo**: as dimensões lidas do cabeçalho (0 se ilegível) |
| `Whole` | bool | **novo**: o PNG termina onde um PNG termina (assinatura, `IHDR`, `IEND` no fim), qualquer que seja a resolução |
| `Complete` | bool | como hoje: `Whole` e dimensões iguais às da resolução pedida a `Inspect` (a etapa 5 continua igual) |

A porta ganha **um método**, que a etapa 6 usa; `Inspect`, `Save` (com a marca) e
`Remove` seguem como estão:

```go
// List lists the frame files of dir — the files named FrameFileName(n) —, sorted by
// number, saying for each what its image says of itself: whether this tool drew
// it, its set and plan, its size and whether it is whole. Nothing is
// asked of the resolution. Unlike Inspect, a dir that does not exist, is not a
// directory or cannot be read is an error: ErrFrameDirectoryInvalid.
List(dir string) (FrameDirectory, error)
```

### `FrameDirectory.Verify` (`frame_verification.go`, arquivo novo, mesmo tipo)

```go
// Verify checks that the frames the directory lists can be joined into the video
// of plan and gives their resolution.
func (d FrameDirectory) Verify(plan CameraPlan) (Resolution, error)
```

Regra de domínio pura, sobre `Files`; **só considera quadros nossos** (`Ours`) e
para no primeiro tipo de problema, na ordem de `research.md` item 7:

| # | Verificação | Erro |
|---|---|---|
| 1 | há ao menos um quadro nosso | `ErrFrameDirectoryInvalid` (a mensagem diz quantos arquivos com nome de quadro foram ignorados) |
| 2 | todos trazem `PlanID` | `ErrFramesWithoutPlanID` (quantos não trazem; orientação: desenhar de novo com `render all --overwrite`) |
| 3 | todos têm `PlanID == plan.ID()` | `ErrFramesDoNotMatchPlan` (quantos são do plano e quantos não, com as duas identificações abreviadas) |
| 4 | todos com a mesma largura e altura, pares | `ErrFrameResolutionInvalid` (a esperada = a mais frequente, empate: a do menor número; os que destoam, com a resolução de cada um) |
| 5 | todos têm o mesmo `SetID` | `ErrFramesDoNotMatchPlan` ("N conjuntos misturados": outra fatia de dados, resolução ou versão do desenho) |
| 6 | há um quadro, e só um, para cada número de 0 a `len(plan.Frames)−1` | `ErrFrameSequenceInvalid` (faltam / sobram / repetidos, em faixas, com o total) |
| 7 | todo quadro é `Whole` | `ErrFrameFileInvalid` (até cinco arquivos) |

Devolve a `Resolution` comum. Não valida os limites de `NewResolution` (180 a
3840, 8 294 400 pixels): um diretório vindo de outra ferramenta com outra
resolução par é aceito, desde que o codificador a aceite. **Funções puras de
apoio** (sem estado): formatar uma lista de números em faixas (`12-15, 40`) e
um limite de itens listados. Detalhes de mensagem em `contracts/cli.md`.

### `RenderVersion` (`render_tuning.go`)

De `1` para `2`, com o comentário atualizado: sobe também quando o **formato do
arquivo** do quadro muda (aqui, a identificação do plano dentro dele).

## Entidades novas

### `VideoQuality` (`video_quality.go`)

```go
type VideoQuality int

const (
    VideoQualityLow VideoQuality = iota
    VideoQualityMedium
    VideoQualityHigh
)

func ParseVideoQuality(text string) (VideoQuality, error) // "low", "medium", "high"; erro de uso da CLI se outro
func (q VideoQuality) String() string
```

Só o nome e a ordem `low < medium < high`. O que cada nível significa para o
codificador é do adapter (`research.md` item 5). `ParseVideoQuality` devolve um
erro simples (não sentinela): quem o traduz em erro de uso (código `2`) é a CLI,
como faz com `--distance`.

### `VideoRequest`, `VideoProgress`, `EncodeJob`, `EncoderInfo` e `VideoSummary` (`video.go`)

```go
// VideoRequest asks for the video of a plan from the frames in a directory.
type VideoRequest struct {
    Directory string
    Output    string
    Quality   VideoQuality
    Overwrite bool
}

// VideoProgress says how far the encoding got: Done frames of Total are encoded,
// after Elapsed.
type VideoProgress struct {
    Done, Total int
    Elapsed     time.Duration
}

// EncodeJob is what the encoder is asked: join the Frames frames of Directory
// (named FrameFileName(0) to FrameFileName(Frames-1)) into a video of the given
// frame rate and quality, and write it to Output.
type EncodeJob struct {
    Directory string
    Frames    int
    FrameRate float64
    Quality   VideoQuality
    Output    string
}

// EncoderInfo says what encoder was found, for the summary.
type EncoderInfo struct {
    Name, Version, Codec string
}

func (i EncoderInfo) String() string // "ffmpeg 7.1 (libx264)"

// VideoSummary says what an assembly did (FR-017).
type VideoSummary struct {
    Frames     int
    FrameRate  float64
    Resolution Resolution
    Quality    VideoQuality
    Encoder    EncoderInfo
    SizeBytes  int64
    Elapsed    time.Duration // about the run; never goes into the video
    Encoded    int           // frames encoded when the run ended, for an interrupted one
    Interrupted bool
}

// Duration is the duration of the video: its frames over its frame rate.
func (s VideoSummary) Duration() time.Duration
```

`Duration` = `Frames / FrameRate` segundos, arredondada ao milissegundo, calculada
com aritmética de `float64` numa só divisão. É a "duração do vídeo" da spec
(FR-011).

### Portas `VideoEncoder` e `VideoExporter` (`video.go`, no topo, com `//go:generate` logo após `package`)

```go
// VideoEncoder joins the frames of a flight into a video with an external
// encoder. Concrete implementations live in internal/infra/outbound.
type VideoEncoder interface {
    // Probe says whether the encoder is there and can make the video: the
    // failure is ErrEncoderUnavailable, saying what is missing and what to do.
    Probe(ctx context.Context) (EncoderInfo, error)

    // Encode writes the video of job to job.Output, calling progress with how
    // many frames are encoded as it goes. It stops early, with the context's
    // error, when ctx is done; another failure is ErrVideoEncodingFailed. The
    // file it wrote is not the encoder's to keep: whoever asked deletes it.
    Encode(ctx context.Context, job EncodeJob, progress func(encoded int)) error
}

// VideoExporter keeps a video in a file the user chose. Concrete implementations
// live in internal/infra/outbound.
type VideoExporter interface {
    // Check says, before anything is encoded, whether path can receive the
    // video: ErrVideoDestinationExists unless overwrite is true and path exists;
    // ErrVideoDestinationInvalid when it is a directory or its folder is not there
    // or not writable.
    Check(path string, overwrite bool) error

    // Export gives produce the path of a temporary file next to path — a real
    // path, that can be written to and moved around in — and publishes what
    // produce wrote as path, whole or not at all: on failure, or when produce
    // fails, nothing is left behind and a previous file stays as it was. It
    // returns the size of the file. Failures to publish are the errors of Check;
    // an error of produce is returned as it is.
    Export(path string, overwrite bool, produce func(temporary string) error) (size int64, err error)
}
```

Ambas nomeadas pelo papel (`Encoder`, `Exporter`, como `CameraPlanExporter`,
`FrameExporter`), no arquivo da entidade que produzem (`video.go`, como `frame_set.go`
declara as portas dos quadros).

## Erros sentinela novos (`errors.go`)

| Erro | Quando | Código |
|---|---|---|
| `ErrFrameDirectoryInvalid` | diretório de quadros inexistente, que não é diretório, ilegível, ou sem nenhum quadro reconhecido | `40` |
| `ErrFrameSequenceInvalid` | faltam quadros, sobram números que o plano não tem, ou um número se repete | `41` |
| `ErrFrameResolutionInvalid` | quadros de resoluções diferentes, ou de largura/altura ímpar | `42` |
| `ErrFramesDoNotMatchPlan` | quadros de outro plano, ou de mais de um conjunto misturados (mesma resolução) | `43` |
| `ErrFramesWithoutPlanID` | quadros nossos sem a identificação do plano (de antes da versão 2) | `44` |
| `ErrFrameFileInvalid` | quadro que não é um PNG inteiro (truncado, ilegível) | `45` |
| `ErrEncoderUnavailable` | codificador ausente ou sem o formato necessário | `46` |
| `ErrVideoDestinationExists` | o destino já existe e não se pediu sobrescrita | `47` |
| `ErrVideoDestinationInvalid` | o destino é um diretório, ou sua pasta não existe ou não pode ser gravada | `48` |
| `ErrVideoInterrupted` | a montagem foi interrompida pelo usuário | `49` |
| `ErrVideoEncodingFailed` | o codificador falhou (com a causa) | `50` |

Distintos entre si e dos das etapas 1 a 5 (FR-021). Cada um leva mensagem que diz
o problema **exato** (`contracts/cli.md`); a mensagem de `46` é montada pelo
adapter, que sabe o nome do programa e como instalá-lo. A CLI os traduz em
códigos (`exit_code.go`); o nível de qualidade inexistente e a extensão de
`--output` são erros de **uso** (`2`), sem sentinela nova.

## Serviços (`internal/application`)

### `VideoService` (`video_service.go`, novo)

Um serviço por recurso (o vídeo), um método por operação (Princípio IX):

```go
type VideoService interface {
    // Assemble joins the frames in request.Directory into the video of plan,
    // written to request.Output, calling progress as frames are encoded. The
    // summary says what was done even when the run stopped early, with
    // ErrVideoInterrupted or another error.
    Assemble(ctx context.Context, plan domain.CameraPlan, request domain.VideoRequest, progress func(domain.VideoProgress)) (domain.VideoSummary, error)
}
```

`videoService` depende de `domain.FrameRepository` (só `List`), `domain.VideoEncoder`
e `domain.VideoExporter`. **Só orquestra**, nesta ordem (da mais barata à mais cara;
qualquer erro devolve o resumo com o que se sabe e nada foi criado):

1. `repository.List(request.Directory)` → `FrameDirectory` (erro `40`);
2. `directory.Verify(plan)` → `Resolution` (erros `40`–`45`);
3. `exporter.Check(request.Output, request.Overwrite)` (erros `47`, `48`);
4. `encoder.Probe(ctx)` → `EncoderInfo` (erro `46`);
5. `exporter.Export(request.Output, request.Overwrite, produce)`, com `produce` que
   chama `encoder.Encode(ctx, EncodeJob{…, Output: temporary}, onProgress)`, e
   `onProgress` que guarda `Encoded` e chama `progress` com `VideoProgress`;
6. se o contexto foi cancelado e o erro é do cancelamento → `ErrVideoInterrupted` e
   `summary.Interrupted`; senão o erro como veio; no sucesso, preenche `SizeBytes`,
   `Elapsed` e faz o último aviso `Done = Total`.

Nenhuma regra de negócio no serviço: a conferência, a duração, a decisão de qual
resolução é a esperada e as mensagens são do domínio; a decisão de "interrompido"
é a mesma função `interruption` que o `FrameService` usa (compartilhada dentro do
pacote `application`).

`FrameService` só muda por causa da marca (`domain.NewFrameMark`); nenhuma regra
nova. `CameraPlanService.Load` (etapa 3) carrega o plano, como em `render`.

## Adapters (`internal/infra/outbound`) — o que cada porta nova ou alterada tem

| Porta | Adapter | Notas |
|---|---|---|
| `FrameRepository.List` (nova), `Save` (marca) | `pngfile.NewFrameRepository()` | `List` e `Inspect` compartilham a leitura de cada arquivo (cabeçalho, marca, fim); `png_mark.go` grava e lê os dois blocos `tEXt` |
| `FrameExporter.Export` (marca) | `pngfile.NewFrameExporter()` | recebe a marca |
| `VideoEncoder` | `videoencoder.NewFFmpeg(binary)` | tipo `FFmpeg`; processo externo; item 6, 9 e 11 da pesquisa |
| `VideoExporter` | `videofile.NewVideoExporter()` | usa `atomicfile.PublishPath` (novo) |

## Ciclo de vida de uma montagem

```text
plano lido ──► diretório listado ──► quadros conferidos ──► destino conferido ──► codificador verificado
   (17,18)         (40)                 (40–45)                  (47,48)                 (46)
                                                                                             │
                                          temporário criado ◄────────────────────────────────┘
                                                │
                                          codificando ──► interrompido ──► temporário apagado (49)
                                                │    └────► falhou ───────► temporário apagado (50)
                                                ▼
                                          arquivo publicado (fsync + rename/link) ──► resumo (0)
```

Cada seta para a esquerda de "temporário criado" é uma recusa **sem nenhum
arquivo criado**; depois dele, qualquer saída que não seja o arquivo publicado
apaga o temporário e deixa o destino como estava.
