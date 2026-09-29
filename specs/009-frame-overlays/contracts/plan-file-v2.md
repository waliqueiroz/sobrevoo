# Contrato do Arquivo de Plano Exportado — mudanças da versão 2

**Feature**: `009-frame-overlays` | **Data**: 2026-09-28

Estende `specs/003-camera-path-planning/contracts/plan-file.md` (a
referência completa da estrutura). Este arquivo documenta só o que muda:
`format_version` sobe de `1` para `2`, três campos novos por quadro e um novo
campo de resumo — todos necessários para que as sobreposições de tela
(`spec.md` desta etapa) nunca precisem reler o trajeto GPS nem os dados
geográficos registrados (FR-005).

## Estrutura (só os campos novos, em contexto)

```json
{
  "format_version": 2,
  "summary": {
    "...": "campos da versão 1, inalterados",
    "elevation_available": true
  },
  "frames": [
    {"index":0,"time_s":0,"activity_time_s":0,"phase":"opening","camera":{"...":"..."},"heading_deg":0,"tilt_deg":60,"marker":{"lat":-23.5505199,"lon":-46.6333094,"distance_m":0,"elevation_m":842.5,"gain_m":0},"camera_to_marker_m":1054.02}
  ]
}
```

## Campos novos

| Campo | Tipo | Semântica |
|---|---|---|
| `format_version` | inteiro | Sobe de `1` para `2` — muda porque três campos passam a ser **obrigatórios** por quadro, algo que uma versão anterior não escreve (ao contrário de `parameters.aspect_ratio`, que foi acrescentado opcional). Um arquivo `format_version: 1` é recusado com `ErrPlanFormatVersionUnsupported` e uma mensagem que orienta a gerar o plano de novo (FR-007 do `spec.md`). |
| `summary.elevation_available` | booleano | Se o trajeto tinha elevação em todos os pontos. Quando `false`, `frames[].marker.elevation_m` e `.gain_m` são `0` em todo quadro — ausência de dado, não um valor real (mesmo tratamento que `summary.time_fallback_reason` já dá à ausência de tempo). |
| `frames[].activity_time_s` | número | Tempo real decorrido da atividade (segundos, 9 casas decimais, mesma resolução de `time_s`) desde o primeiro ponto do trajeto até o instante em que o marcador está neste quadro. `0` em todo quadro quando `summary.time_reference` é `distance` (nenhum instante real existe para interpolar). **Não** é o mesmo que `time_s` (o tempo do vídeo: `index / frame_rate`) — `activity_time_s` é o relógio da atividade, não o do vídeo. |
| `frames[].marker.elevation_m` | número | Elevação do trajeto (metros, 3 casas), interpolada linearmente no ponto do marcador deste quadro. `0` quando `summary.elevation_available` é `false`. |
| `frames[].marker.gain_m` | número | Ganho de elevação acumulado (metros, 3 casas) do início do trajeto até o ponto do marcador deste quadro — não decresce entre quadros. No último quadro, igual ao ganho de elevação total do trajeto tratado (o mesmo valor que `sobrevoo inspect` relataria para o mesmo trajeto tratado com os mesmos parâmetros de simplificação/suavização). `0` quando `summary.elevation_available` é `false`. |

## Garantias verificáveis novas

1. `frames[].marker.gain_m` nunca decresce de um quadro para o seguinte.
2. `frames[last].marker.gain_m == Route.ElevationGain()` do trajeto tratado,
   bit a bit (a mesma exatidão que já vale para `frames[last].marker.
   distance_m` == comprimento do trajeto tratado).
3. `frames[0].activity_time_s == 0` quando `summary.time_reference ==
   "clock"`; em todo quadro, `activity_time_s` não decresce.
4. Um arquivo com `format_version: 1` nunca é aceito por um leitor desta
   etapa em diante — `ErrPlanFormatVersionUnsupported`, antes de qualquer
   outra validação.
5. Um arquivo `format_version: 2` sem um dos três campos novos por quadro,
   ou sem `summary.elevation_available`, é `ErrPlanFileInvalid` (mesmo
   padrão que `camera_to_marker_m` ausente já é hoje).

## Compatibilidade

Nenhum campo da versão 1 muda de tipo, de posição ou de significado.
`sobrevoo plan` (a etapa 3) passa a exportar sempre `format_version: 2`;
não existe uma flag para pedir o formato antigo. Qualquer plano `.json`
gerado antes desta etapa deve ser gerado de novo antes de ser usado por
`render frame`, `render all`, `video` ou `fly` a partir desta versão da
ferramenta.
