// Copyright (c) Lapse Technologies, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDashboardResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "clickstack_dashboard" "test" {
  name = "tf-acc-test-basic"
  tags = ["test", "acceptance"]

  tile {
    name = "Request Count"
    x    = 0
    y    = 0
    w    = 12
    h    = 4
    config_json = jsonencode({
      source  = "log"
      type    = "number"
      query   = ""
      field   = ""
      numberFormat = "short"
    })
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("clickstack_dashboard.test", "id"),
					resource.TestCheckResourceAttr("clickstack_dashboard.test", "name", "tf-acc-test-basic"),
					resource.TestCheckResourceAttr("clickstack_dashboard.test", "tags.#", "2"),
					resource.TestCheckResourceAttr("clickstack_dashboard.test", "tile.#", "1"),
					resource.TestCheckResourceAttr("clickstack_dashboard.test", "tile.0.name", "Request Count"),
					resource.TestCheckResourceAttr("clickstack_dashboard.test", "tile.0.w", "12"),
					resource.TestCheckResourceAttrSet("clickstack_dashboard.test", "tile.0.id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "clickstack_dashboard.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update testing
			{
				Config: `
resource "clickstack_dashboard" "test" {
  name = "tf-acc-test-updated"
  tags = ["test"]

  tile {
    name = "Request Count"
    x    = 0
    y    = 0
    w    = 24
    h    = 6
    config_json = jsonencode({
      source  = "log"
      type    = "number"
      query   = ""
      field   = ""
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
					resource.TestCheckResourceAttr("clickstack_dashboard.test", "name", "tf-acc-test-updated"),
					resource.TestCheckResourceAttr("clickstack_dashboard.test", "tags.#", "1"),
					resource.TestCheckResourceAttr("clickstack_dashboard.test", "tile.#", "2"),
					resource.TestCheckResourceAttr("clickstack_dashboard.test", "tile.0.w", "24"),
				),
			},
		},
	})
}

func TestAccDashboardResource_withFilters(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "clickstack_sources" "all" {}

resource "clickstack_dashboard" "filtered" {
  name        = "tf-acc-test-filters"
  saved_query = "level:error"

  filter {
    type       = "field"
    name       = "service"
    expression = "service"
    source_id  = data.clickstack_sources.all.sources[0].id
  }

  tile {
    name = "Filtered Logs"
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
					resource.TestCheckResourceAttr("clickstack_dashboard.filtered", "filter.#", "1"),
					resource.TestCheckResourceAttr("clickstack_dashboard.filtered", "filter.0.type", "field"),
				),
			},
		},
	})
}
