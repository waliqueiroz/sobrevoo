# Plano de Implementação: Aparência Ajustável dos Quadros

**Branch**: `008-frame-appearance` | **Data**: 2026-09-28 | **Especificação**: [spec.md](./spec.md)

**Entrada**: Especificação de funcionalidade de `/specs/008-frame-appearance/spec.md`

## Resumo

Oitava etapa do Sobrevoo: torna ajustável, por flag, a cor e a espessura do
traçado, a cor e o raio do marcador, e a cor do fundo — os cinco valores que
hoje são fixos em `internal/domain/render_tuning.go`. Nada do que a ferramenta
calcula muda (trajeto, câmera, dados, enquadramento, vídeo); só cinco
parâmetros de estilo passam a ser lidos pela CLI, com os mesmos nomes, em
`render frame`, `render all` e `fly`, e a fazer parte da identidade de um
conjunto de quadros.

A abordagem técnica (detalhada em `research.md`):

- Um tipo de domínio novo, `Appearance` (`internal/domain/frame_appearance.go`):
  os cinco campos ajustáveis, `ParseColor` (formato `#RRGGBB`), `NewAppearance`
  (valida as duas proporções contra um intervalo documentado) e um
  `Fingerprint()`, no mesmo estilo de `RenderTuning.Fingerprint()` (item 1-2).
  **Não** vira campo de `RenderTuning`: é um valor por chamada, do mesmo jeito
  que `Resolution`/`Overwrite` já são — entra em `SingleFrameRequest`,
  `FrameSetRequest` e `FlightRequest`, não no construtor do serviço (item 1).
- **Um parser só**, `parseAppearance` (`internal/infra/inbound/cli/appearance.go`),
  chamado pelos três comandos — garante nomes e comportamento idênticos por
  construção (FR-002, item 4). Cor malformada vira `ErrInvalidColor`
  diretamente (como `--aspect` já faz); espessura/raio que não é um número
  vira erro de uso (como `--fps`/`--duration` já fazem), fora do intervalo vira
  `ErrInvalidTrailWidth`/`ErrInvalidMarkerRadius` (item 3).
- **Identidade do conjunto**: `NewFrameSetID` passa a escrever também
  `appearance.Fingerprint()` no hash que já monta — nenhuma versão de desenho
  nova, porque estender o hash já garante, sozinho, que um conjunto de outra
  aparência (inclusive de antes desta etapa) nunca bate com um de agora (item
  5). **Reaproveitamento do `fly --keep`**: nenhuma lógica nova em
  `FlightService` — o mecanismo de `FrameDirectory.Plan` que já decide
  reaproveitar ou redesenhar por `FrameSetID` passa a enxergar aparência
  diferente de graça; `CameraPlan.ID()`/`GeoSlice.ContentID`, dos quais o plano
  e o recorte dependem, nunca dependeram de aparência e continuam sem
  depender (item 7).
- **Configuração**: os cinco valores de hoje migram de `var` de domínio para
  `config.RenderDefaults` (que já guarda "o que se usa quando o usuário não
  escolhe nada" para a resolução) e um mapeamento novo,
  `domainAppearance`, no composition root (item 6) — mesma técnica que
  `domainRenderResolution` já usa.

A arquitetura hexagonal é preservada: nenhuma porta nova (validar e desenhar
com uma aparência são funções puras de domínio), nenhum I/O novo. O núcleo
ganha um tipo, dois erros sentinela a mais que os três já contados
(`ErrInvalidColor`, `ErrInvalidTrailWidth`, `ErrInvalidMarkerRadius`), e
parâmetros novos em assinaturas existentes; a CLI só traduz cinco flags a mais
e três códigos de saída novos.

## Contexto Técnico

**Linguagem/Versão**: Go 1.26 (sem mudança)

**Dependências Principais**: nenhuma nova no módulo Go — só biblioteca padrão
(`strconv`, `strings`, já usadas por `ParseAspectRatio`/`ParseResolution`, o
mesmo estilo para `ParseColor`). Cobra, testify e `go.uber.org/mock` como
antes.

**Armazenamento**: nenhum estado persistente novo. A aparência não é
gravada em nenhum arquivo por si — só participa do hash `FrameSetID` que já
vai dentro de cada PNG (`FrameMark`, sem mudança de forma). `plan.json` e
`slice.zip` continuam sem nenhum campo de aparência, porque nunca dependeram
dela (item 7 de `research.md`).

**Testes**: mesmo padrão — `go test` com testify, given/when/then, um
`t.Run` por cenário, sem tabelas. `Appearance`/`ParseColor`/`NewAppearance`
testados como qualquer outro construtor de domínio validado
(`NewResolution`/`ParseAspectRatio`), sem mock. `Scene`/`overlay` continuam
testados com um `TileDecoder` de mentira, agora também variando a aparência
(um teste novo: duas aparências diferentes produzem imagens diferentes; a
mesma aparência, duas vezes, produz a mesma imagem — reforça SC-003).
`FrameService`/`FlightService`, com as portas/serviços já mockados de sempre,
verificando que `request.Appearance` chega a `NewScene`/`NewFrameMark` sem
alteração. A CLI, com os serviços mockados, cobrindo os três comandos e os
três erros novos.

**Plataforma-Alvo**: mesmo binário multiplataforma (Linux, macOS, Windows);
nenhuma dependência de plataforma nova.

**Tipo de Projeto**: CLI (projeto único em Go, sem frontend/backend).

**Metas de Desempenho**: nenhuma — `ParseColor`/`NewAppearance` são
comparações e conversões triviais; `Fingerprint()` é uma concatenação de texto
curta, chamada uma vez por conjunto de quadros (já é o que
`RenderTuning.Fingerprint()` faz hoje). Nenhum impacto mensurável no tempo de
desenho de um quadro.

**Restrições**: o resultado padrão (sem nenhuma flag de aparência) precisa
continuar pixel a pixel idêntico ao de hoje (FR-003/SC-002); o mesmo plano,
recorte, resolução e aparência continuam produzindo imagens idênticas byte a
byte (FR-010/SC-003); nenhum valor de aparência pode alterar enquadramento,
posição do marcador, geometria do terreno ou duração (FR-006); o hachurado e o
xadrez de falta de dado não podem ser afetados (FR-005).

**Escala/Escopo**: uso pessoal — três comandos ganham cinco flags cada;
nenhum limite novo de volume de dados.

## Verificação da Constituição

*PORTÃO: Deve passar antes da Fase 0 de pesquisa. Reverificar após o design da Fase 1.*

Avaliada antes da pesquisa e **reavaliada após o design** (data-model,
contrato e quickstart); o resultado não mudou.

| Princípio | Avaliação | Como o design atende |
|---|---|---|
| I. Arquitetura Hexagonal | PASSA | `Appearance`, `ParseColor` e `NewAppearance` são domínio puro, sem I/O. `Scene`/`overlay`/`imagery` continuam só recebendo o valor já validado, sem saber de CLI nem de config. |
| II. Portas para Toda Dependência Externa | PASSA | Nenhuma porta nova: validar uma cor/proporção e desenhar com ela não é I/O nem depende de estado fora do processo — só um parâmetro a mais em funções já existentes. |
| III. Entrypoints Descartáveis | PASSA | `parseAppearance` traduz flag → `domain.Appearance`; a regra de validação (intervalo, formato) vive em `NewAppearance`/`ParseColor`, não na CLI — um futuro adapter REST reaproveitaria as mesmas funções de domínio, só trocando de onde o texto vem. |
| IV. Neutralidade Geográfica | PASSA | Nenhum dado, constante ou caso especial de região; a aparência não depende de latitude/longitude. |
| V. Funcionamento Offline | PASSA | Nenhuma dependência nova de rede ou serviço externo. |
| VI. Testes Automatizados no Núcleo | PASSA | `Appearance`/`ParseColor`/`NewAppearance` testados sem mock (funções puras); `Scene`/`overlay` continuam com `TileDecoder` mockado; `FrameService`/`FlightService` com as portas/serviços já mockados de sempre. |
| VII. Erros Sentinela no Domínio | PASSA | Três erros novos em `errors.go` (`ErrInvalidColor`, `ErrInvalidTrailWidth`, `ErrInvalidMarkerRadius`); a CLI os traduz nos códigos `52`–`54` (`contracts/appearance-flags.md`). |
| VIII. Configuração Injetada | PASSA | Os cinco valores-padrão migram de `var` de domínio para `config.RenderDefaults`, resolvidos por `config.Load()` e mapeados por `domainAppearance` no composition root — o núcleo nunca lê configuração diretamente; a CLI só passa o resultado já resolvido como o padrão das cinco flags novas, do mesmo jeito que já faz com `--resolution`. |
| IX. Organização de Portas, Service Layer e Mocks | PASSA | Sem `ports.go`; `Appearance` (com `ParseColor`/`NewAppearance`/`Fingerprint`) em `frame_appearance.go`, arquivo próprio — não tem uma única entidade dona, como `simplification.go`. Nenhuma interface nova, logo nenhum mock novo — só a regeneração de `mockapplication.FrameService`/`mockapplication.FlightService` pela mudança de assinatura das structs de request que já usam (não das interfaces em si, que não mudam de método). Nomeação de erro e de flag segue o padrão já existente (`ErrInvalidX`, `--palavra-kebab-case`). |
| X. Testes: Given/When/Then, Builders e Isolamento por Camada | PASSA | Todo teste em `t.Run("should ...")` com `// given`/`// when`/`// then`, sem tabelas. Builder novo `builddomain.NewAppearanceBuilder()` (cinco campos, valor padrão de hoje); `FlightRequestBuilder` ganha `WithAppearance`. Cada camada testada isolada, como já é. Sem teste de ponta a ponta automatizado; `quickstart.md` cobre o manual. |
| Idioma dos Artefatos | PASSA | Artefatos do Spec Kit em português; identificadores, pacotes, arquivos, comentários, mensagens de commit e toda E/S em tempo de execução (flags, saída, erros) em inglês, como nas etapas anteriores. |

Nenhuma violação identificada. Rastreamento de Complexidade não se aplica.

## Estrutura do Projeto

### Documentação (desta funcionalidade)

```text
specs/008-frame-appearance/
├── plan.md                       # Este arquivo (saída do comando /speckit-plan)
├── research.md                   # Saída da Fase 0 (comando /speckit-plan)
├── data-model.md                 # Saída da Fase 1 (comando /speckit-plan)
├── quickstart.md                 # Saída da Fase 1 (comando /speckit-plan)
├── contracts/
│   └── appearance-flags.md       # Saída da Fase 1: as cinco flags novas nos três comandos
├── checklists/
│   └── requirements.md           # Gerado por /speckit-specify
└── tasks.md                      # Saída da Fase 2 (comando /speckit-tasks - NÃO criado pelo /speckit-plan)
```

### Código-Fonte (raiz do repositório)

```text
cmd/sobrevoo/
├── main.go                                     # (alterado) resolve domainAppearance uma vez, passa aos três comandos
└── config_mapping.go                           # (alterado) domainAppearance(config.RenderDefaults) (domain.Appearance, error)

internal/
├── domain/
│   ├── frame_appearance.go                     # NOVO: Appearance, ParseColor, NewAppearance, Fingerprint
│   ├── frame_appearance_test.go                # NOVO
│   ├── errors.go                                # (estendido) ErrInvalidColor, ErrInvalidTrailWidth, ErrInvalidMarkerRadius
│   ├── frame_set.go                             # (alterado) SingleFrameRequest/FrameSetRequest +Appearance; NewFrameMark/NewFrameSetID +parâmetro
│   ├── frame_set_test.go                        # (alterado)
│   ├── frame_scene.go                           # (alterado) Scene +appearance; NewScene +parâmetro; Render usa s.appearance
│   ├── frame_scene_test.go                      # (alterado + cenários novos: aparências diferentes → imagens diferentes)
│   ├── frame_overlay.go                         # (alterado) overlay +appearance; drawTrail/drawMarker usam o.appearance
│   ├── frame_overlay_test.go                    # (alterado)
│   ├── frame_image.go                           # (alterado) NewFrameImage +parâmetro background
│   ├── frame_imagery.go                         # (alterado) imagery +background; newImagery +parâmetro; newTileTexture +parâmetro
│   ├── frame_imagery_test.go                    # (alterado)
│   ├── render_tuning.go                         # (alterado) remove BackgroundColor/TrailColor/MarkerColor/TrailWidthRatio/MarkerRadiusRatio (migram)
│   ├── flight.go                                # (alterado) FlightRequest +Appearance
│   ├── flight_test.go                           # (alterado, se existir asserção de forma)
│   └── builddomain/
│       ├── appearance_builder.go                # NOVO: NewAppearanceBuilder()
│       └── flight_request_builder.go            # (alterado) +Appearance no padrão, +WithAppearance
├── application/
│   ├── frame_service.go                          # (alterado) DrawFrame/DrawFrames passam request.Appearance a NewScene/NewFrameMark
│   ├── frame_service_test.go                     # (alterado + cenário novo: aparência chega sem alteração)
│   ├── flight_service.go                         # (alterado) FrameSetRequest recebe Appearance: request.Appearance
│   ├── flight_service_test.go                    # (alterado + cenário novo)
│   └── mockapplication/                          # REGERADO (make generate; assinaturas mudam, não os métodos)
└── infra/
    ├── inbound/cli/
    │   ├── appearance.go                          # NOVO: parseAppearance (compartilhado pelos três comandos), flag usage strings
    │   ├── appearance_test.go                     # NOVO
    │   ├── render_frame.go                        # (alterado) +5 flags, chama parseAppearance
    │   ├── render_frame_test.go                   # (alterado)
    │   ├── render_all.go                          # (alterado) +5 flags, chama parseAppearance
    │   ├── render_all_test.go                     # (alterado)
    │   ├── fly.go                                 # (alterado) +5 flags, chama parseAppearance
    │   ├── fly_test.go                             # (alterado)
    │   ├── exit_code.go                            # (estendido) códigos 52, 53, 54
    │   └── exit_code_test.go                       # (estendido)
    └── outbound/config/
        └── config.go                              # (alterado) RenderDefaults +5 campos, config.Load() os preenche
```

**Documentação de apoio, alterada na implementação**: `CLAUDE.md` e
`README.md` (oitava etapa: as cinco flags de aparência, nos três comandos, e
a observação de que nenhuma regra de câmera/dados/vídeo muda).

**Decisão de Estrutura**: mesmo projeto único em Go. `Appearance` em arquivo
próprio (`frame_appearance.go`), como `AspectRatio` já está em
`camera_aspect.go` — um valor de domínio sem uma única entidade dona, mas com
construtor, parser e validação juntos. `appearance.go` na CLI segue o mesmo
raciocínio de `resolutionUsage`/`formatResolution` (compartilhado entre
comandos, sem duplicar).

**Ordem de execução (base para o `/speckit-tasks`)**: (1) domínio puro, sem
E/S: `Appearance`, `ParseColor`, `NewAppearance`, `Fingerprint`, os três erros
sentinela — `make test` verde para o arquivo novo sozinho; (2) threading pelo
desenho: `NewFrameImage`, `imagery`/`newTileTexture`, `overlay`, `Scene` — um
de cada vez, cada um com seus testes existentes ainda verdes (a aparência
padrão nos testes reproduz os valores de hoje, então nenhum teste de imagem
muda de resultado esperado); (3) identidade: `NewFrameMark`/`NewFrameSetID`
com o parâmetro novo; (4) `config.RenderDefaults` + `domainAppearance` +
`main.go` (as vars de domínio somem, tudo que as lia agora lê o parâmetro
threaded); (5) `application`: `SingleFrameRequest`/`FrameSetRequest`/
`FlightRequest` +`Appearance`, `FrameService`/`FlightService` repassando;
`make generate` para os mocks; (6) CLI: `appearance.go` (`parseAppearance`),
as cinco flags nos três comandos, os três códigos de saída novos; (7)
`quickstart.md` (validação manual completa), `CLAUDE.md`, `README.md`. A
História 1 (P1, escolher e ver o efeito num quadro) já exige o desenho
completo (2), (3), (5) e (6) para `render frame`; as Histórias 2 e 3 (P2/P3)
saem de graça do mesmo parser compartilhado (4); a História 4 (P4, retomada e
`fly --keep`) não pede nenhuma linha nova além do que (3) já garante — só
precisa do quickstart para confirmar.

## Rastreamento de Complexidade

Nenhuma violação da constituição; nada a justificar.

## Riscos e Pontos de Atenção

- **R1 — Superfície de mudança de assinatura**: `NewScene`, `NewFrameImage`,
  `newImagery`, `NewFrameMark` e `NewFrameSetID` ganham um parâmetro cada —
  toda chamada existente (produção e teste) precisa do valor novo. O risco não
  é de design (a mudança é mecânica), mas de volume: `frame_scene_test.go`,
  `frame_overlay_test.go`, `frame_imagery_test.go` e `frame_set_test.go`
  têm muitos cenários que constroem esses valores diretamente. Mitigação: uma
  aparência-padrão de teste (idêntica aos valores de hoje) declarada uma vez
  por arquivo de teste, reaproveitada em todos os cenários que não testam
  aparência — o mesmo problema que `tuning` (RenderTuning) já resolve hoje nos
  mesmos arquivos.
- **R2 — `newTileTexture` e a memorização de textura por tile**: o cache de
  tiles decodificados (`tileCache`, referenciado por `imagery`) guarda
  texturas já misturadas com o fundo; se o cache sobreviver entre duas
  chamadas de `Scene.Render` com fundos diferentes (não é o caso hoje — um
  `Scene` é construído uma vez por `NewScene`, que já recebe a aparência final
  daquela chamada), a mistura ficaria presa ao primeiro fundo. Como
  `imagery`/`tileCache` são sempre recriados dentro de `NewScene` (nunca
  reaproveitados entre chamadas com aparências diferentes — cada `render
  frame`/cada quadro de `render all` usa o `Scene` da própria chamada de
  `DrawFrame`/`DrawFrames`), o risco não se concretiza; documentado para quem
  mexer nesse cache no futuro não reintroduzir o problema.
- **R3 — `TrailMinWidth`/`MarkerMinRadius` e uma proporção muito pequena**:
  como antes desta etapa, uma proporção que resultaria em menos pixels que o
  piso mínimo é elevada até o piso — o piso continua fixo (não é ajustável),
  então um `--trail-width` no mínimo do intervalo (`0.0005`) numa resolução
  pequena (180 px de altura) produz exatamente `TrailMinWidth` (2 px), não
  menos. Comportamento intencional (Caso Extremo do `spec.md`), só reafirmado
  aqui para quem for escrever o teste de fronteira.
- **R4 — Mensagem de erro de intervalo com proporção, não com pixels**: como o
  valor validado é a proporção (não o pixel final), a mensagem de
  `ErrInvalidTrailWidth`/`ErrInvalidMarkerRadius` cita o intervalo em
  proporção (`"0.5, must be from 0.0005 to 0.05"`), não em pixels — o usuário
  vê o mesmo número que digitou, não uma conversão. Decisão deliberada
  (consistente com a clarificação de que é a proporção que é validada), não
  um descuido a corrigir depois.
