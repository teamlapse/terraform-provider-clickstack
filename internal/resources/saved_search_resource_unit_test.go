// Copyright (c) Lapse Technologies, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/teamlapse/terraform-provider-clickstack/internal/testmock"
)

func TestUnitSavedSearchResource_basic(t *testing.T) {
	mock := testmock.NewServer(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: mock.ProviderConfig() + `
resource "clickstack_saved_search" "test" {
  name           = "API Errors"
  source_id      = "src-log"
  select         = "Timestamp, ServiceName, Body"
  where          = "SeverityText:ERROR"
  where_language = "lucene"
  order_by       = "Timestamp DESC"
  tags           = ["api", "errors"]

  filters {
    condition = "ServiceName IN ('checkout', 'payments')"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("clickstack_saved_search.test", "id"),
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "name", "API Errors"),
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "source_id", "src-log"),
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "select", "Timestamp, ServiceName, Body"),
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "where", "SeverityText:ERROR"),
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "where_language", "lucene"),
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "order_by", "Timestamp DESC"),
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "tags.#", "2"),
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "filters.#", "1"),
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "filters.0.type", "sql"),
				),
			},
			{
				Config: mock.ProviderConfig() + `
resource "clickstack_saved_search" "test" {
  name           = "All Errors"
  source_id      = "src-log"
  where          = "SeverityText IN ('ERROR', 'WARN')"
  where_language = "sql"
  tags           = ["errors"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "name", "All Errors"),
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "where_language", "sql"),
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "tags.#", "1"),
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "filters.#", "0"),
				),
			},
		},
	})
}

func TestUnitSavedSearchResource_defaults(t *testing.T) {
	mock := testmock.NewServer(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: mock.ProviderConfig() + `
resource "clickstack_saved_search" "minimal" {
  name      = "Simple Search"
  source_id = "src-log"
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("clickstack_saved_search.minimal", "id"),
				resource.TestCheckResourceAttr("clickstack_saved_search.minimal", "where_language", "lucene"),
				resource.TestCheckResourceAttr("clickstack_saved_search.minimal", "tags.#", "0"),
				resource.TestCheckResourceAttr("clickstack_saved_search.minimal", "filters.#", "0"),
			),
		}},
	})
}
