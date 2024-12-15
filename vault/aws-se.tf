# Configure the Vault provider
provider "vault" {
  address = "http://vault.example.com:8200"
}

# Enable the AWS secrets engine
resource "vault_mount" "aws" {
  path        = "aws"
  type        = "aws"
  description = "AWS secrets engine for dynamic credentials"
}

# Configure the AWS secrets engine with IAM role
resource "vault_aws_secret_backend" "aws" {
  path = vault_mount.aws.path

  # Use IAM role in the primary account
  iam_endpoint    = "https://iam.amazonaws.com"
  sts_endpoint    = "https://sts.amazonaws.com"
  region          = var.aws_region

  # Role that Vault uses in the primary account
  aws_role_arn    = var.vault_aws_role_arn

  default_lease_ttl_seconds = 3600        # 1 hour
  max_lease_ttl_seconds    = 86400        # 24 hours
}

# Create a backend role that assumes a role in another account
resource "vault_aws_secret_backend_role" "cross_account_role" {
  backend = vault_mount.aws.path
  name    = "cross-account-role"

  # Configure to assume IAM role instead of creating users
  credential_type = "assumed_role"
  role_arns      = [var.target_role_arn]

  # Default duration for the assumed role session
  default_sts_ttl = 3600
  max_sts_ttl     = 14400
}

# Variables
variable "vault_aws_role_arn" {
  description = "ARN of the IAM role that Vault should assume in the primary account"
  type        = string
}

variable "target_role_arn" {
  description = "ARN of the role to assume in the target AWS account"
  type        = string
}

variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-west-2"
}

# Optional: Output the role name for reference
output "role_name" {
  value = vault_aws_secret_backend_role.cross_account_role.name
}

# Trust policy for the role in the target account (create separately in AWS):
/*
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "AWS": "arn:aws:iam::PRIMARY_ACCOUNT_ID:role/VaultRole"
            },
            "Action": "sts:AssumeRole",
            "Condition": {
                "StringEquals": {
                    "sts:ExternalId": "vault-token-XXXX"  # Optional but recommended
                }
            }
        }
    ]
}
*/