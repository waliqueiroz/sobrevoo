# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Projeto

Sobrevoo é uma ferramenta de linha de comando pessoal e open source, em Go,
que vai gerar vídeos de sobrevoo a partir de trajetos GPS (no estilo
Relive/Strava). Treze features estão implementadas até agora:
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
onde guardar e reaproveitar o plano, o recorte e os quadros entre execuções;
`specs/008-frame-appearance/` torna ajustável, por flag, a cor e a
espessura do traçado, a cor e o raio do marcador, e a cor do fundo — os
cinco valores que antes eram fixos no código —, nos mesmos comandos que já
desenham quadros (`render frame`, `render all` e `fly`), com os valores de
sempre como padrão e passando a fazer parte da identidade do conjunto de
quadros; e `specs/009-frame-overlays/` desenha, fixos na tela (não colados
no terreno), a distância percorrida, a elevação do trajeto e o ganho
acumulado no ponto do marcador, o tempo decorrido da atividade e um perfil
de elevação do trajeto inteiro com um marcador que avança com o voo —
ligados por padrão, desligáveis por inteiro ou por bloco
(`--overlays`/`--overlay-blocks`, nos mesmos três comandos), com uma fonte
embutida na ferramenta e sem nenhuma leitura nova do trajeto GPS ou dos
dados geográficos registrados: o plano de câmera passou a guardar, por
quadro, o instante real da atividade e a elevação do trajeto no ponto do
marcador (`format_version` 2; um plano da versão 1 é recusado, pedindo para
ser gerado de novo); e `specs/010-geo-data-source-control/` acrescenta,
sobre o registro de dados geográficos, um comando que limpa o registro
inteiro de uma vez, só com confirmação explícita (`geodata clear
--confirm`), e a possibilidade de escolher explicitamente, pelo nome já
usado ao registrar, qual fonte de mapa base e/ou de elevação usar
(`--base-map`/`--elevation`, em `geodata check`, `geodata slice` e `fly`),
em vez de depender sempre da seleção automática por área; a escolha
explícita nunca mistura fontes nem completa em silêncio uma cobertura
incompleta, e passa a fazer parte da decisão de reaproveitar um recorte
guardado por `fly --keep`; e `specs/011-overlay-polish/` corrige o
acabamento das sobreposições de tela que a nona etapa introduziu, sem
acrescentar nem mudar nenhum valor exibido, bloco ou configuração: o texto
passa de uma fonte bitmap ampliada por fator inteiro para uma fonte
vetorial embutida, rasterizada com suavização por um rasterizador próprio;
os três painéis numéricos passam a compartilhar a largura do mais largo; o
marcador do perfil de elevação ganha um raio proporcional à altura do
quadro, com piso em pixels, como o marcador do mapa já tem; o texto ganha
um contorno escuro fixo, legível sobre qualquer fundo; e a margem de
segurança passa a ser maior na borda inferior que nas demais, adequada ao
vídeo vertical — o que muda os pixels de quadros já desenhados, então a
versão do desenho sobe (`RenderVersion` 3); `specs/012-overlay-ptbr-readability/`
traduz para português do Brasil todo rótulo que a sobreposição desenha
(mantendo as abreviações de unidade, "km"/"m", como já estão) e corrige
três defeitos de legibilidade do texto introduzidos pela etapa anterior —
o contorno escuro, grosso demais, passa a ser fino e proporcional ao
tamanho em que a letra está sendo desenhada, a fonte embutida passa a ter
peso mais forte (ainda vetorial, embutida no binário, sem dependência
nova) e a largura dos painéis numéricos deixa de pulsar quadro a quadro —,
o que muda os pixels de novo e sobe a versão do desenho outra vez
(`RenderVersion` 4); e `specs/013-treatment-level-flags/` leva os níveis
de simplificação e de suavização do trajeto — até então só escolhíveis no
`inspect` — para `plan` e `fly`, com os mesmos nomes de opção, os mesmos
valores aceitos e os mesmos padrões vindos da configuração, passando a
fazer parte da identidade do plano de câmera (`CameraPlan.ID()`) e, por
isso, da decisão de reaproveitar um plano, um recorte ou quadros guardados
por `fly --keep` entre execuções; e `specs/014-speed-overlay-block/`
acrescenta às sobreposições de tela um quinto bloco, a velocidade da
atividade (`speed`, rótulo "VEL"), calculado por quadro como a velocidade
média numa janela de tempo fixa de 30 segundos em torno do instante real da
atividade — nunca a velocidade instantânea entre dois pontos consecutivos
do trajeto, que oscilaria demais para ser lida —, encurtada nos extremos
do trajeto em vez de ausente ou descontínua; ao contrário dos quatro blocos
de antes, nasce fora da escolha padrão de blocos, só aparece quando pedido
por nome pelo mesmo mecanismo (`--overlay-blocks`), e só existe quando o
trajeto tem horário em todos os pontos, exatamente como o bloco de tempo
decorrido já exige. A velocidade passa a fazer parte do conteúdo do plano
de câmera, do arquivo de plano exportado e da identidade do plano
(`CameraPlan.ID()`); como o arquivo ganha um campo por quadro que nenhuma
versão anterior escrevia, `format_version` sobe de 2 para 3, e um plano
mais antigo é recusado com a mesma mensagem e o mesmo código de saída que
uma versão desconhecida já recebe, pedindo para ser gerado de novo.
`specs/015-overlay-redesign/` redesenha as sobreposições de tela que a
etapa 9 introduziu e as etapas 11/12 poliram: os blocos numéricos deixam
de ser uma pilha de faixas escuras no alto da tela e passam a ser texto
puro, sem nenhum painel de fundo, lado a lado numa única faixa horizontal,
repartindo a largura útil em colunas de mesma largura, numa ordem fixa e
documentada (velocidade, elevação, distância, ganho, tempo decorrido) que
não depende da ordem pedida em `--overlay-blocks`; dentro de cada bloco, o
rótulo por extenso (não mais abreviado em caixa alta), o valor num corpo
bem maior, e a unidade — as três alturas que já eram "rótulo e valor na
mesma linha" viram três linhas, com a terceira omitida sem deixar vão para
um bloco sem unidade, como o tempo decorrido. O ganho de elevação
acumulado deixa de ser um segundo número colado ao bloco de elevação e
vira um bloco próprio (`gain`), ligado pelo mesmo mecanismo dos demais; a
escolha padrão de blocos muda de `distance,elevation,time,profile` para
`distance,elevation,speed,profile` — tempo decorrido e ganho continuam
disponíveis, só saem do padrão. O gráfico de elevação no rodapé também
perde o painel e passa a se destacar do terreno pela mesma técnica de
contorno escuro que o texto já usa. Nenhum valor passa a ser calculado,
formatado ou arredondado de outro jeito — só a apresentação muda —, e como
os pixels de um quadro com sobreposições ligadas mudam de verdade,
`RenderVersion` sobe de 4 para 5. Ainda não há áudio.

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
go run ./cmd/sobrevoo geodata slice plan.json --export slice.zip --base-map ruas --elevation srtm-sp
go run ./cmd/sobrevoo geodata elevation --lat -23.5505 --lon -46.6333
go run ./cmd/sobrevoo geodata clear --confirm   # remove todas as entradas do registro, nunca os arquivos
go run ./cmd/sobrevoo render frame plan.json slice.zip --number 300 --output frame.png --trail-color "#00FF00" --marker-radius 0.03 --overlay-blocks distance,time
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
  `RenderTuning`, `Appearance` (a cor/espessura do traçado, a cor/raio do
  marcador e a cor do fundo — etapa 8; `NoMapColors`/`NoElevationColors`/
  `TrailCasingColor`/`MarkerRingColor`, em `render_tuning.go`, continuam
  fixos, fora de `Appearance`, por serem o significado da imagem, não
  estilo), `FrameImage`, `FrameStats`, `FrameSetID`,
  `FrameDirectory`, `RenderSummary`, e os da montagem do vídeo: `FrameMark`, `VideoQuality`,
  `VideoRequest`, `VideoProgress`, `EncodeJob`, `EncoderInfo`, `VideoSummary`,
  e os do comando único: `FlightRequest`, `FlightStage`, `FlightProgress`,
  `FlightSummary`, e os da sobreposição de tela (etapa 9): `OverlayConfig`,
  `OverlayBlock` e `ElevationProfile` — o perfil de elevação pré-computado
  de um trajeto, consultado por distância (`ElevationProfile.At`), a mesma
  técnica de busca por bracket que `PlanarRoute.PointAt` já usa; `CameraFrame`
  ganhou `ActivityElapsed`, `TrackElevation` e `TrackElevationGain`, e
  `CameraPlan`/`PlanSummary` ganharam `ElevationAvailable`; e, na etapa 10,
  `SourceSelection` — a escolha explícita, por nome, de fonte de mapa base
  e/ou de elevação, com o método `Resolve(baseMaps, elevations)` que
  restringe as listas de candidatos a um só elemento por tipo antes de
  entregá-las ao mesmo `Route.Coverage`/`SelectSource` de sempre; e
  `FlightRequest` ganhou `Selection`), construtores
  que carregam regra de negócio (`NewGeoDataSource`, `NewTrackSummary`,
  `NewCameraPlan`, que calcula o resumo a partir dos quadros, `NewGeoSlice`,
  que calcula o resumo do recorte e o põe em ordem, `NewCoordinate`,
  `NewOverlayConfig`), e métodos de
  entidade que carregam o comportamento de cada uma: `TrackPoint.DistanceTo`
  (Haversine, com o "wrap" do antimeridiano), `Route` (`Length`, `Duration`,
  `BoundingBox`, `ElevationGain`, `Coverage`, `ReorderByTime`, os `Discard*`,
  `TimeAt` e `ElevationProfile` — etapa 9),
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
  `Sources`, `TilesFor`), `ElevationGridInfo` (`CellAt`, `Window`) e `ElevationGrid` (`At`,
  `NoValueCount`, `Range`) — ver `specs/004-geo-data-slice/research.md`; e, na
  etapa 10, `GeoSlice.EnsureUsesSources`, que recusa um recorte cuja
  procedência (`Summary.Sources`, já existente desde a etapa 4) não é
  exatamente o conjunto de fontes dado — comparação usada só por
  `FlightService.reuseSlice` para decidir reaproveitar ou não um recorte
  guardado por `--keep`;
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
  `ErrVideoEncodingFailed`; o do comando único: `ErrFlightInterrupted`; e os
  da aparência: `ErrInvalidColor`, `ErrInvalidTrailWidth`,
  `ErrInvalidMarkerRadius`; e o da sobreposição de tela:
  `ErrInvalidOverlayBlock`; e os do controle do registro (etapa 10):
  `ErrDataSourceTypeMismatch` (um nome pedido explicitamente existe, mas é
  do outro tipo), `ErrRegistryClearNotConfirmed` (`geodata clear` sem
  `--confirm`) e `ErrSliceUsesDifferentSource` (a comparação de
  `GeoSlice.EnsureUsesSources`, nunca devolvido a um usuário)), e
  as portas
  `TrackParser`, `Simplifier`, `Smoother`, `GeoDataInspector`,
  `GeoDataRepository` (ganhou `Clear`, etapa 10), `FileChecker`, `CameraPlanExporter`,
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
  `GeoDataService` (`Register`, `List`, `Remove`, `Clear` — etapa 10,
  limpa o registro inteiro, sem confirmação recusa citando a contagem —,
  `CheckCoverage`, `ElevationAt`; `CheckCoverage` ganhou um parâmetro
  `SourceSelection`, resolvido antes de tratar o trajeto), `CameraPlanService`
  (`Generate`, `Export`, `Load`) e
  `GeoSliceService` (`Generate` — ganhou o mesmo parâmetro `SourceSelection`
  —, `Sources`, `Export`, `Load`), `FrameService`
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
  `config.RenderDefaults` — resolução e, desde a etapa 8, também a aparência
  padrão (cor/espessura do traçado, cor/raio do marcador, cor do fundo, como
  texto hexadecimal) e, desde a etapa 9, a configuração de sobreposição
  padrão (`OverlaysEnabled`, `OverlayBlocks`) —, `config.VideoDefaults`) e
  **não importa o domínio**; quem os mapeia para os
  tipos de domínio é o composition root (`cmd/sobrevoo/config_mapping.go`,
  função `domainAppearance` para a aparência e `domainOverlayConfig` para a
  sobreposição).
- **`internal/infra/inbound/cli`** — o(s) comando(s) Cobra, e o lugar que
  traduz erros sentinela do domínio em códigos de saída de processo
  (`exit_code.go`); ver `specs/001-gps-track-processing/contracts/cli.md` e
  `specs/002-geo-data-registry/contracts/cli.md` e
  `specs/003-camera-path-planning/contracts/cli.md` e
  `specs/004-geo-data-slice/contracts/cli.md` e
  `specs/005-frame-rendering/contracts/cli.md` e
  `specs/006-video-assembly/contracts/cli.md` e
  `specs/007-full-flight-pipeline/contracts/cli.md` e
  `specs/008-frame-appearance/contracts/appearance-flags.md` e
  `specs/009-frame-overlays/contracts/overlay-flags.md` e
  `specs/010-geo-data-source-control/contracts/registry-clear.md` e
  `specs/010-geo-data-source-control/contracts/source-selection-flags.md`
  para o mapeamento exato.
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

### A aparência dos quadros (etapa 8)

`--trail-color`, `--trail-width`, `--marker-color`, `--marker-radius` e
`--background-color` (`specs/008-frame-appearance/`) existem, com o mesmo
nome e o mesmo efeito, em `render frame`, `render all` e `fly` — um único
parser compartilhado, `parseAppearance` (`internal/infra/inbound/cli/
appearance.go`), garante isso por construção. As cores são `#RRGGBB`
(`domain.ParseColor`, erro `ErrInvalidColor`); a espessura do traçado e o
raio do marcador continuam sendo uma **proporção da altura do quadro** —
como o desenho já calculava antes, só que agora ajustável —, validada contra
um intervalo (`domain.NewAppearance`, `ErrInvalidTrailWidth`/
`ErrInvalidMarkerRadius`); o piso mínimo em pixels que evita o traçado/
marcador sumirem numa resolução pequena (`TrailMinWidth`, `MarkerMinRadius`)
não muda. `domain.Appearance` **não** é um campo de `RenderTuning` (que
continua resolvido uma vez, por processo, pela configuração): é um valor por
chamada, como `Resolution` já é — entra em `SingleFrameRequest`,
`FrameSetRequest` e `FlightRequest`, e percorre `Scene`/`overlay`/`imagery`/
`NewFrameImage` até o pixel. A aparência escolhida participa da identidade
do conjunto de quadros: `NewFrameSetID` inclui `Appearance.Fingerprint()` no
hash, ao lado do de `RenderTuning` — sem precisar de um `RenderVersion`
novo, porque estender o hash já garante, sozinho, que uma aparência
diferente (inclusive a de antes desta etapa, que nunca escreveu esse
segmento) nunca bate com a de agora. Por isso o reaproveitamento de `render
all` retomado e de `fly --keep` não precisou de nenhuma lógica nova: o
mecanismo que já decide reaproveitar-ou-redesenhar por `FrameSetID` passou a
enxergar aparência diferente de graça. O hachurado de "sem mapa" e o xadrez
de "sem elevação" (`NoMapColors`, `NoElevationColors`) não são ajustáveis —
são o significado da imagem, não estilo — e continuam fixos em
`render_tuning.go`, junto com `TrailCasingColor`/`MarkerRingColor` (a casca
do traçado e o anel do marcador, que também não são ajustáveis). Sem nenhuma
flag informada, o resultado é pixel a pixel igual ao de antes desta etapa —
os cinco valores de hoje viraram os padrões de `config.RenderDefaults`, no
lugar de `var` fixas do domínio.

### As sobreposições de tela (etapa 9)

`--overlays` e `--overlay-blocks` (`specs/009-frame-overlays/`) existem, com
o mesmo nome e o mesmo efeito, em `render frame`, `render all` e `fly` — o
mesmo `parseOverlay` compartilhado (`internal/infra/inbound/cli/overlay.go`)
que `appearance.go` já estabeleceu para a aparência garante isso por
construção. Os quatro blocos (`OverlayBlockDistance`, `OverlayBlockElevation`,
`OverlayBlockTime`, `OverlayBlockProfile`) vêm ligados por padrão; desligar
`--overlays` desliga todos, mesmo com `--overlay-blocks` também informado
(`domain.NewOverlayConfig`, `ErrInvalidOverlayBlock` para um nome
desconhecido). O desenho em si é código de domínio puro
(`internal/domain/frame_screen_overlay.go`, tipo não exportado
`screenOverlay` — não confundir com `overlay`, de `frame_overlay.go`, que
desenha o traçado e o marcador colados no terreno): `Scene.Render` o chama
por último, depois do traçado e do marcador, sempre por cima, sem nenhum
teste de profundidade. O texto usa uma fonte embutida para não depender de
nenhuma fonte do sistema (FR-008) — nesta etapa, uma fonte bitmap
(`golang.org/x/image/font/inconsolata`), cada glifo ampliado por replicação
inteira de pixel; a etapa 11 a substitui por uma fonte vetorial suavizada,
sem mudar o requisito (ver abaixo). A mistura alfa reaproveita a mesma
fórmula que `overlay.blend` já usa — mantendo o determinismo byte a byte
entre arquiteturas que a etapa 5 exige. Cada bloco desenha sobre uma placa
semitransparente de cor fixa (`OverlayPanelColor`/`OverlayPanelOpacity`, em
`render_tuning.go`) para continuar legível sobre qualquer fundo, sem nunca
amostrar o pixel por baixo — a margem de segurança que a protege das quatro
bordas do quadro também muda na etapa 11.

Os valores exibidos vêm exclusivamente do plano de câmera e do recorte —
nenhuma leitura nova do trajeto GPS nem dos dados geográficos registrados
(FR-005): o plano passou a guardar, por quadro, `ActivityElapsed` (o instante
real da atividade), `TrackElevation` (a elevação bruta do trajeto no ponto do
marcador) e `TrackElevationGain` (o ganho acumulado até ali). O ganho
acumulado é a parte delicada (`research.md` item 7): calculá-lo por uma soma
corrente entre quadros consecutivos subestimaria o total verdadeiro sempre
que um trecho de sobe-desce coubesse inteiro entre dois quadros vizinhos —
por isso `TreatedTrack.PlanCamera` pré-computa o ganho acumulado **por ponto
do trajeto tratado** (`Route.ElevationProfile`, guardando cada soma parcial,
não só o total) e só então interpola por distância
(`ElevationProfile.At`), garantindo que o último quadro bata, bit a bit, com
`Route.ElevationGain()` — o mesmo valor que `inspect` relata (FR-014). A
interpolação usa a forma segura `(1-t)*a + t*b`, nunca `a+t*(b-a)`, porque
só a primeira garante um resultado exato em `t=1` em ponto flutuante.
`CameraPlan.ID()` não inclui os três campos novos nem `ElevationAvailable`:
são funções determinísticas do que já entra no hash, incluí-los só forçaria
uma migração de identidade sem nenhum ganho de correção. O formato do plano
exportado sobe de `format_version` 1 para 2 — um plano da versão 1 é
recusado com `ErrPlanFormatVersionUnsupported` e uma mensagem que orienta a
gerar o plano de novo (FR-007), antes mesmo de checar se os campos novos
estão presentes.

A configuração de sobreposição escolhida participa da identidade do
conjunto de quadros do mesmo jeito que a aparência já participa: `NewFrameSetID`
inclui `OverlayConfig.Fingerprint()` no hash, ao lado do de `Appearance` —
sem precisar de um `RenderVersion` novo, e sem nenhuma lógica nova de
reaproveitamento em `render all`/`fly --keep`, que já decide
reaproveitar-ou-redesenhar por `FrameSetID`.

### O controle do registro de dados geográficos (etapa 10)

`geodata clear --confirm` (`specs/010-geo-data-source-control/`) remove
todas as entradas do registro de uma vez — nunca os arquivos de dado
geográfico no disco (`GeoDataRepository.Clear`, escrita atômica igual a
`Save`/`Delete`) — e sem `--confirm` recusa (`ErrRegistryClearNotConfirmed`)
citando quantas entradas seriam removidas, sem precisar de `geodata list`
antes. `--base-map <nome>`/`--elevation <nome>` (mesmo `parseSourceSelection`
compartilhado, `internal/infra/inbound/cli/source_selection.go`) existem em
`geodata check`, `geodata slice` e `fly` — não em `plan` nem em `geodata
elevation` (o primeiro não lê dados geográficos; o segundo já opera sobre
um só tipo). A escolha se encaixa no algoritmo de seleção automática sem
alterá-lo: `domain.SourceSelection.Resolve` restringe, antes de chamar
`Route.Coverage`/`SelectSource`, a lista de candidatos de um tipo a um só
elemento — o mesmo algoritmo de sempre, sobre uma lista de um, garante de
graça tanto o uso exclusivo (nunca mistura, FR-010) quanto a recusa por
cobertura incompleta (FR-007, `ErrAreaNotCovered` em `slice`/`fly`; em
`check`, que só relata, a lacuna aparece no relatório). Um nome que não
existe é `ErrDataSourceNotRegistered` (reaproveitado); um nome que existe
mas é do outro tipo é `ErrDataSourceTypeMismatch`.

O reaproveitamento de um recorte guardado por `fly --keep` passou a exigir
também que a procedência que o recorte já registra (`Summary.Sources`,
existente desde a etapa 4) seja exatamente o conjunto de fontes que um
recorte gerado agora usaria (`GeoSliceService.Sources`, que resolve as
fontes do mesmo jeito que `Generate` — registro, arquivos presentes,
`SourceSelection.Resolve`, `BoundingBox.Regions` —, só pelos metadados, sem
ler conteúdo; `GeoSlice.EnsureUsesSources`, `ErrSliceUsesDifferentSource`) —
ao lado da checagem de plano que `EnsureMatches` já fazia, em
`FlightService.reuseSlice`. A primeira versão (`EnsureUsesSelection`) só
conferia se o nome pedido explicitamente estava *entre* as fontes gravadas, e
nada quando a seleção era automática: voltar de `--base-map X` para a
seleção automática reaproveitava em silêncio o recorte de `X`, e um recorte
automático que misturara dois mapas era reaproveitado para um pedido
explícito de um deles. Comparar com o que seria gerado agora cobre a troca
entre seleção automática e explícita em qualquer direção (e também um
registro que mudou entre as execuções), sem nenhum registro novo nem mudança
no formato do arquivo do recorte; um recorte que sairia igual — um pedido
explícito da única fonte que a seleção automática já tinha usado — continua
reaproveitado.

### O acabamento das sobreposições de tela (etapa 11)

`specs/011-overlay-polish/` corrige o acabamento visual das sobreposições de
tela que a etapa 9 introduziu — nenhum valor exibido, bloco, configuração de
sobreposição, enquadramento, terreno ou traçado muda. O texto passa da fonte
de bitmap (`golang.org/x/image/font/inconsolata`, réplica de pixel) para uma
fonte vetorial embutida (`golang.org/x/image/font/gofont/goregular`, "Go
Regular", licença BSD-3-Clause compatível com o MIT do projeto — já ao
alcance do módulo, nenhuma dependência nova), rasterizada por um
rasterizador próprio, escrito à mão em `internal/domain/vector_font.go`:
`golang.org/x/image/font/sfnt` só extrai os contornos do glifo (dado fixo,
determinístico), e a rasterização em si (achatamento de curvas em um número
fixo de segmentos de reta, cobertura por superamostragem 4×4 com a regra do
número de voltas) usa só os operadores que o resto do desenho de quadros já
usa — nunca `golang.org/x/image/vector.Rasterizer`, que tem um caminho em
assembly só para amd64 (`acc_amd64.s`) com um equivalente em Go puro só para
as demais arquiteturas, exatamente o tipo de divergência entre arquiteturas
que o hash de referência de `frame_scene_test.go` já proíbe
(`research.md` item 1). Cada glifo rasterizado fica em cache por `(rune,
ppem)` em `Scene` (`vectorFace`, construído uma vez em `NewScene`), já que o
tamanho do texto não muda entre quadros de uma mesma execução.

Três acabamentos novos, cada um sua própria história de usuário, todos
dentro de `internal/domain/frame_screen_overlay.go`: um contorno escuro
fixo (`OverlayTextOutlineColor`) desenhado, para cada glifo, antes do
preenchimento — a mesma técnica de casca-antes-do-núcleo que
`TrailCasingColor`/`MarkerRingColor` já usam —, que mantém o texto legível
sobre qualquer fundo sem depender de `OverlayPanelOpacity`; os três painéis
numéricos (distância; elevação e ganho; tempo decorrido) passam a
compartilhar a largura do mais largo presente naquele quadro
(`numericPanelWidth`), em vez de cada um ter a largura do próprio texto; e o
marcador do perfil de elevação ganha raio próprio, proporcional à altura do
quadro com piso em pixels (`ProfileMarkerRadiusRatio`/
`ProfileMarkerMinRadius`, mesmo padrão de `MarkerRadiusRatio`/
`MarkerMinRadius`, mas fixo — não ligado à aparência que o usuário escolhe
para o marcador do terreno). A margem de segurança, antes uma única razão
igual nas quatro bordas (`OverlayMarginRatio`, fração do lado menor), vira
três (`OverlayTopMarginRatio`/`OverlaySideMarginRatio`, fração da
altura/largura; `OverlayBottomMarginRatio`, maior, fração da altura) — a
base de um vídeo vertical é a faixa que redes sociais tipicamente cobrem
com legenda e botões.

Como os pixels de um quadro com sobreposição ligada mudam de verdade (ao
contrário da etapa 6, que só acrescentou um bloco ao arquivo),
`RenderVersion` sobe de `2` para `3` — já suficiente, sem nenhuma mudança de
código além do valor da constante, porque `NewFrameSetID` já inclui
`RenderVersion` no hash: quadros da versão `2` passam a ser, automaticamente,
de outro conjunto, recusados por `render all` sem `--overwrite` e refeitos
por `fly --keep` ao notar a mudança.

### Os níveis de tratamento em `plan` e `fly` (etapa 13)

`--simplification`/`--smoothing` (`specs/013-treatment-level-flags/`)
existem, com o mesmo nome, os mesmos valores aceitos e o mesmo padrão
vindo da configuração, em `plan` e em `fly` — as duas únicas escolhas que,
desde a etapa 1, só o `inspect` deixava o usuário fazer. Não há parser
compartilhado novo: `parsePlanParameters`
(`internal/infra/inbound/cli/plan.go`) já é a única função que `plan.go` e
`fly.go` chamam para interpretar `--duration`/`--fps`/`--distance`/`--tilt`/
`--aspect`, e `parseLevel`/`levelName` já são genéricas no pacote `cli`
(reaproveitadas de `--distance`/`--tilt`) — as duas flags novas entram pelo
mesmo caminho, sem nenhum código de parsing próprio. Os dois níveis
tornam-se dois campos de `domain.PlanParameters` (`Simplification`,
`Smoothing`), ao lado de `Distance`/`Tilt`: `CameraPlanService.Generate`
passa a chamar `TrackService.Treat` com `parameters.Simplification`/
`.Smoothing` em vez de um `defaultLevel` próprio injetado no serviço, que
deixa de existir — a decisão de qual nível é "o padrão" passa a ser
inteiramente da CLI, o mesmo padrão que `Appearance`/`Resolution` já
seguem desde a etapa 8. Isso basta para os dois efeitos que a etapa pede
como regra de negócio: `CameraPlan.ID()` já grava `Distance`/`Tilt` no
hash de identidade do plano — gravar `Simplification`/`Smoothing` do
mesmo jeito é a mesma linha, duas vezes —, e `FlightService.reusePlan` já
decide reaproveitar um plano guardado por `fly --keep` comparando esse
`ID()`, então o reaproveitamento passa a respeitar os dois níveis novos
sem nenhuma lógica própria. O arquivo de plano exportado ganha dois
campos de texto (`parameters.simplification`/`.smoothing`), escritos e
lidos pelas mesmas funções `levelText`/`parseLevel` que o adapter
`jsonfile` já usa para `distance`/`tilt` — a ausência desses dois campos
num arquivo de antes desta etapa cai, sem nenhum código dedicado, no
mesmo "texto desconhecido lê como `medium`" que `parseLevel` já garante
(decisão tomada na sessão de `/speckit-clarify`: como a ferramenta ainda
não tinha sido lançada quando esta etapa foi escrita, um `--keep` sem
nível registrado é, por definição, um plano feito com o único nível que
existia antes, o padrão — nunca um caso incerto). `format_version`
continua `2`: os dois campos são opcionais na leitura, pelo mesmo motivo
que `parameters.aspect_ratio` (etapa 3) não subiu a versão ao ser
acrescentado. Nenhum sentinela de erro novo, nenhum código de saída novo:
um valor fora de `low`/`medium`/`high` continua sendo um erro de uso da
CLI (código `2`), a mesma categoria que `--distance`/`--tilt` já são.
`inspect`, `render frame`, `render all`, `video` e `geodata check` não
mudam — os quatro primeiros porque já faziam ou nunca precisavam fazer o
que a etapa pede; `geodata check` porque chama `TrackService.Clean`,
nunca `Treat`, deliberadamente (a verificação de cobertura é sobre o
trajeto limpo, não simplificado/suavizado, para não mascarar uma lacuna
real — `specs/002-geo-data-registry/research.md` item 9).

### A velocidade da atividade na sobreposição (etapa 14)

`specs/014-speed-overlay-block/` acrescenta às sobreposições de tela um
quinto bloco, `OverlayBlockSpeed` (nome `speed`, rótulo "VEL"), pelo mesmo
mecanismo com que os quatro já existentes são escolhidos
(`OverlayConfig`/`--overlay-blocks`) — mas, diferente deles, nasce fora da
escolha padrão de blocos nesta etapa (a etapa 15 muda esse padrão,
incluindo `speed` nele): só aparece quando pedido por nome.
`CameraFrame` ganha `MarkerSpeed` (metros por segundo), a velocidade média
da atividade numa janela de tempo fixa (`CameraTuning.SpeedWindow`, 30
segundos, um limiar interno de `config.go`, nunca uma flag) centrada no
instante real da atividade do quadro (`ActivityElapsed`) — nunca a
velocidade instantânea entre dois pontos consecutivos do trajeto, que
oscila demais para ser lida. O cálculo reaproveita o mecanismo que já
produz `ActivityElapsed`: `Route.DistanceAt`, o espelho exato de
`Route.TimeAt` que esta etapa acrescenta (tempo decorrido → distância, em
vez de distância → tempo decorrido, com a mesma busca por bracket e o
mesmo `lerp` seguro `(1-t)*a + t*b`), acha a distância nas duas pontas da
janela; a janela já sai encurtada, nunca ausente nem descontínua, nos
extremos do trajeto, porque a mesma forma de `clamp` que `TimeAt` já
aplica faz `DistanceAt` parar no primeiro ou no último ponto. A
disponibilidade da velocidade usa o mesmo critério que já decide
`TimeReference == TimeReferenceClock` (trajeto com horário em todo ponto)
— nenhum campo novo de disponibilidade em `CameraPlan`, ao contrário de
`ElevationAvailable`, porque não é um critério independente. A velocidade
passa a fazer parte do conteúdo do plano, do arquivo de plano exportado
(`frames[].marker.speed_mps`, campo novo e obrigatório) e da identidade do
plano (`CameraPlan.ID()`, ao lado de `CameraToMarkerDistance`) — diferente
de `ActivityElapsed`/`TrackElevation`/`TrackElevationGain`, que ficam de
fora do hash por serem funções determinísticas do que já o compõe, a
velocidade entra porque o pedido original exige explicitamente que dois
planos iguais em tudo menos nela sejam planos diferentes. Como o arquivo
ganha um campo obrigatório que nenhuma versão anterior escrevia,
`format_version` sobe de `2` para `3` pelo mesmo raciocínio que já valeu
na transição `1` → `2` (etapa 9): um plano mais antigo é recusado por
`ErrPlanFormatVersionUnsupported`, a mesma mensagem e o mesmo código de
saída que já recusam qualquer versão desconhecida, sem nenhum sentinela
novo. `inspect`, `geodata check`, `video` e a montagem do vídeo não mudam:
a velocidade é um dado do plano de câmera, consumido só pelo desenho de
quadros (`render frame`, `render all`, `fly`).

### O redesenho das sobreposições de tela em colunas (etapa 15)

`specs/015-overlay-redesign/` reescreve a apresentação das sobreposições
de tela que a etapa 9 introduziu e as etapas 11/12 poliram, sem mudar
nenhum valor, cálculo, arredondamento ou unidade que elas já mostravam
(`internal/domain/frame_screen_overlay.go`). Os blocos numéricos deixam de
ser empilhados verticalmente, cada um sobre uma faixa semitransparente
(`drawPanel`/`OverlayPanelColor`/`OverlayPanelOpacity`, removidos — sem
mais nenhum chamador), e passam a ser texto puro, desenhado direto sobre a
imagem, lado a lado numa única faixa horizontal no alto do quadro: a
largura útil (`largura do quadro − 2×margem lateral`) é repartida em
colunas de mesma largura entre os blocos presentes — uma divisão inteira
simples, sem nenhum escaneamento do plano inteiro — cada bloco
centralizado na própria coluna. A ordem em que aparecem é sempre a mesma,
velocidade → elevação → distância → ganho → tempo decorrido
(`overlayBlockOrder`, em `frame_overlay_config.go`), nunca a ordem em que
o usuário os escreveu em `--overlay-blocks` — `screenOverlay.draw` nunca lê
essa ordem, só os campos booleanos de `OverlayConfig`. Dentro de cada
bloco, a informação vira até três linhas (`overlayBlockText{label, value,
unit}`, `drawBlock`): o rótulo por extenso e com capitalização normal
("Distância", não mais "DIST"), num corpo pequeno
(`OverlayLabelHeightRatio`); o valor, num corpo bem maior
(`OverlayValueHeightRatio`, o dobro do rótulo); e a unidade, de novo no
corpo pequeno, só quando o bloco tem uma — o tempo decorrido não desenha
essa terceira linha, sem deixar o vão que ela ocuparia. As quatro funções
`formatOverlay*` com unidade passam a devolver o valor e a unidade
separados, em vez de uma única string concatenada, para alimentar as duas
linhas; o cálculo e o arredondamento de cada uma continuam exatamente os
mesmos de antes.

O ganho de elevação acumulado, até aqui um segundo número colado ao bloco
de elevação na mesma linha ("ELEV 120 m   GANHO +45 m"), vira um bloco
independente, `OverlayBlockGain` (nome `gain`), pelo mesmo mecanismo de
seleção dos demais (`NewOverlayConfig`, `OverlayConfig.Gain`): o bloco de
elevação passa a mostrar só a altitude, o de ganho só o ganho, cada um com
seu rótulo e sua unidade — pode ser pedido só um, só o outro, os dois, ou
nenhum. A escolha padrão de blocos (`config.RenderDefaults.OverlayBlocks`)
muda de `distance,elevation,time,profile` para
`distance,elevation,speed,profile`: tempo decorrido e ganho continuam
disponíveis, só saem do padrão, porque mudam menos ao longo de um vídeo e
porque três colunas cabem com folga onde cinco não caberiam. O gráfico de
elevação no rodapé (`drawProfile`) também perde o painel e passa a se
destacar do terreno pela mesma técnica de casca-antes-do-núcleo que
`TrailCasingColor` já usa para o traçado: a linha e o marcador são
desenhados duas vezes, primeiro alguns pixels mais largos em
`OverlayTextOutlineColor` (`profileCasingExtra`, um incremento fixo em
pixels, não uma razão), depois no tamanho normal por cima — nenhuma
mudança no conteúdo do gráfico em si (a forma da linha, a posição do
marcador), só na moldura. Como os blocos presentes não precisam mais
compartilhar a largura do texto mais largo de todo o plano,
`stablePanelWidth` e o cache em `Scene` (`panelWidth`/`panelWidthHeight`/
`panelWidthSet`/`numericPanelWidth`) também saem — mais barato do que o
mecanismo que substitui, não só diferente. Como os pixels de um quadro com
sobreposições ligadas mudam de verdade, `RenderVersion` sobe de `4` para
`5`, e `OverlayConfig.Fingerprint()` ganha um sétimo segmento para o bit
de `gain` — o mesmo mecanismo de sempre (`FrameSetID`) garante que
nenhum conjunto de quadros de antes desta etapa se misture com um de
depois, sem nenhum código novo de comparação.

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
