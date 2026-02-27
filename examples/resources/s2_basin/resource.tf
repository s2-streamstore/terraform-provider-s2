resource "s2_basin" "example" {
  name  = "my-example-basin"
  scope = "aws:us-east-1"

  create_stream_on_append = false
  create_stream_on_read   = false

  default_stream_config {
    storage_class = "express"

    retention_policy {
      age = 604800
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
