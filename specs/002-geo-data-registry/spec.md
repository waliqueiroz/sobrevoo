# Especificação de Funcionalidade: Registro de Dados Geográficos Locais

**Branch da Funcionalidade**: `002-geo-data-registry`

**Criado em**: 2026-09-13

**Status**: Rascunho

**Entrada**: Descrição do usuário: "Segunda etapa do Sobrevoo. Esta etapa cobre apenas o gerenciamento dos dados geográficos locais que serão usados para renderizar o vídeo. Nada de câmera, renderização ou vídeo ainda. Como a ferramenta funciona totalmente offline e não embute dados de nenhuma região, o usuário precisa registrar quais arquivos de mapa e de relevo ele já baixou para a sua máquina. A partir desse registro, a ferramenta sabe que áreas do planeta consegue renderizar. O usuário consegue: registrar um arquivo de dados local informando um nome de sua escolha (a ferramenta descobre sozinha o tipo e a área coberta); listar o que já está registrado (nome, tipo, área coberta, se o arquivo ainda existe); remover um registro sem apagar o arquivo; verificar se um trajeto está coberto pelos dados registrados, e entender o que ficou de fora quando não estiver. Requisitos: registro persistente entre execuções e independente de diretório; recusar arquivo inexistente/ilegível/formato não suportado; recusar nome duplicado; detectar arquivo movido/apagado sem falhar; escolha determinística e explicável quando múltiplas fontes cobrem a mesma área; trajeto só é coberto com mapa base E relevo para toda a extensão; nenhuma região privilegiada, incluindo antimeridiano. Fora de escopo: baixar dados, converter formatos, renderização, vídeo, GUI, API."

## Clarifications

### Session 2026-09-13

- Q: Quando duas fontes registradas do mesmo tipo (mapa base ou relevo) cobrem a área necessária de um trecho do trajeto, qual critério a ferramenta deve usar para escolher entre elas? → A: Preferir a fonte com a menor área coberta (mais específica); empate resolvido pelo registro mais antigo.
- Q: Quando um trajeto está parcialmente coberto, como a ferramenta deve descrever para o usuário qual trecho ficou de fora? → A: Como uma lista de subtrechos contínuos do trajeto, cada um identificado pelas coordenadas geográficas de início e fim do trecho descoberto.

## Cenários de Usuário e Testes *(obrigatório)*

### História de Usuário 1 - Registrar um arquivo de dados geográficos (Prioridade: P1)

Como usuário do Sobrevoo, eu já baixei manualmente alguns arquivos de mapa base e de relevo para a minha máquina. Eu quero apontar a ferramenta para um desses arquivos e dar um nome de minha escolha a ele, sem precisar dizer se é mapa ou relevo nem qual área ele cobre — a ferramenta descobre isso sozinha ao examinar o arquivo.

**Por que esta prioridade**: sem essa capacidade não existe nenhum registro para listar, consultar ou usar na verificação de cobertura — é o alicerce de toda a funcionalidade.

**Teste Independente**: pode ser totalmente testado registrando um arquivo de dados válido com um nome escolhido e confirmando que a ferramenta reporta corretamente o tipo (mapa base ou relevo) e a área geográfica descobertos, sem que o usuário os tenha informado.

**Cenários de Aceitação**:

1. **Dado** um arquivo de mapa base válido em um caminho conhecido, **Quando** o usuário registra esse arquivo com um nome ainda não usado, **Então** o registro é criado com sucesso, com tipo "mapa base" e a área geográfica coberta determinados automaticamente.
2. **Dado** um arquivo de relevo válido em um caminho conhecido, **Quando** o usuário registra esse arquivo com um nome ainda não usado, **Então** o registro é criado com sucesso, com tipo "relevo" e a área geográfica coberta determinados automaticamente.
3. **Dado** um caminho de arquivo que não existe no sistema de arquivos, **Quando** o usuário tenta registrá-lo, **Então** o registro é recusado com uma mensagem que deixa claro que o arquivo não foi encontrado.
4. **Dado** um arquivo existente cujo conteúdo não corresponde a nenhum formato de mapa base ou de relevo reconhecido pela ferramenta, **Quando** o usuário tenta registrá-lo, **Então** o registro é recusado com uma mensagem que deixa claro que o formato não é suportado.
5. **Dado** um registro já existente com o nome "europa-central", **Quando** o usuário tenta registrar outro arquivo usando o mesmo nome "europa-central", **Então** o registro é recusado com uma mensagem que deixa claro que o nome já está em uso.
6. **Dado** um arquivo registrado com sucesso, **Quando** o usuário encerra o programa, abre um novo terminal em outro diretório e inicia a ferramenta novamente, **Então** o registro continua presente e consultável.

---

### História de Usuário 2 - Verificar a cobertura de um trajeto (Prioridade: P2)

Como usuário do Sobrevoo, antes de seguir para as próximas etapas eu quero saber se já tenho, entre os dados que registrei, mapa base e relevo suficientes para cobrir um trajeto GPS específico do início ao fim. Quando não tenho, quero saber exatamente qual trecho do trajeto ainda não está coberto, para saber o que ainda preciso baixar.

**Por que esta prioridade**: é o motivo prático pelo qual o usuário mantém o registro — decidir se pode prosseguir com um trajeto ou se precisa obter mais dados antes disso.

**Teste Independente**: pode ser totalmente testado com um conjunto de registros já existentes (criados via História de Usuário 1) e um trajeto de exemplo, verificando que a ferramenta reporta corretamente se o trajeto está totalmente coberto, parcialmente coberto (indicando o trecho descoberto) ou não coberto.

**Cenários de Aceitação**:

1. **Dado** registros de mapa base e de relevo que cobrem toda a extensão de um trajeto, **Quando** o usuário verifica a cobertura desse trajeto, **Então** a ferramenta reporta o trajeto como totalmente coberto.
2. **Dado** um registro de mapa base que cobre toda a extensão do trajeto mas nenhum registro de relevo que a cubra, **Quando** o usuário verifica a cobertura desse trajeto, **Então** a ferramenta reporta o trajeto como parcialmente coberto (há mapa base em toda a extensão, mas falta relevo — cobertura real, porém incompleta) e indica que falta relevo.
3. **Dado** registros que cobrem apenas uma parte da extensão geográfica de um trajeto, **Quando** o usuário verifica a cobertura desse trajeto, **Então** a ferramenta reporta cobertura parcial e indica especificamente qual trecho do trajeto não está coberto.
4. **Dado** nenhum registro cadastrado, **Quando** o usuário verifica a cobertura de um trajeto, **Então** a ferramenta reporta o trajeto inteiro como não coberto.
5. **Dado** dois registros do mesmo tipo cujas áreas se sobrepõem e ambos cobrem a extensão necessária do trajeto, **Quando** o usuário verifica a cobertura desse trajeto, **Então** a ferramenta escolhe um dos registros seguindo uma regra fixa e informa ao usuário qual registro foi escolhido para aquele trecho.
6. **Dado** um trajeto cuja extensão cruza o meridiano de 180°, com registros que cobrem essa área em ambos os lados do meridiano, **Quando** o usuário verifica a cobertura desse trajeto, **Então** a ferramenta reporta a cobertura corretamente, sem tratar essa área de forma diferente de qualquer outra.

---

### História de Usuário 3 - Listar os dados registrados (Prioridade: P3)

Como usuário do Sobrevoo, eu quero ver de uma vez todos os dados que já registrei, com nome, tipo, área coberta, e se o arquivo original ainda está acessível no caminho que informei — para entender meu acervo de dados e perceber se algum arquivo foi movido ou apagado sem eu ter avisado a ferramenta.

**Por que esta prioridade**: dá visibilidade sobre o estado do registro, mas o usuário já consegue obter valor da ferramenta (registrar e verificar cobertura) sem essa listagem explícita, daí a prioridade menor que as duas primeiras histórias.

**Teste Independente**: pode ser totalmente testado registrando previamente alguns arquivos (um deles depois movido ou apagado do disco) e confirmando que a listagem mostra nome, tipo e área de cada um, sinalizando corretamente qual arquivo não está mais acessível.

**Cenários de Aceitação**:

1. **Dado** dois ou mais registros existentes, **Quando** o usuário lista os dados registrados, **Então** a ferramenta exibe, para cada um, nome, tipo e área geográfica coberta.
2. **Dado** um registro cujo arquivo original foi movido ou apagado do caminho informado no momento do registro, **Quando** o usuário lista os dados registrados, **Então** esse registro aparece na listagem sinalizado como indisponível, e os demais registros continuam sendo exibidos normalmente.
3. **Dado** nenhum registro existente, **Quando** o usuário lista os dados registrados, **Então** a ferramenta indica claramente que não há nenhum dado registrado.

---

### História de Usuário 4 - Remover um registro (Prioridade: P4)

Como usuário do Sobrevoo, eu quero remover do registro um dado que não uso mais ou que registrei por engano, sem que isso apague o arquivo original do meu disco.

**Por que esta prioridade**: é uma operação de manutenção do registro, útil mas menos frequente e menos crítica que registrar, verificar cobertura e listar.

**Teste Independente**: pode ser totalmente testado registrando um arquivo, removendo-o pelo nome, e confirmando que ele desaparece da listagem enquanto o arquivo original continua existindo no disco.

**Cenários de Aceitação**:

1. **Dado** um registro existente com um determinado nome, **Quando** o usuário remove esse registro pelo nome, **Então** o registro deixa de aparecer na listagem e o arquivo de dados original permanece intacto no sistema de arquivos.
2. **Dado** um nome que não corresponde a nenhum registro existente, **Quando** o usuário tenta remover um registro com esse nome, **Então** a ferramenta informa que não há registro com esse nome, sem alterar o registro existente.

---

### Casos Extremos

- O que acontece quando o usuário registra o mesmo arquivo físico duas vezes, sob nomes diferentes? Ambos os registros são aceitos e tratados como entradas independentes.
- O que acontece quando um arquivo registrado é substituído no disco por outro conteúdo, mantendo o mesmo caminho? A ferramenta não detecta automaticamente essa alteração de conteúdo; o registro continua apontando para o mesmo caminho com o tipo e a área descobertos no momento do registro original.
- Como o sistema lida com um trajeto inválido ou vazio informado para verificação de cobertura? Reaproveita a mesma validação de trajeto já existente na etapa anterior do Sobrevoo, recusando o trajeto pelo mesmo motivo.
- Como o sistema lida com uma área de dado registrado (mapa base ou relevo) que cruza o meridiano de 180°? A área é reconhecida e usada na verificação de cobertura da mesma forma que qualquer outra área, sem tratamento especial.
- O que acontece se dois registros do mesmo tipo cobrirem exatamente a mesma área, sem nenhum ser mais específico que o outro? A regra determinística de escolha ainda produz um único resultado consistente, e a ferramenta informa qual dos dois foi escolhido.
- O que acontece quando o caminho informado no registro aponta para um diretório, não para um arquivo? É tratado como arquivo ilegível/inválido para os fins desta funcionalidade, e o registro é recusado.

## Requisitos *(obrigatório)*

### Requisitos Funcionais

- **FR-001**: O sistema DEVE permitir que o usuário registre um arquivo de dados geográficos local informando o caminho do arquivo e um nome de sua escolha.
- **FR-002**: O sistema DEVE examinar o conteúdo do arquivo informado e determinar automaticamente se ele é um dado de mapa base ou um dado de relevo, sem exigir que o usuário informe o tipo.
- **FR-003**: O sistema DEVE examinar o conteúdo do arquivo informado e determinar automaticamente a área geográfica que ele cobre, sem exigir que o usuário a informe.
- **FR-004**: O sistema DEVE recusar o registro de um caminho de arquivo inexistente, com uma mensagem que deixe claro que o arquivo não foi encontrado.
- **FR-005**: O sistema DEVE recusar o registro de um arquivo que existe mas não pode ser lido, com uma mensagem que deixe claro o motivo.
- **FR-006**: O sistema DEVE recusar o registro de um arquivo cujo conteúdo não corresponda a nenhum formato de mapa base ou de relevo reconhecido pela ferramenta, com uma mensagem que deixe claro que o formato não é suportado.
- **FR-007**: O sistema DEVE recusar o registro de um novo dado com um nome que já esteja em uso por outro registro existente, com uma mensagem que deixe claro que o nome já está em uso.
- **FR-008**: O sistema DEVE manter o registro entre execuções da ferramenta, disponível independentemente do diretório de onde a ferramenta é executada.
- **FR-009**: O sistema DEVE permitir que o usuário liste todos os dados registrados, exibindo para cada um nome, tipo (mapa base ou relevo) e área geográfica coberta.
- **FR-010**: O sistema DEVE sinalizar, na listagem, qualquer registro cujo arquivo não seja mais encontrado no caminho original, sem interromper a exibição dos demais registros.
- **FR-011**: O sistema DEVE permitir que o usuário remova um registro pelo nome, sem apagar o arquivo de dados referenciado no sistema de arquivos.
- **FR-012**: O sistema DEVE informar claramente quando o usuário tenta remover um nome que não corresponde a nenhum registro existente.
- **FR-013**: O sistema DEVE permitir que o usuário verifique se um trajeto específico está coberto pelos dados registrados.
- **FR-014**: O sistema DEVE considerar um trajeto totalmente coberto somente quando existir, para toda a sua extensão, tanto um registro de mapa base quanto um registro de relevo cujo arquivo esteja presente.
- **FR-015**: Quando um trajeto não estiver totalmente coberto, o sistema DEVE reportá-lo como cobertura parcial (ou nula) e indicar, para cada subtrecho contínuo não coberto, as coordenadas geográficas de início e fim desse subtrecho, e se falta mapa base, relevo, ou ambos nele.
- **FR-016**: Quando mais de um registro do mesmo tipo cobrir a área necessária para um mesmo trecho de um trajeto, o sistema DEVE escolher o registro com a menor área geográfica coberta (o mais específico); em caso de empate, DEVE prevalecer o registro mais antigo. Essa regra DEVE produzir sempre a mesma escolha para o mesmo trecho e o mesmo conjunto de registros, e o sistema DEVE informar ao usuário qual registro foi escolhido.
- **FR-017**: O sistema NÃO DEVE considerar, para fins de verificação de cobertura, um registro cujo arquivo não seja mais encontrado no caminho registrado.
- **FR-018**: O sistema DEVE tratar áreas geográficas e trajetos que cruzam o meridiano de 180° corretamente, sem lógica especial ou resultado diferente do aplicado a qualquer outra região do planeta.
- **FR-019**: O comportamento de registro, listagem, remoção e verificação de cobertura DEVE ser idêntico para qualquer área geográfica do planeta, sem privilegiar nenhuma região.

### Entidades-Chave *(incluir se a funcionalidade envolver dados)*

- **Registro de Dado Geográfico**: representa um arquivo local que o usuário declarou à ferramenta. Atributos-chave: nome (identificador escolhido pelo usuário, único entre todos os registros), caminho do arquivo no sistema de arquivos, tipo (mapa base ou relevo), área geográfica coberta, e se o arquivo ainda é encontrado no caminho informado.
- **Trajeto**: o trajeto GPS existente (já processado pela etapa anterior do Sobrevoo) cuja cobertura está sendo verificada contra o conjunto de registros.
- **Resultado de Verificação de Cobertura**: o veredito produzido ao verificar um trajeto contra o registro — se está totalmente coberto, parcialmente coberto ou não coberto; quais registros (mapa base e relevo) foram escolhidos para cobri-lo e por quê, quando havia mais de uma opção; e, em caso de cobertura parcial ou nula, a lista de subtrechos contínuos não cobertos, cada um com suas coordenadas geográficas de início e fim e o(s) tipo(s) de dado (mapa base e/ou relevo) que falta(m) nele.

## Critérios de Sucesso *(obrigatório)*

### Resultados Mensuráveis

- **SC-001**: Um usuário consegue registrar um arquivo de dados local em uma única operação, sem precisar informar manualmente o tipo (mapa base ou relevo) nem a área geográfica coberta — ambos são determinados corretamente pela ferramenta.
- **SC-002**: A partir da listagem de registros, o usuário consegue identificar, sem consultar nenhuma outra ferramenta ou arquivo, quais registros são de mapa base, quais são de relevo, qual área cada um cobre, e quais arquivos não estão mais acessíveis.
- **SC-003**: Para qualquer trajeto verificado, o usuário consegue determinar, sem ambiguidade, se ele está totalmente coberto, parcialmente coberto, ou não coberto pelos dados registrados.
- **SC-004**: Quando a cobertura de um trajeto é parcial ou nula, o usuário consegue identificar exatamente qual trecho do trajeto está descoberto e que tipo de dado falta, sem precisar inspecionar manualmente os arquivos registrados.
- **SC-005**: Repetir a mesma verificação de cobertura sobre o mesmo trajeto e o mesmo conjunto de registros sempre produz exatamente o mesmo resultado, incluindo a mesma escolha de registros quando há sobreposição.
- **SC-006**: Remover um registro nunca altera nem apaga o arquivo de dados original no sistema de arquivos.
- **SC-007**: O comportamento de registro, listagem, remoção e verificação de cobertura é idêntico para trajetos e áreas em qualquer parte do planeta, incluindo áreas que cruzam o meridiano de 180°.

## Suposições

- A ferramenta reconhece um conjunto definido de formatos de arquivo para mapa base e para relevo; qualquer arquivo fora desse conjunto é tratado como formato não suportado. Os formatos exatos reconhecidos serão detalhados na fase de planejamento técnico desta funcionalidade, não nesta especificação.
- A área geográfica coberta por um registro é expressa como uma extensão retangular de coordenadas (bounding box), no mesmo estilo já usado para trajetos nas etapas anteriores do Sobrevoo.
- O trajeto usado na verificação de cobertura é fornecido no(s) mesmo(s) formato(s) já suportado(s) pela etapa anterior do Sobrevoo (leitura de trajeto GPX), reaproveitando a validação de trajeto já existente.
- Registrar o mesmo arquivo físico mais de uma vez, sob nomes diferentes, é permitido; cada nome é um registro independente e nenhum deles é privilegiado sobre o outro por esse motivo.
- Nomes de registro são identificadores de texto de escolha do usuário; a verificação de nome duplicado compara o texto exatamente como informado.
- Não há um limite superior definido para a quantidade de registros que o usuário pode manter.
- O local de armazenamento do registro é próprio da ferramenta (não é, por exemplo, um dos arquivos de mapa/relevo do usuário) e não precisa ser informado nem gerenciado manualmente pelo usuário.
