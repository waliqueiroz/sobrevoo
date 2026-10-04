# Especificação de Funcionalidade: Redesenho das Sobreposições de Tela em Colunas

**Branch da Funcionalidade**: `015-overlay-redesign`

**Criado em**: 2026-10-04

**Status**: Rascunho

**Entrada**: Descrição do usuário: "As sobreposições de tela precisam ficar mais limpas. Hoje cada bloco numérico é uma faixa escura atrás de uma linha de texto em caixa alta, com rótulo e valor lado a lado, e os blocos ficam empilhados um sobre o outro no alto da tela — muito peso visual para pouca informação, e a imagem por baixo fica escondida. A partir desta etapa os números aparecem sem nenhum fundo, desenhados direto sobre a imagem, e os blocos presentes ficam lado a lado numa única faixa horizontal no alto do quadro, repartindo a largura útil entre eles em colunas de mesma largura, cada bloco centralizado na sua coluna, numa ordem fixa e documentada que não depende da ordem em que o usuário os pediu. Dentro de cada bloco a informação passa a ser lida de cima para baixo em três alturas: o rótulo, por extenso e com capitalização normal em vez de abreviado em caixa alta, num corpo pequeno; o valor, num corpo bem maior, que é o que a pessoa enxerga de relance; e a unidade, de novo num corpo pequeno, embaixo — um bloco cujo valor não tem unidade, como o tempo decorrido, simplesmente não desenha a terceira altura, sem deixar buraco. Com o fundo escuro indo embora, a legibilidade passa a depender inteiramente do contorno escuro do texto, e sobre imagem de satélite clara ele precisa dar conta sozinho — sem fechar o vão interno das letras nem engrossar o traço, como já foi acertado antes. A escolha padrão de blocos muda junto, para o conjunto que faz sentido num vídeo: velocidade, elevação e distância no alto, mais o gráfico de elevação no rodapé. O tempo decorrido e o ganho de elevação passam a ser opcionais — continuam disponíveis para quem pedir, mas saem do padrão, porque são os dois que menos mudam ao longo do vídeo e porque três colunas cabem com folga onde cinco não caberiam. Para que o ganho possa ser escolhido sozinho, ele deixa de ser desenhado grudado no bloco de elevação, como um segundo número na mesma linha, e passa a ser um bloco próprio, pelo nome `gain`, ligado pelo mesmo mecanismo dos demais: o bloco de elevação passa a mostrar só a altitude do ponto, e o de ganho só o ganho, cada um com seu rótulo e sua unidade. O gráfico de elevação no rodapé continua existindo, porque é justamente o que esta ferramenta mostra e as outras não, mas acompanha a mesma limpeza: perde o painel escuro e passa a se destacar do terreno pelo mesmo recurso de contorno que o texto usa, em vez de por um retângulo. Nada disso muda como os valores são calculados, formatados ou arredondados, nem quais unidades usam. Como a quantidade de blocos visíveis varia conforme a escolha do usuário e o que o plano permite, a faixa se arranja com quantos houver, de um só até todos, sem deixar um buraco onde estaria um bloco ausente. As posições e larguras das colunas são decididas uma única vez por voo, nunca quadro a quadro, para que nada escorregue de lado ao longo do vídeo, pela mesma razão que as larguras já foram estabilizadas antes. Os pixels mudam sem o usuário ter pedido, e a escolha de blocos ganha uma opção nova, então a versão de desenho sobe e a impressão digital da configuração de sobreposição passa a contar o bloco novo — quadros antigos nunca se juntam aos novos no mesmo conjunto. A garantia de que as mesmas entradas produzem a mesma imagem byte a byte em qualquer máquina continua valendo. Fora de escopo: acrescentar qualquer valor que o plano de câmera não calcule hoje; acrescentar um título, nome de atividade, marca ou logotipo; mudar o que cada bloco mede, como é calculado ou formatado, a janela de tempo da velocidade e as unidades; mudar o nome da opção com que se escolhem os blocos, ou como ela é informada; deixar o usuário escolher fonte, corpo, cores, posição da faixa ou arranjo; mudar o conteúdo do gráfico de elevação, só sua moldura; mudar as margens de segurança, a fonte embutida ou o rasterizador; e mudar o traçado, o marcador sobre o terreno ou qualquer coisa do desenho do terreno."

## Clarifications

### Session 2026-10-04

- Q: Em que ordem fixa, da esquerda para a direita, os blocos numéricos da faixa do alto devem aparecer quando mais de um estiver presente? → A: Velocidade, Elevação, Distância, Ganho, Tempo decorrido.

## Cenários de Usuário e Testes *(obrigatório)*

### História de Usuário 1 - Ver os números direto sobre a imagem, sem faixas escuras (Prioridade: P1)

Como usuário do Sobrevoo, hoje cada bloco de sobreposição é uma faixa escura empilhada no alto da tela, com texto em caixa alta — peso visual grande para pouca informação, que esconde boa parte da imagem por baixo. Quero que os números apareçam direto sobre a imagem, sem nenhum fundo, lado a lado numa única faixa horizontal no alto do quadro, repartindo a largura entre si em colunas iguais, cada bloco centralizado na própria coluna — e, dentro de cada bloco, o rótulo por extenso (não mais abreviado em caixa alta) pequeno, o valor bem maior embaixo dele, e a unidade pequena por último.

**Por que esta prioridade**: é a mudança central da etapa — todas as demais (novo bloco, nova escolha padrão, moldura do gráfico) só fazem sentido dentro deste novo arranjo visual; sem isso não há "sobreposição mais limpa" nenhuma.

**Teste Independente**: pode ser totalmente testado desenhando um quadro com dois ou mais blocos pedidos (por exemplo, `--overlay-blocks distance,elevation,speed`) e conferindo que: nenhum bloco tem um retângulo de fundo; os blocos pedidos ficam lado a lado, não empilhados; cada um ocupa uma coluna da mesma largura que as demais, centralizado nela; e, dentro de cada bloco, aparecem até três linhas, nessa ordem — rótulo por extenso pequeno, valor grande, unidade pequena —, sem a terceira linha para um bloco sem unidade (tempo decorrido).

**Cenários de Aceitação**:

1. **Dado** um plano e um recorte válidos, **Quando** o usuário desenha um quadro pedindo dois ou mais blocos numéricos, **Então** nenhum deles é desenhado sobre um retângulo ou faixa de fundo — os números aparecem direto sobre a imagem do terreno.
2. **Dado** o mesmo quadro, **Quando** o usuário observa a posição dos blocos pedidos, **Então** eles aparecem lado a lado, em uma única faixa horizontal no alto do quadro, cada um centralizado numa coluna de largura igual à dos demais, nunca empilhados um sobre o outro.
3. **Dado** um bloco numérico qualquer, com unidade (distância, elevação, ganho, velocidade), **Quando** o usuário observa seu conteúdo de cima para baixo, **Então** vê, nessa ordem, o rótulo por extenso em corpo pequeno, o valor em corpo bem maior, e a unidade em corpo pequeno.
4. **Dado** o bloco de tempo decorrido, que não tem unidade, **Quando** o usuário o observa, **Então** vê apenas o rótulo por extenso e o valor — sem uma terceira linha vazia ou espaço reservado para uma unidade que não existe.
5. **Dado** os blocos pedidos, independentemente da ordem em que o usuário os escreveu em `--overlay-blocks`, **Quando** o quadro é desenhado, **Então** eles aparecem sempre nesta ordem — velocidade, elevação, distância, ganho, tempo decorrido —, pulando os que estiverem ausentes, nunca na ordem em que foram escritos.

---

### História de Usuário 2 - Escolher a elevação e o ganho acumulado de forma independente (Prioridade: P2)

Como usuário do Sobrevoo, hoje o ganho de elevação acumulado vem sempre junto do bloco de elevação, como um segundo número colado na mesma linha — não posso pedir um sem o outro. Quero poder escolher o ganho (`gain`) como um bloco independente, pelo mesmo mecanismo com que já escolho os demais, de forma que o bloco de elevação passe a mostrar só a altitude do ponto, e o bloco de ganho, quando pedido, mostre só o ganho acumulado — cada um com seu próprio rótulo e sua própria unidade.

**Por que esta prioridade**: depende do novo arranjo visual (História 1) para fazer sentido como um bloco à parte, mas é a mudança que de fato liberta o ganho de estar sempre amarrado à elevação — um pré-requisito direto para a nova escolha padrão (História 3), que não inclui o ganho.

**Teste Independente**: pode ser totalmente testado pedindo só `elevation`, só `gain`, e os dois juntos, em três desenhos separados do mesmo quadro, e conferindo que cada combinação mostra exatamente os blocos pedidos — elevação sem o número do ganho, ganho sem a altitude, e os dois como blocos distintos quando pedidos juntos.

**Cenários de Aceitação**:

1. **Dado** um plano com elevação disponível, **Quando** o usuário pede só o bloco `elevation`, **Então** o quadro mostra um bloco com a altitude do ponto do marcador e nenhum número de ganho acumulado.
2. **Dado** o mesmo plano, **Quando** o usuário pede só o bloco `gain`, **Então** o quadro mostra um bloco com o ganho de elevação acumulado até o ponto do marcador, com seu próprio rótulo e unidade, e nenhuma altitude.
3. **Dado** o mesmo plano, **Quando** o usuário pede `elevation` e `gain` juntos (por exemplo, `--overlay-blocks elevation,gain`), **Então** o quadro mostra os dois como blocos distintos, cada um na sua coluna.
4. **Dado** um nome de bloco inválido em `--overlay-blocks`, **Quando** o usuário tenta desenhar, **Então** a ferramenta recusa antes de desenhar qualquer quadro, com uma mensagem que lista os blocos aceitos, agora incluindo `gain`.

---

### História de Usuário 3 - Obter, sem pedir nada, o conjunto de blocos que faz sentido num vídeo (Prioridade: P3)

Como usuário do Sobrevoo, hoje ligar as sobreposições sem escolher blocos mostra distância, elevação (com o ganho embutido), tempo decorrido e o gráfico de elevação — cinco números que, empilhados, pesavam na tela. Quero que, sem eu pedir nada além de ligar as sobreposições, o vídeo mostre o conjunto que mais importa enquanto ele roda: velocidade, elevação e distância no alto, e o gráfico de elevação no rodapé — com tempo decorrido e ganho de elevação disponíveis, mas fora da escolha padrão, para quem quiser pedi-los.

**Por que esta prioridade**: é a consequência direta das duas histórias anteriores (o novo arranjo cabe menos blocos confortavelmente lado a lado, e o ganho só pode saber fora do padrão depois de virar um bloco próprio); sem elas, mudar o padrão não teria como ser feito do jeito pedido.

**Teste Independente**: pode ser totalmente testado desenhando um voo só com `--overlays` ligado (sem `--overlay-blocks`) e conferindo que aparecem exatamente velocidade, elevação, distância e o gráfico de elevação — nunca tempo decorrido nem ganho — e, separadamente, pedindo `time` e/ou `gain` explicitamente e conferindo que aparecem quando pedidos.

**Cenários de Aceitação**:

1. **Dado** um plano com elevação e horário de atividade disponíveis, **Quando** o usuário desenha um voo com as sobreposições ligadas e nenhum `--overlay-blocks` informado, **Então** o quadro mostra velocidade, elevação e distância no alto, e o gráfico de elevação no rodapé, sem tempo decorrido nem ganho.
2. **Dado** o mesmo plano, **Quando** o usuário pede explicitamente `time` e/ou `gain` em `--overlay-blocks`, **Então** o bloco pedido aparece, ao lado dos demais blocos ligados.
3. **Dado** um plano sem horário de atividade em todo ponto, **Quando** o usuário desenha um voo com a escolha padrão de blocos, **Então** o bloco de velocidade (que depende de horário) não aparece, sem erro, e os demais blocos padrão (elevação, distância, perfil) continuam aparecendo normalmente.

---

### História de Usuário 4 - Destacar o gráfico de elevação do terreno sem um painel (Prioridade: P4)

Como usuário do Sobrevoo, hoje o gráfico de elevação no rodapé fica sobre um painel escuro retangular, que compete visualmente com o terreno por baixo. Quero que ele acompanhe a mesma limpeza dos demais blocos: sem painel, destacando-se do terreno pelo mesmo contorno escuro que já torna o texto legível sobre qualquer fundo.

**Por que esta prioridade**: é a parte da limpeza visual isolada no gráfico de elevação; pode ser entregue depois das demais porque não depende delas, mas completa a remoção de todos os fundos escuros que a etapa promete.

**Teste Independente**: pode ser totalmente testado desenhando um quadro com o bloco `profile` ligado e conferindo que não há nenhum retângulo de fundo na área do gráfico, e que a linha do perfil e o marcador se distinguem do terreno por um contorno, como o texto já usa.

**Cenários de Aceitação**:

1. **Dado** um plano com elevação disponível, **Quando** o usuário desenha um quadro com o bloco `profile` ligado, **Então** a área do gráfico de elevação não tem nenhum retângulo ou faixa de fundo.
2. **Dado** o mesmo quadro, **Quando** o usuário observa a linha do perfil e o marcador sobre ela, **Então** ambos se destacam do terreno por baixo por um contorno escuro, a mesma técnica que já torna o texto legível — e o conteúdo do gráfico (a forma da linha, a posição do marcador) é o mesmo que seria desenhado antes desta etapa.

---

### Casos Extremos

- O que acontece quando só um bloco numérico está presente (pedido e com dado disponível)? Ele ocupa a faixa inteira, como uma única coluna, centralizado nela — a faixa nunca reserva espaço para colunas que não existem.
- O que acontece quando todos os cinco blocos numéricos (`distance`, `elevation`, `gain`, `time`, `speed`) são pedidos ao mesmo tempo? Os cinco aparecem lado a lado, cada um na sua coluna de largura igual, nesta ordem fixa — velocidade, elevação, distância, ganho, tempo decorrido —, nunca na ordem em que foram escritos em `--overlay-blocks`.
- O que acontece quando um bloco pedido não tem dado disponível no plano (ex.: `speed` ou `time` num trajeto sem horário em todo ponto; `elevation`, `gain` ou `profile` num trajeto sem elevação)? Ele simplesmente não aparece, sem erro, e a faixa se rearranja com os blocos restantes, sem deixar um vão onde ele estaria.
- O que acontece com a largura e a posição das colunas ao longo do vídeo? São calculadas uma única vez por voo, a partir de quantos blocos estarão presentes naquele voo (configuração e dados do plano) — nunca recalculadas quadro a quadro — para que nenhuma coluna se mova de um quadro para o próximo.
- O que acontece quando o usuário desliga as sobreposições por inteiro (`--overlays=false`)? Nenhum bloco é desenhado, incluindo `gain` e o novo arranjo em colunas — como já acontece hoje para qualquer bloco.
- O que acontece com um diretório de quadros já desenhado antes desta etapa (`render all` retomado ou `fly --keep`)? Nunca é reaproveitado junto com quadros desenhados depois desta etapa — a ferramenta recusa a mistura exatamente como já recusa hoje diante de uma aparência ou configuração de sobreposição diferente, sem apagar nada; só redesenha com a sobrescrita pedida explicitamente.
- O que acontece com o bloco de tempo decorrido, que não tem unidade, dentro do novo arranjo de três alturas? Desenha só rótulo e valor — a terceira altura (unidade) simplesmente não existe para ele, sem deixar um espaço vazio em branco no lugar dela.

## Requisitos *(obrigatório)*

### Requisitos Funcionais

- **FR-001**: O sistema DEVE desenhar o valor de cada bloco numérico direto sobre a imagem do quadro, sem nenhuma faixa, painel ou retângulo de fundo atrás dele.
- **FR-002**: Os blocos numéricos presentes em um quadro DEVEM ser desenhados lado a lado, numa única faixa horizontal no alto do quadro — nunca empilhados verticalmente um sobre o outro, como acontecia antes desta etapa.
- **FR-003**: A faixa horizontal DEVE repartir a largura útil do quadro entre os blocos presentes em colunas de mesma largura, com cada bloco centralizado na própria coluna.
- **FR-004**: Os blocos numéricos presentes na faixa do alto DEVEM aparecer sempre nesta ordem fixa, da esquerda para a direita — velocidade, elevação, distância, ganho, tempo decorrido —, pulando os que estiverem ausentes, independentemente da ordem em que o usuário os escreveu em `--overlay-blocks`.
- **FR-005**: Dentro de cada bloco, a informação DEVE ser desenhada de cima para baixo em até três alturas, nesta ordem: o rótulo, por extenso e com capitalização normal (nunca abreviado nem em caixa alta), num corpo pequeno; o valor, num corpo bem maior que o do rótulo; e a unidade, quando o valor tiver uma, de novo num corpo pequeno.
- **FR-006**: Um bloco cujo valor não tem unidade (o tempo decorrido) NÃO DEVE desenhar a terceira altura, e NÃO DEVE deixar um espaço vazio reservado para ela — o bloco ocupa só as duas alturas que tem.
- **FR-007**: A legibilidade do texto sobre qualquer imagem de fundo DEVE continuar dependendo inteiramente do contorno escuro de texto já existente, sem fechar o vão interno das letras nem engrossar o traço do contorno, e sem introduzir nenhum painel, faixa ou fundo novo para compensar a ausência do painel removido.
- **FR-008**: O ganho de elevação acumulado DEVE deixar de ser desenhado junto do bloco de elevação (como um segundo número na mesma linha) e passar a ser um bloco independente, de nome `gain`, selecionável pelo mesmo mecanismo (`--overlay-blocks`) com que o usuário já escolhe os demais blocos.
- **FR-009**: O bloco de elevação (`elevation`) DEVE passar a mostrar somente a altitude do trajeto no ponto do marcador; o bloco de ganho (`gain`) DEVE mostrar somente o ganho acumulado até esse ponto — cada um com seu próprio rótulo e sua própria unidade.
- **FR-010**: `gain` passa a fazer parte do conjunto de nomes de bloco aceitos (`distance`, `elevation`, `gain`, `time`, `profile`, `speed`), e a mensagem de erro para um nome de bloco inválido DEVE listar os seis.
- **FR-011**: A escolha padrão de blocos (sobreposições ligadas, sem `--overlay-blocks` explícito) DEVE passar a ser `distance`, `elevation` e `speed` na faixa do alto, mais `profile` no rodapé — `time` e `gain` DEVEM sair da escolha padrão, continuando disponíveis quando pedidos explicitamente por nome.
- **FR-012**: O gráfico de elevação no rodapé (`profile`) NÃO DEVE mais ser desenhado sobre uma faixa, painel ou retângulo de fundo; ele DEVE se destacar do terreno por baixo pelo mesmo recurso de contorno escuro já usado pelo texto, aplicado à sua linha e ao seu marcador.
- **FR-013**: Nenhum valor mostrado por qualquer bloco (incluindo o conteúdo do gráfico de elevação — sua linha e seu marcador) DEVE mudar em como é calculado, formatado, arredondado, ou em qual unidade usa, em relação a antes desta etapa.
- **FR-014**: Quando a quantidade de blocos numéricos presentes em um voo variar (por escolha do usuário ou por disponibilidade de dado no plano), a faixa horizontal DEVE se arranjar com exatamente essa quantidade de colunas, de uma só até todas as possíveis, sem deixar um vão no lugar de um bloco ausente.
- **FR-015**: As posições e as larguras das colunas da faixa horizontal DEVEM ser decididas uma única vez por voo, a partir de quantos blocos estarão presentes nesse voo — nunca recalculadas quadro a quadro —, de forma que nenhuma coluna mude de posição ou largura entre o primeiro e o último quadro do mesmo voo.
- **FR-016**: Como um quadro com sobreposições ligadas passa a ter pixels diferentes dos de antes desta etapa, a versão de desenho DEVE subir, e a impressão digital da configuração de sobreposição (usada para identificar um conjunto de quadros) DEVE passar a contar também o novo bloco `gain` — um conjunto de quadros desenhado antes desta etapa nunca é reaproveitado junto de um desenhado depois.
- **FR-017**: A garantia de que as mesmas entradas (plano, recorte, aparência, configuração de sobreposição) produzem sempre a mesma imagem, byte a byte, em qualquer máquina, DEVE continuar valendo para o novo arranjo.
- **FR-018**: Nada nesta etapa DEVE acrescentar um valor que o plano de câmera não calcule hoje; um título, nome de atividade, marca ou logotipo; nem mudar o que cada bloco mede, como é calculado ou formatado, a janela de tempo usada pelo bloco de velocidade, as unidades usadas por qualquer bloco, o nome da opção com que se escolhem os blocos ou como ela é informada, o conteúdo do gráfico de elevação (só sua moldura muda), as margens de segurança, a fonte embutida, o rasterizador de texto, o traçado, o marcador sobre o terreno, ou qualquer parte do desenho do terreno — nem DEVE deixar o usuário escolher fonte, corpo, cores, posição da faixa ou arranjo das colunas.

### Entidades-Chave

- **Faixa de Blocos**: a região horizontal no alto do quadro onde os blocos numéricos presentes são desenhados lado a lado, nesta ordem fixa — velocidade, elevação, distância, ganho, tempo decorrido —, repartida em colunas de mesma largura — quantas colunas houver blocos presentes naquele voo, de uma até todas.
- **Coluna**: o espaço horizontal igual que cada bloco presente ocupa dentro da faixa, dentro do qual o bloco (suas três alturas de texto) é centralizado; sua posição e largura são as mesmas em todo quadro do mesmo voo.
- **Bloco de Ganho (`gain`)**: bloco novo, independente do bloco de elevação, que mostra somente o ganho de elevação acumulado até o ponto do marcador, com rótulo e unidade próprios — antes desta etapa, esse valor só existia colado ao bloco de elevação.
- **Bloco de Elevação (`elevation`, atualizado)**: passa a mostrar somente a altitude do trajeto no ponto do marcador, sem o ganho acumulado que antes vinha junto.
- **Gráfico de Elevação (`profile`, moldura atualizada)**: o mesmo gráfico de sempre — a linha do perfil de elevação do trajeto inteiro e o marcador que avança com o voo —, agora destacado do terreno por contorno em vez de por um painel de fundo.

## Critérios de Sucesso *(obrigatório)*

### Resultados Mensuráveis

- **SC-001**: Em qualquer quadro com sobreposições ligadas, nenhum bloco numérico nem o gráfico de elevação é desenhado sobre um retângulo, faixa ou painel de fundo — só o contorno escuro de texto/linha separa cada um do terreno.
- **SC-002**: De 1 a 5 blocos numéricos presentes num voo aparecem sempre lado a lado, em colunas de mesma largura, na mesma ordem fixa, sem vão para um bloco ausente, qualquer que seja a combinação pedida.
- **SC-003**: Um usuário consegue pedir o bloco de elevação e o de ganho de forma totalmente independente — só um, só o outro, os dois, ou nenhum — e obtém exatamente os blocos pedidos.
- **SC-004**: Um voo desenhado só com `--overlays` ligado, sem `--overlay-blocks`, mostra exatamente velocidade, elevação e distância no alto e o gráfico de elevação no rodapé — nunca tempo decorrido nem ganho.
- **SC-005**: As colunas da faixa horizontal nunca mudam de posição ou largura entre o primeiro e o último quadro do mesmo voo.
- **SC-006**: Nenhum valor exibido por qualquer bloco, incluindo o gráfico de elevação, muda em como é calculado, formatado, arredondado, ou em que unidade usa, em relação a antes desta etapa.
- **SC-007**: A mesma combinação de plano, recorte, aparência e configuração de sobreposição produz sempre a mesma imagem, byte a byte, em qualquer máquina.
- **SC-008**: Um conjunto de quadros desenhado antes desta etapa nunca é aceito como parte de um conjunto desenhado depois dela — `render all` sem `--overwrite` recusa a mistura, e `fly --keep` redesenha em vez de reaproveitar.

## Suposições

- O texto de cada bloco (rótulo por extenso, valor, unidade) continua em português do Brasil, seguindo a mesma convenção de rótulos e abreviação de unidade já estabelecida nas etapas anteriores de sobreposição — só a forma do rótulo muda (de abreviado em caixa alta para por extenso, capitalização normal), não a língua nem as abreviações de unidade.
- Quando o texto de um bloco for mais largo que a coluna que lhe caberia (por exemplo, com muitos blocos pedidos ao mesmo tempo numa resolução pequena), o bloco continua sendo desenhado por inteiro, centralizado no centro da própria coluna — a coluna define o centro e a largura-alvo do bloco para fins de distribuição do espaço, não um limite que recorta o texto; como não há mais um painel marcando a borda de cada bloco, um leve excesso sobre o espaço vizinho não compromete a leitura.
- Os corpos de letra exatos do rótulo, do valor e da unidade (as três alturas) são uma decisão de planejamento técnico a ser registrada quando definida; o pedido original só exige que o valor seja visivelmente maior que o rótulo e a unidade, e que ambos sejam pequenos.
- `inspect`, `geodata check`, `video` e a montagem do vídeo não são afetados: o arranjo das sobreposições é só uma questão de como o desenho de quadros (`render frame`, `render all`, `fly`) apresenta valores que o plano de câmera já calcula.
