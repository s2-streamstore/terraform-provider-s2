resource "s2_access_token" "example" {
  token_id            = "my-token"
  auto_prefix_streams = false

  scope = {
    basins = {
      prefix = ""
    }

    streams = {
      prefix = "logs/"
    }

    access_tokens = {
      exact = "my-token"
    }

    op_groups = {
      account_read  = true
      account_write = false
      basin_read    = true
      basin_write   = true
      stream_read   = true
      stream_write  = true
    }

    ops = ["append", "read"]
  }
}
