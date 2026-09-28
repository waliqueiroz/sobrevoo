# Pesquisa: Aparência Ajustável dos Quadros

**Feature**: `008-frame-appearance` | **Data**: 2026-09-28

Sem `NEEDS CLARIFICATION` pendente no Contexto Técnico do `plan.md` — as três
decisões que a etapa precisava (formato de cor, unidade de espessura/raio,
escopo de "proporção") já foram resolvidas em `/speckit-clarify` e estão
registradas em `spec.md` → `## Clarifications`. Esta pesquisa cobre as decisões
de desenho técnico que a especificação deixou para o planejamento.

## 1. Onde os cinco valores vivem: um tipo novo, não um campo a mais em `RenderTuning`

**Decisão**: um tipo de domínio novo, `Appearance` (`internal/domain/frame_appearance.go`),
com os cinco campos ajustáveis; **não** viram campos de `domain.RenderTuning`.

**Razão**: `RenderTuning` é resolvido **uma vez**, no processo inteiro, pelo
adapter de configuração (`config.Load()` → `config_mapping.go` →
`NewFrameService(..., renderTuning, ...)` — Princípio VIII) e não muda entre
chamadas. A aparência, ao contrário, é **por execução de comando** — cada
`render frame`/`render all`/`fly` pode pedir uma aparência diferente na mesma
sessão do processo (não é o caso hoje, mas nada no desenho atual impede, e a
CLI já lê `--resolution`/`--overwrite` assim, por chamada). O lugar certo para
um valor por chamada, no código já existente, é o `Request` da operação —
exatamente onde `Resolution` e `Overwrite` já vivem hoje
(`domain.SingleFrameRequest`, `domain.FrameSetRequest`, e, por extensão,
`domain.FlightRequest`) — não o `xTuning` injetado uma vez no construtor do
serviço.

**Alternativas consideradas**:
- Campos novos em `RenderTuning`, com o valor vindo do `Request` sobrescrevendo
  a cópia injetada no serviço a cada chamada: rejeitada — misturaria, na mesma
  struct, o que é config de processo (FOV, workers, cache) com o que é entrada
  do usuário por chamada, e `frameService.renderTuning` deixaria de ser a
  fonte única de verdade sem um mecanismo de "override" novo e sem precedente
  no código.
- Um quinto valor de `RenderTuning` fixo por processo (aparência só ajustável
  via variável de ambiente/config, nunca por flag): rejeitada — contraria
  frontalmente a spec (FR-001/FR-002: ajuste por comando, nomeado por flag).

## 2. Formato de cor e limites de espessura/raio

**Decisão**: `ParseColor(text string) (RGB, error)` em domínio, aceitando
exatamente `#RRGGBB` (7 caracteres: `#` + 6 dígitos hexadecimais,
maiúsculos ou minúsculos) — sem abreviação de 3 dígitos, sem canal alfa, sem
nomes — já decidido em `/speckit-clarify`. Erro: `ErrInvalidColor`, no mesmo
formato de mensagem que `ParseAspectRatio`/`ParseResolution` já usam (valor
recebido entre aspas + formato esperado com exemplo).

Espessura do traçado e raio do marcador continuam proporções da altura do
quadro (`TrailWidthRatio`, `MarkerRadiusRatio`, decisão do `/speckit-clarify`),
validadas por `NewAppearance` contra um intervalo documentado, tomando os
valores de hoje como centro:

| Valor | Hoje (padrão) | Mínimo aceito | Máximo aceito |
|---|---|---|---|
| `TrailWidthRatio` | 0.005 | 0.0005 | 0.05 |
| `MarkerRadiusRatio` | 0.012 | 0.001 | 0.1 |

**Razão dos limites**: cada intervalo cobre uma década para baixo e para cima
do valor de hoje (aproximadamente), o suficiente para um traçado quase
imperceptível (contido pelo piso mínimo em pixels, que não muda) até um
traçado que ocupa uma fração visível e deliberadamente grossa do quadro
(5% da altura), sem chegar a dominar a cena por engano — e o mesmo raciocínio
para o raio do marcador (até 10% da altura). Números redondos, fáceis de
documentar e de testar nas bordas; nada no pedido original fixava um valor
exato, e a spec deixou a decisão explicitamente para esta fase (`Suposições`).
Erros: `ErrInvalidTrailWidth`, `ErrInvalidMarkerRadius` — dois sentinelas
distintos (como `ErrInvalidDuration`/`ErrInvalidFrameRate` já são dois, não
um genérico), cada um citando o valor recebido e o intervalo aceito.

**Alternativas consideradas**: pixels absolutos — rejeitada em
`/speckit-clarify` (Q3). Um intervalo único e genérico ("de 0 a 1") — rejeitada
por não recusar valores tecnicamente dentro de `[0,1]` mas visualmente
absurdos (por exemplo, um raio de marcador cobrindo metade do quadro).

## 3. Onde a validação de "malformado" vira erro de uso (Cobra) e onde vira erro de domínio

**Decisão**: replicando o padrão já usado por `--fps`/`--duration` em `plan.go`:
- `--trail-color`, `--marker-color`, `--background-color`: qualquer texto que
  não seja `#RRGGBB` vira **diretamente** `ErrInvalidColor` (como
  `ParseAspectRatio` já faz para `--aspect`) — a spec pede um erro próprio
  para "cor mal formada" (FR-004), não um erro de uso genérico.
- `--trail-width`, `--marker-radius`: o texto é lido com o mesmo
  `parseFiniteNumber` que `--fps`/`--duration` já usam; um texto que não é um
  número finito vira `usageError` (código de saída `2`, como hoje);
  um número sintaticamente válido mas fora do intervalo documentado vira
  `ErrInvalidTrailWidth`/`ErrInvalidMarkerRadius` (FR-004: "fora do intervalo
  documentado").

**Razão**: consistência com o único precedente que já existe no código para
"flag numérica lida como texto" — a mesma regra que já separa "não é um
número" (uso) de "é um número, mas fora da faixa" (domínio) para `--fps` e
`--duration`.

## 4. Um único parser de aparência, compartilhado pelos três comandos

**Decisão**: uma função nova, `parseAppearance(cmd *cobra.Command, trailColorFlag,
trailWidthFlag, markerColorFlag, markerRadiusFlag, backgroundColorFlag string,
defaults domain.Appearance) (domain.Appearance, error)`, num arquivo novo
`internal/infra/inbound/cli/appearance.go`, chamada por `render frame`,
`render all` e `fly` — as três `RunE` que hoje já chamam
`domain.ParseResolution(resolutionFlag)` antes de delegar para a função
`run*`.

**Razão**: é o único jeito de garantir FR-002 (mesmos nomes, mesmos valores
aceitos, mesmo efeito) **por construção**, sem duplicar a lógica de parsing em
três arquivos que poderiam divergir com o tempo — o mesmo raciocínio que já
levou `resolutionUsage`/`formatResolution` a serem funções/constantes
compartilhadas entre `render_frame.go` e `render_all.go` hoje.

## 5. A aparência entra na identidade do conjunto de quadros estendendo o hash existente, sem um novo `RenderVersion`

**Decisão**: `Appearance` ganha um método `Fingerprint() string`, no mesmo
formato de `RenderTuning.Fingerprint()` (texto canônico dos campos que mudam a
imagem). `NewFrameSetID` (`frame_set.go`) passa a receber `appearance
Appearance` e escrever `appearance.Fingerprint()` no hash, depois de
`tuning.Fingerprint()`. `RenderVersion` **continua 2** — não sobe.

**Razão**: `NewFrameSetID` já é um SHA-256 de um texto canônico
comprimento-prefixado; acrescentar um segmento novo no fim muda o hash para
**qualquer** combinação anterior, sem exceção — inclusive para todo quadro já
desenhado por um binário anterior a esta etapa (que nunca escreveu esse
segmento). Isso já satisfaz sozinho a exigência da spec de que um conjunto
antigo nunca seja confundido com um conjunto novo de aparência padrão
(Suposição do `spec.md`), sem precisar bater a versão: `RenderVersion` existe
para quando o **algoritmo de desenho** muda a imagem para a mesma entrada
(pixels diferentes para os mesmos parâmetros) — não é o caso aqui, já que o
resultado padrão (sem nenhuma flag de aparência) continua pixel a pixel igual
ao de hoje (FR-003/SC-002).

**Alternativas consideradas**: subir `RenderVersion` para 3 — rejeitada por
desnecessária (o hash já garante a disjunção) e por sugerir, incorretamente,
que o algoritmo de desenho mudou.

## 6. Os cinco valores fixos de configuração migram de `var` de domínio para `config.RenderDefaults`

**Decisão**: `BackgroundColor`, `TrailColor`, `MarkerColor`, `TrailWidthRatio`,
`MarkerRadiusRatio` deixam de ser `var`/`const` de pacote em
`internal/domain/render_tuning.go` e passam a ser os valores-padrão de
`config.RenderDefaults` (estendida com os cinco campos), mapeados para um
`domain.Appearance` por uma função nova em `config_mapping.go`
(`domainAppearance`), do mesmo jeito que `domainRenderResolution` já mapeia
`config.RenderDefaults.Width/Height` para `domain.Resolution`. `NoMapColors`,
`NoElevationColors`, `PatternPeriod`, `TrailCasingColor`, `MarkerRingColor`,
`TrailMinWidth`, `MarkerMinRadius`, `MarkerRingRatio`, `MarkerRingMin`
continuam exatamente onde estão — fixos, sem ajuste (FR-005).

**Razão**: é a mesma migração que `RenderDefaults{Width: 1080, Height: 1920}`
já fez para a resolução — um valor "o que a ferramenta usa quando o usuário
não escolhe nada" é, por definição, configuração injetada (Princípio VIII), não
uma constante do núcleo. Os números em si não mudam (`0xFFB000`, `0.005`,
`0xE5252A`, `0.012`, `0x20262E`): só o lugar onde moram.

## 7. Reaproveitamento do `fly --keep`: nenhuma lógica nova

**Decisão**: `domain.FlightRequest` ganha `Appearance domain.Appearance`;
`FlightService.Fly` passa `Appearance: request.Appearance` para o
`domain.FrameSetRequest` que já monta para `FrameService.DrawFrames` (a mesma
literal que já preenche `Directory`/`Resolution`/`Overwrite`). Nenhuma linha
nova de decisão "redesenhar ou reaproveitar": o mecanismo já existente
(`FrameDirectory.Plan`, comparando `FrameSetID`) passa a enxergar aparências
diferentes como conjuntos diferentes **de graça**, porque `FrameSetID` agora
depende de `Appearance.Fingerprint()` (item 5). `CameraPlan.ID()` e
`GeoSlice.ContentID` — dos quais o reaproveitamento do plano e do recorte
dependem — nunca dependeram de `RenderTuning`/`Appearance` e continuam sem
depender: é por isso que FR-009 (mudar só a aparência reaproveita plano e
recorte, refaz quadros e vídeo) já sai de graça da arquitetura existente, sem
nenhum `if` novo em `FlightService`.

**Razão**: é a mesma técnica que a etapa 7 já usou para cada um dos outros
parâmetros ajustáveis (duração, distância, resolução, ...) — nenhum deles tem
lógica de reaproveitamento própria em `FlightService`; todos passam pela
identidade que o domínio já calcula.

## 8. Nenhum builder novo para `Appearance` nos serviços; um builder novo só no domínio

**Decisão**: `internal/domain/builddomain/appearance_builder.go`
(`NewAppearanceBuilder()`, com os cinco valores de hoje como padrão,
`WithTrailColor`/`WithTrailWidthRatio`/`WithMarkerColor`/
`WithMarkerRadiusRatio`/`WithBackgroundColor`/`Build()`) — segue a regra do
Princípio X (builder para struct com vários campos, usado repetidamente em
teste). `FlightRequestBuilder` ganha `WithAppearance(...)`, com o mesmo padrão
de hoje como valor inicial. `SingleFrameRequest`/`FrameSetRequest` continuam
sem builder próprio (nenhum tinha antes desta etapa; cinco campos a mais não
muda esse cálculo — os testes que precisarem de uma aparência não padrão
constroem o literal com `builddomain.NewAppearanceBuilder()...Build()`).

## 9. Nenhuma porta nova, nenhum adapter novo

**Decisão**: `ParseColor`/`NewAppearance`/`Appearance.Fingerprint()` são
funções e métodos puros do domínio — sem I/O, sem estado fora do processo.
Nenhuma porta nova, nenhum mock novo além do que a mudança de assinatura de
interfaces já existentes (`FrameService`, `FlightService`) já exige regenerar.

**Razão**: Princípio II — só exige porta o que faz I/O real; ler e validar uma
string de flag já lida (a CLI a entrega como `string`) não é I/O.
