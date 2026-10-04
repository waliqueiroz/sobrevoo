# Especificação de Funcionalidade: Níveis de tratamento em `plan` e `fly`

**Branch da Funcionalidade**: `013-treatment-level-flags`

**Criado em**: 2026-10-03

**Status**: Rascunho

**Entrada**: Descrição do usuário: "Os níveis de simplificação e de suavização do trajeto hoje só podem ser escolhidos no `inspect`; o `plan` e o `fly` usam sempre o nível padrão da configuração, sem nenhuma forma de o usuário pedir outro. Isso deixa o `inspect` prometendo um ajuste que o resto da ferramenta não cumpre: o usuário confere ali que um nível mais alto de simplificação deixa o traçado do jeito que ele quer e, na hora de gerar o vídeo, não tem como pedir aquilo — o único caminho é alterar o padrão no arquivo de configuração, o que muda os dois níveis ao mesmo tempo e vale para todos os comandos. A partir desta etapa, quem planeja uma câmera ou gera um voo escolhe esses dois níveis do mesmo jeito que já escolhe no `inspect`, com os mesmos nomes de opção, os mesmos valores aceitos, os mesmos padrões vindos da configuração e as mesmas mensagens de erro — de modo que conferir um trajeto e depois voar sobre ele com o mesmo tratamento seja a mesma escolha escrita duas vezes, e não duas coisas diferentes. Esses dois níveis mudam a geometria do trajeto tratado, e com ela o caminho da câmera, a duração calculada e a distância que aparece na sobreposição, então passam a fazer parte da identidade do plano de câmera: dois planos do mesmo trajeto com níveis diferentes são planos diferentes, nunca se confundem, e o reaproveitamento de plano, recorte e quadros de uma execução anterior só acontece quando os níveis também batem. Os valores escolhidos ficam registrados no plano exportado, junto dos outros parâmetros, para que quem lê o arquivo saiba com que tratamento ele foi feito. Fora de escopo: mudar os algoritmos de simplificação ou de suavização, os valores que cada nível representa ou a quantidade de níveis; acrescentar um nível novo ou um modo 'sem tratamento'; permitir níveis numéricos em vez de nomes; mudar o padrão da configuração ou como ele é lido; mexer no `inspect`, que já faz o que precisa; e mudar qualquer outro parâmetro de câmera, recorte, desenho ou vídeo."

## Clarifications

### Session 2026-10-03

- Q: Quando um diretório `--keep` foi criado antes desta funcionalidade existir — sem nenhum nível de tratamento registrado — o que `fly` deve fazer na primeira execução depois de atualizada? → A: tratar a ausência de registro como "nível padrão" implícito (reaproveita se a execução atual também pede o padrão; recalcula se pedir outro nível) — a ferramenta ainda não foi lançada, então não há `--keep` real de versão anterior a proteger; um plano sem o campo é, por definição, um plano gerado com o único nível que existia antes desta etapa, o padrão.

## Cenários de Usuário e Testes *(obrigatório)*

### História de Usuário 1 - Planejar a câmera com o nível conferido no `inspect` (Prioridade: P1)

Um usuário já rodou `inspect` no seu trajeto algumas vezes, comparando
`--simplification=high` com o nível padrão, e decidiu que o traçado mais
simplificado é o que quer para o vídeo. Ao rodar `plan` sobre o mesmo
trajeto, ele pede o mesmo nível com a mesma flag e obtém um plano de câmera
calculado sobre aquele traçado — não sobre o traçado do nível padrão que o
`plan` sempre usou até aqui.

**Por que esta prioridade**: é o problema relatado — sem isso, a promessa
de ajuste que o `inspect` demonstra nunca chega ao plano de câmera, que é o
que de fato alimenta o vídeo. Sem esta história, a funcionalidade inteira
não existe.

**Teste Independente**: pode ser testado rodando `plan <trajeto>
--simplification=high --smoothing=low` e `plan <trajeto>` (sem as flags) e
comparando os planos resultantes — a duração calculada, o caminho da câmera
e a distância total devem refletir o traçado de cada nível, do mesmo jeito
que `inspect <trajeto> --simplification=high --smoothing=low` já mostra
para aquele trajeto.

**Cenários de Aceitação**:

1. **Dado** um arquivo de trajeto válido, **Quando** o usuário roda `plan
   <trajeto> --simplification=high --smoothing=low`, **Então** o plano de
   câmera gerado é calculado sobre o trajeto tratado com simplificação
   `high` e suavização `low`, e o resumo impresso (duração, quadros,
   distâncias) corresponde a esse traçado.
2. **Dado** o mesmo arquivo de trajeto, **Quando** o usuário roda `plan
   <trajeto>` sem informar `--simplification` nem `--smoothing`, **Então**
   o resultado é idêntico ao que o `plan` já produzia antes desta
   funcionalidade existir (os níveis padrão da configuração).
3. **Dado** um valor de flag fora do conjunto aceito (ex.:
   `--simplification=extreme`), **Quando** o usuário roda `plan
   <trajeto> --simplification=extreme`, **Então** o comando recusa com a
   mesma mensagem de erro que `inspect` já usa para o mesmo engano, sem
   gerar nenhum plano.

---

### História de Usuário 2 - Voar sobre o trajeto com o tratamento escolhido (Prioridade: P2)

O mesmo usuário, satisfeito com o nível que conferiu, quer o vídeo
completo. Ele roda `fly <trajeto> --output voo.mp4` com as mesmas duas
flags que usou no `plan`, e o comando único aplica esse tratamento em toda
a tubulação — câmera, recorte, quadros e vídeo — sem precisar repetir a
escolha em mais de um lugar nem editar a configuração.

**Por que esta prioridade**: fecha o caminho descrito na história 1 até o
entregável final do projeto (o vídeo), que é o que o `fly` existe para
produzir. Depende da história 1 porque reaproveita o mesmo mecanismo de
escolha de nível.

**Teste Independente**: pode ser testado rodando `fly <trajeto> --output
voo.mp4 --simplification=high --smoothing=low` e comparando o vídeo (ou,
mais praticamente, o plano de câmera que o `fly` teria gerado, usando
`--keep` para inspecioná-lo) com o de um `plan` equivalente sobre o mesmo
trajeto e os mesmos níveis.

**Cenários de Aceitação**:

1. **Dado** um arquivo de trajeto válido e um destino de vídeo válido,
   **Quando** o usuário roda `fly <trajeto> --output voo.mp4
   --simplification=high --smoothing=low`, **Então** o vídeo produzido
   corresponde ao trajeto tratado com esses dois níveis, do início ao fim
   da tubulação (câmera, recorte, quadros e montagem).
2. **Dado** o mesmo trajeto, **Quando** o usuário roda `fly` sem informar
   as duas flags, **Então** o resultado é idêntico ao que o `fly` já
   produzia antes desta funcionalidade existir.
3. **Dado** um valor de flag fora do conjunto aceito, **Quando** o usuário
   roda `fly <trajeto> --output voo.mp4 --smoothing=extreme`, **Então** o
   comando recusa cedo, antes de tocar qualquer etapa da tubulação, com a
   mesma mensagem de erro que `inspect`/`plan` usam para o mesmo engano.

---

### História de Usuário 3 - Reaproveitar `--keep` com confiança quando o tratamento muda (Prioridade: P3)

O usuário roda `fly <trajeto> --output voo.mp4 --keep intermediarios/` uma
primeira vez com o nível padrão, decide testar um nível de simplificação
diferente, e roda `fly` de novo apontando para o mesmo diretório
`--keep`. Ele espera que o plano, o recorte e os quadros guardados da
execução anterior sejam refeitos do zero para o novo nível, nunca
reaproveitados como se já correspondessem a ele.

**Por que esta prioridade**: sem esta garantia, o `--keep` (um recurso que
já existe desde a etapa 7, com o objetivo de acelerar execuções repetidas)
passaria a entregar, silenciosamente, um vídeo com o tratamento da
execução anterior em vez do tratamento pedido agora — o tipo de engano que
o usuário só notaria olhando o traçado com atenção. É prioridade menor
porque só se manifesta para quem já usa `--keep` entre execuções com
níveis diferentes; quem não usa `--keep`, ou sempre usa o mesmo nível,
nunca encontra esse caso.

**Teste Independente**: pode ser testado rodando `fly <trajeto> --output
voo1.mp4 --keep intermediarios/` (nível padrão), depois `fly <trajeto>
--output voo2.mp4 --keep intermediarios/ --simplification=high` sobre o
mesmo diretório, e confirmando que o segundo plano, recorte e conjunto de
quadros guardados em `intermediarios/` refletem a simplificação `high`, e
não os da primeira execução.

**Cenários de Aceitação**:

1. **Dado** um diretório `--keep` com plano, recorte e quadros de uma
   execução anterior com os níveis padrão, **Quando** o usuário roda `fly`
   de novo sobre o mesmo trajeto e o mesmo diretório, mas com
   `--simplification` ou `--smoothing` diferentes dos da execução
   anterior, **Então** o plano, o recorte e os quadros são recalculados
   para o novo tratamento, não reaproveitados.
2. **Dado** o mesmo diretório `--keep` de uma execução anterior, **Quando**
   o usuário roda `fly` de novo com exatamente os mesmos níveis (informados
   ou omitidos, desde que resultem no mesmo nível efetivo), **Então** o
   plano, o recorte e os quadros continuam sendo reaproveitados, como já
   acontecia antes desta funcionalidade para os demais parâmetros.

---

### Casos Extremos

- Um diretório `--keep` guardado por uma execução de antes desta
  funcionalidade (sem nenhum registro de nível de tratamento) é reaproveitado
  por uma execução que roda sem informar `--simplification`/`--smoothing`
  (nível padrão), mas é tratado como desatualizado — e recalculado do zero —
  por qualquer execução que informe um nível diferente do padrão, do mesmo
  jeito que já é tratado como desatualizado hoje quando qualquer outro
  parâmetro do plano muda.
- Informar `--simplification` ou `--smoothing` explicitamente com o próprio
  valor padrão (ex.: `--simplification=medium`, se `medium` já é o padrão)
  produz exatamente o mesmo plano, e a mesma identidade de plano, que omitir
  a flag — a escolha é pelo nível efetivo, não pela presença da flag na
  linha de comando.
- Simplificação e suavização continuam sendo escolhidas de forma
  independente: pedir `--simplification=high` sem informar `--smoothing` (ou
  vice-versa) aplica o nível padrão só à flag omitida, exatamente como já
  acontece no `inspect`.
- Um plano exportado por uma versão da ferramenta anterior a esta
  funcionalidade, sem os níveis de tratamento registrados, continua sendo
  lido normalmente por comandos que só leem o plano (ele não precisa dizer
  com que nível foi gerado para ser válido) — a ausência desse registro só
  importa para a decisão de reaproveitamento do `--keep`, coberta no caso
  acima.

## Requisitos *(obrigatório)*

### Requisitos Funcionais

- **FR-001**: O comando `plan` DEVE aceitar uma opção para escolher o nível
  de simplificação e outra para o nível de suavização do trajeto, com o
  mesmo nome, os mesmos valores aceitos (baixo, médio, alto) e o mesmo
  padrão vindo da configuração que o comando `inspect` já usa para as
  mesmas duas escolhas.
- **FR-002**: O comando `fly` DEVE aceitar as mesmas duas opções, com o
  mesmo nome, os mesmos valores aceitos e o mesmo padrão que `plan` e
  `inspect`.
- **FR-003**: Quando o usuário informa um valor fora do conjunto aceito
  para qualquer uma das duas opções, em `plan` ou em `fly`, o sistema DEVE
  recusar o comando com a mesma mensagem de erro que `inspect` já usa para
  o mesmo engano, antes de executar qualquer etapa.
- **FR-004**: Quando o usuário não informa uma das duas opções (ou nenhuma
  delas) em `plan` ou em `fly`, o sistema DEVE aplicar o nível padrão da
  configuração a essa escolha, produzindo exatamente o mesmo resultado que
  o comando já produzia antes de esta funcionalidade existir.
- **FR-005**: O plano de câmera gerado por `plan` ou por `fly` DEVE ser
  calculado sobre o trajeto tratado com os níveis de simplificação e de
  suavização efetivamente escolhidos (informados ou padrão) — o caminho da
  câmera, a duração calculada quando automática, e qualquer distância ou
  elevação derivada do trajeto tratado refletem esse tratamento.
- **FR-006**: Os dois níveis efetivamente usados para gerar um plano de
  câmera DEVEM fazer parte do que identifica aquele plano: dois planos do
  mesmo trajeto e dos mesmos demais parâmetros, mas com um nível de
  simplificação ou de suavização diferente entre si, são sempre
  reconhecidos como planos diferentes, nunca equivalentes.
- **FR-007**: Quando `plan` exporta um plano em arquivo, o arquivo DEVE
  registrar os dois níveis efetivamente usados junto dos demais parâmetros
  do plano, de forma que quem lê o arquivo depois saiba com que tratamento
  aquele plano foi gerado.
- **FR-008**: Quando `fly` é executado com um diretório para guardar e
  reaproveitar resultados intermediários entre execuções, o reaproveitamento
  de um plano, recorte ou conjunto de quadros guardado por uma execução
  anterior só DEVE ocorrer quando os dois níveis de tratamento da execução
  atual também batem com os da execução que gerou o que está guardado; caso
  contrário, o sistema recalcula a partir do plano em diante. Um plano
  guardado sem nenhum nível de tratamento registrado (de antes desta
  funcionalidade existir) DEVE ser tratado como gerado com o nível padrão em
  ambas as escolhas, para efeito desta comparação.
- **FR-009**: As duas opções DEVEM continuar sendo escolhidas de forma
  independente uma da outra, em `plan` e em `fly`, exatamente como já são
  em `inspect` — escolher um nível para uma não obriga a escolher também
  para a outra.

### Entidades-Chave

- **Plano de câmera**: já existente; passa a ter, entre os parâmetros que o
  descrevem e que participam da sua identidade, os dois níveis de
  tratamento (simplificação e suavização) efetivamente usados para gerá-lo,
  além dos parâmetros que já tinha (duração, taxa de quadros, distância,
  inclinação, proporção).

## Critérios de Sucesso *(obrigatório)*

### Resultados Mensuráveis

- **SC-001**: Para qualquer trajeto e qualquer combinação dos dois níveis,
  o traçado usado para planejar a câmera com `plan --simplification=X
  --smoothing=Y` é o mesmo traçado que `inspect --simplification=X
  --smoothing=Y` já mostra para aquele trajeto, sem nenhuma diferença.
- **SC-002**: Rodar `plan` ou `fly` sem informar as duas opções, em
  qualquer trajeto, produz o mesmo resultado que o comando produzia antes
  desta funcionalidade existir, em 100% dos casos.
- **SC-003**: Rodar `fly --keep` duas vezes sobre o mesmo trajeto e o mesmo
  diretório, mudando apenas um dos dois níveis entre as execuções, sempre
  resulta num plano, recorte e conjunto de quadros recalculados para o novo
  nível — nunca num vídeo que mistura o tratamento de uma execução com o
  enquadramento ou os quadros de outra.
- **SC-004**: Todo valor inválido informado para as duas opções, em `plan`
  ou em `fly`, é rejeitado com uma mensagem que nomeia a opção e os valores
  aceitos, sem nenhuma etapa da tubulação (câmera, recorte, quadros, vídeo)
  chegar a rodar.

## Suposições

- Os nomes das opções, os valores aceitos (`low`, `medium`, `high`) e as
  mensagens de erro são exatamente os que `inspect` já usa hoje — esta
  funcionalidade estende onde essa escolha pode ser feita, não cria uma
  escolha nova.
- O padrão de cada uma das duas opções, quando omitida, continua vindo da
  mesma configuração única que hoje serve `inspect`, `plan` e `fly` — esta
  funcionalidade não introduz um padrão diferente por comando.
- `render frame`, `render all` e `video` não são afetados: eles operam
  sobre um plano e um recorte já gerados (que já carregam o tratamento
  aplicado), nunca tratam o trajeto GPS de novo.
