// Copyright (c) Lapse Technologies, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/teamlapse/terraform-provider-clickstack/internal/client"
)

func TestExpandAlertNumConsecutiveWindows(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		value types.Int64
		want  *int64
	}{
		"configured": {
			value: types.Int64Value(3),
			want:  int64Pointer(3),
		},
		"unset": {
			value: types.Int64Null(),
			want:  nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var diags diag.Diagnostics
			alert := expandAlert(context.Background(), alertResourceModel{
				Source:                types.StringValue("tile"),
				Threshold:             types.Float64Value(1),
				ThresholdType:         types.StringValue("above"),
				Interval:              types.StringValue("1m"),
				NumConsecutiveWindows: test.value,
				Channel: &channelModel{
					Type: types.StringValue("email"),
				},
			}, &diags)

			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			assertOptionalInt64(t, alert.NumConsecutiveWindows, test.want)
		})
	}
}

func TestFlattenAlertNumConsecutiveWindows(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		value *int64
		want  types.Int64
	}{
		"configured": {
			value: int64Pointer(3),
			want:  types.Int64Value(3),
		},
		"unset": {
			value: nil,
			want:  types.Int64Null(),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var diags diag.Diagnostics
			model := flattenAlert(context.Background(), &client.Alert{
				ID:                    "alert-1",
				Source:                "tile",
				Threshold:             1,
				ThresholdType:         "above",
				Interval:              "1m",
				NumConsecutiveWindows: test.value,
				Channel:               client.AlertChannel{Type: "email"},
			}, &diags)

			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if !model.NumConsecutiveWindows.Equal(test.want) {
				t.Fatalf("expected %v, got %v", test.want, model.NumConsecutiveWindows)
			}
		})
	}
}

func int64Pointer(value int64) *int64 {
	return &value
}

func assertOptionalInt64(t *testing.T, got, want *int64) {
	t.Helper()
	if got == nil || want == nil {
		if got != want {
			t.Fatalf("expected %v, got %v", want, got)
		}
		return
	}
	if *got != *want {
		t.Fatalf("expected %d, got %d", *want, *got)
	}
}
