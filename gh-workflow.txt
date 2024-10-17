name: Manual Update TFE SMTP Settings from Vault (with Test Mode)

on:
  workflow_dispatch:
    inputs:
      environment:
        description: 'Environment to update (e.g., production, staging, test)'
        required: true
        default: 'test'
      confirm_update:
        description: 'Type YES to confirm the update'
        required: true
      test_mode:
        description: 'Run in test mode (no actual updates)'
        type: boolean
        default: true

jobs:
  update-smtp-settings:
    runs-on: ubuntu-latest
    if: github.event.inputs.confirm_update == 'YES'
    steps:
    - name: Checkout code
      uses: actions/checkout@v2

    - name: Install dependencies
      run: |
        sudo apt-get update
        sudo apt-get install -y jq curl

    - name: Vault Authentication
      uses: hashicorp/vault-action@v2
      with:
        url: ${{ secrets.VAULT_ADDR }}
        method: approle
        roleId: ${{ secrets.VAULT_ROLE_ID }}
        secretId: ${{ secrets.VAULT_SECRET_ID }}
        secrets: |
          secret/data/tfe-smtp/${{ github.event.inputs.environment }} TFE_ADDRESS ;
          secret/data/tfe-smtp/${{ github.event.inputs.environment }} TFE_TOKEN ;
          secret/data/tfe-smtp/${{ github.event.inputs.environment }} SMTP_HOST ;
          secret/data/tfe-smtp/${{ github.event.inputs.environment }} SMTP_PORT ;
          secret/data/tfe-smtp/${{ github.event.inputs.environment }} SMTP_USER ;
          secret/data/tfe-smtp/${{ github.event.inputs.environment }} SMTP_PASS ;
          secret/data/tfe-smtp/${{ github.event.inputs.environment }} SMTP_AUTH ;
          secret/data/tfe-smtp/${{ github.event.inputs.environment }} SENDER_EMAIL ;
          secret/data/tfe-smtp/${{ github.event.inputs.environment }} SENDER_NAME ;
          secret/data/tfe-smtp/${{ github.event.inputs.environment }} ENABLE_STARTTLS

    - name: Create SMTP update script
      run: |
        cat << EOF > update_tfe_smtp.sh
        #!/bin/bash

        TEST_MODE=${{ github.event.inputs.test_mode }}

        get_current_settings() {
            if [ "\$TEST_MODE" = "true" ]; then
                echo '{"data": {"attributes": {"host": "old-smtp.example.com", "port": 25, "username": "old-user", "auth": "login", "from": "old@example.com", "from_name": "Old Sender", "starttls": false}}}'
            else
                curl -s -X GET \
                -H "Authorization: Bearer \$TFE_TOKEN" \
                -H "Content-Type: application/vnd.api+json" \
                "\${TFE_ADDRESS}/api/v2/admin/smtp-settings"
            fi
        }

        compare_settings() {
            local current_settings="\$1"

            local current_host=\$(echo "\$current_settings" | jq -r '.data.attributes.host')
            local current_port=\$(echo "\$current_settings" | jq -r '.data.attributes.port')
            local current_user=\$(echo "\$current_settings" | jq -r '.data.attributes.username')
            local current_auth=\$(echo "\$current_settings" | jq -r '.data.attributes.auth')
            local current_from=\$(echo "\$current_settings" | jq -r '.data.attributes.from')
            local current_from_name=\$(echo "\$current_settings" | jq -r '.data.attributes.from_name')
            local current_starttls=\$(echo "\$current_settings" | jq -r '.data.attributes.starttls')

            if [ "\$current_host" != "\$SMTP_HOST" ] || \
               [ "\$current_port" != "\$SMTP_PORT" ] || \
               [ "\$current_user" != "\$SMTP_USER" ] || \
               [ "\$current_auth" != "\$SMTP_AUTH" ] || \
               [ "\$current_from" != "\$SENDER_EMAIL" ] || \
               [ "\$current_from_name" != "\$SENDER_NAME" ] || \
               [ "\$current_starttls" != "\$ENABLE_STARTTLS" ]; then
                return 0  # Changes needed
            else
                return 1  # No changes needed
            fi
        }

        current_settings=\$(get_current_settings)

        if compare_settings "\$current_settings"; then
            echo "Changes detected. Updating SMTP settings..."

            JSON_PAYLOAD=\$(cat <<EEOF
            {
              "data": {
                "type": "smtp-settings",
                "attributes": {
                  "host": "\$SMTP_HOST",
                  "port": \$SMTP_PORT,
                  "username": "\$SMTP_USER",
                  "password": "\$SMTP_PASS",
                  "auth": "\$SMTP_AUTH",
                  "from": "\$SENDER_EMAIL",
                  "from_name": "\$SENDER_NAME",
                  "starttls": \$ENABLE_STARTTLS
                }
              }
            }
        EEOF
            )

            if [ "\$TEST_MODE" = "true" ]; then
                echo "Test mode: Would send the following payload:"
                echo "\$JSON_PAYLOAD"
            else
                response=\$(curl -s -X PATCH \
                  -H "Authorization: Bearer \$TFE_TOKEN" \
                  -H "Content-Type: application/vnd.api+json" \
                  -d "\$JSON_PAYLOAD" \
                  "\${TFE_ADDRESS}/api/v2/admin/smtp-settings")

                if [ \$? -eq 0 ]; then
                  echo "SMTP settings updated successfully"
                else
                  echo "Failed to update SMTP settings"
                  echo "Response: \$response"
                  exit 1
                fi
            fi
        else
            echo "No changes needed. Current settings are up to date."
        fi
        EOF

        chmod +x update_tfe_smtp.sh

    - name: Run SMTP update script
      run: ./update_tfe_smtp.sh