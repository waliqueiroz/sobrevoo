# Contrato do Arquivo de Plano Exportado

**Feature**: `003-camera-path-planning` | **Data**: 2026-09-19

Este é o contrato de dados que as etapas seguintes (desenho do mapa,
renderização) e qualquer ferramenta externa consomem (FR-020). É um único
arquivo JSON, codificação UTF-8, gerado por `sobrevoo plan --export`. Um
mesmo trajeto com os mesmos parâmetros produz um arquivo idêntico byte a
byte (SC-003).

## Estrutura

Ordem de campos fixa; objetos de topo indentados com 2 espaços, cada item de
`frames` compacto, em uma linha.

```json
{
  "format_version": 1,
  "parameters": {
    "duration_s": 42,
    "frame_rate": 30,
    "distance": "medium",
    "tilt": "medium"
  },
  "summary": {
    "duration_s": 42,
    "duration_mode": "automatic",
    "frame_rate": 30,
    "frame_count": 1260,
    "camera_altitude_m": { "min": 127.312, "max": 912.804 },
    "camera_distance_m": { "min": 300.011, "max": 1204.527 },
    "time_reference": "clock",
    "time_fallback_reason": "",
    "smoothed_spans": [
      { "start_s": 0, "end_s": 1.2, "quantity": "heading" }
    ]
  },
  "frames": [
    {"index":0,"time_s":0,"phase":"opening","camera":{"lat":-23.5505199,"lon":-46.6333094,"altitude_m":912.804},"heading_deg":0,"tilt_deg":60,"marker":{"lat":-23.5505199,"lon":-46.6333094,"distance_m":0},"camera_to_marker_m":1054.02}
  ]
}
```

## Campos

| Campo | Tipo | Semântica |
|---|---|---|
| `format_version` | inteiro | Versão do formato. Muda **somente** quando um campo é removido ou muda de significado; acrescentar campos não muda a versão. Esta etapa emite `1`. |
| `parameters.duration_s` | número | Duração **efetiva** do plano, em segundos: a informada pelo usuário ou a calculada automaticamente |
| `parameters.frame_rate` | número | Quadros por segundo |
| `parameters.distance`, `parameters.tilt` | texto | `low`, `medium` ou `high` |
| `summary.*` | — | Mesmos valores de `PlanSummary` (`data-model.md`) |
| `summary.duration_mode` | texto | `automatic` (calculada a partir do trajeto) ou `explicit` (informada pelo usuário) |
| `summary.time_reference` | texto | `clock` ou `distance` |
| `summary.time_fallback_reason` | texto | Vazio quando `clock`; senão, o motivo do recuo |
| `summary.smoothed_spans[]` | lista | Vazia (`[]`) quando não houve suavização forçada. `quantity`: `heading`, `tilt`, `zoom` ou `target_speed` |
| `frames[].index` | inteiro | 0 a `frame_count − 1` |
| `frames[].time_s` | número | `index / frame_rate` |
| `frames[].phase` | texto | `opening`, `following` ou `closing` |
| `frames[].camera.lat`, `.lon` | número | Graus decimais; `lon` em `[-180, 180)`; 7 casas decimais |
| `frames[].camera.altitude_m` | número | Altura da câmera **acima do ponto observado** (relativa ao chão sob o alvo, não sobre o nível do mar — esta etapa não conhece relevo); 3 casas |
| `frames[].heading_deg` | número | Direção horizontal para a qual a câmera aponta; graus, sentido horário a partir do norte, em `[0, 360)`; 3 casas |
| `frames[].tilt_deg` | número | Inclinação em graus **abaixo do horizonte**: 0 = horizontal, 90 = vertical para baixo; 3 casas |
| `frames[].marker.lat`, `.lon` | número | Posição do marcador da atividade; 7 casas |
| `frames[].marker.distance_m` | número | Metros percorridos desde o início do trajeto tratado; não decresce entre quadros; 3 casas |
| `frames[].camera_to_marker_m` | número | Distância em linha reta da câmera ao marcador; 3 casas |

Convenções: números sem zeros à direita desnecessários; nenhum `NaN` nem
infinito (o núcleo nunca os produz); `time_fallback_reason` e listas vazias
sempre presentes, nunca omitidos; tempo em segundos como número, nunca
como texto.

## Garantias verificáveis (usadas nos testes e no quickstart)

1. `len(frames) == summary.frame_count == round(duration_s × frame_rate)`.
2. Fases em ordem: `opening*`, `following+`, `closing*`.
3. `frames[0].marker.distance_m == 0` e, no último quadro,
   `marker.distance_m` = comprimento do trajeto tratado; nunca decresce.
4. Entre quadros consecutivos, as variações de `heading_deg`, `tilt_deg` e
   do zoom relativo `ln(camera_to_marker_m)` respeitam os limites por
   segundo de `research.md` item 5, e o deslocamento do alvo respeita o
   limite relativo.
5. `summary` é recalculável a partir de `frames` sem perda.

## Compatibilidade

O arquivo não referencia caminhos, nomes de máquina, data/hora de geração
nem versão do binário — para preservar a igualdade byte a byte entre
execuções (FR-014, FR-020). Consumidores devem ignorar campos
desconhecidos.
