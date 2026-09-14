# Pesquisa: Registro de Dados Geográficos Locais

**Feature**: `002-geo-data-registry` | **Data**: 2026-09-13

Este documento consolida as decisões técnicas necessárias para resolver os
pontos deliberadamente deixados em aberto pela especificação (a Suposição
sobre "formatos exatos reconhecidos" e o restante do Contexto Técnico do
`plan.md`), a partir da entrada do usuário e da especificação já
clarificada.

## 1. Formato de mapa base: MBTiles

- **Decisão**: o único formato reconhecido para dados de mapa base nesta
  etapa é MBTiles — um container SQLite com um esquema padronizado
  (especificação MBTiles 1.3), cuja tabela `metadata` inclui uma chave
  `bounds` com a extensão geográfica coberta (`minLon,minLat,maxLon,maxLat`).
- **Racional**: é o formato de fato mais usado para empacotar mapas offline
  (saída padrão de ferramentas como MapLibre, Tippecanoe, TileMill, e
  exportações do QGIS), já traz a área geográfica coberta pronta e
  padronizada no próprio conteúdo (sem exigir decodificar tiles), e é
  identificável por assinatura de arquivo (é literalmente um banco SQLite —
  os primeiros 16 bytes são o cabeçalho `"SQLite format 3\0"`), permitindo
  detecção de tipo por conteúdo (FR-002) tão simples quanto a verificação de
  elemento raiz XML já usada pelo `GPXParser` da etapa 1.
- **Alternativas consideradas**: GeoPackage (rejeitado — container mais
  genérico, usado tanto para raster quanto vetor, sem uma convenção única e
  simples de "bounds" equivalente à de MBTiles, exigindo mais lógica para
  cobrir todos os casos); PMTiles (rejeitado — formato mais recente, sem
  driver maduro e amplamente adotado em Go puro no momento desta decisão);
  suportar mais de um formato de mapa base de uma vez (rejeitado — não
  agrega valor a esta etapa e amplia sem necessidade a superfície de código
  e de testes, na mesma linha da decisão de escopo já tomada para trajetos
  na etapa 1).

## 2. Biblioteca para ler MBTiles

- **Decisão**: usar `modernc.org/sqlite`, um driver SQLite escrito
  inteiramente em Go (sem cgo), acessado via `database/sql`, apenas para
  consultar a tabela `metadata` (chaves `bounds` e `format`) — nunca para
  decodificar tiles.
- **Racional**: por ser puro Go, preserva a compilação cruzada sem
  dependência de bibliotecas de sistema, mantendo a mesma característica de
  "binário multiplataforma" já estabelecida na etapa 1. `database/sql` é a
  mesma abstração idiomática de acesso a banco da biblioteca padrão.
- **Alternativas consideradas**: `mattn/go-sqlite3` (rejeitado — depende de
  cgo, o que complicaria a compilação cruzada e introduziria uma
  dependência de toolchain C); implementar um parser do formato de arquivo
  SQLite do zero (rejeitado — o formato de página B-tree do SQLite é
  complexo demais para valer a pena reimplementar apenas para ler uma
  tabela de metadados).

## 3. Formato de relevo: GeoTIFF (apenas CRS geográfico)

- **Decisão**: o único formato reconhecido para dados de relevo nesta etapa
  é GeoTIFF em CRS geográfico (WGS84 / EPSG:4326) — a área geográfica é
  calculada a partir das tags de georreferenciamento do próprio arquivo
  (`ModelPixelScaleTag`, `ModelTiepointTag` ou `ModelTransformationTag`,
  mais `ImageWidth`/`ImageLength`), e o CRS é confirmado pela
  `GeoKeyDirectoryTag` (`GTModelTypeGeoKey = 2`, Geographic). Um GeoTIFF em
  CRS projetado (ex.: UTM) é tratado como formato não suportado nesta etapa.
- **Racional**: GeoTIFF é o formato de fato mais comum para dados de
  elevação (DEM) distribuídos por fontes amplamente usadas para uso
  offline (SRTM, Copernicus DEM, ASTER GDEM, USGS 3DEP), e — ao contrário de
  um `.hgt` bruto do SRTM, cuja localização geográfica vem inteiramente do
  **nome do arquivo** por convenção — carrega a área geográfica coberta no
  próprio **conteúdo**, o que o FR-003 exige explicitamente ("examinar o
  conteúdo do arquivo"). Restringir a CRS geográfico evita a necessidade de
  reprojeção: as fontes DEM globais mais comuns já distribuem nesse CRS por
  padrão.
- **Alternativas consideradas**: `.hgt` bruto do SRTM (rejeitado — a área
  geográfica não está no conteúdo do arquivo, só no nome, o que
  contrariaria o FR-003 e tornaria a detecção frágil a renomeação);
  NetCDF (rejeitado — formato genérico para dados científicos
  multidimensionais, mais complexo de interpretar do que o necessário para
  este caso de uso); suportar também CRS projetado, com reprojeção para
  geográfico (rejeitado — adicionaria complexidade de projeção cartográfica
  sem necessidade real, dado que as fontes DEM mais comuns já usam CRS
  geográfico).

## 4. Parsing de GeoTIFF: leitor de tags próprio, sem biblioteca externa

- **Decisão**: implementar um leitor mínimo de tags TIFF/GeoTIFF
  diretamente no adapter (`internal/infra/outbound/geodatainspector/geotiff.go`),
  usando apenas `encoding/binary` e `io.ReaderAt` sobre a biblioteca padrão
  — lendo unicamente o cabeçalho TIFF, o IFD (Image File Directory) e as
  poucas tags necessárias, sem decodificar o raster completo.
- **Racional**: mesma filosofia já registrada em `research.md` (etapa 1,
  item 4) para os algoritmos de simplificação/suavização — bibliotecas
  gerais de imagem TIFF (ex.: `golang.org/x/image/tiff`) decodificam a
  imagem, mas não expõem as tags GeoTIFF privadas necessárias para
  georreferenciamento; bibliotecas GeoTIFF/GDAL completas (ex.:
  `airbusgeo/godal`) exigem cgo e a instalação da biblioteca GDAL no
  sistema, o que quebraria a compilação cruzada sem dependências de sistema
  e o funcionamento "só binário" já estabelecido. O conjunto de tags
  necessário é pequeno e bem documentado (especificação GeoTIFF 1.8.1),
  tornando um leitor próprio pequeno, autocontido e fácil de testar
  diretamente — sem superfície de ataque de um parser de imagem completo.
- **Alternativas consideradas**: `golang.org/x/image/tiff` (rejeitada —
  decodifica a imagem, mas não expõe as tags privadas de GeoTIFF);
  `airbusgeo/godal`/bindings de GDAL (rejeitada — cgo + dependência de
  sistema, incompatível com a meta de binário multiplataforma sem
  dependências externas); adotar uma biblioteca Go de terceiros dedicada a
  GeoTIFF puro (nenhuma opção amplamente adotada e mantida foi encontrada
  no momento desta decisão).

## 5. Persistência do registro: arquivo JSON em `~/.sobrevoo`

- **Decisão**: o registro é um único arquivo JSON em
  `~/.sobrevoo/registry.json` (caminho resolvido a partir de
  `os.UserHomeDir()`, o mesmo em qualquer sistema operacional), resolvido
  por `internal/infra/outbound/config` (estendendo o `Config` já existente
  com um campo `RegistryPath`) e lido/escrito por um adapter dedicado
  (`internal/infra/outbound/geodatastore/jsonfile`). Toda escrita é atômica:
  o novo conteúdo é escrito em um arquivo temporário no mesmo diretório e
  então promovido com `os.Rename`, evitando um registro corrompido caso o
  processo seja interrompido no meio da escrita.
- **Racional**: atende ao FR-008 (persistente entre execuções, independente
  do diretório de onde a ferramenta é executada) sem exigir nenhum serviço
  externo (Princípio V da constituição). Um caminho fixo e único
  (`~/.sobrevoo`, mesmo padrão de ferramentas de linha de comando pessoais
  bem conhecidas como `~/.ssh`, `~/.aws`, `~/.kube`, `~/.docker`,
  `~/.cargo`) é mais fácil de encontrar, inspecionar e apagar manualmente
  do que um diretório de configuração cuja localização varia por SO — o que
  importa mais para o Sobrevoo hoje, uma ferramenta pessoal (CLAUDE.md),
  inclusive durante o próprio desenvolvimento/teste desta etapa (ver
  `quickstart.md`), do que aderir estritamente à convenção de cada
  plataforma. Caso o Sobrevoo um dia ganhe adoção em escala que justifique
  esse cuidado, migrar para `os.UserConfigDir()` (ou equivalente) fica para
  uma atualização futura dedicada a isso — não é uma decisão que precisa
  ser acertada de antemão nesta etapa. Um arquivo JSON simples é suficiente
  para o volume esperado (Suposições do spec: sem limite superior definido,
  mas escala pessoal — dezenas a poucas centenas de registros).
- **Alternativas consideradas**: `os.UserConfigDir()` (decisão original
  deste item, superada — mais "correta" por convenção de SO, mas o
  caminho final varia por plataforma, complicando instruções de teste e
  dificultando inspeção/backup/remoção manual por um único usuário pessoal,
  sem nenhum benefício real na escala atual da ferramenta); um banco de
  dados embutido (ex.: SQLite) para o próprio registro (rejeitado —
  complexidade desnecessária para uma lista pequena de registros, cada um
  com poucos campos); persistir no diretório de trabalho atual (rejeitada —
  violaria diretamente o FR-008, que exige que o registro valha
  independentemente do diretório de execução).

## 6. `list` e `check` nunca reabrem o arquivo de dados original

- **Decisão**: `ListGeoDataService` e `CheckCoverageService` usam apenas o
  tipo e a área geográfica já persistidos no registro, calculados uma única
  vez durante o `register`. Nenhum dos dois reabre nem reprocessa o
  conteúdo do arquivo MBTiles/GeoTIFF original. A única coisa reverificada
  a cada execução é a disponibilidade do arquivo (FR-010, FR-017), por uma
  checagem leve de existência (`os.Stat`), não uma releitura de conteúdo.
- **Racional**: mantém `list` e `check` rápidos mesmo com muitos registros
  ou arquivos de dados grandes (um MBTiles ou GeoTIFF pode ter vários
  gigabytes), e evita reintroduzir a complexidade de parsing de formato
  fora do fluxo de `register`. É consistente com o caso extremo já
  documentado no `spec.md` (a ferramenta não detecta automaticamente uma
  troca de conteúdo no mesmo caminho) — o tipo/área só é recalculado quando
  o usuário registra novamente.
- **Alternativas consideradas**: reinspecionar o conteúdo do arquivo a cada
  `list`/`check` (rejeitada — desnecessariamente lento e redundante, dado
  que o conteúdo de um arquivo de dados geográficos não muda organicamente
  entre execuções).

## 7. Checagem de disponibilidade de arquivo: porta `FileChecker` dedicada

- **Decisão**: uma nova porta pequena e de propósito único,
  `FileChecker.Exists(path string) bool`, implementada por um adapter
  simples baseado em `os.Stat`
  (`internal/infra/outbound/filechecker`). Usada tanto por
  `ListGeoDataService` (FR-010) quanto por `CheckCoverageService` (FR-017).
- **Racional**: mesmo espírito de `Simplifier`/`Smoother` na etapa 1— uma
  porta pequena, de único método, fácil de mockar — evitando duplicar essa
  checagem atrás de duas portas diferentes para os dois casos de uso que
  precisam exatamente da mesma informação.
- **Alternativas consideradas**: embutir a checagem de existência dentro do
  próprio `GeoDataRepository` (rejeitada — misturaria duas responsabilidades
  distintas — persistência do registro e verificação do sistema de
  arquivos — na mesma porta).

## 8. `GeoDataInspector` recebe um caminho, não um `io.Reader`

- **Decisão**: diferente de `TrackParser` (que recebe um `io.Reader` já
  aberto por quem chama), `GeoDataInspector.Inspect` recebe o caminho do
  arquivo (`string`) e é o próprio adapter que o abre.
- **Racional**: tanto o driver SQLite (MBTiles) quanto o leitor de tags
  TIFF (GeoTIFF) precisam de acesso posicional/aleatório ao arquivo
  (`io.ReaderAt`, ou um handle de banco de dados) — algo que um `io.Reader`
  simples não oferece sem primeiro carregar o arquivo inteiro em memória, o
  que seria custoso para arquivos de mapa/relevo potencialmente grandes
  (diferente do trajeto GPX da etapa 1, tipicamente pequeno). O adapter
  concreto (não o núcleo) é quem chama `os.Open`/abre o banco e traduz
  erros de sistema de arquivo para os sentinelas do domínio
  (`ErrDataFileNotFound`, `ErrDataFileUnreadable`) — o núcleo continua sem
  importar `os` ou qualquer pacote de infraestrutura (Princípios I e II
  preservados; é apenas uma escolha diferente de qual adapter concreto
  realiza a chamada de I/O real, dada a necessidade técnica destes formatos
  binários).
- **Alternativas consideradas**: carregar o arquivo inteiro em memória e
  expor um `io.Reader`/`bytes.Reader` (rejeitada — desnecessariamente
  custoso para arquivos de mapa/relevo grandes, quando só um cabeçalho ou
  uma pequena tabela de metadados precisa ser lida); exigir
  `io.ReaderAt` na porta em vez do caminho (rejeitada — ainda exigiria que
  o SQLite recebesse um caminho de arquivo real na prática, então não
  eliminaria a necessidade, apenas adicionaria indireção).

## 9. `CheckCoverageService` reaproveita o pipeline de limpeza da etapa 1

- **Decisão**: extrair a lógica de "ler o trajeto e limpá-lo" (parse via
  `TrackParser` + `ReorderByTime` + os três `Discard*`) de
  `InspectTrackService` para uma função interna não exportada em
  `internal/application/track_loading.go`, chamada tanto por
  `InspectTrackService` quanto pelo novo `CheckCoverageService`. A
  verificação de cobertura usa a rota **limpa, mas não simplificada nem
  suavizada**.
- **Racional**: evita duplicar uma regra de negócio já existente entre dois
  casos de uso (Princípio III). Simplificação (Douglas-Peucker) e
  suavização (Catmull-Rom) são etapas de preparação voltadas à renderização
  (fora do escopo desta etapa) que podem deslocar levemente pontos —
  suavização em particular interpola posições — o que poderia mascarar ou
  distorcer uma lacuna de cobertura real. Usar a rota limpa (sem esses dois
  passos) garante que a verificação reflita o trajeto GPS real do usuário.
- **Alternativas consideradas**: reusar a rota totalmente tratada
  (simplificada + suavizada) (rejeitada — pontos interpolados pela
  suavização poderiam cair dentro ou fora de uma bounding box de forma
  diferente do trajeto real, produzindo um veredito de cobertura
  tecnicamente incorreto); duplicar a lógica de limpeza dentro do novo
  serviço em vez de extraí-la (rejeitada — viola DRY e arrisca as duas
  cópias divergirem com futuras mudanças).
- **Atualização (item 15)**: o mecanismo de extração descrito acima —
  `internal/application/track_loading.go` — foi substituído por
  `domain.CleanTrack`, uma função pura de domínio. A decisão de **não**
  simplificar/suavizar a rota usada na verificação de cobertura, e o
  racional para isso, continuam valendo sem mudança nenhuma — só onde a
  composição "reorder + discard" mora é que mudou. Ver item 15 para o
  histórico completo.

## 10. Critério de desempate como área em graus (não área geodésica real)

- **Decisão**: a "especificidade" de uma fonte (Clarification, `spec.md`)
  é medida como a área da sua bounding box em graus quadrados (largura ×
  altura, com a mesma técnica de "unwrap" de longitude já usada para
  antimeridiano), não uma área geodésica real (que dependeria da
  latitude). Empate é resolvido por `RegisteredAt` mais antigo — um novo
  campo persistido no registro, preenchido com `time.Now()` no momento do
  `register`, chamado diretamente por `RegisterGeoDataService` (sem uma
  porta `Clock` — ver item 10.1 abaixo).
- **Racional**: a regra de desempate só precisa ser determinística e
  produzir uma ordenação relativa e explicável (Clarification do spec) —
  não uma medida de área fisicamente exata. Área em graus é suficiente
  para isso e reaproveita diretamente a mesma matemática já usada e testada
  em `ComputeBoundingBox`.
- **Alternativas consideradas**: área geodésica real via projeção
  (rejeitada — complexidade desnecessária para uma comparação puramente
  relativa/ordinal, sem nenhum requisito do spec exigindo precisão de área
  absoluta).

## 10.1. `RegisteredAt` via `time.Now()` direto, sem porta `Clock`

- **Decisão**: `RegisterGeoDataService` chama `time.Now()` diretamente para
  preencher `RegisteredAt`, em vez de receber esse valor de uma porta
  `Clock` dedicada. Decisão revisada em relação à primeira versão desta
  etapa, que introduziu `domain.Clock` + um adapter
  `internal/infra/outbound/clock` — ambos removidos.
- **Racional**: os "dependências externas" do Princípio II da constituição
  são explicitamente parser de arquivo, renderizador, filesystem, processo
  externo, browser — `time` é biblioteca padrão da linguagem, não um
  desses casos, e o núcleo já importa outras bibliotecas padrão puras
  diretamente (`math` em `bounding_box.go`, `sort` em `cleaning.go`) sem
  porta. `time.Now()` é impuro (não determinístico), mas `RegisteredAt` não
  é um dado de negócio exposto ao usuário — é usado apenas internamente
  para desempate — então a precisão exigida do teste é baixa: comparar
  `RegisteredAt` contra um intervalo `[antes, depois]` em torno da chamada
  (como o teste de `register_geo_data_service_test.go` já faz) é
  suficiente, sem precisar de um mock. O custo da porta (arquivo de porta,
  mock gerado, adapter, mais um parâmetro no construtor) não se pagava para
  esse ganho.
- **Alternativas consideradas**: manter a porta `Clock` (decisão original
  desta etapa, superada — ver Princípio II acima: não é uma dependência
  externa no sentido que a constituição usa o termo, e o ganho de precisão
  no teste não compensava a abstração extra para um campo que nunca é
  exibido ao usuário).

## 11. Granularidade do relatório de cobertura

- **Decisão**: implementação direta da Clarification já registrada em
  `spec.md`: cada ponto da rota limpa recebe um status de cobertura (total,
  falta mapa base, falta relevo, ou falta ambos); pontos consecutivos com o
  mesmo status formam um subtrecho, relatado pelas coordenadas do primeiro
  e do último ponto do subtrecho. O relatório também lista, separadamente,
  o conjunto de fontes de mapa base e de relevo que efetivamente cobriram
  pelo menos um ponto do trajeto (não uma atribuição fonte-por-subtrecho) —
  suficiente para satisfazer o FR-016 ("informar qual registro foi
  escolhido" quando há mais de uma opção) sem introduzir uma estrutura de
  relatório mais complexa do que o necessário.
- **Racional**: reaproveita a mesma estrutura de iteração sequencial já
  usada por `ComputeBoundingBox`/`ReorderByTime` no domínio. Reportar o
  conjunto de fontes usadas (em vez de uma atribuição detalhada por
  subtrecho) é suficiente para que o usuário entenda qual registro foi
  escolhido em caso de sobreposição, sem exigir uma estrutura de dados
  desproporcional ao valor entregue.
- **Alternativas consideradas**: atribuir explicitamente uma fonte a cada
  subtrecho coberto (rejeitada — informação adicional que a especificação
  não exige de forma explícita, e que tornaria a saída bem mais verbosa
  para trajetos com muitas fontes sobrepostas em pontos diferentes).

## 12. Comandos CLI agrupados sob `geodata`

- **Decisão**: um novo comando pai Cobra, `geodata`, agrupa quatro
  subcomandos — `register`, `list`, `remove`, `check` — em vez de quatro
  comandos soltos na raiz.
- **Racional**: mantém o espaço de nomes da CLI organizado por área de
  responsabilidade à medida que o Sobrevoo cresce (próximas etapas trarão
  câmera, renderização, vídeo), evitando colisão de nomes de comando de
  alto nível e deixando claro que os quatro comandos operam sobre o mesmo
  conceito (o registro de dados geográficos).
- **Alternativas consideradas**: quatro comandos de nível raiz (`register`,
  `list`, `remove`, `check-coverage`) (rejeitada — nomes genéricos demais
  no nível raiz, com risco de colisão com comandos de etapas futuras).

## 13. Um único `GeoDataService`, não um serviço por caso de uso

- **Decisão**: os quatro casos de uso desta etapa (`register`, `list`,
  `remove`, `check`) são expostos por uma única interface `GeoDataService`,
  com um método nomeado por operação (`Register`, `List`, `Remove`,
  `CheckCoverage`) — não quatro interfaces separadas de método único
  (`RegisterGeoDataService.Execute`, `ListGeoDataService.Execute`, etc.,
  como a primeira versão desta etapa implementou). Os helpers internos de
  `CheckCoverage` que precisam de estado do serviço
  (`partitionAvailableSources`, `buildCoverageOutput`) viraram métodos não
  exportados de `geoDataService`; os que são puramente algébricos, sem
  estado (`pickCoverageWinner`, `isMoreSpecific`, `missingDataType`,
  `sortedSources`), continuam funções livres no mesmo arquivo — o mesmo
  padrão já usado por funções puras do domínio (`Haversine`,
  `ComputeBoundingBox`).
- **Racional**: decisão revisada por pedido explícito do usuário, que
  apontou `waliqueiroz/mystery-gifter-api`
  (`internal/application/group_service.go`) como referência do padrão que
  já usa em outros projetos: um serviço por *recurso/agregado* (`Group`,
  `User`, `GroupInvite`), com métodos nomeados pela operação
  (`Create`, `GetByID`, `AddUser`, `Reopen`, ...), não um objeto por caso
  de uso com um único método `Execute`. Os quatro casos de uso desta etapa
  operam sobre o mesmo recurso — o registro de dados geográficos — então
  cabem naturalmente em um único `GeoDataService`, do mesmo jeito que
  `GroupService` reúne `Create`/`GetByID`/`Search`/`AddUser`/`RemoveUser`/
  `GenerateMatches`/`Reopen`/`Archive`/`GetUserMatch` numa só interface. Um
  serviço também pode depender de outro serviço (no repositório de
  referência, `GroupService` depende de `UserService`) — não só de portas
  do domínio — mas não foi necessário aqui, já que `GeoDataService` não
  precisa de nenhum outro serviço de aplicação.
- **Alternativas consideradas**: manter quatro interfaces de método único
  (decisão original desta etapa, superada — contraria o padrão de service
  layer que o usuário já usa e o Princípio IX, lido à luz dessa correção:
  "cada caso de uso" não significa "uma interface por método", significa
  "cada recurso/agregado da aplicação").

**Nota de amendment sugerida**: a redação atual do Princípio IX da
constituição ("Cada caso de uso em `internal/application`... MUST ser
modelado como uma interface exportada terminada em `Service`") é ambígua
o bastante para levar a exatamente o erro que esta decisão corrige — foi
seguida ao pé da letra na primeira versão desta etapa, produzindo quatro
serviços de método único. Vale abrir um amendment dedicado (via
`/speckit-constitution`) explicitando o padrão "um serviço por recurso,
métodos nomeados pela operação — não um objeto por caso de uso", com
`waliqueiroz/mystery-gifter-api` como referência, para que as próximas
etapas (câmera, renderização, vídeo) não repitam o mesmo engano.

## 14. DTOs e algoritmo de cobertura movidos para `internal/domain`

- **Decisão**: os tipos que antes eram DTOs de `internal/application`
  (`CheckCoverageOutput`, `CoverageStatus`, `MissingDataType`,
  `UncoveredSegment`, `GeoDataSummary`) viraram tipos de domínio comuns, em
  `internal/domain/geo_data_coverage.go` (o primeiro renomeado para
  `CoverageReport`) e `internal/domain/geo_data_source.go`
  (`GeoDataSummary`). O algoritmo de cobertura em si — escolher o vencedor
  entre candidatos sobrepostos, decidir o que falta em cada ponto, agrupar
  em subtrechos, decidir o veredito geral — virou uma função pura de
  domínio, `domain.ComputeCoverage(route []TrackPoint, baseMaps,
  elevations []GeoDataSource) CoverageReport`, com seus helpers privados
  (`pickCoverageWinner`, `isMoreSpecific`, `missingDataType`,
  `sortedSources`) como funções livres no mesmo arquivo — mesmo padrão já
  usado por `ComputeBoundingBox`/`Haversine`/`ReorderByTime` no domínio.
  `GeoDataSource` também ganhou um construtor, `domain.NewGeoDataSource(name,
  path string, inspected InspectedGeoData) GeoDataSource`, que monta a
  entidade (incluindo `RegisteredAt: time.Now()`) — antes montada como
  literal de struct dentro do serviço de aplicação.
  `internal/application/geo_data_service.go` ficou só com orquestração:
  cada método chama as portas (`GeoDataRepository`, `GeoDataInspector`,
  `FileChecker`, `TrackParser`) e delega a regra de negócio para o
  construtor/função de domínio correspondente — nunca decide nada sozinho
  além de qual porta chamar em qual ordem.
- **Racional**: decisão revisada por pedido explícito do usuário, apontando
  novamente `waliqueiroz/mystery-gifter-api` como referência — lá, DTOs de
  saída (`GroupSummary`, `SearchResult[T]`) e construtores com regra de
  negócio (`NewGroup` já monta `CreatedAt`/`UpdatedAt: time.Now()`) vivem
  em `internal/domain` junto das entidades, como qualquer outro objeto de
  domínio; a camada de serviço (`group_service.go`) só orquestra — busca
  via repositório, delega a regra para um método/construtor de domínio,
  salva, devolve. Isso já era exatamente o estilo das funções puras que o
  domínio deste projeto usa desde a etapa 1 (`ComputeBoundingBox`,
  `Haversine`, `ReorderByTime`, os `Discard*`) — a etapa 2 só não tinha
  seguido esse mesmo padrão para as regras novas (cobertura, construção de
  `GeoDataSource`), deixando-as na camada de aplicação por engano. Mover
  para o domínio também tem um efeito prático nos testes: os cenários de
  cobertura (sobreposição, desempate, antimeridiano, segmentos) agora são
  testados como função pura em `internal/domain/geo_data_coverage_test.go`,
  sem nenhum mock — só `internal/application/geo_data_service_test.go`
  ficou com testes de orquestração (propagação de erro de cada porta,
  filtragem de fontes indisponíveis antes de chamar `ComputeCoverage`).
- **Alternativas consideradas**: manter DTOs e algoritmo em
  `internal/application` (decisão original desta etapa, superada — mistura
  regra de negócio com orquestração na mesma camada, e contraria o padrão
  já usado tanto pelo domínio deste projeto quanto pelo repositório de
  referência do usuário).

## 15. `InspectTrackService` (etapa 1) alinhado ao mesmo padrão; `track_loading.go` removido

- **Decisão**: por pedido explícito do usuário, o mesmo tratamento do item
  14 foi aplicado a `InspectTrackService`, da etapa 1 — não só ao que esta
  etapa introduziu:
  - `InspectTrackInput`/`InspectTrackOutput` deixam de existir.
    `Inspect` passa a receber argumentos simples
    (`reader io.Reader, simplificationLevel, smoothingLevel domain.Level`)
    em vez de um DTO de entrada — mesmo estilo de `GeoDataService.Register(name,
    path string)` e de `Create(ctx, name, description, ownerID string)` em
    `waliqueiroz/mystery-gifter-api`. `InspectTrackOutput` vira
    `domain.TrackSummary`, construído por uma nova função pura de domínio,
    `domain.SummarizeTrack(track, route, discarded)`, em
    `internal/domain/track_summary.go` — a mesma lógica que antes era o
    método privado `buildOutput` do serviço.
  - `internal/application/track_loading.go` (o helper `cleanTrack`,
    compartilhado por `InspectTrackService` e `GeoDataService.CheckCoverage`)
    é removido inteiramente. A parte que chamava a porta `TrackParser`
    permanece em cada serviço (é orquestração de verdade — chama uma porta);
    a parte que **compunha** `ReorderByTime` + os três `Discard*` + a
    checagem de mínimo de pontos (antes e depois) — ou seja, a regra "o que
    significa limpar um trajeto" — virou uma função pura de domínio nova,
    `domain.CleanTrack(points, minPoints, maxPlausibleSpeedKmh)`, em
    `internal/domain/cleaning.go`, ao lado das funções que ela já compõe
    (`ReorderByTime`, `DiscardImpossibleCoordinates`, ...). Cada serviço
    agora só faz `parser.Parse(reader)` seguido de `domain.CleanTrack(...)`
    — duas linhas, sem precisar de um helper compartilhado.
  - O builder de teste correspondente também migrou:
    `internal/application/build_application/inspect_track_output_builder.go`
    virou `internal/domain/build_domain/track_summary_builder.go`
    (`TrackSummaryBuilder`), e o pacote `build_application` — que não tinha
    mais nenhum outro arquivo — deixou de existir.
- **Racional**: a pergunta do usuário ("o `track_loading` precisa ficar na
  camada de aplicação mesmo?") aponta exatamente a distinção que já valia
  para `ComputeCoverage`/`NewGeoDataSource` (item 14): `cleanTrack` como um
  todo *parecia* orquestração só porque chamava `parser.Parse` no início,
  mas a maior parte do seu corpo — a sequência reorder→discard→discard→discard
  e a semântica dos dois erros de "pontos insuficientes" — é regra de
  negócio pura sobre `[]TrackPoint`, sem nenhuma porta envolvida. Separar
  as duas coisas deixa cada uma no lugar certo: a chamada de porta
  (`parser.Parse`) é orquestração e fica no serviço; a composição de regras
  de limpeza (`domain.CleanTrack`) é domínio e fica com as funções que ela
  já usa. Isso também elimina o único "helper solto" que ainda restava em
  `internal/application` depois do item 14.
- **Alternativas consideradas**: manter `cleanTrack` como estava, um helper
  de pacote em `internal/application` chamado por ambos os serviços
  (decisão original, superada — exatamente o tipo de função solta na
  camada de aplicação que o usuário pediu para eliminar; também escondia
  regra de negócio real, não só orquestração, atrás de uma assinatura que
  recebia uma porta); manter `cleanTrack` completo (parse + limpeza) como
  função de domínio, recebendo `TrackParser` como parâmetro (rejeitada —
  domínio nunca recebe nem chama uma porta; isso é o próprio papel da
  camada de aplicação, igual a nenhuma entidade/função de domínio em
  `waliqueiroz/mystery-gifter-api` receber um repositório como argumento).

## 16. `GeoDataRegistry` renomeada para `GeoDataRepository`

- **Decisão**: a porta de persistência do registro de dados geográficos,
  declarada em `internal/domain/geo_data_source.go`, passou a se chamar
  `GeoDataRepository` (era `GeoDataRegistry`). Acompanham a mudança: o mock
  gerado (`mock_domain/geo_data_repository.go`, era `geo_data_registry.go`,
  com `MockGeoDataRepository`/`NewMockGeoDataRepository`), o campo/parâmetro
  correspondente em `geoDataService`/`NewGeoDataService`
  (`internal/application/geo_data_service.go`, era `registry`, agora
  `repository`) e a variável local equivalente em `cmd/sobrevoo/main.go`
  (era `geoDataRegistry`, agora `geoDataRepository`). Não muda: o nome do
  pacote adapter (`internal/infra/outbound/geodatastore/jsonfile`, já
  neutro), o struct `Store` que o implementa, o caminho do arquivo
  persistido (`~/.sobrevoo/registry.json`, campo `Config.RegistryPath`), o
  texto de ajuda da CLI ("local geographic data registry") e o nome desta
  feature (`002-geo-data-registry`) — todos esses continuam descrevendo
  "registro" como conceito de domínio (o conjunto de fontes registradas, e
  o arquivo que o guarda), não o nome do tipo Go da porta.
- **Racional**: pergunta direta do usuário ("dá pra gente passar a chamar
  registry de repository porque envolve persistência e fica mais fácil de
  ler?"). `GeoDataRepository` nomeia a porta pelo que ela é
  arquiteturalmente — uma porta de persistência — o mesmo papel de
  `GroupRepository`/`UserRepository` em
  `waliqueiroz/mystery-gifter-api` (`internal/domain/group.go`,
  `internal/domain/user.go`), incluindo o nome do campo correspondente em
  `groupService` (`groupRepository domain.GroupRepository`, em
  `internal/application/group_service.go`) — mesma convenção que este
  projeto já vem espelhando desde os itens 13 e 14. "Registry" descrevia
  bem o dado em si (um registro de fontes), mas não o papel arquitetural da
  porta que o acessa; "Repository" é o termo já consagrado nesse papel,
  inclusive no repositório de referência do próprio usuário.
- **Alternativas consideradas**: manter `GeoDataRegistry` (decisão original,
  superada — funcionava, mas divergia sem necessidade do vocabulário que o
  resto do projeto usa para portas de persistência); renomear também o
  conceito de "registro" em si (pacote `geodatastore`, `RegistryPath`,
  `registry.json`, texto de ajuda da CLI, nome da feature) — rejeitada: o
  pedido do usuário era especificamente sobre o nome do tipo/porta, não
  sobre o vocabulário do domínio, e "registro"/"registry" continuam nomes
  corretos para o dado persistido e para a feature, independente de como a
  porta que o acessa se chama.
