# 🤖 commitai

AI-powered git commit messages using Google Gemini. One command, smart commits.

```bash
$ git add .
$ commitai

🔍 Analyzing staged changes...

📂 Staged files (3):
  ✚ src/auth/login.go
  ● src/api/routes.go
  ✚ tests/auth_test.go

✨ Generating commit message(s) with Gemini...

💬 Suggested commit message:
────────────────────────────────────────────────────────────
feat(auth): add JWT login endpoint with route registration

- Implement JWT-based authentication in login.go
- Register /auth/login route in routes.go
- Add unit tests for auth flow
────────────────────────────────────────────────────────────

⚡ Use this message? [Y/n/e(dit)]: y

✅ Committed successfully!
```

---

## ✨ Features

- **Single AI request** — all staged files analyzed in one Gemini call
- **Auto-detection** — smart mode picks single or granular commits based on your changes
- **Granular mode** — separate commit per file, each with its own message
- **Conventional Commits** — follows the standard format automatically
- **Release automation** — AI-generated release notes + automatic semver tagging
- **Multilingual** — supports English, Portuguese, Spanish, and more
- **GitHub Actions** — full CI/CD workflow with AI-generated releases

---

## 📦 Installation

### One-line installer (macOS & Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/kaiqui/commitai/main/scripts/install.sh | bash
```

### Manual installation

```bash
# Download the binary for your platform from GitHub Releases
# https://github.com/kaiqui/commitai/releases

# macOS (Apple Silicon)
curl -fsSL https://github.com/kaiqui/commitai/releases/latest/download/commitai_darwin_arm64.tar.gz | tar -xz
sudo mv commitai /usr/local/bin/

# macOS (Intel)
curl -fsSL https://github.com/kaiqui/commitai/releases/latest/download/commitai_darwin_amd64.tar.gz | tar -xz
sudo mv commitai /usr/local/bin/

# Linux (x86_64)
curl -fsSL https://github.com/kaiqui/commitai/releases/latest/download/commitai_linux_amd64.tar.gz | tar -xz
sudo mv commitai /usr/local/bin/

# Linux (ARM64)
curl -fsSL https://github.com/kaiqui/commitai/releases/latest/download/commitai_linux_arm64.tar.gz | tar -xz
sudo mv commitai /usr/local/bin/
```

### Build from source

```bash
git clone https://github.com/kaiqui/commitai
cd commitai
go build -o commitai .
sudo mv commitai /usr/local/bin/
```

---

## ⚙️ Setup

1. **Get a free Gemini API key**: [aistudio.google.com/app/apikey](https://aistudio.google.com/app/apikey)

2. **Configure commitai**:
   ```bash
   commitai config --key YOUR_GEMINI_API_KEY
   ```

   Or use environment variable (useful for CI):
   ```bash
   export GEMINI_API_KEY=your_key_here
   ```

---

## 🚀 Usage

### Basic (auto mode)

```bash
git add .
commitai
```

Auto mode detects whether to use a single commit or granular commits based on the number and type of staged files.

### Commit modes

| Mode | Command | Description |
|------|---------|-------------|
| Auto | `commitai` | Smart detection (default) |
| All | `commitai --all` | One message for all files |
| Granular | `commitai --granular` | One commit per file |
| Dry run | `commitai --dry-run` | Preview without committing |
| Skip confirm | `commitai --yes` | No prompts |

### Language support

```bash
commitai --lang pt-br   # Portuguese
commitai --lang en      # English (default)
commitai --lang es      # Spanish
```

Or set permanently:
```bash
commitai config --lang pt-br
```

### Commit style

```bash
commitai config --style conventional  # feat(scope): message (default)
commitai config --style simple        # Plain messages
```

---

## 🏷️ Release Management

Create AI-generated releases with automatic semantic versioning:

```bash
# Let AI decide the version bump
commitai release --auto

# Manual version control
commitai release --major    # 1.0.0 → 2.0.0
commitai release --minor    # 1.0.0 → 1.1.0
commitai release --patch    # 1.0.0 → 1.0.1

# Specific version
commitai release --tag v2.0.0

# Preview only
commitai release --auto --dry-run

# Create and push to origin
commitai release --auto --push
```

---

## ⚙️ Configuration

Config file: `~/.commitai.json`

```json
{
  "language": "pt-br",
  "commit_style": "conventional",
  "max_tokens": 1024,
  "model": "gemini-2.0-flash"
}
```

Available models:
- `gemini-2.0-flash` (default, fastest)
- `gemini-1.5-pro` (more capable)
- `gemini-1.5-flash` (balanced)

Show current config:
```bash
commitai config --show
```

---

## 🔄 GitHub Actions

The included workflow (`.github/workflows/release.yml`) provides:

- **CI**: Build + vet on every push
- **Auto release**: Triggered by `[release]` in commit message or manual dispatch
- **AI versioning**: Gemini analyzes commits to suggest major/minor/patch
- **AI release notes**: Categorized changelog generated automatically
- **Multi-platform builds**: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64

### Setup

1. Add your Gemini API key as a repository secret: `GEMINI_API_KEY`

2. Trigger a release:
   ```bash
   git commit -m "feat: add new feature [release]"
   git push
   ```
   
   Or go to **Actions → Release → Run workflow** and choose the bump type.

---

## 📋 Command Reference

```
commitai [flags]          Generate commit message for staged files
commitai config           Configure settings
commitai release          Create a tagged release
commitai version          Show version

Flags:
  -g, --granular    One commit per staged file
  -a, --all         One commit for all staged files
  -d, --dry-run     Preview without committing
  -y, --yes         Skip confirmation prompts
  -l, --lang        Language for messages
      --style       Commit style (conventional, simple)

Release flags:
      --auto        AI-suggested version bump
      --major       Bump major version
      --minor       Bump minor version
      --patch       Bump patch version
      --tag         Use specific tag
  -p, --push        Push tag to origin
  -d, --dry-run     Preview without creating tag
```

---

## 📄 License

MIT
