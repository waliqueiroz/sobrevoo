# Plano de Implementação: Planejamento do Movimento de Câmera

**Branch**: `003-camera-path-planning` | **Data**: 2026-09-19 | **Especificação**: [spec.md](./spec.md)

**Entrada**: Especificação de funcionalidade de `/specs/003-camera-path-planning/spec.md`

**Nota**: Este template é preenchido pelo comando `/speckit-plan`; sua definição descreve o fluxo de execução.

## Resumo

Terceira etapa do Sobrevoo: um novo comando `sobrevoo plan` que, a partir
de um trajeto GPS já tratado pelo pipeline da etapa 1, calcula um **plano de
câmera** determinístico — para cada quadro do futuro vídeo, a posição da
câmera (latitude, longitude, altitude), a direção, a inclinação e a posição
do marcador da atividade — e o expõe como resumo no terminal e, opcionalmente,
como arquivo JSON versionado. A duração do vídeo é calculada a partir da
extensão do trajeto quando o usuário não a informa (curva sublinear, entre
20 s e 120 s, nunca abaixo do mínimo para um voo suave) e respeitada
exatamente quando informada. Nenhum mapa, quadro ou vídeo é gerado.

A abordagem técnica (detalhada em `research.md`) modela a câmera como
**alvo + deslocamento orbital** (rumo, inclinação, distância) sobre um
**plano tangente local** (projeção azimutal equidistante centrada no
centroide do trajeto, o que torna o comportamento idêntico no antimeridiano
e em latitudes altas). O tempo do vídeo é uma função linear do "tempo
efetivo" do trajeto (horário real com paradas longas comprimidas, ou
distância quando não há horário). O rumo vem de uma janela gaussiana de
tangentes, escalada pela distância da câmera, para não oscilar em retornos
e voltas. Todas as trilhas (rumo, inclinação, zoom, alvo) passam por
limitação de taxa relativa em passe direto e inverso mais suavização
gaussiana, e os trechos em que a limitação atuou viram os "trechos
suavizados" do resumo. Abertura e fechamento interpolam, com *smoothstep*,
entre uma pose de visão geral e a pose de acompanhamento.

A arquitetura hexagonal é preservada: o núcleo ganha entidades e funções
puras de câmera (`CameraPlan`, `CameraFrame`, `PlanParameters`,
`CameraTuning`, `PlanCamera`, `MinimumDuration`, `DefaultDuration`), uma única porta nova
(`CameraPlanExporter`), sete erros sentinela e um serviço por recurso,
`CameraPlanService` (`Generate`, `Export`). Toda escrita de arquivo fica no
adapter `jsonfile`; a CLI apenas traduz flags e erros.

**Refatoração prévia das etapas 1 e 2 (`research.md` item 13)**: o
tratamento do trajeto (`Parse` → `CleanTrack` → `Simplify` → `Smooth`) hoje
está duplicado em `InspectTrackService` e `GeoDataService`. Antes de criar
o `CameraPlanService`, essa orquestração é extraída para um único
`TrackService` (`Clean`, `Treat`, `Inspect`), que absorve o
`InspectTrackService` (serviço de método único, anterior à redação atual do
Princípio IX). `GeoDataService` e `CameraPlanService` passam a depender
dele, e deixam de conhecer `TrackParser`, `Simplifier`, `Smoother` e os
limiares de limpeza. Nenhum comportamento das etapas 1 e 2 muda.

## Contexto Técnico

**Linguagem/Versão**: Go 1.26 (sem mudança)

**Dependências Principais**: nenhuma nova. Somente biblioteca padrão
(`math`, `time`, `encoding/json`, `os`) além de Cobra, testify e
`go.uber.org/mock`, já existentes.

**Armazenamento**: nenhum estado persistente. Único artefato externo: o
arquivo JSON exportado sob demanda (`contracts/plan-file.md`), escrito de
forma atômica e sem sobrescrita por padrão (`research.md` item 11).

**Testes**: mesmo padrão já estabelecido — `go test` com testify,
given/when/then, um `t.Run` por cenário, sem testes tabulares; builders
`CameraFrameBuilder`, `CameraPlanBuilder` e `PlanParametersBuilder` em
`builddomain`; mock gerado da nova porta (`CameraPlanExporter`) e dos serviços
`CameraPlanService` e `TrackService` (`mockapplication`). O algoritmo é testado como funções
puras de domínio, sem mock, com trajetos sintéticos (reta, círculo, retorno,
U, antimeridiano, latitude alta) e **testes de propriedade** que varrem todos
os pares de quadros consecutivos verificando os limites de suavidade
(FR-006), a monotonicidade do marcador (FR-013) e a contagem de quadros
(FR-004). A CLI é testada com `CameraPlanService` mockado; o adapter
`jsonfile.CameraPlanExporter` é testado em diretório temporário, incluindo
recusa de sobrescrita e ausência de arquivo parcial. Não há teste
automatizado de ponta a ponta; `quickstart.md` é o checklist manual.

**Plataforma-Alvo**: mesmo binário multiplataforma (Linux, macOS, Windows)
das etapas anteriores.

**Tipo de Projeto**: CLI (projeto único em Go, sem frontend/backend).

**Metas de Desempenho**: SC-001 — resumo e plano completo em menos de 5 s
para um trajeto de até 10 000 pontos com os parâmetros padrão. O custo
dominante é linear em `quadros × janela gaussiana` (dezenas de milhares de
operações por quadro no pior caso); a janela é truncada em ±3σ. Sem
concorrência (também por determinismo, `research.md` item 9).

**Restrições**: 100% offline; núcleo sem I/O; determinismo bit a bit no
mesmo binário e máquina (valores quantizados, `research.md` item 9); nenhum
dado de região embutido; extensão do trajeto entre 50 m e 2 000 km;
frequência entre 1 e 120 fps; duração automática entre 20 s e 120 s (ou o
mínimo do trajeto, se maior), quando `--duration` não é informada.

**Escala/Escopo**: trajetos pessoais de metros a centenas de quilômetros,
até dezenas de milhares de pontos (a mesma escala da etapa 1); planos de
até alguns milhares de quadros no uso típico.

## Verificação da Constituição

*PORTÃO: Deve passar antes da Fase 0 de pesquisa. Reverificar após o design da Fase 1.*

Avaliada antes da pesquisa e **reavaliada após o design** (data-model,
contratos e quickstart); o resultado não mudou.

| Princípio | Avaliação | Como o design atende |
|---|---|---|
| I. Arquitetura Hexagonal | PASSA | Todo o algoritmo (projeção, linha do tempo, suavização, abertura/fechamento) é função pura em `internal/domain`, sem importar `os`, `encoding/json` para I/O, Cobra ou qualquer pacote de `internal/infra`. `CameraPlanService` em `internal/application` só orquestra (chama `TrackService.Treat`, `domain.PlanCamera` e a porta de exportação). A escrita do arquivo é um adapter (`jsonfile`). |
| II. Portas para Toda Dependência Externa | PASSA | A única E/S nova — gravar o plano em disco — passa pela porta `CameraPlanExporter`, declarada no domínio, implementada em `internal/infra/outbound/jsonfile`. A leitura do trajeto reaproveita `TrackParser` (a CLI abre o arquivo e passa um `io.Reader`, como em `inspect`). Nenhuma porta para `math`, `time` ou outra função pura da biblioteca padrão. |
| III. Entrypoints Descartáveis | PASSA | `CameraPlanService` recebe e devolve tipos de domínio; não conhece flags, texto, códigos de saída ou Cobra. O comando `plan` é só tradução de flags → `PlanParameters` e de `CameraPlan`/erros → texto/código de saída. Formatar o resumo é responsabilidade do adapter; o resumo em si (`PlanSummary`) é do domínio. |
| IV. Neutralidade Geográfica | PASSA | Nenhum dado, constante ou caso especial de região. A projeção azimutal equidistante centrada no centroide por vetores unitários trata antimeridiano, hemisférios e latitudes altas com a mesma fórmula (`research.md` item 2); SC-007 verifica que trajetos equivalentes deslocados dão o mesmo resultado. |
| V. Funcionamento Offline | PASSA | Nenhuma rede, chave ou serviço; nenhum uso do registro da etapa 2; sem relógio (o resultado não depende de `time.Now()`). |
| VI. Testes Automatizados no Núcleo | PASSA | Domínio testado sem I/O; `CameraPlanService` testado com `TrackService` (`mockapplication`) e `CameraPlanExporter` (`mockdomain`) mockados; `TrackService` testado com `TrackParser`, `Simplifier` e `Smoother` mockados (`mockdomain`); nenhum teste do núcleo toca disco. |
| VII. Erros Sentinela no Domínio | PASSA | `ErrInvalidDuration`, `ErrInvalidFrameRate`, `ErrDurationTooShort`, `ErrTrackTooShort`, `ErrTrackTooLarge`, `ErrPlanDestinationExists`, `ErrPlanDestinationInvalid` declarados em `errors.go`; a CLI os traduz em códigos 10–16 (`contracts/cli.md`). Erros com dados variáveis usam `fmt.Errorf("%w: ...")`, preservando `errors.Is`. O adapter `jsonfile` traduz erros do `os` (`fs.ErrExist`, permissão, diretório inexistente) para os sentinelas; o núcleo nunca vê um erro de `os`. |
| VIII. Configuração Injetada | PASSA | Todos os valores heurísticos ficam em `domain.CameraTuning` (formato definido pelo núcleo), preenchida por `internal/infra/outbound/config` e injetada em `NewCameraPlanService` por `cmd/sobrevoo`. Duração, fps, distância e inclinação chegam por parâmetro. O núcleo não lê flag, ambiente ou arquivo. |
| IX. Organização de Portas, Service Layer e Mocks | PASSA | Sem `ports.go`: `CameraPlanExporter`, ligada à entidade `CameraPlan`, fica no topo de `camera_plan.go` (após `//go:generate` e imports). Nomeada pelo papel (`Exporter`; não é um repositório: não lê nem consulta). Um serviço por recurso: `CameraPlanService`/`cameraPlanService`/`NewCameraPlanService` (`Generate`, `Export`) e `TrackService`/`trackService`/`NewTrackService` (`Clean`, `Treat`, `Inspect`), sem `Execute`; a refatoração elimina o `InspectTrackService`, único serviço de método único que restava, e a duplicação do tratamento do trajeto. Serviços dependendo de serviços (`CameraPlanService` e `GeoDataService` → `TrackService`) é permitido pelo princípio. Toda regra vive em `internal/domain` (`PlanParameters.Validate`, `PlanCamera`, `MinimumDuration`, `NewCameraPlan`, que calcula o resumo); `application` só chama portas na ordem certa. Adapter: `jsonfile.NewCameraPlanExporter()` em `camera_plan_exporter.go` (pacote de tecnologia; construtor nomeado pela porta; sem repetir o nome do pacote). Mocks por `//go:generate` acima de cada interface, em `mockdomain`/`mockapplication`. Receivers: `p` (`PlanParameters`), `c` (`CameraPlan`), `f` (`CameraFrame`), `s` (`cameraPlanService`), `e` (exporter). |
| X. Testes: Given/When/Then, Builders e Isolamento por Camada | PASSA | Todo teste em `t.Run("should ...")` com `// given`/`// when`/`// then`, sem tabelas; builders em `builddomain`; domínio/serviço com portas mockadas, CLI com `CameraPlanService` mockado (`mockapplication`), adapter com diretório temporário (é o próprio adapter, então tocar disco é o objeto do teste). Sem teste de ponta a ponta automatizado. |
| Idioma dos Artefatos | PASSA | Artefatos do Spec Kit em português; identificadores, pacotes, arquivos, comentários e mensagens de commit em inglês; I/O em tempo de execução (flags, saída, erros) em inglês, como nas etapas 1 e 2. |

Nenhuma violação identificada. Rastreamento de Complexidade não se aplica.

## Estrutura do Projeto

### Documentação (desta funcionalidade)

```text
specs/003-camera-path-planning/
├── plan.md              # Este arquivo (saída do comando /speckit-plan)
├── research.md          # Saída da Fase 0 (comando /speckit-plan)
├── data-model.md        # Saída da Fase 1 (comando /speckit-plan)
├── quickstart.md        # Saída da Fase 1 (comando /speckit-plan)
├── contracts/
│   ├── cli.md           # Saída da Fase 1: comando `sobrevoo plan`
│   └── plan-file.md     # Saída da Fase 1: formato do arquivo exportado
├── checklists/
│   └── requirements.md  # Gerado por /speckit-specify
└── tasks.md             # Saída da Fase 2 (comando /speckit-tasks - NÃO criado pelo /speckit-plan)
```

### Código-Fonte (raiz do repositório)

```text
cmd/sobrevoo/
└── main.go                                    # (alterado) monta TrackService (usado por inspect, geodata e plan), o exporter, CameraPlanService e o comando "plan"

internal/
├── domain/
│   ├── camera_plan.go                         # NOVO: porta CameraPlanExporter (no topo), CameraPlan, CameraFrame, Phase, TimeReference, SmoothedSpan, PlanSummary, NewCameraPlan
│   ├── camera_plan_parameters.go              # NOVO: PlanParameters (+Validate, FrameCount), CameraTuning
│   ├── camera_planning.go                     # NOVO: PlanCamera, MinimumDuration
│   ├── camera_projection.go                   # NOVO: plano tangente local (azimutal equidistante)
│   ├── camera_timeline.go                     # NOVO: referência de tempo, compressão de paradas, s(t)
│   ├── camera_motion.go                       # NOVO: rumo, limitação de taxa, suavização gaussiana, trechos suavizados
│   ├── camera_framing.go                      # NOVO: pose de visão geral, abertura/fechamento
│   ├── errors.go                              # (estendido) sete sentinelas novos
│   ├── track.go                               # (estendido) CleanedTrack e TreatedTrack (DTOs de domínio)
│   ├── *_test.go                              # NOVOS: um por arquivo acima (funções puras + propriedades)
│   ├── builddomain/
│   │   ├── camera_frame_builder.go            # NOVO
│   │   ├── camera_plan_builder.go             # NOVO
│   │   └── plan_parameters_builder.go         # NOVO
│   └── mockdomain/
│       └── camera_plan_exporter.go            # GERADO (make generate)
├── application/
│   ├── track_service.go                       # NOVO (refatoração): TrackService (Clean, Treat, Inspect); substitui inspect_track_service.go
│   ├── track_service_test.go                  # NOVO: migra os cenários de inspect_track_service_test.go
│   ├── inspect_track_service.go               # REMOVIDO (absorvido por TrackService.Inspect)
│   ├── inspect_track_service_test.go          # REMOVIDO (migrado)
│   ├── geo_data_service.go                    # (alterado) CheckCoverage usa TrackService.Clean; perde parser e limiares
│   ├── geo_data_service_test.go               # (alterado) construtor e mocks
│   ├── camera_plan_service.go                 # NOVO: CameraPlanService / cameraPlanService / NewCameraPlanService
│   ├── camera_plan_service_test.go            # NOVO
│   └── mockapplication/
│       ├── track_service.go                   # GERADO (substitui inspect_track_service.go)
│       └── camera_plan_service.go             # GERADO
└── infra/
    ├── inbound/cli/
    │   ├── inspect.go                         # (alterado) depende de application.TrackService
    │   ├── inspect_test.go                    # (alterado) mock de TrackService
    │   ├── plan.go                            # NOVO: comando "plan", flags, formatação do resumo
    │   ├── plan_test.go                       # NOVO
    │   └── exit_code.go                       # (estendido) códigos 10–16
    └── outbound/
        ├── config/config.go                   # (estendido) Config.CameraTuning
        └── jsonfile/
            ├── camera_plan_exporter.go        # NOVO: JSON versionado, escrita atômica
            └── camera_plan_exporter_test.go   # NOVO
```

**Decisão de Estrutura**: mesmo projeto único em Go da etapa 1, seguindo as
convenções já consolidadas: um arquivo de domínio por responsabilidade (nada
de `helpers.go` nem subpacote por algoritmo), porta no arquivo da entidade,
adapter no pacote de tecnologia que já serve outra porta (`jsonfile`),
composition root em `cmd/sobrevoo/main.go`. `CLAUDE.md` (que cita
`InspectTrackService` nas seções de arquitetura e testes) e `README.md`
serão atualizados na implementação para refletir `TrackService`, `plan` e a
terceira etapa.

**Ordem de execução (base para o `/speckit-tasks`)**: a primeira fase é a
refatoração de `TrackService`, concluída e com `make test` verde antes de
qualquer código de câmera; ela isola qualquer regressão das etapas 1 e 2 de
defeitos da etapa 3.

## Rastreamento de Complexidade

Nenhuma violação da constituição; nada a justificar.

## Riscos e Pontos de Atenção

- **Valores heurísticos** (`research.md` itens 3 a 8) são iniciais e só serão
  validados visualmente na etapa de renderização; por isso ficam em
  `CameraTuning`. Os testes de propriedade verificam os *limites*, não a
  estética.
- **Duração automática** (`research.md` item 8.1): substitui o padrão fixo
  de 60 s, que seria recusado em trajetos longos. Como sempre parte da
  duração mínima, nunca é recusada; uma duração *informada* abaixo do
  mínimo continua sendo recusada, com a mensagem que informa o mínimo. A
  curva (√, piso 20 s, teto 120 s) é inicial e fica em `CameraTuning`.
- **Paradas fundidas pela simplificação** da etapa 1 podem não ser detectadas
  (`research.md` item 3, "Limitação conhecida"); efeito conservador.
- **Determinismo entre arquiteturas** limitado à quantização
  (`research.md` item 9).
- **Refatoração das etapas 1 e 2** (`research.md` item 13): toca
  `InspectTrackService`, `GeoDataService`, o comando `inspect`, `main.go`,
  testes e `CLAUDE.md`. Mitigação: comportamento preservado, cenários de
  teste migrados sem alteração de expectativa, e fase própria com testes
  verdes antes do restante.
