# Contrato de CLI: `--base-map`/`--elevation` em `geodata check`, `geodata slice` e `fly`

**Feature**: `010-geo-data-source-control` | **Data**: 2026-09-29

Duas flags novas, com o mesmo nome, o mesmo formato e o mesmo efeito em
`sobrevoo geodata check`, `sobrevoo geodata slice` e `sobrevoo fly` —
seguindo o mesmo padrão que as flags de aparência e de sobreposição já
estabeleceram (`specs/008-frame-appearance/contracts/appearance-flags.md`,
`specs/009-frame-overlays/contracts/overlay-flags.md`): um único parser
compartilhado, `parseSourceSelection`
(`internal/infra/inbound/cli/source_selection.go`). Os contratos existentes
desses três comandos (`specs/002-geo-data-registry/contracts/cli.md`,
`specs/004-geo-data-slice/contracts/cli.md`,
`specs/007-full-flight-pipeline/contracts/cli.md`) permanecem válidos —
este arquivo documenta só o que esta etapa acrescenta. `sobrevoo plan` e
`sobrevoo geodata elevation` não recebem estas flags (Clarifications e
Suposições de `spec.md`).

## As duas flags

| Flag | Formato | Padrão | Efeito quando ausente |
|---|---|---|---|
| `--base-map <nome>` | nome de um registro já feito com `geodata register` | (nenhum) | Seleção automática, idêntica à de antes desta etapa (FR-009). |
| `--elevation <nome>` | nome de um registro já feito com `geodata register` | (nenhum) | Idem, para elevação. |

As duas são independentes (FR-005): informar uma não afeta a outra. Nenhuma
das duas valida o nome no momento em que a flag é lida — a validação
acontece depois, contra o registro (ver "Saída (erro)" abaixo), a mesma
ordem que `--base-map`/`--elevation` teriam se fossem checadas por um
comando qualquer.

## `sobrevoo geodata check`

```text
sobrevoo geodata check <arquivo-de-trajeto> [--base-map <nome>] [--elevation <nome>]
```

O relatório de cobertura (`Coverage: ...`, `Base map sources used: ...`,
`Elevation sources used: ...`, ver `specs/002-geo-data-registry/contracts/
cli.md`) passa a refletir exclusivamente a fonte pedida para cada tipo
informado — nunca a escolha automática para esse tipo. Uma fonte pedida que
não cobre toda a área **não é um erro**: aparece como lacuna no relatório
(`Uncovered segments: ...`), exatamente como uma lacuna da seleção
automática já aparecia antes desta etapa — `check` só relata, nunca recusa
por cobertura incompleta (FR-007).

## `sobrevoo geodata slice`

```text
sobrevoo geodata slice <arquivo-de-plano> [--export <caminho>] [--overwrite] [--base-map <nome>] [--elevation <nome>]
```

O recorte usa exclusivamente a fonte pedida para cada tipo informado
(FR-010); a linha `Sources:` do resumo (`specs/004-geo-data-slice/contracts/
cli.md`) lista só ela para esse tipo. Uma fonte pedida que não cobre toda a
área que o plano precisa recusa exatamente como uma cobertura incompleta já
recusa hoje: `domain.ErrAreaNotCovered`, código `19`, sem nenhuma mudança
nesse código (FR-007).

## `sobrevoo fly`

```text
sobrevoo fly <trajeto> --output voo.mp4 [flags de plano] [--base-map <nome>] [--elevation <nome>] [--resolution] [flags de aparência] [flags de sobreposição] [--quality] [--keep <diretório>] [--overwrite]
```

A validação das duas flags acontece contra o registro no mesmo ponto em que
`geodata slice` já a faria — dentro da etapa de recorte, logo após a
cobertura (o mesmo ponto que já recusa cedo hoje, `specs/007-full-flight-
pipeline/`). Uma fonte pedida que não cobre toda a área recusa com o mesmo
`ErrAreaNotCovered`, código `19`.

### Com `--keep <diretório>`

A escolha de fonte participa da decisão de reaproveitar o recorte guardado
(FR-011): uma execução anterior que usou, para um tipo, uma fonte diferente
da pedida agora (seja porque outra foi pedida explicitamente, seja porque
antes era automática e agora é explícita, ou vice-versa) faz o recorte
guardado ser tratado como desatualizado — um novo é gerado e grava por
cima, com a mesma proteção de "outro conjunto" que já existe hoje
(`--overwrite`, senão `domain.ErrSliceDestinationExists`). O plano guardado
não é afetado (`plan` não lê dados geográficos) — só a decisão de
reaproveitar o **recorte** depende da fonte.

## Exit codes novos

| Código | Erro sentinela | Quando |
|---|---|---|
| `56` | `ErrDataSourceTypeMismatch` | `--base-map`/`--elevation` pede um nome que está registrado, mas como o outro tipo (por exemplo, `--base-map` apontando para um registro de elevação). |
| `58` | `ErrSliceUsesDifferentSource` | Reservado para `GeoSlice.EnsureUsesSources` (antes `EnsureUsesSelection`) — hoje só usado internamente por `FlightService.reuseSlice` para decidir reaproveitar ou não um recorte guardado (FR-011); nunca devolvido a um usuário por nenhum comando desta etapa, porque a decisão de não reaproveitar nunca é, por si só, um erro — o comando simplesmente gera um recorte novo. Mapeado aqui por completude, como todo sentinela do domínio já é (`exit_code.go`). |

`ErrDataSourceNotRegistered` (já existente, código `9`) é reaproveitado
para um nome pedido que não existe no registro em nenhum tipo — a mensagem
cita a flag (`--base-map` ou `--elevation`) e o nome pedido, para diferenciar
do uso já existente em `geodata remove`.

`57` é o outro código novo desta etapa — ver `contracts/registry-clear.md`.
Nenhum código existente muda de significado.

## Reuso futuro

Como todo método de serviço nesta ferramenta, `GeoDataService.CheckCoverage`
e `GeoSliceService.Generate` recebem `domain.SourceSelection` como um
argumento comum, sem nenhum conhecimento de flag de CLI — um futuro adapter
HTTP passaria a mesma seleção a partir do corpo de uma requisição, sem
duplicar a validação (Princípio III).
