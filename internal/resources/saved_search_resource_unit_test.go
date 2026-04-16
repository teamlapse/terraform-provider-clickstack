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
  name    = "API Errors"
  query   = "level:error service:api"
  source  = "log"
  tags    = ["api", "errors"]
  columns = ["timestamp", "level", "message"]

  sort {
    field = "timestamp"
    order = "desc"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("clickstack_saved_search.test", "id"),
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "name", "API Errors"),
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "query", "level:error service:api"),
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "source", "log"),
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "tags.#", "2"),
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "columns.#", "3"),
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "sort.field", "timestamp"),
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "sort.order", "desc"),
				),
			},
			// Update: change query and tags
			{
				Config: mock.ProviderConfig() + `
resource "clickstack_saved_search" "test" {
  name    = "All Errors"
  query   = "level:error OR level:warn"
  source  = "log"
  tags    = ["errors"]

  sort {
    field = "timestamp"
    order = "asc"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "name", "All Errors"),
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "query", "level:error OR level:warn"),
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "tags.#", "1"),
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "sort.order", "asc"),
				),
			},
		},
	})
}

func TestUnitSavedSearchResource_minimal(t *testing.T) {
	mock := testmock.NewServer(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: mock.ProviderConfig() + `
resource "clickstack_saved_search" "minimal" {
  name   = "Simple Search"
  query  = "level:error"
  source = "log"

  sort {
    field = "timestamp"
    order = "desc"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("clickstack_saved_search.minimal", "id"),
					resource.TestCheckResourceAttr("clickstack_saved_search.minimal", "name", "Simple Search"),
					resource.TestCheckResourceAttr("clickstack_saved_search.minimal", "tags.#", "0"),
					resource.TestCheckResourceAttr("clickstack_saved_search.minimal", "columns.#", "0"),
				),
			},
		},
	})
}

func TestUnitSavedSearchResource_trace(t *testing.T) {
	mock := testmock.NewServer(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: mock.ProviderConfig() + `
resource "clickstack_saved_search" "traces" {
  name   = "Slow Traces"
  query  = "duration:>1000"
  source = "trace"
  tags   = ["performance"]

  sort {
    field = "duration"
    order = "desc"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("clickstack_saved_search.traces", "source", "trace"),
				),
			},
		},
	})
}
