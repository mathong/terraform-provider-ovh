package ovh

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	ovhtypes "github.com/ovh/terraform-provider-ovh/v2/ovh/types"
)

func TestNetworkDefenseVacResourceSchemaIsValid(t *testing.T) {
	ctx := context.Background()
	resp := &resource.SchemaResponse{}

	NewNetworkDefenseVacResourceResource().Schema(ctx, resource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Fatalf("invalid schema: %v", diags)
	}
	for _, attr := range []string{"ovh_subsidiary", "plan", "order", "name", "id"} {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("missing attribute %q", attr)
		}
	}
}

func TestVacResolvePlanConfigurationSetsUnknownToEmpty(t *testing.T) {
	ctx := context.Background()
	plan := PlanValue{
		state:         attr.ValueStateKnown,
		PlanCode:      ovhtypes.NewTfStringValue("vac"),
		Configuration: ovhtypes.TfListNestedValue[PlanConfigurationValue]{ListValue: basetypes.NewListUnknown(PlanConfigurationValue{}.Type(ctx))},
	}
	plans := ovhtypes.TfListNestedValue[PlanValue]{ListValue: basetypes.NewListValueMust(PlanValue{}.Type(ctx), []attr.Value{plan})}

	got, diags := vacResolvePlanConfiguration(ctx, plans)
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}

	var out []PlanValue
	if diags := got.ElementsAs(ctx, &out, false); diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}
	if len(out) != 1 || out[0].Configuration.IsUnknown() || len(out[0].Configuration.Elements()) != 0 {
		t.Fatalf("expected an empty known configuration, got %+v", out)
	}
	if out[0].PlanCode.ValueString() != "vac" {
		t.Errorf("plan_code changed: %q", out[0].PlanCode.ValueString())
	}
}

// nullObject returns typ with every attribute null, then applies set.
func nullObject(typ tftypes.Object, set map[string]tftypes.Value) tftypes.Value {
	vals := map[string]tftypes.Value{}
	for k, t := range typ.AttributeTypes {
		vals[k] = tftypes.NewValue(t, nil)
	}
	for k, v := range set {
		vals[k] = v
	}
	return tftypes.NewValue(typ, vals)
}

// A rename must update the VAC in place and keep its id known, so that policies
// pointing at vac.id are not replaced. A plan change must still replace the VAC.
func TestNetworkDefenseVacResourcePlanReplacement(t *testing.T) {
	ctx := context.Background()
	typ := NetworkDefenseVacResourceResourceSchema(ctx).Type().TerraformType(ctx).(tftypes.Object)
	planList := typ.AttributeTypes["plan"].(tftypes.List)
	planObj := planList.ElementType.(tftypes.Object)
	cfgList := planObj.AttributeTypes["configuration"].(tftypes.List)
	str := func(s string) tftypes.Value { return tftypes.NewValue(tftypes.String, s) }

	plan := func(duration string, cfg tftypes.Value) tftypes.Value {
		return tftypes.NewValue(planList, []tftypes.Value{nullObject(planObj, map[string]tftypes.Value{
			"plan_code":     str("vac"),
			"duration":      str(duration),
			"pricing_mode":  str("default"),
			"configuration": cfg,
		})})
	}
	emptyCfg := tftypes.NewValue(cfgList, []tftypes.Value{})
	nullCfg := tftypes.NewValue(cfgList, nil)

	dv := func(v tftypes.Value) *tfprotov6.DynamicValue {
		d, err := tfprotov6.NewDynamicValue(typ, v)
		if err != nil {
			t.Fatal(err)
		}
		return &d
	}

	prior := nullObject(typ, map[string]tftypes.Value{
		"id":         str("uuid"),
		"name":       str("old"),
		"created_at": str("2026-01-01T00:00:00Z"),
		"plan":       plan("P1M", emptyCfg),
	})

	for _, tc := range []struct {
		name, newName, duration string
		wantReplace             bool
	}{
		{"rename", "new", "P1M", false},
		{"duration change", "old", "P1Y", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := nullObject(typ, map[string]tftypes.Value{
				"name": str(tc.newName),
				"plan": plan(tc.duration, nullCfg),
			})
			// Terraform core fills null Optional+Computed config values from prior state.
			proposed := nullObject(typ, map[string]tftypes.Value{
				"id":         str("uuid"),
				"name":       str(tc.newName),
				"created_at": str("2026-01-01T00:00:00Z"),
				"plan":       plan(tc.duration, emptyCfg),
			})

			srv := providerserver.NewProtocol6(&OvhProvider{})()
			resp, err := srv.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
				TypeName:         "ovh_network_defense_vac_resource",
				PriorState:       dv(prior),
				ProposedNewState: dv(proposed),
				Config:           dv(config),
			})
			if err != nil {
				t.Fatal(err)
			}
			for _, d := range resp.Diagnostics {
				t.Errorf("diagnostic: %s: %s", d.Summary, d.Detail)
			}
			if got := len(resp.RequiresReplace) != 0; got != tc.wantReplace {
				t.Errorf("replace = %v (%v), want %v", got, resp.RequiresReplace, tc.wantReplace)
			}
			if tc.wantReplace {
				return
			}

			planned, err := resp.PlannedState.Unmarshal(typ)
			if err != nil {
				t.Fatal(err)
			}
			var attrs map[string]tftypes.Value
			if err := planned.As(&attrs); err != nil {
				t.Fatal(err)
			}
			if !attrs["id"].IsKnown() {
				t.Error("id is unknown after a rename")
			}
		})
	}
}

// Create fills order and id, so the plan must leave them unknown.
func TestNetworkDefenseVacResourceCreatePlansComputedUnknown(t *testing.T) {
	ctx := context.Background()
	typ := NetworkDefenseVacResourceResourceSchema(ctx).Type().TerraformType(ctx).(tftypes.Object)
	planList := typ.AttributeTypes["plan"].(tftypes.List)
	planObj := planList.ElementType.(tftypes.Object)
	str := func(s string) tftypes.Value { return tftypes.NewValue(tftypes.String, s) }

	config := nullObject(typ, map[string]tftypes.Value{
		"name": str("n"),
		"plan": tftypes.NewValue(planList, []tftypes.Value{nullObject(planObj, map[string]tftypes.Value{
			"plan_code":    str("vac"),
			"duration":     str("P1M"),
			"pricing_mode": str("default"),
		})}),
	})
	dv := func(v tftypes.Value) *tfprotov6.DynamicValue {
		d, err := tfprotov6.NewDynamicValue(typ, v)
		if err != nil {
			t.Fatal(err)
		}
		return &d
	}

	srv := providerserver.NewProtocol6(&OvhProvider{})()
	resp, err := srv.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
		TypeName:         "ovh_network_defense_vac_resource",
		PriorState:       dv(tftypes.NewValue(typ, nil)),
		ProposedNewState: dv(config),
		Config:           dv(config),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range resp.Diagnostics {
		t.Errorf("diagnostic: %s: %s", d.Summary, d.Detail)
	}

	planned, err := resp.PlannedState.Unmarshal(typ)
	if err != nil {
		t.Fatal(err)
	}
	var attrs map[string]tftypes.Value
	if err := planned.As(&attrs); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"order", "id", "created_at"} {
		if attrs[name].IsKnown() {
			t.Errorf("%s is known in the create plan: %v", name, attrs[name])
		}
	}
}
