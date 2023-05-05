package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccApiKeyResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccApiKeyResourceConfig("one"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("essecurity_api_key.test", "name", "one"),
				),
			},
			// Update and Read testing
			{
				Config: testAccApiKeyResourceConfig("two"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("essecurity_api_key.test", "name", "two"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccApiKeyResourceConfig(configurableAttribute string) string {
	return fmt.Sprintf(`
resource "essecurity_api_key" "test" {
  name = %[1]q
	role_descriptors = [
		{
			name: "role-a"
		}
	]
}
`, configurableAttribute)
}
