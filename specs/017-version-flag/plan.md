# Plano de Implementação: Informação de Versão da CLI

**Branch**: `017-version-flag` | **Data**: 2026-10-05 | **Especificação**: [spec.md](./spec.md)

**Entrada**: Especificação de funcionalidade de `/specs/017-version-flag/spec.md`

**Nota**: Este template é preenchido pelo comando `/speckit-plan`; sua definição descreve o fluxo de execução.

## Resumo

`sobrevoo --version` imprime, numa única linha, o nome do programa e a sua
versão, e sai com código `0` — sem tocar em arquivo, registro ou rede. A
versão vem do mecanismo de `--version` já embutido no Cobra
(`Command.Version` + um `VersionTemplate` próprio, sem a palavra "version"
que o template padrão insere), armazenado no comando raiz. Quem resolve
*qual* string de versão usar é o ponto de entrada descartável
(`cmd/sobrevoo`): um valor fixado no build via `-ldflags
"-X main.version=..."` tem precedência; na ausência dele, `runtime/debug`
(biblioteca padrão, sem porta — mesma isenção do Princípio II já aplicada a
`time.Now()`) lê a versão do módulo que o próprio `go install
.../sobrevoo@vX.Y.Z` grava no binário; sem nenhum dos dois, a ferramenta
informa honestamente uma compilação de desenvolvimento (`(devel)`, o mesmo
literal que o Go já usa para isso) em vez de inventar um número de release.
Nenhum pacote de domínio ou de application é tocado — é puramente
`cmd/sobrevoo` (resolução) e `internal/infra/inbound/cli` (exposição da
flag no comando raiz).

## Contexto Técnico

**Linguagem/Versão**: Go 1.26.4 (já fixado em `go.mod`)

**Dependências Principais**: `github.com/spf13/cobra` (já usado por toda a
CLI; fornece o mecanismo de `--version` reaproveitado aqui) e
`runtime/debug` da biblioteca padrão (`debug.ReadBuildInfo`) — nenhuma
dependência nova.

**Armazenamento**: N/A — nenhum arquivo é lido ou escrito por esta
funcionalidade.

**Testes**: `go test` com `testify` (`assert`/`require`), seguindo o
Princípio X (given/when/then, sem tabela); nenhum mock novo, porque nenhuma
porta nova é criada.

**Plataforma-Alvo**: a mesma CLI multiplataforma (macOS/Linux/Windows) de
sempre — `debug.ReadBuildInfo` e `-ldflags -X` são suportados pelo
toolchain do Go em qualquer uma delas.

**Tipo de Projeto**: CLI (projeto único, `cmd/sobrevoo` +
`internal/infra/inbound/cli`) — sem nenhum componente de domínio ou de
application.

**Metas de Desempenho**: resposta imediata (sub-segundo); nenhuma meta
específica além disso, dado que não há I/O envolvido.

**Restrições**: `--version` NUNCA DEVE ler/escrever arquivo, consultar o
registro de dados geográficos, nem acessar rede (FR-003); a saída DEVE ser
exatamente uma linha, sem texto além do nome do programa e da versão
(FR-002).

**Escala/Escopo**: uma flag no comando raiz, dois arquivos novos pequenos
(`cmd/sobrevoo/version.go` + teste) e uma assinatura de construtor alterada
(`cli.NewRootCommand`); nenhum outro comando muda.

## Verificação da Constituição

*PORTÃO: Deve passar antes da Fase 0 de pesquisa. Reverificar após o design da Fase 1.*

| Princípio | Avaliação |
|---|---|
| I. Arquitetura Hexagonal (Núcleo Isolado) | **Passa.** Nem `internal/domain` nem `internal/application` são tocados — a versão nunca chega ao núcleo; ela nasce e termina entre `cmd/sobrevoo` e `internal/infra/inbound/cli`. |
| II. Portas para Toda Dependência Externa | **Passa, com a isenção já prevista no princípio.** `runtime/debug.ReadBuildInfo()` não lê arquivo, rede, nem processo externo — lê metadados que o linker do Go já embutiu no próprio binário em tempo de build, a mesma categoria de "função pura da biblioteca padrão" que o princípio já isenta para `time.Now()`. Nenhuma porta nova é criada; a lógica de decisão em si (qual das três fontes prevalece) é extraída numa função pura (`pickVersion`) só para ficar testável sem depender do binário de teste ter ou não metadados de módulo — não é uma porta, é só separar I/O de decisão dentro do mesmo arquivo descartável. |
| III. Entrypoints Descartáveis | **Passa.** Nenhuma regra de negócio é introduzida (a especificação é explícita: "não entra no domínio"); a CLI permanece um adapter fino que só expõe o que `cmd/sobrevoo` já resolveu. |
| IV. Neutralidade Geográfica | **N/A.** Nenhum dado geográfico envolvido. |
| V. Funcionamento Offline | **Passa.** `--version` não acessa rede (FR-003) — é, inclusive, mais restrito que o resto da CLI, que já não depende de rede para operar. |
| VI. Testes Automatizados no Núcleo | **N/A.** Nenhum código de núcleo é adicionado; os testes desta etapa vivem em `cmd/sobrevoo` e `internal/infra/inbound/cli`, não em `internal/domain`/`internal/application`. |
| VII. Erros Sentinela no Domínio | **N/A.** `--version` não tem caminho de erro — sempre sai com código `0` (FR-002); nenhum sentinela novo, nenhum código de saída novo. |
| VIII. Configuração Injetada | **Passa, por analogia.** Embora a versão nunca entre no núcleo (diferente da configuração que o princípio descreve), o padrão é o mesmo: o ponto de entrada descartável resolve o valor e o entrega pronto (uma `string`) ao comando raiz, que nunca vai buscá-lo sozinho — o mesmo papel que `cmd/sobrevoo` já exerce para `cfg`. |
| IX. Organização de Portas, Service Layer, Localização da Regra de Negócio e Mocks | **Passa.** Nenhuma porta, nenhum serviço novo; `cli.NewRootCommand` ganha um parâmetro (`version string`) do mesmo jeito que outros construtores de comando já recebem valores resolvidos pelo composition root (ex.: `defaultResolution`, `defaultAppearance`). |
| X. Testes: Given/When/Then, Builders e Isolamento por Camada | **Passa.** Cada cenário é seu próprio `t.Run("should ...")`; nenhum builder é necessário (não há entidade/DTO complexo); a função pura `pickVersion` é testada diretamente, sem mock, porque não há porta para mockar. |

Nenhuma violação — sem necessidade da seção de Rastreamento de Complexidade.

## Estrutura do Projeto

### Documentação (desta funcionalidade)

```text
specs/017-version-flag/
├── plan.md              # Este arquivo (saída do comando /speckit-plan)
├── research.md          # Saída da Fase 0 (comando /speckit-plan)
├── data-model.md        # Saída da Fase 1 (comando /speckit-plan)
├── quickstart.md        # Saída da Fase 1 (comando /speckit-plan)
├── contracts/           # Saída da Fase 1 (comando /speckit-plan)
│   └── version-flag.md
└── tasks.md             # Saída da Fase 2 (comando /speckit-tasks - NÃO criado pelo /speckit-plan)
```

### Código-Fonte (raiz do repositório)

Projeto único, já existente (Go, hexagonal) — nenhuma estrutura nova,
só os arquivos que esta etapa acrescenta ou edita dentro dela:

```text
cmd/sobrevoo/
├── main.go                       # editado: chama resolveVersion() e passa para cli.NewRootCommand
├── version.go                    # novo: resolveVersion, readModuleVersion, pickVersion, var version (ldflags)
└── version_test.go               # novo: os três cenários de pickVersion

internal/infra/inbound/cli/
├── root.go                       # editado: NewRootCommand(version string) seta Version + VersionTemplate
└── root_test.go                  # editado: cenário novo cobrindo --version
```

Nenhum arquivo em `internal/domain`, `internal/application`, ou
`internal/infra/outbound` é criado ou alterado.

**Decisão de Estrutura**: a versão é resolvida inteiramente no composition
root (`cmd/sobrevoo`), no mesmo arquivo descartável que já resolve `cfg` e
monta os serviços — um arquivo próprio (`version.go`) em vez de inflar
`main.go`, pelo mesmo motivo que `config_mapping.go` já é separado de
`main.go`. `internal/infra/inbound/cli` só recebe a string pronta e decide
como apresentá-la (a flag, o template de saída) — o mesmo papel que já
exerce para `Resolution`/`Appearance`/`OverlayConfig` resolvidos por
`cmd/sobrevoo/config_mapping.go`.

## Rastreamento de Complexidade

*Sem violações a justificar.*
