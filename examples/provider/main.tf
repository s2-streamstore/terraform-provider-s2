provider "s2" {
  access_token = var.s2_access_token
  # base_url = "https://aws.s2.dev/v1"
}

variable "s2_access_token" {
  type      = string
  sensitive = true
}
