package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccLocationsDataSource_basic(t *testing.T) {
	t.Parallel()

	if testAccIsLite() {
		t.Skip("S2 Lite location endpoints are not implemented")
	}

	resourceName := "data.s2_locations.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
provider "s2" {}

data "s2_locations" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith(resourceName, "locations.#", func(value string) error {
						if value == "0" {
							return fmt.Errorf("expected at least one location")
						}
						return nil
					}),
					resource.TestCheckResourceAttrSet(resourceName, "locations.0.name"),
					resource.TestCheckResourceAttrSet(resourceName, "locations.0.is_private"),
				),
			},
		},
	})
}

func TestAccDefaultLocationDataSource_basic(t *testing.T) {
	t.Parallel()

	if testAccIsLite() {
		t.Skip("S2 Lite location endpoints are not implemented")
	}

	resourceName := "data.s2_default_location.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
provider "s2" {}

data "s2_default_location" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "name"),
					resource.TestCheckResourceAttrSet(resourceName, "is_private"),
				),
			},
		},
	})
}
