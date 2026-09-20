# Pesquisa: Recorte de Dados Geográficos para o Voo

**Feature**: `004-geo-data-slice` | **Data**: 2026-09-20

Este documento consolida as decisões técnicas que a especificação deixou
deliberadamente em aberto (ver "Adiado" do relatório do `/speckit-clarify`):
limite de tamanho, formato do destino da exportação, forma de ler a elevação,
fórmula do nível de detalhe e margem da área — e as escolhas necessárias para
ler o **conteúdo** dos dois formatos que a etapa 2 já reconhece (MBTiles e
GeoTIFF). Os valores numéricos são **valores iniciais**, concentrados em uma
única estrutura de configuração (`domain.SliceTuning`, item 15), para serem
ajustados sem alterar nenhum contrato.

Uma dependência nova: `golang.org/x/image/tiff/lzw` (item 8). Todo o resto é
biblioteca padrão e os módulos que o projeto já usa (`modernc.org/sqlite`,
Cobra, testify, mockgen).

## 1. Como o plano entra: porta de leitura + validação no domínio

- **Decisão**: um novo método `CameraPlanService.Load(path)` lê o arquivo de
  plano por uma nova porta `CameraPlanReader` (irmã de `CameraPlanExporter`,
  no mesmo arquivo `camera_plan.go`; adapter `jsonfile.NewCameraPlanReader()`,
  que já conhece o formato do arquivo) e devolve um `domain.CameraPlan`. A
  validação de coerência é método de domínio, `CameraPlan.Validate()`. O
  recorte (`GeoSliceService.Generate(plan)`) trabalha só sobre um
  `CameraPlan` já carregado, sem saber de onde ele veio; a CLI encadeia
  `Load` → `Generate` → `Export`.
- **Racional**: é a divisão que o Princípio IX pede (E/S no adapter, regra
  no domínio) e é o que deixa um comando futuro que encadeie tudo em memória
  reutilizar `GeoSliceService.Generate` sem tocar em arquivo (Clarificação
  de 2026-09-20). `CameraPlanService` já é o dono do recurso "plano"
  (`Generate`, `Export`), então `Load` é mais um método dele, não um serviço
  novo.
- **O que `Validate` confere** (FR-001a): `frames` não vazio;
  `summary.frame_count == len(frames) == round(duration_s × frame_rate)`;
  `frames[i].index == i`; fases só `opening`/`following`/`closing`;
  latitude em [-90, 90] e longitude em [-180, 180] em câmera e marcador;
  distância câmera–marcador finita e ≥ 0; números finitos. O adapter só
  decodifica; recusa JSON inválido, campos obrigatórios ausentes
  (`frames[].marker`, `frames[].camera`, `frames[].camera_to_marker_m`,
  `parameters`, `summary.frame_count`) e `format_version` ausente. Campos
  desconhecidos são ignorados (contrato do plano: "consumidores devem
  ignorar campos desconhecidos").
- **Versão**: `format_version == 1` é aceita; qualquer outro valor →
  `ErrPlanFormatVersionUnsupported`, com a versão encontrada e a lista das
  aceitas (`[1]`).
- **Alternativas rejeitadas**: a CLI decodificar o JSON (regra e formato
  vazando para o adapter de entrada, contra o Princípio III); um serviço
  `PlanFileService` só para ler (serviço por caso de uso, proibido pelo
  Princípio IX).
- **Ajuste na spec**: a suposição que dizia "a leitura e a validação do
  arquivo de plano são responsabilidade da porta de entrada" foi corrigida
  para "de um adapter de saída, invocado antes do recorte".

## 2. Área de interesse: o que "a área que a câmera percorre" quer dizer

- **Decisão**: `CameraPlan.AreaOfInterest(tuning SliceTuning) BoundingBox`.
  Para cada quadro, o terreno relevante é o quadrado, no plano local do
  próprio quadro, de meio-lado `MarginFactor × camera_to_marker_m` (inicial:
  `MarginFactor = 1.0`) centrado no marcador. A câmera está a
  `camera_to_marker_m` do marcador, portanto sempre dentro desse quadrado. A
  área é a caixa envolvente da união de todos os quadrados.
- **Conversão metros → graus** (por quadro): `Δlat = m / 111 320`,
  `Δlon = m / (111 320 · cos φ)`, com `cos φ` limitado inferiormente a
  `1e-6` para que a longitude não exploda nos polos; a largura total em
  longitude é limitada a 360°. Latitudes finais limitadas a [-90, 90].
- **Antimeridiano**: as longitudes dos marcadores dos quadros são
  desembrulhadas somando deltas consecutivos (o plano é contínuo), como
  `Route.BoundingBox` já faz (etapa 1); só depois se expande e se
  normaliza, produzindo um `BoundingBox` com `CrossesAntimeridian` — o
  mesmo tipo que a etapa 2 usa nos registros. As coordenadas resultantes são
  arredondadas a 1e-7° (mesma quantização do plano) para que a área e tudo
  que deriva dela seja idêntico entre plataformas (item 12).
- **Racional**: os quadros de abertura e fechamento têm `camera_to_marker_m`
  grande (o trajeto inteiro enquadrado) e naturalmente alargam a área; o
  acompanhamento contribui com faixas estreitas. Não há suposição sobre o
  campo de visão real (o plano não o carrega) — a margem é uma folga
  proporcional à distância, ajustável.
- **Alternativas rejeitadas**: caixa apenas dos marcadores mais uma margem
  fixa em metros (deixa de fora o terreno visto pela câmera alta na
  abertura); traçar o cone de visão de cada quadro (depende do campo de
  visão e da inclinação reais, que só a etapa de renderização conhece).

## 3. Verificação de cobertura de uma área, reaproveitando `Route.Coverage`

- **Problema**: `Route.Coverage` (etapa 2) verifica **pontos**; o recorte
  precisa de uma **área**. Verificar só o perímetro ou uma grade fixa
  deixaria passar buracos entre registros retangulares.
- **Decisão**: decompor a área em **regiões** por compressão de
  coordenadas. Os limites (latitudes e longitudes) das caixas dos registros
  candidatos que intersectam a área, mais os da própria área, particionam a
  área em retângulos; dentro de cada retângulo, a lista de registros que o
  contêm é uniforme, logo o **centro** de cada um representa o retângulo
  inteiro **exatamente**. `BoundingBox.Regions(baseMaps, elevations)`
  devolve as regiões em ordem determinística (linhas de sul a norte,
  colunas de oeste a leste, no espaço de longitude desembrulhado a partir do
  limite oeste da área) e, junto, um `Route` cujos pontos são os centros. A
  verificação é `Route.Coverage(baseMaps, elevations)` **sem alteração**:
  escolha do registro (menor área, empate pelo mais antigo), exclusão de
  arquivo ausente (feita por quem chama, como na etapa 2), agrupamento em
  subtrechos e `MissingDataType` são os mesmos.
- **Relatório**: os subtrechos não cobertos trazem as coordenadas dos
  **centros** da primeira e da última região não coberta do subtrecho
  contíguo (nas mesmas linhas de varredura). A spec exige "coordenadas de
  início e fim"; a precisão é a da região, documentada em `contracts/cli.md`.
  O texto da SC-004 foi ajustado: "mesmas regras e mesmo formato", não
  "idêntico ao relatório de um trajeto".
- **Custo**: com `n` registros candidatos há no máximo `(2n+1)²` regiões;
  na prática, uma região (um registro de cada tipo cobre tudo).
- **Alternativa rejeitada**: reimplementar a verificação para áreas
  (duplicaria a regra de FR-014/FR-016 da etapa 2).

## 4. Regiões e o "vencedor" de cada região

- **Decisão**: cada região tem exatamente um registro vencedor de mapa base
  e um de relevo (a cobertura garante que existem). O recorte é montado por
  região: as amostras de elevação da região são lidas do relevo vencedor e
  as peças de mapa da região, do mapa base vencedor. Uma **amostra** pertence
  à região que contém o **centro** da amostra (intervalo semiaberto: limite
  sul/oeste inclusive, norte/leste exclusive, exceto na borda externa da
  área, que é inclusiva); uma **peça** pertence à região que contém o centro
  de `peça ∩ área`. Assim toda amostra e toda peça pertence a exatamente uma
  região e não há duplicidade quando dois registros se sobrepõem.
- **Consequência**: com um registro de cada tipo, há uma única região e o
  recorte tem uma grade de elevação e um conjunto de peças. Com vários, o
  recorte tem uma grade por (região com amostras) e um conjunto de peças por
  (registro de mapa base, nível).
- **Mesma regra na consulta isolada** (item 9): a consulta usa
  `SelectSource` (o vencedor de `Route.Coverage`) num ponto só.

## 5. Nível de detalhe do mapa base

- **Decisão**: o nível ideal `z*` é o menor nível de zoom cuja resolução no
  terreno é menor ou igual ao tamanho, no terreno, de um pixel de tela na
  distância mínima da câmera, ampliado por `TexelScreenRatio`:

  ```
  d_min     = summary.camera_distance_m.min      (do plano)
  footprint = TexelScreenRatio · 2 · d_min · tan(FOV/2) / ReferenceHeightPixels
  res(z, φ) = 156543.03392 · cos φ / 2^z         (metros por pixel, peça de 256 px)
  φ_ref     = latitude com o menor |φ| dentro da área (0 se a área cruza o equador)
  z*        = min { z ∈ ℕ : res(z, φ_ref) ≤ footprint }
  ```

  com `FOV = CameraTuning.OverviewVerticalFOVDegrees` (o mesmo campo de visão
  que o plano assume). Valores iniciais: `ReferenceHeightPixels = 1080`,
  `TexelScreenRatio = 2` (um pixel da peça pode ocupar até 2 pixels de tela
  na aproximação máxima).
- **Dentro do que a fonte oferece**: por registro de mapa base usado, o nível
  efetivo é `clamp(z*, minzoom, maxzoom)` do arquivo (item 7). O resumo
  informa `z*`, o efetivo, o intervalo oferecido e o motivo em texto: qual
  `d_min`, qual `φ_ref`, qual resolução exigida e se houve limitação
  ("above the source's maximum level" / "below the source's minimum level" /
  "within the source's range").
- **Determinismo e monotonicidade** (FR-006, FR-007): `z*` é função só de
  `d_min`, `FOV`, `φ_ref` e constantes; `d_min` menor ⇒ `footprint` menor ⇒
  `z*` maior ou igual. `φ_ref` usa o menor `|φ|` da área porque a resolução
  exigida é mais fina onde `cos φ` é maior — a escolha atende a área inteira
  e não depende de hemisfério.
- **Como a "área do voo" entra**: por `φ_ref` (a área decide onde a peça é
  mais grossa), pela quantidade de peças e amostras que resulta do nível
  escolhido (item 11) e pelo motivo impresso no resumo. Não há
  "afrouxamento automático" do nível para caber no limite: um recorte
  grande demais é **recusado** (FR-011), com a dica de aumentar
  `--distance` do plano ou encurtar o trajeto.
- **Alternativas rejeitadas**: um nível por quadro (LOD dinâmico — o mesmo
  recorte serviria a várias resoluções, mas a etapa de renderização ainda
  não existe para dizer se compensa); baixar o nível até caber no limite
  (esconderia a perda de detalhe do usuário).

## 6. Peças de mapa base: grade, antimeridiano e polos

- **Decisão**: a grade é a do esquema XYZ de Web Mercator (peças de 256 px)
  — o que MBTiles carrega. Para o nível `z` (`n = 2^z`):
  `x = floor((lon + 180) / 360 · n)`, `y = floor((1 − asinh(tan φ)/π) / 2 · n)`,
  com `φ` limitada a ±85,0511287798° (limite de Web Mercator) e índices
  limitados a `[0, n−1]`. As peças necessárias são o retângulo
  `[xmin..xmax] × [ymin..ymax]`; quando a área cruza o antimeridiano, são
  dois retângulos de `x`: `[xmin..n−1]` e `[0..xmax]`.
- **Polos**: acima de 85,0511° não existe peça; a linha mais ao norte (ou
  sul) da grade é usada como está, e o resumo não trata isso como erro
  (a cobertura por metadados já foi verificada). Nenhum tratamento por
  região: é a mesma fórmula em qualquer latitude (Princípio IV).
- **Ausentes** (FR-010): uma peça requerida que a fonte não contém é
  registrada como `TileID{Level, X, Y}` em `Missing`; o recorte segue. Zero
  peças presentes é um recorte válido, com o resumo destacando "0 of N tiles
  present".
- **Linha TMS**: MBTiles guarda `tile_row` em TMS (origem no sul);
  `tile_row = 2^z − 1 − y`. A conversão fica no adapter; o domínio só conhece
  XYZ.

## 7. Leitura de MBTiles (`BaseMapReader`)

- **Decisão**: porta `BaseMapReader`, declarada em `tile.go` (entidade
  dona: `Tile`). O adapter vive num pacote novo, `basemapreader` (pacote de
  estratégias de uma única porta; hoje só há MBTiles), com
  `basemapreader.NewMBTiles()` (arquivo `mbtiles_base_map_reader.go`),
  reaproveitando `modernc.org/sqlite`. Não entra em `geodatainspector`: aquela
  é a porta de *inspeção* (tipo e área no registro); ler conteúdo é outra
  razão para mudar. Métodos:
  - `Levels(path string) (LevelRange, error)`: `minzoom`/`maxzoom` do
    `metadata`; se ausentes, `SELECT MIN(zoom_level), MAX(zoom_level) FROM tiles`.
    Erro (tabela ausente, SQL inválido, arquivo truncado) →
    `ErrGeoDataContentUnreadable`.
  - `ReadTiles(path string, level int, ids []TileID) (TileRead, error)`:
    uma consulta preparada por peça (`WHERE zoom_level=? AND tile_column=? AND
    tile_row=?`), numa única abertura do arquivo, em modo somente leitura
    (`?mode=ro`); devolve as peças encontradas (bytes crus, sem decodificar
    imagem) e o formato (`metadata.format`: `png`, `jpg`, `webp`, `pbf`;
    ausente → `png`). Peça inexistente não é erro.
- **Sem processamento de imagem** (spec, Suposições): os bytes da peça
  passam intactos do arquivo ao recorte.
- **Erro de arquivo**: arquivo ausente → o filtro de disponibilidade
  (`FileChecker`) já o excluiu antes; falha ao abrir/consultar um registro
  disponível → `ErrGeoDataContentUnreadable` com o nome do registro.
- **Alternativa rejeitada**: reusar `geodatainspector` (a porta de inspeção
  passaria a ter métodos de leitura de conteúdo, misturando duas razões
  para mudar).

## 8. Leitura de GeoTIFF (`ElevationReader`)

- **Decisão**: porta `ElevationReader`, declarada em `elevation_grid.go`
  (entidade dona: `ElevationGrid`), com adapter em Go puro
  `elevationreader.NewGeoTIFF()` (arquivo `geotiff_elevation_reader.go`),
  no mesmo estilo da leitura de tags já feita pelo `geodatainspector`
  (que continua só lendo tags). Nenhuma biblioteca de GeoTIFF (as
  disponíveis em Go ou não leem tiles/float, ou dependem de GDAL/CGO, o que
  violaria "funcionar offline e sem dependência de sistema"). Métodos:
  - `Describe(path) (ElevationGridInfo, error)`: lê **só tags** — largura,
    altura, escala de pixel, ponto de amarração, tipo e tamanho da amostra,
    compressão, layout (faixas ou peças), valor de "sem dado"
    (`GDAL_NODATA`, tag 42113, texto ASCII), unidade vertical e tipo de
    raster (ver abaixo). Nunca lê amostras. Devolve a geometria da grade e
    `ErrGeoDataContentUnreadable` se qualquer parte necessária faltar ou for
    incoerente.
  - `ReadWindow(path, window GridWindow) (ElevationWindow, error)`: lê só as
    faixas/peças TIFF que intersectam a janela, decodifica e devolve as
    amostras em metros como `float32`, com "sem dado" marcado.
- **Subconjunto suportado**: 1 amostra por pixel, arranjo contíguo; tipos
  inteiro 8/16/32 bits (com ou sem sinal) e ponto flutuante 32/64 bits
  (`SampleFormat`); compressão nenhuma (1), Deflate (8 e 32946, `compress/zlib`)
  e LZW (5, `golang.org/x/image/tiff/lzw`); predictor 1 (nenhum), 2
  (horizontal, inteiros) e 3 (ponto flutuante); layout em faixas
  (`StripOffsets`) ou em peças (`TileOffsets`); ordem de bytes `II` ou `MM`.
  Tudo fora disso (JPEG, PackBits, multibanda, BigTIFF — que o inspetor já
  nem reconhece) → `ErrGeoDataContentUnreadable` com "unsupported encoding:
  <o quê>".
- **Unidade** (FR-008): `VerticalUnitsGeoKey` (4099): 9001 = metro (sem
  conversão), 9002 = pé internacional (×0,3048), 9003 = pé US survey
  (×1200/3937); ausente = metro (o que a maioria dos modelos digitais de
  elevação assume); qualquer outro valor → `ErrElevationUnitUnsupported`,
  nomeando o registro.
- **Referência da amostra**: `GTRasterTypeGeoKey` (1025): `PixelIsArea` (padrão)
  → o ponto de amarração é o **canto noroeste** da célula (0,0);
  `PixelIsPoint` → é o **centro** dessa célula, e a grade é deslocada de
  meia célula para noroeste. (O `geodatainspector` da etapa 2 trata o ponto como canto; a
  diferença é de meia célula na caixa e não afeta a cobertura, mas o leitor
  de conteúdo faz certo.)
- **"Sem dado"** (FR-009): uma amostra é "sem valor" se for igual ao
  `GDAL_NODATA` (comparado no tipo original, antes de qualquer conversão de
  unidade) ou, para ponto flutuante, se for NaN. Nunca vira zero.
- **Alternativas rejeitadas**: GDAL via CGO (não é offline puro nem
  portável); escrever um decodificador de LZW próprio (200 linhas mantidas
  por nós contra um pacote oficial do `golang.org/x`); só aceitar
  GeoTIFF sem compressão (a maioria dos DEMs reais é Deflate ou LZW).

## 9. Elevação: a célula que contém o ponto

- **Decisão**: a elevação de uma coordenada é o valor da **célula da grade
  que a contém** (vizinho mais próximo, sem interpolação):
  `linha = floor((norte − lat) / Δlat)`, `coluna = floor((lon − oeste) / Δlon)`,
  com `lon − oeste` medido no sentido leste, módulo 360; um ponto exatamente
  sobre o limite entre duas células pertence à célula ao sul/leste, e os
  índices são limitados à última linha/coluna na borda sul/leste externa. O recorte guarda as células inteiras,
  então o valor consultado e o valor do recorte para a mesma coordenada
  são, por construção, o mesmo (FR-018, SC-006).
- **Racional**: interpolação inventaria valor (e misturaria "sem dado" com
  dado); vizinho mais próximo é o que um usuário confere contra uma fonte
  externa (célula do arquivo). A resolução vertical do arquivo é o erro
  máximo esperado em relação a essa fonte.
- **Consulta**: `GeoDataService.ElevationAt(latitude, longitude)` escolhe o
  vencedor entre os relevos disponíveis (mesma regra), pede à porta uma
  janela de 1×1 e devolve `ElevationReading{Meters, HasValue, Source}`. Se
  nenhum relevo contém o ponto → `ErrElevationNotCovered`. Longitude 180 e
  −180 são normalizadas para a mesma posição antes de qualquer escolha.

## 10. Como "sem valor" viaja dentro do núcleo e no arquivo

- **Decisão**: `ElevationGrid` guarda `[]float32` e expõe `At(row, col)
  (meters float64, hasValue bool)`; internamente "sem valor" é o NaN de
  `float32` (encapsulado: nenhum outro código do núcleo vê NaN). No
  arquivo exportado (`contracts/slice-file.md`), o vetor de amostras é
  gravado como `float32` little-endian e "sem valor" é o NaN silencioso
  `0x7FC00000`, documentado; o `manifest.json` traz `no_value_count` por
  grade para conferência.
- **Racional**: uma máscara à parte custaria ~25% a mais de memória e de
  código sem benefício; a representação NaN é padrão IEEE e o contrato a
  documenta.

## 11. Tamanho do recorte: estimativa antes de ler, guarda durante a leitura

- **Decisão**: `MaxSizeBytes = 256 MiB` (`SliceTuning`). **Estimativa** (sem
  ler conteúdo, só contas e `Describe`/`Levels`):
  `bytes = tileCount · EstimatedTileBytes + sampleCount · 4`, com
  `EstimatedTileBytes = 65 536` e `tileCount`, `sampleCount` calculados a
  partir da área, do nível efetivo e da geometria das grades. Se
  `bytes > MaxSizeBytes` → `ErrSliceTooLarge`, com estimativa, limite, e a
  causa provável (nível efetivo e extensão da área). **Guarda**: como o
  tamanho real de uma peça só se conhece ao lê-la, a leitura acumula os
  bytes reais e aborta com o mesmo erro se ultrapassar o limite. Um recorte
  exatamente no limite é aceito (`>` estrito).
- **Racional**: SC-010 exige recusar em menos de 5 s sem ler conteúdo; a
  estimativa por contagem cumpre isso. O tamanho real (o do resumo) pode
  diferir da estimativa; o limite vale para os dois.
- **Alternativa rejeitada**: consultar `length(tile_data)` de cada peça
  antes de ler (tamanho exato, mas uma passada extra por peça; a
  estimativa por contagem basta e é O(1) em relação ao arquivo).

## 12. Determinismo

- Nenhuma iteração de `map` que influencie a saída (chaves ordenadas),
  nenhuma concorrência, nenhuma dependência de relógio ou de caminho: o
  recorte depende só do plano, dos registros e do conteúdo dos arquivos.
- Ordem de saída: regiões (item 3) → dentro delas, elevação por linha (norte
  para sul) e coluna (oeste para leste); peças ordenadas por
  `(registro, nível, x, y)`; registros ordenados por nome (como
  `CoverageReport`).
- A única aritmética de ponto flutuante do recorte é a da área (item 2) e do
  nível (item 5); ambas quantizadas (1e-7°) ou inteiras no resultado (`z`,
  índices de peça, janela de amostras). Valores de elevação só são copiados
  (e, para pés, multiplicados por uma constante), nunca interpolados.
- O arquivo exportado é determinístico (item 13).

## 13. Formato do destino da exportação: um arquivo ZIP

- **Decisão**: **um único arquivo** (`--export slice.zip`), um ZIP sem
  compressão (método *store*) com ordem de entradas fixa e data de
  modificação fixa, contendo `manifest.json` (JSON versionado: área,
  resumo, procedência, nível de detalhe, peças ausentes, metadados de
  cada grade), `elevation/NNN.f32` (uma por grade) e
  `tiles/<registro>/<z>/<x>/<y>.<ext>` (bytes originais). Contrato
  completo em `contracts/slice-file.md`.
- **Racional**: é uma **unidade única e atômica** (FR-014): escrita em
  arquivo temporário no mesmo diretório e publicada com o mesmo
  procedimento que o exportador do plano usa (recusa exclusiva sem
  `--overwrite`, `rename` com `--overwrite`) — sem janela de estado parcial,
  o que um diretório não permite (renomear sobre um diretório não vazio não
  é atômico). Fica legível por qualquer ferramenta (`unzip -l`, Python
  `zipfile`, etc.), preserva as peças como bytes originais (base64 dentro de
  JSON incharia ~33%) e é determinístico sem compressão (o `deflate` do Go
  não garante os mesmos bytes entre versões do compilador).
- **Reuso**: a publicação atômica hoje vive em `jsonfile` (`publishExclusive`
  e a sequência temporário → chmod → rename). Ela é extraída para um pacote
  utilitário `internal/infra/outbound/atomicfile` (`Publish(path, overwrite,
  write)`), usado pelos dois exportadores; cada um traduz `atomicfile.ErrExists`
  e falhas de destino para o seu sentinela. Sem mudança de comportamento do
  `plan --export` (os testes existentes continuam válidos).
- **Adapter**: pacote de tecnologia `zipfile`, `zipfile.NewGeoSliceExporter()`
  (arquivo `geo_slice_exporter.go`), seguindo a convenção do `jsonfile`.
- **Alternativas rejeitadas**: diretório (não atômico); um único JSON com
  peças em base64; arquivo SQLite/MBTiles (a elevação não é grade de peças);
  formato próprio binário (ilegível por ferramentas comuns).

## 14. Erros sentinela e códigos de saída

Dez sentinelas novos em `errors.go` (a CLI os traduz em códigos 17 a 26,
`contracts/cli.md`): `ErrPlanFileInvalid`, `ErrPlanFormatVersionUnsupported`,
`ErrAreaNotCovered` (tipo `AreaNotCoveredError` com o `CoverageReport`,
`Is` casando o sentinela, mensagem com os subtrechos), `ErrSliceTooLarge`,
`ErrGeoDataContentUnreadable`, `ErrElevationUnitUnsupported`,
`ErrSliceDestinationExists`, `ErrSliceDestinationInvalid`,
`ErrElevationNotCovered`, `ErrInvalidCoordinate`. Plano inexistente ou
ilegível por E/S usa o código genérico `4`, como o arquivo de trajeto nas
etapas anteriores. "Elevação sem valor" **não** é erro (código `0`).

## 15. Portas, serviços e configuração

- **Portas novas** (Princípios II e IX; nomeadas pelo papel; cada uma no
  arquivo da entidade que produz, interface no topo, `//go:generate`
  logo após `package`): `CameraPlanReader` (`camera_plan.go`),
  `BaseMapReader` (`tile.go`), `ElevationReader` (`elevation_grid.go`),
  `GeoSliceExporter` (`geo_slice.go`).
- **Serviços** (um por recurso): novo `GeoSliceService` (`Generate(plan)`,
  `Export(slice, path, overwrite)`); `CameraPlanService` ganha `Load(path)`;
  `GeoDataService` ganha `ElevationAt(latitude, longitude)`. Dependências:
  `GeoSliceService` ← `GeoDataRepository`, `FileChecker`, `BaseMapReader`,
  `ElevationReader`, `GeoSliceExporter`, `domain.SliceTuning`,
  `domain.CameraTuning` (o campo de visão); `GeoDataService` passa a
  receber também `ElevationReader`.
- **Regra no domínio**: `CameraPlan.Validate`, `CameraPlan.AreaOfInterest`,
  `BoundingBox.Regions`, `SliceTuning.DetailLevel`, `BoundingBox.TileRange`,
  `ElevationGridInfo.Window`, `ElevationGrid.At`, `NewGeoSlice` (resumo a
  partir do conteúdo, como `NewCameraPlan`), `SelectSource`. O serviço só
  chama portas e esses métodos, na ordem certa: `Generate` = validar plano →
  área → listar registros disponíveis → regiões + `Coverage` (recusa) →
  nível e estimativa (recusa) → ler grades e peças → `NewGeoSlice`.
- **CLI**: novos comandos filhos de `geodata` — `slice` e `elevation` — em
  arquivos próprios (`geodata_slice.go`, `geodata_elevation.go`), como os
  demais; testados com `GeoSliceService`, `GeoDataService` e
  `CameraPlanService` mockados.
- **Configuração** (Princípio VIII): `domain.SliceTuning` (formato do
  núcleo) e `config.SliceTuning` (tipos próprios do adapter de
  configuração, sem importar o domínio), mapeados em
  `cmd/sobrevoo/config_mapping.go`. Valores iniciais:

  | Constante | Valor | Significado |
  |---|---|---|
  | `MarginFactor` | 1,0 | meio-lado do terreno relevante por quadro, em múltiplos de `camera_to_marker_m` |
  | `ReferenceHeightPixels` | 1080 | altura de tela de referência para o nível de detalhe |
  | `TexelScreenRatio` | 2,0 | pixels de tela que um pixel de peça pode ocupar na aproximação máxima |
  | `EstimatedTileBytes` | 65 536 | tamanho assumido por peça na estimativa |
  | `MaxSizeBytes` | 268 435 456 | limite do recorte (256 MiB) |

  Constantes que são fato da grade e não ajuste (peça de 256 px,
  4 bytes por amostra, limite de Web Mercator, metros por grau) ficam como
  constantes do domínio, não em `SliceTuning`.

## 16. Desempenho

O custo é dominado por E/S: uma consulta SQLite por peça (dezenas a poucos
milhares) e a decodificação de faixas/peças TIFF que intersectam a janela
(uma passada; nada além do necessário é lido). SC-001 (menos de 30 s para
um voo de até 50 km, parâmetros padrão) é atendido com folga nos tamanhos
alvo. Sem concorrência (determinismo e simplicidade, item 12).

## 17. Testes e fixtures

- `test/helper` ganha construtores de fixtures com **conteúdo**: MBTiles com
  peças em níveis e posições escolhidos (e com buracos), e GeoTIFF com
  amostras (sem compressão, Deflate, LZW, faixas e peças, `int16` e
  `float32`, com e sem `GDAL_NODATA`, em metros e em pés), inclusive
  cruzando o antimeridiano e em latitude alta. Todos gerados em código, sem
  arquivos binários versionados (mesmo estilo dos fixtures atuais).
- Domínio: funções puras com testes de propriedade (nível monotônico na
  distância; área contém todas as câmeras e marcadores; cada amostra e cada
  peça em exatamente uma região; deslocamento da área para outras regiões
  do planeta dá as mesmas contagens, SC-009).
- Adapters (`basemapreader`, `elevationreader`, `jsonfile.CameraPlanReader`,
  `zipfile`): testados contra os fixtures em diretório temporário — é o
  adapter que toca o disco, então é o objeto do teste. CLI e serviços:
  isolados por mocks (`mockdomain`, `mockapplication`).
- Não há teste automatizado de ponta a ponta; `quickstart.md` é o checklist
  manual.
