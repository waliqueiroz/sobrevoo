# Guia de Validação Rápida: Registro de Dados Geográficos Locais

**Feature**: `002-geo-data-registry` | **Data**: 2026-09-13

Este guia valida, de ponta a ponta, que os comandos `sobrevoo geodata
register|list|remove|check` funcionam conforme `spec.md` e o contrato em
`contracts/cli.md`. Não substitui os testes automatizados — é a validação
manual/exploratória do resultado da feature.

## Pré-requisitos

- Go 1.26 instalado.
- Repositório clonado, na branch `002-geo-data-registry`.
- Nenhuma configuração adicional: a ferramenta funciona 100% offline
  (Princípio V da constituição).
- Pelo menos um arquivo MBTiles (mapa base) e um GeoTIFF em CRS geográfico
  (relevo) reais, cobrindo a mesma região de um trajeto GPX de teste — por
  exemplo, exportados de qualquer ferramenta de mapas offline (MBTiles) e
  um recorte de DEM público em WGS84 (GeoTIFF), ou gerados sinteticamente
  para os cenários de antimeridiano.

## Preparar

```bash
cd sobrevoo
go build -o bin/sobrevoo ./cmd/sobrevoo
```

O registro é armazenado em `~/.sobrevoo/registry.json` (ver `research.md`
item 5). Para repetir os cenários abaixo a partir de um estado limpo,
remova esse arquivo entre execuções:

```bash
rm -rf ~/.sobrevoo
```

## Cenário 1 — Registrar um mapa base e um relevo (História de Usuário 1)

```bash
./bin/sobrevoo geodata register mapa-regiao.mbtiles --name europa-central-mapa
./bin/sobrevoo geodata register relevo-regiao.tif --name europa-central-relevo
```

**Resultado esperado**: cada comando confirma o registro criado, com tipo
(`base map`/`elevation`) e área geográfica coberta determinados
automaticamente — sem que você tenha informado nenhum dos dois. Código de
saída `0` em ambos.

Em seguida, confirme que o registro independe do diretório de execução
(FR-008, Cenário de Aceitação 6 da História de Usuário 1):

```bash
cd /tmp
/caminho/para/sobrevoo/bin/sobrevoo geodata list
cd -
```

**Resultado esperado**: os dois registros do Cenário 1 continuam
aparecendo, mesmo executando a partir de um diretório diferente do usado
para registrá-los.

## Cenário 2 — Listar registros, incluindo um arquivo movido (História de Usuário 3)

```bash
mv relevo-regiao.tif /tmp/relevo-regiao.tif
./bin/sobrevoo geodata list
```

**Resultado esperado**: a listagem mostra os dois registros; o de nome
`europa-central-relevo` aparece sinalizado como indisponível (arquivo não
encontrado no caminho original), e o outro registro continua sendo exibido
normalmente. Restaure o arquivo antes de continuar:

```bash
mv /tmp/relevo-regiao.tif relevo-regiao.tif
```

## Cenário 3 — Recusar arquivo inexistente, ilegível ou de formato não suportado (História de Usuário 1)

```bash
./bin/sobrevoo geodata register /caminho/que/nao/existe.mbtiles --name x
echo $?   # esperado: 5

echo "isto não é um dado geográfico" > /tmp/invalido.mbtiles
./bin/sobrevoo geodata register /tmp/invalido.mbtiles --name y
echo $?   # esperado: 7
```

**Resultado esperado**: mensagens claras em `stderr` explicando o motivo em
cada caso; nenhum registro é criado.

## Cenário 4 — Recusar nome duplicado (História de Usuário 1)

```bash
./bin/sobrevoo geodata register mapa-regiao.mbtiles --name europa-central-mapa
echo $?   # esperado: 8
```

**Resultado esperado**: mensagem em `stderr` informando que o nome já está
em uso; o registro original (do Cenário 1) permanece intacto.

## Cenário 5 — Remover um registro sem apagar o arquivo (História de Usuário 4)

```bash
./bin/sobrevoo geodata remove europa-central-mapa
./bin/sobrevoo geodata list
ls mapa-regiao.mbtiles   # o arquivo original continua existindo no disco
```

**Resultado esperado**: o registro `europa-central-mapa` some da listagem;
`mapa-regiao.mbtiles` continua no disco, inalterado.

```bash
./bin/sobrevoo geodata remove nome-que-nao-existe
echo $?   # esperado: 9
```

## Cenário 6 — Trajeto totalmente coberto (História de Usuário 2)

Usando um trajeto GPX cuja extensão geográfica esteja inteiramente dentro
das áreas registradas (re-registre `europa-central-mapa` se removido no
Cenário 5):

```bash
./bin/sobrevoo geodata check trajeto-na-regiao.gpx
```

**Resultado esperado**: relatório indicando cobertura total, listando os
registros de mapa base e de relevo usados. Código de saída `0`.

## Cenário 7 — Trajeto parcialmente coberto (História de Usuário 2, FR-015)

Usando um trajeto GPX que começa dentro da área registrada e termina fora
dela:

```bash
./bin/sobrevoo geodata check trajeto-parcial.gpx
```

**Resultado esperado**: relatório indicando cobertura parcial, com pelo
menos um subtrecho não coberto — descrito pelas coordenadas geográficas de
início e fim desse subtrecho — e indicando se falta mapa base, relevo, ou
ambos nesse trecho.

## Cenário 8 — Nenhum registro cadastrado (História de Usuário 2)

Com o registro vazio (ou usando um diretório de configuração limpo):

```bash
./bin/sobrevoo geodata check qualquer-trajeto.gpx
```

**Resultado esperado**: relatório indicando o trajeto inteiro como não
coberto.

## Cenário 9 — Duas fontes sobrepostas do mesmo tipo (História de Usuário 2, FR-016)

```bash
./bin/sobrevoo geodata register mapa-grande.mbtiles --name regiao-ampla
./bin/sobrevoo geodata register mapa-pequeno.mbtiles --name regiao-especifica
./bin/sobrevoo geodata check trajeto-na-regiao.gpx
```

Onde `mapa-pequeno.mbtiles` cobre uma área menor, contida na área de
`mapa-grande.mbtiles`, e ambas cobrem o trajeto.

**Resultado esperado**: o relatório de cobertura lista `regiao-especifica`
(a fonte mais específica, de menor área) entre os registros de mapa base
usados — mesma escolha em execuções repetidas (determinismo, SC-005).

## Cenário 10 — Antimeridiano (Princípio IV, FR-018)

Usando um GeoTIFF/MBTiles sintético cuja área cruza a longitude 180° e um
trajeto GPX sintético com pontos alternando entre longitudes próximas de
`+179.9` e `-179.9` (mesmo estilo de fixture do Cenário 3 do
`quickstart.md` da etapa 1):

```bash
./bin/sobrevoo geodata register dados-antimeridiano.mbtiles --name antimeridiano-mapa
./bin/sobrevoo geodata register dados-antimeridiano.tif --name antimeridiano-relevo
./bin/sobrevoo geodata check trajeto-antimeridiano.gpx
```

**Resultado esperado**: cobertura calculada corretamente, sem tratamento
diferente do aplicado a qualquer outra região do planeta.

## Rodar a suíte automatizada

```bash
go test ./... -cover
```

**Resultado esperado**: todos os testes passam, incluindo os já existentes
da etapa 1 (o ajuste em `inspect_track_service.go` para reaproveitar
`track_loading.go` não deve alterar nenhum comportamento observável já
testado). Os testes do núcleo (`internal/domain` e `internal/application`)
não tocam disco, rede, nem processo externo — as novas portas
(`GeoDataInspector`, `GeoDataRegistry`, `FileChecker`, `Clock`) são
substituídas por mocks gerados com `go.uber.org/mock` nos testes de
`internal/application`; nenhum teste abre um MBTiles/GeoTIFF real ou toca o
arquivo de registro de verdade.
