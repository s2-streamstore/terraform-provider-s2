package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/s2-streamstore/s2-sdk-go/s2"
)

func accessTokenOperations() []string {
	return []string{
		s2.OperationListBasins,
		s2.OperationCreateBasin,
		s2.OperationDeleteBasin,
		s2.OperationReconfigureBasin,
		s2.OperationGetBasinConfig,
		s2.OperationIssueAccessToken,
		s2.OperationRevokeAccessToken,
		s2.OperationListAccessTokens,
		s2.OperationListStreams,
		s2.OperationCreateStream,
		s2.OperationDeleteStream,
		s2.OperationGetStreamConfig,
		s2.OperationReconfigureStream,
		s2.OperationCheckTail,
		s2.OperationAppend,
		s2.OperationRead,
		s2.OperationTrim,
		s2.OperationFence,
		s2.OperationAccountMetrics,
		s2.OperationBasinMetrics,
		s2.OperationStreamMetrics,
	}
}

func opGroupsHasAnyPermission(groups OpGroupsModel) bool {
	return boolIsTrue(groups.AccountRead) ||
		boolIsTrue(groups.AccountWrite) ||
		boolIsTrue(groups.BasinRead) ||
		boolIsTrue(groups.BasinWrite) ||
		boolIsTrue(groups.StreamRead) ||
		boolIsTrue(groups.StreamWrite)
}

func opGroupsHasUnknownPermission(groups OpGroupsModel) bool {
	return groups.AccountRead.IsUnknown() ||
		groups.AccountWrite.IsUnknown() ||
		groups.BasinRead.IsUnknown() ||
		groups.BasinWrite.IsUnknown() ||
		groups.StreamRead.IsUnknown() ||
		groups.StreamWrite.IsUnknown()
}

func boolIsTrue(v types.Bool) bool {
	return !v.IsNull() && !v.IsUnknown() && v.ValueBool()
}
