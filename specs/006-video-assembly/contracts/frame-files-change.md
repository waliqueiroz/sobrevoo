# Mudança no Contrato dos Arquivos de Quadro (etapa 5)

**Feature**: `006-video-assembly` | **Data**: 2026-09-27

Esta etapa acrescenta **um bloco** ao PNG que a etapa 5 grava, definido em
`specs/005-frame-rendering/contracts/frame-files.md` (Clarificação de 2026-09-27
da spec desta etapa). É a única mudança em artefatos de etapas anteriores; na
implementação, o contrato da etapa 5 recebe a mesma nota (como a etapa 5 fez
com o contrato do recorte, `slice-file-change.md`).

## O bloco

Logo depois do bloco `tEXt` da identificação do conjunto, um **segundo** bloco
`tEXt`: palavra-chave `Sobrevoo`, um byte nulo, e o texto `plan=<id>`:

```text
IHDR
tEXt  "Sobrevoo\0frame-set=<id do conjunto>"      ← como hoje
tEXt  "Sobrevoo\0plan=<id do plano>"              ← novo
IDAT …
IEND
```

| Campo | Semântica |
|---|---|
| `plan=<id>` | `CameraPlan.ID()` do plano de que o quadro foi desenhado: SHA-256 hexadecimal (64 caracteres minúsculos) da codificação canônica do plano — a mesma identificação que a etapa 4 grava no recorte (`plan_id`). Igual para o mesmo conteúdo de plano, seja qual for a formatação do arquivo. |

Nenhum outro bloco é acrescentado, e continua **sem** data, hora, caminho,
versão do binário ou resumo dentro do arquivo.

## Versão e compatibilidade

- **`RenderVersion` sobe de 1 para 2.** A identificação do conjunto (`frame-set=`)
  inclui a versão do desenho, então os quadros antigos passam a ser de **outro
  conjunto**: sem `--overwrite`, `render all` os recusa (`ErrFrameSetConflict`,
  código `37`, "N frames of another set; use --overwrite to replace them, or
  another --output"); com `--overwrite`, os redesenha, e remove, como sempre, os
  nossos de outro conjunto com número fora do plano. Não há migração: o quadro se
  desenha de novo com um comando.
- **Os pixels não mudam**: o desenho é o mesmo; a versão marca o formato do
  arquivo. O hash de pixels que os testes da etapa 5 fixam continua igual.
- A igualdade byte a byte do desenho continua: o `plan=` é função só do conteúdo
  do plano (FR-015 da etapa 5 mantido).
- Um leitor que só conhece o bloco do conjunto (anterior a esta mudança) segue
  funcionando: os blocos auxiliares que ele não conhece são ignorados (o contrato
  já dizia isso). Um quadro **sem** o bloco `plan=` é "nosso, sem identificação do
  plano": a etapa 6 o recusa (`ErrFramesWithoutPlanID`, código `44`).
- `render frame` grava o mesmo bloco, para o quadro isolado ser idêntico byte a
  byte ao mesmo quadro de `render all` (FR-018a da etapa 5).

## O que muda no código da etapa 5

- `domain.FrameMark{SetID, PlanID}` e `domain.NewFrameMark(...)`; as portas
  `FrameRepository.Save` e `FrameExporter.Export` recebem a marca no lugar do
  `FrameSetID`; `FrameService` a monta com o plano.
- `pngfile/png_mark.go`: `encodeFrame` grava os dois blocos; `readMark` devolve o
  conjunto e o plano (o plano vazio se o bloco não existe).
- `domain.FrameFile` ganha `PlanID`, `Width`, `Height` e `Whole`, e a porta
  `FrameRepository` ganha `List` (`data-model.md`).
- `domain.RenderVersion` = 2; os testes que comparam a versão e os que montam a
  marca são atualizados.
- `specs/005-frame-rendering/contracts/frame-files.md` recebe a nota do bloco
  `plan=` e da nova versão; o `render all` e o `render frame` imprimem o mesmo
  resumo (a identificação não aparece nele).
