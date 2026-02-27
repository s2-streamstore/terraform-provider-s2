package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccStreamDataSource_basic(t *testing.T) {
	t.Parallel()

	basinName := testAccBasinName()
	streamName := testAccStreamName()
	resourceName := "data.s2_stream.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccStreamDataSourceConfigBasic(basinName, streamName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "basin", basinName),
					resource.TestCheckResourceAttr(resourceName, "name", streamName),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
				),
			},
		},
	})
}

func TestAccStreamDataSource_fullConfig(t *testing.T) {
	t.Parallel()

	basinName := testAccBasinName()
	streamName := testAccStreamName()
	resourceName := "data.s2_stream.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccStreamDataSourceConfigFull(basinName, streamName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "basin", basinName),
					resource.TestCheckResourceAttr(resourceName, "name", streamName),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckResourceAttr(resourceName, "storage_class", "standard"),
					resource.TestCheckResourceAttr(resourceName, "retention_policy.age", "3600"),
					resource.TestCheckResourceAttr(resourceName, "timestamping.mode", "arrival"),
					resource.TestCheckResourceAttr(resourceName, "delete_on_empty.min_age_secs", "120"),
				),
			},
		},
	})
}

func testAccStreamDataSourceConfigBasic(basinName, streamName string) string {
	return fmt.Sprintf(`
provider "s2" {}

resource "s2_basin" "test" {
  name = %q
}

resource "s2_stream" "test" {
  basin = s2_basin.test.name
  name  = %q
}

data "s2_stream" "test" {
  basin = s2_basin.test.name
  name  = s2_stream.test.name
}
`, basinName, streamName)
}

func testAccStreamDataSourceConfigFull(basinName, streamName string) string {
	return fmt.Sprintf(`
provider "s2" {}

resource "s2_basin" "test" {
  name = %q
}

resource "s2_stream" "test" {
  basin = s2_basin.test.name
  name  = %q

  storage_class = "standard"

  retention_policy {
    age = 3600
  }

  timestamping {
    mode     = "arrival"
    uncapped = false
  }

  delete_on_empty {
    min_age_secs = 120
  }
}

data "s2_stream" "test" {
  basin = s2_basin.test.name
  name  = s2_stream.test.name
}
`, basinName, streamName)
}
