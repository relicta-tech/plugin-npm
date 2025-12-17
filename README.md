# npm Plugin for Relicta

Official npm plugin for [Relicta](https://github.com/relicta-tech/relicta) - AI-powered release management.

## Features

- Publish packages to npm registry
- Automatic package.json version updates
- Support for custom registries
- Dist-tag management
- 2FA/OTP support
- Dry-run mode for testing
- Secure path and input validation

## Installation

```bash
relicta plugin install npm
relicta plugin enable npm
```

## Configuration

Add to your `release.config.yaml`:

```yaml
plugins:
  - name: npm
    enabled: true
    config:
      # npm registry URL (optional, defaults to public npm)
      registry: "https://registry.npmjs.org"

      # dist-tag for the package (default: "latest")
      tag: "latest"

      # Package access level: "public" or "restricted"
      access: "public"

      # Directory containing package.json (default: current directory)
      package_dir: "."

      # Update package.json version before publishing (default: true)
      update_version: true

      # Perform dry-run publish (default: false)
      dry_run: false
```

### Environment Variables

For 2FA-protected publishes, set the OTP via environment variable:

```bash
export NPM_OTP=123456
```

Or in CI/CD:

```yaml
env:
  NPM_OTP: ${{ secrets.NPM_OTP }}
```

## Hooks

This plugin responds to the following hooks:

| Hook | Behavior |
|------|----------|
| `pre-publish` | Updates package.json version (if enabled) |
| `post-publish` | Publishes package to npm registry |

## Security Features

- **Registry validation**: Only HTTPS registries allowed (except localhost for development)
- **Path traversal protection**: Package directory must be within working directory
- **Input sanitization**: All configuration values are validated
- **OTP redaction**: OTP values are not logged

## Requirements

- `npm` CLI must be installed and in PATH
- npm authentication configured (via `npm login` or `NPM_TOKEN`)

## Example Workflow

```yaml
# release.config.yaml
versioning:
  strategy: conventional

plugins:
  - name: npm
    enabled: true
    config:
      access: public
      update_version: true
```

```bash
# Run release
relicta plan
relicta bump
relicta notes
relicta approve
relicta publish
```

## Private Packages

If `package.json` has `"private": true`, the plugin will skip publishing.

## Monorepo Support

For monorepos, specify the package directory:

```yaml
plugins:
  - name: npm
    config:
      package_dir: "packages/my-library"
```

## License

MIT License - see [LICENSE](LICENSE) for details.
