// Copyright (c) Lapse Technologies, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/teamlapse/terraform-provider-clickstack/internal/client"
)

var (
	_ resource.Resource                = &SavedSearchResource{}
	_ resource.ResourceWithImportState = &SavedSearchResource{}
)

type SavedSearchResource struct {
	client *client.Client
}

type savedSearchResourceModel struct {
	ID            types.String             `tfsdk:"id"`
	Name          types.String             `tfsdk:"name"`
	SourceID      types.String             `tfsdk:"source_id"`
	Select        types.String             `tfsdk:"select"`
	Where         types.String             `tfsdk:"where"`
	WhereLanguage types.String             `tfsdk:"where_language"`
	OrderBy       types.String             `tfsdk:"order_by"`
	Tags          types.List               `tfsdk:"tags"`
	Filters       []savedSearchFilterModel `tfsdk:"filters"`
}

type savedSearchFilterModel struct {
	Type      types.String `tfsdk:"type"`
	Condition types.String `tfsdk:"condition"`
}

func NewSavedSearchResource() resource.Resource {
	return &SavedSearchResource{}
}

func (r *SavedSearchResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_saved_search"
}

func (r *SavedSearchResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a ClickStack saved search. Saved searches can be used as alert sources.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Saved search ID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Display name for the saved search.",
				Validators:  []validator.String{stringvalidator.UTF8LengthAtMost(1024)},
			},
			"source_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the ClickStack source to query. Use data.clickstack_sources to discover source IDs.",
			},
			"select": schema.StringAttribute{
				Optional:    true,
				Description: "Comma-separated column expressions to display. Empty uses the source default.",
				Validators:  []validator.String{stringvalidator.UTF8LengthAtMost(4096)},
			},
			"where": schema.StringAttribute{
				Optional:    true,
				Description: "Row filter expression, interpreted according to where_language.",
				Validators:  []validator.String{stringvalidator.UTF8LengthAtMost(8192)},
			},
			"where_language": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("lucene"),
				Description: "Language used for where: lucene or sql. Defaults to lucene.",
				Validators:  []validator.String{stringvalidator.OneOf("lucene", "sql")},
			},
			"order_by": schema.StringAttribute{
				Optional:    true,
				Description: "ORDER BY expression. Empty uses the source default.",
				Validators:  []validator.String{stringvalidator.UTF8LengthAtMost(1024)},
			},
			"tags": schema.ListAttribute{
				Optional:    true,
				Description: "Tags used to organize saved searches.",
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.SizeAtMost(50),
					listvalidator.ValueStringsAre(stringvalidator.UTF8LengthAtMost(32)),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"filters": schema.ListNestedBlock{
				Description: "Structured SQL filters pinned to the saved search.",
				NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("sql"),
						Description: "Filter type. ClickStack currently supports sql.",
						Validators:  []validator.String{stringvalidator.OneOf("sql")},
					},
					"condition": schema.StringAttribute{
						Required:    true,
						Description: "SQL predicate, for example ServiceName IN ('checkout', 'payments').",
					},
				}},
				Validators: []validator.List{listvalidator.SizeAtMost(100)},
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
	resp.Diagnostics.Append(resp.State.Set(ctx, flattenSavedSearch(ctx, created, &resp.Diagnostics))...)
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
	resp.Diagnostics.Append(resp.State.Set(ctx, flattenSavedSearch(ctx, search, &resp.Diagnostics))...)
}

func (r *SavedSearchResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state savedSearchResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
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
	resp.Diagnostics.Append(resp.State.Set(ctx, flattenSavedSearch(ctx, updated, &resp.Diagnostics))...)
}

func (r *SavedSearchResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state savedSearchResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteSavedSearch(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete saved search", err.Error())
	}
}

func (r *SavedSearchResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func expandSavedSearch(ctx context.Context, plan savedSearchResourceModel, diags *diag.Diagnostics) client.SavedSearch {
	search := client.SavedSearch{
		Name:          plan.Name.ValueString(),
		SourceID:      plan.SourceID.ValueString(),
		Select:        plan.Select.ValueString(),
		Where:         plan.Where.ValueString(),
		WhereLanguage: plan.WhereLanguage.ValueString(),
		OrderBy:       plan.OrderBy.ValueString(),
	}
	if !plan.Tags.IsNull() && !plan.Tags.IsUnknown() {
		diags.Append(plan.Tags.ElementsAs(ctx, &search.Tags, false)...)
	}
	for _, filter := range plan.Filters {
		search.Filters = append(search.Filters, client.SavedSearchFilter{
			Type: filter.Type.ValueString(), Condition: filter.Condition.ValueString(),
		})
	}
	return search
}

func flattenSavedSearch(ctx context.Context, search *client.SavedSearch, diags *diag.Diagnostics) savedSearchResourceModel {
	whereLanguage := search.WhereLanguage
	if whereLanguage == "" {
		whereLanguage = "lucene"
	}
	state := savedSearchResourceModel{
		ID:            types.StringValue(search.ID),
		Name:          types.StringValue(search.Name),
		SourceID:      types.StringValue(search.SourceID),
		Select:        stringValueOrNull(search.Select),
		Where:         stringValueOrNull(search.Where),
		WhereLanguage: types.StringValue(whereLanguage),
		OrderBy:       stringValueOrNull(search.OrderBy),
	}
	if len(search.Tags) == 0 {
		state.Tags = types.ListNull(types.StringType)
	} else {
		var tagDiags diag.Diagnostics
		state.Tags, tagDiags = types.ListValueFrom(ctx, types.StringType, search.Tags)
		diags.Append(tagDiags...)
	}
	for _, filter := range search.Filters {
		filterType := filter.Type
		if filterType == "" {
			filterType = "sql"
		}
		state.Filters = append(state.Filters, savedSearchFilterModel{
			Type: types.StringValue(filterType), Condition: types.StringValue(filter.Condition),
		})
	}
	return state
}

func stringValueOrNull(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}
