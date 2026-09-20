# Pesquisa: Planejamento do Movimento de Câmera

**Feature**: `003-camera-path-planning` | **Data**: 2026-09-19

Este documento consolida as decisões técnicas que a especificação deixou
deliberadamente em aberto (limites numéricos de suavidade, compressão de
paradas, fração de abertura/fechamento, formato de exportação — ver a seção
"Adiado" do relatório do `/speckit-clarify`) e as escolhas de algoritmo
necessárias para cumprir FR-006 a FR-015. Todos os valores numéricos abaixo
são **valores iniciais**, concentrados em uma única estrutura de
configuração (`domain.CameraTuning`, item 12), pensados para serem
ajustados durante a implementação sem alterar nenhum contrato.

Nenhum item exige biblioteca nova: tudo é matemática sobre `math` da
biblioteca padrão e `encoding/json`.

## 1. Modelo de câmera: alvo + deslocamento (orbital)

- **Decisão**: a pose da câmera em cada quadro é uma função de cinco
  grandezas: o **alvo** `T` (ponto no plano local que a câmera olha), o
  **rumo** `ψ` (direção horizontal para a qual a câmera aponta), a
  **inclinação** `θ` (ângulo de depressão abaixo do horizonte: 0° =
  horizontal, 90° = vertical para baixo), a **distância** `D` (distância em
  linha reta da câmera ao alvo) e a posição do marcador. A câmera fica
  *atrás* do alvo, em relação ao rumo:
  `posição_horizontal = T − D·cos θ·(sen ψ, cos ψ)` e
  `altitude = D·sen θ`.
- **Racional**: as três coisas que o usuário controla ou vê (distância,
  inclinação, direção) viram parâmetros diretos e independentes. Suavizar
  `ψ`, `θ`, `ln D` e `T` separadamente garante que a pose resultante seja
  suave por composição, sem precisar suavizar posições 3D e reconstruir
  direção depois (o que produz artefatos em curvas fechadas).
- **Altitude**: é altura *relativa ao ponto observado* (o chão sob o alvo),
  não elevação sobre o nível do mar, pois esta etapa não conhece relevo
  (spec: Suposições; FR-024). O campo é exportado como `altitude_m` com essa
  semântica documentada em `contracts/plan-file.md`.
- **Alternativas rejeitadas**: câmera que acompanha a posição por spline 3D
  (dificulta controlar distância/inclinação como o usuário pediu);
  keyframes manuais (fora de escopo).

## 2. Plano local em vez de latitude/longitude cruas (antimeridiano, latitudes altas)

- **Decisão**: todo o cálculo geométrico acontece num **plano tangente local**
  usando projeção **azimutal equidistante** centrada no centroide do
  trajeto. O centroide é a média dos vetores unitários 3D dos pontos
  (normalizada de volta para latitude/longitude). Cada ponto vira `(x leste,
  y norte)` em metros; ao final, cada pose da câmera volta por projeção
  inversa para latitude/longitude, com longitude normalizada em
  `[-180, 180)`.
- **Racional**: o centroide por vetor unitário não tem descontinuidade no
  meridiano de 180° nem colapso perto dos polos (uma média aritmética de
  longitudes falha nos dois casos). A azimutal equidistante preserva
  distâncias e rumos a partir do centro, então distância, ângulos e
  suavização são idênticos em qualquer lugar do planeta (FR-015, SC-007) —
  a mesma ideia do "unwrap" de longitude já usado por `ComputeBoundingBox`
  na etapa 1, agora estendida a duas dimensões. Nenhuma região recebe
  tratamento especial (Princípio IV).
- **Limite**: a projeção se degrada acima de alguns milhares de
  quilômetros de abrangência. A **abrangência** do trajeto é o dobro da
  maior distância do centro da projeção a um ponto do trajeto tratado
  (aproximadamente o maior afastamento entre dois pontos); o **comprimento**
  é a distância percorrida (soma dos segmentos). A abrangência máxima é
  2 000 km (`MaxTrackSpanMeters`, item 12), acima da qual o plano é recusado
  com `ErrTrackTooLarge` (FR-017b).
- **Alternativas rejeitadas**: equiretangular local com
  `cos(latitude)` (falha em latitudes altas e exige tratar o antimeridiano
  à parte); trabalhar em ECEF 3D (correto, mas transforma a suavização de
  ângulos em geometria esférica sem ganho para trajetos de até 2 000 km).

## 3. Linha do tempo do marcador (proporcionalidade e paradas longas)

- **Decisão**: o avanço do marcador é uma função `s(t_vídeo)` — distância
  acumulada ao longo do trajeto tratado — construída assim:
  1. **Referência de tempo**: usa horário se *todos* os pontos têm horário,
     a duração total é positiva e nenhum segmento tem `dt ≤ 0` com
     deslocamento maior que 1 m. Caso contrário, usa **distância percorrida**
     (FR-011), e o resumo registra qual referência foi usada e, quando
     houve recuo para distância, o motivo.
  2. **Tempo efetivo**: cada segmento com horário tem `dt` real. Um
     segmento é *parado* se sua velocidade média fica abaixo de 0,5 m/s.
     Uma sequência contínua de segmentos parados com duração real acima de
     30 s é uma **parada longa**; seu `dt` efetivo passa a ser
     `min(dt real, 2 s, 5% do tempo em movimento)`. Paradas curtas (≤ 30 s)
     não são alteradas.
  3. **Mapeamento linear**: o tempo efetivo total é distribuído
     linearmente sobre a fase de acompanhamento; assim a razão vídeo/real é
     **constante** em todo o trajeto, exceto nas paradas longas (FR-009,
     FR-010, SC-005). Sem horário, o "tempo efetivo" de cada segmento é o
     seu comprimento.
  4. `s(t)` é interpolada linearmente entre pontos: monotônica
     não decrescente por construção (FR-013).
- **Racional**: comprimir só o `dt` (e não descartar pontos) mantém o
  marcador sem saltos durante a parada — ele apenas atravessa o trecho
  parado mais rápido. O teto de 5% torna SC-005 verdadeiro por construção.
- **Ritmo pelos pontos limpos, geometria pela rota tratada**: a simplificação
  da etapa 1 (Douglas-Peucker) funde uma parada com o trecho seguinte num
  só segmento cuja velocidade média fica acima do limiar — descoberto ao
  rodar o quickstart com uma parada real de 10 min, que passava despercebida.
  Por isso `TreatedTrack` carrega também os `CleanedPoints`, e a linha do
  tempo (referência de tempo, paradas, ritmo) é construída sobre eles; a
  câmera continua seguindo a rota simplificada e suavizada. A posição do
  marcador é a fração do comprimento da rota tratada correspondente à fração
  do percurso limpo (`distância_limpa × comprimento_rota / comprimento_limpo`).
  O jitter de GPS dentro de uma parada longa não conta como distância
  percorrida na linha do tempo, senão as poucas frames restantes da parada
  fariam o marcador "pular" dezenas de metros.
- **Alternativas rejeitadas**: velocidade constante do marcador ignorando o
  horário (contraria FR-009); descartar pontos parados (cria salto).

## 4. Direção da câmera: janela de tangentes, sem instabilidade em retornos e voltas

- **Decisão**: o rumo desejado em cada quadro é a direção da **soma
  ponderada dos vetores tangentes** do trajeto numa janela de arco
  centrada no marcador, com pesos gaussianos e largura proporcional à
  distância `D` (janela `L = 2·D` de comprimento de arco). Se a norma da
  soma cai abaixo de um limiar (retorno em 180° ou volta fechada menor que
  a janela), o rumo **mantém o valor do quadro anterior**. Os ângulos são
  desenrolados (*unwrap*) pelo menor arco, com desempate determinístico
  no sentido anti-horário para diferenças de exatamente 180°.
- **Racional**: a janela é escalonada por `D` (não fixa em metros), então o
  comportamento é invariante à escala do trajeto: voltas menores que a
  câmera "enxerga" são promediadas e a câmera não gira a cada volta
  (FR-007). Em um retorno pelo mesmo caminho, os vetores se cancelam e a
  câmera gira gradualmente à taxa máxima (item 5) em vez de inverter num
  quadro.
- **Alternativas rejeitadas**: rumo do segmento atual (ruidoso e com
  saltos em cada vértice); filtro passa-baixa apenas em ângulos
  desenrolados (não resolve o cancelamento em retornos).

## 5. Limites de suavidade: definição, aplicação e registro dos trechos suavizados

- **Decisão**: a suavidade é medida **por segundo de vídeo** e em unidades
  **relativas** (independentes da escala do trajeto), sobre cada trilha
  paramétrica de câmera:

  | Grandeza | Limite inicial |
  |---|---|
  | Taxa do rumo `ψ` | 45 °/s |
  | Taxa da inclinação `θ` | 30 °/s |
  | Taxa de `ln D` (zoom relativo) | 1,5 /s |
  | Velocidade do alvo `T` relativa a `D` | 1,0 (D por segundo) |

  Aplicação, em ordem, para cada trilha: (a) passe direto e depois
  passe inverso de *clamp* de inclinação máxima por quadro (`limite/fps`);
  (b) suavização gaussiana de σ ≈ 0,5 s (a convolução de um sinal
  Lipschitz com um kernel normalizado preserva o limite, então o passo (b)
  nunca o viola). O passe inverso por último garante o limite em todos os
  pares de quadros consecutivos, sem lag acumulado.
- **Trecho suavizado**: sequência contínua e máxima de quadros em que o
  *clamp* alterou o valor desejado em mais que uma tolerância (0,05° para
  ângulos; 0,001 para `ln D`; 1% para a velocidade relativa do alvo).
  Cada trecho é registrado com início e fim em segundos de vídeo, e a
  grandeza limitada (FR-008, FR-019). A suavização gaussiana comum *não*
  conta como trecho suavizado — só a limitação forçada.
- **Racional**: limites relativos tornam SC-002 e SC-007 verificáveis igual
  para 500 m ou 500 km. Um teste de propriedade pode varrer todos os pares
  de quadros consecutivos e checar os quatro limites — o critério de
  aceite de FR-006 é literalmente executável.
- **Alternativas rejeitadas**: limites absolutos em metros (só funcionam
  para uma faixa de tamanhos de trajeto); filtro de Kalman (estado
  desnecessário, mais difícil de tornar bit a bit determinístico).

## 6. Distância e inclinação por níveis

- **Decisão**: valores iniciais.

  | Nível | Distância base `D₀` | Antecipação `T_a` | Inclinação `θ` |
  |---|---|---|---|
  | `low` | 300 m | 2 s | 25° |
  | `medium` (padrão) | 600 m | 4 s | 45° |
  | `high` | 1 200 m | 8 s | 65° |

  `D = D₀ + T_a · v_marcador`, onde `v_marcador` é a velocidade do marcador
  *no vídeo* (metros de trajeto por segundo de vídeo, suavizada), de modo
  que a câmera sempre "enxerga" `T_a` segundos de movimento à frente do
  marcador. Isso resolve o caso "trajeto muito longo": num percurso de
  200 km em 45 s de acompanhamento o marcador anda ~4,4 km/s, e `D`
  cresce junto, mantendo o marcador em quadro.
- **Racional**: nível `high` de inclinação = mais vertical (mais de cima),
  conforme o cenário de aceite 3 da História 3. Para todos os níveis vale:
  a distância é monotonicamente maior em `high` que em `low` em qualquer
  quadro de acompanhamento (cenário 2), pois ambos `D₀` e `T_a` crescem.
- **Alternativas rejeitadas**: valores numéricos livres por flag (spec:
  níveis nomeados; Suposições); `D` fixo em metros (perderia o marcador em
  trajetos rápidos).

## 7. Abertura e fechamento

- **Decisão**: abertura e fechamento ocupam **10% da duração cada**
  (`OpeningFraction`, `ClosingFraction`), o acompanhamento os 80%
  restantes; em quadros, `abertura = round(N × OpeningFraction)`,
  `fechamento = round(N × ClosingFraction)` e o acompanhamento recebe o
  restante (`N − abertura − fechamento`). A **pose de visão geral** é: alvo
  no centro do trajeto projetado, inclinação 60°, **rumo igual ao do
  primeiro quadro de acompanhamento** na abertura e **ao do último** no
  fechamento (a câmera não gira durante a abertura nem o fechamento — só
  se aproxima ou se afasta), e distância
  `D_geral = max(margem · R / tan(FOV/2), OverviewMinDistanceFactor · D_a)`
  com `R` = maior distância do centro da caixa envolvente do trajeto
  projetado a um de seus pontos (aproxima o menor círculo que o contém),
  margem 1,2, fator 2 e `D_a` = distância de acompanhamento do quadro
  adjacente (o primeiro na abertura, o último no fechamento; sempre ≥ `D₀`,
  a distância base do nível escolhido). O piso `2·D_a` garante que a abertura
  sempre aproxime a câmera e o fechamento sempre a afaste (trajetos pequenos,
  ou de ritmo muito rápido no vídeo, também ganham uma visão de conjunto
  mais afastada que o acompanhamento). Como `D_a ≥ D₀`, a razão de zoom
  fica limitada por `D_geral/D₀`, que é o que `MinimumDuration` (item 8)
  usa como cota conservadora. O campo de visão
  vertical de referência é de 45° (`OverviewVerticalFOV` — a etapa de
  renderização poderá usar outro, mas o plano precisa de um valor para
  garantir que "o trajeto inteiro esteja enquadrado", FR-012).
  Na abertura, `T`, `ψ`, `θ` e `ln D` interpolam da visão geral para a pose
  de acompanhamento do primeiro quadro do acompanhamento com **smoothstep**
  (`3u² − 2u³`), cuja derivada nula nas pontas elimina saltos na junção. O
  fechamento é o espelho: da pose de acompanhamento do último quadro para a
  visão geral. O marcador fica no início durante toda a abertura e no fim
  durante todo o fechamento (FR-013).
- **Racional**: os quatro parâmetros continuam nas mesmas trilhas
  suavizadas do item 5, então a junção entre fases e as fases em si obedecem
  aos mesmos limites; nada é "colado" em separado.

## 8. Duração mínima (FR-017) e trajeto curto (FR-017a)

- **Decisão**: `MinimumDuration` é função pura do domínio, independente do
  restante do plano (usa cotas conservadoras, sem depender da própria
  duração escolhida, para evitar dependência circular):
  `abertura_mín = max(2 s, 1,5·|θ_geral−θ|/30°/s,
  1,5·|ln(D_geral/D₀)|/1,5/s)`, com `θ` e `D₀` do nível escolhido (não há
  termo de rumo: a visão geral já usa o rumo do acompanhamento, item 7);
  `fechamento_mín` idêntico; `acomp_mín = 5 s`.
  `duração_mín = max(abertura_mín/0,10, fechamento_mín/0,10,
  acomp_mín/0,80)`, arredondada para cima ao próximo quadro. O fator 1,5 é o
  pico de inclinação do smoothstep. A duração `d` é aceita se `d ≥
  duração_mín` (igualdade aceita; FR: "limítrofe").
- **Assinatura**: `MinimumDuration(points, frameRate, distance, tilt Level,
  tuning)` — depende dos níveis escolhidos, pois `θ` e `D₀` entram na fórmula.
- **Trajeto curto**: comprimento (distância acumulada pós-tratamento)
  abaixo de 50 m → `ErrTrackTooShort`, com o comprimento encontrado e o
  mínimo exigidos na mensagem. Acima disso o plano é gerado (Clarification Q2).
  **Trajeto grande**: abrangência acima de 2 000 km → `ErrTrackTooLarge`
  (FR-017b), com a abrangência encontrada e o máximo na mensagem.
- **Efeito prático** (nível médio de distância): mínimo de 20 s (o piso de
  2 s ÷ 10%) até poucos quilômetros; ≈ 25 s para 5 km; ≈ 39 s para 20 km;
  ≈ 55 s para 100 km; ≈ 85–92 s no limite de 2 000 km de abrangência
  (níveis médio e baixo). Uma duração **informada** abaixo do mínimo é
  recusada, e a mensagem indica o mínimo a usar (spec: "duração curta demais
  para o trajeto"). A duração **automática** (item 8.1) nunca é recusada,
  pois sempre parte do mínimo.

## 8.1 Duração automática (FR-003a, Clarification Q4)

- **Decisão**: sem `--duration`, a duração vem de `DefaultDuration(points,
  frameRate, tuning)`, função pura do domínio:
  `base = clamp(15 s + 6 s·√(comprimento_km), 20 s, 120 s)`, arredondada ao
  segundo mais próximo, e `duração = max(base, duração_mín)`, onde
  `duração_mín` é a de `MinimumDuration` (item 8, já arredondada ao próximo
  quadro). Comprimento = distância acumulada do trajeto tratado
  (`TotalDistance`). A duração informada pelo usuário nunca passa por esta
  função.
- **Valores de referência** (piso, teto e coeficientes ficam em
  `CameraTuning`): 1 km → 21 s; 5 km → 28 s; 20 km → 42 s; 50 km → 57 s;
  200 km → 100 s; 400 km ou mais → 120 s (teto). Se o mínimo do item 8 for
  maior que 120 s, o mínimo prevalece sobre o teto (cenário de aceite 10).
  **Com os valores iniciais isso nunca ocorre** (o mínimo fica abaixo de
  ≈ 92 s em todo trajeto aceito, item 8); a cláusula é uma salvaguarda para
  quando os valores de `CameraTuning` mudarem, e é testada com um
  `CameraTuning` sintético mais restritivo. Com os valores iniciais, o
  mínimo também fica abaixo da curva base para a maioria dos comprimentos
  (por exemplo, 20 km: mínimo ≈ 39 s, curva 42 s), então `max(base, mínimo)`
  raramente altera a curva.
- **Racional**: `√` é monotônica e sublinear (cenário 9, SC-011): dobrar a
  comprimento nunca dobra o vídeo. `max(base, mínimo)` garante por construção
  que uma duração calculada nunca é recusada e que a saída é suave. Usa
  distância, e não tempo real, porque a distância existe em qualquer
  trajeto (inclusive sem horário) e evita que uma pedalada lenta de 3 h
  gere um vídeo mais longo que uma corrida de mesmo comprimento.
- **Determinismo**: função só do trajeto tratado, da taxa de quadros e de
  `CameraTuning`; sem relógio.
- **Alternativas rejeitadas**: duração fixa de 60 s (recusaria trajetos
  longos e seria longa demais para trajetos de poucos quilômetros);
  proporcional ao tempo real (falha sem horário); linear na distância
  (vídeos de minutos para trajetos longos).

## 9. Determinismo e independência de plataforma (FR-014, SC-003)

- **Decisão**: (a) sem `map` iterado, sem goroutines, sem relógio, sem
  aleatoriedade no cálculo; (b) toda soma é feita em ordem fixa; (c) o Go
  pode fundir `x*y+z` em FMA em algumas arquiteturas (arm64), o que muda o
  último bit — os pontos sensíveis usam conversão explícita
  `float64(x*y) + z`, que a especificação da linguagem define como
  barreira de arredondamento; (d) cada valor do plano é **quantizado** ao
  final (latitude/longitude 1e-7°, alturas e distâncias 1e-3 m, ângulos
  1e-3°), o que absorve diferenças residuais de arredondamento entre
  máquinas — o valor quantizado é o valor oficial do plano, exportado e
  comparado nos testes.
- **Risco residual**: um valor que caia exatamente na fronteira de
  quantização em duas arquiteturas diferentes poderia diferir em 1 unidade
  do último dígito. Aceito e documentado; SC-003 é verificado na mesma
  máquina (o critério fala de execuções repetidas, não de arquiteturas), e
  um teste de regressão com valores esperados cobre a mesma máquina.

## 10. Formato de exportação: JSON versionado

- **Decisão**: um único arquivo JSON, com campo `"format_version": 1`, na
  ordem fixa de campos: `format_version`, `parameters`, `summary`,
  `frames`. Cada quadro é um objeto compacto em **uma linha** (fácil de
  inspecionar com `head`, `grep`, `diff`); o resto do arquivo é indentado.
  Números com precisão fixa (item 9); chaves sempre na mesma ordem
  (structs, nunca `map`), então a saída é idêntica byte a byte (FR-020).
  Detalhes em `contracts/plan-file.md`.
- **Racional**: `encoding/json` da biblioteca padrão já é usado pelo
  adapter `jsonfile` da etapa 2; JSON é lido por qualquer ferramenta e
  pelas etapas seguintes. Um formato binário/CSV perderia a estrutura
  (parâmetros, resumo, trechos suavizados) ou exigiria múltiplos arquivos.
- **Alternativas rejeitadas**: CSV (sem estrutura para resumo/trechos);
  NDJSON (bom para streaming, ruim para "um plano = um documento").

## 11. Escrita do arquivo: atômica e sem sobrescrita por padrão

- **Decisão**: `CameraPlanExporter.Export(plan, path, overwrite)` grava
  primeiro num arquivo temporário **no mesmo diretório** do destino e só
  depois o publica: sem `overwrite`, com `os.Link` (falha atomicamente se
  o destino existir → `ErrPlanDestinationExists`; o temporário é sempre
  removido); com `overwrite`, com `os.Rename`. Qualquer falha antes da
  publicação remove o temporário, então nunca resta arquivo parcial
  (FR-021). Diretório inexistente ou sem permissão →
  `ErrPlanDestinationInvalid`.
- **Racional**: `Link` evita a condição de corrida entre "checar se existe"
  e "criar"; `Rename` substitui atomicamente em POSIX e (Go ≥ 1.5) também no
  Windows. Em sistemas de arquivos sem suporte a hard link, o adapter
  recorre a `O_CREATE|O_EXCL` na criação direta, sem atomicidade da
  substituição — que só ocorre no caminho `overwrite`.
- **Limitação conhecida de teste**: o recurso ao `O_CREATE|O_EXCL` em
  sistemas de arquivos sem hard link não é exercitado pela suíte automatizada
  (o ambiente de teste suporta `os.Link`); o trecho fica isolado numa função
  pequena e comentada, coberto por revisão de código.
- **Alternativa rejeitada**: checar `os.Stat` antes de gravar (corrida) ou
  pedir confirmação interativa (Clarification Q1: recusar por padrão).

## 12. Configuração injetada: `domain.CameraTuning`

- **Decisão**: todos os valores numéricos dos itens 3 a 8 vivem numa
  estrutura `domain.CameraTuning`, declarada no domínio (o núcleo define o
  formato) e preenchida com os valores padrão por `config.Load()` (Princípio
  VIII), e passada por `cmd/sobrevoo/main.go` para o construtor do
  serviço — mesmo caminho de `MinPoints` e `MaxPlausibleSpeedKmh` na etapa
  1. Ainda não existe fonte externa de configuração (variável de ambiente
  ou arquivo); o ponto de extensão já existe.
- **Racional**: os valores são heurísticas de aparência, vão mudar quando a
  etapa de renderização mostrar como o voo realmente parece; concentrá-los
  evita "número mágico" espalhado e deixa os testes de domínio construírem
  variações sem tocar em código.
- **Padrões de parâmetros do usuário**: taxa de quadros 30, distância
  `medium` e inclinação `medium` (FR-003, FR-005) não são heurísticas do
  algoritmo e não ficam em `CameraTuning`. Vêm de
  `Config.DefaultPlanParameters` (um `domain.PlanParameters` com `Duration`
  `nil`), preenchido por `config.Load()` e injetado no comando `plan`, que o
  usa como valor padrão das flags (Princípio VIII). O `defaultLevel` da etapa
  1 continua sendo só o nível de simplificação/suavização do trajeto; os
  dois conceitos deixam de compartilhar a mesma constante.

## 13. Reuso do pipeline da etapa 1: extrair `TrackService`

- **Problema**: a sequência `Parse` → `CleanTrack` (→ `Simplify` → `Smooth`)
  já existe duplicada em `InspectTrackService.Inspect` e em
  `GeoDataService.CheckCoverage` (esta sem simplificar/suavizar, por
  decisão da etapa 2), e os dois serviços carregam os mesmos limiares
  (`minPoints`, `maxPlausibleSpeedKmh`) e as mesmas portas
  (`TrackParser`, `Simplifier`, `Smoother`). Um `CameraPlanService` que
  repetisse a sequência seria a terceira cópia.
- **Decisão**: evoluir as etapas anteriores e extrair a orquestração
  para **um único serviço por recurso, `TrackService`**, o recurso "trajeto"
  (Princípio IX), com três métodos nomeados pela operação:

  | Método | Faz | Quem usa |
  |---|---|---|
  | `Clean(reader) (domain.CleanedTrack, error)` | `Parse` → `CleanTrack` | `GeoDataService.CheckCoverage` |
  | `Treat(reader, simplification, smoothing Level) (domain.TreatedTrack, error)` | `Clean` → `Simplify` → `Smooth` | `CameraPlanService.Generate`, `TrackService.Inspect` |
  | `Inspect(reader, simplification, smoothing Level) (domain.TrackSummary, error)` | `Treat` → `domain.SummarizeTrack` | comando `inspect` |

  `CleanedTrack` (`Track`, `Points`, `Discarded`) e `TreatedTrack`
  (`Track`, `Route`, `Discarded`) são tipos de **domínio**, pois são DTOs
  de saída não triviais (Princípio IX). `TrackService` é o único que
  conhece `TrackParser`, `Simplifier`, `Smoother` e os limiares; os demais
  serviços passam a depender dele (serviço dependendo de serviço, permitido
  pelo Princípio IX, como `GroupInviteService` → `UserService` no
  repositório de referência).
- **Efeito sobre `InspectTrackService`**: o serviço atual é uma interface
  de método único, herdada da etapa 1, anterior à redação atual do
  Princípio IX. Ele é **absorvido** por `TrackService` como o método
  `Inspect`, em vez de sobreviver como serviço à parte só para chamar
  `Treat`. Isso elimina um desvio remanescente da constituição e um
  serviço que só delegaria. O comando `inspect` passa a depender de
  `application.TrackService`.
- **Comportamento preservado**: nenhuma saída, código de saída ou regra
  das etapas 1 e 2 muda. Os testes existentes de `Inspect` e de
  `CheckCoverage` são migrados (mesmos cenários, mesmas expectativas), e
  esta refatoração é a **primeira fase de tarefas**, executada e verde
  antes de qualquer código de câmera, para que uma regressão nas etapas
  anteriores nunca se misture com um defeito da etapa 3.
- **Níveis de simplificação e suavização do trajeto**: `plan` não expõe
  flags novas; usa o nível padrão de configuração (`DefaultLevel`), como
  `inspect` sem flags. Acrescentar flags depois é uma mudança compatível.
- **Alternativas rejeitadas**: manter `InspectTrackService` e só acrescentar
  um `RouteService` (mantém um serviço de método único e duas camadas onde
  uma basta); duplicar a sequência mais uma vez (a opção original deste
  item, descartada); mover a orquestração para o domínio (precisa das
  portas, então continua sendo trabalho da service layer).

## 14. Nomes e organização (Princípio IX)

- **Porta nova**: `CameraPlanExporter` — papel arquitetural: escreve o
  plano fora do processo (não é um repositório: não há leitura nem
  consulta). Ligada à entidade `CameraPlan`, é declarada em
  `camera_plan.go`, no início do arquivo, com `//go:generate` logo após o
  `package`.
- **Adapter**: pacote `jsonfile` (tecnologia que já serve outra porta) →
  `jsonfile.NewCameraPlanExporter()` em `camera_plan_exporter.go`.
- **Pacote de domínio**: continua `internal/domain`, um arquivo por
  responsabilidade (`camera_plan.go`, `camera_plan_parameters.go`,
  `camera_planning.go`, `camera_projection.go`, `camera_timeline.go`,
  `camera_motion.go`, `camera_framing.go`), nada de `helpers.go` ou
  subpacote por algoritmo.
- **Receivers**: `p` para `PlanParameters`, `c` para `CameraPlan`, `f` para
  `CameraFrame`, `s` para `cameraPlanService` e `trackService`, `e` para o
  exporter.

## 15. Nome e forma do comando

- **Decisão**: `sobrevoo plan <arquivo>` (comando único, sem grupo
  `camera`), com `--duration <segundos>` (opcional; ausente = duração
  automática, item 8.1), `--fps <n>` (padrão 30, vindo de
  `Config.DefaultPlanParameters`), `--distance
  low|medium|high`, `--tilt low|medium|high`, `--export <caminho>` e
  `--overwrite`. Segundos e quadros por segundo são números; valor não
  numérico é erro de uso (código 2, como `parseLevel` na etapa 1); valor
  numérico fora do domínio (≤ 0, fps fora de 1–120) é regra de negócio
  (códigos próprios, item 16).
- **Racional**: `geodata` virou um grupo porque tem quatro operações;
  `plan` é uma operação só. Quando etapas futuras precisarem de mais
  comandos de câmera, promover para grupo é uma mudança pequena.

## 16. Códigos de saída novos

Sequência a partir de 10 (1–9 já usados nas etapas 1 e 2):

| Erro sentinela | Código |
|---|---|
| `ErrInvalidDuration` | 10 |
| `ErrInvalidFrameRate` | 11 |
| `ErrDurationTooShort` | 12 |
| `ErrTrackTooShort` | 13 |
| `ErrTrackTooLarge` | 14 |
| `ErrPlanDestinationExists` | 15 |
| `ErrPlanDestinationInvalid` | 16 |

Erros de leitura/tratamento do trajeto reaproveitam os códigos 1–3 já
existentes (FR-022). Valor não numérico e níveis desconhecidos continuam
sendo erro de uso, código 2.

## 17. Funções puras de câmera exportadas para teste isolado

- **Problema**: os testes de `internal/domain` são do pacote externo
  `domain_test` (necessário porque usam `internal/domain/builddomain`, que
  importa `domain`; um teste de pacote interno criaria ciclo de importação).
  Funções não exportadas não seriam testáveis isoladamente.
- **Decisão**: as primitivas puras de câmera são **exportadas**, seguindo o
  precedente de `Haversine`, `ComputeBoundingBox` e `ComputeCoverage`:
  `NewLocalPlane` (com `Project` e `Unproject` em `LocalPlane`),
  `BuildMarkerTimeline` (devolve `MarkerTimeline`), `DesiredHeading`,
  `UnwrapAngles`, `FollowDistance`, `ComputeCameraPose` (devolve
  `CameraPose`), `LimitRate`, `GaussianSmooth`, `DetectSmoothedSpans`,
  `OverviewPose`, `BlendPose`, `MinimumDuration`, `DefaultDuration` e
  `PlanCamera`. Cada uma tem comentário de documentação em inglês e nenhuma
  depende de I/O, relógio ou estado. Só helpers triviais e de escopo
  estrito (por exemplo a quantização) permanecem não exportados e são
  cobertos via `PlanCamera`.
- **Racional**: testar cada regra numérica isoladamente (com entradas
  pequenas e exatas) é bem mais barato e preciso do que inferi-las de planos
  completos, e não custa nada de arquitetura: são funções de domínio, não de
  infraestrutura.
- **Alternativa rejeitada**: testar tudo apenas por `PlanCamera` (falhas
  ficariam difíceis de localizar); usar um arquivo `export_test.go` para
  expor as funções internas (mascara o contrato real do pacote).
