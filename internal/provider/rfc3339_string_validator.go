package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = rfc3339TimestampValidator{}

type rfc3339TimestampValidator struct{}

func (v rfc3339TimestampValidator) Description(_ context.Context) string {
	return "value must be a valid RFC3339 timestamp"
}

func (v rfc3339TimestampValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v rfc3339TimestampValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	if _, err := time.Parse(time.RFC3339, req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid RFC3339 Timestamp",
			"Value must be a valid RFC3339 timestamp, for example 2030-01-01T00:00:00Z.",
		)
	}
}

func rfc3339TimestampString() validator.String {
	return rfc3339TimestampValidator{}
}
