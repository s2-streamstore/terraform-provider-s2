package provider

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

func testAccPreCheck(t *testing.T) {
	t.Helper()

	if testAccIsLite() && strings.TrimSpace(os.Getenv("S2_BASE_URL")) == "" {
		t.Skip("S2_BASE_URL must be set when S2_ACC_TARGET=lite")
	}

	if strings.TrimSpace(os.Getenv("S2_ACCESS_TOKEN")) == "" && testAccIsLite() {
		if err := os.Setenv("S2_ACCESS_TOKEN", "test"); err != nil {
			t.Fatalf("failed to set default S2_ACCESS_TOKEN for lite tests: %v", err)
		}
	}

	if strings.TrimSpace(os.Getenv("S2_ACCESS_TOKEN")) == "" {
		t.Skip("S2_ACCESS_TOKEN must be set for acceptance tests")
	}
}

func testAccIsLite() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("S2_ACC_TARGET")), "lite")
}

func testAccProtoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"s2": providerserver.NewProtocol6WithError(New("test")()),
	}
}

func testAccBasinName() string {
	return fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(12, acctest.CharSetAlphaNum))
}

func testAccStreamName() string {
	return fmt.Sprintf("tf-stream-%s", acctest.RandStringFromCharSet(12, acctest.CharSetAlphaNum))
}

func testAccTokenID() string {
	return fmt.Sprintf("tf-token-%s", acctest.RandStringFromCharSet(12, acctest.CharSetAlphaNum))
}
