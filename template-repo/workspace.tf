    # Option 2: Using Terraform CLI variables
      - name: Initialize Terraform with Dynamic Workspace
        run: |
          BRANCH_NAME=${GITHUB_HEAD_REF:-${GITHUB_REF#refs/heads/}}
          WORKSPACE_NAME="${BRANCH_NAME//\//-}-workspace"
          terraform init \
            -backend-config="hostname=tfe.your-domain.com" \
            -backend-config="organization=your-organization-name" \
            -backend-config="workspaces.name=$WORKSPACE_NAME"
        env:
          TF_TOKEN_app_terraform_io: ${{ secrets.TF_API_TOKEN }}

      - name: Terraform Plan
        run: terraform plan
        env:
          TF_TOKEN_app_terraform_io: ${{ secrets.TF_API_TOKEN }}