// Copyright (c) Lapse Technologies, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	"github.com/teamlapse/terraform-provider-clickstack/internal/client"
	"github.com/teamlapse/terraform-provider-clickstack/internal/datasources"
	"github.com/teamlapse/terraform-provider-clickstack/internal/resources"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &ClickStackProvider{}

// ClickStackProvider implements the Terraform provider for managed ClickStack.
type ClickStackProvider struct {
	version string
}

type clickStackProviderModel struct {
	BaseURL        types.String `tfsdk:"base_url"`
	OrganizationID types.String `tfsdk:"organization_id"`
	ServiceID      types.String `tfsdk:"service_id"`
	APIKeyID       types.String `tfsdk:"api_key_id"`
	APIKeySecret   types.String `tfsdk:"api_key_secret"`
}

// New returns a new provider factory function.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ClickStackProvider{version: version}
	}
}

func (p *ClickStackProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "clickstack"
	resp.Version = p.version
}

func (p *ClickStackProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for managing dashboards and alerts on managed ClickStack (HyperDX on ClickHouse Cloud).",
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				Description: "Base URL of the ClickHouse Cloud API. Defaults to https://api.clickhouse.cloud. Can also be set via CLICKSTACK_BASE_URL env var.",
				Optional:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "ClickHouse Cloud organization ID. Can also be set via CLICKSTACK_ORGANIZATION_ID env var.",
				Required:    true,
			},
			"service_id": schema.StringAttribute{
				Description: "ClickHouse Cloud service ID for the ClickStack instance. Can also be set via CLICKSTACK_SERVICE_ID env var.",
				Required:    true,
			},
			"api_key_id": schema.StringAttribute{
				Description: "ClickHouse Cloud API key ID. Can also be set via CLICKSTACK_API_KEY_ID env var.",
				Required:    true,
				Sensitive:   true,
			},
			"api_key_secret": schema.StringAttribute{
				Description: "ClickHouse Cloud API key secret. Can also be set via CLICKSTACK_API_KEY_SECRET env var.",
				Required:    true,
				Sensitive:   true,
			},
		},
	}
}

func (p *ClickStackProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config clickStackProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	baseURL := "https://api.clickhouse.cloud"
	if !config.BaseURL.IsNull() && !config.BaseURL.IsUnknown() {
		baseURL = config.BaseURL.ValueString()
	} else if v := os.Getenv("CLICKSTACK_BASE_URL"); v != "" {
		baseURL = v
	}

	organizationID := config.OrganizationID.ValueString()
	if organizationID == "" {
		organizationID = os.Getenv("CLICKSTACK_ORGANIZATION_ID")
	}
	if organizationID == "" {
		resp.Diagnostics.AddError("Missing organization_id", "organization_id must be set in the provider configuration or via CLICKSTACK_ORGANIZATION_ID env var.")
		return
	}

	serviceID := config.ServiceID.ValueString()
	if serviceID == "" {
		serviceID = os.Getenv("CLICKSTACK_SERVICE_ID")
	}
	if serviceID == "" {
		resp.Diagnostics.AddError("Missing service_id", "service_id must be set in the provider configuration or via CLICKSTACK_SERVICE_ID env var.")
		return
	}

	apiKeyID := config.APIKeyID.ValueString()
	if apiKeyID == "" {
		apiKeyID = os.Getenv("CLICKSTACK_API_KEY_ID")
	}
	if apiKeyID == "" {
		resp.Diagnostics.AddError("Missing api_key_id", "api_key_id must be set in the provider configuration or via CLICKSTACK_API_KEY_ID env var.")
		return
	}

	apiKeySecret := config.APIKeySecret.ValueString()
	if apiKeySecret == "" {
		apiKeySecret = os.Getenv("CLICKSTACK_API_KEY_SECRET")
	}
	if apiKeySecret == "" {
		resp.Diagnostics.AddError("Missing api_key_secret", "api_key_secret must be set in the provider configuration or via CLICKSTACK_API_KEY_SECRET env var.")
		return
	}

	c := client.NewClient(baseURL, organizationID, serviceID, apiKeyID, apiKeySecret)
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *ClickStackProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewDashboardResource,
		resources.NewAlertResource,
		resources.NewSavedSearchResource,
	}
}

func (p *ClickStackProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		datasources.NewSourcesDataSource,
		datasources.NewWebhooksDataSource,
	}
}
