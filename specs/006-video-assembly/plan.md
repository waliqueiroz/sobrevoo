# Plano de Implementação: Montagem do Vídeo do Voo

**Branch**: `006-video-assembly` | **Data**: 2026-09-27 | **Especificação**: [spec.md](./spec.md)

**Entrada**: Especificação de funcionalidade de `/specs/006-video-assembly/spec.md`

## Resumo

Sexta etapa do Sobrevoo: um comando novo, `video`.
`sobrevoo video <plano> <diretório-de-quadros> --output voo.mp4 [--quality
low|medium|high] [--overwrite]` junta os quadros que `render all` desenhou num
**único arquivo MP4**, na ordem e na taxa de quadros do plano, com progresso e
resumo. As entradas são o plano exportado pela etapa 3 e o diretório de quadros
da etapa 5; a saída é o vídeo. Nada é redesenhado, baixado, sobreposto com texto
ou sonorizado.

A abordagem técnica (detalhada em `research.md`):

- A codificação é feita pelo **`ffmpeg`** (com `libx264`), instalado pelo usuário e
  chamado como **processo externo** por um adapter, atrás da porta de domínio
  `VideoEncoder` (item 1). O `ffmpeg` lê os quadros direto do disco pelo padrão
  `frame_%06d.png`, com o diretório dos quadros como diretório de trabalho (item
  3), e escreve MP4 H.264 `yuv420p` BT.709, sem áudio (item 2).
- A **taxa e a duração** são exatas: `-framerate` com a taxa do plano,
  `-frames:v N`, sem quadro duplicado nem descartado (item 4). Três **níveis
  de qualidade** nomeados, `low`/`medium`/`high`, que o adapter traduz em
  `preset` e `crf` (item 5).
- O **determinismo** e a **ausência de dado do ambiente** vêm de uma linha de
  comando que fixa tudo o que o `ffmpeg` deixaria variar: `bitexact`, sem
  metadados, `x264` com threads fixas e sem a mensagem SEI, cor explícita (item
  6). A promessa vale para o mesmo `ffmpeg`, que o resumo informa.
- Os quadros são **conferidos antes de codificar** por `FrameDirectory.Verify`,
  regra de domínio: há quadros, todos trazem a identificação do plano informado,
  têm a mesma resolução (par) e o mesmo conjunto, formam a numeração de 0 a N−1
  sem falta, excesso nem repetição, e são PNG inteiros (item 7), com mensagens que
  dizem exatamente o que falta ou destoa.
- A **Clarificação de 2026-09-27**: a etapa 5 passa a gravar em cada quadro a
  **identificação do plano** (segundo bloco `tEXt`), com `RenderVersion` 2, e a
  etapa 6 a confere (item 8). É a única mudança na etapa 5.
- O **codificador é verificado** antes de começar (`Probe`: presente e com
  `libx264`) com uma mensagem que diz o que instalar (item 9). O **arquivo** é
  publicado por inteiro, com uma checagem prévia do destino e um
  `atomicfile.PublishPath` novo que entrega ao codificador o caminho de um
  temporário (item 10). O **progresso** vem do `-progress` do `ffmpeg` e a
  **interrupção** mata o processo e apaga o temporário (item 11).

A arquitetura hexagonal é preservada: o núcleo ganha entidades e funções puras
(`VideoQuality`, `VideoSummary`, `FrameMark`, `FrameDirectory.Verify`), duas
portas (`VideoEncoder`, `VideoExporter`) mais um método em `FrameRepository`
(`List`), onze erros e um serviço novo — `VideoService` (`Assemble`). Toda E/S e
todo processo externo ficam nos adapters; a CLI só traduz flags, texto, sinais e
códigos de saída.

## Contexto Técnico

**Linguagem/Versão**: Go 1.26 (sem mudança)

**Dependências Principais**: nenhuma nova no módulo Go. Só biblioteca padrão
(`os/exec`, `bufio`, `context`, `strconv`, `path/filepath`); Cobra, testify e
`go.uber.org/mock` como antes. Dependência **de execução**, fora do módulo: o
programa `ffmpeg` com `libx264`, instalado pelo usuário (não é embutido nem
baixado, Princípio V).

**Armazenamento**: nenhum estado persistente novo. Lê o plano e os quadros
(somente leitura) e grava **um** arquivo MP4 (temporário na pasta do destino,
publicado atomicamente, sem sobrescrita por padrão). O registro da etapa 2 e o
recorte da etapa 4 **não** são lidos.

**Testes**: mesmo padrão — `go test` com testify, given/when/then, um `t.Run` por
cenário, sem testes tabulares. Builders em `builddomain` (estendido:
`FrameDirectoryBuilder` com `PlanID`, `Size` e a marca); mocks gerados de
`VideoEncoder` e `VideoExporter` (`mockdomain`), regerados de `FrameRepository` e
`FrameExporter`, e de `VideoService` (`mockapplication`). O domínio é testado
sem E/S: `Verify` com listagens em memória para cada tipo de problema, na ordem
prevista, incluindo o que o adapter de hoje não produz (repetição) e as mensagens
(faixas, limites, abreviações); `VideoSummary.Duration` com taxas inteiras e
decimais; `VideoQuality`. O serviço, com as portas mockadas (ordem das chamadas,
nada criado quando uma verificação falha, interrupção). A CLI, com o serviço
mockado (flags, extensão, progresso, resumo, códigos de saída). Os adapters em
diretório temporário: `pngfile.List` e a marca do plano (com fixtures de quadro em
código), `videofile.VideoExporter` (checagem, publicação, falha, sobrescrita),
`atomicfile.PublishPath`, e `videoencoder.FFmpeg` contra um **`ffmpeg` de mentira**
(script de shell) que confere a linha de comando, o progresso, a falha, o
cancelamento e o `Probe`. **Nenhum teste automatizado exige o `ffmpeg` real**;
`quickstart.md` é o checklist manual, com ele.

**Plataforma-Alvo**: mesmo binário multiplataforma (Linux, macOS, Windows). O
`ffmpeg` existe nas três; o script de mentira dos testes do adapter exige `sh`
(os testes são pulados onde não há).

**Tipo de Projeto**: CLI (projeto único em Go, sem frontend/backend).

**Metas de Desempenho** (referência inicial, a confirmar com medição na
implementação: `research.md` item 15): a conferência dos quadros de 1350
arquivos em < 2 s (SC-005); a montagem de 1350 quadros de 1080 × 1920 no nível
`medium` em ≤ 10 min (SC-007), em computador pessoal comum; o início do
progresso em < 2 s depois da conferência.

**Restrições**: 100% offline; núcleo sem E/S e sem processo; determinismo byte a
byte para o mesmo `ffmpeg` e a mesma versão da ferramenta (`research.md` item 6);
o vídeo sem data, hora, caminho, nome de máquina ou de usuário nem qualquer dado do
ambiente; nenhum dado de região embutido; nada de quadro inteiro na memória do
Sobrevoo; largura e altura pares; só MP4, só H.264, sem áudio.

**Escala/Escopo**: uso pessoal — voos de centenas a milhares de quadros (432 000
no máximo, o teto do plano; um vídeo assim é grande e lento, mas o pedido não
falha por tamanho); quadros de até 3840 × 2160.

## Verificação da Constituição

*PORTÃO: Deve passar antes da Fase 0 de pesquisa. Reverificar após o design da Fase 1.*

Avaliada antes da pesquisa e **reavaliada após o design** (data-model, contratos e
quickstart); o resultado não mudou.

| Princípio | Avaliação | Como o design atende |
|---|---|---|
| I. Arquitetura Hexagonal | PASSA | Toda a regra — a conferência dos quadros (`FrameDirectory.Verify`), a duração (`VideoSummary.Duration`), a marca do plano, os níveis, as mensagens de faltas e destoantes — é função pura ou método de entidade em `internal/domain`, sem importar `os`, `os/exec`, `image/png` nem `internal/infra`. `VideoService` só chama portas e métodos de domínio, na ordem certa. O `ffmpeg`, o sistema de arquivos, a leitura do PNG e a configuração ficam nos adapters. `context` é biblioteca padrão pura. |
| II. Portas para Toda Dependência Externa | PASSA | O `ffmpeg` (processo externo, exemplo que a própria constituição dá) passa por `VideoEncoder`; gravar o vídeo por `VideoExporter`; listar os quadros por `FrameRepository.List`. O plano continua lido por `CameraPlanReader`. Nenhuma porta para `time.Now()` (o tempo gasto), `strconv`, `sort` ou `context`, funções puras/estado da biblioteca padrão. |
| III. Entrypoints Descartáveis | PASSA | `VideoService` recebe e devolve tipos de domínio (`CameraPlan`, `VideoRequest`, `VideoSummary`, `VideoProgress`); nada de flags, texto, terminal ou códigos de saída. `video` só traduz flags → chamada de serviço, o progresso → linhas no `stderr`, sinais → `context`, resultado/erro → texto/código. A regra "o destino termina em `.mp4`" é do entrypoint (validação de flag), não do núcleo: outro entrypoint escolheria o nome como quisesse. |
| IV. Neutralidade Geográfica | PASSA | Nenhum dado, constante ou caso especial de região: a etapa opera sobre imagens, números de quadro e uma taxa. SC-011 verifica voos equivalentes deslocados pelo planeta (180°, latitudes acima de 80°). |
| V. Funcionamento Offline | PASSA | Só lê o plano e os quadros informados; o `ffmpeg` roda local, sem rede; a ferramenta não baixa nem embute o codificador (FR-001, item 1). O relógio só mede o tempo gasto e nunca entra no vídeo. |
| VI. Testes Automatizados no Núcleo | PASSA | Domínio testado sem E/S (listagens e planos em memória); `VideoService` com as portas mockadas (`mockdomain`); nenhum teste do núcleo toca disco, rede ou processo externo. |
| VII. Erros Sentinela no Domínio | PASSA | `ErrFrameDirectoryInvalid`, `ErrFrameSequenceInvalid`, `ErrFrameResolutionInvalid`, `ErrFramesDoNotMatchPlan`, `ErrFramesWithoutPlanID`, `ErrFrameFileInvalid`, `ErrEncoderUnavailable`, `ErrVideoDestinationExists`, `ErrVideoDestinationInvalid`, `ErrVideoInterrupted`, `ErrVideoEncodingFailed` em `errors.go`; a CLI os traduz em códigos 40–50 (`contracts/cli.md`). Erros de `os`, `os/exec` e do `ffmpeg` (código de saída, texto de erro) são traduzidos nos adapters, que devolvem o sentinela com a causa; o núcleo nunca vê um deles. |
| VIII. Configuração Injetada | PASSA | `config.VideoDefaults{Quality}` e `config.FFmpegBinary` têm **tipos próprios** e não importam o domínio; o composition root (`cmd/sobrevoo/config_mapping.go`) mapeia o nível para `domain.VideoQuality` e passa o nome do programa ao adapter. Caminhos, nível, `--overwrite` e o contexto chegam ao núcleo por parâmetro; o núcleo não lê flag, ambiente, arquivo nem sinal. Os parâmetros do `libx264` são constantes do adapter (é o adapter que fala o vocabulário do `ffmpeg`). |
| IX. Organização de Portas, Service Layer e Mocks | PASSA | Sem `ports.go`: `VideoEncoder` e `VideoExporter` no topo de `video.go`, com `//go:generate` logo após `package` (as portas dos quadros continuam em `frame_set.go`, com o método novo `List`). Nomeadas pelo papel (`Encoder`, `Exporter`, como `FrameExporter`, `CameraPlanExporter`). Um serviço por recurso, sem `Execute`: `VideoService`/`videoService`/`NewVideoService` (`Assemble`). Toda regra em `internal/domain`; `application` só chama portas e métodos de domínio. Adapters: `videofile.NewVideoExporter()` (pacote de tecnologia, construtor pela porta, como `pngfile`), `videoencoder.NewFFmpeg(binary)` (o pacote leva o nome da porta, tipo e construtor só a estratégia, arquivo `ffmpeg_video_encoder.go`), `pngfile` (porta `FrameRepository` com o método novo); `atomicfile` continua utilitário. Nenhum nome exportado repete o pacote. Mocks por `//go:generate` acima de cada interface, em `mockdomain`/`mockapplication`. Receivers: `d` (`FrameDirectory`), `s` (`VideoSummary`, `videoService`), `q` (`VideoQuality`), `i` (`EncoderInfo`), `e` (`FFmpeg`, `VideoExporter`, exporters), `r` (repository). |
| X. Testes: Given/When/Then, Builders e Isolamento por Camada | PASSA | Todo teste em `t.Run("should ...")` com `// given`/`// when`/`// then`, sem tabelas; builders em `builddomain`; domínio/serviço com portas mockadas; CLI com serviço mockado (`mockapplication`); adapters com diretório temporário e o `ffmpeg` de mentira (é o adapter que toca disco e processo, então é o objeto do teste). Sem teste de ponta a ponta automatizado; `quickstart.md` cobre o manual, com o `ffmpeg` real. |
| Idioma dos Artefatos | PASSA | Artefatos do Spec Kit em português; identificadores, pacotes, arquivos, comentários e mensagens de commit em inglês; E/S em tempo de execução (flags, saída, erros) em inglês, como nas etapas anteriores. |

Nenhuma violação identificada. Rastreamento de Complexidade não se aplica.

## Estrutura do Projeto

### Documentação (desta funcionalidade)

```text
specs/006-video-assembly/
├── plan.md              # Este arquivo (saída do comando /speckit-plan)
├── research.md          # Saída da Fase 0 (comando /speckit-plan)
├── data-model.md        # Saída da Fase 1 (comando /speckit-plan)
├── quickstart.md        # Saída da Fase 1 (comando /speckit-plan)
├── contracts/
│   ├── cli.md                    # Saída da Fase 1: o comando `video`
│   ├── video-file.md             # Saída da Fase 1: o arquivo MP4 (faixa, níveis, o que não carrega, determinismo)
│   └── frame-files-change.md     # Saída da Fase 1: o bloco `plan=` acrescentado aos quadros da etapa 5
├── checklists/
│   └── requirements.md  # Gerado por /speckit-specify
└── tasks.md             # Saída da Fase 2 (comando /speckit-tasks - NÃO criado pelo /speckit-plan)
```

### Código-Fonte (raiz do repositório)

```text
cmd/sobrevoo/
├── main.go                                    # (alterado) monta videofile.NewVideoExporter, videoencoder.NewFFmpeg, VideoService e o comando "video"
└── config_mapping.go                          # (alterado) mapeia config.VideoDefaults → domain.VideoQuality

internal/
├── domain/
│   ├── frame_set.go                           # (estendido) FrameMark, NewFrameMark; FrameFile.PlanID/Width/Height/Whole; porta FrameRepository ganha List e Save/Export recebem a marca
│   ├── frame_verification.go                  # NOVO: FrameDirectory.Verify (na ordem do item 7) e o apoio de mensagens (faixas de números, limites)
│   ├── render_tuning.go                       # (alterado) RenderVersion = 2
│   ├── video.go                               # NOVO: portas VideoEncoder e VideoExporter (no topo), VideoRequest, VideoProgress, EncodeJob, EncoderInfo, VideoSummary (+Duration)
│   ├── video_quality.go                       # NOVO: VideoQuality, ParseVideoQuality
│   ├── errors.go                              # (estendido) onze sentinelas novos
│   ├── *_test.go                              # NOVOS/estendidos: um por arquivo acima; frame_set_test.go (marca, versão)
│   ├── builddomain/
│   │   └── frame_directory_builder.go         # (estendido) PlanID, dimensões, quadro truncado, repetido
│   └── mockdomain/
│       ├── video_encoder.go                   # GERADO (make generate)
│       ├── video_exporter.go                  # GERADO
│       ├── frame_repository.go                # REGERADO (List, marca)
│       └── frame_exporter.go                  # REGERADO (marca)
├── application/
│   ├── video_service.go                       # NOVO: VideoService / videoService / NewVideoService (Assemble)
│   ├── video_service_test.go                  # NOVO
│   ├── frame_service.go                       # (alterado) monta a marca com domain.NewFrameMark
│   ├── frame_service_test.go                  # (alterado)
│   └── mockapplication/
│       └── video_service.go                   # GERADO
└── infra/
    ├── inbound/cli/
    │   ├── video.go                           # NOVO: comando "video" (--output .mp4, --quality, --overwrite), contexto de sinais, resumo
    │   ├── video_progress.go                  # NOVO: progresso no stderr (terminal: linha reescrita; log: no máximo a cada 5 s)
    │   ├── video_test.go / video_progress_test.go  # NOVOS
    │   ├── exit_code.go                       # (estendido) códigos 40–50
    │   └── exit_code_test.go                  # (estendido)
    └── outbound/
        ├── config/config.go                   # (estendido) Config.VideoDefaults, Config.FFmpegBinary (tipos do próprio pacote)
        ├── atomicfile/
        │   ├── atomic_file.go                 # (alterado) PublishPath(path, overwrite, produce func(temporary string) error); Publish reusa o miolo
        │   └── atomic_file_test.go            # (estendido)
        ├── pngfile/
        │   ├── png_mark.go                    # (alterado) grava e lê os dois blocos (conjunto e plano)
        │   ├── frame_repository.go            # (alterado) List; Save recebe a marca; Inspect/List compartilham a leitura de cada arquivo
        │   ├── frame_exporter.go              # (alterado) Export recebe a marca
        │   └── *_test.go                      # (estendidos)
        ├── videofile/
        │   ├── video_exporter.go              # NOVO: Check e Export (usa atomicfile.PublishPath)
        │   └── video_exporter_test.go         # NOVO
        └── videoencoder/
            ├── ffmpeg_video_encoder.go        # NOVO: Probe e Encode (processo externo, -progress, cancelamento por contexto)
            ├── ffmpeg_progress.go             # NOVO: leitura do -progress e da cauda da saída de erro
            └── *_test.go                      # NOVOS (com o ffmpeg de mentira em shell)

test/
└── helper/
    └── png_fixture.go                         # (estendido) quadros PNG mínimos com a marca do conjunto e do plano, truncados, sem o bloco do plano
```

**Documentação de apoio, alterada na implementação**: `CLAUDE.md` e `README.md`
(sexta etapa: o comando `video`, o serviço, as portas e os adapters, o
pré-requisito do `ffmpeg`), `specs/005-frame-rendering/contracts/frame-files.md`
(a nota do bloco `plan=` e da versão 2, conforme `contracts/frame-files-change.md`),
`specs/005-frame-rendering/contracts/cli.md` (a nota de que quadros anteriores
à versão 2 são recusados por `render all` sem `--overwrite`) e as notas de
`spec.md` desta feature (ver "Ajustes na spec" abaixo).

**Decisão de Estrutura**: mesmo projeto único em Go, seguindo as convenções
consolidadas: um arquivo de domínio por responsabilidade (a conferência dos
quadros num arquivo próprio, para ficar testável sozinha), portas no arquivo da
entidade que produzem, adapter de tecnologia (`videofile`, como `pngfile`) e de
estratégia (`videoencoder`, como `tiledecoder`) nomeados pela porta,
composition root em `cmd/sobrevoo/main.go`.

**Ordem de execução (base para o `/speckit-tasks`)**: (0) **instalar o
`ffmpeg`** (ação do usuário: `brew install ffmpeg`), sem o qual os pontos
**(confirmar)** de `research.md` não se fecham; (1) `atomicfile.PublishPath`,
com `make test` verde, isolando qualquer regressão das etapas 3 a 5; (2) a
**mudança na etapa 5**: `FrameMark`, `png_mark` (dois blocos), portas
`Save`/`Export` com a marca, `RenderVersion` 2, testes e a nota no contrato, com
`make test` verde antes de seguir; (3) fixtures de quadro em `test/helper`; (4)
domínio da etapa 6, puro e sem E/S: `VideoQuality`, `VideoSummary`,
`FrameFile` estendido, `FrameDirectory.Verify` (cada tipo de problema, na ordem),
`VideoRequest`/`EncodeJob`/`EncoderInfo`, portas e erros; (5) adapters:
`pngfile.List`, `videofile.VideoExporter`, `videoencoder.FFmpeg` (com o
`ffmpeg` de mentira e, então, uma primeira montagem real para fechar a linha de
comando); (6) `VideoService`; (7) CLI, códigos de saída, configuração e
`main.go`; (8) `quickstart.md` (valores marcados "(anotar)" e os
**(confirmar)**), `CLAUDE.md`, `README.md`. As Histórias P1 a P7 da spec são
entregáveis nesta ordem de prioridade (montagem → conferência → codificador
ausente → qualidade → progresso e resumo → destino e interrupção →
determinismo): a **montagem** (P1) é o MVP e exige a mudança da etapa 5 só
para a conferência do plano (P2); o resto acrescenta verificações e a
proteção do arquivo sobre o mesmo núcleo.

## Ajustes na spec (feitos junto com este plano)

Dois trechos de `spec.md` mudam para refletir decisões do planejamento, sem mudar
nenhum requisito verificável:

1. **Suposição "Formato de saída"**: o destino deve terminar em `.mp4` (erro de
   uso, código `2`, se não terminar), para o nome não mentir sobre o formato
   (`research.md` item 12).
2. **Suposição "Reconhecimento de quadros"**: com o nome exato dos quadros da
   etapa 5, dois arquivos nunca ocupam o mesmo número; a "duplicata" da História
   2 é uma regra do domínio, testada sobre a listagem, que o adapter de hoje não
   consegue produzir (`research.md` item 3). Idem para a ordem das
   verificações da conferência (item 7): a resolução vem antes do conjunto único.

## Rastreamento de Complexidade

Nenhuma violação da constituição; nada a justificar. Duas observações fora da
constituição, mas que pesam no custo: a **mudança na etapa 5** (a marca do
plano) toca o formato dos quadros e a versão do desenho de uma etapa já
entregue — pequena, e isolada na ordem de execução (fase 2); e a **dependência de
execução do `ffmpeg`**, que o usuário instala e que o núcleo desconhece, mas cujos
detalhes (opções, saída de progresso, mensagens de erro) só se confirmam com ele
instalado.

## Riscos e Pontos de Atenção

- **R1 — Determinismo do `libx264` e do MP4**: **confirmado** com o `ffmpeg`
  real (9.0.2/`libx264` core 165, instalado depois deste plano). A linha de
  comando original tinha dois enganos que só o `ffmpeg` real revelou:
  `-x264-params info=0` não é uma opção válida (silenciosamente ignorada) e
  `-movflags +bitexact` não existe mais como flag do muxer MOV nesta versão;
  corrigidos com `-bitexact` (global) e `-metadata:s:v:0 encoder=`
  (`research.md` item 6). Dois vídeos gerados com intervalo e diretórios de
  trabalho diferentes são idênticos byte a byte, e `ffprobe`/`strings` não
  acham nada do ambiente.
- **R2 — Versão do `ffmpeg`**: as opções escolhidas existem há muitas versões; o
  `Probe` só exige o `libx264`. O resumo informa a versão, e o teste do adapter
  fixa a linha de comando para uma mudança não passar despercebida.
- **R3 — Mudança na etapa 5**: quadros já desenhados precisam ser desenhados de
  novo (a etapa 6 os recusa com `44`; `render all` os recusa com `37` sem
  `--overwrite`). A mensagem diz o que fazer; não há migração.
- **R4 — PNG com o meio corrompido**: a conferência lê só o cabeçalho e o fim; a
  primeira tentativa mostrou que o `ffmpeg`, **por padrão, tolera** um erro de
  decodificação isolado numa sequência de imagens e segue em frente (código
  `0`, silencioso) — não bastava "o codificador vê". Corrigido com `-xerror`,
  que faz qualquer erro de decodificação abortar a codificação; confirmado com
  um quadro corrompido de propósito (código `50`, sem arquivo deixado para
  trás). Documentado em `research.md` item 6 e raro na prática (o quadro é
  gravado atomicamente com `fsync`).
- **R5 — Threads fixas do `x264`** (`threads=4`): reprodutível em qualquer máquina,
  mas mais lento onde há muitos núcleos. Trocar o valor muda os bytes: só com uma
  versão nova da ferramenta.
- **R6 — Duração medida pelo `ffprobe`**: o MP4 arredonda a duração à escala de
  tempo da faixa; o critério (SC-001) já admite um quadro de diferença.
- **R7 — `Kill` do `ffmpeg` numa interrupção**: o arquivo temporário é sempre
  apagado pelo `VideoExporter`; se o **próprio Sobrevoo** for morto por
  `SIGKILL`, sobra um `.sobrevoo-*.tmp` na pasta do destino (o mesmo risco das
  etapas anteriores, sem limpeza de órfãos nesta etapa).
- **R8 — Vídeo com resolução que o `libx264` não aceite** (muito alta): o
  `ffmpeg` recusa e o erro `50` traz a mensagem; a etapa 5 limita a 3840 × 2160,
  que o nível 5.1 do H.264 cobre.
