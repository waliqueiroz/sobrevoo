# Modelo de Dados: Bloco de Velocidade na Sobreposição

**Feature**: `014-speed-overlay-block` | **Data**: 2026-10-04

Esta etapa não cria nenhuma entidade nova de alto nível — estende quatro
tipos já existentes e acrescenta uma função de domínio. Entidades e campos
não listados aqui permanecem exatamente como `specs/003-camera-path-planning/
data-model.md` e `specs/009-frame-overlays/data-model.md` já os descrevem.

## `CameraFrame` (`internal/domain/camera_plan.go`) — campo novo

| Campo | Tipo | Semântica |
|---|---|---|
| `MarkerSpeed` | `float64` (metros por segundo) | Velocidade média da atividade numa janela de tempo fixa (`CameraTuning.SpeedWindow`) em torno de `ActivityElapsed`, encurtada nos extremos do trajeto quando a janela completa não cabe. `0` quando o trajeto não tem horário em todos os pontos (mesmo critério de `TimeReference == TimeReferenceDistance`) ou quando a janela efetiva coberta é zero (trajeto mais curto que a janela). Quantizado com `speedStep = 1e-3` antes de ser gravado, como os demais campos numéricos do quadro. |

**Validação** (`CameraPlan.Validate()`): `MarkerSpeed` deve ser um número
finito e não-negativo — a mesma checagem que `CameraToMarkerDistance` já
tem, acrescentada à mesma lista de `switch` por quadro.

**Identidade** (`CameraPlan.ID()`): `quantize(f.MarkerSpeed, speedStep)`
entra no hash, na mesma lista de escritas por quadro que
`CameraToMarkerDistance` já ocupa (ver `research.md` item 6).

## `CameraTuning` (`internal/domain/camera_plan_parameters.go`) — campo novo

| Campo | Tipo | Semântica |
|---|---|---|
| `SpeedWindow` | `time.Duration` | A duração fixa da janela de tempo usada para calcular `MarkerSpeed` — a mesma em todo vídeo e trajeto, nunca lida de flag nem de arquivo de configuração do usuário. Valor padrão: 30s (`research.md` item 3), fornecido por `config.CameraTuning.SpeedWindowSeconds` e mapeado em `domainCameraTuning`. |

## `Route` (`internal/domain/duration.go`) — método novo

| Método | Assinatura | Semântica |
|---|---|---|
| `DistanceAt` | `func (r Route) DistanceAt(distances []float64, at time.Duration) (float64, bool)` | O espelho exato de `Route.TimeAt`: dado o tempo decorrido desde o primeiro ponto, devolve a distância percorrida correspondente (interpolada linearmente, com a mesma forma segura de `lerp`), clampada às duas pontas do trajeto. `ok` é `false` quando `!r.allHaveTime()`, o mesmo critério de `TimeAt`/`Duration`. |

## `OverlayBlock` / `OverlayConfig` (`internal/domain/frame_overlay_config.go`) — extensão

| Elemento | Mudança |
|---|---|
| `OverlayBlockSpeed` | Constante nova, valor `"speed"` — o quinto nome aceito por `NewOverlayConfig`/`--overlay-blocks`. |
| `OverlayConfig.Speed` | Campo booleano novo, `false` a menos que `"speed"` esteja na lista de blocos pedida e `Enabled` seja `true` — a mesma regra dos quatro campos já existentes. |
| `OverlayConfig.Fingerprint()` | Ganha um quinto dígito (`flag(o.Speed)`), na mesma posição relativa (depois de `Profile`) — participa de `FrameSetID` como os demais, sem mudança de formato além do dígito extra. |

Diferente dos quatro blocos já existentes, `Speed` **não** entra na lista
de blocos que `config.RenderDefaults.OverlayBlocks` traz por padrão — é
opt-in puro (FR-007 do `spec.md`); isso é uma questão de qual configuração
o composition root (`cmd/sobrevoo/config_mapping.go`) injeta, não uma regra
de domínio diferente para esse bloco.

## Arquivo de plano exportado — campo novo (JSON, `format_version: 3`)

| Campo | Tipo | Obrigatório | Semântica |
|---|---|---|---|
| `frames[].marker.speed_mps` | número (3 casas) | Sim | `CameraFrame.MarkerSpeed`, em metros por segundo. Ver `contracts/plan-file-v3.md`. |

## Diagrama de dependência dos dados novos

```text
Route.Points[].Time  ──┐
Route (trackRoute)      ├─► Route.DistanceAt(route.Distances, t) ─┐
route.Distances ────────┘                                         │
                                                                    ├─► CameraFrame.MarkerSpeed
CameraFrame.ActivityElapsed (já existente) ──► janela [t-½w, t+½w] ┘
                                               (CameraTuning.SpeedWindow)
```

Nenhuma seta nova sai de `Track`/`GeoSlice` — a entrada continua sendo
exclusivamente o trajeto tratado já carregado em memória durante
`TreatedTrack.PlanCamera` (FR-001 do `spec.md`: "derivada exclusivamente da
distância percorrida e dos horários já presentes no trajeto").
