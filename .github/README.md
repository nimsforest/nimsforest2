# GitHub Configuration

This directory contains GitHub-specific configurations for the NimsForest project.

## Contents

### Workflows (`workflows/`)

Automated CI/CD pipelines using GitHub Actions:

- **`ci.yml`** - Main continuous integration pipeline
  - Runs on every push and pull request
  - Tests on multiple Go versions (1.23.x, 1.24.x)
  - Performs linting and code quality checks
  - Generates code coverage reports
  - Builds for multiple platforms
  
- **`release.yml`** - Release automation
  - Triggers on version tags (v*)
  - Builds multi-platform binaries
  - Creates GitHub releases with changelogs
  - Publishes Docker images
  
- **`debian-package.yml`** - Debian package builder
  - Creates native .deb packages
  - Builds for amd64 and arm64
  - Includes systemd service integration

### Templates

- **`PULL_REQUEST_TEMPLATE.md`** - Standard PR template with checklist
- **`ISSUE_TEMPLATE/bug_report.md`** - Bug report template
- **`ISSUE_TEMPLATE/feature_request.md`** - Feature request template

## Usage

### Running CI Locally

Before pushing, test your changes locally:

```bash
# Run tests
make test

# Run linting
make lint

# Run all checks
make check
```

### Creating a Release

1. Ensure all tests pass
2. Create and push a version tag:
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```
3. Watch the Actions tab for workflow progress
4. Download artifacts from the releases page

### Viewing Workflow Results

- Go to: https://github.com/yourusername/nimsforest/actions
- Click on any workflow run to see details
- Download build artifacts from successful runs

## Configuration

### Required Secrets (Optional)

Add these in Settings → Secrets and variables → Actions:

- `CODECOV_TOKEN` - For code coverage reporting
- `DOCKER_USERNAME` - For Docker Hub publishing
- `DOCKER_PASSWORD` - Docker Hub token

### Branch Protection

Recommended settings for the main branch:

- Require status checks to pass (CI workflow)
- Require code review
- Require linear history
- Include administrators

## Support

For CI/CD issues, see:
- [CI_CD.md](../CI_CD.md) - Complete CI/CD documentation
- [GitHub Actions logs](https://github.com/yourusername/nimsforest/actions)
