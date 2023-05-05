package provider

import (
	"context"
	"crypto/tls"
	"net/http"

	"github.com/elastic/go-elasticsearch/v8"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure ElasticsearchSecurityProvider satisfies various provider interfaces.
var _ provider.Provider = &ElasticsearchSecurityProvider{}

// ElasticsearchSecurityProvider defines the provider implementation.
type ElasticsearchSecurityProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// ElasticsearchSecurityProviderModel describes the provider data model.
type ElasticsearchSecurityProviderModel struct {
	Url      types.String `tfsdk:"url"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

func (p *ElasticsearchSecurityProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "essecurity"
	resp.Version = p.version
}

func (p *ElasticsearchSecurityProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				MarkdownDescription: "Elasticsearch URL",
				Required:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Username to use to connect to elasticsearch using basic auth",
				Required:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Password to use to connect to elasticsearch using basic auth",
				Required:            true,
			},
		},
	}
}

func (p *ElasticsearchSecurityProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data ElasticsearchSecurityProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	config := elasticsearch.Config{
		Addresses: []string{data.Url.ValueString()},
		Username:  data.Username.ValueString(),
		Password:  data.Password.ValueString(),
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	client, err := elasticsearch.NewClient(config)
	if err != nil {
		resp.Diagnostics.AddError("Client initialization failed", "Failed to create Elasticearch client")
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *ElasticsearchSecurityProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewApikeyResource,
	}
}

func (p *ElasticsearchSecurityProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ElasticsearchSecurityProvider{
			version: version,
		}
	}
}
