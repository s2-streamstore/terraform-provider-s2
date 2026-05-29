data "s2_locations" "available" {}

output "available_locations" {
  value = data.s2_locations.available.locations
}
