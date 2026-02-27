resource "s2_basin" "example" {
  name = "my-example-basin"
}

resource "s2_stream" "example" {
  basin = s2_basin.example.name
  name  = "my-stream"

  storage_class = "express"

  retention_policy {
    age = 604800
  }

  timestamping {
    mode     = "client-prefer"
    uncapped = false
  }

  delete_on_empty {
    min_age_secs = 3600
  }
}
