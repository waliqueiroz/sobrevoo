# Especificação de Funcionalidade: Iluminação Direcional do Terreno

**Branch da Funcionalidade**: `016-terrain-lighting`

**Criado em**: 2026-10-05

**Status**: Implementada, validada com o binário real e dados reais, inclusive geração de vídeo com o `ffmpeg` real

**Entrada**: Descrição do usuário: "O terreno é desenhado hoje apenas vestindo a malha com as cores do mapa base, sem nenhuma noção de luz: uma encosta virada para o sol e outra virada para a sombra recebem exatamente a mesma cor, e o relevo só se percebe pela silhueta contra o horizonte. O resultado é um chão que parece um mapa esticado, não uma paisagem — e a saída que se costuma usar para contornar isso, assar um sombreado dentro do próprio arquivo de mapa base, não serve aqui: a sombra assada é calculada para uma vista de cima com o sol num ângulo fixo, e quando a imagem é esticada sobre a malha vista de outro ângulo ela deixa de corresponder à orientação real da superfície na tela, aparecendo como sujeira e, ao longe, como listras. A partir desta etapa o próprio desenho ilumina o terreno: em cada ponto onde um raio toca o chão, a cor que vem do mapa base é clareada ou escurecida conforme a inclinação da superfície naquele ponto em relação a uma luz direcional fixa, de direção e altura documentadas, de modo que uma encosta voltada para a luz fique mais clara e uma voltada para o lado oposto, mais escura. Como a orientação é lida da mesma superfície que o raio acabou de atingir, a iluminação concorda com a geometria que está na tela em qualquer ângulo de câmera, em qualquer distância, e funciona com qualquer mapa base — satélite, mapa de ruas, o que o usuário tiver registrado — sem que ele precise preparar arquivo nenhum. A iluminação modula a cor do mapa, nunca a substitui: as ruas, os rótulos e as cores do mapa continuam legíveis, e mesmo a encosta mais escura continua mostrando o que há desenhada nela. A faixa entre o mais claro e o mais escuro é fixa e documentada, escolhida para dar volume sem sujar o mapa. A luz atinge somente o terreno: o traçado, o marcador, as sobreposições de tela e os dois padrões de 'sem dado' — as listras de terreno sem mapa e o xadrez de terreno sem elevação — não são iluminados, porque não são superfície do mundo e perderiam o sentido se mudassem de tom conforme a encosta. Um cuidado decide se isto fica bom ou vira ruído: ao longe, um único pixel da tela cobre centenas de metros de chão, e ler a inclinação num ponto minúsculo ali produziria um valor que salta de um quadro para o outro, cintilando. A inclinação precisa ser lida numa vizinhança proporcional ao pedaço de terreno que aquele pixel realmente cobre — grande ao longe, pequena de perto —, de forma que a iluminação seja estável ao longo do voo e suave entre pixels vizinhos, em vez de granulada. Da mesma forma, um ponto sem elevação conhecida não inventa inclinação. Os pixels mudam sem o usuário ter pedido, então a versão de desenho sobe e quadros antigos nunca se juntam aos novos no mesmo conjunto. Todo o cálculo obedece à mesma disciplina aritmética do resto do desenho, usando apenas operações cuja implementação não varia por arquitetura, para que a garantia de mesma imagem byte a byte em qualquer máquina continue valendo. Fora de escopo: deixar o usuário escolher direção, altura, cor ou intensidade da luz; mais de uma fonte de luz, luz de preenchimento ou céu colorido; sombras projetadas de um relevo sobre outro; oclusão de ambiente; hora do dia ou estação; neblina, perspectiva atmosférica ou qualquer efeito de distância; mudar o mapa base, o recorte, o plano de câmera, a malha do terreno ou como ela é construída; e mudar o traçado, o marcador ou as sobreposições."

## Clarifications

### Session 2026-10-05

- Q: Quando a vizinhança de amostragem de inclinação de um ponto com
  elevação conhecida está incompleta (alguns vizinhos sem elevação
  conhecida), o que a iluminação deve fazer com ele? → A: A inclinação é
  calculada só com as amostras da vizinhança que têm elevação conhecida;
  se nenhuma amostra tiver elevação conhecida, o ponto recebe o tom
  neutro de superfície plana. Um ponto que tem elevação conhecida nunca
  passa a ser tratado como "sem elevação" (xadrez) por causa da
  vizinhança — esse padrão continua reservado exclusivamente aos pontos
  cuja própria elevação é desconhecida, exatamente como hoje.

## Cenários de Usuário e Testes *(obrigatório)*

### História de Usuário 1 - Relevo com volume real, sem nenhuma flag nova (Prioridade: P1)

Como usuário que já gera vídeos de sobrevoo com os comandos existentes
(`render frame`, `render all`, `fly`), quero que o terreno do vídeo mostre
volume de verdade — encostas claras onde a luz bate, escuras onde não bate
— em vez de um mapa esticado sobre uma malha, sem precisar aprender nem
informar nenhuma opção nova.

**Por que esta prioridade**: é o valor central da funcionalidade — sem
volume perceptível no relevo, nada mais importa. É também o que torna a
funcionalidade testável de ponta a ponta de forma independente.

**Teste Independente**: pode ser totalmente testado gerando um quadro ou
vídeo sobre um trajeto que sobrevoa um morro ou vale conhecido, sem passar
nenhuma flag nova, e comparando os tons de duas encostas opostas do mesmo
acidente de relevo.

**Cenários de Aceitação**:

1. **Dado** um trajeto que sobrevoa um relevo com uma encosta voltada para
   a direção da luz e outra voltada para o lado oposto, **Quando** um
   quadro é desenhado, **Então** as duas encostas aparecem com tons
   visivelmente diferentes entre si, mesmo vestidas com a mesma cor de
   mapa base.
2. **Dado** um comando `render frame`, `render all` ou `fly` já em uso
   antes desta funcionalidade, **Quando** ele é executado sem nenhuma flag
   nova, **Então** o comando continua funcionando normalmente e o terreno
   do quadro resultante passa a mostrar a iluminação descrita, sem exigir
   nenhum parâmetro adicional.
3. **Dado** um trajeto sobrevoado em dois ângulos de câmera diferentes
   sobre o mesmo relevo, **Quando** os quadros correspondentes são
   desenhados, **Então** a iluminação de cada ponto do terreno concorda
   com a orientação da superfície exibida naquele ângulo específico, não
   com uma vista fixa de cima.

---

### História de Usuário 2 - Mapa continua legível na encosta mais escura (Prioridade: P2)

Como usuário, quero que a iluminação nunca torne o mapa base ilegível —
ruas, rótulos e cores do mapa precisam continuar aparecendo mesmo na
encosta mais escura do quadro —, porque o vídeo ainda precisa servir para
reconhecer o lugar sobrevoado, não só para mostrar relevo.

**Por que esta prioridade**: sem este cuidado, a funcionalidade central da
História 1 teria o efeito colateral de sujar ou escurecer demais o mapa,
tornando o resultado pior do que o visual atual sem iluminação.

**Teste Independente**: pode ser totalmente testado desenhando um quadro
com uma encosta acentuada sobre um mapa de ruas ou rótulos e verificando
que esses elementos continuam visíveis na parte mais escura do quadro.

**Cenários de Aceitação**:

1. **Dado** um quadro com uma encosta voltada totalmente para o lado
   oposto à luz, **Quando** o quadro é desenhado sobre um mapa base com
   ruas e rótulos, **Então** essas ruas e rótulos continuam distinguíveis
   na área mais escura do quadro.
2. **Dado** qualquer ponto do terreno iluminado, **Quando** a cor final do
   pixel é calculada, **Então** ela é sempre uma versão clareada ou
   escurecida da cor que o mapa base forneceria para aquele ponto, nunca
   uma cor diferente da do mapa.

---

### História de Usuário 3 - Iluminação estável ao longo do voo (Prioridade: P3)

Como usuário que grava voos em que a câmera se aproxima e se afasta do
terreno, quero que a iluminação de uma mesma área não trema nem pareça
granulada de um quadro para o outro, para o vídeo final parecer suave em
vez de ruidoso.

**Por que esta prioridade**: é um refinamento de qualidade sobre o
resultado da História 1 — o volume já existe sem este cuidado, mas pode
ficar visualmente ruim (cintilante) em voos com variação de distância, o
que esta história evita.

**Teste Independente**: pode ser totalmente testado gerando uma sequência
de quadros consecutivos sobre o mesmo trecho de terreno, com a câmera se
afastando e se aproximando ao longo da sequência, e verificando que a
iluminação daquele trecho muda suavemente, sem saltos nem textura
granulada.

**Cenários de Aceitação**:

1. **Dado** uma sequência de quadros consecutivos de um voo sobre o mesmo
   trecho de terreno, **Quando** a distância da câmera até esse trecho
   varia pouco entre dois quadros vizinhos, **Então** a iluminação
   daquele trecho também varia pouco entre esses dois quadros.
2. **Dado** um trecho de terreno visto de muito longe, onde um único pixel
   da tela cobre uma grande área de chão, **Quando** o quadro é desenhado,
   **Então** a iluminação daquele trecho aparece suave, sem um padrão
   granulado pixel a pixel.

---

### Casos Extremos

- O que acontece num ponto do terreno que não tem elevação conhecida? Ele
  cai no padrão "sem elevação" já existente (xadrez), sem nenhuma
  iluminação — exatamente como hoje, sem nenhuma área nova virando
  xadrez por causa desta funcionalidade.
- O que acontece num ponto do terreno que tem elevação conhecida, mas
  cuja vizinhança de amostragem da inclinação está incompleta (parte dos
  vizinhos sem elevação conhecida)? A inclinação é calculada só com as
  amostras da vizinhança que têm elevação conhecida; se nenhuma amostra
  da vizinhança tiver elevação conhecida, o ponto recebe o tom neutro de
  superfície plana — ele nunca passa a ser tratado como "sem elevação"
  só por causa da vizinhança.
- O que acontece numa superfície praticamente plana (sem inclinação
  perceptível)? Ela recebe um tom neutro, dentro da faixa fixa, nem o mais
  claro nem o mais escuro.
- O que acontece numa encosta extremamente inclinada, voltada totalmente
  para o lado oposto à luz? Ela fica no tom mais escuro que a faixa fixa
  permite, nunca preto puro, e a cor original do mapa continua
  reconhecível ali.
- O que acontece com um terreno sem nenhum mapa base registrado (padrão de
  listras de "sem mapa")? Essa área não é iluminada; as listras
  permanecem exatamente como são hoje.
- O que acontece quando um conjunto de quadros desenhado antes desta
  funcionalidade é encontrado por um `render all` retomado ou por um `fly
  --keep` de uma execução anterior? Ele é reconhecido como pertencente a
  outro conjunto e redesenhado do zero, nunca misturado com quadros novos.

## Requisitos *(obrigatório)*

### Requisitos Funcionais

- **FR-001**: O sistema DEVE, ao desenhar cada quadro, iluminar cada ponto
  do terreno conforme a inclinação da superfície exatamente naquele ponto,
  em relação a uma luz direcional fixa, de direção e altura documentadas.
- **FR-002**: O sistema DEVE clarear a cor do mapa base num ponto do
  terreno voltado para a luz, e escurecê-la num ponto voltado para o lado
  oposto, sem nunca substituir a cor original por outra.
- **FR-003**: A faixa entre o tom mais claro e o mais escuro que a
  iluminação pode produzir DEVE ser fixa e documentada, escolhida para dar
  volume ao relevo sem comprometer a legibilidade do mapa base em nenhuma
  parte do quadro.
- **FR-004**: A iluminação DEVE funcionar da mesma forma com qualquer mapa
  base que o usuário já tenha registrado (satélite, mapa de ruas, ou
  outro), sem exigir nenhum preparo, conversão ou arquivo adicional por
  parte do usuário.
- **FR-005**: A iluminação de um ponto do terreno DEVE concordar com a
  orientação real da superfície exibida na tela, em qualquer ângulo e
  qualquer distância de câmera, porque é calculada a partir da mesma
  superfície que o desenho acabou de atingir naquele ponto, nunca a partir
  de um sombreamento pré-calculado para uma vista fixa.
- **FR-006**: O traçado da atividade, o marcador, as sobreposições de
  tela, o padrão de "terreno sem mapa" e o padrão de "terreno sem
  elevação" NÃO DEVEM ser afetados pela iluminação, mantendo exatamente a
  cor que teriam sem esta funcionalidade.
- **FR-007**: A inclinação usada para iluminar um ponto do terreno DEVE
  ser lida numa vizinhança proporcional à área de terreno que o pixel da
  tela correspondente efetivamente cobre no mundo real — maior quando o
  ponto está longe da câmera, menor quando está perto —, de modo que a
  iluminação permaneça estável ao longo do voo e suave entre pixels
  vizinhos, em vez de cintilante ou granulada.
- **FR-008**: O sistema DEVE calcular a inclinação de um ponto do terreno
  usando apenas as amostras de elevação conhecida dentro da sua
  vizinhança de amostragem: se o próprio ponto não tiver elevação
  conhecida, ele cai no padrão "sem elevação" já existente (FR-006), sem
  nenhuma iluminação; se o ponto tiver elevação conhecida mas parte da
  vizinhança não tiver, a inclinação DEVE ser calculada só com as
  amostras disponíveis; se nenhuma amostra da vizinhança tiver elevação
  conhecida, o ponto DEVE receber o tom neutro de superfície plana. Um
  ponto com elevação conhecida NUNCA DEVE passar a ser tratado como "sem
  elevação" por causa de uma vizinhança incompleta.
- **FR-009**: O cálculo da iluminação DEVE produzir exatamente a mesma
  imagem, pixel a pixel, independentemente da arquitetura de processador
  ou do número de processos usados para desenhar — a mesma garantia de
  reprodutibilidade byte a byte que o restante do desenho de quadros já
  oferece.
- **FR-010**: Como o resultado visual de um quadro com o terreno iluminado
  é diferente do resultado sem esta funcionalidade, o sistema DEVE tratar
  quadros desenhados antes desta funcionalidade como pertencentes a um
  conjunto diferente dos quadros desenhados depois dela, nunca misturando
  os dois num mesmo conjunto reaproveitado ou retomado.
- **FR-011**: A funcionalidade NÃO DEVE expor, em nenhum comando, nenhuma
  opção para o usuário escolher ou desligar a direção, a altura, a cor ou
  a intensidade da luz.
- **FR-012**: A funcionalidade NÃO DEVE introduzir mais de uma fonte de
  luz, luz de preenchimento, céu colorido, sombra projetada de um relevo
  sobre outro, oclusão de ambiente, variação por hora do dia ou estação,
  nem qualquer efeito de neblina, perspectiva atmosférica ou atenuação por
  distância.
- **FR-013**: A funcionalidade NÃO DEVE alterar o mapa base, o recorte de
  dados geográficos, o plano de câmera, ou a malha do terreno e a forma
  como ela é construída.

### Entidades-Chave *(incluir se a funcionalidade envolver dados)*

- **Luz Direcional do Terreno**: a direção e a altura fixas, documentadas
  e não configuráveis usadas para decidir, em cada ponto do terreno, o
  quanto ele está voltado para a luz ou para o lado oposto.
- **Faixa de Iluminação**: o intervalo fixo e documentado entre o tom mais
  claro e o mais escuro que a modulação de cor pode produzir sobre a cor
  do mapa base.
- **Vizinhança de Amostragem da Inclinação**: a área ao redor de um ponto
  do terreno, proporcional ao quanto o pixel da tela correspondente cobre
  de chão no mundo real, usada para calcular a inclinação daquele ponto de
  forma estável e suave.

## Critérios de Sucesso *(obrigatório)*

### Resultados Mensuráveis

- **SC-001**: Em qualquer quadro com relevo variado, duas encostas do
  mesmo acidente de terreno, uma voltada para a luz e outra para o lado
  oposto, apresentam tons visivelmente diferentes entre si, mesmo vestidas
  com a mesma cor de mapa base.
- **SC-002**: Em qualquer quadro, o conteúdo do mapa base (ruas, rótulos,
  textos) permanece reconhecível mesmo na área mais escura do quadro.
- **SC-003**: Ao longo de um voo sobre um mesmo trecho de terreno, a
  iluminação de uma área cuja distância até a câmera varia pouco entre
  quadros consecutivos também varia pouco, sem saltos perceptíveis de um
  quadro para o outro.
- **SC-004**: Dois quadros gerados a partir do mesmo plano de câmera e do
  mesmo recorte de dados, em máquinas diferentes ou com números diferentes
  de processos de desenho, são idênticos byte a byte.
- **SC-005**: Nenhum conjunto de quadros desenhado antes desta
  funcionalidade é reaproveitado ou misturado com quadros desenhados
  depois dela, numa retomada de `render all` ou numa execução de `fly
  --keep`.
- **SC-006**: O traçado da atividade, o marcador, as sobreposições de
  tela, o padrão de "terreno sem mapa" e o padrão de "terreno sem
  elevação" aparecem com exatamente a mesma cor que teriam sem esta
  funcionalidade.

## Suposições

- A direção e a altura exatas da luz, e os valores exatos da faixa fixa
  entre o tom mais claro e o mais escuro, são decisões de design técnico a
  serem tomadas e documentadas durante o planejamento desta funcionalidade
  — não são informadas pelo usuário nem dependem de esclarecimento de
  escopo.
- A forma exata de dimensionar a vizinhança de amostragem da inclinação
  conforme a distância da câmera (por exemplo, quantos pontos da grade de
  elevação considerar) é um detalhe de design resolvido no planejamento
  técnico, desde que cumpra a exigência de estabilidade e suavidade
  descrita nos requisitos.
- Esta funcionalidade não adiciona nenhuma flag, comando ou parâmetro novo
  à interface de linha de comando; o efeito aparece automaticamente em
  `render frame`, `render all` e `fly`.
- Mapas base de qualquer tipo que o usuário já tenha registrado (satélite,
  ruas, ou outro) continuam sendo usados sem nenhuma preparação ou
  conversão adicional.
