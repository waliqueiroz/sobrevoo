---

description: "Task list for feature implementation"
---

# Tarefas: Controle do Registro de Dados Geográficos

**Entrada**: Documentos de design de `/specs/010-geo-data-source-control/`

**Pré-requisitos**: `plan.md`, `spec.md`, `research.md`, `data-model.md`,
`contracts/registry-clear.md`, `contracts/source-selection-flags.md`,
`quickstart.md`

**Testes**: incluídos. A constituição do projeto (Princípio VI — Testes
Automatizados no Núcleo; Princípio X — Given/When/Then, builders e
isolamento por camada) exige testify e uber-go/mock e proíbe testes
tabulares: cada cenário é um `t.Run("should ...")` com `// given`, `//
when`, `// then`. As tarefas de teste ficam antes da implementação
correspondente e devem falhar primeiro.

**Organização**: as tarefas são agrupadas por história de usuário (P1–P4 de
`spec.md`). Diferente das etapas anteriores, **não há Fase Foundational**
com tarefas: as duas trilhas desta etapa — limpar o registro (US1) e
escolher explicitamente uma fonte (US2, com US3/US4 como sua rede de
segurança) — não compartilham nenhum ponto de extensão novo; cada uma
parte direto do registro (etapa 2) e do recorte (etapa 4) já existentes.
US3 e US4 não acrescentam lógica de produção nova: `SourceSelection.Resolve`
(US2) já garante, por construção, o "nunca misturar" e a recusa por
cobertura incompleta (US3); só falta, para o FR-011, um método novo em
`GeoSlice` e uma condição a mais em `FlightService.reuseSlice` (US4).

**Divisão do trabalho entre histórias**:

| História | O que entrega | Veículo |
|---|---|---|
| US1 (P1) | limpar o registro inteiro com um comando | `geodata clear --confirm` |
| US2 (P2) | escolher a fonte pelo nome, nos três comandos que leem o registro | `--base-map`/`--elevation` em `geodata check`, `geodata slice`, `fly` |
| US3 (P3) | prova de que a escolha explícita nunca mistura nem completa em silêncio | nenhuma lógica nova — só testes de integração e `quickstart.md` |
| US4 (P4) | `fly --keep` nunca reaproveita um recorte de outra fonte | `GeoSlice.EnsureUsesSelection` + uma condição em `FlightService.reuseSlice` |

Enquanto a US1 não existe, `geodata clear` não existe. Enquanto a US2 não
existe, nenhum comando aceita `--base-map`/`--elevation` — tudo continua na
seleção automática de sempre. Enquanto a US3 não existe (mas a US2 já
existe), o comportamento de "nunca misturar" já está correto na prática
(por construção de `Resolve`) — só falta a prova em teste. Enquanto a US4
não existe, `fly --keep` já aceita as flags novas (via US2), mas pode
reaproveitar em silêncio um recorte de outra fonte — exatamente o bug que
motivou a História 4.

## Formato: `[ID] [P?] [Story] Descrição`

- **[P]**: pode ser executado em paralelo (arquivos diferentes, sem
  dependência de tarefa incompleta). Tarefas que editam o mesmo arquivo
  NUNCA são marcadas `[P]` entre si, mesmo quando logicamente independentes.
- **[Story]**: a qual história de usuário esta tarefa pertence (US1 a US4).
  Tarefas de Setup e Polish não têm esse rótulo.
- Toda tarefa inclui o caminho de arquivo exato a criar/editar.
- Comentários de código, identificadores, mensagens de commit, flags, saída
  e mensagens de erro em tempo de execução: **inglês**; artefatos do Spec
  Kit: português (constituição, "Idioma dos Artefatos").

## Convenções de Caminho

Mesmo projeto único em Go, mesma estrutura hexagonal de `plan.md`:
`cmd/sobrevoo/`, `internal/domain/`, `internal/application/`,
`internal/infra/outbound/jsonfile`, `internal/infra/inbound/cli/`.
Builders em `internal/domain/builddomain`. Regras de estilo: sem
`ports.go`; `SourceSelection` em arquivo próprio, sem entidade dona, como
`Appearance`/`OverlayConfig` já são; receivers curtos e consistentes com o
tipo (`s` `SourceSelection`, `g` `GeoSlice`, `s` `*Service`); nome
exportado nunca repete o pacote; receiver sem uso fica sem nome.

---

## Phase 1: Setup (Shared Infrastructure)

**Propósito**: um ponto de partida próprio (branch) e verde.

- [X] T001 Criar e mudar para o branch `010-geo-data-source-control` a partir de `main` (`git checkout -b 010-geo-data-source-control`)
- [X] T002 Confirmar `make build`, `make test`, `make lint` e `make generate` verdes antes de qualquer mudança (linha de base)

**Checkpoint**: repositório pronto para a Phase 3 (não há Phase 2/Foundational nesta etapa — ver nota no topo do arquivo).

---

## Phase 3: User Story 1 — Recomeçar o registro do zero (Priority: P1) 🎯 MVP

**Objetivo**: `sobrevoo geodata clear --confirm` remove todas as entradas
do registro de uma vez, sem tocar nenhum arquivo de dado geográfico; sem
`--confirm`, recusa e informa quantas entradas seriam removidas.

**Teste Independente**: `quickstart.md` itens 1 e 2.

- [X] T003 [P] [US1] Adicionar `ErrRegistryClearNotConfirmed` a `internal/domain/errors.go` (FR-003)
- [X] T004 [P] [US1] Adicionar `Clear() error` à interface `GeoDataRepository` em `internal/domain/geo_data_source.go`, com o comentário do contrato: remove todas as entradas de uma vez, nunca toca um arquivo de dado geográfico (FR-001, FR-002)
- [X] T005 [US1] Gerar o mock (`make generate`), produzindo `internal/domain/mockdomain/geo_data_repository.go` com o método `Clear` novo (depende de T004)
- [X] T006 [P] [US1] Estender `internal/infra/outbound/jsonfile/geo_data_repository_test.go`: `Clear()` sobre um registro com entradas grava um arquivo cujo `List()` seguinte devolve vazio; `Clear()` sobre um registro já vazio (arquivo inexistente ou com uma lista vazia) não é erro; a escrita usa o mesmo caminho atômico que `Save`/`Delete` já usam (arquivo temporário no mesmo diretório + `os.Rename`)
- [X] T007 [US1] Implementar `Clear() error` em `internal/infra/outbound/jsonfile/geo_data_repository.go` (`return r.write(nil)`); `make test` verde (depende de T004, T006)
- [X] T008 [P] [US1] Estender `internal/application/geo_data_service_test.go`: `Clear(false)`, com o repositório mockado devolvendo 3 fontes via `List`, devolve `(3, ErrRegistryClearNotConfirmed)` e nunca chama `repository.Clear` (`.Times(0)`); `Clear(true)` chama `repository.Clear()` e devolve `(3, nil)`; `Clear(true)` sobre um registro vazio devolve `(0, nil)`
- [X] T009 [US1] Adicionar `Clear(confirmed bool) (int, error)` à interface `GeoDataService` e implementar em `internal/application/geo_data_service.go`: busca `repository.List()`, sem `confirmed` devolve a contagem com `ErrRegistryClearNotConfirmed`, com `confirmed` chama `repository.Clear()` e devolve a contagem; `make test` verde (depende de T004, T008)
- [X] T010 [P] [US1] Gerar o mock (`make generate`), produzindo `internal/application/mockapplication/geo_data_service.go` com `Clear` (depende de T009)
- [X] T011 [P] [US1] Escrever `internal/infra/inbound/cli/geodata_clear_test.go` (novo arquivo): sem `--confirm`, o serviço mockado devolvendo `(3, ErrRegistryClearNotConfirmed)` propaga o erro (mapeia para o código `57` via `exit_code.go`); com `--confirm`, chama `geoDataService.Clear(true)` e imprime `Cleared the registry: 3 entries removed.`; um argumento posicional extra é erro de uso do próprio Cobra
- [X] T012 [US1] Implementar `NewGeoDataClearCommand(geoDataService application.GeoDataService) *cobra.Command` em `internal/infra/inbound/cli/geodata_clear.go`: flag `--confirm` (booleana), chama `Clear(confirm)`, formata a saída de `contracts/registry-clear.md` (singular/plural de "entry"/"entries" não é exigido pelo contrato — usar sempre "entries", igual a `contracts/registry-clear.md`); `make test` verde (depende de T009, T011)
- [X] T013 [P] [US1] Estender `internal/infra/inbound/cli/exit_code_test.go`: `errors.Is(err, domain.ErrRegistryClearNotConfirmed)` → `57`
- [X] T014 [US1] Adicionar o `case` a `internal/infra/inbound/cli/exit_code.go` (`57`, na posição ordenada depois do `55` já existente — `56` fica reservado para a US2, Phase 4) (depende de T003, T013)
- [X] T015 [US1] Em `cmd/sobrevoo/main.go`: `geoDataCommand.AddCommand(cli.NewGeoDataClearCommand(geoDataService))` (depende de T012)
- [X] T016 [US1] Validação manual: `quickstart.md` itens 1 e 2 (recusa sem `--confirm` citando a contagem; `--confirm` limpa e preserva os arquivos; registro já vazio não é erro)

**Checkpoint**: `geodata clear` funciona de ponta a ponta — MVP entregue.

---

## Phase 4: User Story 2 — Escolher explicitamente qual fonte usar (Priority: P2)

**Objetivo**: `--base-map <nome>`/`--elevation <nome>`, com o mesmo nome e
o mesmo efeito, em `geodata check`, `geodata slice` e `fly`; um nome
inexistente ou do tipo errado recusa antes de qualquer outro trabalho;
nada muda para quem não usa as flags.

**Teste Independente**: `quickstart.md` itens 3, 4, 6 e 9.

### `domain.SourceSelection` (research.md item 1; data-model.md)

- [X] T017 [P] [US2] Adicionar `ErrDataSourceTypeMismatch` a `internal/domain/errors.go` (FR-006)
- [X] T018 [P] [US2] Escrever `internal/domain/source_selection_test.go` (novo arquivo): `Resolve` com os dois campos `nil` devolve as duas listas de candidatos inalteradas (seleção automática, FR-009); com `BaseMapName` apontando para um nome presente em `baseMaps`, devolve `[]GeoDataSource{aquele}` para o mapa base e `elevations` inalterada (FR-008); simétrico para `ElevationName`; um nome presente em `elevations` mas pedido como `BaseMapName` devolve `ErrDataSourceTypeMismatch` citando o nome e o tipo pedido; um nome ausente das duas listas devolve `ErrDataSourceNotRegistered` citando o nome; os dois campos preenchidos ao mesmo tempo, um válido e outro com nome inexistente, recusa pelo campo inválido (FR-005, FR-008 — os dois tipos são resolvidos de forma independente, mas a chamada só devolve um erro por vez)
- [X] T019 [US2] Implementar `internal/domain/source_selection.go` (novo arquivo): `SourceSelection{BaseMapName, ElevationName *string}`, `Resolve(baseMaps, elevations []GeoDataSource) (resolvedBaseMaps, resolvedElevations []GeoDataSource, err error)` (research.md item 1: função pura, sem I/O, só filtra as listas recebidas); `make test` verde (depende de T017, T018)

### `geodata check` (research.md item 2; contracts/source-selection-flags.md)

- [X] T020 [P] [US2] Estender `internal/application/geo_data_service_test.go` (`CheckCoverage`): a assinatura ganha `selection domain.SourceSelection`; com uma seleção que resolve para um único candidato de cada tipo, `Route.Coverage` é chamado com as listas já resolvidas; um nome pedido que não existe (ou é do tipo errado) recusa **antes** de `trackService.Clean` ser chamado (mockar `trackService` para garantir zero chamadas); uma fonte registrada cujo arquivo não está mais no disco não pode ser pedida pelo nome (mesmo erro de "não registrada" — `partitionAvailableSources` já a exclui antes de `Resolve` ver as listas)
- [X] T021 [US2] Em `internal/application/geo_data_service.go`: `CheckCoverage(reader io.Reader, selection domain.SourceSelection) (domain.CoverageReport, error)` — busca e particiona as fontes, chama `selection.Resolve` **antes** de `trackService.Clean`, e passa as listas resolvidas a `route.Coverage`; `make test` verde (depende de T019, T020)
- [X] T022 [P] [US2] Gerar o mock (`make generate`), produzindo `internal/application/mockapplication/geo_data_service.go` com a assinatura nova de `CheckCoverage` (depende de T021)
- [X] T023 [P] [US2] Escrever `internal/infra/inbound/cli/source_selection_test.go` (novo arquivo): `parseSourceSelection` devolve `SourceSelection{}` (os dois campos `nil`) quando nenhuma das duas flags foi mudada; `--base-map`/`--elevation` mudados preenchem o respectivo ponteiro com o valor exato da flag; as duas flags são independentes uma da outra
- [X] T024 [US2] Implementar `parseSourceSelection(cmd *cobra.Command, baseMapFlag, elevationFlag string) domain.SourceSelection` em `internal/infra/inbound/cli/source_selection.go` (novo arquivo), usando `cmd.Flags().Changed(...)` como `parseAppearance`/`parseOverlay` já fazem; sem validação (nunca devolve erro); `make test` verde (depende de T023)
- [X] T025 [P] [US2] Estender `internal/infra/inbound/cli/geodata_check_test.go`: `--base-map`/`--elevation` chegam em `geoDataService.CheckCoverage` como o `domain.SourceSelection` esperado; omitidas, chega `domain.SourceSelection{}`; um erro de `CheckCoverage` (nome inexistente/tipo errado) propaga sem imprimir nenhum relatório
- [X] T026 [US2] Adicionar as duas flags a `NewGeoDataCheckCommand`, chamando `parseSourceSelection` antes de `runGeoDataCheck`, em `internal/infra/inbound/cli/geodata_check.go`; `make test` verde (depende de T024, T025)

### `geodata slice` (research.md item 2; contracts/source-selection-flags.md)

- [X] T027 [P] [US2] Estender `internal/application/geo_slice_service_test.go` (`Generate`): a assinatura ganha `selection domain.SourceSelection`; com uma seleção que resolve para um único mapa base, `area.Regions`/o restante do método usam só esse candidato — o `Sources` do resumo final lista exclusivamente ele; um nome pedido que não existe (ou é do tipo errado) recusa **antes** de `area.Regions`/qualquer estimativa de tamanho ou leitura de conteúdo (mockar `baseMapReader`/`elevationReader` para garantir zero chamadas)
- [X] T028 [US2] Em `internal/application/geo_slice_service.go`: `Generate(plan domain.CameraPlan, selection domain.SourceSelection) (domain.GeoSlice, error)` — chama `selection.Resolve` logo depois de `partitionAvailableSources`, antes de `area.Regions`; `make test` verde (depende de T019, T027)
- [X] T029 [P] [US2] Gerar o mock (`make generate`), produzindo `internal/application/mockapplication/geo_slice_service.go` com a assinatura nova de `Generate` (depende de T028)
- [X] T030 [P] [US2] Estender `internal/infra/inbound/cli/geodata_slice_test.go`: `--base-map`/`--elevation` chegam em `geoSliceService.Generate` como o `domain.SourceSelection` esperado; omitidas, chega `domain.SourceSelection{}`
- [X] T031 [US2] Adicionar as duas flags a `NewGeoDataSliceCommand`, chamando `parseSourceSelection`, em `internal/infra/inbound/cli/geodata_slice.go`; `make test` verde (depende de T024, T030)

### `fly` (research.md item 2; contracts/source-selection-flags.md)

- [X] T032 [P] [US2] Adicionar `Selection domain.SourceSelection` a `FlightRequest` em `internal/domain/flight.go`
- [X] T033 [P] [US2] Em `internal/domain/builddomain/flight_request_builder.go`: adicionar `WithSelection(domain.SourceSelection) *FlightRequestBuilder` (padrão: `domain.SourceSelection{}`) (depende de T032)
- [X] T034 [P] [US2] Estender `internal/application/flight_service_test.go`: `Fly` repassa `request.Selection`, sem alteração, a `geoSliceService.Generate` — tanto quando gera direto (sem `--keep`) quanto quando `--keep` decide regenerar (a `reuseSlice` ainda sem a checagem de fonte da US4 — a chamada de `Generate` dentro dela já recebe `selection`)
- [X] T035 [US2] Em `internal/application/flight_service.go`: `Fly` passa `request.Selection` a `s.geoSliceService.Generate(plan)`; `reuseSlice` ganha o parâmetro `selection domain.SourceSelection` e o repassa à sua própria chamada de `Generate`; `make test` verde (depende de T028, T032, T034)
- [X] T036 [P] [US2] Estender `internal/infra/inbound/cli/fly_test.go`: `--base-map`/`--elevation` chegam em `domain.FlightRequest.Selection`; omitidas, `domain.SourceSelection{}`
- [X] T037 [US2] Adicionar as duas flags a `NewFlightCommand`, chamando `parseSourceSelection`, preenchendo `FlightRequest.Selection`, em `internal/infra/inbound/cli/fly.go`; `make test` verde (depende de T024, T036)

### Exit code e validação

- [X] T038 [P] [US2] Estender `internal/infra/inbound/cli/exit_code_test.go`: `errors.Is(err, domain.ErrDataSourceTypeMismatch)` → `56`
- [X] T039 [US2] Adicionar o `case` a `internal/infra/inbound/cli/exit_code.go` (`56`, na posição ordenada antes do `57` que a US1 introduziu) (depende de T017, T038)
- [X] T040 [US2] Validação manual: `quickstart.md` itens 3, 4, 5, 6 e 9 (fonte pedida reflete no relatório de `check`; nome inexistente/tipo errado recusa cedo; `slice` usa exclusivamente a fonte pedida; sem flags, nada muda)

**Checkpoint**: as três flags novas funcionam, com o mesmo nome e o mesmo efeito, em `geodata check`, `geodata slice` e `fly` — MVP de seleção explícita entregue.

---

## Phase 5: User Story 3 — Nunca misturar com outra fonte sem avisar (Priority: P3)

**Objetivo**: confirmar — sem nenhuma lógica nova (já garantido, por
construção, pela Phase 4) — que uma fonte explícita que não cobre toda a
área nunca é completada em silêncio com outra fonte registrada, e que um
nome inexistente ou do tipo errado recusa antes de qualquer trabalho.

**Teste Independente**: `quickstart.md` item 7.

- [X] T041 [P] [US3] Estender `internal/application/geo_slice_service_test.go`: com uma fonte de mapa base pedida explicitamente que cobre só parte da área do plano, e outra fonte registrada (não pedida) que cobriria o restante, `Generate` recusa com `ErrAreaNotCovered` — sem usar a segunda fonte para completar (mesma verificação de hoje para cobertura incompleta, agora sobre a lista já restrita por `Resolve`)
- [X] T042 [P] [US3] Estender `internal/application/geo_data_service_test.go` (`CheckCoverage`): o mesmo cenário do teste anterior não recusa — o relatório mostra a lacuna (`UncoveredSegments` não vazio, `Missing` do tipo certo), e `BaseMapSourcesUsed` reflete só a fonte pedida, nunca a que cobriria o resto (FR-007)
- [X] T043 [US3] Validação manual: `quickstart.md` item 7 (fonte explícita sem cobertura total: `geodata slice`/`fly` recusam, `geodata check` relata a lacuna — nenhum dos dois mistura com outra fonte registrada)

**Checkpoint**: a escolha explícita nunca mistura fontes nem é completada em silêncio, em nenhum dos três comandos.

---

## Phase 6: User Story 4 — Nunca reaproveitar em `fly --keep` um recorte da fonte errada (Priority: P4)

**Objetivo**: `fly --keep` deixa de reaproveitar, em silêncio, um recorte
guardado cuja procedência não usa a fonte pedida agora — a escolha de
fonte passa a fazer parte da decisão de reaproveitar (FR-011).

**Teste Independente**: `quickstart.md` item 8.

- [X] T044 [P] [US4] Adicionar `ErrSliceUsesDifferentSource` a `internal/domain/errors.go`
- [X] T045 [P] [US4] Estender `internal/domain/geo_slice_test.go`: `EnsureUsesSelection` com os dois campos de `selection` `nil` sempre devolve `nil` (nenhuma checagem, mesmo com `Summary.Sources` vazio); com um nome que aparece em `Summary.Sources` do tipo certo, `nil`; com um nome que não aparece nesse tipo (outro nome usado, ou nenhum uso desse tipo em `Summary.Sources`), `ErrSliceUsesDifferentSource`; os dois tipos são checados de forma independente (um bate, o outro não → erro só do que não bate)
- [X] T046 [US4] Implementar `func (g GeoSlice) EnsureUsesSelection(selection SourceSelection) error` em `internal/domain/geo_slice.go`, ao lado de `EnsureMatches`/`EnsureCovers`; `make test` verde (depende de T044, T045)
- [X] T047 [P] [US4] Estender `internal/application/flight_service_test.go` (`reuseSlice`, via `Fly` com `Keep` preenchido): um recorte guardado cuja procedência usa `mapa-a` e uma chamada nova pedindo `Selection{BaseMapName: &"mapa-b"}` NÃO reaproveita (`SliceReused=false`; `geoSliceService.Generate` chamado; o resultado é exportado por cima do recorte guardado); a mesma chamada pedindo de novo `mapa-a` reaproveita (`SliceReused=true`; `Generate` **não** chamado); uma chamada sem nenhuma fonte pedida (automática) depois de uma anterior que pedia `mapa-a` explicitamente também NÃO reaproveita, e vice-versa
- [X] T048 [US4] Em `internal/application/flight_service.go`: `reuseSlice` passa a exigir `existing.EnsureMatches(plan) == nil && existing.EnsureUsesSelection(selection) == nil`; `make test` verde (depende de T035, T046, T047)
- [X] T049 [P] [US4] Estender `internal/infra/inbound/cli/exit_code_test.go`: `errors.Is(err, domain.ErrSliceUsesDifferentSource)` → `58`
- [X] T050 [US4] Adicionar o `case` a `internal/infra/inbound/cli/exit_code.go` (`58`, mapeado por completude — nenhum comando desta etapa o devolve a um usuário, ver `contracts/source-selection-flags.md`) (depende de T044, T049)
- [X] T051 [US4] Validação manual: `quickstart.md` item 8 (trocar a fonte pedida entre execuções de `fly --keep` sempre gera um recorte novo; a mesma fonte, ou a ausência dela repetida, reaproveita)

**Checkpoint**: `fly --keep` nunca reaproveita, em silêncio, um recorte guardado de uma fonte diferente da pedida agora.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Propósito**: os casos que não pertencem a nenhuma história específica, a documentação e a verificação final.

- [X] T052 [P] Atualizar `CLAUDE.md`: a décima etapa (resumo do projeto), `geodata clear`, `--base-map`/`--elevation`, `SourceSelection`, `GeoSlice.EnsureUsesSelection`, os três sentinelas novos, os três exit codes novos
- [X] T053 [P] Atualizar `README.md`: `geodata clear` e as duas flags novas nas seções de `geodata check`, `geodata slice` e `fly` já existentes (mesma estrutura que a aparência/sobreposição já têm)
- [X] T054 Verificação final: `make build`, `make test`, `make lint`, `make generate` verdes (confirmar que `make generate` não produz nenhuma diferença depois de T005/T010/T022/T029); `gofmt -l .` limpo; reexecutar `quickstart.md` por inteiro

---

## Dependências e Ordem de Execução

### Dependências entre Fases

- **Setup (Phase 1)**: sem dependências.
- **User Stories (Phase 3+)**: todas dependem só do Setup — não há fase Foundational nesta etapa.
  - US1 (Phase 3) é totalmente independente das demais.
  - US2 (Phase 4) é independente da US1, mas é pré-requisito de fato para US3 e US4 (elas dependem do código que a US2 introduz — `SourceSelection`, as três flags).
  - US3 (Phase 5) depende da US2 já estar implementada (usa `Resolve`/as flags já existentes; não adiciona produção nova).
  - US4 (Phase 6) depende da US2 (o campo `Selection` em `FlightRequest`, já repassado por `Fly`/`reuseSlice`).
- **Polish (Phase 7)**: depende de todas as histórias desejadas estarem completas.

### Dentro de Cada História de Usuário

- Testes são escritos e devem falhar antes da tarefa de implementação correspondente.
- Domínio antes de aplicação; aplicação antes de CLI; `exit_code.go` depois do sentinela que ele mapeia.
- `make generate` roda imediatamente depois de qualquer tarefa que muda uma interface (`GeoDataRepository`, `GeoDataService`, `GeoSliceService`).

### Oportunidades de Paralelização

- Todas as tarefas de Setup marcadas com `[P]` podem ser executadas em paralelo.
- US1 e US2 podem ser trabalhadas em paralelo por pessoas diferentes (nenhum arquivo em comum, exceto `exit_code.go`, cujas duas tarefas de `case` — T014 e T039 — devem ser sequenciadas entre si, não em paralelo).
- Dentro da US2, as três subseções (`geodata check`, `geodata slice`, `fly`) podem ser trabalhadas em paralelo depois que `source_selection.go` (T019, T024) estiver pronto — cada uma edita arquivos próprios.
- US3 e US4 só podem começar depois da US2 estar completa, mas podem então ser trabalhadas em paralelo entre si (arquivos diferentes: US3 só estende testes já existentes, US4 cria `geo_slice.go`/edita `flight_service.go`).

---

## Estratégia de Implementação

### MVP Primeiro (Somente História de Usuário 1)

1. Completar Phase 1: Setup.
2. Completar Phase 3: História de Usuário 1 (`geodata clear`).
3. **PARAR E VALIDAR**: `quickstart.md` itens 1–2.
4. Implantar/demonstrar se estiver pronta.

### Entrega Incremental

1. Setup → `geodata clear` (US1, MVP) → validar → demonstrar.
2. US2 (`--base-map`/`--elevation` nos três comandos) → validar → demonstrar.
3. US3 (prova de que nunca mistura) → validar.
4. US4 (`fly --keep` nunca reaproveita a fonte errada) → validar.
5. Polish.

Cada história agrega valor sem quebrar as anteriores; US3 é a mais barata
de todas (nenhuma produção nova), e US4 é a que resolve o ponto mais
importante levantado antes do planejamento (o reaproveitamento silencioso
da fonte errada em `fly --keep`).
