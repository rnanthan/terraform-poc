#!/bin/bash
# scripts/manage-env-variable.sh

set -e

# Function to get environment ID
get_environment_id() {
    local response
    response=$(curl -s \
        -H "Authorization: token $INPUT_GITHUB_TOKEN" \
        -H "Accept: application/vnd.github.v3+json" \
        "https://api.github.com/repos/$INPUT_REPOSITORY/environments/$INPUT_ENVIRONMENT")

    if echo "$response" | grep -q "\"id\":[0-9]"; then
        echo "$response" | grep -o "\"id\":[0-9]*" | cut -d":" -f2
        return 0
    else
        echo "❌ Environment '$INPUT_ENVIRONMENT' not found"
        echo "$response"
        exit 1
    fi
}

# Function to check if variable exists
check_variable() {
    local env_id=$1
    local response
    response=$(curl -s -w "%{http_code}" -o /dev/null \
        -H "Authorization: token $INPUT_GITHUB_TOKEN" \
        -H "Accept: application/vnd.github.v3+json" \
        "https://api.github.com/repositories/$(echo $INPUT_REPOSITORY | cut -d'/' -f2)/environments/$env_id/variables/$INPUT_VARIABLE_NAME")

    if [ "$response" -eq 200 ]; then
        return 0
    else
        return 1
    fi
}

# Function to create/update environment variable
manage_env_variable() {
    local env_id=$1
    local method="POST"
    local url="https://api.github.com/repositories/$(echo $INPUT_REPOSITORY | cut -d'/' -f2)/environments/$env_id/variables"

    if check_variable "$env_id"; then
        method="PATCH"
        url="$url/$INPUT_VARIABLE_NAME"
        echo "📝 Updating existing variable '$INPUT_VARIABLE_NAME' in environment '$INPUT_ENVIRONMENT'..."
        echo "::set-output name=status::updated"
    else
        echo "➕ Creating new variable '$INPUT_VARIABLE_NAME' in environment '$INPUT_ENVIRONMENT'..."
        echo "::set-output name=status::created"
    fi

    local response
    response=$(curl -s -X "$method" \
        -H "Authorization: token $INPUT_GITHUB_TOKEN" \
        -H "Accept: application/vnd.github.v3+json" \
        "$url" \
        -d "{\"name\":\"$INPUT_VARIABLE_NAME\",\"value\":\"$INPUT_VARIABLE_VALUE\"}")

    if echo "$response" | grep -q "\"name\":\"$INPUT_VARIABLE_NAME\""; then
        echo "✅ Variable '$INPUT_VARIABLE_NAME' processed successfully"
        echo "::set-output name=variable-name::$INPUT_VARIABLE_NAME"
    else
        echo "❌ Failed to process variable '$INPUT_VARIABLE_NAME'"
        echo "$response"
        exit 1
    fi
}

# Validate inputs
if [ -z "$INPUT_GITHUB_TOKEN" ]; then
    echo "❌ GitHub token is required"
    exit 1
fi

if [ -z "$INPUT_ENVIRONMENT" ]; then
    echo "❌ Environment name is required"
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

# Get environment ID
echo "🔍 Getting environment ID for '$INPUT_ENVIRONMENT'..."
env_id=$(get_environment_id)

# Process the variable
manage_env_variable "$env_id"