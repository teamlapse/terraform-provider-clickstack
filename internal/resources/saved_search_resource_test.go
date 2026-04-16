// Copyright (c) Lapse Technologies, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSavedSearchResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "clickstack_saved_search" "test" {
  name    = "tf-acc-test-saved-search"
  query   = "level:error service:api"
  source  = "log"
  tags    = ["test", "errors"]
  columns = ["timestamp", "level", "message", "service"]

  sort {
    field = "timestamp"
    order = "desc"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("clickstack_saved_search.test", "id"),
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "name", "tf-acc-test-saved-search"),
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "query", "level:error service:api"),
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "source", "log"),
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "tags.#", "2"),
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "columns.#", "4"),
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "sort.field", "timestamp"),
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "sort.order", "desc"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "clickstack_saved_search.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				Config: `
resource "clickstack_saved_search" "test" {
  name    = "tf-acc-test-saved-search-updated"
  query   = "level:error OR level:warn"
  source  = "log"
  tags    = ["test"]

  sort {
    field = "timestamp"
    order = "asc"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "name", "tf-acc-test-saved-search-updated"),
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "query", "level:error OR level:warn"),
					resource.TestCheckResourceAttr("clickstack_saved_search.test", "tags.#", "1"),
				),
			},
		},
	})
}
