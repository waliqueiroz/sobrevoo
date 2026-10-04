# Pesquisa: Controle do Registro de Dados Geográficos

**Feature**: `010-geo-data-source-control` | **Data**: 2026-09-29

Sem nenhum `NEEDS CLARIFICATION` pendente do Contexto Técnico — as duas
capacidades desta etapa (limpar o registro inteiro; escolher explicitamente
uma fonte) reutilizam tecnologia já presente no projeto (o próprio
`GeoDataRepository` em JSON, a CLI Cobra, os erros sentinela). O que segue
são as decisões de desenho tomadas para encaixar as duas capacidades no
código existente sem duplicar nem contornar nenhuma regra (Princípio IX).

## 1. Onde a seleção explícita se encaixa no algoritmo de seleção automática

**Decisão**: a seleção explícita não substitui `Route.Coverage`/
`SelectSource` — ela **restringe, antes de chamá-los, a lista de candidatos**
de um tipo a um único elemento (o registro pedido). Um novo tipo de domínio,
`SourceSelection` (`internal/domain/source_selection.go`), ganha um método
`Resolve(baseMaps, elevations []GeoDataSource) (resolvedBaseMaps,
resolvedElevations []GeoDataSource, err error)`: quando `BaseMapName`/
`ElevationName` é `nil`, a lista correspondente passa inalterada (seleção
automática, FR-009); quando é um nome, `Resolve` procura esse nome nas duas
listas — encontrado no tipo certo, a lista vira `[]GeoDataSource{aquele}`;
encontrado no tipo errado, `ErrDataSourceTypeMismatch`; não encontrado em
nenhuma, `ErrDataSourceNotRegistered` (sentinela já existente, reaproveitado
com uma mensagem que cita o nome e o tipo pedido).

**Justificativa**: `Route.Coverage`/`SelectSource`/`BoundingBox.Regions` já
implementam exatamente a regra "menor área cobrindo o ponto, desempate pelo
registro mais antigo" sobre a lista de candidatos que recebem — uma lista de
um único elemento faz esse mesmo algoritmo, sem nenhuma mudança, escolher
sempre aquele elemento (ele goes ou não cobre o ponto; se não cobrir, o ponto
fica sem cobertura daquele tipo, e vira lacuna — exatamente o comportamento
de FR-007, de graça). Isso também garante FR-010 (nunca misturar): com um só
candidato na lista, é estruturalmente impossível `SelectSource` escolher
outro. Nenhuma linha de `geo_data_coverage.go` muda — cumpre a exigência de
"Fora de escopo: mudar o algoritmo de seleção automática em si" da entrada
da spec, porque de fato não muda.

**Alternativas consideradas**:
- Um parâmetro novo em `Route.Coverage`/`SelectSource` para "forçar" um nome:
  rejeitado — introduziria um `if` de seleção explícita dentro do algoritmo
  automático, misturando as duas regras no mesmo método e arriscando
  exatamente o tipo de mudança que a spec pede para não fazer.
- Resolver a seleção dentro do repositório (`GeoDataRepository.FindByName`
  sozinho): rejeitado — a lista de candidatos que os serviços já montam
  (`partitionAvailableSources`) já exclui fontes cujo arquivo não existe
  mais; resolver contra as *duas listas já filtradas* (não direto contra o
  repositório) garante que uma fonte pedida sem arquivo no disco seja
  recusada pelo mesmo motivo que hoje a torna invisível à seleção
  automática, sem duplicar essa checagem.

## 2. Onde a validação de nome/tipo entra em cada comando

**Decisão**: em `GeoDataService.CheckCoverage` e `GeoSliceService.Generate`,
`SourceSelection.Resolve` é chamado imediatamente depois de
`partitionAvailableSources` — antes de qualquer leitura de conteúdo (peças de
mapa, amostras de elevação) e, em `CheckCoverage`, antes mesmo de computar a
cobertura. Para "recusar cedo" com o menor custo possível, `CheckCoverage`
passa a buscar os registros (`repository.List()`) e resolver a seleção
**antes** de tratar o trajeto (`trackService.Clean`) — inversão pequena da
ordem atual, mas que evita gastar o tratamento do trajeto quando o nome
pedido já está errado.

**Justificativa**: FR-006 exige recusar "antes de qualquer outro trabalho";
como `Resolve` é uma função pura sobre listas já em memória (sem I/O), ela é
o trabalho mais barato possível de fazer primeiro.

## 3. Onde o reaproveitamento de `fly --keep` verifica a fonte (FR-011)

**Decisão**: um novo método em `GeoSlice`, `EnsureUsesSelection(selection
SourceSelection) error` (`internal/domain/geo_slice.go`, ao lado de
`EnsureMatches`/`EnsureCovers`): para cada tipo com um nome pedido
(`BaseMapName`/`ElevationName` não nulo), procura em `Summary.Sources` — a
procedência que `NewGeoSlice` já calcula a partir do conteúdo do recorte,
independente desta etapa — um uso daquele tipo cujo `Source.Name` seja
exatamente o pedido; não achando, `ErrSliceUsesDifferentSource`. Quando
nenhum nome é pedido para um tipo, nenhuma verificação é feita para esse
tipo (o mesmo silêncio que a seleção automática já tinha antes desta etapa).
`FlightService.reuseSlice` passa a exigir `existing.EnsureMatches(plan) ==
nil && existing.EnsureUsesSelection(request.Selection) == nil` para
reaproveitar — uma condição a mais na mesma linha que já decidia isso,
nenhuma reestruturação do método.

**Justificativa**: é exatamente o mecanismo que o usuário sugeriu antes do
planejamento — comparar o que está sendo pedido agora contra a procedência
que o próprio recorte guardado já registra, sem nenhum registro novo à
parte. `Summary.Sources` (`SliceSourceUse{Source, Detail}`) já existe desde a
etapa 4 e já é escrito no arquivo exportado (`sources[]`,
`specs/004-geo-data-slice/contracts/slice-file.md`) — nada precisa mudar no
formato do arquivo nem na exportação/leitura do recorte.

**Alternativas consideradas**:
- Recomputar, a cada reaproveitamento, o que a seleção automática escolheria
  *agora* (chamando de novo `partitionAvailableSources`/`area.Regions`/
  `route.Coverage` sobre o registro atual) e comparar contra o que o recorte
  guardado usou, mesmo quando nenhum nome é pedido agora: rejeitada por ir
  além do que a spec pede (FR-011 e a História de Usuário 4 só cobrem a
  troca de *seleção explícita*, não a deriva do registro entre execuções —
  que já era, antes desta etapa, uma lacuna aceita do reaproveitamento por
  `--keep`, fora do escopo desta etapa) e por adicionar uma segunda forma de
  decidir "o que seria usado agora", quando a spec já aponta para uma única
  fonte de verdade (a procedência gravada). Registrada aqui para não ser
  redescoberta como uma "correção" durante a implementação.
- Guardar, num arquivo à parte, a seleção pedida em cada execução com
  `--keep` (não só o que foi usado): rejeitada — exigiria um novo arquivo ou
  campo no formato do recorte só para uma informação que `Summary.Sources`
  já contém (o nome do registro usado é, por construção de FR-010, sempre o
  nome pedido quando a seleção foi explícita).

**Revisão (2026-10-04)**: a decisão acima não cumpria FR-011 por inteiro e
foi substituída. Ela não verificava nada quando a seleção ficava automática,
então voltar de `--base-map X` para a seleção automática reaproveitava em
silêncio o recorte de `X` — o "ou vice-versa" de FR-011 —, e só conferia se
o nome pedido estava *entre* as fontes gravadas, então um recorte automático
que misturara dois mapas era reaproveitado para um pedido explícito de um
deles, contra FR-010. Nenhuma das duas falhas se resolve só com a
procedência gravada (ela não diz se a escolha foi explícita), então a
primeira alternativa rejeitada acima foi adotada, na forma mais estreita:
`GeoSliceService.Sources(plan, selection)` calcula, só pelos metadados do
registro e pelo mesmo caminho de `Generate` (`partitionAvailableSources`,
`SourceSelection.Resolve`, `BoundingBox.Regions`, `SliceRegions.Sources`),
as fontes de que um recorte gerado agora seria tirado, e
`GeoSlice.EnsureUsesSources(sources)` só aceita o recorte guardado se a sua
procedência for exatamente esse conjunto (tipo, nome e arquivo). Continua sem
nenhum registro novo nem mudança no formato do arquivo do recorte; a deriva
do registro entre execuções passa a ser detectada como efeito colateral, e um
recorte que sairia igual (um pedido explícito da única fonte que a seleção
automática já usava) continua reaproveitado.

## 4. Comando de limpeza do registro

**Decisão**: `geodata clear`, com uma flag `--confirm` (booleana, sem valor).
Sem ela, o comando não chama a limpeza — só lista quantas entradas existem
hoje (`GeoDataRepository.List`) e recusa com `ErrRegistryClearNotConfirmed`,
cuja mensagem cita a contagem. Com ela, `GeoDataRepository` ganha um método
novo, `Clear() error`, que grava um registro vazio (mesma escrita atômica
que `Save`/`Delete` já usam) — `GeoDataService.Clear(confirmed bool) (int,
error)` conta as entradas antes de decidir.

**Justificativa**: `--confirm` segue o mesmo padrão que `--overwrite` já
estabeleceu no resto da CLI (flag explícita, nunca prompt — Suposições da
spec). O nome do comando evita "prune", que em ferramentas como
`docker`/`git` sugere remover só o que está obsoleto/sem uso — o oposto do
que esta etapa pede (remover tudo, mesmo o que ainda tem arquivo no disco,
FR-001); "clear"/"reset" não carrega essa conotação. Entre os dois, "clear"
casa melhor com o vocabulário já usado por `GeoDataRepository.Save/Delete` (o
registro é uma coleção que se limpa, não um estado que se "reseta").

**Alternativas consideradas**: `geodata reset` — rejeitado por sugerir,
nalgumas ferramentas, "voltar a um estado padrão" (que aqui não existe: o
estado padrão é vazio, mas nada é reconfigurado); `--yes` no lugar de
`--confirm` — rejeitado por ser menos autoexplicativo isolado (`--yes` exige
contexto de qual pergunta está sendo respondida; `--confirm` é uma frase
completa por si).

## 5. Onde as duas novas flags de seleção são compartilhadas

**Decisão**: `--base-map <nome>` e `--elevation <nome>` (strings, vazio por
padrão), lidas por um parser compartilhado novo,
`parseSourceSelection(cmd, baseMapFlag, elevationFlag string)
domain.SourceSelection` (`internal/infra/inbound/cli/source_selection.go`),
no mesmo padrão de `parseAppearance`/`parseOverlay`: usa
`cmd.Flags().Changed(...)` para decidir entre `nil` (não pedido) e um
ponteiro para o valor. Diferente de `parseAppearance`/`parseOverlay`,
`parseSourceSelection` nunca falha (não valida o nome — quem sabe se ele
existe e é do tipo certo é `SourceSelection.Resolve`, que só roda depois de
buscar o registro), então sua assinatura não devolve `error`.

**Justificativa**: mesmo racional das etapas 8/9 — um único parser
compartilhado garante, por construção, que `geodata check`, `geodata slice`
e `fly` aceitem exatamente as mesmas duas flags, com o mesmo nome e o mesmo
efeito (FR-004).

## 6. `geodata elevation` fica fora

**Decisão confirmada (não uma decisão nova)**: `geodata elevation` não recebe
`--base-map`/`--elevation` nesta etapa. A Entrada da spec já não o cita, e a
elevação de uma coordenada (`GeoDataService.ElevationAt`) já opera sobre um
só tipo de dado (elevação) — a ambiguidade "mapa base vs. elevação" que
motiva `--base-map`/`--elevation` como duas flags separadas não existe ali.
Fica registrado aqui só para não ser levantado como dúvida durante a
implementação.
