# Pesquisa: Bloco de Velocidade na Sobreposição

**Feature**: `014-speed-overlay-block` | **Data**: 2026-10-04

## 1. Como calcular a velocidade média numa janela de tempo, sem ler o trajeto de novo

**Decisão**: acrescentar `Route.DistanceAt(distances []float64, at time.Duration) (float64, bool)`
— o espelho exato de `Route.TimeAt` (`internal/domain/duration.go`), já
existente desde a etapa 9. `TimeAt` busca por distância (`sort.
SearchFloat64s(distances, at)`) e interpola o tempo decorrido entre os dois
pontos vizinhos; `DistanceAt` busca por tempo decorrido (`sort.Search` sobre
`r.Points[i].Time`) e interpola a distância, com a mesma forma segura de
`lerp`, `(1-t)*a + t*b`, nunca `a + t*(b-a)` (garante o resultado exato nas
pontas em ponto flutuante — o motivo já documentado em
`009-frame-overlays/research.md` item 7). As duas funções compartilham
`allHaveTime()` como critério de disponibilidade.

Com `Route.DistanceAt` em mãos, o cálculo por quadro em
`cameraPlanner.frame` (`camera_planning.go`) fica:

```go
var markerSpeed float64
if ok { // mesmo ok de ActivityElapsed = trackRoute.TimeAt(...)
    half := tuning.SpeedWindow / 2
    total, _ := c.trackRoute.Duration()
    startT := clampDuration(activityElapsed-half, 0, total)
    endT := clampDuration(activityElapsed+half, 0, total)
    startDist, _ := c.trackRoute.DistanceAt(c.route.Distances, startT)
    endDist, _ := c.trackRoute.DistanceAt(c.route.Distances, endT)
    if span := endT - startT; span > 0 {
        markerSpeed = (endDist - startDist) / span.Seconds()
    }
}
```

Isso cobre, de graça, o requisito de encurtar a janela nos extremos (FR-004):
`clampDuration` já limita `startT`/`endT` a `[0, total]`, então perto do
início ou do fim do trajeto a janela efetiva (`span`) já é menor que
`tuning.SpeedWindow` — nunca ausente, nunca um valor fora do trecho real da
atividade. Quando `span` chega a zero (trajeto mais curto que a janela, ou
um único ponto), a velocidade fica `0`, o mesmo tratamento que um trajeto
sem movimento já teria de qualquer forma.

**Alternativas consideradas**:
- *Diferença entre dois pontos consecutivos do GPS*: é exatamente o que o
  pedido original exclui — ruído de posição de poucos metros virando dezenas
  de km/h de diferença entre quadros.
- *Média móvel sobre os próprios quadros do vídeo* (em vez de sobre o
  tempo real da atividade): acoplaria a janela à duração do vídeo e ao
  frame rate escolhidos, então o mesmo trecho da atividade teria uma
  "janela" de tamanho diferente num vídeo de 30s e num de 90s — o pedido
  original exige uma janela igual "em todo o vídeo e em qualquer trajeto",
  o que só a janela em tempo de atividade garante.
- *Resolver a janela com uma varredura linear nos pontos do trajeto a cada
  quadro*: funciona, mas é O(n) por quadro em vez de O(log n); rejeitada
  porque `Route.DistanceAt`/`TimeAt` já pagam o custo de uma busca binária e
  o padrão já está estabelecido no código.

## 2. Onde a disponibilidade da velocidade é decidida — sem flag nova

**Decisão**: a velocidade só existe quando `allHaveTime()` é verdadeiro —
exatamente o critério que já decide `TimeReference == TimeReferenceClock`
hoje (o mesmo `ok` de `Route.TimeAt`, reaproveitado para `ActivityElapsed`).
Não é necessário nenhum campo novo em `PlanSummary`/`CameraPlan` (como
`ElevationAvailable` existe para elevação): a sobreposição já tem, pronto, o
sinal que precisa — `plan.TimeReference == TimeReferenceClock` — e é
exatamente o mesmo sinal que já esconde o bloco de tempo decorrido
(`showTime` em `frame_screen_overlay.go`). `showSpeed` usa a mesma
expressão.

**Alternativa considerada**: um `SpeedAvailable bool` dedicado em
`CameraPlan`, pelo padrão de `ElevationAvailable`. Rejeitada por
redundância: `ElevationAvailable` existe porque elevação e tempo são
critérios *independentes* (um trajeto pode ter um e não o outro); velocidade
depende exclusivamente do mesmo critério que já governa `TimeReference`, não
de um terceiro critério — um campo novo só repetiria uma informação que o
plano já guarda.

## 3. O tamanho da janela

**Decisão**: `CameraTuning.SpeedWindow = 30s` (±15s em torno do instante do
marcador), um campo novo de `CameraTuning`, com valor fixo em
`config.CameraTuning.SpeedWindowSeconds` (`internal/infra/outbound/
config/config.go`) — mapeado para o domínio por `domainCameraTuning`
(`cmd/sobrevoo/config_mapping.go`), exatamente como `GaussianSigmaSeconds`/
`OverviewMinDistanceFactor` já são. Não há flag de CLI para ele (Suposição
do `spec.md`): é um limiar interno, não uma escolha do usuário, no mesmo
grupo de valores que `config.go` já guarda sem nenhuma fonte de configuração
externa (Princípio VIII da constituição, "o ponto de extensão já existe").
Trinta segundos é grande o bastante para absorver o ruído de posição comum
de um GPS de pulso/celular (erro típico de alguns metros, que numa janela de
1–2s produziria variações de dezenas de km/h) e pequeno o bastante para
ainda acompanhar uma subida ou descida que dure, digamos, um minuto ou mais
— a mesma faixa de janela ("smoothed pace/speed") que apps de atividade
física populares já usam como padrão não ajustável.

**Alternativas consideradas**: uma janela de 5–10s reagiria mais rápido a
mudanças de inclinação, mas deixaria o número tremendo em trajetos urbanos
com GPS ruidoso (exatamente o defeito que o pedido original pede para
evitar); uma janela de 60s+ estabilizaria demais e esconderia uma subida ou
descida de duração moderada (o pedido original exige "ainda acompanhar uma
subida ou descida"). 30s é o valor documentado aqui; como o pedido original
proíbe explicitamente deixá-lo ajustável pelo usuário, não há necessidade de
uma faixa configurável — só de um valor fixo e citado em código e nesta
pesquisa.

## 4. Unidade e formato de exibição

**Decisão**: `CameraFrame.MarkerSpeed` é armazenado em metros por segundo
(SI, a mesma convenção interna de todo o resto do plano — `CameraToMarkerDistance`,
`TrackElevation` etc. já estão em metros), quantizado com um novo passo
`speedStep = 1e-3` (mesma ordem de grandeza de `lengthStep`, só nomeado
separadamente por ser outra unidade). A exibição na sobreposição converte
para quilômetros por hora, uma casa decimal — `formatOverlaySpeed(mps
float64) string { return fmt.Sprintf("%.1f km/h", mps*3.6) }` —, a unidade
que qualquer pessoa que pedala ou corre já espera, e a mesma convenção
métrica que `formatOverlayDistance`/`formatOverlayElevation` já seguem.

**Alternativa considerada**: metros por segundo também na exibição (sem
conversão) — rejeitada porque não é a unidade que o público-alvo do pedido
original (quem pedala ou corre) usa para ler velocidade; minutos por
quilômetro (ritmo) — explicitamente fora de escopo no pedido original.

## 5. Formato do arquivo de plano exportado e subida de versão

**Decisão**: `frames[].marker.speed_mps` (número, 3 casas decimais, mesmo
tipo `measure()` que `distance_m`/`elevation_m`/`gain_m` já usam) é um campo
novo e **obrigatório** — não opcional — porque não há valor implícito seguro
para a ausência dele (ao contrário de `parameters.aspect_ratio` na etapa 3,
ou de `parameters.simplification`/`.smoothing` na etapa 13, cuja ausência
lê como um padrão razoável e documentado). Por isso `format_version` sobe
de `2` para `3`, exatamente pelo mesmo raciocínio que já subiu de `1` para
`2` na etapa 9 (`plan-file-v2.md`): um plano `format_version: 2`, sem
`speed_mps`, é recusado por `ErrPlanFormatVersionUnsupported`, a mesma
mensagem e o mesmo código de saída que já recusam qualquer versão
desconhecida — nenhum sentinela novo, nenhum código de saída novo.

## 6. Onde a velocidade entra na identidade do plano

**Decisão**: `CameraPlan.ID()` grava `quantize(f.MarkerSpeed, speedStep)`
no hash, ao lado de `CameraToMarkerDistance` — a mesma lista de campos por
quadro, só com mais uma linha. Isso basta para que dois planos iguais em
tudo menos na velocidade calculada sejam reconhecidos como planos
diferentes (FR-011 do `spec.md`), e para que `FlightService.reusePlan`
(que já decide por esse `ID()`) nunca reaproveite um pelo outro — sem
nenhuma lógica de reaproveitamento nova, o mesmo efeito que a etapa 13 já
obteve ao gravar `Simplification`/`Smoothing` no mesmo hash. `ActivityElapsed`/
`TrackElevation`/`TrackElevationGain` não entram no hash hoje (são funções
determinísticas do que já entra); `MarkerSpeed` é diferente: dado o mesmo
trajeto e os mesmos demais parâmetros, duas execuções sempre produzem a
mesma velocidade — mas o pedido original exige explicitamente que ela
participe da identidade ("entra na identidade do plano como tudo o que o
plano carrega"), então a trata como os campos geométricos do quadro
(posição, distância), não como os três campos derivados que ficaram de
fora na etapa 9.

## 7. O bloco novo na sobreposição — reaproveitando o mecanismo dos quatro já existentes

**Decisão**: `OverlayBlockSpeed OverlayBlock = "speed"`, um quinto campo
booleano em `OverlayConfig` (`Speed`), uma branch a mais em
`NewOverlayConfig` e um dígito a mais em `Fingerprint()` — exatamente a
mesma forma dos quatro já existentes. A diferença inteira está em quem
constrói a lista de blocos quando nenhuma é pedida: `config.RenderDefaults.
OverlayBlocks` (`internal/infra/outbound/config/config.go`) continua
`{"distance", "elevation", "time", "profile"}`, sem `"speed"` — a escolha
padrão não muda (fora de escopo, FR-016/FR-007). No desenho
(`frame_screen_overlay.go`), `speed` entra na mesma lista de blocos
numéricos que `distance`/`elevation`/`time` já formam, com `showSpeed :=
s.config.Speed && plan.TimeReference == TimeReferenceClock` (item 2 acima),
participando de `stablePanelWidth` e do empilhamento vertical em `draw` do
mesmo jeito que os outros três — então o painel de velocidade compartilha a
largura estável com os demais automaticamente, sem nenhuma mudança na
mecânica de painéis que a etapa 11/12 já estabeleceram (fora de escopo desta
etapa).

**Alternativa considerada**: tratar `speed` como um bloco "fora do padrão"
de um jeito especial dentro de `OverlayConfig` (por exemplo, um campo
separado de `Enabled`/blocos). Rejeitada: o pedido original só pede que ele
não esteja na lista *padrão* de blocos — o mecanismo de seleção
(`--overlay-blocks`) é o mesmo para os cinco; tratá-lo diferente dentro do
domínio criaria uma segunda forma de "bloco" sem necessidade.

## 8. Mensagens e código de saída

**Decisão**: nenhum sentinela novo, nenhum código de saída novo.
`ErrInvalidOverlayBlock` (código `55`, inalterado) passa a listar `speed`
entre os nomes aceitos na mensagem (`"expected one of distance, elevation,
time, profile, speed"`); `ErrPlanFormatVersionUnsupported` (código `18`,
inalterado) passa a citar `3` como a versão aceita. `--overlay-blocks`
(texto de ajuda) passa a mencionar `speed` e deixa explícito que ele não
faz parte do padrão — o único lugar em que a documentação de ajuda precisa
dizer isso, já que o padrão impresso por `--help` (`formatOverlayBlocks`
sobre `config.RenderDefaults`) continua mostrando só os quatro de sempre.
