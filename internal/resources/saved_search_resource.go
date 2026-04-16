// Copyright (c) Lapse Technologies, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"

	"github.com/teamlapse/terraform-provider-clickstack/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
	_ resource.Resource                = &SavedSearchResource{}
	_ resource.ResourceWithImportState = &SavedSearchResource{}
)

type SavedSearchResource struct {
	client *client.Client
}

type savedSearchResourceModel struct {
	ID      types.String          `tfsdk:"id"`
	Name    types.String          `tfsdk:"name"`
	Query   types.String          `tfsdk:"query"`
	Source  types.String          `tfsdk:"source"`
	Tags    types.List            `tfsdk:"tags"`
	Columns types.List            `tfsdk:"columns"`
	Sort    *savedSearchSortModel `tfsdk:"sort"`
}

type savedSearchSortModel struct {
	Field types.String `tfsdk:"field"`
	Order types.String `tfsdk:"order"`
}

func NewSavedSearchResource() resource.Resource {
	return &SavedSearchResource{}
}

func (r *SavedSearchResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_saved_search"
}

func (r *SavedSearchResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a ClickStack saved search. Saved searches can be referenced by alerts as a source.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Saved search ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Saved search name.",
			},
			"query": schema.StringAttribute{
				Required:    true,
				Description: "Search query string.",
			},
			"source": schema.StringAttribute{
				Required:    true,
				Description: "Data source kind: 'log', 'trace', 'metric', or 'session'.",
				Validators: []validator.String{
					stringvalidator.OneOf("log", "trace", "metric", "session"),
				},
			},
			"tags": schema.ListAttribute{
				Optional:    true,
				Description: "Tags for organizing saved searches.",
				ElementType: types.StringType,
			},
			"columns": schema.ListAttribute{
				Optional:    true,
				Description: "Columns to display in search results.",
				ElementType: types.StringType,
			},
		},
		Blocks: map[string]schema.Block{
			"sort": schema.SingleNestedBlock{
				Description: "Sort order for search results.",
				Attributes: map[string]schema.Attribute{
					"field": schema.StringAttribute{
						Required:    true,
						Description: "Field to sort by.",
					},
					"order": schema.StringAttribute{
						Required:    true,
						Description: "Sort direction: 'asc' or 'desc'.",
						Validators: []validator.String{
							stringvalidator.OneOf("asc", "desc"),
						},
					},
				},
			},
		},
	}
}

func (r *SavedSearchResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SavedSearchResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan savedSearchResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	search := expandSavedSearch(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateSavedSearch(ctx, search)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create saved search", err.Error())
		return
	}

	state := flattenSavedSearch(ctx, created, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SavedSearchResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state savedSearchResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	search, err := r.client.GetSavedSearch(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read saved search", err.Error())
		return
	}

	newState := flattenSavedSearch(ctx, search, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *SavedSearchResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan savedSearchResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state savedSearchResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	search := expandSavedSearch(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateSavedSearch(ctx, state.ID.ValueString(), search)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update saved search", err.Error())
		return
	}

	newState := flattenSavedSearch(ctx, updated, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *SavedSearchResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state savedSearchResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSavedSearch(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Unable to delete saved search", err.Error())
	}
}

func (r *SavedSearchResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func expandSavedSearch(ctx context.Context, plan savedSearchResourceModel, diags *diag.Diagnostics) client.SavedSearch {
	s := client.SavedSearch{
		Name:   plan.Name.ValueString(),
		Query:  plan.Query.ValueString(),
		Source: plan.Source.ValueString(),
	}

	if !plan.Tags.IsNull() && !plan.Tags.IsUnknown() {
		var tags []string
		diags.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)
		s.Tags = tags
	}

	if !plan.Columns.IsNull() && !plan.Columns.IsUnknown() {
		var columns []string
		diags.Append(plan.Columns.ElementsAs(ctx, &columns, false)...)
		s.Columns = columns
	}

	if plan.Sort != nil {
		s.Sort = &client.SavedSearchSort{
			Field: plan.Sort.Field.ValueString(),
			Order: plan.Sort.Order.ValueString(),
		}
	}

	return s
}

func flattenSavedSearch(ctx context.Context, s *client.SavedSearch, diags *diag.Diagnostics) savedSearchResourceModel {
	model := savedSearchResourceModel{
		ID:     types.StringValue(s.ID),
		Name:   types.StringValue(s.Name),
		Query:  types.StringValue(s.Query),
		Source: types.StringValue(s.Source),
	}

	if len(s.Tags) > 0 {
		tagList, d := types.ListValueFrom(ctx, types.StringType, s.Tags)
		diags.Append(d...)
		model.Tags = tagList
	} else {
		model.Tags = types.ListNull(types.StringType)
	}

	if len(s.Columns) > 0 {
		colList, d := types.ListValueFrom(ctx, types.StringType, s.Columns)
		diags.Append(d...)
		model.Columns = colList
	} else {
		model.Columns = types.ListNull(types.StringType)
	}

	if s.Sort != nil {
		model.Sort = &savedSearchSortModel{
			Field: types.StringValue(s.Sort.Field),
			Order: types.StringValue(s.Sort.Order),
		}
	}

	return model
}
