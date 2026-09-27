data "s2_default_location" "current" {}

output "default_location" {
  value = data.s2_default_location.current.name
}

output "available_storage_classes" {
  value = data.s2_default_location.current.storage_classes
}

output "default_storage_class" {
  value = data.s2_default_location.current.default_storage_class
}
