# Especificação de Funcionalidade: Montagem do Vídeo do Voo

**Branch da Funcionalidade**: `006-video-assembly`

**Criado em**: 2026-09-26

**Status**: Implementada, com `quickstart.md` validado por inteiro com o `ffmpeg` real

**Entrada**: Descrição do usuário: "Sexta etapa do Sobrevoo. Esta etapa cobre só a montagem do vídeo a partir dos quadros já desenhados pela quinta etapa. Nada de sobreposições de texto ou estatísticas, nem áudio. A partir do plano de câmera exportado na terceira etapa e do diretório de quadros desenhado na quinta, a ferramenta junta os quadros, na ordem do plano, num único arquivo de vídeo pronto para assistir e publicar, usando a taxa de quadros do próprio plano. O usuário consegue: gerar o vídeo informando o diretório de quadros e o destino; escolher a qualidade por níveis nomeados; acompanhar o progresso da codificação; e ver um resumo ao final (quantidade de quadros, duração, resolução, taxa de quadros, qualidade, tamanho do arquivo e tempo gasto). Requisitos: usar exclusivamente o plano e os quadros informados, sem rede e sem redesenhar quadro algum; conferir, antes de começar, que os quadros do diretório são do plano informado, que estão todos presentes, sem lacuna nem duplicata, e que todos têm a mesma resolução, recusando com erro próprio e mensagem que diz exatamente o que falta ou o que destoa; recusar com mensagem clara e acionável quando o codificador de vídeo necessário não estiver disponível na máquina, dizendo o que instalar; a duração do vídeo resultante corresponde à quantidade de quadros dividida pela taxa de quadros do plano, e a ordem dos quadros é a do plano; o vídeo não carrega data/hora de geração, caminho, nome de máquina nem qualquer dado do ambiente, e a mesma entrada com a mesma qualidade e o mesmo codificador produz sempre o mesmo arquivo; por padrão recusar um destino que já exista, com opção explícita de sobrescrita, e nunca deixar arquivo parcial em caso de falha ou interrupção; ao ser interrompido, encerrar de forma ordenada, dizer que foi interrompido e sair com código próprio; comportamento idêntico para qualquer voo, em qualquer lugar do planeta. Fora de escopo: desenhar ou redesenhar quadros, sobreposições de texto ou de estatísticas, áudio, trilha sonora, publicação em qualquer serviço, edição do vídeo depois de gerado, interface gráfica e API."

## Clarificações

### Sessão 2026-09-27

- Q: Como a etapa 6 deve saber que os quadros do diretório são do plano informado? → A: A etapa 5 passa a gravar em cada quadro, além da identificação do conjunto, a identificação do plano de que ele veio (opção A), e esta etapa a confere (FR-004 e FR-004a). Isso altera a saída da etapa 5; quadros desenhados antes do campo não o têm e precisam ser desenhados de novo.

## Cenários de Usuário e Testes *(obrigatório)*

### História de Usuário 1 - Montar o vídeo a partir do plano e dos quadros (Prioridade: P1)

Como usuário do Sobrevoo, eu já exportei o plano de câmera de um trajeto (terceira etapa) e já desenhei todos os quadros desse plano num diretório (quinta etapa). Quero informar o plano, o diretório de quadros e o destino, e receber um único arquivo de vídeo, pronto para assistir e publicar, com os quadros na ordem do plano e na taxa de quadros do plano.

**Por que esta prioridade**: é o núcleo da etapa e o produto final do Sobrevoo até aqui: transforma uma pasta de imagens num vídeo que se assiste. Todas as demais histórias conferem, dimensionam ou protegem esta montagem.

**Teste Independente**: pode ser totalmente testado com um plano curto e o diretório dos quadros desse plano (desenhados com dados de exemplo), gerando o vídeo e verificando, com uma ferramenta de inspeção de mídia, que o arquivo existe, abre, tem a quantidade de quadros do plano, a duração e a taxa de quadros esperadas e a resolução das imagens.

**Cenários de Aceitação**:

1. **Dado** um plano válido e um diretório com todos os quadros desenhados a partir dele, **Quando** o usuário pede a montagem do vídeo para um destino que não existe, **Então** a ferramenta grava um único arquivo de vídeo nesse destino, que abre e reproduz nos reprodutores de uso comum.
2. **Dado** o vídeo gerado, **Quando** o usuário o inspeciona, **Então** ele tem exatamente um quadro de vídeo por imagem do diretório, na ordem numérica dos quadros do plano (do primeiro ao último), com a mesma largura e a mesma altura das imagens.
3. **Dado** um plano com N quadros e taxa de quadros F, **Quando** o vídeo é gerado, **Então** a taxa de quadros do vídeo é F (inclusive quando F não é inteiro, como 29,97) e a duração é N ÷ F segundos, sem acréscimo nem corte de quadros.
4. **Dado** o vídeo gerado, **Quando** o usuário o assiste, **Então** ele não contém áudio, sobreposição de texto nem de estatísticas: só o que as imagens mostram.
5. **Dado** que o vídeo foi gerado, **Quando** o usuário olha o diretório de quadros, o plano e qualquer outro arquivo de entrada, **Então** nada neles foi alterado, e nenhum quadro foi redesenhado.
6. **Dado** o diretório de quadros que contém, além dos quadros, arquivos que a ferramenta não reconhece como quadros seus (por exemplo, notas ou miniaturas), **Quando** o usuário pede a montagem, **Então** esses arquivos são ignorados e não alterados, e não entram no vídeo.

---

### História de Usuário 2 - Conferir os quadros antes de começar (Prioridade: P2)

Como usuário do Sobrevoo, montar um vídeo leva tempo, e um diretório errado produziria, sem aviso, um vídeo errado: quadros de outro voo, com um buraco no meio, repetidos ou de tamanhos diferentes. Quero que a ferramenta confira os quadros **antes** de gastar tempo com a codificação e me diga exatamente o que falta ou o que destoa.

**Por que esta prioridade**: é a proteção do resultado. O caminho feliz (P1) entrega o vídeo, mas sem esta conferência um diretório errado geraria um vídeo errado em silêncio, e o usuário só perceberia depois de assistir.

**Teste Independente**: pode ser testado com um diretório completo e correto, e com variações dele — um quadro removido, um quadro de outro voo, um quadro com resolução diferente, quadros a mais, um quadro truncado — verificando, para cada uma, que a mensagem diz exatamente o problema, que o código de saída é próprio e que nenhum vídeo é criado.

**Cenários de Aceitação**:

1. **Dado** um diretório em que faltam quadros (numeração com lacunas ou terminando antes do último quadro do plano), **Quando** o usuário pede a montagem, **Então** a ferramenta recusa com erro próprio e mensagem que lista quais números faltam (resumindo quando são muitos, sem esconder o total), sem criar arquivo de vídeo.
2. **Dado** um diretório com quadros que o plano não tem (número maior ou igual à quantidade de quadros do plano), **Quando** o usuário pede a montagem, **Então** a ferramenta recusa com erro próprio e mensagem que diz quais números sobram e quantos quadros o plano tem.
3. **Dado** um diretório que contém, para o mesmo número de quadro, mais de uma imagem reconhecida como quadro (duplicata), **Quando** o usuário pede a montagem, **Então** a ferramenta recusa com erro próprio e mensagem que diz qual número está duplicado e quais arquivos o repetem.
4. **Dado** um diretório em que os quadros não têm todos a mesma resolução, **Quando** o usuário pede a montagem, **Então** a ferramenta recusa com erro próprio e mensagem que diz qual é a resolução esperada (a do primeiro quadro, ou a da maioria) e quais quadros destoam, com a resolução de cada um.
5. **Dado** um diretório com quadros de outro voo (de outro plano), ou de mais de um conjunto misturados, **Quando** o usuário pede a montagem, **Então** a ferramenta recusa com erro próprio, distinto dos demais, e mensagem que diz que os quadros não pertencem ao plano informado (ou que pertencem a conjuntos diferentes), mesmo que a quantidade e a resolução coincidam por acaso.
6. **Dado** um diretório de quadros desenhados antes de a etapa 5 gravar a identificação do plano (quadros sem esse campo), **Quando** o usuário pede a montagem, **Então** a ferramenta recusa com erro próprio, com mensagem que diz que os quadros não trazem a identificação do plano e que devem ser desenhados de novo com a versão atual da ferramenta, sem criar arquivo de vídeo.
7. **Dado** um quadro que não é uma imagem legível ou que está truncado (por exemplo, por uma execução interrompida da etapa anterior), **Quando** o usuário pede a montagem, **Então** a ferramenta recusa com erro próprio e mensagem que identifica o arquivo, sem criar arquivo de vídeo.
8. **Dado** um diretório que não existe, que não é um diretório, que não pode ser lido ou que não contém nenhum quadro reconhecido, **Quando** o usuário pede a montagem, **Então** a ferramenta recusa com erro próprio e mensagem clara que diz qual é o problema com o diretório.
9. **Dado** um plano inexistente, ilegível, que não seja um plano exportado, truncado, corrompido, de versão de formato desconhecida ou incoerente consigo mesmo, **Quando** o usuário pede a montagem, **Então** a ferramenta recusa com os mesmos erros próprios que as etapas anteriores já usam para o plano, com a mesma mensagem que diz o que está errado.
10. **Dado** qualquer um dos casos acima, **Quando** a ferramenta recusa, **Então** a recusa acontece antes de qualquer codificação, e nenhum arquivo é criado ou alterado no destino.
11. **Dado** um diretório com quadros de dimensões que os codificadores de vídeo de uso comum não aceitam (largura ou altura ímpar), **Quando** o usuário pede a montagem, **Então** a ferramenta recusa com erro próprio e mensagem que diz as dimensões encontradas e que elas precisam ser pares, em vez de falhar de forma obscura no meio da codificação.

---

### História de Usuário 3 - Saber quando falta o codificador de vídeo (Prioridade: P3)

Como usuário do Sobrevoo, montar um vídeo exige um codificador de vídeo instalado na minha máquina. Se ele não estiver lá (ou não tiver o formato de que a ferramenta precisa), quero uma mensagem clara que diga o que está faltando e o que instalar, em vez de um erro técnico do sistema.

**Por que esta prioridade**: sem o codificador nada é gerado, mas a falta dele é uma situação de ambiente, não de dados; o que importa é que a mensagem seja acionável e que a recusa aconteça cedo, sem deixar rastro.

**Teste Independente**: pode ser testado numa máquina (ou num ambiente simulado) sem o codificador, e com um codificador que não suporte o formato exigido, verificando a mensagem, o código de saída e a ausência de qualquer arquivo criado.

**Cenários de Aceitação**:

1. **Dado** uma máquina em que o codificador de vídeo necessário não está instalado ou não é encontrado, **Quando** o usuário pede a montagem, **Então** a ferramenta recusa com erro próprio e mensagem que diz qual programa falta, o que instalar e como confirmar que ficou disponível, sem criar arquivo algum.
2. **Dado** um codificador instalado que não suporta o formato de vídeo necessário, **Quando** o usuário pede a montagem, **Então** a ferramenta recusa com erro próprio, com mensagem que diz o que o codificador encontrado não oferece e o que instalar ou trocar.
3. **Dado** um codificador presente e apto, **Quando** o usuário pede a montagem, **Então** a ferramenta o usa sem exigir nenhuma configuração, e o resumo informa qual codificador foi usado.
4. **Dado** que o codificador não está disponível **e** que os quadros (ou o destino) também têm problema, **Quando** o usuário pede a montagem, **Então** a ferramenta recusa sempre primeiro pelo problema nas entradas (plano e quadros), depois pelo do destino, e só por último pelo do codificador; assim, o usuário que resolve um problema não é surpreendido por outro que já existia e podia ter sido dito antes.

---

### História de Usuário 4 - Escolher a qualidade por níveis nomeados (Prioridade: P4)

Como usuário do Sobrevoo, quero escolher a qualidade do vídeo por níveis com nomes (por exemplo, para conferir rápido, para publicar, para guardar com a máxima fidelidade), sem precisar conhecer parâmetros de codificação. Quando não escolho, quero um padrão sensato para publicar.

**Por que esta prioridade**: a montagem já funciona com o nível padrão (P1 a P3); escolher a qualidade é uma comodidade importante que troca tamanho e tempo por fidelidade, mas não bloqueia as demais.

**Teste Independente**: pode ser testado gerando o vídeo do mesmo plano e dos mesmos quadros em cada nível e comparando tamanho de arquivo e tempo de codificação, e pedindo um nível inexistente e verificando a recusa.

**Cenários de Aceitação**:

1. **Dado** que o usuário não informa a qualidade, **Quando** ele pede a montagem, **Então** o vídeo é gerado no nível padrão documentado, e o resumo informa o nível usado.
2. **Dado** os níveis nomeados documentados, **Quando** o usuário gera o mesmo vídeo em cada um, **Então** um nível mais alto produz um arquivo maior ou igual ao de um nível mais baixo, sem alterar a duração, a resolução, a taxa de quadros nem a ordem dos quadros.
3. **Dado** um nível que não existe, **Quando** o usuário pede a montagem, **Então** a ferramenta recusa como erro de uso, com mensagem que lista os níveis aceitos, sem criar arquivo algum.
4. **Dado** o mesmo nível pedido de novo, **Quando** o usuário gera o vídeo, **Então** o resultado é o mesmo (ver História 7).

---

### História de Usuário 5 - Acompanhar o progresso e ver o resumo (Prioridade: P5)

Como usuário do Sobrevoo, um voo tem centenas ou milhares de quadros e a codificação leva tempo. Quero acompanhar o andamento enquanto ela roda e, ao final, ver um resumo que me diga o que foi gerado.

**Por que esta prioridade**: sem isso a codificação é opaca, mas o resultado de P1 já é correto sem este recurso.

**Teste Independente**: pode ser testado gerando o vídeo de um plano com muitos quadros e verificando o progresso durante a execução, e o resumo ao final, contra os valores conhecidos da entrada e do arquivo gerado.

**Cenários de Aceitação**:

1. **Dado** uma montagem em andamento, **Quando** o usuário a acompanha, **Então** a ferramenta informa continuamente quantos quadros já foram codificados, de quantos, e o tempo decorrido, sem alterar o conteúdo do vídeo.
2. **Dado** que a montagem terminou, **Quando** o usuário lê o resumo, **Então** ele informa: a quantidade de quadros; a duração do vídeo; a resolução; a taxa de quadros; o nível de qualidade; o tamanho do arquivo gerado; o tempo gasto; e o codificador usado.
3. **Dado** o resumo, **Quando** o usuário compara a duração, a resolução, a taxa e a quantidade de quadros com o que uma ferramenta de inspeção de mídia mostra para o arquivo, **Então** os valores coincidem.
4. **Dado** que a montagem falhou ou foi recusada, **Quando** a ferramenta encerra, **Então** ela não imprime um resumo de sucesso: imprime o erro (ou, se interrompida, o resumo de interrupção da História 6).

---

### História de Usuário 6 - Proteger o destino e não deixar arquivo parcial (Prioridade: P6)

Como usuário do Sobrevoo, quero que a ferramenta nunca sobrescreva um vídeo existente sem que eu peça explicitamente, nunca deixe um arquivo de vídeo pela metade e, quando eu a interrompo, encerre com ordem e me diga que foi interrompida.

**Por que esta prioridade**: é a proteção do resultado, análoga à exportação segura das etapas 3 e 4 e à gravação dos quadros da etapa 5. O caminho feliz funciona sem ela, mas sem ela um erro de digitação apaga um vídeo pronto, e uma falha no meio deixa um arquivo que parece um vídeo mas não toca.

**Teste Independente**: pode ser testado com destinos que já existem, com e sem a opção de sobrescrita, provocando uma falha no meio da codificação e interrompendo a execução em pontos diferentes.

**Cenários de Aceitação**:

1. **Dado** um destino que já existe, **Quando** o usuário pede a montagem sem a opção de sobrescrita, **Então** a ferramenta recusa com erro próprio e mensagem que informa que o arquivo já existe e como sobrescrevê-lo, deixando o arquivo intacto e sem iniciar a codificação.
2. **Dado** um destino que já existe, **Quando** o usuário pede a montagem com a opção explícita de sobrescrita, **Então** o arquivo é substituído por inteiro pelo novo vídeo.
3. **Dado** que a codificação falha no meio (por exemplo, falta de espaço em disco ou falha do codificador), **Quando** a falha acontece, **Então** nenhum arquivo parcial ou ilegível permanece no destino e nenhum resíduo (arquivo temporário) fica no diretório do destino; se havia um arquivo anterior no destino (caso de sobrescrita), ele permanece intacto.
4. **Dado** uma montagem em andamento, **Quando** o usuário a interrompe (por exemplo, com Ctrl+C), **Então** a ferramenta encerra o codificador de forma ordenada, não deixa arquivo parcial nem resíduo, imprime uma mensagem que deixa claro que a montagem foi interrompida (e quantos quadros já tinham sido codificados), e sai com um código de saída próprio, distinto do de sucesso e dos demais erros.
5. **Dado** um destino cujo diretório não existe, ou que não pode ser gravado (sem permissão), ou que é um diretório, **Quando** o usuário pede a montagem, **Então** a ferramenta recusa com erro próprio e mensagem clara sobre o destino, sem iniciar a codificação.
6. **Dado** uma montagem interrompida, **Quando** o usuário repete o mesmo pedido, **Então** a montagem recomeça do zero (não há retomada) e produz o mesmo vídeo que uma execução sem interrupção.

---

### História de Usuário 7 - Obter sempre o mesmo vídeo, sem dado do ambiente (Prioridade: P7)

Como usuário do Sobrevoo, quero que o mesmo plano, os mesmos quadros e a mesma qualidade produzam sempre o mesmo arquivo, para poder comparar, versionar e verificar meus vídeos, e quero que o arquivo não vaze dados da minha máquina quando eu o publicar.

**Por que esta prioridade**: é a garantia que mantém o padrão das etapas anteriores (mesma entrada, mesma saída, byte a byte), e uma proteção de privacidade sobre um arquivo que costuma ser compartilhado. O vídeo é gerado sem ela, mas sem ela não há como verificar que nada mudou.

**Teste Independente**: pode ser testado gerando o mesmo vídeo várias vezes, em momentos e diretórios diferentes, comparando os arquivos byte a byte, e inspecionando os metadados do arquivo em busca de data, caminho, nome de máquina ou usuário.

**Cenários de Aceitação**:

1. **Dado** o mesmo plano, os mesmos quadros, a mesma qualidade e o mesmo codificador, **Quando** o usuário gera o vídeo duas vezes (em momentos, diretórios de trabalho e destinos diferentes), **Então** os dois arquivos são idênticos, byte a byte.
2. **Dado** o vídeo gerado, **Quando** o usuário inspeciona os metadados do arquivo, **Então** não há data ou hora de geração, caminho de arquivo, nome de máquina, nome de usuário, nem qualquer outro dado do ambiente em que foi gerado.
3. **Dado** o mesmo plano e os mesmos quadros, **Quando** o usuário gera o vídeo em níveis de qualidade diferentes, **Então** só o nível muda o arquivo: a ordem, a duração, a resolução e a taxa de quadros continuam as mesmas.
4. **Dado** dois voos equivalentes em regiões diferentes do planeta (inclusive um que cruza o meridiano de 180° e um em latitudes altas), **Quando** o usuário gera o vídeo de cada um, **Então** as mesmas regras se aplicam a ambos, sem tratamento especial de região.

---

### Casos Extremos

- **Diretório de quadros gerado com plano de mesma quantidade de quadros, mas de outro voo**: recusado pela conferência de correspondência (História 2, cenário 5), nunca montado em silêncio.
- **Quadros de uma versão anterior da etapa 5 (sem a identificação do plano)**: recusados, com a orientação de desenhá-los de novo (História 2, cenário 6).
- **Diretório com quadros de duas resoluções (dois desenhos misturados)**: recusado, com a lista dos que destoam (História 2, cenário 4).
- **Diretório com uma única imagem e plano de um único quadro**: é um voo válido; o vídeo tem um só quadro e duração de 1 ÷ F segundos.
- **Voo muito longo (até o máximo de quadros que o plano aceita)**: a montagem não depende de manter todos os quadros na memória ao mesmo tempo; o tempo cresce com a quantidade de quadros, mas o pedido não falha por causa do tamanho.
- **Taxa de quadros não inteira** (por exemplo, 29,97): respeitada exatamente; a duração continua N ÷ F.
- **Resolução muito alta ou muito baixa dentro do que a etapa 5 aceita**: a montagem usa a resolução dos quadros; se o codificador não suportar essa resolução, a ferramenta recusa com mensagem que diz o limite encontrado, em vez de falhar no meio.
- **Plano com proporção diferente da dos quadros** (por exemplo, quadros desenhados num plano vertical mas com resolução horizontal): não é erro desta etapa; a ferramenta monta o vídeo com a resolução dos quadros (o aviso de proporção já é dado pela etapa 5).
- **Arquivos alheios no diretório de quadros** (que não trazem a identificação de quadro da ferramenta): ignorados, nunca alterados nem removidos, e não contam como lacuna, duplicata nem quadro a mais (História 1, cenário 6).
- **Quadro com nome de quadro mas sem a identificação da ferramenta**: não é reconhecido como quadro; se com isso faltar um número, a conferência recusa por lacuna, dizendo qual arquivo foi ignorado e por quê.
- **Falta de espaço em disco durante a codificação**: a execução termina com erro claro, sem arquivo parcial nem resíduo (História 6, cenário 3).
- **Quadros modificados enquanto o vídeo é montado**: não é uma situação suportada; a ferramenta lê os quadros conferidos, e o vídeo reflete o conteúdo lido, sem nunca alterar os arquivos de entrada.
- **Dois pedidos para o mesmo destino ao mesmo tempo**: não é uma situação suportada; mesmo assim a publicação atômica do arquivo garante que nenhum arquivo parcial apareça no destino.
- **Interrupção enquanto o arquivo final está sendo publicado**: o destino fica com o vídeo inteiro ou como estava antes; nunca com o vídeo pela metade.

## Requisitos *(obrigatório)*

### Requisitos Funcionais

- **FR-001**: O sistema DEVE montar um único arquivo de vídeo a partir de exatamente duas entradas informadas pelo usuário — o plano de câmera exportado pela terceira etapa e o diretório de quadros desenhado pela quinta — sem acessar rede, sem ler o recorte de dados, os dados registrados, nem o trajeto GPS, sem redesenhar quadro algum e sem alterar nenhuma das entradas.
- **FR-002**: O sistema DEVE validar o plano e recusar, com os mesmos erros próprios e regras já definidos nas etapas anteriores, um plano inexistente, ilegível, que não seja um plano exportado, truncado, corrompido, de versão de formato desconhecida (informando a versão encontrada e as aceitas) ou incoerente consigo mesmo.
- **FR-003**: O sistema DEVE reconhecer como quadros os arquivos do diretório informado que trazem a identificação de quadro da ferramenta (a que a quinta etapa grava dentro de cada imagem), e DEVE ignorar, sem alterar nem remover, os demais arquivos, que não contam como lacuna, duplicata nem quadro a mais.
- **FR-004**: Antes de qualquer codificação, o sistema DEVE conferir que os quadros reconhecidos pertencem ao plano informado e formam um só conjunto: cada quadro carrega a identificação do plano de que veio (FR-004a) e a identificação do conjunto a que pertence, todas as identificações de plano devem ser iguais à do plano informado (derivada do conteúdo do plano, a mesma que a etapa 4 guarda no recorte) e todas as identificações de conjunto devem ser iguais entre si. O sistema DEVE recusar, com erro próprio, distinto dos demais, um diretório com quadros de outro plano ou de mais de um conjunto misturados, com mensagem que diz quantos quadros pertencem ao plano informado e quantos não. Um quadro sem a identificação do plano (desenhado por uma versão anterior da etapa 5) DEVE ser recusado com erro próprio, com mensagem que diz que ele não traz a identificação do plano e que os quadros devem ser desenhados de novo.
- **FR-004a**: A etapa 5 DEVE gravar dentro de cada quadro, além da identificação do conjunto, a identificação do plano de que o quadro veio, derivada do conteúdo do plano (dois planos de conteúdo igual têm a mesma identificação, planos diferentes têm identificações diferentes), sem data/hora nem dado do ambiente, de modo que o desenho continue idêntico byte a byte para o mesmo plano, o mesmo recorte e a mesma resolução. Quadros sem esse campo NÃO DEVEM ser tratados pela etapa 5 como parte de um conjunto existente na retomada: são de um conjunto anterior, e são redesenhados com a sobrescrita explícita ou recusados como de outro conjunto sem ela. O contrato dos arquivos de quadro e a documentação da etapa 5 são atualizados junto com esta funcionalidade; esta é a única mudança de comportamento das etapas anteriores.
- **FR-005**: Antes de qualquer codificação, o sistema DEVE conferir que o diretório contém exatamente um quadro para cada número do plano, de 0 a N−1, sem lacunas, sem duplicatas e sem números que o plano não tem; e DEVE recusar, com erro próprio e mensagem que diz exatamente o que falta (números ausentes), o que sobra (números excedentes) e o que se repete (número e arquivos), o diretório que não atenda.
- **FR-006**: Antes de qualquer codificação, o sistema DEVE conferir que todos os quadros têm a mesma resolução e que a largura e a altura são pares, e DEVE recusar, com erro próprio, um diretório que não atenda, com mensagem que diz a resolução esperada e quais quadros destoam, com a resolução de cada um.
- **FR-007**: O sistema DEVE recusar, com erro próprio e mensagem que identifica o arquivo, um quadro que não seja uma imagem legível ou que esteja truncado, antes de iniciar a codificação.
- **FR-008**: O sistema DEVE recusar, com erro próprio e mensagem clara, um diretório de quadros inexistente, que não seja um diretório, que não possa ser lido ou que não contenha nenhum quadro reconhecido.
- **FR-009**: O sistema DEVE verificar, antes de qualquer codificação e antes de criar qualquer arquivo — e depois da conferência do plano, dos quadros e do destino, nessa ordem —, que o codificador de vídeo necessário está disponível na máquina e apto a produzir o formato de saída, e DEVE recusar, com erro próprio e mensagem clara e acionável, quando não estiver: a mensagem diz qual programa falta, o que instalar e como confirmar que ficou disponível; e, quando o codificador existe mas não oferece o formato necessário, diz o que ele não oferece.
- **FR-010**: O vídeo DEVE conter exatamente um quadro de vídeo para cada imagem do diretório, na ordem numérica do plano (do quadro 0 ao último), sem acrescentar, repetir, omitir nem reordenar quadros, e DEVE ter a mesma largura e a mesma altura das imagens, sem redimensionar, recortar nem bordas acrescentadas.
- **FR-011**: A taxa de quadros do vídeo DEVE ser a do plano, exatamente (inclusive quando não é um número inteiro), e a duração do vídeo DEVE ser a quantidade de quadros dividida pela taxa de quadros do plano.
- **FR-012**: O sistema DEVE produzir um arquivo de vídeo num formato de uso comum, que abra e reproduza nos reprodutores e nas plataformas de publicação de uso comum sem nova conversão, sem áudio, sem sobreposição de texto ou de estatísticas e sem qualquer alteração do conteúdo das imagens.
- **FR-013**: O sistema DEVE permitir escolher a qualidade do vídeo por níveis nomeados e documentados, com um nível padrão quando o usuário não escolher; um nível mais alto DEVE produzir um arquivo de tamanho maior ou igual ao de um nível mais baixo, sem alterar a ordem, a duração, a resolução nem a taxa de quadros; um nível inexistente DEVE ser recusado como erro de uso, com a lista dos níveis aceitos.
- **FR-014**: O mesmo plano, os mesmos quadros, o mesmo nível de qualidade e o mesmo codificador DEVEM produzir sempre exatamente o mesmo arquivo, byte a byte, independentemente do momento, do diretório de trabalho, do destino e de quantas execuções houve antes.
- **FR-015**: O arquivo de vídeo NÃO DEVE conter data ou hora de geração, caminhos, nome de máquina, nome de usuário, nem qualquer outro dado do ambiente em que foi gerado, nem o tempo gasto na montagem.
- **FR-016**: Durante a codificação, o sistema DEVE informar o progresso — quantos quadros já foram codificados, de quantos, e o tempo decorrido — atualizado enquanto a codificação avança; a informação de progresso NÃO DEVE alterar o conteúdo do vídeo.
- **FR-017**: Ao final de uma montagem bem-sucedida, o sistema DEVE imprimir um resumo com: a quantidade de quadros; a duração do vídeo; a resolução; a taxa de quadros; o nível de qualidade; o tamanho do arquivo gerado; o tempo gasto; e o codificador usado. O tempo gasto é informação de execução e NÃO DEVE aparecer no vídeo.
- **FR-018**: Por padrão, o sistema DEVE recusar, com erro próprio e mensagem que diz como sobrescrever, um destino que já exista, sem alterá-lo e antes de iniciar a codificação; com a opção explícita de sobrescrita, o arquivo é substituído por inteiro. Um destino cujo diretório não exista, que seja um diretório ou que não possa ser gravado DEVE ser recusado com erro próprio e mensagem clara, antes de iniciar a codificação.
- **FR-019**: O arquivo de vídeo DEVE ser publicado inteiro ou não ser publicado: uma falha ou interrupção em qualquer momento (falta de espaço, falha do codificador, cancelamento) NUNCA DEVE deixar arquivo parcial, ilegível ou temporário no destino nem no seu diretório; e, numa sobrescrita, o arquivo anterior permanece intacto se o novo não puder ser gerado.
- **FR-020**: Quando o usuário interrompe a execução, o sistema DEVE encerrar de forma ordenada, inclusive o processo do codificador, imprimir uma mensagem que deixa claro que a montagem foi interrompida e quantos quadros já tinham sido codificados, e sair com um código de saída próprio, distinto do de sucesso e dos demais erros. Não há retomada: repetir o pedido recomeça a montagem.
- **FR-021**: Todos os erros de negócio (plano ilegível, inválido, de versão desconhecida ou incoerente; diretório de quadros inválido; quadros que faltam, sobram ou se repetem; quadros de resoluções diferentes ou de dimensão ímpar; quadros que não pertencem ao plano; quadros sem a identificação do plano; quadro ilegível ou truncado; codificador ausente ou sem o formato necessário; destino já existente; destino inválido; montagem interrompida; falha do codificador) DEVEM ser distinguíveis entre si e dos erros das etapas anteriores, comunicados com mensagem clara e código de saída próprio; e, quando o erro for de recusa, nenhum arquivo é criado ou alterado.
- **FR-022**: O comportamento DEVE ser idêntico para qualquer voo em qualquer lugar do planeta, incluindo áreas que cruzam o meridiano de 180° e latitudes altas, sem tratamento especial de região e sem embutir dados de nenhuma região.
- **FR-023**: A ferramenta NÃO DEVE, nesta etapa, desenhar ou redesenhar quadros, incluir sobreposições de texto ou de estatísticas, incluir áudio ou trilha sonora, publicar o vídeo em qualquer serviço, editar um vídeo já gerado, nem oferecer interface gráfica ou API.

### Entidades-Chave *(incluir se a funcionalidade envolver dados)*

- **Plano de Câmera Exportado**: o arquivo da terceira etapa (versionado): parâmetros (entre eles, a taxa de quadros), resumo e todos os quadros, cada um com o número. Entrada desta etapa; fornece a quantidade de quadros, a ordem e a taxa de quadros do vídeo; validado como dado externo, pelas regras já existentes.
- **Diretório de Quadros**: a pasta desenhada pela quinta etapa, com uma imagem por quadro do plano, numerada pelo índice do plano, cada uma trazendo a identificação de quadro da ferramenta e a do plano de que veio. Entrada desta etapa, somente leitura; pode conter arquivos alheios, que são ignorados.
- **Identificação do Plano**: valor derivado do conteúdo de um plano exportado, o mesmo que a etapa 4 já guarda no recorte; a etapa 5 passa a gravá-lo em cada quadro e esta etapa o confere com o do plano informado.
- **Quadro**: uma imagem do diretório, identificada pelo número no plano, com a resolução em que foi desenhada.
- **Conferência dos Quadros**: o conjunto de verificações feitas antes de codificar — quadros do plano informado, todos presentes, sem lacunas, duplicatas nem excedentes, todos legíveis, todos com a mesma resolução e dimensões pares.
- **Nível de Qualidade**: um dos níveis nomeados e documentados que troca tamanho de arquivo e tempo de codificação por fidelidade; há um nível padrão.
- **Codificador de Vídeo**: o programa externo, instalado na máquina do usuário, que transforma a sequência de imagens em vídeo; sua presença e sua aptidão são verificadas antes de começar.
- **Vídeo**: o arquivo gerado — uma faixa de vídeo, sem áudio, com um quadro por imagem, a taxa de quadros do plano e a resolução das imagens.
- **Resumo da Montagem**: quantidade de quadros, duração, resolução, taxa de quadros, nível de qualidade, tamanho do arquivo, tempo gasto, codificador usado e, se interrompida, a indicação disso.

## Critérios de Sucesso *(obrigatório)*

### Resultados Mensuráveis

- **SC-001**: Em 100% dos voos testados, o vídeo gerado tem exatamente a quantidade de quadros do plano, a taxa de quadros do plano e a duração igual à quantidade de quadros dividida pela taxa de quadros (com diferença de no máximo um quadro de vídeo quando medida por uma ferramenta de inspeção de mídia), e os quadros aparecem na ordem numérica do plano.
- **SC-002**: Em 100% das verificações de correspondência exata, o quadro de vídeo de número k mostra a imagem de número k do diretório (a menos da perda inerente à compressão do nível de qualidade escolhido), incluindo o primeiro e o último quadro.
- **SC-003**: Gerar o mesmo vídeo 100 vezes seguidas, com o mesmo plano, os mesmos quadros, o mesmo nível de qualidade e o mesmo codificador, produz 100 arquivos idênticos byte a byte, inclusive quando gerados em momentos, diretórios de trabalho e destinos diferentes.
- **SC-004**: Em 100% dos vídeos gerados, a inspeção dos metadados do arquivo não revela data ou hora de geração, caminho, nome de máquina, nome de usuário nem outro dado do ambiente.
- **SC-005**: Em 100% dos casos de recusa testados (plano ilegível, inválido, de versão desconhecida ou incoerente; diretório inválido; quadro faltando, sobrando ou duplicado; resoluções diferentes ou dimensão ímpar; quadros de outro voo; quadros sem a identificação do plano; quadro truncado; codificador ausente; destino existente sem sobrescrita; destino inválido; nível de qualidade inexistente), a mensagem identifica exatamente o problema, o código de saída é próprio e nenhum arquivo é criado ou alterado; e a conferência dos quadros de um voo típico (1350 quadros) termina em menos de 10 segundos, sem codificar nada.
- **SC-006**: Em 100% das interrupções e falhas provocadas em pontos diferentes da montagem (cancelamento pelo usuário, falha do codificador, falta de espaço), nenhum arquivo parcial, ilegível ou temporário permanece no destino nem no seu diretório, e um arquivo anterior sobrescrito permanece intacto.
- **SC-007**: Um voo típico de 45 segundos a 30 quadros por segundo (1350 quadros) na resolução padrão da etapa 5 é montado no nível padrão em até 10 minutos num computador pessoal comum.
- **SC-008**: O vídeo gerado no nível padrão abre e reproduz sem erro em pelo menos três reprodutores de uso comum e é aceito por pelo menos duas plataformas de publicação de vídeo de uso comum sem nova conversão.
- **SC-009**: Para o mesmo plano e os mesmos quadros, o tamanho do arquivo cresce (ou permanece igual) a cada nível de qualidade mais alto, em todos os níveis documentados, sem que duração, resolução, taxa ou ordem dos quadros mudem.
- **SC-010**: Durante uma montagem longa, o usuário vê o progresso atualizado de forma contínua, com a quantidade de quadros codificados, o total e o tempo decorrido; e, ao final, consegue responder em menos de 1 minuto, apenas lendo o resumo, quantos quadros o vídeo tem, quanto dura, em que resolução e taxa, em que qualidade, quanto pesa, quanto tempo levou e com qual codificador.
- **SC-011**: Voos equivalentes deslocados para regiões diferentes do planeta (inclusive cruzando o meridiano de 180° e em latitudes acima de 80°) produzem vídeos com as mesmas propriedades (quantidade de quadros, duração, taxa, ordem e regras de recusa), sem tratamento especial de região.

## Suposições

- O usuário já cumpriu as etapas anteriores: exportou o plano (etapa 3) e desenhou os quadros desse plano num diretório (etapa 5, `render all`). Esta etapa não regera o plano nem os quadros, e não lê o recorte de dados (etapa 4) nem o trajeto GPS; usa só o plano e as imagens como estão. Os comandos das etapas anteriores continuam se comportando como antes, salvo a etapa 5 (`render frame` e `render all`), que passa a gravar a identificação do plano em cada quadro (FR-004a) — a única mudança de comportamento das etapas anteriores; quadros desenhados antes dela precisam ser desenhados de novo.
- **Formato de saída**: um vídeo em contêiner MP4 com compressão H.264 e formato de cor compatível com a maioria dos reprodutores e plataformas de publicação (o que exige largura e altura pares, já garantidas pela etapa 5). O formato e o contêiner exatos são fixados no planejamento técnico (`research.md` itens 2 e 6). Este é o "formato de uso comum" do FR-012. O arquivo de destino deve terminar em `.mp4`: outro nome é recusado como erro de uso, para o nome não mentir sobre o formato.
- **Codificador**: um codificador de vídeo externo, instalado pelo usuário na própria máquina e acessado como programa externo, atrás de uma porta (Princípios I e II da constituição). Qual programa e como ele é localizado é decisão do planejamento técnico; a mensagem de codificador ausente diz o que instalar, sem exigir configuração. A ferramenta não embute nem baixa o codificador (Princípio V: nada de rede).
- **Níveis de qualidade**: três níveis nomeados como nos demais comandos da ferramenta (`low`, `medium`, `high`), com `medium` como padrão, pensado para publicar; o que cada nível significa em parâmetros de codificação é decidido no planejamento técnico e documentado. O nome dos níveis segue o inglês por serem valores de linha de comando, como `--simplification=low`.
- **Determinismo**: vale para o mesmo codificador (o mesmo programa e a mesma versão) na mesma plataforma; a igualdade byte a byte entre versões ou builds diferentes do codificador só é prometida se o planejamento técnico conseguir garanti-la. Para isso, o planejamento deve garantir que a codificação não dependa de fatores variáveis do ambiente (por exemplo, número de processadores) e que o codificador não grave no arquivo seus próprios metadados de geração (data, versão, parâmetros com caminhos); o que o codificador gravar sobre si mesmo que varie com o ambiente conta como dado do ambiente (FR-015).
- **Sem retomada**: diferentemente da etapa 5, a montagem é uma única operação, curta perto do desenho dos quadros; interrompida, recomeça do zero (FR-020). Como o destino é um só arquivo e é publicado por inteiro, não há estado parcial a reaproveitar.
- **Sem áudio nem metadados de conteúdo**: o vídeo tem só a faixa de vídeo; sem título, autor, capítulos, legendas ou miniatura embutida.
- **Proporção e resolução**: a resolução do vídeo é a das imagens; esta etapa não a escolhe nem a altera, e não julga se ela combina com a proporção do plano (o aviso já existe na etapa 5).
- **Correspondência quadros-plano**: decidida pela identificação do plano gravada em cada quadro (Clarificação de 2026-09-27). Onde e como gravá-la dentro da imagem é decidido no planejamento técnico, preservando a igualdade byte a byte, a leitura por qualquer ferramenta de imagem e o reconhecimento dos quadros pela etapa 5 na retomada. A cobertura da área do recorte (etapa 5) e o dado do recorte não são reexaminados nesta etapa.
- **Reconhecimento de quadros**: segue a etapa 5 — um arquivo é um quadro da ferramenta quando é uma imagem que traz a identificação de quadro dentro dela, e não apenas quando o nome parece o de um quadro (`frame_NNNNNN.png`). A largura fixa da numeração do nome (6 dígitos) e a ordem alfabética coincidindo com a numérica são as do contrato de arquivos de quadro da etapa 5. Por isso dois arquivos nunca ocupam o mesmo número: a recusa de duplicata (FR-005) é regra do domínio, verificada sobre a listagem, e não é produzida pelo adapter de hoje. A conferência da resolução (FR-006) acontece antes da conferência de que os quadros são de um só conjunto, porque a identificação do conjunto inclui a resolução (uma sobrescrita interrompida em outra resolução deixa quadros das duas, e a mensagem útil é a de resolução).
- **Taxa de quadros**: a do plano, de 1 a 120 quadros por segundo, podendo ser decimal (etapa 3); esta etapa a usa exatamente como está, sem arredondar.
- **Desempenho**: as metas de tempo dos critérios de sucesso são referências iniciais para um computador pessoal comum e voos típicos, a serem confirmadas no planejamento técnico. A montagem deve ler os quadros um a um, sem exigir todos na memória ao mesmo tempo.
- **Ferramentas de verificação**: a verificação da duração, da taxa e da quantidade de quadros do vídeo (SC-001, SC-002) usa uma ferramenta de inspeção de mídia instalada na máquina de quem valida; ela não é requisito de execução da ferramenta.
- **Dados reais do usuário**: a validação real desta etapa usa os quadros desenhados pela etapa 5 a partir do trajeto real do usuário em `resources/` (com o mapa base sintético em imagem, dado que o mapa vetorial real é recusado pela etapa 5), e exige o codificador instalado na máquina; se ele não estiver instalado, a validação real exercita a recusa de codificador ausente.
