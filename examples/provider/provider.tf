locals {
  url      = "https://localhost:9200"
  username = "elastic"
  password = "password"
}

provider "essecurity" {
  url      = local.url
  username = local.username
  password = local.password
}
