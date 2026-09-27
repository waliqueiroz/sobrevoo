# Especificação de Funcionalidade: Desenho dos Quadros do Voo

**Branch da Funcionalidade**: `005-frame-rendering`

**Criado em**: 2026-09-26

**Status**: Implementada

**Entrada**: Descrição do usuário: "Quinta etapa do Sobrevoo. Esta etapa cobre só o desenho dos quadros do vídeo, como imagens. Nada de juntar os quadros num arquivo de vídeo, sobreposições de estatísticas nem áudio ainda. A partir do plano de câmera exportado na terceira etapa e do recorte de dados exportado na quarta, a ferramenta desenha o que a câmera vê em cada quadro: o relevo do terreno em perspectiva, vestido com as peças do mapa base, o traçado do trajeto sobre o terreno e o marcador da atividade na posição daquele instante. O usuário consegue: desenhar um quadro isolado, pelo número, para conferir o enquadramento antes de gastar tempo com o voo inteiro; desenhar todos os quadros num diretório de destino; escolher a resolução da imagem; acompanhar o progresso e retomar de onde parou quando a execução for interrompida; e ver um resumo ao final (quantos quadros foram desenhados, resolução, tempo gasto, e quantos quadros tiveram buraco por peça de mapa ausente ou amostra de elevação sem valor). Requisitos: usar exclusivamente o plano e o recorte informados, sem rede e sem voltar aos arquivos registrados; não inventar dado — onde falta peça de mapa ou valor de elevação, o quadro mostra a falta de forma explícita e o resumo a contabiliza; o mesmo plano e o mesmo recorte produzem sempre exatamente as mesmas imagens, byte a byte, independentemente de momento, ordem ou de quantos quadros foram desenhados por execução; a numeração e a ordem dos quadros seguem o plano, para que a etapa seguinte os junte sem ambiguidade; recusar, com mensagem clara e erro próprio, um plano ou um recorte inválido, truncado ou de versão de formato desconhecida, um recorte que não cubra a área do plano informado, e um par plano-recorte que não corresponda um ao outro; aceitar peças de mapa base em formato de imagem e recusar, com mensagem clara, peças vetoriais, cujo desenho fica para uma etapa futura; recusar resolução fora dos limites documentados; por padrão não sobrescrever imagens já existentes no destino, com opção explícita de sobrescrita, e nunca deixar imagem parcial; comportamento idêntico em qualquer lugar do planeta, inclusive cruzando o meridiano de 180° e em latitudes altas. Fora de escopo: codificar o vídeo, sobreposições de texto ou de estatísticas, áudio, iluminação e sombras realistas, desenho de peças vetoriais, interface gráfica e API."

## Clarificações

### Sessão 2026-09-26

- Q: Como a ferramenta deve saber que o recorte foi feito a partir do plano informado, e não de outro plano? → A: O recorte guarda uma identificação do plano de que veio (opção B), e esta etapa a confere. Isso acrescenta um campo à saída da etapa 4; recortes exportados antes dele não têm o campo e precisam ser gerados de novo.
- Q: Se o destino já tiver parte dos quadros do mesmo voo, a ferramenta continua sozinha de onde parou, ou só com um pedido explícito? → A: Continua sozinha quando os quadros existentes são do mesmo plano, recorte e resolução (mantém o que existe e desenha o que falta); recusa quando são de outro conjunto; a sobrescrita explícita refaz tudo. Sem opção `--resume` (FR-021 e FR-022).
- Q: O traçado sobre o terreno mostra o percurso inteiro desde o primeiro quadro, ou vai sendo revelado conforme o marcador avança? → A: Vai sendo revelado (opção B): o quadro k mostra só o trecho do início até o marcador daquele quadro, e o restante do trajeto nunca é desenhado (FR-010).
- Q: O quadro isolado vai para um arquivo escolhido pelo usuário ou para o diretório de destino do voo? → A: Para um arquivo escolhido pelo usuário (opção B), fora do conjunto do voo, com recusa de arquivo existente salvo sobrescrita explícita (FR-018a).
- Q: Com a sobrescrita, a ferramenta apaga os quadros que sobram de um voo anterior maior, ou os deixa e só avisa? → A: Apaga (opção A), somente os quadros que a ferramenta reconhece como seus (pela identificação que ela grava em cada imagem), de outro conjunto e com número que o plano novo não tem; qualquer outro arquivo do diretório nunca é tocado (FR-023).

## Cenários de Usuário e Testes *(obrigatório)*

### História de Usuário 1 - Desenhar um quadro isolado pelo número (Prioridade: P1)

Como usuário do Sobrevoo, eu já exportei o plano de câmera de um trajeto (terceira etapa) e o recorte de dados desse plano (quarta etapa). Antes de gastar tempo desenhando o voo inteiro, quero pedir um único quadro, pelo número, e ver o que a câmera enxerga naquele instante: o relevo do terreno em perspectiva, vestido com as peças do mapa base, o traçado do trajeto sobre o terreno e o marcador da atividade. Assim confiro o enquadramento e a aparência sem esperar.

**Por que esta prioridade**: é o núcleo da etapa e a menor fatia que entrega valor: transforma os dois arquivos das etapas anteriores em algo que se vê. Todas as demais histórias repetem, protegem ou dimensionam este desenho.

**Teste Independente**: pode ser totalmente testado com um plano e um recorte de exemplo (mapa base em imagem, relevo conhecido), pedindo um quadro pelo número e verificando que a imagem é gerada, tem a resolução esperada, mostra o marcador na posição que a projeção do plano indica e é idêntica a cada repetição.

**Cenários de Aceitação**:

1. **Dado** um plano e um recorte válidos que correspondem um ao outro, **Quando** o usuário pede o quadro de número N do plano, **Então** a ferramenta grava uma imagem no arquivo que o usuário escolheu, com o relevo em perspectiva do ponto de vista da câmera daquele quadro (posição, direção e inclinação do plano), vestido com as peças do mapa base, com o traçado do trajeto sobre o terreno e com o marcador na posição do marcador daquele quadro.
2. **Dado** o mesmo plano, o mesmo recorte e a mesma resolução, **Quando** o usuário pede o mesmo quadro duas vezes (inclusive em execuções e momentos diferentes), **Então** as duas imagens são exatamente iguais, byte a byte.
3. **Dado** um número que não existe no plano (negativo, igual ou maior que a quantidade de quadros, ou não inteiro), **Quando** o usuário pede o quadro, **Então** a ferramenta recusa com erro próprio e mensagem que informa o número recebido e a faixa válida do plano, sem gravar imagem alguma.
4. **Dado** que a ferramenta desenhou o quadro N, **Quando** o usuário vê o resumo ao final, **Então** ele informa que 1 quadro foi desenhado, a resolução usada, o tempo gasto e se o quadro teve buraco de mapa ou de elevação.
5. **Dado** um quadro da fase de abertura ou de fechamento do voo (em que o trajeto inteiro está enquadrado), **Quando** o usuário o pede, **Então** ele é desenhado pelas mesmas regras de qualquer outro quadro: o traçado vai só até o marcador daquele quadro; assim, na abertura (marcador no início do trajeto) ainda não há linha para mostrar, e no fechamento (marcador no fim) o traçado aparece completo.
6. **Dado** dois quadros do mesmo voo, um anterior e outro posterior, **Quando** o usuário os compara, **Então** o traçado do posterior contém o do anterior e mais o trecho percorrido entre os dois; o restante do trajeto, adiante do marcador, nunca é desenhado.

---

### História de Usuário 2 - Desenhar todos os quadros do voo (Prioridade: P2)

Como usuário do Sobrevoo, depois de conferir o enquadramento, quero desenhar todos os quadros do plano num diretório de destino, com a numeração e a ordem do plano, para que a etapa seguinte (que junta os quadros num vídeo) os leia sem ambiguidade.

**Por que esta prioridade**: é o objetivo final da etapa; depende de o desenho de um quadro (P1) funcionar e o multiplica por todo o plano, com a garantia de numeração.

**Teste Independente**: pode ser testado desenhando todos os quadros de um plano curto de exemplo e verificando que o destino contém exatamente uma imagem por quadro, numeradas como o plano, cuja ordem alfabética coincide com a ordem numérica, e que o resumo informa a quantidade correta.

**Cenários de Aceitação**:

1. **Dado** um plano com N quadros e um recorte que corresponde a ele, **Quando** o usuário pede o desenho de todos os quadros para um diretório de destino, **Então** o destino passa a conter exatamente N imagens, uma por quadro, cada uma com o nome determinado pelo número do quadro no plano.
2. **Dado** o conjunto de imagens gerado, **Quando** outra ferramenta as ordena pelo nome, **Então** a ordem obtida é a ordem numérica dos quadros do plano, do primeiro ao último, sem lacunas nem duplicatas.
3. **Dado** o mesmo plano, o mesmo recorte e a mesma resolução, **Quando** o usuário desenha todos os quadros duas vezes, **Então** os dois conjuntos são idênticos, imagem a imagem, byte a byte.
4. **Dado** o quadro N desenhado sozinho (História 1) e o quadro N desenhado como parte do voo inteiro, **Quando** o usuário compara as duas imagens, **Então** elas são exatamente iguais: o resultado de um quadro não depende de quantos quadros foram pedidos na execução, nem da ordem em que foram desenhados.
5. **Dado** um destino que não existe ainda, **Quando** o usuário pede o desenho de todos os quadros, **Então** o destino é criado.
6. **Dado** que o destino existe mas não é um diretório, ou não pode ser gravado (sem permissão), **Quando** o usuário pede o desenho, **Então** a ferramenta recusa com erro próprio e mensagem clara sobre o destino, sem gravar imagem alguma.
7. **Dado** que o desenho terminou, **Quando** o usuário lê o resumo, **Então** ele informa quantos quadros foram desenhados, a resolução, o tempo gasto e quantos quadros tiveram buraco (ver História 3).

---

### História de Usuário 3 - Ver a falta de dado de forma explícita e contabilizada (Prioridade: P3)

Como usuário do Sobrevoo, eu sei que meus dados têm buracos: peças de mapa que o arquivo não contém e pontos de relevo para os quais o arquivo não informa elevação. Quero que o quadro **mostre** essa falta de forma clara, em vez de esconder o buraco com uma cor qualquer ou com um valor inventado, e que o resumo me diga quantos quadros foram afetados, para eu decidir se vale procurar dados melhores antes de gerar o vídeo.

**Por que esta prioridade**: é o que faz a ferramenta honesta com dados reais imperfeitos (princípio já adotado nas etapas 2 e 4: nunca inventar dado). O caminho feliz (P1 e P2) entrega o desenho; esta história garante que o desenho não minta.

**Teste Independente**: pode ser testado com um recorte de exemplo do qual foram removidas algumas peças de mapa e em que algumas amostras de elevação não têm valor, desenhando quadros que enxergam essas regiões e quadros que não as enxergam, e verificando as marcações nas imagens e as contagens no resumo.

**Cenários de Aceitação**:

1. **Dado** um quadro cuja parte visível do terreno inclui uma peça de mapa que o recorte registra como ausente, **Quando** o quadro é desenhado, **Então** a região correspondente aparece com uma marcação explícita de "sem imagem de mapa", visualmente distinta de qualquer imagem de mapa real e de qualquer outra marcação, e o terreno continua desenhado com o seu relevo.
2. **Dado** um quadro cuja parte visível do terreno inclui amostras de elevação sem valor, **Quando** o quadro é desenhado, **Então** a região correspondente aparece com uma marcação explícita de "sem elevação", visualmente distinta das demais; nenhum valor é inventado (nem zero) para a região.
3. **Dado** um voo em que alguns quadros enxergam regiões com falta de dado e outros não, **Quando** o usuário desenha o voo e lê o resumo, **Então** o resumo informa quantos quadros tiveram buraco de mapa e quantos tiveram buraco de elevação (um quadro com os dois entra nas duas contagens), e esses números coincidem exatamente com os quadros em cujas imagens há a marcação correspondente.
4. **Dado** um quadro cuja parte visível não inclui nenhuma região com falta de dado, **Quando** o quadro é desenhado, **Então** ele não traz nenhuma marcação de falta e não entra em nenhuma contagem.
5. **Dado** um quadro em que parte do campo de visão da câmera fica fora da área do recorte (por exemplo, o horizonte, ou uma proporção de imagem muito larga), **Quando** o quadro é desenhado, **Então** a parte fora do recorte aparece como fundo neutro, distinto das marcações de falta, e não é contada como buraco: a ferramenta não desenha terreno que o recorte não tem.
6. **Dado** um recorte em que todas as peças de mapa estão ausentes, **Quando** o usuário desenha o voo, **Então** todos os quadros são desenhados com o relevo e a marcação de "sem imagem de mapa", o resumo deixa isso evidente (todos os quadros com buraco de mapa) e a ferramenta não trata o resultado como sucesso silencioso.

---

### História de Usuário 4 - Escolher a resolução da imagem (Prioridade: P4)

Como usuário do Sobrevoo, quero escolher a resolução (largura e altura, em pixels) das imagens: uma resolução baixa para conferir rápido o enquadramento, e uma alta para o vídeo final. Quando não escolho, quero um padrão sensato.

**Por que esta prioridade**: o desenho já funciona com a resolução padrão (P1 a P3); escolher a resolução é uma comodidade importante, mas não bloqueia as demais.

**Teste Independente**: pode ser testado desenhando o mesmo quadro em resoluções diferentes e verificando as dimensões exatas de cada imagem, e pedindo resoluções fora dos limites e verificando a recusa.

**Cenários de Aceitação**:

1. **Dado** que o usuário não informa resolução, **Quando** ele pede o desenho, **Então** as imagens têm a resolução padrão documentada, e o resumo a informa.
2. **Dado** uma resolução dentro dos limites documentados, **Quando** o usuário pede o desenho, **Então** todas as imagens têm exatamente a largura e a altura pedidas, e o resumo informa essa resolução.
3. **Dado** uma resolução com largura ou altura fora dos limites documentados, ímpar, zero, negativa ou não numérica, **Quando** o usuário pede o desenho, **Então** a ferramenta recusa com erro próprio e mensagem que aponta o valor recebido e os limites aceitos, sem gravar imagem alguma.
4. **Dado** o mesmo quadro desenhado em duas resoluções, **Quando** o usuário compara os dois, **Então** o enquadramento é o mesmo — o mesmo trecho vertical de terreno e o marcador na mesma posição relativa da imagem — e só mudam os detalhes que a resolução permite.

---

### História de Usuário 5 - Acompanhar o progresso e retomar de onde parou (Prioridade: P5)

Como usuário do Sobrevoo, um voo tem centenas ou milhares de quadros e o desenho leva tempo. Quero acompanhar o andamento enquanto ele roda e, se a execução for interrompida (eu cancelo, o computador desliga, falta espaço em disco), quero rodar o mesmo comando de novo e continuar de onde parou, sem redesenhar o que já está pronto.

**Por que esta prioridade**: sem isso o desenho do voo inteiro é frágil e opaco, mas o resultado final de P2 já é correto sem este recurso; ele reduz o custo de uma interrupção.

**Teste Independente**: pode ser testado desenhando um plano de exemplo, interrompendo a execução no meio, rodando de novo o mesmo pedido e verificando que só os quadros faltantes são desenhados e que o conjunto final é idêntico ao de uma execução sem interrupção.

**Cenários de Aceitação**:

1. **Dado** um desenho de muitos quadros em andamento, **Quando** o usuário o acompanha, **Então** a ferramenta informa continuamente quantos quadros já foram concluídos, de quantos, e o tempo decorrido, sem alterar o conteúdo das imagens.
2. **Dado** uma execução interrompida pelo usuário, **Quando** a interrupção acontece, **Então** a ferramenta encerra sem deixar nenhuma imagem parcial, imprime um resumo que deixa claro que o desenho foi interrompido e quantos quadros ficaram prontos, e sai com um código de saída distinto do de sucesso.
3. **Dado** um destino com parte dos quadros de um voo já desenhados (por uma execução anterior interrompida) com o mesmo plano, o mesmo recorte e a mesma resolução, **Quando** o usuário repete o mesmo pedido, **Então** os quadros já existentes são mantidos sem serem redesenhados, só os que faltam são desenhados, e o resumo informa quantos foram desenhados e quantos foram mantidos.
4. **Dado** uma execução retomada até o fim, **Quando** o usuário compara o conjunto final com o de uma execução ininterrupta, **Então** os dois conjuntos são idênticos, imagem a imagem, byte a byte.
5. **Dado** um destino que já contém todos os quadros do mesmo voo, **Quando** o usuário repete o pedido sem pedir sobrescrita, **Então** nada é redesenhado, nada é alterado e o resumo informa que todos os quadros já existiam.

---

### História de Usuário 6 - Proteger as imagens já existentes no destino (Prioridade: P6)

Como usuário do Sobrevoo, desenhar leva tempo e as imagens são o produto do meu trabalho. Quero que a ferramenta nunca sobrescreva imagens existentes sem que eu peça explicitamente, nunca misture, na mesma pasta, quadros de voos ou resoluções diferentes (o que estragaria o vídeo sem eu perceber) e nunca deixe uma imagem pela metade.

**Por que esta prioridade**: é a proteção do resultado, análoga à exportação segura das etapas 3 e 4. O caminho feliz funciona sem ela, mas sem ela um erro de digitação apaga horas de trabalho.

**Teste Independente**: pode ser testado com destinos que já contêm imagens do mesmo voo, de outro voo e de outra resolução, pedindo o desenho com e sem a opção de sobrescrita, e provocando uma falha no meio de uma gravação.

**Cenários de Aceitação**:

1. **Dado** um destino que já contém imagens de um conjunto **diferente** (outro plano, outro recorte ou outra resolução), **Quando** o usuário pede o desenho sem a opção de sobrescrita, **Então** a ferramenta recusa com erro próprio e mensagem que explica que o destino contém quadros de outro conjunto e como proceder (sobrescrever explicitamente ou usar outro destino), sem alterar nada no destino.
2. **Dado** um destino com imagens existentes, **Quando** o usuário pede o desenho com a opção explícita de sobrescrita, **Então** todos os quadros pedidos são redesenhados e substituem os existentes, e ao final de um desenho completo o destino contém somente os quadros do plano informado: quadros de um conjunto anterior cujos números o plano atual não tem não permanecem, para que a etapa seguinte não junte quadros que não pertencem ao voo.
3. **Dado** que a gravação de uma imagem falha no meio (por exemplo, falta de espaço em disco), **Quando** a falha acontece, **Então** nenhuma imagem parcial ou ilegível permanece, e, se havia uma imagem anterior com o mesmo nome (caso de sobrescrita), ela permanece intacta.
4. **Dado** que o usuário pede um quadro isolado para um arquivo que já existe, sem a opção de sobrescrita, **Quando** o pedido é feito, **Então** a ferramenta recusa com erro próprio e mensagem que informa que o arquivo já existe e como sobrescrevê-lo, deixando o arquivo intacto; com a opção de sobrescrita, o arquivo é substituído por inteiro (ou permanece intacto se a nova gravação falhar).
5. **Dado** um quadro isolado gravado num arquivo, **Quando** o usuário desenha depois o voo inteiro em qualquer resolução para um diretório de destino, **Então** o quadro isolado não interfere: ele não é registrado como parte de nenhum conjunto, nem é lido, mantido ou alterado pelo desenho do voo.
6. **Dado** um destino que contém, além dos quadros, arquivos que a ferramenta não reconhece como quadros seus, **Quando** o usuário pede o desenho, **Então** esses arquivos não são alterados nem removidos.

---

### História de Usuário 7 - Recusar entradas que não servem, com mensagem clara (Prioridade: P7)

Como usuário do Sobrevoo, quero que a ferramenta me diga claramente o que há de errado quando o plano ou o recorte não servem: arquivo ilegível, inválido, de versão desconhecida, recorte que não cobre o plano ou que é de outro voo, mapa base vetorial ou recorte sem elevação alguma — em vez de desenhar imagens erradas ou falhar de forma obscura.

**Por que esta prioridade**: são proteções de robustez. O caminho feliz (P1 a P3) entrega o valor, mas sem elas um par de arquivos errado geraria, sem aviso, um vídeo errado.

**Teste Independente**: pode ser testado com um arquivo de teste para cada tipo de recusa, verificando que a mensagem identifica o problema, que o código de saída é próprio e que nenhuma imagem é gravada.

**Cenários de Aceitação**:

1. **Dado** um caminho de plano ou de recorte que não existe, ou que existe mas não pode ser lido, **Quando** o usuário pede o desenho, **Então** a ferramenta recusa com mensagem clara que diz qual arquivo é o problema, sem gravar imagem alguma.
2. **Dado** um arquivo de plano que não é um plano exportado (conteúdo estranho, campos ausentes, truncado, corrompido ou incoerente consigo mesmo) ou que é de versão de formato desconhecida, **Quando** o usuário pede o desenho, **Então** a ferramenta recusa com os mesmos erros próprios que a etapa anterior já usa para o plano, com mensagem que diz o que está errado (ou a versão encontrada e as aceitas).
3. **Dado** um arquivo de recorte que não é um recorte exportado (não é um arquivo de recorte, manifesto ausente ou inválido, truncado, corrompido, ou incoerente consigo mesmo — por exemplo, a contagem de peças ou de amostras do manifesto não bate com o conteúdo), **Quando** o usuário pede o desenho, **Então** a ferramenta recusa com erro próprio de recorte inválido e mensagem que diz o que está errado.
4. **Dado** um recorte de versão de formato que a ferramenta não reconhece, **Quando** o usuário pede o desenho, **Então** a ferramenta recusa com erro próprio e mensagem que informa a versão encontrada e as aceitas.
5. **Dado** um recorte que foi gerado a partir de outro plano (a identificação do plano guardada no recorte não é a do plano informado), **Quando** o usuário pede o desenho, **Então** a ferramenta recusa com erro próprio, distinto dos demais, e mensagem que diz que o recorte não corresponde ao plano informado, mesmo que a área dos dois seja igual ou pareça compatível.
6. **Dado** um recorte que corresponde ao plano informado, mas cuja área não contém toda a área de que o plano precisa (por exemplo, gerado por uma versão da ferramenta com outra margem), **Quando** o usuário pede o desenho, **Então** a ferramenta recusa com erro próprio, distinto do de correspondência, e mensagem que compara a área que o plano exige com a que o recorte tem, sem gravar imagem alguma.
7. **Dado** um recorte exportado antes de a etapa 4 guardar a identificação do plano (sem esse campo), **Quando** o usuário pede o desenho, **Então** a ferramenta recusa com o erro de recorte inválido, com mensagem que diz que o recorte não traz a identificação do plano e que ele deve ser gerado de novo.
8. **Dado** um recorte cujo mapa base é formado por peças vetoriais (por exemplo, `pbf`), **Quando** o usuário pede o desenho, **Então** a ferramenta recusa **antes de desenhar qualquer quadro**, com erro próprio e mensagem que identifica o registro e o formato encontrado e diz que o desenho de peças vetoriais ainda não é suportado; peças em formato de imagem (PNG, JPG, WebP) são aceitas.
9. **Dado** um recorte em que nenhuma amostra de elevação tem valor, **Quando** o usuário pede o desenho, **Então** a ferramenta recusa com erro próprio e mensagem clara, pois não há terreno a desenhar nem referência de altura para a câmera.
10. **Dado** uma peça de mapa do recorte cujos bytes não são uma imagem legível, **Quando** o desenho a alcança, **Então** a ferramenta recusa com erro próprio de recorte inválido identificando a peça (registro, nível e posição), sem gravar imagem parcial; quadros já concluídos permanecem válidos.
11. **Dado** qualquer um dos casos acima, **Quando** a ferramenta recusa, **Então** nenhuma imagem é criada ou alterada no destino.

---

### Casos Extremos

- **Voo que cruza o meridiano de 180°**: os quadros são desenhados com as mesmas regras de qualquer outro voo; o terreno, as peças de mapa e o traçado de um lado e do outro do meridiano formam uma imagem contínua, sem emenda, salto ou deformação na junção.
- **Voo em latitudes altas (próximo aos polos)**: as formas do terreno e o traçado não são distorcidos pela convergência das longitudes (a escala em metros é a mesma nas duas direções da imagem), e o comportamento não muda por causa da latitude. Porção do terreno que está dentro da área do recorte mas além do limite de latitude do mapa base (onde não existem peças) aparece com a marcação de "sem imagem de mapa" e conta como buraco de mapa.
- **Proporções de imagem extremas** (retrato, ultralarga): a proporção muda apenas a largura do que se vê; o que ficar fora da área do recorte aparece como fundo neutro (ver História 3), nunca como terreno inventado.
- **Resolução maior que a de referência do recorte**: o recorte foi dimensionado para uma imagem de altura de referência; numa resolução maior as peças de mapa ficam menos nítidas, sem ser um erro nem um buraco.
- **Amostra sem valor sob o alvo, o traçado ou o marcador**: a altura da câmera é relativa ao chão sob o alvo; quando a amostra ali não tem valor, a ferramenta usa uma regra determinística e documentada para obter a altura de referência (a definir no planejamento), desenha o quadro normalmente, e o quadro entra na contagem de buraco de elevação. Nunca usa zero como se fosse elevação.
- **Marcador ou traçado encoberto pelo relevo**: eles obedecem à mesma oclusão do terreno (um morro mais próximo pode encobri-los; o marcador, um símbolo de tamanho fixo, some inteiro quando o terreno esconde o pixel do seu centro). Como a altura da câmera do plano é relativa ao chão sob o alvo, isso deve ser raro.
- **Marcador fora do campo de visão**: o quadro é desenhado normalmente, sem marcador visível; não é erro.
- **Diretório de destino com arquivos alheios**: nunca são alterados nem removidos (História 6).
- **Dois desenhos ao mesmo tempo no mesmo destino**: não é uma situação suportada; mesmo assim, a gravação atômica de cada imagem garante que nenhuma imagem parcial apareça.
- **Recorte com uma única peça de mapa ou uma única amostra de elevação**: é um recorte válido; o quadro é desenhado com o que há e o resto vira fundo neutro.
- **Falta de espaço em disco no meio do voo**: a execução termina com erro claro, sem imagem parcial; os quadros já gravados permanecem válidos e a retomada continua dali.
- **Arquivos de plano e recorte modificados entre duas execuções**: a retomada só reaproveita quadros do mesmo conjunto (mesmo plano, recorte e resolução); se algum dos dois mudou, o destino é tratado como de outro conjunto (História 6).

## Requisitos *(obrigatório)*

### Requisitos Funcionais

- **FR-001**: O sistema DEVE desenhar os quadros do voo a partir de exatamente dois arquivos informados pelo usuário — o plano de câmera exportado pela terceira etapa e o recorte de dados exportado pela quarta — sem acessar rede, sem ler os arquivos de dados registrados na segunda etapa, sem ler o trajeto GPS e sem alterar nenhum dos arquivos de entrada.
- **FR-002**: O sistema DEVE validar o plano e recusar, com os mesmos erros próprios e regras já definidos na etapa anterior, um plano inexistente, ilegível, que não seja um plano exportado, truncado, corrompido, de versão de formato desconhecida (informando a versão encontrada e as aceitas) ou incoerente consigo mesmo.
- **FR-003**: O sistema DEVE validar o recorte antes de desenhar e recusar, com erro próprio, distinto dos do plano, e mensagem clara, um recorte inexistente, ilegível, que não seja um recorte exportado, truncado, corrompido, de versão de formato desconhecida (informando a versão encontrada e as aceitas) ou incoerente consigo mesmo (por exemplo, quantidades do manifesto que não batem com o conteúdo, ou peças e grades referidas que não existem).
- **FR-004**: O sistema DEVE verificar, antes de desenhar qualquer quadro, que a área do recorte contém toda a área de que o plano precisa, calculada pela mesma regra que a quarta etapa usa para a área de interesse, e DEVE recusar com erro próprio, comparando as duas áreas, quando não contém.
- **FR-005**: O recorte exportado pela etapa 4 DEVE passar a guardar uma identificação do plano a partir do qual foi gerado, derivada do conteúdo do plano (dois planos de conteúdo igual têm a mesma identificação, planos diferentes têm identificações diferentes), sem data/hora nem dado do ambiente, de modo que a exportação continue idêntica byte a byte para o mesmo plano e os mesmos registros. Esta etapa DEVE conferir essa identificação com a do plano informado, antes de qualquer outra verificação de conteúdo, e DEVE recusar com erro próprio, distinto do de cobertura, um recorte cuja identificação não é a do plano; um recorte sem esse campo DEVE ser recusado como recorte inválido, com a orientação de gerá-lo de novo. O formato do recorte continua na mesma versão, pois o campo é acrescentado, não muda o significado de nenhum campo existente.
- **FR-006**: O sistema DEVE aceitar peças de mapa base em formato de imagem (PNG, JPG e WebP) e DEVE recusar, antes de desenhar qualquer quadro, um recorte cujo mapa base tenha peças vetoriais, com erro próprio e mensagem que identifica o registro e o formato e informa que o desenho vetorial ainda não é suportado.
- **FR-007**: O sistema DEVE recusar, com erro próprio e mensagem clara, um recorte em que nenhuma amostra de elevação tem valor.
- **FR-008**: Para cada quadro do plano, o sistema DEVE desenhar uma imagem do que a câmera enxerga, a partir da posição, da direção e da inclinação registradas no plano para aquele quadro: o relevo do terreno em perspectiva, construído a partir das amostras de elevação do recorte, com oclusão correta (o terreno mais próximo encobre o mais distante); vestido com as peças de mapa base do recorte, posicionadas onde o mapa as posiciona; com o traçado do trajeto sobre o terreno; e com o marcador da atividade na posição do marcador daquele quadro.
- **FR-009**: A altura da câmera de cada quadro DEVE ser obtida somando a altura do plano (relativa ao chão sob o alvo) à elevação do terreno no recorte, de modo que a câmera nunca fique embaixo do terreno que enxerga; o campo de visão vertical e as demais constantes de enquadramento DEVEM ser fixos e documentados, iguais para todos os quadros e todas as resoluções.
- **FR-010**: O traçado desenhado num quadro DEVE ser formado pelas posições do marcador dos quadros do plano do primeiro até o quadro desenhado (o trecho já percorrido), acompanhar o relevo (sem flutuar sobre ele nem afundar nele) e terminar exatamente no marcador daquele quadro. O trecho adiante do marcador NÃO DEVE ser desenhado: o traçado cresce a cada quadro, e o do quadro k contém o do quadro k−1. O primeiro quadro, com o marcador no início do trajeto, não tem linha a mostrar.
- **FR-011**: O marcador DEVE aparecer sobre o terreno, na posição que a projeção do plano indica, de forma visualmente distinta do traçado. O traçado e o marcador obedecem à mesma oclusão do terreno.
- **FR-012**: Onde uma peça de mapa registrada como ausente no recorte cobriria o terreno visível, o quadro DEVE mostrar uma marcação explícita de "sem imagem de mapa"; onde uma amostra de elevação não tem valor, o quadro DEVE mostrar uma marcação explícita de "sem elevação". As duas marcações DEVEM ser visualmente distintas entre si, de qualquer imagem de mapa e do fundo neutro. O sistema NÃO DEVE inventar imagem de mapa nem elevação (nem usar zero) para essas regiões.
- **FR-013**: Uma porção do terreno que está dentro da área do recorte, mas sem peça de mapa possível (além do limite de latitude do mapa base), DEVE ser marcada como "sem imagem de mapa". A parte do campo de visão que cai fora da área do recorte DEVE aparecer como fundo neutro, sem terreno inventado, e NÃO DEVE ser contada como buraco.
- **FR-014**: O sistema DEVE contar, para o resumo, um quadro como "com buraco de mapa" quando a parte visível do seu terreno inclui alguma região marcada como "sem imagem de mapa", e como "com buraco de elevação" quando inclui alguma região marcada como "sem elevação"; um quadro com as duas condições entra nas duas contagens.
- **FR-015**: O mesmo plano, o mesmo recorte e a mesma resolução DEVEM produzir sempre exatamente as mesmas imagens, byte a byte, independentemente do momento, da ordem em que os quadros são desenhados, de quantos quadros são desenhados por execução, de a execução ter sido retomada ou não e de quantos processadores estão em uso. As imagens NÃO DEVEM conter data/hora de geração, caminhos, nome de máquina, nem qualquer dado do ambiente.
- **FR-016**: O nome de cada imagem DEVE ser determinado apenas pelo número do quadro no plano (o índice do plano, a partir de 0), com uma largura fixa de dígitos definida pela quantidade de quadros do plano, de modo que a ordem alfabética dos nomes seja a ordem numérica dos quadros. A numeração e a ordem dos quadros DEVEM seguir o plano, sem lacunas nem duplicatas num desenho completo.
- **FR-017**: O sistema DEVE permitir desenhar um único quadro, informando o número dele; um número que não exista no plano DEVE ser recusado com erro próprio e mensagem que informa o número recebido e a faixa válida, sem gravar imagem alguma.
- **FR-018**: O sistema DEVE permitir desenhar todos os quadros do plano num diretório de destino, criando o diretório quando ele não existir; um destino que exista mas não seja um diretório, ou que não possa ser gravado, DEVE ser recusado com erro próprio e mensagem clara, sem gravar imagem alguma.
- **FR-018a**: O quadro isolado (FR-017) DEVE ser gravado num arquivo de imagem escolhido pelo usuário, à parte do diretório de destino do voo, e NÃO DEVE fazer parte de nenhum conjunto de quadros (FR-021 a FR-023 não se aplicam a ele). Se o arquivo já existe, o sistema DEVE recusá-lo por padrão, com erro próprio e mensagem que diz como sobrescrever, deixando-o intacto; com a opção explícita de sobrescrita, o arquivo é substituído por inteiro. O local que não puder ser gravado (pasta inexistente, sem permissão) DEVE ser recusado com o mesmo erro de destino inválido. O quadro isolado DEVE ser idêntico, byte a byte, ao mesmo quadro desenhado como parte do voo inteiro na mesma resolução.
- **FR-019**: O sistema DEVE permitir escolher a resolução das imagens (largura e altura, em pixels), com um padrão documentado quando o usuário não escolher; DEVE recusar, com erro próprio e mensagem que aponta o valor recebido e os limites documentados, uma resolução fora dos limites, com dimensão ímpar, zero, negativa ou não numérica; e as imagens DEVEM ter exatamente a resolução pedida. O enquadramento (o trecho vertical de terreno visto e a posição relativa do marcador) DEVE ser o mesmo em qualquer resolução de mesma proporção.
- **FR-020**: Durante o desenho de vários quadros, o sistema DEVE informar o progresso — quantos quadros já foram concluídos, de quantos, e o tempo decorrido — atualizado a cada quadro concluído; a informação de progresso NÃO DEVE alterar o conteúdo das imagens.
- **FR-021**: Por padrão, o sistema NÃO DEVE sobrescrever imagens existentes no destino: uma imagem existente que pertença ao mesmo conjunto (mesmo plano, mesmo recorte e mesma resolução) DEVE ser mantida sem ser redesenhada, e só os quadros que faltam DEVEM ser desenhados — é assim que o desenho é retomado depois de uma interrupção.
- **FR-022**: O sistema DEVE reconhecer a que conjunto pertencem as imagens existentes no destino e, por padrão, DEVE recusar, com erro próprio e mensagem que explica como proceder, um destino que contenha quadros de outro conjunto (outro plano, outro recorte ou outra resolução), sem alterá-lo; para isso cada imagem carrega, dentro dela, a identificação do conjunto a que pertence (plano, recorte, resolução e versão do desenho), de modo que o destino contém somente imagens, sem arquivo de registro à parte.
- **FR-023**: O sistema DEVE oferecer uma opção explícita de sobrescrita: com ela, todos os quadros pedidos são redesenhados e substituem os existentes, e, ao final de um desenho completo, o destino contém somente os quadros do plano informado (quadros de um conjunto anterior, cujos números o plano atual não tem, não permanecem). Arquivos do destino que a ferramenta não reconhece como quadros seus NUNCA DEVEM ser alterados nem removidos.
- **FR-024**: Cada imagem DEVE ser publicada inteira ou não ser publicada: uma interrupção em qualquer momento (cancelamento pelo usuário, falha, falta de espaço) NUNCA DEVE deixar imagem parcial ou ilegível; imagens já concluídas permanecem válidas; e, numa sobrescrita, a imagem anterior de mesmo nome permanece intacta se a nova não puder ser gravada.
- **FR-025**: Quando o usuário interrompe a execução, o sistema DEVE encerrar de forma ordenada, imprimir o resumo do que foi feito, deixando claro que o desenho foi interrompido, e sair com um código de saída distinto do de sucesso.
- **FR-026**: Ao final de cada execução, o sistema DEVE imprimir um resumo com: a quantidade de quadros pedidos; quantos foram desenhados nesta execução e quantos foram mantidos por já existirem; a resolução; o tempo gasto; e quantos dos quadros desenhados nesta execução tiveram buraco de mapa e quantos tiveram buraco de elevação. O tempo gasto é informação de execução e NÃO DEVE aparecer nas imagens.
- **FR-027**: Todos os erros de negócio (plano ilegível, inválido, de versão desconhecida ou incoerente; recorte ilegível, inválido, de versão desconhecida ou incoerente; recorte que não cobre a área do plano; recorte que não corresponde ao plano; peças vetoriais; recorte sem nenhuma elevação com valor; peça de mapa ilegível; número de quadro inválido; resolução inválida; destino inválido; arquivo do quadro isolado já existente; destino com quadros de outro conjunto) DEVEM ser distinguíveis entre si e dos erros das etapas anteriores, comunicados com mensagem clara e código de saída próprio. Quando o erro for de recusa, nenhuma imagem é criada ou alterada no destino.
- **FR-028**: O comportamento DEVE ser idêntico para qualquer trajeto em qualquer lugar do planeta, incluindo áreas que cruzam o meridiano de 180° e latitudes altas, sem tratamento especial ou privilegiado de região, e sem embutir dados de nenhuma região.
- **FR-029**: A ferramenta NÃO DEVE, nesta etapa, codificar o vídeo, incluir sobreposições de texto ou de estatísticas, incluir áudio, simular iluminação e sombras realistas, desenhar peças vetoriais, nem oferecer interface gráfica ou API.

### Entidades-Chave *(incluir se a funcionalidade envolver dados)*

- **Plano de Câmera Exportado**: o arquivo da terceira etapa (versionado): parâmetros, resumo e todos os quadros, cada um com o número, a posição da câmera, a direção, a inclinação e o marcador. Entrada desta etapa; validado como dado externo, pelas regras já existentes.
- **Recorte de Dados Exportado**: o arquivo da quarta etapa (versionado): a área coberta, as grades de elevação, as peças de mapa base (com o formato delas), as peças ausentes, a procedência e o resumo. Entrada desta etapa; validado como dado externo.
- **Identificação do Plano**: valor derivado do conteúdo de um plano exportado, guardado no recorte que foi gerado a partir dele.
- **Correspondência Plano-Recorte**: a relação entre um plano e um recorte que foi gerado a partir dele; verificada comparando a identificação do plano guardada no recorte com a do plano informado, e, depois, a cobertura da área que o plano exige.
- **Quadro Desenhado**: a imagem do que a câmera vê num instante do plano, identificada pelo número do quadro e com a resolução escolhida.
- **Marcação de Falta**: o sinal visual explícito, dentro do quadro, de que ali não há dado — "sem imagem de mapa" ou "sem elevação" —, distinto do fundo neutro (fora do recorte).
- **Quadro Isolado**: uma imagem de um único quadro, gravada num arquivo escolhido pelo usuário para conferir o enquadramento; não pertence a nenhum conjunto de quadros.
- **Conjunto de Quadros**: os quadros de um diretório de destino que pertencem ao mesmo plano, recorte e resolução; é o que define se imagens existentes podem ser mantidas, ou se o destino é de outro conjunto.
- **Resolução**: a largura e a altura, em pixels, das imagens, dentro dos limites documentados.
- **Resumo do Desenho**: quadros pedidos, desenhados e mantidos; resolução; tempo gasto; quadros com buraco de mapa e com buraco de elevação; e, se interrompido, a indicação disso.

## Critérios de Sucesso *(obrigatório)*

### Resultados Mensuráveis

- **SC-001**: Um usuário consegue, informando só o plano, o recorte e o número do quadro, ver um quadro isolado na resolução padrão em menos de 10 segundos, para um voo típico (trajeto de até 50 km com os parâmetros padrão de plano e de recorte), num computador pessoal comum.
- **SC-002**: Desenhar o mesmo voo 100 vezes seguidas, com o mesmo plano, o mesmo recorte e a mesma resolução, produz 100 conjuntos de imagens idênticos byte a byte; e cada quadro desenhado sozinho, dentro do voo inteiro ou numa execução retomada é idêntico byte a byte ao mesmo quadro de qualquer outra forma.
- **SC-003**: Em 100% dos voos testados, o destino de um desenho completo contém exatamente uma imagem por quadro do plano, numeradas de 0 a N−1 sem lacunas nem duplicatas, e ordená-las pelo nome produz a ordem numérica.
- **SC-004**: Em 100% dos quadros testados, o marcador aparece no centro da posição em que a projeção do plano o coloca, com erro de no máximo 1 pixel.
- **SC-005**: Em 100% dos quadros testados com região sem imagem de mapa ou sem elevação dentro do campo de visão, a marcação explícita correspondente aparece; em 100% dos quadros sem essas regiões, nenhuma marcação de falta aparece; e as contagens do resumo coincidem exatamente com os quadros que trazem cada marcação.
- **SC-006**: Em 100% das interrupções provocadas em pontos diferentes do desenho (cancelamento, falha de gravação), nenhuma imagem parcial ou ilegível permanece no destino, e retomar o mesmo pedido produz, ao final, um conjunto idêntico ao de uma execução sem interrupção, sem redesenhar os quadros já concluídos.
- **SC-007**: Em 100% dos casos de recusa testados (plano ou recorte ilegível, inválido, de versão desconhecida ou incoerente; recorte que não cobre; recorte que não corresponde; peças vetoriais; sem elevação alguma; número de quadro inválido; resolução inválida; destino inválido; destino de outro conjunto sem sobrescrita), a mensagem identifica o problema, o código de saída é próprio e nenhuma imagem é criada ou alterada; a recusa por peças vetoriais, por cobertura e por correspondência ocorre em menos de 5 segundos, sem desenhar quadro algum.
- **SC-008**: Em 100% das resoluções dentro dos limites documentados, as imagens têm exatamente a largura e a altura pedidas, e o enquadramento em duas resoluções de mesma proporção coincide (o marcador cai na mesma posição relativa da imagem, com diferença de no máximo 1 pixel na menor delas).
- **SC-009**: Voos equivalentes deslocados para regiões diferentes do planeta (inclusive cruzando o meridiano de 180° e em latitudes acima de 80°) produzem quadros com as mesmas propriedades — mesmo enquadramento e mesmas regras de marcação —, sem emenda, salto ou descontinuidade na junção do meridiano e sem distorção do terreno ou do traçado por causa da latitude.
- **SC-010**: Um usuário consegue, apenas lendo o resumo, responder em menos de 1 minuto: quantos quadros foram desenhados e quantos já existiam, qual a resolução, quanto tempo levou, quantos quadros tiveram buraco de mapa e quantos de elevação, e se a execução foi interrompida.
- **SC-011**: Durante um desenho de muitos quadros, o usuário vê o progresso atualizado a cada quadro concluído, com a quantidade concluída, o total e o tempo decorrido, e nenhum quadro leva mais de 15 segundos na resolução padrão para um voo típico.
- **SC-012**: Um voo típico de 45 segundos a 30 quadros por segundo (1350 quadros) na resolução padrão é desenhado por inteiro em até 45 minutos num computador pessoal comum.

## Suposições

- O usuário já cumpriu as etapas anteriores: registrou os dados (etapa 2), exportou o plano (etapa 3) e exportou o recorte daquele mesmo plano (etapa 4). Esta etapa não regera o plano nem o recorte; usa os arquivos como estão. Os comandos das etapas anteriores (`inspect`, `geodata register|list|remove|check|slice|elevation`, `plan`) continuam se comportando como antes, com a única exceção da identificação do plano no recorte exportado (ver "Correspondência plano-recorte").
- A leitura e a validação do plano reaproveitam a da etapa 4 (mesmos erros); o recorte exportado passa a ter um leitor próprio, que valida o arquivo por completo antes de desenhar. O recorte exportado pela etapa 4 já traz tudo de que o desenho precisa: as grades de elevação, as peças de mapa como foram lidas dos arquivos originais (com o formato de cada uma), as peças ausentes e a área.
- **Correspondência plano-recorte**: decidida pela identificação do plano guardada no recorte (Clarificação de 2026-09-26). Isso altera a etapa 4 (`geodata slice --export`), que passa a gravar o campo, com o contrato do arquivo de recorte e a documentação da etapa 4 atualizados nesta feature; é a única mudança de comportamento das etapas anteriores. Recortes exportados antes disso não têm o campo e são recusados, com orientação de gerá-los de novo. Como a identificação é derivada do conteúdo do plano, a forma exata (por exemplo, sobre os bytes do arquivo ou sobre o conteúdo já validado) é definida no planejamento técnico. A verificação de cobertura (FR-004) continua como proteção adicional, para um recorte do mesmo plano gerado com outra margem.
- **Formato de saída**: cada quadro é uma imagem PNG (sem perdas, legível por qualquer ferramenta de vídeo e de codificação determinística); a etapa seguinte lê o diretório de destino pela ordem dos nomes. O nome do arquivo é `frame_NNNNNN.png` (número do quadro no plano, 6 dígitos), e a identificação do conjunto vai dentro de cada imagem (`contracts/frame-files.md`). Quando o destino contiver quadros de outro conjunto, a ferramenta o reconhece por essa identificação, e não por comparar bytes nem pelo nome; um arquivo que tenha nome de quadro mas não traga a identificação não é tratado como quadro da ferramenta.
- **Resolução**: o padrão é 1080 × 1920 pixels, com largura e altura pares (a etapa de vídeo seguinte exige dimensões pares para os codecs de uso comum), cada uma de 180 a 3840, e no máximo 8 294 400 pixels no total (o de 3840 × 2160), o que aceita retrato e ultralargo. Estes limites, decididos no planejamento técnico (reduzindo a referência inicial de 8K, que um desenho por software não sustenta), podem ser revistos sem alterar o comportamento descrito aqui. O nível de detalhe do mapa no recorte da etapa 4 foi dimensionado para uma imagem de 1080 pixels de altura; em resoluções maiores a nitidez do mapa cai sem ser erro.
- **Traçado**: como só o plano e o recorte são usados, o traçado do quadro k é formado pelas posições do marcador dos quadros 0 a k do plano (que amostram o trajeto tratado); ele não é o trajeto GPS original, e por isso um quadro isolado só precisa do plano para saber o que desenhar (Clarificação de 2026-09-26).
- **Perspectiva e oclusão**: o terreno é desenhado com projeção em perspectiva, com o campo de visão vertical fixo e documentado no planejamento técnico; o desenho é uma superfície contínua construída a partir da grade de elevação do recorte, com oclusão correta. Não há iluminação nem sombras realistas (fora de escopo): o terreno mostra a imagem do mapa como ela é, sem sombreamento. O fundo neutro (fora do recorte) tem uma cor fixa documentada, que não se confunde com as marcações de falta.
- **Referência de altura da câmera**: a etapa 3 gerou a altura da câmera relativa ao chão sob o alvo (não sobre o nível do mar); esta etapa soma a elevação do recorte. Quando a amostra sob o alvo não tem valor, a regra determinística de referência é definida no planejamento técnico, sempre com o quadro contado como buraco de elevação.
- **Resumo e retomada**: as contagens de buraco do resumo referem-se aos quadros desenhados **na execução**; os quadros mantidos por já existirem são informados à parte, sem serem recontados. Um quadro só é considerado "mantido" se pertencer ao mesmo conjunto (plano, recorte e resolução).
- **Determinismo**: vale para o mesmo binário da ferramenta; a igualdade byte a byte entre arquiteturas de processador diferentes só é prometida se o planejamento técnico conseguir garanti-la; o desenho não pode depender de fuso horário, idioma, quantidade de processadores nem da ordem de execução.
- **Desempenho**: as metas de tempo dos critérios de sucesso são referências iniciais para um computador pessoal comum e voos típicos, a serem confirmadas no planejamento técnico.
- **Dados reais do usuário**: o mapa base real do usuário (`resources/`, do BBBike) é formado por peças vetoriais (`pbf`), que esta etapa recusa por escopo; a validação real desta etapa exercita, portanto, a recusa vetorial com esses dados e o desenho em si com um mapa base sintético em imagem sobre o relevo e o trajeto reais. O desenho de peças vetoriais fica para uma etapa futura.
