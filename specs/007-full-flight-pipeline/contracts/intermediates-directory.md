# Contrato: o diretório de `--keep`

**Feature**: `007-full-flight-pipeline` | **Data**: 2026-09-27

O que `sobrevoo fly ... --keep <diretório>` grava nesse diretório, e a regra
que decide, numa execução seguinte, o que ele reaproveita em vez de refazer
(FR-009, FR-010, FR-010a, FR-010b). Sem `--keep`, nada deste contrato se
aplica: o plano e o recorte nunca tocam disco, e os quadros vivem num
diretório temporário do sistema, sempre removido ao final.

## Layout

```text
<diretório de --keep>/
├── plan.json     # o mesmo formato de "plan --export" (specs/003-camera-path-planning/contracts/plan-file.md)
├── slice.zip     # o mesmo formato de "geodata slice --export" (specs/004-geo-data-slice/contracts/slice-file.md)
└── frames/       # o mesmo formato de "render all --output" (specs/005-frame-rendering/contracts/frame-files.md)
```

Três nomes fixos, não escolhidos pelo usuário — só o diretório-pai é dele
(`research.md` item 10). Os três arquivos são exatamente os que os comandos
individuais já produzem: abrem, inspecionam ou reaproveitam com as mesmas
ferramentas e os mesmos comandos de sempre (`sobrevoo plan` não precisa gerar
esse `plan.json` de novo para inspecioná-lo — é um JSON comum; `unzip -l
slice.zip` funciona; `frames/` é um diretório de PNG comuns).

O diretório-pai é criado se não existir (a pasta-mãe dele precisa existir),
mesma regra de `render all --output`. Um `<diretório>` que já existir e não
puder ser usado para algum dos três arquivos (por exemplo, é na verdade um
arquivo, ou sem permissão de escrita) produz o mesmo erro de destino inválido
que o comando daquele arquivo já daria isoladamente
(`ErrPlanDestinationInvalid`, `ErrSliceDestinationInvalid` ou
`ErrFrameDestinationInvalid`) — nenhuma checagem nova do diretório como um
todo.

## Regra de reaproveitamento

Para cada um dos três, na ordem em que `fly` os produz:

| Arquivo | Ainda vale quando | Não vale (refeito) quando |
|---|---|---|
| `plan.json` | existe, é um plano lido com sucesso (`ErrPlanFileInvalid`/`ErrPlanFormatVersionUnsupported` não se aplicam) e sua identificação (`CameraPlan.ID()`) é igual à do plano recém-calculado a partir do trajeto e dos valores desta execução | ausente, ilegível, ou de identificação diferente |
| `slice.zip` | existe, é um recorte lido com sucesso e sua identificação de plano (`GeoSlice.PlanID`) bate com a do plano desta execução (`GeoSlice.EnsureMatches`) | ausente, ilegível, ou de outro plano |
| `frames/` | cada quadro é reaproveitado individualmente, pela mesma regra de conjunto que `render all` já usa (`FrameMark.SetID`: plano + recorte + resolução + versão do desenho) — quadros do mesmo conjunto ficam; os que faltam são desenhados | quadros de outro conjunto: recusados por padrão (`ErrFrameSetConflict`), substituídos com `--overwrite` |

`plan.json` e `slice.zip` são reaproveitados **por inteiro** (o arquivo vale
ou não vale) — diferente de `frames/`, onde cada quadro é avaliado
individualmente, porque é exatamente assim que `FrameDirectory.Plan` (dentro
de `FrameService.DrawFrames`) já funciona. Quando `plan.json` não vale, ele é
sobrescrito só com `--overwrite` (senão, `ErrPlanDestinationExists`); mesma
regra para `slice.zip` (`ErrSliceDestinationExists`) — a mesma flag
`--overwrite` do comando (`research.md` item 11).

Mudar um valor que só afeta uma etapa posterior — por exemplo `--quality`,
que só a montagem do vídeo usa — deixa `plan.json` e `slice.zip` (e os
quadros, que não dependem da qualidade) intocados: só a montagem do vídeo é
refeita. Mudar `--duration`/`--fps`/`--distance`/`--tilt`/`--aspect` muda a
identificação do plano, o que também invalida o recorte (sua identificação de
plano deixa de bater) e os quadros (o conjunto muda) — tudo a partir dali é
refeito. Mudar `--resolution` deixa plano e recorte intocados, mas muda o
conjunto dos quadros — eles são refeitos; a montagem do vídeo também, porque
depende dos quadros.

## O que nunca é tocado

Arquivos dentro de `<diretório de --keep>` que não sejam `plan.json`,
`slice.zip` ou algo dentro de `frames/` — por exemplo, anotações do usuário —
nunca são lidos, alterados ou removidos.
