// Copyright (c) Lapse Technologies, Inc.
// SPDX-License-Identifier: MPL-2.0

package datasources

import (
	"context"

	"github.com/teamlapse/terraform-provider-clickstack/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &WebhooksDataSource{}

type WebhooksDataSource struct {
	client *client.Client
}

type webhooksDataSourceModel struct {
	Webhooks []webhookModel `tfsdk:"webhooks"`
}

type webhookModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Service     types.String `tfsdk:"service"`
	URL         types.String `tfsdk:"url"`
	Description types.String `tfsdk:"description"`
}

func NewWebhooksDataSource() datasource.DataSource {
	return &WebhooksDataSource{}
}

func (d *WebhooksDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhooks"
}

func (d *WebhooksDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all ClickStack webhooks.",
		Attributes: map[string]schema.Attribute{
			"webhooks": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of webhooks.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Webhook ID.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Webhook name.",
						},
						"service": schema.StringAttribute{
							Computed:    true,
							Description: "Webhook service type (slack, incidentio, generic, slack_api, pagerduty_api).",
						},
						"url": schema.StringAttribute{
							Computed:    true,
							Sensitive:   true,
							Description: "Webhook URL.",
						},
						"description": schema.StringAttribute{
							Computed:    true,
							Description: "Webhook description.",
						},
					},
				},
			},
		},
	}
}

func (d *WebhooksDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected data source configure type", "Expected *client.Client")
		return
	}
	d.client = c
}

func (d *WebhooksDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	webhooks, err := d.client.ListWebhooks(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list webhooks", err.Error())
		return
	}

	state := webhooksDataSourceModel{
		Webhooks: []webhookModel{},
	}
	for _, w := range webhooks {
		desc := types.StringNull()
		if w.Description != nil {
			desc = types.StringValue(*w.Description)
		}
		state.Webhooks = append(state.Webhooks, webhookModel{
			ID:          types.StringValue(w.ID),
			Name:        types.StringValue(w.Name),
			Service:     types.StringValue(w.Service),
			URL:         types.StringValue(w.URL),
			Description: desc,
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
