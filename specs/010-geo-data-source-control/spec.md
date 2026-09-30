# Especificação de Funcionalidade: Controle do Registro de Dados Geográficos

**Branch da Funcionalidade**: `010-geo-data-source-control`

**Criado em**: 2026-09-29

**Status**: Implementada, com `quickstart.md` validado por inteiro com o `ffmpeg` real

**Entrada**: Descrição do usuário: "Décima etapa do Sobrevoo. Hoje o registro local de dados geográficos (`geodata register|list|remove|check`) só permite remover uma fonte de cada vez, e nenhum comando que usa dados geográficos (`plan`, `geodata slice`, e por consequência `fly`) permite escolher qual fonte registrada usar — a escolha é sempre automática: para cada ponto do trajeto, vence a fonte de menor área que cobre aquele ponto, com empate resolvido pelo registro mais antigo. Isso causa dois problemas na prática: primeiro, quem registra várias fontes ao longo do tempo (testando mapas diferentes, atualizando um recorte) perde o controle de quais tem registrado, e limpar tudo para recomeçar do zero exige remover uma por uma; segundo, quando duas fontes se sobrepõem, a seleção automática pode escolher uma fonte diferente da que o usuário pretendia, sem nenhum aviso, e até misturar fontes diferentes no mesmo recorte, ponto a ponto. Esta etapa acrescenta duas capacidades ao registro: um comando que limpa o registro inteiro de uma vez — todas as entradas, mesmo as que ainda têm arquivo correspondente em disco, mas nunca os arquivos em si, só os ponteiros do registro —, para quem quer recomeçar do zero sem remover fonte por fonte; e uma forma de escolher explicitamente, por nome, qual fonte de mapa base e qual fonte de elevação usar em `plan` e `geodata slice` (e por consequência em `fly`, que os usa por baixo dos panos), em vez de depender da seleção automática por área. O usuário consegue: limpar o registro inteiro com um único comando; continuar usando a seleção automática de hoje quando não informar nada, sem nenhuma mudança de comportamento para quem não pede nada de novo; e, quando quiser, forçar explicitamente qual mapa base e/ou qual fonte de elevação usar. Requisitos: o comando de limpeza DEVE remover todas as entradas do registro, mesmo as que ainda apontam para um arquivo existente, e NUNCA DEVE apagar, mover ou alterar nenhum arquivo de dado geográfico do usuário — só os ponteiros do registro; por ser uma ação sem volta sobre o registro, DEVE pedir confirmação antes de agir (ou exigir uma flag explícita de confirmação, para uso não interativo/scripts); a seleção explícita DEVE aceitar o nome de uma fonte de mapa base e/ou de uma fonte de elevação já registradas; um nome pedido que não existe no registro DEVE ser recusado, com mensagem clara, antes de qualquer outro trabalho; quando a fonte pedida não cobre totalmente a área que o plano precisa, a ferramenta DEVE recusar exatamente como já recusa hoje uma cobertura incompleta — sem tentar completar automaticamente com outra fonte registrada; ou seja, pedir uma fonte explícita para um tipo de dado (mapa base ou elevação) desliga a seleção automática por ponto só para aquele tipo, não para o outro; quando nenhuma fonte é pedida para um tipo de dado, a seleção automática de hoje continua valendo, idêntica, para esse tipo. Fora de escopo: mudar o algoritmo de seleção automática em si (continua sendo o de menor área, desempate pelo registro mais antigo, quando nenhuma fonte é pedida); editar ou renomear uma entrada já registrada; desfazer a limpeza do registro (sem undo); selecionar mais de uma fonte do mesmo tipo ao mesmo tempo (mesclar mapas, por exemplo); e qualquer mudança nas regras de cobertura, de planejamento de câmera, de recorte, de desenho de quadros ou de vídeo."

## Clarifications

### Session 2026-09-29

- Q: A escolha explícita de fonte deve valer só para `geodata slice` e
  `fly` (os dois comandos que de fato leem o registro), tirando `plan` da
  lista? → A: Sim — `plan` não lê dados geográficos registrados hoje (nem
  para checar cobertura), então a escolha explícita não se aplica a ele; a
  spec foi corrigida para citar só `geodata slice` e `fly`.

## Cenários de Usuário e Testes *(obrigatório)*

### História de Usuário 1 - Recomeçar o registro do zero (Prioridade: P1)

Como usuário do Sobrevoo, registrei várias fontes de mapa e de elevação ao
longo do tempo — testando mapas diferentes, atualizando um recorte de uma
região — e perdi o controle de quais ainda fazem sentido. Quero um jeito de
limpar o registro inteiro com um único comando, sem precisar identificar e
remover cada fonte uma por uma, para poder recomeçar do zero e registrar só
o que preciso agora.

**Por que esta prioridade**: é o valor central da etapa e o mais simples de
entregar — resolve sozinho a dor mais imediata (perder o controle do que
está registrado), sem depender de nenhuma outra capacidade desta etapa.

**Teste Independente**: pode ser totalmente testado registrando várias
fontes, rodando o comando de limpeza, e conferindo que `geodata list` não
mostra mais nenhuma entrada, enquanto os arquivos originais continuam no
disco, prontos para serem registrados de novo.

**Cenários de Aceitação**:

1. **Dado** um registro com várias fontes de mapa base e de elevação,
   **Quando** o usuário roda o comando de limpeza confirmando a ação,
   **Então** `geodata list` não mostra mais nenhuma entrada, e todos os
   arquivos originais continuam intactos no disco.
2. **Dado** o mesmo registro, **Quando** o usuário roda o comando de
   limpeza sem confirmar a ação, **Então** a ferramenta recusa, sem remover
   nada, e informa quantas entradas seriam removidas se confirmada.
3. **Dado** um registro já vazio, **Quando** o usuário roda o comando de
   limpeza confirmando a ação, **Então** a ferramenta aceita normalmente,
   sem erro, e o registro continua vazio.

---

### História de Usuário 2 - Escolher explicitamente qual fonte usar (Prioridade: P2)

Como usuário do Sobrevoo, tenho mais de uma fonte de mapa base ou de
elevação registrada cobrindo a mesma região — por exemplo, duas versões do
mesmo mapa, ou um recorte mais detalhado ao lado de um mais abrangente.
Quero poder dizer explicitamente, pelo nome que dei ao registrar, qual
fonte usar ao conferir cobertura, ao gerar um recorte ou ao rodar o
comando único, em vez de depender da escolha automática — que hoje nem
sempre escolhe a que eu queria, e pode até misturar fontes diferentes no
mesmo recorte sem eu perceber.

**Por que esta prioridade**: depende da capacidade de registrar mais de uma
fonte (já existente) e entrega o valor central de previsibilidade — não
depende da História 1, mas só se torna urgente para quem já registrou o
suficiente para se perder, o que a História 1 também resolve.

**Teste Independente**: pode ser totalmente testado registrando duas fontes
do mesmo tipo cobrindo a mesma área, pedindo uma delas pelo nome ao gerar
um recorte, e conferindo que o recorte resultante usa exclusivamente a
fonte pedida.

**Cenários de Aceitação**:

1. **Dado** duas fontes de mapa base registradas cobrindo a mesma área,
   **Quando** o usuário gera um recorte pedindo uma delas pelo nome,
   **Então** o recorte usa exclusivamente a fonte pedida para o mapa base,
   mesmo que a outra fosse a escolha automática de hoje.
2. **Dado** o mesmo cenário, **Quando** o usuário roda o comando único
   pedindo a mesma fonte pelo nome, **Então** o resultado é consistente
   com o que `geodata slice` produziria pedindo a mesma fonte.
3. **Dado** fontes de mapa base e de elevação registradas, **Quando** o
   usuário pede explicitamente só a fonte de mapa base, **Então** a fonte
   de elevação continua sendo escolhida automaticamente, como hoje.
4. **Dado** o mesmo registro, **Quando** o usuário não pede nenhuma fonte
   explícita, **Então** o resultado é idêntico ao que a ferramenta já
   produzia antes desta etapa.
5. **Dado** duas fontes de mapa base registradas cobrindo áreas
   diferentes, **Quando** o usuário roda `geodata check` pedindo uma delas
   pelo nome, **Então** o relatório de cobertura reflete só a fonte pedida
   — consistente com o que `geodata slice` reportaria para a mesma
   escolha —, antes de qualquer recorte ser gerado.

---

### História de Usuário 3 - Nunca misturar com outra fonte sem avisar (Prioridade: P3)

Como usuário do Sobrevoo, se eu pedir explicitamente uma fonte e ela não
cobrir toda a área que o voo precisa, quero que a ferramenta me avise e
pare — nunca que ela complete silenciosamente o que falta com outra fonte
registrada, porque aí eu não teria certeza de qual fonte foi realmente
usada em cada parte do vídeo.

**Por que esta prioridade**: é a rede de segurança da História 2 — sem ela,
pedir uma fonte explícita poderia dar uma falsa sensação de controle,
já que a ferramenta poderia complementar por baixo dos panos exatamente o
comportamento que o usuário está tentando evitar.

**Teste Independente**: pode ser totalmente testado pedindo uma fonte
explícita cuja área não cobre o trajeto inteiro (mesmo com outra fonte
registrada que cobriria o resto) e conferindo que a ferramenta recusa antes
de gerar qualquer coisa, sem usar a outra fonte para completar.

**Cenários de Aceitação**:

1. **Dado** uma fonte explícita que cobre só parte da área que o plano
   precisa, e outra fonte registrada que cobriria o restante, **Quando** o
   usuário pede a primeira explicitamente, **Então** a ferramenta recusa,
   com uma mensagem de cobertura incompleta, sem usar a segunda fonte para
   completar.
2. **Dado** um nome de fonte que não existe no registro, **Quando** o
   usuário o pede explicitamente em qualquer comando, **Então** a
   ferramenta recusa antes de qualquer outro trabalho, citando o nome
   pedido.

---

### História de Usuário 4 - Nunca reaproveitar em `fly --keep` um recorte da fonte errada (Prioridade: P4)

Como usuário do Sobrevoo, se eu rodar `fly --keep <diretório>` pedindo uma
fonte explícita, interromper, e rodar de novo pedindo uma fonte diferente
(ou voltando para a seleção automática), quero que a ferramenta gere um
recorte novo — nunca que reaproveite, em silêncio, o recorte guardado da
execução anterior, montado com uma fonte diferente da que estou pedindo
agora.

**Por que esta prioridade**: é a rede de segurança da História 2 para o
caso específico do `--keep` — sem ela, a escolha explícita de fonte
convive com um reaproveitamento que não sabe da própria existência dela, e
o usuário teria a falsa impressão de estar vendo o resultado da fonte
pedida quando na verdade está vendo o de uma execução anterior, com outra
fonte. Vem depois da História 2 porque só existe algo a reaproveitar
indevidamente depois que a escolha explícita de fonte já existe.

**Teste Independente**: pode ser totalmente testado rodando `fly --keep
<dir>` pedindo a fonte A, interrompendo antes do fim, rodando de novo
pedindo a fonte B, e conferindo que o recorte final foi montado com a
fonte B — não o guardado da primeira execução.

**Cenários de Aceitação**:

1. **Dado** um `fly --keep <dir>` já executado até gerar e guardar um
   recorte pedindo a fonte de mapa base A, **Quando** o usuário roda de
   novo, mesmo trajeto e mesmos parâmetros, pedindo a fonte de mapa base B,
   **Então** a ferramenta gera um recorte novo com a fonte B — o recorte
   guardado da execução anterior não é reaproveitado.
2. **Dado** o mesmo cenário, **Quando** o usuário roda de novo pedindo a
   mesma fonte A (ou sem pedir nenhuma, se a primeira execução também não
   pediu), **Então** o recorte guardado é reaproveitado normalmente, sem
   gerar um novo.
3. **Dado** um `fly --keep <dir>` executado sem pedir nenhuma fonte
   explícita (seleção automática), **Quando** o usuário roda de novo
   pedindo uma fonte explícita, **Então** a mudança também é tratada como
   mudança de fonte — o recorte guardado não é reaproveitado.

---

### Casos Extremos

- O que acontece ao pedir explicitamente uma fonte registrada como
  elevação no lugar de mapa base (ou vice-versa)? A ferramenta recusa,
  como recusaria um nome inexistente — o tipo da fonte pedida precisa
  corresponder ao tipo do parâmetro.
- O que acontece ao gerar um recorte para um trajeto que cruza o
  antimeridiano ou os polos, pedindo uma fonte explícita? A mesma regra de
  cobertura de sempre vale — a fonte pedida precisa cobrir a área calculada
  do mesmo jeito que precisaria sem escolha explícita.
- O que acontece com `geodata check`, que só relata cobertura e não gera
  plano nem recorte? Passa a aceitar a mesma escolha explícita de fonte
  que `geodata slice` e `fly` — o usuário confere, antes de gastar tempo
  gerando um recorte, se a fonte que pretende usar cobre o trajeto; como
  `check` só relata e nunca gera nada, uma cobertura incompleta aparece no
  relatório como lacuna, não como recusa (FR-007) — mas o relatório passa
  a ser sobre a fonte pedida, não sobre a escolha automática.
- O que acontece com `plan`, que já não lê os dados geográficos
  registrados hoje (só o trajeto GPS)? Continua exatamente assim — a
  escolha explícita não se aplica a ele, porque não há nada nele para
  escolher (Clarifications).
- O que acontece se `fly --keep <dir>` tem um recorte guardado de uma
  execução anterior e a fonte explícita pedida agora (ou a ausência dela)
  é diferente da usada para gerar aquele recorte? O recorte guardado é
  tratado como desatualizado e um novo é gerado — a escolha de fonte entra
  na decisão de reaproveitar, ao lado do que já entra hoje (trajeto e
  parâmetros do plano), comparada contra a procedência que o próprio
  recorte guardado já registra (FR-011), sem precisar de nenhum registro
  novo à parte.

## Requisitos *(obrigatório)*

### Requisitos Funcionais

- **FR-001**: O usuário DEVE poder limpar o registro de dados geográficos
  inteiro com um único comando, removendo todas as entradas de uma vez —
  inclusive as que ainda apontam para um arquivo existente em disco.
- **FR-002**: Limpar o registro NUNCA DEVE apagar, mover ou alterar nenhum
  arquivo de dado geográfico do usuário — a ação afeta somente as entradas
  do registro, nunca os arquivos que elas apontam.
- **FR-003**: Limpar o registro DEVE exigir confirmação explícita antes de
  agir; sem ela, a ferramenta recusa, sem remover nada, e informa quantas
  entradas seriam removidas.
- **FR-004**: O usuário DEVE poder escolher explicitamente, pelo nome já
  usado ao registrar, qual fonte de mapa base e/ou qual fonte de elevação
  usar ao conferir cobertura (`geodata check`), ao gerar um recorte
  (`geodata slice`) e ao rodar o comando único (`fly`) — os três comandos
  que efetivamente leem os dados geográficos registrados; `plan` não lê
  esses dados hoje e continua sem eles (Clarifications).
- **FR-005**: A escolha de mapa base e a escolha de elevação são
  independentes: o usuário pode informar uma, as duas, ou nenhuma.
- **FR-006**: Pedir um nome de fonte que não existe no registro, ou que
  existe mas é de outro tipo (pedir uma fonte de elevação como mapa base,
  por exemplo), DEVE ser recusado, com uma mensagem que cita o nome
  pedido, antes de qualquer outro trabalho.
- **FR-007**: Quando uma fonte é pedida explicitamente para um tipo de
  dado e ela não cobre totalmente a área que a operação precisa, a
  ferramenta NUNCA DEVE completar automaticamente com outra fonte
  registrada do mesmo tipo: em `geodata slice` e `fly`, que geram um
  recorte, isso é recusado exatamente como já é recusada hoje uma
  cobertura incompleta; em `geodata check`, que só relata e não gera nada,
  a lacuna aparece no relatório de cobertura, calculado só com a fonte
  pedida, do mesmo jeito que hoje relata uma lacuna da seleção automática.
- **FR-008**: Pedir uma fonte explícita para um tipo de dado NÃO DEVE
  afetar a seleção do outro tipo: pedir só o mapa base, por exemplo,
  mantém a fonte de elevação sendo escolhida automaticamente, como hoje.
- **FR-009**: Quando nenhuma fonte é pedida para um tipo de dado, o
  comportamento de seleção automática de hoje (a fonte de menor área que
  cobre cada ponto, desempate pelo registro mais antigo) DEVE continuar
  idêntico, sem nenhuma mudança para quem não usa a escolha explícita.
- **FR-010**: Escolher uma fonte explícita para um tipo de dado NUNCA DEVE
  resultar num recorte que mistura, para esse tipo, a fonte pedida com
  outra fonte registrada — o resultado usa exclusivamente a fonte pedida,
  ou a operação é recusada (FR-007).
- **FR-011**: Num `fly --keep <diretório>` que reaproveita um recorte
  guardado de uma execução anterior, a escolha de fonte pedida agora
  (explícita ou não) DEVE fazer parte do que decide se aquele recorte
  ainda vale — um recorte gerado com uma fonte diferente da pedida agora
  (inclusive a automática de antes, ou vice-versa) NUNCA é reaproveitado
  silenciosamente; a ferramenta o trata como desatualizado e gera um novo,
  do mesmo jeito que já trata hoje uma mudança no trajeto ou nos
  parâmetros do plano. A comparação usa a procedência que o próprio
  recorte guardado já registra — cada registro efetivamente usado, com
  nome e tipo — sem precisar de nenhum registro à parte: se a fonte que
  ele lista para um tipo de dado não é a mesma que está sendo pedida agora
  (ou deixou de ser pedida, ou passou a ser pedida), o recorte é
  desatualizado para esse tipo.

### Entidades-Chave

- **Fonte de Dados Geográficos**: a entrada já existente do registro (nome,
  tipo — mapa base ou elevação —, arquivo, área coberta). Esta etapa não
  muda o que ela é, só acrescenta duas formas novas de operar sobre o
  conjunto de fontes registradas: limpar todas de uma vez, e escolher uma
  pelo nome em vez de deixar a seleção automática decidir.

## Critérios de Sucesso *(obrigatório)*

### Resultados Mensuráveis

- **SC-001**: Um usuário com várias fontes registradas consegue voltar a
  um registro vazio com um único comando, sem precisar identificar e
  remover cada entrada manualmente.
- **SC-002**: Depois de limpar o registro, todo arquivo de dado geográfico
  do usuário continua no disco, inalterado, pronto para ser registrado de
  novo.
- **SC-003**: Um usuário com duas fontes do mesmo tipo cobrindo a mesma
  área consegue garantir, escolhendo uma pelo nome, que um recorte gerado
  usa exclusivamente essa fonte — nunca uma mistura das duas.
- **SC-004**: Pedir uma fonte que não existe, que é do tipo errado, ou que
  não cobre a área toda, é sempre recusado antes de qualquer trabalho
  começar, com uma mensagem que identifica exatamente o problema.
- **SC-005**: Um usuário que nunca usa as capacidades novas desta etapa
  obtém, em todo comando existente, exatamente o mesmo comportamento de
  antes desta etapa.
- **SC-006**: Um usuário consegue confirmar, com `geodata check`, se a
  fonte que pretende usar cobre o trajeto inteiro, antes de gastar tempo
  gerando um recorte — a resposta é consistente com o que `geodata slice`
  daria para a mesma escolha.
- **SC-007**: Trocar a fonte explícita pedida entre duas execuções de
  `fly --keep` sobre o mesmo diretório sempre resulta num recorte novo —
  nunca reaproveita, mesmo em silêncio, um recorte guardado que foi
  gerado com uma fonte diferente da pedida agora.

## Suposições

- O comando de limpeza do registro é tratado, nesta especificação, como um
  comando novo dentro do grupo `geodata` (ao lado de `register`, `list`,
  `remove`, `check`) — o nome exato é decidido no planejamento técnico.
- A confirmação exigida para limpar o registro é uma flag explícita (não
  um prompt interativo), consistente com o resto da CLI do Sobrevoo, que
  já pede confirmação de ações que sobrescrevem algo (como `--overwrite`)
  sempre por flag, nunca por prompt.
- A escolha explícita de fonte é feita pelo nome que o usuário já deu ao
  registrar (`geodata register --name`), nunca por caminho de arquivo — o
  registro já é o único lugar que associa um nome a um arquivo.
- `geodata check` passa a aceitar a mesma escolha explícita de fonte que
  `geodata slice` e `fly`, para o usuário conferir a cobertura da fonte
  que pretende usar antes de gastar tempo gerando um recorte; como
  `check` só relata e nunca gera nada, uma fonte pedida que não cobre tudo
  aparece como lacuna no relatório, não como recusa (FR-007). `plan` não
  lê dados geográficos registrados hoje (só o trajeto GPS) e continua
  assim, sem nenhuma flag de fonte (Clarifications).
- O reaproveitamento de um recorte guardado por `fly --keep` já usa, hoje,
  a identidade do plano e a do próprio recorte para decidir se ainda vale;
  esta etapa estende essa mesma decisão para incluir a escolha de fonte.
  O recorte exportado já registra, para cada registro efetivamente usado,
  o nome e o tipo (a procedência descrita em
  `specs/004-geo-data-slice/contracts/slice-file.md`, campo `sources[]`) —
  é essa informação, já presente no arquivo guardado, que a comparação usa;
  não é necessário nenhum registro novo à parte para lembrar qual fonte foi
  pedida numa execução anterior. O mecanismo exato de como essa comparação
  se encaixa na decisão de reaproveitar fica para o planejamento técnico,
  seguindo o mesmo princípio já usado nas etapas 8 e 9 para a aparência e
  os overlays: tudo que muda o artefato entra na identidade dele.
- O algoritmo de seleção automática usado quando nenhuma fonte é pedida
  não muda nesta etapa (continua sendo o de menor área cobrindo o ponto,
  com empate resolvido pelo registro mais antigo).
- Editar ou renomear uma entrada já registrada, e desfazer uma limpeza do
  registro, permanecem fora do escopo desta etapa — quem quiser um nome ou
  arquivo diferente remove e registra de novo.
