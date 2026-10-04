# Plano de Implementação: Bloco de Velocidade na Sobreposição

**Branch**: `014-speed-overlay-block` | **Data**: 2026-10-04 | **Especificação**: [spec.md](./spec.md)

**Entrada**: Especificação de funcionalidade de `/specs/014-speed-overlay-block/spec.md`

**Nota**: Este template é preenchido pelo comando `/speckit-plan`; sua definição descreve o fluxo de execução.

## Resumo

O plano de câmera passa a calcular, por quadro, a velocidade média da
atividade numa janela de tempo fixa (`CameraTuning.SpeedWindow`, um tuning
interno novo, não exposto por flag) centrada no instante do marcador
(`CameraFrame.ActivityElapsed`). O cálculo reaproveita exatamente o
mecanismo que já produz `ActivityElapsed`/`TrackElevation`: `Route.TimeAt`
já converte distância → tempo decorrido; esta etapa acrescenta o espelho
exato, `Route.DistanceAt` (tempo decorrido → distância, mesma busca por
bracket e o mesmo `lerp` seguro `(1-t)*a + t*b`), para achar a distância nas
duas pontas da janela e dividir pelo tempo realmente coberto — que
`Route.DistanceAt` já encurta sozinho nos extremos, porque sua busca por
`sort.Search` já fica presa no primeiro ou no último ponto quando o tempo
pedido cai fora do trajeto, exatamente como `Route.TimeAt` já faz hoje para
distância. Nenhum sinalizador novo de disponibilidade é necessário: a
velocidade usa exatamente o mesmo `ok` que já guarda se o trajeto tem
horário em todo ponto (o mesmo que decide `TimeReferenceClock`), e a
sobreposição a exibe sob o mesmo gate que já esconde o bloco de tempo
decorrido (`plan.TimeReference == TimeReferenceClock`) — nenhum campo novo
em `PlanSummary`/`CameraPlan`. O valor novo (`CameraFrame.MarkerSpeed`,
metros por segundo) entra no plano, no arquivo exportado
(`frames[].marker.speed_mps`, novo campo obrigatório) e na identidade do
plano (`CameraPlan.ID()`), pelos mesmos três lugares que `CameraToMarkerDistance`
já ocupa. Como o campo passa a ser obrigatório onde um plano anterior não o
escreve, `format_version` sobe de `2` para `3` — mesma recusa
(`ErrPlanFormatVersionUnsupported`) e mesmo código de saída que já existem
para qualquer versão desconhecida, sem nenhum sentinela novo. Na
sobreposição, um quinto bloco (`OverlayBlockSpeed`, nome `speed`) entra no
mesmo mecanismo de seleção que os quatro já existentes (`OverlayConfig`,
`parseOverlay`), mas nasce fora da lista de blocos padrão
(`config.RenderDefaults.OverlayBlocks`, inalterada) — opt-in puro, sem
nenhuma flag nova. O desenho do bloco (texto, painel, largura
compartilhada) reaproveita inteiramente `screenOverlay.draw`/
`stablePanelWidth`, só acrescentando `speed` à mesma lista de blocos
numéricos que `distance`/`elevation`/`time` já formam — nenhuma mudança de
arranjo, tipografia ou rótulo dos blocos existentes (fora de escopo,
reservado para a etapa seguinte).

## Contexto Técnico

**Linguagem/Versão**: Go 1.26.4 (`go.mod`), sem mudança.

**Dependências Principais**: nenhuma nova — reutiliza `spf13/cobra` (CLI),
`golang.org/x/image/font/sfnt` (já usado pela fonte vetorial da
sobreposição) e a exportação/leitura de plano já existente
(`internal/infra/outbound/jsonfile`).

**Armazenamento**: o arquivo de plano exportado (JSON,
`jsonfile.CameraPlanExporter`/`CameraPlanReader`) sobe de `format_version`
`2` para `3`: um campo novo e obrigatório por quadro
(`frames[].marker.speed_mps`), pelo mesmo motivo que a versão `2` já subiu
na etapa 9 (um campo que uma versão anterior não escreve, sem valor
implícito que se possa supor para a ausência dele) — ao contrário de
`parameters.simplification`/`.smoothing` (etapa 13), que ficaram opcionais
sem subir a versão porque tinham um padrão implícito seguro.

**Testes**: `go test ./... -cover`, `testify` + `uber-go/mock`, sem mudança
de ferramenta. `Route.DistanceAt` ganha testes espelhados dos de
`Route.TimeAt` (`duration_test.go`); o cálculo de `MarkerSpeed` em
`camera_planning.go` ganha testes de unidade cobrindo janela cheia, janela
encurtada nas duas pontas e trajeto sem horário.

**Plataforma-Alvo**: CLI de linha de comando, macOS/Linux — sem mudança.

**Tipo de Projeto**: CLI de projeto único (`cmd/sobrevoo`), sem mudança de
estrutura.

**Metas de Desempenho**: nenhuma meta numérica nova — `Route.DistanceAt` é
a mesma busca binária (`O(log n)`) que `Route.TimeAt` já faz, chamada duas
vezes por quadro (início e fim da janela) em vez de uma; o custo adicional
é desprezível frente ao resto do planejamento de câmera.

**Restrições**: a janela de velocidade (`CameraTuning.SpeedWindow`) é um
valor fixo, interno, documentado em `research.md` — não é lida de
`--flag` nem de arquivo de configuração do usuário, pelo mesmo motivo que
`GaussianSigmaSeconds`/`OverviewMinDistanceFactor` já são constantes
internas de `CameraTuning` nunca expostas por flag; offline (Princípio V) —
sem mudança, nenhuma dependência nova; determinismo (Princípio IV) — o
cálculo usa só `+ − × ÷`, `Sqrt`/`Min`/`Max`/`Floor` e o mesmo `lerp`
seguro `(1-t)*a + t*b` que `Route.TimeAt`/`ElevationProfile.At` já usam,
nunca `a + t*(b-a)`.

**Escala/Escopo**: mesma escala das etapas 9 e 13 (um trajeto por
execução); esta etapa não afeta `inspect`, `geodata check`, `video` nem a
montagem do vídeo — só o planejamento de câmera (`plan`, `fly`) e o desenho
de quadros (`render frame`, `render all`, `fly`).

## Verificação da Constituição

*PORTÃO: Deve passar antes da Fase 0 de pesquisa. Reverificar após o design da Fase 1.*

| Princípio | Verificação |
|---|---|
| I. Arquitetura Hexagonal | Nenhuma biblioteca de infraestrutura cruza para `internal/domain`/`internal/application`; `Route.DistanceAt`, o cálculo de `MarkerSpeed` em `camera_planning.go` e o desenho do bloco em `frame_screen_overlay.go` são domínio puro, como `Route.TimeAt`/`ElevationProfile.At`/`screenOverlay.draw` já são. |
| II. Portas para Toda Dependência Externa | Nenhuma porta nova; nenhum método novo em porta existente. `CameraPlanExporter`/`CameraPlanReader` continuam com a mesma assinatura — só o que cada implementação escreve/lê muda (um campo a mais por quadro). |
| III. Entrypoints Descartáveis | A CLI só acrescenta `speed` à lista de nomes que `--overlay-blocks` já aceita (mesmo `parseOverlay`) e ao texto de ajuda — nenhuma regra de negócio nova na CLI; a disponibilidade da velocidade, sua inclusão na identidade do plano e o gate de exibição continuam inteiramente em `internal/domain`. |
| IV. Neutralidade Geográfica | Sem mudança: nenhum dado geográfico fixo é introduzido ou alterado. |
| V. Funcionamento Offline | Sem mudança: nenhuma dependência de rede ou serviço externo novo. |
| VI. Testes Automatizados no Núcleo | `Route.DistanceAt`, `TreatedTrack.PlanCamera` (o cálculo de `MarkerSpeed`), `CameraPlan.ID()`/`.Validate()` e `screenOverlay.draw` continuam testáveis com as portas mockadas (`mockdomain`/`mockapplication`) ou sem nenhuma porta (são domínio puro), sem tocar disco; a CLI continua mockando `application.CameraPlanService`/`FrameService`/`FlightService`. |
| VII. Erros Sentinela no Domínio | Nenhum sentinela novo: a recusa de um plano `format_version` desconhecido já é `ErrPlanFormatVersionUnsupported`; um nome de bloco desconhecido (agora incluindo tentar usar `speed` grafado errado) já é `ErrInvalidOverlayBlock` — os dois sentinelas de 009/003 reaproveitados sem alteração de assinatura. |
| VIII. Configuração Injetada | `CameraTuning.SpeedWindow` é lido de `config.CameraTuning` (campo novo, valor fixo em `config.go`) e injetado como qualquer outro campo de `CameraTuning` — nenhuma leitura direta de ambiente/arquivo/CLI dentro do núcleo; não há flag de CLI para ele (Suposição do spec: a janela não é ajustável pelo usuário). |
| IX. Portas/Service Layer/Regra de Negócio | Nenhum serviço novo. `cameraPlanService.Generate` não muda (os campos novos vêm resolvidos dentro de `TreatedTrack.PlanCamera`, como `ActivityElapsed`/`TrackElevation` já vêm); `FrameService` não muda (o bloco novo é só mais uma branch dentro de `screenOverlay`, que `FrameService` já invoca por inteiro via `Scene.Render`). A regra de negócio (disponibilidade, cálculo, participação na identidade, gate de exibição) vive inteiramente em `internal/domain`. |
| X. Testes: Given/When/Then, Builders, Isolamento | Sem mudança de convenção; `builddomain.CameraFrameBuilder` ganha `WithMarkerSpeed`; `builddomain.CameraTuningBuilder` ganha `WithSpeedWindow` (ou o padrão já cobre, se o builder usa um valor padrão sensato) pela mesma razão que os campos já existentes de `CameraFrame`/`CameraTuning` têm seus próprios `With*`. |

Nenhuma violação; nada a registrar em Rastreamento de Complexidade.

## Estrutura do Projeto

### Documentação (desta funcionalidade)

```text
specs/014-speed-overlay-block/
├── plan.md               # Este arquivo (saída do comando /speckit-plan)
├── research.md           # Saída da Fase 0 (comando /speckit-plan)
├── data-model.md         # Saída da Fase 1 (comando /speckit-plan)
├── quickstart.md         # Saída da Fase 1 (comando /speckit-plan)
├── contracts/            # Saída da Fase 1 (comando /speckit-plan)
│   ├── plan-file-v3.md
│   └── speed-overlay-block.md
└── tasks.md              # Saída da Fase 2 (comando /speckit-tasks — NÃO criado pelo /speckit-plan)
```

### Código-Fonte (raiz do repositório)

Projeto único já existente (hexagonal), sem mudança de layout — só
extensões pontuais dentro da árvore já estabelecida:

```text
internal/domain/
├── duration.go                    # +Route.DistanceAt (espelho de TimeAt: tempo → distância)
├── camera_plan_parameters.go      # CameraTuning: +SpeedWindow time.Duration
├── camera_planning.go             # cameraPlanner.frame: calcula e grava CameraFrame.MarkerSpeed;
│                                   #  +speedStep (quantização, ao lado de coordinateStep/lengthStep)
├── camera_plan.go                 # CameraFrame: +MarkerSpeed float64;
│                                   #  Validate(): +checagem de MarkerSpeed finito e ≥ 0;
│                                   #  ID(): +1 escrita no hash (MarkerSpeed quantizado)
├── frame_overlay_config.go        # +OverlayBlockSpeed; OverlayConfig: +Speed bool;
│                                   #  NewOverlayConfig: +case speed; Fingerprint: +1 flag
└── frame_screen_overlay.go        # draw: +showSpeed (gate: config.Speed && TimeReferenceClock);
                                    #  +speedBlockText, +formatOverlaySpeed;
                                    #  stablePanelWidth: +ramo speed na mesma lista de blocos numéricos

internal/infra/outbound/jsonfile/
├── camera_plan_file.go            # planFormatVersion: 2 → 3; markerFile/frameFile: +Speed;
│                                   #  encodePlan: +measure(frame.MarkerSpeed)
└── camera_plan_reader.go          # acceptedPlanFormatVersions segue planFormatVersion (3);
                                    #  readMarker: +Speed *float64 (obrigatório);
                                    #  Read: +checagem de ausência; CameraFrame: +MarkerSpeed

internal/infra/inbound/cli/
└── overlay.go                     # overlaysUsage/overlayBlocksUsage: menciona speed (não-padrão);
                                    #  overlayBlocksOf/formatOverlayBlocks: +ramo Speed

internal/infra/outbound/config/
└── config.go                      # CameraTuning: +SpeedWindowSeconds float64; default() preenche

cmd/sobrevoo/
└── config_mapping.go              # domainCameraTuning: +SpeedWindow: time.Duration a partir de
                                    #  cfg.SpeedWindowSeconds

internal/domain/builddomain/
├── camera_frame_builder.go        # +WithMarkerSpeed
└── camera_tuning_builder.go       # +WithSpeedWindow (se o builder cobrir esse campo)
```

**Decisão de Estrutura**: nenhuma pasta nova de alto nível — tudo estende
`internal/domain`, `internal/infra/{outbound/{jsonfile,config},inbound/cli}`
e `cmd/sobrevoo` já existentes, seguindo exatamente o padrão das etapas 9 e
13 (um valor novo por quadro que entra no plano, no arquivo exportado e na
identidade do plano, sem porta nova nem serviço novo).

## Rastreamento de Complexidade

> **Preencher SOMENTE se a Verificação da Constituição tiver violações que precisam ser justificadas**

Nenhuma violação.
