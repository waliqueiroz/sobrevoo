---

description: "Task list template for feature implementation"
---

# Tarefas: [NOME DA FUNCIONALIDADE]

**Entrada**: Documentos de design de `/specs/[###-feature-name]/`

**Pré-requisitos**: plan.md (obrigatório), spec.md (obrigatório para histórias de usuário), research.md, data-model.md, contracts/

**Testes**: Os exemplos abaixo incluem tarefas de teste. Os testes são OPCIONAIS - inclua-os apenas se explicitamente solicitado na especificação da funcionalidade.

**Organização**: As tarefas são agrupadas por história de usuário para permitir implementação e teste independentes de cada história.

## Formato: `[ID] [P?] [Story] Descrição`

- **[P]**: Pode ser executado em paralelo (arquivos diferentes, sem dependências)
- **[Story]**: A qual história de usuário esta tarefa pertence (ex.: US1, US2, US3)
- Inclua caminhos de arquivo exatos nas descrições

## Convenções de Caminho

- **Projeto único**: `src/`, `tests/` na raiz do repositório
- **Aplicação web**: `backend/src/`, `frontend/src/`
- **Mobile**: `api/src/`, `ios/src/` ou `android/src/`
- Os caminhos mostrados abaixo assumem projeto único - ajuste com base na estrutura do plan.md

<!--
  ============================================================================
  IMPORTANTE: As tarefas abaixo são TAREFAS DE EXEMPLO apenas para fins de ilustração.

  O comando /speckit-tasks DEVE substituí-las por tarefas reais com base em:
  - Histórias de usuário do spec.md (com suas prioridades P1, P2, P3...)
  - Requisitos da funcionalidade do plan.md
  - Entidades do data-model.md
  - Endpoints do contracts/

  As tarefas DEVEM ser organizadas por história de usuário para que cada história possa ser:
  - Implementada independentemente
  - Testada independentemente
  - Entregue como um incremento de MVP

  NÃO mantenha essas tarefas de exemplo no arquivo tasks.md gerado.
  ============================================================================
-->

## Phase 1: Setup (Shared Infrastructure)

**Propósito**: Inicialização do projeto e estrutura básica

- [ ] T001 Criar estrutura do projeto conforme o plano de implementação
- [ ] T002 Inicializar projeto [linguagem] com dependências do [framework]
- [ ] T003 [P] Configurar ferramentas de linting e formatação

---

## Phase 2: Foundational (Blocking Prerequisites)

**Propósito**: Infraestrutura principal que DEVE estar completa antes que QUALQUER história de usuário possa ser implementada

**⚠️ CRÍTICO**: Nenhum trabalho de história de usuário pode começar até que esta fase esteja completa

Exemplos de tarefas fundamentais (ajuste com base no seu projeto):

- [ ] T004 Configurar esquema de banco de dados e framework de migrações
- [ ] T005 [P] Implementar framework de autenticação/autorização
- [ ] T006 [P] Configurar roteamento de API e estrutura de middleware
- [ ] T007 Criar modelos/entidades base dos quais todas as histórias dependem
- [ ] T008 Configurar infraestrutura de tratamento de erros e logging
- [ ] T009 Configurar gerenciamento de configuração de ambiente

**Checkpoint**: Fundação pronta - a implementação das histórias de usuário pode começar em paralelo

---

## Phase 3: User Story 1 - [Título] (Priority: P1) 🎯 MVP

**Objetivo**: [Breve descrição do que esta história entrega]

**Teste Independente**: [Como verificar se esta história funciona por conta própria]

### Testes para a História de Usuário 1 (OPCIONAL - apenas se testes forem solicitados) ⚠️

> **NOTA: Escreva esses testes PRIMEIRO, garanta que eles FALHEM antes da implementação**

- [ ] T010 [P] [US1] Teste de contrato para [endpoint] em tests/contract/test_[name].py
- [ ] T011 [P] [US1] Teste de integração para [jornada do usuário] em tests/integration/test_[name].py

### Implementação da História de Usuário 1

- [ ] T012 [P] [US1] Criar modelo [Entity1] em src/models/[entity1].py
- [ ] T013 [P] [US1] Criar modelo [Entity2] em src/models/[entity2].py
- [ ] T014 [US1] Implementar [Service] em src/services/[service].py (depende de T012, T013)
- [ ] T015 [US1] Implementar [endpoint/funcionalidade] em src/[location]/[file].py
- [ ] T016 [US1] Adicionar validação e tratamento de erros
- [ ] T017 [US1] Adicionar logging para as operações da história de usuário 1

**Checkpoint**: Neste ponto, a História de Usuário 1 deve estar totalmente funcional e testável de forma independente

---

## Phase 4: User Story 2 - [Título] (Priority: P2)

**Objetivo**: [Breve descrição do que esta história entrega]

**Teste Independente**: [Como verificar se esta história funciona por conta própria]

### Testes para a História de Usuário 2 (OPCIONAL - apenas se testes forem solicitados) ⚠️

- [ ] T018 [P] [US2] Teste de contrato para [endpoint] em tests/contract/test_[name].py
- [ ] T019 [P] [US2] Teste de integração para [jornada do usuário] em tests/integration/test_[name].py

### Implementação da História de Usuário 2

- [ ] T020 [P] [US2] Criar modelo [Entity] em src/models/[entity].py
- [ ] T021 [US2] Implementar [Service] em src/services/[service].py
- [ ] T022 [US2] Implementar [endpoint/funcionalidade] em src/[location]/[file].py
- [ ] T023 [US2] Integrar com componentes da História de Usuário 1 (se necessário)

**Checkpoint**: Neste ponto, as Histórias de Usuário 1 E 2 devem funcionar de forma independente

---

## Phase 5: User Story 3 - [Título] (Priority: P3)

**Objetivo**: [Breve descrição do que esta história entrega]

**Teste Independente**: [Como verificar se esta história funciona por conta própria]

### Testes para a História de Usuário 3 (OPCIONAL - apenas se testes forem solicitados) ⚠️

- [ ] T024 [P] [US3] Teste de contrato para [endpoint] em tests/contract/test_[name].py
- [ ] T025 [P] [US3] Teste de integração para [jornada do usuário] em tests/integration/test_[name].py

### Implementação da História de Usuário 3

- [ ] T026 [P] [US3] Criar modelo [Entity] em src/models/[entity].py
- [ ] T027 [US3] Implementar [Service] em src/services/[service].py
- [ ] T028 [US3] Implementar [endpoint/funcionalidade] em src/[location]/[file].py

**Checkpoint**: Todas as histórias de usuário devem agora estar funcionais de forma independente

---

[Adicione mais fases de histórias de usuário conforme necessário, seguindo o mesmo padrão]

---

## Phase N: Polish & Cross-Cutting Concerns

**Propósito**: Melhorias que afetam múltiplas histórias de usuário

- [ ] TXXX [P] Atualizações de documentação em docs/
- [ ] TXXX Limpeza de código e refatoração
- [ ] TXXX Otimização de desempenho em todas as histórias
- [ ] TXXX [P] Testes unitários adicionais (se solicitado) em tests/unit/
- [ ] TXXX Reforço de segurança
- [ ] TXXX Executar validação do quickstart.md

---

## Dependências e Ordem de Execução

### Dependências entre Fases

- **Setup (Phase 1)**: Sem dependências - pode começar imediatamente
- **Foundational (Phase 2)**: Depende da conclusão do Setup - BLOQUEIA todas as histórias de usuário
- **User Stories (Phase 3+)**: Todas dependem da conclusão da fase Foundational
  - As histórias de usuário podem então prosseguir em paralelo (se houver equipe suficiente)
  - Ou sequencialmente em ordem de prioridade (P1 → P2 → P3)
- **Polish (Fase Final)**: Depende da conclusão de todas as histórias de usuário desejadas

### Dependências entre Histórias de Usuário

- **História de Usuário 1 (P1)**: Pode começar após Foundational (Phase 2) - Sem dependências de outras histórias
- **História de Usuário 2 (P2)**: Pode começar após Foundational (Phase 2) - Pode integrar-se com US1, mas deve ser testável independentemente
- **História de Usuário 3 (P3)**: Pode começar após Foundational (Phase 2) - Pode integrar-se com US1/US2, mas deve ser testável independentemente

### Dentro de Cada História de Usuário

- Testes (se incluídos) DEVEM ser escritos e FALHAR antes da implementação
- Modelos antes de serviços
- Serviços antes de endpoints
- Implementação principal antes da integração
- História completa antes de avançar para a próxima prioridade

### Oportunidades de Paralelização

- Todas as tarefas de Setup marcadas com [P] podem ser executadas em paralelo
- Todas as tarefas Foundational marcadas com [P] podem ser executadas em paralelo (dentro da Phase 2)
- Uma vez concluída a fase Foundational, todas as histórias de usuário podem começar em paralelo (se a capacidade da equipe permitir)
- Todos os testes de uma história de usuário marcados com [P] podem ser executados em paralelo
- Modelos dentro de uma história marcados com [P] podem ser executados em paralelo
- Histórias de usuário diferentes podem ser trabalhadas em paralelo por membros diferentes da equipe

---

## Exemplo de Paralelização: História de Usuário 1

```bash
# Disparar todos os testes da História de Usuário 1 juntos (se testes forem solicitados):
Task: "Teste de contrato para [endpoint] em tests/contract/test_[name].py"
Task: "Teste de integração para [jornada do usuário] em tests/integration/test_[name].py"

# Disparar todos os modelos da História de Usuário 1 juntos:
Task: "Criar modelo [Entity1] em src/models/[entity1].py"
Task: "Criar modelo [Entity2] em src/models/[entity2].py"
```

---

## Estratégia de Implementação

### MVP Primeiro (Somente História de Usuário 1)

1. Completar Phase 1: Setup
2. Completar Phase 2: Foundational (CRÍTICO - bloqueia todas as histórias)
3. Completar Phase 3: História de Usuário 1
4. **PARAR E VALIDAR**: Testar a História de Usuário 1 de forma independente
5. Implantar/demonstrar se estiver pronta

### Entrega Incremental

1. Completar Setup + Foundational → Fundação pronta
2. Adicionar História de Usuário 1 → Testar independentemente → Implantar/Demonstrar (MVP!)
3. Adicionar História de Usuário 2 → Testar independentemente → Implantar/Demonstrar
4. Adicionar História de Usuário 3 → Testar independentemente → Implantar/Demonstrar
5. Cada história agrega valor sem quebrar as histórias anteriores

### Estratégia para Equipe em Paralelo

Com múltiplos desenvolvedores:

1. A equipe completa Setup + Foundational em conjunto
2. Uma vez concluído o Foundational:
   - Desenvolvedor A: História de Usuário 1
   - Desenvolvedor B: História de Usuário 2
   - Desenvolvedor C: História de Usuário 3
3. As histórias se completam e se integram de forma independente

---

## Notas

- Tarefas [P] = arquivos diferentes, sem dependências
- O rótulo [Story] mapeia a tarefa para uma história de usuário específica para rastreabilidade
- Cada história de usuário deve ser completável e testável de forma independente
- Verifique se os testes falham antes de implementar
- Faça commit após cada tarefa ou grupo lógico
- Pare em qualquer checkpoint para validar a história de forma independente
- Evite: tarefas vagas, conflitos no mesmo arquivo, dependências entre histórias que quebrem a independência
