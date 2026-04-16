// Copyright (c) Lapse Technologies, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"encoding/json"

	"github.com/teamlapse/terraform-provider-clickstack/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &DashboardResource{}
	_ resource.ResourceWithImportState = &DashboardResource{}
)

type DashboardResource struct {
	client *client.Client
}

type dashboardResourceModel struct {
	ID                 types.String  `tfsdk:"id"`
	Name               types.String  `tfsdk:"name"`
	Tags               types.List    `tfsdk:"tags"`
	SavedQuery         types.String  `tfsdk:"saved_query"`
	SavedQueryLanguage types.String  `tfsdk:"saved_query_language"`
	Tiles              []tileModel   `tfsdk:"tile"`
	Filters            []filterModel `tfsdk:"filter"`
}

type filterModel struct {
	ID               types.String `tfsdk:"id"`
	Type             types.String `tfsdk:"type"`
	Name             types.String `tfsdk:"name"`
	Expression       types.String `tfsdk:"expression"`
	SourceID         types.String `tfsdk:"source_id"`
	SourceMetricType types.String `tfsdk:"source_metric_type"`
}

type tileModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	X          types.Int64  `tfsdk:"x"`
	Y          types.Int64  `tfsdk:"y"`
	W          types.Int64  `tfsdk:"w"`
	H          types.Int64  `tfsdk:"h"`
	ConfigJSON types.String `tfsdk:"config_json"`
	SeriesJSON types.String `tfsdk:"series_json"`
}

func NewDashboardResource() resource.Resource {
	return &DashboardResource{}
}

func (r *DashboardResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dashboard"
}

func (r *DashboardResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a ClickStack dashboard.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Dashboard ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Dashboard name.",
			},
			"tags": schema.ListAttribute{
				Optional:    true,
				Description: "Tags for organizing dashboards.",
				ElementType: types.StringType,
			},
			"saved_query": schema.StringAttribute{
				Optional:    true,
				Description: "Default query pre-populated in the dashboard search bar.",
			},
			"saved_query_language": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Query language for the saved query: 'lucene' or 'sql'. Defaults to 'lucene' if not specified.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"tile": schema.ListNestedBlock{
				Description: "Dashboard tiles.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Optional:    true,
							Description: "Tile ID. Auto-generated if omitted. Note: the API always generates IDs; user-provided values are treated as hints only.",
							PlanModifiers: []planmodifier.String{
								useUnknownOnCreate{},
							},
						},
						"name": schema.StringAttribute{
							Required:    true,
							Description: "Tile name.",
						},
						"x": schema.Int64Attribute{
							Required:    true,
							Description: "X position on the 24-column grid.",
						},
						"y": schema.Int64Attribute{
							Required:    true,
							Description: "Y position on the grid.",
						},
						"w": schema.Int64Attribute{
							Required:    true,
							Description: "Width in grid columns (max 24).",
						},
						"h": schema.Int64Attribute{
							Required:    true,
							Description: "Height in grid rows.",
						},
						"config_json": schema.StringAttribute{
							Optional:    true,
							Description: "Tile configuration as JSON (builder format). Use jsonencode() to construct. Supports all chart types: line, stacked_bar, table, number, pie, search, markdown.",
							Validators: []validator.String{
								jsonValidator{},
							},
						},
						"series_json": schema.StringAttribute{
							Optional:    true,
							Description: "Tile data series as JSON array (series format). Use jsonencode() to construct. Each series has type, sourceId, aggFn, where, whereLanguage, groupBy. Use this for non-search tiles that need query filtering.",
							Validators: []validator.String{
								jsonValidator{},
							},
						},
					},
				},
			},
			"filter": schema.ListNestedBlock{
				Description: "Dashboard-level filters applied across all tiles.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Optional:    true,
							Description: "Filter ID. Auto-generated if omitted.",
							PlanModifiers: []planmodifier.String{
								useUnknownOnCreate{},
							},
						},
						"type": schema.StringAttribute{
							Required:    true,
							Description: "Filter type (e.g. 'field', 'sql').",
						},
						"name": schema.StringAttribute{
							Required:    true,
							Description: "Filter display name.",
						},
						"expression": schema.StringAttribute{
							Required:    true,
							Description: "Filter expression.",
						},
						"source_id": schema.StringAttribute{
							Required:    true,
							Description: "Source ID this filter applies to.",
						},
						"source_metric_type": schema.StringAttribute{
							Optional:    true,
							Description: "Metric type when the source is a metric source.",
						},
					},
				},
			},
		},
	}
}

func (r *DashboardResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected resource configure type", "Expected *client.Client")
		return
	}
	r.client = c
}

func (r *DashboardResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dashboardResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dashboard := expandDashboard(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateDashboard(ctx, dashboard)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create dashboard", err.Error())
		return
	}

	state := mergePlanWithAPIResponse(ctx, plan, created, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DashboardResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dashboardResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dashboard, err := r.client.GetDashboard(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read dashboard", err.Error())
		return
	}

	newState := flattenDashboard(ctx, dashboard, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *DashboardResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dashboardResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var currentState dashboardResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &currentState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dashboard := expandDashboard(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateDashboard(ctx, currentState.ID.ValueString(), dashboard)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update dashboard", err.Error())
		return
	}

	newState := mergePlanWithAPIResponse(ctx, plan, updated, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *DashboardResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dashboardResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteDashboard(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Unable to delete dashboard", err.Error())
	}
}

func (r *DashboardResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// mergePlanWithAPIResponse creates the Terraform state by using plan values
// for user-specified fields and API response values for server-generated fields.
func mergePlanWithAPIResponse(ctx context.Context, plan dashboardResourceModel, api *client.Dashboard, diags *diag.Diagnostics) dashboardResourceModel {
	state := plan

	state.ID = types.StringValue(api.ID)

	if api.SavedQueryLanguage != nil {
		state.SavedQueryLanguage = types.StringValue(*api.SavedQueryLanguage)
	} else {
		state.SavedQueryLanguage = types.StringNull()
	}

	for i := range state.Tiles {
		if i < len(api.Tiles) {
			state.Tiles[i].ID = types.StringValue(api.Tiles[i].ID)
		}
		if !state.Tiles[i].ConfigJSON.IsNull() && !state.Tiles[i].ConfigJSON.IsUnknown() {
			state.Tiles[i].ConfigJSON = types.StringValue(normalizeJSON(state.Tiles[i].ConfigJSON.ValueString()))
		}
		if !state.Tiles[i].SeriesJSON.IsNull() && !state.Tiles[i].SeriesJSON.IsUnknown() {
			state.Tiles[i].SeriesJSON = types.StringValue(normalizeJSON(state.Tiles[i].SeriesJSON.ValueString()))
		}
	}
	for i := range state.Filters {
		if i < len(api.Filters) {
			state.Filters[i].ID = types.StringValue(api.Filters[i].ID)
		}
	}

	return state
}

// normalizeJSON re-marshals JSON to produce canonical key ordering.
func normalizeJSON(raw string) string {
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return raw
	}
	b, err := json.Marshal(v)
	if err != nil {
		return raw
	}
	return string(b)
}

// expandDashboard converts the Terraform model to an API request.
func expandDashboard(ctx context.Context, plan dashboardResourceModel, diags *diag.Diagnostics) client.Dashboard {
	d := client.Dashboard{
		Name: plan.Name.ValueString(),
	}

	if !plan.Tags.IsNull() && !plan.Tags.IsUnknown() {
		var tags []string
		diags.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)
		d.Tags = tags
	}

	if !plan.SavedQuery.IsNull() && !plan.SavedQuery.IsUnknown() {
		v := plan.SavedQuery.ValueString()
		d.SavedQuery = &v
	}
	if !plan.SavedQueryLanguage.IsNull() && !plan.SavedQueryLanguage.IsUnknown() {
		v := plan.SavedQueryLanguage.ValueString()
		d.SavedQueryLanguage = &v
	}

	for _, t := range plan.Tiles {
		tile := client.Tile{
			Name: t.Name.ValueString(),
			X:    int(t.X.ValueInt64()),
			Y:    int(t.Y.ValueInt64()),
			W:    int(t.W.ValueInt64()),
			H:    int(t.H.ValueInt64()),
		}
		if !t.ID.IsNull() && !t.ID.IsUnknown() {
			tile.ID = t.ID.ValueString()
		}
		if !t.ConfigJSON.IsNull() && !t.ConfigJSON.IsUnknown() {
			tile.Config = json.RawMessage(t.ConfigJSON.ValueString())
		}
		if !t.SeriesJSON.IsNull() && !t.SeriesJSON.IsUnknown() {
			tile.Series = json.RawMessage(t.SeriesJSON.ValueString())
		}
		d.Tiles = append(d.Tiles, tile)
	}

	for _, f := range plan.Filters {
		filter := client.Filter{
			Type:       f.Type.ValueString(),
			Name:       f.Name.ValueString(),
			Expression: f.Expression.ValueString(),
			SourceID:   f.SourceID.ValueString(),
		}
		if !f.ID.IsNull() && !f.ID.IsUnknown() {
			filter.ID = f.ID.ValueString()
		}
		if !f.SourceMetricType.IsNull() && !f.SourceMetricType.IsUnknown() {
			v := f.SourceMetricType.ValueString()
			filter.SourceMetricType = &v
		}
		d.Filters = append(d.Filters, filter)
	}

	return d
}

// flattenDashboard converts an API response to the Terraform model.
func flattenDashboard(ctx context.Context, d *client.Dashboard, diags *diag.Diagnostics) dashboardResourceModel {
	model := dashboardResourceModel{
		ID:   types.StringValue(d.ID),
		Name: types.StringValue(d.Name),
	}

	if len(d.Tags) > 0 {
		tagList, tagDiags := types.ListValueFrom(ctx, types.StringType, d.Tags)
		diags.Append(tagDiags...)
		model.Tags = tagList
	} else {
		model.Tags = types.ListNull(types.StringType)
	}

	if d.SavedQuery != nil {
		model.SavedQuery = types.StringValue(*d.SavedQuery)
	} else {
		model.SavedQuery = types.StringNull()
	}
	if d.SavedQueryLanguage != nil {
		model.SavedQueryLanguage = types.StringValue(*d.SavedQueryLanguage)
	} else {
		model.SavedQueryLanguage = types.StringNull()
	}

	for _, t := range d.Tiles {
		tile := tileModel{
			ID:   types.StringValue(t.ID),
			Name: types.StringValue(t.Name),
			X:    types.Int64Value(int64(t.X)),
			Y:    types.Int64Value(int64(t.Y)),
			W:    types.Int64Value(int64(t.W)),
			H:    types.Int64Value(int64(t.H)),
		}
		if len(t.Config) > 0 {
			var normalized any
			if err := json.Unmarshal(t.Config, &normalized); err == nil {
				if b, err := json.Marshal(normalized); err == nil {
					tile.ConfigJSON = types.StringValue(string(b))
				} else {
					tile.ConfigJSON = types.StringValue(string(t.Config))
				}
			} else {
				tile.ConfigJSON = types.StringValue(string(t.Config))
			}
		} else {
			tile.ConfigJSON = types.StringNull()
		}
		if len(t.Series) > 0 {
			var normalized any
			if err := json.Unmarshal(t.Series, &normalized); err == nil {
				if b, err := json.Marshal(normalized); err == nil {
					tile.SeriesJSON = types.StringValue(string(b))
				} else {
					tile.SeriesJSON = types.StringValue(string(t.Series))
				}
			} else {
				tile.SeriesJSON = types.StringValue(string(t.Series))
			}
		} else {
			tile.SeriesJSON = types.StringNull()
		}
		model.Tiles = append(model.Tiles, tile)
	}

	for _, f := range d.Filters {
		filter := filterModel{
			ID:         types.StringValue(f.ID),
			Type:       types.StringValue(f.Type),
			Name:       types.StringValue(f.Name),
			Expression: types.StringValue(f.Expression),
			SourceID:   types.StringValue(f.SourceID),
		}
		if f.SourceMetricType != nil {
			filter.SourceMetricType = types.StringValue(*f.SourceMetricType)
		} else {
			filter.SourceMetricType = types.StringNull()
		}
		model.Filters = append(model.Filters, filter)
	}

	return model
}

// useUnknownOnCreate is a plan modifier that marks a value as unknown during
// resource creation.
type useUnknownOnCreate struct{}

func (m useUnknownOnCreate) Description(_ context.Context) string {
	return "Use unknown value during create, preserve state during update."
}

func (m useUnknownOnCreate) MarkdownDescription(_ context.Context) string {
	return "Use unknown value during create, preserve state during update."
}

func (m useUnknownOnCreate) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() && (req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown()) {
		resp.PlanValue = types.StringUnknown()
		return
	}
	if !req.StateValue.IsNull() {
		resp.PlanValue = req.StateValue
	}
}

// jsonValidator validates that a string is valid JSON.
type jsonValidator struct{}

func (v jsonValidator) Description(_ context.Context) string {
	return "value must be valid JSON"
}

func (v jsonValidator) MarkdownDescription(_ context.Context) string {
	return "value must be valid JSON"
}

func (v jsonValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if !json.Valid([]byte(req.ConfigValue.ValueString())) {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid JSON", "config_json must be valid JSON")
	}
}
