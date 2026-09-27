# Contrato de CLI: `sobrevoo video`

**Feature**: `006-video-assembly` | **Data**: 2026-09-27

Um comando novo, de topo, que expõe `VideoService.Assemble` (ver
`data-model.md`) através de `internal/infra/inbound/cli`. Não há API HTTP nem
GUI (fora de escopo). Os contratos de `inspect` (etapa 1), `geodata …` (etapas 2
e 4), `plan` (etapa 3) e `render …` (etapa 5) permanecem inalterados, com **uma**
exceção, na etapa 5: os quadros passam a trazer, dentro de cada imagem, a
identificação do plano
([`frame-files-change.md`](./frame-files-change.md)). Os arquivos que o comando
lê e grava têm contrato próprio: o plano, em
`specs/003-camera-path-planning/contracts/plan-file.md`; os quadros, em
`specs/005-frame-rendering/contracts/frame-files.md` (+ a mudança acima); o
vídeo, em [`video-file.md`](./video-file.md).

## `sobrevoo video`

```text
sobrevoo video <arquivo-de-plano> <diretório-de-quadros> --output <arquivo.mp4> [--quality <nível>] [--overwrite]
```

- `<arquivo-de-plano>`: plano exportado por `sobrevoo plan --export`, o **mesmo**
  de que os quadros foram desenhados. Dele vêm a quantidade de quadros, a ordem
  e a taxa de quadros do vídeo.
- `<diretório-de-quadros>`: o diretório que `sobrevoo render all --output`
  gravou. É lido, nunca alterado; arquivos que não são quadros da ferramenta são
  ignorados.
- `--output` (obrigatória): o arquivo de vídeo a gravar. **Deve terminar em
  `.mp4`** (maiúsculas ou minúsculas): o formato é sempre MP4, e outro nome seria
  enganoso (erro de uso, código `2`). A pasta precisa existir.
- `--quality` (opcional): `low`, `medium` ou `high`; padrão `medium` (de
  `Config.VideoDefaults`). Ver "Qualidade". Outro valor é erro de uso (`2`), com a
  lista dos aceitos.
- `--overwrite` (opcional): permite substituir o arquivo, se existir.

Nada é redesenhado, baixado ou lido além do plano e dos quadros (FR-001). O
vídeo não tem áudio, texto nem estatísticas. O comando precisa do programa
`ffmpeg` instalado (ver "Codificador").

### Saída (sucesso)

Durante a codificação, em `stderr`, o progresso: num terminal, uma linha reescrita,

```text
Encoding frame 412/1350 (30.5%), elapsed 00:00:41
```

sem terminal (um log), uma linha nova **no máximo a cada 5 segundos** e a última.
Ao final, em `stdout`, o resumo (rótulos e ordem estáveis):

```text
Video written to /tmp/voo.mp4
Frames: 1350
Duration: 00:00:45.000
Resolution: 1080x1920
Frame rate: 30 fps
Quality: medium
Size: 18.4 MiB
Encoder: ffmpeg 7.1 (libx264)
Time: 00:01:52
```

- `Duration` é `Frames ÷ Frame rate`, em `hh:mm:ss.mmm` (`1350 ÷ 29.97` →
  `00:00:45.045`).
- `Frame rate` é a do plano, escrita como o plano a guarda (`30`, `29.97`).
- `Size` é o do arquivo publicado, em `B`, `KiB`, `MiB` ou `GiB`, com uma casa.
- `Encoder` é o programa, a versão e o codec de vídeo usados: dele depende a
  igualdade byte a byte entre execuções (`video-file.md`).
- `Time` (tempo gasto) é informação da execução e **não** entra no vídeo.

**Código de saída**: `0`.

### Interrupção (`Ctrl+C` ou `SIGTERM`)

A codificação é encerrada (o processo do codificador termina), nenhum arquivo
parcial ou temporário fica, um arquivo anterior no destino permanece como estava
e o comando diz que foi interrompida. Não há retomada: repetir o comando
recomeça.

```text
Interrupted: 412 of 1350 frames were encoded; no video was written, run the same command again to start over
Time: 00:00:41
```

**Código de saída**: `49`.

## Qualidade

`--quality low|medium|high`. O nível troca tamanho do arquivo e tempo de
codificação por fidelidade, **sem** mudar a ordem, a duração, a resolução nem a
taxa de quadros. Um nível mais alto produz um arquivo de tamanho maior ou igual ao
de um nível mais baixo (FR-013). O que cada nível pede ao codificador está em
[`video-file.md`](./video-file.md):

| Nível | Para quê |
|---|---|
| `low` | conferir rápido o voo: arquivo menor, codificação mais rápida |
| `medium` (padrão) | publicar |
| `high` | guardar com a máxima fidelidade prática: arquivo maior, codificação mais lenta |

## Codificador

O programa **`ffmpeg`** (com o codificador de vídeo `libx264`) precisa estar
instalado e no `PATH`. A ferramenta não o traz nem o baixa. Se ele falta, ou não
tem o `libx264`, o comando recusa antes de codificar, com o que instalar
(erro `46`, ver "Mensagens").

## Ordem das verificações

Uma recusa nunca cria arquivo. Da mais barata à mais cara: uso (`--output`,
extensão, `--quality`: `2`) → leitura do plano (`4`, `17`, `18`) → o diretório de
quadros (`40`), a identificação do plano (`44`, `43`), a resolução (`42`), o
conjunto único (`43`), a numeração (`41`) e a integridade (`45`) → o destino (`47`, `48`) → o codificador
(`46`) → a codificação (`50`, `49`). Quando há mais de um problema, a ferramenta
diz o primeiro da lista; resolvido, o próximo aparece.

## Saída (erro)

Mensagem em `stderr`, sem arquivo criado ou alterado no destino.

| Cenário | Erro do domínio | Código |
|---|---|---|
| Uso inválido: argumento ou flag obrigatória faltando, `--quality` desconhecido, `--output` que não termina em `.mp4` | erro de uso da CLI | `2` |
| Plano inexistente / ilegível (E/S) | erro genérico | `4` |
| Arquivo não é um plano / campo ausente / truncado / incoerente | `domain.ErrPlanFileInvalid` | `17` |
| Plano de `format_version` desconhecida | `domain.ErrPlanFormatVersionUnsupported` | `18` |
| Diretório de quadros inexistente, que não é diretório, ilegível ou sem nenhum quadro da ferramenta | `domain.ErrFrameDirectoryInvalid` | `40` |
| Quadros que faltam, sobram ou se repetem | `domain.ErrFrameSequenceInvalid` | `41` |
| Quadros de resoluções diferentes, ou de dimensão ímpar | `domain.ErrFrameResolutionInvalid` | `42` |
| Quadros de outro plano, ou de mais de um conjunto misturados | `domain.ErrFramesDoNotMatchPlan` | `43` |
| Quadros sem a identificação do plano (desenhados antes da versão 2 da etapa 5) | `domain.ErrFramesWithoutPlanID` | `44` |
| Quadro que não é um PNG inteiro (truncado, ilegível) | `domain.ErrFrameFileInvalid` | `45` |
| Codificador ausente ou sem o formato necessário | `domain.ErrEncoderUnavailable` | `46` |
| Destino já existe (sem `--overwrite`) | `domain.ErrVideoDestinationExists` | `47` |
| Destino inválido (é um diretório, pasta inexistente, sem permissão) | `domain.ErrVideoDestinationInvalid` | `48` |
| Montagem interrompida pelo usuário | `domain.ErrVideoInterrupted` | `49` |
| O codificador falhou (falta de espaço, imagem corrompida por dentro, …) | `domain.ErrVideoEncodingFailed` | `50` |

### Mensagens (SC-005)

Cada uma diz **o que** está errado e **como** proceder. Números de quadro saem
em faixas (`12-15, 40`), com o total à vista; arquivos, até cinco por causa
(`, and 12 more`).

| Código | Mensagem (exemplos) |
|---|---|
| `40` | `frame directory cannot be used: /tmp/quadros does not exist` · `frame directory cannot be used: /tmp/quadros is not a directory` · `frame directory cannot be used: holds no frames drawn by this tool (3 files named like frames were ignored: they do not carry this tool's identification); draw them with "render all"` |
| `41` | `frames do not form the flight of the plan: 4 missing (12-15), 2 not in the plan (1260-1261; the plan has frames 0 to 1259); frame_000012.png was ignored: it does not carry this tool's identification; draw the missing frames with "render all"` (na ordem: faltas, excedentes, repetições — `number 3 appears more than once` —, arquivos ignorados, orientação) |
| `42` | `frame resolution cannot be used for the video: 1080x1920 is the resolution of 1340 frames, but 10 differ (frame_000100.png is 540x960, frame_000101.png is 540x960, …, and 5 more)` · `frame resolution cannot be used for the video: the frames are 1081x1921, but width and height must be even for the video` |
| `43` | `frames were not drawn from the plan: 1350 of 1350 frames carry another plan identification (plan 3fa9c1d2e4b7, frames b71e0a55c2d9)` · `frames were not drawn from the plan: the directory mixes frames of 2 different sets (another slice, resolution or drawing version); draw them again into an empty directory` |
| `44` | `frames do not say which plan they were drawn from: 1350 of 1350 frames were drawn by an earlier version of Sobrevoo; draw them again with "render all --overwrite"` |
| `45` | `frame file cannot be used: frame_000007.png is not a whole PNG image (truncated?); draw it again with "render all"` · com vários: `frame file cannot be used: 7 frames are not whole PNG images (truncated?): frame_000002.png, …, and 2 more; draw them again with "render all"` |
| `46` | `video encoder not available: "ffmpeg" was not found on the PATH; install it (macOS: brew install ffmpeg; Debian/Ubuntu: sudo apt install ffmpeg; Windows: winget install Gyan.FFmpeg), then check it with: ffmpeg -version` · `video encoder not available: ffmpeg 6.0 has no libx264 encoder; install a build that includes it (macOS: brew install ffmpeg; Debian/Ubuntu: sudo apt install ffmpeg)` |
| `47` | `video destination already exists: /tmp/voo.mp4; use --overwrite to replace it` |
| `48` | `video destination cannot be used: /tmp/nada/voo.mp4: the folder does not exist` · `video destination cannot be used: /tmp/voo.mp4 is a directory` (ou a causa do sistema) |
| `50` | `video encoding failed: ffmpeg exited with status 1: <final da saída de erro do ffmpeg, até 2 KiB>` |

Os códigos `17` e `18` usam as mensagens que a etapa 3 já define; `2`, as do
Cobra (`--quality "ultra": use one of low, medium, high`; `--output must end in
.mp4: the video is always an MP4 file`).

## Exemplos

```bash
# Fluxo completo: trajeto → plano → recorte → quadros → vídeo
sobrevoo plan pedalada.gpx --export plano.json
sobrevoo geodata slice plano.json --export recorte.zip
sobrevoo render all plano.json recorte.zip --output quadros/
sobrevoo video plano.json quadros/ --output pedalada.mp4

# Conferir rápido, sem gastar tempo com a qualidade
sobrevoo video plano.json quadros/ --output rascunho.mp4 --quality low

# Refazer o vídeo no mesmo destino, para guardar
sobrevoo video plano.json quadros/ --output pedalada.mp4 --quality high --overwrite
```
