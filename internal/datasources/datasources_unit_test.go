// Copyright (c) Lapse Technologies, Inc.
// SPDX-License-Identifier: MPL-2.0

package datasources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/teamlapse/terraform-provider-clickstack/internal/testmock"
)

func TestUnitSourcesDataSource(t *testing.T) {
	mock := testmock.NewServer(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: mock.ProviderConfig() + `data "clickstack_sources" "all" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.clickstack_sources.all", "sources.#", "4"),
					resource.TestCheckResourceAttr("data.clickstack_sources.all", "sources.0.id", "src-log"),
					resource.TestCheckResourceAttr("data.clickstack_sources.all", "sources.0.kind", "log"),
					resource.TestCheckResourceAttr("data.clickstack_sources.all", "sources.1.kind", "trace"),
					resource.TestCheckResourceAttr("data.clickstack_sources.all", "sources.2.kind", "metric"),
					resource.TestCheckResourceAttr("data.clickstack_sources.all", "sources.3.kind", "session"),
				),
			},
		},
	})
}

func TestUnitWebhooksDataSource(t *testing.T) {
	mock := testmock.NewServer(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: mock.ProviderConfig() + `data "clickstack_webhooks" "all" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.clickstack_webhooks.all", "webhooks.#", "2"),
					resource.TestCheckResourceAttr("data.clickstack_webhooks.all", "webhooks.0.id", "wh-slack"),
					resource.TestCheckResourceAttr("data.clickstack_webhooks.all", "webhooks.0.service", "slack"),
					resource.TestCheckResourceAttr("data.clickstack_webhooks.all", "webhooks.1.id", "wh-pd"),
					resource.TestCheckResourceAttr("data.clickstack_webhooks.all", "webhooks.1.service", "pagerduty_api"),
				),
			},
		},
	})
}
