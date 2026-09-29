# Plano de Implementação: Sobreposições de Tela nos Quadros

**Branch**: `009-frame-overlays` | **Data**: 2026-09-28 | **Especificação**: [spec.md](./spec.md)

**Entrada**: Especificação de funcionalidade de `/specs/009-frame-overlays/spec.md`

**Nota**: Este template é preenchido pelo comando `/speckit-plan`; sua definição descreve o fluxo de execução.

## Resumo

Acrescenta, aos quadros que `render frame`/`render all`/`fly` já desenham,
sobreposições de tela fixas — não coladas no terreno: distância percorrida,
elevação do trajeto e ganho acumulado no ponto do marcador, tempo decorrido
da atividade, e um perfil de elevação do trajeto inteiro com um marcador que
avança com o voo — ligadas por padrão, desligáveis por inteiro ou por
bloco. Os valores vêm exclusivamente do plano de câmera e do recorte já
informados: o plano passa a guardar, por quadro, o instante real da
atividade e (por research.md item 7) tanto a elevação bruta quanto o ganho
de elevação acumulado no ponto do marcador — três campos novos, não dois,
porque só um ganho pré-computado sobre o trajeto tratado, e não uma soma
corrente sobre valores reamostrados por quadro, garante bater byte a byte
com o que `inspect` relata ao final do voo (FR-014). O texto usa uma fonte
bitmap já disponível via `golang.org/x/image` (nenhuma dependência nova),
desenhada por replicação de pixel inteira — sem nenhuma operação de ponto
flutuante além da já existente mistura alfa por `+ − × ÷` que o traçado e o
marcador já usam, preservando o determinismo byte a byte entre arquiteturas.
A configuração de sobreposição escolhida entra no hash de `FrameSetID`, ao
lado da aparência, sem precisar de um `RenderVersion` novo — o mesmo
mecanismo que já decide reaproveitar-ou-redesenhar por conjunto passa a
enxergar sobreposição diferente de graça.

## Contexto Técnico

**Linguagem/Versão**: Go 1.26.4 (`go.mod`), sem mudança.

**Dependências Principais**: `golang.org/x/image` (já em `go.mod`, hoje
indireta) — usada agora também para a fonte embutida
(`font/inconsolata`), sem subir de versão. Nenhuma dependência nova.

**Armazenamento**: arquivos locais (plano `.json`, recorte `.zip`, quadros
`.png`), sem mudança de tecnologia — só de formato (plano sobe para
`format_version: 2`, `contracts/plan-file-v2.md`).

**Testes**: `go test ./... -cover`, `testify` + `uber-go/mock`, sem mudança
de ferramenta.

**Plataforma-Alvo**: CLI de linha de comando, macOS/Linux, amd64/arm64 — o
determinismo byte a byte entre arquiteturas (já exigido desde a etapa 5)
passa a valer também para o texto desenhado (research.md item 4).

**Tipo de Projeto**: CLI de projeto único (`cmd/sobrevoo`), sem mudança de
estrutura.

**Metas de Desempenho**: desenhar a sobreposição de tela não pode dominar o
tempo de um quadro — é um passo sequencial de texto/linhas sobre uma imagem
já pronta, ordens de grandeza mais barato que o traçado de raios por pixel
do terreno que `Scene.drawTerrain` já faz; nenhuma meta numérica nova além
da que já existe para o desenho do quadro inteiro.

**Restrições**: byte a byte determinístico entre goroutines, processos e
arquiteturas (Constitution, `CLAUDE.md` "O desenho dos quadros"); nenhuma
leitura nova do trajeto GPS nem dos dados geográficos registrados durante o
desenho (FR-005); nenhuma dependência de fonte instalada no sistema
(FR-008); offline (Princípio V) — nenhuma das duas restrições muda a lista
de bibliotecas já vendorizadas.

**Escala/Escopo**: mesma escala das etapas 5–8 (até ~3840×2160, até 432 000
quadros por plano); a sobreposição de tela não introduz nenhum novo limite.

## Verificação da Constituição

*PORTÃO: Deve passar antes da Fase 0 de pesquisa. Reverificar após o design da Fase 1.*

| Princípio | Verificação |
|---|---|
| I. Arquitetura Hexagonal | O desenho da sobreposição (texto, placa, perfil) é código de domínio puro (`internal/domain`), como o resto do renderizador (etapa 5) — nenhuma biblioteca de infraestrutura cruza a fronteira. A fonte embutida (`golang.org/x/image/font/inconsolata`) é dado estático (bitmap pré-gerado), não I/O: importá-la no domínio não viola o princípio, do mesmo jeito que `math` ou `time` já não violam. |
| II. Portas para Toda Dependência Externa | Nenhuma porta nova: a sobreposição de tela não introduz I/O — lê só o `CameraPlan` e o `Resolution` que `Scene.Render` já recebe. `golang.org/x/image/font/inconsolata` é uma função pura da biblioteca (dado estático embutido no binário), mesmo carve-out já aplicado a `time.Now()`. |
| III. Entrypoints Descartáveis | A CLI só traduz duas flags novas (`--overlays`, `--overlay-blocks`) num `domain.OverlayConfig` via `parseOverlay`, e um erro sentinela num código de saída — nenhuma regra de negócio na CLI. |
| IV. Neutralidade Geográfica | Sem mudança: a sobreposição de tela não introduz nenhum dado geográfico fixo; os quatro blocos derivam só do plano e do recorte que o usuário já forneceu. |
| V. Funcionamento Offline | A fonte é embutida no binário (nenhum download, nenhuma fonte do sistema) — reforça, não enfraquece, este princípio. |
| VI. Testes Automatizados no Núcleo | `OverlayConfig`, `ElevationProfile`, `Route.TimeAt`, o desenho da sobreposição e a extensão de `FrameSetID` são testados no domínio sem tocar disco; `FrameService`/CLI continuam mockando as portas e os serviços, como já fazem. |
| VII. Erros Sentinela no Domínio | Um sentinela novo, `ErrInvalidOverlayBlock`, ao lado dos já existentes em `errors.go`; a rejeição do plano de versão anterior reusa `ErrPlanFormatVersionUnsupported`, já existente. |
| VIII. Configuração Injetada | `config.RenderDefaults` ganha os padrões de sobreposição (ligada, quatro blocos); o composition root (`config_mapping.go`) os mapeia para `domain.OverlayConfig`, do mesmo jeito que já faz para `domain.Appearance`. |
| IX. Portas/Service Layer/Regra de Negócio | Nenhuma porta nova, nenhum serviço novo: `FrameService`/`VideoService`/`FlightService` continuam as mesmas interfaces, só com um campo `Overlay` a mais nos DTOs de domínio que já usam (`SingleFrameRequest`, `FrameSetRequest`, `FlightRequest`) — mesmo padrão que `Appearance` já seguiu na etapa 8. A regra de negócio (interpolação de tempo/elevação, ganho acumulado, validação de bloco, fingerprint) vive inteiramente em `internal/domain`, como construtores/métodos de `Route`/`ElevationProfile`/`OverlayConfig`. |
| X. Testes: Given/When/Then, Builders, Isolamento | Sem mudança de convenção; `builddomain` ganha builders para `CameraFrame` (se ainda não cobrir os campos novos) e `OverlayConfig`/`ElevationProfile` conforme os testes pedirem. |

Nenhuma violação; nada a registrar em Rastreamento de Complexidade.

## Estrutura do Projeto

### Documentação (desta funcionalidade)

```text
specs/009-frame-overlays/
├── plan.md              # Este arquivo (saída do comando /speckit-plan)
├── research.md          # Saída da Fase 0 (comando /speckit-plan)
├── data-model.md         # Saída da Fase 1 (comando /speckit-plan)
├── quickstart.md        # Saída da Fase 1 (comando /speckit-plan)
├── contracts/            # Saída da Fase 1 (comando /speckit-plan)
│   ├── overlay-flags.md
│   └── plan-file-v2.md
└── tasks.md              # Saída da Fase 2 (comando /speckit-tasks — NÃO criado pelo /speckit-plan)
```

### Código-Fonte (raiz do repositório)

Projeto único já existente (hexagonal), sem mudança de layout — só arquivos
novos e extensões pontuais dentro da árvore já estabelecida:

```text
internal/domain/
├── camera_plan.go              # CameraFrame: +ActivityElapsed, +TrackElevation, +TrackElevationGain;
│                                #   CameraPlan/PlanSummary: +ElevationAvailable
├── camera_planning.go          # cameraPlanner: +trackRoute, +elevation; frame() preenche os 3 campos novos
├── duration.go                 # +Route.TimeAt
├── elevation_profile.go        # NOVO: Route.ElevationProfile, ElevationProfile.At
├── frame_overlay_config.go     # NOVO: OverlayBlock, OverlayConfig, NewOverlayConfig, Fingerprint
├── frame_screen_overlay.go     # NOVO: desenho da sobreposição de tela (texto, placa, perfil) — tipo screenOverlay
├── frame_scene.go              # Scene: +overlayConfig; Render chama screenOverlay por último
├── frame_set.go                # SingleFrameRequest/FrameSetRequest: +Overlay; NewFrameSetID/NewFrameMark: +overlay
├── render_tuning.go            # +OverlayMarginRatio, +OverlayPanelColor, +OverlayPanelOpacity, +OverlayTextColor
├── flight.go                    # tipo FlightRequest: +Overlay
└── errors.go                   # +ErrInvalidOverlayBlock

internal/infra/outbound/jsonfile/
├── camera_plan_file.go         # planFormatVersion 1→2; frameFile/markerFile/summaryFile: campos novos
├── camera_plan_exporter.go     # sem mudança de lógica (usa encodePlan)
└── camera_plan_reader.go       # rejeita format_version 1; valida presença dos campos novos

internal/infra/outbound/config/
└── config.go                   # RenderDefaults: +OverlaysEnabled, +OverlayBlocks

internal/infra/inbound/cli/
├── overlay.go                  # NOVO: parseOverlay (mesmo padrão de appearance.go)
├── render_frame.go             # +flags --overlays/--overlay-blocks
├── render_all.go                # idem
└── fly.go                      # idem

cmd/sobrevoo/
├── config_mapping.go           # +domainOverlayConfig
└── main.go                     # passa domainOverlayConfig aos comandos

internal/domain/mockdomain/, internal/application/mockapplication/
└── (sem porta nova — nenhum mock novo necessário; DTOs, não interfaces)

internal/domain/builddomain/
└── camera_frame_builder.go, e builders novos que os testes pedirem
```

**Decisão de Estrutura**: nenhuma pasta nova de alto nível — tudo estende
`internal/domain`, `internal/infra/outbound/{jsonfile,config}` e
`internal/infra/inbound/cli` já existentes, seguindo exatamente o padrão
das etapas 5 e 8 (mesmas árvores, mesmos pontos de extensão).
