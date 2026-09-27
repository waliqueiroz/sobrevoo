# Contrato do Arquivo de Vídeo

**Feature**: `006-video-assembly` | **Data**: 2026-09-27

É o contrato de dados que o usuário, os reprodutores e as plataformas de
publicação consomem: o arquivo que `sobrevoo video` grava. O mesmo plano, os
mesmos quadros, o mesmo nível de qualidade e o mesmo codificador produzem o mesmo
arquivo, byte a byte (FR-014, SC-003) — o que exclui data e hora de geração,
caminhos, nome de máquina, nome de usuário, tempo gasto e qualquer dado do
ambiente (FR-015, SC-004). **Confirmado** com o `ffmpeg` real (9.0.2, `libx264`
core 165; `quickstart.md`, item 4).

## Arquivo

- **Contêiner MP4** (`.mp4`), com o índice (`moov`) no início do arquivo
  (`faststart`): o vídeo começa a tocar antes de terminar de ser baixado.
- **Uma faixa** de vídeo e **nenhuma outra**: sem áudio, sem legenda, sem
  capítulos, sem imagem de capa, sem título nem autor.
- Permissões `0644`, publicado inteiro (`fsync`, depois `rename` ou `link`
  exclusivo): nunca aparece pela metade.

## A faixa de vídeo

| Propriedade | Valor |
|---|---|
| Codec | H.264 (`libx264`), perfil `high` |
| Formato de pixel | `yuv420p`, 8 bits |
| Cor | matriz, primárias e transferência **BT.709**, faixa **limitada** (`tv`), etiquetadas no fluxo |
| Quantidade de quadros | **exatamente** a do plano (`summary.frame_count`), na ordem do plano (do 0 ao último), sem quadro repetido, omitido nem inserido |
| Taxa de quadros | a do plano (`parameters.frame_rate`), exata (`30`, `29.97`, `59.94`), taxa constante |
| Duração | `frame_count ÷ frame_rate` (o `ffprobe` arredonda a duração do MP4: até **um quadro** de diferença, SC-001) |
| Resolução | a das imagens, **sem** redimensionar, recortar nem acrescentar bordas; largura e altura pares |
| Conteúdo | só o que as imagens mostram: nenhum texto, estatística ou marca d'água |

## Níveis de qualidade

Constantes do adapter `videoencoder.FFmpeg`. Dentro do mesmo `preset`, um `crf`
menor dá um arquivo maior e mais fiel; entre os níveis, os dois eixos andam
juntos, então o tamanho cresce de `low` para `high` (FR-013, SC-009).

| Nível | `-preset` | `-crf` | Para quê |
|---|---|---|---|
| `low` | `veryfast` | 28 | conferir rápido |
| `medium` (padrão) | `medium` | 23 | publicar |
| `high` | `slow` | 18 | guardar com a máxima fidelidade prática |

**Confirmado** (`quickstart.md`, itens 3 e 9): com 380 quadros de 360 × 640,
`low` 930 KB/0,67 s, `medium` 1,4 MB/1,13 s, `high` 2,4 MB/2,08 s; com 1140
quadros reais de 720 × 1280, 2,9 MB/5,0 s, 4,4 MB/7,6 s, 6,5 MB/11,2 s — o
tamanho e o tempo crescem de `low` para `high`, com a mesma duração, resolução,
taxa e quantidade de quadros nos três.

## O que o arquivo **não** carrega (FR-015)

Nenhum dos itens abaixo aparece nos metadados nem no fluxo — **confirmado**
com o `ffmpeg` real (`ffprobe -show_format -show_streams` e `strings -a`,
`quickstart.md` item 4):

- data ou hora de geração (`creation_time`, datas do cabeçalho `mvhd`/`tkhd`/`mdhd`);
- a etiqueta `encoder` do contêiner (limpa explicitamente,
  `-metadata:s:v:0 encoder=`: sem ela, o muxer grava `Lavc <versão>
  libx264`/`Lavf <versão>` mesmo em modo `-bitexact`);
- a mensagem SEI que o `libx264` escreve dentro do próprio fluxo H.264, com a
  versão e todos os parâmetros de codificação (`x264 - core … - options: …`;
  nenhuma opção do `ffmpeg` ou do `x264` a desliga — é removida do fluxo depois
  de codificado, com o filtro de bitstream `filter_units=remove_types=6`, que
  tira todo NAL de tipo SEI; nada que esta ferramenta use depende de SEI);
- caminhos de arquivos de entrada ou de saída, nome da máquina e nome do usuário;
- os metadados dos PNG de entrada (a marca do conjunto e do plano dos quadros);
- o tempo gasto na montagem.

O que o cabeçalho tem é constante: os nomes de manipulador (`VideoHandler`) e as
etiquetas de cor.

## Determinismo e o seu limite

- **Vale para**: o mesmo `ffmpeg` (a mesma versão e a mesma compilação, que o
  resumo mostra na linha `Encoder`), a mesma versão do Sobrevoo, o mesmo plano,
  os mesmos quadros e o mesmo nível. Independe do momento, do diretório de
  trabalho, do nome do destino e do número de processadores: as threads do
  `x264` são fixas (`threads=4`).
- **Não vale para**: versões diferentes do `ffmpeg` ou do `x264`, que podem
  codificar diferente. A linha `Encoder` do resumo existe para o usuário saber
  com que comparar. Uma versão nova do Sobrevoo que mude os parâmetros desta
  página o informa nas notas de versão.
- A igualdade é do **arquivo**; os quadros de entrada já são idênticos byte a byte
  (etapa 5).

## Como conferir (usado nos testes e no quickstart)

```sh
ffprobe -v error -count_frames -select_streams v:0 \
  -show_entries stream=codec_name,profile,pix_fmt,width,height,r_frame_rate,nb_read_frames,color_space,color_range \
  -show_entries format=duration,format_name -of default=nw=1 voo.mp4
cmp voo-a.mp4 voo-b.mp4                       # dois vídeos do mesmo pedido: nenhuma diferença
ffprobe -v error -show_format -show_streams voo.mp4 | grep -i -E "creation|encoder|/tmp|/Users"   # nada
strings -a voo.mp4 | grep -i -E "x264|Lavc|Lavf"   # nada: sem a etiqueta de versão nem a SEI do x264
```

## Compatibilidade

Consumidores devem tratar o arquivo como um MP4 comum. O contrato acima é o que
a ferramenta promete; qualquer mudança nele (contêiner, codec, cor, taxa, o
mapeamento dos níveis) é registrada aqui e nas notas de versão.
