# Contrato de CLI: flags de aparência em `render frame`, `render all` e `fly`

**Feature**: `008-frame-appearance` | **Data**: 2026-09-28

Cinco flags novas, com o mesmo nome, o mesmo formato e o mesmo padrão em
`sobrevoo render frame`, `sobrevoo render all` e `sobrevoo fly` (FR-002). Os
contratos existentes desses três comandos
(`specs/005-frame-rendering/contracts/cli.md`,
`specs/007-full-flight-pipeline/contracts/cli.md`) permanecem válidos sem
nenhuma outra mudança — este arquivo documenta só o que esta etapa acrescenta.

## As cinco flags

| Flag | Formato | Padrão (hoje) | Erro se malformada | Erro se fora do intervalo |
|---|---|---|---|---|
| `--trail-color` | `#RRGGBB` (hex, maiúsculo ou minúsculo) | `#FFB000` | `ErrInvalidColor` (código `52`) | — |
| `--trail-width` | número decimal, proporção da altura do quadro | `0.005` | erro de uso (código `2`) | `ErrInvalidTrailWidth` (código `53`), intervalo `0.0005`–`0.05` |
| `--marker-color` | `#RRGGBB` | `#E5252A` | `ErrInvalidColor` (código `52`) | — |
| `--marker-radius` | número decimal, proporção da altura do quadro | `0.012` | erro de uso (código `2`) | `ErrInvalidMarkerRadius` (código `54`), intervalo `0.001`–`0.1` |
| `--background-color` | `#RRGGBB` | `#20262E` | `ErrInvalidColor` (código `52`) | — |

Nenhuma combinação de cores é recusada por si (inclusive coincidir com
`NoMapColors`/`NoElevationColors` — ver Casos Extremos do `spec.md`). Todas as
cinco são opcionais; quem não informa nenhuma obtém exatamente a aparência de
hoje (FR-003), pixel a pixel.

## `sobrevoo render frame`

```text
sobrevoo render frame <arquivo-de-plano> <arquivo-de-recorte> --number <n> --output <arquivo.png> [--resolution <LxA>] [--trail-color <#RRGGBB>] [--trail-width <proporção>] [--marker-color <#RRGGBB>] [--marker-radius <proporção>] [--background-color <#RRGGBB>] [--overwrite]
```

O quadro gravado, com uma aparência escolhida, é **byte a byte igual** ao
mesmo quadro de `render all`/`fly` desenhado com a mesma aparência (SC-005). O
arquivo continua sem pertencer a nenhum conjunto de quadros.

## `sobrevoo render all`

```text
sobrevoo render all <arquivo-de-plano> <arquivo-de-recorte> --output <diretório> [--resolution <LxA>] [--trail-color <#RRGGBB>] [--trail-width <proporção>] [--marker-color <#RRGGBB>] [--marker-radius <proporção>] [--background-color <#RRGGBB>] [--overwrite]
```

A aparência passa a fazer parte do que identifica o conjunto de quadros do
diretório, junto com o plano, o recorte e a resolução (FR-007). Retomar um
desenho interrompido com uma aparência diferente da já presente no diretório
recusa com `ErrFrameSetConflict` (código `37`, sem mudança — o mesmo erro que
já existe hoje para outro plano/recorte/resolução), a menos que `--overwrite`
seja informado.

## `sobrevoo fly`

```text
sobrevoo fly <trajeto> --output voo.mp4 [--duration] [--fps] [--distance] [--tilt] [--aspect] [--resolution] [--trail-color <#RRGGBB>] [--trail-width <proporção>] [--marker-color <#RRGGBB>] [--marker-radius <proporção>] [--background-color <#RRGGBB>] [--quality] [--keep <diretório>] [--overwrite]
```

A validação das cinco flags acontece no mesmo ponto em que `--resolution` e
`--aspect` já são validados hoje — antes de qualquer etapa interna começar
(mesma regra de recusa cedo da etapa 7). Com `--keep <diretório>` apontando
para intermediários de uma execução anterior:

- **Mesma aparência**: quadros e vídeo reaproveitados, como hoje.
- **Aparência diferente**: plano e recorte reaproveitados (não dependem de
  aparência); quadros e vídeo refeitos (FR-009) — sem `--overwrite`, a
  primeira tentativa de gravar um quadro recusa com `ErrFrameSetConflict`
  (código `37`), como qualquer outro conjunto de quadros diferente; com
  `--overwrite`, refaz.

## Exit codes novos

| Código | Erro sentinela | Quando |
|---|---|---|
| `52` | `ErrInvalidColor` | `--trail-color`, `--marker-color` ou `--background-color` não é `#RRGGBB`. |
| `53` | `ErrInvalidTrailWidth` | `--trail-width` fora de `0.0005`–`0.05`. |
| `54` | `ErrInvalidMarkerRadius` | `--marker-radius` fora de `0.001`–`0.1`. |

Nenhum código existente muda de significado; `52`–`54` são os três primeiros
livres depois do `51` que a etapa 7 (`fly`) introduziu.
