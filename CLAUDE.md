# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Projeto

Sobrevoo é uma ferramenta de linha de comando pessoal e open source, em Go,
que vai gerar vídeos de sobrevoo a partir de trajetos GPS (no estilo
Relive/Strava). Sete features estão implementadas até agora:
`specs/001-gps-track-processing/` lê um trajeto GPX, trata ele (descarta
pontos inválidos, reordena por tempo), reduz/suaviza o traçado, e imprime um
resumo (comando `inspect`); `specs/002-geo-data-registry/` gerencia o
registro local de mapas base e dados de relevo que o usuário já baixou, e
verifica se um trajeto está coberto por eles (comandos `geodata
register|list|remove|check`); `specs/003-camera-path-planning/` calcula o
plano de câmera do vídeo de sobrevoo — para cada quadro, onde a câmera está,
para onde aponta e onde está o marcador da atividade —, imprime um resumo e
o exporta em JSON (comando `plan`); `specs/004-geo-data-slice/` lê o
conteúdo dos dados registrados e reúne o recorte de mapa base e relevo que um
plano de câmera exportado precisa, com resumo e exportação em ZIP
(`geodata slice <plano.json>`), e consulta a elevação de uma coordenada
(`geodata elevation --lat --lon`); `specs/005-frame-rendering/` desenha os
quadros do voo como imagens PNG — o relevo em perspectiva, vestido com as peças
do mapa base, o traçado até o marcador e o marcador — a partir do plano e do
recorte exportados, com progresso, retomada e resumo (`render frame` e
`render all`); `specs/006-video-assembly/` junta esses quadros num único
vídeo MP4, na ordem e na taxa de quadros do plano, com a qualidade escolhida
por nível nomeado, progresso e resumo (`video`), usando o `ffmpeg` que o usuário
instala; e `specs/007-full-flight-pipeline/` encadeia as seis etapas
anteriores atrás de um único comando (`fly`), do arquivo de trajeto direto ao
vídeo, com os mesmos parâmetros, recusa cedo (destino e codificador antes de
qualquer etapa, cobertura logo após o plano) e, opcionalmente, um diretório
onde guardar e reaproveitar o plano, o recorte e os quadros entre execuções.
Ainda não há sobreposição de texto ou estatísticas, nem áudio.

**A constituição do projeto (`.specify/memory/constitution.md`) é
vinculante.** Ela é curta — leia antes de fazer mudanças estruturais. As
seções abaixo são um resumo de consulta rápida dela, não um substituto.

## Comandos

```sh
make build      # go build -o bin/sobrevoo ./cmd/sobrevoo
make test       # go test ./... -cover
make generate   # go generate ./...  (regenera os mocks; precisa do mockgen, declarado como tool no go.mod)
make lint       # go vet ./...

# Rodar um teste específico (por pacote + nome do teste/subteste):
go test ./internal/domain/... -run Test_TrackPoint_DistanceTo -v
go test ./internal/infra/inbound/cli/... -run 'Test_InspectCommand_Execute/should_map_ErrEmptyFile' -v

# Rodar a CLI direto, sem compilar um binário:
go run ./cmd/sobrevoo inspect path/to/track.gpx --simplification=low --smoothing=high
go run ./cmd/sobrevoo plan path/to/track.gpx --duration 45 --distance high --aspect 9:16 --export plan.json
go run ./cmd/sobrevoo geodata slice plan.json --export slice.zip
go run ./cmd/sobrevoo geodata elevation --lat -23.5505 --lon -46.6333
go run ./cmd/sobrevoo render frame plan.json slice.zip --number 300 --output frame.png
go run ./cmd/sobrevoo render all plan.json slice.zip --output frames/ --resolution 1280x720
go run ./cmd/sobrevoo video plan.json frames/ --output flight.mp4 --quality medium   # precisa do ffmpeg instalado
go run ./cmd/sobrevoo fly path/to/track.gpx --output flight.mp4 --duration 45 --keep intermediarios/   # as seis etapas num só comando

# Gerar os dados de exemplo sintéticos dos quickstarts das etapas 4 e 5:
go run ./test/samples --out specs/005-frame-rendering/amostras
go run ./test/samples --out specs/005-frame-rendering/amostras --raster-over resources/passeio.gpx
```

## Arquitetura

Hexagonal / portas e adapters. O núcleo (`internal/domain` +
`internal/application`) nunca importa nada de `internal/infra` — toda
dependência externa (parsing de GPX, algoritmos de simplificação/suavização,
configuração, terminal) é acessada por uma porta implementada por um
adapter.

- **`internal/domain`** — entidades (`TrackPoint`, `Track`, `Route`,
  `BoundingBox`, `Level`, `DiscardStats`, `GeoDataSource`, `GeoDataSummary`,
  `CoverageReport`, `TrackSummary`, `CleanedTrack`, `TreatedTrack`,
  `PlanParameters`, `CameraTuning`, `CameraPlan`, `CameraFrame`, e os do recorte
  de dados: `SliceTuning`, `SliceRegion(s)`, `DetailLevel`, `Tile`, `TileSet`,
  `ElevationGrid`, `ElevationReading`, `Coordinate`, `GeoSlice`,
  `SliceSummary`, e os do desenho dos quadros: `Scene`, `Resolution`,
  `RenderTuning`, `FrameImage`, `FrameStats`, `FrameSetID`, `FrameDirectory`,
  `RenderSummary`, e os da montagem do vídeo: `FrameMark`, `VideoQuality`,
  `VideoRequest`, `VideoProgress`, `EncodeJob`, `EncoderInfo`, `VideoSummary`,
  e os do comando único: `FlightRequest`, `FlightStage`, `FlightProgress`,
  `FlightSummary`), construtores
  que carregam regra de negócio (`NewGeoDataSource`, `NewTrackSummary`,
  `NewCameraPlan`, que calcula o resumo a partir dos quadros, `NewGeoSlice`,
  que calcula o resumo do recorte e o põe em ordem, `NewCoordinate`), e métodos de
  entidade que carregam o comportamento de cada uma: `TrackPoint.DistanceTo`
  (Haversine, com o "wrap" do antimeridiano), `Route` (`Length`, `Duration`,
  `BoundingBox`, `ElevationGain`, `Coverage`, `ReorderByTime` e os `Discard*`),
  `Track.Clean`, `TreatedTrack.PlanCamera`, `PlanParameters`
  (`MinimumDuration`, `DefaultDuration`), `CameraTuning.FollowDistance`,
  `LocalPlane`, `PlanarRoute` (`HeadingAt`, `OverviewView`, ...), `CameraView`
  (`Pose`, `Blend`) e `Signal` (`Unwrap`, `LimitRate`, `Smooth`,
  `SmoothedSpans`) — ver `specs/003-camera-path-planning/research.md` item 17 —,
  e, na etapa 4, `CameraPlan` (`Validate`, `AreaOfInterest`), `BoundingBox`
  (`Intersects`, `TileRange`, `Regions`, `Extent`), `SliceTuning`
  (`DetailLevel`, `Estimate`, `EnsureFits`, `EnsurePlanFits`, `NewSizeGuard`),
  `SlicePlan` (`TileCount`, `SampleCount`, `Level`) e `SizeGuard` (o que conta
  para o limite de tamanho e qual nível é reportado), `SliceRegions` (`BaseMaps`,
  `TilesFor`), `ElevationGridInfo` (`CellAt`, `Window`) e `ElevationGrid` (`At`,
  `NoValueCount`, `Range`) — ver `specs/004-geo-data-slice/research.md`;
  o que sobra como função livre é matemática sem dono (`clamp`, `quantize`,
  `frameTime`, `normalizeDegrees`). Distância
  de Haversine, ganho de elevação, duração, cálculo de bounding box —
  incluindo o "unwrap" de longitude no antimeridiano —, a limpeza composta
  em `Track.Clean` e o algoritmo de verificação de cobertura,
  `Route.Coverage`), erros sentinela (`ErrEmptyFile`,
  `ErrUnsupportedFormat`, `ErrInsufficientPoints[AfterCleaning]`, e os do
  planejamento de câmera: `ErrInvalidDuration`, `ErrInvalidFrameRate`,
  `ErrDurationTooShort`, `ErrTrackTooShort`, `ErrTrackTooLarge`,
  `ErrPlanDestinationExists`, `ErrPlanDestinationInvalid`; e os do recorte:
  `ErrPlanFileInvalid`, `ErrPlanFormatVersionUnsupported`, `ErrAreaNotCovered`
  — carregado por `AreaNotCoveredError`, que leva o `CoverageReport` —,
  `ErrSliceTooLarge`, `ErrGeoDataContentUnreadable`,
  `ErrElevationUnitUnsupported`, `ErrSliceDestinationExists`,
  `ErrSliceDestinationInvalid`, `ErrElevationNotCovered`,
  `ErrInvalidCoordinate`; e os do desenho: `ErrSliceFileInvalid`,
  `ErrSliceFormatVersionUnsupported`, `ErrSliceDoesNotMatchPlan`,
  `ErrSliceDoesNotCoverPlan`, `ErrTileFormatUnsupported`, `ErrNoElevationData`,
  `ErrFrameOutOfRange`, `ErrInvalidResolution`, `ErrFrameDestinationInvalid`,
  `ErrFrameDestinationExists`, `ErrFrameSetConflict`,
  `ErrRenderInterrupted`; e os da montagem do vídeo: `ErrFrameDirectoryInvalid`,
  `ErrFrameSequenceInvalid`, `ErrFrameResolutionInvalid`,
  `ErrFramesDoNotMatchPlan`, `ErrFramesWithoutPlanID`, `ErrFrameFileInvalid`,
  `ErrEncoderUnavailable`, `ErrVideoDestinationExists`,
  `ErrVideoDestinationInvalid`, `ErrVideoInterrupted`,
  `ErrVideoEncodingFailed`; e o do comando único: `ErrFlightInterrupted`), e
  as portas
  `TrackParser`, `Simplifier`, `Smoother`, `GeoDataInspector`,
  `GeoDataRepository`, `FileChecker`, `CameraPlanExporter`,
  `CameraPlanReader`, `BaseMapReader`, `ElevationReader`, `GeoSliceExporter`,
  `GeoSliceReader`, `TileDecoder`, `FrameRepository`, `FrameExporter`,
  `VideoEncoder`, `VideoExporter`, `Workspace` (o diretório de quadros de
  uma execução do comando único sem `--keep`).
  Qualquer DTO de saída que não seja um
  valor trivial (ex.: `GeoDataSummary`, `CoverageReport`, `TrackSummary`)
  também é um tipo de domínio comum — não um DTO de `internal/application`
  — e qualquer lógica não trivial (construir uma entidade, calcular algo a
  partir de uma coleção) é construtor ou função pura de domínio, nunca um
  helper solto na camada de aplicação; ver "Onde vive a regra de negócio"
  abaixo.
- **`internal/application`** — a *service layer*. Orquestra portas e
  construtores/funções puras do domínio; não conhece Cobra, arquivo, nem
  código de saída, e não decide nenhuma regra de negócio por conta própria
  — só decide qual porta/função de domínio chamar, e em qual ordem. Há um
  serviço por recurso: `TrackService` (`Clean`, `Treat`, `Inspect` — o único
  lugar que sabe transformar um trajeto bruto em limpo ou tratado),
  `GeoDataService` (`Register`, `List`, `Remove`, `CheckCoverage`,
  `ElevationAt`), `CameraPlanService` (`Generate`, `Export`, `Load`) e
  `GeoSliceService` (`Generate`, `Export`, `Load`), `FrameService`
  (`DrawFrame`, `DrawFrames`), `VideoService` (`Assemble`, e as operações que
  `Assemble` já fazia por dentro e passam a existir também sozinhas,
  `CheckEncoder` e `CheckDestination`) e `FlightService` (`Fly`, o comando
  único); `GeoDataService` e `CameraPlanService` dependem de `TrackService`,
  e `FlightService` de `CameraPlanService`, `GeoSliceService`, `FrameService`
  e `VideoService`, em vez de repetir a orquestração que cada um já faz.
- **`internal/infra/outbound/*`** — adapters que implementam as portas do
  domínio: `trackparser` (GPX via `tkrajina/gpxgo`),
  `simplifier` (Douglas-Peucker), `smoother` (Catmull-Rom), `jsonfile`
  (registro de dados geográficos e exportação do plano de câmera em JSON,
  atômica e sem sobrescrita por padrão; e a leitura do plano exportado,
  `jsonfile.NewCameraPlanReader()`), `atomicfile` (a publicação atômica de
  arquivo, compartilhada por `jsonfile`, `zipfile`, `pngfile` e `videofile`;
  `PublishPath` entrega ao codificador o caminho de um temporário),
  `basemapreader`
  (`NewMBTiles()`: níveis e peças de um MBTiles, somente leitura),
  `elevationreader` (`NewGeoTIFF()`: GeoTIFF em Go puro — faixas ou peças, sem
  compressão/Deflate/LZW, predictors 1, 2 e 3 — que lê só o que uma janela
  precisa), `zipfile` (`NewGeoSliceExporter()`: o recorte num ZIP
  determinístico; `NewGeoSliceReader()`: lê e valida o recorte por inteiro),
  `tiledecoder` (`NewRaster()`: PNG/JPEG/WebP → pixels), `pngfile`
  (`NewFrameRepository()` e `NewFrameExporter()`: os quadros como PNG atômicos,
  com a identificação do conjunto e a do plano dentro de cada imagem),
  `videofile` (`NewVideoExporter()`: o vídeo como um arquivo publicado por
  inteiro), `videoencoder` (`NewFFmpeg(binary)`: o processo externo `ffmpeg`, com
  `libx264`, que o usuário instala — a ferramenta não o traz nem o baixa),
  `workingdir` (`NewOS()`: o diretório de quadros temporário de uma execução
  do comando único sem `--keep`, e o `--keep` criado se ainda não existir),
  `config` (limiares internos fixos:
  mínimo de pontos, velocidade máxima plausível, nível padrão, os
  `CameraTuning` do planejamento de câmera e os parâmetros padrão do plano —
  ainda sem fonte de configuração externa, mas o ponto de extensão já
  existe, conforme o Princípio VIII da constituição). O pacote `config`
  tem tipos próprios (`config.Level`, `config.CameraTuning`,
  `config.PlanDefaults`, `config.SliceTuning`, `config.RenderTuning`,
  `config.RenderDefaults`, `config.VideoDefaults`) e **não importa o domínio**; quem os mapeia para os
  tipos de domínio é o composition root (`cmd/sobrevoo/config_mapping.go`).
- **`internal/infra/inbound/cli`** — o(s) comando(s) Cobra, e o lugar que
  traduz erros sentinela do domínio em códigos de saída de processo
  (`exit_code.go`); ver `specs/001-gps-track-processing/contracts/cli.md` e
  `specs/002-geo-data-registry/contracts/cli.md` e
  `specs/003-camera-path-planning/contracts/cli.md` e
  `specs/004-geo-data-slice/contracts/cli.md` e
  `specs/005-frame-rendering/contracts/cli.md` e
  `specs/006-video-assembly/contracts/cli.md` e
  `specs/007-full-flight-pipeline/contracts/cli.md` para o mapeamento exato.
  Na etapa 1, era também o único lugar que tocava o filesystem (`os.Open`,
  para obter o `io.Reader` que `TrackParser` espera). A partir da etapa 2
  isso não é mais universal: adapters de saída que precisam de acesso
  posicional a um arquivo — `geodatainspector` (lê SQLite/TIFF por
  caminho), `jsonfile` (lê/escreve o registro) e `filechecker`
  (`os.Stat`) — abrem o arquivo eles mesmos, dado apenas o caminho; a CLI
  continua sendo quem abre o arquivo só quando o método do serviço exige um
  `io.Reader` (`register` não abre nada, pois passa um caminho;
  `check` abre, pois `GeoDataService.CheckCoverage` exige um `Reader`,
  igual a `inspect` e a `plan`, cujo `CameraPlanService.Generate` também
  recebe um `Reader`; já o arquivo do plano exportado é escrito pelo
  adapter `jsonfile`, dado só o caminho). Ambos os padrões respeitam os Princípios I e II da
  constituição — é só uma questão de qual adapter concreto faz a chamada de
  I/O real (`specs/002-geo-data-registry/research.md`, item 8).
- **`cmd/sobrevoo/main.go`** — composition root; o único lugar que conecta
  todos os adapters concretos entre si.

### O desenho dos quadros (etapa 5)

O renderizador é código de domínio (`internal/domain/frame_*.go`), em Go puro,
por lançamento de raios sobre as grades de elevação do recorte
(`specs/005-frame-rendering/research.md`). Para o desenho ser **idêntico byte
a byte** em qualquer processador e por qualquer número de goroutines: todo
produto que entra numa soma é envolvido em `float64(...)` (nenhuma fusão
multiplicação-soma); só `+ − × ÷`, `Sqrt`, `Floor`, `Abs`, `Min`, `Max`, `Sin`,
`Cos`, `Tan`, `Atan2`, `Log`, `Log2` e `Ldexp` — nunca `Exp`, `Pow` nem `Sinh`
(têm assembly por arquitetura); nunca iterar `map` para produzir valor; cada
pixel só depende de si mesmo. O teste de `frame_scene_test.go` compara o hash
dos pixels de um quadro com uma referência (igual em arm64 e amd64): se ele
falhar em outra máquina, corrija a aritmética, não a constante — a constante só
muda junto com `domain.RenderVersion`. Cada quadro carrega, dentro da imagem,
a identificação do conjunto e a do plano de que veio (`FrameMark`): a
`RenderVersion` 2 é a que grava a do plano, e quadros de versão anterior são
de outro conjunto.

### A montagem do vídeo (etapa 6)

`sobrevoo video <plano> <quadros/> --output voo.mp4` junta os quadros num MP4
(`specs/006-video-assembly/`). O `ffmpeg` roda como **processo externo**, atrás
da porta `VideoEncoder` (`videoencoder.FFmpeg`): lê os quadros pelo nome
(`frame_%06d.png`), com o diretório dos quadros como diretório de trabalho, e
escreve num arquivo temporário que o `videofile.VideoExporter` publica por
inteiro (`atomicfile.PublishPath`). Antes de codificar, `FrameDirectory.Verify`
(regra de domínio) confere, nesta ordem, se há quadros da ferramenta, se
trazem a identificação do plano informado, se têm a mesma resolução (par), se
são de um só conjunto, se a numeração é exatamente 0 a N−1 e se são PNG inteiros;
depois vêm o destino e o codificador (`VideoExporter.Check`, `Probe`). Para o
**mesmo `ffmpeg`** o arquivo é idêntico byte a byte (confirmado com o `ffmpeg`
real): a linha de comando fixa o que variaria com o ambiente (`-bitexact`
global, `-metadata:s:v:0 encoder=`, `x264` com `threads=4`, cor BT.709
explícita), remove do fluxo a mensagem SEI do `x264` com a versão e as opções
de codificação, que nenhuma opção desliga (`-bsf:v
filter_units=remove_types=6`), e faz um quadro que não decodifica falhar em
vez de ser tolerado em silêncio (`-xerror`) — e os testes de `videoencoder` a
fixam por inteiro: mudá-la muda os bytes de todos os vídeos e pede uma nota de
versão em `specs/006-video-assembly/contracts/video-file.md`. Os testes do
adapter usam um `ffmpeg` de mentira (script de shell); o `ffmpeg` real só entra
na validação manual (`specs/006-video-assembly/quickstart.md`).

### O comando único (etapa 7)

`sobrevoo fly <trajeto> --output voo.mp4` encadeia as seis etapas anteriores
(`specs/007-full-flight-pipeline/`) atrás de `FlightService.Fly`, que só
**orquestra** `CameraPlanService`, `GeoSliceService`, `FrameService` e
`VideoService` — nenhuma regra de negócio nova nasce nele; o resultado é
garantido idêntico ao de rodar os seis comandos na mão porque é, literalmente,
a mesma chamada de serviço. O codificador e o destino do vídeo (`VideoService
.CheckEncoder`/`CheckDestination`, extraídos de dentro de `Assemble`) são
conferidos antes de qualquer etapa; a cobertura dos dados registrados já é a
primeira coisa que `GeoSliceService.Generate` confere, então chamá-lo logo
após o plano já recusa cedo, sem checagem nova. Sem `--keep`, o plano e o
recorte **nunca tocam disco** — os serviços recebem os valores diretamente,
em memória —, e só os quadros precisam de um diretório real (a porta nova
`Workspace`, temporário e sempre removido ao final). Com `--keep
<diretório>`, três arquivos previsíveis (`plan.json`, `slice.zip`, `frames/`,
`specs/007-full-flight-pipeline/contracts/intermediates-directory.md`) são
guardados e, numa execução seguinte, reaproveitados sempre que ainda valem
para o trajeto e os valores informados — pelas identidades que o domínio já
tinha (`CameraPlan.ID()`, `GeoSlice.EnsureMatches`) e pela mesma regra de
conjunto que o desenho dos quadros já usa (`FrameService.DrawFrames`, chamado
sem nenhuma lógica própria de reaproveitamento). Dois detalhes só a validação
manual revelou: o `--keep` de uma execução nova precisa ser criado
(`Workspace.EnsureDirectory`, como `render all --output` já faz com o
próprio); e um recorte recém-gerado não tem `GeoSlice.ContentID` (só a
leitura de um recorte já gravado o preenche) — do qual a identidade dos
quadros depende —, então `FlightService` **relê** um recorte recém-exportado
antes de desenhar, para os quadros carregarem a mesma identidade que uma
execução futura, reaproveitando o recorte pelo arquivo, vai calcular; sem
isso, os quadros da primeira execução nunca bateriam com os de uma segunda.
Interrupção (`Ctrl+C`/`SIGTERM`) sempre sai com o código próprio do comando
único (`ErrFlightInterrupted`), nunca o de uma etapa — a única exceção
deliberada à regra geral de "mesmo erro que o comando individual".

### Portas, service layer e regra de negócio (Princípios I, II e IX da constituição)

Estas convenções são regra permanente do projeto — Princípio IX da
constituição — não algo específico da etapa 2; aplique-as desde o primeiro
rascunho de qualquer feature nova, sem esperar por um pedido de ajuste. A
referência de estilo é `waliqueiroz/mystery-gifter-api`
(`internal/domain`, `internal/application`); `specs/002-geo-data-registry/research.md`
(itens 10.1, 13, 14, 16) registra o histórico de como o projeto chegou até
aqui, inclusive um engano real (uma interface de método único por caso de
uso) que a redação anterior da constituição permitia.

- Nada de `ports.go`/`interfaces.go`. Uma porta ligada a uma única entidade
  fica no arquivo dessa entidade (`TrackParser` é declarada em `track.go`,
  já que produz `Track`). Uma porta sem entidade dona ganha seu próprio
  arquivo, nomeado pelo conceito que representa (`Simplifier` em
  `simplification.go`, `Smoother` em `smoothing.go`). Toda porta é nomeada
  pelo papel arquitetural que exerce, não pelo dado que manipula: uma porta
  de persistência é `XRepository` (nunca `XRegistry`, `XStore`, ou
  similar), com o campo correspondente na struct do serviço seguindo o
  mesmo nome (`xRepository domain.XRepository`) — ex.: `GeoDataRepository`.
- Adapters de saída (`internal/infra/outbound`): se o pacote é uma
  **tecnologia** que pode servir várias portas (`jsonfile`, como `postgres`
  no `mystery-gifter-api`), o construtor é nomeado pela **porta**
  (`jsonfile.NewGeoDataRepository(path)`, arquivo `geo_data_repository.go`).
  Se o pacote agrupa **estratégias/algoritmos** de uma única porta, o pacote
  leva o nome da porta (`simplifier`, `smoother`, `trackparser`,
  `filechecker`) e o tipo/construtor nomeiam só a estratégia, sem repetir o
  pacote (`simplifier.NewDouglasPeucker()`, `smoother.NewCatmullRom()`,
  `trackparser.NewGPX()`, `filechecker.NewOS()`); o arquivo leva o nome
  completo (`douglas_peucker_simplifier.go`). Pacote com uma implementação
  só e sem estratégia distinguível usa `New()` (`geodatainspector.New()` →
  `Inspector`). Estilo Go: nome exportado nunca repete o nome do pacote
  (nada de `geodatainspector.GeoDataInspector`). Nada de subpacote por
  algoritmo só para chamar `algoritmo.New()`. Diferente do
  `mystery-gifter-api`, o construtor pode devolver o struct concreto do
  adapter (não precisa devolver a interface do domínio).
- Num arquivo de domínio que declara uma porta, a ordem é: `package`,
  diretiva `//go:generate` (logo após o `package`), imports, **a interface
  logo no início**, e só depois a entidade, os enums e os construtores —
  nunca a interface no meio ou no fim do arquivo (como em `user.go`/
  `group.go` do `mystery-gifter-api`; ver `track.go` e `geo_data_source.go`).
- `XService` (interface exportada) / `xService` (struct não exportada) /
  `NewXService(...)` (construtor) é **um serviço por recurso/agregado, não
  um serviço por caso de uso**: `X` nomeia o que o serviço gerencia (ex.:
  `GeoDataService`), e cada caso de uso vira um método nomeado pela
  operação (`Register`, `List`, `Remove`, `CheckCoverage` — nunca um
  `Execute` genérico, nunca uma interface por método). Vários casos de uso
  que operam sobre o mesmo recurso pertencem à mesma interface e à mesma
  struct — padrão espelhado de `waliqueiroz/mystery-gifter-api`
  (`internal/application/group_service.go`: `GroupService` reúne
  `Create`/`GetByID`/`Search`/`AddUser`/`RemoveUser`/`GenerateMatches`/
  `Reopen`/`Archive`/`GetUserMatch`). Um serviço pode depender de outro
  serviço de aplicação (não só de portas do domínio) quando isso faz
  sentido — ver `GroupInviteService` dependendo de `UserService` no mesmo
  repositório de referência. Adapters de entrada dependem só da interface,
  nunca da struct concreta.
- **Onde vive a regra de negócio**: DTOs de saída não triviais e qualquer
  lógica que não seja "chamar uma porta na ordem certa" vivem em
  `internal/domain`, nunca em `internal/application` — nem como DTO
  próprio da camada de aplicação, nem como função solta no pacote
  `application`. Um método de serviço busca/checa via porta, delega a
  regra para um construtor (`domain.NewGeoDataSource`) ou método de uma
  entidade de domínio (`Route.Coverage`), e devolve o resultado — a mesma
  divisão de `GroupService.AddUser` (busca via repositório, delega para
  `domain.Group.AddUser`) em `waliqueiroz/mystery-gifter-api`. **Dentro do
  domínio, o comportamento vai para a entidade que o possui**: se existe um
  tipo dono do dado (`TrackPoint`, `Route`, `Track`, `PlanParameters`,
  `CameraView`, `Signal`, ...), a lógica é método dele (`route.Length()`,
  `point.DistanceTo(other)`, `track.Clean(...)`); se uma lista de pontos ou de
  números virou o argumento de uma função, falta um tipo — dê a ele um nome e
  mova a função para lá (foi o que `Route`, `PlanarRoute` e `Signal` fizeram).
  Função livre só para matemática sem dono e sem estado (`clamp`, `quantize`)
  ou para um construtor (`NewXxx`). Foi um ajuste pedido em revisão de PR:
  não repita o desvio de escrever funções `Compute...`/`Build...` soltas.
- Uma chamada direta a uma função pura da biblioteca padrão do Go (ex.:
  `time.Now()`) não é dependência externa (Princípio II) e não precisa de
  porta nem de abstração — não reintroduza algo como um `Clock` só para
  poder mockar isso; só exige porta o que de fato faz I/O real ou depende
  de estado fora do processo.
- Mocks são gerados com `go.uber.org/mock/mockgen` via diretiva
  `//go:generate` posicionada diretamente acima da interface que ela
  mocka — nunca em um arquivo central. A saída vai para um subpacote
  irmão `mock<pacote>`, sem underscore
  (`internal/domain/mockdomain`, `internal/application/mockapplication`), um
  arquivo gerado por interface, com `-package mockdomain` (etc.) explícito na
  diretiva. O nome carrega a camada (nada de `mocks`/`builders` genérico)
  para um teste poder importar mocks de duas camadas sem alias. Rode `make generate` depois de adicionar ou alterar uma
  porta/interface.

### Testes (Princípio X da constituição)

- given/when/then: todo teste é
  `t.Run("should ...", func(t *testing.T) { // given ... // when ... // then ... })`.
- Nada de testes tabulares (`[]struct{...}` + `for`) — um cenário, um
  `t.Run`, mesmo que isso repita configuração.
- Test data builders vivem em subpacotes `build<pacote>`
  (hoje só `internal/domain/builddomain` existe; `buildapplication` só
  seria criado se um serviço passasse a precisar de builder próprio):
  `NewXBuilder()` com defaults sensatos, `WithCampo(...)`/`WithoutCampo()`
  fluentes, `Build()` terminal. Use um sempre que um literal de struct
  repetido ou grande demais deixaria o teste poluído.
- Cada camada é testada isolada, com o que ela depende mockado: os testes
  de domain/application mockam as portas do domínio (`mockdomain`); os
  testes de `internal/infra/inbound/cli` mockam
  `application.TrackService`, `application.GeoDataService`,
  `application.CameraPlanService`, `application.GeoSliceService`,
  `application.FrameService`, `application.VideoService` e
  `application.FlightService` (`mockapplication`) e nunca conectam
  um serviço ou adapter de saída real. Não existe teste automatizado de
  ponta a ponta — `specs/<feature>/quickstart.md` é o checklist manual, com
  o binário real, pra isso.

### Receivers (estilo Go)

O nome do receiver é uma abreviação curta do **tipo**, igual em todos os
métodos dele: `r` para `GeoDataRepository`, `d` para `DiscardStats`, `b`
para `BoundingBox` e para os `*Builder`, `s` para os `*Service`
(`geoDataService`, `inspectTrackService`), `p` para `TrackPoint`. Nunca a
inicial de outra palavra ou de um nome antigo do tipo (o `s` de `Store`
sobrou em `GeoDataRepository` depois do rename). Receiver sem uso fica sem
nome (`func (OS) Exists`).

### Detalhe do Go 1.26

Este módulo usa Go 1.26, que estendeu o builtin `new` para aceitar uma
expressão, não só um tipo: `new(x)` devolve um `*T` apontando pra uma cópia
de `x`. O código usa isso diretamente (ex.: `Elevation: new(50.0)` em
testes, ou `Time: new(someTime)`) em vez de um pacote helper do tipo
`ptr.Of[T]` escrito à mão — não reintroduza um.

### Fluxo do Spec Kit

O desenvolvimento de features passa por `specs/<NNN-feature-name>/`
(`spec.md`, `plan.md`, `tasks.md`, `research.md`, `data-model.md`,
`contracts/`, `quickstart.md`), conduzido pelos slash commands `/speckit-*`
(`/speckit-specify`, `/speckit-clarify`, `/speckit-plan`, `/speckit-tasks`,
`/speckit-implement`, ...). Todos os artefatos do Spec Kit e toda a
comunicação durante esse fluxo DEVEM ser em português do Brasil
(constituição: "Stack Tecnológica e Idioma dos Artefatos") — código-fonte,
identificadores, nomes de pacote/arquivo, nomes de branch, mensagens de
commit e comentários de código permanecem em inglês; termos técnicos já
consagrados não são traduzidos em nenhuma direção.
