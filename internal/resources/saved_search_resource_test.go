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
		Steps: []resource.TestStep{{
			Config: `
data "clickstack_sources" "all" {}

resource "clickstack_saved_search" "test" {
  name      = "tf-acc-test-saved-search"
  source_id = data.clickstack_sources.all.sources[0].id
  where     = "SeverityText:ERROR"
  tags      = ["test", "errors"]
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("clickstack_saved_search.test", "id"),
				resource.TestCheckResourceAttr("clickstack_saved_search.test", "where_language", "lucene"),
			),
		}},
	})
}
