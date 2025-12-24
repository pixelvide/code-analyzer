# Code Analysis Tool

A tool to analyze code files and identify commented code/functions.

## 🔍 Analyzers

### HTML Analyzer
Detects commented-out HTML code blocks (`<!-- -->`)
- **Reports**: Files with commented code, comment size, ratios
- **Use**: Find dead HTML pages or large comment blocks

### PHP Analyzer
Detects commented-out functions (class methods and standalone)
- **Reports**: Files with commented functions, function names
- **Use**: Find dead PHP code and unused functions

### JS Analyzer
Detects commented-out code in JavaScript/TypeScript files
- **Reports**: Files with commented blocks (multi-line `/* */` and single-line `//`)
- **Use**: Find unused logic and technical debt in frontend code

### Conflicts Analyzer
Detects unresolved Git merge conflict markers (`<<<<<<<`, `=======`, `>>>>>>>`)
- **Reports**: Files with conflict markers, line numbers
- **Use**: Find files pushed with unresolved merge conflicts
- **Note**: May detect some false positives in CSS/comment decorators

## 🚀 Quick Start

```bash
# Build
cd scripts/code-analyzer
go build -o code-analyzer

# Run (uses analysis-config.yaml by default)
./code-analyzer

# Run with custom config
./code-analyzer -config=my-config.yaml
```

## 📋 Usage

All configuration is managed via the `analysis-config.yaml` file.

### Default Run
```bash
./code-analyzer
```

### Custom Configuration
```bash
./code-analyzer -config=staging-config.yaml
```

## ⚙️ Configuration

The `analysis-config.yaml` file controls all settings:

```yaml
dir: "api"                       # Root directory to scan
output: "artifacts/analysis"     # Output directory for JSON reports
gitlab_report: "gl-report.json"  # Optional GitLab Code Quality report path

analyzers:
  html:
    enabled: true
    min: 100          # Minimum bytes to report
    min_ratio: 10     # Minimum comment ratio %
    top: 50           # Top N files to report
    sort: "ratio"     # "ratio" or "bytes"
    exclude: ["test", "backup"]

  php:
    enabled: true
    min: 1            # Minimum commented functions
    min_ratio: 0
    top: 50
    exclude: ["vendor", "tests"]
    
  js:
    enabled: true
    top: 50
    
  conflicts:
    enabled: true
```

## 🎛️ Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-config` | `analysis-config.yaml` | Path to YAML configuration file |

## 🐳 Docker Support

### Build
```bash
docker build -t code-analyzer .
```

### Run
```bash
docker run --rm -v $(pwd):/app/data code-analyzer
```

### CI/CD (GHCR)
A GitHub Actions workflow (`.github/workflows/docker-publish.yml`) is included to build and publish the container image to GHCR on pushes to `main`.

## 🧪 Testing & Linting

### Unit Tests
Run standard Go tests:
```bash
go test ./... -v
```

### Linting
The project uses `golangci-lint` for static analysis. A workflow (`.github/workflows/lint.yml`) runs this on every push.

## 🎯 GitLab Code Quality
The tool can generate a GitLab-compatible Code Quality report.

```yaml
code-quality:
  stage: test
  script:
    - ./code-analyzer -config=analysis-config.yaml
  artifacts:
    paths:
      - artifacts/*.json
      - gl-code-quality-report.json
    expire_in: 30 days
```

Ensure `analysis-config.yaml` has `gitlab_report` set to the desired output path.

## 🏗️ Architecture & Development

### Project Structure
```
scripts/code-analyzer/
├── main.go                    # Entry point and CLI
├── go.mod                     # Go module definition
├── analyzers/
│   ├── analyzer.go           # Analyzer interface (contract)
│   ├── html/                 # HTML analyzer
│   ├── php/                  # PHP analyzer
│   ├── js/                   # JS/TS analyzer
│   └── conflicts/            # Conflicts analyzer
├── models/                   # Data structures
├── utils/                    # Shared utilities
└── Dockerfile                # Container definition
```

### Key Components
1.  **Analyzer Interface**: Defines the `Run(config)` contract.
2.  **Rules**: Each analyzer contains specific rules (e.g., `CommentedCodeRule`, `CommentedFunctionsRule`).
3.  **Configuration**: Loaded from YAML, supporting per-analyzer settings.

### Adding New Analyzers
1.  Create `analyzers/newlang/newlang.go`.
2.  Implement the `Analyzer` interface.
3.  Register it in `main.go`.

### Adding New Rules
1.  Define a struct implementing the `Rule` interface.
2.  Add logic in `Apply(content string)`.
3.  Register the rule in the Analyzer's `New...Analyzer` function.
