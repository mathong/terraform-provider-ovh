package ovh

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/ovh/terraform-provider-ovh/v2/ovh/helpers"
)

var (
	_ resource.ResourceWithConfigure   = (*networkDefenseVacPolicyResource)(nil)
	_ resource.ResourceWithImportState = (*networkDefenseVacPolicyResource)(nil)
)

func NewNetworkDefenseVacPolicyResource() resource.Resource {
	return &networkDefenseVacPolicyResource{}
}

type networkDefenseVacPolicyResource struct {
	config *Config
}

func (r *networkDefenseVacPolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_defense_vac_policy"
}

func (r *networkDefenseVacPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *Config, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.config = config
}

func (r *networkDefenseVacPolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = NetworkDefenseVacPolicyResourceSchema(ctx)
}

func (r *networkDefenseVacPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ids := strings.Split(req.ID, "/")
	if len(ids) != 2 {
		resp.Diagnostics.AddError("Error importing network_defense_vac_policy resource.", "ID should be of the format '{resourceName}/{policyId}'")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_name"), ids[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), ids[1])...)
}

func vacPolicyEndpoint(data *NetworkDefenseVacPolicyModel) string {
	return "/v2/networkDefense/vac/resource/" + url.PathEscape(data.ResourceName.ValueString()) + "/policy/" + url.PathEscape(data.Id.ValueString())
}

// The API takes force as a query parameter. It is sent only when true, because it
// overwrites protection settings that cannot be restored.
func vacPolicyForceQuery(data *NetworkDefenseVacPolicyModel) string {
	if data.Force.ValueBool() {
		return "?force=true"
	}
	return ""
}

func (r *networkDefenseVacPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data, createdData, responseData NetworkDefenseVacPolicyModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := "/v2/networkDefense/vac/resource/" + url.PathEscape(data.ResourceName.ValueString()) + "/policy" + vacPolicyForceQuery(&data)
	payload := map[string]any{"targetSpec": data.TargetSpec.ToCreate()}
	if err := r.config.OVHClient.PostWithContext(ctx, endpoint, payload, &createdData); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error calling Post %s", endpoint), err.Error())
		return
	}
	data.Id = createdData.Id

	// Save the ID now so a failed wait does not leave an untracked policy behind.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), data.Id)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_name"), data.ResourceName)...)

	endpoint = vacPolicyEndpoint(&data)
	if err := helpers.WaitForAPIv2ResourceStatusReady(ctx, r.config.OVHClient, endpoint); err != nil {
		resp.Diagnostics.AddError("Error waiting for VAC policy to be ready", err.Error())
		return
	}

	if err := r.config.OVHClient.GetWithContext(ctx, endpoint, &responseData); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error calling Get %s", endpoint), err.Error())
		return
	}

	responseData.MergeWith(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &responseData)...)
}

func (r *networkDefenseVacPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data, responseData NetworkDefenseVacPolicyModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := vacPolicyEndpoint(&data)
	if err := r.config.OVHClient.GetWithContext(ctx, endpoint, &responseData); err != nil {
		if helpers.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(fmt.Sprintf("Error calling Get %s", endpoint), err.Error())
		return
	}

	// API values win so that drift shows up. State keeps what the API does not return (force, resource_name).
	responseData.MergeWith(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &responseData)...)
}

func (r *networkDefenseVacPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, planData, responseData NetworkDefenseVacPolicyModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := vacPolicyEndpoint(&data)
	// The checksum comes from state: the API rejects the update if the policy changed since the last read.
	payload := map[string]any{
		"checksum":   data.Checksum,
		"targetSpec": planData.TargetSpec.ToUpdate(),
	}
	if err := r.config.OVHClient.PutWithContext(ctx, endpoint+vacPolicyForceQuery(&planData), payload, nil); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error calling Put %s", endpoint), err.Error())
		return
	}

	if err := helpers.WaitForAPIv2ResourceStatusReady(ctx, r.config.OVHClient, endpoint); err != nil {
		resp.Diagnostics.AddError("Error waiting for VAC policy to be ready", err.Error())
		return
	}

	if err := r.config.OVHClient.GetWithContext(ctx, endpoint, &responseData); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error calling Get %s", endpoint), err.Error())
		return
	}

	planData.Id = data.Id
	responseData.MergeWith(&planData)
	resp.Diagnostics.Append(resp.State.Set(ctx, &responseData)...)
}

func (r *networkDefenseVacPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NetworkDefenseVacPolicyModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := vacPolicyEndpoint(&data)
	if err := r.config.OVHClient.DeleteWithContext(ctx, endpoint, nil); err != nil {
		if helpers.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError(fmt.Sprintf("Error calling Delete %s", endpoint), err.Error())
		return
	}

	// Deletion is async: the policy stays in DELETING until the API returns 404.
	if err := helpers.WaitForAPIv2ResourceDeleted(ctx, r.config.OVHClient, endpoint); err != nil {
		resp.Diagnostics.AddError("Error waiting for VAC policy deletion", err.Error())
	}
}
