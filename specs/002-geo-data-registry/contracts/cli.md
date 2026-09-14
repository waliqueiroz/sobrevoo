# Contrato de CLI: `sobrevoo geodata`

**Feature**: `002-geo-data-registry` | **Data**: 2026-09-13

Este é o único contrato externo desta etapa: o comando pai `geodata` e seus
quatro subcomandos, que expõem `RegisterGeoDataService`,
`ListGeoDataService`, `RemoveGeoDataService` e `CheckCoverageService`
(`internal/infra/inbound/cli`). Não há API HTTP nem GUI nesta etapa (fora de
escopo, ver `spec.md`). O contrato de `sobrevoo inspect` da etapa 1
(`specs/001-gps-track-processing/contracts/cli.md`) permanece inalterado.

## `sobrevoo geodata register`

```text
sobrevoo geodata register <arquivo> --name <nome>
```

- `<arquivo>` (argumento posicional, obrigatório): caminho para um arquivo
  local de mapa base (MBTiles) ou de relevo (GeoTIFF em CRS geográfico). O
  tipo e a área geográfica cobertos são determinados automaticamente a
  partir do conteúdo do arquivo (FR-002, FR-003) — a extensão do arquivo é
  ignorada para essa decisão.
- `--name` (obrigatório): nome escolhido pelo usuário para este registro;
  deve ser único entre todos os registros existentes (FR-007).

### Saída (sucesso)

Texto legível em inglês, impresso em `stdout`, confirmando o registro
criado com pelo menos: nome, tipo (`base map` ou `elevation`) e área
geográfica coberta (mesmo formato de bounding box já usado por
`sobrevoo inspect`, incluindo a indicação de antimeridiano quando
aplicável).

**Código de saída**: `0`.

### Saída (erro)

| Cenário | Erro sentinela do domínio | Código de saída |
|---|---|---|
| Caminho de arquivo inexistente | `domain.ErrDataFileNotFound` | `5` |
| Arquivo existe mas não pode ser lido | `domain.ErrDataFileUnreadable` | `6` |
| Formato de arquivo não reconhecido (não é MBTiles nem GeoTIFF em CRS geográfico) | `domain.ErrUnsupportedDataFormat` | `7` |
| Nome já usado por outro registro | `domain.ErrDataSourceNameAlreadyUsed` | `8` |
| Uso inválido da CLI (`--name` ausente, argumento faltando) | erro do próprio Cobra | `2` (padrão do Cobra) |

## `sobrevoo geodata list`

```text
sobrevoo geodata list
```

Sem argumentos nem flags.

### Saída (sucesso)

Texto legível em inglês, impresso em `stdout`, com uma linha por registro
contendo nome, tipo, área geográfica coberta e disponibilidade do arquivo
(FR-009, FR-010). Um registro cujo arquivo não é mais encontrado é
sinalizado explicitamente (ex.: `(file not found)`) em vez de omitido ou de
interromper a listagem dos demais. Quando não há nenhum registro, a saída
indica isso claramente em vez de imprimir uma lista vazia sem explicação.

**Código de saída**: `0` (mesmo quando não há nenhum registro — lista vazia
não é um erro).

## `sobrevoo geodata remove`

```text
sobrevoo geodata remove <nome>
```

- `<nome>` (argumento posicional, obrigatório): nome do registro a remover.
  O arquivo de dados original nunca é apagado (FR-011).

### Saída (sucesso)

Confirmação textual em `stdout` de que o registro foi removido.

**Código de saída**: `0`.

### Saída (erro)

| Cenário | Erro sentinela do domínio | Código de saída |
|---|---|---|
| Nome não corresponde a nenhum registro existente | `domain.ErrDataSourceNotRegistered` | `9` |
| Uso inválido da CLI (argumento faltando) | erro do próprio Cobra | `2` (padrão do Cobra) |

## `sobrevoo geodata check`

```text
sobrevoo geodata check <arquivo-de-trajeto>
```

- `<arquivo-de-trajeto>` (argumento posicional, obrigatório): caminho para
  um trajeto GPS local, no mesmo formato aceito por `sobrevoo inspect`
  (GPX). O trajeto passa pela mesma validação e limpeza já usada por
  `inspect` (parse, reordenação por tempo, descarte de pontos
  problemáticos) — mas **não** é simplificado nem suavizado, para não
  mascarar lacunas reais de cobertura (ver `research.md` item 9).

### Saída (sucesso)

Texto legível em inglês, impresso em `stdout`, contendo pelo menos (FR-013
a FR-017):

- O veredito geral: totalmente coberto, parcialmente coberto, ou não
  coberto (SC-003).
- Quando não totalmente coberto: a lista de subtrechos não cobertos, cada
  um com as coordenadas geográficas de início e fim, e se falta mapa base,
  relevo, ou ambos naquele subtrecho (FR-015).
- Os registros de mapa base e de relevo efetivamente usados para cobrir
  pelo menos um ponto do trajeto — relevante para o usuário entender qual
  registro foi escolhido quando mais de uma fonte cobria a mesma área
  (FR-016).

O comando **sempre** produz esse relatório com sucesso (código `0`),
mesmo quando o veredito é "não coberto" — um trajeto sem cobertura
suficiente não é, por si só, um erro de execução.

**Código de saída**: `0`.

### Saída (erro)

Reaproveita exatamente os mesmos erros sentinela e códigos de saída de
`sobrevoo inspect` para um trajeto inválido (ver
`specs/001-gps-track-processing/contracts/cli.md`) — a verificação de
cobertura não introduz nenhum erro novo para essa parte:

| Cenário | Erro sentinela do domínio | Código de saída |
|---|---|---|
| Arquivo de trajeto vazio | `domain.ErrEmptyFile` | `1` |
| Formato de trajeto não reconhecido | `domain.ErrUnsupportedFormat` | `2` |
| Pontos insuficientes (arquivo original) | `domain.ErrInsufficientPoints` | `3` |
| Pontos insuficientes após a limpeza | `domain.ErrInsufficientPointsAfterCleaning` | `3` |
| Arquivo de trajeto inexistente ou sem permissão de leitura | erro de I/O (não é um erro sentinela do domínio) | `4` |
| Uso inválido da CLI (argumento faltando) | erro do próprio Cobra | `2` (padrão do Cobra) |

## Tabela consolidada de códigos de saída (novos nesta etapa)

| Código | Significado |
|---|---|
| `5` | Arquivo de dados geográfico não encontrado (`register`) |
| `6` | Arquivo de dados geográfico ilegível (`register`) |
| `7` | Formato de dado geográfico não suportado (`register`) |
| `8` | Nome de registro já em uso (`register`) |
| `9` | Nome de registro não encontrado (`remove`) |

Os códigos `0` a `4`, já contratados pela etapa 1, mantêm exatamente o
mesmo significado. A tradução de todo erro sentinela para código de saída
continua centralizada inteiramente no adapter de CLI (`exit_code.go`,
Princípio VII da constituição) — o núcleo nunca chama `os.Exit` nem conhece
códigos de saída.

## Reuso futuro

Nenhum dos quatro novos serviços conhece este contrato de CLI: cada um
recebe seu próprio DTO de entrada e devolve seu próprio DTO de saída como
dado puro (ver `data-model.md`). Um futuro adapter HTTP construirá os
mesmos DTOs de entrada a partir de uma requisição, chamará os mesmos
serviços, e traduzirá os DTOs de saída e os mesmos erros sentinela para um
corpo de resposta e um status HTTP — sem duplicar nenhuma regra de negócio
(Princípio III).
