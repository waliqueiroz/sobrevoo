# Contrato do Arquivo de Plano Exportado — mudanças da versão 3

**Feature**: `014-speed-overlay-block` | **Data**: 2026-10-04

Estende `specs/003-camera-path-planning/contracts/plan-file.md` (a
referência completa da estrutura) e `specs/009-frame-overlays/contracts/
plan-file-v2.md` (as mudanças da versão 2). Este arquivo documenta só o que
muda: `format_version` sobe de `2` para `3`, e um campo novo e obrigatório
por quadro — necessário para que o bloco de velocidade (`spec.md` desta
etapa) nunca precise reler o trajeto GPS (FR-001).

## Estrutura (só o campo novo, em contexto)

```json
{
  "format_version": 3,
  "frames": [
    {"index":0,"time_s":0,"activity_time_s":0,"phase":"opening","camera":{"...":"..."},"heading_deg":0,"tilt_deg":60,"marker":{"lat":-23.5505199,"lon":-46.6333094,"distance_m":0,"elevation_m":842.5,"gain_m":0,"speed_mps":0.0},"camera_to_marker_m":1054.02}
  ]
}
```

## Campo novo

| Campo | Tipo | Semântica |
|---|---|---|
| `format_version` | inteiro | Sobe de `2` para `3` — muda porque `frames[].marker.speed_mps` passa a ser **obrigatório**, algo que uma versão anterior não escreve (ao contrário de `parameters.simplification`/`.smoothing` da etapa 13, acrescentados opcionais sem subir a versão). Um arquivo `format_version: 2` ou anterior é recusado com `ErrPlanFormatVersionUnsupported` e uma mensagem que orienta a gerar o plano de novo (mesmo sentinela e mesmo código de saída — `18` — que já recusam qualquer versão desconhecida hoje). |
| `frames[].marker.speed_mps` | número (3 casas decimais) | A velocidade média da atividade (metros por segundo) numa janela de tempo fixa e documentada (`CameraTuning.SpeedWindow`, 30s — `research.md` item 3) em torno do instante real da atividade neste quadro (`activity_time_s`). `0` em todo quadro quando `summary.time_reference` é `"distance"` (nenhum instante real existe para ancorar a janela) — o mesmo tratamento que `activity_time_s` já recebe nesse caso. Nunca a velocidade instantânea entre dois pontos consecutivos do trajeto. |

## Garantias verificáveis novas

1. `frames[].marker.speed_mps` é sempre finito e ≥ 0.
2. Em todo quadro de um plano com `summary.time_reference == "distance"`,
   `frames[].marker.speed_mps == 0`.
3. Dois planos do mesmo trajeto e dos mesmos demais parâmetros, cujo
   `frames[].marker.speed_mps` difira em pelo menos um quadro, produzem
   `CameraPlan.ID()` diferentes — nunca são tratados como o mesmo plano por
   `fly --keep`.
4. Um arquivo com `format_version: 2` (ou `1`) nunca é aceito por um leitor
   desta etapa em diante — `ErrPlanFormatVersionUnsupported`, antes de
   qualquer outra validação, como já acontecia na transição `1` → `2`.
5. Um arquivo `format_version: 3` sem `frames[].marker.speed_mps` é
   `ErrPlanFileInvalid` (mesmo padrão que `camera_to_marker_m` ausente já é
   hoje).

## Compatibilidade

Nenhum campo das versões 1/2 muda de tipo, de posição ou de significado.
`sobrevoo plan` (e `fly`, internamente) passa a exportar sempre
`format_version: 3`; não existe uma flag para pedir um formato antigo.
Qualquer plano `.json` gerado antes desta etapa deve ser gerado de novo
antes de ser usado por `render frame`, `render all`, `video` ou `fly` a
partir desta versão da ferramenta.
