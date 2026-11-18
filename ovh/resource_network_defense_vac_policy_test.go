package ovh

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/ovh/terraform-provider-ovh/v2/ovh/helpers"
)

func init() {
	resource.AddTestSweepers("ovh_network_defense_vac_policy", &resource.Sweeper{
		Name: "ovh_network_defense_vac_policy",
		F:    testSweepNetworkDefenseVacPolicy,
	})
}

// testSweepNetworkDefenseVacPolicy deletes the policies an aborted acceptance run
// left on OVH_VAC_RESOURCE. A leftover policy keeps OVH_VAC_PREFIX, so the next run gets a 409.
func testSweepNetworkDefenseVacPolicy(region string) error {
	client, err := sharedClientForRegion(region)
	if err != nil {
		return fmt.Errorf("error getting client: %s", err)
	}

	resourceName := os.Getenv("OVH_VAC_RESOURCE")
	if resourceName == "" {
		return nil
	}

	type vacPolicy struct {
		Id         string `json:"id"`
		TargetSpec *struct {
			Name string `json:"name"`
		} `json:"targetSpec"`
	}

	endpoint := "/v2/networkDefense/vac/resource/" + url.PathEscape(resourceName) + "/policy"
	policies, err := helpers.GetAllPagesV2[vacPolicy](context.Background(), client, endpoint)
	if err != nil {
		return fmt.Errorf("error listing VAC policies: %s", err)
	}

	for _, p := range policies {
		if p.TargetSpec == nil || !strings.HasPrefix(p.TargetSpec.Name, test_prefix) {
			continue
		}
		policyEndpoint := endpoint + "/" + url.PathEscape(p.Id)
		if err := client.Delete(policyEndpoint, nil); err != nil {
			return fmt.Errorf("error deleting VAC policy %s: %s", p.Id, err)
		}
		// The policy holds OVH_VAC_PREFIX until the async deletion ends.
		if err := helpers.WaitForAPIv2ResourceDeleted(context.Background(), client, policyEndpoint); err != nil {
			return fmt.Errorf("error waiting for VAC policy %s deletion: %s", p.Id, err)
		}
	}

	return nil
}

func testAccNetworkDefenseVacPolicyConfig(name, sensitivity string, enabled bool) string {
	return fmt.Sprintf(`
resource "ovh_network_defense_vac_policy" "policy" {
  resource_name = "%s"
  target_spec = {
    name        = "%s"
    enabled     = %t
    profile_id  = "%s"
    sensitivity = "%s"
    targets = [
      { prefix = { cidr = "%s" } }
    ]
  }
}

data "ovh_network_defense_vac_policy" "policy" {
  resource_name = ovh_network_defense_vac_policy.policy.resource_name
  policy_id     = ovh_network_defense_vac_policy.policy.id
}
`, os.Getenv("OVH_VAC_RESOURCE"), name, enabled, os.Getenv("OVH_VAC_PROFILE"), sensitivity, os.Getenv("OVH_VAC_PREFIX"))
}

func TestAccResourceNetworkDefenseVacPolicy_basic(t *testing.T) {
	name := acctest.RandomWithPrefix(test_prefix)
	res := "ovh_network_defense_vac_policy.policy"
	data := "data.ovh_network_defense_vac_policy.policy"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckNetworkDefenseVac(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// The update step switches MEDIUM to HIGH, so fail before creating anything
			// if OVH_VAC_PROFILE does not offer every sensitivity.
			{
				Config: fmt.Sprintf(`
data "ovh_network_defense_vac_profile" "profile" {
  profile_id = "%s"
}
`, os.Getenv("OVH_VAC_PROFILE")),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckTypeSetElemAttr("data.ovh_network_defense_vac_profile.profile", "sensitivities.*", "LOW"),
					resource.TestCheckTypeSetElemAttr("data.ovh_network_defense_vac_profile.profile", "sensitivities.*", "MEDIUM"),
					resource.TestCheckTypeSetElemAttr("data.ovh_network_defense_vac_profile.profile", "sensitivities.*", "HIGH"),
				),
			},
			{
				Config: testAccNetworkDefenseVacPolicyConfig(name, "MEDIUM", true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(res, "id"),
					resource.TestCheckResourceAttr(res, "resource_status", "READY"),
					resource.TestCheckResourceAttr(res, "current_state.name", name),
					resource.TestCheckResourceAttr(res, "current_state.sensitivity", "MEDIUM"),
					resource.TestCheckResourceAttr(res, "current_state.targets.0.prefix.cidr", os.Getenv("OVH_VAC_PREFIX")),
					resource.TestCheckResourceAttrPair(data, "id", res, "id"),
					resource.TestCheckResourceAttr(data, "target_spec.name", name),
				),
			},
			{
				Config: testAccNetworkDefenseVacPolicyConfig(name, "HIGH", true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(res, "resource_status", "READY"),
					resource.TestCheckResourceAttr(res, "current_state.sensitivity", "HIGH"),
				),
			},
			{
				Config: testAccNetworkDefenseVacPolicyConfig(name, "HIGH", false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(res, "resource_status", "READY"),
					resource.TestCheckResourceAttr(res, "target_spec.enabled", "false"),
					resource.TestCheckResourceAttr(res, "current_state.enabled", "false"),
				),
			},
			{
				ResourceName:      res,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources[res]
					return rs.Primary.Attributes["resource_name"] + "/" + rs.Primary.Attributes["id"], nil
				},
				ImportStateVerifyIdentifierAttribute: "id",
				ImportStateVerifyIgnore:              []string{"force"},
			},
		},
	})
}
