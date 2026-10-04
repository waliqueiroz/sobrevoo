# Plano de Implementação: Níveis de tratamento em `plan` e `fly`

**Branch**: `013-treatment-level-flags` | **Data**: 2026-10-03 | **Especificação**: [spec.md](./spec.md)

**Entrada**: Especificação de funcionalidade de `/specs/013-treatment-level-flags/spec.md`

**Nota**: Este template é preenchido pelo comando `/speckit-plan`; sua definição descreve o fluxo de execução.

## Resumo

`plan` e `fly` passam a aceitar `--simplification`/`--smoothing` — os mesmos
nomes, valores e padrões que `inspect` já aceita — em vez de usar sempre o
nível padrão da configuração. O mecanismo de compartilhamento já existe:
`parsePlanParameters` já é a única função que `plan.go` e `fly.go` chamam
para interpretar `--distance`/`--tilt`, e `parseLevel`/`levelName` já são
funções genéricas (não específicas de `inspect`) usadas por ela; as duas
flags novas entram pelo mesmo caminho, sem nenhum parser novo. Os dois
níveis tornam-se dois campos de `domain.PlanParameters` — o mesmo lugar onde
`Distance`/`Tilt` já vivem, pelo mesmo motivo (são escolhas do usuário sobre
o plano) —, e `cameraPlanService.Generate` passa a usá-los em vez de um
`defaultLevel` próprio injetado no serviço (que deixa de existir): a decisão
de qual nível é o "padrão" já é só da CLI, igual a `Appearance`/`Resolution`.
Isso dá, de graça, os dois efeitos que a spec pede como regra de negócio:
`CameraPlan.ID()` já grava `Distance`/`Tilt` no hash de identidade — gravar
`Simplification`/`Smoothing` do mesmo jeito é a mesma linha, e
`FlightService.reusePlan` já decide por esse `ID()`, então o reaproveitamento
de `--keep` passa a respeitar os dois níveis novos sem nenhuma lógica de
reaproveitamento nova. O arquivo de plano exportado ganha dois campos de
texto (`parameters.simplification`/`.smoothing`), escritos e lidos pelas
mesmas funções `levelText`/`parseLevel` que já existem no adapter
`jsonfile` para `distance`/`tilt` — a ausência desses campos num arquivo
anterior a esta etapa já cai, sem nenhum código novo, no mesmo "texto
desconhecido lê como medium" que `parseLevel` já garante, o que resolve a
decisão de compatibilidade da sessão de `/speckit-clarify` sem nenhum `if`
dedicado a ela.

## Contexto Técnico

**Linguagem/Versão**: Go 1.26.4 (`go.mod`), sem mudança.

**Dependências Principais**: nenhuma nova — reutiliza `spf13/cobra` (CLI) e
a exportação/leitura de plano já existente (`internal/infra/outbound/
jsonfile`).

**Armazenamento**: o arquivo de plano exportado (JSON,
`jsonfile.CameraPlanExporter`/`CameraPlanReader`), sem mudança de tecnologia
nem de versão de formato — só dois campos novos em `parameters`
(`format_version` continua `2`, pelo mesmo motivo que `parameters.
aspect_ratio` não subiu a versão ao ser acrescentado: um campo opcional,
nunca obrigatório).

**Testes**: `go test ./... -cover`, `testify` + `uber-go/mock`, sem mudança
de ferramenta.

**Plataforma-Alvo**: CLI de linha de comando, macOS/Linux — sem mudança.

**Tipo de Projeto**: CLI de projeto único (`cmd/sobrevoo`), sem mudança de
estrutura.

**Metas de Desempenho**: nenhuma meta numérica nova — os dois níveis apenas
escolhem qual tabela de simplificação/suavização já existente (Douglas-
Peucker, Catmull-Rom) é aplicada ao trajeto; o custo já é pago hoje com o
nível padrão, só a escolha de qual nível passa a ser do usuário.

**Restrições**: recusar cedo (FR-003) — um valor de flag fora do conjunto
aceito é um erro de uso da CLI, resolvido antes de abrir o arquivo de
trajeto, igual a `--distance`/`--tilt` hoje; nenhuma etapa da tubulação do
`fly` roda antes dessa validação; offline (Princípio V) — sem mudança,
nenhuma dependência nova.

**Escala/Escopo**: mesma escala das etapas 1 e 3 (um trajeto por execução);
esta etapa não introduz nenhum novo limite nem afeta `geodata check`
(que só usa `TrackService.Clean`, nunca `Treat`) nem `render frame`/
`render all`/`video` (que operam sobre um plano e um recorte já tratados).

## Verificação da Constituição

*PORTÃO: Deve passar antes da Fase 0 de pesquisa. Reverificar após o design da Fase 1.*

| Princípio | Verificação |
|---|---|
| I. Arquitetura Hexagonal | Nenhuma biblioteca de infraestrutura cruza para `internal/domain`/`internal/application`; os dois campos novos de `PlanParameters` e a leitura/escrita deles no plano exportado seguem exatamente a fronteira que `Distance`/`Tilt` já respeitam. |
| II. Portas para Toda Dependência Externa | Nenhuma porta nova; nenhum método novo em porta existente. `CameraPlanExporter`/`CameraPlanReader` continuam com a mesma assinatura — só o que cada implementação escreve/lê muda. |
| III. Entrypoints Descartáveis | A CLI só acrescenta duas flags a uma função de parsing já existente (`parsePlanParameters`) e as repassa — nenhuma regra de negócio nova na CLI; a identidade do plano e a decisão de reaproveitamento continuam inteiramente em `internal/domain`/`internal/application`. |
| IV. Neutralidade Geográfica | Sem mudança: nenhum dado geográfico fixo é introduzido ou alterado. |
| V. Funcionamento Offline | Sem mudança: nenhuma dependência de rede ou serviço externo novo. |
| VI. Testes Automatizados no Núcleo | `PlanParameters`, `CameraPlan.ID()` e `cameraPlanService.Generate` continuam testáveis com as portas mockadas (`mockdomain`/`mockapplication`), sem tocar disco; a CLI continua mockando `application.CameraPlanService`/`FlightService`. |
| VII. Erros Sentinela no Domínio | Nenhum sentinela novo: um valor de flag inválido continua sendo um erro de uso da CLI (código `2`), exatamente como `--distance`/`--tilt` já são — não uma regra de negócio do domínio. |
| VIII. Configuração Injetada | Nenhuma configuração nova: os dois níveis usam o mesmo `config.DefaultLevel` que `inspect` já usa (mapeado pelo já existente `domainLevel`); `config.go` não ganha nenhum campo. |
| IX. Portas/Service Layer/Regra de Negócio | Nenhum serviço novo. `CameraPlanService.Generate` perde um parâmetro de construção (`defaultLevel`, que deixa de existir no serviço) e passa a usar os dois campos já resolvidos de `parameters` — mesma forma como `Appearance`/`Resolution` já chegam resolvidos a `FrameService`. A regra de negócio (a identidade do plano incluir os dois níveis; o reaproveitamento respeitá-la) vive inteiramente em `CameraPlan.ID()` (domínio) e é só consultada por `FlightService.reusePlan`, sem nenhuma lógica nova nele. |
| X. Testes: Given/When/Then, Builders, Isolamento | Sem mudança de convenção; `builddomain.PlanParametersBuilder` ganha `WithSimplification`/`WithSmoothing` (e os dois campos no construtor padrão, como `Distance`/`Tilt` já têm) pela mesma razão que já o têm — literais de `PlanParameters` repetidos em teste. |

Nenhuma violação; nada a registrar em Rastreamento de Complexidade.

## Estrutura do Projeto

### Documentação (desta funcionalidade)

```text
specs/013-treatment-level-flags/
├── plan.md               # Este arquivo (saída do comando /speckit-plan)
├── research.md           # Saída da Fase 0 (comando /speckit-plan)
├── data-model.md         # Saída da Fase 1 (comando /speckit-plan)
├── quickstart.md         # Saída da Fase 1 (comando /speckit-plan)
├── contracts/            # Saída da Fase 1 (comando /speckit-plan)
│   ├── treatment-level-flags.md
│   └── plan-file-addendum.md
└── tasks.md              # Saída da Fase 2 (comando /speckit-tasks — NÃO criado pelo /speckit-plan)
```

### Código-Fonte (raiz do repositório)

Projeto único já existente (hexagonal), sem mudança de layout — só
extensões pontuais dentro da árvore já estabelecida:

```text
internal/domain/
├── camera_plan_parameters.go     # PlanParameters: +Simplification, +Smoothing Level
└── camera_plan.go                 # ID(): +2 escritas no hash (Simplification, Smoothing)

internal/application/
└── camera_plan_service.go         # NewCameraPlanService: -defaultLevel (parâmetro removido);
                                    #  Generate: usa parameters.Simplification/.Smoothing
                                    #  no lugar de s.defaultLevel

internal/infra/outbound/jsonfile/
├── camera_plan_file.go            # parametersFile/encodePlan: +simplification, +smoothing
└── camera_plan_reader.go          # readParameters/Read: +simplification, +smoothing
                                    #  (ausência → parseLevel("") → medium, já existente)

internal/infra/inbound/cli/
├── plan.go                        # parsePlanParameters: +2 parâmetros/flags;
│                                   #  NewPlanCommand: +2 flags (--simplification, --smoothing)
└── fly.go                         # NewFlightCommand: +2 flags, mesmo parsePlanParameters

cmd/sobrevoo/
├── config_mapping.go              # domainPlanParameters: +parâmetro defaultLevel,
│                                   #  usado para preencher Simplification/Smoothing
└── main.go                        # domainPlanParameters(cfg.PlanDefaults, cfg.DefaultLevel)
                                    #  nos dois call sites (plan, fly);
                                    #  NewCameraPlanService sem o argumento defaultLevel

internal/domain/builddomain/
└── plan_parameters_builder.go     # +WithSimplification, +WithSmoothing;
                                    #  construtor padrão: +Simplification/Smoothing: LevelMedium

internal/application/mockapplication/
└── camera_plan_service.go         # regenerado (make generate) — assinatura de Generate não
                                    #  muda (PlanParameters já carrega os dois campos);
                                    #  NewCameraPlanService não é mockado (é construtor concreto)
```

**Decisão de Estrutura**: nenhuma pasta nova de alto nível — tudo estende
`internal/domain`, `internal/application`, `internal/infra/{outbound/
jsonfile,inbound/cli}` e `cmd/sobrevoo` já existentes, seguindo exatamente o
padrão das etapas 8 e 9 (um valor por chamada que passa a fazer parte da
identidade de algo já existente, sem porta nova).

## Rastreamento de Complexidade

> **Preencher SOMENTE se a Verificação da Constituição tiver violações que precisam ser justificadas**

Nenhuma violação.
