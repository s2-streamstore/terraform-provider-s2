package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccBasinResource_basic(t *testing.T) {
	t.Parallel()

	basinName := testAccBasinName()
	resourceName := "s2_basin.test"
	scopeCheck := resource.TestCheckResourceAttr(resourceName, "scope", defaultBasinScope)
	if testAccIsLite() {
		scopeCheck = resource.TestCheckResourceAttr(resourceName, "scope", "")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccBasinResourceConfigBasic(basinName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", basinName),
					scopeCheck,
					resource.TestCheckResourceAttr(resourceName, "state", "active"),
				),
			},
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateId:                        basinName,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func TestAccBasinResource_fullConfigAndUpdate(t *testing.T) {
	t.Parallel()

	basinName := testAccBasinName()
	resourceName := "s2_basin.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccBasinResourceConfigFull(basinName, false, false, "express", 604800, "client-prefer", false, 0),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "create_stream_on_append", "false"),
					resource.TestCheckResourceAttr(resourceName, "create_stream_on_read", "false"),
					resource.TestCheckResourceAttr(resourceName, "default_stream_config.storage_class", "express"),
					resource.TestCheckResourceAttr(resourceName, "default_stream_config.retention_policy.age", "604800"),
					resource.TestCheckResourceAttr(resourceName, "default_stream_config.timestamping.mode", "client-prefer"),
				),
			},
			{
				Config: testAccBasinResourceConfigFull(basinName, true, false, "standard", 3600, "arrival", false, 60),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "create_stream_on_append", "true"),
					resource.TestCheckResourceAttr(resourceName, "create_stream_on_read", "false"),
					resource.TestCheckResourceAttr(resourceName, "default_stream_config.storage_class", "standard"),
					resource.TestCheckResourceAttr(resourceName, "default_stream_config.retention_policy.age", "3600"),
					resource.TestCheckResourceAttr(resourceName, "default_stream_config.timestamping.mode", "arrival"),
					resource.TestCheckResourceAttr(resourceName, "default_stream_config.delete_on_empty.min_age_secs", "60"),
				),
			},
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateId:                        basinName,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func TestAccBasinResource_nameChangeReplaces(t *testing.T) {
	t.Parallel()

	firstName := testAccBasinName()
	secondName := testAccBasinName()
	resourceName := "s2_basin.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccBasinResourceConfigBasic(firstName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", firstName),
				),
			},
			{
				Config: testAccBasinResourceConfigBasic(secondName),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", secondName),
				),
			},
		},
	})
}

func testAccBasinResourceConfigBasic(name string) string {
	return fmt.Sprintf(`
provider "s2" {}

resource "s2_basin" "test" {
  name = %q
}
`, name)
}

func testAccBasinResourceConfigFull(name string, createOnAppend bool, createOnRead bool, storageClass string, retentionAge int, timestampMode string, uncapped bool, minAge int) string {
	return fmt.Sprintf(`
provider "s2" {}

resource "s2_basin" "test" {
  name = %q

  create_stream_on_append = %t
  create_stream_on_read   = %t

  default_stream_config {
    storage_class = %q

    retention_policy {
      age = %d
    }

    timestamping {
      mode     = %q
      uncapped = %t
    }

    delete_on_empty {
      min_age_secs = %d
    }
  }
}
`, name, createOnAppend, createOnRead, storageClass, retentionAge, timestampMode, uncapped, minAge)
}
