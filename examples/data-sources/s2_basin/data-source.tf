data "s2_basin" "example" {
  name = "my-example-basin"
}

output "basin_location" {
  value = data.s2_basin.example.location
}
