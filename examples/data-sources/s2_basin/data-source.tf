data "s2_basin" "example" {
  name = "my-example-basin"
}

output "basin_scope" {
  value = data.s2_basin.example.scope
}
