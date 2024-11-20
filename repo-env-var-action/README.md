# GitHub Environment Variable Manager Action

A GitHub Action to create or update a single environment variable in your repository's environment. This action handles the complete lifecycle of an environment variable, including creation and updates, with proper error handling and status reporting.

## Features

- ✨ Create new environment variables
- 🔄 Update existing environment variables
- 🔒 Secure variable handling
- ✅ Automatic environment validation
- 📝 Detailed operation status reporting
- 🚀 Simple and straightforward usage

## Prerequisites

1. GitHub repository with environments configured
2. GitHub token with appropriate permissions (`repo` scope)
3. Environment already created in your repository

## Usage

### Basic Workflow

```yaml
name: Manage Environment Variable
on:
  workflow_dispatch:
    inputs:
      environment:
        description: 'Environment name'
        required: true
        default: 'production'
      variable_name:
        description: 'Variable name'
        required: true
      variable_value:
        description: 'Variable value'
        required: true

jobs:
  set-env-variable:
    runs-on: ubuntu-latest
    steps:
      - uses: your-username/env-variable-manager@v1
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          environment: ${{ github.event.inputs.environment }}
          variable-name: ${{ github.event.inputs.variable_name }}
          variable-value: ${{ github.event.inputs.variable_value }}
```

### Direct Usage in Workflow

```yaml
steps:
  - uses: your-username/env-variable-manager@v1
    with:
      github-token: ${{ secrets.GITHUB_TOKEN }}
      environment: 'production'
      variable-name: 'API_URL'
      variable-value: 'https://api.example.com'
```

### Integration with Deployment Workflow

```yaml
name: Deploy and Configure Environment
on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    environment: production
    steps:
      - name: Configure API URL
        uses: your-username/env-variable-manager@v1
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          environment: 'production'
          variable-name: 'API_URL'
          variable-value: ${{ secrets.API_URL }}

      - name: Deploy Application
        run: |
          # Your deployment steps here
          echo "Deploying with configured environment variables..."
```

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `github-token` | GitHub token with repository access | Yes | - |
| `environment` | Name of the target environment | Yes | - |
| `variable-name` | Name of the environment variable | Yes | - |
| `variable-value` | Value for the environment variable | Yes | - |
| `repository` | Repository in format owner/repo | No | Current repository |

## Outputs

| Output | Description |
|--------|-------------|
| `status` | Status of the operation (`created` or `updated`) |
| `variable-name` | Name of the processed variable |

## Error Handling

The action includes comprehensive error handling for common scenarios:

- ❌ Missing required inputs
- ❌ Environment not found
- ❌ Invalid variable names
- ❌ API request failures
- ❌ Permission issues

Each error will provide a clear message indicating the issue and how to resolve it.

## Permissions

The action requires a GitHub token with the following permissions:

- `repo` scope for private repositories
- Access to environment settings
- Permission to manage environment variables

The default `GITHUB_TOKEN` provided by GitHub Actions should have sufficient permissions in most cases.

## Examples

### Setting an API URL

```yaml
- uses: your-username/env-variable-manager@v1
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    environment: 'production'
    variable-name: 'API_URL'
    variable-value: 'https://api.example.com'
```

### Setting a Database Connection String

```yaml
- uses: your-username/env-variable-manager@v1
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    environment: 'staging'
    variable-name: 'DATABASE_URL'
    variable-value: ${{ secrets.DB_CONNECTION_STRING }}
```

### Setting a Feature Flag

```yaml
- uses: your-username/env-variable-manager@v1
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    environment: 'development'
    variable-name: 'FEATURE_FLAG_NEW_UI'
    variable-value: 'true'
```

## Common Issues and Solutions

### Environment Not Found

```
❌ Environment 'production' not found
```

**Solution:** Ensure the environment is created in your repository settings before running the action.

### Permission Denied

```
❌ Resource not accessible by integration
```

**Solution:** Verify that your GitHub token has the necessary permissions and that you have access to the environment.

### Variable Already Exists

This is handled automatically - the action will update the existing variable.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. Here's how you can contribute:

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

If you encounter any problems or have suggestions, please open an issue in the repository.

## Acknowledgments

- GitHub Actions Documentation
- GitHub REST API Documentation
- Contributors and users of this action

---

Made with ❤️ by [Your Name]