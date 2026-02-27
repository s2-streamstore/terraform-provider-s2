package provider

import (
	"testing"

	"github.com/s2-streamstore/s2-sdk-go/s2"
)

func TestIsStreamDeletionPending(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "code and status match",
			err: &s2.S2Error{
				Code:    "stream_deletion_pending",
				Message: "Stream is being deleted",
				Status:  409,
			},
			want: true,
		},
		{
			name: "message fallback matches",
			err: &s2.S2Error{
				Code:    "",
				Message: "Stream is being deleted",
				Status:  409,
			},
			want: true,
		},
		{
			name: "wrong status",
			err: &s2.S2Error{
				Code:    "stream_deletion_pending",
				Message: "Stream is being deleted",
				Status:  404,
			},
			want: false,
		},
		{
			name: "different conflict code",
			err: &s2.S2Error{
				Code:    "conflict",
				Message: "Conflict",
				Status:  409,
			},
			want: false,
		},
		{
			name: "non api error",
			err:  nil,
			want: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isStreamDeletionPending(tc.err)
			if got != tc.want {
				t.Fatalf("isStreamDeletionPending() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsBasinDeletionPending(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "code and status match",
			err: &s2.S2Error{
				Code:    "basin_deletion_pending",
				Message: "Basin deleted while trying to reconfigure",
				Status:  409,
			},
			want: true,
		},
		{
			name: "message fallback matches",
			err: &s2.S2Error{
				Code:    "",
				Message: "Basin is being deleted",
				Status:  409,
			},
			want: true,
		},
		{
			name: "wrong status",
			err: &s2.S2Error{
				Code:    "basin_deletion_pending",
				Message: "Basin is being deleted",
				Status:  404,
			},
			want: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isBasinDeletionPending(tc.err)
			if got != tc.want {
				t.Fatalf("isBasinDeletionPending() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsTransactionConflict(t *testing.T) {
	t.Parallel()

	if !isTransactionConflict(&s2.S2Error{Code: "transaction_conflict", Status: 409}) {
		t.Fatal("expected transaction_conflict with HTTP 409 to match")
	}

	if isTransactionConflict(&s2.S2Error{Code: "resource_already_exists", Status: 409}) {
		t.Fatal("did not expect resource_already_exists to match transaction conflict")
	}
}

func TestIsUnavailable(t *testing.T) {
	t.Parallel()

	if !isUnavailable(&s2.S2Error{Code: "unavailable", Status: 503}) {
		t.Fatal("expected unavailable code with HTTP 503 to match")
	}
	if !isUnavailable(&s2.S2Error{Code: "", Status: 503}) {
		t.Fatal("expected HTTP 503 to match")
	}
	if isUnavailable(&s2.S2Error{Code: "permission_denied", Status: 403}) {
		t.Fatal("did not expect permission_denied to match unavailable")
	}
}

func TestIsNotFoundByCode(t *testing.T) {
	t.Parallel()

	if !isNotFound(&s2.S2Error{Code: "stream_not_found", Status: 0}) {
		t.Fatal("expected stream_not_found code to be treated as not found")
	}
	if !isNotFound(&s2.S2Error{Code: "access_token_not_found", Status: 0}) {
		t.Fatal("expected access_token_not_found code to be treated as not found")
	}
}
