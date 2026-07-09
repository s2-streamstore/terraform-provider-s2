provider "s2" {
  access_token = var.s2_access_token
  # account_endpoint = "a.s2.dev"
}

variable "s2_access_token" {
  type      = string
  sensitive = true
}
