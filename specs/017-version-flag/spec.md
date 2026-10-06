# Especificação de Funcionalidade: Informação de Versão da CLI

**Branch da Funcionalidade**: `017-version-flag`

**Criado em**: 2026-10-05

**Status**: Implementada, validada com o binário real; a validação da
instalação a partir de uma tag publicada (`quickstart.md` item 5) fica
pendente até a primeira tag deste repositório existir

**Entrada**: Descrição do usuário: "O Sobrevoo não tem como dizer qual versão está rodando: não há nada em `root.go` nem em lugar nenhum da CLI. Com as tags de versão que o repositório vai passar a usar, isso é a primeira coisa que falta quando alguém relata um problema, e também a forma de conferir o que foi instalado depois de um `go install`. Acrescente ao comando raiz a capacidade de informar a própria versão. Comportamento esperado: `sobrevoo --version` imprime uma única linha com a versão e sai com código 0, sem precisar de nenhum outro argumento e sem tocar em arquivo, registro ou rede. A linha é curta e estável o bastante para um script extrair o número: o nome do programa e a versão, nessa ordem. Nada de banner, arte ou texto extra. A versão não é uma constante escrita à mão que alguém precisa lembrar de atualizar. Ela vem do próprio binário: quando o programa é instalado a partir de uma tag (`go install github.com/waliqueiroz/sobrevoo/cmd/sobrevoo@v0.1.0`), é essa tag que aparece, porque o Go grava a versão do módulo no binário e a biblioteca padrão sabe lê-la. Um binário compilado localmente a partir da árvore de trabalho, sem tag, informa isso de forma honesta em vez de mentir uma versão — algo que deixe claro que é uma compilação de desenvolvimento. Uma compilação de release pode querer fixar a versão explicitamente no momento do build, sem depender do módulo; deixe esse caminho possível, e que ele tenha precedência sobre o que vem do módulo quando for usado. Restrições do projeto que valem aqui: a leitura da versão é um detalhe de ambiente, não regra de negócio: ela não entra no domínio. O ponto de entrada descartável resolve a versão e a entrega pronta ao comando raiz, da mesma forma que já entrega a configuração e os serviços — o pacote da CLI recebe uma string, não vai buscá-la sozinho. A biblioteca padrão pode ser usada diretamente para isso, sem porta nem adaptador, pela mesma exceção que o projeto já aplica a dependências puras da biblioteca padrão. Testes seguem a disciplina de sempre: um `t.Run(\"should ...\")` por cenário, nada de tabela, e os cenários que importam são o binário com versão de módulo, o binário sem versão nenhuma, e a versão fixada no build. Atualizar também o README, na seção de instalação, mostrando `sobrevoo --version` como forma de conferir o que foi instalado — no mesmo tom didático e autocontido do resto do arquivo, sem citar specs. Fora de escopo: um subcomando `version` separado além da flag; informar data de compilação, hash de commit, versão do Go ou da arquitetura; verificar se há versão mais nova; qualquer coisa que acesse a rede; e mexer em qualquer outro comando."

## Cenários de Usuário e Testes *(obrigatório)*

### História de Usuário 1 - Conferir a versão instalada (Prioridade: P1)

Como usuário do Sobrevoo, depois de instalar a ferramenta com `go install
.../sobrevoo@vX.Y.Z` (ou de atualizar para uma tag mais nova), quero rodar
um único comando e ver, numa linha curta, qual versão está de fato
instalada no meu sistema — sem precisar abrir o código, olhar o
`go.mod` ou adivinhar.

**Por que esta prioridade**: é o valor central da etapa — sem isso, não há
nenhuma forma de a ferramenta responder "qual versão é essa?", nem para o
próprio usuário, nem para quem vai ajudá-lo a resolver um problema.

**Teste Independente**: pode ser totalmente testado instalando o binário a
partir de uma tag conhecida e rodando `sobrevoo --version`, conferindo que
a linha impressa cita exatamente essa tag.

**Cenários de Aceitação**:

1. **Dado** um binário instalado a partir da tag `v0.1.0` (`go install
   .../sobrevoo@v0.1.0`), **Quando** o usuário roda `sobrevoo --version`,
   **Então** a ferramenta imprime uma única linha contendo o nome do
   programa e `v0.1.0`, e sai com código 0.
2. **Dado** esse mesmo binário, **Quando** o usuário roda `sobrevoo
   --version` sem nenhum outro argumento, **Então** a ferramenta não lê
   nem escreve nenhum arquivo, não consulta o registro de dados
   geográficos e não acessa a rede.

---

### História de Usuário 2 - Relatar um problema com a versão certa (Prioridade: P2)

Como usuário do Sobrevoo relatando um comportamento inesperado, quero
poder incluir a versão exata que estou usando no relato, para que quem for
investigar saiba, sem perguntar, qual código estava rodando.

**Por que esta prioridade**: depende da História 1 já existir; o valor
aqui é o uso concreto da informação (comunicar a versão a outra pessoa),
não a capacidade de obtê-la.

**Teste Independente**: pode ser totalmente testado rodando `sobrevoo
--version` e colando a saída, sem edição, num relato — a linha por si só
já identifica a versão sem contexto adicional.

**Cenários de Aceitação**:

1. **Dado** qualquer binário do Sobrevoo, **Quando** o usuário roda
   `sobrevoo --version`, **Então** a saída é uma única linha, sem banner,
   arte ou texto decorativo, fácil de copiar e colar inteira num relato.

---

### História de Usuário 3 - Saber que é uma compilação de desenvolvimento (Prioridade: P3)

Como usuário (ou colaborador) que compilou o Sobrevoo localmente a partir
da árvore de trabalho, sem usar `go install` a partir de uma tag, quero que
`--version` me diga honestamente que esta é uma compilação de
desenvolvimento, em vez de inventar ou repetir um número de versão de
release que não corresponde ao que está de fato rodando.

**Por que esta prioridade**: evita um problema de confiança — sem isso, um
binário de desenvolvimento poderia ser confundido com uma release
publicada, levando a relatos de bug atribuídos à versão errada.

**Teste Independente**: pode ser totalmente testado compilando o binário
diretamente da árvore de trabalho (sem tag) e conferindo que a saída de
`--version` não aparenta ser um número de release.

**Cenários de Aceitação**:

1. **Dado** um binário compilado localmente a partir da árvore de
   trabalho, sem nenhuma tag de versão envolvida, **Quando** o usuário roda
   `sobrevoo --version`, **Então** a linha impressa deixa claro que se
   trata de uma compilação de desenvolvimento, nunca um número de versão
   de release inventado.

---

### História de Usuário 4 - Fixar a versão explicitamente numa compilação de release (Prioridade: P4)

Como responsável por publicar uma compilação de release do Sobrevoo (por
exemplo, um binário pré-compilado anexado a uma release do GitHub, fora do
fluxo de `go install`), quero poder fixar a versão explicitamente no
momento da compilação, para que `--version` informe essa versão mesmo
quando a informação do módulo não é suficiente ou não está disponível.

**Por que esta prioridade**: é o caminho menos usado no dia a dia (a
maioria dos usuários instala via `go install` a partir de uma tag), mas
importante para quem automatiza releases fora desse fluxo; não bloqueia
nenhuma das três histórias anteriores.

**Teste Independente**: pode ser totalmente testado compilando o binário
com a versão fixada explicitamente no momento do build e conferindo que
`--version` informa exatamente essa versão, mesmo que a árvore de trabalho
não tenha nenhuma tag correspondente.

**Cenários de Aceitação**:

1. **Dado** um binário compilado com a versão fixada explicitamente no
   momento do build, **Quando** o usuário roda `sobrevoo --version`,
   **Então** a linha impressa reflete exatamente a versão fixada no build,
   não a que viria do módulo.
2. **Dado** esse mesmo binário, fixado com uma versão no build e também
   instalado a partir de uma tag diferente, **Quando** o usuário roda
   `sobrevoo --version`, **Então** a versão fixada no build prevalece sobre
   a da tag.

---

### Casos Extremos

- O que acontece se o usuário roda `sobrevoo --version` junto com outras
  flags ou um subcomando (ex.: `sobrevoo plan --version`)? Fora de escopo
  desta etapa — a flag é reconhecida no comando raiz, sem argumentos nem
  subcomando; o comportamento de `--version` combinado com um subcomando
  não é um requisito aqui.
- O que acontece se o binário foi compilado de um jeito que nem fixa a
  versão no build nem carrega informação de módulo (ex.: `go build` sem
  nenhum metadado de módulo disponível)? O resultado é o mesmo da História
  3 — uma indicação honesta de compilação de desenvolvimento, nunca uma
  versão inventada.
- O que acontece se a tag usada na instalação não segue o padrão `vX.Y.Z`?
  A ferramenta apenas repete o que o Go gravou no binário — não valida nem
  reformata o texto da versão.

## Requisitos *(obrigatório)*

### Requisitos Funcionais

- **FR-001**: O sistema DEVE reconhecer uma flag `--version` no comando
  raiz da CLI (`sobrevoo --version`), sem exigir nenhum argumento
  adicional.
- **FR-002**: Ao rodar `sobrevoo --version`, o sistema DEVE imprimir
  exatamente uma linha de saída, contendo o nome do programa seguido da
  versão, nessa ordem — sem banner, arte, ou qualquer texto além disso —, e
  terminar com código de saída 0.
- **FR-003**: `sobrevoo --version` NUNCA DEVE ler ou escrever qualquer
  arquivo do usuário, consultar o registro de dados geográficos, nem
  acessar a rede.
- **FR-004**: A versão informada NÃO DEVE ser um valor fixo no código-fonte
  que precise ser lembrado e atualizado manualmente a cada release — ela
  DEVE ser obtida a partir de metadados que o próprio processo de
  compilação já registra no binário.
- **FR-005**: Quando o binário é instalado a partir de uma tag de versão
  do módulo (ex.: `go install .../sobrevoo@v0.1.0`), a versão informada
  DEVE corresponder exatamente a essa tag.
- **FR-006**: Quando o binário é compilado localmente a partir da árvore
  de trabalho, sem nenhuma tag de versão associada, a versão informada
  DEVE indicar claramente que se trata de uma compilação de
  desenvolvimento — nunca um número de versão de release que não
  corresponde à realidade.
- **FR-007**: DEVE existir uma forma de fixar a versão explicitamente no
  momento da compilação (independente da tag do módulo), destinada a
  compilações de release feitas fora do fluxo padrão de `go install`.
- **FR-008**: Quando uma versão é fixada explicitamente no momento da
  compilação (FR-007), ela DEVE prevalecer sobre a versão que viria dos
  metadados do módulo (FR-005), mesmo que ambas estejam presentes no
  mesmo binário.
- **FR-009**: Nenhum comando existente do Sobrevoo, além do comportamento
  novo do comando raiz descrito aqui, DEVE ter seu comportamento alterado
  por esta funcionalidade.
- **FR-010**: A documentação de instalação do projeto DEVE mostrar
  `sobrevoo --version` como a forma de confirmar o que foi efetivamente
  instalado.

### Entidades-Chave

- **Versão do programa**: um texto curto que identifica o build em
  execução — uma tag de release (ex.: `v0.1.0`), uma indicação explícita de
  compilação de desenvolvimento, ou um valor fixado manualmente no momento
  da compilação. Não é um dado de negócio do Sobrevoo; é um metadado do
  próprio binário.

## Critérios de Sucesso *(obrigatório)*

### Resultados Mensuráveis

- **SC-001**: Um usuário consegue descobrir a versão exata de qualquer
  binário do Sobrevoo que tenha em mãos rodando um único comando, em menos
  de um segundo, sem precisar consultar nenhuma outra fonte.
- **SC-002**: Um binário instalado a partir de uma tag publicada sempre
  informa exatamente essa tag — nunca um valor desatualizado, genérico ou
  divergente.
- **SC-003**: Um binário compilado localmente, sem tag, nunca é confundido
  com uma versão de release: a saída de `--version` deixa isso evidente
  sempre que não houver uma versão fixada explicitamente no build.
- **SC-004**: Quem publica uma compilação de release fora do fluxo de `go
  install` consegue garantir que `--version` informa a versão correta,
  fixando-a explicitamente no momento do build.
- **SC-005**: A saída de `--version` é suficientemente estável e livre de
  texto extra para ser usada diretamente em um script ou relato de
  problema, sem edição.

## Suposições

- O formato exato da linha impressa (ex.: `sobrevoo v0.1.0` ou variação
  equivalente) fica a cargo do planejamento técnico; o requisito desta
  especificação é que ela contenha o nome do programa e a versão, nessa
  ordem, numa única linha sem texto extra.
- O texto que indica uma compilação de desenvolvimento (História 3) segue
  o mesmo princípio de honestidade (nunca inventar uma versão de release),
  mas seu valor exato também fica a cargo do planejamento técnico.
- `--version` é reconhecida apenas na invocação direta do comando raiz
  (`sobrevoo --version`), sem argumentos ou subcomando adicionais; seu
  comportamento quando combinada com um subcomando está fora do escopo
  desta etapa (Casos Extremos).
- Não há subcomando `version` separado — só a flag no comando raiz, como
  já restringe o pedido original.
- Informar data de compilação, hash de commit, versão do Go ou da
  arquitetura, verificar se há versão mais nova, ou qualquer acesso à
  rede, permanecem fora do escopo desta etapa.
- Nenhum outro comando do Sobrevoo muda nesta etapa.
