# GitHub Repository Variable Manager Action

This action allows you to create or update GitHub repository variables through your workflows.

## Features

- Create new repository variables
- Update existing repository variables
- Automatic detection of existing variables
- Detailed error handling and feedback
- Support for custom repositories

## Usage

```yaml
- uses: your-username/repo-variable-manager@v1
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    variable-name: 'MY_VARIABLE'
    variable-value: 'my-value'
    # Optional: specify a different repository (default is current repository)
    repository: 'owner/repo'
```

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `github-token` | GitHub token with repository access | Yes | - |
| `variable-name` | Name of the variable to create/update | Yes | - |
| `variable-value` | Value to set for the variable | Yes | - |
| `repository` | Repository in format owner/repo | No | Current repository |

## Outputs

| Output | Description |
|--------|-------------|
| `status` | Status of the operation (created/updated) |
| `variable-name` | Name of the variable that was created/updated |

## Example Workflow

```yaml
name: Manage Variables
on:
  workflow_dispatch:
    inputs:
      variable_name:
        description: 'Variable name'
        required: true
      variable_value:
        description: 'Variable value'
        required: true

jobs:
  manage-variable:
    runs-on: ubuntu-latest
    steps:
      - uses: your-username/repo-variable-manager@v1
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          variable-name: ${{ github.event.inputs.variable_name }}
          variable-value: ${{ github.event.inputs.variable_value }}
```

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.