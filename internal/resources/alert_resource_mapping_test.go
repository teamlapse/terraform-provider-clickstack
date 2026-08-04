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

func TestFlattenAlertIgnoresIrrelevantSourceIDs(t *testing.T) {
	t.Parallel()

	dashboardID := "dashboard-1"
	tileID := "tile-1"
	savedSearchID := "saved-search-1"

	tests := map[string]struct {
		source            string
		wantDashboardID   types.String
		wantTileID        types.String
		wantSavedSearchID types.String
	}{
		"saved search ignores retained tile IDs": {
			source:            "saved_search",
			wantDashboardID:   types.StringNull(),
			wantTileID:        types.StringNull(),
			wantSavedSearchID: types.StringValue(savedSearchID),
		},
		"tile ignores retained saved-search ID": {
			source:            "tile",
			wantDashboardID:   types.StringValue(dashboardID),
			wantTileID:        types.StringValue(tileID),
			wantSavedSearchID: types.StringNull(),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var diags diag.Diagnostics
			model := flattenAlert(context.Background(), &client.Alert{
				ID:            "alert-1",
				Source:        test.source,
				Threshold:     1,
				ThresholdType: "above",
				Interval:      "1m",
				Channel:       client.AlertChannel{Type: "email"},
				DashboardID:   &dashboardID,
				TileID:        &tileID,
				SavedSearchID: &savedSearchID,
			}, &diags)

			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if !model.DashboardID.Equal(test.wantDashboardID) {
				t.Fatalf("expected dashboard_id %v, got %v", test.wantDashboardID, model.DashboardID)
			}
			if !model.TileID.Equal(test.wantTileID) {
				t.Fatalf("expected tile_id %v, got %v", test.wantTileID, model.TileID)
			}
			if !model.SavedSearchID.Equal(test.wantSavedSearchID) {
				t.Fatalf("expected saved_search_id %v, got %v", test.wantSavedSearchID, model.SavedSearchID)
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
