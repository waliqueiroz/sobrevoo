# Mudança no Contrato do Recorte Exportado (etapa 4)

**Feature**: `005-frame-rendering` | **Data**: 2026-09-26

Esta etapa acrescenta **um campo** ao arquivo de recorte definido em
`specs/004-geo-data-slice/contracts/slice-file.md` (Clarificação de
2026-09-26 da spec desta etapa). É a única mudança em artefatos de etapas
anteriores; na implementação, o contrato da etapa 4 recebe a mesma nota (como a
etapa 4 fez com o contrato de `register`).

## O campo

`manifest.json`, logo depois de `format_version`:

```json
{
  "format_version": 1,
  "plan_id": "9f2c…64 caracteres hexadecimais minúsculos…",
  "area": { … },
  …
}
```

| Campo | Tipo | Semântica |
|---|---|---|
| `plan_id` | texto | `CameraPlan.ID()` do plano informado a `geodata slice`: SHA-256 hexadecimal (64 caracteres minúsculos) da codificação canônica do plano (parâmetros efetivos e, por quadro, índice, fase e valores quantizados aos passos do plano). Igual para o mesmo conteúdo de plano, seja qual for a formatação do arquivo. |

## Versão e compatibilidade

- `format_version` **continua 1**: o campo é acrescentado, não muda o
  significado de nenhum existente (regra do próprio contrato: "acrescentar
  campos não muda a versão"). Consumidores devem ignorar campos que não
  conhecem.
- A igualdade byte a byte da exportação (SC-002 da etapa 4) continua: o
  `plan_id` é função só do conteúdo do plano.
- Um recorte exportado **antes** desta mudança não tem `plan_id`. Esta etapa o
  recusa (`ErrSliceFileInvalid`, código 27, com a orientação de gerá-lo de
  novo); `geodata slice` da etapa 4 não lê recortes, então nada mais é afetado.
- `geodata slice` (etapa 4) imprime o mesmo resumo; a identificação não aparece
  nele.

## O que muda no código da etapa 4

- `GeoSliceService.Generate` grava `plan.ID()` em `GeoSlice.PlanID`.
- `zipfile.GeoSliceExporter` escreve `plan_id` no manifesto (uma linha a mais,
  entre `format_version` e `area`).
- Os testes do exportador que comparam o manifesto passam a esperar a linha.
