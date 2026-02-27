package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/s2-streamstore/s2-sdk-go/s2"
)

const (
	defaultBasinScope    = string(s2.BasinScopeAwsUsEast1)
	basinAsyncTimeout    = 5 * time.Minute
	streamDeleteTimeout  = 2 * time.Minute
	initialPollBackoff   = 500 * time.Millisecond
	maxPollBackoff       = 5 * time.Second
	defaultListPageLimit = 1000
)

type StreamConfigModel struct {
	StorageClass    types.String `tfsdk:"storage_class"`
	RetentionPolicy types.Object `tfsdk:"retention_policy"`
	Timestamping    types.Object `tfsdk:"timestamping"`
	DeleteOnEmpty   types.Object `tfsdk:"delete_on_empty"`
}

type RetentionPolicyModel struct {
	Age      types.Int64 `tfsdk:"age"`
	Infinite types.Bool  `tfsdk:"infinite"`
}

type TimestampingModel struct {
	Mode     types.String `tfsdk:"mode"`
	Uncapped types.Bool   `tfsdk:"uncapped"`
}

type DeleteOnEmptyModel struct {
	MinAgeSecs types.Int64 `tfsdk:"min_age_secs"`
}

type AccessTokenScopeModel struct {
	Basins       types.Object `tfsdk:"basins"`
	Streams      types.Object `tfsdk:"streams"`
	AccessTokens types.Object `tfsdk:"access_tokens"`
	OpGroups     types.Object `tfsdk:"op_groups"`
	Ops          types.Set    `tfsdk:"ops"`
}

type ResourceSetModel struct {
	Exact  types.String `tfsdk:"exact"`
	Prefix types.String `tfsdk:"prefix"`
}

type OpGroupsModel struct {
	AccountRead  types.Bool `tfsdk:"account_read"`
	AccountWrite types.Bool `tfsdk:"account_write"`
	BasinRead    types.Bool `tfsdk:"basin_read"`
	BasinWrite   types.Bool `tfsdk:"basin_write"`
	StreamRead   types.Bool `tfsdk:"stream_read"`
	StreamWrite  types.Bool `tfsdk:"stream_write"`
}

func streamConfigAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"storage_class":    types.StringType,
		"retention_policy": types.ObjectType{AttrTypes: retentionPolicyAttrTypes()},
		"timestamping":     types.ObjectType{AttrTypes: timestampingAttrTypes()},
		"delete_on_empty":  types.ObjectType{AttrTypes: deleteOnEmptyAttrTypes()},
	}
}

func retentionPolicyAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"age":      types.Int64Type,
		"infinite": types.BoolType,
	}
}

func timestampingAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"mode":     types.StringType,
		"uncapped": types.BoolType,
	}
}

func deleteOnEmptyAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"min_age_secs": types.Int64Type,
	}
}

func accessTokenScopeAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"basins":        types.ObjectType{AttrTypes: resourceSetAttrTypes()},
		"streams":       types.ObjectType{AttrTypes: resourceSetAttrTypes()},
		"access_tokens": types.ObjectType{AttrTypes: resourceSetAttrTypes()},
		"op_groups":     types.ObjectType{AttrTypes: opGroupsAttrTypes()},
		"ops":           types.SetType{ElemType: types.StringType},
	}
}

func resourceSetAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"exact":  types.StringType,
		"prefix": types.StringType,
	}
}

func opGroupsAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"account_read":  types.BoolType,
		"account_write": types.BoolType,
		"basin_read":    types.BoolType,
		"basin_write":   types.BoolType,
		"stream_read":   types.BoolType,
		"stream_write":  types.BoolType,
	}
}

func nullStreamConfigObject() types.Object {
	return types.ObjectNull(streamConfigAttrTypes())
}

func nullRetentionPolicyObject() types.Object {
	return types.ObjectNull(retentionPolicyAttrTypes())
}

func nullTimestampingObject() types.Object {
	return types.ObjectNull(timestampingAttrTypes())
}

func nullDeleteOnEmptyObject() types.Object {
	return types.ObjectNull(deleteOnEmptyAttrTypes())
}

func nullAccessTokenScopeObject() types.Object {
	return types.ObjectNull(accessTokenScopeAttrTypes())
}

func nullResourceSetObject() types.Object {
	return types.ObjectNull(resourceSetAttrTypes())
}

func nullOpGroupsObject() types.Object {
	return types.ObjectNull(opGroupsAttrTypes())
}

func flattenStreamConfig(cfg *s2.StreamConfig) StreamConfigModel {
	model := StreamConfigModel{
		StorageClass:    types.StringNull(),
		RetentionPolicy: nullRetentionPolicyObject(),
		Timestamping:    nullTimestampingObject(),
		DeleteOnEmpty:   nullDeleteOnEmptyObject(),
	}

	if cfg == nil {
		return model
	}

	if cfg.StorageClass != nil {
		model.StorageClass = types.StringValue(string(*cfg.StorageClass))
	}

	retention := flattenRetentionPolicy(cfg.RetentionPolicy)
	model.RetentionPolicy = types.ObjectValueMust(retentionPolicyAttrTypes(), map[string]attr.Value{
		"age":      retention.Age,
		"infinite": retention.Infinite,
	})

	timestamping := flattenTimestamping(cfg.Timestamping)
	model.Timestamping = types.ObjectValueMust(timestampingAttrTypes(), map[string]attr.Value{
		"mode":     timestamping.Mode,
		"uncapped": timestamping.Uncapped,
	})

	deleteOnEmpty := flattenDeleteOnEmpty(cfg.DeleteOnEmpty)
	model.DeleteOnEmpty = types.ObjectValueMust(deleteOnEmptyAttrTypes(), map[string]attr.Value{
		"min_age_secs": deleteOnEmpty.MinAgeSecs,
	})

	return model
}

func flattenRetentionPolicy(rp *s2.RetentionPolicy) RetentionPolicyModel {
	model := RetentionPolicyModel{
		Age:      types.Int64Null(),
		Infinite: types.BoolNull(),
	}

	if rp == nil {
		return model
	}

	if rp.Age != nil {
		model.Age = types.Int64Value(*rp.Age)
	}
	if rp.Infinite != nil {
		model.Infinite = types.BoolValue(true)
	}

	return model
}

func flattenTimestamping(ts *s2.TimestampingConfig) TimestampingModel {
	model := TimestampingModel{
		Mode:     types.StringNull(),
		Uncapped: types.BoolNull(),
	}

	if ts == nil {
		return model
	}

	if ts.Mode != nil {
		model.Mode = types.StringValue(string(*ts.Mode))
	}
	if ts.Uncapped != nil {
		model.Uncapped = types.BoolValue(*ts.Uncapped)
	}

	return model
}

func flattenDeleteOnEmpty(doe *s2.DeleteOnEmptyConfig) DeleteOnEmptyModel {
	if doe == nil || doe.MinAgeSecs == nil {
		return DeleteOnEmptyModel{MinAgeSecs: types.Int64Value(0)}
	}

	return DeleteOnEmptyModel{MinAgeSecs: types.Int64Value(*doe.MinAgeSecs)}
}

func flattenAccessTokenScope(ctx context.Context, scope s2.AccessTokenScope) (AccessTokenScopeModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	model := AccessTokenScopeModel{
		Basins:       nullResourceSetObject(),
		Streams:      nullResourceSetObject(),
		AccessTokens: nullResourceSetObject(),
		OpGroups:     nullOpGroupsObject(),
		Ops:          types.SetNull(types.StringType),
	}

	if scope.Basins != nil {
		model.Basins = resourceSetObjectValue(flattenResourceSet(scope.Basins))
	}
	if scope.Streams != nil {
		model.Streams = resourceSetObjectValue(flattenResourceSet(scope.Streams))
	}
	if scope.AccessTokens != nil {
		model.AccessTokens = resourceSetObjectValue(flattenResourceSet(scope.AccessTokens))
	}
	if scope.OpGroups != nil {
		opGroups := flattenOpGroups(scope.OpGroups)
		if (!opGroups.AccountRead.IsNull() && opGroups.AccountRead.ValueBool()) ||
			(!opGroups.AccountWrite.IsNull() && opGroups.AccountWrite.ValueBool()) ||
			(!opGroups.BasinRead.IsNull() && opGroups.BasinRead.ValueBool()) ||
			(!opGroups.BasinWrite.IsNull() && opGroups.BasinWrite.ValueBool()) ||
			(!opGroups.StreamRead.IsNull() && opGroups.StreamRead.ValueBool()) ||
			(!opGroups.StreamWrite.IsNull() && opGroups.StreamWrite.ValueBool()) {
			model.OpGroups = opGroupsObjectValue(opGroups)
		}
	}

	if len(scope.Ops) > 0 {
		setValue, setDiags := types.SetValueFrom(ctx, types.StringType, scope.Ops)
		diags.Append(setDiags...)
		if !setDiags.HasError() {
			model.Ops = setValue
		}
	}

	return model, diags
}

func flattenResourceSet(rs *s2.ResourceSet) ResourceSetModel {
	model := ResourceSetModel{Exact: types.StringNull(), Prefix: types.StringNull()}

	if rs == nil {
		return model
	}
	if rs.Exact != nil {
		model.Exact = types.StringValue(*rs.Exact)
	}
	if rs.Prefix != nil {
		model.Prefix = types.StringValue(*rs.Prefix)
	}

	return model
}

func flattenOpGroups(op *s2.PermittedOperationGroups) OpGroupsModel {
	model := OpGroupsModel{
		AccountRead:  types.BoolNull(),
		AccountWrite: types.BoolNull(),
		BasinRead:    types.BoolNull(),
		BasinWrite:   types.BoolNull(),
		StreamRead:   types.BoolNull(),
		StreamWrite:  types.BoolNull(),
	}

	if op == nil {
		return model
	}

	if op.Account != nil {
		model.AccountRead = types.BoolValue(op.Account.Read)
		model.AccountWrite = types.BoolValue(op.Account.Write)
	}
	if op.Basin != nil {
		model.BasinRead = types.BoolValue(op.Basin.Read)
		model.BasinWrite = types.BoolValue(op.Basin.Write)
	}
	if op.Stream != nil {
		model.StreamRead = types.BoolValue(op.Stream.Read)
		model.StreamWrite = types.BoolValue(op.Stream.Write)
	}

	return model
}

func expandStreamConfig(model StreamConfigModel) (*s2.StreamConfig, diag.Diagnostics) {
	var diags diag.Diagnostics

	cfg := &s2.StreamConfig{}
	setAny := false

	if !model.StorageClass.IsNull() && !model.StorageClass.IsUnknown() {
		storageClass := s2.StorageClass(model.StorageClass.ValueString())
		cfg.StorageClass = &storageClass
		setAny = true
	}

	if !model.RetentionPolicy.IsNull() && !model.RetentionPolicy.IsUnknown() {
		var retentionModel RetentionPolicyModel
		retentionDiags := model.RetentionPolicy.As(context.Background(), &retentionModel, basetypes.ObjectAsOptions{})
		diags.Append(retentionDiags...)
		if !retentionDiags.HasError() {
			cfg.RetentionPolicy = expandRetentionPolicy(retentionModel)
			if cfg.RetentionPolicy != nil {
				setAny = true
			}
		}
	}

	if !model.Timestamping.IsNull() && !model.Timestamping.IsUnknown() {
		var tsModel TimestampingModel
		tsDiags := model.Timestamping.As(context.Background(), &tsModel, basetypes.ObjectAsOptions{})
		diags.Append(tsDiags...)
		if !tsDiags.HasError() {
			cfg.Timestamping = expandTimestamping(tsModel)
			if cfg.Timestamping != nil {
				setAny = true
			}
		}
	}

	if !model.DeleteOnEmpty.IsNull() && !model.DeleteOnEmpty.IsUnknown() {
		var doeModel DeleteOnEmptyModel
		doeDiags := model.DeleteOnEmpty.As(context.Background(), &doeModel, basetypes.ObjectAsOptions{})
		diags.Append(doeDiags...)
		if !doeDiags.HasError() {
			cfg.DeleteOnEmpty = expandDeleteOnEmpty(doeModel)
			if cfg.DeleteOnEmpty != nil {
				setAny = true
			}
		}
	}

	if !setAny {
		return nil, diags
	}

	return cfg, diags
}

func expandRetentionPolicy(model RetentionPolicyModel) *s2.RetentionPolicy {
	policy := &s2.RetentionPolicy{}
	setAny := false

	if !model.Age.IsNull() && !model.Age.IsUnknown() {
		age := model.Age.ValueInt64()
		policy.Age = &age
		setAny = true
	}

	if !model.Infinite.IsNull() && !model.Infinite.IsUnknown() && model.Infinite.ValueBool() {
		policy.Infinite = &s2.InfiniteRetention{}
		setAny = true
	}

	if !setAny {
		return nil
	}

	return policy
}

func expandTimestamping(model TimestampingModel) *s2.TimestampingConfig {
	timestamping := &s2.TimestampingConfig{}
	setAny := false

	if !model.Mode.IsNull() && !model.Mode.IsUnknown() {
		mode := s2.TimestampingMode(model.Mode.ValueString())
		timestamping.Mode = &mode
		setAny = true
	}

	if !model.Uncapped.IsNull() && !model.Uncapped.IsUnknown() {
		uncapped := model.Uncapped.ValueBool()
		timestamping.Uncapped = &uncapped
		setAny = true
	}

	if !setAny {
		return nil
	}

	return timestamping
}

func expandDeleteOnEmpty(model DeleteOnEmptyModel) *s2.DeleteOnEmptyConfig {
	if model.MinAgeSecs.IsNull() || model.MinAgeSecs.IsUnknown() {
		return nil
	}
	minAge := model.MinAgeSecs.ValueInt64()
	return &s2.DeleteOnEmptyConfig{MinAgeSecs: &minAge}
}

func expandAccessTokenScope(ctx context.Context, model AccessTokenScopeModel) (s2.AccessTokenScope, diag.Diagnostics) {
	var diags diag.Diagnostics

	scope := s2.AccessTokenScope{}

	if !model.Basins.IsNull() && !model.Basins.IsUnknown() {
		var basins ResourceSetModel
		basinDiags := model.Basins.As(ctx, &basins, basetypes.ObjectAsOptions{})
		diags.Append(basinDiags...)
		if !basinDiags.HasError() {
			scope.Basins = expandResourceSet(basins)
		}
	}

	if !model.Streams.IsNull() && !model.Streams.IsUnknown() {
		var streams ResourceSetModel
		streamDiags := model.Streams.As(ctx, &streams, basetypes.ObjectAsOptions{})
		diags.Append(streamDiags...)
		if !streamDiags.HasError() {
			scope.Streams = expandResourceSet(streams)
		}
	}

	if !model.AccessTokens.IsNull() && !model.AccessTokens.IsUnknown() {
		var accessTokens ResourceSetModel
		tokenDiags := model.AccessTokens.As(ctx, &accessTokens, basetypes.ObjectAsOptions{})
		diags.Append(tokenDiags...)
		if !tokenDiags.HasError() {
			scope.AccessTokens = expandResourceSet(accessTokens)
		}
	}

	if !model.OpGroups.IsNull() && !model.OpGroups.IsUnknown() {
		var opGroups OpGroupsModel
		opDiags := model.OpGroups.As(ctx, &opGroups, basetypes.ObjectAsOptions{})
		diags.Append(opDiags...)
		if !opDiags.HasError() {
			scope.OpGroups = expandOpGroups(opGroups)
		}
	}

	if !model.Ops.IsNull() && !model.Ops.IsUnknown() {
		var ops []string
		opsDiags := model.Ops.ElementsAs(ctx, &ops, false)
		diags.Append(opsDiags...)
		if !opsDiags.HasError() {
			scope.Ops = ops
		}
	}

	return scope, diags
}

func expandResourceSet(model ResourceSetModel) *s2.ResourceSet {
	resourceSet := &s2.ResourceSet{}
	setAny := false

	if !model.Exact.IsNull() && !model.Exact.IsUnknown() {
		exact := model.Exact.ValueString()
		resourceSet.Exact = &exact
		setAny = true
	}
	if !model.Prefix.IsNull() && !model.Prefix.IsUnknown() {
		prefix := model.Prefix.ValueString()
		resourceSet.Prefix = &prefix
		setAny = true
	}

	if !setAny {
		return nil
	}

	return resourceSet
}

func expandOpGroups(model OpGroupsModel) *s2.PermittedOperationGroups {
	groups := &s2.PermittedOperationGroups{}
	setAny := false

	if !model.AccountRead.IsNull() || !model.AccountWrite.IsNull() {
		account := &s2.ReadWritePermissions{}
		if !model.AccountRead.IsNull() && !model.AccountRead.IsUnknown() {
			account.Read = model.AccountRead.ValueBool()
		}
		if !model.AccountWrite.IsNull() && !model.AccountWrite.IsUnknown() {
			account.Write = model.AccountWrite.ValueBool()
		}
		groups.Account = account
		setAny = true
	}

	if !model.BasinRead.IsNull() || !model.BasinWrite.IsNull() {
		basin := &s2.ReadWritePermissions{}
		if !model.BasinRead.IsNull() && !model.BasinRead.IsUnknown() {
			basin.Read = model.BasinRead.ValueBool()
		}
		if !model.BasinWrite.IsNull() && !model.BasinWrite.IsUnknown() {
			basin.Write = model.BasinWrite.ValueBool()
		}
		groups.Basin = basin
		setAny = true
	}

	if !model.StreamRead.IsNull() || !model.StreamWrite.IsNull() {
		stream := &s2.ReadWritePermissions{}
		if !model.StreamRead.IsNull() && !model.StreamRead.IsUnknown() {
			stream.Read = model.StreamRead.ValueBool()
		}
		if !model.StreamWrite.IsNull() && !model.StreamWrite.IsUnknown() {
			stream.Write = model.StreamWrite.ValueBool()
		}
		groups.Stream = stream
		setAny = true
	}

	if !setAny {
		return nil
	}

	return groups
}

func expandStreamReconfiguration(model StreamConfigModel) (s2.StreamReconfiguration, diag.Diagnostics) {
	var diags diag.Diagnostics
	var reconfigure s2.StreamReconfiguration

	if !model.StorageClass.IsNull() && !model.StorageClass.IsUnknown() {
		storageClass := s2.StorageClass(model.StorageClass.ValueString())
		reconfigure.StorageClass = &storageClass
	}

	if !model.RetentionPolicy.IsNull() && !model.RetentionPolicy.IsUnknown() {
		var retentionModel RetentionPolicyModel
		retentionDiags := model.RetentionPolicy.As(context.Background(), &retentionModel, basetypes.ObjectAsOptions{})
		diags.Append(retentionDiags...)
		if !retentionDiags.HasError() {
			reconfigure.RetentionPolicy = expandRetentionPolicy(retentionModel)
		}
	}

	if !model.Timestamping.IsNull() && !model.Timestamping.IsUnknown() {
		var timestampingModel TimestampingModel
		tsDiags := model.Timestamping.As(context.Background(), &timestampingModel, basetypes.ObjectAsOptions{})
		diags.Append(tsDiags...)
		if !tsDiags.HasError() {
			timestamping := expandTimestamping(timestampingModel)
			if timestamping != nil {
				reconfigure.Timestamping = &s2.TimestampingReconfiguration{
					Mode:     timestamping.Mode,
					Uncapped: timestamping.Uncapped,
				}
			}
		}
	}

	if !model.DeleteOnEmpty.IsNull() && !model.DeleteOnEmpty.IsUnknown() {
		var deleteOnEmptyModel DeleteOnEmptyModel
		doeDiags := model.DeleteOnEmpty.As(context.Background(), &deleteOnEmptyModel, basetypes.ObjectAsOptions{})
		diags.Append(doeDiags...)
		if !doeDiags.HasError() {
			if deleteOnEmpty := expandDeleteOnEmpty(deleteOnEmptyModel); deleteOnEmpty != nil {
				reconfigure.DeleteOnEmpty = &s2.DeleteOnEmptyReconfiguration{MinAgeSecs: deleteOnEmpty.MinAgeSecs}
			}
		}
	}

	return reconfigure, diags
}

func isDefaultStreamConfig(cfg *s2.StreamConfig) bool {
	if cfg == nil {
		return true
	}

	if cfg.StorageClass != nil && *cfg.StorageClass != s2.StorageClassExpress {
		return false
	}

	if cfg.RetentionPolicy != nil {
		if cfg.RetentionPolicy.Age != nil || cfg.RetentionPolicy.Infinite != nil {
			return false
		}
	}

	if cfg.Timestamping != nil {
		if cfg.Timestamping.Mode != nil && *cfg.Timestamping.Mode != s2.TimestampingModeClientPrefer {
			return false
		}
		if cfg.Timestamping.Uncapped != nil && *cfg.Timestamping.Uncapped {
			return false
		}
	}

	if cfg.DeleteOnEmpty != nil && cfg.DeleteOnEmpty.MinAgeSecs != nil && *cfg.DeleteOnEmpty.MinAgeSecs != 0 {
		return false
	}

	return true
}

func waitForBasinState(ctx context.Context, client *s2.Client, name s2.BasinName, target s2.BasinState, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	backoff := initialPollBackoff

	for {
		basinInfo, found, err := findBasinByName(ctx, client, name, true)
		if err != nil {
			return err
		}

		if found {
			if basinInfo.State == target {
				return nil
			}
			if basinInfo.State == s2.BasinStateDeleting && target != s2.BasinStateDeleting {
				return fmt.Errorf("basin %q entered deleting while waiting for %q", name, target)
			}
		} else {
			return fmt.Errorf("basin %q not found while waiting for %q", name, target)
		}

		if err := sleepWithBackoff(ctx, deadline, &backoff); err != nil {
			return fmt.Errorf("timed out waiting for basin %q to reach %q", name, target)
		}
	}
}

func waitForBasinDeletion(ctx context.Context, client *s2.Client, name s2.BasinName, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	backoff := initialPollBackoff

	for {
		_, found, err := findBasinByName(ctx, client, name, true)
		if err != nil {
			if isNotFound(err) {
				return nil
			}
			return err
		}
		if !found {
			return nil
		}

		if err := sleepWithBackoff(ctx, deadline, &backoff); err != nil {
			return fmt.Errorf("timed out waiting for basin %q deletion", name)
		}
	}
}

func waitForStreamDeletion(ctx context.Context, client *s2.Client, basin string, name s2.StreamName, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	backoff := initialPollBackoff

	for {
		_, err := client.Basin(basin).Streams.GetConfig(ctx, name)
		if err != nil {
			if isNotFound(err) {
				return nil
			}
			if isBasinDeletionPending(err) {
				return nil
			}
			if isStreamDeletionPending(err) || isTransactionConflict(err) || isConflict(err) {
				if err := sleepWithBackoff(ctx, deadline, &backoff); err != nil {
					return fmt.Errorf("timed out waiting for stream %q in basin %q deletion", name, basin)
				}
				continue
			}
			return err
		}

		if err := sleepWithBackoff(ctx, deadline, &backoff); err != nil {
			return fmt.Errorf("timed out waiting for stream %q in basin %q deletion", name, basin)
		}
	}
}

func isNotFound(err error) bool {
	var apiErr *s2.S2Error
	if !errors.As(err, &apiErr) {
		return false
	}
	if apiErr.Status == 404 {
		return true
	}
	return strings.EqualFold(apiErr.Code, "basin_not_found") ||
		strings.EqualFold(apiErr.Code, "stream_not_found") ||
		strings.EqualFold(apiErr.Code, "access_token_not_found")
}

func isConflict(err error) bool {
	var apiErr *s2.S2Error
	return errors.As(err, &apiErr) && apiErr.Status == 409
}

func isUnavailable(err error) bool {
	var apiErr *s2.S2Error
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.Status == 503 || strings.EqualFold(apiErr.Code, "unavailable")
}

func isResourceAlreadyExists(err error) bool {
	var apiErr *s2.S2Error
	if !errors.As(err, &apiErr) {
		return false
	}
	if apiErr.Status != 409 {
		return false
	}
	return strings.EqualFold(apiErr.Code, "resource_already_exists")
}

func isTransactionConflict(err error) bool {
	var apiErr *s2.S2Error
	if !errors.As(err, &apiErr) {
		return false
	}
	if apiErr.Status != 409 {
		return false
	}
	return strings.EqualFold(apiErr.Code, "transaction_conflict")
}

func isBasinDeletionPending(err error) bool {
	var apiErr *s2.S2Error
	if !errors.As(err, &apiErr) {
		return false
	}
	if apiErr.Status != 409 {
		return false
	}

	if strings.EqualFold(apiErr.Code, "basin_deletion_pending") {
		return true
	}

	msg := strings.ToLower(apiErr.Message)
	return strings.Contains(msg, "basin") && strings.Contains(msg, "delet")
}

func isStreamDeletionPending(err error) bool {
	var apiErr *s2.S2Error
	if !errors.As(err, &apiErr) {
		return false
	}
	if apiErr.Status != 409 {
		return false
	}

	if strings.EqualFold(apiErr.Code, "stream_deletion_pending") {
		return true
	}

	return strings.EqualFold(apiErr.Message, "Stream is being deleted")
}

func findBasinByName(ctx context.Context, client *s2.Client, name s2.BasinName, includeDeleted bool) (s2.BasinInfo, bool, error) {
	iter := client.Basins.Iter(ctx, &s2.ListBasinsArgs{
		Prefix:         string(name),
		Limit:          s2.Ptr(defaultListPageLimit),
		IncludeDeleted: includeDeleted,
	})

	for iter.Next() {
		candidate := iter.Value()
		if candidate.Name == name {
			return candidate, true, nil
		}
	}

	if err := iter.Err(); err != nil {
		return s2.BasinInfo{}, false, err
	}

	return s2.BasinInfo{}, false, nil
}

func findStreamByName(ctx context.Context, client *s2.Client, basin string, name s2.StreamName, includeDeleted bool) (s2.StreamInfo, bool, error) {
	iter := client.Basin(basin).Streams.Iter(ctx, &s2.ListStreamsArgs{
		Prefix:         string(name),
		Limit:          s2.Ptr(defaultListPageLimit),
		IncludeDeleted: includeDeleted,
	})

	for iter.Next() {
		candidate := iter.Value()
		if candidate.Name == name {
			return candidate, true, nil
		}
	}

	if err := iter.Err(); err != nil {
		return s2.StreamInfo{}, false, err
	}

	return s2.StreamInfo{}, false, nil
}

func findAccessTokenByID(ctx context.Context, client *s2.Client, id s2.AccessTokenID) (s2.AccessTokenInfo, bool, error) {
	iter := client.AccessTokens.Iter(ctx, &s2.ListAccessTokensArgs{
		Prefix: string(id),
		Limit:  s2.Ptr(defaultListPageLimit),
	})

	for iter.Next() {
		candidate := iter.Value()
		if candidate.ID == id {
			return candidate, true, nil
		}
	}

	if err := iter.Err(); err != nil {
		return s2.AccessTokenInfo{}, false, err
	}

	return s2.AccessTokenInfo{}, false, nil
}

func sleepWithBackoff(ctx context.Context, deadline time.Time, backoff *time.Duration) error {
	if time.Now().After(deadline) {
		return context.DeadlineExceeded
	}

	wait := *backoff
	if remaining := time.Until(deadline); wait > remaining {
		wait = remaining
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
	}

	next := *backoff * 2
	if next > maxPollBackoff {
		next = maxPollBackoff
	}
	*backoff = next

	return nil
}

func stringFromStringValue(v types.String) string {
	if v.IsNull() || v.IsUnknown() {
		return ""
	}
	return strings.TrimSpace(v.ValueString())
}

func resourceSetObjectValue(model ResourceSetModel) types.Object {
	return types.ObjectValueMust(resourceSetAttrTypes(), map[string]attr.Value{
		"exact":  model.Exact,
		"prefix": model.Prefix,
	})
}

func opGroupsObjectValue(model OpGroupsModel) types.Object {
	return types.ObjectValueMust(opGroupsAttrTypes(), map[string]attr.Value{
		"account_read":  model.AccountRead,
		"account_write": model.AccountWrite,
		"basin_read":    model.BasinRead,
		"basin_write":   model.BasinWrite,
		"stream_read":   model.StreamRead,
		"stream_write":  model.StreamWrite,
	})
}
