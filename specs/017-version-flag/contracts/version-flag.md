# Contrato de CLI: `sobrevoo --version`

**Feature**: `017-version-flag` | **Data**: 2026-10-05

Flag nova no comando raiz. Nenhum contrato existente muda — nenhum outro
comando aceita ou reconhece `--version` (FR-009).

```text
sobrevoo --version
```

Sem argumentos posicionais e sem nenhuma outra flag necessária. Reconhecida
apenas na invocação direta do comando raiz, sem subcomando (ver Casos
Extremos de `spec.md` — combiná-la com um subcomando fica fora de escopo).

## Saída (sempre sucesso)

Uma única linha em `stdout`, nome do programa seguido da versão, nessa
ordem, separados por um espaço — nada além disso (FR-002):

```text
sobrevoo v0.1.0
```

Quando o binário não carrega nenhuma tag de módulo nem versão fixada no
build, o resultado comum (compilado dentro de um checkout git, com o "VCS
stamping" que o Go já faz por padrão) é uma pseudo-versão derivada do
commit atual:

```text
sobrevoo v0.0.0-20261006015100-c169b2b38ac9+dirty
```

Sem VCS disponível (`-buildvcs=false`, sem `git` instalado, ou build sem
suporte a módulo), o resultado é o literal `(devel)` — igualmente honesto:

```text
sobrevoo (devel)
```

**Código de saída**: `0`, sempre — `--version` não tem caminho de erro.

Nenhum arquivo é lido ou escrito, nenhuma entrada do registro de dados
geográficos é consultada, e nenhuma rede é acessada (FR-003).

## Precedência das fontes da versão

| Fonte | Quando prevalece |
|---|---|
| Fixada no build (`-ldflags "-X main.version=..."`) | Sempre que não está vazia — mesmo que o binário também carregue uma versão de módulo diferente (FR-008). |
| Versão do módulo, de uma tag real (`go install .../sobrevoo@vX.Y.Z`) | Quando nada foi fixado no build, e o binário foi instalado a partir de uma tag (FR-005). |
| Versão do módulo, sem tag (pseudo-versão `v0.0.0-<timestamp>-<hash>[+dirty]` com VCS, ou `(devel)` sem VCS) | Quando nada foi fixado no build e o binário não foi instalado a partir de uma tag — nunca um número de versão inventado (FR-006). |

Ver `research.md` item 3 para o mecanismo exato e `quickstart.md` para a
validação dos três casos com binários reais.

## Exit code

Nenhum código de saída novo — `--version` sempre sai com `0`; nenhum
sentinela de erro é introduzido por esta etapa.
