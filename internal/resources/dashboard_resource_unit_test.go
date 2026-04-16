// Copyright (c) Lapse Technologies, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/teamlapse/terraform-provider-clickstack/internal/testmock"
)

func TestUnitDashboardResource_basic(t *testing.T) {
	mock := testmock.NewServer(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: mock.ProviderConfig() + `
resource "clickstack_dashboard" "test" {
  name = "Unit Test Dashboard"
  tags = ["unit", "test"]

  tile {
    name = "Request Count"
    x    = 0
    y    = 0
    w    = 12
    h    = 4
    config_json = jsonencode({
      source       = "log"
      type         = "number"
      query        = ""
      field        = ""
      numberFormat = "short"
    })
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("clickstack_dashboard.test", "id"),
					resource.TestCheckResourceAttr("clickstack_dashboard.test", "name", "Unit Test Dashboard"),
					resource.TestCheckResourceAttr("clickstack_dashboard.test", "tags.#", "2"),
					resource.TestCheckResourceAttr("clickstack_dashboard.test", "tile.#", "1"),
					resource.TestCheckResourceAttr("clickstack_dashboard.test", "tile.0.name", "Request Count"),
					resource.TestCheckResourceAttr("clickstack_dashboard.test", "tile.0.w", "12"),
					resource.TestCheckResourceAttrSet("clickstack_dashboard.test", "tile.0.id"),
				),
			},
			// Update: rename, change tile size, add a second tile
			{
				Config: mock.ProviderConfig() + `
resource "clickstack_dashboard" "test" {
  name = "Updated Dashboard"
  tags = ["unit"]

  tile {
    name = "Request Count"
    x    = 0
    y    = 0
    w    = 24
    h    = 6
    config_json = jsonencode({
      source       = "log"
      type         = "number"
      query        = ""
      field        = ""
      numberFormat = "short"
    })
  }

  tile {
    name = "Error Rate"
    x    = 0
    y    = 6
    w    = 12
    h    = 4
    config_json = jsonencode({
      source = "log"
      type   = "line"
      query  = "level:error"
      field  = ""
    })
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("clickstack_dashboard.test", "name", "Updated Dashboard"),
					resource.TestCheckResourceAttr("clickstack_dashboard.test", "tags.#", "1"),
					resource.TestCheckResourceAttr("clickstack_dashboard.test", "tile.#", "2"),
					resource.TestCheckResourceAttr("clickstack_dashboard.test", "tile.0.w", "24"),
				),
			},
		},
	})
}

func TestUnitDashboardResource_withFilter(t *testing.T) {
	mock := testmock.NewServer(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: mock.ProviderConfig() + `
data "clickstack_sources" "all" {}

resource "clickstack_dashboard" "filtered" {
  name        = "Filtered Dashboard"
  saved_query = "level:error"

  filter {
    type       = "field"
    name       = "service"
    expression = "service"
    source_id  = data.clickstack_sources.all.sources[0].id
  }

  tile {
    name = "Logs"
    x    = 0
    y    = 0
    w    = 24
    h    = 8
    config_json = jsonencode({
      source = "log"
      type   = "search"
      query  = ""
    })
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("clickstack_dashboard.filtered", "id"),
					resource.TestCheckResourceAttr("clickstack_dashboard.filtered", "saved_query", "level:error"),
					resource.TestCheckResourceAttr("clickstack_dashboard.filtered", "saved_query_language", "lucene"),
					resource.TestCheckResourceAttr("clickstack_dashboard.filtered", "filter.#", "1"),
					resource.TestCheckResourceAttr("clickstack_dashboard.filtered", "filter.0.type", "field"),
					resource.TestCheckResourceAttrSet("clickstack_dashboard.filtered", "filter.0.id"),
				),
			},
		},
	})
}

func TestUnitDashboardResource_withSeries(t *testing.T) {
	mock := testmock.NewServer(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: mock.ProviderConfig() + `
resource "clickstack_dashboard" "series" {
  name = "Series Dashboard"

  tile {
    name = "Latency"
    x    = 0
    y    = 0
    w    = 12
    h    = 6
    series_json = jsonencode([{
      type          = "time"
      sourceId      = "src-metric"
      aggFn         = "p95"
      where         = "http.server.duration"
      whereLanguage = "lucene"
    }])
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("clickstack_dashboard.series", "id"),
					resource.TestCheckResourceAttrSet("clickstack_dashboard.series", "tile.0.series_json"),
				),
			},
		},
	})
}
