<!--
Sync Impact Report
==================
Version change: 1.2.0 → 1.2.1
Rationale: PATCH — clarifies file layout within a domain file that declares
a port: the //go:generate directive goes right after the package clause, and
the interface is declared at the top of the file (right after imports),
before the entity, enums and constructors — the layout used by
waliqueiroz/mystery-gifter-api (user.go, group.go). No rule is added or
removed; it makes an existing organization convention (Principle IX)
precise. Applied to track.go and geo_data_source.go, the only domain files
that still declared their interfaces at the bottom.

Previous amendment (1.1.1 → 1.2.0), kept for context:
Rationale: MINOR — materially expands Principles II and IX to codify, as
standing governance rather than one-off feature history, patterns the user
had the agent learn from an external reference repository
(waliqueiroz/mystery-gifter-api) and apply piecemeal across four rounds of
feedback on feature 002. The ambiguity being closed here already caused one
real rework in this project (a first draft of feature 002 implemented one
single-method interface per use case, read literally from the previous
wording of Principle IX) — this amendment exists so future features don't
repeat it without needing another explicit correction from the user. No
principle is removed or contradicted; both affected principles keep their
prior rules intact and add scope that was previously undocumented at the
constitution level (it lived only in a feature's research.md and in
CLAUDE.md).

Modified principles:
  - II. Portas para Toda Dependência Externa — added an explicit carve-out:
    a direct call to a pure Go standard-library function (e.g. time.Now())
    is not an "external dependency" under this principle and needs no port
    or wrapper abstraction. Only genuine I/O or out-of-process state needs
    one.
  - IX. Organização de Portas, Service Layer e Mocks -> IX. Organização de
    Portas, Service Layer, Localização da Regra de Negócio e Mocks —
    (a) replaced the ambiguous "cada caso de uso... MUST ser modelado como
    uma interface exportada terminada em Service" with an explicit
    service-per-resource rule (one exported XService interface per
    resource/aggregate, one named method per operation, Execute banned, one
    interface-per-use-case banned, services may depend on other services);
    (b) added a port-naming-by-architectural-role rule (persistence ports
    are XRepository, not XRegistry/XStore/...); (c) added a new paragraph
    stating where business logic lives: non-trivial DTOs and any logic
    beyond "call a port in order" belong in internal/domain (as an entity
    method or a free pure function), never as a DTO or loose helper in
    internal/application.

Added principles: N/A
Removed sections: N/A

Templates requiring alignment review:
  - .specify/templates/plan-template.md — still pending manual check that the
    "Constitution Check" gate's Principle IX row reflects the expanded rule
    (service-per-resource, business-logic placement, port naming), not just
    the file-organization rule it originally covered.
  - CLAUDE.md — MUST be updated so its "Onde vive a regra de negócio" and
    service-layer guidance read as standing project convention (already
    true in practice for features 001/002), not as feature-002-specific
    history, and so its Princípio II mention doesn't imply time.Now() ever
    needed a port. Tracked as a same-session follow-up, not deferred.

Follow-up TODOs:
  - None. All placeholders were resolved from user-supplied input and from
    specs/002-geo-data-registry/research.md (items 10.1, 13, 14, 16).
-->

# Sobrevoo Constitution

## Core Principles

### I. Arquitetura Hexagonal (Núcleo Isolado)
O núcleo da aplicação (domain + application) NUNCA importa bibliotecas de
infraestrutura. É proibido que parsers de GPX, browsers headless, ffmpeg, drivers
de renderização de vídeo ou qualquer acesso a filesystem cruzem essa fronteira. O
núcleo contém exclusivamente regras de negócio e casos de uso, livres de qualquer
dependência de I/O concreta.
**Rationale**: garante que a lógica de geração de sobrevoos possa ser testada e
evoluída independentemente de qualquer biblioteca externa, e que trocar uma
implementação de infraestrutura (por exemplo, trocar o parser de GPX) nunca exija
alterar regra de negócio.

### II. Portas para Toda Dependência Externa
Toda dependência externa — parser de arquivo GPS, renderizador de vídeo, sistema de
arquivos, processo externo como ffmpeg, browser headless — é acessada
exclusivamente através de uma porta (interface) declarada no núcleo. A
implementação concreta de cada porta vive em `internal/infra/outbound`. Nenhum
adapter concreto pode ser referenciado diretamente pelo domain ou pela
application. Uma chamada direta a uma função pura da biblioteca padrão do Go
(por exemplo, `time.Now()`) NÃO é considerada dependência externa para efeito
deste princípio, e NÃO exige porta nem abstração dedicada: só exige porta o
que de fato realiza I/O real ou depende de estado fora do processo em
execução — arquivo, rede, processo externo.
**Rationale**: portas explícitas tornam as dependências do núcleo visíveis,
substituíveis e mockáveis, e impedem o vazamento de detalhes de infraestrutura
para dentro das regras de negócio. Embrulhar uma função pura e determinística
da biblioteca padrão atrás de uma porta só para poder mockar algo que já é
trivial de verificar em teste (por exemplo, `time.Now()` checado por uma
janela de tempo, não por um valor exato) traz o custo de uma abstração sem
nenhum ganho real de testabilidade ou de substituibilidade — este princípio
existe para dependências que de fato variam por ambiente ou infraestrutura,
não para a linguagem em si.

### III. Entrypoints Descartáveis
CLI, uma futura API REST, uma GUI, ou qualquer outro ponto de entrada são adapters
equivalentes sobre o mesmo caso de uso da application layer. Nenhuma regra de
negócio pode ser duplicada ou implementada exclusivamente dentro de um entrypoint.
Um entrypoint MUST ser substituível ou removível sem exigir alteração no núcleo.
**Rationale**: mantém real a possibilidade de adicionar novos modos de uso (API,
GUI) sem reescrever a lógica existente, e evita que atalhos de conveniência em um
entrypoint se transformem em regra de negócio oculta e não reaproveitável.

### IV. Neutralidade Geográfica
A aplicação NUNCA embute dados de mapa, relevo, ou coordenadas de nenhuma região
específica do planeta. Todo dado geográfico usado no processamento é fornecido e
registrado pelo próprio usuário, via arquivos de entrada (GPS/GPX) ou configuração
explícita. O comportamento da aplicação MUST ser idêntico para qualquer trajeto,
em qualquer lugar do mundo.
**Rationale**: preserva o caráter genérico e pessoal da ferramenta; privilegiar
uma região específica (por exemplo, bounding boxes fixos ou datasets locais
embutidos) tornaria a ferramenta inutilizável ou tendenciosa fora desse contexto.

### V. Funcionamento Offline
Nenhuma etapa do processamento — parsing, cálculo de trajeto, renderização de
vídeo — pode depender de rede, chave de API, ou serviço externo em tempo de
execução. Toda dependência necessária à execução MUST estar disponível
localmente: binários instalados, bibliotecas locais, e arquivos fornecidos pelo
usuário.
**Rationale**: Sobrevoo é uma ferramenta pessoal; depender de serviços externos
introduz custo, indisponibilidade e risco de privacidade sobre dados de
localização do usuário.

### VI. Testes Automatizados no Núcleo
Testes unitários usam testify para asserções e uber-go/mock para dublês de porta.
O núcleo (domain + application) MUST ser inteiramente testável sem tocar em
disco, rede, ou processo externo — toda porta consumida em teste é substituída
por um mock gerado a partir da interface declarada no núcleo.
**Rationale**: garante velocidade e determinismo dos testes, e força o desenho
correto das portas exigido pelo Princípio II — uma porta difícil de mockar é
sinal de vazamento de infraestrutura.

### VII. Erros Sentinela no Domínio
Erros de negócio são declarados como valores sentinela no domain (por exemplo,
`var ErrTrackEmpty = errors.New(...)`). Cada adapter é responsável por traduzir
esses erros sentinela para sua própria representação: a CLI os converte em código
de saída de processo, e um futuro adapter REST MUST traduzi-los em status HTTP.
O núcleo NUNCA retorna, loga, ou depende de tipos de erro específicos de
infraestrutura.
**Rationale**: centraliza o vocabulário de falhas de negócio em um só lugar e
permite que cada adapter decida como comunicar essas falhas em seu próprio
protocolo, sem duplicar lógica de classificação de erro entre entrypoints.

### VIII. Configuração Injetada
Configuração — flags de CLI, arquivos de config, variáveis de ambiente — é lida
e resolvida inteiramente pelo adapter e injetada no núcleo como parâmetros ou
estruturas explícitas definidas pelo próprio núcleo. O núcleo NUNCA lê
configuração diretamente de ambiente, arquivo, ou linha de comando.
**Rationale**: mantém o núcleo livre de acoplamento com a forma de configuração
de um entrypoint específico, e permite que adapters futuros configurem os mesmos
casos de uso por vias diferentes (flags vs. corpo de requisição HTTP, etc.) sem
tocar na application layer.

### IX. Organização de Portas, Service Layer, Localização da Regra de Negócio e Mocks
Nenhum arquivo de porta genérico (`ports.go`, `interfaces.go`) é permitido em
nenhuma camada. Uma interface que manipula ou produz uma entidade específica
MUST ser declarada no mesmo arquivo dessa entidade (por exemplo, `TrackParser`
em `track.go`, pois produz `Track`), no início do arquivo — logo após os
imports e antes da entidade, dos enums e dos construtores, com a diretiva
`//go:generate` imediatamente após a cláusula `package`. Uma interface sem entidade dona MUST
ganhar um arquivo próprio, nomeado pelo conceito que representa (por exemplo,
`Simplifier` em `simplification.go`, `Smoother` em `smoothing.go`). Toda porta
MUST ser nomeada pelo papel arquitetural que exerce, não pelo dado que
manipula — por exemplo, uma porta de persistência chama-se `XRepository`
(nunca `XRegistry`, `XStore`, ou similar), com o campo correspondente na
struct do serviço que a usa seguindo o mesmo nome (`xRepository
domain.XRepository`).

A service layer de `internal/application` é organizada por **recurso ou
agregado**, não por caso de uso. Cada recurso MUST ser modelado como uma
única interface exportada terminada em `Service` (`XService`), onde `X`
nomeia o recurso que o serviço gerencia (por exemplo, `GeoDataService`) —
implementada por uma struct não exportada com o mesmo nome em minúsculas
(`xService`), construída por uma função `NewXService(...)`. Cada caso de uso
relacionado a esse recurso MUST virar um método nomeado pela operação que
realiza (`Register`, `List`, `Remove`, `CheckCoverage`) na mesma interface —
é PROIBIDO criar uma interface nova por caso de uso, e é PROIBIDO modelar
qualquer operação como um método genérico `Execute`. Um serviço de aplicação
PODE depender de outro serviço de aplicação (não só de portas do domínio)
quando uma operação precisa de lógica que já vive em outro serviço. Um
adapter de entrada MUST depender exclusivamente da interface do serviço,
nunca da struct concreta.

Dentro do núcleo, um serviço de aplicação só orquestra: chama uma ou mais
portas, delega a regra de negócio para um construtor ou função de domínio, e
devolve o resultado — ele NUNCA decide uma regra de negócio por conta
própria. Qualquer DTO de saída não trivial (isto é, que não seja um valor
escalar simples) usado por um serviço de aplicação MUST ser um tipo
declarado em `internal/domain`, nunca um DTO próprio de
`internal/application`. Qualquer lógica que vá além de "chamar uma porta na
ordem certa" — construir uma entidade, calcular algo a partir de uma
coleção, aplicar uma regra de negócio — MUST ser um construtor ou função
pura de domínio: um método da entidade quando a lógica pertence a uma única
entidade, ou uma função livre quando opera sobre coleções ou múltiplas
entradas sem uma entidade dona. É PROIBIDO qualquer função solta ou helper
que contenha regra de negócio em `internal/application`.

Mocks de qualquer interface — porta de domínio ou serviço de aplicação —
MUST ser gerados com `go.uber.org/mock/mockgen`, via diretiva `//go:generate`
posicionada imediatamente acima da própria interface, nunca centralizada em
um arquivo à parte. A saída MUST viver em um subpacote `mock_<nome do
pacote>` dentro do pacote onde a interface é declarada (por exemplo,
`internal/domain/mock_domain`, `internal/application/mock_application`), um
arquivo gerado por interface.
**Rationale**: nomes de arquivo e de porta genéricos escondem o que o código
realmente faz e viram um "catch-all" para qualquer interface nova,
independentemente de ela pertencer ali — nomear uma porta pelo papel
arquitetural que exerce (por exemplo, `XRepository` para persistência) é tão
importante quanto nomear o arquivo que a declara pelo conceito certo.
Modelar a service layer por recurso, com um método por operação, e proibir
tanto uma interface por caso de uso quanto um método genérico `Execute`,
evita que a application layer vire uma coleção de objetos soltos em vez de
uma service layer coesa — o padrão de referência deste projeto é
`waliqueiroz/mystery-gifter-api` (`GroupService`, `UserService`,
`GroupInviteService`, cada um reunindo todas as operações do recurso que
gerencia, podendo depender de outro serviço). Exigir que toda regra de
negócio não trivial — DTO ou lógica — viva em `internal/domain`, nunca em
`internal/application`, é o que garante que um serviço de aplicação
permaneça pura orquestração, legível método a método, sem regra de negócio
real escondida atrás de uma porta ou de um helper solto. Nomear a service
layer de forma consistente (`XService`/`xService`/`NewXService`) e
mantê-la sempre atrás de uma interface é o que permite a um adapter de
entrada (CLI hoje, REST amanhã) ser testado sem nunca instanciar a
implementação real — pré-requisito para o Princípio X. Gerar os mocks junto
da interface que representam, em vez de centralizados, evita que a geração
de mocks de um pacote dependa de outro.

### X. Testes: Given/When/Then, Builders e Isolamento por Camada
Todo teste unitário MUST ser escrito como um ou mais
`t.Run("should ...", func(t *testing.T) {...})`, com comentários `// given`,
`// when` e `// then` demarcando cada etapa dentro do subteste. Testes
tabulares (`[]struct{...}` percorrido por um `for` que gera os casos) são
PROIBIDOS — cada cenário MUST ser seu próprio `t.Run`, mesmo que isso repita
configuração entre cenários.

Quando a construção de uma entidade ou DTO em teste é repetitiva ou tem
muitos campos, um builder fluente MUST ser criado em um subpacote
`build_<nome do pacote>` (por exemplo, `internal/domain/build_domain`,
`internal/application/build_application`), no formato `NewXBuilder()` com
defaults sensatos, métodos `WithCampo(...)`/`WithoutCampo()` retornando o
próprio builder, e um método terminal `Build()`.

Cada camada MUST ser testada isoladamente das demais: o núcleo (domain +
application) é testado com portas mockadas, nunca com adapters reais; um
adapter de entrada (por exemplo, a CLI) MUST ser testado com a service layer
mockada, nunca com a implementação real do serviço nem com os adapters de
saída reais por trás dele. Nenhuma suíte automatizada substitui uma
verificação de integração real de ponta a ponta — quando essa verificação é
necessária, ela é manual (por exemplo, via `quickstart.md`).
**Rationale**: o formato given/when/then torna a intenção de cada teste
legível sem decifrar uma tabela de casos, e proibir testes tabulares evita
que um único `t.Run` combine múltiplas asserções não relacionadas sob um
nome genérico. Builders eliminam a repetição de literais de struct extensos
sem esconder o dado relevante para o cenário sendo testado. Isolar cada
camada atrás de mocks é o que torna possível testar um adapter de entrada
sem reexecutar toda a lógica de negócio a cada mudança — a mesma disciplina
que o Princípio VI já exige do núcleo, estendida para os adapters.

## Stack Tecnológica e Idioma dos Artefatos

Sobrevoo é implementado em Go, como projeto pessoal e open source.

Todos os artefatos de especificação do fluxo Spec Kit — `spec.md`, `plan.md`,
`tasks.md`, `research.md`, checklists, e qualquer documento gerado por esse fluxo
— assim como toda comunicação conduzida durante o fluxo, DEVEM ser escritos em
português do Brasil. A mesma regra se aplica a qualquer documento de orientação
do repositório dirigido a pessoas ou a agentes de codificação (por exemplo,
`CLAUDE.md` ou equivalente) — a prosa MUST estar em português do Brasil, com
blocos de código, comandos, caminhos de arquivo e identificadores permanecendo
em inglês, como em qualquer outro artefato deste projeto.

Código-fonte, identificadores, nomes de pacotes, nomes de arquivos, nomes de
branch, mensagens de commit e comentários no código permanecem em inglês. Termos
técnicos já consagrados (por exemplo, "waypoint", "keyframe", "pipeline") não são
traduzidos, mesmo em texto redigido em português.

## Fluxo de Desenvolvimento e Verificação de Conformidade

Qualquer mudança que introduza ou modifique uma porta, um adapter, ou que cruze a
fronteira hexagonal MUST justificar explicitamente, no `plan.md` da feature, como
a arquitetura descrita nos Princípios I–III é preservada.

Uma revisão (pull request) que adicione lógica de negócio fora de
`internal/domain` ou `internal/application` MUST ser rejeitada, a menos que a
adição seja código de acoplamento (glue code) estritamente limitado ao adapter, e
isso esteja explicitado na descrição da mudança.

A seção "Constitution Check" dos artefatos de planejamento do Spec Kit MUST
confirmar, para cada um dos dez princípios acima, que ele não foi violado antes
de a fase de implementação começar.

## Governance

Esta constituição prevalece sobre qualquer outra prática, convenção ou
documento de desenvolvimento do projeto Sobrevoo. Em caso de conflito entre esta
constituição e qualquer outro artefato (README, templates, guias de agente), a
constituição prevalece.

**Procedimento de emenda**: qualquer alteração a esta constituição MUST ser
proposta por escrito (pull request alterando este arquivo), descrever o motivo da
mudança, e incluir a atualização do Sync Impact Report no topo do arquivo antes
de ser mesclada.

**Política de versionamento**: esta constituição segue versionamento semântico:
- MAJOR: remoção ou redefinição incompatível de um princípio existente.
- MINOR: adição de um novo princípio ou expansão material de uma seção existente.
- PATCH: correções de redação, esclarecimentos e refinamentos não semânticos.

**Revisão de conformidade**: todo `plan.md` gerado pelo fluxo Spec Kit MUST
incluir uma seção "Constitution Check" que valide a aderência da feature a cada
princípio listado acima antes do início da fase de design detalhado, e novamente
antes do início da implementação. Qualquer desvio MUST ser justificado
explicitamente na seção de Complexity Tracking do plano, ou o desvio MUST ser
eliminado.

**Version**: 1.2.1 | **Ratified**: 2026-09-13 | **Last Amended**: 2026-09-19
