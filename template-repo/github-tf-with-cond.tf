terraform {
  required_providers {
    github = {
      source  = "integrations/github"
      version = "~> 5.0"
    }
  }
}

provider "github" {
  token = var.github_token
}

variable "github_token" {
  description = "GitHub Personal Access Token"
  type        = string
  sensitive   = true
}

variable "repository_name" {
  description = "Name of the repository to check"
  type        = string
}

variable "organization" {
  description = "GitHub organization name (optional - leave empty for personal repos)"
  type        = string
  default     = ""
}

data "github_repository" "repo" {
  name = var.repository_name
  full_name = var.organization != "" ? "${var.organization}/${var.repository_name}" : null

  lifecycle {
    ignore_changes = all
  }
}

locals {
  repo_exists = can(data.github_repository.repo.name)
}

output "repository_exists" {
  value = local.repo_exists
}

resource "github_repository" "repo" {
  count = local.repo_exists ? 0 : 1

  name        = var.repository_name
  description = "Created by Terraform"
  visibility  = "private"

  # Delete protection settings
  allow_merge_commit     = true
  allow_squash_merge    = true
  allow_rebase_merge    = true
  delete_branch_on_merge = false

  archival_protection        = true
  allow_auto_merge          = false
  allow_update_branch       = false
  delete_head_on_merge      = false
  has_downloads            = true
  has_issues              = true
  has_projects            = true
  has_wiki                = true
  vulnerability_alerts    = true

  # Prevent terraform destroy from deleting the repository
  lifecycle {
    prevent_destroy = true
    ignore_changes = [
      description,
      homepage_url,
      visibility,
      has_downloads,
      has_issues,
      has_projects,
      has_wiki,
      allow_merge_commit,
      allow_squash_merge,
      allow_rebase_merge,
      allow_auto_merge,
      allow_update_branch,
      delete_branch_on_merge,
      vulnerability_alerts,
      topics
    ]
  }

  security_and_analysis {
    advanced_security {
      status = "enabled"
    }
    secret_scanning {
      status = "enabled"
    }
    secret_scanning_push_protection {
      status = "enabled"
    }
  }
}

resource "github_branch_protection" "main" {
  count = local.repo_exists ? 0 : 1

  repository_id = github_repository.repo[0].node_id
  pattern       = "main"

  required_pull_request_reviews {
    dismiss_stale_reviews           = true
    require_code_owner_reviews      = true
    required_approving_review_count = 1
  }

  allows_force_pushes = false
  allows_deletions = false

  required_status_checks {
    strict = true
    contexts = ["continuous-integration"]
  }

  enforce_admins = true
  require_signed_commits = true
  require_conversation_resolution = true

  # Prevent terraform destroy from removing branch protection
  lifecycle {
    prevent_destroy = true
    ignore_changes = [
      required_pull_request_reviews,
      required_status_checks,
      enforce_admins,
      require_signed_commits,
      require_conversation_resolution
    ]
  }
}

resource "github_repository_file" "codeowners" {
  count = local.repo_exists ? 0 : 1

  repository = github_repository.repo[0].name
  branch     = "main"
  file       = "CODEOWNERS"
  content    = "* @${var.organization != "" ? var.organization : "owner"}"
  commit_message = "Add CODEOWNERS file"

  depends_on = [github_repository.repo]

  # Prevent terraform destroy from deleting CODEOWNERS file
  lifecycle {
    prevent_destroy = true
    ignore_changes = [
      content,
      commit_message
    ]
  }
}