package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func NewApikeyResource() resource.Resource {
	return &ApiKeyResource{}
}

type ApiKeyResource struct {
	client *elasticsearch.Client
}

type ApiKeyResourceModel struct {
	Id              types.String `tfsdk:"id"`
	ApiKey          types.String `tfsdk:"api_key"`
	Encoded         types.String `tfsdk:"encoded"`
	Name            types.String `tfsdk:"name"`
	RoleDescriptors types.Set    `tfsdk:"role_descriptors"`
}

type RoleDescriptor struct {
	Name     types.String `tfsdk:"name"`
	Cluster  types.List   `tfsdk:"cluster"`
	Indicies types.Set    `tfsdk:"indices"`
}

type Index struct {
	Names      types.List `tfsdk:"names"`
	Privileges types.List `tfsdk:"privileges"`
}

func (r *ApiKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (r *ApiKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "API Key Identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"api_key": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Generated API Key",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"encoded": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "API key credentials which is the Base64-encoding of the UTF-8 representation of the id and api_key joined by a colon (:).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the API Key to create",
				Required:            true,
			},
			"role_descriptors": schema.SetNestedAttribute{
				MarkdownDescription: "Role Descriptors for the API Key",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required: true,
						},
						"cluster": schema.ListAttribute{
							MarkdownDescription: "A list of cluster privileges",
							ElementType:         types.StringType,
							Optional:            true,
						},
						"indices": schema.SetNestedAttribute{
							MarkdownDescription: "A list of indices permissions entries",
							Optional:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"names": schema.ListAttribute{
										MarkdownDescription: "A list of indices (or index name patterns) to which the permissions in this entry apply",
										ElementType:         types.StringType,
										Required:            true,
									},
									"privileges": schema.ListAttribute{
										MarkdownDescription: "The index level privileges that the owners of the role have on the specified indices.",
										ElementType:         types.StringType,
										Required:            true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *ApiKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*elasticsearch.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *elasticsearch.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func transformApiKeyResourceBody(ctx context.Context, data ApiKeyResourceModel) map[string]interface{} {
	bodyJson := make(map[string]interface{})

	roleDescriptors := make(map[string]interface{})

	rawRoleDescriptors := make([]RoleDescriptor, len(data.RoleDescriptors.Elements()))
	data.RoleDescriptors.ElementsAs(ctx, &rawRoleDescriptors, false)

	for _, roleDescriptor := range rawRoleDescriptors {
		roleName := roleDescriptor.Name.ValueString()

		cluster := make([]string, len(roleDescriptor.Cluster.Elements()))
		roleDescriptor.Cluster.ElementsAs(ctx, &cluster, false)

		rawIndices := make([]Index, len(roleDescriptor.Indicies.Elements()))
		roleDescriptor.Indicies.ElementsAs(ctx, &rawIndices, false)

		index := make([]interface{}, len(rawIndices))
		for i, rawIndex := range rawIndices {
			names := make([]string, len(rawIndex.Names.Elements()))
			rawIndex.Names.ElementsAs(ctx, &names, false)

			privileges := make([]string, len(rawIndex.Privileges.Elements()))
			rawIndex.Privileges.ElementsAs(ctx, &privileges, false)

			index[i] = map[string]interface{}{
				"names":      names,
				"privileges": privileges,
			}
		}

		roleDescriptors[roleName] = map[string]interface{}{
			"cluster": cluster,
			"index":   index,
		}
	}

	bodyJson["role_descriptors"] = roleDescriptors

	return bodyJson
}

func (r *ApiKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *ApiKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError("Configuration Error", "Error when parsing attributes")
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("[Parameters] %#v;", data.RoleDescriptors.Elements()))

	bodyJson := transformApiKeyResourceBody(ctx, *data)
	bodyJson["name"] = data.Name.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("[RoleDescriptor] %#v;", bodyJson))

	b := new(bytes.Buffer)
	if err := json.NewEncoder(b).Encode(bodyJson); err != nil {
		resp.Diagnostics.AddError("JSON Encode Error", fmt.Sprintf("Error encoding query: %s", err))
	}
	apiReq := esapi.SecurityCreateAPIKeyRequest{
		Body: b,
	}

	apiRes, err := apiReq.Do(ctx, r.client)
	if err != nil {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Error getting response: %s", err))
		return
	}
	defer apiRes.Body.Close()

	if apiRes.IsError() {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("[%s] Error creating API key", apiRes.Status()))
		return
	}

	// Deserialize the response into a map.
	var result map[string]interface{}
	if err := json.NewDecoder(apiRes.Body).Decode(&result); err != nil {
		resp.Diagnostics.AddError("JSON Decode Error", fmt.Sprintf("Error parsing the response body: %s", err))
		return
	} else {
		tflog.Debug(ctx, fmt.Sprintf("[%s] %#v;", apiRes.Status(), result))
	}

	data.Id = types.StringValue(result["id"].(string))
	data.ApiKey = types.StringValue(result["api_key"].(string))
	data.Encoded = types.StringValue(result["encoded"].(string))

	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ApiKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *ApiKeyResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read example, got error: %s", err))
	//     return
	// }

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ApiKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *ApiKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError("Configuration Error", "Error when parsing attributes")
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("[Parameters] %#v;", data.RoleDescriptors.Elements()))

	bodyJson := transformApiKeyResourceBody(ctx, *data)
	tflog.Debug(ctx, fmt.Sprintf("[RoleDescriptor] %#v;", bodyJson))

	b := new(bytes.Buffer)
	if err := json.NewEncoder(b).Encode(bodyJson); err != nil {
		resp.Diagnostics.AddError("JSON Encode Error", fmt.Sprintf("Error encoding query: %s", err))
	}
	apiReq := esapi.SecurityUpdateAPIKeyRequest{
		DocumentID: data.Id.ValueString(),
		Body:       b,
	}

	apiRes, err := apiReq.Do(ctx, r.client)
	if err != nil {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Error getting response: %s", err))
		return
	}
	defer apiRes.Body.Close()

	if apiRes.IsError() {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("[%s] Error updating API key", apiRes.Status()))
		return
	}

	// Deserialize the response into a map.
	var result map[string]interface{}
	if err := json.NewDecoder(apiRes.Body).Decode(&result); err != nil {
		resp.Diagnostics.AddError("JSON Decode Error", fmt.Sprintf("Error parsing the response body: %s", err))
		return
	} else {
		tflog.Debug(ctx, fmt.Sprintf("[%s] %#v;", apiRes.Status(), result))
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ApiKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *ApiKeyResourceModel

	tflog.Trace(ctx, "debugging delete method")

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError("Configutation Error", "Error when parsing attributes")
		return
	}

	bodyJson := map[string]interface{}{
		"ids": []string{data.Id.ValueString()},
	}
	b := new(bytes.Buffer)
	if err := json.NewEncoder(b).Encode(bodyJson); err != nil {
		resp.Diagnostics.AddError("JSON Encode Error", fmt.Sprintf("Error encoding query: %s", err))
	}
	apiReq := esapi.SecurityInvalidateAPIKeyRequest{
		Body: b,
	}

	apiRes, err := apiReq.Do(context.Background(), r.client)
	if err != nil {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Error getting response: %s", err))
	}
	defer apiRes.Body.Close()

	if apiRes.IsError() {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("[%s] Error creating API key", apiRes.Status()))
		return
	}

	// Deserialize the response into a map.
	var result map[string]interface{}
	if err := json.NewDecoder(apiRes.Body).Decode(&result); err != nil {
		resp.Diagnostics.AddError("JSON Decode Error", fmt.Sprintf("Error parsing the response body: %s", err))
		return
	} else {
		tflog.Debug(ctx, fmt.Sprintf("[%s] %#v;", apiRes.Status(), result))
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete example, got error: %s", err))
	//     return
	// }
}
