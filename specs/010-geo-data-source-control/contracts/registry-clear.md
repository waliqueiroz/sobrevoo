# Contrato de CLI: `sobrevoo geodata clear`

**Feature**: `010-geo-data-source-control` | **Data**: 2026-09-29

Comando novo, filho de `geodata`, que expõe `GeoDataService.Clear`
(`data-model.md`). Os contratos existentes de `register`/`list`/`remove`/
`check` (`specs/002-geo-data-registry/contracts/cli.md`) permanecem válidos
sem nenhuma mudança.

```text
sobrevoo geodata clear [--confirm]
```

Sem argumentos posicionais. `--confirm` (booleana, padrão `false`) é a única
flag — a mesma convenção de flag explícita de confirmação que `--overwrite`
já estabelece no resto da CLI (nunca um prompt interativo).

## Saída (sucesso, com `--confirm`)

Texto legível em inglês, em `stdout`, confirmando quantas entradas foram
removidas (FR-001):

```text
Cleared the registry: 3 entries removed.
```

Um registro já vazio também é aceito normalmente, sem erro:

```text
Cleared the registry: 0 entries removed.
```

Nenhum arquivo de dado geográfico do usuário é tocado (FR-002) — só as
entradas do registro.

**Código de saída**: `0`.

## Saída (erro)

| Cenário | Erro sentinela do domínio | Código de saída |
|---|---|---|
| `--confirm` não informado | `domain.ErrRegistryClearNotConfirmed` | `57` |

A mensagem do erro cita quantas entradas seriam removidas, para o usuário
decidir sem precisar rodar `geodata list` antes (FR-003):

```text
error: registry clear was not confirmed: 3 entries would be removed; re-run with --confirm
```

Nenhum outro cenário de erro: o comando não lê nenhum arquivo de dado
geográfico (só o registro), então não há caminho de arquivo para validar, e
não aceita argumento posicional (um argumento extra é erro de uso do
próprio Cobra, código `2`).

## Exit code novo

| Código | Erro sentinela | Quando |
|---|---|---|
| `57` | `ErrRegistryClearNotConfirmed` | `geodata clear` rodado sem `--confirm`. |

`56` e `58` são os outros dois códigos novos desta etapa — ver
`contracts/source-selection-flags.md`. Nenhum código existente muda de
significado.
