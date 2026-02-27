package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccAccessTokenResource_basic(t *testing.T) {
	t.Parallel()
	testAccSkipIfLiteNoAccessTokenSupport(t)

	tokenID := testAccTokenID()
	resourceName := "s2_access_token.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccAccessTokenResourceConfigBasic(tokenID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "token_id", tokenID),
					resource.TestCheckResourceAttr(resourceName, "id", tokenID),
					resource.TestCheckResourceAttrSet(resourceName, "access_token"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     tokenID,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"access_token",
					"auto_prefix_streams",
					"expires_at",
					"scope.access_tokens",
					"scope.op_groups",
				},
			},
		},
	})
}

func TestAccAccessTokenResource_fullScope(t *testing.T) {
	t.Parallel()
	testAccSkipIfLiteNoAccessTokenSupport(t)

	tokenID := testAccTokenID()
	resourceName := "s2_access_token.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccAccessTokenResourceConfigFullScope(tokenID, "2030-01-01T00:00:00Z", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "token_id", tokenID),
					resource.TestCheckResourceAttr(resourceName, "id", tokenID),
					resource.TestCheckResourceAttr(resourceName, "auto_prefix_streams", "false"),
					resource.TestCheckResourceAttr(resourceName, "expires_at", "2030-01-01T00:00:00Z"),
					resource.TestCheckResourceAttr(resourceName, "scope.basins.prefix", ""),
					resource.TestCheckResourceAttr(resourceName, "scope.streams.prefix", "logs/"),
					resource.TestCheckResourceAttr(resourceName, "scope.access_tokens.exact", tokenID),
					resource.TestCheckResourceAttr(resourceName, "scope.op_groups.account_read", "true"),
					resource.TestCheckResourceAttr(resourceName, "scope.op_groups.stream_write", "true"),
					resource.TestCheckResourceAttr(resourceName, "scope.ops.#", "2"),
				),
			},
		},
	})
}

func TestAccAccessTokenResource_changeRequiresReplace(t *testing.T) {
	t.Parallel()
	testAccSkipIfLiteNoAccessTokenSupport(t)

	tokenID := testAccTokenID()
	resourceName := "s2_access_token.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccAccessTokenResourceConfigFullScope(tokenID, "2030-01-01T00:00:00Z", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "expires_at", "2030-01-01T00:00:00Z"),
				),
			},
			{
				Config: testAccAccessTokenResourceConfigFullScope(tokenID, "2031-01-01T00:00:00Z", true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "expires_at", "2031-01-01T00:00:00Z"),
					resource.TestCheckResourceAttr(resourceName, "auto_prefix_streams", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "access_token"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     tokenID,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"access_token",
					"scope.access_tokens",
					"scope.op_groups",
				},
			},
		},
	})
}

func testAccAccessTokenResourceConfigBasic(tokenID string) string {
	return fmt.Sprintf(`
provider "s2" {}

resource "s2_access_token" "test" {
  token_id = %q

  scope = {
    basins = {
      prefix = ""
    }
    streams = {
      prefix = ""
    }
    ops = ["append", "read"]
  }
}
`, tokenID)
}

func testAccAccessTokenResourceConfigFullScope(tokenID, expiresAt string, autoPrefix bool) string {
	return fmt.Sprintf(`
provider "s2" {}

resource "s2_access_token" "test" {
  token_id            = %q
  auto_prefix_streams = %t
  expires_at          = %q

  scope = {
    basins = {
      prefix = ""
    }

    streams = {
      prefix = "logs/"
    }

    access_tokens = {
      exact = %q
    }

    op_groups = {
      account_read  = true
      account_write = false
      basin_read    = true
      basin_write   = true
      stream_read   = true
      stream_write  = true
    }

    ops = ["append", "read"]
  }
}
`, tokenID, autoPrefix, expiresAt, tokenID)
}

func testAccSkipIfLiteNoAccessTokenSupport(t *testing.T) {
	t.Helper()

	if testAccIsLite() {
		t.Skip("s2-lite does not implement /access-tokens")
	}
}
