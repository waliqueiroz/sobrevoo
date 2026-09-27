# Pesquisa: Voo em um Único Comando

**Feature**: `007-full-flight-pipeline` | **Data**: 2026-09-27

A especificação não deixou nenhum `[NEEDS CLARIFICATION]` (ver `spec.md` e a
sessão de `/speckit-clarify`). Os itens abaixo não resolvem ambiguidade — eles
registram as decisões técnicas tomadas durante o planejamento, cada uma
verificada contra o código das etapas 1 a 6 já implementadas, para que nenhuma
regra de negócio existente seja duplicada ou contornada (FR-002).

## 1. `FlightService` depende de outros serviços de aplicação, não de portas do domínio diretamente

**Decisão**: `FlightService` recebe, no construtor, `CameraPlanService`,
`GeoSliceService`, `FrameService` e `VideoService` — as quatro *service
interfaces* já existentes — mais a porta nova `Workspace`. Ele nunca chama uma
porta de domínio (`CameraPlanExporter`, `BaseMapReader`, `FrameRepository`,
`VideoEncoder`, ...) diretamente; só orquestra os quatro serviços, na ordem
das cinco etapas.

**Motivo**: é o padrão que a constituição já registra e que
`CameraPlanService`/`GeoSliceService` já seguem com `TrackService` ("Um
serviço de aplicação PODE depender de outro serviço de aplicação... quando
uma operação precisa de lógica que já vive em outro serviço", Princípio IX).
Chamar os serviços em vez das portas é o que garante FR-002 *por construção*:
o mesmo `CameraPlanService.Generate` que `plan` chama é o único jeito de gerar
um plano; não existe um segundo caminho que pudesse divergir.

**Alternativas consideradas**: um `FlightService` que injeta as mesmas portas
de domínio que os quatro serviços já usam, chamando-as diretamente — rejeitada
porque duplicaria a orquestração que já existe dentro de cada serviço (por
exemplo, a ordem exata das chamadas dentro de `GeoSliceService.Generate`),
violando FR-002 na prática, mesmo que cada peça individual continuasse
correta.

## 2. Recusa cedo do codificador e do destino: dois métodos novos em `VideoService`

**Decisão**: `VideoService` ganha `CheckEncoder(ctx) (domain.EncoderInfo,
error)` e `CheckDestination(output string, overwrite bool) error`, que
`Assemble` passa a chamar internamente (em vez de falar com `encoder.Probe` e
`exporter.Check` do próprio corpo do método). `FlightService.Fly` chama os dois
como o primeiro passo, antes de tocar no trajeto.

**Motivo**: FR-005/FR-007 exigem a checagem do codificador e do destino do
vídeo *antes de qualquer etapa* — inclusive antes de tratar o trajeto, que a
própria montagem do vídeo só faria muito depois. Expor as duas checagens como
operações nomeadas do mesmo recurso (`Video`) é a mesma divisão que
`CameraPlanService` já faz entre `Generate`/`Export`/`Load`: cada operação é um
passo que faz sentido sozinho. Refatorar `Assemble` para chamá-las evita
duplicar a lógica (Probe/Check só existem uma vez).

**Alternativas consideradas**: injetar `domain.VideoEncoder` e
`domain.VideoExporter` diretamente em `FlightService` — rejeitada pelo mesmo
motivo do item 1 (duas fontes de verdade para a mesma checagem); um método
único `VideoService.CheckReadiness(ctx, output, overwrite) error` que junta as
duas checagens — rejeitada porque perderia `EncoderInfo` (usado no resumo
final) sem necessidade e criaria um método que nenhum outro chamador (a CLI de
`video`, se quisesse checar cedo) poderia reaproveitar em parte.

## 3. Cobertura dos dados registrados: nenhuma checagem nova, a ordem já existente em `GeoSliceService.Generate` basta

**Decisão**: `FlightService` não faz nenhuma checagem de cobertura própria.
Ele chama `GeoSliceService.Generate(plan)` (quando o recorte não é
reaproveitado) exatamente como `geodata slice` já chama, e essa função **já**
lista as fontes registradas e confere a cobertura (`route.Coverage`) como o
primeiro passo, antes de ler qualquer conteúdo de peça ou de relevo (ver
`internal/application/geo_slice_service.go`, função `Generate`).

**Motivo**: FR-006 pede que a cobertura seja verificada "assim que o plano de
câmera existir... antes do desenho de qualquer quadro" — exatamente onde
`GeoSliceService.Generate` já verifica, por já ser a etapa que precisa saber
disso antes de gastar E/S com peças e relevo. Reimplementar essa checagem em
`FlightService` duplicaria uma regra que já existe (proibido por FR-002).

**Alternativas consideradas**: chamar `GeoDataService.CheckCoverage` (usado
pelo comando standalone `geodata check`) antes do planejamento da câmera —
rejeitada porque essa checagem usa a área do **trajeto tratado**, não a área
de interesse do **plano** (`CameraPlan.AreaOfInterest`, que inclui a margem de
enquadramento da câmera); as duas áreas não são a mesma, e usar a errada
poderia aceitar um trajeto que `GeoSliceService.Generate` recusaria de
qualquer forma (ou vice-versa) — teria sido uma regra nova e potencialmente
divergente, não uma reafirmação da existente.

## 4. Reaproveitamento do plano e do recorte: as identidades que já existem no domínio, sem nenhuma nova

**Decisão**: para decidir se um `plan.json`/`slice.zip` já presente no
diretório de `--keep` ainda vale, `FlightService`:

1. Tenta `cameraPlanService.Load(planPath)`. Se ler com sucesso e
   `existing.ID() == fresh.ID()` (o plano recém-computado, em memória) —
   reaproveita, sem escrever nada.
2. Caso contrário (arquivo ausente, ilegível, ou de outro plano), chama
   `cameraPlanService.Export(fresh, planPath, overwrite)` — que **já** recusa
   por padrão um arquivo existente (`ErrPlanDestinationExists`) e só substitui
   com `overwrite=true`.

O recorte segue o mesmo padrão com `geoSliceService.Load`/`.Export` e
`GeoSlice.EnsureMatches(plan)` (que já compara `slice.PlanID` com
`plan.ID()`, usada hoje por `FrameService.check`).

**Motivo**: `CameraPlan.ID()` já é "o SHA-256... de uma codificação canônica
dos parâmetros efetivos e de cada quadro" (comentário de
`internal/domain/camera_plan.go`) — uma identidade por **conteúdo**, não por
caminho nem data de modificação, exatamente a suposição documentada em
`spec.md`. `GeoSlice.EnsureMatches` já existe para o mesmo propósito, usado
por `FrameService` antes de desenhar. FR-010a exige "a mesma regra de conjunto
que o desenho dos quadros já usa" — usar as identidades que o próprio desenho
já confia é a leitura mais literal possível dessa frase. FR-010b (recusar por
padrão um conteúdo de outro conjunto) sai de graça: `Export(..., overwrite)`
já tem exatamente essa regra, para qualquer um dos três arquivos.

**Alternativas consideradas**: uma identidade nova, específica desta etapa
(hash do arquivo de trajeto bruto + parâmetros brutos, calculado antes de
tratar/planejar) — rejeitada por duplicar uma regra que `CameraPlan.ID()` já
resolve melhor (inclui o efeito de qualquer parâmetro, sem a etapa 7 precisar
saber quais parâmetros existem); comparar só metadados do arquivo (tamanho,
data de modificação) — rejeitada explicitamente pela Suposição da spec
("não no caminho do arquivo nem em datas de modificação").

## 5. Reaproveitamento dos quadros: nenhuma lógica nova — é a mesma chamada de `render all`

**Decisão**: `FlightService` chama `frameService.DrawFrames(ctx, plan, slice,
domain.FrameSetRequest{Directory: framesDir, Resolution: ..., Overwrite:
...}, progress)` — a **mesma** chamada, com os mesmos argumentos, que
`runRenderAll` já faz. Nenhum código novo decide o que manter, o que desenhar
ou o que recusar: `FrameDirectory.Plan` (chamado dentro de `DrawFrames`) já
faz isso, pelo `FrameMark.SetID` (plano + recorte + resolução + versão do
desenho) que a etapa 5 já grava em cada quadro.

**Motivo**: é a leitura mais direta de FR-010a ("a mesma regra de conjunto que
o desenho dos quadros já usa") — não uma regra *parecida*, a regra *dela
mesma*, chamada de novo. `framesDir` tanto faz ser o diretório temporário
(execução sem `--keep`) quanto `<diretório-guardado>/frames` (com `--keep`):
`DrawFrames` não sabe nem precisa saber a diferença.

**Alternativas consideradas**: nenhuma — qualquer lógica própria aqui seria,
por definição, uma duplicata da regra de conjunto já existente (proibida por
FR-002).

## 6. O plano e o recorte só tocam disco quando `--keep` é usado; os quadros sempre precisam de um diretório real

**Decisão**: `CameraPlanService.Generate`, `GeoSliceService.Generate`,
`FrameService.DrawFrames` e `VideoService.Assemble` recebem e devolvem os
valores de domínio (`domain.CameraPlan`, `domain.GeoSlice`) diretamente — eles
**não** exigem um arquivo. `Export`/`Load` só são chamados quando `--keep`
aponta para um diretório. Sem `--keep`, o plano e o recorte vivem só na
memória do processo pelo tempo da execução; só os quadros — que
`FrameService`/`VideoService` esperam como arquivos num diretório real, para o
`ffmpeg` ler — precisam de um diretório de verdade, criado por uma porta nova
e mínima:

```go
// Workspace fornece o diretório de quadros de uma execução cujos
// intermediários o usuário não pediu para guardar.
type Workspace interface {
    // NewTemporary cria um diretório vazio, de uso exclusivo desta execução,
    // e a função que o remove por inteiro. remove DEVE ser chamada
    // exatamente uma vez, qualquer que seja o desfecho da execução.
    NewTemporary() (path string, remove func() error, err error)
}
```

**Motivo**: satisfaz FR-009 ("ficam num local temporário... e são apagados ao
final") do jeito mais forte possível para o plano e o recorte — eles nunca
existem como arquivo, então não há nada para "esquecer" de apagar. Só os
quadros exigem E/S real (potencialmente gigabytes de PNG que o `ffmpeg`
precisa ler do disco, não da memória do Sobrevoo — Restrição já existente
"nada de quadro inteiro na memória do Sobrevoo" da etapa 6), e por isso são a
única coisa que precisa de uma porta nova.

**Alternativas consideradas**: sempre exportar plano e recorte para um
diretório temporário, mesmo sem `--keep` (por uniformidade com os quadros) —
rejeitada por gastar E/S e round-trip de (de)serialização que a Restrição de
desempenho não pede e a spec não exige, sem nenhum ganho — `Load`/`Export`
existem para a persistência que o usuário *pediu* (FR-010), não para uso
interno.

## 7. Progresso por etapa: `FlightStage` + `FlightProgress`, encaminhando o progresso que cada etapa já emite

**Decisão**: um enum novo, `domain.FlightStage` (`TrackProcessing`,
`CameraPlanning`, `GeoDataSlicing`, `FrameRendering`, `VideoEncoding`, com
`String()`), e `domain.FlightProgress{Stage FlightStage; Render
*RenderProgress; Video *VideoProgress}`. `FlightService.Fly` chama `progress`
com só `Stage` preenchido ao entrar em cada uma das cinco etapas, e encaminha
o `RenderProgress`/`VideoProgress` que `DrawFrames`/`Assemble` já relatam,
embrulhado com a etapa. A CLI reaproveita as mesmas funções de formatação de
linha de `render all`/`video` para a parte de quadro/percentual/tempo, só
acrescentando o anúncio "Stage N/5: ..." ao entrar numa etapa (inclusive
quando ela foi pulada por reaproveitamento: "Stage 3/5: geo data slice already
valid for this plan, reusing it").

**Motivo**: é a leitura literal de FR-004 (saber "em qual das cinco etapas
internas está" e, nas duas mais lentas, "o mesmo progresso... que o comando
individual dessa etapa já mostra"). Encaminhar em vez de recalcular evita
qualquer risco de os dois progressos divergirem.

**Alternativas consideradas**: uma barra de progresso única, agregando as
cinco etapas num só percentual — rejeitada porque a spec pede identificação
por etapa, não um número agregado (que também seria enganoso: as etapas têm
custos muito diferentes entre si, então um "60% geral" não diria nada útil).

## 8. Interrupção: um código de saída próprio, com a granularidade que cada etapa já suporta

**Decisão**: `domain.ErrFlightInterrupted` (código de saída `51`), devolvido
por `FlightService.Fly` sempre que a execução é interrompida, **substituindo**
o erro de interrupção da etapa em que ocorreu (`ErrRenderInterrupted`,
`ErrVideoInterrupted`) — a única exceção deliberada à regra geral de FR-008
("mesmo erro... que o comando correspondente daria"), prevista pela própria
FR-011. A checagem de `ctx.Err()` acontece nos pontos de retomada entre
etapas — inclusive para `CameraPlanService.Generate` e
`GeoSliceService.Generate`, que **não** recebem `context.Context` (nunca
receberam) e por isso só podem ser interrompidas *entre* chamadas, não durante
uma delas.

**Motivo**: é a leitura literal de FR-011. A granularidade desigual não é uma
regra nova — é a mesma limitação que `plan` e `geodata slice`, rodados
sozinhos, já têm hoje (nenhum dos dois responde a `Ctrl+C` no meio do
cálculo); documentá-la (R1 do `plan.md`) é mais honesto do que fingir uma
capacidade de cancelamento que essas duas etapas nunca tiveram.

**Alternativas consideradas**: adicionar `context.Context` a
`CameraPlanService.Generate`/`GeoSliceService.Generate` para permitir
cancelamento fino ali também — rejeitada: mudaria a assinatura de dois
serviços de etapas já entregues, fora do escopo desta etapa ("nenhuma regra
existente pode ser... contornada" inclui não abrir mão da estabilidade da
assinatura só para um ganho de granularidade que a spec não pede
explicitamente).

## 9. Nomes: comando `fly`, serviço `FlightService`

**Decisão**: comando `sobrevoo fly <trajeto> --output <vídeo.mp4> ...`;
serviço `FlightService`/`flightService`/`NewFlightService`, método único
`Fly(ctx, reader, request, progress) (FlightSummary, error)`.

**Motivo**: "fly" é o verbo do próprio nome do projeto (Sobrevoo = sobrevoar);
lido como comando, `sobrevoo fly trajeto.gpx --output voo.mp4` descreve
exatamente o que a etapa faz, sem colidir com nenhum substantivo já usado
(`plan`, `video`, `geodata`, `render`). O método `Fly` não repete o nome do
pacote (`application.Fly`, não `flightService.Flight`), seguindo a mesma regra
de nomes exportados da Seção IX.

**Alternativas consideradas**: `PipelineService`/`sobrevoo run` — mais
genérico, mas perde a identidade do domínio (o padrão de nomes do projeto usa
vocabulário de voo — "flight" já aparece informalmente na documentação de
`CameraPlanService" para descrever o que o plano de câmera sobrevoa);
`sobrevoo video` sobrecarregado com uma flag `--from-track` — rejeitada por
misturar dois contratos de CLI incompatíveis (um espera um plano+quadros
prontos, o outro um trajeto bruto) sob o mesmo comando.

## 10. Layout do diretório de `--keep`: três nomes fixos

**Decisão**: dentro do diretório passado a `--keep`, sempre `plan.json`,
`slice.zip` e `frames/` — nomes fixos, não escolhidos pelo usuário (a spec só
lhe dá o diretório, FR-010, Suposição "guarda os três arquivos juntos"). Ver
`contracts/intermediates-directory.md`.

**Motivo**: nomes fixos e previsíveis são o que torna o reaproveitamento
possível sem um arquivo de metadados à parte — a próxima execução sabe
exatamente onde procurar. É também o que permite a um usuário abrir o
`plan.json` num visualizador de JSON e o `slice.zip` como um ZIP comum, sem
precisar de nenhuma ferramenta própria do Sobrevoo (mesmo espírito de
`plan --export`/`geodata slice --export`, que já produzem arquivos comuns).

**Alternativas consideradas**: nomes com o hash do plano embutido (por
exemplo, `plan-<id>.json`), permitindo guardar várias execuções no mesmo
diretório — rejeitada: a spec já define o reaproveitamento como "a mesma
identificação vence, outra é recusada por padrão" (FR-010b), não "várias
versões convivem"; um diretório por execução (o usuário escolhe um diferente
para cada trajeto) já resolve o caso de uso de guardar mais de um voo.

## 11. Uma única flag de sobrescrita, para os três intermediários e para o vídeo

**Decisão**: `--overwrite` é a mesma flag, sem variante por artefato,
confirmando a Suposição de `spec.md`. `Export(plan, planPath, overwrite)`,
`Export(slice, slicePath, overwrite)`, `DrawFrames(..., Overwrite: overwrite,
...)` e `Assemble(..., Overwrite: overwrite, ...)` recebem todos o mesmo
booleano vindo da CLI.

**Motivo**: simplicidade de um único comando (a própria razão de ser da
etapa); e, na prática, o cenário em que um intermediário ficou "de outro
conjunto" é o mesmo em que o usuário mudou parâmetros de propósito — ele
já quer um vídeo novo no destino escolhido, então a mesma flag cobrindo os
dois é o comportamento esperado, não um risco de sobrescrita cega (se o
destino do vídeo aponta para um arquivo *não relacionado* que por acaso já
existe, `--overwrite` continua uma escolha explícita do usuário, como em
`video` hoje).

**Alternativas consideradas**: `--overwrite` só para o vídeo, mais uma flag
`--force`/`--refresh` separada para os intermediários — rejeitada por
complicar a interface de um comando cujo objetivo é ser simples, sem um
cenário concreto em que as duas precisassem divergir.

## 12. Nenhuma configuração nova

**Decisão**: `fly` usa exatamente `Config.PlanDefaults`, `Config.RenderDefaults`
e `Config.VideoDefaults.Quality` já resolvidos por `config.Load()` — os mesmos
valores que `plan`, `render all` e `video` já recebem — como os padrões das
mesmas flags. Nenhum campo novo em `Config`.

**Motivo**: FR-003 exige "os mesmos... padrões dos comandos existentes";
reaproveitar os mesmos campos de configuração é a única forma de garantir que
um padrão nunca diverge silenciosamente entre `plan --fps` e `fly --fps`
depois de uma mudança futura num dos dois.

**Alternativas consideradas**: nenhuma — um valor-padrão próprio da etapa 7
seria, por definição, uma segunda fonte de verdade proibida por FR-002/FR-003.
