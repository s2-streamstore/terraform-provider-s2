package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/s2-streamstore/s2-sdk-go/s2"
)

var (
	_ datasource.DataSource              = &StreamDataSource{}
	_ datasource.DataSourceWithConfigure = &StreamDataSource{}
)

type StreamDataSource struct {
	client *s2.Client
}

type StreamDataSourceModel struct {
	Basin           types.String `tfsdk:"basin"`
	Name            types.String `tfsdk:"name"`
	CreatedAt       types.String `tfsdk:"created_at"`
	StorageClass    types.String `tfsdk:"storage_class"`
	RetentionPolicy types.Object `tfsdk:"retention_policy"`
	Timestamping    types.Object `tfsdk:"timestamping"`
	DeleteOnEmpty   types.Object `tfsdk:"delete_on_empty"`
}

func NewStreamDataSource() datasource.DataSource {
	return &StreamDataSource{}
}

func (d *StreamDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_stream"
}

func (d *StreamDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches an S2 stream by basin and name.",
		Attributes: map[string]schema.Attribute{
			"basin": schema.StringAttribute{
				Required:   true,
				Validators: basinNameValidators(),
			},
			"name": schema.StringAttribute{
				Required:   true,
				Validators: streamNameValidators(),
			},
			"created_at":    schema.StringAttribute{Computed: true},
			"storage_class": schema.StringAttribute{Computed: true},
			"retention_policy": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"age":      schema.Int64Attribute{Computed: true},
					"infinite": schema.BoolAttribute{Computed: true},
				},
			},
			"timestamping": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"mode":     schema.StringAttribute{Computed: true},
					"uncapped": schema.BoolAttribute{Computed: true},
				},
			},
			"delete_on_empty": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"min_age_secs": schema.Int64Attribute{Computed: true},
				},
			},
		},
	}
}

func (d *StreamDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = client
}

func (d *StreamDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config StreamDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	streamConfig, err := d.client.Basin(config.Basin.ValueString()).Streams.GetConfig(ctx, s2.StreamName(config.Name.ValueString()))
	if err != nil {
		if isNotFound(err) || isStreamDeletionPending(err) || isBasinDeletionPending(err) {
			resp.Diagnostics.AddError(
				"Stream Not Found",
				fmt.Sprintf("No stream %q found in basin %q.", config.Name.ValueString(), config.Basin.ValueString()),
			)
			return
		}
		resp.Diagnostics.AddError("Failed Reading Stream", err.Error())
		return
	}

	streamInfo, found, err := findStreamByName(ctx, d.client, config.Basin.ValueString(), s2.StreamName(config.Name.ValueString()), true)
	if err != nil {
		resp.Diagnostics.AddError("Failed Listing Streams", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("Stream Not Found", fmt.Sprintf("No stream %q found in basin %q.", config.Name.ValueString(), config.Basin.ValueString()))
		return
	}

	state := flattenStreamModelFromAPI(config.Basin.ValueString(), config.Name.ValueString(), streamInfo.CreatedAt, streamConfig)
	if state.CreatedAt.IsNull() {
		state.CreatedAt = types.StringValue(streamInfo.CreatedAt.Format(time.RFC3339Nano))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
