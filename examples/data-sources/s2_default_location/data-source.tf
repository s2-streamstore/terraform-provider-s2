data "s2_default_location" "current" {}

output "default_location" {
  value = data.s2_default_location.current.name
}
