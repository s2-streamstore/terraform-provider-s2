package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/s2-streamstore/s2-sdk-go/s2"
)

var (
	_ datasource.DataSource              = &BasinDataSource{}
	_ datasource.DataSourceWithConfigure = &BasinDataSource{}
)

type BasinDataSource struct {
	client *s2.Client
}

type BasinDataSourceModel struct {
	Name                 types.String `tfsdk:"name"`
	Scope                types.String `tfsdk:"scope"`
	State                types.String `tfsdk:"state"`
	CreateStreamOnAppend types.Bool   `tfsdk:"create_stream_on_append"`
	CreateStreamOnRead   types.Bool   `tfsdk:"create_stream_on_read"`
	DefaultStreamConfig  types.Object `tfsdk:"default_stream_config"`
}

func NewBasinDataSource() datasource.DataSource {
	return &BasinDataSource{}
}

func (d *BasinDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_basin"
}

func (d *BasinDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches an S2 basin by name.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:   true,
				Validators: basinNameValidators(),
			},
			"scope":                   schema.StringAttribute{Computed: true},
			"state":                   schema.StringAttribute{Computed: true},
			"create_stream_on_append": schema.BoolAttribute{Computed: true},
			"create_stream_on_read":   schema.BoolAttribute{Computed: true},
			"default_stream_config": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
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
			},
		},
	}
}

func (d *BasinDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *BasinDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config BasinDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	basinName := s2.BasinName(config.Name.ValueString())
	basinConfig, err := d.client.Basins.GetConfig(ctx, basinName)
	if err != nil && !isUnavailable(err) {
		if isNotFound(err) || isBasinDeletionPending(err) {
			resp.Diagnostics.AddError("Basin Not Found", fmt.Sprintf("No basin named %q was found.", basinName))
			return
		}
		resp.Diagnostics.AddError("Failed Reading Basin", err.Error())
		return
	}

	basinInfo, found, err := findBasinByName(ctx, d.client, basinName, true)
	if err != nil {
		resp.Diagnostics.AddError("Failed Listing Basin", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("Basin Not Found", fmt.Sprintf("No basin named %q was found.", basinName))
		return
	}

	state := flattenBasinModelFromAPI(ctx, basinInfo, basinConfig)
	state.Name = config.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
