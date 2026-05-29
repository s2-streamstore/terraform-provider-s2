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
	_ datasource.DataSource              = &LocationsDataSource{}
	_ datasource.DataSourceWithConfigure = &LocationsDataSource{}
	_ datasource.DataSource              = &DefaultLocationDataSource{}
	_ datasource.DataSourceWithConfigure = &DefaultLocationDataSource{}
)

type LocationModel struct {
	Name      types.String `tfsdk:"name"`
	IsPrivate types.Bool   `tfsdk:"is_private"`
}

type LocationsDataSource struct {
	client *s2.Client
}

type LocationsDataSourceModel struct {
	Locations []LocationModel `tfsdk:"locations"`
}

func NewLocationsDataSource() datasource.DataSource {
	return &LocationsDataSource{}
}

func (d *LocationsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_locations"
}

func (d *LocationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists S2 basin placement locations available to the account.",
		Attributes: map[string]schema.Attribute{
			"locations": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: locationAttributeSchema(),
				},
			},
		},
	}
}

func (d *LocationsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *LocationsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	locations, err := d.client.Locations.List(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed Listing Locations", err.Error())
		return
	}

	state := LocationsDataSourceModel{
		Locations: make([]LocationModel, 0, len(locations)),
	}
	for _, location := range locations {
		state.Locations = append(state.Locations, flattenLocationInfo(location))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

type DefaultLocationDataSource struct {
	client *s2.Client
}

type DefaultLocationDataSourceModel struct {
	Name      types.String `tfsdk:"name"`
	IsPrivate types.Bool   `tfsdk:"is_private"`
}

func NewDefaultLocationDataSource() datasource.DataSource {
	return &DefaultLocationDataSource{}
}

func (d *DefaultLocationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_default_location"
}

func (d *DefaultLocationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the default S2 basin placement location for the account.",
		Attributes:  locationAttributeSchema(),
	}
}

func (d *DefaultLocationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DefaultLocationDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	location, err := d.client.Locations.GetDefault(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed Reading Default Location", err.Error())
		return
	}
	if location == nil {
		resp.Diagnostics.AddError("Default Location Not Found", "S2 did not return a default location.")
		return
	}

	state := DefaultLocationDataSourceModel(flattenLocationInfo(*location))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func locationAttributeSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name":       schema.StringAttribute{Computed: true},
		"is_private": schema.BoolAttribute{Computed: true},
	}
}

func flattenLocationInfo(info s2.LocationInfo) LocationModel {
	return LocationModel{
		Name:      types.StringValue(string(info.Name)),
		IsPrivate: types.BoolValue(info.IsPrivate),
	}
}
