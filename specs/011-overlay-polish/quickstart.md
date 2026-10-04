# Quickstart: validação manual do Acabamento das Sobreposições de Tela

**Feature**: `011-overlay-polish` | **Data**: 2026-10-02

Checklist manual, com o binário real, para conferir que o acabamento
funciona como o `spec.md` exige: texto vetorial suave e legível sobre
qualquer fundo (História 1), painéis numéricos de largura uniforme
(História 2), marcador do perfil visível (História 3), margem de segurança
adequada ao vídeo vertical (História 4), determinismo byte a byte mantido,
e quadros de antes desta etapa nunca reaproveitados. Não há teste
automatizado de ponta a ponta; ver `plan.md`. Contrato:
[`contracts/frame-files-change.md`](./contracts/frame-files-change.md). Os
itens marcados **(anotar)** são conferidos visualmente na execução, por não
haver uma métrica automática de "serrilhado" ou "legibilidade".

## Pré-requisitos

```sh
make build
BIN=$PWD/bin/sobrevoo
SVHOME=$(mktemp -d)
sv() { HOME=$SVHOME $BIN "$@"; }
A=$PWD/specs/005-frame-rendering/amostras
G=specs/003-camera-path-planning/amostras
go run ./test/samples --out $A
sv geodata register $A/mapa-imagem-sp.mbtiles --name mapa
sv geodata register $A/relevo-sp.tif --name relevo
TRACK=$G/pedalada.gpx

rm -rf /tmp/polish && mkdir /tmp/polish
sv plan $TRACK --duration 10 --fps 10 --export /tmp/polish/plan.json
sv geodata slice /tmp/polish/plan.json --export /tmp/polish/slice.zip
PLAN=/tmp/polish/plan.json
SLICE=/tmp/polish/slice.zip
```

## 1. Texto vetorial suave, sobre fundo claro e escuro (História 1 / FR-001–FR-004 / SC-001 / SC-002)

```sh
sv render frame $PLAN $SLICE --number 100 --output /tmp/polish/claro.png --resolution 1080x1920 \
  --background-color "#E8E4DA"
sv render frame $PLAN $SLICE --number 100 --output /tmp/polish/escuro.png --resolution 1080x1920 \
  --background-color "#101418"
```

**(anotar)**: abrir `claro.png` e `escuro.png` — o contorno de cada letra e
número é suave (sem blocos quadrados de serrilhado); o texto se distingue
claramente da placa e do fundo nos dois casos, sem depender da opacidade da
placa (que não muda em relação a antes desta etapa).

## 2. Painéis numéricos de largura uniforme, mesmo quando o texto cresce (História 2 / FR-005 / SC-003)

```sh
sv render frame $PLAN $SLICE --number 10 --output /tmp/polish/inicio.png --resolution 1080x1920
FRAME_COUNT=$(grep -o '"frame_count": [0-9]*' $PLAN | grep -o '[0-9]*')
sv render frame $PLAN $SLICE --number $(( FRAME_COUNT - 10 )) --output /tmp/polish/fim.png --resolution 1080x1920
```

**(anotar)**: em `inicio.png` e em `fim.png`, os três painéis numéricos
(distância; elevação e ganho; tempo decorrido) têm exatamente a mesma
largura entre si em cada imagem — mesmo em `fim.png`, onde a distância e o
tempo decorrido têm mais dígitos que em `inicio.png`, os três continuam
alinhados, sem nenhum painel sobrando ou faltando em relação aos outros.

## 3. Marcador do perfil visível na menor resolução aceita (História 3 / FR-006 / SC-004)

```sh
sv render frame $PLAN $SLICE --number 100 --output /tmp/polish/pequeno.png --resolution 180x320
sv render frame $PLAN $SLICE --number 100 --output /tmp/polish/grande.png --resolution 1080x1920
```

**(anotar)**: em `pequeno.png` (a menor resolução que a ferramenta aceita),
o marcador sobre o perfil de elevação ainda é claramente visível, nunca um
ponto que desaparece no gráfico; em `grande.png`, na mesma proporção 9:16,
o marcador ocupa visualmente a mesma fração do quadro.

## 4. Margem de segurança maior na base, em vídeo vertical (História 4 / FR-007 / SC-005)

```sh
sv render frame $PLAN $SLICE --number 100 --output /tmp/polish/vertical.png --resolution 1080x1920
```

**(anotar)**: em `vertical.png`, a distância entre a sobreposição mais
baixa (o painel de tempo decorrido ou o perfil, o que estiver mais abaixo)
e a borda inferior é visivelmente maior que a distância entre a
sobreposição mais alta e o topo, ou entre qualquer sobreposição e as
laterais — simular o corte de legenda/botões de uma rede social (cobrir
~15% da altura a partir da base) e conferir que nada é cortado.

## 5. Determinismo byte a byte preservado (SC-007)

```sh
sv render frame $PLAN $SLICE --number 100 --output /tmp/polish/det1.png --resolution 1080x1920
sv render frame $PLAN $SLICE --number 100 --output /tmp/polish/det2.png --resolution 1080x1920 --overwrite

cmp /tmp/polish/det1.png /tmp/polish/det2.png && echo "IDÊNTICOS" || echo "DIFERENTES (falha — anotar o diff)"
```

**(anotar)**: esperado `IDÊNTICOS`. Se o binário foi compilado e executado
em outra arquitetura (amd64 vs. arm64) para o mesmo plano/recorte/
resolução/aparência/sobreposição, repetir e comparar os dois arquivos
também devem ser idênticos — esta é a garantia central da etapa
(`research.md` item 1).

## 6. Nenhuma sobreposição some; `--overlays=false` continua inalterado

```sh
sv render frame $PLAN $SLICE --number 100 --output /tmp/polish/sem.png --resolution 1080x1920 --overlays=false
```

**(anotar)**: `sem.png` não tem nenhum texto, painel ou marcador de perfil
— exatamente como antes desta etapa; esta etapa não introduz nenhuma
mudança quando as sobreposições estão desligadas.

## 7. Quadros de uma versão anterior nunca são reaproveitados (FR-008 / SC-006)

```sh
rm -rf /tmp/polish/antigo && mkdir /tmp/polish/antigo
sv render all $PLAN $SLICE --output /tmp/polish/antigo --resolution 360x640

# Simula quadros de antes desta etapa alterando o RenderVersion embutido
# não é possível sem recompilar uma versão anterior; na prática, valide
# isto comparando um checkout do commit anterior a esta etapa: gere
# /tmp/polish/antigo com o binário antigo, troque para o binário novo e
# rode de novo sem --overwrite.

sv render all $PLAN $SLICE --output /tmp/polish/antigo --resolution 360x640; echo "código: $?"   # esperado: 37 (ErrFrameSetConflict)
sv render all $PLAN $SLICE --output /tmp/polish/antigo --resolution 360x640 --overwrite             # redesenha todos
```

**(anotar)**: sem `--overwrite`, a segunda execução com o binário novo
recusa com o código `37`, citando quadros de outro conjunto; com
`--overwrite`, redesenha todos os quadros com o acabamento novo.

## 8. `fly --keep`: só o acabamento muda → quadros e vídeo refeitos, plano e recorte reaproveitados

```sh
sv fly $TRACK --output /tmp/polish/voo-antigo.mp4 --duration 10 --fps 10 --resolution 360x640 \
  --keep /tmp/polish/keep

md5=$(md5sum /tmp/polish/keep/plan.json /tmp/polish/keep/slice.zip)

# Com o binário novo (RenderVersion 3), reaproveitando o mesmo --keep de
# uma execução feita com o binário anterior a esta etapa:
sv fly $TRACK --output /tmp/polish/voo-novo.mp4 --duration 10 --fps 10 --resolution 360x640 \
  --keep /tmp/polish/keep --overwrite

md5 -c <<< "$md5" 2>/dev/null || echo "(conferir manualmente: plan.json e slice.zip inalterados; voo-novo.mp4 difere de voo-antigo.mp4 no acabamento das sobreposições)"
```

**(anotar)**: `plan.json` e `slice.zip` continuam com o mesmo conteúdo
antes e depois (o acabamento não depende de nenhum dos dois); os quadros e
`voo-novo.mp4` saem com o acabamento desta etapa.
