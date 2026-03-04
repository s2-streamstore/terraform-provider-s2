package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/s2-streamstore/s2-sdk-go/s2"
)

var (
	_ resource.Resource                = &BasinResource{}
	_ resource.ResourceWithConfigure   = &BasinResource{}
	_ resource.ResourceWithImportState = &BasinResource{}
)

type BasinResource struct {
	client *s2.Client
}

type BasinResourceModel struct {
	Name                 types.String `tfsdk:"name"`
	Scope                types.String `tfsdk:"scope"`
	State                types.String `tfsdk:"state"`
	CreateStreamOnAppend types.Bool   `tfsdk:"create_stream_on_append"`
	CreateStreamOnRead   types.Bool   `tfsdk:"create_stream_on_read"`
	DefaultStreamConfig  types.Object `tfsdk:"default_stream_config"`
}

func NewBasinResource() resource.Resource {
	return &BasinResource{}
}

func (r *BasinResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_basin"
}

func (r *BasinResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an S2 basin.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: basinNameValidators(),
			},
			"scope": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(defaultBasinScope),
				},
			},
			"state": schema.StringAttribute{
				Computed: true,
			},
			"create_stream_on_append": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"create_stream_on_read": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"default_stream_config": schema.SingleNestedBlock{
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"storage_class": schema.StringAttribute{
						Optional: true,
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
						Validators: []validator.String{
							stringvalidator.OneOf(string(s2.StorageClassExpress), string(s2.StorageClassStandard)),
						},
					},
				},
				Blocks: map[string]schema.Block{
					"retention_policy": schema.SingleNestedBlock{
						Attributes: map[string]schema.Attribute{
							"age": schema.Int64Attribute{
								Optional: true,
								PlanModifiers: []planmodifier.Int64{
									int64planmodifier.UseStateForUnknown(),
								},
								Validators: []validator.Int64{
									int64validator.AtLeast(1),
									int64validator.ConflictsWith(path.MatchRelative().AtParent().AtName("infinite")),
								},
							},
							"infinite": schema.BoolAttribute{
								Optional: true,
								Validators: []validator.Bool{
									boolvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("age")),
								},
							},
						},
					},
					"timestamping": schema.SingleNestedBlock{
						Attributes: map[string]schema.Attribute{
							"mode": schema.StringAttribute{
								Optional: true,
								Computed: true,
								Validators: []validator.String{
									stringvalidator.OneOf(string(s2.TimestampingModeClientPrefer), string(s2.TimestampingModeClientRequire), string(s2.TimestampingModeArrival)),
								},
							},
							"uncapped": schema.BoolAttribute{
								Optional: true,
								Computed: true,
							},
						},
					},
					"delete_on_empty": schema.SingleNestedBlock{
						Attributes: map[string]schema.Attribute{
							"min_age_secs": schema.Int64Attribute{
								Optional: true,
								Computed: true,
								Validators: []validator.Int64{
									int64validator.AtLeast(0),
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *BasinResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*s2.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Configuration Type",
			fmt.Sprintf("Expected *s2.Client, got %T. This is a provider implementation bug.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *BasinResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BasinResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scope := plan.Scope
	if scope.IsNull() || scope.IsUnknown() {
		scope = types.StringValue(defaultBasinScope)
	}

	createArgs := s2.CreateBasinArgs{
		Basin: s2.BasinName(plan.Name.ValueString()),
		Scope: s2.Ptr(s2.BasinScope(scope.ValueString())),
	}

	basinConfig := &s2.BasinConfig{}
	setConfig := false

	if !plan.CreateStreamOnAppend.IsNull() && !plan.CreateStreamOnAppend.IsUnknown() {
		value := plan.CreateStreamOnAppend.ValueBool()
		basinConfig.CreateStreamOnAppend = &value
		setConfig = true
	}
	if !plan.CreateStreamOnRead.IsNull() && !plan.CreateStreamOnRead.IsUnknown() {
		value := plan.CreateStreamOnRead.ValueBool()
		basinConfig.CreateStreamOnRead = &value
		setConfig = true
	}
	if !plan.DefaultStreamConfig.IsNull() && !plan.DefaultStreamConfig.IsUnknown() {
		var cfgModel StreamConfigModel
		cfgDiags := plan.DefaultStreamConfig.As(ctx, &cfgModel, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(cfgDiags...)
		if resp.Diagnostics.HasError() {
			return
		}

		streamConfig, streamConfigDiags := expandStreamConfig(cfgModel)
		resp.Diagnostics.Append(streamConfigDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if streamConfig != nil {
			basinConfig.DefaultStreamConfig = streamConfig
			setConfig = true
		}
	}
	if setConfig {
		createArgs.Config = basinConfig
	}

	basinInfo, err := r.client.Basins.Create(ctx, createArgs)
	if err != nil {
		if isResourceAlreadyExists(err) {
			resp.Diagnostics.AddError("Basin Already Exists", fmt.Sprintf("Basin %q already exists.", plan.Name.ValueString()))
			return
		}
		if isBasinDeletionPending(err) {
			resp.Diagnostics.AddError(
				"Basin Deletion Pending",
				fmt.Sprintf("Basin %q is currently being deleted. Wait for deletion to complete before creating it again.", plan.Name.ValueString()),
			)
			return
		}
		if isTransactionConflict(err) {
			resp.Diagnostics.AddError(
				"Basin Operation Conflict",
				fmt.Sprintf("Another operation is in progress for basin %q. Retry this apply.", plan.Name.ValueString()),
			)
			return
		}
		resp.Diagnostics.AddError("Failed to Create Basin", err.Error())
		return
	}

	if basinInfo.State == s2.BasinStateCreating {
		if err := waitForBasinState(ctx, r.client, basinInfo.Name, s2.BasinStateActive, basinAsyncTimeout); err != nil {
			resp.Diagnostics.AddError("Failed Waiting for Basin Creation", err.Error())
			return
		}
		refreshedInfo, found, err := findBasinByName(ctx, r.client, basinInfo.Name, false)
		if err != nil {
			resp.Diagnostics.AddError("Failed Fetching Basin", err.Error())
			return
		}
		if found {
			basinInfo = &refreshedInfo
		}
	}

	config, err := r.client.Basins.GetConfig(ctx, basinInfo.Name)
	if err != nil {
		resp.Diagnostics.AddError("Failed Reading Basin Configuration", err.Error())
		return
	}

	state := flattenBasinModelFromAPI(ctx, *basinInfo, config)
	state.Name = plan.Name
	if state.Scope.IsNull() || state.Scope.IsUnknown() {
		state.Scope = scope
	}
	if plan.DefaultStreamConfig.IsNull() {
		state.DefaultStreamConfig = plan.DefaultStreamConfig
	} else {
		state.DefaultStreamConfig = applyDefaultStreamConfigNullOverrides(ctx, plan.DefaultStreamConfig, state.DefaultStreamConfig)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *BasinResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BasinResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := s2.BasinName(state.Name.ValueString())
	config, err := r.client.Basins.GetConfig(ctx, name)
	if err != nil {
		if isNotFound(err) || isBasinDeletionPending(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		if isUnavailable(err) {
			basinInfo, found, listErr := findBasinByName(ctx, r.client, name, true)
			if listErr != nil {
				resp.Diagnostics.AddError("Failed Listing Basin", listErr.Error())
				return
			}
			if !found {
				resp.State.RemoveResource(ctx)
				return
			}

			newState := state
			newState.State = types.StringValue(string(basinInfo.State))
			if newState.Scope.IsNull() || newState.Scope.IsUnknown() {
				newState.Scope = types.StringValue(string(basinInfo.Scope))
			}
			resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
			return
		}
		resp.Diagnostics.AddError("Failed Reading Basin", err.Error())
		return
	}

	basinInfo, found, err := findBasinByName(ctx, r.client, name, true)
	if err != nil {
		resp.Diagnostics.AddError("Failed Listing Basin", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	newState := flattenBasinModelFromAPI(ctx, basinInfo, config)
	newState.Name = state.Name
	if newState.Scope.IsNull() {
		newState.Scope = state.Scope
	}
	if state.DefaultStreamConfig.IsNull() && isDefaultStreamConfig(config.DefaultStreamConfig) {
		newState.DefaultStreamConfig = state.DefaultStreamConfig
	} else {
		newState.DefaultStreamConfig = applyDefaultStreamConfigNullOverrides(ctx, state.DefaultStreamConfig, newState.DefaultStreamConfig)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *BasinResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan BasinResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	reconfigure := s2.BasinReconfiguration{}
	if !plan.CreateStreamOnAppend.IsNull() && !plan.CreateStreamOnAppend.IsUnknown() {
		value := plan.CreateStreamOnAppend.ValueBool()
		reconfigure.CreateStreamOnAppend = &value
	}
	if !plan.CreateStreamOnRead.IsNull() && !plan.CreateStreamOnRead.IsUnknown() {
		value := plan.CreateStreamOnRead.ValueBool()
		reconfigure.CreateStreamOnRead = &value
	}
	if !plan.DefaultStreamConfig.IsNull() && !plan.DefaultStreamConfig.IsUnknown() {
		var cfgModel StreamConfigModel
		cfgDiags := plan.DefaultStreamConfig.As(ctx, &cfgModel, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(cfgDiags...)
		if resp.Diagnostics.HasError() {
			return
		}

		reconfiguredStream, reconfigureDiags := expandStreamReconfiguration(cfgModel)
		resp.Diagnostics.Append(reconfigureDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		reconfigure.DefaultStreamConfig = &reconfiguredStream
	}

	config, err := r.client.Basins.Reconfigure(ctx, s2.ReconfigureBasinArgs{
		Basin:  s2.BasinName(plan.Name.ValueString()),
		Config: reconfigure,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed Updating Basin", err.Error())
		return
	}

	basinInfo, found, err := findBasinByName(ctx, r.client, s2.BasinName(plan.Name.ValueString()), true)
	if err != nil {
		resp.Diagnostics.AddError("Failed Listing Basin", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("Basin Not Found After Update", fmt.Sprintf("Basin %q no longer exists.", plan.Name.ValueString()))
		return
	}

	state := flattenBasinModelFromAPI(ctx, basinInfo, config)
	state.Name = plan.Name
	if state.Scope.IsNull() || state.Scope.IsUnknown() {
		state.Scope = plan.Scope
	}
	if plan.DefaultStreamConfig.IsNull() {
		state.DefaultStreamConfig = plan.DefaultStreamConfig
	} else {
		state.DefaultStreamConfig = applyDefaultStreamConfigNullOverrides(ctx, plan.DefaultStreamConfig, state.DefaultStreamConfig)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *BasinResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BasinResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	basinName := s2.BasinName(state.Name.ValueString())
	deleteDeadline := time.Now().Add(basinAsyncTimeout)
	deleteBackoff := initialPollBackoff

	for {
		err := r.client.Basins.Delete(ctx, basinName)
		if err == nil || isNotFound(err) || isBasinDeletionPending(err) {
			break
		}
		if isTransactionConflict(err) || isUnavailable(err) {
			if sleepErr := sleepWithBackoff(ctx, deleteDeadline, &deleteBackoff); sleepErr != nil {
				resp.Diagnostics.AddError(
					"Failed Deleting Basin",
					fmt.Sprintf("Timed out waiting to initiate deletion for basin %q: %v", state.Name.ValueString(), err),
				)
				return
			}
			continue
		}

		resp.Diagnostics.AddError("Failed Deleting Basin", err.Error())
		return
	}

	if err := waitForBasinDeletion(ctx, r.client, basinName, basinAsyncTimeout); err != nil {
		resp.Diagnostics.AddError("Failed Waiting for Basin Deletion", err.Error())
	}
}

func (r *BasinResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

func flattenBasinModelFromAPI(ctx context.Context, info s2.BasinInfo, config *s2.BasinConfig) BasinResourceModel {
	state := BasinResourceModel{
		Name:                 types.StringValue(string(info.Name)),
		Scope:                types.StringValue(string(info.Scope)),
		State:                types.StringValue(string(info.State)),
		CreateStreamOnAppend: types.BoolNull(),
		CreateStreamOnRead:   types.BoolNull(),
		DefaultStreamConfig:  nullStreamConfigObject(),
	}

	if config == nil {
		return state
	}
	if config.CreateStreamOnAppend != nil {
		state.CreateStreamOnAppend = types.BoolValue(*config.CreateStreamOnAppend)
	}
	if config.CreateStreamOnRead != nil {
		state.CreateStreamOnRead = types.BoolValue(*config.CreateStreamOnRead)
	}

	flattenedStreamConfig := flattenStreamConfig(config.DefaultStreamConfig)
	obj, diags := types.ObjectValueFrom(ctx, streamConfigAttrTypes(), flattenedStreamConfig)
	if !diags.HasError() {
		state.DefaultStreamConfig = obj
	}

	return state
}
