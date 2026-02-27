package provider

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func streamNameValidators() []validator.String {
	return []validator.String{
		stringvalidator.UTF8LengthBetween(1, 512),
	}
}
