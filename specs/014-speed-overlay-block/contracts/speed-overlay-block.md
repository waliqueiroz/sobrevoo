# Contrato de CLI: o bloco `speed` em `--overlay-blocks`

**Feature**: `014-speed-overlay-block` | **Data**: 2026-10-04

Estende `specs/009-frame-overlays/contracts/overlay-flags.md` (a referência
completa das duas flags de sobreposição). Este arquivo documenta só o que
esta etapa acrescenta: um quinto nome aceito por `--overlay-blocks`, em
`sobrevoo render frame`, `sobrevoo render all` e `sobrevoo fly` — o mesmo
`parseOverlay` compartilhado, sem nenhuma flag nova.

## O bloco novo

| Nome | Flag | No padrão? | Disponível quando |
|---|---|---|---|
| `speed` | `--overlay-blocks` (junto de `distance`, `elevation`, `time`, `profile`) | **Não** — só aparece se pedido explicitamente por nome | O plano tem `summary.time_reference == "clock"` (o mesmo critério que já decide se o bloco `time` aparece) |

Ligar as sobreposições sem informar `--overlay-blocks`
(`--overlays=true`, o padrão) continua mostrando exatamente os quatro
blocos de sempre — `speed` nunca aparece por conta própria. Só
`--overlay-blocks` informado com `speed` na lista (por exemplo,
`--overlay-blocks distance,speed` ou `--overlay-blocks speed`) desenha o
bloco novo; como já acontece com os quatro blocos de hoje, informar
`--overlay-blocks` substitui a lista inteira — não a soma com o padrão.

Pedir `speed` para um plano sem `time_reference == "clock"` é aceito sem
erro; o bloco simplesmente não é desenhado naquele voo (mesmo tratamento
que `time` já recebe hoje nesse caso).

## Texto exibido

`VEL <valor> km/h`, uma casa decimal, convertido de metros por segundo
(`frames[].marker.speed_mps`) — a mesma convenção de rótulo em português
que `distanceBlockText`/`elevationBlockText`/`timeBlockText` já seguem
(etapa 12), com a abreviação de unidade (`km/h`) mantida como está,
seguindo a mesma regra que já vale para `km`/`m`.

## Painel compartilhado

O painel do bloco `speed`, quando desenhado, participa da mesma largura
compartilhada que os painéis de `distance`/`elevation`/`time` já usam
(`stablePanelWidth`, etapas 11/12) — nenhuma mudança na mecânica de
painéis; só mais um candidato ao cálculo do mais largo presente no plano.

## Nenhum exit code novo

`ErrInvalidOverlayBlock` (código `55`, inalterado) passa a listar `speed`
entre os nomes aceitos na mensagem de erro
(`"expected one of distance, elevation, time, profile, speed"`); nenhum
código de saída muda de significado.

## O plano precisa ser da versão 3

O campo que alimenta este bloco (`frames[].marker.speed_mps`) só existe a
partir de `format_version: 3` (`contracts/plan-file-v3.md`). Um plano de
versão anterior é recusado — `ErrPlanFormatVersionUnsupported` (código
`18`, já existente) — antes de `--overlays`/`--overlay-blocks` chegarem a
importar, exatamente como já acontecia para a versão 2 em
`specs/009-frame-overlays/contracts/overlay-flags.md`.
