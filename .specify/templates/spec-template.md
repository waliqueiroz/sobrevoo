# Especificação de Funcionalidade: [NOME DA FUNCIONALIDADE]

**Branch da Funcionalidade**: `[###-feature-name]`

**Criado em**: [DATE]

**Status**: Rascunho

**Entrada**: Descrição do usuário: "$ARGUMENTS"

## Cenários de Usuário e Testes *(obrigatório)*

<!--
  IMPORTANTE: As histórias de usuário devem ser PRIORIZADAS como jornadas de usuário ordenadas por importância.
  Cada história/jornada de usuário deve ser TESTÁVEL DE FORMA INDEPENDENTE - ou seja, se você implementar
  apenas UMA delas, ainda deve ter um MVP (Produto Mínimo Viável) viável que entregue valor.

  Atribua prioridades (P1, P2, P3, etc.) a cada história, sendo P1 a mais crítica.
  Pense em cada história como uma fatia independente de funcionalidade que pode ser:
  - Desenvolvida independentemente
  - Testada independentemente
  - Implantada independentemente
  - Demonstrada aos usuários independentemente
-->

### História de Usuário 1 - [Título Breve] (Prioridade: P1)

[Descreva esta jornada de usuário em linguagem simples]

**Por que esta prioridade**: [Explique o valor e por que ela tem este nível de prioridade]

**Teste Independente**: [Descreva como isso pode ser testado de forma independente - ex.: "Pode ser totalmente testado ao [ação específica] e entrega [valor específico]"]

**Cenários de Aceitação**:

1. **Dado** [estado inicial], **Quando** [ação], **Então** [resultado esperado]
2. **Dado** [estado inicial], **Quando** [ação], **Então** [resultado esperado]

---

### História de Usuário 2 - [Título Breve] (Prioridade: P2)

[Descreva esta jornada de usuário em linguagem simples]

**Por que esta prioridade**: [Explique o valor e por que ela tem este nível de prioridade]

**Teste Independente**: [Descreva como isso pode ser testado de forma independente]

**Cenários de Aceitação**:

1. **Dado** [estado inicial], **Quando** [ação], **Então** [resultado esperado]

---

### História de Usuário 3 - [Título Breve] (Prioridade: P3)

[Descreva esta jornada de usuário em linguagem simples]

**Por que esta prioridade**: [Explique o valor e por que ela tem este nível de prioridade]

**Teste Independente**: [Descreva como isso pode ser testado de forma independente]

**Cenários de Aceitação**:

1. **Dado** [estado inicial], **Quando** [ação], **Então** [resultado esperado]

---

[Adicione mais histórias de usuário conforme necessário, cada uma com uma prioridade atribuída]

### Casos Extremos

<!--
  AÇÃO NECESSÁRIA: O conteúdo desta seção representa espaços reservados.
  Preencha-os com os casos extremos corretos.
-->

- O que acontece quando [condição limite]?
- Como o sistema lida com [cenário de erro]?

## Requisitos *(obrigatório)*

<!--
  AÇÃO NECESSÁRIA: O conteúdo desta seção representa espaços reservados.
  Preencha-os com os requisitos funcionais corretos.
-->

### Requisitos Funcionais

- **FR-001**: O sistema DEVE [capacidade específica, ex.: "permitir que usuários criem contas"]
- **FR-002**: O sistema DEVE [capacidade específica, ex.: "validar endereços de e-mail"]
- **FR-003**: Os usuários DEVEM ser capazes de [interação chave, ex.: "redefinir sua senha"]
- **FR-004**: O sistema DEVE [requisito de dados, ex.: "persistir preferências do usuário"]
- **FR-005**: O sistema DEVE [comportamento, ex.: "registrar todos os eventos de segurança"]

*Exemplo de marcação de requisitos pouco claros:*

- **FR-006**: O sistema DEVE autenticar usuários via [NEEDS CLARIFICATION: método de autenticação não especificado - e-mail/senha, SSO, OAuth?]
- **FR-007**: O sistema DEVE reter dados do usuário por [NEEDS CLARIFICATION: período de retenção não especificado]

### Entidades-Chave *(incluir se a funcionalidade envolver dados)*

- **[Entidade 1]**: [O que representa, atributos-chave sem detalhes de implementação]
- **[Entidade 2]**: [O que representa, relacionamentos com outras entidades]

## Critérios de Sucesso *(obrigatório)*

<!--
  AÇÃO NECESSÁRIA: Defina critérios de sucesso mensuráveis.
  Eles devem ser agnósticos de tecnologia e mensuráveis.
-->

### Resultados Mensuráveis

- **SC-001**: [Métrica mensurável, ex.: "Usuários conseguem concluir a criação de conta em menos de 2 minutos"]
- **SC-002**: [Métrica mensurável, ex.: "O sistema suporta 1000 usuários simultâneos sem degradação"]
- **SC-003**: [Métrica de satisfação do usuário, ex.: "90% dos usuários concluem a tarefa principal com sucesso na primeira tentativa"]
- **SC-004**: [Métrica de negócio, ex.: "Reduzir chamados de suporte relacionados a [X] em 50%"]

## Suposições

<!--
  AÇÃO NECESSÁRIA: O conteúdo desta seção representa espaços reservados.
  Preencha-os com as suposições corretas com base em padrões razoáveis
  escolhidos quando a descrição da funcionalidade não especificou certos detalhes.
-->

- [Suposição sobre usuários-alvo, ex.: "Usuários têm conexão de internet estável"]
- [Suposição sobre limites de escopo, ex.: "Suporte móvel está fora do escopo para v1"]
- [Suposição sobre dados/ambiente, ex.: "O sistema de autenticação existente será reutilizado"]
- [Dependência de sistema/serviço existente, ex.: "Requer acesso à API de perfil de usuário existente"]
