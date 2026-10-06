# Modelo de Dados: Informação de Versão da CLI

**Feature**: `017-version-flag` | **Data**: 2026-10-05

Esta etapa não acrescenta nenhuma entidade de domínio — a especificação é
explícita: a versão "não entra no domínio" (`spec.md`, Entidades-Chave).
Documentado aqui apenas para registrar a única entidade conceitual
envolvida e onde ela vive de fato.

## Versão do Programa

Um texto curto, nunca persistido, que identifica o build em execução.
Não é um tipo Go dedicado — é uma `string` simples, resolvida uma única
vez por processo.

| Origem | Exemplo de valor | Quando é usado |
|---|---|---|
| Fixada no build (`-ldflags "-X main.version=..."`) | `v0.1.0` | Compilação de release publicada fora do fluxo `go install` (FR-007); prevalece sobre as outras duas (FR-008). |
| Metadado do módulo, de uma tag real (`debug.ReadBuildInfo`) | `v0.1.0` | Binário instalado via `go install .../sobrevoo@v0.1.0` (FR-005) — nenhuma flag de build envolvida. |
| Metadado do módulo, sem tag (`debug.ReadBuildInfo`) | `v0.0.0-20261006015100-c169b2b38ac9+dirty` (compilado num checkout git, com o "VCS stamping" padrão do Go) ou `(devel)` (sem VCS disponível) | Binário compilado localmente, sem tag nem valor fixado no build (FR-006) — nenhum dos dois é uma tag real, então nenhum inventa um número de release. |

**Ciclo de vida**: resolvida uma única vez, em `cmd/sobrevoo` (composition
root), no início de `run()`; entregue como parâmetro simples a
`cli.NewRootCommand(version string)`, que a guarda em
`cobra.Command.Version` para o próprio Cobra formatar na saída de
`--version`. Nunca é lida de volta, nunca é comparada, nunca participa de
nenhuma outra decisão da ferramenta (ao contrário de, por exemplo,
`Appearance`/`OverlayConfig`, que entram na identidade de um conjunto de
quadros) — seu único destino é ser impressa.
