# scripts/manage-variable.sh
#!/bin/bash

set -e

# Function to check if variable exists
check_variable() {
    local response
    response=$(curl -s -w "%{http_code}" -o /dev/null \
        -H "Authorization: token $INPUT_GITHUB_TOKEN" \
        -H "Accept: application/vnd.github.v3+json" \
        "https://api.github.com/repos/$INPUT_REPOSITORY/actions/variables/$INPUT_VARIABLE_NAME")

    if [ "$response" -eq 200 ]; then
        return 0
    else
        return 1
    fi
}

# Function to create new variable
create_variable() {
    local response
    response=$(curl -s -X POST \
        -H "Authorization: token $INPUT_GITHUB_TOKEN" \
        -H "Accept: application/vnd.github.v3+json" \
        "https://api.github.com/repos/$INPUT_REPOSITORY/actions/variables" \
        -d "{\"name\":\"$INPUT_VARIABLE_NAME\",\"value\":\"$INPUT_VARIABLE_VALUE\"}")

    if echo "$response" | grep -q "\"name\":\"$INPUT_VARIABLE_NAME\""; then
        echo "::set-output name=status::created"
        echo "✅ Variable '$INPUT_VARIABLE_NAME' created successfully"
    else
        echo "❌ Failed to create variable '$INPUT_VARIABLE_NAME'"
        echo "$response"
        exit 1
    fi
}

# Function to update existing variable
update_variable() {
    local response
    response=$(curl -s -X PATCH \
        -H "Authorization: token $INPUT_GITHUB_TOKEN" \
        -H "Accept: application/vnd.github.v3+json" \
        "https://api.github.com/repos/$INPUT_REPOSITORY/actions/variables/$INPUT_VARIABLE_NAME" \
        -d "{\"name\":\"$INPUT_VARIABLE_NAME\",\"value\":\"$INPUT_VARIABLE_VALUE\"}")

    if echo "$response" | grep -q "\"name\":\"$INPUT_VARIABLE_NAME\""; then
        echo "::set-output name=status::updated"
        echo "✅ Variable '$INPUT_VARIABLE_NAME' updated successfully"
    else
        echo "❌ Failed to update variable '$INPUT_VARIABLE_NAME'"
        echo "$response"
        exit 1
    fi
}

# Validate inputs
if [ -z "$INPUT_GITHUB_TOKEN" ]; then
    echo "❌ GitHub token is required"
    exit 1
fi

if [ -z "$INPUT_VARIABLE_NAME" ]; then
    echo "❌ Variable name is required"
    exit 1
fi

if [ -z "$INPUT_VARIABLE_VALUE" ]; then
    echo "❌ Variable value is required"
    exit 1
fi

# Set variable name as output
echo "::set-output name=variable-name::$INPUT_VARIABLE_NAME"

# Main logic
echo "🔍 Checking if variable '$INPUT_VARIABLE_NAME' exists..."

if check_variable; then
    echo "📝 Variable exists, updating..."
    update_variable
else
    echo "➕ Variable doesn't exist, creating..."
    create_variable
fi