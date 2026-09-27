# Pesquisa: Montagem do Vídeo do Voo

**Feature**: `006-video-assembly` | **Data**: 2026-09-27

Cada item registra a decisão, o motivo e as alternativas descartadas. Onde a
decisão vem de código já existente, o arquivo é citado. Os parâmetros de
codificação foram **confirmados** com o `ffmpeg` real (9.0.2, `libx264` core
165 r3222), instalado depois do planejamento inicial (`quickstart.md`, itens 1
a 10); o item 6 registra duas correções que a linha de comando original (escrita
sem o `ffmpeg` disponível) precisou.

## 1. O codificador: `ffmpeg`, como processo externo, atrás de uma porta

**Decisão**: a codificação é feita pelo programa **`ffmpeg`** (com o codificador
`libx264`), instalado pelo usuário e chamado como **processo externo** pelo
adapter `videoencoder.FFmpeg`, atrás da porta de domínio `VideoEncoder` (ver
`data-model.md`). O núcleo não sabe que o `ffmpeg` existe: só conhece a porta
(Princípios I e II; a própria constituição cita "processo externo como ffmpeg"
como exemplo de dependência que passa por porta). A ferramenta **não embute nem
baixa** o `ffmpeg` (Princípio V): se ele falta, recusa com instruções (item 9).

**Motivo**:
- É o codificador que as plataformas de publicação e os reprodutores de uso comum
  esperam como referência; H.264 em MP4 é o formato "pronto para assistir e
  publicar" da spec (FR-012).
- Escrever um codificador H.264 em Go puro está fora de proporção com o projeto;
  as bibliotecas Go de codificação de vídeo ou envolvem `cgo` (quebram o binário
  único e multiplataforma das etapas anteriores) ou são de qualidade e
  desempenho incompatíveis com 1350 quadros de 1080 × 1920.
- Processo externo mantém o binário do Sobrevoo puro Go e faz do `ffmpeg` uma
  dependência de execução **local** e substituível: outro adapter (outro
  programa, outra biblioteca) implementa a mesma porta sem mexer no núcleo.

**Alternativas descartadas**:
- *Biblioteca com `cgo` (bindings de libav/x264)*: liga o binário a bibliotecas do
  sistema, complica o build multiplataforma e o `make build`.
- *Codificador em Go puro*: o único que existe para H.264 é limitado (sem
  controle de qualidade de taxa de uso geral) e lento.
- *Gerar GIF, APNG ou AVI/MJPEG só com a biblioteca padrão*: não são "vídeo
  pronto para publicar" (tamanho enorme, sem suporte das plataformas).
- *Baixar o `ffmpeg` sozinho*: viola o Princípio V (nada de rede) e o Princípio
  IV (a ferramenta não traz dado ou binário de terceiros embutido).

## 2. O formato: MP4 com H.264, sem áudio

**Decisão**: o vídeo é um arquivo **MP4** com uma só faixa de vídeo **H.264**
(`libx264`, perfil `high`), formato de pixel **`yuv420p`**, com a matriz de cor,
as primárias e a característica de transferência **BT.709** e a faixa de valores
**limitada** (`tv`) etiquetadas no fluxo, e com o índice (`moov`) no início do
arquivo (`-movflags +faststart`), para o vídeo tocar antes de terminar de baixar.
Sem áudio, sem legenda, sem capítulos, sem miniatura.

**Motivo**:
- `yuv420p` + H.264 + MP4 é o denominador comum aceito por reprodutores,
  navegadores e plataformas de publicação sem nova conversão (SC-008). O formato
  exige largura e altura **pares**: já garantido pela etapa 5 (`NewResolution`),
  e reconferido na conferência dos quadros (item 7).
- As etiquetas de cor evitam que reprodutores adivinhem a matriz (para imagens
  em RGB de resolução alta, alguns adivinham BT.601 e mostram tons deslocados).
- O conteúdo do vídeo não depende de nada do ambiente (item 6).

**Alternativas descartadas**:
- *H.265/HEVC*: menor, mas com suporte irregular em navegadores e plataformas.
- *VP9/AV1 em WebM*: suporte bom em navegadores, mas não é o que as plataformas
  de vídeo curto e os aplicativos de celular aceitam como entrada padrão.
- *ProRes ou H.264 sem perdas*: arquivos enormes, não são "prontos para
  publicar"; ficam para uma etapa de edição, se um dia houver.
- *Deixar o formato de pixel e as etiquetas de cor por conta do `ffmpeg`*: as
  escolhas automáticas variam com a versão e afetam a cor e o determinismo.

## 3. Como o codificador recebe os quadros: a sequência de imagens, pelo nome

**Decisão**: o `ffmpeg` lê os quadros direto do disco pelo **padrão de nome**
`frame_%06d.png` (demuxer de sequência de imagens), com `-framerate <taxa>
-start_number 0 -i frame_%06d.png -frames:v <N>`, rodando **com o diretório dos
quadros como diretório de trabalho** do processo (`cmd.Dir`), de modo que o
argumento é só o nome do arquivo e nenhum caractere do caminho (um `%`, um `:`,
espaços) é interpretado como padrão ou como protocolo. Os quadros são lidos um a
um pelo `ffmpeg`, sem ficarem todos na memória (Suposição de desempenho).

**Motivo**:
- A conferência do item 7 já garante que existe **exatamente** um arquivo para
  cada número de 0 a N−1, com o nome canônico de `domain.FrameFileName`. O padrão
  do `ffmpeg` lê a mesma coisa, então a ordem do vídeo é a do plano por
  construção (FR-010) e `-frames:v N` impede que um arquivo a mais (que a
  conferência já recusaria) entre.
- Sem canal de entrada entre os dois processos: não há bloqueio por tubo cheio,
  e a leitura e a decodificação dos PNG usam os núcleos do próprio `ffmpeg`.
- O nome do diretório nunca aparece na saída (o vídeo não guarda o caminho,
  FR-015) porque o `ffmpeg` só grava o que a faixa de vídeo é.

**Alternativas descartadas**:
- *Enviar os PNG pela entrada padrão (`image2pipe`)*: permitiria nomes e ordem
  livres e um progresso "quadros enviados", mas exige uma goroutine escritora,
  drenagem concorrente das três saídas e tratamento de tubo quebrado, para um
  ganho que a conferência de nomes exatos já dá. Fica como alternativa se um
  dia os quadros deixarem de ser arquivos com esse nome.
- *Lista do demuxer `concat`*: exige escapar caminhos em arquivo de texto e
  reintroduz o caminho no processo.
- *Passar o caminho completo no padrão*: quebra com `%` no caminho.

**Nota sobre "duplicata"** (FR-005): com nome exato de arquivo (seis dígitos), dois
arquivos nunca ocupam o mesmo número, então, com o adapter de hoje, uma duplicata
**não pode aparecer**. A regra fica no domínio (`FrameDirectory.Verify`), testada
com uma listagem que repete um número, porque a garantia é da regra e não do
nome do arquivo: se a listagem um dia vier de outra fonte, a duplicata é
recusada com a mensagem que diz o número e os arquivos.

## 4. Taxa de quadros e duração: exatas

**Decisão**: `-framerate` recebe a taxa do plano escrita como decimal mais curto
que a representa (`30`, `29.97`, `59.94`, a que o plano guarda), o vídeo tem
**exatamente N quadros** (`-frames:v N`) e nenhum quadro é duplicado ou
descartado (o fluxo sai com a mesma taxa da entrada, sem `-r` nem `-vf fps`).
A duração é **N ÷ taxa**, calculada no domínio (`VideoSummary.Duration`) e
reportada no resumo; a inspeção do arquivo (`ffprobe`) confere.

**Motivo**: a taxa do plano vai de 1 a 120 e pode ser decimal (contrato da etapa
3). O `ffmpeg` converte `29.97` na razão exata `2997/100`; como a base de tempo
do demuxer de imagens é a própria razão, os carimbos de tempo saem múltiplos
exatos de `1/taxa`, sem arredondar e sem `dup`/`drop` (SC-001: no máximo um quadro
de diferença na duração medida pelo `ffprobe`, que arredonda a duração do MP4).

**Alternativas descartadas**: *converter para racional no domínio* (o plano só
guarda o decimal; qualquer racional escolhido seria um palpite); *forçar `-r`*
(reintroduz a decisão de duplicar/descartar quadros).

## 5. Níveis de qualidade: três nomes, três combinações de `preset` e `crf`

**Decisão**: `low`, `medium` (padrão) e `high`, como os demais níveis nomeados
da ferramenta (`--simplification`, `--distance`), com o tipo de domínio
`VideoQuality`. O significado em parâmetros de codificação **é do adapter** e
está documentado em `contracts/video-file.md`:

| Nível | `-preset` | `-crf` | Para quê |
|---|---|---|---|
| `low` | `veryfast` | 28 | conferir rápido o voo (arquivo menor, codificação mais rápida) |
| `medium` (padrão) | `medium` | 23 | publicar |
| `high` | `slow` | 18 | guardar com a máxima fidelidade prática |

**Motivo**: `crf` menor (com o mesmo `preset` ou mais lento) dá arquivo maior e
mais fiel; a ordem de tamanho `low ≤ medium ≤ high` de FR-013 vale porque os
dois eixos andam juntos (mais lento e `crf` menor). São os valores de uso comum
(o `crf` 23 é o padrão do `libx264`). Sem parâmetros de codificação soltos na
CLI: o usuário escolhe um nome (Suposição).

**Confirmado** (`quickstart.md`, item 3, 380 quadros de 360 × 640): tamanho e
tempo crescem de `low` a `high` (930 KB/0,67 s → 1,4 MB/1,13 s → 2,4 MB/2,08 s),
com a mesma duração, resolução, taxa e quantidade de quadros nos três; e
(`quickstart.md`, item 9, 1140 quadros reais de 720 × 1280) 2,9 MB/5,0 s →
4,4 MB/7,6 s → 6,5 MB/11,2 s.

**Onde vivem**: os valores (`preset`, `crf`) são detalhe do `libx264`, então são
constantes **do adapter**, não de `config` nem do domínio (o domínio só conhece o
nome do nível e a ordem entre os níveis). O nível **padrão** vem da configuração
(`config.VideoDefaults`), mapeado para o domínio pelo composition root, como
`PlanDefaults` e `RenderDefaults` (Princípio VIII).

**Alternativas descartadas**: *nível numérico (`--crf`)* (exige conhecer o
codificador; a spec pede nomes); *mais de três níveis* (sem ganho para o uso
pessoal); *`crf` como constante de domínio* (vazaria o `libx264` para o núcleo).

## 6. Determinismo e ausência de dado do ambiente

**Decisão** (**confirmada** com o `ffmpeg` real, 9.0.2/libx264 core 165 — a
versão inicial abaixo foi corrigida na implementação, ver "O que mudou"): o
mesmo plano, os mesmos quadros, o mesmo nível e o mesmo `ffmpeg` produzem o
mesmo arquivo porque a chamada fixa tudo o que o `ffmpeg` deixaria variar com
o ambiente ou com o relógio:

```text
ffmpeg -nostdin -hide_banner -loglevel error -y -bitexact -xerror
  -nostats -progress pipe:1
  -framerate <taxa> -start_number 0 -i frame_%06d.png -frames:v <N>
  -vf "scale=out_color_matrix=bt709:out_range=tv:flags=accurate_rnd+full_chroma_int+bitexact,format=yuv420p"
  -c:v libx264 -preset <preset> -crf <crf> -profile:v high
  -x264-params threads=4
  -colorspace bt709 -color_primaries bt709 -color_trc bt709 -color_range tv
  -fflags +bitexact -flags:v +bitexact -map_metadata -1 -metadata:s:v:0 encoder=
  -bsf:v filter_units=remove_types=6
  -an -movflags +faststart -f mp4 file:<arquivo temporário>
```

- **`-bitexact` (global) é a opção que de fato tira o nome e a versão do
  `ffmpeg`/`x264` do arquivo** — `-fflags +bitexact`/`-flags:v +bitexact`
  (por si só) só tiram o **número da versão** da etiqueta `encoder` do
  contêiner (`Lavc63.1.102 libx264` → `Lavc libx264`), não a etiqueta em si; é
  o `-metadata:s:v:0 encoder=` que a apaga por completo. `-map_metadata -1`
  tira o que sobraria dos metadados de entrada (aqui, nada: os PNG não têm
  metadados que o `ffmpeg` leia como tal).
- **A mensagem SEI do `x264`** (a string `x264 - core … - options: …`, com a
  versão e todos os parâmetros de codificação) **não tem opção que a
  desligue** — nem no `ffmpeg` (`-x264-params info=0` **não existe**: essa
  suposição do plano estava errada, `-x264-params` recusa `info` como chave
  desconhecida), nem no `x264` standalone (`x264 --fullhelp` não lista
  `--no-info` nesta versão; a mensagem aparece até codificando fora do
  `ffmpeg`). A saída: um filtro de bitstream que remove o NAL de SEI do fluxo
  H.264 depois de codificado — `-bsf:v filter_units=remove_types=6` (tipo 6 =
  SEI; nada que esta ferramenta precise usa SEI: sem `pic_struct`, sem HRD).
  Conferido que o vídeo continua decodificando normalmente (mesmo PSNR do item
  16) depois do filtro.
- **Sem dependência do número de processadores**: o `x264` só é reproduzível com o
  **mesmo número de threads**; o valor é fixo (`threads=4`), não o padrão
  "quantos núcleos houver". O custo é conhecido (menos paralelismo em máquinas
  grandes) e o valor pode mudar só junto com a versão da ferramenta (item 14).
- **Sem dependência do formato de pixel automático**: conversão explícita
  (`scale=… ,format=yuv420p`, aritmética exata, sem dithering) e etiquetas de cor.
- **Sem dado do caminho**: `-i` é só o nome do arquivo (item 3) e a saída vai
  para um arquivo temporário que é renomeado (item 10); nada disso entra no
  contêiner.
- **`-xerror`**: sem ele, um quadro cujo `IDAT` está corrompido por dentro
  (assinatura, `IHDR` e `IEND` intactos — o que `Verify`, item 7, aceita) **não
  falha a codificação**: por padrão o `ffmpeg` tolera um erro de decodificação
  isolado numa sequência de imagens e segue em frente, silenciosamente, com o
  quadro reaproveitado ou vazio, e sai com código `0`. Isso contradiz a
  garantia da spec (nunca um vídeo "errado" sem aviso) e o risco R4 previsto no
  planejamento. `-xerror` faz **qualquer** erro de decodificação ou
  multiplexação abortar a codificação, e é isso que se traduz em
  `ErrVideoEncodingFailed` (código `50`).

**Confirmado** (`quickstart.md`, item 4, com o `ffmpeg` 9.0.2/libx264 core
165 real): dois vídeos gerados com 3 s de intervalo, em diretórios de trabalho
diferentes, são idênticos byte a byte (`cmp`); `ffprobe -show_format
-show_streams` e `strings -a` não encontram data, versão do `ffmpeg`/`x264`,
caminho, nome de máquina nem nome de usuário; repetido 5 vezes, os 5 arquivos
têm o mesmo hash.

**A promessa e o seu limite**: igualdade byte a byte para o **mesmo `ffmpeg`**
(mesma versão e mesma compilação, o que o resumo mostra) e a mesma versão da
ferramenta. Versões diferentes do `ffmpeg`/`x264` podem produzir fluxos
diferentes: isso está na spec (Suposição "Determinismo") e o resumo imprime o
codificador usado justamente para o usuário poder comparar.

**Alternativas descartadas**: *`-threads 1`* (reprodutível em qualquer máquina,
mas 3 a 5 vezes mais lento); *`sliced-threads`* (independe do número de threads,
mas reduz a qualidade por byte); *pós-processar o arquivo para apagar campos*
(desnecessário: `-bitexact` + `-metadata:s:v:0 encoder=` + `filter_units` já
bastam, confirmado); *`-movflags +bitexact`* (existiu em versões antigas do
`ffmpeg` como flag do muxer MOV; nesta versão (9.0.2) não é mais uma opção de
`-movflags` válida — o `-bitexact` global é o mecanismo atual e é o que este
adapter usa).

## 7. Conferir os quadros antes de codificar: `FrameDirectory.Verify`

**Decisão**: a conferência é regra de domínio, um método de `FrameDirectory`
(o dono da listagem, como `Plan` da etapa 5), sobre a listagem que a porta
`FrameRepository` devolve (`List`, item 8). Reconhece como **quadro** o que é
nosso (PNG com a marca da ferramenta, como na etapa 5) e **ignora**, sem contar
como lacuna, duplicata ou excedente, o que não é (FR-003). Confere, **da mais
barata à mais cara**, e para no primeiro tipo de problema:

1. **Há quadros?** Diretório sem nenhum quadro reconhecido → `ErrFrameDirectoryInvalid`.
2. **Identificação do plano** (FR-004): algum quadro sem ela (desenhado antes da
   mudança do item 8) → `ErrFramesWithoutPlanID`; algum de identificação
   diferente da do plano informado → `ErrFramesDoNotMatchPlan`.
3. **Resolução** (FR-006): todos com as mesmas dimensões, pares →
   `ErrFrameResolutionInvalid`.
4. **Um só conjunto** (FR-004): todos com a mesma identificação de conjunto (a
   mesma fatia de dados, resolução e versão do desenho) → senão,
   `ErrFramesDoNotMatchPlan` ("N conjuntos misturados").
5. **Numeração** (FR-005): faltas, excedentes (número ≥ quadros do plano) e
   repetições → `ErrFrameSequenceInvalid`, dizendo **exatamente** quais.
6. **Integridade** (FR-007): quadro que não é um PNG inteiro (truncado, sem o
   fim do arquivo) → `ErrFrameFileInvalid`, com o arquivo.

**Por que essa ordem**: a correspondência ao plano vem antes de tudo para que um
diretório de **outro voo** com número de quadros diferente diga "não é deste
plano" e não "faltam 800 quadros" (a informação útil é a primeira). A
**resolução vem antes do conjunto** porque a identificação do conjunto já
inclui a resolução: se viesse depois, quadros de resoluções diferentes seriam
sempre "conjuntos misturados" e a mensagem de resolução (a que diz o que fazer)
nunca apareceria. O caso real é uma sobrescrita interrompida em outra
resolução, que deixa quadros novos e antigos do mesmo plano no diretório. A
numeração e a integridade só fazem sentido para o conjunto certo.

**Mensagens (SC-005)**: dizem o que falta ou destoa. Números de quadro saem em
**faixas** (`12-15, 40`), no máximo dez faixas, com o total sempre à vista
(`(37 missing in all)`); arquivos, no máximo cinco por causa; a resolução
esperada é a **mais frequente** (empate: a do menor número), e os que destoam
saem com a resolução de cada um. Quando faltam números e há arquivos com nome
de quadro que **não são nossos**, a mensagem diz que foram ignorados e por quê
(sem a marca da ferramenta).

**Só cabeçalho e final, não o PNG inteiro**: a integridade é a da etapa 5
(assinatura, `IHDR`, `IEND` no fim: `pngfile.readInfo`), o que detecta
truncamento em ~1 ms por quadro (1350 quadros em < 2 s, SC-005). Um PNG com o
meio corrompido e o fim intacto passa pela conferência e faz o codificador
falhar (`ErrVideoEncodingFailed`, com a mensagem do `ffmpeg`): documentado, e
raro (o quadro é gravado atomicamente, com `fsync`).

**Alternativas descartadas**: *decodificar todos os PNG na conferência* (custo de
um minuto ou mais sem ganho prático, e o codificador já os decodifica);
*conferir só a quantidade* (não vê lacuna compensada por excedente).

## 8. A identificação do plano dentro dos quadros (Clarificação de 2026-09-27)

**Decisão**: a etapa 5 passa a gravar em cada quadro um **segundo bloco `tEXt`**,
logo depois do da identificação do conjunto: palavra-chave `Sobrevoo`, texto
`plan=<id>`, com `<id>` o `CameraPlan.ID()` (SHA-256 hexadecimal, 64 caracteres)
— a mesma identificação que a etapa 4 guarda no recorte. A gravação passa a
receber um `FrameMark{SetID, PlanID}` no lugar do `FrameSetID` solto
(`FrameRepository.Save` e `FrameExporter.Export`), montado por
`NewFrameMark(plan, slice, resolution, tuning)`. A leitura da marca devolve os
dois campos; um quadro só com o bloco do conjunto (de antes da mudança) é
**nosso, sem identificação do plano**.

**A versão do desenho sobe de 1 para 2** (`domain.RenderVersion`): a
identificação do conjunto já inclui a versão, então os quadros antigos passam a
ser de **outro conjunto**: sem `--overwrite`, `render all` os recusa como quadros
de outro conjunto (código 37) e, com `--overwrite`, os redesenha. É o mecanismo
que a etapa 5 já tinha para isso; a alternativa (reconhecer "mesmo conjunto,
mas sem a identificação do plano" na retomada) espalharia uma exceção pelo
`FrameDirectory.Plan`. O custo é único: quem já desenhou quadros os desenha de
novo, como o recorte da etapa 5 (FR-004a). **Os pixels não mudam** (a versão
de desenho aqui marca o formato do arquivo): o teste que fixa o hash dos
pixels de um quadro continua igual; só o teste que compara `RenderVersion` com a
próxima muda de expectativa.

**Determinismo dos quadros**: o campo é função só do conteúdo do plano; o
desenho continua idêntico byte a byte para o mesmo plano, o mesmo recorte e a
mesma resolução (FR-015 da etapa 5 mantido).

**Onde isso aparece no código da etapa 5**: `png_mark.go` (codifica e lê os
dois blocos), `frame_repository.go` e `frame_exporter.go` (recebem a marca),
`frame_service.go` (monta a marca), `frame_set.go` (`FrameMark`, `FrameFile.PlanID`,
portas) e `contracts/frame-files.md` (a nota da mudança está em
`contracts/frame-files-change.md`).

**Alternativas descartadas**: *identificação do plano dentro do bloco do
conjunto* (`frame-set=…;plan=…`: mudaria o formato de um bloco que a etapa 5
já documentou, e uma leitura antiga não veria o conjunto); *arquivo de registro
no diretório* (rejeitado na etapa 5, item 17); *a etapa 6 receber também o
recorte* (fora do que o usuário pediu: só o plano e os quadros); *conferir só
quantidade e resolução* (a opção B da clarificação, recusada).

## 9. Verificar o codificador antes de começar

**Decisão**: `VideoEncoder.Probe(ctx)` (chamada antes de qualquer codificação e
depois da conferência do plano, dos quadros e do destino, FR-009) faz:

1. localizar o programa (`exec.LookPath` do nome configurado, `ffmpeg`);
2. `ffmpeg -hide_banner -version` → a primeira linha dá o nome e a versão
   (`ffmpeg version 7.1 …` → `7.1`);
3. `ffmpeg -hide_banner -encoders` → confere que o codificador `libx264` está na
   lista.

O resultado, `EncoderInfo{Name, Version, Codec}`, vai para o resumo
(`ffmpeg 7.1 (libx264)`). Falhas → `ErrEncoderUnavailable`, sempre com a
mensagem **do que falta e do que fazer**, montada pelo adapter (é ele quem sabe
o nome do programa):

```text
video encoder not available: "ffmpeg" was not found on the PATH; install it (macOS: brew install ffmpeg; Debian/Ubuntu: sudo apt install ffmpeg; Windows: winget install Gyan.FFmpeg), then check it with: ffmpeg -version
video encoder not available: ffmpeg 6.0 has no libx264 encoder; install a build that includes it (macOS: brew install ffmpeg; Debian/Ubuntu: sudo apt install ffmpeg)
```

**Motivo**: são duas chamadas curtas (~100 ms), antes de gastar minutos; e a
mensagem cobre os dois casos da spec (programa ausente, programa sem o
formato). O nome do programa e as dicas de instalação **são do adapter**, não do
domínio.

**Alternativas descartadas**: *descobrir só quando a codificação falha* (o
usuário espera a conferência dos quadros inteira para ler um erro de sistema);
*variável de ambiente ou flag com o caminho do `ffmpeg`* (ponto de extensão
existe em `config.Config.FFmpegBinary`, hoje fixo em `ffmpeg`; abrir uma fonte
externa é decisão futura, Princípio VIII).

## 10. O destino: publicação atômica de um caminho, e a checagem prévia

**Decisão**: duas operações na porta `VideoExporter` (`data-model.md`):

- `Check(path, overwrite)` — **antes** de codificar: recusa um destino que já
  existe sem `--overwrite` (`ErrVideoDestinationExists`), que é um diretório, cuja
  pasta não existe ou não pode ser gravada (`ErrVideoDestinationInvalid`; a
  gravabilidade se testa criando e apagando um arquivo temporário na pasta). É a
  recusa **barata**: nada foi codificado. Não é a garantia (a corrida entre a
  checagem e a publicação é fechada em `Export`).
- `Export(path, overwrite, produce)` — cria um arquivo temporário na pasta do
  destino, entrega o **caminho** dele a `produce` (o codificador escreve ali), e
  só então o publica: `fsync`, `chmod 0644` e `rename` (com `--overwrite`) ou
  `link` exclusivo (sem ele), a mesma regra de `atomicfile.Publish`. Falha ou
  interrupção em qualquer ponto remove o temporário e não toca o arquivo
  anterior. Devolve o tamanho do arquivo publicado.

O `atomicfile.Publish` atual entrega um `io.Writer`; o `ffmpeg` precisa de um
**caminho que se possa reposicionar** (o MP4 com `faststart` reescreve o início do
arquivo), então o pacote ganha `atomicfile.PublishPath(path, overwrite,
produce func(temporary string) error)`, e `Publish` passa a reusar o miolo
comum (publicar o arquivo já escrito). O comportamento de `Publish` não muda
(seus testes continuam valendo).

**Detalhes que importam**: o caminho de saída é tornado **absoluto** antes de
tudo (o `ffmpeg` roda com outro diretório de trabalho, item 3); o temporário
recebe o formato por `-f mp4` (a extensão `.tmp` não diz nada) e o prefixo
`file:` (nenhum caminho vira protocolo); o temporário existe vazio quando o
`ffmpeg` começa, então `-y`.

**Alternativas descartadas**: *o `ffmpeg` escrever direto no destino final*
(deixa arquivo parcial em falha); *escrever na saída padrão do `ffmpeg`* (MP4
com índice no início exige arquivo reposicionável; MP4 fragmentado não é o que as
plataformas esperam); *checar só no fim* (a pessoa esperaria minutos para
descobrir que o destino existia).

## 11. Progresso, interrupção e falha do codificador

**Decisão**:

- **Progresso** — `-progress pipe:1` faz o `ffmpeg` escrever, a cada ~0,5 s, blocos
  de `chave=valor` na saída padrão; o adapter lê as linhas `frame=<n>` e chama
  `progress(n)` — **quadros já codificados**, o número verdadeiro, não o que se
  enviou (FR-016). O serviço o converte em `VideoProgress{Done, Total, Elapsed}`. Ao
  terminar com sucesso, o serviço garante um último aviso com `Done = Total`.
- **Interrupção** — o adapter usa `exec.CommandContext`: o cancelamento do
  contexto (Ctrl+C ou `SIGTERM`, que a CLI já converte em contexto, como em
  `render all`) **mata o processo** e o adapter espera o término (sem processo
  órfão); o arquivo temporário é apagado por `Export`; o serviço devolve
  `ErrVideoInterrupted` com o resumo (`Interrupted`, `Encoded` = último
  progresso) — o mesmo tratamento de `interruption` de `FrameService`. Não há
  retomada: o arquivo final só existe inteiro (spec, História 6, cenário 6).
- **Falha** — código de saída do `ffmpeg` diferente de zero, com o contexto vivo →
  `ErrVideoEncodingFailed`, com o código e o **final da saída de erro** (no máximo
  2 KiB, já com `-loglevel error`, que traz o motivo, como "No space left on
  device" ou "Invalid PNG signature"). O temporário é apagado.

**Motivo**: `-progress` é a interface prevista pelo `ffmpeg` para isso, estável
entre versões e sem depender do formato da linha de estatística do terminal
(`-nostats` desliga essa). A saída padrão e a de erro são lidas por goroutines
próprias enquanto o processo roda, para nunca bloquear por tubo cheio.

**Alternativas descartadas**: *contar quadros enviados* (só existiria com entrada por tubo,
alternativa descartada no item 3, e mesmo assim adianta o real em dezenas de
quadros pelo *lookahead*);
*`SIGINT` gracioso antes do `Kill`* (o resultado é descartado de qualquer forma; o
`Kill` é imediato e igual em Windows); *ler a saída de erro para o progresso*
(formato instável).

## 12. A CLI: `video`, ordem das verificações e códigos de saída

**Decisão**: um comando novo, **`sobrevoo video <plano> <diretório-de-quadros>
--output <arquivo.mp4> [--quality low|medium|high] [--overwrite]`**, que expõe
`VideoService.Assemble` (`contracts/cli.md`). O comando de topo se chama `video`
(a etapa é a montagem do vídeo; `render` continua sendo o desenho dos quadros).
`--output` é obrigatório e **deve terminar em `.mp4`** (sem diferenciar maiúsculas
de minúsculas): o formato é sempre MP4, e um `--output voo.mkv` seria um vídeo
MP4 com nome enganoso, então é recusado como erro de uso (código `2`). O nível
inexistente também é erro de uso (código `2`), com a lista dos aceitos, como
`--distance`.

**Ordem das verificações** (da mais barata à mais cara, sempre antes de
codificar e sem criar arquivo): uso (`--output`, extensão, `--quality`) → leitura
do plano (`17`, `18`) → o diretório e os quadros (`40`–`45`, na ordem do item 7) →
o destino (`47`, `48`) → o codificador (`46`) → a codificação (`50`, `49`).

**Códigos novos** (depois do `39` da etapa 5): `40` `ErrFrameDirectoryInvalid`, `41`
`ErrFrameSequenceInvalid`, `42` `ErrFrameResolutionInvalid`, `43`
`ErrFramesDoNotMatchPlan`, `44` `ErrFramesWithoutPlanID`, `45` `ErrFrameFileInvalid`,
`46` `ErrEncoderUnavailable`, `47` `ErrVideoDestinationExists`, `48`
`ErrVideoDestinationInvalid`, `49` `ErrVideoInterrupted`, `50` `ErrVideoEncodingFailed`.
Distintos entre si e dos das etapas 1 a 5 (FR-021).

**Progresso na CLI**: em terminal, uma linha reescrita (`Encoding frame
412/1350 (30.5%), elapsed 00:00:41`); fora dele (um arquivo de log), uma linha
**no máximo a cada 5 s** e a última, decidido pelo `Elapsed` que o serviço
informa (os quadros são atualizados a cada ~0,5 s, então a regra "a cada dez
quadros" da etapa 5 não serve). Reusa `RenderOption` (`WithTerminalCheck`),
`interruptContext`, `formatElapsed` e `formatResolution` da etapa 5.

**Alternativa descartada**: *`render video`* (mistura o desenho, que precisa de
plano e recorte, com a montagem, que precisa de plano e quadros); *aceitar
qualquer extensão* (vídeo com nome que mente).

## 13. Configuração

**Decisão**: `config.Config` ganha dois campos, com tipos próprios (o pacote não
importa o domínio): `VideoDefaults{Quality Level}` (padrão `LevelMedium`) e
`FFmpegBinary string` (padrão `"ffmpeg"`, procurado no `PATH`). O composition
root mapeia o nível para `domain.VideoQuality` (`config_mapping.go`) e passa o
nome do programa a `videoencoder.NewFFmpeg`. Os parâmetros do `libx264` ficam no
adapter (item 5). O número de threads (item 6) também é constante do adapter: é
o que define os bytes do fluxo, não uma heurística ajustável pelo usuário.

**Motivo**: mesmo padrão de `PlanDefaults`/`RenderDefaults` (Princípio VIII).

## 14. Versão da montagem

**Decisão**: **não há** constante de versão do vídeo, como há `RenderVersion`
para os quadros: nada persiste um identificador de vídeo, e a promessa de bytes
iguais vale "para o mesmo `ffmpeg` e a mesma versão da ferramenta" (item 6). Se
os parâmetros do adapter mudarem (o `preset`, o `crf`, as threads), a mudança é
registrada no contrato `video-file.md` e nas notas de versão, sem mecanismo de
código.

**Alternativa descartada**: *`VideoVersion` gravada num metadado* (é um dado a
mais no arquivo, que a spec quer sem nada do ambiente, e ninguém o lê).

## 15. Desempenho e memória

**Metas**: a conferência dos quadros de 1350 arquivos em < 2 s (lê 4 KiB e 12
bytes de cada); a montagem de 1350 quadros de 1080 × 1920 no nível `medium` em
≤ 10 min (SC-007); o `ffmpeg` decodifica os PNG em paralelo e o `x264` usa 4
threads (item 6).

**Confirmado** (`quickstart.md`, itens 5, 9 e 10, mesmo Apple M1 de 8
núcleos): a conferência de 1140 quadros recusados por conjunto levou **0,40 s**
(bem abaixo de 2 s); o passeio real (1020 quadros, 1080 × 1920, relevo
Copernicus real) montou em `medium` em **14 s** e em `high` em **20 s** — bem
abaixo dos 10 min de SC-007 (a medida é com 1020 quadros, 330 a menos que os
1350 do critério, mas na mesma resolução e no mesmo computador, o resultado é
consistente com a folga). Com 1140 quadros reais de 720 × 1280 (`item 9`):
`low` 5,04 s, `medium` 7,57 s, `high` 11,24 s. A estimativa inicial (1 a 3
minutos para 1350 quadros de 1080 × 1920) era conservadora: a codificação de
quadros já decodificados e redimensionados de forma trivial (sem escala real,
já que os PNG saem na resolução final) é mais rápida do que a estimativa
supôs.

**Memória**: a do `ffmpeg` (algumas centenas de MiB) mais uns MiB do Sobrevoo
(os buffers de leitura das saídas do processo); nenhum quadro fica na memória
do Sobrevoo.

## 16. Dados de teste, testes automatizados e validação com dados reais

- **Testes automatizados** (sem `ffmpeg` instalado): o adapter `videoencoder` é
  testado contra um **`ffmpeg` de mentira** — um script de shell gravado num
  diretório temporário, que imita o que importa (versão, lista de codificadores,
  `frame=` no progresso, código de saída, escrever no arquivo de saída, ficar
  esperando o sinal) e registra os argumentos que recebeu, para o teste conferir a
  linha de comando do item 6. Os testes de domínio e de serviço usam as portas
  mockadas; os de CLI, o serviço mockado (Princípio X). Nenhuma suíte
  automatizada exige o `ffmpeg` real.
- **Validação manual com o `ffmpeg` real** (`quickstart.md`): instalar o `ffmpeg`
  (`brew install ffmpeg`), desenhar os quadros de um voo sintético e do voo real
  (a etapa 5 já validou o `resources/`), montar o vídeo, medir com o `ffprobe`
  (quantidade de quadros, taxa, duração, resolução, codec, formato de pixel,
  cor), comparar dois vídeos byte a byte, procurar dado do ambiente e exercitar
  cada recusa. **Feito**: `ffmpeg` 9.0.2/`libx264` core 165 instalado pelo
  usuário depois do planejamento (o binário não o instala sozinho); a validação
  do `quickstart.md` (itens 1 a 10) confirmou os pontos **(confirmar)** e achou
  os dois ajustes do item 6 (`-bitexact`, `-metadata:s:v:0 encoder=`,
  `filter_units` no lugar de `info=0`, e `-xerror`, que não estava previsto).
- **Fixtures**: `test/helper` ganha a construção de um diretório de quadros
  válido em código (PNG mínimos com a marca do conjunto e do plano), para os
  testes do `pngfile.List`, sem depender de `render all`.

## 17. Riscos e alternativas rejeitadas (resumo)

- **R1 — Determinismo do `libx264` e do MP4**: **confirmado** (item 6): dois
  vídeos gerados com intervalo e diretórios de trabalho diferentes são
  idênticos byte a byte, e `ffprobe`/`strings` não acham nada do ambiente. A
  linha de comando original tinha dois enganos (`-x264-params info=0` não
  existe; `-movflags +bitexact` não é mais uma opção do muxer MOV nesta
  versão) que a validação real corrigiu, com o `-bitexact` global e o
  `-metadata:s:v:0 encoder=`.
- **R2 — Versão do `ffmpeg`**: opções usadas existem há muitas versões
  (`-progress`, `-x264-params`, `-fflags +bitexact`); o `Probe` só exige o
  `libx264`. Uma versão muito nova que mude um padrão é detectada pelo
  quickstart e pelos testes do adapter, que fixam a linha de comando.
- **R3 — Mudança na etapa 5**: quadros antigos são recusados e precisam ser
  desenhados de novo (R9 da etapa 5 tinha o mesmo perfil para o recorte); a
  mensagem diz o que fazer.
- **R4 — PNG com o meio corrompido**: **era um risco maior do que o previsto**:
  sem tratamento, o `ffmpeg` **tolera** um quadro cujo `IDAT` não decodifica e
  segue em frente, saindo com sucesso (código `0`) — não bastava "o codificador
  vê", era preciso mandá-lo parar. Corrigido com `-xerror` (item 6, achado na
  validação real com um quadro corrompido de propósito): qualquer erro de
  decodificação vira `ErrVideoEncodingFailed` com a causa, sem arquivo deixado
  para trás.
- **R5 — Encerrar o `ffmpeg` no Windows**: `Kill` é suportado em todas as
  plataformas; o teste do adapter que mata o processo usa o script de mentira
  em shell, então roda onde há `sh`.
- **R6 — Threads fixas**: mais lento em máquinas com muitos núcleos; aceitável
  (item 15) e trocável junto com uma versão da ferramenta.
- **R7 — Duração medida pelo `ffprobe`**: o MP4 arredonda a duração à escala de
  tempo da faixa; SC-001 já admite um quadro de diferença.

**Alternativas globais rejeitadas**: *codificar em Go* (item 1), *baixar o
`ffmpeg`* (item 1), *outro contêiner/codec* (item 2), *entrada por tubo* (item
3), *flags soltas de codificação na CLI* (item 5), *retomar uma montagem
interrompida* (uma montagem dura minutos e o arquivo só existe inteiro: sem
estado parcial a reaproveitar).
