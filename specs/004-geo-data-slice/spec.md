# Especificação de Funcionalidade: Recorte de Dados Geográficos para o Voo

**Branch da Funcionalidade**: `004-geo-data-slice`

**Criado em**: 2026-09-20

**Status**: Rascunho

**Entrada**: Descrição do usuário: "Quarta etapa do Sobrevoo. Esta etapa cobre só a leitura do conteúdo dos dados geográficos que o usuário já registrou. Nada de projetar em 3D, desenhar, renderizar quadros ou gerar vídeo ainda. Até aqui a ferramenta só olhou os metadados dos arquivos registrados, para saber que área cada um cobre; agora ela lê o conteúdo de verdade. A partir de um plano de câmera gerado na etapa anterior, a ferramenta reúne o recorte de dados necessário para aquele voo: os valores de elevação do terreno sob a área que a câmera vai percorrer e as peças de imagem do mapa base dessa mesma área, no nível de detalhe adequado à distância em que a câmera voa. O usuário consegue: pedir o recorte de um plano; ver um resumo dele (área coberta, nível de detalhe escolhido e por quê, quantidade de peças de mapa e de amostras de elevação, faixa de elevação encontrada, quais registros foram usados, tamanho do recorte); exportar o recorte para ser consumido pelas etapas seguintes; e consultar a elevação de uma coordenada isolada, para conferir o resultado contra uma fonte externa. Requisitos: usar exclusivamente os arquivos registrados na segunda etapa, sem rede e sem baixar nada; recusar, com mensagem clara, quando a área do plano não estiver totalmente coberta por mapa base e relevo, indicando o que falta, reaproveitando a verificação de cobertura já existente; escolher o nível de detalhe do mapa base de forma determinística e explicável a partir da distância da câmera e da área do voo, dentro do que a fonte registrada oferece, e informar a escolha; reportar elevação sempre em metros, deixando claro quando o arquivo de origem não informa valor para um ponto, sem inventar dado nem falhar por causa disso; lidar com peça de mapa ausente dentro de uma fonte registrada sem interromper o recorte, informando quantas faltaram e onde; recusar, com mensagem clara e limite documentado, um recorte grande demais para ser mantido; o mesmo plano com os mesmos registros produz sempre exatamente o mesmo recorte; comportamento idêntico em qualquer lugar do planeta, inclusive cruzando o meridiano de 180° e em latitudes altas; a exportação recusa por padrão um destino já existente, com opção explícita de sobrescrita, e nunca deixa resultado parcial, como na etapa anterior. Fora de escopo: baixar ou converter dados, projeção em 3D, desenho, texturização, renderização de quadros, vídeo, sobreposições, interface gráfica e API."

## Clarificações

### Sessão 2026-09-20

- Q: O recorte deve ser pedido a partir do trajeto com os parâmetros do plano, ou a partir do arquivo de plano exportado? → A: A partir do arquivo de plano exportado na etapa 3 (opção B). O plano é o contrato entre as etapas: o recorte usa exatamente o plano que o usuário exportou e inspecionou, sem regenerá-lo a partir do trajeto.

## Cenários de Usuário e Testes *(obrigatório)*

### História de Usuário 1 - Obter o recorte de dados de um plano de câmera (Prioridade: P1)

Como usuário do Sobrevoo, eu já registrei meus mapas base e meus dados de relevo (segunda etapa) e já exportei o plano de câmera de um trajeto (terceira etapa). Quero pedir à ferramenta o recorte de dados daquele voo: os valores de elevação do terreno sob a área que a câmera vai percorrer e as peças de imagem do mapa base dessa mesma área, no nível de detalhe adequado à distância em que a câmera voa. Assim, as etapas seguintes recebem apenas o que o voo precisa, e não os arquivos inteiros.

**Por que esta prioridade**: é o núcleo da etapa. Sem o recorte não há o que resumir, exportar ou conferir, e o desenho do terreno e do mapa nas etapas seguintes depende dele.

**Teste Independente**: pode ser totalmente testado registrando um mapa base e um relevo de exemplo que cobrem um trajeto de exemplo, exportando o plano desse trajeto com os parâmetros padrão, pedindo o recorte a partir do arquivo exportado e verificando que o resultado contém peças de mapa e amostras de elevação que cobrem toda a área do voo, sem baixar nada nem usar arquivos não registrados.

**Cenários de Aceitação**:

1. **Dado** um arquivo de plano exportado válido e registros de mapa base e de relevo que cobrem toda a área do plano, **Quando** o usuário pede o recorte a partir desse arquivo, **Então** a ferramenta produz um recorte com as amostras de elevação e as peças de mapa base que cobrem toda a área que a câmera percorre, todas lidas exclusivamente dos arquivos registrados.
2. **Dado** o mesmo plano e os mesmos registros, **Quando** o usuário pede o recorte duas vezes (inclusive em execuções e momentos diferentes), **Então** os dois recortes são exatamente iguais, com o mesmo conteúdo na mesma ordem.
3. **Dado** que a área do plano não está totalmente coberta por mapa base, por relevo, ou por ambos, **Quando** o usuário pede o recorte, **Então** a ferramenta recusa, sem produzir recorte algum, e informa o que falta: para cada subtrecho contínuo não coberto, as coordenadas de início e de fim e se falta mapa base, relevo ou ambos — o mesmo relatório da verificação de cobertura da segunda etapa.
4. **Dado** um registro cujo arquivo não está mais no caminho registrado, **Quando** o usuário pede o recorte, **Então** esse registro não é usado, exatamente como na verificação de cobertura da segunda etapa, e a recusa (se houver) diz que falta cobertura, sem falha inesperada.
5. **Dado** um caminho de plano que não existe, ou que existe mas não pode ser lido, **Quando** o usuário pede o recorte, **Então** a ferramenta recusa com uma mensagem clara que diz qual é o problema, sem produzir recorte algum.
6. **Dado** um arquivo que não é um plano de câmera exportado (conteúdo que não é um plano, campos obrigatórios ausentes, arquivo truncado ou corrompido), **Quando** o usuário pede o recorte, **Então** a ferramenta recusa com erro próprio e mensagem que diz o que está errado no arquivo, sem produzir recorte algum.
7. **Dado** um arquivo de plano exportado numa versão do formato que a ferramenta não reconhece, **Quando** o usuário pede o recorte, **Então** a ferramenta recusa com erro próprio e mensagem que informa a versão encontrada e as versões aceitas.
8. **Dado** um arquivo de plano cujo conteúdo é incoerente consigo mesmo (por exemplo, a quantidade de quadros não bate com a duração e a taxa de quadros, ou há coordenadas fora do intervalo válido), **Quando** o usuário pede o recorte, **Então** a ferramenta recusa com erro próprio e mensagem que aponta a incoerência encontrada, em vez de recortar a partir de dados duvidosos.
9. **Dado** que há mais de um registro do mesmo tipo cobrindo a mesma área, **Quando** o usuário pede o recorte, **Então** vale a mesma regra de escolha da segunda etapa (o registro de menor área coberta; em empate, o mais antigo), de forma que o recorte é sempre o mesmo.

---

### História de Usuário 2 - Nível de detalhe do mapa base explicável (Prioridade: P2)

Como usuário do Sobrevoo, eu quero que a ferramenta escolha sozinha o nível de detalhe das peças de mapa base, sem que eu precise conhecer a pirâmide de níveis do arquivo: quando a câmera voa perto, precisa de peças mais detalhadas; quando voa longe, peças mais gerais bastam e evitam um recorte enorme. Quero que essa escolha seja sempre a mesma para o mesmo plano, que a ferramenta me diga qual nível escolheu e por quê, e que ela respeite o que o arquivo registrado realmente oferece.

**Por que esta prioridade**: sem uma escolha sensata do nível de detalhe, o recorte sai borrado (nível baixo demais) ou gigante (nível alto demais). Depende do recorte existir (P1), mas é o que o torna utilizável e previsível.

**Teste Independente**: pode ser testado com planos de câmera de distâncias diferentes sobre o mesmo mapa base e verificando que o nível escolhido é mais detalhado para a câmera mais próxima e menos detalhado para a mais distante, que fica sempre dentro do intervalo de níveis que o arquivo oferece, e que o resumo explica a escolha.

**Cenários de Aceitação**:

1. **Dado** dois planos sobre a mesma região, um com a câmera baixa e outro alto (distâncias diferentes, mesmo trajeto), **Quando** o usuário pede o recorte de cada um, **Então** o nível de detalhe escolhido para a câmera mais próxima é maior ou igual ao da mais distante, e o recorte da mais próxima tem peças mais detalhadas.
2. **Dado** um plano cujo nível de detalhe ideal seria mais detalhado do que o máximo que o mapa base registrado oferece, **Quando** o usuário pede o recorte, **Então** a ferramenta usa o nível máximo disponível e o resumo informa que o nível ideal não estava disponível e qual foi o limite da fonte.
3. **Dado** um plano cujo nível ideal seria menos detalhado do que o mínimo que o mapa base registrado oferece, **Quando** o usuário pede o recorte, **Então** a ferramenta usa o nível mínimo disponível e o resumo informa isso.
4. **Dado** qualquer recorte gerado, **Quando** o usuário lê o resumo, **Então** ele informa o nível de detalhe escolhido e o motivo em termos compreensíveis (por exemplo, a distância da câmera e a extensão da área do voo que o determinaram, e se o resultado foi limitado pelo que a fonte oferece).
5. **Dado** o mesmo plano e os mesmos registros, **Quando** o usuário pede o recorte várias vezes, **Então** o nível de detalhe escolhido é sempre o mesmo.

---

### História de Usuário 3 - Ver o resumo do recorte (Prioridade: P3)

Como usuário do Sobrevoo, eu quero ver um resumo do recorte — a área coberta, o nível de detalhe escolhido e o motivo, a quantidade de peças de mapa e de amostras de elevação, a faixa de elevação encontrada, os registros usados e o tamanho do recorte — para julgar rapidamente se ele está adequado sem precisar examinar o conteúdo.

**Por que esta prioridade**: o recorte já existe e é utilizável sem o resumo; o resumo acelera a inspeção e ajuda a decidir se vale mudar o plano (por exemplo, uma câmera mais alta gera um recorte menor).

**Teste Independente**: pode ser testado pedindo o recorte de um plano conhecido e conferindo que cada valor do resumo corresponde ao que se calcula a partir do conteúdo do recorte.

**Cenários de Aceitação**:

1. **Dado** um recorte gerado, **Quando** o usuário o visualiza, **Então** o resumo apresenta a área geográfica coberta, o nível de detalhe escolhido e o motivo, a quantidade de peças de mapa base, a quantidade de amostras de elevação, as elevações mínima e máxima encontradas (em metros), o nome e o tipo de cada registro usado e o tamanho total do recorte, em unidades identificadas.
2. **Dado** um recorte em que algumas amostras de elevação não têm valor informado pelo arquivo, **Quando** o usuário visualiza o resumo, **Então** o resumo informa a quantidade de amostras sem valor, e a faixa de elevação é calculada apenas com as amostras que têm valor.
3. **Dado** um recorte em que nenhuma amostra de elevação tem valor, **Quando** o usuário visualiza o resumo, **Então** o resumo diz explicitamente que não há faixa de elevação a informar, em vez de mostrar um número inventado.
4. **Dado** um recorte em que faltaram peças de mapa (ver História 5), **Quando** o usuário visualiza o resumo, **Então** o resumo informa quantas peças faltaram e onde.

---

### História de Usuário 4 - Exportar o recorte (Prioridade: P4)

Como usuário do Sobrevoo, eu quero exportar o recorte para um destino, para que as etapas seguintes (desenho do terreno e do mapa, renderização) o consumam sem precisar reler os arquivos originais nem recalcular nada.

**Por que esta prioridade**: é a ponte para as próximas etapas, mas o resumo (P3) já atende à inspeção rápida e o recorte em si já cumpre o valor de P1.

**Teste Independente**: pode ser testado exportando o recorte de um plano, relendo o que foi exportado e verificando que ele contém todas as peças de mapa e todas as amostras de elevação, com os mesmos valores do recorte gerado, além do resumo e da procedência.

**Cenários de Aceitação**:

1. **Dado** um recorte gerado, **Quando** o usuário pede a exportação para um destino, **Então** o destino é criado contendo o recorte completo — as amostras de elevação com a indicação de quais não têm valor, as peças de mapa base com sua posição e nível, e o resumo (incluindo os registros usados) — de forma que outro programa possa consumi-lo sem acesso aos arquivos originais.
2. **Dado** o mesmo plano e os mesmos registros, **Quando** o usuário exporta o recorte duas vezes, **Então** os dois resultados são idênticos byte a byte.
3. **Dado** que o destino não pode ser gravado (local inexistente, sem permissão), **Quando** o usuário pede a exportação, **Então** a ferramenta recusa com uma mensagem clara sobre o destino, sem deixar resultado parcial.
4. **Dado** que o destino já existe, **Quando** o usuário pede a exportação sem a opção explícita de sobrescrita, **Então** a ferramenta recusa com uma mensagem clara informando que o destino já existe e como sobrescrevê-lo, e o conteúdo existente permanece intacto.
5. **Dado** que o destino já existe, **Quando** o usuário pede a exportação com a opção explícita de sobrescrita, **Então** o destino passa a conter integralmente o novo recorte, sem conteúdo parcial ou misturado com o anterior.
6. **Dado** uma falha no meio da exportação (por exemplo, falta de espaço em disco), **Quando** a exportação é interrompida, **Então** nenhum resultado parcial permanece e, se havia um destino anterior, ele permanece intacto.

---

### História de Usuário 5 - Tolerar peça de mapa ausente e recusar recorte grande demais (Prioridade: P5)

Como usuário do Sobrevoo, eu sei que arquivos de mapa base reais podem ter buracos: peças que o arquivo não contém, mesmo dentro da área que ele declara cobrir. Quero que o recorte não seja interrompido por isso, e que a ferramenta me diga quantas peças faltaram e onde. Quero também que a ferramenta recuse, de forma clara, um recorte grande demais para ser mantido, em vez de travar ou consumir toda a memória do computador.

**Por que esta prioridade**: são proteções de robustez. O caminho feliz (P1 a P4) já entrega valor, mas sem elas o comportamento com dados reais imperfeitos ou com voos muito extensos seria imprevisível.

**Teste Independente**: pode ser testado com um mapa base de exemplo do qual foram removidas algumas peças, verificando que o recorte é produzido e que as peças faltantes são listadas; e com um plano cuja área exigiria mais do que o limite documentado, verificando a recusa.

**Cenários de Aceitação**:

1. **Dado** um mapa base registrado que declara cobrir a área do plano mas não contém algumas das peças necessárias, **Quando** o usuário pede o recorte, **Então** o recorte é produzido com as peças disponíveis, e a ferramenta informa a quantidade de peças ausentes e a posição de cada uma (nível e coordenadas da peça), sem interromper o recorte nem terminar com erro.
2. **Dado** que o recorte tem peças ausentes, **Quando** o usuário o exporta, **Então** a exportação registra quais peças faltaram, de modo que as etapas seguintes saibam que ali não há imagem.
3. **Dado** um plano cuja área, no nível de detalhe escolhido, exigiria um recorte maior do que o limite documentado, **Quando** o usuário pede o recorte, **Então** a ferramenta recusa com erro próprio e mensagem que informa o tamanho estimado, o limite máximo aceito e a causa provável (área do voo grande ou câmera muito próxima), sem começar a ler dados.
4. **Dado** um recorte de tamanho exatamente igual ao limite, **Quando** o usuário o pede, **Então** ele é aceito; um valor imediatamente acima é recusado.

---

### História de Usuário 6 - Consultar a elevação de uma coordenada (Prioridade: P6)

Como usuário do Sobrevoo, eu quero perguntar à ferramenta qual é a elevação do terreno numa coordenada isolada (latitude e longitude), para conferir o que ela lê dos meus arquivos contra uma fonte externa (um mapa topográfico, um GPS, outro programa) e ganhar confiança nos dados antes de gerar um voo.

**Por que esta prioridade**: é uma ferramenta de verificação, independente do recorte de um plano; útil, mas não bloqueia nenhuma outra história.

**Teste Independente**: pode ser testado registrando um relevo de exemplo com valores conhecidos e consultando coordenadas cujos valores esperados são conhecidos (um ponto num valor de amostra, um ponto sem valor informado, um ponto fora da cobertura) e conferindo a resposta.

**Cenários de Aceitação**:

1. **Dado** uma coordenada coberta por um relevo registrado, **Quando** o usuário consulta a elevação, **Então** a ferramenta informa a elevação em metros e qual registro de relevo foi usado.
2. **Dado** uma coordenada coberta por um relevo registrado cujo arquivo não informa valor para aquele ponto, **Quando** o usuário consulta a elevação, **Então** a ferramenta informa claramente que o arquivo não tem valor para aquele ponto, sem inventar um número e sem terminar com erro.
3. **Dado** uma coordenada que nenhum relevo registrado cobre, **Quando** o usuário consulta a elevação, **Então** a ferramenta informa que nenhum relevo registrado cobre aquele ponto (distinto de "sem valor no arquivo"), com mensagem clara e código de saída próprio.
4. **Dado** que mais de um relevo registrado cobre a coordenada, **Quando** o usuário consulta a elevação, **Então** vale a mesma regra de escolha da segunda etapa (menor área coberta; em empate, o mais antigo), e a resposta informa qual registro foi usado.
5. **Dado** uma latitude fora de -90 a 90 ou uma longitude fora de -180 a 180, ou um valor não numérico, **Quando** o usuário consulta a elevação, **Então** a ferramenta recusa com mensagem que aponta o valor inválido e o intervalo aceito.
6. **Dado** uma coordenada sobre o meridiano de 180° (por exemplo, longitude 180 ou -180), **Quando** o usuário consulta a elevação, **Então** as duas escritas da mesma posição produzem a mesma resposta.
7. **Dado** a mesma coordenada e os mesmos registros, **Quando** o usuário consulta várias vezes, **Então** a resposta é sempre a mesma.

---

### Casos Extremos

- **Voo que cruza o meridiano de 180°**: o recorte é idêntico em qualidade e regras ao de qualquer outro voo — a área é contínua (não varre o planeta ao contrário), as peças de mapa e as amostras de elevação de um lado e do outro do meridiano são reunidas normalmente, mesmo que venham de registros diferentes ou de um único registro que cruza o meridiano.
- **Voo em latitudes altas (próximo aos polos)**: a escolha do nível de detalhe e a quantidade de peças e de amostras continuam corretas apesar de as longitudes convergirem; a área do recorte não é distorcida nem descontínua, e o comportamento não muda por causa da latitude.
- **Valores de "sem dado" do relevo**: amostras que o arquivo marca como sem valor (oceano, falha de aquisição) nunca são tratadas como elevação zero nem como um número qualquer; ficam registradas como sem valor no recorte e contadas no resumo.
- **Relevo em unidade que não seja metros, se o arquivo a declarar**: a elevação é sempre reportada em metros; quando a unidade declarada pelo arquivo não puder ser convertida com segurança para metros, a ferramenta recusa o uso daquele registro com mensagem clara, em vez de reportar número na unidade errada.
- **Peças de mapa ausentes em grande quantidade**: se todas as peças necessárias estiverem ausentes, o recorte ainda é produzido (sem imagens) e o resumo deixa isso evidente; a ferramenta não trata a ausência como sucesso silencioso.
- **Plano em que um único registro de mapa base ou de relevo cobre a área inteira, ou em que vários se complementam**: ambos os casos funcionam; o resumo lista cada registro usado e para qual parte do recorte contribuiu.
- **Registro que cobre a área por metadados, mas cujo conteúdo está corrompido ou ilegível**: a ferramenta recusa o recorte com mensagem clara que identifica o registro problemático, sem terminar com erro genérico e sem produzir recorte parcial silencioso.
- **Recorte com uma única amostra de elevação ou uma única peça de mapa**: é um recorte válido, sem tratamento especial.
- **Área do plano muito pequena** (voo curto e câmera baixa): o recorte é produzido normalmente, com pelo menos uma peça de mapa e uma amostra de elevação cobrindo a área.
- **Registros modificados entre duas execuções**: o recorte reflete o conteúdo atual dos arquivos; a determinação "mesmos registros" implica arquivos com o mesmo conteúdo, e o recorte informa quais registros (nome e caminho) o originaram.

## Requisitos *(obrigatório)*

### Requisitos Funcionais

- **FR-001**: O sistema DEVE permitir que o usuário peça o recorte de dados geográficos de um plano de câmera informando o caminho do arquivo de plano exportado pela terceira etapa; o recorte DEVE ser feito exatamente sobre o plano contido no arquivo, sem regenerá-lo a partir do trajeto nem recalcular nenhum de seus quadros.
- **FR-001a**: O sistema DEVE validar o arquivo de plano antes de usá-lo e recusar, com erro próprio e mensagem clara, um arquivo inexistente, ilegível, que não seja um plano exportado (conteúdo estranho, campos obrigatórios ausentes, truncado ou corrompido), de versão de formato não reconhecida (informando a versão encontrada e as aceitas) ou incoerente consigo mesmo (por exemplo, quantidade de quadros diferente da duração vezes a taxa de quadros, ou coordenadas fora do intervalo válido); nenhum recorte DEVE ser produzido nesses casos.
- **FR-002**: O recorte DEVE conter as amostras de elevação do terreno e as peças de imagem do mapa base que cobrem toda a área que a câmera percorre no plano, incluindo a abertura, o acompanhamento e o fechamento, com uma margem documentada em volta dessa área para conter o terreno visível do ponto de vista da câmera.
- **FR-003**: O sistema DEVE ler o conteúdo dos dados geográficos exclusivamente dos arquivos registrados na segunda etapa, sem acessar rede, sem baixar nada e sem usar arquivos que não estejam registrados; NÃO DEVE converter nem alterar os arquivos originais.
- **FR-004**: Antes de ler qualquer conteúdo, o sistema DEVE verificar que a área do plano está totalmente coberta por mapa base e por relevo, reaproveitando a verificação de cobertura e as regras de escolha de registro da segunda etapa (registro com arquivo presente; menor área coberta, e em empate o mais antigo), sem duplicá-las.
- **FR-005**: Quando a área do plano não estiver totalmente coberta, o sistema DEVE recusar o recorte, sem produzir resultado algum, com erro próprio e mensagem que informe, para cada subtrecho contínuo não coberto, as coordenadas de início e fim e se falta mapa base, relevo ou ambos.
- **FR-006**: O sistema DEVE escolher o nível de detalhe do mapa base de forma determinística e explicável, a partir da distância da câmera ao terreno durante o voo e da extensão da área do voo, e sempre dentro do intervalo de níveis que o mapa base registrado oferece; DEVE informar no resumo o nível escolhido e o motivo, inclusive quando o resultado foi limitado (para mais ou para menos) pelo que a fonte oferece.
- **FR-007**: A escolha do nível de detalhe DEVE ser monotônica em relação à distância da câmera: para o mesmo trajeto e a mesma fonte, uma câmera mais próxima nunca resulta em nível menos detalhado do que uma câmera mais distante.
- **FR-008**: O sistema DEVE reportar toda elevação em metros. Quando a unidade declarada pelo arquivo de relevo for diferente de metros e puder ser convertida com segurança, a conversão DEVE ser feita e informada; quando não puder, o sistema DEVE recusar o uso do registro com mensagem clara.
- **FR-009**: Quando o arquivo de relevo não informar valor para um ponto (valor de "sem dado"), o sistema NÃO DEVE inventar um valor, NÃO DEVE tratá-lo como zero e NÃO DEVE falhar por causa disso: o ponto DEVE ser registrado como sem valor no recorte e contado no resumo.
- **FR-010**: Quando uma peça de mapa base necessária não existir dentro de uma fonte registrada, o sistema DEVE continuar o recorte, e DEVE informar quantas peças faltaram e a posição de cada uma (nível e coordenadas da peça), tanto no resumo quanto no recorte exportado.
- **FR-011**: O sistema DEVE recusar, com erro próprio e mensagem clara, um recorte cujo tamanho estimado exceda um limite máximo documentado, informando o tamanho estimado, o limite e a causa provável; a recusa DEVE ocorrer antes de a leitura do conteúdo começar. Um recorte exatamente no limite DEVE ser aceito.
- **FR-012**: O sistema DEVE apresentar um resumo do recorte com: a área geográfica coberta; o nível de detalhe escolhido e o motivo; a quantidade de peças de mapa base (e quantas faltaram, quando houver); a quantidade de amostras de elevação (e quantas sem valor, quando houver); as elevações mínima e máxima encontradas, em metros, calculadas apenas sobre as amostras com valor (ou a indicação explícita de que não há faixa a informar); os registros usados (nome e tipo); e o tamanho do recorte, em unidades identificadas.
- **FR-013**: O sistema DEVE permitir exportar o recorte completo — amostras de elevação (com a indicação das que não têm valor), peças de mapa base (com posição e nível), peças ausentes, resumo e procedência (registros usados) — para um destino consumível por outro programa sem acesso aos arquivos originais; a exportação DEVE ser determinística (mesma entrada, resultado idêntico byte a byte).
- **FR-014**: Por padrão, a exportação DEVE recusar, com mensagem clara, um destino que já exista, deixando-o intacto; o usuário DEVE poder pedir explicitamente a sobrescrita por meio de uma opção dedicada; em qualquer caso, a exportação NÃO DEVE deixar resultado parcial em caso de falha, e um destino anterior DEVE permanecer intacto se a nova exportação falhar.
- **FR-015**: O mesmo plano com os mesmos registros (mesmo conteúdo de arquivo) DEVE produzir sempre exatamente o mesmo recorte — as mesmas peças, as mesmas amostras, na mesma ordem e com os mesmos valores —, independentemente de momento ou ordem de execução.
- **FR-016**: O comportamento DEVE ser idêntico para qualquer trajeto em qualquer lugar do planeta, incluindo áreas que cruzam o meridiano de 180° e latitudes altas, sem qualquer tratamento especial ou privilegiado de região; a ferramenta NÃO DEVE embutir dados de nenhuma região.
- **FR-017**: O sistema DEVE permitir consultar a elevação de uma coordenada isolada (latitude e longitude), informando a elevação em metros e o registro de relevo usado, e DEVE distinguir três respostas: elevação encontrada, "o arquivo não informa valor para este ponto" e "nenhum relevo registrado cobre este ponto".
- **FR-018**: A consulta de elevação de uma coordenada DEVE usar a mesma regra de escolha de registro e a mesma forma de leitura da elevação que o recorte usa, de modo que o valor consultado coincida com o valor do recorte para o mesmo ponto; DEVE recusar latitude fora de -90 a 90, longitude fora de -180 a 180 e valores não numéricos, com mensagem que aponte o valor inválido e o intervalo aceito; e DEVE tratar as duas escritas da longitude do meridiano de 180° como a mesma posição.
- **FR-019**: O sistema DEVE recusar o uso de um registro cujo conteúdo esteja corrompido ou ilegível, com mensagem que identifique o registro, sem produzir um recorte parcial silencioso.
- **FR-020**: Todos os erros de negócio (plano ilegível, inválido, de versão desconhecida ou incoerente, área não coberta, recorte grande demais, registro ilegível, unidade de elevação não suportada, coordenada fora de cobertura, destino de exportação inválido ou já existente) DEVEM ser distinguíveis entre si e dos erros das etapas anteriores, comunicados com mensagem clara e código de saída próprio.
- **FR-021**: A ferramenta NÃO DEVE, nesta etapa, baixar ou converter dados, projetar em 3D, desenhar mapa ou terreno, texturizar, renderizar quadros, gerar vídeo, incluir sobreposições nem oferecer interface gráfica ou API.

### Entidades-Chave *(incluir se a funcionalidade envolver dados)*

- **Plano de Câmera Exportado**: o arquivo gerado pela etapa 3, versionado, que é a entrada desta etapa; traz os parâmetros usados, o resumo e todos os quadros, e é tratado como dado externo que precisa ser validado antes do uso.
- **Recorte de Dados Geográficos**: o resultado completo da etapa — a área coberta, o conjunto de amostras de elevação, o conjunto de peças de mapa base, a indicação de peças ausentes, a procedência e o resumo. É a entrada das etapas seguintes.
- **Área de Interesse**: a região da superfície da Terra que o voo precisa ver — a área percorrida pela câmera e pelo marcador, com margem — definida sem depender de nenhuma região específica e contínua mesmo ao cruzar o meridiano de 180°.
- **Amostra de Elevação**: um ponto da grade de terreno, com sua posição e sua elevação em metros — ou a indicação explícita de que o arquivo não informa valor para ele.
- **Peça de Mapa Base**: uma imagem quadrada de uma região do mapa, identificada por nível de detalhe e posição na grade daquele nível, lida de um registro de mapa base.
- **Peça Ausente**: a identificação (nível e posição) de uma peça que era necessária, mas que a fonte registrada não contém.
- **Nível de Detalhe**: o grau de detalhe das peças de mapa escolhido para o recorte, com o motivo da escolha e o intervalo que a fonte oferece.
- **Procedência**: a lista de registros (nome, tipo e caminho) dos quais o recorte foi extraído e o que cada um contribuiu.
- **Resumo do Recorte**: área, nível de detalhe e motivo, quantidades (peças, ausentes, amostras, sem valor), faixa de elevação, registros usados e tamanho.
- **Consulta de Elevação**: a resposta a uma coordenada isolada — elevação em metros e registro usado, ou "sem valor no arquivo", ou "sem cobertura".

## Critérios de Sucesso *(obrigatório)*

### Resultados Mensuráveis

- **SC-001**: Um usuário consegue, com um único comando, apenas informando o arquivo de plano exportado, obter o recorte de um voo típico (trajeto de até 50 km com os parâmetros padrão de plano) e o resumo dele em menos de 30 segundos, sobre arquivos registrados de tamanho típico de uso pessoal.
- **SC-002**: Gerar o mesmo recorte 100 vezes seguidas, com o mesmo plano e os mesmos registros, produz 100 recortes idênticos e 100 exportações idênticas byte a byte.
- **SC-003**: Em 100% dos recortes gerados nos casos de teste, todas as peças de mapa e amostras de elevação necessárias para a área do plano estão presentes, ou estão explicitamente marcadas como ausentes ou sem valor; nenhuma lacuna passa despercebida.
- **SC-004**: Em 100% dos planos testados com área não totalmente coberta (falta de mapa base, falta de relevo, falta de ambos, registro com arquivo ausente), a recusa lista todos os subtrechos não cobertos e o que falta em cada um, no mesmo formato e pelas mesmas regras (escolha de registro, arquivo ausente, tipo que falta) da verificação de cobertura da segunda etapa.
- **SC-005**: Em 100% dos casos de teste, o nível de detalhe escolhido nunca sai do intervalo oferecido pela fonte e nunca é menos detalhado para uma câmera mais próxima do que para uma mais distante, para o mesmo trajeto e a mesma fonte; e o resumo sempre informa o nível e o motivo.
- **SC-006**: Em 100% dos pontos de teste, a elevação consultada por coordenada coincide com a elevação do recorte para o mesmo ponto, e, em relação a uma fonte de referência conhecida (arquivo de teste com valores conhecidos), a diferença é de no máximo a própria resolução vertical do arquivo.
- **SC-007**: Em 100% dos testes com amostras sem valor (relevo com buracos), nenhum valor é inventado: a quantidade de amostras sem valor no resumo é exatamente a quantidade de pontos sem valor no arquivo dentro da área, e o recorte é concluído sem erro.
- **SC-008**: Em 100% dos testes com peças de mapa removidas de uma fonte, o recorte é concluído, e a quantidade e a posição das peças ausentes informadas coincidem exatamente com as peças removidas que eram necessárias.
- **SC-009**: Trajetos equivalentes deslocados para regiões diferentes do planeta (inclusive cruzando o meridiano de 180° e em latitudes acima de 80°) produzem recortes com as mesmas propriedades — mesmo nível de detalhe, mesma cobertura completa da área e sem descontinuidade na junção do meridiano —, e a quantidade de peças e de amostras difere no máximo pelo que a geometria da região exige, de forma explicável.
- **SC-010**: 100% dos casos de recusa testados (plano ilegível, inválido, de versão desconhecida ou incoerente, área não coberta, recorte grande demais, registro ilegível, unidade não suportada, coordenada sem cobertura, coordenada inválida, destino inválido ou já existente sem opção de sobrescrita) produzem uma mensagem que identifica o problema, e nenhum deixa resultado parcial; a recusa por recorte grande demais ocorre em menos de 5 segundos, sem ler o conteúdo.
- **SC-011**: Um usuário consegue, apenas lendo o resumo, responder em menos de 1 minuto: qual área o recorte cobre, qual nível de detalhe foi usado e por quê, quantas peças e amostras tem, em que faixa de elevação o terreno está, quais registros foram usados, quanto o recorte ocupa e se faltou alguma peça ou valor.
- **SC-012**: O recorte exportado é lido de volta sem perda: todas as peças, amostras, indicações de ausência e o resumo coincidem com os do recorte gerado.

## Suposições

- O usuário já registrou, na segunda etapa, os mapas base e os dados de relevo que cobrem o voo, nos formatos que aquela etapa reconhece; esta etapa NÃO acrescenta formatos novos, apenas lê o conteúdo dos que já são aceitos.
- O recorte é pedido a partir do arquivo de plano exportado pela terceira etapa (ver Clarificações), que é o contrato entre as etapas; o usuário exporta o plano, o inspeciona e pede o recorte daquele mesmo plano. Esta etapa NÃO aceita o trajeto GPS diretamente. Um comando futuro que encadeie todas as etapas sem arquivos intermediários é possível, mas fora do escopo desta etapa; por isso a leitura e a validação do arquivo de plano são feitas por um adapter de saída, invocado antes do recorte, e o cálculo do recorte trabalha sobre um plano já carregado, sem saber de onde ele veio.
- O formato do plano exportado é o definido na etapa 3 e é versionado; esta etapa reconhece a versão atual desse formato e recusa versões desconhecidas, sem tentar adivinhar o conteúdo.
- A "área que a câmera vai percorrer" é a região que contém todas as posições da câmera e do marcador do plano — inclusive na abertura e no fechamento, em que o trajeto inteiro está enquadrado — mais uma margem, proporcional à distância da câmera, para conter o terreno visível do ponto de vista dela. A margem exata é documentada no planejamento técnico.
- A escolha do nível de detalhe busca que a resolução das peças no terreno seja suficiente para a menor distância entre a câmera e o terreno durante o voo, sem ultrapassar o que a fonte oferece; a fórmula exata e as constantes são documentadas no planejamento técnico, mas seguem os princípios das Histórias 2 e FR-006/FR-007 (determinismo, monotonicidade e explicabilidade).
- A elevação de uma coordenada é lida da grade de terreno do registro de relevo escolhido, por um método de leitura determinístico e documentado no planejamento técnico; a mesma forma de leitura vale para o recorte e para a consulta isolada (FR-018).
- O limite de tamanho do recorte é documentado e definido no planejamento técnico; a referência inicial é 256 MiB de conteúdo por recorte, valor que pode ser revisto sem alterar o comportamento descrito aqui (a recusa clara, antes da leitura, com o tamanho estimado e o limite).
- O destino da exportação é uma unidade única e atômica (a natureza exata — arquivo único ou diretório — é decidida no planejamento técnico), com o formato versionado para permitir evolução sem quebrar as etapas seguintes.
- O recorte NÃO faz nenhum tratamento de imagem nem de terreno além de selecionar e ler: nada de reamostragem para outra grade, projeção, suavização ou preenchimento de buracos; as peças e amostras seguem a grade e o nível dos arquivos de origem, exceto pela escolha do nível.
- Esta etapa não altera o arquivo de plano nem o registro de dados geográficos, apenas os lê.
- Os comandos existentes das etapas anteriores (`inspect`, `geodata register|list|remove|check`, `plan`) continuam se comportando exatamente como hoje.
