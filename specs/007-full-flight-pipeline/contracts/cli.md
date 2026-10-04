# Contrato de CLI: `sobrevoo fly`

**Feature**: `007-full-flight-pipeline` | **Data**: 2026-09-27

Um comando novo, de topo, que expõe `FlightService.Fly` (ver `data-model.md`)
através de `internal/infra/inbound/cli`. Não há API HTTP nem GUI (fora de
escopo). Os contratos de `inspect` (etapa 1), `geodata …` (etapas 2 e 4),
`plan` (etapa 3), `render …` (etapa 5) e `video` (etapa 6) permanecem
inalterados — `fly` não muda nenhum deles, só os chama pelos mesmos serviços
de aplicação. O layout do diretório de `--keep` e a regra de reaproveitamento
têm contrato próprio em
[`intermediates-directory.md`](./intermediates-directory.md).

## `sobrevoo fly`

```text
sobrevoo fly <arquivo-de-trajeto> --output <vídeo.mp4>
             [--duration <segundos>] [--fps <n>]
             [--distance low|medium|high] [--tilt low|medium|high]
             [--aspect <L:A>] [--resolution <LxA>] [--quality low|medium|high]
             [--keep <diretório>] [--overwrite]
```

- `<arquivo-de-trajeto>` (posicional, obrigatório): o mesmo arquivo que
  `plan` aceita (GPX). Passa pelo mesmo tratamento completo da etapa 1.
- `--output` (obrigatória): o vídeo a gravar. **Deve terminar em `.mp4`**,
  mesma regra e mesmo erro de uso (código `2`) que `video --output` já usa.
- `--duration`, `--fps`, `--distance`, `--tilt`, `--aspect`: exatamente as
  flags de `plan`, com os mesmos nomes, valores aceitos e padrões
  (`Config.PlanDefaults`) — ver `specs/003-camera-path-planning/contracts/cli.md`.
- `--resolution`: exatamente a flag de `render all`, mesmo formato, mesmos
  limites e mesmo padrão (`Config.RenderDefaults`) — ver
  `specs/005-frame-rendering/contracts/cli.md`. O mesmo aviso de proporção
  (`warnIfFramingCut`) sai em `stderr` quando a resolução é mais estreita que
  o `--aspect` efetivo.
- `--quality`: exatamente a flag de `video`, mesmos valores e mesmo padrão
  (`Config.VideoDefaults`) — ver `specs/006-video-assembly/contracts/cli.md`.
- `--keep` (opcional): um diretório onde o plano, o recorte e os quadros
  ficam guardados, e de onde uma execução futura com o mesmo trajeto e os
  mesmos valores os reaproveita (ver `intermediates-directory.md`). Sem
  `--keep`, nada disso toca o disco do usuário: o plano e o recorte ficam só
  em memória, e os quadros vivem num diretório temporário do sistema, sempre
  removido ao final — inclusive em caso de falha ou interrupção.
- `--overwrite` (opcional): permite substituir o destino do vídeo, se já
  existir, e qualquer intermediário guardado que seja de outro trajeto ou de
  outros valores (mesma flag para as duas coisas — `research.md` item 11).

### Saída (sucesso)

Em `stderr`, ao entrar em cada uma das cinco etapas — sempre antes do
trabalho dela, nunca depois —, uma linha:

```text
Stage 1/5: treating the track
Stage 2/5: planning the camera
Stage 3/5: slicing the geo data
Stage 4/5: drawing the frames
Drawing frame 412/1260 (32.7%), elapsed 00:07:41
Stage 5/5: encoding the video
Encoding frame 412/1350 (30.5%), elapsed 00:00:41
```

A linha de progresso de quadro/codificação é exatamente a mesma que `render
all`/`video` já escrevem (num terminal, uma linha reescrita; sem terminal,
uma nova a cada 10 quadros/5 segundos e a última, escrita uma vez só). Como
cada etapa é anunciada ao começar, um erro sai sempre depois do anúncio da
etapa em que aconteceu — uma recusa do recorte (`19`, `20`) vem depois de
`Stage 3/5`, não de `Stage 2/5`. O tratamento do trajeto e o planejamento da
câmera acontecem numa só chamada (`CameraPlanService.Generate`), que avisa,
por um callback, quando o trajeto acabou de ser tratado: é nesse momento que
`Stage 2/5` é anunciado.

Quando o plano ou o recorte são reaproveitados, isso só se sabe depois de a
etapa começar — o plano é sempre recalculado para ser comparado com o
guardado —, então o aviso sai numa linha própria, logo abaixo do anúncio:

```text
Stage 2/5: planning the camera
  unchanged since the last run under --keep, reusing plan.json
Stage 3/5: slicing the geo data
  unchanged, reusing slice.zip
```

Ao final, em `stdout`, o resumo — os mesmos rótulos de `render
all`/`video`, na mesma ordem, com uma linha de reaproveitamento entre os dois
blocos quando aplicável:

```text
Frames: 1260 requested, 1260 drawn, 0 kept (already in the destination)
Resolution: 1080x1920
Time: 00:31:07
Holes (in the frames drawn now): none
Video written to /tmp/voo.mp4
Frames: 1260
Duration: 00:00:42.000
Resolution: 1080x1920
Frame rate: 30 fps
Quality: medium
Size: 17.1 MiB
Encoder: ffmpeg 7.1 (libx264)
Time: 00:00:53
Total time: 00:31:07 (plan: generated, slice: generated, frames: drawn, video: encoded)
```

Do resumo de `render all`, o de `fly` só traz a linha `Destination` com
`--keep` (`Destination: <keep>/frames (frame_000000.png to ...)`): sem
`--keep`, os quadros viveram num diretório temporário que já não existe
quando o resumo é lido. A última linha (`Total time`)
é própria de `fly`: soma o tempo das cinco etapas e diz, entre parênteses,
"generated"/"reused" para o plano e o recorte e "drawn"/"encoded" para os
quadros e o vídeo (sempre um dos dois — nunca "reused" para eles, que a spec
não pede — FR-010a só cobre plano, recorte e quadros; o vídeo é sempre
produzido).

**Código de saída**: `0`.

### Interrupção (`Ctrl+C` ou `SIGTERM`)

A execução encerra de forma ordenada assim que possível — imediatamente
durante o desenho dos quadros ou a codificação (que já respondem à
interrupção etapa a etapa, como `render all`/`video`); no próximo ponto de
checagem, ao terminar, para o tratamento do trajeto, o planejamento da câmera
ou o recorte dos dados (que não têm ponto de cancelamento próprio —
`research.md` item 8 e `plan.md` R1). Diz o que já havia sido concluído e sai
com o código próprio do comando único, nunca o de uma etapa:

```text
Interrupted after stage 3/5 (geo data slicing); completed: track processing, camera planning, geo data slicing
Frames: 0 of 1260 drawn
```

Quando a interrupção acontece durante o desenho dos quadros ou a codificação,
o resumo parcial de quadros/vídeo aparece também, como `render all`/`video`
já mostram na interrupção deles. Nenhum vídeo parcial fica; sem `--keep`,
nada fica em disco algum — o diretório temporário de quadros, mesmo parcial,
é removido antes de o comando sair.

**Código de saída**: `51`.

## Ordem das verificações

Uma recusa nunca cria nem altera arquivo do usuário. Da mais barata à mais
cara: uso (`--output`, extensão, `--quality`, `--distance`/`--tilt`, `--aspect`,
`--resolution`: `2`) → o destino do vídeo (`47`, `48`) e o codificador (`46`) —
antes de qualquer etapa → leitura/tratamento do trajeto (`1`, `3`, e as de
`plan`: `10`-`14`, `39`) → planejamento da câmera (`15`, `16`, ou nada, se
reaproveitado) → recorte dos dados: a cobertura (`19`) e as demais de
`geodata slice` (`20`-`26`), ou nada, se reaproveitado (`23`, `24` se a
gravação falhar) → desenho dos quadros (as de `render all`: `31`-`38`) →
montagem do vídeo (as de `video`: `40`-`45`, `49`, `50`) → interrupção em
qualquer ponto (`51`).

## Saída (erro)

Toda falha de negócio usa **o mesmo erro sentinela, a mesma mensagem e o
mesmo código de saída** que o comando individual da etapa em que ocorreu já
usa (FR-008) — as tabelas de
`specs/003-camera-path-planning/contracts/cli.md`,
`specs/004-geo-data-slice/contracts/cli.md`,
`specs/005-frame-rendering/contracts/cli.md` e
`specs/006-video-assembly/contracts/cli.md` continuam valendo sem nenhuma
mudança. `fly` acrescenta só uma linha à tabela de códigos:

| Cenário | Erro do domínio | Código |
|---|---|---|
| Execução interrompida pelo usuário, em qualquer etapa | `domain.ErrFlightInterrupted` | `51` |

Cada mensagem de erro de etapa sai exatamente como o comando individual a
escreveria, precedida por uma linha que diz em qual etapa o percurso parou:

```text
Stage 3/5 (geo data slicing) failed: the registered geo data does not cover the whole area the plan needs: <resto da mensagem de ErrAreaNotCovered, igual a "geodata slice">
```

## Exemplos

```bash
# Do trajeto ao vídeo, num só comando, com os padrões de cada etapa
sobrevoo fly pedalada.gpx --output pedalada.mp4

# Ajustando o que já era ajustável em cada etapa — mesmos nomes de sempre
sobrevoo fly pedalada.gpx --output pedalada.mp4 --duration 60 --distance high --aspect 16:9 --resolution 1920x1080 --quality high

# Guardando os intermediários para inspecionar depois
sobrevoo fly pedalada.gpx --output pedalada.mp4 --keep ./intermediarios/

# Rodando de novo com o mesmo trajeto e os mesmos valores: reaproveita
# plan.json, slice.zip e os quadros já prontos; só refaz o vídeo
sobrevoo fly pedalada.gpx --output pedalada.mp4 --keep ./intermediarios/

# Mudando só a qualidade: reaproveita plano, recorte e quadros; recodifica
sobrevoo fly pedalada.gpx --output pedalada-hq.mp4 --keep ./intermediarios/ --quality high

# Equivalente, na mão, com os seis comandos existentes (mesmo resultado, byte a byte)
sobrevoo plan pedalada.gpx --export ./intermediarios/plan.json
sobrevoo geodata slice ./intermediarios/plan.json --export ./intermediarios/slice.zip
sobrevoo render all ./intermediarios/plan.json ./intermediarios/slice.zip --output ./intermediarios/frames/
sobrevoo video ./intermediarios/plan.json ./intermediarios/frames/ --output pedalada.mp4
```
