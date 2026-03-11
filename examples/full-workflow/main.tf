terraform {
  required_version = ">= 1.5.0"

  required_providers {
    s2 = {
      source  = "s2-streamstore/s2"
      version = "0.0.0-dev"
    }
  }
}

provider "s2" {
  # access_token is read from S2_ACCESS_TOKEN by default.
  # Optional override:
  # account_endpoint = "aws.s2.dev"
}

variable "name_prefix" {
  description = "Lowercase prefix used for test resources (letters, numbers, hyphen)."
  type        = string
  default     = "tfmanual"
}

variable "name_suffix" {
  description = "Lowercase suffix to keep names unique (recommended 6-12 chars)."
  type        = string
  default     = "demo01"
}

variable "token_expires_at" {
  description = "RFC3339 expiration for issued access token."
  type        = string
  default     = "2030-01-01T00:00:00Z"
}

locals {
  basin_name = "${var.name_prefix}-${var.name_suffix}"
  stream_names = {
    audit = "audit-events"
    app   = "app-events"
    dlq   = "dead-letter"
  }

  token_id = "${var.name_prefix}-token-${var.name_suffix}"
}

resource "s2_basin" "main" {
  name  = local.basin_name
  scope = "aws:us-east-1"

  create_stream_on_append = false
  create_stream_on_read   = false

  default_stream_config {
    storage_class = "standard"

    retention_policy {
      age = 1209600 # 14 days
    }

    timestamping {
      mode     = "client-prefer"
      uncapped = false
    }

    delete_on_empty {
      min_age_secs = 0
    }
  }
}

resource "s2_stream" "audit" {
  basin = s2_basin.main.name
  name  = local.stream_names.audit

  storage_class = "express"

  retention_policy {
    age = 2419200 # 28 days (free-tier max)
  }

  timestamping {
    mode     = "arrival"
    uncapped = false
  }

  delete_on_empty {
    min_age_secs = 0
  }
}

resource "s2_stream" "app" {
  basin = s2_basin.main.name
  name  = local.stream_names.app

  storage_class = "standard"

  retention_policy {
    age = 604800 # 7 days
  }

  timestamping {
    mode     = "client-prefer"
    uncapped = false
  }

  delete_on_empty {
    min_age_secs = 3600
  }
}

resource "s2_stream" "dlq" {
  basin = s2_basin.main.name
  name  = local.stream_names.dlq

  storage_class = "standard"

  retention_policy {
    age = 1209600 # 14 days
  }

  timestamping {
    mode     = "arrival"
    uncapped = false
  }

  delete_on_empty {
    min_age_secs = 0
  }
}

resource "s2_access_token" "automation" {
  token_id            = local.token_id
  auto_prefix_streams = false
  expires_at          = var.token_expires_at

  scope = {
    basins = {
      exact = s2_basin.main.name
    }

    streams = {
      prefix = "app-"
    }

    access_tokens = {
      exact = local.token_id
    }

    op_groups = {
      account_read  = true
      account_write = false
      basin_read    = true
      basin_write   = true
      stream_read   = true
      stream_write  = true
    }

    ops = [
      "append",
      "read",
      "list-streams",
      "get-stream-config",
      "reconfigure-stream",
    ]
  }
}

data "s2_basin" "main" {
  name = s2_basin.main.name
}

data "s2_stream" "audit" {
  basin = s2_basin.main.name
  name  = s2_stream.audit.name
}

output "basin_name" {
  value = s2_basin.main.name
}

output "basin_state" {
  value = data.s2_basin.main.state
}

output "stream_audit_created_at" {
  value = s2_stream.audit.created_at
}

output "stream_audit_storage_class" {
  value = data.s2_stream.audit.storage_class
}

output "token_id" {
  value = s2_access_token.automation.token_id
}

output "issued_access_token" {
  value     = s2_access_token.automation.access_token
  sensitive = true
}
