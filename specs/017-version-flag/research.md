# Pesquisa: Informação de Versão da CLI

**Feature**: `017-version-flag` | **Data**: 2026-10-05

Nenhum `NEEDS CLARIFICATION` ficou aberto no Contexto Técnico do `plan.md`
— esta pesquisa documenta as decisões de design que a especificação deixou
explicitamente para o planejamento técnico (ver Suposições de `spec.md`).

## 1. Como expor `--version` no comando raiz

**Decisão**: usar o mecanismo de versão já embutido no Cobra —
`root.Version = <versão resolvida>` — em vez de implementar o parsing da
flag à mão.

**Motivo**: o Cobra, já em uso em toda a CLI do Sobrevoo, registra
automaticamente uma flag `--version` booleana no comando quando `Version`
não é vazio (`Command.InitDefaultVersionFlag`), e a trata **antes** de
checar se o comando é executável (`Command.execute`, em
`github.com/spf13/cobra@v1.10.2/command.go`): quando a flag é passada, o
Cobra imprime o template de versão em `OutOrStdout()` e retorna, sem rodar
nenhum `RunE`, sem tocar em nenhum outro código do comando — já satisfaz
FR-001 (nenhum argumento adicional exigido) e FR-003 (nenhum I/O) só por
usar o mecanismo pronto.

**Alternativas consideradas**:
- Um `PersistentPreRunE` que verifica `--version` manualmente antes de
  qualquer subcomando — rejeitada: duplica uma lógica que o próprio Cobra
  já implementa corretamente (incluindo a ordem de precedência com
  `--help`), com mais código para manter e testar, sem nenhum ganho.
- Um subcomando `version` separado — explicitamente fora de escopo
  (`spec.md`, Fora de escopo).

## 2. Formato exato da linha impressa

**Decisão**: um `VersionTemplate` próprio, `"{{.DisplayName}} {{.Version}}\n"`,
em vez do template padrão do Cobra.

**Motivo**: o template padrão do Cobra
(`defaultVersionTemplate = "{{with .DisplayName}}{{printf "%s " .}}{{end}}{{printf "version %s" .Version}}\n"`)
produz `"sobrevoo version v0.1.0\n"` — a palavra `version` no meio não é um
banner nem arte, mas é texto além do nome do programa e da versão, o que o
FR-002 proíbe ("nada de banner, arte ou texto extra"). O template próprio
produz exatamente `"sobrevoo v0.1.0\n"`: dois tokens, nome e versão, nessa
ordem, fáceis de extrair com `cut -d' ' -f2` ou `awk '{print $2}'` num
script (SC-005). `DisplayName()` (não `Use` direto) é o método que o
próprio Cobra usa para nomear o programa — já lida com qualquer override de
nome de exibição, se um dia existir.

**Alternativas consideradas**:
- Deixar o template padrão do Cobra (`"sobrevoo version v0.1.0"`) —
  rejeitada: viola literalmente "nada ... além disso" do FR-002, mesmo
  sendo um formato comum em outras CLIs.
- Formato `"v0.1.0"` sozinho (só a versão) — rejeitada: FR-002 exige o
  nome do programa *e* a versão, nessa ordem; só a versão não identifica a
  ferramenta quando a linha é colada isolada num relato (História 2).

## 3. De onde vem o valor da versão, e sua precedência

**Decisão**: três fontes, nesta ordem de precedência:

1. Um `var version string` em `cmd/sobrevoo`, vazio por padrão, fixável no
   momento do build com `-ldflags "-X main.version=vX.Y.Z"` (FR-007) —
   quando não vazio, prevalece sobre qualquer outra fonte (FR-008).
2. `debug.ReadBuildInfo().Main.Version` (pacote `runtime/debug` da
   biblioteca padrão) — o Go já grava aqui a tag do módulo quando o binário
   foi obtido via `go install .../sobrevoo@vX.Y.Z` (FR-005), e grava o
   literal `"(devel)"` quando o binário foi compilado a partir da árvore de
   trabalho sem nenhuma tag envolvida.
3. Na ausência de ambas (por exemplo, um binário compilado sem suporte a
   módulos, caso em que `ReadBuildInfo` devolve `ok == false`), um literal
   fixo de compilação de desenvolvimento no próprio código
   (`devVersion = "(devel)"`) — o mesmo valor que a fonte 2 já produziria no
   caso comum, mantendo a saída consistente nos dois casos (FR-006).

**Motivo**: é exatamente o mecanismo que a especificação já descreve
("a biblioteca padrão sabe ler" a versão do módulo) e que `go install`
preenche de graça, sem exigir nenhuma flag de build para o caminho mais
comum de instalação — e o `var version` dá à publicação de uma release fora
de `go install` (binário pré-compilado anexado a uma release do GitHub,
por exemplo) um jeito de fixar a versão sem depender do módulo (FR-007).

**Alternativas consideradas**:
- Um arquivo `VERSION` lido em tempo de execução — rejeitada: reintroduz,
  como arquivo em vez de constante no código, exatamente o problema que o
  FR-004 proíbe (alguém precisa lembrar de atualizá-lo a cada release), e
  acrescenta uma leitura de arquivo que o FR-003 proíbe.
- Depender só do `var version` fixado por `-ldflags`, sem
  `debug.ReadBuildInfo` — rejeitada: um `go install
  .../sobrevoo@vX.Y.Z` comum (o caminho de instalação que a própria
  especificação usa como exemplo) nunca passa `-ldflags`, então a versão
  ficaria sempre vazia/development para a maioria dos usuários — viola
  FR-005.
- Depender só de `debug.ReadBuildInfo`, sem o `var version` — rejeitada:
  não deixaria nenhum caminho para fixar a versão numa compilação de
  release fora de `go install` (FR-007), que é justamente o caso que
  `debug.ReadBuildInfo` não resolve (um binário pré-compilado publicado
  como asset de release não carrega uma tag de módulo instalada por
  `go install`).

## 4. Valor exato do indicador de "compilação de desenvolvimento"

**Decisão**: reaproveitar o literal `"(devel)"` — o mesmo texto que
`debug.ReadBuildInfo` devolve quando não há nenhuma informação de build
disponível (fonte 3 da decisão 3) — como a constante usada quando nem o
build nem o módulo têm uma versão.

**Motivo**: é o vocabulário que a própria toolchain do Go já usa para essa
situação (visível também em `go version -m` sobre um binário sem suporte a
módulo, ou compilado com `-buildvcs=false`); usar o mesmo texto evita um
terceiro vocabulário para "não é uma release" além dos dois que o próprio
Go já produz, e satisfaz FR-006 sem nenhum caso especial.

**Correção depois da implementação (achado real, não previsto aqui)**: um
`go build`/`make build` comum, dentro do checkout git deste projeto, com o
Go 1.26 (e qualquer Go ≥ 1.18, que já faz "VCS stamping" por padrão), **não**
devolve o literal `"(devel)"` — devolve uma pseudo-versão derivada do commit
atual, no formato `v0.0.0-<timestamp>-<hash>[+dirty]` (`+dirty` quando há
mudanças não commitadas). O literal `"(devel)"` só aparece quando o VCS
stamping está desligado ou indisponível (`-buildvcs=false`, `git` não
instalado, repositório fora de um clone git, ou build sem suporte a
módulo) — exatamente o caminho que `devVersion`, nesta decisão, cobre.
Isso não exige nenhuma mudança de código: `readModuleVersion()` já repassa
o que `debug.ReadBuildInfo` devolver, seja a pseudo-versão ou `"(devel)"`,
e os dois são igualmente honestos (nenhum dos dois é uma tag real) —
`devVersion` continua sendo só o terceiro caso, build info ausente por
completo. Os exemplos de saída em `quickstart.md` e `contracts/
version-flag.md` foram corrigidos para mostrar a pseudo-versão como o
resultado comum de um build local, com `(devel)` citado como o resultado
de `-buildvcs=false`.

**Alternativas consideradas**:
- Um texto em português, como `"(compilação de desenvolvimento)"` —
  rejeitada: a regra de idioma do projeto (`CLAUDE.md`, "Fluxo do Spec
  Kit") cobre a prosa dos artefatos e da documentação, não um token de
  metadado de build no mesmo registro de `"(devel)"`/`v0.1.0` — misturar os
  dois obrigaria um caso especial só para reconhecer e traduzir o literal
  que o próprio Go já produz na fonte 2, sem nenhum ganho prático (quem lê
  a saída de `--version` para colar num relato de bug já está lidando com
  um token técnico, não com prosa).
- Um texto mais descritivo, como `"dev-build"` ou `"unreleased"` —
  rejeitada: nenhum ganho sobre reaproveitar o que o Go já escreve, e
  introduz um segundo vocabulário para a mesma situação (quando
  `ReadBuildInfo` já devolve `"(devel)"` sozinho, sem passar pelo literal
  do código).

## 5. Como testar a resolução da versão sem built/instalar binários reais

**Decisão**: extrair a decisão (qual das três fontes prevalece) para uma
função pura, `pickVersion(buildOverride, moduleVersion string) string`,
separada da chamada real a `debug.ReadBuildInfo()` (`readModuleVersion`,
sem lógica própria). Os três cenários que a especificação pede (FR-005,
FR-006, FR-007/FR-008) são testados chamando `pickVersion` diretamente com
valores já resolvidos — sem compilar nem instalar nenhum binário de
verdade.

**Motivo**: `debug.ReadBuildInfo()` dentro de um teste `go test` reflete o
binário de teste, não um binário instalado via `go install @tag` nem um
compilado com `-ldflags` — não há como fazer esse teste observar os três
cenários reais chamando a função de I/O diretamente. Separar a decisão
pura do acesso ao build info é a mesma técnica que o Princípio II já
legitima para `time.Now()` (checar a lógica, não a fonte externa em si) e
que o Princípio X já exige (cada camada testada isolada do que ela
depende) — aqui aplicada dentro do mesmo arquivo descartável, sem precisar
de porta nem mock, porque não há I/O real nenhum para substituir.

**Alternativas consideradas**:
- Testar `resolveVersion()` fim a fim via subprocesso, compilando o
  binário com `go build -ldflags` dentro do teste — rejeitada: o projeto
  não tem testes automatizados de ponta a ponta (Princípio X,
  `specs/.../quickstart.md` é o lugar para isso); seria lento,
  dependente do ambiente de CI ter a toolchain do Go disponível, e a
  mesma verificação de precedência já é inteiramente coberta testando
  `pickVersion` puro. A validação com um binário de verdade instalado a
  partir de uma tag real fica para `quickstart.md`.
