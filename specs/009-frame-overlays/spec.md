# Especificação de Funcionalidade: Sobreposições de Tela nos Quadros

**Branch da Funcionalidade**: `009-frame-overlays`

**Criado em**: 2026-09-28

**Status**: Implementada, com `quickstart.md` validado por inteiro com o `ffmpeg` real

**Entrada**: Descrição do usuário: "Nona etapa do Sobrevoo. Esta etapa acrescenta ao vídeo as sobreposições de tela: os números da atividade desenhados por cima da imagem, e não colados no terreno. Hoje o vídeo mostra o voo, o traçado e o marcador, mas nada diz quanto foi percorrido, quanto se subiu ou quanto tempo durou — é o que separa um sobrevoo bonito de um resumo de atividade. Com esta etapa, cada quadro passa a exibir, em posição fixa na tela, a distância percorrida até aquele instante, a elevação do trajeto naquele ponto e o ganho acumulado, o tempo decorrido da atividade, e um perfil de elevação do trajeto inteiro com um marcador andando sobre ele conforme o voo avança. O usuário consegue: gerar o vídeo com as sobreposições ligadas por padrão; desligá-las por inteiro quando quiser só o voo; escolher quais blocos aparecem; e conferir o resultado num quadro isolado antes de desenhar o voo inteiro, como já faz com a aparência. Requisitos: os valores exibidos DEVEM vir exclusivamente do plano de câmera e do recorte informados — nenhuma leitura nova do trajeto GPS nem dos dados registrados —, o que exige que o plano passe a guardar, por quadro, o instante real da atividade e a elevação do trajeto no ponto do marcador, acrescentados ao formato do plano sem mudar o significado de nenhum campo existente e com um plano antigo, sem esses campos, recusado com mensagem que diz para gerá-lo de novo; o texto DEVE ser desenhado com uma fonte embutida na própria ferramenta, sem depender de nenhuma fonte instalada no sistema, para que o mesmo plano produza o mesmo quadro em qualquer máquina; as sobreposições DEVEM respeitar uma margem de segurança documentada nas quatro bordas, porque as redes sociais cortam as extremidades do vídeo vertical; DEVEM permanecer legíveis sobre qualquer fundo, claro ou escuro; a configuração de sobreposição escolhida DEVE fazer parte da identidade do conjunto de quadros, do mesmo modo que a aparência, de maneira que quadros com sobreposições diferentes nunca sejam tomados como do mesmo conjunto numa retomada ou num reaproveitamento; o mesmo plano, o mesmo recorte, a mesma resolução, a mesma aparência e a mesma configuração de sobreposição DEVEM continuar produzindo imagens idênticas byte a byte; as unidades DEVEM ser métricas e documentadas, e os valores DEVEM coincidir, ao final do voo, com o que o inspect reporta para o mesmo trajeto; e nada nesta etapa pode alterar a câmera, o enquadramento, o terreno, o traçado ou a duração do vídeo. Fora de escopo: áudio, música, escolha de fonte pelo usuário, logotipos e marcas, textos livres digitados pelo usuário, animações de entrada e saída dos blocos, gráficos interativos, e qualquer mudança nas regras de trajeto, câmera, dados geográficos ou codificação."

## Clarifications

### Session 2026-09-28

- Q: Quando o usuário escolhe quais blocos de sobreposição aparecem, em que granularidade ele controla isso — os quatro blocos assumidos (distância | elevação+ganho | tempo | perfil), um só bloco de "estatísticas" mais o perfil separado, ou os cinco valores de forma totalmente independente? → A: Quatro blocos: distância percorrida; elevação do trajeto e ganho acumulado (juntos); tempo decorrido; perfil de elevação — a leitura mais literal do pedido original, que já junta elevação e ganho numa só cláusula.

## Cenários de Usuário e Testes *(obrigatório)*

### História de Usuário 1 - Ver as estatísticas da atividade sobre o voo (Prioridade: P1)

Como usuário do Sobrevoo, hoje o vídeo mostra o terreno, o traçado e o marcador, mas nada diz quanto foi percorrido, quanto se subiu ou quanto tempo a atividade durou — um sobrevoo bonito, mas mudo sobre a própria atividade. Quero que, por padrão, cada quadro do vídeo mostre, em posição fixa na tela (não colado no terreno, para não girar ou inclinar com a câmera), a distância percorrida até aquele instante, a elevação do trajeto e o ganho acumulado naquele ponto, o tempo decorrido da atividade, e um perfil de elevação do trajeto inteiro com um marcador que anda sobre ele conforme o voo avança.

**Por que esta prioridade**: é o valor central da etapa — sem isso, nada mais nesta funcionalidade tem sentido; as demais histórias só garantem que essas sobreposições sejam controláveis, verificáveis com baixo custo e nunca confundidas entre execuções.

**Teste Independente**: pode ser totalmente testado gerando o voo inteiro de um trajeto conhecido, sem informar nada sobre sobreposições, e conferindo que todo quadro traz as cinco informações em posição fixa, e que os valores ao final do voo (distância total, ganho acumulado, tempo decorrido) coincidem com o que `inspect` relata para o mesmo trajeto.

**Cenários de Aceitação**:

1. **Dado** um plano e um recorte válidos de um trajeto com dado de tempo e de elevação, **Quando** o usuário desenha o voo inteiro sem informar nada sobre sobreposições, **Então** cada quadro mostra a distância percorrida até aquele instante, a elevação do trajeto e o ganho acumulado naquele ponto, o tempo decorrido da atividade, e o perfil de elevação com o marcador na posição correspondente.
2. **Dado** um voo assim gerado, **Quando** o usuário observa o valor de distância, de ganho acumulado e de tempo decorrido no último quadro, **Então** esses valores coincidem, exatamente, com o que `inspect` relata para o mesmo trajeto (distância total, ganho de elevação, duração).
3. **Dado** um voo assim gerado, **Quando** a câmera gira, inclina ou muda de posição entre um quadro e outro, **Então** as sobreposições continuam no mesmo lugar da tela, sem acompanhar esse movimento — só o conteúdo dos números muda, nunca sua posição.

---

### História de Usuário 2 - Conferir a sobreposição num quadro isolado antes do voo inteiro (Prioridade: P2)

Como usuário do Sobrevoo, sei que desenhar o voo inteiro ou montar o vídeo final pode levar bastante tempo. Antes de me comprometer com uma configuração de sobreposição para o vídeo inteiro, quero desenhar um único quadro com os valores que estou considerando, olhar o resultado, e só então decidir se aplico a mesma configuração ao voo inteiro — com a certeza de que o quadro isolado mostra exatamente o que o voo inteiro vai mostrar.

**Por que esta prioridade**: sem isso, ajustar a legibilidade ou a composição das sobreposições custaria o tempo de um voo inteiro a cada tentativa; depende da História 1 já existir. É a mesma necessidade que já levou a etapa anterior a garantir isso para a aparência.

**Teste Independente**: pode ser totalmente testado desenhando um quadro isolado com uma configuração de sobreposição escolhida, depois desenhando o voo inteiro (ou o mesmo quadro dentro dele) com a mesma configuração, e comparando as duas imagens desse quadro byte a byte.

**Cenários de Aceitação**:

1. **Dado** que o usuário desenhou um quadro isolado com uma configuração de sobreposição escolhida, **Quando** ele desenha o voo inteiro (ou gera o vídeo com o comando único) usando a mesma configuração, **Então** o quadro correspondente do voo inteiro é idêntico, byte a byte, ao quadro isolado.
2. **Dado** que o usuário não gostou do resultado de um quadro isolado, **Quando** ele desenha outro quadro isolado com uma configuração diferente, **Então** só precisa esperar o tempo de desenhar um quadro, não o de um voo inteiro, para ver o novo resultado.

---

### História de Usuário 3 - Desligar tudo ou escolher só alguns blocos (Prioridade: P3)

Como usuário do Sobrevoo, às vezes quero só o voo, sem nenhum número sobre a tela — por exemplo, para usar o vídeo como fundo de algo, ou porque a atividade não tem um dado que valha a pena mostrar. Outras vezes quero mostrar só parte da informação — por exemplo, a distância e o tempo, mas não o perfil de elevação, porque a subida foi pequena e o gráfico não acrescenta nada. Quero poder desligar as sobreposições por inteiro, ou escolher exatamente quais blocos aparecem, sem precisar aceitar o pacote completo ou nada.

**Por que esta prioridade**: dá ao usuário o controle que falta depois que a História 1 já entrega o resultado padrão; sem ela, a funcionalidade seria tudo ou nada, incluindo em casos em que um bloco não faz sentido para a atividade.

**Teste Independente**: pode ser totalmente testado desenhando um quadro com as sobreposições desligadas por inteiro (conferindo que não sobra nenhum pixel de texto ou gráfico sobre a imagem) e, separadamente, desenhando quadros com diferentes subconjuntos de blocos ligados, conferindo que só os blocos pedidos aparecem.

**Cenários de Aceitação**:

1. **Dado** um plano e um recorte válidos, **Quando** o usuário desenha um quadro pedindo que as sobreposições fiquem desligadas, **Então** o quadro resultante é idêntico, pixel a pixel, ao que a ferramenta produziria sem nenhuma sobreposição (mesmo voo, traçado e marcador de hoje, sem nenhum texto ou gráfico por cima).
2. **Dado** um plano e um recorte válidos, **Quando** o usuário pede que só alguns dos blocos (por exemplo, distância e tempo decorrido, sem o perfil de elevação) apareçam, **Então** o quadro mostra exatamente esses blocos e nenhum outro.
3. **Dado** que o usuário pede um nome de bloco que não existe, **Quando** ele tenta desenhar um quadro ou o voo inteiro, **Então** a ferramenta recusa antes de desenhar qualquer coisa, com uma mensagem que lista os blocos aceitos.

---

### História de Usuário 4 - Nunca confundir quadros com sobreposições diferentes (Prioridade: P4)

Como usuário do Sobrevoo, às vezes interrompo um `render all` no meio e retomo depois, ou guardo os intermediários de um `fly` para reaproveitar numa execução seguinte. Se eu mudar a configuração de sobreposição entre uma execução e a próxima, quero que a ferramenta perceba a mudança e redesenhe os quadros com a nova configuração, em vez de misturar, no mesmo vídeo, quadros com números e quadros sem eles, ou quadros com blocos diferentes; e quando eu não mudo nada, quero que ela continue reaproveitando o que já está pronto.

**Por que esta prioridade**: é a rede de segurança da funcionalidade — sem ela, um vídeo poderia sair com metade dos quadros exibindo a distância percorrida e a outra metade não; depende das Histórias 1 e 3 já existirem, e só importa para quem usa retomada ou reaproveitamento, um subconjunto de uso mais avançado.

**Teste Independente**: pode ser totalmente testado desenhando parte de um voo com uma configuração de sobreposição, mudando a configuração e desenhando o resto (ou reaproveitando um `fly --keep` anterior com uma configuração diferente), e conferindo que a ferramenta recusa por padrão a mistura e redesenha por completo quando pedido, e que, sem mudar a configuração, uma nova execução reaproveita os quadros já prontos.

**Cenários de Aceitação**:

1. **Dado** um `render all` interrompido no meio com uma configuração de sobreposição, **Quando** o usuário o retoma pedindo uma configuração diferente, **Então** a ferramenta recusa por padrão, avisando que os quadros já lá são de outra configuração, e só redesenha com a sobrescrita pedida explicitamente.
2. **Dado** um `fly --keep <diretório>` já executado com uma configuração de sobreposição, **Quando** o usuário roda de novo apontando para o mesmo diretório, com o mesmo trajeto e os mesmos outros valores, mas com uma configuração de sobreposição diferente, **Então** o plano de câmera e o recorte de dados geográficos são reaproveitados, mas os quadros e o vídeo são refeitos com a nova configuração.
3. **Dado** o mesmo cenário do item anterior, mas sem nenhuma mudança na configuração de sobreposição, **Quando** o usuário roda de novo, **Então** os quadros e o vídeo também são reaproveitados, sem nenhum trabalho refeito.

---

### Casos Extremos

- O que acontece quando o usuário informa, para `render frame`, `render all` ou `fly`, um plano de câmera exportado por uma versão da ferramenta anterior a esta etapa (sem o instante real da atividade e sem a elevação do trajeto por quadro)? A ferramenta recusa antes de desenhar qualquer quadro, com uma mensagem que diz que esse plano precisa ser gerado de novo.
- O que acontece quando o trajeto não tem dado de tempo real (o plano foi gerado a partir da distância, não do relógio do GPS)? O bloco de tempo decorrido não é mostrado — como não existe instante real da atividade para esse trajeto, a ferramenta não inventa um; os demais blocos continuam normais, exatamente como `inspect` já reporta a duração como ausente para esse mesmo trajeto.
- O que acontece quando o trajeto não tem dado de elevação? Os blocos que dependem de elevação (elevação atual, ganho acumulado e o perfil de elevação) não são mostrados; os demais blocos continuam normais, exatamente como `inspect` já reporta o ganho de elevação como ausente para esse mesmo trajeto.
- O que acontece quando o usuário pede um bloco que depende de um dado ausente (por exemplo, pede o perfil de elevação para um trajeto sem elevação)? A ferramenta aceita o pedido normalmente e simplesmente não desenha esse bloco naquele voo, sem erro — a ausência é do dado, não do pedido.
- O que acontece no primeiro quadro do voo, antes de qualquer distância ter sido percorrida? A distância percorrida e o tempo decorrido aparecem como zero, e o marcador do perfil de elevação aparece no início do perfil — nunca em branco.
- O que acontece com a legibilidade das sobreposições quando a cor de fundo do quadro (ajustável desde a etapa anterior) é muito clara ou muito escura, ou quando o relevo por baixo do texto muda de claro para escuro entre quadros? O texto e o gráfico permanecem legíveis nos dois extremes, por um tratamento de contraste que não depende da cor de fundo escolhida.
- O que acontece nas extremidades de um vídeo vertical (9:16), que redes sociais costumam cortar? Nenhuma sobreposição é desenhada além da margem de segurança documentada, então o corte de uma rede social nunca corta um número ou um traço do perfil ao meio.
- O que acontece quando um diretório de quadros (de `render all` ou de um `fly --keep`) tem quadros de uma configuração de sobreposição diferente da pedida agora, e o usuário não pede a sobrescrita? A ferramenta recusa exatamente como já recusa hoje diante de um conjunto de quadros de outro plano, outro recorte, outra resolução ou outra aparência, sem apagar nada.

## Requisitos *(obrigatório)*

### Requisitos Funcionais

- **FR-001**: O sistema DEVE permitir que, ao desenhar quadros (`render frame`, `render all`) e ao rodar o comando único (`fly`), cada quadro exiba, em posição fixa na tela — não desenhada sobre o terreno, não afetada pelo movimento da câmera —, a distância percorrida até aquele instante, a elevação do trajeto e o ganho de elevação acumulado naquele ponto, o tempo decorrido da atividade, e um perfil de elevação do trajeto inteiro com um marcador que acompanha o avanço do voo.
- **FR-002**: As sobreposições DEVEM vir ligadas por padrão: gerar um quadro ou um voo sem informar nada sobre sobreposições produz, a partir desta etapa, um resultado com todos os blocos da FR-001 visíveis, usando os valores do plano e do recorte informados.
- **FR-003**: O usuário DEVE poder desligar as sobreposições por inteiro, obtendo um quadro com só o voo, o traçado e o marcador — sem nenhum texto ou gráfico desenhado por cima.
- **FR-004**: O usuário DEVE poder escolher, de forma independente, se cada um destes blocos aparece: (a) distância percorrida; (b) elevação do trajeto e ganho acumulado no ponto do marcador; (c) tempo decorrido da atividade; (d) perfil de elevação com marcador — em qualquer combinação, sem que ligar ou desligar um afete os demais.
- **FR-005**: Todo valor exibido nas sobreposições DEVE vir exclusivamente do plano de câmera e do recorte de dados geográficos informados ao comando; nenhuma etapa desta funcionalidade lê de novo o arquivo de trajeto GPS nem os dados geográficos registrados.
- **FR-006**: O formato do plano de câmera DEVE passar a guardar, por quadro, o instante real da atividade e a elevação do trajeto no ponto do marcador — campos novos, acrescentados sem mudar o significado de nenhum campo hoje existente no plano.
- **FR-007**: Um plano de câmera exportado por uma versão da ferramenta anterior a esta etapa, sem os dois campos novos da FR-006, DEVE ser recusado por `render frame`, `render all` e `fly`, com uma mensagem que diz ao usuário para gerar o plano de novo.
- **FR-008**: Todo texto das sobreposições DEVE ser desenhado com uma fonte embutida na própria ferramenta, nunca uma fonte instalada no sistema operacional, de modo que o mesmo plano produza o mesmo quadro em qualquer máquina.
- **FR-009**: As sobreposições DEVEM respeitar uma margem de segurança documentada, medida a partir das quatro bordas do quadro, dentro da qual nenhum texto ou gráfico de sobreposição é desenhado — para que o corte de borda que redes sociais aplicam a vídeos verticais nunca corte uma sobreposição ao meio.
- **FR-010**: As sobreposições DEVEM permanecer legíveis sobre qualquer fundo sobre o qual forem desenhadas, claro ou escuro, sem depender da cor de fundo escolhida pelo usuário (etapa anterior) nem do relevo por baixo delas.
- **FR-011**: A configuração de sobreposição escolhida (ligada ou desligada por inteiro, e quais blocos aparecem) DEVE fazer parte do que identifica um conjunto de quadros, do mesmo modo que a aparência já faz — quadros desenhados com configurações de sobreposição diferentes nunca podem ser tomados como pertencentes ao mesmo conjunto, nem ao retomar um desenho interrompido (`render all`), nem ao reaproveitar os intermediários guardados do comando único (`fly --keep`).
- **FR-012**: Ao retomar um desenho ou reaproveitar intermediários guardados com a mesma configuração de sobreposição da execução anterior, a ferramenta DEVE reaproveitar os quadros já prontos; ao encontrar quadros de uma configuração diferente da pedida agora, DEVE recusar por padrão e só redesenhar com a sobrescrita pedida explicitamente — exatamente como já acontece hoje para plano, recorte, resolução e aparência diferentes.
- **FR-013**: O mesmo plano, o mesmo recorte, a mesma resolução, a mesma aparência e a mesma configuração de sobreposição DEVEM continuar produzindo, em qualquer execução, imagens idênticas byte a byte.
- **FR-014**: As unidades exibidas DEVEM ser métricas (distância e elevação em metros/quilômetros, tempo como duração) e documentadas; ao final do voo, os valores acumulados exibidos (distância percorrida, ganho de elevação, tempo decorrido) DEVEM coincidir, exatamente, com o que `inspect` relata para o mesmo trajeto.
- **FR-015**: Nada nesta etapa pode alterar a posição ou o enquadramento da câmera, o terreno desenhado, o traçado, a posição do marcador sobre o terreno, ou a duração do vídeo — o efeito de qualquer configuração de sobreposição se limita ao que é desenhado por cima da cena já existente.
- **FR-016**: Quando o trajeto não tem instante real de atividade (o plano foi gerado a partir da distância, não de um relógio de GPS) ou não tem dado de elevação, os blocos que dependem desse dado ausente (tempo decorrido; elevação e ganho; perfil de elevação, respectivamente) NÃO DEVEM ser desenhados nesse voo, mesmo que pedidos — sem erro, e sem inventar um valor —, exatamente como `inspect` já reporta esses dados como ausentes para o mesmo trajeto.
- **FR-017**: Um nome de bloco de sobreposição que não existe DEVE ser recusado antes de desenhar qualquer quadro, com uma mensagem que lista os blocos aceitos.
- **FR-018**: O usuário DEVE poder ver o efeito de uma configuração de sobreposição escolhida desenhando um único quadro antes de desenhar o voo inteiro, sem nenhuma diferença de comportamento entre os valores usados nesse quadro isolado e os mesmos valores usados depois no voo inteiro.

### Entidades-Chave

- **Configuração de Sobreposição**: se as sobreposições estão ligadas, e quais dos quatro blocos aparecem. Passa a fazer parte do que identifica um conjunto de quadros, junto com o plano, o recorte, a resolução e a aparência que já identificavam um conjunto hoje.
- **Bloco de Sobreposição**: uma das quatro unidades de informação que podem ser mostradas ou não, independentemente: distância percorrida; elevação do trajeto e ganho acumulado no ponto do marcador; tempo decorrido da atividade; perfil de elevação do trajeto inteiro com marcador de progresso.
- **Instante da Atividade e Elevação do Trajeto por Quadro**: os dois dados novos que o plano de câmera passa a guardar para cada quadro — o instante real da atividade e a elevação do trajeto no ponto do marcador —, que tornam possível calcular todos os valores das sobreposições sem reler o trajeto GPS ou os dados geográficos registrados.

## Critérios de Sucesso *(obrigatório)*

### Resultados Mensuráveis

- **SC-001**: Um usuário consegue produzir um vídeo mostrando distância percorrida, elevação, ganho acumulado, tempo decorrido e perfil de elevação, sem alterar nenhum outro aspecto do vídeo (câmera, enquadramento, terreno, traçado, duração).
- **SC-002**: Um usuário que desliga as sobreposições por inteiro obtém um vídeo idêntico, pixel a pixel, ao que a ferramenta produzia antes desta funcionalidade existir, para o mesmo plano, recorte, resolução e aparência.
- **SC-003**: Desenhar o mesmo quadro duas vezes com a mesma configuração de sobreposição produz sempre a mesma imagem, byte a byte.
- **SC-004**: Todo plano de câmera gerado por uma versão anterior da ferramenta é recusado, com uma mensagem que orienta a gerá-lo de novo, antes de qualquer quadro ser desenhado.
- **SC-005**: Ao final de um voo gerado com as sobreposições ligadas, a distância percorrida, o ganho de elevação e o tempo decorrido mostrados no último quadro coincidem, exatamente, com o que `inspect` relata para o mesmo trajeto.
- **SC-006**: Num vídeo vertical (9:16), nenhuma sobreposição fica fora da margem de segurança documentada — um corte de borda comum de rede social nunca corta um número ou o perfil de elevação ao meio.
- **SC-007**: As sobreposições permanecem legíveis tanto sobre o fundo mais claro quanto sobre o mais escuro que a ferramenta pode produzir.
- **SC-008**: Um quadro isolado desenhado com uma configuração de sobreposição escolhida é sempre idêntico, byte a byte, ao quadro correspondente de um voo inteiro desenhado com a mesma configuração, numa fração pequena do tempo de desenhar o voo inteiro.
- **SC-009**: Uma retomada de `render all` ou um reaproveitamento de `fly --keep` nunca mistura, no mesmo vídeo, quadros com configurações de sobreposição diferentes; sem mudança de configuração, os quadros prontos continuam sendo reaproveitados sem trabalho refeito.

## Suposições

- Os quatro blocos de sobreposição são exatamente os agrupamentos descritos no pedido original: distância percorrida; elevação do trajeto e ganho acumulado (mostrados juntos); tempo decorrido; perfil de elevação com marcador. Não há um quinto bloco nem uma divisão mais fina dentro de cada um (ver Clarifications).
- A elevação do trajeto e o ganho acumulado exibidos vêm do próprio perfil de elevação do trajeto (o mesmo dado que `inspect` já usa para relatar ganho de elevação), e não do relevo/terreno do recorte — é isso que torna possível coincidir exatamente com o que `inspect` relata (FR-014), e é a leitura literal de "a elevação do trajeto" no pedido original.
- O formato exato de exibição de cada valor (casas decimais, unidade usada em cada faixa de grandeza — por exemplo, metros ou quilômetros —, formato do tempo decorrido) é decidido no planejamento técnico e documentado; não é escolhido pelo usuário nesta etapa (escolha de formato de texto livre está fora de escopo).
- A posição exata de cada bloco na tela, o tamanho da margem de segurança e o tratamento de contraste que garante legibilidade sobre qualquer fundo são decisões de planejamento técnico, documentadas quando definidas; o pedido só exige que a margem seja documentada e respeitada, e que a legibilidade se sustente em qualquer fundo — não prescreve os números exatos.
- A fonte embutida é única e fixa, escolhida pela ferramenta — escolher a fonte está explicitamente fora de escopo desta etapa.
- Um trajeto sem instante real de atividade (referência por distância, não por relógio) ou sem dado de elevação não impede o restante das sobreposições: só os blocos que dependem do dado ausente deixam de ser desenhados, sem erro.
- A configuração de sobreposição é escolhida a cada execução do comando, do mesmo jeito que a aparência já é hoje — não há memória de uma escolha anterior entre execuções independentes.
- Nem áudio, nem música, nem logotipos/marcas, nem texto livre digitado pelo usuário, nem animação de entrada/saída dos blocos, nem gráfico interativo fazem parte desta etapa — permanecem fora de escopo, como o pedido original define.
