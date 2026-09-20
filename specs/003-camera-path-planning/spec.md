# Especificação de Funcionalidade: Planejamento do Movimento de Câmera

**Branch da Funcionalidade**: `003-camera-path-planning`

**Criado em**: 2026-09-19

**Status**: Rascunho

**Entrada**: Descrição do usuário: "Terceira etapa do Sobrevoo. Esta etapa cobre só o planejamento do movimento de câmera do vídeo de sobrevoo. Nada de desenhar mapa, renderizar quadros ou gerar vídeo ainda. A partir de um trajeto já tratado pela primeira etapa, a ferramenta calcula o caminho que uma câmera virtual percorre ao acompanhar o trajeto do início ao fim, e o usuário recebe um plano de câmera que pode inspecionar antes de existir qualquer renderização. O plano diz, para cada instante do vídeo, onde a câmera está (latitude, longitude e altitude), para onde aponta (direção e inclinação) e em que ponto do trajeto está o marcador da atividade. O usuário consegue: gerar o plano de câmera de um trajeto; escolher a duração do vídeo e a taxa de quadros; escolher o quanto a câmera fica distante e inclinada em relação ao trajeto; ver um resumo do plano (duração, quantidade de quadros, faixa de altitude e de distância da câmera, trechos em que a câmera precisou ser suavizada); e exportar o plano completo num arquivo legível, para inspeção e para as etapas seguintes. Requisitos: movimento sempre suave, sem giros bruscos nem saltos, mesmo em curvas fechadas, retornos e voltas no mesmo lugar; o tempo do vídeo segue o tempo real do trajeto de forma proporcional, comprimindo paradas longas; trajetos sem horário usam a distância percorrida como referência; o vídeo tem uma abertura que mostra o trajeto inteiro antes de começar a acompanhá-lo e um fechamento que mostra o trajeto completo ao final; o mesmo trajeto com os mesmos parâmetros produz sempre exatamente o mesmo plano; o comportamento é idêntico em qualquer lugar do planeta, inclusive trajetos que cruzam o meridiano de 180° e em latitudes altas; recusar parâmetros inválidos (duração ou taxa de quadros não positivas, duração curta demais para o trajeto) com mensagem clara. Fora de escopo: desenhar mapa ou relevo, renderização de quadros, geração de vídeo, sobreposições (textos, estatísticas, elevação), áudio, escolha manual de keyframes, interface gráfica e API."

## Clarifications

### Session 2026-09-19

- Q: Quando o arquivo de exportação já existe no destino escolhido, o que a ferramenta deve fazer por padrão? → A: Recusar com mensagem clara, a menos que o usuário passe uma opção explícita de sobrescrita.
- Q: Quando o trajeto é curto demais para ser acompanhado (poucos metros ou pontos praticamente no mesmo lugar), a ferramenta deve recusar ou gerar um plano mesmo assim? → A: Recusar com erro próprio e mensagem clara quando o comprimento do trajeto ficar abaixo de um mínimo documentado.
- Q: Existe um intervalo de taxa de quadros aceito pela ferramenta, ou qualquer valor positivo é aceito? → A: Aceitar de 1 a 120 quadros por segundo; fora disso, recusar informando o intervalo válido.
- Q: A duração padrão do vídeo deve ser fixa (60 s) ou depender do trajeto? → A: Depender do trajeto: quando o usuário não informa a duração, ela é calculada a partir do comprimento do trajeto (cresce de forma sublinear), nunca fica abaixo do mínimo daquele trajeto e tem um teto; o usuário continua podendo informar a duração que quiser.
- Revisão de consistência (`/speckit-analyze`), aprovada pelo usuário: (a) "extensão" passou a designar dois conceitos distintos — *comprimento* (distância percorrida, usado por FR-003a e FR-017a) e *abrangência* (maior afastamento entre pontos, usado pelo novo FR-017b, máximo 2 000 km); (b) o FR-014 passou a distinguir o determinismo bit a bit (mesma plataforma) da coincidência dentro da precisão documentada (entre plataformas); (c) erros de uso da linha de comando usam o código de uso já existente (FR-023).

## Cenários de Usuário e Testes *(obrigatório)*

### História de Usuário 1 - Gerar o plano de câmera de um trajeto (Prioridade: P1)

Como usuário do Sobrevoo, eu tenho um arquivo de trajeto GPS de uma atividade e quero que a ferramenta calcule como uma câmera virtual sobrevoaria esse trajeto do início ao fim. Como resultado, quero um plano que, para cada quadro do futuro vídeo, diga onde a câmera está (latitude, longitude e altitude), para onde ela aponta (direção e inclinação) e em que ponto do trajeto está o marcador da atividade — tudo isso antes de existir qualquer renderização.

**Por que esta prioridade**: é o núcleo da funcionalidade. Sem o plano não há nada para resumir, exportar ou ajustar, e as etapas seguintes (desenho do mapa, renderização, vídeo) dependem dele.

**Teste Independente**: pode ser totalmente testado apontando a ferramenta para um trajeto de exemplo, com os parâmetros padrão, e verificando que o plano produzido tem um item para cada quadro do vídeo e que cada item traz posição da câmera, direção, inclinação e posição do marcador.

**Cenários de Aceitação**:

1. **Dado** um arquivo de trajeto válido, **Quando** o usuário gera o plano de câmera sem informar nenhum parâmetro, **Então** a ferramenta produz um plano com duração calculada a partir do trajeto e taxa de quadros padrão, em que cada quadro tem latitude, longitude e altitude da câmera, direção, inclinação e posição do marcador ao longo do trajeto.
2. **Dado** um plano gerado, **Quando** o usuário examina o primeiro e o último quadros, **Então** o marcador está no início do trajeto no primeiro quadro que começa a acompanhá-lo e no fim do trajeto no último quadro, e nunca retrocede entre quadros consecutivos.
3. **Dado** um arquivo inválido, vazio ou com pontos insuficientes, **Quando** o usuário tenta gerar o plano, **Então** a ferramenta recusa com a mesma mensagem clara e o mesmo tratamento de erro da primeira etapa, sem produzir plano algum.
4. **Dado** o mesmo trajeto e os mesmos parâmetros, **Quando** o usuário gera o plano duas vezes (inclusive em execuções e momentos diferentes), **Então** os dois planos são exatamente iguais, quadro a quadro.

---

### História de Usuário 2 - Movimento suave e vídeo com abertura e fechamento (Prioridade: P2)

Como usuário do Sobrevoo, eu quero que o voo da câmera seja agradável de assistir: sem giros bruscos, sem saltos de posição, mesmo quando o trajeto tem curvas fechadas, retornos em sentido oposto ou voltas repetidas no mesmo lugar. Também quero que o vídeo comece mostrando o trajeto inteiro, antes de a câmera passar a acompanhar a atividade, e termine mostrando o trajeto completo, para dar contexto do que foi percorrido.

**Por que esta prioridade**: é o que diferencia um plano de câmera utilizável de uma sequência de posições correta porém enjoativa. Depende da existência do plano (P1), mas é o que torna o resultado final apresentável.

**Teste Independente**: pode ser testado com trajetos de exemplo que incluam curva de 180°, retorno pelo mesmo caminho e voltas em círculo, verificando que a variação de posição, de direção e de inclinação entre quadros consecutivos nunca ultrapassa os limites de suavidade da ferramenta, e que os primeiros e últimos trechos do plano enquadram o trajeto inteiro.

**Cenários de Aceitação**:

1. **Dado** um trajeto com uma curva fechada (ângulo maior que 150°), **Quando** o plano é gerado, **Então** a direção da câmera muda de forma gradual ao longo de vários quadros, sem nenhum giro brusco entre dois quadros consecutivos.
2. **Dado** um trajeto que vai e volta pelo mesmo caminho, **Quando** o plano é gerado, **Então** a câmera não dá um salto nem inverte de direção instantaneamente no ponto de retorno.
3. **Dado** um trajeto com várias voltas no mesmo lugar, **Quando** o plano é gerado, **Então** a câmera não oscila nem gira em alta frequência acompanhando cada volta, mantendo um movimento contínuo e legível.
4. **Dado** qualquer plano gerado, **Quando** o usuário examina os quadros de abertura, **Então** a câmera enquadra o trajeto inteiro no primeiro quadro e se aproxima gradualmente até o ponto de partida onde começa a acompanhar o marcador.
5. **Dado** qualquer plano gerado, **Quando** o usuário examina os quadros de fechamento, **Então** a câmera se afasta gradualmente do ponto final até enquadrar o trajeto inteiro no último quadro.
6. **Dado** um trajeto em que o movimento natural da câmera exigiria uma mudança mais brusca do que o limite permitido, **Quando** o plano é gerado, **Então** a ferramenta limita a mudança, o plano continua respeitando o limite de suavidade e o trecho afetado é registrado como "suavizado" para constar no resumo.

---

### História de Usuário 3 - Ajustar duração, taxa de quadros, distância e inclinação (Prioridade: P3)

Como usuário do Sobrevoo, eu quero escolher a duração do vídeo e a taxa de quadros, e também o quanto a câmera fica distante e inclinada em relação ao trajeto, para adequar o resultado ao que pretendo publicar (um vídeo curto para rede social, um mais longo e contemplativo etc.). Quero ainda que o tempo do vídeo acompanhe o tempo real da atividade de forma proporcional, para que trechos rápidos e lentos pareçam rápidos e lentos, sem que paradas longas travem o vídeo.

**Por que esta prioridade**: o plano padrão já entrega valor (P1 e P2), mas a possibilidade de ajuste é o que permite ao usuário adaptar o vídeo ao seu propósito.

**Teste Independente**: pode ser testado gerando planos do mesmo trajeto com valores diferentes de duração, taxa de quadros, distância e inclinação, e verificando que a quantidade de quadros, a faixa de distância da câmera e a inclinação mudam de forma coerente; e usando um trajeto com uma parada longa para verificar que a parada é comprimida.

**Cenários de Aceitação**:

1. **Dado** uma duração de 60 segundos e uma taxa de 30 quadros por segundo, **Quando** o plano é gerado, **Então** o plano tem exatamente 1800 quadros.
2. **Dado** o mesmo trajeto, **Quando** o usuário gera o plano com a distância da câmera "alta" e depois com "baixa", **Então** a câmera fica, em todo o trecho de acompanhamento, mais afastada do marcador com "alta" do que com "baixa".
3. **Dado** o mesmo trajeto, **Quando** o usuário gera o plano com a inclinação "alta" e depois com "baixa", **Então** a câmera olha mais de cima para baixo (mais próxima da vertical) com uma das opções e mais rente ao horizonte com a outra, de forma consistente em todo o trecho de acompanhamento.
4. **Dado** um trajeto com horários e uma parada longa (por exemplo, dez minutos parado num semáforo ou num café), **Quando** o plano é gerado, **Então** a parada ocupa no vídeo uma fração de tempo muito menor do que a proporcional ao tempo real, e o restante do trajeto avança em proporção ao tempo real.
5. **Dado** um trajeto sem informação de horário, **Quando** o plano é gerado, **Então** o avanço do marcador é proporcional à distância percorrida, e o resumo indica que a distância foi usada como referência de tempo.
6. **Dado** uma duração ou taxa de quadros igual a zero ou negativa, **Quando** o usuário tenta gerar o plano, **Então** a ferramenta recusa com uma mensagem que aponta qual parâmetro é inválido.
7. **Dado** uma duração tão curta que não comporta a abertura, o fechamento e um trecho de acompanhamento suave para aquele trajeto, **Quando** o usuário tenta gerar o plano, **Então** a ferramenta recusa com uma mensagem que informa a duração mínima necessária para o trajeto e taxa de quadros escolhidos.
8. **Dado** um valor de distância ou de inclinação que não é um dos níveis oferecidos, **Quando** o usuário tenta gerar o plano, **Então** a ferramenta recusa listando os valores aceitos.
9. **Dado** dois trajetos de comprimentos diferentes, **Quando** o usuário gera o plano de cada um sem informar a duração, **Então** o vídeo do trajeto mais longo tem duração maior ou igual à do mais curto, e o crescimento é proporcionalmente menor que o do comprimento (um trajeto dez vezes mais longo não gera um vídeo dez vezes mais longo).
10. **Dado** um trajeto tão grande que o mínimo necessário para um voo suave excede a duração que a regra automática daria, **Quando** o usuário gera o plano sem informar a duração, **Então** a duração usada é a mínima necessária, o plano é gerado sem erro e o resumo indica que a duração foi automática.
11. **Dado** qualquer trajeto válido, **Quando** o usuário gera o plano sem informar a duração, **Então** a duração automática nunca fica abaixo de 20 segundos nem acima de 120 segundos, exceto quando o mínimo necessário para aquele trajeto for maior que 120 segundos (caso em que vale o mínimo).
12. **Dado** um trajeto qualquer, **Quando** o usuário informa a duração explicitamente, **Então** a duração informada é usada exatamente (respeitando o mínimo e as demais validações), sem ajuste pela regra automática.

---

### História de Usuário 4 - Ver o resumo do plano (Prioridade: P4)

Como usuário do Sobrevoo, eu quero ver um resumo do plano gerado — duração, quantidade de quadros, faixa de altitude da câmera, faixa de distância da câmera ao marcador e os trechos em que a câmera precisou ser suavizada — para julgar rapidamente se o plano está adequado sem precisar ler quadro a quadro.

**Por que esta prioridade**: o plano já existe e é utilizável sem o resumo; ele acelera a inspeção e a decisão de ajustar parâmetros.

**Teste Independente**: pode ser testado gerando o plano de um trajeto conhecido e conferindo que cada valor do resumo corresponde ao que se calcula a partir dos quadros do plano.

**Cenários de Aceitação**:

1. **Dado** um plano gerado, **Quando** o usuário o visualiza, **Então** o resumo apresenta a duração do vídeo, a taxa de quadros, a quantidade total de quadros, as altitudes mínima e máxima da câmera e as distâncias mínima e máxima da câmera ao marcador, em unidades identificadas.
2. **Dado** um plano em que houve suavização forçada, **Quando** o usuário visualiza o resumo, **Então** cada trecho suavizado é listado com o instante de início e de fim no vídeo, e o resumo informa a quantidade total de trechos.
3. **Dado** um plano em que nenhuma suavização forçada foi necessária, **Quando** o usuário visualiza o resumo, **Então** o resumo diz explicitamente que nenhum trecho precisou ser suavizado.
4. **Dado** um trajeto sem horário, **Quando** o usuário visualiza o resumo, **Então** ele informa qual foi a referência usada para o tempo do vídeo (horário real ou distância percorrida).

---

### História de Usuário 5 - Exportar o plano completo (Prioridade: P5)

Como usuário do Sobrevoo, eu quero exportar o plano completo para um arquivo legível por pessoas e por programas, para inspecioná-lo com outras ferramentas e para que as etapas seguintes (desenho do mapa e renderização) o consumam sem precisar recalcular nada.

**Por que esta prioridade**: é a ponte para as próximas etapas e para a inspeção detalhada, mas o resumo (P4) já atende à inspeção rápida.

**Teste Independente**: pode ser testado exportando o plano de um trajeto, relendo o arquivo exportado e verificando que ele contém todos os quadros com todos os campos, além dos parâmetros usados, e que os valores coincidem com os do plano gerado.

**Cenários de Aceitação**:

1. **Dado** um plano gerado, **Quando** o usuário pede a exportação para um caminho de arquivo, **Então** o arquivo é criado contendo os parâmetros usados (duração, taxa de quadros, distância, inclinação), o resumo e todos os quadros, cada um com posição da câmera, direção, inclinação e posição do marcador.
2. **Dado** o mesmo trajeto e os mesmos parâmetros, **Quando** o usuário exporta o plano duas vezes, **Então** os dois arquivos são idênticos byte a byte.
3. **Dado** que o caminho de destino não pode ser gravado (diretório inexistente, sem permissão), **Quando** o usuário pede a exportação, **Então** a ferramenta recusa com uma mensagem clara sobre o destino, sem deixar um arquivo parcial.
4. **Dado** que o caminho de destino já contém um arquivo, **Quando** o usuário pede a exportação sem a opção explícita de sobrescrita, **Então** a ferramenta recusa com uma mensagem clara informando que o arquivo já existe e como sobrescrevê-lo, e o arquivo existente permanece intacto.
5. **Dado** que o caminho de destino já contém um arquivo, **Quando** o usuário pede a exportação com a opção explícita de sobrescrita, **Então** o arquivo é substituído integralmente pelo novo plano, sem deixar conteúdo parcial ou misturado.

---

### Casos Extremos

- **Trajeto que cruza o meridiano de 180°**: o plano é idêntico em qualidade e regras ao de qualquer outro trajeto — a câmera atravessa o meridiano sem salto, sem inverter o sentido do movimento e sem varrer o planeta ao contrário; longitudes do plano permanecem em uma faixa contínua e válida.
- **Trajeto em latitudes altas (próximo aos polos)**: a distância entre a câmera e o marcador e a direção da câmera continuam corretas e suaves, apesar de as longitudes convergirem; o resultado não é distorcido nem descontínuo.
- **Trajeto muito curto ou quase parado**: se o comprimento do trajeto (distância percorrida) ficar abaixo de um mínimo documentado, a ferramenta recusa com erro próprio e mensagem que informa o comprimento encontrado e o mínimo exigido, sem produzir plano; acima desse mínimo, o plano é gerado normalmente.
- **Trajeto muito longo** (por exemplo, centenas de quilômetros ou muitas horas): o plano continua suave e proporcional, e a distância da câmera se ajusta para não perder o marcador de vista. Acima da abrangência máxima documentada (2 000 km), a ferramenta recusa com erro próprio e mensagem que informa a abrangência encontrada e o máximo aceito.
- **Trajeto em que o marcador fica parado por tempo muito maior que o restante da atividade**: a compressão de paradas não faz o marcador saltar nem faz a câmera girar de forma errática enquanto ele está parado.
- **Trajeto sem altitude**: o plano é gerado normalmente; a altitude da câmera é definida a partir da distância e da inclinação escolhidas, não da elevação da atividade.
- **Trajeto com horários inconsistentes ou repetidos** (mesmo instante para pontos distintos): o plano é gerado sem falha, e o resumo indica se a referência de tempo precisou recorrer à distância percorrida.
- **Duração limítrofe**: a duração exatamente igual à mínima necessária é aceita e produz um plano válido e suave; um valor imediatamente abaixo é recusado.
- **Taxa de quadros fora do intervalo aceito** (menor que 1 ou maior que 120 quadros por segundo): a ferramenta recusa com mensagem que informa o intervalo válido; 1 e 120 são aceitos.
- **Parâmetro numérico ausente ou não numérico**: a ferramenta recusa com mensagem que indica o parâmetro e o valor esperado.

## Requisitos *(obrigatório)*

### Requisitos Funcionais

- **FR-001**: O sistema DEVE aceitar como entrada um trajeto GPS, submetê-lo ao mesmo tratamento da primeira etapa (descarte de pontos inválidos, reordenação por tempo, simplificação e suavização) e a partir dele gerar o plano de câmera, sem exigir que o usuário execute manualmente etapas intermediárias.
- **FR-002**: O plano DEVE conter exatamente um item por quadro do vídeo; cada item DEVE informar a posição da câmera (latitude, longitude e altitude), a direção para a qual ela aponta, a inclinação e a posição do marcador da atividade ao longo do trajeto.
- **FR-003**: O usuário DEVE poder escolher a duração do vídeo e a taxa de quadros; quando a taxa de quadros não for informada, DEVE ser usado um valor padrão documentado (30 quadros por segundo).
- **FR-003a**: Quando a duração não for informada, o sistema DEVE calculá-la a partir do comprimento do trajeto tratado, com crescimento sublinear em relação ao comprimento, entre um piso de 20 segundos e um teto de 120 segundos, e nunca inferior à duração mínima necessária para aquele trajeto e taxa de quadros (FR-017), que prevalece sobre o teto. O cálculo DEVE ser determinístico (mesmo trajeto e mesma taxa de quadros produzem a mesma duração) e a duração informada explicitamente pelo usuário DEVE sempre prevalecer sobre o cálculo automático. Uma duração calculada nunca DEVE ser recusada por ser curta demais.
- **FR-004**: A quantidade de quadros do plano DEVE ser exatamente a duração multiplicada pela taxa de quadros (arredondada de forma documentada quando o produto não for inteiro).
- **FR-005**: O usuário DEVE poder escolher o quanto a câmera fica distante e o quanto fica inclinada em relação ao trajeto, por meio de níveis nomeados (baixo, médio, alto), com um nível padrão documentado para cada um.
- **FR-006**: O movimento da câmera DEVE ser contínuo: entre dois quadros consecutivos, a posição, a direção e a inclinação DEVEM variar dentro de limites de suavidade definidos pela ferramenta e documentados; nenhum plano gerado pode violar esses limites.
- **FR-007**: A suavidade DEVE ser mantida em curvas fechadas, retornos em sentido oposto e voltas repetidas no mesmo lugar, sem giros bruscos, sem saltos de posição e sem oscilação de alta frequência.
- **FR-008**: Quando o movimento natural da câmera exigir uma mudança maior que os limites de suavidade, o sistema DEVE limitá-la e DEVE registrar cada trecho contínuo em que isso ocorreu, com instantes de início e fim no vídeo.
- **FR-009**: O tempo do vídeo DEVE seguir o tempo real do trajeto de forma proporcional: a razão entre tempo de vídeo e tempo real DEVE ser a mesma em todo o trajeto, exceto nas paradas longas.
- **FR-010**: Paradas longas (períodos prolongados sem deslocamento significativo) DEVEM ser comprimidas, ocupando no vídeo um tempo curto e limitado, e o tempo assim economizado DEVE ser redistribuído entre os demais trechos.
- **FR-011**: Quando o trajeto não tiver informação de horário utilizável, o sistema DEVE usar a distância percorrida como referência do avanço do marcador e DEVE informar essa escolha no resumo.
- **FR-012**: O vídeo DEVE começar com uma abertura em que o trajeto inteiro está enquadrado no primeiro quadro e a câmera se aproxima gradualmente do ponto de partida, e terminar com um fechamento em que a câmera se afasta gradualmente e enquadra o trajeto inteiro no último quadro.
- **FR-013**: A posição do marcador DEVE ser monotônica: nunca retroceder entre quadros consecutivos; DEVE estar no início do trajeto ao começar o acompanhamento e no fim do trajeto ao terminá-lo.
- **FR-014**: O mesmo trajeto com os mesmos parâmetros DEVE produzir sempre exatamente o mesmo plano, quadro a quadro e valor a valor, independentemente de momento ou ordem de execução. No mesmo programa e na mesma plataforma o resultado DEVE ser idêntico bit a bit; entre plataformas diferentes, os valores DEVEM coincidir dentro da precisão de arredondamento documentada (latitude e longitude em 1e-7 grau, alturas e distâncias em 1e-3 metro, ângulos em 1e-3 grau), que é a precisão em que o plano é definido e exportado.
- **FR-015**: O comportamento DEVE ser idêntico para qualquer trajeto em qualquer lugar do planeta, incluindo trajetos que cruzam o meridiano de 180° e trajetos em latitudes altas, sem qualquer tratamento especial ou privilegiado de região; a ferramenta NÃO DEVE embutir dados de nenhuma região.
- **FR-016**: O sistema DEVE recusar duração ou taxa de quadros que sejam zero, negativas ou não numéricas, com mensagem que aponte o parâmetro inválido; a duração informada DEVE ser de no máximo 3 600 segundos (uma hora), pois acima disso o plano deixa de ser prático de calcular e inspecionar, com mensagem que informe o máximo; a taxa de quadros DEVE estar entre 1 e 120 quadros por segundo (inclusive), e valores fora desse intervalo DEVEM ser recusados com mensagem que informe o intervalo válido.
- **FR-017**: O sistema DEVE recusar uma duração curta demais para acomodar abertura, fechamento e um acompanhamento suave do trajeto, com mensagem que informe a duração mínima necessária para aquele trajeto e aquela taxa de quadros.
- **FR-017a**: O sistema DEVE recusar um trajeto cujo comprimento (distância percorrida após o tratamento) seja inferior a um mínimo documentado (50 metros), com erro próprio e mensagem que informe o comprimento encontrado e o mínimo exigido.
- **FR-017b**: O sistema DEVE recusar um trajeto cuja abrangência (o maior afastamento entre pontos do trajeto tratado, medido como o dobro da maior distância do centro do trajeto a um de seus pontos) seja superior a um máximo documentado (2 000 quilômetros), com erro próprio e mensagem que informe a abrangência encontrada e o máximo aceito; o máximo existe porque a precisão do plano não é garantida acima dele.
- **FR-018**: O sistema DEVE recusar níveis de distância ou de inclinação desconhecidos, listando os valores aceitos.
- **FR-019**: O sistema DEVE apresentar um resumo do plano com: duração (indicando se foi calculada automaticamente ou informada pelo usuário), taxa de quadros, quantidade de quadros, altitude mínima e máxima da câmera, distância mínima e máxima da câmera ao marcador, referência de tempo usada (horário ou distância) e a lista de trechos suavizados (ou a indicação explícita de que não houve nenhum).
- **FR-020**: O sistema DEVE permitir exportar o plano completo — parâmetros usados, resumo e todos os quadros — para um arquivo de texto estruturado, legível por pessoas e por programas, e a exportação DEVE ser determinística (mesma entrada, arquivo idêntico).
- **FR-021**: Por padrão, a exportação DEVE recusar, com mensagem clara, um destino que já contenha um arquivo, deixando-o intacto; o usuário DEVE poder pedir explicitamente a sobrescrita por meio de uma opção dedicada, e, em qualquer caso, a exportação NÃO DEVE deixar arquivo parcial em caso de falha.
- **FR-022**: Arquivos de trajeto inválidos, vazios ou com pontos insuficientes DEVEM ser recusados com as mesmas mensagens e o mesmo tratamento de erro da primeira etapa.
- **FR-023**: Todos os erros de negócio (parâmetro numérico inválido, duração insuficiente, trajeto curto demais, trajeto grande demais, destino de exportação inválido ou já existente) DEVEM ser distinguíveis entre si e comunicados com mensagem clara e código de saída próprio, sem confundir com erros de leitura do trajeto. Erros de uso da linha de comando (valor não numérico, nível desconhecido, opção usada sem a que ela exige) usam o código de saída de uso já existente nas etapas anteriores, com mensagem que aponta o argumento.
- **FR-024**: A geração do plano NÃO DEVE depender de rede, de serviço externo nem dos dados geográficos registrados na segunda etapa; DEVE funcionar totalmente offline.
- **FR-025**: A ferramenta NÃO DEVE, nesta etapa, desenhar mapa ou relevo, renderizar quadros, gerar vídeo, incluir sobreposições ou áudio, aceitar keyframes manuais, nem oferecer interface gráfica ou API.

### Entidades-Chave *(incluir se a funcionalidade envolver dados)*

- **Plano de Câmera**: o resultado completo da etapa — os parâmetros usados, a sequência ordenada de quadros, a referência de tempo empregada e o resumo. É a entrada das etapas seguintes.
- **Quadro do Plano**: um instante do vídeo, com seu número e tempo no vídeo, a posição da câmera (latitude, longitude, altitude), a direção, a inclinação e a posição do marcador ao longo do trajeto (por distância percorrida e por coordenada).
- **Parâmetros do Plano**: duração (calculada a partir do trajeto quando não informada), taxa de quadros, nível de distância da câmera e nível de inclinação, com seus valores padrão.
- **Fase do Vídeo**: cada quadro pertence a uma de três fases — abertura, acompanhamento ou fechamento —, o que permite às etapas seguintes distinguir o comportamento esperado em cada trecho.
- **Trecho Suavizado**: um intervalo contínuo do vídeo (início e fim) em que a câmera precisou ter sua mudança limitada para respeitar os limites de suavidade.
- **Resumo do Plano**: duração, taxa e quantidade de quadros, faixa de altitude e de distância da câmera, referência de tempo usada e lista de trechos suavizados.
- **Trajeto Tratado**: o resultado da primeira etapa (pontos limpos, ordenados, simplificados e suavizados), que é a base do plano.

## Critérios de Sucesso *(obrigatório)*

### Resultados Mensuráveis

- **SC-001**: Um usuário consegue, com um único comando e sem informar nenhum parâmetro além do arquivo de trajeto, obter um plano de câmera completo e o resumo dele em menos de 5 segundos para um trajeto de até 10 000 pontos.
- **SC-002**: Em 100% dos planos gerados nos trajetos de teste (incluindo curva de 180°, retorno pelo mesmo caminho, voltas no mesmo lugar, cruzamento do meridiano de 180° e latitudes acima de 80°), nenhuma variação entre quadros consecutivos de posição, direção ou inclinação ultrapassa os limites de suavidade documentados.
- **SC-003**: Gerar o mesmo plano 100 vezes seguidas, com o mesmo trajeto e os mesmos parâmetros, produz 100 resultados idênticos e 100 arquivos exportados idênticos byte a byte.
- **SC-004**: A quantidade de quadros do plano é exatamente igual à duração multiplicada pela taxa de quadros em 100% dos casos com valores válidos.
- **SC-005**: Em trajetos com horários e sem paradas longas, a razão entre o tempo de vídeo e o tempo real varia no máximo 5% entre quaisquer dois trechos do acompanhamento; em trajetos com uma parada longa, a parada ocupa no máximo 5% da duração do vídeo.
- **SC-006**: Em 100% dos planos, o trajeto inteiro aparece enquadrado no primeiro e no último quadro, e o marcador nunca retrocede entre quadros consecutivos.
- **SC-007**: Trajetos equivalentes deslocados para regiões diferentes do planeta (inclusive cruzando o meridiano de 180°) produzem planos com as mesmas propriedades de suavidade, mesma quantidade de quadros e mesmos valores de resumo, dentro de uma tolerância de 1% nas distâncias.
- **SC-008**: 100% dos parâmetros inválidos testados (duração não positiva ou acima de 3 600 s, taxa não positiva ou fora de 1 a 120, duração abaixo do mínimo, trajeto curto demais, trajeto grande demais, níveis desconhecidos, destino de exportação inválido ou já existente sem opção de sobrescrita) são recusados com uma mensagem que identifica o parâmetro ou destino problemático, e a recusa por duração insuficiente sempre informa a duração mínima necessária.
- **SC-009**: Um usuário consegue, apenas lendo o resumo, responder em menos de 1 minuto: quanto dura o vídeo, quantos quadros tem, em que faixa de altitude e distância a câmera opera e se algum trecho precisou ser suavizado.
- **SC-010**: O plano exportado é lido de volta sem perda: todos os quadros e valores do arquivo coincidem com os do plano gerado.
- **SC-011**: Em 100% dos trajetos de teste (de dezenas de metros a centenas de quilômetros), gerar o plano sem informar a duração produz um plano válido, sem recusa por duração curta, com duração monotonicamente não decrescente em relação ao comprimento do trajeto e dentro de 20 a 120 segundos (ou igual ao mínimo necessário quando este for maior que 120 segundos).

## Suposições

- O usuário já dispõe de um arquivo de trajeto que a primeira etapa consegue ler; esta etapa reaproveita integralmente o tratamento da primeira etapa (mesmos formatos aceitos, mesmos erros, mesmos níveis de simplificação e suavização do trajeto) e NÃO depende do registro de dados geográficos da segunda etapa.
- Os valores padrão são: taxa de quadros de 30 por segundo, distância da câmera "média" e inclinação "média". A duração padrão não é fixa: é calculada a partir do comprimento do trajeto (FR-003a), como fazem outros serviços de vídeo de atividades, com curva de referência de aproximadamente 30 s para 5 km, 45 s para 20 km, 60 s para 50 km e 100 s para 200 km. A curva exata, o piso e o teto podem ser revistos no planejamento técnico sem alterar esta especificação, desde que continuem monotônicos, sublineares e dentro dos limites de FR-003a. O comprimento (distância percorrida) é usado em vez do tempo real porque existe em qualquer trajeto, inclusive sem horário.
- Distância e inclinação da câmera são escolhidas por níveis nomeados (baixo, médio, alto), em linha com os níveis já usados nos parâmetros de simplificação e suavização da primeira etapa, e não por valores numéricos livres.
- A abertura e o fechamento ocupam, cada um, uma fração fixa e documentada da duração total (ordem de 10% cada), e o restante da duração é dedicado ao acompanhamento; a duração mínima aceita é aquela que ainda permite um acompanhamento suave, e é calculada pela ferramenta para cada trajeto e taxa de quadros.
- Uma "parada longa" é um período contínuo, acima de um limiar documentado (ordem de dezenas de segundos), em que o deslocamento é desprezível; o tempo de vídeo dedicado a cada parada é curto e limitado.
- A altitude da câmera é derivada da distância e da inclinação escolhidas em relação ao marcador, e não depende de dados de relevo nem da elevação registrada no trajeto; considerar o relevo real fica para etapas posteriores.
- O plano é gerado a partir do trajeto já suavizado da primeira etapa, e o marcador acompanha esse trajeto tratado, não os pontos brutos.
- A exportação usa um formato de texto estruturado, de uso comum e estável, escolhido no planejamento técnico; o formato é versionado para permitir evolução sem quebrar as etapas seguintes.
- Como nas etapas anteriores, a ferramenta é operada por linha de comando, e os códigos de saída para os novos erros de negócio seguem a mesma convenção de mapeamento de erros da CLI existente.
- Trajetos extremamente longos (acima de milhares de quilômetros) e planos com milhões de quadros estão fora do uso esperado; a recusa por abrangência (FR-017b) e os limites de duração informada (3 600 s) e de taxa de quadros (FR-016) tornam esses limites explícitos.
- Todo o processamento é local e offline, conforme a constituição do projeto; o resultado é determinístico e não depende do relógio do sistema.
- Com os valores de referência da duração automática, a duração mínima necessária para um voo suave fica abaixo do teto de 120 s para qualquer trajeto aceito (em torno de 55 s para 100 km e de 90 s no limite de abrangência). A regra "o mínimo prevalece sobre o teto" (FR-003a) é uma salvaguarda para o caso de os valores de ajuste serem alterados no futuro, e é verificada com valores de ajuste sintéticos.
- O determinismo bit a bit (FR-014) vale no mesmo programa e na mesma plataforma; entre plataformas, o plano é definido pela precisão de arredondamento documentada, dentro da qual os valores coincidem.
