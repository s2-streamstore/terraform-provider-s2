package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccStreamResource_basic(t *testing.T) {
	t.Parallel()

	basinName := testAccBasinName()
	streamName := testAccStreamName()
	resourceName := "s2_stream.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccStreamResourceConfigBasic(basinName, streamName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "basin", basinName),
					resource.TestCheckResourceAttr(resourceName, "name", streamName),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
				),
			},
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateId:                        fmt.Sprintf("%s/%s", basinName, streamName),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func TestAccStreamResource_fullConfigAndUpdate(t *testing.T) {
	t.Parallel()

	basinName := testAccBasinName()
	streamName := testAccStreamName()
	resourceName := "s2_stream.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccStreamResourceConfigFull(basinName, streamName, "express", 604800, "client-prefer", false, 0),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "storage_class", "express"),
					resource.TestCheckResourceAttr(resourceName, "retention_policy.age", "604800"),
					resource.TestCheckResourceAttr(resourceName, "timestamping.mode", "client-prefer"),
				),
			},
			{
				Config: testAccStreamResourceConfigFull(basinName, streamName, "standard", 3600, "arrival", false, 120),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "storage_class", "standard"),
					resource.TestCheckResourceAttr(resourceName, "retention_policy.age", "3600"),
					resource.TestCheckResourceAttr(resourceName, "timestamping.mode", "arrival"),
					resource.TestCheckResourceAttr(resourceName, "delete_on_empty.min_age_secs", "120"),
				),
			},
		},
	})
}

func TestAccStreamResource_replaceOnBasinOrNameChange(t *testing.T) {
	t.Parallel()

	firstBasin := testAccBasinName()
	secondBasin := testAccBasinName()
	firstStream := testAccStreamName()
	secondStream := testAccStreamName()
	resourceName := "s2_stream.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccStreamResourceConfigWithNamedBasin("one", firstBasin, firstStream),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "basin", firstBasin),
					resource.TestCheckResourceAttr(resourceName, "name", firstStream),
				),
			},
			{
				Config: testAccStreamResourceConfigWithNamedBasin("two", secondBasin, secondStream),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "basin", secondBasin),
					resource.TestCheckResourceAttr(resourceName, "name", secondStream),
				),
			},
		},
	})
}

func testAccStreamResourceConfigBasic(basinName, streamName string) string {
	return fmt.Sprintf(`
provider "s2" {}

resource "s2_basin" "test" {
  name = %q
}

resource "s2_stream" "test" {
  basin = s2_basin.test.name
  name  = %q
}
`, basinName, streamName)
}

func testAccStreamResourceConfigFull(basinName, streamName, storageClass string, retentionAge int, timestampMode string, uncapped bool, minAge int) string {
	return fmt.Sprintf(`
provider "s2" {}

resource "s2_basin" "test" {
  name = %q
}

resource "s2_stream" "test" {
  basin = s2_basin.test.name
  name  = %q

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
`, basinName, streamName, storageClass, retentionAge, timestampMode, uncapped, minAge)
}

func testAccStreamResourceConfigWithNamedBasin(basinResourceName, basinName, streamName string) string {
	return fmt.Sprintf(`
provider "s2" {}

resource "s2_basin" %q {
  name = %q
}

resource "s2_stream" "test" {
  basin = s2_basin.%s.name
  name  = %q
}
`, basinResourceName, basinName, basinResourceName, streamName)
}
