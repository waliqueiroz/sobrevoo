# Plano de Implementação: Iluminação Direcional do Terreno

**Branch**: `016-terrain-lighting` | **Data**: 2026-10-05 | **Especificação**: [spec.md](./spec.md)

**Entrada**: Especificação de funcionalidade de `/specs/016-terrain-lighting/spec.md`

**Nota**: Este template é preenchido pelo comando `/speckit-plan`; sua definição descreve o fluxo de execução.

## Resumo

Acrescenta sombreamento direcional ao terreno já desenhado por lançamento de
raios (etapa 5): em cada pixel cujo raio atinge a malha, a cor que vem do
mapa base é multiplicada por um fator de brilho calculado a partir da
inclinação real da superfície naquele ponto em relação a uma luz direcional
fixa (azimute e altura documentados, convenção de bússola igual à do
`heading` da câmera) — nunca um sombreado assado no mapa. A inclinação é
lida de uma pirâmide de gradiente pré-computada por grade (o mesmo padrão de
mipmap que `imagery.go` já usa para a textura do mapa, só que para a
derivada da altura em vez da cor), escolhida por nível conforme o quanto um
pixel da tela cobre de terreno naquela distância — grande ao longe, pequena
de perto —, o que resolve a estabilidade/suavidade exigida (FR-007) sem
introduzir nenhuma operação não determinística. O fator de brilho é a
diferença entre o produto escalar da normal real e de uma superfície plana
com a luz, escalada e limitada a uma faixa fixa — o que garante, de forma
puramente aritmética e sem nenhum caso especial, que uma superfície plana
sempre sai neutra (FR-008/caso de borda), qualquer que seja a altura da luz
escolhida. A pirâmide é construída só a partir das amostras reais da grade
(nunca das alturas com buracos preenchidos que a geometria do raio usa), com
uma contagem de cobertura por nível — a mesma técnica de agregação do
mipmap de textura —, de forma que um ponto com elevação cuja vizinhança
tem buracos usa só as amostras reais disponíveis, subindo um nível da
pirâmide quando a vizinhança mais fina não tem nenhuma, e cai no tom neutro
só quando a pirâmide inteira não tem nenhuma amostra real por ali — nunca
tratando esse ponto como se fosse, ele próprio, um buraco de elevação
(FR-008, regra dos três casos da sessão de `/speckit-clarify`). O traçado,
o marcador, as sobreposições de tela e os dois padrões de "sem dado"
continuam fora do alcance da luz, por não passarem pelo ramo de código que
a aplica. Nenhuma porta, nenhum serviço, nenhuma flag e nenhum formato de
arquivo mudam — só `internal/domain`, e `RenderVersion` sobe de 5 para 6
porque os pixels do terreno mudam de verdade.

## Contexto Técnico

**Linguagem/Versão**: Go 1.26.4 (`go.mod`), sem mudança.

**Dependências Principais**: nenhuma nova. Toda a funcionalidade usa só
`math` da biblioteca padrão (`Sin`, `Cos`, `Sqrt`, `Log2`, `Floor`, `Min`,
`Max`, `Abs` — todos já na lista de operadores permitidos pelo desenho de
quadros) sobre dados que o pipeline já carrega (a grade de elevação do
recorte).

**Armazenamento**: arquivos locais (plano `.json`, recorte `.zip`, quadros
`.png`), sem mudança de formato nem de tecnologia — nenhum campo novo em
nenhum arquivo; só os pixels do terreno nos quadros mudam.

**Testes**: `go test ./... -cover`, `testify` + `uber-go/mock`, sem mudança
de ferramenta. O hash de referência de `frame_scene_test.go` precisa de uma
nova constante — os pixels do terreno mudam de propósito (CLAUDE.md: "a
constante só muda junto com `domain.RenderVersion`", que sobe nesta etapa).
Novos testes de domínio cobrem a pirâmide de gradiente, o cálculo do fator
de brilho a partir de uma normal conhecida, e a regra dos três casos de
cobertura, todos sem tocar disco (dados de grade sintéticos, com os
builders de `builddomain` que `ElevationGrid`/`GeoSlice` já têm).

**Plataforma-Alvo**: CLI de linha de comando, macOS/Linux, amd64/arm64 — o
determinismo byte a byte entre arquiteturas (já exigido desde a etapa 5) é
a restrição central desta etapa, igual à da etapa 11: toda a aritmética da
pirâmide e do fator de brilho usa só os operadores já permitidos, nunca
`Pow`/`Exp`/`Sinh` nem iteração de `map` para produzir valor.

**Tipo de Projeto**: CLI de projeto único (`cmd/sobrevoo`), sem mudança de
estrutura.

**Metas de Desempenho**: a pirâmide de gradiente é construída uma vez por
`Scene` (como `imagery`/`surface.heights`/`surface.zmin/zmax` já são),
custando o mesmo tipo de varredura O(linhas×colunas) que `fillHoles` e o
cálculo de `zmin`/`zmax` já fazem em `newSurface` — desprezível ao lado do
custo de decodificar as peças do mapa base. Por pixel, o custo adicional é
uma busca em dois níveis da pirâmide (interpolação bilinear em cada,
mistura entre os dois) mais um produto escalar — mesma ordem de grandeza do
que `sampler.color` já faz para escolher o nível de detalhe da textura, e
ordens de grandeza mais barato que o próprio lançamento do raio.

**Restrições**: byte a byte determinístico entre goroutines, processos e
arquiteturas (Constitution; CLAUDE.md "O desenho dos quadros"); nenhuma
leitura nova do trajeto GPS, dos dados geográficos registrados ou do
recorte exportado (FR-013 — a iluminação é só uma função da mesma grade de
elevação já carregada); offline (Princípio V); nenhuma opção nova de CLI
(FR-011).

**Escala/Escopo**: mesma escala das etapas 5–15 (até ~3840×2160, até
432 000 quadros por plano, grades de elevação de qualquer tamanho que o
recorte já suporta); a pirâmide tem no máximo
⌈log2(max(linhas, colunas))⌉+1 níveis por grade, a mesma profundidade da
pirâmide de mipmaps que `newTileTexture` já constrói para cada peça do mapa
base.

## Verificação da Constituição

*PORTÃO: Deve passar antes da Fase 0 de pesquisa. Reverificar após o design da Fase 1.*

| Princípio | Verificação |
|---|---|
| I. Arquitetura Hexagonal | A pirâmide de gradiente, o cálculo da normal e o fator de brilho são código de domínio puro (`internal/domain`), estendendo o mesmo renderizador por lançamento de raios da etapa 5 (`frame_surface.go`, `frame_scene.go`) e um novo arquivo de domínio (`frame_terrain_light.go`) — nenhuma dependência de infraestrutura cruza a fronteira. |
| II. Portas para Toda Dependência Externa | Nenhuma porta nova: nada aqui faz I/O — a pirâmide é calculada a partir da `ElevationGrid` que `ElevationReader` já entrega hoje. As únicas funções de `math` usadas (`Sin`/`Cos` da direção fixa, `Sqrt`/`Log2` do nível da pirâmide) são chamadas puras da biblioteca padrão, o mesmo carve-out que já cobre `newCamera`/`newFramePlane`. |
| III. Entrypoints Descartáveis | Nenhuma mudança na CLI: nenhuma flag nova, nenhum comando novo (FR-011) — o efeito aparece automaticamente em `render frame`, `render all` e `fly`. |
| IV. Neutralidade Geográfica | O azimute fixo da luz é relativo ao norte verdadeiro (a mesma convenção de bússola que `CameraFrame.Heading` já usa) — funciona identicamente em qualquer latitude/longitude, sem nenhum dado geográfico embutido nem privilégio de região (FR-004). |
| V. Funcionamento Offline | Nenhuma dependência nova, nenhum arquivo adicional, nenhuma rede. |
| VI. Testes Automatizados no Núcleo | A pirâmide, a normal e o fator de brilho são testados no domínio com grades sintéticas (builders de `builddomain`), sem tocar disco — nenhum mock novo é necessário (nenhuma porta nova). |
| VII. Erros Sentinela no Domínio | Nenhum erro sentinela novo: nada aqui é uma escolha do usuário que possa ficar malformada (FR-011). |
| VIII. Configuração Injetada | A direção, a altura e a faixa fixa de brilho são constantes de domínio em `render_tuning.go`, no mesmo padrão de `NoMapColors`/`OverlayTextColor` — não há escolha de usuário para injetar, então não há configuração nova em `config.go`. |
| IX. Portas/Service Layer/Regra de Negócio | Nenhuma porta nova, nenhum serviço novo, nenhum DTO novo. A pirâmide e a normal são comportamento do tipo que já possui a geometria (`surface`/`placedSurface`); o fator de brilho a partir de uma normal é a única função livre nova — matemática sem dono sobre dois vetores, a mesma categoria de `clamp`/`quantize` já aceita pela convenção do projeto. |
| X. Testes: Given/When/Then, Builders, Isolamento | Sem mudança de convenção: novos `t.Run("should ...")` em `frame_surface_test.go`/`frame_terrain_light_test.go`, isolados do resto do pipeline; o teste de hash de referência de `frame_scene_test.go` ganha uma nova constante, calculada depois da implementação. |

Nenhuma violação; nada a registrar em Rastreamento de Complexidade.

## Estrutura do Projeto

### Documentação (desta funcionalidade)

```text
specs/016-terrain-lighting/
├── plan.md              # Este arquivo (saída do comando /speckit-plan)
├── research.md          # Saída da Fase 0 (comando /speckit-plan)
├── data-model.md        # Saída da Fase 1 (comando /speckit-plan)
├── quickstart.md         # Saída da Fase 1 (comando /speckit-plan)
├── contracts/            # Saída da Fase 1 (comando /speckit-plan)
│   └── frame-files-change.md
└── tasks.md              # Saída da Fase 2 (comando /speckit-tasks — NÃO criado pelo /speckit-plan)
```

### Código-Fonte (raiz do repositório)

Projeto único já existente (hexagonal), sem mudança de layout — só arquivos
novos e extensões pontuais dentro de `internal/domain`, a única camada que
esta etapa toca:

```text
internal/domain/
├── frame_terrain_light.go      # NOVO: direção fixa da luz (azimute/altura), a faixa fixa de
│                                 #   brilho, e terrainLightFactor(normal) — a função pura que
│                                 #   mapeia uma normal no fator de brilho
├── frame_surface.go            # surface ganha a pirâmide de gradiente (construída em
│                                 #   newSurface, a partir das amostras cruas, nunca das
│                                 #   alturas com buracos preenchidos); placedSurface ganha
│                                 #   normalAt(x, y, pegada) — a normal adaptativa pela distância
├── frame_scene.go               # drawPixel: depois de obter a cor do mapa (stateImage), multiplica
│                                 #   pelo fator de brilho da normal no ponto do raio; stateNoMap e
│                                 #   stateNoElevation continuam sem nenhuma mudança
├── render_tuning.go             # RenderVersion 5→6; +TerrainLightAzimuthDegrees,
│                                 #   +TerrainLightAltitudeDegrees, +TerrainLightMinFactor,
│                                 #   +TerrainLightMaxFactor
└── (testes correspondentes a cada arquivo acima, incluindo o novo hash de
    frame_scene_test.go)

internal/domain/builddomain/
└── (builders existentes de ElevationGrid/GeoSlice reaproveitados; nenhum
    builder novo é estritamente necessário, mas um WithSlope(...) ou
    equivalente pode ser acrescentado a ElevationGridBuilder se os testes
    da pirâmide pedirem uma grade inclinada conhecida com conveniência)
```

Nenhum arquivo fora de `internal/domain` muda: sem porta nova, sem serviço
novo, sem flag de CLI nova, sem campo novo em `config`, sem mudança em
`cmd/sobrevoo`.

**Decisão de Estrutura**: toda a mudança fica dentro de `internal/domain`,
seguindo o mesmo ponto de extensão que a etapa 5 já estabeleceu para o
desenho do terreno e a etapa 11 confirmou para uma mudança puramente visual
sem nenhuma superfície nova de configuração — só que, desta vez, a regra de
negócio nova (a pirâmide de gradiente e o fator de brilho) estende a
própria entidade `surface`/`placedSurface`, em vez de criar uma entidade
paralela, porque a inclinação é, literalmente, uma propriedade da geometria
que `surface` já possui.

## Rastreamento de Complexidade

> **Preencher SOMENTE se a Verificação da Constituição tiver violações que precisam ser justificadas**

Nenhuma violação a registrar.
