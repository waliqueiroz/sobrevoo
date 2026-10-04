# Plano de Implementação: Controle do Registro de Dados Geográficos

**Branch**: `010-geo-data-source-control` | **Data**: 2026-09-29 | **Especificação**: [spec.md](./spec.md)

**Entrada**: Especificação de funcionalidade de `/specs/010-geo-data-source-control/spec.md`

**Nota**: Este template é preenchido pelo comando `/speckit-plan`; sua definição descreve o fluxo de execução.

## Resumo

Acrescenta duas capacidades ao registro local de dados geográficos: um
comando (`geodata clear`) que remove todas as entradas do registro de uma
vez — nunca os arquivos em disco — só com confirmação explícita
(`--confirm`); e duas flags (`--base-map`, `--elevation`) que deixam o
usuário escolher, pelo nome já usado ao registrar, qual fonte usar em
`geodata check`, `geodata slice` e `fly`, em vez de depender sempre da
seleção automática por área. A escolha explícita se encaixa no algoritmo
existente sem alterá-lo: um novo tipo `SourceSelection` restringe, antes de
chamar `Route.Coverage`/`SelectSource`, a lista de candidatos de um tipo a
um único elemento — o mesmo algoritmo de sempre, sobre uma lista de um,
resolve sozinho tanto o uso exclusivo (nunca mistura) quanto a recusa por
cobertura incompleta. O reaproveitamento de um recorte guardado por `fly
--keep` passa a exigir também que a procedência que o recorte já registra
(`Summary.Sources`, existente desde a etapa 4) bata com a fonte pedida
agora — sem nenhum registro novo à parte, e sem mudar o formato do arquivo
do recorte.

## Contexto Técnico

**Linguagem/Versão**: Go 1.26.4 (`go.mod`), sem mudança.

**Dependências Principais**: nenhuma nova — reutiliza `spf13/cobra` (CLI) e
a persistência já existente do registro (`internal/infra/outbound/jsonfile`).

**Armazenamento**: o registro local em JSON (`~/.sobrevoo/registry.json`,
`jsonfile.GeoDataRepository`), sem mudança de tecnologia nem de formato —
só um método novo (`Clear`) sobre o arquivo já existente. O formato do
recorte exportado (`slice-file.md`) também não muda: a procedência que
`fly --keep` passa a conferir (`Summary.Sources`) já existe desde a etapa
4.

**Testes**: `go test ./... -cover`, `testify` + `uber-go/mock`, sem mudança
de ferramenta.

**Plataforma-Alvo**: CLI de linha de comando, macOS/Linux — sem mudança.

**Tipo de Projeto**: CLI de projeto único (`cmd/sobrevoo`), sem mudança de
estrutura.

**Metas de Desempenho**: `SourceSelection.Resolve` é uma função pura sobre
listas já em memória (sem I/O) — o custo de validar uma escolha explícita é
desprezível frente à leitura de peças de mapa/amostras de elevação que
`geodata slice`/`fly` já fazem; nenhuma meta numérica nova.

**Restrições**: recusar cedo (FR-006) — a validação do nome pedido
acontece antes de qualquer leitura de conteúdo geográfico, e em `geodata
check`, antes até do tratamento do trajeto; `--confirm` nunca é um prompt
interativo (mesma convenção de `--overwrite`); `geodata clear` nunca toca
um arquivo de dado geográfico (FR-002); offline (Princípio V) — sem mudança,
nenhuma dependência nova.

**Escala/Escopo**: mesma escala das etapas 2 e 4 (um registro de até
algumas dezenas de fontes, cada checagem/recorte sobre um único trajeto);
esta etapa não introduz nenhum novo limite.

## Verificação da Constituição

*PORTÃO: Deve passar antes da Fase 0 de pesquisa. Reverificar após o design da Fase 1.*

| Princípio | Verificação |
|---|---|
| I. Arquitetura Hexagonal | `SourceSelection.Resolve` e `GeoSlice.EnsureUsesSources` (antes `EnsureUsesSelection` — `research.md` item 3, "Revisão") são código de domínio puro (`internal/domain`), sem nenhuma biblioteca de infraestrutura — mesma fronteira de sempre. `geodata clear` só acrescenta um método (`Clear`) ao adapter `jsonfile` já existente. |
| II. Portas para Toda Dependência Externa | Um método novo numa porta já existente (`GeoDataRepository.Clear`) — nenhuma porta nova. `SourceSelection.Resolve` não faz I/O (opera sobre listas já buscadas pelo chamador), então não precisa de porta nenhuma. |
| III. Entrypoints Descartáveis | A CLI só traduz três flags novas (`--confirm`, `--base-map`, `--elevation`) em chamadas de `GeoDataService`/`GeoSliceService`, e erros sentinela em códigos de saída — nenhuma regra de negócio na CLI; `parseSourceSelection` não valida nada (só monta o DTO), a validação real fica inteiramente em `SourceSelection.Resolve`. |
| IV. Neutralidade Geográfica | Sem mudança: nenhum dado geográfico fixo é introduzido; a escolha de fonte é sempre pelo nome que o próprio usuário deu ao registrar. |
| V. Funcionamento Offline | Sem mudança: nenhuma dependência de rede ou serviço externo novo. |
| VI. Testes Automatizados no Núcleo | `SourceSelection`, `GeoSlice.EnsureUsesSources` e o método `Clear` de `GeoDataService` são testados com as portas mockadas (`mockdomain`), sem tocar disco; a CLI continua mockando `application.GeoDataService`/`GeoSliceService`/`FlightService` (`mockapplication`). |
| VII. Erros Sentinela no Domínio | Três sentinelas novos em `errors.go` — `ErrDataSourceTypeMismatch`, `ErrRegistryClearNotConfirmed`, `ErrSliceUsesDifferentSource` —, ao lado dos já existentes; `ErrDataSourceNotRegistered` (já existente) é reaproveitado para um nome pedido que não existe. |
| VIII. Configuração Injetada | Nenhuma configuração nova: nem `--confirm` nem `--base-map`/`--elevation` têm um padrão configurável (a ausência já é o padrão — seleção automática, sem confirmação); `config.go`/`config_mapping.go` não mudam. |
| IX. Portas/Service Layer/Regra de Negócio | Nenhum serviço novo: `GeoDataService` ganha um método (`Clear`) e um parâmetro (`CheckCoverage`); `GeoSliceService.Generate` ganha um parâmetro (e, na revisão do `research.md` item 3, `GeoSliceService` ganha `Sources`); `FlightService.reuseSlice` ganha uma condição. A regra de negócio (resolver a seleção contra os candidatos, verificar a procedência do recorte guardado) vive inteiramente em `internal/domain`, como método de `SourceSelection`/`GeoSlice` — os serviços só orquestram (buscam os candidatos via porta, delegam a regra, devolvem o resultado), exatamente como `GeoDataService.CheckCoverage` já faz hoje com `Route.Coverage`. |
| X. Testes: Given/When/Then, Builders, Isolamento | Sem mudança de convenção; `builddomain` ganha um builder para `SourceSelection` se os testes pedirem (provavelmente não — é um struct de dois campos opcionais, sem literal repetido o bastante para justificar um). |

Nenhuma violação; nada a registrar em Rastreamento de Complexidade.

## Estrutura do Projeto

### Documentação (desta funcionalidade)

```text
specs/010-geo-data-source-control/
├── plan.md              # Este arquivo (saída do comando /speckit-plan)
├── research.md          # Saída da Fase 0 (comando /speckit-plan)
├── data-model.md         # Saída da Fase 1 (comando /speckit-plan)
├── quickstart.md        # Saída da Fase 1 (comando /speckit-plan)
├── contracts/            # Saída da Fase 1 (comando /speckit-plan)
│   ├── registry-clear.md
│   └── source-selection-flags.md
└── tasks.md              # Saída da Fase 2 (comando /speckit-tasks — NÃO criado pelo /speckit-plan)
```

### Código-Fonte (raiz do repositório)

Projeto único já existente (hexagonal), sem mudança de layout — só arquivos
novos e extensões pontuais dentro da árvore já estabelecida:

```text
internal/domain/
├── source_selection.go          # NOVO: SourceSelection, Resolve
├── geo_data_source.go            # GeoDataRepository: +Clear() error
├── geo_slice.go                  # +GeoSlice.EnsureUsesSources, +SliceRegions.Sources
├── flight.go                     # FlightRequest: +Selection
└── errors.go                     # +ErrDataSourceTypeMismatch, +ErrRegistryClearNotConfirmed,
                                   #  +ErrSliceUsesDifferentSource

internal/application/
├── geo_data_service.go           # +Clear(confirmed bool); CheckCoverage: +selection
├── geo_slice_service.go          # Generate: +selection; +Sources (revisão)
└── flight_service.go             # reuseSlice: +selection, checa GeoSliceService.Sources + EnsureUsesSources também

internal/infra/outbound/jsonfile/
└── geo_data_repository.go        # +Clear() error (mesma escrita atômica de Save/Delete)

internal/infra/inbound/cli/
├── source_selection.go           # NOVO: parseSourceSelection (mesmo padrão de appearance.go/overlay.go)
├── geodata_clear.go               # NOVO: NewGeoDataClearCommand
├── geodata_check.go               # +flags --base-map/--elevation
├── geodata_slice.go               # idem
├── fly.go                         # idem
└── exit_code.go                  # +códigos 56, 57, 58

cmd/sobrevoo/
└── main.go                       # registra geodata clear; passa as duas novas flags aos 3 comandos

internal/domain/mockdomain/, internal/application/mockapplication/
└── geo_data_service.go, geo_slice_service.go, flight_service.go  # regenerados (make generate) —
    assinaturas mudam, nenhuma porta/interface nova

internal/domain/builddomain/
└── flight_request_builder.go     # +WithSelection, se os testes pedirem
```

**Decisão de Estrutura**: nenhuma pasta nova de alto nível — tudo estende
`internal/domain`, `internal/application`, `internal/infra/{outbound/
jsonfile,inbound/cli}` já existentes, seguindo exatamente o padrão das
etapas 2, 4 e 7 (mesmas árvores, mesmos pontos de extensão).
