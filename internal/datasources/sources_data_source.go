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

var _ datasource.DataSource = &SourcesDataSource{}

type SourcesDataSource struct {
	client *client.Client
}

type sourcesDataSourceModel struct {
	Sources []sourceModel `tfsdk:"sources"`
}

type sourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Kind types.String `tfsdk:"kind"`
}

func NewSourcesDataSource() datasource.DataSource {
	return &SourcesDataSource{}
}

func (d *SourcesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sources"
}

func (d *SourcesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all ClickStack data sources (log, trace, metric, session).",
		Attributes: map[string]schema.Attribute{
			"sources": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of data sources.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Source ID.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Source name.",
						},
						"kind": schema.StringAttribute{
							Computed:    true,
							Description: "Source kind (log, trace, metric, session).",
						},
					},
				},
			},
		},
	}
}

func (d *SourcesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SourcesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	sources, err := d.client.ListSources(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list sources", err.Error())
		return
	}

	state := sourcesDataSourceModel{
		Sources: []sourceModel{},
	}
	for _, s := range sources {
		state.Sources = append(state.Sources, sourceModel{
			ID:   types.StringValue(s.ID),
			Name: types.StringValue(s.Name),
			Kind: types.StringValue(s.Kind),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
