package provider

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/s2-streamstore/s2-sdk-go/s2"
)

var _ provider.Provider = &S2Provider{}

type S2Provider struct {
	version string
}

type S2ProviderModel struct {
	AccessToken types.String `tfsdk:"access_token"`
	BaseURL     types.String `tfsdk:"base_url"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &S2Provider{version: version}
	}
}

func (p *S2Provider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "s2"
	resp.Version = p.version
}

func (p *S2Provider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = providerschema.Schema{
		Description: "Terraform provider for S2 basins, streams, and access tokens.",
		Attributes: map[string]providerschema.Attribute{
			"access_token": providerschema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "S2 access token. Can also be set via S2_ACCESS_TOKEN.",
			},
			"base_url": providerschema.StringAttribute{
				Optional:    true,
				Description: fmt.Sprintf("S2 API base URL. Can also be set via S2_BASE_URL. Defaults to %q.", s2.DefaultBaseURL),
			},
		},
	}
}

func (p *S2Provider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config S2ProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	accessToken := strings.TrimSpace(config.AccessToken.ValueString())
	baseURL := strings.TrimSpace(config.BaseURL.ValueString())

	if accessToken == "" {
		accessToken = strings.TrimSpace(os.Getenv("S2_ACCESS_TOKEN"))
	}
	if baseURL == "" {
		baseURL = strings.TrimSpace(os.Getenv("S2_BASE_URL"))
	}
	if baseURL == "" {
		baseURL = s2.DefaultBaseURL
	}

	if accessToken == "" {
		resp.Diagnostics.AddError(
			"Missing S2 Access Token",
			"The provider cannot create an S2 client because access_token is unset. Set provider.access_token or the S2_ACCESS_TOKEN environment variable.",
		)
		return
	}

	clientOptions := &s2.ClientOptions{BaseURL: baseURL}
	if strings.TrimRight(baseURL, "/") != strings.TrimRight(s2.DefaultBaseURL, "/") {
		clientOptions.MakeBasinBaseURL = func(_ string) string { return baseURL }
	}

	client := s2.New(accessToken, clientOptions)
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *S2Provider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewBasinResource,
		NewStreamResource,
		NewAccessTokenResource,
	}
}

func (p *S2Provider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewBasinDataSource,
		NewStreamDataSource,
	}
}
