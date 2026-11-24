package ovh

import (
	"context"
	"fmt"
	"log"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/ovh/go-ovh/ovh"
	"github.com/ovh/terraform-provider-ovh/v2/ovh/helpers"
	ovhtypes "github.com/ovh/terraform-provider-ovh/v2/ovh/types"
)

var _ resource.ResourceWithConfigure = (*networkDefenseVacResourceResource)(nil)
var _ resource.ResourceWithImportState = (*networkDefenseVacResourceResource)(nil)

func NewNetworkDefenseVacResourceResource() resource.Resource {
	return &networkDefenseVacResourceResource{}
}

type networkDefenseVacResourceResource struct {
	config *Config
}

func vacResourceEndpoint(id string) string {
	return "/v2/networkDefense/vac/resource/" + url.PathEscape(id)
}

// plan[].configuration is Optional+Computed and the API never returns it, so when the
// config omits it Terraform rejects the unknown value left after apply. Null is handled
// too, for comparing a config value with state.
func vacResolvePlanConfiguration(ctx context.Context, plans ovhtypes.TfListNestedValue[PlanValue]) (ovhtypes.TfListNestedValue[PlanValue], diag.Diagnostics) {
	var elems []PlanValue
	diags := plans.ElementsAs(ctx, &elems, false)
	if diags.HasError() {
		return plans, diags
	}

	values := make([]attr.Value, 0, len(elems))
	for _, p := range elems {
		if p.Configuration.IsUnknown() || p.Configuration.IsNull() {
			p.Configuration = ovhtypes.TfListNestedValue[PlanConfigurationValue]{
				ListValue: basetypes.NewListValueMust(PlanConfigurationValue{}.Type(ctx), []attr.Value{}),
			}
		}
		values = append(values, p)
	}

	return ovhtypes.TfListNestedValue[PlanValue]{
		ListValue: basetypes.NewListValueMust(PlanValue{}.Type(ctx), values),
	}, diags
}

func (r *networkDefenseVacResourceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_defense_vac_resource"
}

func (r *networkDefenseVacResourceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *networkDefenseVacResourceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := NetworkDefenseVacResourceResourceSchema(ctx)

	// On update the framework plans unconfigured computed values as unknown. An unknown id
	// replaces every policy on vac.id, and an unknown plan[].configuration replaces the VAC.
	id := s.Attributes["id"].(schema.StringAttribute)
	id.PlanModifiers = append(id.PlanModifiers, stringplanmodifier.UseStateForUnknown())
	s.Attributes["id"] = id

	createdAt := s.Attributes["created_at"].(schema.StringAttribute)
	createdAt.PlanModifiers = append(createdAt.PlanModifiers, stringplanmodifier.UseStateForUnknown())
	s.Attributes["created_at"] = createdAt

	// RequiresReplaceIfConfigured runs before the nested UseStateForUnknown below, so it
	// would see configuration as unknown. Compare the config with state instead.
	plan := s.Attributes["plan"].(schema.ListNestedAttribute)
	plan.PlanModifiers = []planmodifier.List{listplanmodifier.RequiresReplaceIf(
		vacPlanRequiresReplace,
		"Changing plan forces a new VAC to be ordered.",
		"Changing plan forces a new VAC to be ordered.",
	)}
	cfg := plan.NestedObject.Attributes["configuration"].(schema.ListNestedAttribute)
	cfg.PlanModifiers = append(cfg.PlanModifiers, listplanmodifier.UseStateForUnknown())
	plan.NestedObject.Attributes["configuration"] = cfg
	s.Attributes["plan"] = plan

	resp.Schema = s
}

func vacPlanRequiresReplace(ctx context.Context, req planmodifier.ListRequest, resp *listplanmodifier.RequiresReplaceIfFuncResponse) {
	if req.ConfigValue.IsNull() {
		return
	}
	config, diags := vacResolvePlanConfiguration(ctx, ovhtypes.TfListNestedValue[PlanValue]{ListValue: req.ConfigValue})
	resp.Diagnostics.Append(diags...)
	state, diags := vacResolvePlanConfiguration(ctx, ovhtypes.TfListNestedValue[PlanValue]{ListValue: req.StateValue})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.RequiresReplace = !config.Equal(state)
}

func (r *networkDefenseVacResourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *networkDefenseVacResourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NetworkDefenseVacResourceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var diags diag.Diagnostics
	data.Plan, diags = vacResolvePlanConfiguration(ctx, data.Plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	order := data.ToOrder()
	if err := orderCreate(order, r.config, "vac", true, defaultOrderTimeout); err != nil {
		resp.Diagnostics.AddError("failed to create order", err.Error())
		return
	}
	// orderCreate fills the copy returned by ToOrder, not data.
	data.Order = OrderValue{
		state:   attr.ValueStateKnown,
		OrderId: order.Order.OrderId,
	}

	plans := []PlanValue{}
	resp.Diagnostics.Append(data.Plan.ElementsAs(ctx, &plans, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The order detail's domain holds the VAC resource UUID.
	id, err := serviceNameFromOrder(r.config.OVHClient, order.Order.OrderId.ValueInt64(), plans[0].PlanCode.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("failed to retrieve service name", err.Error())
		return
	}
	data.Id = ovhtypes.NewTfStringValue(id)

	// The VAC is paid for now: save the ID so a later failure does not lead the next apply to order a second one.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The order can be delivered before the APIv2 resource accepts writes.
	endpoint := vacResourceEndpoint(id)
	if err := helpers.WaitForAPIv2ResourceStatusReady(ctx, r.config.OVHClient, endpoint); err != nil {
		resp.Diagnostics.AddError("Error waiting for resource to be ready", err.Error())
		return
	}

	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		if err := r.config.OVHClient.PutWithContext(ctx, endpoint, data.ToUpdate(), nil); err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Error calling Put %s", endpoint), err.Error())
			return
		}
		if err := helpers.WaitForAPIv2ResourceStatusReady(ctx, r.config.OVHClient, endpoint); err != nil {
			resp.Diagnostics.AddError("Error waiting for resource to be ready", err.Error())
			return
		}
	}

	var responseData NetworkDefenseVacResourceResourceModel
	if err := r.config.OVHClient.GetWithContext(ctx, endpoint, &responseData); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error calling Get %s", endpoint), err.Error())
		return
	}

	data.MergeWith(&responseData)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *networkDefenseVacResourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data, responseData NetworkDefenseVacResourceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := vacResourceEndpoint(data.Id.ValueString())
	if err := r.config.OVHClient.GetWithContext(ctx, endpoint, &responseData); err != nil {
		if helpers.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(fmt.Sprintf("Error calling Get %s", endpoint), err.Error())
		return
	}

	// Order fields are not returned by the API: keep them from state.
	responseData.MergeWith(&data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &responseData)...)
}

func (r *networkDefenseVacResourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, planData, responseData NetworkDefenseVacResourceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var diags diag.Diagnostics
	planData.Plan, diags = vacResolvePlanConfiguration(ctx, planData.Plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := vacResourceEndpoint(data.Id.ValueString())
	if err := r.config.OVHClient.PutWithContext(ctx, endpoint, planData.ToUpdate(), nil); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error calling Put %s", endpoint), err.Error())
		return
	}

	if err := helpers.WaitForAPIv2ResourceStatusReady(ctx, r.config.OVHClient, endpoint); err != nil {
		resp.Diagnostics.AddError("Error waiting for resource to be ready", err.Error())
		return
	}

	if err := r.config.OVHClient.GetWithContext(ctx, endpoint, &responseData); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error calling Get %s", endpoint), err.Error())
		return
	}

	responseData.MergeWith(&planData)
	responseData.MergeWith(&data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &responseData)...)
}

func (r *networkDefenseVacResourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NetworkDefenseVacResourceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := data.Id.ValueString()
	serviceID, err := serviceIdFromResourceName(r.config.OVHClient, id)
	if err != nil {
		resp.Diagnostics.AddError("failed to get service id to terminate service", err.Error())
		return
	}

	terminate := func() (string, error) {
		log.Printf("[DEBUG] Will terminate VAC %s (service %d)", id, serviceID)
		endpoint := fmt.Sprintf("/services/%d/terminate", serviceID)
		if err := r.config.OVHClient.Post(endpoint, nil, nil); err != nil {
			if errOvh, ok := err.(*ovh.APIError); ok && (errOvh.Code == 404 || errOvh.Code == 460) {
				return "", nil
			}
			return "", fmt.Errorf("calling Post %s:\n\t %q", endpoint, err)
		}
		return id, nil
	}

	confirmTerminate := func(token string) error {
		log.Printf("[DEBUG] Will confirm termination of VAC %s (service %d)", id, serviceID)
		endpoint := fmt.Sprintf("/services/%d/terminate/confirm", serviceID)
		if err := r.config.OVHClient.Post(endpoint, &ConfirmTerminationOpts{Token: token}, nil); err != nil {
			return fmt.Errorf("calling Post %s:\n\t %q", endpoint, err)
		}
		return nil
	}

	if err := orderDelete(r.config, terminate, confirmTerminate); err != nil {
		resp.Diagnostics.AddError("failed to delete resource", err.Error())
	}
}
