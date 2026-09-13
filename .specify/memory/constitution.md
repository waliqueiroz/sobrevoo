<!--
Sync Impact Report
==================
Version change: 1.1.0 → 1.1.1
Rationale: PATCH — clarified that the pt-BR language policy also covers
repository guidance documents aimed at people or coding agents (e.g.
CLAUDE.md), not only Spec Kit flow artifacts. Triggered by CLAUDE.md having
been generated in English in this session, which the user flagged as a
violation of the language policy's intent. No principle added, removed, or
redefined; no new governance introduced.

Modified sections:
  - Stack Tecnológica e Idioma dos Artefatos — extended the pt-BR requirement
    to repository guidance docs (CLAUDE.md or equivalent), with code blocks,
    commands, paths and identifiers still in English.

Added principles: N/A
Removed sections: N/A

Templates requiring alignment review:
  - .specify/templates/plan-template.md — ⚠ still pending manual check that the
    "Constitution Check" gate references all ten principles explicitly (was eight,
    from the previous 1.0.0 → 1.1.0 amendment).
  - .specify/templates/tasks-template.md — ⚠ still pending manual check that generated
    task descriptions follow the given/when/then + no-table-tests convention
    (Principle X) and the ports/service-layer file-naming convention (Principle IX).
  - CLAUDE.md — MUST be rewritten in Portuguese prose (code/commands/paths/identifiers
    stay in English) to comply with this amendment.

Follow-up TODOs:
  - None. All placeholders were resolved from user-supplied input.
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
application.
**Rationale**: portas explícitas tornam as dependências do núcleo visíveis,
substituíveis e mockáveis, e impedem o vazamento de detalhes de infraestrutura
para dentro das regras de negócio.

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

### IX. Organização de Portas, Service Layer e Mocks
Nenhum arquivo de porta genérico (`ports.go`, `interfaces.go`) é permitido em
nenhuma camada. Uma interface que manipula ou produz uma entidade específica
MUST ser declarada no mesmo arquivo dessa entidade (por exemplo, `TrackParser`
em `track.go`, pois produz `Track`). Uma interface sem entidade dona MUST
ganhar um arquivo próprio, nomeado pelo conceito que representa (por exemplo,
`Simplifier` em `simplification.go`, `Smoother` em `smoothing.go`).

Cada caso de uso em `internal/application` — a service layer do projeto —
MUST ser modelado como uma interface exportada terminada em `Service` (por
exemplo, `InspectTrackService`), implementada por uma struct não exportada
com o mesmo nome em minúsculas (`inspectTrackService`), construída por uma
função `NewXService(...)`. Um adapter de entrada MUST depender exclusivamente
da interface, nunca da struct concreta.

Mocks de qualquer interface — porta de domínio ou serviço de aplicação —
MUST ser gerados com `go.uber.org/mock/mockgen`, via diretiva `//go:generate`
posicionada imediatamente acima da própria interface, nunca centralizada em
um arquivo à parte. A saída MUST viver em um subpacote `mock_<nome do
pacote>` dentro do pacote onde a interface é declarada (por exemplo,
`internal/domain/mock_domain`, `internal/application/mock_application`), um
arquivo gerado por interface.
**Rationale**: nomes de arquivo genéricos escondem o que o código realmente
faz e viram um "catch-all" para qualquer interface nova, independentemente de
ela pertencer ali. Nomear a service layer de forma consistente
(`XService`/`xService`/`NewXService`) e mantê-la sempre atrás de uma
interface é o que permite a um adapter de entrada (CLI hoje, REST amanhã) ser
testado sem nunca instanciar a implementação real — pré-requisito para o
Princípio X. Gerar os mocks junto da interface que representam, em vez de
centralizados, evita que a geração de mocks de um pacote dependa de outro.

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

**Version**: 1.1.1 | **Ratified**: 2026-09-13 | **Last Amended**: 2026-09-13
