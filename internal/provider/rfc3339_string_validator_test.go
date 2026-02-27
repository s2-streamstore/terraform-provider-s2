package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestRFC3339TimestampStringValidator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   types.String
		wantErr bool
	}{
		{
			name:    "null value",
			value:   types.StringNull(),
			wantErr: false,
		},
		{
			name:    "unknown value",
			value:   types.StringUnknown(),
			wantErr: false,
		},
		{
			name:    "valid timestamp",
			value:   types.StringValue("2030-01-01T00:00:00Z"),
			wantErr: false,
		},
		{
			name:    "invalid timestamp",
			value:   types.StringValue("2030-99-99T25:61:61Z"),
			wantErr: true,
		},
	}

	v := rfc3339TimestampString()
	ctx := context.Background()

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := validator.StringRequest{
				Path:        path.Root("expires_at"),
				ConfigValue: tc.value,
			}
			var resp validator.StringResponse

			v.ValidateString(ctx, req, &resp)

			if gotErr := resp.Diagnostics.HasError(); gotErr != tc.wantErr {
				t.Fatalf("ValidateString() diagnostics has error = %v, want %v", gotErr, tc.wantErr)
			}
		})
	}
}
