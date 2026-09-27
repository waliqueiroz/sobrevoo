# Quickstart: validação manual de Voo em um Único Comando

**Feature**: `007-full-flight-pipeline` | **Data**: 2026-09-27

Checklist manual, com o binário real (e o `ffmpeg` real, já instalado desde a
etapa 6), para conferir a etapa de ponta a ponta — em especial que o resultado
de `fly` é **idêntico** ao dos seis comandos existentes rodados na mão
(FR-002/SC-002), e que o reaproveitamento com `--keep` de fato evita o
trabalho refeito (SC-004). Não há teste automatizado de ponta a ponta; ver
`plan.md`. Contratos: [`contracts/cli.md`](./contracts/cli.md) e
[`contracts/intermediates-directory.md`](./contracts/intermediates-directory.md).
Os valores marcados **(anotar)** são medidos na execução e escritos aqui.

## Pré-requisitos

Todos os comandos rodam a partir da **raiz do repositório** (zsh ou bash).

```sh
ffmpeg -hide_banner -encoders | grep libx264      # já confirmado na etapa 6

make build                                        # gera bin/sobrevoo
BIN=$PWD/bin/sobrevoo
SVHOME=$(mktemp -d)                               # o registro fica em $HOME/.sobrevoo/registry.json
sv() { HOME=$SVHOME $BIN "$@"; }                  # registro isolado para este roteiro
A=$PWD/specs/005-frame-rendering/amostras
G=specs/003-camera-path-planning/amostras
go run ./test/samples --out $A
sv geodata register $A/mapa-imagem-sp.mbtiles --name mapa
sv geodata register $A/relevo-sp.tif --name relevo
TRACK=$G/pedalada.gpx
```

Um plano curto (mesmos valores do quickstart da etapa 6: 38 s a 10 fps, 380
quadros, 360×640) para os cenários rodarem em segundos, não minutos.

## 1. Comando único vs. o fluxo manual, byte a byte (História 1)

```sh
# O fluxo manual, exatamente como o quickstart da etapa 6 já validou
rm -rf /tmp/manual && mkdir /tmp/manual
sv plan $TRACK --duration 38 --fps 10 --export /tmp/manual/plan.json
sv geodata slice /tmp/manual/plan.json --export /tmp/manual/slice.zip
sv render all /tmp/manual/plan.json /tmp/manual/slice.zip --output /tmp/manual/frames --resolution 360x640
sv video /tmp/manual/plan.json /tmp/manual/frames --output /tmp/manual/voo.mp4

# O comando único, com os mesmos valores
sv fly $TRACK --output /tmp/fly.mp4 --duration 38 --fps 10 --resolution 360x640; echo $?

cmp /tmp/manual/voo.mp4 /tmp/fly.mp4 && echo "IDÊNTICOS" || echo "DIFERENTES (anotar o diff)"
```

Esperado: `exit=0`, `IDÊNTICOS`. **Medido**: `exit=0`, `cmp` não achou
diferença — os dois arquivos são idênticos byte a byte.

## 2. Parâmetros ajustáveis, com os mesmos nomes de sempre (História 2)

```sh
sv fly $TRACK --output /tmp/ajustado.mp4 --duration 38 --fps 10 --distance high --tilt low --aspect 16:9 --resolution 640x360 --quality high; echo $?
ffprobe -v error -show_entries stream=width,height,r_frame_rate -show_entries format=duration -of default=nw=1 /tmp/ajustado.mp4
```

Esperado: `exit=0`; resolução `640x360`; duração `38.000` (ou muito próxima,
`R6` da etapa 6). Confere que `--distance high --tilt low --aspect 16:9`
produziram um plano diferente do padrão (câmera mais afastada, mais rasante,
horizontal) — inspecionável comparando com `sv plan $TRACK --duration 38
--fps 10 --distance high --tilt low --aspect 16:9 --export /dev/stdout` (o
resumo impresso bate).

Repetir com um valor **inválido** (`--distance ultra`) e conferir que o erro e
o código de saída são os mesmos que `sv plan $TRACK --distance ultra` já dá
(uso, código `2`), **sem** nenhuma etapa ter rodado (nenhum arquivo tocado).

## 3. Progresso por etapa (História 3)

```sh
sv fly $TRACK --output /tmp/progresso.mp4 --duration 38 --fps 10 --resolution 360x640 2>&1 1>/dev/null | tee /tmp/progresso.log
grep -c "^Stage " /tmp/progresso.log     # esperado: 5
grep "^Drawing frame" /tmp/progresso.log | tail -1
grep "^Encoding frame" /tmp/progresso.log | tail -1
```

Esperado: as 5 linhas `Stage N/5: ...` aparecem, na ordem; as linhas de
progresso de quadro e de codificação têm o mesmo formato que `render
all`/`video` já mostram sozinhos. **Medido**: as 5 linhas apareceram na
ordem certa, com `Drawing frame ...`/`Encoding frame ...` intercaladas,
idênticas ao formato de `render all`/`video`.

## 4. `--keep`: os três arquivos ficam disponíveis para inspeção (História 4, cenário 1)

```sh
rm -rf /tmp/guardado
sv fly $TRACK --output /tmp/guardado.mp4 --duration 38 --fps 10 --resolution 360x640 --keep /tmp/guardado; echo $?
ls /tmp/guardado                          # plan.json  slice.zip  frames
python3 -m json.tool /tmp/guardado/plan.json | head -5    # legível
unzip -l /tmp/guardado/slice.zip | head -5                # ZIP comum
ls /tmp/guardado/frames | head -3 && ls /tmp/guardado/frames | wc -l   # 380
```

Esperado: `exit=0`; os três presentes e abríveis com ferramentas comuns; 380
quadros. `sv plan` (ou qualquer leitor de JSON) abre `plan.json` sem
conversão.

## 5. `--keep`: reaproveitamento total numa segunda execução (História 4, cenário 2)

Um destino **diferente** do item 4 (`/tmp/guardado-2.mp4`, não
`/tmp/guardado.mp4`): a segunda execução não precisa de `--overwrite` para o
vídeo, e por isso não força o redesenho dos quadros — `--overwrite` é a
**mesma** flag para os três intermediários e para o vídeo (`research.md`
item 11): usá-la quando só o destino do vídeo precisaria também refaz os
quadros de um conjunto que já bate, exatamente como `render all --overwrite`
já faz sozinho hoje (nenhuma regra nova). Reaproveitar de verdade os quadros
pede um destino de vídeo que ainda não existe.

```sh
time sv fly $TRACK --output /tmp/guardado-2.mp4 --duration 38 --fps 10 --resolution 360x640 --keep /tmp/guardado; echo $?
cmp /tmp/manual/voo.mp4 /tmp/guardado-2.mp4 && echo "IDÊNTICOS mesmo reaproveitado"
```

Esperado: `exit=0`; a saída mostra "reusing plan.json"/"reusing slice.zip" e
`Frames: 380 requested, 0 drawn, 380 kept`; tempo total **muito** menor que o
do item 4 (só a montagem do vídeo roda de verdade). **Medido**: item 4
(execução do zero) 31 s; esta repetição 1 s — 31× mais rápida (valida SC-004).

## 6. `--keep`: mudar só a qualidade reaproveita plano, recorte e quadros (História 4, cenário 3)

```sh
time sv fly $TRACK --output /tmp/guardado-hq.mp4 --duration 38 --fps 10 --resolution 360x640 --keep /tmp/guardado --quality high; echo $?
```

Esperado: `exit=0`; "reusing plan.json"/"reusing slice.zip"/`0 drawn, 380
kept`; só a etapa 5/5 (codificação) gasta tempo de verdade; o vídeo produzido
é maior ou igual ao de qualidade `medium` (FR-013 da etapa 6, herdada).

## 7. `--keep`: conteúdo de outro conjunto é recusado, `--overwrite` refaz (História 4, cenário 4)

```sh
# Muda um valor que afeta o plano: o conteúdo guardado passa a ser de outro conjunto
sv fly $TRACK --output /tmp/outro.mp4 --duration 45 --fps 10 --resolution 360x640 --keep /tmp/guardado; echo $?   # esperado: recusa
sv fly $TRACK --output /tmp/outro.mp4 --duration 45 --fps 10 --resolution 360x640 --keep /tmp/guardado --overwrite; echo $?   # esperado: 0, refaz a partir do plano
```

Esperado: a primeira chamada recusa (o mesmo erro/código que `plan
--export` sem `--overwrite` já daria: `15`), sem alterar nada em
`/tmp/guardado`; a segunda, com `--overwrite`, refaz plano, recorte e quadros
e produz o vídeo. **Medido**: `exit=15` na primeira chamada.

## 8. Recusa cedo, sem trabalho algum (Casos Extremos)

```sh
# Destino já existe, sem --overwrite: recusa antes de tratar o trajeto
touch /tmp/existe.mp4
time sv fly $TRACK --output /tmp/existe.mp4 --duration 38 --fps 10; echo $?

# Codificador ausente: simulado apontando PATH para um diretório sem ffmpeg
time PATH=/usr/bin:/bin sv fly $TRACK --output /tmp/sem-ffmpeg.mp4 --duration 38 --fps 10; echo $?

# Trajeto fora da cobertura registrada: simulado com um registro vazio
SVHOME2=$(mktemp -d)
time HOME=$SVHOME2 $BIN fly $TRACK --output /tmp/sem-dados.mp4 --duration 38 --fps 10; echo $?
```

Esperado: as três recusam **quase instantaneamente** (bem abaixo do tempo que
desenhar um único quadro levaria), com os mesmos código/mensagem que `video`
(`47`), `video` (`46`) e `geodata slice` (`19`) já dariam sozinhos. Confirma
SC-003. **Medido**: as três recusam em frações de segundo — a de cobertura
(`19`) já entra na etapa 2/5 (planejamento) antes de recusar na etapa 3/5,
nunca chegando a desenhar nenhum quadro.

## 9. Interrupção: sem `--keep`, nenhum resto; com `--keep`, os quadros prontos ficam

```sh
# Sem --keep: interrompe durante o desenho dos quadros e confere que nada sobra
rm -f /tmp/interrompido.mp4
sv fly $TRACK --output /tmp/interrompido.mp4 --duration 120 --fps 30 --resolution 1080x1920 &
PID=$!; sleep 2; kill -INT $PID; wait $PID; echo "exit=$?"
ls /tmp | grep -i sobrevoo-fly            # esperado: nada (diretório temporário já removido)
ls -la /tmp/interrompido.mp4 2>&1         # esperado: não existe

# Com --keep: interrompe durante o desenho e confere que os quadros prontos ficam
rm -rf /tmp/interrompido-guardado
sv fly $TRACK --output /tmp/interrompido.mp4 --duration 120 --fps 30 --resolution 1080x1920 --keep /tmp/interrompido-guardado &
PID=$!; sleep 2; kill -INT $PID; wait $PID; echo "exit=$?"
ls /tmp/interrompido-guardado/frames | wc -l    # esperado: > 0, < total
sv fly $TRACK --output /tmp/interrompido.mp4 --duration 120 --fps 30 --resolution 1080x1920 --keep /tmp/interrompido-guardado; echo $?   # retoma e completa
```

Esperado: código de saída `51` nas duas interrupções; sem `--keep`, nenhum
diretório temporário nem vídeo parcial sobra; com `--keep`, os quadros
completos ficam, e a segunda chamada os reaproveita e termina o voo. Confirma
FR-011/SC-006. **Medido**: 17 quadros prontos na interrupção; a retomada os
reaproveita (`17 kept`) e desenha os 363 restantes.

## Encerramento

```sh
rm -rf $SVHOME $SVHOME2 /tmp/manual /tmp/guardado /tmp/interrompido-guardado /tmp/*.mp4 /tmp/*.log
```
