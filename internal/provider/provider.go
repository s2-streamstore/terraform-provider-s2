package provider

import (
	"context"
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
	AccessToken     types.String `tfsdk:"access_token"`
	AccountEndpoint types.String `tfsdk:"account_endpoint"`
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
			"account_endpoint": providerschema.StringAttribute{
				Optional:    true,
				Description: "S2 account endpoint. Can also be set via S2_ACCOUNT_ENDPOINT. Defaults to the S2 production endpoint.",
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
	accountEndpoint := strings.TrimSpace(config.AccountEndpoint.ValueString())

	if accessToken == "" {
		accessToken = strings.TrimSpace(os.Getenv("S2_ACCESS_TOKEN"))
	}
	if accountEndpoint == "" {
		accountEndpoint = strings.TrimSpace(os.Getenv("S2_ACCOUNT_ENDPOINT"))
	}

	if accessToken == "" {
		resp.Diagnostics.AddError(
			"Missing S2 Access Token",
			"The provider cannot create an S2 client because access_token is unset. Set provider.access_token or the S2_ACCESS_TOKEN environment variable.",
		)
		return
	}

	clientOptions := &s2.ClientOptions{}
	if accountEndpoint != "" {
		baseURL := accountEndpointToBaseURL(accountEndpoint)
		clientOptions.BaseURL = baseURL
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

// accountEndpointToBaseURL converts a user-supplied endpoint (e.g. "aws.s2.dev" or
// "localhost:8080") to the full base URL used by the SDK (e.g. "https://aws.s2.dev/v1").
// A scheme is added when absent (http for localhost, https otherwise).
// /v1 is appended when the endpoint has no explicit path component.
func accountEndpointToBaseURL(endpoint string) string {
	if !strings.Contains(endpoint, "://") {
		host := endpoint
		if idx := strings.Index(endpoint, "/"); idx != -1 {
			host = endpoint[:idx]
		}
		// Strip port for localhost check.
		if idx := strings.LastIndex(host, ":"); idx != -1 {
			host = host[:idx]
		}
		scheme := "https"
		if host == "localhost" || host == "127.0.0.1" {
			scheme = "http"
		}
		endpoint = scheme + "://" + endpoint
	}
	// If no path after scheme://host, append /v1.
	afterScheme := endpoint[strings.Index(endpoint, "://")+3:]
	if !strings.Contains(afterScheme, "/") {
		return endpoint + "/v1"
	}
	return endpoint
}
