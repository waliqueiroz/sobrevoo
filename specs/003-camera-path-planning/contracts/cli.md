# Contrato de CLI: `sobrevoo plan`

**Feature**: `003-camera-path-planning` | **Data**: 2026-09-19

Único contrato de linha de comando desta etapa: o comando `plan`, que expõe
`CameraPlanService.Generate` e `CameraPlanService.Export` (ver
`data-model.md`) através de `internal/infra/inbound/cli`. Não há API HTTP nem
GUI (fora de escopo). Os contratos de `inspect` (etapa 1) e de `geodata`
(etapa 2) permanecem inalterados. O formato do arquivo exportado tem
contrato próprio em [`plan-file.md`](./plan-file.md).

## `sobrevoo plan`

```text
sobrevoo plan <arquivo-de-trajeto> [--duration <segundos>] [--fps <n>]
              [--distance low|medium|high] [--tilt low|medium|high]
              [--export <caminho>] [--overwrite]
```

- `<arquivo-de-trajeto>` (posicional, obrigatório): trajeto GPS local, no
  mesmo formato aceito por `sobrevoo inspect` (GPX). Passa pelo tratamento
  completo da etapa 1 — parse, reordenação, descarte, simplificação e
  suavização — com o nível padrão (FR-001).
- `--duration` (opcional): duração do vídeo em segundos; número decimal
  aceito (ex.: `45.5`). Quando omitida, a duração é **calculada a partir da
  comprimento do trajeto** (de 20 s a 120 s, crescimento sublinear, nunca
  abaixo do mínimo daquele trajeto; FR-003a). Quando informada, é usada
  exatamente e passa pelas validações de duração.
- `--fps` (padrão `30`, de `Config.PlanDefaults`): quadros por segundo; número decimal aceito
  (ex.: `29.97`), entre 1 e 120.
- `--distance` (padrão `medium`): afastamento da câmera em relação ao
  trajeto.
- `--tilt` (padrão `medium`): inclinação da câmera; `high` olha de mais
  perto da vertical, `low` mais rente ao horizonte.
- `--export` (opcional): grava o plano completo neste caminho, no formato
  de `plan-file.md`. Sem esta flag, nada é gravado em disco.
- `--overwrite` (opcional, só faz sentido com `--export`): permite
  substituir um arquivo já existente no destino. Usada sem `--export`, é
  erro de uso (código `2`).

### Saída (sucesso)

Resumo legível em inglês, em `stdout`, no formato abaixo (FR-019). Os
rótulos e a ordem são estáveis:

```text
Duration: 42.0 s (automatic)
Frame rate: 30.0 fps
Frames: 1260
Camera altitude: 127.3 m - 912.8 m
Camera distance: 300.0 m - 1204.5 m
Time reference: clock
Smoothed spans: 2
  0.00 s - 1.20 s (heading)
  31.40 s - 32.10 s (zoom)
```

- `Duration` traz `(automatic)` quando calculada pela ferramenta e
  `(requested)` quando informada pelo usuário.
- `Time reference` é `clock` ou `distance`. Quando a distância foi usada, a
  linha traz o motivo: `Time reference: distance (no time data)` (o trajeto
  não tem horário em todos os pontos) ou
  `Time reference: distance (time data is inconsistent)` (tem horário, mas
  ele não pôde ser usado).
- Sem trechos suavizados, a linha é `Smoothed spans: none` e nenhuma lista
  segue.
- Com `--export`, uma última linha `Plan written to <caminho>` é impressa.

**Código de saída**: `0`.

### Saída (erro)

Mensagem em `stderr`, sem resumo impresso (a exportação, quando pedida, é feita
antes de qualquer saída) e — em qualquer erro — sem arquivo criado ou
alterado no destino da exportação.

| Cenário | Erro sentinela do domínio | Código |
|---|---|---|
| Arquivo de trajeto vazio | `domain.ErrEmptyFile` | `1` |
| Formato de trajeto não suportado | `domain.ErrUnsupportedFormat` | `2` |
| Pontos insuficientes (antes ou depois do tratamento) | `domain.ErrInsufficientPoints[AfterCleaning]` | `3` |
| Arquivo de trajeto inexistente ou não legível (E/S) | erro genérico | `4` |
| `--duration` informada e ≤ 0 | `domain.ErrInvalidDuration` | `10` |
| `--fps` ≤ 0 ou fora de 1–120 | `domain.ErrInvalidFrameRate` | `11` |
| `--duration` informada abaixo do mínimo para o trajeto (nunca ocorre com duração automática) | `domain.ErrDurationTooShort` | `12` |
| Trajeto curto demais (comprimento < 50 m) | `domain.ErrTrackTooShort` | `13` |
| Trajeto grande demais (abrangência > 2 000 km) | `domain.ErrTrackTooLarge` | `14` |
| Destino da exportação já existe (sem `--overwrite`) | `domain.ErrPlanDestinationExists` | `15` |
| Destino da exportação inválido (diretório inexistente, sem permissão) | `domain.ErrPlanDestinationInvalid` | `16` |
| Valor não numérico em `--duration`/`--fps`, nível desconhecido, argumento faltando, `--overwrite` sem `--export` | erro de uso da CLI | `2` |

As mensagens dos erros `10` a `16` sempre nomeiam o parâmetro ou o destino
problemático (SC-008); a de código `12` traz a duração mínima calculada
para aquele trajeto e taxa de quadros; a de `13`, o comprimento encontrado e o
mínimo exigido; a de `11`, o intervalo válido; a de `14`, a abrangência
encontrada e o máximo aceito; a de `15`, o caminho e a dica
`use --overwrite to replace it`.

O código `2` é compartilhado por três casos herdados/consolidados: formato
de trajeto não suportado (etapa 1), erro de uso da CLI (valor não numérico,
nível desconhecido, argumento faltando, `--overwrite` sem `--export`) e o
que o Cobra já classificava assim. A mensagem distingue cada um (FR-023).

### Exemplos

```bash
# Plano padrão (duração automática, 30 fps, distância e inclinação médias), só o resumo
sobrevoo plan pedalada.gpx

# Vídeo curto, câmera mais afastada e mais vertical, plano exportado
sobrevoo plan pedalada.gpx --duration 45 --fps 60 --distance high --tilt high --export plano.json

# Regerar sobre o mesmo arquivo
sobrevoo plan pedalada.gpx --export plano.json --overwrite
```
