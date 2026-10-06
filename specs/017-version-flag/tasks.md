---

description: "Task list for feature implementation"
---

# Tarefas: Informação de Versão da CLI

**Entrada**: Documentos de design de `/specs/017-version-flag/`

**Pré-requisitos**: `plan.md`, `spec.md`, `research.md`, `data-model.md`,
`contracts/version-flag.md`, `quickstart.md`

**Testes**: incluídos — a própria `spec.md` pede explicitamente os três
cenários de `pickVersion` ("Testes seguem a disciplina de sempre..."). A
constituição (Princípio X) exige given/when/then (`// given`/`// when`/
`// then`) e proíbe tabela: cada cenário é seu próprio
`t.Run("should ...")`. Nenhum mock é gerado nesta etapa — nenhuma porta
nova, nenhum método novo em nenhuma interface existente.

**Organização**: as tarefas são agrupadas pelas quatro histórias de
usuário (P1–P4) de `spec.md`. A **Phase 2 (Foundational)** carrega a
lógica de resolução em si (`cmd/sobrevoo/version.go`: `pickVersion`,
`readModuleVersion`, `resolveVersion`) com os três cenários de precedência
já testados ali, como função pura — sem isso nenhuma história tem o que
mostrar, e o próprio FR-006 (nunca mentir uma versão) exige que o
*fallback* já exista desde o primeiro binário que alguém roda, mesmo que
o objetivo central da etapa (US1) seja só o caminho mais comum (versão de
módulo). A **Phase 3 (US1)** é quem liga esse resultado à flag `--version`
de verdade (`root.go` + `main.go`) — só depois dela existe um binário real
para validar manualmente qualquer coisa. US2, US3 e US4 não acrescentam
nenhum código de produção novo: suas três histórias já saem verdadeiras da
Phase 2 (a função pura) e da Phase 3 (o binário real); o que falta a cada
uma é só a prova — um teste que não existia, ou uma validação manual.

**Divisão do trabalho entre histórias**:

| História | O que entrega | Veículo | Código de produção novo? |
|---|---|---|---|
| US1 (P1) 🎯 MVP | `sobrevoo --version` existe, mostra a tag instalada, uma linha, código 0 | `root.go` (`Version`/`VersionTemplate`) + `main.go` (wiring) | Sim |
| US2 (P2) | a linha é limpa o bastante para colar num relato | — já garantido pelo teste de US1 | Não |
| US3 (P3) | build local sem tag nunca inventa uma versão de release | — já garantido pela Phase 2 | Não |
| US4 (P4) | uma versão fixada no build vence a do módulo | — já garantido pela Phase 2 | Não |

## Formato: `[ID] [P?] [Story] Descrição`

- **[P]**: pode ser executado em paralelo (arquivos diferentes, sem
  dependência de tarefa incompleta).
- **[Story]**: a qual história de usuário esta tarefa pertence (US1–US4).
  Tarefas de Setup, Foundational e Polish não têm esse rótulo.
- Toda tarefa inclui o caminho de arquivo exato a criar/editar.
- Comentários de código, identificadores, mensagens de commit, flags,
  saída e mensagens de erro em tempo de execução: **inglês**; artefatos
  do Spec Kit: português (constituição, "Idioma dos Artefatos").

## Convenções de Caminho

Projeto único em Go, mesma estrutura hexagonal de `plan.md`: só
`cmd/sobrevoo/` (composition root) e `internal/infra/inbound/cli/`. Nenhum
arquivo em `internal/domain`, `internal/application` ou
`internal/infra/outbound` é tocado — a especificação exige isso
explicitamente (a versão "não entra no domínio").

---

## Phase 1: Setup (Shared Infrastructure)

**Propósito**: um ponto de partida próprio (branch) e verde.

- [X] T001 Confirmar que o branch `017-version-flag` está ativo (já criado por `/speckit-specify`, se o hook de git estiver configurado; caso contrário, `git checkout -b 017-version-flag`)
- [X] T002 Confirmar `make build`, `make test` e `make lint` verdes antes de qualquer mudança (linha de base)

**Checkpoint**: repositório pronto para a Phase 2.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Propósito**: a lógica de resolução da versão em si — qual das três
fontes prevalece (`research.md` itens 3–5) — como função pura, sem nenhuma
dependência do Cobra nem de um binário real. **Nenhuma história de usuário
pode ser validada de ponta a ponta antes desta fase estar completa.**

- [X] T003 [P] Criar `cmd/sobrevoo/version_test.go` (`package main`), `Test_pickVersion`, com os três cenários que `spec.md` pede, cada um seu próprio `t.Run` com `// given`/`// when`/`// then`:
  - "should return the module version when no build-time version is set" — `pickVersion("", "v0.1.0")` deve ser `"v0.1.0"` (FR-005)
  - "should return the development version when neither a build-time version nor a module version is set" — `pickVersion("", "")` deve ser `"(devel)"` (FR-006)
  - "should prefer the build-time version over the module version when both are set" — `pickVersion("v0.2.0", "v0.1.0")` deve ser `"v0.2.0"` (FR-007, FR-008)

  Deve falhar ao compilar/rodar até T004.
- [X] T004 Criar `cmd/sobrevoo/version.go` (`package main`):
  - `var version string` — vazio por padrão; fixado em tempo de build via `-ldflags "-X main.version=vX.Y.Z"` para compilações de release fora de `go install` (FR-007); comentário citando isso
  - `const devVersion = "(devel)"` — o mesmo literal que `debug.ReadBuildInfo` já devolve para um build sem tag (`research.md` item 4)
  - `func pickVersion(buildOverride, moduleVersion string) string` — `buildOverride` se não vazio (FR-008); senão `moduleVersion` se não vazio (FR-005); senão `devVersion` (FR-006)
  - `func readModuleVersion() string` — chama `debug.ReadBuildInfo()` (pacote `runtime/debug`); devolve `""` quando `!ok`, senão `info.Main.Version`
  - `func resolveVersion() string { return pickVersion(version, readModuleVersion()) }`
  - `make test ./cmd/sobrevoo/...` verde (depende de T003)

**Checkpoint**: `pickVersion`/`resolveVersion` corretos e testados — mas nenhum comando da CLI ainda expõe isso.

---

## Phase 3: User Story 1 — Conferir a versão instalada (Priority: P1) 🎯 MVP

**Objetivo**: `sobrevoo --version` existe no comando raiz, imprime uma
única linha (nome do programa + versão, nessa ordem, nada mais) e sai com
código `0`, sem tocar em arquivo, registro ou rede; instalado a partir de
uma tag (`go install .../sobrevoo@vX.Y.Z`), mostra exatamente essa tag.

**Teste Independente**: `quickstart.md` itens 2, 4 e 5.

- [X] T005 [P] [US1] Estender `internal/infra/inbound/cli/root_test.go`:
  - atualizar as duas chamadas já existentes de `cli.NewRootCommand()` para `cli.NewRootCommand("v0.0.0-test")` (a assinatura muda em T006 — sem isso o pacote não compila)
  - novo cenário `t.Run("should print the program name and version, nothing else, and exit 0 when run with --version", ...)`: `root := cli.NewRootCommand("v0.1.0")`; `root.SetArgs([]string{"--version"})`; `err := root.Execute()`; `require.NoError(t, err)`; `assert.Equal(t, "sobrevoo v0.1.0\n", out.String())`

  Deve falhar até T006.
- [X] T006 [US1] Editar `internal/infra/inbound/cli/root.go`: `NewRootCommand` passa a receber `version string`; `root.Version = version`; `root.SetVersionTemplate("{{.DisplayName}} {{.Version}}\n")` (o template padrão do Cobra insere a palavra "version" no meio, o que `FR-002` proíbe — `research.md` item 2); comentário citando que o Cobra já trata `--version` antes de qualquer outro código do comando, garantindo FR-003 de graça; `make test` verde (depende de T005)
- [X] T007 [US1] Editar `cmd/sobrevoo/main.go`: trocar `cli.NewRootCommand()` por `cli.NewRootCommand(resolveVersion())`, logo no início de `run()`, antes da montagem dos demais serviços (não depende de `cfg`); `make build` verde (depende de T004, T006)
- [X] T008 [US1] Validação manual: `quickstart.md` itens 2, 4 e 5 confirmados com o binário real — uma linha só (`sobrevoo v0.0.0-20261006015100-c169b2b38ac9+dirty`), código de saída `0`, nenhum `~/.sobrevoo` criado; depois do merge e da publicação da tag `v0.1.0`, `go install github.com/waliqueiroz/sobrevoo/cmd/sobrevoo@v0.1.0` seguido de `sobrevoo --version` reportou exatamente `sobrevoo v0.1.0`

**Checkpoint**: MVP entregue — qualquer usuário consegue conferir a versão instalada.

---

## Phase 4: User Story 2 — Relatar um problema com a versão certa (Priority: P2)

**Objetivo**: a saída de `--version` é limpa o bastante (uma linha, sem
banner nem texto decorativo) para ser colada direto num relato de
problema, sem edição.

**Teste Independente**: `quickstart.md` item 2 (o mesmo da US1, reafirmado
do ponto de vista de "colar sem editar").

- [X] T009 [US2] Comprovar, sem nenhuma tarefa de código nova: o cenário de T005 já confere que a saída inteira é exatamente `"sobrevoo v0.1.0\n"` — uma linha, sem nenhum caractere além do nome e da versão —, o que já é, por construção, uma string pronta para colar num relato sem edição. Nenhum teste nem código adicional é necessário; validação manual: `quickstart.md` item 2 (confirmado em T008).

**Checkpoint**: nada novo a construir — a história já estava satisfeita desde o MVP.

---

## Phase 5: User Story 3 — Saber que é uma compilação de desenvolvimento (Priority: P3)

**Objetivo**: um binário compilado localmente, sem tag nem versão fixada
no build, informa honestamente `(devel)` — nunca um número de versão de
release inventado.

**Teste Independente**: `quickstart.md` item 1.

- [X] T010 [US3] Comprovar, sem nenhuma tarefa de código nova: o cenário "should return the development version..." de T003 já prova a lógica; com US1 (Phase 3) completa, o comportamento existe de ponta a ponta num binário real. Validação manual (achado real, documentado em `research.md`/`quickstart.md` atualizados): com o VCS stamping padrão do Go 1.26 (`make build`, dentro do checkout git), o resultado é uma pseudo-versão (`sobrevoo v0.0.0-20261006015100-c169b2b38ac9+dirty`) — não o literal `(devel)`, mas igualmente honesto (não é uma tag real); com `-buildvcs=false` (sem stamping), o resultado é exatamente `sobrevoo (devel)`. Ambos os casos nunca inventam um número de release.

**Checkpoint**: nada novo a construir — confirmado com um binário real.

---

## Phase 6: User Story 4 — Fixar a versão explicitamente numa compilação de release (Priority: P4)

**Objetivo**: uma versão fixada no momento do build (`-ldflags "-X
main.version=..."`) prevalece sobre a versão do módulo, mesmo quando as
duas estão presentes no mesmo binário.

**Teste Independente**: `quickstart.md` item 3.

- [X] T011 [US4] Comprovar, sem nenhuma tarefa de código nova: o cenário "should prefer the build-time version over the module version..." de T003 já prova a precedência; com US1 (Phase 3) completa, o comportamento existe de ponta a ponta num binário real. Validação manual confirmada: `go build -ldflags "-X main.version=v9.9.9-quickstart" -o /tmp/sobrevoo-release ./cmd/sobrevoo`; `/tmp/sobrevoo-release --version` imprimiu exatamente `sobrevoo v9.9.9-quickstart` — a versão fixada no build venceu a versão de módulo (que seria a pseudo-versão do item anterior).

**Checkpoint**: todas as quatro histórias de usuário validadas.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Propósito**: documentação e verificação final.

- [X] T012 [P] Atualizar `README.md`, seção "Instalação": acrescentar `sobrevoo --version` como a forma de confirmar o que foi efetivamente instalado, logo depois do bloco `go install github.com/waliqueiroz/sobrevoo/cmd/sobrevoo@latest` — no mesmo tom didático e autocontido do resto do arquivo, sem citar `specs/` (FR-010)
- [X] T013 [P] Atualizar `CLAUDE.md`: acrescentar a décima sétima etapa ao resumo do projeto (mesmo estilo das demais) e uma seção dedicada `### A informação de versão da CLI (etapa 17)`, citando a flag `--version` no comando raiz, as três fontes de versão e sua precedência, e que nada disso entra no domínio
- [X] T014 Validação final: `make build`, `make test`, `make lint` verdes; `make generate` confirmado sem nenhuma diferença (nenhuma interface mudou); `gofmt -l .` limpo; confirmado que `plan --version` continua recusando como flag desconhecida (código `2`, FR-009, item 6 do `quickstart.md`); `quickstart.md` reexecutado por inteiro (itens 1–4, 6–7 confirmados com o binário real; item 5 pendente de uma tag publicada — ver nota em T008)

---

## Dependências e Ordem de Execução

### Dependências entre Fases

- **Phase 1 (Setup)**: sem dependências.
- **Phase 2 (Foundational)**: depende da Phase 1; bloqueia toda validação de ponta a ponta das quatro histórias (a lógica pura de precedência precisa existir e estar testada antes de qualquer coisa).
- **Phase 3 (US1)**: depende da Phase 2 completa — é quem liga a lógica pura a um binário real.
- **Phase 4/5/6 (US2/US3/US4)**: ao contrário do padrão usual deste projeto (onde cada história só depende da Foundational), aqui as três dependem também da **Phase 3 (US1)** completa — nenhuma delas acrescenta código de produção, e a validação manual que cada uma pede só é possível sobre um binário real, que só existe depois do `root.go`/`main.go` da US1 estarem prontos. Podem ser feitas em qualquer ordem entre si depois disso.
- **Phase 7 (Polish)**: depende das quatro histórias completas.

### Dentro de Cada História de Usuário

- Testes são escritos (ou estendidos) e devem falhar antes da tarefa de implementação correspondente (T003 antes de T004; T005 antes de T006).
- Domínio/composition root (`version.go`) antes de infraestrutura de entrada (`root.go`); infraestrutura de entrada antes do composition root final (`main.go`).

### Oportunidades de Paralelização

- T003 (Foundational) e T005 (US1) editam arquivos diferentes e usam a mesma assinatura final de `pickVersion`/`NewRootCommand` já decidida no plano — podem ser escritas em paralelo, mas T006 só fica verde depois de T004 existir (a implementação real), então a ordem de execução das implementações é T004 → T006 → T007.
- T012 e T013 (Polish, arquivos diferentes) podem ser feitas em paralelo.

---

## Estratégia de Implementação

### MVP Primeiro (Somente História de Usuário 1)

1. Completar Phase 1: Setup.
2. Completar Phase 2: Foundational (CRÍTICO — bloqueia a validação de qualquer história).
3. Completar Phase 3: História de Usuário 1.
4. **PARAR E VALIDAR**: `quickstart.md` itens 2, 4 e 5.
5. Implantar/demonstrar se estiver pronta — `sobrevoo --version` já funciona por completo neste ponto, incluindo o caso de desenvolvimento e a precedência do build, mesmo que US2–US4 ainda não tenham sido formalmente "provadas".

### Entrega Incremental

1. Setup + Foundational → a lógica de precedência existe e está testada, nada visível ainda.
2. US1 (MVP) → a flag funciona de ponta a ponta → validar → demonstrar.
3. US2, US3, US4 → cada uma só confirma, com uma validação manual própria, um ângulo que o MVP já cobria — a parte mais barata da etapa inteira, pelo mesmo motivo que histórias de prioridade mais baixa já foram as mais baratas em etapas anteriores (010, 013, 014).
4. Polish.
