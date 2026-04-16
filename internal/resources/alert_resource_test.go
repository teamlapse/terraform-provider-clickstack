// Copyright (c) Lapse Technologies, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAlertResource_tile(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "clickstack_dashboard" "test" {
  name = "tf-acc-test-alert-dashboard"

  tile {
    name = "Error Count"
    x    = 0
    y    = 0
    w    = 12
    h    = 4
    config_json = jsonencode({
      source = "log"
      type   = "number"
      query  = "level:error"
      field  = ""
    })
  }
}

data "clickstack_webhooks" "all" {}

resource "clickstack_alert" "test" {
  name           = "tf-acc-test-alert"
  message        = "Error count exceeded threshold"
  source         = "tile"
  threshold      = 100
  threshold_type = "above"
  interval       = "5m"
  dashboard_id   = clickstack_dashboard.test.id
  tile_id        = clickstack_dashboard.test.tile[0].id

  channel {
    type    = "webhook"
    webhook_id      = data.clickstack_webhooks.all.webhooks[0].id
    webhook_service = data.clickstack_webhooks.all.webhooks[0].service
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("clickstack_alert.test", "id"),
					resource.TestCheckResourceAttr("clickstack_alert.test", "name", "tf-acc-test-alert"),
					resource.TestCheckResourceAttr("clickstack_alert.test", "source", "tile"),
					resource.TestCheckResourceAttr("clickstack_alert.test", "threshold", "100"),
					resource.TestCheckResourceAttr("clickstack_alert.test", "threshold_type", "above"),
					resource.TestCheckResourceAttr("clickstack_alert.test", "interval", "5m"),
					resource.TestCheckResourceAttrSet("clickstack_alert.test", "state"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "clickstack_alert.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update threshold
			{
				Config: `
resource "clickstack_dashboard" "test" {
  name = "tf-acc-test-alert-dashboard"

  tile {
    name = "Error Count"
    x    = 0
    y    = 0
    w    = 12
    h    = 4
    config_json = jsonencode({
      source = "log"
      type   = "number"
      query  = "level:error"
      field  = ""
    })
  }
}

data "clickstack_webhooks" "all" {}

resource "clickstack_alert" "test" {
  name           = "tf-acc-test-alert-updated"
  message        = "Error count exceeded updated threshold"
  source         = "tile"
  threshold      = 50
  threshold_type = "above"
  interval       = "15m"
  dashboard_id   = clickstack_dashboard.test.id
  tile_id        = clickstack_dashboard.test.tile[0].id

  channel {
    type    = "webhook"
    webhook_id      = data.clickstack_webhooks.all.webhooks[0].id
    webhook_service = data.clickstack_webhooks.all.webhooks[0].service
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("clickstack_alert.test", "name", "tf-acc-test-alert-updated"),
					resource.TestCheckResourceAttr("clickstack_alert.test", "threshold", "50"),
					resource.TestCheckResourceAttr("clickstack_alert.test", "interval", "15m"),
				),
			},
		},
	})
}

func TestAccAlertResource_savedSearch(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "clickstack_saved_search" "test" {
  name   = "tf-acc-test-saved-search-for-alert"
  query  = "level:error"
  source = "log"
}

resource "clickstack_alert" "saved_search_alert" {
  name            = "tf-acc-test-saved-search-alert"
  source          = "saved_search"
  saved_search_id = clickstack_saved_search.test.id
  threshold       = 10
  threshold_type  = "above"
  interval        = "5m"

  channel {
    type             = "email"
    email_recipients = ["oncall@example.com"]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("clickstack_alert.saved_search_alert", "id"),
					resource.TestCheckResourceAttr("clickstack_alert.saved_search_alert", "source", "saved_search"),
					resource.TestCheckResourceAttrSet("clickstack_alert.saved_search_alert", "saved_search_id"),
				),
			},
		},
	})
}
