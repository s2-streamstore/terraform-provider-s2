package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBasinDataSource_basic(t *testing.T) {
	t.Parallel()

	basinName := testAccBasinName()
	resourceName := "data.s2_basin.test"
	scopeCheck := resource.TestCheckResourceAttrSet(resourceName, "scope")
	if testAccIsLite() {
		scopeCheck = resource.TestCheckResourceAttr(resourceName, "scope", "")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccBasinDataSourceConfigBasic(basinName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", basinName),
					scopeCheck,
					resource.TestCheckResourceAttr(resourceName, "state", "active"),
				),
			},
		},
	})
}

func TestAccBasinDataSource_fullConfig(t *testing.T) {
	t.Parallel()

	basinName := testAccBasinName()
	resourceName := "data.s2_basin.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccBasinDataSourceConfigFull(basinName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", basinName),
					resource.TestCheckResourceAttr(resourceName, "create_stream_on_append", "true"),
					resource.TestCheckResourceAttr(resourceName, "create_stream_on_read", "false"),
					resource.TestCheckResourceAttr(resourceName, "default_stream_config.storage_class", "standard"),
					resource.TestCheckResourceAttr(resourceName, "default_stream_config.retention_policy.age", "3600"),
					resource.TestCheckResourceAttr(resourceName, "default_stream_config.timestamping.mode", "arrival"),
					resource.TestCheckResourceAttr(resourceName, "default_stream_config.delete_on_empty.min_age_secs", "60"),
				),
			},
		},
	})
}

func testAccBasinDataSourceConfigBasic(name string) string {
	return fmt.Sprintf(`
provider "s2" {}

resource "s2_basin" "test" {
  name = %q
}

data "s2_basin" "test" {
  name = s2_basin.test.name
}
`, name)
}

func testAccBasinDataSourceConfigFull(name string) string {
	return fmt.Sprintf(`
provider "s2" {}

resource "s2_basin" "test" {
  name = %q

  create_stream_on_append = true
  create_stream_on_read   = false

  default_stream_config {
    storage_class = "standard"

    retention_policy {
      age = 3600
    }

    timestamping {
      mode     = "arrival"
      uncapped = false
    }

    delete_on_empty {
      min_age_secs = 60
    }
  }
}

data "s2_basin" "test" {
  name = s2_basin.test.name
}
`, name)
}
