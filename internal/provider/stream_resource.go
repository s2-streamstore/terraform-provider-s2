package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/s2-streamstore/s2-sdk-go/s2"
)

var (
	_ resource.Resource                = &StreamResource{}
	_ resource.ResourceWithConfigure   = &StreamResource{}
	_ resource.ResourceWithImportState = &StreamResource{}
)

type StreamResource struct {
	client *s2.Client
}

type StreamResourceModel struct {
	Basin           types.String `tfsdk:"basin"`
	Name            types.String `tfsdk:"name"`
	CreatedAt       types.String `tfsdk:"created_at"`
	Cipher          types.String `tfsdk:"cipher"`
	StorageClass    types.String `tfsdk:"storage_class"`
	RetentionPolicy types.Object `tfsdk:"retention_policy"`
	Timestamping    types.Object `tfsdk:"timestamping"`
	DeleteOnEmpty   types.Object `tfsdk:"delete_on_empty"`
}

func NewStreamResource() resource.Resource {
	return &StreamResource{}
}

func (r *StreamResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_stream"
}

func (r *StreamResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an S2 stream in a basin.",
		Attributes: map[string]schema.Attribute{
			"basin": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: basinNameValidators(),
			},
			"name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: streamNameValidators(),
			},
			"created_at": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cipher": schema.StringAttribute{
				Computed: true,
			},
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
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"age": schema.Int64Attribute{
						Optional: true,
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
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
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
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
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
	}
}

func (r *StreamResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *StreamResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan StreamResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	streamConfig, configDiags := expandStreamConfig(StreamConfigModel{
		StorageClass:    plan.StorageClass,
		RetentionPolicy: plan.RetentionPolicy,
		Timestamping:    plan.Timestamping,
		DeleteOnEmpty:   plan.DeleteOnEmpty,
	})
	resp.Diagnostics.Append(configDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ensureResp, err := r.client.Basin(plan.Basin.ValueString()).Streams.Ensure(ctx, s2.EnsureStreamArgs{
		Stream: s2.StreamName(plan.Name.ValueString()),
		Config: streamConfig,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to Create Stream", err.Error())
		return
	}

	cfg, err := r.client.Basin(plan.Basin.ValueString()).Streams.GetConfig(ctx, s2.StreamName(plan.Name.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Failed Reading Stream Configuration", err.Error())
		return
	}

	state := flattenStreamModelFromAPI(plan.Basin.ValueString(), plan.Name.ValueString(), ensureResp.Stream, cfg)
	if plan.RetentionPolicy.IsNull() {
		state.RetentionPolicy = plan.RetentionPolicy
	}
	if plan.Timestamping.IsNull() {
		state.Timestamping = plan.Timestamping
	}
	if plan.DeleteOnEmpty.IsNull() {
		state.DeleteOnEmpty = plan.DeleteOnEmpty
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *StreamResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state StreamResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.Basin(state.Basin.ValueString()).Streams.GetConfig(ctx, s2.StreamName(state.Name.ValueString()))
	if err != nil {
		if isNotFound(err) || isStreamDeletionPending(err) || isBasinDeletionPending(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed Reading Stream", err.Error())
		return
	}

	streamInfo, found, err := findStreamByName(ctx, r.client, state.Basin.ValueString(), s2.StreamName(state.Name.ValueString()), true)
	if err != nil {
		resp.Diagnostics.AddError("Failed Listing Stream", err.Error())
		return
	}

	createdAt := state.CreatedAt.ValueString()
	cipher := state.Cipher
	if found {
		createdAt = streamInfo.CreatedAt.Format(time.RFC3339Nano)
		cipher = cipherValue(streamInfo.Cipher)
	}

	newState := flattenStreamModelFromAPI(state.Basin.ValueString(), state.Name.ValueString(), streamInfo, cfg)
	newState.CreatedAt = types.StringValue(createdAt)
	newState.Cipher = cipher
	if state.RetentionPolicy.IsNull() {
		newState.RetentionPolicy = state.RetentionPolicy
	}
	if state.Timestamping.IsNull() {
		newState.Timestamping = state.Timestamping
	}
	if state.DeleteOnEmpty.IsNull() {
		newState.DeleteOnEmpty = state.DeleteOnEmpty
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *StreamResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan StreamResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var prior StreamResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	priorConfig := StreamConfigModel{
		StorageClass:    prior.StorageClass,
		RetentionPolicy: prior.RetentionPolicy,
		Timestamping:    prior.Timestamping,
		DeleteOnEmpty:   prior.DeleteOnEmpty,
	}
	reconfigure, reconfigureDiags := expandStreamReconfigurationWithPrior(StreamConfigModel{
		StorageClass:    plan.StorageClass,
		RetentionPolicy: plan.RetentionPolicy,
		Timestamping:    plan.Timestamping,
		DeleteOnEmpty:   plan.DeleteOnEmpty,
	}, &priorConfig)
	resp.Diagnostics.Append(reconfigureDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.Basin(plan.Basin.ValueString()).Streams.Reconfigure(ctx, s2.ReconfigureStreamArgs{
		Stream: s2.StreamName(plan.Name.ValueString()),
		Config: reconfigure,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed Updating Stream", err.Error())
		return
	}

	createdAt := prior.CreatedAt.ValueString()
	cipher := prior.Cipher
	streamInfo, found, err := findStreamByName(ctx, r.client, plan.Basin.ValueString(), s2.StreamName(plan.Name.ValueString()), true)
	if err == nil && found {
		createdAt = streamInfo.CreatedAt.Format(time.RFC3339Nano)
		cipher = cipherValue(streamInfo.Cipher)
	}

	newState := flattenStreamModelFromAPI(plan.Basin.ValueString(), plan.Name.ValueString(), s2.StreamInfo{}, cfg)
	newState.CreatedAt = types.StringValue(createdAt)
	newState.Cipher = cipher
	if plan.RetentionPolicy.IsNull() {
		newState.RetentionPolicy = plan.RetentionPolicy
	}
	if plan.Timestamping.IsNull() {
		newState.Timestamping = plan.Timestamping
	}
	if plan.DeleteOnEmpty.IsNull() {
		newState.DeleteOnEmpty = plan.DeleteOnEmpty
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *StreamResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state StreamResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteDeadline := time.Now().Add(streamDeleteTimeout)
	deleteBackoff := initialPollBackoff
	streamName := s2.StreamName(state.Name.ValueString())

	for {
		err := r.client.Basin(state.Basin.ValueString()).Streams.Delete(ctx, streamName)
		if err == nil || isNotFound(err) || isStreamDeletionPending(err) || isBasinDeletionPending(err) {
			break
		}
		if isTransactionConflict(err) || isUnavailable(err) {
			if sleepErr := sleepWithBackoff(ctx, deleteDeadline, &deleteBackoff); sleepErr != nil {
				resp.Diagnostics.AddError(
					"Failed Deleting Stream",
					fmt.Sprintf("Timed out waiting to initiate deletion for stream %q in basin %q: %v", state.Name.ValueString(), state.Basin.ValueString(), err),
				)
				return
			}
			continue
		}

		resp.Diagnostics.AddError("Failed Deleting Stream", err.Error())
		return
	}

	if err := waitForStreamDeletion(ctx, r.client, state.Basin.ValueString(), streamName, streamDeleteTimeout); err != nil {
		resp.Diagnostics.AddError("Failed Waiting for Stream Deletion", err.Error())
	}
}

func (r *StreamResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Expected import ID format: {basin}/{stream}",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("basin"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[1])...)
}

func flattenStreamModelFromAPI(basin string, name string, info s2.StreamInfo, cfg *s2.StreamConfig) StreamResourceModel {
	state := StreamResourceModel{
		Basin:           types.StringValue(basin),
		Name:            types.StringValue(name),
		CreatedAt:       types.StringNull(),
		Cipher:          types.StringNull(),
		StorageClass:    types.StringNull(),
		RetentionPolicy: nullRetentionPolicyObject(),
		Timestamping:    nullTimestampingObject(),
		DeleteOnEmpty:   nullDeleteOnEmptyObject(),
	}

	if !info.CreatedAt.IsZero() {
		state.CreatedAt = types.StringValue(info.CreatedAt.Format(time.RFC3339Nano))
	}
	state.Cipher = cipherValue(info.Cipher)

	if cfg == nil {
		return state
	}

	flattenedCfg := flattenStreamConfig(cfg)
	state.StorageClass = flattenedCfg.StorageClass
	state.RetentionPolicy = flattenedCfg.RetentionPolicy
	state.Timestamping = flattenedCfg.Timestamping
	state.DeleteOnEmpty = flattenedCfg.DeleteOnEmpty

	return state
}
