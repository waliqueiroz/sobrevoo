# Contrato de CLI: `--simplification`/`--smoothing` em `plan` e `fly`

**Feature**: `013-treatment-level-flags` | **Data**: 2026-10-03

Duas flags novas, com o mesmo nome, os mesmos valores aceitos e o mesmo
padrão em `sobrevoo plan` e `sobrevoo fly` (FR-001, FR-002) — exatamente as
duas que `sobrevoo inspect` já aceita
(`specs/001-gps-track-processing/contracts/cli.md`). Os contratos
existentes desses dois comandos
(`specs/003-camera-path-planning/contracts/cli.md`,
`specs/007-full-flight-pipeline/contracts/cli.md`) permanecem válidos sem
nenhuma outra mudança — este arquivo documenta só o que esta etapa
acrescenta.

## As duas flags

| Flag | Valores aceitos | Padrão | Erro se fora do conjunto |
|---|---|---|---|
| `--simplification` | `low`, `medium`, `high` | o mesmo `Config.DefaultLevel` que `inspect` já usa (hoje, `medium`) | erro de uso da CLI, código `2` |
| `--smoothing` | `low`, `medium`, `high` | idem | erro de uso da CLI, código `2` |

As duas são escolhidas de forma independente (FR-009): pedir uma sem a
outra aplica o padrão só à que foi omitida. Sem nenhuma das duas, o
resultado é idêntico ao que `plan`/`fly` já produziam antes desta etapa
(FR-004, SC-002). Informar explicitamente o próprio valor padrão produz o
mesmo plano — e a mesma identidade de plano (ver abaixo) — que omitir a
flag.

## `sobrevoo plan`

```text
sobrevoo plan <arquivo-de-trajeto> [--duration <segundos>] [--fps <n>]
              [--distance low|medium|high] [--tilt low|medium|high]
              [--simplification low|medium|high] [--smoothing low|medium|high]
              [--aspect <L:A>] [--export <caminho>] [--overwrite]
```

O trajeto tratado usado para planejar a câmera é, para os mesmos valores de
`--simplification`/`--smoothing`, exatamente o mesmo traçado que `sobrevoo
inspect <mesmo-trajeto> --simplification=X --smoothing=Y` já mostra
(SC-001) — a duração automática, o caminho da câmera e todas as distâncias
do resumo refletem esse traçado.

## `sobrevoo fly`

```text
sobrevoo fly <trajeto> --output voo.mp4 [--duration] [--fps] [--distance]
             [--tilt] [--simplification] [--smoothing] [--aspect]
             [--resolution] [...aparência...] [...sobreposição...]
             [...seleção de fonte...] [--quality] [--keep <diretório>] [--overwrite]
```

A validação das duas flags acontece no mesmo ponto em que `--distance`/
`--tilt`/`--aspect` já são validadas hoje — antes de qualquer etapa interna
começar (mesma regra de recusa cedo da etapa 7, FR-003). Com `--keep
<diretório>` apontando para intermediários de uma execução anterior:

- **Mesmos níveis efetivos** (informados ou omitidos, desde que resultem no
  mesmo nível): plano, recorte e quadros reaproveitados, como hoje
  (FR-008, Cenário de Aceitação 2 da História 3).
- **Simplificação ou suavização diferente**: plano, recorte e quadros são
  **recalculados** — ao contrário de aparência/sobreposição (etapas 8/9),
  onde só os quadros e o vídeo são refeitos, aqui o plano em si muda (os
  dois níveis afetam a geometria do trajeto tratado, logo o caminho da
  câmera), então nada da execução anterior é reaproveitado (FR-008, Cenário
  de Aceitação 1 da História 3).
- **`--keep` de antes desta etapa, sem níveis registrados**: lido como se
  os dois níveis tivessem sido `medium` (a Clarification de `spec.md`) —
  reaproveita se a execução atual também não pede nada diferente de
  `medium`; recalcula caso contrário, exatamente como qualquer outra
  mudança de nível entre execuções.

## Sem exit codes novos

Um valor fora de `low`/`medium`/`high`, em qualquer uma das duas flags, é
um erro de uso da CLI (`newUsageError`, código `2`) — a mesma categoria que
`--distance`/`--tilt` já usam hoje. Nenhum sentinela de erro do domínio é
introduzido; `exit_code.go` não ganha nenhuma entrada nova (SC-004).
