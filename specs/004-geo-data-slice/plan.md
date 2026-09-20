# Plano de Implementação: Recorte de Dados Geográficos para o Voo

**Branch**: `004-geo-data-slice` | **Data**: 2026-09-20 | **Especificação**: [spec.md](./spec.md)

**Entrada**: Especificação de funcionalidade de `/specs/004-geo-data-slice/spec.md`

## Resumo

Quarta etapa do Sobrevoo: dois comandos novos, filhos de `geodata`.
`sobrevoo geodata slice <plano.json>` lê o **conteúdo** dos dados que o
usuário registrou na etapa 2 e reúne o recorte de que um voo precisa — as
amostras de elevação do terreno e as peças do mapa base sob a área que a
câmera percorre, no nível de detalhe adequado à distância em que ela voa —,
imprime um resumo e, opcionalmente, o exporta num único arquivo ZIP
versionado. `sobrevoo geodata elevation --lat --lon` consulta a elevação de
uma coordenada para conferência contra outra fonte. A entrada do recorte é o
**arquivo de plano exportado pela etapa 3** (Clarificação de 2026-09-20): o
plano é o contrato entre as etapas. Nada é baixado, convertido, projetado,
desenhado ou renderizado.

A abordagem técnica (detalhada em `research.md`):

- A **área de interesse** é a caixa envolvente, por quadro, de um quadrado de
  meio-lado `margem × distância câmera–marcador` em torno do marcador
  (item 2), com desembrulho de longitude — mesmo tipo `BoundingBox` da etapa 2.
- A **cobertura** reaproveita `Route.Coverage` **sem alteração**: a área é
  decomposta em regiões retangulares em que o vencedor de cada tipo é
  uniforme, e o `Route` de seus centros passa pela verificação existente
  (item 3). Área não coberta → recusa com o mesmo tipo de relatório.
- O **nível de detalhe** é o menor zoom cuja resolução no terreno é ≤ ao
  tamanho de um pixel de tela na menor distância da câmera (com folga de
  2×), no menor `|latitude|` da área, limitado ao intervalo que o registro
  oferece (item 5); determinístico, monotônico na distância e explicado no
  resumo.
- A **elevação** é o valor da célula da grade que contém o ponto — sem
  interpolar, portanto sem inventar dado; "sem valor" é uma resposta própria,
  nunca zero nem erro (itens 9 e 10). Consulta e recorte usam a mesma regra.
- A **leitura de conteúdo** ganha duas portas novas — `BaseMapReader`
  (MBTiles, sobre o SQLite que o projeto já usa) e `ElevationReader` (GeoTIFF
  em Go puro: faixas ou peças; sem compressão, Deflate ou LZW; inteiros e
  ponto flutuante; unidade em metros ou pés) — separadas da porta de
  inspeção da etapa 2, que continua só lendo metadados.
- **Peça ausente** dentro de um registro é registrada e o recorte segue;
  **recorte grande demais** (limite inicial de 256 MiB) é recusado antes de
  ler conteúdo, com uma guarda também durante a leitura (item 11).
- A **exportação** é um único arquivo ZIP (método *store*, ordem e datas
  fixas) com `manifest.json`, uma grade `float32` por região e as peças
  originais; publicada atomicamente, sem sobrescrita por padrão, reusando o
  procedimento do exportador do plano, extraído para `atomicfile` (item 13).

A arquitetura hexagonal é preservada: o núcleo ganha entidades e funções
puras (`SliceTuning`, `SliceRegion`, `DetailLevel`, `Tile`, `TileSet`,
`ElevationGrid`, `ElevationReading`, `GeoSlice`, `SliceSummary`,
`AreaNotCoveredError`, mais métodos em `CameraPlan` e `BoundingBox`), quatro
portas (`CameraPlanReader`, `BaseMapReader`, `ElevationReader`,
`GeoSliceExporter`), dez erros e um serviço novo por recurso —
`GeoSliceService` (`Generate`, `Export`) — mais um método em cada serviço que
já existe (`CameraPlanService.Load`, `GeoDataService.ElevationAt`). Toda E/S
fica nos adapters; a CLI só traduz flags, texto e códigos de saída.

## Contexto Técnico

**Linguagem/Versão**: Go 1.26 (sem mudança)

**Dependências Principais**: uma nova, `golang.org/x/image/tiff/lzw`
(decodificador LZW de TIFF, item 8 da pesquisa). O resto é biblioteca padrão
(`math`, `compress/zlib`, `archive/zip`, `encoding/binary`, `encoding/json`,
`database/sql`), `modernc.org/sqlite` (já usado pela etapa 2), Cobra, testify
e `go.uber.org/mock`.

**Armazenamento**: nenhum estado persistente novo. Lê os arquivos
registrados (somente leitura) e o arquivo de plano; único artefato gravado: o
ZIP exportado sob demanda (`contracts/slice-file.md`), de forma atômica e sem
sobrescrita por padrão. O registro da etapa 2 não muda.

**Testes**: mesmo padrão — `go test` com testify, given/when/then, um
`t.Run` por cenário, sem testes tabulares. Builders em `builddomain`
(`ElevationGridBuilder`, `TileSetBuilder`, `GeoSliceBuilder`,
`SliceTuningBuilder`, `ElevationReadingBuilder`); mocks gerados das quatro
portas novas (`mockdomain`) e de `GeoSliceService`
(`mockapplication`). O domínio é testado como funções puras, com **testes de
propriedade** (nível monotônico na distância; a área contém todas as câmeras
e marcadores; cada amostra e cada peça pertence a exatamente uma região;
deslocar o plano para outras regiões do planeta preserva contagens e
propriedades, SC-009). Serviços testados com portas mockadas; a CLI, com os
serviços mockados. Os adapters (`basemapreader`, `elevationreader`,
`jsonfile.CameraPlanReader`, `zipfile`, `atomicfile`) são testados contra
**fixtures com conteúdo** geradas em código em `test/helper` (MBTiles com
peças e buracos; GeoTIFF sem compressão/Deflate/LZW, faixas e peças, `int16` e
`float32`, com e sem "sem dado", em metros e pés), em diretório temporário.
Não há teste automatizado de ponta a ponta; `quickstart.md` é o checklist
manual.

**Plataforma-Alvo**: mesmo binário multiplataforma (Linux, macOS, Windows).

**Tipo de Projeto**: CLI (projeto único em Go, sem frontend/backend).

**Metas de Desempenho**: SC-001 — recorte de um voo de até 50 km com os
parâmetros padrão e resumo em menos de 30 s; SC-010 — recusa por tamanho em
menos de 5 s, sem ler conteúdo (só contas e metadados). O custo é dominado
por E/S: uma consulta SQLite por peça e a decodificação apenas das
faixas/peças TIFF que intersectam a janela. Sem concorrência (determinismo).

**Restrições**: 100% offline; núcleo sem I/O; determinismo bit a bit no mesmo
binário e plataforma (área quantizada a 1e-7°, resto em inteiros ou cópia de
valores — `research.md` item 12); nenhum dado de região embutido; recorte de
no máximo 256 MiB (estimado e real); elevação sempre em metros; nenhuma
interpolação, reamostragem ou reprocessamento de imagem; somente os formatos
já reconhecidos pela etapa 2 (MBTiles; GeoTIFF em coordenadas geográficas).

**Escala/Escopo**: uso pessoal — registros de dezenas de MB a alguns GB;
recortes de até 256 MiB (milhares de peças e milhões de amostras);
tipicamente um registro de cada tipo, com suporte a vários.

## Verificação da Constituição

*PORTÃO: Deve passar antes da Fase 0 de pesquisa. Reverificar após o design da Fase 1.*

Avaliada antes da pesquisa e **reavaliada após o design** (data-model,
contratos e quickstart); o resultado não mudou.

| Princípio | Avaliação | Como o design atende |
|---|---|---|
| I. Arquitetura Hexagonal | PASSA | Toda a regra (área de interesse, regiões, nível de detalhe, peças, janela de amostras, resumo, validação do plano) é função pura ou método de entidade em `internal/domain`, sem importar `os`, SQLite, `archive/zip`, Cobra nem `internal/infra`. `GeoSliceService` em `internal/application` só chama portas e métodos de domínio, na ordem certa. SQLite, TIFF, ZIP e JSON ficam nos adapters. |
| II. Portas para Toda Dependência Externa | PASSA | Toda E/S nova passa por porta declarada no domínio: ler o plano (`CameraPlanReader`), ler peças (`BaseMapReader`), ler amostras (`ElevationReader`), gravar o recorte (`GeoSliceExporter`). A checagem de "arquivo presente" reusa `FileChecker`. Nenhuma porta para `math`, `time`, `sort` ou outra função pura da biblioteca padrão. |
| III. Entrypoints Descartáveis | PASSA | Os serviços recebem e devolvem tipos de domínio; nada de flags, texto ou códigos de saída. `geodata slice` e `geodata elevation` só traduzem flags → chamadas de serviço e resultados/erros → texto/código. `GeoSliceService.Generate(plan)` não sabe de onde o plano veio — um comando futuro que encadeie tudo em memória o reusa sem tocar em arquivo. |
| IV. Neutralidade Geográfica | PASSA | Nenhum dado, constante ou caso especial de região. A área usa o desembrulho de longitude da etapa 1/2 e a fórmula de Web Mercator igual em qualquer latitude (limite de 85,05° é propriedade do esquema de peças, não de região); o nível usa o menor `|φ|` da área com a mesma fórmula em qualquer hemisfério; SC-009 verifica planos equivalentes deslocados pelo planeta, inclusive no antimeridiano e acima de 80°. |
| V. Funcionamento Offline | PASSA | Somente arquivos locais registrados; nenhuma rede, chave ou serviço; nada é baixado nem convertido (FR-003). Sem relógio. |
| VI. Testes Automatizados no Núcleo | PASSA | Domínio testado sem I/O; `GeoSliceService`, `GeoDataService` e `CameraPlanService` testados com as portas mockadas (`mockdomain`); nenhum teste do núcleo toca disco, rede ou processo externo. |
| VII. Erros Sentinela no Domínio | PASSA | `ErrPlanFileInvalid`, `ErrPlanFormatVersionUnsupported`, `ErrAreaNotCovered`, `ErrSliceTooLarge`, `ErrGeoDataContentUnreadable`, `ErrElevationUnitUnsupported`, `ErrSliceDestinationExists`, `ErrSliceDestinationInvalid`, `ErrElevationNotCovered`, `ErrInvalidCoordinate` em `errors.go`; a CLI os traduz em códigos 17–26 (`contracts/cli.md`). `AreaNotCoveredError` é um tipo de domínio com `Is(ErrAreaNotCovered)` para carregar o relatório de cobertura sem perder `errors.Is`. Erros de `os`, SQLite, TIFF, ZIP e JSON são traduzidos nos adapters; o núcleo nunca vê um deles. |
| VIII. Configuração Injetada | PASSA | Constantes heurísticas em `domain.SliceTuning` (formato definido pelo núcleo). `config.SliceTuning` tem **tipos próprios** e não importa o domínio; o composition root (`cmd/sobrevoo/config_mapping.go`) os mapeia e injeta em `NewGeoSliceService`. O caminho do plano, do destino, as coordenadas e `--overwrite` chegam ao núcleo por parâmetro. O núcleo não lê flag, ambiente nem arquivo. |
| IX. Organização de Portas, Service Layer e Mocks | PASSA | Sem `ports.go`: cada porta no arquivo da entidade que produz, no topo, com `//go:generate` logo após `package` (`CameraPlanReader` em `camera_plan.go`, ao lado de `CameraPlanExporter`; `BaseMapReader` em `tile.go`; `ElevationReader` em `elevation_grid.go`; `GeoSliceExporter` em `geo_slice.go`). Nomeadas pelo papel (`Reader`, `Exporter`; nenhuma é repositório). Um serviço por recurso, sem `Execute`: `GeoSliceService`/`geoSliceService`/`NewGeoSliceService` (`Generate`, `Export`) e um método novo em cada serviço existente (`CameraPlanService.Load`, `GeoDataService.ElevationAt`) — nenhum serviço por caso de uso. Toda regra em `internal/domain` (métodos de `CameraPlan`, `BoundingBox`, `SliceTuning`, `ElevationGridInfo`, `ElevationGrid`, construtor `NewGeoSlice` que calcula o resumo); `application` só chama portas. Adapters: `jsonfile.NewCameraPlanReader()` (pacote de tecnologia que já serve outras portas: construtor pela porta), `zipfile.NewGeoSliceExporter()` (idem), `basemapreader.NewMBTiles()` e `elevationreader.NewGeoTIFF()` (pacotes de estratégias de uma única porta: pacote leva o nome da porta, tipo/construtor só a estratégia, arquivos `mbtiles_base_map_reader.go` e `geotiff_elevation_reader.go`); `atomicfile` é utilitário de publicação de arquivo, não adapter de porta. Nenhum nome exportado repete o pacote. Mocks por `//go:generate` acima de cada interface, em `mockdomain`/`mockapplication`. Receivers: `c` (`CameraPlan`), `b` (`BoundingBox`), `t` (`SliceTuning`), `g` (`ElevationGrid`), `s` (`geoSliceService`, `geoDataService`, `cameraPlanService`), `r` (readers), `e` (exporters). |
| X. Testes: Given/When/Then, Builders e Isolamento por Camada | PASSA | Todo teste em `t.Run("should ...")` com `// given`/`// when`/`// then`, sem tabelas; builders em `builddomain`; domínio/serviços com portas mockadas; CLI com serviços mockados (`mockapplication`); adapters com diretório temporário (é o adapter que toca disco, então é o objeto do teste). Sem teste de ponta a ponta automatizado; `quickstart.md` cobre o manual. |
| Idioma dos Artefatos | PASSA | Artefatos do Spec Kit em português; identificadores, pacotes, arquivos, comentários e mensagens de commit em inglês; I/O em tempo de execução (flags, saída, erros) em inglês, como nas etapas anteriores. |

Nenhuma violação identificada. Rastreamento de Complexidade não se aplica.

## Estrutura do Projeto

### Documentação (desta funcionalidade)

```text
specs/004-geo-data-slice/
├── plan.md              # Este arquivo (saída do comando /speckit-plan)
├── research.md          # Saída da Fase 0 (comando /speckit-plan)
├── data-model.md        # Saída da Fase 1 (comando /speckit-plan)
├── quickstart.md        # Saída da Fase 1 (comando /speckit-plan)
├── contracts/
│   ├── cli.md           # Saída da Fase 1: comandos `geodata slice` e `geodata elevation`
│   └── slice-file.md    # Saída da Fase 1: formato do arquivo exportado
├── checklists/
│   └── requirements.md  # Gerado por /speckit-specify
├── amostras/            # Gerado pela ferramenta `test/samples` (ver quickstart); não versionar binários
└── tasks.md             # Saída da Fase 2 (comando /speckit-tasks - NÃO criado pelo /speckit-plan)
```

### Código-Fonte (raiz do repositório)

```text
cmd/sobrevoo/
├── main.go                                    # (alterado) monta os readers, o exporter, GeoSliceService, os métodos novos e os comandos "geodata slice" e "geodata elevation"
└── config_mapping.go                          # (alterado) mapeia config.SliceTuning → domain.SliceTuning

internal/
├── domain/
│   ├── camera_plan.go                         # (estendido) porta CameraPlanReader no topo; CameraPlan.Validate, CameraPlan.AreaOfInterest
│   ├── bounding_box.go                        # (estendido) Intersects, Regions, TileRange
│   ├── geo_data_coverage.go                   # (estendido) SelectSource exporta a regra de pickCoverageWinner; sem mudança de lógica
│   ├── geo_slice.go                           # NOVO: porta GeoSliceExporter (no topo), SliceTuning, SliceRegion, GeoSlice, SliceSummary, SliceSourceUse, NewGeoSlice
│   ├── tile.go                                # NOVO: porta BaseMapReader (no topo), LevelRange, TileID, TileRange, Tile, TileRead, TileSet, DetailLevel, SliceTuning.DetailLevel
│   ├── elevation_grid.go                      # NOVO: porta ElevationReader (no topo), ElevationGridInfo (+Window, CellAt), GridWindow, ElevationGrid, ElevationWindow, ElevationReading
│   ├── errors.go                              # (estendido) dez sentinelas novos + AreaNotCoveredError
│   ├── *_test.go                              # NOVOS/estendidos: um por arquivo acima (funções puras + propriedades)
│   ├── builddomain/
│   │   ├── elevation_grid_builder.go          # NOVO
│   │   ├── tile_set_builder.go                # NOVO
│   │   ├── geo_slice_builder.go               # NOVO
│   │   ├── slice_tuning_builder.go            # NOVO
│   │   └── elevation_reading_builder.go       # NOVO
│   └── mockdomain/
│       ├── camera_plan_reader.go              # GERADO (make generate)
│       ├── base_map_reader.go                 # GERADO
│       ├── elevation_reader.go                # GERADO
│       └── geo_slice_exporter.go              # GERADO
├── application/
│   ├── geo_slice_service.go                   # NOVO: GeoSliceService / geoSliceService / NewGeoSliceService (Generate, Export)
│   ├── geo_slice_service_test.go              # NOVO
│   ├── camera_plan_service.go                 # (alterado) Load(path); recebe CameraPlanReader
│   ├── camera_plan_service_test.go            # (alterado)
│   ├── geo_data_service.go                    # (alterado) ElevationAt(lat, lon); recebe ElevationReader
│   ├── geo_data_service_test.go               # (alterado)
│   └── mockapplication/
│       ├── geo_slice_service.go               # GERADO
│       ├── camera_plan_service.go             # REGERADO (novo método)
│       └── geo_data_service.go                # REGERADO (novo método)
└── infra/
    ├── inbound/cli/
    │   ├── geodata_slice.go                   # NOVO: comando "geodata slice", flags, formatação do resumo
    │   ├── geodata_slice_test.go              # NOVO
    │   ├── geodata_elevation.go               # NOVO: comando "geodata elevation"
    │   ├── geodata_elevation_test.go          # NOVO
    │   └── exit_code.go                       # (estendido) códigos 17–26
    └── outbound/
        ├── config/config.go                   # (estendido) Config.SliceTuning (tipos do próprio pacote)
        ├── atomicfile/
        │   ├── atomic_file.go                 # NOVO: Publish(path, overwrite, write) — extraído de jsonfile
        │   └── atomic_file_test.go            # NOVO (migra os cenários de atomicidade de camera_plan_exporter_test.go)
        ├── jsonfile/
        │   ├── camera_plan_exporter.go        # (alterado) usa atomicfile.Publish; comportamento idêntico
        │   ├── camera_plan_reader.go          # NOVO: decodifica plan-file.md (format_version 1)
        │   └── camera_plan_reader_test.go     # NOVO
        ├── zipfile/
        │   ├── geo_slice_exporter.go          # NOVO: ZIP determinístico + manifest.json (usa atomicfile)
        │   └── geo_slice_exporter_test.go     # NOVO
        ├── basemapreader/
        │   ├── mbtiles_base_map_reader.go     # NOVO: Levels, ReadTiles (SQLite somente leitura; TMS→XYZ)
        │   └── mbtiles_base_map_reader_test.go
        └── elevationreader/
            ├── geotiff_elevation_reader.go    # NOVO: Describe (tags), ReadWindow (faixas/peças; none/Deflate/LZW; predictor 1/2/3)
            └── geotiff_elevation_reader_test.go

test/
├── helper/
│   ├── mbtiles_fixture.go                     # (estendido) MBTiles com peças e buracos, níveis e formato configuráveis
│   ├── geotiff_fixture.go                     # (estendido) GeoTIFF com amostras, compressão, layout, "sem dado", unidade, PixelIsPoint
│   └── camera_plan_fixture.go                 # NOVO: bytes de arquivo de plano (válido, versão 2, incoerente, truncado)
└── samples/
    └── main.go                                # NOVO: ferramenta de desenvolvimento que grava as amostras do quickstart
```

**Decisão de Estrutura**: mesmo projeto único em Go, seguindo as convenções
já consolidadas: um arquivo de domínio por responsabilidade, porta no
arquivo da entidade, adapters de tecnologia (`jsonfile`, `zipfile`) nomeados
pela porta e adapters de estratégia (`basemapreader`, `elevationreader`)
nomeados pelo papel, composition root em `cmd/sobrevoo/main.go`. `CLAUDE.md`
e `README.md` serão atualizados na implementação para refletir a quarta
etapa (novos comandos, serviço, portas e adapters).

**Ordem de execução (base para o `/speckit-tasks`)**: (1) extração
`atomicfile` com `make test` verde e comportamento do `plan --export`
inalterado, isolando qualquer regressão da etapa 3; (2) fixtures com
conteúdo em `test/helper`; (3) domínio puro (área, regiões, nível, janela,
resumo, validação do plano); (4) adapters de leitura (`basemapreader`,
`elevationreader`, `jsonfile.CameraPlanReader`); (5) serviços; (6)
exportador ZIP; (7) CLI, `main.go` e configuração; (8)
`test/samples`, `quickstart.md`, `CLAUDE.md` e `README.md`. As Histórias P1
a P6 da spec são entregáveis nesta ordem de prioridade (recorte → nível →
resumo → exportação → robustez → consulta), com `elevation` (P6)
independente e podendo ser adiantado, pois só depende de `ElevationReader` e
de `SelectSource`.

## Rastreamento de Complexidade

Nenhuma violação da constituição; nada a justificar. Duas observações
fora da constituição, mas que pesam no custo: o **leitor de GeoTIFF em Go
puro** (item 8 da pesquisa) é o maior trecho novo de código de adapter, e a
**dependência nova** `golang.org/x/image/tiff/lzw` é a única do módulo — as
alternativas (GDAL via CGO, decodificador LZW próprio) foram rejeitadas na
pesquisa.

## Riscos e Pontos de Atenção

- **Subconjunto de GeoTIFF**: DEMs reais variam (compressões, predictors,
  layouts). Sem dado real de um usuário, o leitor só foi desenhado contra o
  que a especificação TIFF/GeoTIFF define. Mitigação: o subconjunto é
  documentado; qualquer coisa fora dele falha com `ErrGeoDataContentUnreadable`
  e a causa ("unsupported encoding: <o quê>"), nunca com dado errado;
  `quickstart.md` item 10 repete a verificação com um DEM real. Se o
  subconjunto se mostrar curto, ampliar é aditivo.
- **Nível de detalhe "ideal" pesado**: com `d_min` de 300–600 m e a fonte
  oferecendo zoom alto, o recorte tende a passar do limite em voos longos.
  Decidido (pesquisa, item 5): recusar e explicar, não degradar em silêncio;
  as constantes (`TexelScreenRatio`, `MarginFactor`) ficam em `SliceTuning`
  justamente para calibrar quando existir renderização.
- **Coordenadas do relatório de cobertura**: a área é retangular e a
  verificação é por região; o relatório traz os **centros** das regiões, não
  as bordas exatas do buraco. Documentado em `contracts/cli.md`; mais preciso
  que "não coberto" e suficiente para o usuário saber o que registrar.
- **`PixelIsPoint`**: o inspetor da etapa 2 trata o ponto de amarração
  sempre como canto de célula (erro de meia célula na caixa, sem efeito na
  cobertura). O leitor de conteúdo trata corretamente; se isso mudar a
  cobertura de algum registro de borda, será visto como falta de dado
  mínima, não como erro silencioso. Não se altera o inspetor nesta etapa.
- **Tamanho real × estimativa**: a estimativa usa 64 KiB por peça; um mapa
  de peças muito maiores passa da estimativa e é pego pela guarda durante a
  leitura, depois de gastar E/S. Aceitável; a constante é ajustável.
- **Extração `atomicfile`** toca o exportador da etapa 3. Mitigação:
  refatoração de comportamento idêntico, fase própria com testes existentes
  verdes antes de qualquer código novo.
- **Determinismo do ZIP**: garantido pelo método *store*, datas e ordem
  fixas; um teste compara duas exportações byte a byte. Entre versões do
  Go, `archive/zip` com *store* não muda o formato dos bytes.
