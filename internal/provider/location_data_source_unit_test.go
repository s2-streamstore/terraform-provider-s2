package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/s2-streamstore/s2-sdk-go/s2"
)

func TestLocationStorageClasses(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name         string
		info         s2.LocationInfo
		classes      types.List
		defaultClass types.String
	}{
		{
			name:         "older server",
			classes:      types.ListNull(types.StringType),
			defaultClass: types.StringNull(),
		},
		{
			name:         "empty offerings",
			info:         s2.LocationInfo{StorageClasses: []string{}},
			classes:      types.ListValueMust(types.StringType, []attr.Value{}),
			defaultClass: types.StringNull(),
		},
		{
			name:         "future class",
			info:         s2.LocationInfo{StorageClasses: []string{"future"}, DefaultStorageClass: s2.Ptr("future")},
			classes:      types.ListValueMust(types.StringType, []attr.Value{types.StringValue("future")}),
			defaultClass: types.StringValue("future"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := flattenLocationInfo(tc.info)
			if !got.StorageClasses.Equal(tc.classes) || !got.DefaultStorageClass.Equal(tc.defaultClass) {
				t.Fatalf("got classes %s, default %s; want %s, %s", got.StorageClasses, got.DefaultStorageClass, tc.classes, tc.defaultClass)
			}
		})
	}
}
