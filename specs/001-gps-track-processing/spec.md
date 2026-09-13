# Especificação de Funcionalidade: Leitura e Tratamento de Trajeto GPS

**Branch da Funcionalidade**: `001-gps-track-processing`

**Criado em**: 2026-09-13

**Status**: Rascunho

**Entrada**: Descrição do usuário: "Primeira etapa do Sobrevoo, uma ferramenta de linha de comando que gera vídeos de sobrevoo de trajetos, no estilo do Relive e do Strava. Esta etapa cobre apenas a leitura e o tratamento da rota. Nada de mapa, câmera ou vídeo ainda. O usuário aponta a ferramenta para um arquivo de rastreamento GPS de uma atividade (corrida, pedalada, caminhada) e recebe de volta um resumo do que foi lido e de como a rota foi tratada: quantidade de pontos original e após o tratamento, distância total, ganho de elevação, duração, área geográfica ocupada pelo trajeto e eventuais problemas encontrados. Requisitos de comportamento: aceitar mais de um formato de arquivo de rastreamento identificando o formato pelo próprio conteúdo; recusar arquivo inválido, vazio ou com pontos insuficientes com mensagem clara; descartar pontos com coordenadas impossíveis, duplicados consecutivos e saltos fisicamente implausíveis, informando quantos foram descartados; reduzir a quantidade de pontos preservando o formato do trajeto e suavizar o traçado; funcionar para trajetos em qualquer parte do mundo, inclusive cruzando o meridiano de mudança de data; lidar com arquivos sem altitude ou tempo, deixando claro o que estava ausente; permitir ajustar o nível de simplificação e de suavização. Fora de escopo: dados de mapa, relevo, renderização, vídeo, interface gráfica e API."

## Clarifications

### Session 2026-09-13

- Q: Quais formatos de arquivo de rastreamento GPS a ferramenta deve reconhecer e aceitar nesta primeira etapa? → A: GPX e TCX
- Q: O que a ferramenta deve fazer quando, após o descarte de pontos problemáticos, sobrarem menos pontos do que o mínimo necessário para gerar o resumo? → A: Recusar o arquivo, com a mesma mensagem de "pontos insuficientes", mencionando que isso ocorreu após a limpeza
- Q: Como o usuário deve informar os níveis de simplificação e de suavização ao executar a ferramenta? → A: Predefinições nomeadas (baixo, médio, alto) para simplificação e para suavização
- Q: O que a ferramenta deve fazer quando os pontos do arquivo não estão em ordem cronológica crescente (timestamps fora de ordem)? → A: Reordenar os pontos por timestamp antes do tratamento

## Cenários de Usuário e Testes *(obrigatório)*

### História de Usuário 1 - Resumo de um trajeto válido (Prioridade: P1)

Como usuário do Sobrevoo, aponto a ferramenta para um arquivo de rastreamento GPS
de uma atividade que fiz (corrida, pedalada ou caminhada) e recebo um resumo
claro do que foi lido: quantos pontos o arquivo tinha, quantos restaram após o
tratamento, a distância total percorrida, o ganho de elevação, a duração da
atividade e a área geográfica ocupada pelo trajeto — sem precisar informar o
formato do arquivo.

**Por que esta prioridade**: é o valor central desta etapa. Sem um resumo
correto e confiável de um trajeto bem formado, nenhuma etapa futura (câmera,
mapa, vídeo) tem uma base para se apoiar.

**Teste Independente**: pode ser totalmente testado fornecendo um arquivo de
rastreamento GPS válido e completo (com altitude e tempo) e verificando que o
resumo apresentado contém todos os campos esperados com valores coerentes com o
conteúdo do arquivo.

**Cenários de Aceitação**:

1. **Dado** um arquivo de rastreamento GPS válido em um dos formatos suportados,
   contendo altitude e tempo em todos os pontos, **Quando** o usuário aponta a
   ferramenta para esse arquivo, **Então** o sistema identifica o formato
   automaticamente e apresenta um resumo com: quantidade de pontos original,
   quantidade de pontos após o tratamento, distância total, ganho de elevação,
   duração e área geográfica ocupada pelo trajeto.
2. **Dado** um arquivo de rastreamento GPS válido sem informação de altitude em
   nenhum ponto, **Quando** o usuário processa esse arquivo, **Então** o resumo
   apresenta claramente que o ganho de elevação não pôde ser calculado por falta
   de dado de altitude, sem exibir um valor calculado a partir de dados
   inexistentes.
3. **Dado** um arquivo de rastreamento GPS válido sem informação de tempo em
   nenhum ponto, **Quando** o usuário processa esse arquivo, **Então** o resumo
   apresenta claramente que a duração não pôde ser calculada por falta de dado
   de tempo.
4. **Dado** um arquivo de rastreamento GPS de um trajeto que cruza o meridiano
   de mudança de data, **Quando** o usuário processa esse arquivo, **Então** a
   distância total e a área geográfica são calculadas corretamente, sem exigir
   que o usuário informe fuso, hemisfério ou região.

---

### História de Usuário 2 - Recusa clara de arquivo inválido (Prioridade: P2)

Como usuário do Sobrevoo, ao apontar a ferramenta para um arquivo que não é um
rastreamento GPS válido, está vazio, ou não tem pontos suficientes para gerar
qualquer estatística, quero uma mensagem clara explicando o motivo da recusa e o
que posso fazer a respeito, em vez de um resultado incorreto ou uma falha
confusa.

**Por que esta prioridade**: uma ferramenta que produz resumos incorretos ou
falha de forma pouco clara diante de entradas ruins é pior do que uma que se
recusa a processá-las com uma explicação útil. Isso protege a confiabilidade de
tudo o que a História 1 entrega.

**Teste Independente**: pode ser totalmente testado fornecendo um arquivo vazio,
um arquivo que não é um rastreamento GPS reconhecível, e um arquivo com pontos
insuficientes, e verificando que cada caso produz uma mensagem de recusa
específica e compreensível, sem gerar um resumo parcial ou incorreto.

**Cenários de Aceitação**:

1. **Dado** um arquivo vazio, **Quando** o usuário aponta a ferramenta para esse
   arquivo, **Então** o sistema recusa o processamento e informa que o arquivo
   está vazio, sem apresentar nenhum resumo.
2. **Dado** um arquivo que não corresponde a nenhum formato de rastreamento GPS
   reconhecido pela ferramenta, **Quando** o usuário aponta a ferramenta para
   esse arquivo, **Então** o sistema recusa o processamento e informa que o
   formato não foi reconhecido.
3. **Dado** um arquivo de rastreamento GPS válido mas com pontos insuficientes
   para calcular uma distância ou trajeto (por exemplo, um único ponto),
   **Quando** o usuário processa esse arquivo, **Então** o sistema recusa o
   processamento e informa que faltam pontos suficientes, sugerindo o que o
   usuário pode verificar.
4. **Dado** um arquivo de rastreamento GPS cujos pontos, após o descarte de
   coordenadas impossíveis, duplicados consecutivos e saltos implausíveis,
   ficam abaixo do mínimo necessário, **Quando** o usuário processa esse
   arquivo, **Então** o sistema recusa o processamento e informa que os pontos
   restantes após a limpeza são insuficientes, sem apresentar um resumo
   parcial.

---

### História de Usuário 3 - Tratamento de pontos problemáticos (Prioridade: P3)

Como usuário do Sobrevoo, quero que a ferramenta identifique e remova pontos com
coordenadas impossíveis, pontos duplicados consecutivos, e saltos fisicamente
implausíveis entre pontos, e me informe quantos pontos de cada tipo foram
descartados, para que eu confie que a distância e o traçado calculados refletem
minha atividade real e não ruído do receptor de GPS.

**Por que esta prioridade**: melhora a qualidade e a confiabilidade das
estatísticas entregues na História 1, mas o resumo básico já entrega valor
mesmo antes desse refinamento existir.

**Teste Independente**: pode ser totalmente testado fornecendo um arquivo de
rastreamento GPS que contenha deliberadamente coordenadas fora do intervalo
válido, pontos duplicados consecutivos, e um salto de posição implausível para
uma atividade humana, e verificando que o resumo relata a contagem de pontos
descartados por cada motivo, e que os pontos problemáticos não influenciam a
distância total calculada.

**Cenários de Aceitação**:

1. **Dado** um arquivo de rastreamento GPS com um ou mais pontos cuja latitude
   ou longitude está fora do intervalo geograficamente possível, **Quando** o
   usuário processa esse arquivo, **Então** esses pontos são descartados e o
   resumo informa quantos pontos foram descartados por coordenada impossível.
2. **Dado** um arquivo de rastreamento GPS com pontos consecutivos idênticos em
   coordenada, **Quando** o usuário processa esse arquivo, **Então** as
   repetições consecutivas são descartadas e o resumo informa quantos pontos
   duplicados foram descartados.
3. **Dado** um arquivo de rastreamento GPS com um salto de posição entre dois
   pontos consecutivos incompatível com qualquer atividade humana (corrida,
   pedalada ou caminhada) no intervalo de tempo entre eles, **Quando** o usuário
   processa esse arquivo, **Então** esse ponto é descartado e o resumo informa
   quantos pontos foram descartados por salto implausível.
4. **Dado** um arquivo de rastreamento GPS cujos pontos têm instante de tempo
   mas não estão em ordem cronológica crescente, **Quando** o usuário processa
   esse arquivo, **Então** os pontos são reordenados por instante de tempo
   antes da detecção de duplicados e de saltos implausíveis, e o resumo
   reflete o trajeto já reordenado.

---

### História de Usuário 4 - Redução e suavização do traçado (Prioridade: P4)

Como usuário do Sobrevoo, quero que a ferramenta reduza a quantidade de pontos
do meu trajeto preservando seu formato geral, e suavize a trepidação típica de
leituras de GPS, podendo eu ajustar o quanto isso é aplicado, para que o
trajeto tratado fique mais leve e mais limpo sem perder a forma real do
percurso.

**Por que esta prioridade**: prepara o trajeto para as etapas futuras (câmera e
vídeo), mas não é necessária para que a ferramenta já entregue um resumo
correto e confiável nesta etapa.

**Teste Independente**: pode ser totalmente testado processando o mesmo arquivo
de rastreamento GPS com diferentes níveis de simplificação e suavização, e
verificando que a quantidade de pontos final e a suavidade do traçado resultante
mudam de forma consistente com o nível escolhido, sem descaracterizar o formato
geral do trajeto original.

**Cenários de Aceitação**:

1. **Dado** um arquivo de rastreamento GPS válido com uma alta densidade de
   pontos, **Quando** o usuário processa esse arquivo sem informar um nível de
   simplificação ou suavização, **Então** o sistema aplica um tratamento padrão
   e o resumo mostra uma quantidade de pontos após o tratamento menor que a
   original, preservando o formato geral do trajeto.
2. **Dado** o mesmo arquivo de rastreamento GPS, **Quando** o usuário processa
   esse arquivo escolhendo o nível "alto" de simplificação, **Então** a
   quantidade de pontos após o tratamento é menor do que a obtida com o nível
   "médio" (padrão), e essa por sua vez é menor do que a obtida com o nível
   "baixo".
3. **Dado** o mesmo arquivo de rastreamento GPS, **Quando** o usuário escolhe o
   nível "alto" de suavização, **Então** o traçado resultante apresenta menos
   variações bruscas ponto a ponto do que o obtido com o nível "médio"
   (padrão), e essa por sua vez apresenta menos variações bruscas do que o
   obtido com o nível "baixo".

### Casos Extremos

- O que acontece quando o arquivo tem exatamente o número mínimo de pontos
  aceito pela ferramenta?
- Como o sistema lida com um arquivo em que todos os pontos têm a mesma
  coordenada (atividade parada o tempo todo)?
- Como o sistema lida com um arquivo cujos pontos não estão em ordem cronológica
  crescente? (ver FR-027: reordenado por instante de tempo antes do
  tratamento)
- O que acontece quando o nível de simplificação ou suavização informado está
  fora de um intervalo aceitável?
- Como o sistema lida com um arquivo em formato reconhecido, porém corrompido ou
  malformado a partir de um certo ponto do conteúdo?
- O que acontece quando, após o descarte de pontos problemáticos (coordenadas
  impossíveis, duplicados, saltos implausíveis), sobram menos pontos do que o
  mínimo necessário para gerar o resumo?
- Como o sistema lida com um trajeto que passa exatamente sobre um polo (onde a
  noção de longitude se torna degenerada)?

## Requisitos *(obrigatório)*

### Requisitos Funcionais

- **FR-001**: O sistema DEVE aceitar, como entrada, o caminho de um único
  arquivo de rastreamento GPS de uma atividade.
- **FR-002**: O sistema DEVE identificar o formato do arquivo de rastreamento
  pelo próprio conteúdo, sem exigir que o usuário informe o formato ou dependa
  da extensão do arquivo.
- **FR-003**: O sistema DEVE suportar os formatos GPX e TCX de arquivo de
  rastreamento GPS de atividades (corrida, pedalada, caminhada).
- **FR-004**: O sistema DEVE recusar o processamento de um arquivo cujo
  conteúdo não corresponda a nenhum formato de rastreamento suportado,
  informando que o formato não foi reconhecido.
- **FR-005**: O sistema DEVE recusar o processamento de um arquivo vazio,
  informando que o arquivo está vazio.
- **FR-006**: O sistema DEVE recusar o processamento de um arquivo com pontos
  insuficientes para calcular as estatísticas do resumo, informando o motivo da
  recusa. Essa recusa se aplica tanto quando o arquivo já chega com pontos
  insuficientes quanto quando a quantidade de pontos se torna insuficiente
  depois do descarte de pontos problemáticos (coordenadas impossíveis,
  duplicados consecutivos, saltos implausíveis) — nesse segundo caso, a
  mensagem DEVE indicar que a insuficiência ocorreu após a limpeza dos pontos.
- **FR-007**: Toda recusa de processamento DEVE vir acompanhada de uma mensagem
  que explique o motivo da recusa e oriente o que o usuário pode verificar ou
  corrigir no arquivo.
- **FR-008**: O sistema DEVE identificar pontos com coordenadas geográficas
  impossíveis e descartá-los antes de calcular as estatísticas do trajeto.
- **FR-009**: O sistema DEVE identificar pontos consecutivos duplicados (mesma
  posição geográfica registrada em sequência) e descartar as repetições.
- **FR-010**: O sistema DEVE identificar saltos de posição entre pontos
  consecutivos que sejam fisicamente implausíveis para uma atividade humana de
  corrida, pedalada ou caminhada, e descartar os pontos responsáveis pelo
  salto.
- **FR-011**: O sistema DEVE informar, no resumo final, a quantidade de pontos
  descartados, discriminada por motivo do descarte (coordenada impossível,
  duplicado consecutivo, salto implausível).
- **FR-012**: O sistema DEVE reduzir a quantidade de pontos do trajeto tratado
  preservando o formato geral do percurso original.
- **FR-013**: O sistema DEVE suavizar o traçado do trajeto tratado para reduzir
  a trepidação característica de leituras de GPS.
- **FR-014**: O sistema DEVE permitir que o usuário escolha o nível de
  simplificação aplicado à redução de pontos entre as predefinições baixo,
  médio e alto.
- **FR-015**: O sistema DEVE permitir que o usuário escolha o nível de
  suavização aplicado ao traçado entre as predefinições baixo, médio e alto.
- **FR-016**: O sistema DEVE aplicar o nível médio de simplificação e de
  suavização quando o usuário não informar um nível explícito.
- **FR-017**: O sistema DEVE calcular a distância total do trajeto após o
  tratamento dos pontos.
- **FR-018**: O sistema DEVE calcular o ganho de elevação total do trajeto
  quando o arquivo fornecer dados de altitude.
- **FR-019**: O sistema DEVE indicar explicitamente, no resumo, quando o
  arquivo não fornecer dados de altitude, sem apresentar um ganho de elevação
  calculado a partir de dados inexistentes.
- **FR-020**: O sistema DEVE calcular a duração da atividade quando o arquivo
  fornecer dados de tempo.
- **FR-021**: O sistema DEVE indicar explicitamente, no resumo, quando o
  arquivo não fornecer dados de tempo, sem apresentar uma duração calculada a
  partir de dados inexistentes.
- **FR-022**: O sistema DEVE calcular a área geográfica ocupada pelo trajeto
  (extensão de latitude e de longitude cobertas).
- **FR-023**: O sistema DEVE calcular corretamente a distância total e a área
  geográfica ocupada por trajetos que cruzam o meridiano de mudança de data,
  sem exigir do usuário qualquer informação de fuso horário, hemisfério ou
  região.
- **FR-024**: O sistema DEVE processar trajetos de qualquer região do planeta
  de forma idêntica, sem lógica especial baseada em localização geográfica.
- **FR-025**: O sistema DEVE apresentar, ao final do processamento de um
  arquivo válido, um resumo contendo: formato identificado, quantidade de
  pontos original, quantidade de pontos após o tratamento, distância total,
  ganho de elevação (ou indicação de ausência de dado), duração (ou indicação
  de ausência de dado), área geográfica ocupada, e a contagem de pontos
  descartados por motivo.
- **FR-026**: O sistema NÃO DEVE realizar, nesta etapa, nenhuma geração de
  mapa, câmera, vídeo, ou qualquer interface além da linha de comando.
- **FR-027**: Quando os pontos do arquivo tiverem instante de tempo e não
  estiverem em ordem cronológica crescente, o sistema DEVE reordená-los por
  instante de tempo antes de aplicar qualquer outro tratamento (descarte de
  pontos problemáticos, simplificação, suavização).

### Entidades-Chave

- **Trajeto**: representa o percurso completo de uma atividade (corrida,
  pedalada ou caminhada), composto por uma sequência ordenada de pontos, com o
  formato de origem identificado.
- **Ponto de Rastreamento**: um registro individual do trajeto, com posição
  geográfica (latitude e longitude), e opcionalmente altitude e instante de
  tempo.
- **Resumo de Processamento**: o conjunto de informações apresentado ao usuário
  ao final do tratamento de um trajeto — contagens de pontos antes/depois,
  distância total, ganho de elevação, duração, área geográfica ocupada, e
  contagem de pontos descartados por motivo.
- **Área Geográfica Ocupada**: a extensão mínima e máxima de latitude e de
  longitude cobertas pelo trajeto, calculada de forma consistente mesmo quando
  o trajeto cruza o meridiano de mudança de data.

## Critérios de Sucesso *(obrigatório)*

### Resultados Mensuráveis

- **SC-001**: O usuário obtém o resumo completo de um trajeto válido em uma
  única execução da ferramenta, sem etapas manuais adicionais e sem precisar
  informar o formato do arquivo.
- **SC-002**: 100% dos arquivos vazios, em formato não reconhecido, ou com
  pontos insuficientes são recusados com uma mensagem que permite ao usuário
  entender o motivo sem consultar documentação externa.
- **SC-003**: Para qualquer trajeto válido processado, o resumo apresentado
  contém todos os campos solicitados (pontos antes/depois, distância, ganho de
  elevação ou sua ausência, duração ou sua ausência, área geográfica, e
  contagem de descartes por motivo).
- **SC-004**: Trajetos que cruzam o meridiano de mudança de data produzem
  distância total e área geográfica corretas, sem valores incoerentes (por
  exemplo, distância negativa ou absurdamente alta), em 100% dos casos
  testados.
- **SC-005**: Um trajeto de até 20.000 pontos (equivalente a uma atividade de
  várias horas com registro a cada segundo) tem seu resumo apresentado em menos
  de 5 segundos.
- **SC-006**: Para um mesmo trajeto de entrada, aplicar os níveis baixo, médio
  e alto de simplificação produz quantidades de pontos finais visivelmente
  diferentes e decrescentes entre si (baixo > médio > alto).
- **SC-007**: Para um mesmo trajeto de entrada, aplicar os níveis baixo, médio
  e alto de suavização produz traçados com visivelmente menos variação brusca
  ponto a ponto à medida que o nível aumenta.

## Suposições

- Os formatos GPX e TCX (ver FR-003) são baseados em texto estruturado (XML),
  o que permite identificá-los pelo conteúdo sem ambiguidade. Formatos
  adicionais (por exemplo, FIT) podem ser suportados em etapas futuras sem
  alterar esta especificação.
- Um arquivo com menos de dois pontos válidos é considerado insuficiente, pois
  não é possível calcular distância, área ou traçado a partir de um único
  ponto.
- O limite que caracteriza um "salto fisicamente implausível" é definido
  internamente pela ferramenta com base em velocidades incompatíveis com
  atividades humanas de corrida, pedalada ou caminhada; não é um parâmetro
  ajustável pelo usuário nesta etapa (apenas os níveis de simplificação e de
  suavização são ajustáveis).
- Uma coordenada geográfica é considerada impossível quando sua latitude ou
  longitude está fora do intervalo geograficamente válido.
- O resumo é apresentado como texto legível diretamente na saída da linha de
  comando; esta etapa não inclui a geração de um arquivo de relatório
  persistido nem qualquer formato de saída estruturado adicional.
- Distância e elevação são expressas no sistema métrico (metros/quilômetros),
  já que a ferramenta deve se comportar de forma idêntica para qualquer região
  do planeta, sem preferência de unidade por localidade.
- Os níveis de simplificação e de suavização são escolhidos pelo usuário entre
  as predefinições baixo, médio e alto (ver FR-014, FR-015 e FR-016); o nível
  médio é aplicado quando o usuário não informa um nível explícito.
- Conforme informado pelo usuário, ficam fora do escopo desta etapa: dados de
  mapa, dados de relevo, renderização, geração de vídeo, interface gráfica e
  API — o resultado desta etapa é exclusivamente o trajeto lido, tratado, e o
  resumo textual descrito acima.
