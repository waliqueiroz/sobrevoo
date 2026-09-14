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
  próprio `GeoDataRegistry` (rejeitada — misturaria duas responsabilidades
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

## 10. Critério de desempate como área em graus (não área geodésica real)

- **Decisão**: a "especificidade" de uma fonte (Clarification, `spec.md`)
  é medida como a área da sua bounding box em graus quadrados (largura ×
  altura, com a mesma técnica de "unwrap" de longitude já usada para
  antimeridiano), não uma área geodésica real (que dependeria da
  latitude). Empate é resolvido por `RegisteredAt` mais antigo — um novo
  campo persistido no registro, preenchido no momento do `register` a
  partir de uma nova porta `Clock` (ver `data-model.md`).
- **Racional**: a regra de desempate só precisa ser determinística e
  produzir uma ordenação relativa e explicável (Clarification do spec) —
  não uma medida de área fisicamente exata. Área em graus é suficiente
  para isso e reaproveita diretamente a mesma matemática já usada e testada
  em `ComputeBoundingBox`.
- **Alternativas consideradas**: área geodésica real via projeção
  (rejeitada — complexidade desnecessária para uma comparação puramente
  relativa/ordinal, sem nenhum requisito do spec exigindo precisão de área
  absoluta).

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
