// Copyright (c) Lapse Technologies, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/teamlapse/terraform-provider-clickstack/internal/testmock"
)

func TestUnitAlertResource_tile(t *testing.T) {
	mock := testmock.NewServer(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: mock.ProviderConfig() + `
resource "clickstack_dashboard" "test" {
  name = "Alert Test Dashboard"

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
  name           = "High Error Rate"
  message        = "Error count exceeded threshold"
  source         = "tile"
  threshold      = 100
  threshold_type = "above"
  interval       = "5m"
  dashboard_id   = clickstack_dashboard.test.id
  tile_id        = clickstack_dashboard.test.tile[0].id

  channel {
    type            = "webhook"
    webhook_id      = data.clickstack_webhooks.all.webhooks[0].id
    webhook_service = data.clickstack_webhooks.all.webhooks[0].service
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("clickstack_alert.test", "id"),
					resource.TestCheckResourceAttr("clickstack_alert.test", "name", "High Error Rate"),
					resource.TestCheckResourceAttr("clickstack_alert.test", "source", "tile"),
					resource.TestCheckResourceAttr("clickstack_alert.test", "threshold", "100"),
					resource.TestCheckResourceAttr("clickstack_alert.test", "threshold_type", "above"),
					resource.TestCheckResourceAttr("clickstack_alert.test", "interval", "5m"),
					resource.TestCheckResourceAttr("clickstack_alert.test", "state", "OK"),
					resource.TestCheckResourceAttr("clickstack_alert.test", "channel.type", "webhook"),
				),
			},
			// Update threshold and interval
			{
				Config: mock.ProviderConfig() + `
resource "clickstack_dashboard" "test" {
  name = "Alert Test Dashboard"

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
  name           = "High Error Rate (updated)"
  message        = "Updated message"
  source         = "tile"
  threshold      = 50
  threshold_type = "above"
  interval       = "15m"
  dashboard_id   = clickstack_dashboard.test.id
  tile_id        = clickstack_dashboard.test.tile[0].id

  channel {
    type            = "webhook"
    webhook_id      = data.clickstack_webhooks.all.webhooks[0].id
    webhook_service = data.clickstack_webhooks.all.webhooks[0].service
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("clickstack_alert.test", "name", "High Error Rate (updated)"),
					resource.TestCheckResourceAttr("clickstack_alert.test", "threshold", "50"),
					resource.TestCheckResourceAttr("clickstack_alert.test", "interval", "15m"),
				),
			},
		},
	})
}

func TestUnitAlertResource_savedSearch(t *testing.T) {
	mock := testmock.NewServer(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: mock.ProviderConfig() + `
resource "clickstack_saved_search" "errors" {
  name   = "Error Logs"
  query  = "level:error"
  source = "log"

  sort {
    field = "timestamp"
    order = "desc"
  }
}

resource "clickstack_alert" "test" {
  name            = "Saved Search Alert"
  source          = "saved_search"
  saved_search_id = clickstack_saved_search.errors.id
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
					resource.TestCheckResourceAttrSet("clickstack_alert.test", "id"),
					resource.TestCheckResourceAttr("clickstack_alert.test", "source", "saved_search"),
					resource.TestCheckResourceAttrSet("clickstack_alert.test", "saved_search_id"),
					resource.TestCheckResourceAttr("clickstack_alert.test", "channel.type", "email"),
					resource.TestCheckResourceAttr("clickstack_alert.test", "channel.email_recipients.#", "1"),
				),
			},
		},
	})
}

func TestUnitAlertResource_pagerduty(t *testing.T) {
	mock := testmock.NewServer(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: mock.ProviderConfig() + `
resource "clickstack_saved_search" "oom" {
  name   = "OOM Events"
  query  = "OOMKilled"
  source = "log"

  sort {
    field = "timestamp"
    order = "desc"
  }
}

data "clickstack_webhooks" "all" {}

resource "clickstack_alert" "test" {
  name            = "OOM Alert"
  source          = "saved_search"
  saved_search_id = clickstack_saved_search.oom.id
  threshold       = 1
  threshold_type  = "above"
  interval        = "5m"

  channel {
    type            = "webhook"
    webhook_id      = data.clickstack_webhooks.all.webhooks[1].id
    webhook_service = "pagerduty_api"
    severity        = "critical"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("clickstack_alert.test", "channel.type", "webhook"),
					resource.TestCheckResourceAttr("clickstack_alert.test", "channel.webhook_service", "pagerduty_api"),
					resource.TestCheckResourceAttr("clickstack_alert.test", "channel.severity", "critical"),
				),
			},
		},
	})
}
