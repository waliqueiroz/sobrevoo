# Pesquisa: Leitura e Tratamento de Trajeto GPS

**Feature**: `001-gps-track-processing` | **Data**: 2026-09-13

Este documento consolida as decisões técnicas necessárias para resolver os pontos
em aberto do Contexto Técnico do `plan.md`, todas derivadas da entrada fornecida
pelo usuário e da especificação já clarificada.

## 1. Formato suportado: apenas GPX

- **Decisão**: o único formato reconhecido nesta etapa é GPX, conforme FR-003
  da especificação.
- **Racional**: a entrada original do plano mencionava FIT (binário) no lugar
  do TCX que havia sido clarificado na especificação, e uma decisão
  intermediária optou por manter GPX + TCX. Posteriormente, o usuário decidiu
  simplificar ainda mais a implementação desta primeira etapa, removendo
  também o TCX e restringindo o escopo a um único formato. Isso reduz a
  superfície de código e de testes desta etapa (um só adapter de parsing) sem
  perder a extensibilidade: a porta `TrackParser` já isola o núcleo de
  qualquer formato concreto, então adicionar TCX, FIT ou outro formato depois
  é uma questão de escrever um novo adapter, sem tocar em domínio ou
  aplicação (Princípio II da constituição).
- **Alternativas consideradas**: GPX + TCX (decisão intermediária anterior,
  superada pela simplificação pedida pelo usuário); GPX + FIT (rejeitada
  desde a primeira rodada — contrariava o FR-003 já clarificado); suportar
  três formatos de uma vez (rejeitada — não agrega valor a esta primeira
  etapa e adia sem necessidade a entrega do núcleo de tratamento de trajeto).

## 2. Parser de GPX

- **Decisão**: usar `github.com/tkrajina/gpxgo` (conforme solicitado pelo
  usuário) em um único adapter `internal/infra/outbound/trackparser`, que
  também é responsável por validar que o conteúdo é um GPX reconhecível
  (FR-002, FR-004) antes de delegar o parsing propriamente dito à
  biblioteca.
- **Racional**: biblioteca madura e amplamente usada para leitura de GPX em
  Go; evita reescrever um parser XML completo do zero para um formato com
  várias variações de schema entre fabricantes. Com um único formato
  suportado, não há mais necessidade de um adapter "composto" para detectar e
  delegar entre múltiplos parsers — a validação de conteúdo e o parsing GPX
  cabem no mesmo adapter, simplificando a estrutura de pastas.
- **Alternativas consideradas**: implementar parsing manual via
  `encoding/xml` (rejeitada — GPX tem extensões de schema entre fabricantes
  que a lib já trata; reescrever isso do zero não agrega valor nesta etapa);
  manter um adapter "composto" separado do adapter GPX mesmo com um único
  formato (rejeitada — seria uma camada extra sem propósito enquanto houver
  apenas um formato suportado; pode ser reintroduzida se um segundo formato
  for adicionado no futuro).

## 3. Detecção de formato pelo conteúdo (FR-002)

- **Decisão**: ler o arquivo inteiro para `[]byte` uma única vez; usar
  `encoding/xml.Decoder` para avançar até o primeiro `xml.StartElement` e
  verificar se seu nome local é `gpx`. Se não for (ou o conteúdo não for XML
  válido), retornar `ErrUnsupportedFormat` sem chamar `gpxgo`. Se for,
  delegar o parsing completo a `gpxgo` sobre um novo `bytes.Reader` do mesmo
  conteúdo.
- **Racional**: evita exigir `io.Seeker` do chamador (a porta `TrackParser`
  recebe apenas `io.Reader`), e produz uma mensagem de erro clara
  (`ErrUnsupportedFormat`) para conteúdo que não é GPX, em vez de depender do
  erro genérico que `gpxgo` devolveria para XML inesperado. Ler o arquivo
  inteiro em memória é aceitável dado o volume de dados esperado (trajetos de
  atividades individuais, na casa de poucos milhares a dezenas de milhares de
  pontos — ver SC-005).
- **Alternativas consideradas**: exigir `io.ReadSeeker` na porta (rejeitada —
  acoplaria a porta a uma capacidade que nem toda fonte de dados possui, como
  um stream de rede em uma futura API HTTP); repassar diretamente para
  `gpxgo` e traduzir qualquer erro de parsing em `ErrUnsupportedFormat`
  (rejeitada — misturaria "conteúdo não é GPX" com "GPX malformado", casos que
  vale distinguir na mensagem ao usuário, conforme FR-007); detectar pelo
  início bruto da string (ex.: `strings.Contains`) sem parsing real
  (rejeitada — mais frágil a variações de encoding/BOM/formatação).

## 4. Simplificação (Douglas-Peucker) e suavização (Catmull-Rom)

- **Decisão**: implementar os dois algoritmos diretamente nos adapters
  `internal/infra/outbound/simplifier` (Douglas-Peucker) e
  `internal/infra/outbound/smoother` (Catmull-Rom), como código Go próprio, sem
  depender de uma biblioteca de terceiros. O domínio declara as portas
  `Simplifier` e `Smoother` (Princípio II da constituição); os adapters as
  implementam.
- **Racional**: bibliotecas geométricas genéricas em Go (ex.: `paulmach/orb`)
  operam sobre pares de coordenadas simples e descartam pontos sem preservar
  metadados associados (altitude, instante de tempo) de forma direta — mapear
  o resultado de volta aos pontos originais exigiria correlação frágil por
  coordenada, arriscando ambiguidade quando coordenadas se repetem. Douglas-
  Peucker e Catmull-Rom são algoritmos bem documentados e de escopo pequeno o
  suficiente para implementar e testar diretamente no adapter, mantendo o
  controle total sobre como os metadados de cada ponto atravessam o tratamento.
- **Alternativas consideradas**: usar `paulmach/orb/simplify` para Douglas-
  Peucker operando só em latitude/longitude e depois tentar re-associar
  altitude/tempo por índice (rejeitada — funcionaria, mas a lib não expõe os
  índices dos pontos mantidos, apenas a polyline simplificada); buscar uma lib
  dedicada de spline Catmull-Rom (rejeitada — nenhuma opção amplamente adotada
  foi encontrada para esse caso de uso específico).

## 5. Distância, ganho de elevação e bounding box

- **Decisão**: implementar como funções puras no pacote `internal/domain`,
  usando apenas a biblioteca padrão (`math`):
  - Distância: fórmula de Haversine ponto a ponto, somada ao longo da rota.
  - Ganho de elevação: soma de todos os deltas positivos de altitude entre
    pontos consecutivos que possuem altitude.
  - Bounding box: ver decisão dedicada abaixo (item 6).
- **Racional**: são cálculos matemáticos autocontidos, sem necessidade de
  biblioteca externa; mantê-los como funções puras no domínio maximiza a
  testabilidade (Princípio VI) e elimina qualquer dependência a mockar para
  testá-los.
- **Alternativas consideradas**: usar uma biblioteca de terceiros para
  Haversine (ex.: `umahmood/haversine`) (rejeitada — a fórmula é trivial de
  implementar corretamente e testar, e evita uma dependência desnecessária no
  núcleo).

## 6. Bounding box atravessando o antimeridiano

- **Decisão**: calcular a bounding box em duas etapas:
  1. Latitude: mínimo e máximo diretos (o equador nunca exige tratamento
     especial, já que a latitude não "dá a volta").
  2. Longitude: "desembrulhar" (unwrap) a sequência de longitudes — ao
     percorrer os pontos em ordem, sempre que a diferença entre a longitude
     atual e a anterior for maior que 180° em valor absoluto, somar ou
     subtrair 360° para manter a sequência contínua. Calcular mínimo e máximo
     sobre a sequência desembrulhada e então normalizar o resultado de volta
     para o intervalo [-180°, 180°], marcando a bounding box como
     "atravessa o antimeridiano" quando o mínimo desembrulhado for menor que
     -180° ou o máximo maior que 180°.
- **Racional**: é a técnica padrão para lidar com bounding boxes geográficas
  sem assumir hemisfério ou região (Princípio IV da constituição) — funciona
  identicamente para qualquer trajeto do planeta, incluindo os que cruzam a
  linha de 180°.
- **Alternativas consideradas**: pedir ao usuário informação de fuso/região
  (rejeitada — viola FR-023 e o Princípio IV, que proíbem exigir esse tipo de
  informação do usuário); ignorar o problema e calcular min/max diretamente
  sobre longitudes brutas (rejeitada — produz uma bounding box incorreta,
  cobrindo o planeta inteiro, para qualquer trajeto que cruze o antimeridiano).

## 7. Limite de salto fisicamente implausível

- **Decisão**: definir uma constante interna de velocidade máxima plausível de
  130 km/h entre dois pontos consecutivos, usada para descartar saltos
  implausíveis (FR-010). Não é exposta como flag de CLI (conforme suposição já
  registrada na especificação).
- **Racional**: cobre com folga as velocidades de corrida, pedalada e
  caminhada, incluindo descidas acentuadas de bicicleta, sem ser tão alta a
  ponto de deixar passar erros grosseiros de GPS (teleporte por perda de
  sinal).
- **Alternativas consideradas**: um limite mais baixo (ex.: 60 km/h) (rejeitada
  — descartaria descidas de bicicleta legítimas); tornar o limite configurável
  nesta etapa (rejeitada — fora do escopo definido na especificação, que
  reserva o ajuste do usuário apenas aos níveis de simplificação e suavização).

## 8. Reordenação por tempo com dados parciais (FR-027)

- **Decisão**: a reordenação por instante de tempo só é aplicada quando **todos**
  os pontos do trajeto possuem timestamp. Se apenas parte dos pontos tiver
  timestamp, o trajeto é tratado como se não tivesse nenhum dado de tempo
  (duração indisponível, conforme FR-021), sem tentativa de reordenação
  parcial.
- **Racional**: uma ordenação parcial poderia intercalar de forma arbitrária
  pontos com e sem tempo, produzindo um traçado sem sentido físico. Tratar
  timestamps parciais como "sem dado de tempo" é uma extensão direta e segura
  da regra já definida em FR-021 para ausência total de tempo.
- **Alternativas consideradas**: manter a ordem original quando há timestamps
  parciais, mas ainda assim tentar calcular duração parcial (rejeitada —
  contraria o espírito de FR-021, que exige clareza total sobre a ausência do
  dado, não um resultado parcial e potencialmente enganoso).

## 9. Configuração interna (thresholds) via adapter dedicado

- **Decisão**: os valores internos não expostos como flag (mínimo de 2 pontos,
  130 km/h de salto máximo, nível padrão "médio") vivem em um `Config`
  resolvido pelo adapter `internal/infra/outbound/config`, que por enquanto
  apenas retorna esses valores fixos, mas já isola o ponto de leitura de
  configuração para uma futura extensão (arquivo de config, variável de
  ambiente) sem exigir mudança na assinatura do caso de uso.
- **Racional**: atende ao Princípio VIII (configuração injetada a partir do
  adapter, nunca lida diretamente pelo núcleo) mesmo quando, nesta etapa, não
  há nenhuma fonte de configuração externa real — o núcleo já nasce
  desacoplado da forma como esses valores chegam até ele.
- **Alternativas consideradas**: declarar esses valores como constantes
  exportadas diretamente no domínio (rejeitada — funcionaria hoje, mas
  amarraria o núcleo à sua própria configuração, dificultando a evolução
  futura sem violar o Princípio VIII).

## 10. Organização das portas no domínio e mocks

- **Decisão**: não existe um arquivo genérico `ports.go`. Cada porta é
  declarada no arquivo mais específico possível: `TrackParser` — que produz
  `Track` — fica em `track.go`, junto da entidade que ela manipula.
  `Simplifier` e `Smoother` não pertencem a uma única entidade (operam sobre
  `[]TrackPoint` como parte do pipeline de tratamento), então cada uma ganha
  seu próprio arquivo nomeado pelo conceito que representa:
  `simplification.go` e `smoothing.go`. Os mocks são gerados com
  `go.uber.org/mock/mockgen` via diretivas `//go:generate` posicionadas
  diretamente acima de cada interface, com saída em
  `internal/domain/mock_domain/` — um arquivo por porta (ex.:
  `mock_domain/track_parser.go`).
- **Racional**: replica a convenção já usada pelo usuário em
  `waliqueiroz/mystery-gifter-api` (ex.: `GroupRepository` dentro de
  `group.go`; `IdentityGenerator` em seu próprio `identity.go`;
  `PasswordManager`/`AuthTokenManager` agrupados em `security.go` por serem
  conceitos de segurança sem entidade própria; mocks em `mock_domain/`).
  Evita nomes de arquivo genéricos (`ports.go`, `interfaces.go`) que não dizem
  nada sobre o que contêm.
- **Alternativas consideradas**: um único `ports.go` com todas as interfaces
  do domínio (rejeitada — nome genérico, exatamente o que se quer evitar);
  mocks escritos manualmente (rejeitada — mais trabalho de manutenção e maior
  risco de divergência da interface real).

## 11. Idioma de todo o I/O em tempo de execução da CLI

- **Decisão**: todo o I/O da aplicação em tempo de execução é em inglês — não
  só os nomes das flags (`--simplification`, `--smoothing`), mas também os
  valores aceitos para nível (`low`, `medium`, `high`), todo o texto do
  resumo impresso em `stdout`, e todas as mensagens de erro impressas em
  `stderr`.
- **Racional**: decisão explícita do usuário, que substitui uma decisão
  anterior deste documento (valores de nível e mensagens em português). Manter
  todo o I/O em um único idioma, consistente com a convenção usual de
  ferramentas de linha de comando em Go, evita misturar inglês (nomes de
  flag) com português (valores e mensagens) na mesma interação. Não viola a
  política de idioma da constituição, que trata de código, identificadores e
  artefatos de especificação — não do texto voltado ao usuário final em tempo
  de execução; a especificação (`spec.md`) documenta essa decisão em sua nota
  de escopo, tratando os tokens `low`/`medium`/`high` como valores literais da
  interface (equivalentes a um identificador), não como prosa a traduzir.
- **Alternativas consideradas**: valores de nível e mensagens em português com
  nomes de flag em inglês (decisão anterior deste documento, superada —
  rejeitada por misturar os dois idiomas na mesma interação); tudo em
  português, incluindo os nomes das flags (rejeitada — foge da convenção
  usual de ferramentas de linha de comando em Go, que usa nomes de flag em
  inglês mesmo em ferramentas voltadas a público não anglófono).

## 12. Testes do comando Cobra

- **Decisão**: testar o comando `inspect` via `cmd.SetArgs(...)`,
  `cmd.SetOut(&buf)` e `cmd.Execute()`, fornecendo arquivos de fixture GPX
  (válidos e inválidos) via `test/helper`, e comparando a saída de texto e o
  erro/código de saída resultante.
- **Racional**: é o padrão idiomático de teste de comandos Cobra, evita
  precisar rodar o binário compilado, e mantém o teste rápido e determinístico
  (sem tocar em stdin/stdout reais).
- **Alternativas consideradas**: testar via `os/exec` rodando o binário
  compilado (rejeitada — mais lento e desnecessário para validar a lógica de
  conversão flags → input e formatação de saída, que é o único papel deste
  adapter).
