# Plano de Implementação: Redesenho das Sobreposições de Tela em Colunas

**Branch**: `015-overlay-redesign` | **Data**: 2026-10-04 | **Especificação**: [spec.md](./spec.md)

**Entrada**: Especificação de funcionalidade de `/specs/015-overlay-redesign/spec.md`

**Nota**: Este template é preenchido pelo comando `/speckit-plan`; sua definição descreve o fluxo de execução.

## Resumo

O desenho das sobreposições de tela (`internal/domain/frame_screen_overlay.go`)
troca a pilha vertical de faixas escuras por uma única faixa horizontal no
alto do quadro: os blocos numéricos presentes repartem a largura útil em
colunas de mesma largura, cada um centralizado na própria, numa ordem fixa —
velocidade, elevação, distância, ganho, tempo decorrido (Clarifications) —
que nunca depende da ordem pedida em `--overlay-blocks`. Dentro de cada
bloco, três alturas de texto (rótulo por extenso, valor grande, unidade),
sem painel atrás — a legibilidade passa a depender só do contorno escuro de
texto que já existe (etapas 11/12), sem nenhuma mudança nele. O ganho de
elevação deixa de ser o segundo número do bloco de elevação e se torna um
bloco próprio (`OverlayBlockGain`, nome `gain`), pelo mesmo mecanismo de
seleção dos demais (`OverlayConfig`/`NewOverlayConfig`/`parseOverlay`); a
escolha padrão de blocos (`config.RenderDefaults.OverlayBlocks`) passa de
`distance,elevation,time,profile` para `distance,elevation,speed,profile`.
O gráfico de elevação no rodapé (`profile`) perde o mesmo painel e passa a
usar a técnica de contorno escuro antes do traço — a mesma categoria que
`TrailCasingColor`/`MarkerRingColor` e o contorno do texto já usam — para se
destacar do terreno. Nenhum valor é calculado, formatado ou arredondado de
forma diferente (FR-013); o único código que muda é o de desenho, dentro de
`internal/domain`. Como o painel desaparece e o layout deixa de depender de
medir o texto mais largo de todo o plano (`stablePanelWidth`/
`Scene.numericPanelWidth`, removidos — a largura de cada coluna é só
`largura útil / blocos presentes`, uma conta que não precisa olhar nenhum
quadro), `domain.RenderVersion` sobe de `4` para `5`, e
`OverlayConfig.Fingerprint()` passa a contar o bit de `gain` — o mecanismo
já existente de `FrameSetID` cuida do resto (`render all`/`fly --keep`
nunca misturam quadros de antes e depois desta etapa).

## Contexto Técnico

**Linguagem/Versão**: Go 1.26.4 (`go.mod`), sem mudança.

**Dependências Principais**: nenhuma nova — reaproveita
`golang.org/x/image/font/sfnt` (via `vectorFace`, já em uso desde a etapa
11) só para medir e rasterizar texto em dois corpos (rótulo/unidade
pequenos, valor grande) em vez de um; nenhuma biblioteca de layout externa.

**Armazenamento**: N/A — nenhum formato de arquivo muda (nem o plano, nem o
recorte, nem o PNG do quadro em estrutura; só os pixels que o PNG contém).

**Testes**: `go test ./... -cover`, sem mudança de ferramenta. Dado/quando/
então em `frame_overlay_config_test.go` (bloco `gain`, impressão digital) e
`frame_screen_overlay_test.go` (ordem fixa, ausência de painel, três
alturas, arranjo em colunas, contorno do gráfico de elevação); o hash de
referência de `frame_scene_test.go` é recalculado após a implementação,
como toda mudança de `RenderVersion` já exige.

**Plataforma-Alvo**: CLI de linha de comando, macOS/Linux — sem mudança.

**Tipo de Projeto**: CLI de projeto único (`cmd/sobrevoo`), sem mudança de
estrutura.

**Metas de Desempenho**: o layout por colunas é mais barato que o mecanismo
que substitui — `stablePanelWidth` percorria todos os quadros do plano para
achar o texto mais largo; a largura de cada coluna agora só depende de
quantos blocos estão presentes (configuração + disponibilidade no plano),
uma conta O(1) por quadro, sem nenhum laço sobre `plan.Frames`.

**Restrições**: determinismo byte a byte entre arquiteturas (convenção do
projeto, "O desenho dos quadros" no `CLAUDE.md`) — o layout usa só `+ − × ÷`
e `roundHalfUp`, como o resto do desenho de sobreposição já faz; nenhuma
fonte, corpo, cor, posição de faixa ou arranjo fica configurável pelo
usuário (FR-018); os rótulos continuam em português do Brasil.

**Escala/Escopo**: só o desenho de quadros (`render frame`, `render all`,
`fly`) e o valor padrão de `--overlay-blocks`; `inspect`, `geodata check`,
`plan`, `video` e a montagem do vídeo não mudam — a mesma linha que a etapa
14 já traçou.

## Verificação da Constituição

*PORTÃO: Deve passar antes da Fase 0 de pesquisa. Reverificar após o design da Fase 1.*

| Princípio | Verificação |
|---|---|
| I. Arquitetura Hexagonal | Toda a mudança de desenho vive em `internal/domain` (`frame_screen_overlay.go`, `frame_overlay_config.go`, `render_tuning.go`, `frame_scene.go`); nenhuma biblioteca de infraestrutura nova cruza para o núcleo. |
| II. Portas para Toda Dependência Externa | Nenhuma porta nova, nenhuma assinatura de porta existente muda; `vectorFace` (já interno ao domínio desde a etapa 11) só passa a ser chamado com dois tamanhos de glifo em vez de um. |
| III. Entrypoints Descartáveis | A CLI (`overlay.go`) só acrescenta `gain` à lista de nomes que `overlayBlocksOf`/`formatOverlayBlocks` já convertem, e ao texto de ajuda — nenhuma regra de negócio nova; ordem fixa, layout e disponibilidade continuam inteiramente em `internal/domain`. |
| IV. Neutralidade Geográfica | Sem mudança: nenhum dado geográfico fixo é introduzido ou alterado. |
| V. Funcionamento Offline | Sem mudança: nenhuma dependência de rede ou serviço externo novo. |
| VI. Testes Automatizados no Núcleo | `OverlayConfig`/`NewOverlayConfig`/`Fingerprint` e `screenOverlay.draw` continuam domínio puro, testável sem nenhuma porta mockada; a CLI continua mockando `application.FrameService`/`FlightService`. |
| VII. Erros Sentinela no Domínio | Nenhum sentinela novo: `gain` entra no mesmo `ErrInvalidOverlayBlock` (009) que já existe, só a lista de nomes aceitos na mensagem cresce. |
| VIII. Configuração Injetada | A nova lista padrão de blocos é um valor de `config.RenderDefaults.OverlayBlocks` (adapter `internal/infra/outbound/config`), mapeado para `domain.OverlayConfig` por `domainOverlayConfig` (composition root) — o núcleo não lê nenhuma configuração diretamente, como já era. |
| IX. Portas/Service Layer/Regra de Negócio | Nenhum serviço de aplicação muda: `FrameService`/`FlightService` continuam chamando `Scene.Render` por inteiro, sem conhecer o layout por colunas. A regra (ordem fixa, divisão em colunas, disponibilidade de cada bloco, contorno do gráfico) é toda construtor/método de domínio (`OverlayConfig`, `screenOverlay`), nunca um helper em `internal/application`. |
| X. Testes: Given/When/Then, Builders, Isolamento | `builddomain.OverlayConfigBuilder` ganha `WithGain()`, mesmo padrão de `WithSpeed()` (etapa 14); testes seguem given/when/then, um cenário por `t.Run`, sem tabela. |

Nenhuma violação; nada a registrar em Rastreamento de Complexidade.

## Estrutura do Projeto

### Documentação (desta funcionalidade)

```text
specs/015-overlay-redesign/
├── plan.md                        # Este arquivo (saída do comando /speckit-plan)
├── research.md                    # Saída da Fase 0 (comando /speckit-plan)
├── data-model.md                  # Saída da Fase 1 (comando /speckit-plan)
├── quickstart.md                  # Saída da Fase 1 (comando /speckit-plan)
├── contracts/                     # Saída da Fase 1 (comando /speckit-plan)
│   ├── overlay-blocks-update.md
│   └── frame-files-change.md
└── tasks.md                       # Saída da Fase 2 (comando /speckit-tasks — NÃO criado pelo /speckit-plan)
```

### Código-Fonte (raiz do repositório)

Projeto único já existente (hexagonal), sem mudança de layout — só extensões
e remoções pontuais dentro da árvore já estabelecida pelas etapas 9/11/12/14:

```text
internal/domain/
├── frame_overlay_config.go    # +OverlayBlockGain; OverlayConfig: +Gain bool;
│                               #  NewOverlayConfig: +case gain (mensagem de erro
│                               #  lista os 6 nomes); Fingerprint: +1 flag;
│                               #  +overlayBlockOrder (ordem fixa, Clarifications)
├── frame_screen_overlay.go    # draw: reescrito — monta a lista de blocos
│                               #  presentes na ordem fixa, divide a largura útil
│                               #  em colunas, centraliza cada bloco na própria;
│                               #  elevationBlockText perde o ganho; +gainBlockText;
│                               #  +drawBlock (três alturas: rótulo/valor/unidade,
│                               #  sem painel); formatOverlay* passam a devolver
│                               #  valor e unidade separados; drawProfile: remove
│                               #  drawPanel, acrescenta contorno (casca antes do
│                               #  núcleo) à linha e ao marcador; stablePanelWidth
│                               #  e drawPanel removidos (sem mais uso)
├── frame_scene.go             # Scene: remove panelWidth/panelWidthHeight/
│                               #  panelWidthSet e numericPanelWidth (o layout por
│                               #  colunas não depende de medir plan.Frames)
└── render_tuning.go           # RenderVersion: 4 → 5; remove OverlayPanelColor/
                                #  OverlayPanelOpacity (painel removido); +constantes
                                #  de corpo (rótulo/unidade pequenos, valor grande)
                                #  e de contorno do gráfico de elevação

internal/infra/outbound/config/
└── config.go                  # RenderDefaults: OverlayBlocks padrão passa de
                                #  {"distance","elevation","time","profile"} para
                                #  {"distance","elevation","speed","profile"}

internal/infra/inbound/cli/
└── overlay.go                 # overlaysUsage/overlayBlocksUsage: menciona gain e
                                #  a nova escolha padrão; overlayBlocksOf: +ramo Gain

internal/domain/builddomain/
└── overlay_config_builder.go  # +WithGain(), mesmo padrão de WithSpeed()
```

**Decisão de Estrutura**: nenhuma pasta nova de alto nível — tudo estende
`internal/domain`, `internal/infra/{outbound/config,inbound/cli}` e
`internal/domain/builddomain` já existentes, seguindo o mesmo padrão das
etapas 9, 11, 12 e 14 (uma mudança de desenho e/ou de configuração padrão,
sem porta nova nem serviço novo).

## Rastreamento de Complexidade

> **Preencher SOMENTE se a Verificação da Constituição tiver violações que precisam ser justificadas**

Nenhuma violação.
