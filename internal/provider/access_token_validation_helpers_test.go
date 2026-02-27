package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/s2-streamstore/s2-sdk-go/s2"
)

func TestAccessTokenOperationsContainsKnownValues(t *testing.T) {
	t.Parallel()

	ops := accessTokenOperations()
	if len(ops) == 0 {
		t.Fatal("accessTokenOperations() returned empty list")
	}

	want := []string{
		s2.OperationListBasins,
		s2.OperationCreateStream,
		s2.OperationAppend,
		s2.OperationRead,
	}

	seen := map[string]bool{}
	for _, op := range ops {
		if seen[op] {
			t.Fatalf("accessTokenOperations() contains duplicate operation %q", op)
		}
		seen[op] = true
	}

	for _, expected := range want {
		if !seen[expected] {
			t.Fatalf("accessTokenOperations() missing expected operation %q", expected)
		}
	}
}

func TestOpGroupsHasAnyPermission(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		groups OpGroupsModel
		want   bool
	}{
		{
			name:   "all null",
			groups: OpGroupsModel{},
			want:   false,
		},
		{
			name: "all false",
			groups: OpGroupsModel{
				AccountRead:  types.BoolValue(false),
				AccountWrite: types.BoolValue(false),
				BasinRead:    types.BoolValue(false),
				BasinWrite:   types.BoolValue(false),
				StreamRead:   types.BoolValue(false),
				StreamWrite:  types.BoolValue(false),
			},
			want: false,
		},
		{
			name: "one true",
			groups: OpGroupsModel{
				StreamWrite: types.BoolValue(true),
			},
			want: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := opGroupsHasAnyPermission(tc.groups)
			if got != tc.want {
				t.Fatalf("opGroupsHasAnyPermission() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestOpGroupsHasUnknownPermission(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		groups OpGroupsModel
		want   bool
	}{
		{
			name:   "all null",
			groups: OpGroupsModel{},
			want:   false,
		},
		{
			name: "all known",
			groups: OpGroupsModel{
				AccountRead:  types.BoolValue(false),
				AccountWrite: types.BoolValue(false),
				BasinRead:    types.BoolValue(false),
				BasinWrite:   types.BoolValue(false),
				StreamRead:   types.BoolValue(false),
				StreamWrite:  types.BoolValue(true),
			},
			want: false,
		},
		{
			name: "has unknown",
			groups: OpGroupsModel{
				AccountRead: types.BoolUnknown(),
			},
			want: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := opGroupsHasUnknownPermission(tc.groups)
			if got != tc.want {
				t.Fatalf("opGroupsHasUnknownPermission() = %v, want %v", got, tc.want)
			}
		})
	}
}
