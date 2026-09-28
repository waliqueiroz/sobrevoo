# Especificação de Funcionalidade: Aparência Ajustável dos Quadros

**Branch da Funcionalidade**: `008-frame-appearance`

**Criado em**: 2026-09-28

**Status**: Rascunho

**Entrada**: Descrição do usuário: "Oitava etapa do Sobrevoo. Esta etapa não muda nada do que a ferramenta calcula — trajeto, câmera, dados, enquadramento e vídeo seguem idênticos: ela torna ajustável a aparência do que é desenhado sobre o terreno. Hoje a cor e a espessura do traçado, a cor e o tamanho do marcador e a cor do fundo são valores fixos no código, escolhidos uma vez; quem quer um traçado mais fino, um marcador maior ou um laranja diferente precisa recompilar. Com esta etapa, o usuário escolhe esses valores na linha de comando, nos mesmos comandos que já desenham quadros, e vê o efeito num quadro isolado antes de desenhar o voo inteiro. O usuário consegue: escolher a cor e a espessura do traçado, a cor e o raio do marcador, e a cor do fundo; usar os mesmos valores em render frame, render all e fly, com os mesmos nomes; e contar com os valores atuais como padrão, de modo que quem não informa nada obtém exatamente o resultado de hoje. Requisitos: a aparência escolhida DEVE fazer parte da identidade do conjunto de quadros, de modo que quadros desenhados com aparências diferentes nunca sejam tomados como do mesmo conjunto — nem na retomada de um desenho interrompido, nem no reaproveitamento do comando único, que DEVEM redesenhar quando a aparência muda e reaproveitar quando ela é a mesma; a ferramenta DEVE recusar, com erro próprio e mensagem que aponta o valor recebido e o formato ou intervalo aceito, uma cor mal formada ou uma espessura, raio ou proporção fora dos limites documentados; as marcações de falta — o hachurado de 'sem imagem de mapa' e o xadrez de 'sem elevação' — NÃO DEVEM ser ajustáveis, porque não são estilo e sim o significado da imagem: elas precisam ser reconhecíveis sempre, em qualquer vídeo; o mesmo plano, o mesmo recorte, a mesma resolução e a mesma aparência DEVEM continuar produzindo imagens idênticas byte a byte; e nenhum valor de aparência pode alterar o enquadramento, a posição do marcador, a geometria do terreno ou a duração do vídeo — só as cores e os tamanhos do que é desenhado por cima. Fora de escopo: sobreposições de texto ou estatísticas na tela, fontes, legendas, mudança de qualquer regra de câmera, de dados ou de codificação, e qualquer ajuste de aparência do mapa base, que vem pronto do arquivo registrado."

## Clarifications

### Session 2026-09-28

- Q: O texto de requisitos cita "uma espessura, raio ou proporção fora dos limites documentados" — essa "proporção" é um sexto valor ajustável, separado dos cinco já listados, ou é só uma forma de descrever que espessura e raio têm que caber numa proporção aceitável do quadro? → A: Não é um valor à parte — "proporção" descreve o intervalo aceitável de espessura/raio; os cinco valores já listados (cor/espessura do traçado, cor/raio do marcador, cor do fundo) continuam sendo os únicos ajustáveis.
- Q: Em que formato o usuário vai escrever uma cor na linha de comando? → A: Somente hexadecimal RGB (`#RRGGBB`) — um formato único, sem canal alfa e sem lista de nomes de cor.
- Q: Quando o usuário escolhe a espessura do traçado ou o raio do marcador, esse número é um tamanho absoluto em pixels da imagem final, ou uma fração do tamanho do quadro (escala junto com a resolução, como o desenho já faz internamente hoje)? → A: Fração da altura do quadro, do mesmo jeito que o desenho já se comporta hoje (por exemplo, 0.005 para o traçado e 0.012 para o marcador, os valores de hoje). O tamanho em pixels é derivado como o maior valor entre essa proporção vezes a altura da imagem e um piso mínimo em pixels documentado (hoje 2 para o traçado, 4 para o marcador), que existe para o traçado e o marcador não sumirem em resoluções pequenas. É a proporção — não o tamanho em pixels que ela produz numa resolução específica — que entra na identidade de um conjunto de quadros e que é validada contra o intervalo aceito, preservando a garantia de que duas resoluções de mesma proporção produzem o mesmo peso visual do traçado e do marcador.

## Cenários de Usuário e Testes *(obrigatório)*

### História de Usuário 1 - Escolher a cor e o tamanho do que é desenhado (Prioridade: P1)

Como usuário do Sobrevoo, hoje o traçado é sempre laranja, o marcador sempre vermelho com anel branco e o fundo sempre o mesmo tom escuro — valores fixos que só mudariam recompilando a ferramenta. Quero escolher, na linha de comando, a cor e a espessura do traçado, a cor e o raio do marcador, e a cor do fundo, para que o vídeo combine com a identidade visual que eu quiser (por exemplo, as cores do meu clube ou de um evento).

**Por que esta prioridade**: é o valor central da etapa — sem isso, nada mais nesta funcionalidade tem sentido; as demais histórias só garantem que esse ajuste seja previsível, consistente entre comandos e seguro.

**Teste Independente**: pode ser totalmente testado desenhando um quadro com cada um dos cinco ajustes num valor diferente do padrão e conferindo, por inspeção visual e por comparação de pixels, que exatamente aquele elemento mudou e nenhum outro.

**Cenários de Aceitação**:

1. **Dado** um plano e um recorte válidos, **Quando** o usuário desenha um quadro informando uma cor e uma espessura de traçado diferentes das atuais, **Então** o traçado desenhado usa essa cor e essa espessura, e nenhum outro elemento do quadro muda.
2. **Dado** um plano e um recorte válidos, **Quando** o usuário desenha um quadro informando uma cor e um raio de marcador diferentes dos atuais, **Então** o marcador desenhado usa essa cor e esse raio, e nenhum outro elemento do quadro muda.
3. **Dado** um plano e um recorte válidos, **Quando** o usuário desenha um quadro informando uma cor de fundo diferente da atual, **Então** toda área que hoje seria pintada com o fundo padrão passa a usar a nova cor, e nenhum outro elemento do quadro muda.
4. **Dado** que o usuário não informa nenhum dos cinco ajustes, **Quando** um quadro é desenhado, **Então** o resultado é idêntico, pixel a pixel, ao que a ferramenta produz hoje, sem esta funcionalidade.

---

### História de Usuário 2 - Ver o efeito num quadro isolado antes do voo inteiro (Prioridade: P2)

Como usuário do Sobrevoo, sei que desenhar o voo inteiro ou montar o vídeo final pode levar bastante tempo. Antes de me comprometer com uma aparência para o vídeo inteiro, quero desenhar um único quadro com os valores que estou considerando, olhar o resultado, e só então decidir se aplico os mesmos valores ao voo inteiro — com a certeza de que o quadro isolado mostra exatamente o que o voo inteiro vai mostrar.

**Por que esta prioridade**: sem isso, experimentar uma aparência custaria o tempo de um voo inteiro a cada tentativa, o que desencorajaria o ajuste fino que é o próprio motivo da etapa; depende da História 1 já existir.

**Teste Independente**: pode ser totalmente testado desenhando um quadro isolado com uma aparência escolhida, depois desenhando o voo inteiro (ou só o mesmo quadro dentro dele) com os mesmos valores, e comparando as duas imagens desse quadro byte a byte.

**Cenários de Aceitação**:

1. **Dado** que o usuário desenhou um quadro isolado com uma aparência escolhida e gostou do resultado, **Quando** ele desenha o voo inteiro ou gera o vídeo com o comando único usando os mesmos valores de aparência, **Então** o quadro correspondente do voo inteiro é idêntico, byte a byte, ao quadro isolado.
2. **Dado** que o usuário desenhou um quadro isolado com uma aparência e não gostou, **Quando** ele desenha outro quadro isolado com valores diferentes, **Então** só precisa esperar o tempo de desenhar um quadro, não o de um voo inteiro, para ver o novo resultado.

---

### História de Usuário 3 - Os mesmos ajustes, com os mesmos nomes, em qualquer comando (Prioridade: P3)

Como usuário do Sobrevoo, já uso `render frame`, `render all` e `fly`, e já sei como ajustar coisas como resolução ou qualidade em cada um deles. Quero que a cor e a espessura do traçado, a cor e o raio do marcador e a cor do fundo sejam ajustados da mesma forma, com os mesmos nomes e os mesmos valores aceitos, em qualquer um dos três, sem precisar aprender uma forma diferente de pedir a mesma coisa em cada comando.

**Por que esta prioridade**: garante que o que o usuário aprende ajustando um comando vale para os outros dois, e é o que torna a História 2 (testar num quadro e aplicar ao voo inteiro) confiável — sem nomes e valores idênticos, o usuário não poderia ter certeza de estar pedindo a mesma aparência nos dois comandos.

**Teste Independente**: pode ser totalmente testado informando o mesmo valor, com o mesmo nome de ajuste, em `render frame`, `render all` e `fly`, e conferindo que os três aceitam o valor da mesma forma e produzem o mesmo efeito visual.

**Cenários de Aceitação**:

1. **Dado** um mesmo valor de cor ou tamanho, **Quando** o usuário o informa com o mesmo nome de ajuste em `render frame`, `render all` e `fly`, **Então** os três comandos o aceitam sem erro e produzem exatamente o mesmo efeito visual.
2. **Dado** um valor inválido para qualquer um dos cinco ajustes, **Quando** o usuário o informa em qualquer um dos três comandos, **Então** a recusa acontece com a mesma mensagem de erro em todos eles.

---

### História de Usuário 4 - Nunca confundir quadros de aparências diferentes (Prioridade: P4)

Como usuário do Sobrevoo, às vezes interrompo um `render all` no meio e retomo depois, ou guardo os intermediários de um `fly` para reaproveitar numa execução seguinte. Se eu mudar a aparência entre uma execução e a próxima, quero que a ferramenta perceba a mudança e redesenhe os quadros com a nova aparência, em vez de misturar quadros antigos com a aparência velha e quadros novos com a aparência nova no mesmo vídeo; e quando eu não mudo nada, quero que ela continue reaproveitando o que já está pronto, sem redesenhar à toa.

**Por que esta prioridade**: é a rede de segurança da funcionalidade — sem ela, um vídeo poderia sair com um traçado de duas cores diferentes por acidente; depende das Histórias 1 e 3 já existirem, e só importa para quem usa retomada ou reaproveitamento, um subconjunto de uso mais avançado.

**Teste Independente**: pode ser totalmente testado desenhando parte de um voo com uma aparência, mudando a aparência e desenhando o resto (ou reaproveitando um `fly --keep` anterior com uma aparência diferente), e conferindo que a ferramenta recusa por padrão a mistura e redesenha por completo quando pedido, e que, sem mudar a aparência, uma nova execução reaproveita os quadros já prontos.

**Cenários de Aceitação**:

1. **Dado** um `render all` interrompido no meio com uma aparência, **Quando** o usuário o retoma pedindo uma aparência diferente, **Então** a ferramenta recusa por padrão, avisando que os quadros já lá são de outra aparência, e só redesenha com a sobrescrita pedida explicitamente.
2. **Dado** um `render all` interrompido no meio, **Quando** o usuário o retoma com a mesma aparência de antes, **Então** os quadros já desenhados são reaproveitados e só os que faltam são desenhados, exatamente como já acontece hoje.
3. **Dado** um `fly --keep <diretório>` já executado com uma aparência, **Quando** o usuário roda de novo apontando para o mesmo diretório, com o mesmo trajeto e os mesmos outros valores, mas com uma aparência diferente, **Então** o plano de câmera e o recorte de dados geográficos são reaproveitados, mas os quadros e o vídeo são refeitos com a nova aparência.
4. **Dado** o mesmo cenário do item anterior, mas sem nenhuma mudança de aparência, **Quando** o usuário roda de novo, **Então** os quadros e o vídeo também são reaproveitados, sem nenhum trabalho refeito.

---

### Casos Extremos

- O que acontece quando o usuário informa uma cor que não é hexadecimal RGB (`#RRGGBB`) — por exemplo, um nome de cor, um valor com canal alfa, ou um texto com dígitos a mais, a menos ou inválidos? A ferramenta recusa antes de desenhar qualquer quadro, com uma mensagem que mostra o valor recebido e o formato aceito.
- O que acontece quando o usuário pede uma proporção de espessura de traçado ou de raio de marcador fora do intervalo documentado (por exemplo, grande o bastante para dominar o quadro, ou zero/negativa)? A ferramenta recusa, com uma mensagem que mostra o valor recebido e o intervalo aceito, antes de desenhar qualquer quadro.
- O que acontece quando a proporção escolhida, numa resolução pequena, resultaria em menos pixels do que o piso mínimo documentado? O traçado ou o marcador são desenhados com o piso mínimo em pixels, exatamente como já acontece hoje com os valores fixos atuais.
- O que acontece quando a cor escolhida para o traçado, o marcador ou o fundo coincide com uma das cores fixas do hachurado de "sem mapa" ou do xadrez de "sem elevação"? A ferramenta aceita normalmente — essas marcações não deixam de existir nem mudam de cor por causa da aparência escolhida; a ambiguidade visual que resultar é uma escolha de quem define a cor, não algo que a ferramenta impeça.
- O que acontece quando um diretório de quadros (de `render all` ou de um `fly --keep`) tem quadros de uma aparência diferente da pedida agora, e o usuário não pede a sobrescrita? A ferramenta recusa exatamente como já recusa hoje diante de um conjunto de quadros de outro plano, outro recorte ou outra resolução, sem apagar nada.
- O que acontece quando o usuário pede uma aparência em `render frame` mas não pede a mesma em `render all` logo em seguida? Cada comando usa só o que foi informado a ele; se os valores realmente diferirem, os quadros pertencem a conjuntos diferentes, como esperado — não há memória de aparência entre comandos.

## Requisitos *(obrigatório)*

### Requisitos Funcionais

- **FR-001**: O sistema DEVE permitir que o usuário escolha, ao desenhar quadros (`render frame`, `render all`) e ao rodar o comando único (`fly`), a cor e a espessura do traçado do trajeto, a cor e o raio do marcador da atividade, e a cor do fundo.
- **FR-002**: Os três comandos DEVEM aceitar esses cinco ajustes com os mesmos nomes de parâmetro e os mesmos valores aceitos, de modo que um mesmo valor, informado a qualquer um deles, produza sempre o mesmo efeito visual.
- **FR-003**: Quando o usuário não informa um desses ajustes, o comando DEVE usar exatamente o valor que a ferramenta usa hoje para esse elemento, de modo que quem não pede nada continue obtendo, pixel a pixel, o resultado de sempre.
- **FR-004**: A ferramenta DEVE recusar, antes de desenhar qualquer quadro, uma cor que não esteja em hexadecimal RGB (`#RRGGBB`) ou uma proporção de espessura ou de raio fora do intervalo documentado, com uma mensagem que aponta o valor recebido e o formato ou o intervalo aceito.
- **FR-005**: O hachurado que marca a ausência de imagem de mapa base e o xadrez que marca a ausência de dado de elevação NÃO DEVEM ser afetados por nenhum ajuste de aparência — permanecem exatamente como são hoje, em qualquer combinação de valores escolhidos, e não podem ser desativados nem substituídos.
- **FR-006**: Nenhum ajuste de aparência pode alterar o enquadramento da câmera, a posição do marcador, a geometria do terreno desenhado ou a duração do vídeo — o efeito de qualquer valor de aparência se limita às cores e aos tamanhos do que é desenhado por cima da cena.
- **FR-007**: A aparência escolhida DEVE fazer parte do que identifica um conjunto de quadros: quadros desenhados com aparências diferentes nunca podem ser tomados como pertencentes ao mesmo conjunto, nem ao retomar um desenho interrompido (`render all`), nem ao reaproveitar os intermediários guardados do comando único (`fly --keep`).
- **FR-008**: Ao retomar um desenho ou reaproveitar intermediários guardados com a mesma aparência da execução anterior, a ferramenta DEVE reaproveitar os quadros já prontos, exatamente como já faz hoje; ao encontrar quadros de uma aparência diferente da pedida agora, DEVE tratá-los como de qualquer outro conjunto diferente — recusando por padrão e só redesenhando com a sobrescrita pedida explicitamente.
- **FR-009**: Ao mudar só a aparência entre duas execuções do comando único que reaproveitam os mesmos intermediários guardados, apenas os quadros e o vídeo — as etapas cujo resultado depende da aparência — DEVEM ser refeitos; o plano de câmera e o recorte de dados geográficos, que não dependem da aparência, DEVEM continuar sendo reaproveitados.
- **FR-010**: O mesmo plano, o mesmo recorte, a mesma resolução e a mesma aparência DEVEM continuar produzindo, em qualquer execução, imagens idênticas byte a byte.
- **FR-011**: A ferramenta DEVE permitir ver o efeito de uma aparência escolhida desenhando um único quadro antes de desenhar o voo inteiro, sem nenhuma diferença de comportamento entre o valor usado nesse quadro isolado e o mesmo valor usado depois no voo inteiro.
- **FR-012**: A espessura do traçado e o raio do marcador DEVEM ser expressos como uma proporção da altura do quadro — o mesmo jeito que o desenho já os calcula hoje —, sujeita a um piso mínimo em pixels que evita que o traçado ou o marcador sumam em resoluções pequenas, piso que esta etapa não altera; é essa proporção, e não o tamanho em pixels que ela produz numa resolução específica, que faz parte da identidade de um conjunto de quadros (FR-007) e que é validada contra o intervalo aceito (FR-004).

### Entidades-Chave

- **Aparência de Desenho**: o conjunto dos cinco valores ajustáveis — cor e espessura do traçado, cor e raio do marcador, cor do fundo. A espessura e o raio são proporções da altura do quadro, não tamanhos fixos em pixels (FR-012). Não inclui o hachurado de "sem mapa" nem o xadrez de "sem elevação", que são fixos. Passa a fazer parte do que identifica um conjunto de quadros, junto com o plano, o recorte e a resolução que já identificavam um conjunto hoje.

## Critérios de Sucesso *(obrigatório)*

### Resultados Mensuráveis

- **SC-001**: Um usuário consegue produzir um vídeo com o traçado, o marcador e o fundo em cores e tamanhos diferentes dos atuais, sem alterar nenhum outro aspecto do vídeo (enquadramento, duração, terreno).
- **SC-002**: Quem não informa nenhum ajuste de aparência obtém um resultado idêntico, pixel a pixel, ao que a ferramenta produzia antes desta funcionalidade existir.
- **SC-003**: Desenhar o mesmo quadro duas vezes com a mesma aparência produz sempre a mesma imagem, byte a byte.
- **SC-004**: Todo valor de aparência malformado ou fora do intervalo aceito é rejeitado, com uma mensagem que identifica o valor recebido e o formato ou intervalo esperado, antes de qualquer quadro ser desenhado.
- **SC-005**: Um quadro isolado desenhado com uma aparência escolhida é sempre idêntico, byte a byte, ao quadro correspondente de um voo inteiro desenhado com a mesma aparência.
- **SC-006**: Repetir o comando único com os mesmos intermediários guardados e só a aparência alterada termina numa fração pequena do tempo de uma execução do zero, porque só os quadros e o vídeo são refeitos.
- **SC-007**: Uma retomada ou um reaproveitamento nunca mistura, no mesmo vídeo, quadros de aparências diferentes.
- **SC-008**: A mesma aparência escolhida, desenhada em duas resoluções diferentes de mesma proporção, produz o traçado e o marcador com o mesmo peso visual relativo ao quadro.

## Suposições

- Cada cor é informada em hexadecimal RGB (`#RRGGBB`), sem canal alfa e sem nomes de cor — um único formato, igual nos três comandos (ver Clarifications).
- A espessura do traçado e o raio do marcador são informados como uma proporção da altura do quadro — o mesmo jeito que o desenho já calcula esses tamanhos hoje —, não como um número fixo de pixels; os valores exatos do intervalo aceito para essa proporção são decididos no planejamento técnico, tomando os valores de hoje (0.005 e 0.012) como referência central (ver Clarifications). Não há um sexto valor ajustável de "proporção": o termo, no pedido original, descreve esse intervalo aceitável de espessura e raio, não um parâmetro à parte.
- A cor de fundo aceita o mesmo formato de cor dos demais ajustes, sem nenhuma restrição adicional além de ser uma cor válida.
- Escolher uma cor igual a uma das cores fixas de marcação de falta é permitido; a ferramenta não impede nem avisa sobre a coincidência, já que validar isso está fora do escopo desta etapa.
- O piso mínimo em pixels que já protege o traçado e o marcador de sumirem em resoluções pequenas (hoje 2 pixels para o traçado, 4 para o marcador) não muda com esta etapa: continua se aplicando por baixo de qualquer proporção escolhida, inclusive a padrão.
- Um conjunto de quadros desenhado por uma versão da ferramenta anterior a esta funcionalidade, sem nenhuma informação de aparência gravada, nunca é confundido com um conjunto desenhado por esta versão com a aparência padrão — a mesma garantia que já separa hoje quadros de versões diferentes do desenho.
