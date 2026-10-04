# Pesquisa: Níveis de tratamento em `plan` e `fly`

**Feature**: `013-treatment-level-flags` | **Data**: 2026-10-03

Sem nenhum `NEEDS CLARIFICATION` pendente do Contexto Técnico — a única
ambiguidade real da spec (o que fazer com um `--keep` sem nível registrado,
de antes desta etapa) já foi resolvida na sessão de `/speckit-clarify` e
está em `spec.md`. O que segue são as decisões de desenho tomadas para
encaixar as duas flags novas no código existente sem duplicar nem contornar
nenhuma regra (Princípio IX) — e, em boa parte, para constatar que o
mecanismo já existente resolve a maior parte do pedido sem nenhum código
novo de regra de negócio.

## 1. Onde os dois níveis vivem

**Decisão**: dois campos novos em `domain.PlanParameters`
(`internal/domain/camera_plan_parameters.go`): `Simplification Level` e
`Smoothing Level`, ao lado de `Distance`/`Tilt`, já do mesmo tipo `Level`.

**Justificativa**: `PlanParameters` já é, por definição, "as escolhas que um
usuário faz sobre um plano de câmera" (comentário do próprio tipo) — os dois
níveis são exatamente isso, da mesma natureza que `Distance`/`Tilt`: um
valor fechado de três opções, resolvido pela CLI antes de chegar ao domínio,
sem necessidade de validação de intervalo (ao contrário de `Duration`/
`FrameRate`, que têm faixas numéricas). Colocá-los em `PlanParameters` — e
não em `FlightRequest` diretamente, nem num tipo novo — significa que tudo o
que já consome `PlanParameters` (a exportação do plano, `CameraPlan.ID()`, o
resumo impresso) os recebe de graça, sem precisar aprender sobre um
parâmetro novo em cada lugar separadamente.

**Alternativas consideradas**:
- Um parâmetro à parte em `CameraPlanService.Generate(reader, parameters,
  simplification, smoothing)`: rejeitada — duplicaria, na assinatura do
  serviço, dois valores que já cabem dentro do DTO que a própria assinatura
  já recebe; também exigiria um campo a mais em `FlightRequest` só para
  repassar os mesmos dois valores até `Generate`, quando `FlightRequest.
  Parameters` já é o `PlanParameters` que se quer estender.
- Um tipo novo, `TreatmentLevels{Simplification, Smoothing Level}`, embutido
  em `PlanParameters`: rejeitada — dois campos do mesmo tipo `Level` que
  `Distance`/`Tilt` já são não justificam um tipo próprio; seria uma camada
  de indireção sem ganho (a mesma economia que levou `BoundingBox`/`Level`
  a serem usados diretamente em vez de embrulhados).

## 2. Como `cameraPlanService.Generate` deixa de ter um nível padrão próprio

**Decisão**: `cameraPlanService` perde o campo `defaultLevel` (e o parâmetro
correspondente de `NewCameraPlanService`); `Generate` chama
`s.trackService.Treat(reader, parameters.Simplification,
parameters.Smoothing)` em vez de `s.trackService.Treat(reader, s.
defaultLevel, s.defaultLevel)`.

**Justificativa**: hoje o serviço resolve o nível "no lugar do usuário"
porque é o único lugar que conhece um nível — com os dois campos novos em
`PlanParameters`, a CLI já resolveu o valor efetivo (informado ou padrão)
antes de chamar `Generate`, exatamente como já faz para `Distance`/`Tilt`,
para `Appearance` e para `Resolution`. Manter `defaultLevel` como campo do
serviço, ao lado dos dois campos novos em `parameters`, criaria duas fontes
de verdade para "qual nível usar" dentro do mesmo método — e um bug de dia
um, caso a CLI algum dia esquecesse de preencher `parameters.Simplification`
antes de chamar `Generate` e o serviço silenciosamente usasse outro valor em
seu lugar. Removê-lo é consistente com o padrão que a etapa 8 já estabeleceu
(nenhum serviço resolve "o padrão" por conta própria; a CLI sempre entrega
um valor já concreto).

**Alternativas consideradas**:
- Manter `defaultLevel` no serviço como *fallback* para quando
  `parameters.Simplification`/`.Smoothing` vierem zero-value: rejeitada —
  `Level` é um `int` (`LevelLow = iota`), então o zero-value é `LevelLow`,
  não "ausente"; usar isso como sinal de "não informado" inverteria
  silenciosamente o padrão de `medium` para `low` sempre que alguém
  construísse um `PlanParameters` sem pensar nos dois campos novos (um
  teste, por exemplo) — exatamente o tipo de armadilha que builders com
  defaults explícitos (`builddomain.PlanParametersBuilder`) já existem para
  evitar, e que esta decisão evita de raiz ao não depender do zero-value
  para nada.

## 3. Onde a identidade do plano passa a incluir os dois níveis

**Decisão**: `CameraPlan.ID()` (`internal/domain/camera_plan.go`) ganha duas
escritas no hash, `write(int64(c.Parameters.Simplification))` e
`write(int64(c.Parameters.Smoothing))`, na mesma posição/padrão das duas já
existentes para `Distance`/`Tilt`.

**Justificativa**: é literalmente a mesma linha que já existe duas vezes
(`write(int64(c.Parameters.Distance))`, `write(int64(c.Parameters.Tilt))`),
repetida para os dois campos novos — cumpre FR-006 (dois planos com níveis
diferentes nunca são equivalentes) sem nenhuma lógica nova, e sem precisar
de um `RenderVersion`/formato de hash novo (o comentário de `ID()` já diz
que o hash é "dos parâmetros efetivos e de cada quadro" — os níveis são
exatamente isso, um parâmetro efetivo que faltava). Como as frames já
carregam o efeito geométrico do tratamento (posições, distâncias, duração),
isto reforça — não substitui — o que o hash dos quadros já captura
indiretamente; a razão de ainda escrever os níveis explicitamente, em vez de
confiar só no efeito geométrico, é a mesma que já vale para `Distance`/
`Tilt`: um trajeto curto ou simples pode, em tese, produzir a mesma
geometria tratada para dois níveis diferentes (poucos pontos para
simplificar), e nesse caso só o parâmetro explícito distingue os dois
planos.

**Efeito colateral útil, sem código novo**: `FlightService.reusePlan` já
decide reaproveitar um plano guardado por `--keep` comparando `existing.
ID() == plan.ID()` (`internal/application/flight_service.go`) — estender
`ID()` já basta para FR-008 (o reaproveitamento respeitar os dois níveis),
exatamente como a etapa 8 (aparência) e a etapa 9 (sobreposição) já
descrevem para `NewFrameSetID`. Nenhuma condição nova entra em
`reusePlan`.

## 4. Como as duas flags entram em `plan` e em `fly`

**Decisão**: `parsePlanParameters` (`internal/infra/inbound/cli/plan.go`) —
já a única função que `plan.go` e `fly.go` chamam para interpretar
`--duration`/`--fps`/`--distance`/`--tilt`/`--aspect` — ganha dois
parâmetros de flag a mais, `simplificationFlag`/`smoothingFlag string`, e as
preenche com `parameters.Simplification, err = parseLevel(simplificationFlag)`/
`parameters.Smoothing, err = parseLevel(smoothingFlag)` — a mesma função
`parseLevel` que `--distance`/`--tilt` já chamam ali (e que `inspect.go` já
usa para as mesmas duas flags, com o mesmo nome). `NewPlanCommand` e
`NewFlightCommand` registram as duas flags com
`cmd.Flags().StringVar(&simplificationFlag, "simplification",
levelName(defaults.Simplification), ...)` (e o equivalente para
`smoothing`), a mesma forma como `--distance`/`--tilt` já são registradas
nos dois comandos.

**Justificativa**: `parseLevel`/`levelName` já são funções genéricas do
pacote `cli` (não específicas de `inspect`: `plan.go` já as reaproveita para
`--distance`/`--tilt` hoje) — não há necessidade de um arquivo de parser
compartilhado novo, ao contrário das etapas 8/9/10 (`appearance.go`,
`overlay.go`, `source_selection.go`), que precisaram de um porque a
validação ali é mais rica (cores, proporções, intervalos) do que um enum de
três valores já resolvido. `parsePlanParameters` já é o ponto único que
garante, por construção, que `plan` e `fly` aceitam `--duration` etc. com o
mesmo efeito — estendê-la, em vez de duplicar a lógica em cada comando, é a
mesma garantia por construção que FR-001/FR-002 pedem para as duas flags
novas.

**Alternativas consideradas**:
- Duas flags cada uma com seu próprio parsing dentro de `plan.go`/`fly.go`,
  fora de `parsePlanParameters`: rejeitada — duplicaria o parsing entre os
  dois comandos, o problema exato que `parsePlanParameters` já existe para
  evitar.

## 5. Onde o padrão de configuração dos dois níveis entra

**Decisão**: `domainPlanParameters` (`cmd/sobrevoo/config_mapping.go`) ganha
um segundo parâmetro, `defaultLevel config.Level`, e preenche
`Simplification`/`Smoothing` com `domainLevel(defaultLevel)` — a mesma
função de mapeamento que já existe e que `main.go` já usa para `inspect`
(`domainLevel(cfg.DefaultLevel)`). Os dois call sites de
`domainPlanParameters` em `main.go` (para `plan` e para `fly`) passam a
chamar `domainPlanParameters(cfg.PlanDefaults, cfg.DefaultLevel)`.

**Justificativa**: `cfg.DefaultLevel` já é o único valor de configuração que
`inspect` usa para as mesmas duas escolhas (um só campo para simplificação e
suavização, como `CLAUDE.md`/`config.go` já documentam) — FR-001/FR-002
pedem exatamente "o mesmo padrão vindo da configuração que `inspect` já
usa", então não há padrão novo para inventar, só um fio novo levando o
mesmo valor até `domainPlanParameters`. Isto está deliberadamente fora do
escopo "mudar o padrão da configuração ou como ele é lido" da entrada da
spec, porque não muda: nenhum campo novo entra em `config.go`.

## 6. Como o arquivo de plano exportado grava e lê os dois níveis

**Decisão**: `parametersFile`/`readParameters`
(`internal/infra/outbound/jsonfile/camera_plan_file.go`/
`camera_plan_reader.go`) ganham dois campos de texto,
`"simplification"`/`"smoothing"`, escritos por `levelText(plan.Parameters.
Simplification)`/`levelText(plan.Parameters.Smoothing)` — a mesma função já
usada para `distance`/`tilt` — e lidos por `parseLevel(file.Parameters.
Simplification)`/`parseLevel(file.Parameters.Smoothing)` — a mesma função já
usada para `distance`/`tilt` no `Read()`. `format_version` continua `2`.

**Justificativa**: `levelText`/`parseLevel` (do pacote `jsonfile`, distintas
das funções de mesmo nome do pacote `cli`, mas com o mesmo contrato) já
implementam exatamente "texto desconhecido ou vazio lê como `medium`"
(comentário de `parseLevel`: "an unknown text reads as medium, the default
level") — um arquivo de antes desta etapa, sem os dois campos, faz o JSON
desserializar `""` para os dois `string`s, e `parseLevel("")` já cai no
`default: return domain.LevelMedium`. Isso é, bit a bit, a decisão que a
sessão de `/speckit-clarify` tomou (ausência de registro = nível padrão
implícito) — sem precisar de um ponteiro (`*string`) como `aspect_ratio`
usa, porque ali o padrão de ausência (`16:9`, paisagem) é *diferente* do
padrão de uma flag não informada hoje, exigindo distinguir "ausente" de
"presente com o valor que por acaso bate com o padrão"; aqui os dois padrões
são o mesmo valor (`medium`), então a distinção é irrelevante e o caminho
mais simples (string lida como está, convertida pela mesma função que já
teria convertido um valor explícito) já é correto. `format_version`
continua `2` pelo mesmo motivo que `aspect_ratio` não subiu a versão ao ser
acrescentado (ver `specs/003-camera-path-planning/contracts/plan-file.md`,
linha de `format_version`): um campo a mais, nunca obrigatório, nunca muda
de significado um campo existente.

**Alternativas consideradas**:
- Pointer (`*string`) como `aspect_ratio`, distinguindo ausência de
  presença: rejeitada por desnecessária — só se justificaria se o padrão
  para "ausente" precisasse ser diferente do padrão para "presente mas
  desconhecido/vazio", o que não é o caso aqui (os dois já convergem para
  `medium`).
- Bump de `format_version` para `3`: rejeitada — nenhum campo existente
  muda de tipo, posição ou significado, e os dois campos novos nunca são
  obrigatórios (ao contrário dos três campos que a etapa 9 tornou
  obrigatórios e que por isso, sim, pediram a versão `2`).

## 7. Nenhum sentinela de erro novo, nenhum código de saída novo

**Decisão confirmada (não uma decisão nova)**: um valor fora de
`low`/`medium`/`high` para `--simplification`/`--smoothing`, em `plan` ou em
`fly`, é um erro de uso da CLI (`newUsageError`, código de saída `2`) — a
mesma categoria de erro que `--distance`/`--tilt` já produzem hoje para o
mesmo engano (`parseLevel` devolve um `error` comum, não um sentinela do
domínio). Nenhuma entrada em `exit_code.go` muda; o próximo código livre
(`59`) não é usado por esta etapa.

**Justificativa**: FR-003 pede exatamente "a mesma mensagem de erro que
`inspect` já usa" — `inspect` já trata isso como erro de uso, não como
sentinela do domínio, então reaproveitar a mesma categoria é o que a spec
pede, não uma escolha nova.

## 8. `render frame`, `render all`, `video` e `geodata check` ficam fora

**Decisão confirmada (não uma decisão nova)**: nenhum dos quatro recebe as
duas flags novas. `render frame`/`render all`/`video` operam sobre um plano
e um recorte já gerados — o trajeto GPS já foi tratado quando esses comandos
rodam, e reler o trajeto ali duplicaria, não reaproveitaria, a decisão já
tomada por `plan`/`fly`. `geodata check` chama `TrackService.Clean`, nunca
`TrackService.Treat` (`internal/application/geo_data_service.go`,
`CheckCoverage`) — a verificação de cobertura é deliberadamente feita sobre
o trajeto limpo, não simplificado/suavizado (para não mascarar uma lacuna
real de cobertura, `specs/002-geo-data-registry/research.md` item 9), então
os dois níveis não têm nenhum efeito ali para expor. Fica registrado aqui só
para não ser levantado como dúvida durante a implementação.
