package provider

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var basinNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`)

func basinNameValidators() []validator.String {
	return []validator.String{
		stringvalidator.LengthBetween(8, 48),
		stringvalidator.RegexMatches(
			basinNameRegex,
			"must contain lowercase letters, numbers, and hyphens, and cannot begin or end with a hyphen",
		),
	}
}
