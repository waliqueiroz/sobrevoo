# Guia de Validação Rápida: Leitura e Tratamento de Trajeto GPS

**Feature**: `001-gps-track-processing` | **Data**: 2026-09-13

Este guia valida, de ponta a ponta, que o comando `sobrevoo inspect` funciona
conforme `spec.md` e o contrato em `contracts/cli.md`. Não substitui os testes
automatizados — é a validação manual/exploratória do resultado da feature.

## Pré-requisitos

- Go 1.26 instalado.
- Repositório clonado, na branch `001-gps-track-processing`.
- Nenhuma configuração adicional: a ferramenta funciona 100% offline
  (Princípio V da constituição).

## Preparar

```bash
cd sobrevoo
go build -o bin/sobrevoo ./cmd/sobrevoo
```

## Cenário 1 — Trajeto GPX válido e completo (História de Usuário 1)

1. Use um arquivo GPX real de uma corrida, pedalada ou caminhada (exportado de
   qualquer app/dispositivo), com altitude e tempo em todos os pontos.
2. Rode:
   ```bash
   ./bin/sobrevoo inspect caminho/para/atividade.gpx
   ```
3. **Resultado esperado**: resumo em `stdout` com formato identificado como
   GPX, contagem de pontos original e após tratamento, distância total, ganho
   de elevação, duração, e área geográfica ocupada. Código de saída `0`.

## Cenário 2 — Trajeto GPX sem altitude ou sem tempo (História de Usuário 1)

1. Use (ou edite) um arquivo GPX removendo todas as tags de altitude (`<ele>`)
   (ou de tempo, `<time>`) dos pontos.
2. Rode o mesmo comando apontando para esse arquivo.
3. **Resultado esperado**: o resumo indica claramente que o ganho de elevação
   (ou a duração) não pôde ser calculado por falta do dado correspondente —
   nunca um valor calculado a partir de dado inexistente.

## Cenário 3 — Trajeto cruzando o antimeridiano (História de Usuário 1, FR-023)

1. Use um arquivo GPX sintético com pontos alternando entre longitudes
   próximas de `+179.9` e `-179.9`.
2. Rode o comando.
3. **Resultado esperado**: distância total coerente com o deslocamento real
   (não um valor absurdamente alto por "dar a volta ao mundo"), e a área
   geográfica ocupada reportada como atravessando o antimeridiano (ver
   `BoundingBox.CrossesAntimeridian` em `data-model.md`).

## Cenário 4 — Arquivo vazio (História de Usuário 2)

```bash
touch /tmp/vazio.gpx
./bin/sobrevoo inspect /tmp/vazio.gpx
echo $?   # esperado: 1
```

**Resultado esperado**: mensagem em `stderr` informando que o arquivo está
vazio; nenhum resumo impresso.

## Cenário 5 — Formato não reconhecido (História de Usuário 2)

```bash
echo "isto não é um rastreamento GPS" > /tmp/invalido.txt
./bin/sobrevoo inspect /tmp/invalido.txt
echo $?   # esperado: 2
```

## Cenário 6 — Pontos insuficientes após a limpeza (História de Usuário 2, FR-006)

1. Use um arquivo com 3 pontos, sendo 2 deles coordenadas impossíveis ou
   duplicatas consecutivas — restando apenas 1 ponto válido.
2. Rode o comando.
3. **Resultado esperado**: `stderr` indica que os pontos ficaram insuficientes
   **após a limpeza** (mensagem distinta do Cenário com arquivo já
   originalmente insuficiente). Código de saída `3`.

## Cenário 7 — Pontos problemáticos descartados (História de Usuário 3)

1. Use um arquivo com pelo menos: um ponto de coordenada impossível (ex.:
   latitude `200`), um par de pontos consecutivos idênticos, e um salto de
   posição implausível (ex.: dois pontos a 50 km de distância com 1 segundo de
   intervalo).
2. Rode o comando.
3. **Resultado esperado**: o resumo relata a contagem de pontos descartados
   por cada um dos três motivos (FR-011), e a distância total não reflete o
   salto implausível.

## Cenário 8 — Níveis de simplificação e suavização (História de Usuário 4)

```bash
./bin/sobrevoo inspect atividade.gpx --simplification=low --smoothing=low
./bin/sobrevoo inspect atividade.gpx --simplification=high --smoothing=high
```

**Resultado esperado**: a quantidade de pontos após o tratamento no segundo
comando é menor do que no primeiro (SC-006); o formato geral do trajeto é
preservado em ambos os casos.

## Rodar a suíte automatizada

```bash
go test ./... -cover
```

**Resultado esperado**: todos os testes passam; cobertura alta em
`internal/domain` (funções puras e regras de tratamento), conforme exigido
pelo Princípio VI da constituição. Os testes do núcleo (`internal/domain` e
`internal/application`) não tocam disco, rede, nem processo externo — as
portas (`TrackParser`, `Simplifier`, `Smoother`) são substituídas por mocks
gerados com `go.uber.org/mock` nos testes de `internal/application`.
