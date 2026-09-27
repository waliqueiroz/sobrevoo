---

description: "Task list for feature implementation"
---

# Tarefas: Montagem do Vídeo do Voo

**Entrada**: Documentos de design de `/specs/006-video-assembly/`

**Pré-requisitos**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/cli.md`, `contracts/video-file.md`, `contracts/frame-files-change.md`, `quickstart.md`

**Testes**: incluídos. Como nas etapas anteriores, a constituição do projeto
(Princípio VI — Testes Automatizados no Núcleo; Princípio X —
Given/When/Then, builders e isolamento por camada) exige testify e uber-go/mock
e proíbe testes tabulares: cada cenário é um `t.Run("should ...")` com
`// given`, `// when`, `// then`. As tarefas de teste abaixo materializam
essa exigência; ficam antes da implementação correspondente em cada fase e
devem falhar primeiro. **Nenhum teste automatizado exige o `ffmpeg` real**: o
adapter é testado contra um `ffmpeg` de mentira (script de shell, T033); o
`ffmpeg` real só entra nas tarefas de validação manual (`quickstart.md`).

**Organização**: as tarefas são agrupadas por história de usuário (P1–P7 de
`spec.md`) para permitir implementação e teste independentes de cada
história. A **Phase 2** tem três partes: (A) o `atomicfile.PublishPath`
(`research.md` item 10), pequeno e isolado; (B) a **mudança na etapa 5** (a
marca do plano dentro de cada quadro e `RenderVersion` 2, `research.md`
item 8), que precisa estar verde antes de qualquer código novo, para isolar uma
regressão de uma etapa já entregue; e (C) a base compartilhada (erros, códigos
de saída, tipos de domínio e portas, configuração, mocks, builders, fixture).

**Divisão do trabalho entre histórias** (o mesmo arquivo é estendido por mais
de uma história, nunca em paralelo):

| História | O que entrega |
|---|---|
| US1 | **MVP**: o vídeo de ponta a ponta, no nível padrão — `FrameRepository.List` e o adapter `pngfile`, `videofile.VideoExporter` (`Export` e `Check`), `videoencoder.FFmpeg` (`Probe`, `Encode` com a linha de comando base, falha do codificador), `VideoService.Assemble` (lista → resolução do primeiro quadro → `Probe` → `Export`), o comando `video` com o resumo, e a ligação em `main.go` |
| US2 | a conferência dos quadros: `FrameDirectory.Verify` completa (na ordem de `research.md` item 7), inserida no serviço no lugar da resolução do primeiro quadro |
| US3 | codificador ausente: as mensagens acionáveis de `Probe` (programa ausente, sem `libx264`, não executa) e a ordem entrada → destino → codificador |
| US4 | qualidade: a tabela `preset`/`crf` dos três níveis no adapter e a flag `--quality` (padrão da configuração, nível inexistente = erro de uso) |
| US5 | progresso e resumo: `-progress` do `ffmpeg`, `VideoProgress` no serviço, o impressor de progresso da CLI (terminal × log) |
| US6 | destino e interrupção: a checagem prévia do destino no serviço (depois dos quadros, antes do codificador), `--overwrite` na CLI, a regra `.mp4`, o cancelamento por contexto (mata o `ffmpeg`, apaga o temporário), `ErrVideoInterrupted`, sinais, código 49 |
| US7 | determinismo e ausência de dado do ambiente: as opções `bitexact`, `-map_metadata -1`, `x264` com threads fixas e sem SEI, testadas contra a linha de comando fixada, e a conferência real com `cmp`/`ffprobe` |

Enquanto a US2 não existe, o serviço lê a resolução do primeiro quadro
listado e assume o diretório correto (a US1 só recusa a listagem vazia); enquanto
a US3 não existe, `Probe` recusa sem as dicas de instalação; enquanto a US4 não
existe, o adapter usa só o `preset medium` e o `crf 23`; enquanto a US5 não
existe, a CLI não mostra progresso; enquanto a US6 não existe, não há checagem
prévia do destino nem tratamento de interrupção; enquanto a US7 não existe, a
linha de comando não traz as opções de determinismo.

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
tipo (`d` `FrameDirectory`, `s` `VideoSummary` e os `*Service`, `q`
`VideoQuality`, `i` `EncoderInfo`, `e` `FFmpeg`/`VideoExporter`/exporters, `r`
`Resolution` e repositories, `b` os builders); `new(x)` do Go 1.26 em vez de um
helper `ptr`; nome exportado nunca repete o pacote; receiver sem uso fica sem
nome.

**Regras dos testes do adapter `videoencoder`**: o `ffmpeg` de mentira é um
script POSIX (`#!/bin/sh`) gravado em `t.TempDir()` com permissão `0o755` e
passado a `videoencoder.NewFFmpeg(caminho)`; o comportamento vem de variáveis de
ambiente definidas com `t.Setenv` (por isso esses testes não usam
`t.Parallel`); os testes são pulados com `t.Skip` quando `runtime.GOOS ==
"windows"` (não há `sh`).

---

## Phase 1: Setup (Shared Infrastructure)

**Propósito**: garantir um ponto de partida verde e o ambiente da validação manual.

- [X] T001 Conferir que o branch atual é `006-video-assembly` (criado por `/speckit-specify`); rodar `make test`, `make lint` e `make generate` e confirmar que os três passam e que `make generate` não produz diff (`git status` limpo fora de `specs/006-video-assembly/`); registrar qualquer falha pré-existente antes de continuar
- [X] T002 Conferir se o `ffmpeg` e o `ffprobe` estão instalados (`ffmpeg -version | head -1`, `ffprobe -version | head -1`, `ffmpeg -hide_banner -encoders | grep libx264`); se não estiverem, **pedir ao usuário** que rode `brew install ffmpeg` (não instalar sozinho) e seguir com tudo o que é automatizado; as tarefas que dizem "com o `ffmpeg` real" (T043, T059, T068, T076, T079, T080, T083) ficam pendentes até ele existir, e as regras de linha de comando fixadas em T038, T057 e T078 (`research.md` itens 5 e 6, marcados **(confirmar)**) só se fecham nelas

**Checkpoint**: baseline verde; estado do `ffmpeg` conhecido.

---

## Phase 2: Foundational (Blocking Prerequisites)

**⚠️ CRÍTICO**: nenhuma tarefa de história de usuário pode começar até que
esta fase esteja completa.

### Parte A — `atomicfile.PublishPath` (research.md item 10)

- [X] T003 [P] Estender `internal/infra/outbound/atomicfile/atomic_file_test.go` com os cenários de `PublishPath(path string, overwrite bool, produce func(temporary string) error) error`: `produce` recebe o caminho de um arquivo que **já existe** (vazio), na **mesma pasta** do destino, com nome que casa `.sobrevoo-*.tmp`, que se pode abrir, escrever e reposicionar (o teste escreve, dá `Seek` e relê); sem sobrescrita e destino inexistente: o destino aparece com o que `produce` escreveu, com permissão `0o644`, e nenhum `.sobrevoo-*.tmp` fica; destino existente sem sobrescrita: `errors.Is(err, atomicfile.ErrExists)`, destino intacto, nenhum temporário; com sobrescrita: o destino é substituído por inteiro; `produce` devolve um erro: o erro volta **como veio** (`assert.Same`/`ErrorIs`), o destino não é criado, o destino anterior fica intacto e nenhum temporário fica; `Sync` (a variável `syncFile` já existente) é chamada uma vez, **depois** de `produce` e **antes** de publicar; `Sync` que falha → `errors.Is(err, atomicfile.ErrInvalid)`, sem destino nem temporário; pasta do destino inexistente → `ErrInvalid`; os cenários existentes de `Publish` continuam passando
- [X] T004 Editar `internal/infra/outbound/atomicfile/atomic_file.go`: `func PublishPath(path string, overwrite bool, produce func(temporary string) error) error` — cria o temporário com `os.CreateTemp(filepath.Dir(path), ".sobrevoo-*.tmp")`, **fecha-o** e guarda o nome (`defer os.Remove`), chama `produce(nome)`, reabre o arquivo com `os.OpenFile(nome, os.O_RDWR, 0)` só para `syncFile(f)` e `Close`, faz `os.Chmod(nome, 0o644)` e publica com o **mesmo miolo** de `Publish` (`rename` com sobrescrita, `publishExclusive` sem ela): extrair esse miolo para uma função interna `publish(temporary, path string, overwrite bool) error` que `Publish` e `PublishPath` chamam; o erro de `produce` volta como veio; falhas do sistema viram `invalid(...)`; o comportamento de `Publish` não muda (depende de T003)
- [X] T005 Rodar `make test`, `make lint` e `make generate` (sem diff) e conferir que `plan --export`, `geodata slice --export` e `render frame` mantêm o comportamento e os códigos de saída (depende de T004)

### Parte B — Mudança na etapa 5: a marca do plano dentro de cada quadro (research.md item 8; contracts/frame-files-change.md)

**Propósito**: cada quadro passa a carregar a identificação do plano de que veio;
`RenderVersion` sobe para 2. Os pixels não mudam.

- [X] T006 [P] Estender `internal/domain/frame_set_test.go` com os cenários de `NewFrameMark(plan, slice, resolution, tuning) FrameMark`: `SetID` é igual a `NewFrameSetID` dos mesmos argumentos; `PlanID` é igual a `plan.ID()` (64 hexadecimais minúsculos); dois planos de conteúdo igual têm o mesmo `PlanID`; mudar o plano muda o `PlanID` **e** o `SetID`; mudar só a resolução muda o `SetID` e **não** o `PlanID`; e um cenário que fixa `assert.Equal(t, 2, domain.RenderVersion)` (o formato de arquivo do quadro com a identificação do plano); os cenários existentes da versão (`NewFrameSetIDForVersion` com `RenderVersion` e `RenderVersion+1`) continuam passando
- [X] T007 [P] Estender `internal/infra/outbound/pngfile/png_mark_test.go` (o arquivo já existe; ver o estilo dos testes atuais) com os cenários dos dois blocos: o PNG codificado tem, **nesta ordem**, `IHDR`, `tEXt` `Sobrevoo\x00frame-set=<id>`, `tEXt` `Sobrevoo\x00plan=<id>`, depois `IDAT`… `IEND`, cada bloco com CRC válido; o mesmo quadro e a mesma marca dão os mesmos bytes; o PNG ainda é decodificado por `image/png`; ler a marca devolve `SetID` e `PlanID`; um PNG só com o bloco do conjunto (de antes da versão 2) é **nosso** com `PlanID` vazio; um bloco `plan=` com CRC errado é ignorado (`PlanID` vazio, o conjunto continua lido); um bloco `plan=` depois do primeiro `IDAT` é ignorado; um PNG sem o bloco do conjunto não é nosso (mesmo com o bloco do plano)
- [X] T008 [P] Estender `internal/infra/outbound/pngfile/frame_repository_test.go` e `internal/infra/outbound/pngfile/frame_exporter_test.go` (pacote `pngfile_test`): `Save` e `Export` recebem `domain.FrameMark{SetID, PlanID}` e o arquivo gravado traz os dois blocos; `Inspect` devolve, para cada quadro, `PlanID`, `Width`, `Height` e `Whole` (verdadeiro para um PNG inteiro de qualquer resolução, falso para um truncado), e `Complete` continua sendo `Whole` **e** dimensões iguais às da resolução pedida; um quadro de antes da versão 2 (montado tirando o bloco `plan=`) tem `Ours` verdadeiro e `PlanID` vazio; os cenários existentes passam a usar a marca
- [X] T009 Editar `internal/domain/frame_set.go`: `type FrameMark struct{ SetID FrameSetID; PlanID string }` e `func NewFrameMark(plan CameraPlan, slice GeoSlice, resolution Resolution, tuning RenderTuning) FrameMark` (`SetID: NewFrameSetID(...)`, `PlanID: plan.ID()`); as portas passam a `Save(dir string, index int, mark FrameMark, image FrameImage) error` e `Export(image FrameImage, mark FrameMark, path string, overwrite bool) error` (docs atualizadas: "marked as belonging to the set and the plan of mark"); `FrameFile` ganha `PlanID string` ("the plan identification written in the frame; empty when the frame is this tool's but does not carry it"), `Width, Height int` ("read from the header; 0 when unreadable") e `Whole bool` ("the PNG ends where a PNG ends, whatever the resolution"), sem tirar nem mudar `Complete`; editar `internal/domain/render_tuning.go`: `RenderVersion = 2` com o comentário "goes up whenever how a frame is drawn or written changes visibly: version 2 writes the plan identification inside the image" (depende de T006)
- [X] T010 Editar `internal/infra/outbound/pngfile/png_mark.go`: `encodeFrame(frame, mark domain.FrameMark)` grava, logo depois do `IHDR`, o bloco do conjunto (como hoje) e **em seguida** o bloco `tEXt` `Sobrevoo\x00plan=<PlanID>` (constante `planPrefix = markKeyword + "\x00plan="`); `readMark` devolve o `domain.FrameMark` lido (`SetID` do bloco do conjunto; `PlanID` do bloco do plano, vazio se não há) e `ok` só se o bloco do conjunto existe e confere; a leitura continua parando no primeiro `IDAT`/`IEND` e conferindo o CRC; `headBytes` (4096) continua bastando (depende de T007, T009)
- [X] T011 Editar `internal/infra/outbound/pngfile/frame_repository.go`: `Save` recebe a marca e a passa a `encodeFrame`; `inspectFrame` preenche `PlanID`, `Width`, `Height` e `Whole` (a leitura de cabeçalho e do final já existe: `readInfo`) e calcula `Complete = Whole && Width == resolution.Width && Height == resolution.Height`; nada mais muda em `Inspect` (depende de T008, T010)
- [X] T012 Editar `internal/infra/outbound/pngfile/frame_exporter.go`: `Export(image, mark, path, overwrite)` passa a marca a `encodeFrame` (depende de T008, T010)
- [X] T013 Editar `internal/application/frame_service.go` (monta `mark := domain.NewFrameMark(plan, slice, request.Resolution, s.renderTuning)` em `DrawFrame` e em `DrawFrames`; `directory.Plan` continua recebendo `mark.SetID`; `Save`/`Export` recebem `mark`) e `internal/application/frame_service_test.go` (as expectativas dos mocks passam a esperar a marca; um cenário novo: o `PlanID` da marca é o `ID()` do plano); rodar `make generate` para regerar `internal/domain/mockdomain/frame_repository.go` e `frame_exporter.go` (depende de T009)
- [X] T014 [P] Editar `specs/005-frame-rendering/contracts/frame-files.md` (a lista do bloco `tEXt` ganha o segundo bloco `plan=<id>` e a nota da versão do desenho 2, apontando para `specs/006-video-assembly/contracts/frame-files-change.md`) e `specs/005-frame-rendering/contracts/cli.md` (nota na tabela do destino: quadros anteriores à versão 2 são de outro conjunto e `render all` os recusa com `37` sem `--overwrite`)
- [X] T015 Rodar `make test`, `make lint` e `make generate` (sem diff) e, com o binário e os dados de `specs/005-frame-rendering/amostras` (quickstart da etapa 5, "Pré-requisitos"): `render frame … --number 30 --output /tmp/a.png` duas vezes com `cmp`; `render all` de um plano curto e conferir, com um script de uma linha em Python que percorre os blocos do PNG, que `frame_000000.png` tem exatamente os blocos `IHDR`, `tEXt`, `tEXt`, `IDAT`…, `IEND`; conferir que `render all` sobre um diretório de quadros da versão 1 (se ainda existir algum) sai com `37` (depende de T011, T012, T013, T014)

**Checkpoint B**: a etapa 5 verde, com a marca do plano; nenhum código novo da
etapa 6 além dela e do `PublishPath`.

### Parte C — Base compartilhada

- [X] T016 [P] Estender `internal/domain/errors.go` com os 11 sentinelas novos (cada um com comentário em inglês; Princípio VII), com estes nomes e significados (`data-model.md`, "Erros sentinela novos"): `ErrFrameDirectoryInvalid` (a frame directory that does not exist, is not a directory, cannot be read or holds no frame this tool drew), `ErrFrameSequenceInvalid` (frames missing, frames the plan has no number for, or a number that repeats), `ErrFrameResolutionInvalid` (frames of different resolutions, or of an odd width or height), `ErrFramesDoNotMatchPlan` (frames drawn from another plan, or of more than one set mixed), `ErrFramesWithoutPlanID` (frames of this tool that carry no plan identification), `ErrFrameFileInvalid` (a frame that is not a whole PNG image), `ErrEncoderUnavailable` (the video encoder is not there or cannot make the video), `ErrVideoDestinationExists` (a video destination that already exists), `ErrVideoDestinationInvalid` (a destination that is a directory or whose folder is missing or not writable), `ErrVideoInterrupted` (an assembly the user interrupted), `ErrVideoEncodingFailed` (the encoder failed)
- [X] T017 [P] Estender `internal/infra/inbound/cli/exit_code_test.go`: um `t.Run` por sentinela nova, cada um esperando o código (`ErrFrameDirectoryInvalid` → 40, `ErrFrameSequenceInvalid` → 41, `ErrFrameResolutionInvalid` → 42, `ErrFramesDoNotMatchPlan` → 43, `ErrFramesWithoutPlanID` → 44, `ErrFrameFileInvalid` → 45, `ErrEncoderUnavailable` → 46, `ErrVideoDestinationExists` → 47, `ErrVideoDestinationInvalid` → 48, `ErrVideoInterrupted` → 49, `ErrVideoEncodingFailed` → 50), com o erro **embrulhado** por `fmt.Errorf("%w: ...")`; e um cenário de que `ErrFramesDoNotMatchPlan` não é confundido com `ErrSliceDoesNotMatchPlan` (29) nem `ErrFrameDirectoryInvalid` com `ErrFrameDestinationInvalid` (35)
- [X] T018 Editar `internal/infra/inbound/cli/exit_code.go`: os 11 casos novos, depois do `39`, na ordem de T017 (depende de T016, T017)
- [X] T019 [P] Criar `internal/domain/video_quality_test.go`: `ParseVideoQuality` aceita `"low"`, `"medium"` e `"high"` (a mesma regra de maiúsculas e minúsculas que `plan.go` aplica a `--distance`: conferir lá antes de escrever) e devolve `VideoQualityLow`, `VideoQualityMedium`, `VideoQualityHigh`; outro texto (`"ultra"`, vazio) → erro (não sentinela) cuja mensagem é `"ultra": use one of low, medium, high`; `String()` devolve o nome (round-trip com `Parse`); a ordem `VideoQualityLow < VideoQualityMedium < VideoQualityHigh`
- [X] T020 Criar `internal/domain/video_quality.go`: `type VideoQuality int` com `VideoQualityLow`, `VideoQualityMedium`, `VideoQualityHigh` (nessa ordem, `iota`), `ParseVideoQuality(text string) (VideoQuality, error)` e `func (q VideoQuality) String() string`; só o nome e a ordem: o que cada nível pede ao codificador é do adapter (depende de T019)
- [X] T021 [P] Criar `internal/domain/video_test.go`: `VideoSummary.Duration()` = `Frames ÷ FrameRate`, arredondada ao milissegundo, numa só divisão de `float64`: 380 quadros a 10 → 38 s; 1350 a 30 → 45 s; 1350 a 29.97 → 45,045 s; 1 quadro a 30 → 33 ms; 432000 quadros a 1 → 432000 s; taxa zero ou negativa → duração 0 (sem `NaN`, `Inf` nem pânico); `EncoderInfo.String()` → `ffmpeg 7.1 (libx264)` e, sem versão, `ffmpeg (libx264)`
- [X] T022 Criar `internal/domain/video.go` (pacote `domain`, `//go:generate` logo após `package`, uma linha por porta como em `frame_set.go`: `go run go.uber.org/mock/mockgen -destination mockdomain/video_encoder.go -package mockdomain . VideoEncoder` e idem `video_exporter.go`/`VideoExporter`; imports; **as duas interfaces no início**): `VideoEncoder` com `Probe(ctx context.Context) (EncoderInfo, error)` e `Encode(ctx context.Context, job EncodeJob, progress func(encoded int)) error`, e `VideoExporter` com `Check(path string, overwrite bool) error` e `Export(path string, overwrite bool, produce func(temporary string) error) (size int64, err error)` — docs em inglês como em `data-model.md`; depois `VideoRequest{Directory, Output string; Quality VideoQuality; Overwrite bool}`, `VideoProgress{Done, Total int; Elapsed time.Duration}`, `EncodeJob{Directory string; Frames int; FrameRate float64; Quality VideoQuality; Output string}`, `EncoderInfo{Name, Version, Codec string}` (+`String()`), `VideoSummary{Frames int; FrameRate float64; Resolution Resolution; Quality VideoQuality; Encoder EncoderInfo; SizeBytes int64; Elapsed time.Duration; Encoded int; Interrupted bool}` (+`Duration() time.Duration`, `math.Round` ao milissegundo) (depende de T016, T020, T021)
- [X] T023 [P] Estender `internal/infra/outbound/config/config_test.go`: `Load()` devolve `VideoDefaults.Quality == config.LevelMedium` e `FFmpegBinary == "ffmpeg"`; os cenários existentes passam
- [X] T024 Editar `internal/infra/outbound/config/config.go`: `type VideoDefaults struct{ Quality Level }` ("the quality of the video when the user does not choose one: medium, for publishing"), campos `Config.VideoDefaults VideoDefaults` e `Config.FFmpegBinary string` ("the name of the video encoder program, looked for on the PATH; there is no external source yet, but this is the extension point, Principle VIII"), preenchidos em `Load()`; o pacote continua sem importar o domínio (depende de T023)
- [X] T025 [P] Estender `cmd/sobrevoo/config_mapping_test.go`: `domainVideoQuality(config.LevelLow|Medium|High)` → `domain.VideoQualityLow|Medium|High`, um `t.Run` por nível
- [X] T026 Editar `cmd/sobrevoo/config_mapping.go`: `func domainVideoQuality(level config.Level) domain.VideoQuality` (depende de T020, T024, T025)
- [X] T027 [P] Estender `internal/domain/builddomain/frame_directory_builder.go` (o armazenamento passa de `map[int]FrameFile` a uma fatia, para poder repetir um número; `Build` ordena **de forma estável** por `Index`; a API atual continua igual e compila para os usuários existentes — `internal/domain/frame_set_test.go` e `internal/application/frame_service_test.go`): campos do builder `planID string` e `resolution domain.Resolution` (padrão 1080 × 1920); `WithPlanID(id string)` e `WithResolution(r domain.Resolution)` valem para os quadros adicionados por `WithOursFrame`/`WithOursFrames` (que passam a preencher também `PlanID`, `Width`, `Height` e `Whole: true`); novos: `WithFrameOfPlan(index int, id domain.FrameSetID, planID string)`, `WithFrameOfResolution(index int, id domain.FrameSetID, r domain.Resolution)`, `WithTruncatedFrame(index int, id domain.FrameSetID)` (nosso, `Whole` falso, dimensões do cabeçalho), `WithFrameWithoutPlan(index int, id domain.FrameSetID)` (nosso, `PlanID` vazio) e `WithRepeatedFrame(index int, id domain.FrameSetID)` (segunda entrada para o mesmo número); `WithForeignFrame` e `WithIncompleteFrame` seguem como estão
- [X] T028 [P] Estender `test/helper/png_fixture.go`: `func WithoutPlanMark(data []byte) []byte` — percorre os blocos de um PNG de quadro e devolve os bytes sem o bloco `tEXt` cujo conteúdo começa por `Sobrevoo\x00plan=` (mantém assinatura e todos os outros blocos, sem recalcular nada), para os testes de `pngfile` montarem um quadro de antes da versão 2; e `func TruncatedAt(data []byte, size int) []byte`
- [X] T029 Rodar `make generate` (novos `internal/domain/mockdomain/video_encoder.go` e `video_exporter.go`), `make test` e `make lint`; conferir que nada da Parte B regrediu (depende de T018, T022, T026, T027, T028)

**Checkpoint C**: erros, códigos, tipos, portas, configuração, mocks, builders e
fixture prontos; `make test` verde.

---

## Phase 3: User Story 1 — Montar o vídeo a partir do plano e dos quadros (Priority: P1) 🎯 MVP

**Objetivo**: `sobrevoo video <plano> <quadros> --output voo.mp4` grava um MP4 com
um quadro de vídeo por imagem, na ordem e na taxa do plano, e imprime o resumo.

**Teste Independente**: com o `ffmpeg` de mentira, as portas mockadas e o binário
real (`quickstart.md` item 1, com o `ffmpeg` real): o arquivo existe, abre, tem a
quantidade de quadros, a duração e a taxa do plano e a resolução das imagens.

### Testes da US1 (escreva primeiro; devem falhar)

- [X] T030 [P] [US1] Estender `internal/infra/outbound/pngfile/frame_repository_test.go` com os cenários de `List(dir string) (domain.FrameDirectory, error)`: lista só os arquivos que casam exatamente com `frame_NNNNNN.png` (seis dígitos), ordenados por número, ignorando subdiretórios e outros nomes (`notas.txt`, `capa.png`, `frame_3.png`); para cada um devolve `Ours`, `SetID`, `PlanID`, `Width`, `Height` e `Whole` (quadros gravados com o próprio `Save`; um truncado com `os.Truncate`; um sem o bloco do plano com `helper.WithoutPlanMark`; um arquivo com nome de quadro que não é PNG → `Ours` falso); um diretório vazio → `FrameDirectory` sem arquivos e sem erro; diretório inexistente → `errors.Is(err, domain.ErrFrameDirectoryInvalid)` (mensagem `frame directory <dir> does not exist`); caminho que é um arquivo → `ErrFrameDirectoryInvalid` (`is not a directory`); diretório sem permissão de leitura (`skipWithoutPermissions`) → `ErrFrameDirectoryInvalid`; `List` **não** cria nada e **não** pergunta pela resolução (o `Complete` de `Inspect` não é preenchido por `List`)
- [X] T031 [P] [US1] Criar `internal/infra/outbound/videofile/video_exporter_test.go` (pacote `videofile_test`) com os cenários de `Export(path, overwrite, produce)`: entrega a `produce` o caminho **absoluto** de um arquivo na pasta do destino (mesmo com `path` relativo, `filepath.IsAbs`), diferente do destino; publica o que `produce` escreveu como `path` e devolve o tamanho em bytes; destino já existente sem `overwrite`: `errors.Is(err, domain.ErrVideoDestinationExists)`, mensagem `video destination already exists: <path>; use --overwrite to replace it`, arquivo intacto, nenhum `.sobrevoo-*.tmp`; com `overwrite` o arquivo é substituído por inteiro; `produce` devolve um erro: o erro volta como veio (não vira `ErrVideoDestinationInvalid`), nenhum arquivo novo, o anterior intacto, nenhum temporário; pasta do destino inexistente → `ErrVideoDestinationInvalid`; o arquivo publicado tem permissão `0o644`; e os cenários de `Check(path, overwrite)`: arquivo inexistente numa pasta existente → `nil`; existente sem `overwrite` → `ErrVideoDestinationExists` (mesma mensagem de `Export`); existente com `overwrite` → `nil`; o caminho é um diretório (com ou sem `overwrite`) → `ErrVideoDestinationInvalid` com `is a directory`; pasta inexistente → `ErrVideoDestinationInvalid` com `the folder does not exist`; pasta sem permissão de escrita (auxiliar `skipWithoutPermissions`, como em `pngfile`) → `ErrVideoDestinationInvalid`; `Check` **não deixa nada** (nem o destino, nem um temporário)
- [X] T032 [P] [US1] Criar `internal/infra/outbound/videoencoder/fake_ffmpeg_test.go` (pacote `videoencoder_test`, `//go:build !windows`) com o auxiliar `fakeFFmpeg(t *testing.T) string`: grava em `t.TempDir()` o script `#!/bin/sh` que (a) registra, um por linha, todos os argumentos em `$FAKE_FFMPEG_LOG` e o diretório de trabalho (`pwd`) em `$FAKE_FFMPEG_CWD`; (b) com `-version` imprime `ffmpeg version 7.1 Copyright (c) 2000-2024 the FFmpeg developers`; (c) com `-encoders` imprime uma lista com a linha ` V....D libx264              libx264 H.264 / AVC / MPEG-4 AVC / MPEG-4 part 10 (codec h264)` (menos quando `FAKE_FFMPEG_MODE=nolibx264`); (d) com `-i` (uma codificação) cria o arquivo de saída (o último argumento, sem o prefixo `file:`) com o texto `fake-mp4` e sai com 0; `FAKE_FFMPEG_MODE=fail`: escreve `fake: No space left on device` em stderr e sai com 1; `FAKE_FFMPEG_MODE=progress`: escreve em stdout blocos `frame=10\nprogress=continue\n`, `frame=60\nprogress=continue\n`, `frame=120\nprogress=end\n`; `FAKE_FFMPEG_MODE=sleep`: grava `$$` em `$FAKE_FFMPEG_PID` e `exec sleep 30`; `FAKE_FFMPEG_MODE=flood`: escreve 1 MiB em stdout e 1 MiB em stderr antes de criar a saída; devolve o caminho do script; e o auxiliar `argsOf(t)` que lê o registro
- [X] T033 [P] [US1] Criar `internal/infra/outbound/videoencoder/ffmpeg_video_encoder_test.go` (pacote `videoencoder_test`, `//go:build !windows`, sem `t.Parallel`) com os cenários da US1: `Probe` com o script → `EncoderInfo{Name: "ffmpeg", Version: "7.1", Codec: "libx264"}`; `Encode` roda o programa **com o diretório dos quadros como diretório de trabalho** (o `pwd` registrado é o de `job.Directory`) e passa, nesta ordem relativa, `-framerate 29.97`, `-start_number 0`, `-i frame_%06d.png` (só o nome, sem caminho) e `-frames:v 1139` (a taxa escrita como o decimal mais curto: `30`, `29.97`, `12.5`); passa `-nostdin -hide_banner -loglevel error -y`, `-c:v libx264 -preset medium -crf 23 -profile:v high`, `-vf` com `scale=out_color_matrix=bt709:out_range=tv:flags=accurate_rnd+full_chroma_int+bitexact,format=yuv420p`, `-colorspace bt709 -color_primaries bt709 -color_trc bt709 -color_range tv`, `-an -movflags +faststart -f mp4` e, por último, a saída como `file:<job.Output>`; o arquivo de saída existe depois de `Encode` (o script o criou) e `Encode` devolve `nil`; o processo que sai com 1 e escreve em stderr → `errors.Is(err, domain.ErrVideoEncodingFailed)` com a mensagem `video encoding failed: ffmpeg exited with status 1: fake: No space left on device`; a cauda da saída de erro é limitada a 2 KiB (o modo `flood` devolve uma mensagem de no máximo 2 KiB de causa e **não trava**, com prazo de 10 s)
- [X] T034 [P] [US1] Criar `internal/application/video_service_test.go` (pacote `application_test`, portas mockadas de `mockdomain`) com os cenários da US1: `Assemble` chama, **nesta ordem** (`gomock.InOrder`), `repository.List(request.Directory)`, `encoder.Probe(ctx)` e `exporter.Export(request.Output, request.Overwrite, produce)`; o `produce` que o teste captura, ao ser chamado com `"/tmp/x.tmp"`, chama `encoder.Encode(ctx, job, …)` com `job == EncodeJob{Directory: request.Directory, Frames: len(plan.Frames), FrameRate: plan.Parameters.FrameRate, Quality: request.Quality, Output: "/tmp/x.tmp"}`; o resumo traz `Frames`, `FrameRate`, `Quality`, `Encoder` (o `EncoderInfo` do `Probe`), `Resolution` (a do **primeiro quadro listado**, `Width` × `Height`), `SizeBytes` (o que `Export` devolveu) e `Elapsed` ≥ 0; uma listagem sem nenhum quadro → `ErrFrameDirectoryInvalid` sem chamar `Probe` nem `Export`; erro de `List` devolvido como veio (nada mais é chamado); erro de `Probe` devolvido como veio (`Export` não é chamado); erro de `Export`/`Encode` devolvido como veio; nos erros o resumo ainda traz o que se sabe (`Frames`, `FrameRate`, `Quality`)
- [X] T035 [P] [US1] Criar `internal/infra/inbound/cli/video_test.go` (pacote `cli_test`, `mockapplication.MockCameraPlanService` e `MockVideoService`; reusar `planOfFrames` de `render_frame_test.go`) com os cenários da US1: dois argumentos e `--output` obrigatório (senão erro de uso, `ExitCode` 2, serviço não chamado); `Load(planPath)` do `CameraPlanService` e depois `Assemble(ctx, plan, domain.VideoRequest{Directory: <arg 2>, Output: <--output>, Quality: <padrão injetado>, Overwrite: false}, …)`; o resumo em `stdout` com os rótulos e a ordem de `contracts/cli.md` (`Video written to <output>`, `Frames: 1350`, `Duration: 00:00:45.000`, `Resolution: 1080x1920`, `Frame rate: 30 fps`, `Quality: medium`, `Size: 18.4 MiB` para 19293798 bytes, `Encoder: ffmpeg 7.1 (libx264)`, `Time: 00:01:52`); com taxa `29.97` e 1350 quadros: `Frame rate: 29.97 fps` e `Duration: 00:00:45.045`; `Size` em `B` (`512 B`), `KiB` (`1.0 KiB` para 1024), `MiB` e `GiB` (uma casa); erro do `Load` devolvido como veio, serviço não chamado; erro de `Assemble` devolvido como veio e **nenhum** resumo impresso

### Implementação da US1

- [X] T036 [US1] Editar `internal/domain/frame_set.go`: acrescentar à porta `FrameRepository` o método `List(dir string) (FrameDirectory, error)` com o comentário do `data-model.md` ("lists the frame files of dir — the files named FrameFileName(n) —, sorted by number, saying for each what its image says of itself … Unlike Inspect, a dir that does not exist, is not a directory or cannot be read is an error: ErrFrameDirectoryInvalid"); rodar `make generate` (regera `mockdomain/frame_repository.go`); **logo em seguida T037**, porque `pngfile.FrameRepository` deixa de compilar até lá (depende de T029, T030)
- [X] T037 [US1] Editar `internal/infra/outbound/pngfile/frame_repository.go`: extrair de `Inspect` a leitura do diretório e dos arquivos para `scanFrames(dir string) ([]domain.FrameFile, error)` (só nomes de `domain.ParseFrameFileName`, sem diretórios, ordenados por número, com `inspectFrame`), e implementar `List` sobre ela: `os.Stat`/`os.ReadDir` com falha → `fmt.Errorf("%w: frame directory %s does not exist" | "%s is not a directory" | "%s: %w", domain.ErrFrameDirectoryInvalid, …)`; `Inspect` passa a usar `scanFrames` e continua com o mesmo comportamento (diretório inexistente = vazio; erros = `ErrFrameDestinationInvalid`; `Complete` calculado com a resolução) (depende de T036)
- [X] T038 [US1] Criar `internal/infra/outbound/videoencoder/ffmpeg_video_encoder.go` (pacote `videoencoder` com comentário de pacote: strategies of the `VideoEncoder` port; **estratégia** `FFmpeg`, construtor `NewFFmpeg(binary string) FFmpeg`, receiver `e`): `Probe(ctx)` — `exec.LookPath(e.binary)`; `-hide_banner -version` (a versão é o terceiro campo da primeira linha, `ffmpeg version 7.1 Copyright…` → `7.1`); `-hide_banner -encoders` e procura o codificador `libx264`; devolve `EncoderInfo{Name: "ffmpeg", Version, Codec: "libx264"}`; qualquer falha → `fmt.Errorf("%w: …", domain.ErrEncoderUnavailable)` (a redação com as dicas é da US3); `Encode(ctx, job, progress)` — `exec.CommandContext(ctx, binary, args...)` com `cmd.Dir = job.Directory`, os argumentos exatamente como em T033 (a taxa escrita por `strconv.FormatFloat(rate, 'f', -1, 64)`; `preset medium` e `crf 23` fixos por enquanto), `cmd.Stderr` numa cauda limitada a 2 KiB (um `io.Writer` que guarda só os últimos 2048 bytes), `cmd.Stdout` descartado; saída diferente de zero com o contexto vivo → `fmt.Errorf("%w: ffmpeg exited with status %d: %s", domain.ErrVideoEncodingFailed, code, cauda)` (a cauda sem quebras de linha nas pontas); o `progress` ainda não é usado; **não** registrar o caminho de saída em nenhum lugar além do argumento (depende de T032, T033)
- [X] T039 [US1] Criar `internal/infra/outbound/videofile/video_exporter.go` (pacote `videofile`, comentário de pacote: the outbound adapter that keeps a video in a file; construtor pela porta `NewVideoExporter()` → `VideoExporter`, receiver `e`): `Export(path, overwrite, produce)` — `filepath.Abs(path)` (falha → `ErrVideoDestinationInvalid`); `atomicfile.PublishPath(abs, overwrite, produce)`; `errors.Is(err, atomicfile.ErrExists)` → `fmt.Errorf("%w: %s; use --overwrite to replace it", domain.ErrVideoDestinationExists, path)`; `atomicfile.ErrInvalid` → `fmt.Errorf("%w: %s: %w", domain.ErrVideoDestinationInvalid, path, err)`; qualquer outro erro (o de `produce`) volta como veio; o tamanho vem de `os.Stat` do destino publicado; e `Check(path, overwrite)` — `filepath.Abs`; `os.Stat(path)`: diretório → `ErrVideoDestinationInvalid` (`is a directory`); existe e `!overwrite` → `ErrVideoDestinationExists` (mesma redação de `Export`); `os.Stat` da pasta: inexistente → `ErrVideoDestinationInvalid` (`the folder does not exist`), não é diretório → idem; gravabilidade: `os.CreateTemp(pasta, ".sobrevoo-*.tmp")` e remoção imediata, falha → `ErrVideoDestinationInvalid` com a causa; o tipo satisfaz `domain.VideoExporter` (depende de T004, T022, T031)
- [X] T040 [US1] Criar `internal/application/video_service.go` (`//go:generate go run go.uber.org/mock/mockgen -destination mockapplication/video_service.go -package mockapplication . VideoService` logo após `package`; imports; a interface no início): `VideoService` com `Assemble(ctx context.Context, plan domain.CameraPlan, request domain.VideoRequest, progress func(domain.VideoProgress)) (domain.VideoSummary, error)`, `videoService` (`repository domain.FrameRepository`, `encoder domain.VideoEncoder`, `exporter domain.VideoExporter`), `NewVideoService(repository, encoder, exporter)`; a US1 faz `List` → recusa a listagem vazia com `fmt.Errorf("%w: …", domain.ErrFrameDirectoryInvalid)` → resolução de `Files[0]` → `Probe` → `Export` cujo `produce` chama `Encode` com o `EncodeJob` de T034 → preenche `SizeBytes` e `Elapsed` (`time.Since`); o resumo sai preenchido também nos erros; **só orquestra**: nenhuma regra de negócio no serviço; rodar `make generate` (depende de T022, T034, T036)
- [X] T041 [US1] Criar `internal/infra/inbound/cli/video.go`: `NewVideoCommand(cameraPlanService application.CameraPlanService, videoService application.VideoService, defaultQuality domain.VideoQuality, options ...RenderOption) *cobra.Command` (`Use: "video <plan-file> <frames-directory>"`, `Short: "Join the frames of a flight into a video"`, `SilenceErrors`/`SilenceUsage` como `render.go`, `Args` com erro de uso para outro número de argumentos, `SetFlagErrorFunc` → `newUsageError`); flags `--output` (obrigatória: `newUsageError(fmt.Errorf("--output is required"))`) e `--overwrite`; `runVideo`: `Load` do plano, `Assemble` com `VideoRequest{Quality: defaultQuality, …}` e `progress` nulo por enquanto, e, em sucesso, `formatVideoSummary(request.Output, summary)` em `stdout` com os rótulos e a ordem de `contracts/cli.md`; auxiliares `formatSize(bytes int64) string` (`B` sem casa; `KiB`, `MiB`, `GiB` com uma casa; base 1024), `formatVideoDuration(time.Duration) string` (`hh:mm:ss.mmm`) e `formatFrameRate(float64) string` (`strconv.FormatFloat(v, 'f', -1, 64)`), reusando `formatElapsed` e `formatResolution` de `render_frame.go`/`render_all.go` (depende de T035, T040)
- [X] T042 [US1] Editar `cmd/sobrevoo/main.go`: `videoExporter := videofile.NewVideoExporter()`, `videoEncoder := videoencoder.NewFFmpeg(cfg.FFmpegBinary)`, `videoService := application.NewVideoService(frameRepository, videoEncoder, videoExporter)` e `root.AddCommand(cli.NewVideoCommand(cameraPlanService, videoService, domainVideoQuality(cfg.VideoDefaults.Quality)))` (depende de T026, T038, T039, T041)
- [X] T043 [US1] Rodar `make test`, `make lint`, `make generate` (sem diff) e `make build`; e, **com o `ffmpeg` real**, o item 1 do `quickstart.md` (pré-requisitos, `sv video … --output /tmp/voo.mp4`, `ffprobe`: `nb_read_frames=380`, `r_frame_rate=10/1`, `duration=38`, sem áudio; PSNR mínimo acima de 35 dB): registrar tamanho e tempo em "(anotar)" (depende de T042) **[Confirmado com o `ffmpeg` 9.0.2 real (item 1 do `quickstart.md`, refeito depois da correção do item 6): `Frames: 380`, `Duration: 00:00:38.000`, `Resolution: 360x640`, `Frame rate: 10 fps`, `Size: 1.4 MiB`, `Encoder: ffmpeg 9.0.2 (libx264)`; `ffprobe` confere `nb_read_frames=380`, `r_frame_rate=10/1`, `duration=38.000000`, sem áudio; 380 pares de PSNR, pior valor 37,54 dB.]**

**Checkpoint US1**: um vídeo correto, no nível padrão, de ponta a ponta.

---

## Phase 4: User Story 2 — Conferir os quadros antes de começar (Priority: P2)

**Objetivo**: recusar, antes de codificar e com a mensagem exata, quadros de outro
plano ou sem a identificação, de resoluções diferentes ou ímpares, de conjuntos
misturados, com falta, excesso ou repetição, ou truncados.

**Teste Independente**: `quickstart.md` item 5: cada variação do diretório recusa
com o código certo e nenhum `.mp4` é criado.

### Testes da US2 (escreva primeiro; devem falhar)

- [X] T044 [P] [US2] Criar `internal/domain/frame_verification_test.go` (pacote `domain_test`; `builddomain.NewCameraPlanBuilder()` com `N` quadros e `NewFrameDirectoryBuilder()`; o `PlanID` esperado é `plan.ID()`) com os cenários de `FrameDirectory.Verify(plan) (Resolution, error)`, **um `t.Run` por cenário e na ordem das verificações**: (1) diretório sem nenhum quadro nosso → `ErrFrameDirectoryInvalid`, e com 3 arquivos de nome de quadro que não são nossos a mensagem diz `3 files named like frames were ignored` e que não trazem a identificação da ferramenta (com 1: `1 file named like a frame was ignored`); (2) quadros nossos sem `PlanID` → `ErrFramesWithoutPlanID` com `1350 of 1350` e a orientação `draw them again with "render all --overwrite"`; (3) quadros com `PlanID` diferente → `ErrFramesDoNotMatchPlan` com `1350 of 1350 carry another plan identification (plan 3fa9c1d2e4b7, frames b71e0a55c2d9)` — as duas identificações abreviadas em **12 caracteres**; um só quadro de outro plano no meio → `1 of 1350`; a identificação de plano ausente tem precedência sobre a de outro plano; (4) resoluções diferentes → `ErrFrameResolutionInvalid` com `1080x1920 is the resolution of 1340 frames, but 10 differ (…)`, a resolução esperada = a **mais frequente** e, em empate, a do menor número, listando no máximo **5** quadros que destoam (`frame_000100.png is 540x960`) e `, and 5 more`; largura ou altura ímpar → `ErrFrameResolutionInvalid` com `the frames are 1081x1921, but width and height must be even for the video`; a resolução antes do conjunto: quadros do mesmo plano, de resoluções e de `SetID` diferentes → `ErrFrameResolutionInvalid`, **não** `ErrFramesDoNotMatchPlan`; (5) mesmo plano, mesma resolução, `SetID` diferentes → `ErrFramesDoNotMatchPlan` com `mixes frames of 2 different sets` (e `3` para três); (6) numeração: faltas → `ErrFrameSequenceInvalid` com `4 missing (12-15)`, faixas `0-3, 7, 10-12`, no máximo **10 faixas** e depois `and N more`, e o total à vista (`37 missing in all`); excedentes → `2 not in the plan (1260-1261; the plan has frames 0 to 1259)`; repetição (`WithRepeatedFrame(3, …)`) → `number 3 appears more than once`; faltas e excedentes juntos na mesma mensagem, separados por `, `; um arquivo de nome de quadro que não é nosso num número que falta → a mensagem acrescenta `frame_000012.png was ignored: it does not carry this tool's identification`; (7) um quadro nosso não `Whole` → `ErrFrameFileInvalid` com `frame_000007.png` (até 5 nomes, `and N more`); a ordem: numeração antes de integridade; o sucesso devolve a `Resolution` comum e `nil`; e um plano de 1 quadro com 1 quadro é válido
- [X] T045 [P] [US2] Estender `internal/application/video_service_test.go`: `Assemble` chama `directory.Verify(plan)` (a regra concreta do domínio, com `FrameDirectoryBuilder` — não mockada) entre `List` e `Probe`; um `Verify` que recusa (por exemplo, faltam quadros) devolve o **erro do domínio como veio**, sem chamar `Probe` nem `Export`; a `Resolution` do resumo passa a ser a que `Verify` devolve (com quadros de 360 × 640); a listagem sem quadros continua recusada (agora por `Verify`)
- [X] T046 [P] [US2] Estender `internal/infra/inbound/cli/video_test.go`: cada erro de conferência (`ErrFrameDirectoryInvalid`, `ErrFrameSequenceInvalid`, `ErrFrameResolutionInvalid`, `ErrFramesDoNotMatchPlan`, `ErrFramesWithoutPlanID`, `ErrFrameFileInvalid`) devolvido pelo serviço mockado sai como veio (mensagem preservada) e **nenhum** resumo é impresso

### Implementação da US2

- [X] T047 [US2] Criar `internal/domain/frame_verification.go` (pacote `domain`): `func (d FrameDirectory) Verify(plan CameraPlan) (Resolution, error)` — só considera `Ours`, **para no primeiro tipo de problema** e na ordem da tabela de `data-model.md` (1 há quadro nosso; 2 todos com `PlanID`; 3 todos com `PlanID == plan.ID()`; 4 mesma largura e altura, pares; 5 mesmo `SetID`; 6 numeração de 0 a `len(plan.Frames)−1` sem falta, excesso nem repetição; 7 todo quadro `Whole`), com os erros e as mensagens de T044 (todas em inglês, embrulhando o sentinela com `%w`); funções puras de apoio no mesmo arquivo: `numberRanges(numbers []int, maxRanges int) string` (`12-15, 40`, no máximo 10 faixas, depois `, and N more`), `shortID(id string) string` (12 primeiros caracteres), `mostFrequentResolution(...)` (empate: a do menor número) e um limitador de nomes de arquivo (5 e `, and N more`); nenhum `os` nem outra E/S (depende de T027, T044)
- [X] T048 [US2] Editar `internal/application/video_service.go`: depois de `List`, `resolution, err := directory.Verify(plan)` (o erro volta como veio); a resolução do resumo vem dele; remover a leitura de `Files[0]` e a recusa da listagem vazia da US1 (agora é a verificação 1) (depende de T045, T047)
- [X] T049 [US2] Rodar `make test`, `make lint` e `make generate` (sem diff) e, com o `ffmpeg` real ou sem ele (as recusas ocorrem antes do codificador), o item 5 do `quickstart.md`: cada variação sai com `41`, `40`, `43`, `42`, `45` e `44` e nenhum `/tmp/n.mp4` é criado; anotar o tempo da conferência de 380 quadros (depende de T048)

**Checkpoint US2**: nenhum vídeo errado sai em silêncio; cada recusa diz o que falta ou destoa.

---

## Phase 5: User Story 3 — Saber quando falta o codificador de vídeo (Priority: P3)

**Objetivo**: uma mensagem acionável quando o `ffmpeg` não existe ou não tem o
`libx264`, antes de qualquer codificação.

**Teste Independente**: `quickstart.md` item 6 (com `PATH` sem o `ffmpeg`): código
46, mensagem com o que instalar, nenhum arquivo criado.

### Testes da US3 (escreva primeiro; devem falhar)

- [X] T050 [P] [US3] Estender `internal/infra/outbound/videoencoder/ffmpeg_video_encoder_test.go` com os cenários de `Probe` que falham: um nome que não existe no `PATH` (`NewFFmpeg("sobrevoo-no-such-encoder")`) → `errors.Is(err, domain.ErrEncoderUnavailable)` e a mensagem contém `"sobrevoo-no-such-encoder" was not found on the PATH`, `brew install ffmpeg`, `sudo apt install ffmpeg`, `winget install Gyan.FFmpeg` e `ffmpeg -version` — o **nome configurado** aparece nas aspas e no `-version`; o script com `FAKE_FFMPEG_MODE=nolibx264` → `ErrEncoderUnavailable` com `ffmpeg 7.1 has no libx264 encoder; install a build that includes it` (e as dicas de macOS e Debian); um arquivo que existe mas não é executável (`0o644`) → `ErrEncoderUnavailable` com `could not be run`; um script cujo `-version` sai com 1 → `ErrEncoderUnavailable` com `could not be run`
- [X] T051 [P] [US3] Estender `internal/application/video_service_test.go`: um `Probe` que devolve `ErrEncoderUnavailable` → devolvido como veio e `Export` **não** é chamado; a ordem com dois problemas: quadros que `Verify` recusa **e** `Probe` que recusaria → o erro dos quadros e `Probe` **não é chamado** (o mock estrito prova)

### Implementação da US3

- [X] T052 [US3] Editar `internal/infra/outbound/videoencoder/ffmpeg_video_encoder.go`: as mensagens de `Probe`, montadas pelo adapter (ele conhece o programa e como instalá-lo): `video encoder not available: "<binary>" was not found on the PATH; install it (macOS: brew install ffmpeg; Debian/Ubuntu: sudo apt install ffmpeg; Windows: winget install Gyan.FFmpeg), then check it with: <binary> -version`; `video encoder not available: <binary> <versão> has no libx264 encoder; install a build that includes it (macOS: brew install ffmpeg; Debian/Ubuntu: sudo apt install ffmpeg)`; `video encoder not available: "<binary>" could not be run: <causa>`; a distinção entre "não encontrado" (`exec.ErrNotFound`) e "não executa" (qualquer outra falha de `LookPath`/execução) (depende de T050)
- [X] T053 [US3] Rodar `make test` e `make lint`; conferir que a ordem de verificações do serviço (entrada, depois codificador) já vale (T051) e, **sem** precisar do `ffmpeg`, o item 6 do `quickstart.md` (`PATH=/usr/bin:/bin`): `46` com a mensagem e nenhum arquivo; e com quadros errados **e** sem `ffmpeg`: `40`, não `46` (depende de T052)

**Checkpoint US3**: a falta do codificador é uma instrução, não um erro de sistema.

---

## Phase 6: User Story 4 — Escolher a qualidade por níveis nomeados (Priority: P4)

**Objetivo**: `--quality low|medium|high`, com padrão da configuração; um nível
mais alto dá arquivo maior ou igual, sem mudar duração, resolução, taxa nem ordem.

**Teste Independente**: `quickstart.md` item 3: os três níveis do mesmo voo, com
tamanhos crescentes e o mesmo `ffprobe`; nível inexistente = erro de uso.

### Testes da US4 (escreva primeiro; devem falhar)

- [X] T054 [P] [US4] Estender `internal/infra/outbound/videoencoder/ffmpeg_video_encoder_test.go`: um `t.Run` por nível — `Low` → `-preset veryfast -crf 28`, `Medium` → `-preset medium -crf 23`, `High` → `-preset slow -crf 18`; e um cenário de que as listas de argumentos dos três níveis são **idênticas** exceto pelos valores de `-preset` e `-crf`
- [X] T055 [P] [US4] Estender `internal/infra/inbound/cli/video_test.go`: `--quality low`, `medium` e `high` chegam a `VideoRequest.Quality` (um `t.Run` por valor); sem a flag, `VideoRequest.Quality` é o padrão passado ao construtor (o teste constrói o comando com padrão `VideoQualityLow` para provar que não está fixo em `medium`); `--quality ultra` → erro de uso (`ExitCode` 2) com a mensagem `--quality "ultra": use one of low, medium, high` e o serviço **não** chamado; o resumo imprime `Quality: low` para um resumo `VideoQualityLow`
- [X] T056 [P] [US4] Estender `internal/application/video_service_test.go`: a qualidade do pedido chega ao `EncodeJob` e ao resumo, um `t.Run` por nível

### Implementação da US4

- [X] T057 [US4] Editar `internal/infra/outbound/videoencoder/ffmpeg_video_encoder.go`: a tabela `settingsFor(quality domain.VideoQuality) (preset string, crf int)` (constantes do adapter, com o comentário de que são os valores de `contracts/video-file.md`: `low` `veryfast`/28, `medium` `medium`/23, `high` `slow`/18) no lugar do `medium` fixo (depende de T054)
- [X] T058 [US4] Editar `internal/infra/inbound/cli/video.go`: a flag `--quality` (`StringVar`, padrão `defaultQuality.String()`, uso `Video quality: low (fast, small), medium (for publishing) or high (for keeping)`); `domain.ParseVideoQuality` e, em caso de erro, `newUsageError(fmt.Errorf("--quality %w", err))` (a mensagem final `--quality "ultra": use one of low, medium, high`); a qualidade lida vai ao `VideoRequest` (depende de T020, T055)
- [X] T059 [US4] Rodar `make test` e `make lint`; **com o `ffmpeg` real**, o item 3 do `quickstart.md`: `low ≤ medium ≤ high` em tamanho (SC-009), mesmos `nb_read_frames`, duração, resolução e taxa nos três; registrar tamanhos e tempos em "(anotar)" (depende de T057, T058) **[Confirmado com o `ffmpeg` 9.0.2 real: 380 quadros de 360×640 — `low` 930 582 B/0,67 s, `medium` 1 442 368 B/1,13 s, `high` 2 357 240 B/2,08 s; mesma duração/resolução/taxa/quantidade nos três.]**

**Checkpoint US4**: a qualidade se escolhe por nome, sem parâmetros de codificação.

---

## Phase 7: User Story 5 — Acompanhar o progresso e ver o resumo (Priority: P5)

**Objetivo**: mostrar, durante a codificação, quantos quadros já foram
codificados, de quantos e há quanto tempo, sem alterar o vídeo.

**Teste Independente**: `quickstart.md` item 1: linha de progresso em `stderr`
(reescrita em terminal, no máximo a cada 5 s fora dele) e resumo conferido com o
`ffprobe`.

### Testes da US5 (escreva primeiro; devem falhar)

- [X] T060 [P] [US5] Criar `internal/infra/outbound/videoencoder/ffmpeg_progress_test.go` (pacote `videoencoder`, teste **interno**, para alcançar a função não exportada): `readProgress(r io.Reader, progress func(int))` chama `progress` com o valor de cada linha `frame=<n>` (`frame=10`, `frame=60`, `frame=120` → `10`, `60`, `120`); ignora chaves desconhecidas (`fps=…`, `bitrate=…`, `out_time=…`, `progress=continue`), linhas em branco e `frame=N/A`; não chama `progress` para um valor que **diminui** (o `ffmpeg` não faz isso, mas o adapter garante a monotonicidade); uma última linha sem quebra de linha ainda é lida; entrada vazia não chama `progress`
- [X] T061 [P] [US5] Estender `internal/infra/outbound/videoencoder/ffmpeg_video_encoder_test.go`: os argumentos passam a incluir `-nostats -progress pipe:1`; com `FAKE_FFMPEG_MODE=progress`, o `progress` de `Encode` recebe `10`, `60`, `120` **em ordem** e `Encode` devolve `nil`; com `FAKE_FFMPEG_MODE=flood` (1 MiB em stdout e em stderr) `Encode` **não trava** e termina em até 10 s, e uma falha depois do fluxo (`fail`) ainda traz a cauda de stderr; `progress` nulo é aceito
- [X] T062 [P] [US5] Estender `internal/application/video_service_test.go`: para cada `encoded` que o `Encode` mockado informa, `progress` recebe `VideoProgress{Done: encoded, Total: len(plan.Frames), Elapsed: ≥ 0}`; em sucesso há uma **última** chamada com `Done == Total`; em falha do `Encode` não há chamada final; `progress` nulo é aceito
- [X] T063 [P] [US5] Criar `internal/infra/inbound/cli/video_progress_test.go` (pacote `cli_test`, como `render_progress_test.go`): em terminal, cada atualização escreve `\rEncoding frame 412/1350 (30.5%), elapsed 00:00:41` (sem quebra de linha) e `finish()` termina a linha (uma vez, e só se algo foi escrito); fora de terminal: só escreve quando passaram **5 s ou mais** desde a última linha escrita (medidos por `Elapsed`, sem relógio) ou quando `Done == Total`, uma linha por vez (`Encoding frame 380/380 (100.0%), elapsed 00:00:38`); a primeira atualização de 1 s não escreve; a porcentagem com uma casa
- [X] T064 [P] [US5] Estender `internal/infra/inbound/cli/video_test.go`: o comando passa ao serviço um `progress` que escreve em `stderr` (com `WithTerminalCheck` verdadeiro e falso, um `t.Run` para cada), o `finish` é chamado depois de `Assemble` e o resumo sai em `stdout`; em erro, o `finish` também é chamado

### Implementação da US5

- [X] T065 [US5] Criar `internal/infra/outbound/videoencoder/ffmpeg_progress.go` (`readProgress(r io.Reader, progress func(int))` com `bufio.Scanner`, monotônico; e o `tailWriter` de 2 KiB que hoje está em `ffmpeg_video_encoder.go`, se preferir movê-lo) e editar `internal/infra/outbound/videoencoder/ffmpeg_video_encoder.go`: acrescentar `-nostats -progress pipe:1`, ligar o `cmd.StdoutPipe()` a uma goroutine que chama `readProgress` e `cmd.Stderr` à cauda, e **esperar a goroutine terminar antes de devolver** (nada vaza; sem deadlock com o modo `flood`) (depende de T060, T061)
- [X] T066 [US5] Editar `internal/application/video_service.go`: o `onProgress(encoded int)` que guarda `summary.Encoded` e chama `progress(domain.VideoProgress{Done: encoded, Total: len(plan.Frames), Elapsed: time.Since(started)})`; em sucesso, uma chamada final com `Done == Total` (depende de T062)
- [X] T067 [US5] Criar `internal/infra/inbound/cli/video_progress.go` (`videoProgressPrinter` com `out io.Writer`, `terminal bool`, `written bool`, `lastPrinted time.Duration`; `report(domain.VideoProgress)` e `finish()`, como `progressPrinter` de `render_progress.go`, sem o mudar) e editar `internal/infra/inbound/cli/video.go` (o `runVideo` cria o impressor com `settings.isTerminal(cmd.ErrOrStderr())`, passa `printer.report` ao `Assemble` e chama `printer.finish()` depois) (depende de T063, T064, T066)
- [X] T068 [US5] Rodar `make test` e `make lint`; **com o `ffmpeg` real**, o item 1 do `quickstart.md` conferindo o progresso em terminal e com `2>&1 | cat` (no máximo uma linha a cada 5 s e a última) e o resumo contra o `ffprobe` (`Frames`, `Duration`, `Resolution`, `Frame rate`) (depende de T067) **[Confirmado com o `ffmpeg` real: progresso em terminal e em log, e o `ffprobe` bate com o resumo (`Frames`, `Duration`, `Resolution`, `Frame rate`).]**

**Checkpoint US5**: o usuário acompanha a codificação e confere o resumo.

---

## Phase 8: User Story 6 — Proteger o destino e não deixar arquivo parcial (Priority: P6)

**Objetivo**: nunca sobrescrever sem pedir, recusar cedo um destino que não serve,
nunca deixar arquivo parcial nem temporário, e interromper com ordem e código 49.

**Teste Independente**: `quickstart.md` item 7: destino existente, com e sem
`--overwrite`; destino inválido; interrupção em pontos diferentes (sem arquivo, sem
temporário, sem `ffmpeg` órfão); falha do codificador.

### Testes da US6 (escreva primeiro; devem falhar)

- [X] T069 [P] [US6] Estender `internal/infra/outbound/videoencoder/ffmpeg_video_encoder_test.go` com os cenários do cancelamento (com `FAKE_FFMPEG_MODE=sleep`): cancelar o contexto durante `Encode` faz `Encode` voltar em **menos de 3 s** com um erro que satisfaz `errors.Is(err, context.Canceled)` e **não** `ErrVideoEncodingFailed`; o processo do script **não existe mais** depois (o pid lido de `$FAKE_FFMPEG_PID`; `syscall.Kill(pid, 0)` devolve `ESRCH`, em até 1 s); um contexto já cancelado antes de `Encode` devolve o erro do contexto sem deixar processo
- [X] T070 [P] [US6] Estender `internal/application/video_service_test.go`: `exporter.Check` (o mock; o adapter já existe desde a US1) é chamado **depois de `Verify` e antes de `Probe`** (`gomock.InOrder`); um `Check` que recusa (`ErrVideoDestinationExists`, `ErrVideoDestinationInvalid`) devolve o erro como veio, sem chamar `Probe` nem `Export`; a ordem dos erros com problemas em tudo: quadros → destino → codificador; `Overwrite` do pedido chega a `Check` e a `Export`; a interrupção: `ctx` cancelado e o `Export` mockado devolvendo o erro do contexto (`context.Canceled` embrulhado) → `errors.Is(err, domain.ErrVideoInterrupted)`, `summary.Interrupted` verdadeiro e `summary.Encoded` igual ao último progresso informado (ex.: 412); um erro do `Export` sem o contexto cancelado **não** vira interrupção
- [X] T071 [P] [US6] Estender `internal/infra/inbound/cli/video_test.go`: `--output voo.mkv` (e `voo`, `voo.mp4.txt`) → erro de uso (`ExitCode` 2) com `--output must end in .mp4: the video is always an MP4 file` e o serviço **não** chamado; `.MP4` maiúsculo é aceito; `--overwrite` chega a `VideoRequest.Overwrite`; um resumo `Interrupted` (o serviço mockado devolve `ErrVideoInterrupted` com `Encoded: 412`, `Frames: 1350`, `Elapsed: 41 s`) imprime em `stdout` `Interrupted: 412 of 1350 frames were encoded; no video was written, run the same command again to start over` e `Time: 00:00:41`, devolve o erro (`ExitCode` 49) e **não** imprime o resumo de sucesso; um erro qualquer não imprime nada em `stdout`
- [X] T072 [P] [US6] Criar `internal/infra/inbound/cli/video_signal_test.go` (`//go:build !windows`, como `render_signal_test.go`): um SIGINT enviado ao processo cancela o contexto que o comando deu a `Assemble` (o mock espera o `ctx.Done()`), em até 3 s; SIGTERM idem

### Implementação da US6

- [X] T073 [US6] Editar `internal/infra/outbound/videoencoder/ffmpeg_video_encoder.go`: `Encode` verifica `ctx.Err()` antes de iniciar; com o contexto cancelado durante a execução, o `exec.CommandContext` mata o processo (o `Cancel` padrão é `Kill`), o adapter espera o término e devolve `fmt.Errorf("encoding stopped: %w", ctx.Err())` — **nunca** `ErrVideoEncodingFailed` quando o contexto está cancelado (depende de T069)
- [X] T074 [US6] Editar `internal/application/video_service.go`: `exporter.Check(request.Output, request.Overwrite)` entre `Verify` e `Probe`; a função `videoInterruption(ctx, summary *domain.VideoSummary, err error) error` (**não** generalizar `interruption` de `frame_service.go`: outros tipos; mesma regra: `ctx.Err() != nil` e o erro é `context.Canceled`/`DeadlineExceeded` → `summary.Interrupted = true` e `domain.ErrVideoInterrupted`) aplicada ao erro de `Export`; `Encoded` fica com o último progresso (depende de T070)
- [X] T075 [US6] Editar `internal/infra/inbound/cli/video.go`: a regra `--output` termina em `.mp4` (`strings.EqualFold(filepath.Ext(output), ".mp4")`, senão `newUsageError(fmt.Errorf("--output must end in .mp4: the video is always an MP4 file"))`, antes de ler o plano); `ctx, stop := interruptContext(commandContext(cmd))` (reusar de `render_progress.go`) e `defer stop()`; a impressão da interrupção de T071 quando `errors.Is(err, domain.ErrVideoInterrupted)` (depende de T071, T072, T074)
- [X] T076 [US6] Rodar `make test` e `make lint`; **com o `ffmpeg` real**, o item 7 do `quickstart.md`: `47` com o arquivo intacto (o `md5` de antes), `--overwrite` substitui, `48` para pasta inexistente e para um diretório chamado `x.mp4`, a interrupção (código `49`, nenhum `int.mp4`, nenhum `.sobrevoo-*.tmp`, nenhum `ffmpeg` órfão com `pgrep -x ffmpeg`) e a falha do codificador com um quadro corrompido por dentro (código `50`, nenhum arquivo) (depende de T075) **[Confirmado com o `ffmpeg` 9.0.2 real: `47` com o arquivo intacto (mesmo `md5`), `--overwrite` substitui, `48` para pasta e diretório inválidos, interrupção com `49` (zero e não zero quadros codificados, sem arquivo nem processo órfão). A falha do codificador exigiu duas correções, registradas em `research.md` item 6: (1) zerar bytes do `IDAT` não quebra a decodificação — é preciso corromper com bytes aleatórios; (2) mesmo corrompido de verdade, o `ffmpeg` por padrão **tolera** o erro e sai com sucesso — foi preciso acrescentar `-xerror`. Com as duas correções, `50` com a causa do `ffmpeg`, sem arquivo.]**

**Checkpoint US6**: o destino está protegido e nenhuma execução deixa resíduo.

---

## Phase 9: User Story 7 — Obter sempre o mesmo vídeo, sem dado do ambiente (Priority: P7)

**Objetivo**: o mesmo plano, os mesmos quadros, o mesmo nível e o mesmo `ffmpeg`
produzem o mesmo arquivo, sem data, versão, caminho, nome de máquina ou de usuário.

**Teste Independente**: `quickstart.md` item 4 (com o `ffmpeg` real): `cmp` de dois
vídeos gerados com 3 s de intervalo, em diretórios de trabalho e destinos
diferentes; `ffprobe` e `strings` sem nenhum dado do ambiente.

### Testes da US7 (escreva primeiro; devem falhar)

- [X] T077 [P] [US7] Estender `internal/infra/outbound/videoencoder/ffmpeg_video_encoder_test.go`: os argumentos incluem `-fflags +bitexact`, `-flags:v +bitexact`, `-map_metadata -1`, `-x264-params threads=4:info=0` (constante do adapter `x264Threads = 4`); um cenário que fixa **a lista completa e ordenada** de argumentos de um job (`medium`, 380 quadros, taxa 10) contra um literal do teste (para uma mudança na linha de comando não passar despercebida: ela muda os bytes do vídeo e precisa de uma nota de versão, `contracts/video-file.md`); um cenário de que dois jobs que só diferem em `Output`, `Directory` e `FrameRate` têm listas de argumentos iguais fora desses três valores; e um de que a lista **não contém** nada do ambiente: nenhum `$HOME`, o nome da máquina (`os.Hostname()`), o usuário, a data, nem o caminho de `Directory` (só `Output` aparece, com o prefixo `file:`), rodando `Encode` duas vezes com diretórios de trabalho do processo diferentes (`t.Chdir`) e com 1,1 s de intervalo e comparando os registros

### Implementação da US7

- [X] T078 [US7] Editar `internal/infra/outbound/videoencoder/ffmpeg_video_encoder.go`: acrescentar à linha de comando, na posição que T077 fixa, `-fflags +bitexact -flags:v +bitexact -map_metadata -1` e `-x264-params threads=4:info=0` (constante `x264Threads = 4` com o comentário de que o `x264` só é reproduzível com o mesmo número de threads e de que mudar o valor muda os bytes: só com uma versão nova da ferramenta, `research.md` item 6) (depende de T077)
- [X] T079 [US7] Rodar `make test` e `make lint`; **com o `ffmpeg` real**, o item 4 do `quickstart.md` (**confirmar**, `research.md` item 6): `cmp` de dois vídeos gerados com 3 s de intervalo, em diretórios de trabalho e destinos diferentes, `IDENTICOS`; `ffprobe -show_format -show_streams` e `strings -a` sem `creation_time`, `encoder`, `x264`, `Lavf`/`Lavc`, caminho, nome da máquina nem do usuário; repetir 5 vezes; **se algum campo aparecer, acrescentar a opção que o remove à linha de comando (T077 e T078 de novo, com a nota de versão) e só então prosseguir**; registrar em "Notas de implementação" o que foi preciso (depende de T078) **[Confirmado com o `ffmpeg` 9.0.2 real: `cmp` idêntico entre execuções com 3 s de intervalo e diretórios de trabalho diferentes, 5 repetições com o mesmo hash; `ffprobe`/`strings` sem nada do ambiente. A linha de comando original tinha dois enganos que só apareceram aqui — `-x264-params info=0` não existe e `-movflags +bitexact` não é mais válido nesta versão — corrigidos com `-bitexact` global, `-metadata:s:v:0 encoder=` e `-bsf:v filter_units=remove_types=6` (que também resolveu o achado de T076: a SEI do `x264` não tem opção que a desligue).]**
- [X] T080 [US7] **Com o `ffmpeg` real**, o item 8 do `quickstart.md`: os planos de `antimeridiano.gpx` e `polar.gpx` montados em vídeo curto (180 × 320, poucos quadros por segundo) têm as mesmas propriedades do item 1 (`nb_read_frames` igual a `Frames`, taxa, duração `Frames ÷ taxa`, mesmas recusas e códigos), sem tratamento de região (depende de T079) **[Confirmado: antimeridiano — 209 quadros, `r_frame_rate=5/1`, `duration=41.8`; polar — 196 quadros, `r_frame_rate=5/1`, `duration=39.2`; ambos batem exatamente com `Frames ÷ taxa`.]**

**Checkpoint US7**: o vídeo é reproduzível e não vaza nada do ambiente.

---

## Phase 10: Polish & Cross-Cutting Concerns

- [X] T081 [P] Editar `CLAUDE.md` (em português; blocos de código e identificadores em inglês): a descrição do projeto passa a **seis** features implementadas (`specs/006-video-assembly/` monta os quadros num vídeo MP4, `sobrevoo video`), e a frase "Ainda não há geração de vídeo" sai; os comandos de exemplo ganham `go run ./cmd/sobrevoo video plan.json frames/ --output flight.mp4 --quality medium`; a lista de entidades do domínio (`VideoQuality`, `VideoRequest`, `VideoProgress`, `EncodeJob`, `EncoderInfo`, `VideoSummary`, `FrameMark`), de erros sentinela (os 11 novos) e de portas (`VideoEncoder`, `VideoExporter`, e o método `FrameRepository.List`); o serviço `VideoService` (`Assemble`); os adapters `videoencoder` (`NewFFmpeg(binary)`: processo externo, o `ffmpeg` com `libx264`, que o usuário instala) e `videofile` (`NewVideoExporter()`), a mudança em `pngfile` (a marca do plano) e `atomicfile.PublishPath`; os tipos de `config` (`config.VideoDefaults`, `Config.FFmpegBinary`); os contratos (`specs/006-video-assembly/contracts/cli.md` e `video-file.md`); e um parágrafo "A montagem do vídeo (etapa 6)" no estilo de "O desenho dos quadros (etapa 5)": determinismo (mesmo `ffmpeg`, opções fixas, threads fixas), pré-requisito do `ffmpeg`, quadros sem a identificação do plano recusados
- [X] T082 [P] Editar `README.md` (em português): a seção da sexta etapa — o comando `sobrevoo video`, o pré-requisito (`ffmpeg` instalado: `brew install ffmpeg`), o fluxo completo `plan → geodata slice → render all → video`, os níveis de qualidade e um exemplo
- [X] T083 **Com o `ffmpeg` real**, os itens 2 (taxa 29,97), 5, 6 e 9 do `quickstart.md`, e o item 10 (dados reais de `resources/`: 1020 quadros da versão 2, `--quality high` e `medium`), preenchendo cada "(anotar)" com o valor medido (tamanho, tempo, PSNR, tempo da conferência, tempo da montagem de 1020 quadros; o critério de SC-007 é 10 minutos para 1350 quadros de 1080 × 1920 em `medium`), e conferir que o vídeo real abre no QuickTime e no VLC e mostra o relevo real, o traçado do passeio e o marcador em vertical; registrar em "Notas de implementação" os desvios **[Confirmado: o registro antigo (1020 quadros da versão 1) foi recusado com `44`; `render all --overwrite` (versão 2) levou 16 min 05 s, sem buracos; a montagem em `medium` levou 14 s / 6,7 MiB, e em `high`, 20 s / 9,6 MiB — bem dentro do limite de 10 min de SC-007; sem áudio; PSNR de 1020 pares, pior valor 40,07 dB; o vídeo abriu no QuickTime e no Safari (VLC não está instalado nesta máquina) e, extraindo quadros do meio e do fim, mostra o relevo real, o mapa sintético e o traçado crescendo corretamente.]**
- [X] T084 [P] Atualizar `specs/006-video-assembly/spec.md` (o **Status** passa de "Rascunho" para "Implementada") e `specs/006-video-assembly/checklists/requirements.md` se algo mudou; e, se a validação real (T079, T083) mudou algum parâmetro do adapter (opções do `ffmpeg`, `preset`/`crf`, threads), atualizar `research.md` (itens 5 e 6, trocando os **(confirmar)** pelo que foi confirmado) e `contracts/video-file.md`
- [X] T085 Verificação final: `gofmt -l .` sem saída; `make test` (todos os pacotes verdes), `make lint`, `make generate` (sem diff) e `go test -race ./internal/infra/outbound/videoencoder/... ./internal/application/... ./internal/infra/inbound/cli/...`; conferir que nenhum arquivo de `internal/domain` e `internal/application` importa `os`, `os/exec`, `image/png` ou `internal/infra` (`grep -rn '"os"\|"os/exec"\|image/png\|internal/infra' internal/domain internal/application --include='*.go' | grep -v _test.go` sem saída de código de produção); e `git status` sem arquivos temporários (`.sobrevoo-*.tmp`) nem os `.mp4` do quickstart dentro do repositório

---

## Dependências e Ordem de Execução

### Dependências entre fases

- **Phase 1 (Setup)**: sem dependências.
- **Phase 2 (Foundational)**: depende da Phase 1; **bloqueia todas as histórias**. As Partes A e B são sequenciais entre si (A antes de B para isolar regressões; B antes de C); dentro da Parte C, as tarefas `[P]` são paralelas.
- **Phases 3 a 9 (US1 a US7)**: dependem da Phase 2 concluída; cada uma **estende arquivos** das anteriores (ver a tabela "Divisão do trabalho"), então **seguem a ordem P1 → P7**, e só dentro de uma história as tarefas `[P]` são paralelas. A US2 a US7 dependem da US1 (o MVP).
- **Phase 10 (Polish)**: depende das histórias desejadas.

### Dependências entre histórias

- **US1 (P1)**: depende só da Foundational. Ao terminar, existe o vídeo.
- **US2 (P2)**: depende da US1 (`video_service.go`, `frame_verification.go` novo, `video_test.go`).
- **US3 (P3)**: depende da US1 (`Probe` do adapter) e, para a ordem das verificações, da US2.
- **US4 (P4)**: depende da US1 (o adapter e a CLI); independente de US2 e US3.
- **US5 (P5)**: depende da US1; independente de US2 a US4 (edita os mesmos arquivos, em sequência).
- **US6 (P6)**: depende da US1 (o exportador já tem `Export` e `Check`) e da US2 (a ordem entrada → destino → codificador).
- **US7 (P7)**: depende da US1 (a linha de comando) e é verificada com o `ffmpeg` real.

### Dentro de cada história

- Testes antes da implementação (devem falhar primeiro).
- Portas e tipos de domínio antes de adapters e serviços; serviço antes da CLI; CLI antes de `main.go`.
- A história é completa antes da próxima prioridade.

### Oportunidades de paralelismo

```text
# Foundational, Parte B (etapa 5), depois de T006:
T007 png_mark_test.go   T008 frame_repository_test.go + frame_exporter_test.go   T014 docs
T009 → T010 png_mark.go → T011 frame_repository.go + T012 frame_exporter.go → T013 frame_service.go

# Foundational, Parte C, tudo em paralelo:
T016 errors.go   T017 exit_code_test.go   T019 video_quality_test.go   T021 video_test.go
T023 config_test.go   T025 config_mapping_test.go   T027 builder   T028 png_fixture.go

# US1, testes em paralelo:
T030 frame_repository_test.go   T031 video_exporter_test.go   T032 fake_ffmpeg_test.go
T033 ffmpeg_video_encoder_test.go   T034 video_service_test.go   T035 video_test.go (cli)

# Depois, os adapters e o serviço em paralelo (arquivos diferentes):
T036 → T037 pngfile.List      T038 ffmpeg_video_encoder.go      T039 video_exporter.go
T040 video_service.go (depois de T036)

# US2 a US7: os testes de cada história são [P] entre si (arquivos de teste diferentes),
# mas a implementação de cada história edita ffmpeg_video_encoder.go, video_service.go e
# video.go, um de cada vez.
```

---

## Estratégia de Implementação

### MVP Primeiro (Phase 1 + 2 + User Story 1)

1. Phase 1 (baseline e estado do `ffmpeg`) e Phase 2 (`PublishPath`; a marca do plano na etapa 5; base; fixture).
2. Phase 3 (US1): listar os quadros, codificar e publicar → `video`.
3. **PARE e valide**: `quickstart.md` item 1 com o `ffmpeg` real (T043): confira a quantidade de quadros, a taxa, a duração, a resolução e o PSNR antes de acrescentar histórias.

### Entrega Incremental

1. Setup + Foundational → etapa 5 com a marca do plano; portas, erros, configuração e builders prontos.
2. + US1 → o vídeo (MVP) → item 1.
3. + US2 → conferência dos quadros → item 5.
4. + US3 → codificador ausente → item 6.
5. + US4 → qualidade → item 3.
6. + US5 → progresso e resumo → item 1 (progresso).
7. + US6 → destino e interrupção → item 7.
8. + US7 → determinismo → itens 4 e 8.
9. Polish → documentação, dados reais (item 10), desempenho e verificação final.

Cada história agrega valor sem quebrar as anteriores.

---

## Notas

- `[P]` = arquivos diferentes, sem dependência de tarefa incompleta.
- A linha de comando do `ffmpeg` (`research.md` item 6) é o melhor conhecimento
  atual e **não foi executada** (o `ffmpeg` não estava instalado ao planejar): T079
  a confirma. Qualquer mudança nela muda os bytes do vídeo e exige a nota em
  `contracts/video-file.md`; os testes de T077 fixam a lista de argumentos para
  isso não passar despercebido.
- Os parâmetros por nível (`preset`/`crf`) e o número de threads do `x264` são
  constantes **do adapter**; mudar um deles muda os bytes e vai com uma versão nova
  da ferramenta (`research.md` itens 5 e 6, risco R5).
- A **mudança na etapa 5** (Parte B) merece commit próprio, separado do código novo,
  por tocar uma etapa já entregue; a Parte A também.
- Faça commit após cada tarefa ou grupo lógico (só quando o usuário pedir).
- Pare em qualquer checkpoint para validar a história com o `quickstart.md`.
- Evite: tarefas vagas, duas tarefas `[P]` editando o mesmo arquivo, dependências
  que quebrem a independência de teste de uma história.

---

## Notas de implementação (desvios do plano original, registrados ao executar)

- **`ffmpeg` real**: instalado pelo usuário depois do planejamento (9.0.2,
  `libx264` core 165 r3222, Homebrew, macOS/arm64) — a T002 pediu, não instalou
  sozinho. Com ele, `quickstart.md` foi validado **por inteiro** (itens 1 a
  10), e a validação achou dois enganos reais na linha de comando original de
  `research.md` item 6, que só apareceram com o `ffmpeg` de verdade:
  1. **Vazamento de identidade do codificador**: `-x264-params info=0` **não é
     uma opção válida** (nenhuma versão testada do `ffmpeg`/`x264` tem uma
     opção que desligue a mensagem SEI do `x264` — nem `x264 --fullhelp`
     standalone lista uma); e `-movflags +bitexact` **não existe mais** como
     flag do muxer MOV nesta versão. Sem correção, o arquivo carregava
     `TAG:encoder=Lavc<versão> libx264` no contêiner e a string completa
     `x264 - core 165 r3222 … - options: cabac=1 …` dentro do próprio fluxo
     H.264. Corrigido com `-bitexact` (global — é o que de fato tira a
     versão da etiqueta `encoder`), `-metadata:s:v:0 encoder=` (tira a
     etiqueta em si) e `-bsf:v filter_units=remove_types=6` (remove o NAL de
     SEI depois de codificado; nada que esta ferramenta use depende de SEI).
     Confirmado depois: `cmp` idêntico, `ffprobe`/`strings` limpos, mesmo PSNR
     (37,54 dB) antes e depois do filtro.
  2. **Falha silenciosa em quadro corrompido** (R4): por padrão, o `ffmpeg`
     **tolera** um erro de decodificação isolado numa sequência de imagens e
     termina com sucesso (código `0`), quadro descartado ou reaproveitado em
     silêncio — não bastava "o codificador vê" (a suposição original do
     plano). Corrigido com `-xerror`, que faz qualquer erro de decodificação
     ou multiplexação abortar a codificação. Achado com um quadro corrompido
     de propósito (`os.urandom` no meio do `IDAT` — **zerar bytes não basta**:
     um PNG com bytes zerados no meio ainda decodifica, só com pixels
     errados); confirmado depois: código `50`, sem arquivo nem temporário.

  Os testes automatizados (`videoencoder`) usam um `ffmpeg` de mentira e não
  dependiam dessas correções para passar (o script de mentira não verifica
  semântica de codificação real); por isso os dois problemas só apareceram na
  validação manual, não nos testes — um lembrete de que a linha de comando de
  um processo externo, mesmo com testes que fixam os argumentos exatos, só se
  confirma de verdade rodando o programa real.
- **`VideoExporter.Check` na US1**: o método saiu junto com `Export` (T039), e não na
  US6, para o tipo satisfazer a porta antes da ligação em `main.go` (T042); a US6 ficou
  com a ordem no serviço (destino depois dos quadros e antes do codificador), a regra
  `.mp4`, o cancelamento e a interrupção.
- **Ordem de `Verify`**: a resolução vem **antes** do conjunto único (a identificação do
  conjunto já inclui a resolução; do contrário quadros de resoluções diferentes seriam
  sempre "conjuntos misturados"); a ordem final está em `research.md` item 7 e
  `data-model.md`.
- **Mensagens dos sentinelas**: os textos dos sentinelas foram escolhidos para lerem bem
  com o detalhe que vem depois dos dois-pontos (`frame file cannot be used:
  frame_000007.png is not a whole PNG image …`), e os exemplos de `contracts/cli.md` foram
  alinhados às mensagens reais; "1 differs" e "1 frame was …" têm o singular.
- **`FrameRepository.List`**: `Inspect` (etapa 5) e `List` compartilham `scanFrames`; `Inspect`
  continua tratando um diretório inexistente como vazio, e `List` o recusa com `40`.
- **Etapa 5**: a `RenderVersion` passou a `2` (a marca do plano dentro de cada quadro); o
  hash dos pixels de um quadro não mudou, e o `render frame` continua idêntico ao mesmo
  quadro de `render all`.
- **Testes do `videoencoder`**: ~12 s no total (os cenários de fluxo de 1 MiB, de
  cancelamento e o intervalo de 1,1 s do teste de reprodutibilidade); rodam com `-race`.
- **Desempenho real (SC-007)**, mesmo Apple M1 de 8 núcleos das etapas
  anteriores: o passeio real (1020 quadros, 1080×1920, relevo Copernicus real,
  mapa sintético) montou em `medium` em **14 s** e em `high` em **20 s** — bem
  abaixo do limite de 10 min para 1350 quadros na mesma resolução. 1140 quadros
  reais de 720×1280 (`/tmp/qbig`, item 9): `low` 5,04 s, `medium` 7,57 s,
  `high` 11,24 s. A conferência de 1140 quadros recusados (conjunto misturado)
  levou 0,40 s, bem dentro do limite de 2 s de SC-005.
