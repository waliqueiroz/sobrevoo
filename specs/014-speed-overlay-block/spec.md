# Especificação de Funcionalidade: Bloco de Velocidade na Sobreposição

**Branch da Funcionalidade**: `014-speed-overlay-block`

**Criado em**: 2026-10-04

**Status**: Rascunho

**Entrada**: Descrição do usuário: "Os blocos de sobreposição mostram hoje distância, elevação, ganho e o gráfico de elevação, mas não a velocidade — que é, para quem pedala ou corre, a informação que mais muda enquanto o vídeo roda e a primeira que se procura num quadro. A partir desta etapa o plano de câmera passa a calcular, para cada quadro, a velocidade da atividade no ponto onde o marcador está, e a sobreposição ganha um bloco que a mostra, escolhido pelo mesmo mecanismo com que o usuário já escolhe os demais blocos, pelo nome `speed`. Nesta etapa ele nasce fora da escolha padrão: só aparece para quem pedir. A velocidade é derivada do que o trajeto já informa — a distância percorrida e o horário de cada ponto —, então só existe quando o trajeto traz horários em todos os seus pontos; num trajeto sem horários, em que o marcador avança por distância e não por relógio, não há velocidade que se possa afirmar, e o bloco não é desenhado, exatamente como o bloco de tempo decorrido já não é nesse caso. O valor mostrado não é a velocidade entre dois pontos consecutivos do GPS, que oscila demais para ser lida: poucos metros de erro de posição viram dezenas de quilômetros por hora de diferença, e o número ficaria tremendo quadro a quadro. É a velocidade média numa janela de tempo da atividade em torno do ponto do marcador, de duração fixa e documentada, grande o bastante para o número ficar estável e pequena o bastante para ainda acompanhar uma subida ou uma descida — a mesma janela em todo o vídeo e em qualquer trajeto, nunca ajustável pelo usuário. Nos extremos do trajeto, onde a janela não cabe inteira, ela é encurtada para o que existe, em vez de o valor desaparecer ou saltar. A velocidade passa a fazer parte do conteúdo do plano, é gravada no plano exportado junto dos demais valores de cada quadro, e entra na identidade do plano como tudo o que o plano carrega — dois planos iguais em tudo menos na velocidade calculada são planos diferentes. Como o plano ganha informação que um plano anterior a esta etapa não tem, e não há valor implícito que se possa supor para a ausência dela, o formato do arquivo de plano sobe de versão, e um plano mais antigo é recusado com a mesma mensagem e o mesmo código de saída com que um plano de versão desconhecida já é hoje, pedindo que seja gerado de novo. A garantia de que as mesmas entradas produzem o mesmo plano e a mesma imagem, byte a byte, em qualquer máquina continua valendo, e o cálculo obedece à mesma disciplina aritmética do resto do planejamento. Fora de escopo: mudar a escolha padrão de blocos, o arranjo, a tipografia, os painéis ou os rótulos da sobreposição, tudo isso reservado para a etapa seguinte; deixar o usuário escolher a janela, a unidade ou o arredondamento da velocidade; mostrar velocidade máxima, média do percurso inteiro, ritmo em minutos por quilômetro, ou qualquer outra métrica derivada; usar a velocidade para alterar o movimento da câmera, a duração do vídeo ou o avanço do marcador; e mudar o tratamento do trajeto, o recorte, o desenho do terreno ou o traçado."

## Cenários de Usuário e Testes *(obrigatório)*

### História de Usuário 1 - Ver a velocidade estável da atividade sobre o voo (Prioridade: P1)

Como usuário do Sobrevoo que pedala ou corre, a informação que mais me interessa enquanto o vídeo roda é a velocidade — hoje ausente das sobreposições, que só mostram distância, elevação, ganho e tempo. Quero poder pedir, com o mesmo mecanismo com que já escolho quais blocos aparecem, um bloco de velocidade (`speed`) que mostre, em cada quadro, a velocidade da atividade no ponto onde o marcador está — um número estável, que acompanha subidas e descidas sem tremer quadro a quadro por causa do ruído normal de um GPS.

**Por que esta prioridade**: é o valor central da etapa — sem isso, nada mais nesta funcionalidade tem sentido; as demais garantias (plano, identidade, versão) só existem para que este bloco seja confiável e nunca se confunda entre execuções.

**Teste Independente**: pode ser totalmente testado desenhando um voo de um trajeto com horário em todo ponto, pedindo explicitamente o bloco `speed`, e conferindo que todo quadro mostra um valor de velocidade que corresponde à distância percorrida pela atividade dividida pelo tempo decorrido numa janela em torno daquele instante — nunca a velocidade instantânea entre dois pontos consecutivos do GPS.

**Cenários de Aceitação**:

1. **Dado** um plano e um recorte válidos de um trajeto com horário em todos os pontos, **Quando** o usuário desenha um quadro ou o voo inteiro pedindo o bloco `speed` (por exemplo, `--overlay-blocks distance,speed`), **Então** o quadro mostra a velocidade média da atividade numa janela de tempo fixa em torno do ponto do marcador, ao lado dos demais blocos pedidos.
2. **Dado** um voo assim desenhado, **Quando** o usuário observa a sequência de valores de velocidade ao longo dos quadros, **Então** o valor muda suavemente com o perfil da atividade (acompanhando subidas e descidas), sem os saltos bruscos que a velocidade entre dois pontos de GPS consecutivos teria.
3. **Dado** o mesmo plano e o mesmo recorte, **Quando** o usuário desenha o mesmo quadro mais de uma vez, **Então** o valor de velocidade mostrado é exatamente o mesmo em todas as vezes.

---

### História de Usuário 2 - Nunca confundir planos com velocidades diferentes (Prioridade: P2)

Como usuário do Sobrevoo, sei que o plano de câmera pode ser exportado, guardado e reaproveitado entre execuções (`fly --keep`). Quero que um plano que já carrega a velocidade calculada para cada quadro nunca seja confundido com um plano gerado antes de esta funcionalidade existir, nem com outro plano do mesmo trajeto cuja velocidade tenha sido calculada de outro jeito — e que um plano antigo, sem essa informação, seja recusado de forma clara, pedindo para ser gerado de novo, em vez de produzir um vídeo sem sentido ou travar de um jeito confuso.

**Por que esta prioridade**: é a rede de segurança da funcionalidade — sem ela, um plano de antes desta etapa poderia ser lido como se já trouxesse velocidade (um valor inventado ou zerado, nunca avisado), e um reaproveitamento de `--keep` poderia misturar, no mesmo vídeo, quadros com velocidades calculadas de forma diferente. Depende da História 1 já existir.

**Teste Independente**: pode ser totalmente testado tentando ler, com `render frame`, `render all` ou `fly`, um arquivo de plano exportado por uma versão da ferramenta anterior a esta etapa, e confirmando a recusa antes de qualquer quadro ser desenhado; e, separadamente, exportando dois planos do mesmo trajeto que seriam idênticos exceto pela velocidade calculada, e confirmando que a ferramenta os trata como planos diferentes.

**Cenários de Aceitação**:

1. **Dado** um arquivo de plano exportado por uma versão da ferramenta anterior a esta etapa, **Quando** o usuário informa esse arquivo a `render frame`, `render all` ou `fly`, **Então** a ferramenta recusa, antes de desenhar qualquer quadro, com a mesma mensagem e o mesmo código de saída que já usa hoje para um plano de versão de formato desconhecida, pedindo que o plano seja gerado de novo.
2. **Dado** um plano de câmera exportado por esta versão da ferramenta, **Quando** o usuário o lê de volta com `render frame`, `render all` ou `fly`, **Então** a velocidade de cada quadro lida do arquivo é exatamente a mesma que o plano tinha em memória antes de ser exportado.
3. **Dado** dois planos do mesmo trajeto e dos mesmos demais parâmetros, **Quando** a velocidade calculada por quadro difere entre eles (por qualquer razão), **Então** a ferramenta os reconhece como planos diferentes, e um `fly --keep` que guardou um deles nunca reaproveita plano, recorte ou quadros do outro.

---

### Casos Extremos

- O que acontece quando o trajeto não tem horário em todos os pontos (o marcador avança por distância, não por relógio)? O bloco `speed`, se pedido, não é desenhado — sem erro, exatamente como o bloco de tempo decorrido já não é desenhado nesse caso hoje. Os demais blocos pedidos continuam normais.
- O que acontece nas extremidades do trajeto (bem no início ou no fim), onde a janela de tempo usada para calcular a média não cabe inteira para um dos lados? A janela é encurtada para o que existe daquele lado, e a velocidade continua sendo mostrada — nunca desaparece nem salta para um valor muito diferente do quadro vizinho só por estar na borda.
- O que acontece quando o usuário liga as sobreposições (`--overlays`) sem pedir explicitamente o bloco `speed` em `--overlay-blocks`? O bloco `speed` não aparece — a escolha padrão de blocos continua sendo a de hoje (distância, elevação e ganho, tempo decorrido, perfil de elevação), sem o bloco novo.
- O que acontece quando o usuário desliga as sobreposições por inteiro (`--overlays=false`) mesmo tendo pedido `speed` em `--overlay-blocks`? Nenhum bloco é desenhado, incluindo `speed` — desligar por inteiro continua prevalecendo sobre a lista de blocos, como já acontece hoje.
- O que acontece quando o usuário pede `speed` para um trajeto sem dado de horário, mas pede também outros blocos (por exemplo, distância)? O pedido é aceito normalmente; só o bloco `speed` fica ausente daquele voo, os demais blocos pedidos aparecem — a ausência é do dado, não do pedido, exatamente como já acontece para os blocos existentes hoje.
- O que acontece quando um diretório de quadros (`render all` retomado ou `fly --keep`) tem quadros desenhados antes de o usuário passar a pedir o bloco `speed`? A ferramenta recusa reaproveitar esses quadros exatamente como já recusa hoje diante de uma configuração de sobreposição diferente da pedida agora, sem apagar nada; só redesenha com a sobrescrita pedida explicitamente.

## Requisitos *(obrigatório)*

### Requisitos Funcionais

- **FR-001**: O sistema DEVE calcular, para cada quadro do plano de câmera, a velocidade da atividade no ponto onde o marcador está, derivada exclusivamente da distância percorrida e dos horários já presentes no trajeto.
- **FR-002**: A velocidade calculada DEVE ser a média numa janela de tempo da atividade, de duração fixa e documentada, centrada no instante do marcador naquele quadro — nunca a velocidade instantânea entre dois pontos consecutivos do trajeto.
- **FR-003**: A duração da janela usada no cálculo DEVE ser a mesma em todo o vídeo e em qualquer trajeto, e não DEVE ser ajustável pelo usuário.
- **FR-004**: Nos extremos do trajeto, onde a janela completa não cabe de um dos lados do instante do marcador, o sistema DEVE encurtar a janela para o trecho de atividade que existe daquele lado, em vez de deixar de calcular a velocidade ou produzir um salto abrupto em relação ao quadro vizinho.
- **FR-005**: A velocidade por quadro só DEVE ser calculada quando o trajeto traz horário em todos os seus pontos — o mesmo critério que já determina se o trajeto tem tempo real de atividade; quando o trajeto não atende esse critério, a velocidade não existe para nenhum quadro daquele plano.
- **FR-006**: O sistema DEVE acrescentar um bloco de sobreposição novo, de nome `speed`, selecionável pelo mesmo mecanismo (a mesma opção de linha de comando) com que o usuário já escolhe os demais blocos em `render frame`, `render all` e `fly`.
- **FR-007**: O bloco `speed` NÃO DEVE fazer parte da escolha padrão de blocos: ligar as sobreposições sem pedir blocos explicitamente, ou pedir blocos sem incluir `speed` por nome, NUNCA desenha esse bloco.
- **FR-008**: O bloco `speed` só DEVE ser desenhado quando explicitamente pedido por nome e quando o trajeto do plano tem velocidade calculada (FR-005); caso contrário, pedi-lo é aceito sem erro e o bloco simplesmente não aparece naquele voo — exatamente como já acontece hoje para um bloco existente cujo dado está ausente.
- **FR-009**: Desligar as sobreposições por inteiro DEVE continuar desenhando nenhum bloco, incluindo `speed`, mesmo que ele tenha sido pedido explicitamente em `--overlay-blocks`.
- **FR-010**: O formato do plano de câmera exportado DEVE passar a guardar, por quadro, a velocidade calculada, junto dos demais valores que já guarda — um campo novo, sem mudar o significado de nenhum campo hoje existente.
- **FR-011**: A velocidade calculada por quadro DEVE fazer parte do que identifica um plano de câmera: dois planos do mesmo trajeto e dos mesmos demais parâmetros, mas cuja velocidade calculada por quadro difira, DEVEM ser sempre reconhecidos como planos diferentes, nunca equivalentes — inclusive para a decisão de reaproveitar um plano guardado por `fly --keep`.
- **FR-012**: O formato do arquivo de plano exportado DEVE subir de versão, porque o plano passa a conter uma informação que um plano de versão anterior não tem, sem valor implícito que se possa supor para a ausência dela.
- **FR-013**: Um arquivo de plano de uma versão de formato anterior a esta etapa DEVE ser recusado por `render frame`, `render all` e `fly`, com a mesma mensagem e o mesmo código de saída de processo com que a ferramenta já recusa hoje um plano de versão de formato desconhecida, orientando o usuário a gerar o plano de novo.
- **FR-014**: Um nome de bloco de sobreposição inválido continua sendo recusado antes de desenhar qualquer quadro, com uma mensagem que lista os blocos aceitos — a lista passa a incluir `speed`.
- **FR-015**: O cálculo da velocidade DEVE obedecer à mesma disciplina aritmética do resto do planejamento de câmera, de forma que o mesmo plano produza sempre a mesma velocidade por quadro, e a mesma imagem byte a byte, em qualquer máquina.
- **FR-016**: Nada nesta etapa pode alterar a escolha padrão de blocos existente, o arranjo, a tipografia, os painéis ou os rótulos da sobreposição, o movimento da câmera, a duração do vídeo, o avanço do marcador, o tratamento do trajeto, o recorte, o desenho do terreno ou o traçado.

### Entidades-Chave

- **Bloco de Velocidade (`speed`)**: um quinto bloco de sobreposição, independente dos quatro já existentes (distância; elevação e ganho; tempo decorrido; perfil de elevação), que mostra a velocidade da atividade no ponto do marcador. Diferente dos quatro já existentes, nasce desligado por padrão — só aparece quando pedido explicitamente por nome.
- **Velocidade por Quadro**: o valor novo que o plano de câmera passa a guardar para cada quadro — a velocidade média da atividade numa janela de tempo fixa em torno do instante do marcador, derivada da distância percorrida e dos horários do trajeto, sem nenhuma leitura nova do trajeto GPS. Passa a integrar o conteúdo do plano, o arquivo de plano exportado e a identidade do plano.
- **Janela de Velocidade**: a duração fixa, igual em todo vídeo e em qualquer trajeto, usada para calcular a velocidade média de cada quadro; encurtada, nunca ausente, nos extremos do trajeto onde não cabe inteira.

## Critérios de Sucesso *(obrigatório)*

### Resultados Mensuráveis

- **SC-001**: Um usuário consegue produzir um vídeo mostrando a velocidade da atividade, pedindo o bloco `speed`, sem alterar nenhum outro aspecto do vídeo (câmera, enquadramento, terreno, traçado, duração, demais blocos).
- **SC-002**: Um usuário que não pede o bloco `speed` obtém um vídeo idêntico, pixel a pixel, ao que a ferramenta produzia antes desta funcionalidade existir, para o mesmo plano, recorte, resolução, aparência e demais blocos de sobreposição.
- **SC-003**: Desenhar o mesmo quadro duas vezes com o mesmo plano produz sempre o mesmo valor de velocidade, byte a byte.
- **SC-004**: Um trajeto sem horário em todos os pontos nunca mostra o bloco `speed`, mesmo quando pedido, e nunca produz um erro por causa disso.
- **SC-005**: Todo arquivo de plano de câmera gerado por uma versão da ferramenta anterior a esta etapa é recusado, com uma mensagem que orienta a gerá-lo de novo, antes de qualquer quadro ser desenhado.
- **SC-006**: Dois planos do mesmo trajeto e dos mesmos demais parâmetros, cuja velocidade calculada por quadro difira entre eles, nunca são tratados como o mesmo plano por um `fly --keep` — o plano, o recorte e os quadros são recalculados sempre que a velocidade mudar.

## Suposições

- A unidade em que a velocidade é exibida é métrica (quilômetros por hora, seguindo a mesma convenção métrica de distância e elevação já usada pelas demais sobreposições) e fixa, escolhida no planejamento técnico — não é ajustável pelo usuário nesta etapa.
- A duração exata da janela de tempo usada para calcular a média, e a forma exata como ela é encurtada nos extremos do trajeto, são decisões de planejamento técnico, documentadas quando definidas; o pedido original só exige que a janela seja fixa, documentada, igual em todo vídeo e trajeto, e grande o bastante para estabilizar o valor sem deixar de acompanhar uma subida ou descida.
- O formato de exibição do valor (casas decimais, arredondamento) é uma decisão de planejamento técnico, documentada quando definida — escolher esse formato está fora de escopo para o usuário nesta etapa.
- A posição, o tamanho, a tipografia e o rótulo do bloco `speed` na tela seguem o mesmo padrão visual que os quatro blocos já existentes estabelecem hoje; refinamentos de arranjo, tipografia ou rótulos ficam para uma etapa seguinte, como o pedido original define.
- `inspect`, `geodata check`, `video` e a montagem do vídeo não são afetados: a velocidade é um dado do plano de câmera, consumido apenas pelo desenho de quadros (`render frame`, `render all`, `fly`).
- Nem velocidade máxima, nem velocidade média do percurso inteiro, nem ritmo em minutos por quilômetro, nem nenhuma outra métrica derivada de velocidade fazem parte desta etapa — permanecem fora de escopo, como o pedido original define.
