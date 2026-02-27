package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/s2-streamstore/s2-sdk-go/s2"
)

var (
	_ resource.Resource                   = &AccessTokenResource{}
	_ resource.ResourceWithConfigure      = &AccessTokenResource{}
	_ resource.ResourceWithImportState    = &AccessTokenResource{}
	_ resource.ResourceWithValidateConfig = &AccessTokenResource{}
)

type AccessTokenResource struct {
	client *s2.Client
}

type AccessTokenResourceModel struct {
	TokenID           types.String `tfsdk:"token_id"`
	ID                types.String `tfsdk:"id"`
	AutoPrefixStreams types.Bool   `tfsdk:"auto_prefix_streams"`
	ExpiresAt         types.String `tfsdk:"expires_at"`
	AccessToken       types.String `tfsdk:"access_token"`
	Scope             types.Object `tfsdk:"scope"`
}

func NewAccessTokenResource() resource.Resource {
	return &AccessTokenResource{}
}

func (r *AccessTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_token"
}

func (r *AccessTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Issues and revokes S2 access tokens.",
		Attributes: map[string]schema.Attribute{
			"token_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.UTF8LengthBetween(1, 96),
				},
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Terraform resource identifier, equal to token_id.",
			},
			"auto_prefix_streams": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"expires_at": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "RFC3339 timestamp when the token expires, for example 2030-01-01T00:00:00Z.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					rfc3339TimestampString(),
				},
			},
			"access_token": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
			},
			"scope": schema.SingleNestedAttribute{
				Required: true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"ops": schema.SetAttribute{
						Optional:    true,
						ElementType: types.StringType,
						PlanModifiers: []planmodifier.Set{
							setplanmodifier.RequiresReplace(),
						},
						Validators: []validator.Set{
							setvalidator.ValueStringsAre(stringvalidator.OneOf(accessTokenOperations()...)),
						},
					},
					"basins":        resourceSetAttributeSchema(),
					"streams":       resourceSetAttributeSchema(),
					"access_tokens": resourceSetAttributeSchema(),
					"op_groups": schema.SingleNestedAttribute{
						Optional: true,
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.RequiresReplace(),
						},
						Attributes: map[string]schema.Attribute{
							"account_read":  schema.BoolAttribute{Optional: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}},
							"account_write": schema.BoolAttribute{Optional: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}},
							"basin_read":    schema.BoolAttribute{Optional: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}},
							"basin_write":   schema.BoolAttribute{Optional: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}},
							"stream_read":   schema.BoolAttribute{Optional: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}},
							"stream_write":  schema.BoolAttribute{Optional: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}},
						},
					},
				},
			},
		},
	}
}

func resourceSetAttributeSchema() schema.SingleNestedAttribute {
	stringPlanModifiers := []planmodifier.String{}
	stringPlanModifiers = append(stringPlanModifiers, stringplanmodifier.RequiresReplace())

	return schema.SingleNestedAttribute{
		Optional: true,
		PlanModifiers: []planmodifier.Object{
			objectplanmodifier.RequiresReplace(),
		},
		Attributes: map[string]schema.Attribute{
			"exact": schema.StringAttribute{
				Optional:      true,
				PlanModifiers: stringPlanModifiers,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("prefix")),
				},
			},
			"prefix": schema.StringAttribute{
				Optional:      true,
				PlanModifiers: stringPlanModifiers,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("exact")),
				},
			},
		},
	}
}

func (r *AccessTokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AccessTokenResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config AccessTokenResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Scope.IsNull() || config.Scope.IsUnknown() {
		return
	}

	var scope AccessTokenScopeModel
	resp.Diagnostics.Append(config.Scope.As(ctx, &scope, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasKnownOps := !scope.Ops.IsUnknown()
	hasOps := false
	if !scope.Ops.IsNull() && hasKnownOps {
		hasOps = len(scope.Ops.Elements()) > 0
	}

	hasKnownOpGroups := !scope.OpGroups.IsUnknown()
	hasOpGroupPermissions := false
	opGroupPermissionsUnknown := false
	if !scope.OpGroups.IsNull() && hasKnownOpGroups {
		var groups OpGroupsModel
		resp.Diagnostics.Append(scope.OpGroups.As(ctx, &groups, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		hasOpGroupPermissions = opGroupsHasAnyPermission(groups)
		opGroupPermissionsUnknown = opGroupsHasUnknownPermission(groups)
	}

	if hasKnownOps && hasKnownOpGroups && !hasOps && !hasOpGroupPermissions && !opGroupPermissionsUnknown {
		resp.Diagnostics.AddAttributeError(
			path.Root("scope"),
			"Empty Access Token Permissions",
			"scope must grant at least one permission via scope.ops and/or scope.op_groups.",
		)
	}

	if config.AutoPrefixStreams.IsNull() || config.AutoPrefixStreams.IsUnknown() || !config.AutoPrefixStreams.ValueBool() {
		return
	}

	if scope.Streams.IsUnknown() {
		return
	}

	if scope.Streams.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("scope").AtName("streams"),
			"Invalid Auto Prefix Streams Scope",
			"When auto_prefix_streams is true, scope.streams.prefix must be configured. Exact stream matching is not allowed with auto-prefixing.",
		)
		return
	}

	var streams ResourceSetModel
	resp.Diagnostics.Append(scope.Streams.As(ctx, &streams, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}

	if streams.Prefix.IsUnknown() {
		return
	}

	if streams.Prefix.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("scope").AtName("streams").AtName("prefix"),
			"Invalid Auto Prefix Streams Scope",
			"When auto_prefix_streams is true, scope.streams.prefix must be configured. Exact stream matching is not allowed with auto-prefixing.",
		)
	}
}

func (r *AccessTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AccessTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var scopeModel AccessTokenScopeModel
	resp.Diagnostics.Append(plan.Scope.As(ctx, &scopeModel, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}

	scope, scopeDiags := expandAccessTokenScope(ctx, scopeModel)
	resp.Diagnostics.Append(scopeDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	issueArgs := s2.IssueAccessTokenArgs{
		ID:    s2.AccessTokenID(plan.TokenID.ValueString()),
		Scope: scope,
	}

	if !plan.AutoPrefixStreams.IsNull() && !plan.AutoPrefixStreams.IsUnknown() {
		issueArgs.AutoPrefixStreams = plan.AutoPrefixStreams.ValueBool()
	}

	if !plan.ExpiresAt.IsNull() && !plan.ExpiresAt.IsUnknown() {
		expiresAt, err := time.Parse(time.RFC3339, plan.ExpiresAt.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid expires_at", "expires_at must be a valid RFC3339 timestamp")
			return
		}
		issueArgs.ExpiresAt = &expiresAt
	}

	issueResp, err := r.client.AccessTokens.Issue(ctx, issueArgs)
	if err != nil {
		if isResourceAlreadyExists(err) {
			resp.Diagnostics.AddError(
				"Access Token Already Exists",
				fmt.Sprintf("Access token %q already exists.", plan.TokenID.ValueString()),
			)
			return
		}
		resp.Diagnostics.AddError("Failed Issuing Access Token", err.Error())
		return
	}

	info, found, err := findAccessTokenByID(ctx, r.client, s2.AccessTokenID(plan.TokenID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Failed Verifying Access Token", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("Access Token Not Found", fmt.Sprintf("Access token %q was issued but not found in list response.", plan.TokenID.ValueString()))
		return
	}

	state := AccessTokenResourceModel{
		TokenID:           plan.TokenID,
		ID:                plan.TokenID,
		AutoPrefixStreams: plan.AutoPrefixStreams,
		ExpiresAt:         plan.ExpiresAt,
		AccessToken:       types.StringValue(issueResp.AccessToken),
		Scope:             plan.Scope,
	}

	if state.AutoPrefixStreams.IsUnknown() || state.AutoPrefixStreams.IsNull() {
		state.AutoPrefixStreams = types.BoolValue(info.AutoPrefixStreams)
	}
	if state.ExpiresAt.IsUnknown() || state.ExpiresAt.IsNull() {
		if info.ExpiresAt != nil {
			state.ExpiresAt = types.StringValue(info.ExpiresAt.UTC().Format(time.RFC3339))
		} else {
			state.ExpiresAt = types.StringNull()
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AccessTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AccessTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tokenID := accessTokenIDFromState(state)
	info, found, err := findAccessTokenByID(ctx, r.client, tokenID)
	if err != nil {
		resp.Diagnostics.AddError("Failed Reading Access Token", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	scopeState, scopeDiags := flattenAccessTokenScope(ctx, info.Scope)
	resp.Diagnostics.Append(scopeDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	scopeObj, scopeObjDiags := types.ObjectValueFrom(ctx, accessTokenScopeAttrTypes(), scopeState)
	resp.Diagnostics.Append(scopeObjDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	newState := AccessTokenResourceModel{
		TokenID:           types.StringValue(string(info.ID)),
		ID:                types.StringValue(string(info.ID)),
		AutoPrefixStreams: types.BoolValue(info.AutoPrefixStreams),
		ExpiresAt:         types.StringNull(),
		AccessToken:       state.AccessToken,
		Scope:             scopeObj,
	}
	if info.ExpiresAt != nil {
		newState.ExpiresAt = types.StringValue(info.ExpiresAt.UTC().Format(time.RFC3339))
	}

	newState.AccessToken = state.AccessToken

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *AccessTokenResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {
}

func (r *AccessTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AccessTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.AccessTokens.Revoke(ctx, s2.RevokeAccessTokenArgs{ID: accessTokenIDFromState(state)})
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Failed Revoking Access Token", err.Error())
	}
}

func (r *AccessTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("token_id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func accessTokenIDFromState(state AccessTokenResourceModel) s2.AccessTokenID {
	if !state.TokenID.IsNull() && !state.TokenID.IsUnknown() {
		return s2.AccessTokenID(state.TokenID.ValueString())
	}
	return s2.AccessTokenID(state.ID.ValueString())
}
