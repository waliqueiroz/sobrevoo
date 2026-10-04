# Plano de Implementação: Rótulos em Português e Legibilidade do Texto das Sobreposições

**Branch**: `012-overlay-ptbr-readability` | **Data**: 2026-10-03 | **Especificação**: [spec.md](./spec.md)

**Entrada**: Especificação de funcionalidade de `/specs/012-overlay-ptbr-readability/spec.md`

**Nota**: Este template é preenchido pelo comando `/speckit-plan`; sua definição descreve o fluxo de execução.

## Resumo

Traduz para português do Brasil os rótulos que as sobreposições de tela
desenham (`DIST`/`ELEV`/`GAIN`/`TIME` → `DIST`/`ELEV`/`GANHO`/`TEMPO`) e
corrige três defeitos de legibilidade introduzidos na etapa anterior,
visíveis num quadro real: o contorno escuro do texto, cuja espessura era
uma fração da **altura do quadro** (5 px a 1080×1920, do tamanho do
próprio traço da letra, fechando os vãos de "6"/"8"/"0"), passa a ser uma
fração do **tamanho da letra** (`ppem`), ficando proporcionalmente fina em
qualquer resolução; a fonte embutida troca do peso regular (`Go Regular`)
para o peso forte da mesma família (`Go Bold`, mesma licença, nenhuma
dependência nova); e a largura compartilhada dos três painéis numéricos,
hoje recalculada a cada quadro a partir do texto daquele quadro (o que a
fazia "pulsar" ao longo do vídeo), passa a ser calculada uma única vez por
`Scene` — a partir do texto mais largo que cada bloco tem em **qualquer**
quadro do plano —, com o mesmo resultado para um quadro isolado
(`render frame`) e para o mesmo quadro dentro do voo inteiro
(`render all`). O rasterizador e a extração de contornos da etapa anterior
não mudam. Como os pixels de um quadro com sobreposição ligada mudam de
verdade, `RenderVersion` sobe de 3 para 4.

## Contexto Técnico

**Linguagem/Versão**: Go 1.26.4 (`go.mod`), sem mudança.

**Dependências Principais**: `golang.org/x/image` (já em `go.mod`, direta)
— passa a usar `font/gofont/gobold` no lugar de `font/gofont/goregular`;
mesmo módulo, mesma licença (BSD-3-Clause, Bigelow & Holmes), nenhuma
dependência nova.

**Armazenamento**: arquivos locais (plano `.json`, recorte `.zip`, quadros
`.png`), sem mudança de formato — nenhum campo novo em nenhum arquivo; só
os pixels dos quadros e a versão do desenho mudam.

**Testes**: `go test ./... -cover`, `testify` + `uber-go/mock`, sem mudança
de ferramenta.

**Plataforma-Alvo**: CLI de linha de comando, macOS/Linux, amd64/arm64 — o
determinismo byte a byte entre arquiteturas (etapas 5 e 11) continua
valendo; nenhuma das mudanças desta etapa toca o rasterizador em si.

**Tipo de Projeto**: CLI de projeto único (`cmd/sobrevoo`), sem mudança de
estrutura.

**Metas de Desempenho**: calcular a largura compartilhada dos painéis
numéricos uma vez por `Scene` passa a exigir formatar e medir até três
textos por quadro do plano (até 432 000) — um único laço sobre quadros já
em memória, da ordem de milissegundos, pago uma vez por execução de
`render frame`/`render all`/`fly` (não por quadro desenhado); sem isso, o
cálculo atual (por quadro) já custa O(quadros) por quadro — a mudança troca
um custo que se pagaria de novo a cada quadro por um que se paga uma vez.

**Restrições**: byte a byte determinístico entre goroutines, processos e
arquiteturas (Constitution; CLAUDE.md "O desenho dos quadros") — mantido,
porque nenhuma aritmética do rasterizador muda, só constantes e dado
textual; nenhuma leitura nova do trajeto GPS nem dos dados geográficos
registrados; nenhuma dependência de fonte instalada no sistema; offline
(Princípio V).

**Escala/Escopo**: mesma escala das etapas 5–11 (até ~3840×2160, até 432
000 quadros por plano); os rótulos novos ("GANHO", "TEMPO") continuam só
letras maiúsculas ASCII, sem acento, então o alfabeto de glifos usados — e
o cache de glifos da etapa anterior — não cresce.

## Verificação da Constituição

*PORTÃO: Deve passar antes da Fase 0 de pesquisa. Reverificar após o design da Fase 1.*

| Princípio | Verificação |
|---|---|
| I. Arquitetura Hexagonal | Toda a mudança continua em código de domínio puro (`internal/domain`), como a etapa anterior. A troca de `goregular` por `gobold` é a mesma categoria de dado estático embutido, não I/O. |
| II. Portas para Toda Dependência Externa | Nenhuma porta nova: nenhum I/O novo. `gofont/gobold` é a mesma biblioteca pura sobre dado embutido que `gofont/goregular` já era. |
| III. Entrypoints Descartáveis | Nenhuma mudança na CLI: nenhuma flag nova, nenhum comando novo (FR-003/FR-013: idioma e peso da fonte não são escolha do usuário). |
| IV. Neutralidade Geográfica | Sem mudança: nenhum dado geográfico fixo é introduzido. |
| V. Funcionamento Offline | A fonte continua embutida no binário — reforça o princípio. |
| VI. Testes Automatizados no Núcleo | O cálculo da largura estável, o novo raio de contorno e os rótulos em português são testados no domínio sem tocar disco; nenhum mock novo é necessário (nenhuma porta nova). |
| VII. Erros Sentinela no Domínio | Nenhum erro sentinela novo: nada que o usuário escolhe pode ficar malformado, porque nada disso é escolha do usuário. |
| VIII. Configuração Injetada | Nenhuma mudança: os rótulos, o peso da fonte e a espessura do contorno continuam constantes fixas de domínio — não há escolha de usuário para injetar. |
| IX. Portas/Service Layer/Regra de Negócio | Nenhuma porta nova, nenhum serviço novo, nenhum DTO novo. A regra de negócio nova (largura estável calculada a partir do plano inteiro, uma vez por `Scene`) vive em `internal/domain`, como método de `Scene` — a mesma entidade que já guarda o cache de glifos pelo mesmo motivo. |
| X. Testes: Given/When/Then, Builders, Isolamento | Sem mudança de convenção. |

Nenhuma violação; nada a registrar em Rastreamento de Complexidade.

## Estrutura do Projeto

### Documentação (desta funcionalidade)

```text
specs/012-overlay-ptbr-readability/
├── plan.md              # Este arquivo (saída do comando /speckit-plan)
├── research.md          # Saída da Fase 0 (comando /speckit-plan)
├── data-model.md        # Saída da Fase 1 (comando /speckit-plan)
├── quickstart.md        # Saída da Fase 1 (comando /speckit-plan)
├── contracts/           # Saída da Fase 1 (comando /speckit-plan)
│   └── frame-files-change.md
└── tasks.md             # Saída da Fase 2 (comando /speckit-tasks — NÃO criado pelo /speckit-plan)
```

### Código-Fonte (raiz do repositório)

Projeto único já existente (hexagonal), sem mudança de layout — só edições
pontuais nos mesmos quatro arquivos de domínio que a etapa anterior já
tocou:

```text
internal/domain/
├── vector_font.go              # newVectorFace: font/gofont/gobold no lugar de .../goregular
├── frame_screen_overlay.go     # rótulos em português; distanceBlockText/elevationBlockText/
│                                #   timeBlockText (compartilhadas entre draw e o cálculo da
│                                #   largura estável); overlayPpem extraído; raio do contorno
│                                #   em função de ppem; screenOverlay ganha o campo panelWidth
│                                #   (deixa de calculá-lo sozinho)
├── frame_scene.go              # Scene ganha o cache da largura estável (panelWidth/
│                                #   panelWidthHeight/panelWidthSet) e o método
│                                #   numericPanelWidth(plan, height, ppem); Render o calcula
│                                #   antes de construir screenOverlay
├── render_tuning.go            # RenderVersion 3→4; OverlayOutlineRatio redefinida como fração
│                                #   de ppem (não mais da altura do quadro) e seu valor ajustado
└── (testes correspondentes a cada arquivo acima)
```

Nenhum arquivo fora de `internal/domain` muda: sem porta nova, sem serviço
novo, sem flag de CLI nova, sem campo novo em `config`.

**Decisão de Estrutura**: toda a mudança fica dentro dos mesmos quatro
arquivos que `011-overlay-polish` já estabeleceu como o ponto de extensão
do acabamento das sobreposições — nenhuma pasta nova, nenhum arquivo novo
de domínio além dos testes.
