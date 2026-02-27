data "s2_stream" "example" {
  basin = "my-example-basin"
  name  = "my-stream"
}

output "stream_created_at" {
  value = data.s2_stream.example.created_at
}
