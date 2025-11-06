# chex

**chex** (check + executable) is a CLI tool that verifies required CLI tools are installed and match version requirements. Inspired by verchew, but with modern semver ranges, custom version extraction patterns, and support for existing version management files.

## Features

- ✅ **Semver ranges** - Full support for `>=1.20.0`, `^1.2.0`, `||`, `&&`, etc.
- ✅ **Existence checks** - Verify tools exist without checking version
- ✅ **Custom version patterns** - Extract versions with regex patterns
- ✅ **Optional tools** - Mark tools as optional (warnings instead of failures)
- ✅ **Selective checking** - Check specific tools: `chex go docker`
- ✅ **Multiple formats** - Pretty colored output, quiet mode, or JSON
- ✅ **Multi-source config** - Merge configs from mise.toml, .tool-versions
- ✅ **CI-friendly** - Exit codes and JSON output for automation

## Installation

```bash
# From GitHub releases (recommended)
# Download the latest release from https://github.com/drape-io/chex/releases

# Using Docker
docker run ghcr.io/drape-io/chex:latest

# From source
go install github.com/drape-io/chex/cmd/chex@latest

# Or build locally
git clone https://github.com/drape-io/chex.git
cd chex
just build
./bin/chex
```

## Quick Start

1. Generate a sample configuration:
```bash
chex init
```

2. Edit `.chex.toml` to define your tool requirements:
```toml
[go]
cli = "go"
version = ">=1.20.0"

[docker]
cli = "docker"
version = ">=20.0.0"
optional = true

[make]
cli = "make"
# No version = existence check only
```

3. Run chex:
```bash
chex
```

## Configuration

### Basic Tool Definition

```toml
[tool-name]
cli = "command"              # Required: CLI command to execute
version = ">=1.0.0"          # Optional: semver constraint
version_arg = "--version"   # Optional: argument to get version
version_pattern = "v?(\\d+\\.\\d+\\.\\d+)"  # Optional: regex to extract version
optional = false             # Optional: mark as optional
message = "Custom message"   # Optional: message on failure
name = "Display Name"        # Optional: override display name
```

### Semver Constraints

chex supports full semver constraint syntax:

```toml
[go]
version = ">=1.20.0"        # At least 1.20.0

[node]
version = "^18.0.0 || ^20.0.0"  # Node 18.x or 20.x (LTS only)

[terraform]
version = ">=1.0.0 <2.0.0"  # 1.x only

[kubectl]
version = "~1.28.0"         # 1.28.x only
```

### Existence Checks

Omit the `version` field to only check if a tool exists (without executing it):

```toml
[make]
cli = "make"
# No version = uses exec.LookPath, doesn't run make
```

### Optional Tools

Mark tools as optional to show warnings instead of failures:

```toml
[kubectl]
cli = "kubectl"
version = ">=1.25.0"
optional = true
message = "kubectl is optional but useful for Kubernetes development"
```

### External Sources

chex can automatically merge tool definitions from `mise.toml` and `.tool-versions`:

```toml
[chex]
# Optional: explicitly configure sources
sources = [
  { path = "mise.toml", type = "mise" },
  { path = ".tool-versions", type = "tool-versions" },
  { path = "../team-standards.toml", type = "chex" }
]
# Or omit sources to auto-detect mise.toml and .tool-versions
```

**Default behavior (no `[chex]` section):**
- Auto-detects `mise.toml` and `.tool-versions` in current directory
- Merges them with `.chex.toml` (`.chex.toml` takes precedence)

**Disable external sources:**
```toml
[chex]
sources = []  # Empty array = only use .chex.toml
```

## Usage

### Basic Commands

```bash
# Check all tools
chex

# Check specific tools
chex go docker

# Generate sample config
chex init

# Show version
chex --version
```

### Output Formats

```bash
# Pretty colored output (default)
chex

# Quiet mode (only show failures)
chex --quiet

# JSON output (for CI/scripting)
chex --output=json
```

### CI Integration

chex automatically exits with code 1 on failures, making it CI-friendly:

```bash
# In GitHub Actions
chex

# With JSON output for parsing
chex --output=json

# Quiet mode for minimal output
chex --quiet
```

Example GitHub Actions workflow:
```yaml
- name: Check required tools
  run: |
    curl -Lo chex https://github.com/drape-io/chex/releases/latest/download/chex_linux_amd64
    chmod +x chex
    ./chex
```

### Custom Config Location

```bash
# Use custom config file
chex --config=my-tools.toml

# Check from specific directory
chex --root=/path/to/project
```

## Output Examples

### Pretty Output (Default)

```
Checking CLI Tools...

✅ go
   $ go version
   go version go1.25.4 darwin/arm64
   Required: >=1.25.0
   Installed: 1.25.4

❌ docker
   $ docker --version
   Error: docker: command not found
   Required: >=20.0.0

⚠️  kubectl (optional)
   Error: kubectl: command not found
   Message: kubectl is optional but useful for Kubernetes development

✅ make
   Found at: /usr/bin/make

Summary: 2 passed, 1 failed, 1 optional missing
```

### JSON Output

```json
{
  "tools": [
    {
      "name": "go",
      "cli": "go",
      "required": true,
      "status": "pass",
      "version_required": ">=1.25.0",
      "version_installed": "1.25.4",
      "command": "go version",
      "output": "go version go1.25.4 darwin/arm64\n"
    },
    {
      "name": "docker",
      "cli": "docker",
      "required": true,
      "status": "fail",
      "version_required": ">=20.0.0",
      "error": "docker: command not found"
    }
  ],
  "summary": {
    "total": 4,
    "passed": 2,
    "failed": 1,
    "optional_missing": 1
  }
}
```

## Real-World Examples

### Go Project

```toml
[go]
cli = "go"
version = ">=1.20.0"

[golangci-lint]
cli = "golangci-lint"

[just]
cli = "just"
```

### Node.js Project

```toml
[node]
name = "Node.js"
cli = "node"
version = "^18.0.0 || ^20.0.0"  # LTS versions only
version_arg = "-v"

[npm]
cli = "npm"
version = ">=8.0.0"

[docker]
cli = "docker"
optional = true
message = "Docker is needed for containerized development"
```

### DevOps Tools

```toml
[terraform]
cli = "terraform"
version = "~1.5.0"  # 1.5.x only

[kubectl]
cli = "kubectl"
version = ">=1.25.0"

[helm]
cli = "helm"
version = "^3.0.0"

[aws]
cli = "aws"
# Just verify AWS CLI exists
```

## Tips

### Custom Version Patterns

Some tools don't follow standard semver output. Use `version_pattern` to extract the version:

```toml
[node]
cli = "node"
version = "^18.0.0"
version_arg = "-v"
version_pattern = "v?(\\d+\\.\\d+\\.\\d+)"  # Strips leading 'v'
```

### Combining with mise/asdf

If you use mise or asdf, chex can auto-detect your `.tool-versions`:

```
# .tool-versions
go 1.25.4
node 20.10.0
```

Then just run `chex` - it will auto-detect and check these tools!

### Running in Docker

```bash
# Mount your config directory
docker run -v $(pwd):/app ghcr.io/drape-io/chex:latest

# With specific config file
docker run -v $(pwd):/app ghcr.io/drape-io/chex:latest --config=/app/my-tools.toml
```

## Development

```bash
# Build
just build

# Run tests
just test

# Lint
just lint

# Fix linting issues
just lint-fix

# Run locally
just run

# Run with args
just run go docker
```

## Why chex?

- **Modern**: Uses proper semver constraint libraries
- **Flexible**: Supports existence checks, optional tools, custom patterns
- **Integrates**: Works with mise.toml and .tool-versions
- **Fast**: Written in Go, single binary, no dependencies
- **Generic**: No hardcoded tool knowledge, stays flexible

## License

MPL 2.0 - See LICENSE file for details.

## Contributing

Contributions welcome! Please open an issue or PR.
