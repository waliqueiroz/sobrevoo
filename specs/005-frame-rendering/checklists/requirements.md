# Checklist de Qualidade da Especificação: Desenho dos Quadros do Voo

**Propósito**: Validar a completude e a qualidade da especificação antes de seguir para o planejamento
**Criado em**: 2026-09-26
**Funcionalidade**: [spec.md](../spec.md)

## Qualidade do Conteúdo

- [X] Sem detalhes de implementação (linguagens, frameworks, APIs)
- [X] Focada no valor para o usuário e nas necessidades de negócio
- [X] Escrita para partes interessadas não técnicas
- [X] Todas as seções obrigatórias preenchidas

## Completude dos Requisitos

- [X] Nenhum marcador [NEEDS CLARIFICATION] permanece
- [X] Requisitos testáveis e sem ambiguidade
- [X] Critérios de sucesso mensuráveis
- [X] Critérios de sucesso agnósticos de tecnologia (sem detalhes de implementação)
- [X] Todos os cenários de aceitação definidos
- [X] Casos extremos identificados
- [X] Escopo claramente delimitado
- [X] Dependências e suposições identificadas

## Prontidão da Funcionalidade

- [X] Todos os requisitos funcionais têm critérios de aceitação claros
- [X] Os cenários de usuário cobrem os fluxos principais
- [X] A funcionalidade atende aos resultados mensuráveis definidos nos Critérios de Sucesso
- [X] Nenhum detalhe de implementação vaza para a especificação

## Notas

- Validação em 1 iteração; nenhum item falhou.
- Formatos de dado citados (PNG como saída; PNG/JPG/WebP e `pbf` nas peças de mapa) são formatos de arquivo que o usuário lida, não escolhas de implementação, e seguem o padrão das etapas 3 e 4. Os valores exatos de resolução, de nome de arquivo e do registro do conjunto ficam para o planejamento.
- **Decisões tomadas por padrão, que valem ser confirmadas em `/speckit-clarify`** (nenhuma bloqueia o planejamento):
  1. *Correspondência plano-recorte* é geométrica (área recalculada pela regra da etapa 4: contida = cobre; igual = corresponde), porque o recorte da etapa 4 não traz identificação do plano. A alternativa (impressão digital do plano dentro do recorte) alteraria a saída da etapa 4.
  2. *Retomada x "não sobrescrever"*: por padrão o destino com quadros do mesmo conjunto é retomado (mantém o que existe); com quadros de outro conjunto é recusado; a sobrescrita explícita redesenha e remove sobras do conjunto anterior. Isso exige um registro do conjunto no destino.
  3. *Resumo* conta buracos só dos quadros desenhados na execução; os mantidos são informados à parte.
  4. *Recorte sem nenhuma elevação com valor* é recusado (não há terreno nem referência de altura); com valor parcial, o quadro é desenhado e contado como buraco de elevação.
  5. *Marcador e traçado* obedecem à oclusão do terreno.
- **Aviso sobre dados reais**: o mapa base real do usuário (BBBike) é vetorial (`pbf`) e será recusado por esta etapa, por escopo; o desenho será validado com mapa base sintético em imagem.
