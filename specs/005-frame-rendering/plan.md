# Plano de Implementação: Desenho dos Quadros do Voo

**Branch**: `005-frame-rendering` | **Data**: 2026-09-26 | **Especificação**: [spec.md](./spec.md)

**Entrada**: Especificação de funcionalidade de `/specs/005-frame-rendering/spec.md`

## Resumo

Quinta etapa do Sobrevoo: um comando pai novo, `render`, com dois filhos.
`sobrevoo render frame <plano> <recorte> --number N --output arquivo.png`
desenha **um** quadro do voo; `sobrevoo render all <plano> <recorte> --output
diretório` desenha **todos**, com progresso, retomada e resumo. As entradas são
o plano exportado pela etapa 3 e o recorte exportado pela etapa 4; a saída são
imagens PNG (`frame_NNNNNN.png`), uma por quadro, que a etapa seguinte juntará
num vídeo. Nada é baixado, codificado em vídeo, sobreposto com texto ou
sonorizado.

A abordagem técnica (detalhada em `research.md`):

- O quadro é desenhado por **lançamento de raios** sobre a superfície de terreno
  das grades de elevação, em Go puro, **no domínio** (item 1): cada pixel é
  independente, então a oclusão é exata, a falta de dado é uma consulta por
  célula e peça, e o resultado não depende de quantas goroutines ou de que
  ordem de quadros (FR-015).
- A geometria é um **plano local por quadro**, equiretangular em torno da
  latitude da câmera, sem curvatura, igual em qualquer lugar do planeta
  (item 2); a câmera vem do quadro do plano — rumo, inclinação, campo de visão
  vertical fixo de 45° (a mesma constante das etapas 3 e 4) —, com a altura
  absoluta obtida somando a elevação sob o **alvo** (recuperado do próprio
  quadro) à altura relativa do plano (item 3).
- A superfície é **bilinear entre os centros das células** e se estende até as
  bordas (item 4); o raio a atravessa **quadra a quadra (DDA)**, com poda pela
  maior altura (item 5). Células **sem valor** têm geometria preenchida pela
  amostra com valor mais próxima e são **marcadas** com um xadrez magenta; peças
  **ausentes** (ou inexistentes além de ±85,0511°), com uma hachura cinza
  (itens 6 e 8). Os quadros com buraco são contados sobre os mesmos pixels que
  aparecem (item 10).
- O mapa é **decodificado sob demanda** por uma porta nova, `TileDecoder`
  (PNG, JPEG e WebP), e filtrado em **trilinear** com mips por peça (item 7);
  peças vetoriais são recusadas antes de desenhar (item 16).
- O **traçado** cresce a cada quadro (só o trecho já percorrido) e o
  **marcador** é um disco, ambos com oclusão do terreno e borda suave (item 9).
- O recorte passa a guardar a **identificação do plano** (`plan_id`, hash do
  conteúdo do plano), o que permite recusar um recorte de outro plano
  (Clarificação de 2026-09-26; item 14). É a única mudança na etapa 4.
- Um **leitor de recorte** ZIP valida o arquivo por inteiro antes de desenhar
  (item 15). Cada imagem é gravada de forma **atômica** e carrega, num bloco
  `tEXt`, a identificação do **conjunto** (plano + recorte + resolução +
  versão do desenho), o que dá a retomada, a recusa de misturar voos e a
  remoção segura das sobras, sem arquivo de registro (itens 17 e 18).

A arquitetura hexagonal é preservada: o núcleo ganha entidades e funções puras
(`Scene`, `Resolution`, `FrameSetID`, `FrameDirectory`, `RenderSummary`,
`RenderTuning`, mais métodos em `CameraPlan`, `GeoSlice` e `BoundingBox`), quatro
portas (`GeoSliceReader`, `TileDecoder`, `FrameRepository`, `FrameExporter`), doze
erros e um serviço novo por recurso — `FrameService` (`DrawFrame`,
`DrawFrames`) — mais um método em `GeoSliceService` (`Load`). Toda E/S fica nos
adapters; a CLI só traduz flags, texto, sinais e códigos de saída.

## Contexto Técnico

**Linguagem/Versão**: Go 1.26 (sem mudança)

**Dependências Principais**: nenhuma nova. `golang.org/x/image` (já no módulo
desde a etapa 4) fornece o decodificador WebP; PNG e JPEG vêm da biblioteca
padrão (`image/png`, `image/jpeg`), como `archive/zip`, `crypto/sha256`,
`context`, `sync`, `os/signal`. Cobra, testify e `go.uber.org/mock` como antes.

**Armazenamento**: nenhum estado persistente novo. Lê o plano e o recorte
(somente leitura) e grava imagens PNG (atômicas, sem sobrescrita por padrão, com
a marca do conjunto dentro de cada uma; sem arquivo de registro). O registro da
etapa 2 **não** é lido.

**Testes**: mesmo padrão — `go test` com testify, given/when/then, um `t.Run`
por cenário, sem testes tabulares. Builders em `builddomain` (novos:
`RenderTuningBuilder`, `FrameDirectoryBuilder`; estendidos:
`GeoSliceBuilder` com `PlanID`); mocks gerados das quatro portas novas
(`mockdomain`) e de `FrameService` (`mockapplication`). O domínio é testado
sobre **cenas pequenas** (algumas células, 64 × 36 pixels) com terrenos de
fórmula conhecida (plano, rampa, pirâmide, degrau entre grades), com **testes de
propriedade**: o marcador cai a ≤ 1 px da projeção (SC-004); o traçado do quadro
`k` contém o de `k−1`; o resultado é igual byte a byte com 1 e com 8
goroutines e em qualquer ordem de quadros (FR-015); a poda por altura não muda
nenhum pixel; deslocar o voo para outras regiões (inclusive `lon` em torno de
180 e `lat` > 80) mantém as propriedades e não deixa emenda (SC-009). Os
adapters (`zipfile.GeoSliceReader`, `tiledecoder`, `pngfile`) são testados com
**fixtures em código** em `test/helper`, em diretório temporário. Serviços,
com as portas mockadas; a CLI, com os serviços mockados. Não há teste
automatizado de ponta a ponta; `quickstart.md` é o checklist manual.

**Plataforma-Alvo**: mesmo binário multiplataforma (Linux, macOS, Windows).

**Tipo de Projeto**: CLI (projeto único em Go, sem frontend/backend).

**Metas de Desempenho** (referência inicial, a confirmar com medição na
implementação: `research.md` item 23): quadro isolado em ≤ 10 s (SC-001);
nenhum quadro além de 15 s (SC-011); 1350 quadros (45 s a 30 fps) em ≤ 45 min
(SC-012), em 1080p, em computador pessoal comum; recusas por vetorial, cobertura
ou correspondência em < 5 s (SC-007). Um quadro usa todos os núcleos (faixas de
linhas), um quadro por vez.

**Restrições**: 100% offline; núcleo sem I/O; determinismo bit a bit no mesmo
binário e plataforma (`research.md` item 12: aritmética sem fusão, só funções
matemáticas de Go puro, sem estado que influencie valores); nenhum dado de
região embutido; recorte de entrada de no máximo 256 MiB (o teto da etapa 4);
resolução de 180 a 3840 por lado, pares, até 8 294 400 pixels; sem iluminação,
sombras, curvatura da Terra nem suavização das bordas do terreno; somente peças
em imagem (PNG, JPEG, WebP).

**Escala/Escopo**: uso pessoal — voos de centenas a milhares de quadros
(432 000 no máximo, o teto do plano); recortes de até 256 MiB; memória por
execução: o recorte + a cópia preenchida das grades com buracos + um cache de
peças de 256 MiB + os buffers de um quadro (~58 MB em 1080p).

## Verificação da Constituição

*PORTÃO: Deve passar antes da Fase 0 de pesquisa. Reverificar após o design da Fase 1.*

Avaliada antes da pesquisa e **reavaliada após o design** (data-model,
contratos e quickstart); o resultado não mudou.

| Princípio | Avaliação | Como o design atende |
|---|---|---|
| I. Arquitetura Hexagonal | PASSA | Toda a regra — câmera, projeção, superfície, DDA, texturização, marcações, traçado, contagem de buracos, identificação do plano, do recorte e do conjunto, decisão do que manter/desenhar/remover — é função pura ou método de entidade em `internal/domain`, sem importar `os`, `archive/zip`, `image/png`, `image/jpeg`, `golang.org/x/image` nem `internal/infra`. `FrameService` só chama portas e métodos de domínio, na ordem certa. ZIP, decodificação de imagem, codificação PNG, sistema de arquivos e configuração ficam nos adapters. Concorrência (`sync`, goroutines) e `context` são biblioteca padrão pura, sem E/S. |
| II. Portas para Toda Dependência Externa | PASSA | Toda E/S nova passa por porta declarada no domínio: ler o recorte (`GeoSliceReader`), decodificar peças (`TileDecoder`, formato de arquivo como o parser de GPX), persistir quadros num diretório (`FrameRepository`) e gravar um quadro isolado (`FrameExporter`). O plano continua lido por `CameraPlanReader`. Nenhuma porta para `crypto/sha256`, `math`, `time.Now()` (o tempo gasto) ou `context`, que são funções puras/estado da biblioteca padrão. |
| III. Entrypoints Descartáveis | PASSA | `FrameService` recebe e devolve tipos de domínio (`CameraPlan`, `GeoSlice`, `SingleFrameRequest`, `FrameSetRequest`, `RenderSummary`); nada de flags, texto, terminal ou códigos de saída. `render frame` e `render all` só traduzem flags → chamadas de serviço, o progresso → linhas no `stderr`, sinais → `context`, resultados/erros → texto/código. O serviço não sabe de onde o plano e o recorte vieram: um comando futuro que encadeie tudo em memória o reusa. |
| IV. Neutralidade Geográfica | PASSA | Nenhum dado, constante ou caso especial de região. O plano local é o mesmo em qualquer hemisfério, com a longitude relativa à da câmera e normalizada (sem caso do antimeridiano) e um piso em `cos φ` para os polos (o mesmo da etapa 4); o limite de ±85,0511° é propriedade do esquema de peças. SC-009 verifica voos equivalentes deslocados pelo planeta, inclusive em 180° e acima de 80°. |
| V. Funcionamento Offline | PASSA | Só lê o plano e o recorte informados; não abre os arquivos registrados, não usa rede, chave nem serviço, não baixa nada (FR-001). O relógio só mede o tempo gasto e nunca entra nas imagens. |
| VI. Testes Automatizados no Núcleo | PASSA | Domínio testado sem I/O (cenas em memória, `TileDecoder` mockado ou uma implementação de teste em memória); `FrameService` testado com as portas mockadas (`mockdomain`); nenhum teste do núcleo toca disco, rede ou processo externo. |
| VII. Erros Sentinela no Domínio | PASSA | `ErrSliceFileInvalid`, `ErrSliceFormatVersionUnsupported`, `ErrSliceDoesNotMatchPlan`, `ErrSliceDoesNotCoverPlan`, `ErrTileFormatUnsupported`, `ErrNoElevationData`, `ErrFrameOutOfRange`, `ErrInvalidResolution`, `ErrFrameDestinationInvalid`, `ErrFrameDestinationExists`, `ErrFrameSetConflict`, `ErrRenderInterrupted` em `errors.go`; a CLI os traduz em códigos 27–38 (`contracts/cli.md`). Erros de `os`, ZIP, PNG e JSON são traduzidos nos adapters; o núcleo nunca vê um deles. |
| VIII. Configuração Injetada | PASSA | Constantes heurísticas em `domain.RenderTuning` (formato definido pelo núcleo). `config.RenderTuning` e `config.RenderDefaults` têm **tipos próprios** e não importam o domínio; o composition root (`cmd/sobrevoo/config_mapping.go`) os mapeia e injeta em `NewFrameService`. O número de goroutines (`runtime.NumCPU()`) é lido na configuração, não no domínio. Caminhos, número do quadro, resolução (texto), `--overwrite` e o contexto chegam ao núcleo por parâmetro; o núcleo não lê flag, ambiente, arquivo nem sinal. |
| IX. Organização de Portas, Service Layer e Mocks | PASSA | Sem `ports.go`: cada porta no arquivo da entidade que produz, no topo, com `//go:generate` logo após `package` (`GeoSliceReader` em `geo_slice.go`, ao lado de `GeoSliceExporter`; `TileDecoder` em `tile_image.go`; `FrameRepository` e `FrameExporter` em `frame_set.go`). Nomeadas pelo papel (`Reader`, `Decoder`, `Repository` para a persistência dos quadros, `Exporter`). Um serviço por recurso, sem `Execute`: `FrameService`/`frameService`/`NewFrameService` (`DrawFrame`, `DrawFrames`) e um método novo em `GeoSliceService` (`Load`) — nenhum serviço por caso de uso. Toda regra em `internal/domain` (`Scene`, `Resolution`, `FrameFileName`, `FrameDirectory.Plan`, `CameraPlan.ID`, `GeoSlice.Ensure*`, `RenderSummary.Add`); `application` só chama portas e métodos de domínio. Adapters: `zipfile.NewGeoSliceReader()` (pacote de tecnologia que já serve outra porta: construtor pela porta), `pngfile.NewFrameRepository()` e `pngfile.NewFrameExporter()` (tecnologia PNG, duas portas: construtor pela porta), `tiledecoder.NewRaster()` (pacote leva o nome da porta, tipo/construtor só a estratégia, arquivo `raster_tile_decoder.go`); `atomicfile` continua utilitário. Nenhum nome exportado repete o pacote. Mocks por `//go:generate` acima de cada interface, em `mockdomain`/`mockapplication`. Receivers: `s` (`Scene`, `frameService`, `geoSliceService`), `c` (`CameraPlan`, `camera`), `g` (`GeoSlice`), `b` (`BoundingBox`), `r` (`Resolution`, readers/repositories), `d` (`FrameDirectory`), `t` (`RenderTuning`), `e` (exporters). |
| X. Testes: Given/When/Then, Builders e Isolamento por Camada | PASSA | Todo teste em `t.Run("should ...")` com `// given`/`// when`/`// then`, sem tabelas; builders em `builddomain`; domínio/serviços com portas mockadas; CLI com serviços mockados (`mockapplication`); adapters com diretório temporário (é o adapter que toca disco, então é o objeto do teste). Sem teste de ponta a ponta automatizado; `quickstart.md` cobre o manual. |
| Idioma dos Artefatos | PASSA | Artefatos do Spec Kit em português; identificadores, pacotes, arquivos, comentários e mensagens de commit em inglês; I/O em tempo de execução (flags, saída, erros) em inglês, como nas etapas anteriores. |

Nenhuma violação identificada. Rastreamento de Complexidade não se aplica.

## Estrutura do Projeto

### Documentação (desta funcionalidade)

```text
specs/005-frame-rendering/
├── plan.md              # Este arquivo (saída do comando /speckit-plan)
├── research.md          # Saída da Fase 0 (comando /speckit-plan)
├── data-model.md        # Saída da Fase 1 (comando /speckit-plan)
├── quickstart.md        # Saída da Fase 1 (comando /speckit-plan)
├── contracts/
│   ├── cli.md                 # Saída da Fase 1: comandos `render frame` e `render all`
│   ├── frame-files.md         # Saída da Fase 1: formato das imagens, nomes, cores, marca do conjunto
│   └── slice-file-change.md   # Saída da Fase 1: o campo `plan_id` acrescentado ao recorte da etapa 4
├── checklists/
│   └── requirements.md  # Gerado por /speckit-specify
├── amostras/            # Gerado pela ferramenta `test/samples` (ver quickstart); não versionar binários
└── tasks.md             # Saída da Fase 2 (comando /speckit-tasks - NÃO criado pelo /speckit-plan)
```

### Código-Fonte (raiz do repositório)

```text
cmd/sobrevoo/
├── main.go                                    # (alterado) monta o leitor de recorte, o decodificador, os repositórios/exportadores de PNG, FrameService e os comandos "render frame" e "render all"; passa o leitor a GeoSliceService
└── config_mapping.go                          # (alterado) mapeia config.RenderTuning → domain.RenderTuning e config.RenderDefaults

internal/
├── domain/
│   ├── camera_plan.go                         # (estendido) CameraPlan.ID()
│   ├── bounding_box.go                        # (estendido) ContainsBox
│   ├── geo_slice.go                           # (estendido) porta GeoSliceReader no topo; GeoSlice.PlanID, ContentID; EnsureMatches, EnsureCovers, EnsureDrawable
│   ├── tile_image.go                          # NOVO: porta TileDecoder (no topo), TileImage
│   ├── frame_resolution.go                    # NOVO: Resolution, NewResolution, ParseResolution, limites
│   ├── render_tuning.go                       # NOVO: RenderTuning (+ Fingerprint), RenderVersion, cores e padrões (constantes)
│   ├── frame_image.go                         # NOVO: FrameImage, FrameStats
│   ├── frame_camera.go                        # NOVO: câmera do quadro (base, raio do pixel, projeção, alvo, altura absoluta)
│   ├── frame_surface.go                       # NOVO: superfície bilinear, preenchimento das amostras sem valor, DDA por quadra, poda por altura
│   ├── frame_imagery.go                       # NOVO: índice de peças, mapeamento Web Mercator, mips, filtro trilinear, cache
│   ├── frame_overlay.go                       # NOVO: traçado (cápsulas) e marcador, teste de profundidade
│   ├── frame_scene.go                         # NOVO: Scene, NewScene, Scene.Render (faixas de linhas, marcações, contagem)
│   ├── frame_set.go                           # NOVO: portas FrameRepository e FrameExporter (no topo), FrameSetID, FrameFileName, ParseFrameFileName, ParseFrameNumber, FrameFile, FrameDirectory (+Plan), FrameWork, SingleFrameRequest, FrameSetRequest
│   ├── render_summary.go                      # NOVO: RenderSummary (+Add), RenderProgress
│   ├── errors.go                              # (estendido) doze sentinelas novos
│   ├── *_test.go                              # NOVOS/estendidos: um por arquivo acima (funções puras + propriedades)
│   ├── builddomain/
│   │   ├── render_tuning_builder.go           # NOVO
│   │   ├── frame_directory_builder.go         # NOVO
│   │   └── geo_slice_builder.go               # (estendido) WithPlanID
│   └── mockdomain/
│       ├── geo_slice_reader.go                # GERADO (make generate)
│       ├── tile_decoder.go                    # GERADO
│       ├── frame_repository.go                # GERADO
│       └── frame_exporter.go                  # GERADO
├── application/
│   ├── frame_service.go                       # NOVO: FrameService / frameService / NewFrameService (DrawFrame, DrawFrames)
│   ├── frame_service_test.go                  # NOVO
│   ├── geo_slice_service.go                   # (alterado) Load(path); recebe GeoSliceReader; Generate grava PlanID
│   ├── geo_slice_service_test.go              # (alterado)
│   └── mockapplication/
│       ├── frame_service.go                   # GERADO
│       └── geo_slice_service.go               # REGERADO (novo método)
└── infra/
    ├── inbound/cli/
    │   ├── render.go                          # NOVO: comando pai "render"
    │   ├── render_frame.go                    # NOVO: "render frame" (--number, --output, --resolution, --overwrite), resumo
    │   ├── render_all.go                      # NOVO: "render all", contexto de sinais, progresso no stderr, resumo
    │   ├── render_*_test.go                   # NOVOS
    │   └── exit_code.go                       # (estendido) códigos 27–38
    └── outbound/
        ├── config/config.go                   # (estendido) Config.RenderTuning, Config.RenderDefaults (tipos do próprio pacote); FOV compartilhado com CameraTuning
        ├── atomicfile/
        │   ├── atomic_file.go                 # (alterado) Sync do temporário antes de publicar
        │   └── atomic_file_test.go            # (estendido)
        ├── zipfile/
        │   ├── geo_slice_exporter.go          # (alterado) escreve plan_id
        │   ├── geo_slice_exporter_test.go     # (alterado)
        │   ├── geo_slice_reader.go            # NOVO: lê o ZIP, valida, calcula ContentID
        │   └── geo_slice_reader_test.go       # NOVO
        ├── tiledecoder/
        │   ├── raster_tile_decoder.go         # NOVO: png, jpg, webp → TileImage
        │   └── raster_tile_decoder_test.go    # NOVO
        └── pngfile/
            ├── png_mark.go                    # NOVO: codifica RGB com o bloco tEXt; lê a marca, a resolução e o IEND
            ├── frame_repository.go            # NOVO: Inspect, Save, Remove (usa atomicfile)
            ├── frame_exporter.go              # NOVO: Export (usa atomicfile)
            └── *_test.go                      # NOVOS

test/
├── helper/
│   ├── slice_fixture.go                       # NOVO: bytes de um recorte ZIP (válido, truncado, versão 2, sem plan_id, contagens erradas, peça corrompida)
│   ├── png_fixture.go                         # NOVO: peças PNG (cor lisa, xadrez com tom por posição), JPEG e WebP mínimos
│   └── mbtiles_fixture.go                     # (estendido) formato de peça configurável para PNG/pbf
└── samples/
    └── main.go                                # (estendido) mapas em imagem, vetorial, relevo sem dado, antimeridiano/polar em imagem, --raster-over <gpx>
```

**Documentação de apoio, alterada na implementação**: `CLAUDE.md` e
`README.md` (quinta etapa: comandos, serviço, portas e adapters),
`.gitignore` (`specs/005-frame-rendering/amostras/`),
`specs/004-geo-data-slice/contracts/slice-file.md` (a nota do `plan_id`,
conforme `contracts/slice-file-change.md`) e a nota de `spec.md` desta feature
(FR-022 e Suposições, ajustadas para a marca dentro de cada imagem e para os
limites de resolução; ver "Ajustes na spec" abaixo).

**Decisão de Estrutura**: mesmo projeto único em Go, seguindo as convenções já
consolidadas: um arquivo de domínio por responsabilidade (o desenho é dividido
em câmera, superfície, mapa, sobreposição e cena, para que cada parte seja
testável sozinha), porta no arquivo da entidade, adapters de tecnologia
(`zipfile`, `pngfile`) nomeados pela porta e adapter de estratégia
(`tiledecoder`) nomeado pelo papel, composition root em `cmd/sobrevoo/main.go`.

**Ordem de execução (base para o `/speckit-tasks`)**: (1) `atomicfile.Sync`
com `make test` verde, isolando qualquer regressão das etapas 3 e 4; (2)
`CameraPlan.ID`, `GeoSlice.PlanID` e o campo `plan_id` no exportador, com os
testes da etapa 4 atualizados e verdes; (3) fixtures em `test/helper`; (4)
domínio puro sem desenho: `Resolution`, `ParseFrameNumber`, `FrameSetID`,
`FrameFileName`, `FrameDirectory.Plan`, `RenderSummary`, `GeoSlice.Ensure*`,
`BoundingBox.ContainsBox`, erros; (5) adapters de leitura e gravação
(`zipfile.GeoSliceReader`, `tiledecoder`, `pngfile`); (6) o desenho, do menor
ao maior: câmera → superfície e DDA (com o teste de poda) → mapa e trilinear →
traçado e marcador → `Scene.Render`, com **medição de desempenho** ao fim do
DDA e de novo com `Scene.Render`; (7) `FrameService` e `GeoSliceService.Load`;
(8) CLI, `main.go` e configuração; (9) `test/samples`, `quickstart.md`
(valores marcados "(anotar)"), `CLAUDE.md` e `README.md`. As Histórias P1 a P7
da spec são entregáveis nesta ordem de prioridade (quadro isolado → todos →
falta de dado → resolução → retomada → proteção do destino → recusas). O
**quadro isolado** (P1) é o MVP: exige leitor de recorte, desenho, marcações
(a contagem de buracos vem junto do desenho) e o exportador de um quadro; o
conjunto de quadros (P2, P5, P6) acrescenta `FrameRepository`,
`FrameDirectory.Plan` e o progresso.

## Ajustes na spec (feitos junto com este plano)

Dois trechos de `spec.md` mudam para refletir decisões do planejamento, sem
mudar nenhum requisito verificável:

1. **FR-022 e a suposição "Formato de saída"**: a identificação do conjunto vai
   **dentro de cada imagem** (bloco `tEXt`), não num registro no diretório
   (`research.md` item 17). O FR-022 dizia "pode conter um registro" e a
   suposição falava em "registro do conjunto no destino".
2. **Suposição "Resolução"**: a referência de 320–7680 × 180–4320 passa a
   **180–3840 por lado, pares, até 8 294 400 pixels**, padrão 1080 × 1920
   (`research.md` item 11).

## Rastreamento de Complexidade

Nenhuma violação da constituição; nada a justificar. Duas observações fora
da constituição, mas que pesam no custo: o **renderizador** (câmera, DDA,
texturização trilinear, traçado) é o maior trecho novo de código do
projeto, todo no domínio e todo testável sem E/S; e a **mudança na etapa 4**
(`plan_id`) toca o exportador e o contrato de uma etapa já entregue — é
pequena e fica isolada na ordem de execução (fase 2).

## Riscos e Pontos de Atenção

- **R1 — Desempenho**: as metas assumem ~35 quadras de DDA por raio. Se a
  medição (feita ao fim do DDA e de novo com a cena completa) passar da meta, a
  primeira ação é a pirâmide de máximos, que não muda nenhum pixel (o teste de
  igualdade byte a byte a protege). Um recorte de 256 MiB com câmera baixa e
  células muito finas é o pior caso.
- **R2 — Memória**: recorte + cópias preenchidas das grades com buracos + cache
  de 256 MiB + buffers do quadro. Documentada; o preenchimento pode ser feito
  por trechos e o cache reduzido se for preciso.
- **R3 — Degrau entre grades de fontes diferentes** na emenda: sem fresta
  horizontal, mas sem parede vertical. Só ocorre com mais de uma fonte de relevo
  no mesmo voo.
- **R4 — Bordas do terreno sem suavização** e mapa "chapado" (sem iluminação):
  serrilhado no horizonte e nas cristas. O custo de supersampling 2×2 é
  conhecido (4×) e fica para depois de o usuário ver os quadros; a iluminação
  é outra etapa.
- **R5 — Sem curvatura da Terra**, e câmera e terreno no mesmo plano local:
  coerente com a etapa 3; o erro fica em poucos pontos percentuais dentro do
  alcance de um quadro (`research.md` item 2).
- **R6 — Determinismo entre arquiteturas**: os pixels devem coincidir (sem FMA e
  só funções de Go puro); os bytes do PNG só coincidem enquanto o
  `compress/flate` for o mesmo. Garantido para o mesmo binário e plataforma.
- **R7 — Mapa vetorial do usuário**: o BBBike em `resources/` é `pbf` e é
  recusado; a validação real usa um raster sintético sobre o relevo e o passeio
  reais. Desenhar vetores é uma etapa futura, e é a decisão que a sucede.
- **R8 — Quadro truncado por queda de energia**: coberto por `fsync` antes do
  `rename` e pela verificação de completude na retomada (`research.md` itens
  17 e 18).
- **R9 — Mudança de contrato da etapa 4**: recortes exportados antes de
  `plan_id` são recusados com orientação clara; não há migração (o recorte se
  regera com um comando).
