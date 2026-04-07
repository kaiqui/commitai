# commitai

CLI em Go que gera mensagens de commit e release notes automaticamente usando a API do Google Gemini.

## Comandos essenciais

```bash
make build          # Compila o binário em ./commitai
make install        # Compila e instala em /usr/local/bin/commitai
make test           # go test ./...
make vet            # go vet ./...
make build-all      # Cross-compile para linux/darwin amd64/arm64 em dist/
make release-dry    # Executa ./commitai release --auto --dry-run
```

## Estrutura do projeto

```
main.go                     # Entry point — chama cmd.Execute()
cmd/
  root.go                   # Comando raiz: flags, modo auto/granular, fluxo de commit
  config.go                 # Subcomando: commitai config
  release.go                # Subcomando: commitai release
  version.go                # Subcomando: commitai version
internal/
  ai/gemini.go              # Cliente HTTP para Gemini API; build de prompts; parsing da resposta
  config/config.go          # Carrega ~/.commitai.json + env GEMINI_API_KEY
  git/git.go                # Wrappers para comandos git (staged changes, commit, tag, etc.)
```

## Fluxo principal

1. `runCommit` obtém staged changes via `git.StagedChanges()`
2. `determineMode` decide entre modo **all** (uma mensagem) ou **granular** (uma por arquivo)
3. `ai.GeminiClient.GenerateCommitMessages` faz **uma única chamada** à API com todos os diffs
4. No modo granular: faz unstage de tudo, depois `git add <file>` + `git commit` por arquivo

## Configuração

- Arquivo: `~/.commitai.json` (nunca armazena a API key se veio de env var)
- Env var: `GEMINI_API_KEY` sobrepõe o arquivo
- Modelo padrão: `gemini-2.5-flash`
- Estilos: `conventional` (padrão) | `simple`
- Idiomas: `en` (padrão) | `pt-br` | qualquer outra string

## Flags do comando raiz

| Flag | Short | Descrição |
|------|-------|-----------|
| `--granular` | `-g` | Um commit por arquivo staged |
| `--all` | `-a` | Uma mensagem para todos os arquivos |
| `--dry-run` | `-d` | Mostra mensagens sem commitar |
| `--yes` | `-y` | Pula confirmações |
| `--lang` | `-l` | Idioma da mensagem |
| `--style` | | Estilo do commit |

## CI/CD

- `.github/workflows/ci.yml` — testes e vet em cada push
- `.github/workflows/release.yml` — publica release no GitHub ao criar tag `v*`
- `.github/workflows/auto-pr.yml` — abre PR automático de `develop` para `main`

## Decisões de arquitetura relevantes

- **Uma chamada só ao Gemini**: mesmo no modo granular, todos os diffs são enviados em um único request. A resposta é parseada pelo formato `FILE: / MESSAGE: / ---`.
- **UnstageAll antes do granular**: usa `git restore --staged .` (com fallback para `git reset HEAD`) para poder fazer commits individuais por arquivo.
- **API key nunca vai para o disco se veio de env**: `config.Save` limpa o campo antes de serializar.
