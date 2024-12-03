# Configure the Vault provider
provider "vault" {
  # Provider configuration should be passed via environment variables
  # VAULT_ADDR and VAULT_TOKEN should be set
}

# Enable the Terraform Cloud secrets backend
resource "vault_mount" "terraform" {
  path = "terraform"
  type = "terraform"
  description = "Terraform Cloud secrets engine"
}

# Configure the Terraform Cloud backend
resource "vault_terraform_cloud_secret_backend" "config" {
  backend      = vault_mount.terraform.path
  token        = var.terraform_cloud_token
  address      = "https://app.terraform.io"  # Default Terraform Cloud address
  description  = "Terraform Cloud secret backend configuration"
}

# Create a role for generating tokens
resource "vault_terraform_cloud_secret_role" "example" {
  backend      = vault_mount.terraform.path
  name         = "example-role"
  organization = var.terraform_org_name
  team_id      = var.terraform_team_id  # Optional: Specify if you want to scope to a specific team
}

# Variables
variable "terraform_cloud_token" {
  description = "Terraform Cloud API token with admin privileges"
  type        = string
  sensitive   = true
}

variable "terraform_org_name" {
  description = "Terraform Cloud organization name"
  type        = string
}

variable "terraform_team_id" {
  description = "Terraform Cloud team ID (optional)"
  type        = string
  default     = ""
}

# Outputs
output "backend_path" {
  value = vault_mount.terraform.path
  description = "The path where the Terraform Cloud secrets backend is mounted"
}

output "role_name" {
  value = vault_terraform_cloud_secret_role.example.name
  description = "The name of the created role"
}