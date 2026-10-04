# Contrato do Arquivo de Plano Exportado — acréscimo desta etapa

**Feature**: `013-treatment-level-flags` | **Data**: 2026-10-03

Estende `specs/003-camera-path-planning/contracts/plan-file.md` (a
referência completa da estrutura) e
`specs/009-frame-overlays/contracts/plan-file-v2.md` (as mudanças da versão
2, que continua sendo a versão atual). Este arquivo documenta só o que
muda: dois campos novos, opcionais, em `parameters` — nenhum campo
existente muda de tipo, posição ou significado, e `format_version`
**permanece `2`** (FR-007 do `spec.md`: os níveis passam a ser registrados
no plano exportado, junto dos outros parâmetros).

## Estrutura (só os campos novos, em contexto)

```json
{
  "format_version": 2,
  "parameters": {
    "duration_s": 42.0,
    "frame_rate": 30,
    "distance": "medium",
    "tilt": "medium",
    "simplification": "high",
    "smoothing": "low",
    "aspect_ratio": "9:16"
  }
}
```

## Campos novos

| Campo | Tipo | Semântica |
|---|---|---|
| `parameters.simplification` | texto | `low`, `medium` ou `high` — o nível de simplificação efetivamente usado para tratar o trajeto antes de planejar a câmera (o mesmo valor que `--simplification` recebeu, informado ou padrão). |
| `parameters.smoothing` | texto | `low`, `medium` ou `high` — idem, para a suavização. |

## Por que `format_version` não sobe

Os dois campos são **opcionais na leitura**, exatamente como
`parameters.aspect_ratio` já é desde a etapa 3 (ver a linha de
`format_version` em `plan-file.md`: "acrescentar campos não muda a
versão"). Um arquivo escrito antes desta etapa simplesmente não tem
`simplification`/`smoothing`; um leitor desta etapa em diante lê a ausência
como `medium` — o mesmo valor que `parseLevel` já devolve para qualquer
texto desconhecido ou vazio (comportamento já existente, reaproveitado sem
mudança). Isto é diferente do que a etapa 9 fez com `activity_time_s`/
`marker.elevation_m`/`marker.gain_m`, que **subiram** a versão para `2`
porque passaram a ser **obrigatórios** — aqui nenhum campo passa a ser
obrigatório, então não há motivo para recusar um arquivo mais antigo por
format_version.

## Garantias verificáveis novas

1. Um plano exportado por `sobrevoo plan --simplification=high
   --smoothing=low` traz `"simplification": "high"` e `"smoothing": "low"`
   em `parameters`.
2. Um plano exportado sem informar as duas flags traz `"simplification":
   "medium"` e `"smoothing": "medium"` (o padrão), não a ausência dos
   campos — `sobrevoo plan`/`fly` sempre escrevem os dois campos; é só um
   arquivo **anterior a esta etapa** que pode não os ter.
3. Um arquivo sem `parameters.simplification`/`.smoothing` é lido sem erro,
   como se os dois campos fossem `"medium"` — não é `ErrPlanFileInvalid`
   nem `ErrPlanFormatVersionUnsupported`.
4. Dois planos do mesmo trajeto e dos mesmos demais parâmetros, exportados
   com `simplification`/`smoothing` diferentes, têm `CameraPlan.ID()`
   diferentes (FR-006) — verificável comparando os arquivos: o conteúdo de
   `frames[]` também difere (a geometria do trajeto tratado muda), então
   os dois arquivos nunca são, por acidente, byte a byte iguais.

## Compatibilidade

Um plano exportado antes desta etapa continua sendo lido normalmente por
`render frame`, `render all`, `video` e `geodata slice` — nenhum desses
comandos precisa saber com que nível de tratamento um plano foi gerado para
usá-lo; a ausência dos dois campos só importa para a decisão de
reaproveitamento de `fly --keep` (ver
`contracts/treatment-level-flags.md`), onde é tratada como `medium` em
ambos.
