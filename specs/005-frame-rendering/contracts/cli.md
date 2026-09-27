# Contrato de CLI: `sobrevoo render frame` e `sobrevoo render all`

**Feature**: `005-frame-rendering` | **Data**: 2026-09-26

Dois comandos novos, filhos de `render`, que expõem `FrameService.DrawFrame` e
`FrameService.DrawFrames` (ver `data-model.md`) através de
`internal/infra/inbound/cli`. Não há API HTTP nem GUI (fora de escopo). Os
contratos de `inspect` (etapa 1), `geodata …` (etapas 2 e 4) e `plan` (etapa 3)
permanecem inalterados, com **uma** exceção, na etapa 4: `geodata slice
--export` passa a gravar a identificação do plano no arquivo do recorte
([`slice-file-change.md`](./slice-file-change.md)). Os arquivos que estes
comandos leem e gravam têm contrato próprio: o plano, em
`specs/003-camera-path-planning/contracts/plan-file.md`; o recorte, em
`specs/004-geo-data-slice/contracts/slice-file.md` (+ a mudança acima); as
imagens, em [`frame-files.md`](./frame-files.md).

## `sobrevoo render frame`

```text
sobrevoo render frame <arquivo-de-plano> <arquivo-de-recorte> --number <n> --output <arquivo.png> [--resolution <LxA>] [--overwrite]
```

- `<arquivo-de-plano>`: plano exportado por `sobrevoo plan --export`.
- `<arquivo-de-recorte>`: recorte exportado por `sobrevoo geodata slice
  --export` **a partir desse mesmo plano**.
- `--number` (obrigatória): número do quadro no plano, de 0 a `frame_count − 1`.
  É lida como texto: `3.5`, `abc` ou `-1` são recusados com o erro de número
  (código `33`), não com erro de uso.
- `--output` (obrigatória): arquivo PNG a gravar. A pasta precisa existir.
- `--resolution` (opcional): `LARGURAxALTURA` em pixels; padrão `1080x1920`
  (ver "Resolução").
- `--overwrite` (opcional): permite substituir o arquivo, se existir.

O quadro gravado é **byte a byte igual** ao mesmo quadro de `render all` na
mesma resolução. O arquivo não pertence a nenhum conjunto de quadros e não
consulta nem altera nenhum diretório.

### Saída (sucesso)

Resumo em `stdout` (rótulos e ordem estáveis):

```text
Frame 300 of 1260 drawn to /tmp/quadro.png
Resolution: 1080x1920
Time: 00:00:02
Holes: map tiles missing: no, elevation without value: no
```

`Holes` diz `yes` ou `no` para cada causa (o quadro isolado é um só).
**Código de saída**: `0`.

## `sobrevoo render all`

```text
sobrevoo render all <arquivo-de-plano> <arquivo-de-recorte> --output <diretório> [--resolution <LxA>] [--overwrite]
```

- `--output` (obrigatória): diretório de destino; criado se não existir
  (a pasta-mãe precisa existir).
- `--overwrite` (opcional): redesenha todos os quadros e substitui os
  existentes; ver "Comportamento no destino".
- Demais argumentos como em `render frame`.

### Comportamento no destino (retomada e proteção)

O diretório recebe `frame_000000.png` … `frame_NNNNNN.png`
([`frame-files.md`](./frame-files.md)). Cada imagem traz a marca do
conjunto (plano + recorte + resolução + versão do desenho).

| Situação do destino | Sem `--overwrite` | Com `--overwrite` |
|---|---|---|
| vazio ou inexistente | desenha todos | desenha todos |
| quadros do **mesmo** conjunto, completos ou parciais | **mantém** os que existem e completos; desenha os que faltam (é a retomada) | redesenha todos e substitui |
| todos os quadros do mesmo conjunto | nada a desenhar (`0 drawn`) | redesenha todos |
| algum quadro de **outro** conjunto (nosso), ou arquivo de nome de quadro que não é nosso | recusa (`37`), nada alterado | desenha todos, substituindo; **remove** os quadros nossos de outro conjunto com número fora do plano |
| arquivo sem o padrão `frame_NNNNNN.png` | nunca alterado | nunca alterado |
| um quadro nosso e do mesmo conjunto, mas truncado/incompleto | é redesenhado | é redesenhado |

Cada imagem é publicada inteira ou não é. Uma interrupção deixa só quadros
completos.

**Nota da sexta etapa**: quadros desenhados antes da versão `2` do desenho (sem a
identificação do plano dentro de cada imagem,
[`frame-files.md`](./frame-files.md)) são de **outro conjunto**: sem
`--overwrite`, `render all` os recusa (`37`); com ele, redesenha tudo.

### Saída (sucesso)

Durante o desenho, em `stderr`, o progresso (item 19 da pesquisa): num terminal,
uma linha reescrita por quadro,

```text
Drawing frame 412/1260 (32.7%), elapsed 00:07:41
```

sem terminal, uma linha nova a cada 10 quadros e no último. Ao final, em
`stdout`, o resumo:

```text
Frames: 1260 requested, 1248 drawn, 12 kept (already in the destination)
Resolution: 1080x1920
Time: 00:31:07
Holes (in the frames drawn now): 87 with missing map tiles, 4 with elevation without value
Destination: /tmp/quadros (frame_000000.png to frame_001259.png)
```

Com sobrescrita que removeu quadros de outro conjunto, uma linha `Removed:
<n> frames from a previous set`. Quando todos os quadros já existiam:
`Frames: 1260 requested, 0 drawn, 1260 kept (already in the destination)` e a
linha `Holes` diz `none drawn now`.

**Código de saída**: `0`.

### Interrupção (`Ctrl+C` ou `SIGTERM`)

O quadro em curso é descartado (nunca fica imagem parcial) e o resumo sai
com o que ficou pronto:

```text
Interrupted: 412 of 1260 frames are ready; run the same command again to continue
Frames: 1260 requested, 412 drawn, 0 kept (already in the destination)
…
```

**Código de saída**: `38`.

## Resolução

`--resolution 1080x1920` (`L` e `A` inteiros, separados por `x` ou `X`).
Largura e altura **pares**, cada uma de **180 a 3840**, com no máximo **8 294
400** pixels no total (3840 × 2160). Padrão **1080x1920** (configuração). O
campo de visão vertical é fixo (45°): mudar a resolução mantém o enquadramento
vertical; a proporção só muda quanto se vê dos lados. Fora dos limites, ímpar,
zero, negativa ou não numérica: erro `34`, com o valor recebido e os limites.

### Aviso de proporção

Se a resolução é **mais estreita** que o vídeo para o qual o plano foi feito
(`parameters.aspect_ratio`, com tolerância de 1%), sai em `stderr`, antes de
desenhar, uma linha `Warning: the plan was made for a 16:9 video, but the
resolution is 1080x1920, which is narrower: the opening and the closing may be
cut at the sides; plan again with --aspect, or draw at a resolution of that
shape`. É só um aviso: o desenho segue, o código de saída não muda. Uma
resolução mais larga que o plano só vê mais dos lados e não avisa.

## Ordem das verificações

Uma recusa nunca grava imagem. Da mais barata à mais cara: resolução (34) →
leitura do plano (17, 18) → número do quadro, em `render frame` (33) → leitura
do recorte (27, 28) → correspondência (29) → cobertura (30) → formato de peça
(31) → elevação (32) → destino (35, 36, 37).

## Saída (erro)

Mensagem em `stderr`, sem imagem criada ou alterada no destino.

| Cenário | Erro do domínio | Código |
|---|---|---|
| Plano ou recorte inexistente / ilegível (E/S) | erro genérico | `4` |
| Uso inválido: argumento ou flag obrigatória faltando | erro de uso da CLI | `2` |
| Arquivo não é um plano / campo ausente / truncado / incoerente | `domain.ErrPlanFileInvalid` | `17` |
| Plano de `format_version` desconhecida | `domain.ErrPlanFormatVersionUnsupported` | `18` |
| Arquivo não é um recorte / truncado / corrompido / incoerente / sem identificação do plano / peça ilegível | `domain.ErrSliceFileInvalid` | `27` |
| Recorte de `format_version` desconhecida | `domain.ErrSliceFormatVersionUnsupported` | `28` |
| Recorte gerado a partir de outro plano | `domain.ErrSliceDoesNotMatchPlan` | `29` |
| Área do recorte não contém a que o plano exige | `domain.ErrSliceDoesNotCoverPlan` | `30` |
| Peças vetoriais (ou formato que não se desenha) | `domain.ErrTileFormatUnsupported` | `31` |
| Recorte sem nenhuma amostra de elevação com valor | `domain.ErrNoElevationData` | `32` |
| Número de quadro inválido | `domain.ErrFrameOutOfRange` | `33` |
| Resolução inválida | `domain.ErrInvalidResolution` | `34` |
| Destino inválido (não é diretório, sem permissão, pasta inexistente no quadro isolado) | `domain.ErrFrameDestinationInvalid` | `35` |
| Arquivo do quadro isolado já existe (sem `--overwrite`) | `domain.ErrFrameDestinationExists` | `36` |
| Diretório com quadros de outro conjunto (sem `--overwrite`) | `domain.ErrFrameSetConflict` | `37` |
| Execução interrompida pelo usuário | `domain.ErrRenderInterrupted` | `38` |

Mensagens (SC-007): a de `27` diz o que está errado (o campo, a contagem que não
bate, a peça — registro, nível e posição —, ou `the slice has no plan
identification; generate it again with "geodata slice --export"`); a de `28`,
a versão encontrada e as aceitas (`found 2, accepted: 1`); a de `29`, que o
recorte foi gerado para outro plano, com as duas identificações abreviadas
(12 caracteres); a de `30`, a área que o plano exige e a que o recorte tem
(o mesmo formato de `Area:` do resumo da etapa 4); a de `31`, o registro e o
formato (`base map "bbbike" has vector tiles (pbf); drawing vector tiles is not
supported yet, use a base map of image tiles (PNG, JPG or WebP)`); a de `32`,
que nenhuma amostra tem valor; a de `33`, o valor e a faixa (`frame number 1260
is out of range, the plan has frames 0 to 1259`; `frame number "3.5" is not a
whole number`); a de `34`, o valor e os limites; a de `36`, o caminho e a dica
`use --overwrite to replace it`; a de `37`, quantos arquivos são de outro
conjunto ou têm nome de quadro sem serem nossos, e `use --overwrite to replace
them, or another --output`.

## Exemplos

```bash
# Fluxo completo: trajeto → plano → recorte → quadros
sobrevoo plan pedalada.gpx --export plano.json
sobrevoo geodata slice plano.json --export recorte.zip
sobrevoo render frame plano.json recorte.zip --number 300 --resolution 960x540 --output conferir.png
sobrevoo render all plano.json recorte.zip --output quadros/

# Continuar depois de uma interrupção: o mesmo comando
sobrevoo render all plano.json recorte.zip --output quadros/

# Refazer tudo no mesmo diretório (por exemplo, em outra resolução)
sobrevoo render all plano.json recorte.zip --output quadros/ --resolution 3840x2160 --overwrite
```
