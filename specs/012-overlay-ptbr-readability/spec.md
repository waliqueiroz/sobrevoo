# Especificação de Funcionalidade: Rótulos em Português e Legibilidade do Texto das Sobreposições

**Branch da Funcionalidade**: `012-overlay-ptbr-readability`

**Criado em**: 2026-10-03

**Status**: Rascunho

**Entrada**: Descrição do usuário: "Décima segunda etapa do Sobrevoo. Os rótulos da sobreposição e a legibilidade do texto precisam de um acerto. Hoje os blocos numéricos são escritos em inglês ('DIST', 'ELEV', 'GAIN', 'TIME') e o projeto é de uso pessoal em português, então todo texto que aparece desenhado no quadro passa a ser em português do Brasil — rótulos e qualquer outra palavra que o desenho escreva — mantendo as abreviações de unidade como já estão ('km', 'm') e sem mudar como os valores são formatados nem quais blocos existem. Junto com isso, três defeitos de legibilidade do desenho do texto, todos introduzidos na etapa anterior e visíveis num quadro real em 1080×1920: o contorno escuro está grosso demais, com raio de 5 px contra hastes de 5 a 6 px da própria letra, de modo que o halo tem a mesma espessura do traço, os contadores do '6', do '8' e do '0' quase se fecham e os halos de letras vizinhas se encostam — o contorno precisa ser fino o bastante para apenas destacar a letra do fundo, e sua espessura deve ser derivada do corpo da fonte e não da altura do quadro, já que é a letra que ele contorna; a fonte embutida é de peso regular, leve demais para texto sobre imagem de satélite clara, e passa a ser uma fonte de peso mais forte, ainda embutida no binário, de licença permissiva compatível com a do projeto e de preferência da mesma família já usada, para não acrescentar dependência nem abrir mão do rasterizador determinístico próprio; e a largura dos painéis numéricos é recalculada a cada quadro a partir dos textos daquele quadro, com dígitos de larguras diferentes, o que faz os painéis pulsarem de largura ao longo do vídeo — a largura precisa ser estável do primeiro ao último quadro de um mesmo sobrevoo. Como os pixels mudam sem o usuário ter pedido, a versão de desenho sobe de novo e quadros antigos nunca se juntam aos novos no mesmo conjunto; a garantia de que o mesmo plano, recorte, resolução, aparência e configuração de sobreposição produzem a mesma imagem byte a byte em qualquer máquina continua valendo. Fora de escopo: permitir que o usuário escolha a fonte, o peso, o tamanho, as cores ou a posição dos blocos; qualquer idioma configurável ou seleção de idioma por flag — o texto desenhado é em português e ponto; traduzir mensagens de erro, saída de terminal, ajuda dos comandos ou documentação, que seguem como estão; criar blocos novos, texto livre ou remover blocos existentes; mudar a formatação dos valores, as unidades ou o arredondamento; mudar as margens de segurança, o tamanho do marcador do perfil ou o painel do perfil de elevação, todos acertados na etapa anterior; e trocar o rasterizador ou o caminho de extração de contornos da fonte."

## Cenários de Usuário e Testes *(obrigatório)*

### História de Usuário 1 - Ler os rótulos das sobreposições em português (Prioridade: P1)

Como usuário do Sobrevoo — uma ferramenta pessoal, de uso em português do Brasil —, hoje vejo os rótulos dos blocos numéricos ("DIST", "ELEV", "GAIN", "TIME") escritos em inglês, destoando do resto da ferramenta. Quero que todo texto desenhado sobre o quadro — os rótulos de hoje e qualquer palavra que o desenho venha a escrever — esteja em português do Brasil, mantendo as abreviações de unidade ("km", "m") como já estão e sem mudar nenhum valor exibido, nenhuma formatação nem quais blocos existem.

**Por que esta prioridade**: é o pedido que abre a descrição da etapa e o que mais salta aos olhos em qualquer quadro — um rótulo em inglês numa ferramenta pessoal em português é a primeira coisa que soa errada; não depende de nenhuma outra história desta etapa.

**Teste Independente**: pode ser totalmente testado desenhando um quadro com os quatro blocos numéricos ligados e conferindo que nenhuma palavra em inglês aparece — só português e as abreviações de unidade já existentes — com os mesmos valores e a mesma formatação de antes.

**Cenários de Aceitação**:

1. **Dado** um plano e um recorte válidos, **Quando** o usuário desenha um quadro com os blocos de distância, elevação/ganho e tempo decorrido ligados, **Então** os rótulos desenhados estão em português do Brasil, sem nenhuma palavra em inglês.
2. **Dado** o mesmo quadro, **Quando** o usuário compara os valores e as unidades exibidas com as de antes desta etapa, **Então** são exatamente os mesmos números, na mesma formatação, com "km"/"m" como abreviação de unidade.
3. **Dado** a ferramenta rodando em qualquer máquina, **Quando** o usuário procura uma flag ou variável de ambiente para escolher o idioma do texto desenhado, **Então** não encontra nenhuma — o texto desenhado é sempre em português do Brasil, e mensagens de erro, saída de terminal, ajuda dos comandos e documentação continuam como estão.

---

### História de Usuário 2 - Ler o texto sem as letras se fecharem por trás do contorno (Prioridade: P2)

Como usuário do Sobrevoo, ao ver um quadro real em 1080×1920, notei que o contorno escuro ao redor do texto (acrescentado na etapa anterior) é grosso demais: tem a mesma espessura do traço das próprias letras, a ponto de os vãos internos do "6", do "8" e do "0" quase se fecharem, e o halo de uma letra encostar no da vizinha. Quero que o contorno seja fino o bastante só para destacar a letra do fundo, com a espessura derivada do tamanho em que a própria letra está sendo desenhada — não da altura do quadro —, para que a mesma espessura relativa valha em qualquer tamanho de texto.

**Por que esta prioridade**: dos três defeitos de legibilidade, é o mais grave — compromete a forma das próprias letras, não só o acabamento ao redor delas; não depende de nenhuma outra história, mas é o problema mais visível depois dos rótulos em inglês.

**Teste Independente**: pode ser totalmente testado desenhando um quadro com números que incluam "6", "8" e "0" e conferindo, a olho e por inspeção de pixel, que os vãos internos dessas letras continuam abertos e que o halo de uma letra não encosta no da vizinha, em diferentes resoluções.

**Cenários de Aceitação**:

1. **Dado** um quadro com os blocos numéricos ligados, cujo texto inclui os dígitos "6", "8" e "0", **Quando** o usuário inspeciona os pixels desses dígitos, **Então** o vão interno de cada um continua claramente aberto — o contorno não o fecha.
2. **Dado** o mesmo quadro, **Quando** o usuário mede a distância entre o contorno de duas letras vizinhas, **Então** os halos não se tocam.
3. **Dado** quadros em resoluções diferentes, **Quando** o usuário mede a espessura do contorno em relação ao tamanho da letra desenhada, **Então** a proporção entre os dois é a mesma em qualquer resolução — a espessura acompanha o tamanho da letra, não a altura do quadro.

---

### História de Usuário 3 - Ler o texto com contraste sobre uma imagem de satélite clara (Prioridade: P3)

Como usuário do Sobrevoo, ao ver um quadro sobre uma área clara de imagem de satélite, notei que o peso regular da fonte embutida é leve demais — o texto compete mal com o fundo, mesmo com o contorno. Quero que a fonte embutida tenha um peso mais forte, mantendo-se embutida no binário, da mesma família tipográfica já usada e com licença permissiva compatível com a do projeto, sem acrescentar nenhuma dependência nova nem abrir mão do rasterizador determinístico próprio.

**Por que esta prioridade**: é um problema de contraste, não de forma das letras — menos grave que a História 2, mas ainda visível em qualquer quadro sobre fundo claro; não depende de nenhuma outra história.

**Teste Independente**: pode ser totalmente testado desenhando quadros sobre fundos de satélite claros e escuros e comparando o peso visual do texto com o de antes desta etapa — mais encorpado, mais fácil de distinguir do fundo claro.

**Cenários de Aceitação**:

1. **Dado** um quadro desenhado sobre uma área clara de imagem de satélite, **Quando** o usuário compara o texto com o de antes desta etapa, **Então** os traços das letras são visivelmente mais grossos (peso mais forte), sem mudar o tamanho nem a posição do texto.
2. **Dado** a ferramenta compilada, **Quando** o usuário verifica suas dependências, **Então** nenhuma dependência nova foi acrescentada, e a fonte mais forte continua embutida no binário, nunca lida do sistema operacional.
3. **Dado** o texto desenhado com a fonte mais forte, **Quando** o usuário desenha o mesmo quadro em máquinas diferentes, **Então** o resultado continua idêntico, byte a byte — o rasterizador determinístico próprio da etapa anterior não muda.

---

### História de Usuário 4 - Ver os painéis numéricos com largura estável do início ao fim do voo (Prioridade: P4)

Como usuário do Sobrevoo, ao assistir ao vídeo inteiro, notei que os painéis dos blocos numéricos mudam de largura de quadro a quadro, porque a largura é recalculada a cada quadro a partir do texto daquele quadro — um efeito de "pulsar" incômodo. Quero que a largura compartilhada dos painéis numéricos seja a mesma do primeiro ao último quadro de um mesmo voo, calculada uma vez a partir de todo o voo, não a cada quadro.

**Por que esta prioridade**: é o defeito menos grave dos três — um incômodo visual ao longo do tempo, não um problema de legibilidade de um quadro isolado; não depende de nenhuma outra história.

**Teste Independente**: pode ser totalmente testado desenhando o voo inteiro (ou vários quadros espalhados ao longo dele) e medindo que a largura dos painéis numéricos é idêntica em todos os quadros, do primeiro ao último.

**Cenários de Aceitação**:

1. **Dado** um voo inteiro desenhado com os blocos numéricos ligados, **Quando** o usuário mede a largura desses painéis em quadros diferentes do mesmo voo, **Então** a largura é exatamente a mesma em todos eles, mesmo quando o texto de um quadro é mais curto que o de outro.
2. **Dado** o mesmo voo, **Quando** o usuário desenha um quadro isolado (`render frame`) e o mesmo quadro dentro do voo inteiro (`render all`), **Então** a largura do painel é idêntica nos dois casos.
3. **Dado** um voo em que a distância e o tempo decorrido só crescem, mas a elevação varia (podendo ter menos dígitos no último quadro que num quadro intermediário), **Quando** o usuário mede a largura do painel em qualquer quadro, **Então** ela reflete o texto mais largo de todo o voo, não apenas o do quadro atual.

---

### Casos Extremos

- O que acontece quando as sobreposições estão desligadas por inteiro (`--overlays=false`)? Nada muda — nenhum texto é desenhado, exatamente como hoje.
- O que acontece com um conjunto de quadros (`render all` interrompido, ou intermediários guardados por `fly --keep`) desenhado por uma versão da ferramenta anterior a esta etapa? É tratado como um conjunto diferente do que esta versão produziria — a mesma regra que já recusa reaproveitar quadros de outro plano, recorte, resolução, aparência ou configuração de sobreposição — e a ferramenta recusa reaproveitá-los sem a sobrescrita pedida explicitamente.
- O que acontece quando só um dos três blocos numéricos está presente num quadro (os outros desligados ou sem dado disponível)? A largura ainda é estável ao longo do voo para os blocos presentes, calculada a partir do texto mais largo que esse bloco (ou os blocos presentes) tem em qualquer quadro do voo — não só no quadro atual.
- O que acontece com o quadro isolado (`render frame`) de um voo cujo texto mais largo aparece num quadro diferente do pedido? A largura do painel do quadro isolado é a mesma que seria usada se o voo inteiro fosse desenhado — calculada a partir de todo o plano, não só do quadro pedido.
- O que acontece com os blocos que não são numéricos (o perfil de elevação) e com o restante do quadro (terreno, traçado, marcador)? Nada muda — só o texto dos rótulos, o contorno, o peso da fonte e a estabilidade da largura dos três painéis numéricos são afetados.

## Requisitos *(obrigatório)*

### Requisitos Funcionais

- **FR-001**: Todo texto desenhado pelas sobreposições de tela — os rótulos dos blocos numéricos (hoje "DIST", "ELEV", "GAIN", "TIME") e qualquer outra palavra que o desenho vier a escrever — DEVE estar em português do Brasil; as abreviações de unidade ("km", "m") permanecem como estão.
- **FR-002**: Os valores exibidos, a formatação de cada um (casas decimais, troca de unidade, arredondamento) e quais blocos existem NÃO DEVEM mudar nesta etapa.
- **FR-003**: O idioma do texto desenhado NÃO É configurável pelo usuário — nenhuma flag, variável de ambiente ou arquivo de configuração nova para escolhê-lo; é sempre português do Brasil.
- **FR-004**: Mensagens de erro, saída de terminal, texto de ajuda dos comandos e a documentação do projeto NÃO são traduzidos por esta etapa — permanecem como estão.
- **FR-005**: O contorno escuro do texto DEVE ter espessura derivada do tamanho em que a letra está sendo desenhada (o corpo da fonte), não da altura do quadro, de modo que, em qualquer resolução, o contorno seja fino o bastante para só destacar a letra do fundo, sem fechar o vão interno de letras como "6", "8" e "0" nem encostar no halo de uma letra vizinha.
- **FR-006**: A fonte embutida usada para o texto das sobreposições DEVE ter peso mais forte que o peso regular usado até a etapa anterior — permanecendo embutida no binário, da mesma família tipográfica já usada, com licença permissiva compatível com a do projeto, sem nenhuma dependência nova.
- **FR-007**: A largura compartilhada dos painéis dos blocos numéricos (distância; elevação e ganho; tempo decorrido) DEVE ser a mesma do primeiro ao último quadro de um mesmo voo — calculada a partir do texto mais largo que cada bloco tem em qualquer quadro do voo, nunca recalculada a partir do texto de um quadro específico durante o desenho.
- **FR-008**: A largura do painel de um quadro isolado (`render frame`) DEVE ser idêntica à que o mesmo quadro teria dentro do voo inteiro (`render all`), mesmo quando o texto mais largo do voo aparece num quadro diferente do pedido.
- **FR-009**: Nesta etapa, nenhum valor exibido, bloco existente, configuração de sobreposição, enquadramento de câmera, terreno, traçado, margem de segurança, tamanho do marcador do perfil de elevação ou painel do perfil de elevação muda — o efeito desta etapa se limita ao idioma do texto, ao contorno, ao peso da fonte e à estabilidade da largura dos painéis numéricos.
- **FR-010**: Como os pixels de um quadro com sobreposição ligada mudam (rótulos, contorno, peso da fonte, largura do painel), a versão do desenho DEVE subir, de modo que um conjunto de quadros desenhado por uma versão anterior da ferramenta nunca seja considerado o mesmo conjunto que o produzido por esta versão — nem ao retomar um `render all` interrompido, nem ao reaproveitar os intermediários guardados por `fly --keep`.
- **FR-011**: O mesmo plano de câmera, o mesmo recorte de dados geográficos, a mesma resolução, a mesma aparência e a mesma configuração de sobreposição DEVEM continuar produzindo, nesta versão, imagens idênticas byte a byte, em qualquer máquina.
- **FR-012**: O rasterizador de texto e o caminho de extração dos contornos do glifo (a fonte vetorial lida pelo mesmo leitor de contornos, rasterizada pelo mesmo rasterizador próprio e determinístico da etapa anterior) NÃO mudam nesta etapa — só o idioma do texto, a espessura do contorno, o peso da fonte e o momento em que a largura do painel é calculada.
- **FR-013**: Esta etapa NÃO DEVE introduzir nenhuma escolha nova para o usuário — fonte, peso, tamanho, cores e posição dos blocos continuam fora do seu controle, exatamente como hoje.

### Entidades-Chave

- **Rótulo do Bloco**: a palavra fixa que identifica cada bloco numérico no desenho (hoje "DIST", "ELEV", "GAIN", "TIME"); passa a ser uma palavra em português do Brasil, sem nenhuma mudança no valor ou na unidade que acompanha.
- **Contorno do Texto**: o halo escuro ao redor de cada letra; sua espessura passa a ser proporcional ao tamanho da letra (o corpo da fonte), não à altura do quadro.
- **Peso da Fonte**: a variante da fonte vetorial embutida usada para rasterizar o texto; passa de peso regular para um peso mais forte, da mesma família.
- **Largura Compartilhada dos Painéis Numéricos (estável por voo)**: a largura única aplicada aos painéis dos blocos de distância, elevação+ganho e tempo decorrido; passa a ser calculada uma vez por voo, a partir do texto mais largo que cada bloco tem em qualquer quadro do plano, em vez de recalculada a cada quadro.
- **Versão do Desenho**: o identificador que distingue conjuntos de quadros produzidos por versões diferentes do desenho da ferramenta; sobe nesta etapa porque os pixels de um quadro com sobreposição mudam, mesmo sem nenhuma mudança pedida pelo usuário ao plano, ao recorte, à aparência ou à configuração de sobreposição.

## Critérios de Sucesso *(obrigatório)*

### Resultados Mensuráveis

- **SC-001**: Um usuário lendo os rótulos das sobreposições em qualquer quadro os reconhece como português do Brasil, sem nenhuma palavra em inglês.
- **SC-002**: O contorno do texto nunca fecha o vão interno de letras como "6", "8" e "0" nem encosta no halo de uma letra vizinha, em nenhuma resolução que a ferramenta aceita.
- **SC-003**: O texto permanece claramente legível sobre a imagem de satélite mais clara que a ferramenta desenha, sem depender de aumentar a opacidade do painel.
- **SC-004**: A largura de um painel numérico nunca muda entre dois quadros de um mesmo voo — idêntica do primeiro ao último quadro, incluindo entre um quadro isolado e o mesmo quadro dentro do voo inteiro.
- **SC-005**: Nenhum conjunto de quadros produzido por uma versão da ferramenta anterior a esta etapa é reaproveitado, por retomada ou por `fly --keep`, sem que o usuário peça a sobrescrita explicitamente.
- **SC-006**: O mesmo plano, recorte, resolução, aparência e configuração de sobreposição produzem, nesta versão, sempre a mesma imagem, byte a byte, em qualquer máquina.
- **SC-007**: Um usuário que já tinha um vídeo satisfatório quanto a valores, blocos, enquadramento, margens e tamanho do marcador do perfil não precisa reconfigurar nada para obter a legibilidade desta etapa — basta gerar o vídeo de novo.

## Suposições

- As quatro palavras em português que substituem os rótulos de hoje são "DIST" (distância), "ELEV" (elevação — já lê bem em português, sem mudança de forma), "GANHO" (no lugar de "GAIN") e "TEMPO" (no lugar de "TIME") — a leitura mais direta do pedido, mantendo o estilo enxuto (maiúsculas, sem acento, curto) dos rótulos de hoje; a escolha exata de cada palavra, mantendo esse estilo, é uma decisão de planejamento técnico.
- A fonte de peso mais forte é a variante "negrito" da mesma família tipográfica já embutida (ver `research.md` da etapa anterior) — a família continua única e fixa, escolhida pela ferramenta; a etapa não introduz uma segunda família nem uma escolha de peso pelo usuário.
- A espessura exata do contorno (a fração do corpo da fonte que ele usa) e a técnica de calcular, uma vez por voo, a largura mais larga de cada painel numérico ao longo de todos os quadros do plano são decisões de planejamento técnico; o pedido exige apenas que o resultado seja estável (FR-007/FR-008) e proporcional ao tamanho da letra (FR-005), não um número exato.
- Nenhum campo novo é acrescentado ao plano de câmera exportado nem ao recorte: os rótulos, o contorno, o peso da fonte e a largura do painel são inteiramente decisões de desenho, sem precisar de nenhum dado que o plano ou o recorte ainda não tenham.
- Como a etapa anterior, a mudança de versão do desenho é o único mecanismo necessário para impedir a mistura de quadros de antes e depois desta etapa.
