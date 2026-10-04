# Plano de Implementação: Acabamento das Sobreposições de Tela

**Branch**: `011-overlay-polish` | **Data**: 2026-10-02 | **Especificação**: [spec.md](./spec.md)

**Entrada**: Especificação de funcionalidade de `/specs/011-overlay-polish/spec.md`

**Nota**: Este template é preenchido pelo comando `/speckit-plan`; sua definição descreve o fluxo de execução.

## Resumo

Corrige o acabamento visual das sobreposições de tela que a nona etapa
introduziu, sem mudar nenhum valor exibido, nenhum bloco, nenhuma
configuração ou flag: o texto passa de uma fonte bitmap ampliada por fator
inteiro para uma fonte vetorial embutida (`golang.org/x/image/font/
gofont/goregular`, já ao alcance do módulo), rasterizada por um rasterizador
próprio, escrito à mão no domínio — nunca o `golang.org/x/image/vector` da
biblioteca padrão de fontes do Go, que tem um caminho em assembly só para
amd64 (`acc_amd64.s`, com fallback em Go puro para as demais arquiteturas) e
reintroduziria exatamente o tipo de divergência entre arquiteturas que o
hash de referência de `frame_scene_test.go` já proíbe. Os três painéis
numéricos passam a compartilhar a largura do mais largo; o marcador do
perfil de elevação ganha um raio proporcional à altura do quadro com piso em
pixels, como o marcador do terreno já tem; o texto ganha um contorno escuro
fixo; e a margem de segurança passa de uma única fração do lado menor para
três frações (topo, laterais, uma fração maior na base), adequadas ao vídeo
vertical. Nenhuma porta, nenhum serviço, nenhuma flag e nenhum formato de
arquivo mudam — só constantes e um novo rasterizador em
`internal/domain`, e `RenderVersion` sobe de 2 para 3 porque os pixels de um
quadro com sobreposições mudam de verdade.

## Contexto Técnico

**Linguagem/Versão**: Go 1.26.4 (`go.mod`), sem mudança.

**Dependências Principais**: `golang.org/x/image` (já em `go.mod`, hoje
direta, usada para `font/inconsolata`) — passa a ser usada também para
`font/sfnt` (parsing de glifos vetoriais, puramente leitura de dado
embutido, sem I/O) e `font/gofont/goregular` (a fonte TrueType "Go Regular",
já embutida como bytes Go pelo próprio módulo). Nenhuma dependência nova em
`go.mod`, nenhum arquivo de fonte vendorizado pelo projeto.
**Deliberadamente não usado**: `golang.org/x/image/vector.Rasterizer` (risco
de divergência entre arquiteturas, ver `research.md` item 1).

**Armazenamento**: arquivos locais (plano `.json`, recorte `.zip`, quadros
`.png`), sem mudança de formato nem de tecnologia — nenhum campo novo em
nenhum arquivo; só os pixels dos quadros mudam.

**Testes**: `go test ./... -cover`, `testify` + `uber-go/mock`, sem mudança
de ferramenta. O hash de referência de `frame_scene_test.go` precisa de uma
nova constante — os pixels mudam de propósito (CLAUDE.md: "a constante só
muda junto com `domain.RenderVersion`", que sobe nesta etapa).

**Plataforma-Alvo**: CLI de linha de comando, macOS/Linux, amd64/arm64 — o
determinismo byte a byte entre arquiteturas (já exigido desde a etapa 5) é
a restrição central desta etapa: é o motivo de não usar o rasterizador
vetorial pronto da biblioteca.

**Tipo de Projeto**: CLI de projeto único (`cmd/sobrevoo`), sem mudança de
estrutura.

**Metas de Desempenho**: rasterizar um glifo é mais caro que replicar pixels
de um bitmap, mas cada glifo único (dígitos, poucas letras de rótulo, poucos
símbolos) é rasterizado e colocado em cache uma vez por execução de
`Scene` — reaproveitado em todos os quadros de um `render all`/`fly`, do
mesmo jeito que `imagery` já cacheia peças decodificadas. O desenho da
sobreposição continua ordens de grandeza mais barato que o traçado de raios
por pixel do terreno.

**Restrições**: byte a byte determinístico entre goroutines, processos e
arquiteturas (Constitution; CLAUDE.md "O desenho dos quadros") — a restrição
que motiva escrever o próprio rasterizador; nenhuma leitura nova do trajeto
GPS nem dos dados geográficos registrados (mantido, nada muda aqui);
nenhuma dependência de fonte instalada no sistema (FR-001); offline
(Princípio V).

**Escala/Escopo**: mesma escala das etapas 5–9 (até ~3840×2160, até 432 000
quadros por plano); o alfabeto de glifos usados pelas sobreposições é
pequeno e fixo (dígitos, `:`, `+`, espaço, e as letras dos rótulos `DIST`,
`ELEV`, `GAIN`, `TIME`, `m`, `km`), então o custo do cache de glifos é
desprezível.

## Verificação da Constituição

*PORTÃO: Deve passar antes da Fase 0 de pesquisa. Reverificar após o design da Fase 1.*

| Princípio | Verificação |
|---|---|
| I. Arquitetura Hexagonal | O novo rasterizador vetorial e o desenho da sobreposição continuam código de domínio puro (`internal/domain`), como o resto do renderizador (etapa 5). A fonte embutida e o parser de glifos (`golang.org/x/image/font/sfnt`, `font/gofont/goregular`) são dado estático e leitura pura, não I/O — mesmo raciocínio já aplicado a `font/inconsolata` na etapa 9. |
| II. Portas para Toda Dependência Externa | Nenhuma porta nova: nenhum I/O é introduzido. `sfnt`/`goregular` são bibliotecas puras sobre dado embutido (mesmo carve-out de `time.Now()`, já estendido a `inconsolata` na etapa 9). |
| III. Entrypoints Descartáveis | Nenhuma mudança na CLI: nenhuma flag nova, nenhum comando novo. |
| IV. Neutralidade Geográfica | Sem mudança: nada de dado geográfico fixo é introduzido. |
| V. Funcionamento Offline | A fonte continua embutida no binário (nenhum download, nenhuma fonte do sistema) — reforça o princípio. |
| VI. Testes Automatizados no Núcleo | O novo rasterizador, a largura compartilhada dos painéis, o raio do marcador do perfil e as margens por borda são testados no domínio sem tocar disco; nenhum mock novo é necessário (nenhuma porta nova). |
| VII. Erros Sentinela no Domínio | Nenhum erro sentinela novo: nada que o usuário escolhe pode ficar malformado, porque nada disso é escolha do usuário (FR-010/FR-011). |
| VIII. Configuração Injetada | Nenhuma mudança: os novos valores (margens por borda, raio do marcador do perfil, cor do contorno) são constantes fixas de domínio, exatamente como `OverlayPanelColor`/`OverlayMarginRatio` já são hoje — não há escolha de usuário para injetar. |
| IX. Portas/Service Layer/Regra de Negócio | Nenhuma porta nova, nenhum serviço novo, nenhum DTO novo. A regra de negócio nova (rasterização vetorial determinística, largura compartilhada, dimensionamento do marcador do perfil) vive inteiramente em `internal/domain`, como funções/métodos do novo tipo de rasterizador e de `screenOverlay`, a mesma entidade que já desenha a sobreposição. |
| X. Testes: Given/When/Then, Builders, Isolamento | Sem mudança de convenção; o rasterizador ganha testes de domínio isolados (given/when/then), comparando cobertura/máscara de glifos conhecidos, sem depender de nenhuma biblioteca de imagem externa para a asserção. |

Nenhuma violação; nada a registrar em Rastreamento de Complexidade.

## Estrutura do Projeto

### Documentação (desta funcionalidade)

```text
specs/011-overlay-polish/
├── plan.md              # Este arquivo (saída do comando /speckit-plan)
├── research.md          # Saída da Fase 0 (comando /speckit-plan)
├── data-model.md        # Saída da Fase 1 (comando /speckit-plan)
├── quickstart.md        # Saída da Fase 1 (comando /speckit-plan)
├── contracts/           # Saída da Fase 1 (comando /speckit-plan)
│   └── frame-files-change.md
└── tasks.md             # Saída da Fase 2 (comando /speckit-tasks — NÃO criado pelo /speckit-plan)
```

### Código-Fonte (raiz do repositório)

Projeto único já existente (hexagonal), sem mudança de layout — só arquivos
novos e extensões pontuais dentro de `internal/domain`, a única camada que
este acabamento toca:

```text
internal/domain/
├── vector_font.go              # NOVO: parsing dos glifos (sfnt + goregular) e o rasterizador
│                                #   determinístico próprio (supersampling, sem vector.Rasterizer)
├── frame_screen_overlay.go     # overlayFace vetorial no lugar do bitmap; largura compartilhada
│                                #   dos três painéis numéricos; contorno do texto; raio do
│                                #   marcador do perfil pela nova constante
├── render_tuning.go            # RenderVersion 2→3; -OverlayMarginRatio,
│                                #   +OverlayTopMarginRatio/+OverlaySideMarginRatio/
│                                #   +OverlayBottomMarginRatio; +OverlayTextOutlineColor,
│                                #   +OverlayOutlineRatio/+OverlayOutlineMinWidth;
│                                #   +ProfileMarkerRadiusRatio/+ProfileMarkerMinRadius
├── frame_scene.go              # Scene: guarda o rasterizador/cache de glifos construído uma
│                                #   vez em NewScene, passado a screenOverlay a cada Render
└── (testes correspondentes a cada arquivo acima)

internal/domain/builddomain/
└── (builders novos só se um teste do rasterizador pedir; nenhum builder existente muda)
```

Nenhum arquivo fora de `internal/domain` muda: sem porta nova, sem serviço
novo, sem flag de CLI nova, sem campo novo em `config`, sem mudança em
`cmd/sobrevoo`.

**Decisão de Estrutura**: toda a mudança fica dentro de `internal/domain`,
seguindo o mesmo ponto de extensão que a etapa 9 já estabeleceu para o
desenho da sobreposição — só que, desta vez, sem nenhuma extensão às camadas
externas, porque nada aqui é uma escolha nova do usuário.
