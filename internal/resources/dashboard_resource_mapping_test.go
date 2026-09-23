// Copyright (c) Lapse Technologies, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import "testing"

func TestJSONSubsetServerDefaults(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		stored   string
		declared string
		want     bool
	}{
		"default fillNulls on a raw SQL line": {
			stored:   `{"configType":"sql","displayType":"line","sqlTemplate":"SELECT 1","fillNulls":true}`,
			declared: `{"configType":"sql","displayType":"line","sqlTemplate":"SELECT 1"}`,
			want:     true,
		},
		"default fillNulls and asRatio on a builder stacked bar": {
			stored:   `{"displayType":"stacked_bar","sourceId":"s","fillNulls":true,"asRatio":false}`,
			declared: `{"displayType":"stacked_bar","sourceId":"s"}`,
			want:     true,
		},
		"default asRatio on a builder table": {
			stored:   `{"displayType":"table","sourceId":"s","asRatio":false}`,
			declared: `{"displayType":"table","sourceId":"s"}`,
			want:     true,
		},
		"default select and where language on a search": {
			stored:   `{"displayType":"search","sourceId":"s","where":"level:error","select":"","whereLanguage":"lucene"}`,
			declared: `{"displayType":"search","sourceId":"s","where":"level:error"}`,
			want:     true,
		},
		"declared where language the server does not keep": {
			stored:   `{"displayType":"search","sourceId":"s","where":"","select":"","whereLanguage":"lucene"}`,
			declared: `{"displayType":"search","sourceId":"s","where":"","whereLanguage":"sql"}`,
			want:     false,
		},
		"resolved source name": {
			stored:   `{"displayType":"number","sourceId":"s"}`,
			declared: `{"displayType":"number","sourceId":"s","source":"Metrics"}`,
			want:     true,
		},
		"non-default fillNulls the declaration leaves out": {
			stored:   `{"displayType":"line","sourceId":"s","fillNulls":false,"asRatio":false}`,
			declared: `{"displayType":"line","sourceId":"s"}`,
			want:     false,
		},
		"declared fillNulls the server does not keep": {
			stored:   `{"displayType":"line","sourceId":"s","fillNulls":true,"asRatio":false}`,
			declared: `{"displayType":"line","sourceId":"s","fillNulls":false}`,
			want:     false,
		},
		"fillNulls on a table is not a default": {
			stored:   `{"displayType":"table","sourceId":"s","fillNulls":true}`,
			declared: `{"displayType":"table","sourceId":"s"}`,
			want:     false,
		},
		"asRatio on raw SQL is not a default": {
			stored:   `{"configType":"sql","displayType":"line","sqlTemplate":"SELECT 1","fillNulls":true,"asRatio":false}`,
			declared: `{"configType":"sql","displayType":"line","sqlTemplate":"SELECT 1"}`,
			want:     false,
		},
		"changed display type": {
			stored:   `{"displayType":"line","sourceId":"s","fillNulls":true,"asRatio":false}`,
			declared: `{"displayType":"table","sourceId":"s"}`,
			want:     false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := jsonSubset(tt.stored, tt.declared); got != tt.want {
				t.Fatalf("jsonSubset() = %v, want %v", got, tt.want)
			}
		})
	}
}
