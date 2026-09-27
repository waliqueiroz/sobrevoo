# Plano de Implementação: Voo em um Único Comando

**Branch**: `007-full-flight-pipeline` | **Data**: 2026-09-27 | **Especificação**: [spec.md](./spec.md)

**Entrada**: Especificação de funcionalidade de `/specs/007-full-flight-pipeline/spec.md`

## Resumo

Sétima etapa do Sobrevoo: um comando novo, `fly`.
`sobrevoo fly <trajeto> --output voo.mp4 [--duration][--fps][--distance][--tilt]
[--aspect][--resolution][--quality][--keep <diretório>][--overwrite]` encadeia
as seis etapas existentes — tratamento do trajeto, planejamento da câmera,
recorte dos dados geográficos já registrados, desenho dos quadros e montagem
do vídeo — atrás de um único comando, sem introduzir nenhuma regra de negócio
nova. Cada etapa interna delega inteiramente para o serviço de aplicação que já
a implementa (`CameraPlanService`, `GeoSliceService`, `FrameService`,
`VideoService`); o resultado é garantido idêntico porque é, literalmente, a
mesma chamada de serviço que o comando individual já faz.

A abordagem técnica (detalhada em `research.md`):

- Um serviço novo, `FlightService` (`Fly`), que **depende de outros serviços de
  aplicação** — o mesmo padrão que `CameraPlanService`/`GeoSliceService` já usam
  com `TrackService` (Princípio IX) — e só orquestra: nenhuma regra de negócio
  nasce nele (item 1).
- **Recusa cedo**: a disponibilidade do codificador e o destino do vídeo são
  conferidos *antes* de qualquer etapa, pela mesma checagem que
  `VideoService.Assemble` já faz internamente, agora também exposta como duas
  operações próprias (`CheckEncoder`, `CheckDestination`) que `Assemble`
  reaproveita (item 2). A cobertura dos dados registrados já é a **primeira**
  coisa que `GeoSliceService.Generate` confere, antes de ler qualquer conteúdo
  — chamar `Generate` logo após o planejamento já satisfaz a recusa cedo, sem
  nenhuma verificação nova (item 3).
- **Reaproveitamento**: usa identidades que **já existem** no domínio —
  `CameraPlan.ID()` (hash determinístico do conteúdo do plano) e
  `GeoSlice.EnsureMatches(plan)` — para decidir se um `plan.json`/`slice.zip`
  já presente no diretório guardado ainda vale; quando vale, pula a etapa cara
  (`Generate`); quando não, tenta `Export` com a política de sobrescrita do
  próprio comando — que **já** recusa por padrão um arquivo existente
  (`ErrPlanDestinationExists`/`ErrSliceDestinationExists`), sem nenhuma
  verificação nova (item 4). Os **quadros** não precisam de nenhuma lógica
  nova: `FrameService.DrawFrames`, chamado sobre o diretório de quadros (guardado
  ou temporário), já reaproveita, completa e recusa por conjunto sozinho — é
  literalmente a mesma chamada que `render all` faz (item 5).
- Quando o plano e o recorte não precisam ficar em arquivo (execução sem
  `--keep`), eles **nunca tocam disco**: os serviços recebem os valores
  `domain.CameraPlan`/`domain.GeoSlice` diretamente, em memória. Só os quadros
  exigem um diretório real (para o codificador ler); uma porta nova e mínima,
  `Workspace`, cria e remove esse diretório temporário quando `--keep` não é
  usado (item 6).
- **Progresso por etapa** (`FlightStage`, novo enum de domínio): a saída
  identifica a etapa em curso e, nas duas mais demoradas, encaminha o mesmo
  `RenderProgress`/`VideoProgress` que `render all`/`video` já emitem — a CLI
  reaproveita as mesmas funções de formatação (item 7).
- **Interrupção**: um código de saída próprio (`ErrFlightInterrupted`), que
  embrulha a interrupção de qualquer etapa — inclusive as que não têm ponto de
  cancelamento (`GeoSliceService.Generate` não recebe `context.Context`, como já
  é hoje): a checagem acontece nos pontos de retomada, com o mesmo escape de
  emergência (`SIGINT`/`SIGTERM` duplicado mata o processo, comportamento já
  existente de `signal.NotifyContext`) para uma etapa que não pode ser
  interrompida no meio (item 8).

A arquitetura hexagonal é preservada: o núcleo ganha entidades e funções puras
(`FlightRequest`, `FlightStage`, `FlightProgress`, `FlightSummary`), uma porta
nova (`Workspace`) e duas operações novas na porta de serviço `VideoService`
(que continuam delegando para as portas `VideoEncoder`/`VideoExporter` já
existentes), um erro sentinela e um serviço novo (`FlightService`). Toda E/S
continua nos adapters; a CLI só traduz flags, texto, sinais e código de saída.

## Contexto Técnico

**Linguagem/Versão**: Go 1.26 (sem mudança)

**Dependências Principais**: nenhuma nova no módulo Go. Só biblioteca padrão
(`context`, `path/filepath`, `os` — no adapter novo de `Workspace`); Cobra,
testify e `go.uber.org/mock` como antes. Nenhuma dependência de execução nova:
o `ffmpeg` continua a única, já coberta pela etapa 6.

**Armazenamento**: nenhum estado persistente novo. Sem `--keep`, o plano e o
recorte nunca tocam disco (ficam só em memória) e os quadros vivem num
diretório temporário do sistema, sempre removido ao final. Com `--keep
<diretório>`, três arquivos previsíveis nesse diretório: `plan.json`,
`slice.zip` e `frames/` — os mesmos formatos que `plan --export`, `geodata
slice --export` e `render all --output` já produzem, nada novo
(`contracts/intermediates-directory.md`).

**Testes**: mesmo padrão — `go test` com testify, given/when/then, um `t.Run`
por cenário, sem testes tabulares. O serviço novo, com as quatro *outras*
service interfaces mockadas (`mockapplication`: `CameraPlanService`,
`GeoSliceService`, `FrameService`, `VideoService`) e a porta `Workspace`
mockada (`mockdomain`) — nenhum I/O real no teste do núcleo. `VideoService`
ganha dois métodos; seus testes existentes continuam valendo, mais os dos dois
novos (que a implementação de `Assemble` passa a chamar internamente, então um
teste de `Assemble` também cobre a chamada). A CLI, com `FlightService`
mockado. O adapter `Workspace` (novo pacote `workingdir`), num diretório
temporário de teste: cria, some ao remover, erro do sistema mapeado.

**Plataforma-Alvo**: mesmo binário multiplataforma (Linux, macOS, Windows);
`os.MkdirTemp` é portável nas três.

**Tipo de Projeto**: CLI (projeto único em Go, sem frontend/backend).

**Metas de Desempenho**: nenhuma nova além das já garantidas por etapa —
`fly` soma o tempo das cinco etapas que chama; a única sobrecarga própria é a
decisão de reaproveitamento (duas comparações de identificação, uma leitura de
`plan.json`/`slice.zip` existente quando `--keep` aponta para um diretório já
preenchido), desprezível perto de qualquer uma das etapas (SC-004).

**Restrições**: 100% offline; núcleo sem E/S e sem processo; nenhuma regra de
negócio nova (FR-002); mesmo erro, mensagem e código de saída de cada etapa
para a mesma falha (FR-008), com uma única exceção deliberada, prevista pela
própria especificação: a interrupção sempre sai com o código próprio do
comando único (FR-011), não com o da etapa em que ocorreu.

**Escala/Escopo**: uso pessoal — o mesmo de cada etapa que `fly` encadeia;
nenhum limite novo.

## Verificação da Constituição

*PORTÃO: Deve passar antes da Fase 0 de pesquisa. Reverificar após o design da Fase 1.*

Avaliada antes da pesquisa e **reavaliada após o design** (data-model,
contratos e quickstart); o resultado não mudou.

| Princípio | Avaliação | Como o design atende |
|---|---|---|
| I. Arquitetura Hexagonal | PASSA | `FlightService` só chama outros serviços de aplicação e a porta `Workspace`, na ordem certa; nenhuma regra de negócio própria. As duas identidades usadas para decidir reaproveitamento (`CameraPlan.ID()`, `GeoSlice.EnsureMatches`) já são domínio puro, existentes desde as etapas 3 e 4. `FlightStage`/`FlightProgress`/`FlightSummary` são tipos de domínio sem E/S. |
| II. Portas para Toda Dependência Externa | PASSA | A única E/S nova — criar/remover um diretório temporário — passa pela porta `Workspace` (novo, papel: fornecer o espaço de trabalho de uma execução). Ler/gravar `plan.json`/`slice.zip`/quadros continua atrás de `CameraPlanExporter/Reader`, `GeoSliceExporter/Reader`, `FrameRepository` — portas já existentes, chamadas pelos serviços que já as possuem. Nenhuma porta nova para `filepath.Join` (função pura). |
| III. Entrypoints Descartáveis | PASSA | `FlightService.Fly` recebe e devolve tipos de domínio (`domain.FlightRequest`, `domain.FlightSummary`); nada de flags, texto ou terminal. A CLI só traduz flags → `FlightRequest`, progresso → linhas com o rótulo da etapa, sinais → `context`, resultado/erro → texto/código — reaproveitando as mesmas funções de formatação de `plan`/`render all`/`video` (nenhuma duplicada). |
| IV. Neutralidade Geográfica | PASSA | Nenhum dado, constante ou caso especial de região: a etapa só encadeia chamadas. FR-012/SC herdam a neutralidade já garantida por cada etapa chamada. |
| V. Funcionamento Offline | PASSA | Nenhuma dependência nova de rede ou serviço externo; usa só os dados já registrados e o mesmo `ffmpeg` local da etapa 6. |
| VI. Testes Automatizados no Núcleo | PASSA | `FlightService` testado com `CameraPlanService`, `GeoSliceService`, `FrameService`, `VideoService` e `Workspace` todos mockados (`mockapplication`/`mockdomain`); nenhum teste do núcleo toca disco, rede ou processo externo. |
| VII. Erros Sentinela no Domínio | PASSA | Um erro novo, `ErrFlightInterrupted`, em `errors.go`; a CLI o traduz no código 51 (`contracts/cli.md`). Toda falha de negócio de uma etapa chamada continua sendo o erro sentinela dessa etapa, sem tradução nem duplicação — `FlightService` só embrulha a interrupção, nunca outro erro. |
| VIII. Configuração Injetada | PASSA | `FlightService` não lê configuração nenhuma: os mesmos `PlanDefaults`, `RenderDefaults`, `VideoDefaults` já resolvidos por `config.Load()` continuam chegando pela CLI, que os passa como os valores-padrão das mesmas flags que `plan`/`render all`/`video` já usam (Princípio VIII, sem alteração). |
| IX. Organização de Portas, Service Layer e Mocks | PASSA | Sem `ports.go`: `Workspace` em `workspace.go` (porta sem entidade dona, como `Simplifier`/`Smoother`); `FlightRequest`/`FlightStage`/`FlightProgress`/`FlightSummary` em `flight.go`. Um serviço por recurso, sem `Execute`: `FlightService`/`flightService`/`NewFlightService` (`Fly`), que **depende de outros serviços de aplicação** — padrão já usado por `CameraPlanService`/`GeoSliceService` com `TrackService`. `VideoService` ganha `CheckEncoder`/`CheckDestination` como operações nomeadas do mesmo recurso (não um "Execute" genérico), e `Assemble` passa a chamá-las internamente (sem duplicar a lógica). Adapter novo: `workingdir.NewOS()` (pacote de estratégia única, como `filechecker`/`geodatainspector` — `New()` e um tipo curto). Nenhum nome exportado repete o pacote. Mocks por `//go:generate` acima de cada interface. |
| X. Testes: Given/When/Then, Builders e Isolamento por Camada | PASSA | Todo teste em `t.Run("should ...")` com `// given`/`// when`/`// then`, sem tabelas; builder novo `builddomain.NewFlightRequestBuilder()` (bastante campo); `FlightService` com os quatro serviços e a porta mockados (`mockapplication`/`mockdomain`); CLI com `FlightService` mockado. Sem teste de ponta a ponta automatizado; `quickstart.md` cobre o manual, encadeando `fly` com um trajeto de amostra e comparando com o fluxo manual. |
| Idioma dos Artefatos | PASSA | Artefatos do Spec Kit em português; identificadores, pacotes, arquivos, comentários e mensagens de commit em inglês; E/S em tempo de execução (flags, saída, erros) em inglês, como nas etapas anteriores. |

Nenhuma violação identificada. Rastreamento de Complexidade não se aplica.

## Estrutura do Projeto

### Documentação (desta funcionalidade)

```text
specs/007-full-flight-pipeline/
├── plan.md              # Este arquivo (saída do comando /speckit-plan)
├── research.md          # Saída da Fase 0 (comando /speckit-plan)
├── data-model.md        # Saída da Fase 1 (comando /speckit-plan)
├── quickstart.md        # Saída da Fase 1 (comando /speckit-plan)
├── contracts/
│   ├── cli.md                          # Saída da Fase 1: o comando `fly`
│   └── intermediates-directory.md      # Saída da Fase 1: o layout de `--keep` e a regra de reaproveitamento
├── checklists/
│   └── requirements.md  # Gerado por /speckit-specify
└── tasks.md              # Saída da Fase 2 (comando /speckit-tasks - NÃO criado pelo /speckit-plan)
```

### Código-Fonte (raiz do repositório)

```text
cmd/sobrevoo/
└── main.go                                    # (alterado) monta workingdir.NewOS, FlightService e o comando "fly"

internal/
├── domain/
│   ├── flight.go                               # NOVO: FlightRequest, FlightStage (+String), FlightProgress, FlightSummary
│   ├── workspace.go                             # NOVO: porta Workspace (no topo do arquivo)
│   ├── errors.go                                # (estendido) ErrFlightInterrupted
│   ├── video.go                                  # (estendido) VideoService não muda — CheckEncoder/CheckDestination são application, não domain
│   ├── flight_test.go / workspace_test.go        # NOVOS
│   ├── builddomain/
│   │   └── flight_request_builder.go             # NOVO: NewFlightRequestBuilder()
│   └── mockdomain/
│       └── workspace.go                          # GERADO (make generate)
├── application/
│   ├── flight_service.go                         # NOVO: FlightService / flightService / NewFlightService (Fly)
│   ├── flight_service_test.go                    # NOVO
│   ├── video_service.go                          # (alterado) +CheckEncoder, +CheckDestination; Assemble passa a chamá-las
│   ├── video_service_test.go                     # (alterado)
│   └── mockapplication/
│       ├── flight_service.go                      # GERADO
│       └── video_service.go                       # REGERADO (dois métodos novos)
└── infra/
    ├── inbound/cli/
    │   ├── fly.go                                 # NOVO: comando "fly" (todas as flags reaproveitadas + --keep), contexto de sinais, resumo
    │   ├── fly_progress.go                        # NOVO: anúncio de etapa (stderr) + delega a render/video para o progresso por quadro
    │   ├── fly_test.go / fly_progress_test.go / fly_signal_test.go  # NOVOS
    │   ├── exit_code.go                           # (estendido) código 51
    │   └── exit_code_test.go                      # (estendido)
    └── outbound/
        └── workingdir/
            ├── os_workspace.go                    # NOVO: Workspace.NewTemporary (os.MkdirTemp / os.RemoveAll)
            └── os_workspace_test.go                # NOVO
```

**Documentação de apoio, alterada na implementação**: `CLAUDE.md` e `README.md`
(sétima etapa: o comando `fly`, o serviço, a porta e o adapter novos, a
observação de que nenhuma regra de negócio nasce aqui).

**Decisão de Estrutura**: mesmo projeto único em Go, seguindo as convenções
consolidadas: um arquivo de domínio por responsabilidade, porta sem entidade
dona em arquivo próprio (`workspace.go`, como `simplification.go`), adapter de
estratégia única com `New()` (`workingdir`, como `filechecker`), composition
root em `cmd/sobrevoo/main.go`.

**Ordem de execução (base para o `/speckit-tasks`)**: (1) `VideoService`:
`CheckEncoder`/`CheckDestination`, com `Assemble` refatorado para chamá-los —
`make test` verde, sem mudar nenhum comportamento observável de `video`; (2)
domínio da etapa 7, puro e sem E/S: `Workspace` (porta), `FlightRequest`,
`FlightStage`, `FlightProgress`, `FlightSummary`, `ErrFlightInterrupted`; (3)
adapter `workingdir.NewOS()` (com o teste de diretório temporário real — é o
único ponto de E/S da etapa); (4) `FlightService.Fly`, com as quatro service
interfaces e `Workspace` mockados: primeiro o caminho sem `--keep` (Histórias
1-3), depois o reaproveitamento com `--keep` (História 4); (5) CLI: comando
`fly`, progresso por etapa, resumo, código de saída 51, `main.go`; (6)
`quickstart.md` (fluxo completo comparado ao manual, reaproveitamento medido),
`CLAUDE.md`, `README.md`. As Histórias P1 a P4 da spec são entregáveis nesta
ordem de prioridade (comando único → parâmetros ajustáveis → progresso por
etapa → guardar/reaproveitar): a **História 1** (P1, sem `--keep`) já é o MVP
completo — as demais só acrescentam flags de repasse e a lógica de
reaproveitamento sobre o mesmo núcleo.

## Rastreamento de Complexidade

Nenhuma violação da constituição; nada a justificar.

## Riscos e Pontos de Atenção

- **R1 — Granularidade da interrupção**: `CameraPlanService.Generate` e
  `GeoSliceService.Generate` não recebem `context.Context` (nunca receberam,
  nem nas etapas 3 e 4 isoladas). `Ctrl+C` durante essas duas etapas só é
  percebido no próximo ponto de checagem, depois que a chamada síncrona
  retorna — não durante ela. Não é uma regra nova nem uma regressão: é o
  mesmo comportamento que `plan` e `geodata slice`, rodados isoladamente, já
  têm hoje (nenhum dos dois responde a `Ctrl+C` no meio do cálculo). Um
  segundo `Ctrl+C`/`SIGTERM`, se necessário, ainda mata o processo na hora —
  `signal.NotifyContext` só intercepta o primeiro sinal, comportamento já
  existente e não alterado aqui. `quickstart.md` documenta isso explicitamente
  para o usuário não esperar uma resposta instantânea durante o recorte.
- **R2 — Reaproveitamento e mudança do registro de dados geográficos entre
  execuções**: se o usuário registra, remove ou substitui uma fonte de dados
  geográficos entre duas execuções de `fly --keep`, um `slice.zip` reaproveitado
  (porque `PlanID` ainda bate) pode não refletir o registro atual. É o mesmo
  comportamento que já vale hoje rodando `geodata slice` de novo por conta
  própria depois de mexer no registro — não uma regra nova desta etapa
  (Suposição da spec). Documentado, não corrigido.
- **R3 — `--keep` apontando para um diretório com conteúdo alheio não
  reconhecido**: preservado e ignorado, tanto pelo `plan.json`/`slice.zip`
  (que só olham para o nome fixo que `fly` usa) quanto pelos quadros (a mesma
  regra que `render all` já aplica ao diretório de destino).
- **R4 — `EnsureDirectory` (achado na validação manual)**: o desenho original
  assumia que `CameraPlanService.Export`/`GeoSliceService.Export` bastavam
  para criar o `--keep` na primeira execução — mas os dois só criam o
  arquivo, exigindo que a pasta-mãe já exista (mesma regra de `plan
  --export`). Um `--keep` para um diretório que ainda não existe falhava com
  `plan destination is not writable`. Corrigido com um método novo na porta
  `Workspace` (`EnsureDirectory`, `os.MkdirAll`), chamado antes de tentar
  reaproveitar ou gravar o plano — a mesma resolução que `render all
  --output` já dá ao próprio diretório de quadros.
- **R5 — Identidade dos quadros e o `ContentID` do recorte (achado na
  validação manual)**: `FrameMark.SetID` inclui `GeoSlice.ContentID`, que só
  a **leitura** de um recorte já gravado preenche (um recorte recém-gerado
  não tem arquivo, e portanto não tem `ContentID`, por contrato do próprio
  tipo). A primeira execução com `--keep` desenhava os quadros com o recorte
  recém-gerado (`ContentID` vazio); uma execução seguinte, ao reaproveitar o
  recorte pelo arquivo (`ContentID` preenchido), calculava um `SetID`
  diferente para o **mesmo** recorte — os quadros da primeira execução nunca
  batiam com os da segunda, quebrando o reaproveitamento (`ErrFrameSetConflict`)
  bem no caso mais comum: gerar e guardar, depois reaproveitar. Corrigido:
  depois de gerar e exportar um recorte novo, `FlightService` o **relê** do
  arquivo (`GeoSliceService.Load`) antes de desenhar, para os quadros
  carregarem a mesma identidade que uma execução futura, reaproveitando-os,
  vai calcular. Confirmado com uma interrupção real no meio do desenho e uma
  retomada: antes da correção, "quadros de outro conjunto"; depois, os
  quadros prontos são mantidos (`N kept`) e só o resto é desenhado.
- **R6 — `--overwrite` força o redesenho mesmo de quadros que batem**: como a
  mesma flag vale para os três intermediários e para o vídeo (`research.md`
  item 11), e `render all --overwrite` **sempre** redesenha tudo (mesmo um
  conjunto que já bate — contrato já existente da etapa 5), pedir `--keep`
  com o **mesmo** destino de vídeo de uma execução anterior (que por isso
  precisa de `--overwrite` só por causa do vídeo) também redesenha os
  quadros à toa, mesmo que o plano e o recorte continuem os mesmos. Não é uma
  regra nova nem um contorno — é `render all --overwrite` fazendo exatamente
  o que já faz sozinho hoje. `quickstart.md` documenta isso: reaproveitar de
  verdade pede um destino de vídeo que ainda não existe (ou uma execução sem
  precisar de `--overwrite`).
- **R7 — Diretório temporário não removido por `SIGKILL`**: se o próprio
  Sobrevoo for morto com `SIGKILL` (não interceptável), o `defer` que chama
  `Workspace.NewTemporary`'s função de remoção nunca roda, e um diretório
  temporário fica para trás em `os.TempDir()`. Mesmo risco documentado para o
  arquivo temporário do vídeo na etapa 6 (R7 de `specs/006-video-assembly/plan.md`);
  sem limpeza de órfãos nesta etapa.
