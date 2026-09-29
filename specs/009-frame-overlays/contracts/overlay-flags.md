# Contrato de CLI: flags de sobreposição em `render frame`, `render all` e `fly`

**Feature**: `009-frame-overlays` | **Data**: 2026-09-28

Duas flags novas, com o mesmo nome, o mesmo formato e o mesmo padrão em
`sobrevoo render frame`, `sobrevoo render all` e `sobrevoo fly`, seguindo o
mesmo padrão que as cinco flags de aparência já estabeleceram
(`specs/008-frame-appearance/contracts/appearance-flags.md`) — um único
parser compartilhado, `parseOverlay` (`internal/infra/inbound/cli/
overlay.go`), ao lado de `parseAppearance`. Os contratos existentes desses
três comandos permanecem válidos sem nenhuma outra mudança — este arquivo
documenta só o que esta etapa acrescenta.

## As duas flags

| Flag | Formato | Padrão (hoje) | Erro se malformada |
|---|---|---|---|
| `--overlays` | `true`/`false` | `true` (ligadas) | erro de uso (código `2`) |
| `--overlay-blocks` | lista separada por vírgula de `distance`, `elevation`, `time`, `profile` | os quatro | `ErrInvalidOverlayBlock` (código `55`) para um nome desconhecido |

`--overlay-blocks` só importa quando `--overlays` não é `false`
(FR-003/FR-004 do `spec.md`): pedir `--overlays=false --overlay-blocks=time`
não é erro — as sobreposições ficam desligadas por inteiro, e a lista de
blocos é ignorada. Informar `--overlay-blocks` sem repetir todos os quatro
nomes desliga só os blocos que faltam na lista (ex.:
`--overlay-blocks=distance,time` desenha só distância e tempo decorrido,
sem elevação nem perfil).

## `sobrevoo render frame`

```text
sobrevoo render frame <arquivo-de-plano> <arquivo-de-recorte> --number <n> --output <arquivo.png> [--resolution <LxA>] [flags de aparência] [--overlays <true|false>] [--overlay-blocks <lista>] [--overwrite]
```

O quadro gravado, com uma configuração de sobreposição escolhida, é **byte a
byte igual** ao mesmo quadro de `render all`/`fly` desenhado com a mesma
configuração (FR-018, mesma garantia que a aparência já tem).

## `sobrevoo render all`

```text
sobrevoo render all <arquivo-de-plano> <arquivo-de-recorte> --output <diretório> [--resolution <LxA>] [flags de aparência] [--overlays <true|false>] [--overlay-blocks <lista>] [--overwrite]
```

A configuração de sobreposição passa a fazer parte do que identifica o
conjunto de quadros do diretório, junto com o plano, o recorte, a resolução
e a aparência (FR-011). Retomar um desenho interrompido com uma
configuração diferente da já presente no diretório recusa com
`ErrFrameSetConflict` (código `37`, sem mudança), a menos que `--overwrite`
seja informado.

## `sobrevoo fly`

```text
sobrevoo fly <trajeto> --output voo.mp4 [flags de plano] [--resolution] [flags de aparência] [--overlays <true|false>] [--overlay-blocks <lista>] [--quality] [--keep <diretório>] [--overwrite]
```

A validação das duas flags acontece no mesmo ponto em que as flags de
aparência já são validadas hoje — antes de qualquer etapa interna começar.
Com `--keep <diretório>`:

- **Mesma configuração de sobreposição**: quadros e vídeo reaproveitados.
- **Configuração diferente**: plano e recorte reaproveitados (não dependem
  de sobreposição); quadros e vídeo refeitos — sem `--overwrite`, recusa com
  `ErrFrameSetConflict` (código `37`); com `--overwrite`, refaz. Mesma regra
  que a etapa 8 já estabeleceu para aparência.

## Exit code novo

| Código | Erro sentinela | Quando |
|---|---|---|
| `55` | `ErrInvalidOverlayBlock` | `--overlay-blocks` contém um nome que não é `distance`, `elevation`, `time` ou `profile`. |

Nenhum código existente muda de significado; `55` é o primeiro livre depois
do `54` que a etapa 8 introduziu.

## O plano precisa ser da versão 2

As três flags acima só fazem sentido com um plano que já carregue o
instante real da atividade e a elevação do trajeto por quadro
(`contracts/plan-file-v2.md`). Um plano de versão anterior é recusado — o
mesmo `ErrPlanFormatVersionUnsupported` (código `18`, já existente) que hoje
recusa qualquer versão que o leitor não conhece — antes de `--overlays`/
`--overlay-blocks` chegarem a importar.
