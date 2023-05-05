# Terraform Provider Scaffolding (Terraform Plugin Framework)

_This repository is built on the [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework)._

Once you've written your provider, you'll want to [publish it on the Terraform Registry](https://www.terraform.io/docs/registry/providers/publishing.html) so that others can use it.

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.19

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

## Using the provider

```hcl
terraform {
  required_version = ">= 1.4.6"

  required_providers {
    essecurity = {
      source  = "ka-nabellinc/elasticsearch-security"
      version = "~> 1.0"
    }

    elasticsearch = {
      source  = "phillbaker/elasticsearch"
      version = "1.6.3"
    }
  }
}

locals {
  url = "https://localhost:9200"
  username = "elastic"
  password = "elastic"
}

provider "elasticsearch" {
  url      = local.url
  username = local.username
  password = local.password
}

resource "elasticsearch_index" "sample" {
  name     = "sample"

  force_destroy      = true
  number_of_shards   = 1
  number_of_replicas = 1
  mappings           = file("mapping.json")

  lifecycle {
    ignore_changes = [
      mappings
    ]
  }
}

provider "essecurity" {
  url      = local.url
  username = local.username
  password = local.password
}

resource "essecurity_api_key" "sample" {
  name = "sample"
  role_descriptors = [
    {
      name = "role-a"
      cluster = ["all"]
      indices = [
        {
          names = ["sample"]
          privileges = ["read", "write"]
        }
      ]
    }
  ]
}
```

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `go generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```shell
make testacc
```

## Local Development

Reference: https://developer.hashicorp.com/terraform/plugin/debugging

- Create `~/.terraformrc`
