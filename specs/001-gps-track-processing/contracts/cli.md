# Contrato de CLI: `sobrevoo inspect`

**Feature**: `001-gps-track-processing` | **Data**: 2026-09-13

Este é o único contrato externo desta etapa: o comando de linha de comando que
expõe o serviço `InspectTrackService` (`internal/infra/inbound/cli`). Não há
API HTTP nem GUI nesta etapa (fora de escopo, ver `spec.md`).

## Comando

```text
sobrevoo inspect <arquivo>
```

- `<arquivo>` (argumento posicional, obrigatório): caminho para um arquivo de
  rastreamento GPS local em formato GPX. O conteúdo é validado como GPX
  (FR-002) — a extensão do arquivo é ignorada para essa decisão.

## Flags

| Flag | Valores aceitos | Padrão | Descrição |
|---|---|---|---|
| `--simplification` | `low`, `medium`, `high` | `medium` | Nível de simplificação aplicado à redução de pontos (FR-014, FR-016). |
| `--smoothing` | `low`, `medium`, `high` | `medium` | Nível de suavização aplicado ao traçado (FR-015, FR-016). |

Um valor de flag fora do conjunto aceito é um erro de uso da CLI (tratado pelo
próprio Cobra, antes de qualquer chamada ao caso de uso) — a mensagem lista os
valores válidos.

## Saída (sucesso)

Texto legível em inglês, impresso em `stdout` (todo o I/O em tempo de
execução da ferramenta é em inglês — ver Suposições em `spec.md`), contendo
pelo menos os seguintes campos (FR-025), na ordem definida pelo adapter (não é
um formato estruturado versionado nesta etapa):

- Formato identificado (GPX)
- Quantidade de pontos original
- Quantidade de pontos após o tratamento
- Distância total (em quilômetros)
- Ganho de elevação (em metros), ou uma indicação explícita de que não havia
  dado de altitude
- Duração da atividade, ou uma indicação explícita de que não havia dado de
  tempo
- Área geográfica ocupada (extremos de latitude e longitude, incluindo o caso
  de trajeto atravessando o antimeridiano)
- Quantidade de pontos descartados, discriminada por motivo (coordenada
  impossível, duplicado consecutivo, salto implausível)

**Código de saída**: `0`.

## Saída (erro)

Mensagem de erro legível em inglês, impressa em `stderr`, explicando o motivo
da recusa e o que o usuário pode verificar (FR-007). Nenhum resumo parcial é
impresso em caso de erro.

| Cenário | Erro sentinela do domínio | Código de saída |
|---|---|---|
| Arquivo vazio | `domain.ErrEmptyFile` | `1` |
| Formato não reconhecido | `domain.ErrUnsupportedFormat` | `2` |
| Pontos insuficientes (arquivo original) | `domain.ErrInsufficientPoints` | `3` |
| Pontos insuficientes após a limpeza | `domain.ErrInsufficientPointsAfterCleaning` | `3` |
| Arquivo inexistente ou sem permissão de leitura | erro de I/O do adapter de CLI (não é um erro sentinela do domínio) | `4` |
| Uso inválido da CLI (flag com valor fora do conjunto aceito, argumento faltando) | erro do próprio Cobra | `2` (padrão do Cobra) |

A tabela de códigos de saída é a tradução, feita inteiramente no adapter de
CLI, dos erros sentinela declarados no domínio (Princípio VII da
constituição) — o núcleo nunca chama `os.Exit` nem conhece códigos de saída.

## Reuso futuro

O serviço `InspectTrackService` (`internal/application`) não conhece este
contrato de CLI: ele recebe `InspectTrackInput` e devolve `InspectTrackOutput`
como dados puros (ver `data-model.md`). Um futuro adapter HTTP construirá o
mesmo `InspectTrackInput` a partir de uma requisição, chamará o mesmo serviço,
e traduzirá `InspectTrackOutput` e os mesmos erros sentinela para um corpo de
resposta e um status HTTP — sem duplicar nenhuma regra de negócio (Princípio
III).
