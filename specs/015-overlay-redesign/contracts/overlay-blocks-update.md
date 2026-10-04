# Mudança no Contrato de CLI: `--overlay-blocks` ganha `gain` e novo padrão

**Feature**: `015-overlay-redesign` | **Data**: 2026-10-04

Esta etapa não acrescenta nem renomeia nenhuma flag — `--overlays` e
`--overlay-blocks` continuam exatamente como o contrato da etapa 9 as
define (`specs/009-frame-overlays/contracts/overlay-flags.md`), estendido
pela etapa 14 (`specs/014-speed-overlay-block/contracts/
speed-overlay-block.md`). Este arquivo documenta só o que muda: um sexto
nome aceito e o conjunto padrão.

## O que muda

| Antes (etapas 9/14) | Depois (esta etapa) |
|---|---|
| Nomes aceitos em `--overlay-blocks`: `distance`, `elevation`, `time`, `profile`, `speed` | `distance`, `elevation`, `gain`, `time`, `profile`, `speed` |
| Padrão (sem `--overlay-blocks`): `distance`, `elevation`, `time`, `profile` | `distance`, `elevation`, `speed`, `profile` |
| `elevation` mostra altitude **e** ganho acumulado na mesma linha | `elevation` mostra só a altitude; o ganho vira o bloco `gain`, à parte |
| Mensagem de erro de nome inválido cita 5 nomes | Cita os 6 nomes |

`gain`, como `speed` já era, **nasce fora do padrão** — só aparece quando
pedido explicitamente por nome (FR-011). O gate de disponibilidade de
`gain` é o mesmo de `elevation` (plano com elevação disponível,
`plan.ElevationAvailable`) — um trajeto sem elevação nunca mostra nenhum
dos dois, pedidos ou não, sem erro (o mesmo comportamento que `elevation`
já tem hoje).

## `sobrevoo render frame` / `render all` / `fly`

As três assinaturas de comando não mudam
(`specs/009-frame-overlays/contracts/overlay-flags.md`). Um pedido de
`--overlay-blocks elevation,gain` desenha os dois como blocos distintos;
`--overlay-blocks gain` sozinho desenha só o ganho, sem a altitude.

## Exit code

Nenhum novo. `ErrInvalidOverlayBlock` (código `55`, já existente desde a
etapa 9) continua sendo a recusa para qualquer nome fora dos seis
aceitos — só a lista de nomes citada na mensagem cresce.

## Reaproveitamento de quadros (`render all --overwrite`, `fly --keep`)

Sem mudança de mecanismo: `OverlayConfig` (agora com o bit `Gain`) continua
fazendo parte do que identifica um conjunto de quadros
(`FrameSetID`/`OverlayConfig.Fingerprint`). Pedir `gain` numa execução que
retoma um diretório desenhado sem ele é tratado exatamente como já era
pedir `speed` numa execução que retoma um diretório desenhado antes da
etapa 14 — `ErrFrameSetConflict` sem `--overwrite`; redesenho com ele.
