// Copyright (c) Lapse Technologies, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/teamlapse/terraform-provider-clickstack/internal/client"
)

var (
	_ resource.Resource                     = &AlertResource{}
	_ resource.ResourceWithConfigValidators = &AlertResource{}
	_ resource.ResourceWithImportState      = &AlertResource{}
)

type AlertResource struct {
	client *client.Client
}

type alertResourceModel struct {
	ID                    types.String  `tfsdk:"id"`
	Name                  types.String  `tfsdk:"name"`
	Message               types.String  `tfsdk:"message"`
	Source                types.String  `tfsdk:"source"`
	Threshold             types.Float64 `tfsdk:"threshold"`
	ThresholdType         types.String  `tfsdk:"threshold_type"`
	Interval              types.String  `tfsdk:"interval"`
	DashboardID           types.String  `tfsdk:"dashboard_id"`
	TileID                types.String  `tfsdk:"tile_id"`
	SavedSearchID         types.String  `tfsdk:"saved_search_id"`
	GroupBy               types.String  `tfsdk:"group_by"`
	ScheduleOffsetMinutes types.Int64   `tfsdk:"schedule_offset_minutes"`
	ScheduleStartAt       types.String  `tfsdk:"schedule_start_at"`
	State                 types.String  `tfsdk:"state"`
	Channel               *channelModel `tfsdk:"channel"`
}

type channelModel struct {
	Type            types.String `tfsdk:"type"`
	WebhookID       types.String `tfsdk:"webhook_id"`
	WebhookService  types.String `tfsdk:"webhook_service"`
	SlackChannelID  types.String `tfsdk:"slack_channel_id"`
	Severity        types.String `tfsdk:"severity"`
	EmailRecipients types.List   `tfsdk:"email_recipients"`
}

func NewAlertResource() resource.Resource {
	return &AlertResource{}
}

func (r *AlertResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert"
}

func (r *AlertResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a ClickStack alert.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Alert ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Description: "Alert name.",
			},
			"message": schema.StringAttribute{
				Optional:    true,
				Description: "Alert message included in notifications.",
			},
			"source": schema.StringAttribute{
				Required:    true,
				Description: "Alert source type: 'tile' or 'saved_search'.",
				Validators: []validator.String{
					stringvalidator.OneOf("tile", "saved_search"),
				},
			},
			"threshold": schema.Float64Attribute{
				Required:    true,
				Description: "Threshold value that triggers the alert.",
			},
			"threshold_type": schema.StringAttribute{
				Required:    true,
				Description: "Threshold direction: 'above' or 'below'.",
				Validators: []validator.String{
					stringvalidator.OneOf("above", "below"),
				},
			},
			"interval": schema.StringAttribute{
				Required:    true,
				Description: "Evaluation interval: '1m', '5m', '15m', '30m', '1h', '6h', '12h', '1d'.",
				Validators: []validator.String{
					stringvalidator.OneOf("1m", "5m", "15m", "30m", "1h", "6h", "12h", "1d"),
				},
			},
			"dashboard_id": schema.StringAttribute{
				Optional:    true,
				Description: "Dashboard ID (required when source is 'tile').",
			},
			"tile_id": schema.StringAttribute{
				Optional:    true,
				Description: "Tile ID (required when source is 'tile').",
			},
			"saved_search_id": schema.StringAttribute{
				Optional:    true,
				Description: "Saved search ID (required when source is 'saved_search').",
			},
			"group_by": schema.StringAttribute{
				Optional:    true,
				Description: "Field to group alert evaluations by.",
			},
			"schedule_offset_minutes": schema.Int64Attribute{
				Optional:    true,
				Description: "Offset in minutes from the interval boundary.",
			},
			"schedule_start_at": schema.StringAttribute{
				Optional:    true,
				Description: "Absolute UTC start time anchor (ISO 8601).",
			},
			"state": schema.StringAttribute{
				Computed:    true,
				Description: "Alert state: ALERT, OK, INSUFFICIENT_DATA, or DISABLED.",
			},
		},
		Blocks: map[string]schema.Block{
			"channel": schema.SingleNestedBlock{
				Description: "Notification channel.",
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Required:    true,
						Description: "Channel type: 'webhook' or 'email'.",
						Validators: []validator.String{
							stringvalidator.OneOf("webhook", "email"),
						},
					},
					"webhook_id": schema.StringAttribute{
						Optional:    true,
						Description: "Webhook ID (required when type is 'webhook').",
					},
					"webhook_service": schema.StringAttribute{
						Optional:    true,
						Description: "Webhook service type (e.g. 'slack', 'slack_api', 'pagerduty_api').",
					},
					"slack_channel_id": schema.StringAttribute{
						Optional:    true,
						Description: "Slack channel ID for Slack webhook alerts.",
					},
					"severity": schema.StringAttribute{
						Optional:    true,
						Description: "Severity for PagerDuty: 'critical', 'error', 'warning', 'info'.",
						Validators: []validator.String{
							stringvalidator.OneOf("critical", "error", "warning", "info"),
						},
					},
					"email_recipients": schema.ListAttribute{
						Optional:    true,
						Description: "Email recipients (required when type is 'email').",
						ElementType: types.StringType,
					},
				},
			},
		},
	}
}

func (r *AlertResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		alertConfigValidator{},
	}
}

func (r *AlertResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AlertResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan alertResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	alert := expandAlert(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateAlert(ctx, alert)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create alert", err.Error())
		return
	}

	state := flattenAlert(ctx, created, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AlertResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state alertResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	alert, err := r.client.GetAlert(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read alert", err.Error())
		return
	}

	newState := flattenAlert(ctx, alert, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *AlertResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan alertResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state alertResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	alert := expandAlert(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateAlert(ctx, state.ID.ValueString(), alert)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update alert", err.Error())
		return
	}

	newState := flattenAlert(ctx, updated, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *AlertResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state alertResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteAlert(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Unable to delete alert", err.Error())
	}
}

func (r *AlertResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type alertConfigValidator struct{}

func (v alertConfigValidator) Description(context.Context) string {
	return "validates required alert fields for the selected source and channel configuration"
}

func (v alertConfigValidator) MarkdownDescription(context.Context) string {
	return v.Description(context.Background())
}

func (v alertConfigValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config alertResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Channel == nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("channel"),
			"Missing channel block",
			"The channel block is required.",
		)
	}

	if config.Source.IsNull() || config.Source.IsUnknown() {
		return
	}

	switch config.Source.ValueString() {
	case "tile":
		if config.DashboardID.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("dashboard_id"),
				"Missing dashboard_id for tile alert",
				"Set dashboard_id when source = \"tile\".",
			)
		}
		if config.TileID.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("tile_id"),
				"Missing tile_id for tile alert",
				"Set tile_id when source = \"tile\".",
			)
		}
		if !config.SavedSearchID.IsNull() && !config.SavedSearchID.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("saved_search_id"),
				"Invalid saved_search_id for tile alert",
				"Do not set saved_search_id when source = \"tile\".",
			)
		}
	case "saved_search":
		if config.SavedSearchID.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("saved_search_id"),
				"Missing saved_search_id for saved_search alert",
				"Set saved_search_id when source = \"saved_search\".",
			)
		}
		if !config.DashboardID.IsNull() && !config.DashboardID.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("dashboard_id"),
				"Invalid dashboard_id for saved_search alert",
				"Do not set dashboard_id when source = \"saved_search\".",
			)
		}
		if !config.TileID.IsNull() && !config.TileID.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("tile_id"),
				"Invalid tile_id for saved_search alert",
				"Do not set tile_id when source = \"saved_search\".",
			)
		}
	}
}

func expandAlert(ctx context.Context, plan alertResourceModel, diags *diag.Diagnostics) client.Alert {
	a := client.Alert{
		Source:        plan.Source.ValueString(),
		Threshold:     plan.Threshold.ValueFloat64(),
		ThresholdType: plan.ThresholdType.ValueString(),
		Interval:      plan.Interval.ValueString(),
	}

	if !plan.Name.IsNull() {
		v := plan.Name.ValueString()
		a.Name = &v
	}
	if !plan.Message.IsNull() {
		v := plan.Message.ValueString()
		a.Message = &v
	}
	if !plan.DashboardID.IsNull() {
		v := plan.DashboardID.ValueString()
		a.DashboardID = &v
	}
	if !plan.TileID.IsNull() {
		v := plan.TileID.ValueString()
		a.TileID = &v
	}
	if !plan.SavedSearchID.IsNull() {
		v := plan.SavedSearchID.ValueString()
		a.SavedSearchID = &v
	}
	if !plan.GroupBy.IsNull() {
		v := plan.GroupBy.ValueString()
		a.GroupBy = &v
	}
	if !plan.ScheduleOffsetMinutes.IsNull() {
		v := int(plan.ScheduleOffsetMinutes.ValueInt64())
		a.ScheduleOffsetMinutes = &v
	}
	if !plan.ScheduleStartAt.IsNull() {
		v := plan.ScheduleStartAt.ValueString()
		a.ScheduleStartAt = &v
	}

	if plan.Channel == nil {
		diags.AddAttributeError(
			path.Root("channel"),
			"Missing channel block",
			"The channel block is required.",
		)
		return a
	}

	ch := plan.Channel
	a.Channel = client.AlertChannel{
		Type: ch.Type.ValueString(),
	}
	if !ch.WebhookID.IsNull() {
		v := ch.WebhookID.ValueString()
		a.Channel.WebhookID = &v
	}
	if !ch.WebhookService.IsNull() {
		v := ch.WebhookService.ValueString()
		a.Channel.WebhookService = &v
	}
	if !ch.SlackChannelID.IsNull() {
		v := ch.SlackChannelID.ValueString()
		a.Channel.SlackChannelID = &v
	}
	if !ch.Severity.IsNull() {
		v := ch.Severity.ValueString()
		a.Channel.Severity = &v
	}
	if !ch.EmailRecipients.IsNull() {
		var recipients []string
		diags.Append(ch.EmailRecipients.ElementsAs(ctx, &recipients, false)...)
		a.Channel.EmailRecipients = recipients
	}

	return a
}

func flattenAlert(ctx context.Context, a *client.Alert, diags *diag.Diagnostics) alertResourceModel {
	model := alertResourceModel{
		ID:            types.StringValue(a.ID),
		Source:        types.StringValue(a.Source),
		Threshold:     types.Float64Value(a.Threshold),
		ThresholdType: types.StringValue(a.ThresholdType),
		Interval:      types.StringValue(a.Interval),
		State:         types.StringValue(a.State),
	}

	if a.Name != nil {
		model.Name = types.StringValue(*a.Name)
	} else {
		model.Name = types.StringNull()
	}
	if a.Message != nil {
		model.Message = types.StringValue(*a.Message)
	} else {
		model.Message = types.StringNull()
	}
	if a.DashboardID != nil {
		model.DashboardID = types.StringValue(*a.DashboardID)
	} else {
		model.DashboardID = types.StringNull()
	}
	if a.TileID != nil {
		model.TileID = types.StringValue(*a.TileID)
	} else {
		model.TileID = types.StringNull()
	}
	if a.SavedSearchID != nil {
		model.SavedSearchID = types.StringValue(*a.SavedSearchID)
	} else {
		model.SavedSearchID = types.StringNull()
	}
	if a.GroupBy != nil {
		model.GroupBy = types.StringValue(*a.GroupBy)
	} else {
		model.GroupBy = types.StringNull()
	}
	if a.ScheduleOffsetMinutes != nil {
		model.ScheduleOffsetMinutes = types.Int64Value(int64(*a.ScheduleOffsetMinutes))
	} else {
		model.ScheduleOffsetMinutes = types.Int64Null()
	}
	if a.ScheduleStartAt != nil {
		model.ScheduleStartAt = types.StringValue(*a.ScheduleStartAt)
	} else {
		model.ScheduleStartAt = types.StringNull()
	}

	ch := channelModel{
		Type: types.StringValue(a.Channel.Type),
	}
	if a.Channel.WebhookID != nil {
		ch.WebhookID = types.StringValue(*a.Channel.WebhookID)
	} else {
		ch.WebhookID = types.StringNull()
	}
	if a.Channel.WebhookService != nil {
		ch.WebhookService = types.StringValue(*a.Channel.WebhookService)
	} else {
		ch.WebhookService = types.StringNull()
	}
	if a.Channel.SlackChannelID != nil {
		ch.SlackChannelID = types.StringValue(*a.Channel.SlackChannelID)
	} else {
		ch.SlackChannelID = types.StringNull()
	}
	if a.Channel.Severity != nil {
		ch.Severity = types.StringValue(*a.Channel.Severity)
	} else {
		ch.Severity = types.StringNull()
	}
	if len(a.Channel.EmailRecipients) > 0 {
		recipientList, d := types.ListValueFrom(ctx, types.StringType, a.Channel.EmailRecipients)
		diags.Append(d...)
		ch.EmailRecipients = recipientList
	} else {
		ch.EmailRecipients = types.ListNull(types.StringType)
	}
	model.Channel = &ch

	return model
}
