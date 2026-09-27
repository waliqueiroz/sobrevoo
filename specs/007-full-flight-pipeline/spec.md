# Especificação de Funcionalidade: Voo em um Único Comando

**Branch da Funcionalidade**: `007-full-flight-pipeline`

**Criado em**: 2026-09-27

**Status**: Implementada, com `quickstart.md` validado por inteiro com o `ffmpeg` real

**Entrada**: Descrição do usuário: "Sétima etapa do Sobrevoo. Esta etapa não acrescenta nenhuma regra nova de trajeto, câmera, dados, desenho ou vídeo: ela junta as seis etapas existentes num único comando, para que o usuário saia de um arquivo GPS e chegue a um vídeo sem precisar encadear comandos nem administrar arquivos intermediários. Hoje é preciso tratar o trajeto, planejar a câmera, exportar o plano, recortar os dados geográficos, desenhar os quadros num diretório e montar o vídeo — seis comandos e três arquivos intermediários. Com esta etapa, o usuário informa o arquivo de trajeto e o destino do vídeo, e a ferramenta faz o percurso inteiro, usando os dados geográficos que ele já registrou. O usuário consegue: gerar o vídeo de um trajeto num único comando; ajustar o que já era ajustável em cada etapa (duração, taxa de quadros, proporção, distância e inclinação da câmera, resolução e qualidade do vídeo), com os mesmos nomes, os mesmos valores aceitos e os mesmos padrões dos comandos existentes; acompanhar o progresso sabendo em que etapa está e quanto falta; e pedir que os arquivos intermediários sejam guardados num diretório de sua escolha, para inspecionar o plano, o recorte ou os quadros depois. Requisitos: o resultado DEVE ser idêntico ao de rodar os comandos existentes na mão com os mesmos valores — mesmo plano, mesmo recorte, mesmos quadros, mesmo vídeo; nenhuma decisão de negócio nova pode nascer aqui, e nenhuma regra existente pode ser duplicada ou contornada; o que puder ser recusado cedo DEVE ser recusado cedo, antes do trabalho caro — a cobertura dos dados registrados e a presença do codificador de vídeo são verificadas assim que houver informação para isso, e não depois de horas desenhando quadros; qualquer falha DEVE ser comunicada com o mesmo erro, a mesma mensagem e o mesmo código de saída que o comando correspondente daria, dizendo em qual etapa o percurso parou; quando os intermediários não são guardados, eles ficam num local temporário próprio e são apagados ao final, inclusive em caso de falha ou interrupção, sem nunca deixar resto no diretório do usuário; quando são guardados, uma nova execução com o mesmo trajeto e os mesmos valores DEVE reaproveitar o que já está pronto e válido, em vez de refazer, com a mesma regra de conjunto que a etapa de desenho já usa; uma interrupção encerra de forma ordenada, diz o que já havia sido feito e sai com código próprio, sem deixar vídeo parcial; e o destino do vídeo segue a mesma regra das demais saídas: recusa por padrão um arquivo existente, com opção explícita de sobrescrita. Fora de escopo: qualquer regra nova de tratamento de trajeto, de câmera, de leitura de dados, de desenho ou de codificação; sobreposições de texto ou estatísticas; áudio; baixar, converter ou registrar dados geográficos; desenho de peças vetoriais; interface gráfica e API."

## Cenários de Usuário e Testes *(obrigatório)*

### História de Usuário 1 - Do trajeto ao vídeo num único comando (Prioridade: P1)

Como usuário do Sobrevoo, eu já tenho um arquivo de trajeto GPS e já registrei os dados geográficos (mapa base e relevo) que cobrem a região por onde passei. Quero informar apenas o arquivo de trajeto e o destino do vídeo, e receber o vídeo pronto, sem precisar rodar os seis comandos existentes na mão nem administrar os arquivos que ficam entre eles.

**Por que esta prioridade**: é o valor central desta etapa — hoje o usuário precisa conhecer e encadear seis comandos e cuidar de três arquivos intermediários; esta história sozinha já entrega o produto final do Sobrevoo (um vídeo) a partir só do que o usuário realmente tem (um trajeto) e do que já preparou antes (os dados registrados).

**Teste Independente**: pode ser totalmente testado rodando o comando único com um trajeto de exemplo e dados registrados que o cobrem, e comparando o vídeo produzido, quadro a quadro e byte a byte, com o vídeo obtido rodando os seis comandos existentes na mão com os mesmos valores.

**Cenários de Aceitação**:

1. **Dado** um arquivo de trajeto válido e dados geográficos registrados que cobrem toda a rota, **Quando** o usuário roda o comando único informando o trajeto e o destino do vídeo, **Então** a ferramenta produz um único arquivo de vídeo nesse destino, sem exigir nenhum outro comando nem nenhum arquivo além do trajeto.
2. **Dado** os mesmos trajeto e valores, **Quando** o usuário compara o vídeo do comando único com o vídeo obtido rodando `plan` → `geodata slice` → `render all` → `video` na mão, **Então** os dois arquivos são idênticos byte a byte.
3. **Dado** que o comando único terminou com sucesso, **Quando** o usuário olha o diretório onde rodou o comando, **Então** só encontra o arquivo de vídeo pedido — nenhum arquivo de plano, de recorte ou de quadro ficou para trás.
4. **Dado** que o comando único terminou com sucesso, **Quando** o usuário confere o resumo mostrado, **Então** ele reconhece as mesmas informações que já via nos resumos das etapas individuais (quantidade de quadros, duração, resolução, taxa de quadros, qualidade, tamanho do arquivo).

---

### História de Usuário 2 - Ajustar os mesmos parâmetros de sempre (Prioridade: P2)

Como usuário do Sobrevoo, eu já sei como ajustar a duração, a taxa de quadros, a proporção do vídeo, a distância e a inclinação da câmera, a resolução dos quadros e a qualidade do vídeo, porque já uso os comandos individuais. Quero ajustar exatamente as mesmas coisas, com os mesmos nomes e os mesmos valores aceitos, ao rodar o comando único.

**Por que esta prioridade**: sem isso, o comando único só serviria para o caso padrão, obrigando quem precisa de um ajuste a voltar para os seis comandos — o que anularia o ganho da primeira história para qualquer uso além do mais simples.

**Teste Independente**: pode ser totalmente testado rodando o comando único com cada parâmetro ajustável em um valor não padrão, e conferindo que o resultado é idêntico ao de ajustar esse mesmo parâmetro no comando individual correspondente (mesmo valor, mesmo efeito, mesma validação, mesma recusa quando o valor é inválido).

**Cenários de Aceitação**:

1. **Dado** que o usuário informa a duração, a taxa de quadros, a proporção do vídeo, a distância da câmera ou a inclinação da câmera, **Quando** o comando único roda, **Então** o plano de câmera gerado é idêntico ao que `plan` geraria com o mesmo valor.
2. **Dado** que o usuário informa a resolução, **Quando** o comando único roda, **Então** os quadros desenhados são idênticos aos que `render all` desenharia com a mesma resolução.
3. **Dado** que o usuário informa a qualidade do vídeo, **Quando** o comando único roda, **Então** o vídeo é idêntico ao que `video` produziria com a mesma qualidade.
4. **Dado** que o usuário não informa um parâmetro ajustável, **Quando** o comando único roda, **Então** é usado exatamente o mesmo valor padrão que o comando individual correspondente usaria.
5. **Dado** que o usuário informa um valor inválido para qualquer parâmetro ajustável, **Quando** o comando único roda, **Então** a recusa acontece com a mesma mensagem e o mesmo código de saída que o comando individual correspondente daria para esse mesmo valor.

---

### História de Usuário 3 - Acompanhar o progresso por etapa (Prioridade: P3)

Como usuário do Sobrevoo, sei que gerar um vídeo de voo pode levar bastante tempo, principalmente para desenhar os quadros e codificar o vídeo. Quero, enquanto o comando único roda, saber em qual das etapas internas ele está e quanto falta, do mesmo jeito que os comandos individuais já mostram progresso.

**Por que esta prioridade**: sem isso, o comando único vira uma caixa-preta que pode levar minutos ou horas sem dar sinal de vida — o que é aceitável quando o usuário está olhando o progresso do `render all` ou do `video` isoladamente, mas não quando ele não sabe nem em que etapa o processo está.

**Teste Independente**: pode ser totalmente testado rodando o comando único com um trajeto grande o bastante para passar um tempo perceptível em mais de uma etapa, e observando que a saída identifica cada etapa em que o processo entra e mostra o progresso já existente dela.

**Cenários de Aceitação**:

1. **Dado** que o comando único está rodando, **Quando** ele entra em cada uma das etapas internas (tratamento do trajeto, planejamento da câmera, recorte dos dados, desenho dos quadros, montagem do vídeo), **Então** a saída identifica claramente em qual etapa o processo está.
2. **Dado** que o comando único está na etapa de desenho dos quadros ou na de montagem do vídeo (as mais demoradas), **Quando** o usuário observa a saída, **Então** vê o mesmo progresso (quadro atual, total, percentual, tempo decorrido) que `render all` ou `video` mostrariam rodando sozinhos.
3. **Dado** que uma etapa termina, **Quando** a próxima começa, **Então** fica claro, pela saída, que uma etapa terminou e outra começou, sem confundir o progresso de uma com o da outra.

---

### História de Usuário 4 - Guardar e reaproveitar os arquivos intermediários (Prioridade: P4)

Como usuário do Sobrevoo, às vezes quero conferir o plano gerado, o recorte de dados ou os quadros desenhados antes de aceitar o vídeo final, ou quero gerar o vídeo de novo com outra qualidade ou resolução sem esperar todo o processo recomeçar do zero. Quero poder pedir que os arquivos intermediários fiquem guardados num diretório de minha escolha, e que uma nova execução com o mesmo trajeto e os mesmos valores reaproveite o que já está pronto.

**Por que esta prioridade**: é o ajuste fino de um fluxo que, por padrão, esconde os intermediários — importante para inspeção e para iteração rápida, mas não bloqueia o valor central das três primeiras histórias, que já funcionam plenamente sem guardar nada.

**Teste Independente**: pode ser totalmente testado pedindo os intermediários guardados num diretório, rodando o comando único duas vezes com o mesmo trajeto e os mesmos valores, e conferindo que a segunda execução é sensivelmente mais rápida que a primeira e produz o mesmo vídeo, além de inspecionar o plano, o recorte e os quadros deixados no diretório.

**Cenários de Aceitação**:

1. **Dado** que o usuário pede os intermediários guardados num diretório de sua escolha, **Quando** o comando único termina com sucesso, **Então** o plano, o recorte e os quadros ficam nesse diretório, nos mesmos formatos que os comandos individuais já produzem, prontos para inspeção.
2. **Dado** um diretório de intermediários já preenchido por uma execução anterior com o mesmo trajeto e os mesmos valores, **Quando** o usuário roda o comando único de novo apontando para o mesmo diretório, **Então** a ferramenta reaproveita o plano, o recorte e os quadros já prontos em vez de refazê-los, e o vídeo final ainda assim é idêntico ao que sairia de uma execução do zero.
3. **Dado** um diretório de intermediários de uma execução anterior, **Quando** o usuário roda de novo mudando só um parâmetro que afeta uma etapa posterior (por exemplo, a qualidade do vídeo, que só afeta a montagem), **Então** apenas as etapas cujo resultado depende do parâmetro alterado são refeitas; as anteriores, cujo resultado não muda, são reaproveitadas.
4. **Dado** um diretório de intermediários que contém arquivos de outro trajeto ou de outros valores de ajuste, **Quando** o usuário roda o comando único apontando para esse diretório sem pedir a sobrescrita, **Então** a ferramenta recusa com uma mensagem que diz que o conteúdo é de outro conjunto, sem apagar nem alterar nada; com a sobrescrita pedida explicitamente, refaz o que for necessário a partir dali.
5. **Dado** que o usuário não pede os intermediários guardados, **Quando** o comando único termina, com sucesso ou não, **Então** nenhum arquivo de plano, recorte ou quadro fica em nenhum diretório do usuário.

---

### Casos Extremos

- O que acontece quando o trajeto informado não é coberto pelos dados geográficos já registrados? A ferramenta recusa assim que o plano de câmera existe e a área necessária é conhecida — bem antes de desenhar qualquer quadro —, com o mesmo erro, a mesma mensagem e o mesmo código de saída que `geodata slice` já daria para essa falta de cobertura.
- O que acontece quando o programa `ffmpeg` não está instalado ou não tem o codificador necessário? A ferramenta recusa antes de começar qualquer etapa, com o mesmo erro, a mesma mensagem e o mesmo código de saída que `video` já daria, dizendo o que instalar.
- O que acontece quando o destino do vídeo já existe e a sobrescrita não foi pedida? A ferramenta recusa antes de começar qualquer etapa, sem tratar nenhum trajeto nem consultar dados geográficos.
- O que acontece quando a execução é interrompida enquanto os quadros estão sendo desenhados, com os intermediários guardados? Os quadros já completos ficam prontos para reaproveitamento numa execução futura, exatamente como `render all` já se comporta hoje quando interrompido.
- O que acontece quando a execução é interrompida durante o recorte dos dados geográficos ou durante a montagem do vídeo? Nenhum recorte nem vídeo parcial fica, exatamente como `geodata slice` e `video` já se comportam hoje quando interrompidos nessas etapas.
- O que acontece quando a execução é interrompida durante o tratamento do trajeto ou o planejamento da câmera, etapas rápidas e sem publicação incremental? Nada fica gravado dessas etapas; a próxima execução as refaz do zero, o que tem custo desprezível perto das demais.
- O que acontece quando o diretório de intermediários contém arquivos que a ferramenta não reconhece como seus (por exemplo, anotações do usuário)? Esses arquivos são ignorados e preservados, exatamente como os comandos individuais já tratam arquivos alheios hoje.
- O que acontece quando o trajeto informado é inválido, vazio ou tem pontos insuficientes? A ferramenta recusa com o mesmo erro, a mesma mensagem e o mesmo código de saída que `inspect`/`plan` já dariam, antes de qualquer outra etapa.

## Requisitos *(obrigatório)*

### Requisitos Funcionais

- **FR-001**: O sistema DEVE oferecer um único comando que, a partir de um arquivo de trajeto GPS e de um destino de vídeo, executa o percurso completo — tratamento do trajeto, planejamento da câmera, recorte dos dados geográficos já registrados, desenho dos quadros e montagem do vídeo —, sem exigir que o usuário rode nenhum outro comando nem administre nenhum arquivo intermediário.
- **FR-002**: O resultado do comando único DEVE ser idêntico, etapa a etapa, ao de rodar os seis comandos existentes na mão com os mesmos valores: mesmo plano, mesmo recorte, mesmos quadros e mesmo vídeo (byte a byte, para o mesmo codificador). O comando único NÃO DEVE introduzir nenhuma decisão de negócio nova, nem duplicar, nem contornar nenhuma regra já existente em qualquer etapa — cada etapa interna delega inteiramente para o serviço que já a implementa.
- **FR-003**: O usuário DEVE poder ajustar, no comando único, tudo o que já era ajustável nas etapas individuais — duração do vídeo, taxa de quadros, proporção do vídeo, distância e inclinação da câmera, resolução dos quadros e qualidade do vídeo —, com os mesmos nomes, os mesmos valores aceitos, as mesmas validações e os mesmos padrões que os comandos individuais já usam; um parâmetro não informado usa exatamente o mesmo padrão que o comando individual correspondente usaria.
- **FR-004**: O usuário DEVE ser informado, durante a execução, em qual das cinco etapas internas o processo está; nas etapas de desenho dos quadros e de montagem do vídeo — as mais demoradas —, DEVE ver o mesmo progresso (quadro atual, total, percentual e tempo decorrido) que o comando individual dessa etapa já mostra.
- **FR-005**: A disponibilidade do codificador de vídeo necessário DEVE ser verificada antes de qualquer etapa começar, e a recusa, quando ele falta, DEVE usar o mesmo erro, a mesma mensagem e o mesmo código de saída que o comando de montagem de vídeo já dá.
- **FR-006**: A cobertura da rota pelos dados geográficos já registrados DEVE ser verificada assim que o plano de câmera existir — antes do recorte propriamente dito e muito antes do desenho de qualquer quadro —, e a recusa, quando a rota não é coberta, DEVE usar o mesmo erro, a mesma mensagem e o mesmo código de saída que o comando de recorte de dados já dá.
- **FR-007**: O destino do vídeo DEVE ser conferido — existência e possibilidade de escrita — antes de qualquer etapa começar, junto com a verificação do codificador (FR-005); por padrão, um destino já existente DEVE ser recusado, com uma opção explícita para sobrescrever.
- **FR-008**: Qualquer falha, em qualquer etapa interna, DEVE ser comunicada com o mesmo erro sentinela, a mesma mensagem e o mesmo código de saída que o comando individual dessa etapa daria para a mesma falha, e DEVE adicionalmente dizer em qual etapa do percurso a falha ocorreu.
- **FR-009**: Quando o usuário não pede que os arquivos intermediários (plano, recorte, quadros) sejam guardados, eles DEVEM ficar num local temporário de uso exclusivo da execução, nunca no diretório do usuário, e DEVEM ser apagados ao final — inclusive quando a execução falha ou é interrompida —, sem deixar nenhum resto.
- **FR-010**: Quando o usuário pede que os arquivos intermediários sejam guardados num diretório de sua escolha, o plano, o recorte e os quadros DEVEM ser gravados nesse diretório, nos mesmos formatos que os comandos individuais já usam, disponíveis para inspeção depois da execução.
- **FR-010a**: Numa nova execução apontando para um diretório de intermediários já preenchido, cada um dos três intermediários (plano, recorte, quadros) DEVE ser reaproveitado, em vez de refeito, sempre que ele já existir e continuar válido para o trajeto e os valores informados agora — usando a mesma regra de conjunto que o desenho dos quadros já usa (identificação combinada do que cada intermediário depende); mudar um valor que só afeta uma etapa posterior DEVE refazer só as etapas a partir dali, reaproveitando as anteriores.
- **FR-010b**: Quando o diretório de intermediários contém, para algum dos três arquivos, conteúdo reconhecido pela ferramenta mas que não corresponde ao trajeto e aos valores da execução atual (outro conjunto), a execução DEVE recusar por padrão, sem apagar nem alterar nada, com uma opção explícita de sobrescrita que refaz o que for necessário a partir dali; conteúdo não reconhecido como sendo da ferramenta DEVE ser ignorado e preservado.
- **FR-011**: Uma interrupção (pedida pelo usuário ou pelo sistema) DEVE encerrar o comando único de forma ordenada, informar quais etapas já haviam sido concluídas (e, quando os intermediários são guardados, quais dos seus arquivos continuam disponíveis para uma execução futura reaproveitar) e sair com um código de saída próprio da interrupção do comando único; em nenhum caso um arquivo de vídeo parcial é deixado, e nenhuma etapa deixa, por causa da interrupção, um resultado parcial que a própria etapa não permitiria deixar quando rodada sozinha.
- **FR-012**: O comportamento do comando único DEVE ser idêntico para qualquer trajeto, em qualquer lugar do planeta, nas mesmas condições em que os comandos individuais já garantem isso.

### Entidades-Chave

- **Execução do Comando Único**: representa uma chamada do comando; agrega os valores informados pelo usuário (trajeto, destino do vídeo e todos os parâmetros ajustáveis), a etapa interna em curso e, ao final, um resumo com as informações que os resumos das etapas individuais já trazem.
- **Etapa Interna**: uma das cinco fases delegadas às etapas já existentes (tratamento do trajeto, planejamento da câmera, recorte dos dados geográficos, desenho dos quadros, montagem do vídeo); cada uma carrega seu próprio progresso e seus próprios erros, sem regra de negócio própria do comando único.
- **Diretório de Intermediários**: local, temporário ou escolhido pelo usuário, onde vivem o plano, o recorte e os quadros gerados durante a execução; quando escolhido pelo usuário, sobrevive ao final da execução e pode ser reaproveitado por uma execução futura.
- **Identificação de Reaproveitamento**: a combinação do trajeto e dos valores de ajuste dos quais cada um dos três intermediários depende, usada para decidir se um arquivo já presente no diretório de intermediários ainda é válido para a execução atual ou pertence a outro conjunto — extensão da mesma identificação que o desenho dos quadros já grava e confere.

## Critérios de Sucesso *(obrigatório)*

### Resultados Mensuráveis

- **SC-001**: Um usuário com os dados geográficos já registrados consegue ir de um arquivo de trajeto GPS a um vídeo pronto digitando um único comando, sem rodar nenhum outro comando e sem lidar com nenhum arquivo intermediário.
- **SC-002**: O vídeo produzido pelo comando único é idêntico, byte a byte, ao vídeo produzido rodando os seis comandos existentes na mão com os mesmos valores.
- **SC-003**: Sempre que o codificador de vídeo está ausente ou a rota não é coberta pelos dados registrados, a execução para antes de desenhar um único quadro.
- **SC-004**: Repetir o mesmo comando com o mesmo trajeto e os mesmos valores, com os intermediários guardados, termina numa fração pequena do tempo da primeira execução, porque só a etapa afetada por qualquer valor alterado (na ausência de alteração, só a montagem final) é refeita.
- **SC-005**: Toda falha, em qualquer etapa, é comunicada com uma mensagem que diz o que deu errado e em qual etapa, e com o mesmo código de saída que o comando individual dessa etapa já usa para a mesma falha.
- **SC-006**: Interromper o comando único, em qualquer ponto da execução, nunca deixa um arquivo de vídeo parcial, e nunca deixa arquivo de plano, recorte ou quadro fora de um diretório que o usuário pediu explicitamente para guardar.
- **SC-007**: Cada parâmetro que já era ajustável num comando individual produz, quando ajustado no comando único, exatamente o mesmo efeito que produziria no comando individual correspondente.

## Suposições

- A identificação usada para decidir se um intermediário guardado ainda é válido (FR-010a) é baseada no conteúdo do trajeto e nos valores de ajuste — não no caminho do arquivo nem em datas de modificação —, do mesmo jeito que a identificação que o desenho dos quadros já grava dentro de cada imagem.
- A opção de sobrescrita do comando único é uma só, e vale tanto para substituir intermediários de outro conjunto (FR-010b) quanto para substituir um destino de vídeo já existente (FR-007) — mantendo a mesma simplicidade de um único comando.
- O diretório de intermediários, quando pedido, guarda os três arquivos (plano, recorte, quadros) juntos, cada um no formato que a etapa correspondente já define; não há como pedir só um deles.
- Mudanças no registro de dados geográficos entre duas execuções (por exemplo, o usuário registrar ou remover uma fonte) não são verificadas além do que os próprios formatos do plano e do recorte já verificam internamente — o mesmo que já vale hoje ao rodar `geodata slice` de novo depois de alterar o registro; esta etapa não muda essa regra.
- Como nas demais etapas, o comando roda inteiramente local, sem interface gráfica nem API.
