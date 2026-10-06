# Especificação de Funcionalidade: Acabamento das Sobreposições de Tela

**Branch da Funcionalidade**: `011-overlay-polish`

**Criado em**: 2026-10-02

**Status**: Implementada, com `quickstart.md` validado por inteiro com o `ffmpeg` real

**Entrada**: Descrição do usuário: "Décima primeira etapa do Sobrevoo. Esta etapa não acrescenta informação nenhuma ao vídeo: ela corrige o acabamento das sobreposições que a nona etapa introduziu, agora que foi possível vê-las num quadro real. Hoje o texto é desenhado a partir de uma fonte de bitmap ampliada por fator inteiro, o que o deixa visivelmente serrilhado na resolução do vídeo; os três blocos numéricos têm painéis de larguras diferentes, conforme o texto de cada um, o que deixa a pilha irregular; o marcador que percorre o perfil de elevação é pequeno demais para ser visto num celular; o texto conta apenas com o painel escurecido para se destacar, e some sobre um fundo claro; e o conjunto fica perto demais da borda inferior, justamente a faixa que as redes sociais cobrem com legenda e botões. Com esta etapa, o texto passa a ser desenhado a partir de uma fonte vetorial embutida na própria ferramenta, rasterizada com suavização em qualquer tamanho; os blocos numéricos passam a compartilhar a mesma largura; o marcador do perfil passa a ter tamanho proporcional à altura do quadro, com mínimo em pixels, como o marcador do mapa já tem; o texto ganha um contorno escuro que o mantém legível sobre qualquer fundo; e as sobreposições passam a respeitar uma margem de segurança maior, documentada e adequada ao vídeo vertical. Requisitos: a fonte DEVE estar embutida no binário, com licença permissiva compatível com a licença do projeto, e NUNCA DEVE depender de nenhuma fonte instalada na máquina; a rasterização DEVE ser feita pela própria ferramenta, de modo que o mesmo texto, no mesmo tamanho, produza sempre exatamente os mesmos pixels em qualquer máquina; o mesmo plano, o mesmo recorte, a mesma resolução, a mesma aparência e a mesma configuração de sobreposição DEVEM continuar produzindo imagens idênticas byte a byte; como esta etapa muda os pixels de quadros que o usuário não pediu para mudar, a versão do desenho DEVE subir, de modo que quadros desenhados por uma versão anterior nunca sejam tomados como do mesmo conjunto — nem numa retomada, nem num reaproveitamento do comando único; os blocos numéricos DEVEM ter a mesma largura entre si, definida pelo bloco mais largo, mantendo a ordem e o espaçamento atuais; o marcador do perfil DEVE ser dimensionado como fração da altura do quadro, com um mínimo em pixels documentado; o texto DEVE permanecer legível sobre fundo claro e sobre fundo escuro, sem depender de aumentar a opacidade do painel, que taparia mais imagem; a margem de segurança DEVE ser documentada em frações da altura e da largura, maior na borda inferior do que nas demais, e NENHUMA sobreposição pode invadi-la em nenhuma resolução aceita; e nada nesta etapa pode alterar os valores exibidos, quais blocos existem, a configuração de sobreposição, o enquadramento, o terreno, o traçado ou a duração do vídeo. Fora de escopo: permitir que o usuário escolha a fonte, o tamanho do texto, as cores das sobreposições ou a posição dos blocos; acrescentar blocos novos ou textos livres; traduzir ou formatar os valores de outro jeito; e qualquer mudança nas regras de trajeto, câmera, dados geográficos ou codificação."

## Cenários de Usuário e Testes *(obrigatório)*

### História de Usuário 1 - Ler o texto das sobreposições sem serrilhado e sobre qualquer fundo (Prioridade: P1)

Como usuário do Sobrevoo, ao ver um quadro real do voo pela primeira vez, notei que os números das sobreposições saem visivelmente serrilhados — a fonte de hoje é um bitmap ampliado pixel a pixel — e que, sobre um fundo claro, o texto quase desaparece, porque só o painel escurecido por baixo o destaca. Quero que o texto seja desenhado com contornos suaves, em qualquer tamanho, e que permaneça legível tanto sobre um fundo claro quanto sobre um fundo escuro, sem que a ferramenta precise escurecer mais o painel (o que tomaria mais imagem por baixo).

**Por que esta prioridade**: é o problema mais visível e mais grave da nona etapa — um texto serrilhado ou ilegível compromete a sobreposição inteira, independente de qualquer outro acabamento; as demais histórias refinam detalhes que só importam depois que o texto em si está legível.

**Teste Independente**: pode ser totalmente testado desenhando um quadro isolado com as sobreposições ligadas sobre um trajeto com fundo claro e outro com fundo escuro (a cor de fundo é ajustável desde a oitava etapa) e inspecionando visualmente que o contorno das letras é suave (sem serrilhado em bloco) e que o texto se distingue claramente do que está por baixo nos dois casos.

**Cenários de Aceitação**:

1. **Dado** um plano e um recorte válidos, **Quando** o usuário desenha um quadro com as sobreposições ligadas, **Então** cada caractere de texto é desenhado por uma fonte vetorial rasterizada com suavização, sem o serrilhado em blocos quadrados da fonte de bitmap anterior.
2. **Dado** o mesmo quadro desenhado sobre um fundo bem claro, **Quando** o usuário observa o texto das sobreposições, **Então** ele permanece claramente legível, sem se misturar ao fundo.
3. **Dado** o mesmo quadro desenhado sobre um fundo bem escuro ou sobre um relevo escuro por baixo do painel, **Quando** o usuário observa o texto, **Então** ele também permanece claramente legível, pelo mesmo tratamento — sem que a ferramenta precise escurecer o painel além da opacidade de hoje.

---

### História de Usuário 2 - Ver uma pilha de blocos numéricos alinhada (Prioridade: P2)

Como usuário do Sobrevoo, notei que os três blocos numéricos (distância percorrida; elevação e ganho; tempo decorrido) têm painéis de larguras diferentes, cada um ajustado ao texto que carrega, o que faz a pilha vertical deles parecer desalinhada e malfeita. Quero que os três painéis tenham sempre a mesma largura entre si — a do bloco que precisar da maior largura naquele momento —, mantendo a ordem e o espaçamento de hoje, para que a pilha pareça um único elemento de interface, não três caixas desencontradas.

**Por que esta prioridade**: é um defeito de acabamento visível em qualquer quadro com sobreposições ligadas, mas menos grave que a legibilidade do texto em si (História 1); não depende de nenhuma outra história desta etapa.

**Teste Independente**: pode ser totalmente testado desenhando um quadro isolado com os três blocos numéricos ligados e medindo que a largura dos três painéis é idêntica, e desenhando um segundo quadro em que um dos três textos é notavelmente mais longo que nos outros dois blocos (por exemplo, perto do fim de um voo longo) e confirmando que os três painéis crescem juntos para a mesma largura.

**Cenários de Aceitação**:

1. **Dado** um quadro com os três blocos numéricos (distância; elevação e ganho; tempo decorrido) ligados, **Quando** o usuário mede a largura dos três painéis, **Então** os três têm exatamente a mesma largura, igual à do bloco que, naquele quadro, precisaria da maior largura para caber seu próprio texto.
2. **Dado** o mesmo quadro, **Quando** o usuário observa a ordem e o espaçamento entre os blocos, **Então** eles permanecem os mesmos de antes desta etapa — só a largura dos painéis muda.
3. **Dado** um quadro em que só um ou dois dos três blocos numéricos estão presentes (porque o usuário desligou algum pelo bloco, ou porque o trajeto não tem o dado de um deles), **Quando** o usuário observa os painéis presentes, **Então** eles continuam com a mesma largura entre si, sem referência a um bloco que não está sendo desenhado.

---

### História de Usuário 3 - Enxergar o marcador do perfil de elevação num celular (Prioridade: P3)

Como usuário do Sobrevoo, ao ver o vídeo num celular, notei que o marcador que percorre o perfil de elevação é pequeno demais para enxergar — ele some no gráfico de tão discreto. Quero que esse marcador tenha um tamanho proporcional à altura do quadro, com um mínimo garantido em pixels, do mesmo jeito que o marcador do mapa já funciona, para que ele seja visível em qualquer resolução, inclusive numa tela pequena.

**Por que esta prioridade**: afeta só o bloco de perfil de elevação, um dos quatro blocos de sobreposição, e não compromete a legibilidade dos demais; é um ajuste de tamanho isolado, sem dependência das outras histórias.

**Teste Independente**: pode ser totalmente testado desenhando quadros em diferentes resoluções (incluindo a menor aceita pela ferramenta) com o bloco de perfil ligado e medindo que o marcador sobre o perfil nunca fica menor que o mínimo em pixels documentado, crescendo proporcionalmente em resoluções maiores.

**Cenários de Aceitação**:

1. **Dado** um quadro numa resolução comum de vídeo vertical, **Quando** o usuário mede o marcador sobre o perfil de elevação, **Então** seu tamanho é proporcional à altura do quadro, visivelmente maior que o de antes desta etapa.
2. **Dado** um quadro na menor resolução aceita pela ferramenta, **Quando** o usuário mede esse mesmo marcador, **Então** ele nunca é menor que o mínimo em pixels documentado, mesmo quando a proporção calculada resultaria em algo menor.
3. **Dado** dois quadros em resoluções diferentes, mas de mesma proporção de tela, **Quando** o usuário compara o marcador do perfil nos dois, **Então** ele ocupa a mesma fração visual do quadro nos dois casos (acima do piso mínimo).

---

### História de Usuário 4 - Publicar o vídeo numa rede social sem perder sobreposições na borda (Prioridade: P4)

Como usuário do Sobrevoo, pretendo publicar o vídeo vertical em redes sociais que cobrem a borda inferior da tela com legenda, nome de usuário e botões de interação, e as bordas laterais com um corte menor. Hoje as sobreposições ficam perto demais da borda inferior, correndo o risco de ficar escondidas atrás dessa interface. Quero que a ferramenta deixe uma margem de segurança maior nas quatro bordas — maior ainda na borda inferior — para que nenhuma sobreposição fique na faixa que essas redes tipicamente cobrem.

**Por que esta prioridade**: importa especificamente para quem publica o vídeo numa rede social; é o ajuste mais específico de uso entre as quatro histórias, e não afeta a legibilidade do texto em si nem o alinhamento dos painéis.

**Teste Independente**: pode ser totalmente testado desenhando um quadro vertical (9:16) com todos os blocos ligados e medindo que nenhum pixel de sobreposição cai dentro da margem de segurança documentada, em especial a faixa inferior, maior que as demais.

**Cenários de Aceitação**:

1. **Dado** um quadro de vídeo vertical (9:16) com as sobreposições ligadas, **Quando** o usuário mede a distância entre o pixel mais baixo de qualquer sobreposição e a borda inferior do quadro, **Então** essa distância é pelo menos a margem de segurança inferior documentada, maior que a margem das outras três bordas.
2. **Dado** o mesmo quadro, **Quando** o usuário mede a distância de qualquer sobreposição até as bordas superior e laterais, **Então** essa distância é pelo menos a margem de segurança documentada para essas bordas (menor que a inferior, mas nunca zero).
3. **Dado** quadros em qualquer resolução aceita pela ferramenta, **Quando** o usuário repete essa medição, **Então** nenhuma sobreposição invade a margem de segurança, em nenhuma das resoluções.

---

### Casos Extremos

- O que acontece com um conjunto de quadros (`render all` interrompido, ou intermediários guardados por `fly --keep`) desenhado por uma versão da ferramenta anterior a esta etapa? A ferramenta os trata como um conjunto diferente do que produziria agora — pela mesma regra que já recusa reaproveitar quadros de outro plano, recorte, resolução, aparência ou configuração de sobreposição — e recusa reaproveitá-los sem a sobrescrita pedida explicitamente, redesenhando do zero quando ela é pedida.
- O que acontece quando apenas um dos três blocos numéricos está presente num quadro (os outros dois desligados ou sem dado disponível)? Esse único painel usa a própria largura necessária, sem ser forçado a combinar com um bloco que não está sendo desenhado.
- O que acontece na menor resolução aceita pela ferramenta, onde a margem de segurança maior e o marcador do perfil com piso mínimo competem por espaço com o restante da cena? A margem de segurança e os mínimos em pixels são escolhidos de modo que caibam em qualquer resolução aceita hoje pela ferramenta, sem se sobrepor entre si nem exigir uma resolução menor que a já recusada por outro motivo.
- O que acontece com o traçado, o marcador sobre o terreno, o enquadramento da câmera ou a duração do vídeo? Nada muda — esta etapa só altera o que é desenhado nas sobreposições de tela; a cena por baixo permanece bit a bit a mesma de antes, exceto pelos pixels cobertos pelas próprias sobreposições.
- O que acontece ao desenhar o mesmo quadro duas vezes, com o mesmo plano, recorte, resolução, aparência e configuração de sobreposição, depois desta etapa em vigor? O resultado continua sendo exatamente o mesmo, byte a byte, nas duas vezes — a suavização da fonte é determinística, não introduz nenhuma variação entre execuções.
- O que acontece quando as sobreposições estão desligadas por inteiro (`--overlays=false`)? Nada desta etapa se aplica — o quadro continua sem nenhum texto, painel ou marcador de perfil desenhado, exatamente como hoje.

## Requisitos *(obrigatório)*

### Requisitos Funcionais

- **FR-001**: Todo texto das sobreposições DEVE ser desenhado a partir de uma fonte vetorial embutida no binário da ferramenta, com licença permissiva compatível com a licença do projeto — nunca uma fonte instalada no sistema operacional.
- **FR-002**: A rasterização do texto DEVE ser feita pela própria ferramenta, de modo que o mesmo texto, no mesmo tamanho, produza sempre exatamente os mesmos pixels, em qualquer máquina, sistema operacional ou número de processos/threads.
- **FR-003**: O texto rasterizado DEVE ter contornos suaves (com atenuação de borda), eliminando o serrilhado em blocos visível com a fonte de bitmap ampliada por fator inteiro que esta etapa substitui.
- **FR-004**: Todo texto das sobreposições DEVE receber um contorno escuro, visualmente distinto do próprio texto, que mantenha a legibilidade sobre qualquer cor de fundo ou relevo por baixo do painel — sem depender de aumentar a opacidade do painel.
- **FR-005**: Os painéis dos três blocos numéricos (distância percorrida; elevação e ganho; tempo decorrido) DEVEM compartilhar sempre a mesma largura entre si, igual à largura exigida pelo mais largo deles naquele quadro, mantendo a ordem e o espaçamento entre blocos de hoje; um bloco numérico ausente (desligado ou sem dado) não influencia a largura dos demais.
- **FR-006**: O marcador que indica a posição atual sobre o perfil de elevação DEVE ser dimensionado como uma fração da altura do quadro, sujeito a um mínimo em pixels documentado — pelo mesmo princípio que já dimensiona o marcador sobre o terreno (proporção da altura, com piso mínimo).
- **FR-007**: As sobreposições DEVEM respeitar uma margem de segurança documentada em frações da altura e da largura do quadro, a partir de cada uma das quatro bordas, sendo a margem da borda inferior maior que a das outras três; nenhum pixel de texto, painel ou marcador de perfil pode cair dentro dessa margem, em nenhuma resolução aceita pela ferramenta.
- **FR-008**: A versão do desenho DEVE subir em relação à etapa anterior, de modo que um conjunto de quadros desenhado por uma versão anterior da ferramenta nunca seja considerado o mesmo conjunto que o produzido por esta versão — nem ao retomar um `render all` interrompido, nem ao reaproveitar os intermediários guardados por `fly --keep`.
- **FR-009**: O mesmo plano de câmera, o mesmo recorte de dados geográficos, a mesma resolução, a mesma aparência e a mesma configuração de sobreposição DEVEM continuar produzindo, nesta versão, imagens idênticas byte a byte entre execuções diferentes.
- **FR-010**: Esta etapa NÃO DEVE alterar nenhum valor exibido pelas sobreposições, quais blocos existem, a configuração de sobreposição (como é ligada, desligada ou escolhida por bloco), o enquadramento da câmera, o terreno desenhado, o traçado, a posição do marcador sobre o terreno, ou a duração do vídeo — o efeito desta etapa se limita à forma como o texto, os painéis, o marcador de perfil e a margem de segurança são desenhados.
- **FR-011**: Esta etapa NÃO DEVE introduzir nenhuma escolha nova para o usuário — fonte, tamanho de texto, cores das sobreposições e posição dos blocos continuam fora do controle do usuário, exatamente como hoje.

### Entidades-Chave

- **Fonte Vetorial Embutida**: a fonte usada para rasterizar todo texto das sobreposições, embutida no binário, com licença permissiva compatível com a do projeto; substitui a fonte de bitmap da etapa anterior sem introduzir escolha de fonte pelo usuário.
- **Largura Compartilhada dos Painéis Numéricos**: a largura única aplicada aos painéis dos blocos de distância, elevação+ganho e tempo decorrido presentes num quadro, determinada pelo mais largo deles naquele quadro.
- **Marcador do Perfil de Elevação**: o indicador de posição sobre o gráfico de perfil de elevação, agora dimensionado como fração da altura do quadro com um piso mínimo em pixels, no mesmo padrão já usado pelo marcador sobre o terreno.
- **Margem de Segurança**: a faixa, a partir de cada uma das quatro bordas do quadro, dentro da qual nenhuma sobreposição é desenhada; documentada em frações da altura e da largura, com a borda inferior maior que as demais.
- **Versão do Desenho**: o identificador que distingue conjuntos de quadros produzidos por versões diferentes do desenho da ferramenta; sobe nesta etapa porque os pixels de um quadro com sobreposições mudam, mesmo sem nenhuma mudança pedida pelo usuário ao plano, ao recorte, à aparência ou à configuração de sobreposição.

## Critérios de Sucesso *(obrigatório)*

### Resultados Mensuráveis

- **SC-001**: Um usuário consegue ler todos os números das sobreposições, sem serrilhado perceptível, em qualquer tamanho de vídeo que a ferramenta produza.
- **SC-002**: O texto das sobreposições permanece legível tanto sobre o fundo mais claro quanto sobre o mais escuro que a ferramenta pode produzir, sem que a opacidade do painel precise aumentar.
- **SC-003**: Em qualquer quadro com dois ou três blocos numéricos presentes, os painéis desses blocos têm exatamente a mesma largura entre si.
- **SC-004**: O marcador do perfil de elevação é visível a olho nu numa tela de celular, em qualquer resolução de vídeo que a ferramenta aceite.
- **SC-005**: Num vídeo vertical (9:16), nenhuma sobreposição cai dentro da margem de segurança documentada, com a margem inferior sempre maior que as demais.
- **SC-006**: Nenhum conjunto de quadros produzido por uma versão da ferramenta anterior a esta etapa é reaproveitado, por retomada ou por `fly --keep`, sem que o usuário peça a sobrescrita explicitamente.
- **SC-007**: O mesmo plano, recorte, resolução, aparência e configuração de sobreposição produzem, nesta versão, sempre a mesma imagem, byte a byte, em qualquer máquina.
- **SC-008**: Um usuário que já tinha um vídeo satisfatório quanto a valores, blocos e enquadramento não precisa reconfigurar nada para obter o acabamento desta etapa — basta gerar o vídeo de novo.

## Suposições

- Os quatro blocos de sobreposição, os valores que cada um exibe e a configuração de sobreposição (ligar/desligar por inteiro ou por bloco) continuam exatamente como a nona etapa definiu; esta etapa só muda a forma como eles são desenhados, nunca o que é desenhado.
- "Os três blocos numéricos" citados nos requisitos de largura compartilhada são os blocos de distância percorrida, de elevação do trajeto e ganho acumulado, e de tempo decorrido — o quarto bloco (perfil de elevação) tem forma de gráfico, não de painel de texto curto, e fica fora dessa regra de largura compartilhada.
- A largura compartilhada dos painéis numéricos é recalculada a cada quadro a partir do texto efetivamente desenhado naquele quadro; como a distância percorrida e o tempo decorrido só crescem ao longo do voo, e a elevação/ganho varia dentro de poucos dígitos, a largura tende a crescer de forma suave ao longo do vídeo, sem produzir uma oscilação perceptível de quadro a quadro.
- Os valores exatos da margem de segurança por borda, do mínimo em pixels do marcador do perfil, da cor e espessura do contorno do texto, e o nome/licença da fonte vetorial escolhida são decisões de planejamento técnico, documentadas quando definidas; o pedido exige apenas que existam, sejam documentadas e cumpram as propriedades descritas (margem inferior maior, piso mínimo em pixels, legibilidade sobre qualquer fundo).
- A fonte vetorial, como a fonte de bitmap que substitui, é única e fixa, escolhida pela ferramenta — escolher a fonte ou o tamanho do texto continua fora de escopo.
- Esta etapa não lê de novo o trajeto GPS nem os dados geográficos registrados, e não adiciona nenhum campo novo ao plano de câmera ou ao recorte exportado — toda a informação exibida já existia desde a nona etapa; só o desenho dela muda.
- A mudança de versão do desenho é o único mecanismo necessário para impedir a mistura de quadros de antes e depois desta etapa; nenhum registro ou migração adicional é necessário além do que o mecanismo de identidade de conjunto de quadros já oferece.
