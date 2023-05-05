resource "essecurity_api_key" "example" {
  name = "sample"
  role_descriptors = [
    {
      name    = "role-a"
      cluster = ["all"]
      indices = [
        {
          names      = ["sample"]
          privileges = ["read", "write"]
        }
      ]
    }
  ]
}
