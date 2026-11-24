package ovh

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// The policy runs on the ordered VAC, so the test needs no existing VAC.
func testAccNetworkDefenseVacResourceConfig(name string) string {
	return fmt.Sprintf(`
data "ovh_me" "myaccount" {}

resource "ovh_network_defense_vac_resource" "vac" {
  name           = "%s"
  ovh_subsidiary = data.ovh_me.myaccount.ovh_subsidiary
  plan = [
    {
      plan_code    = "vac"
      duration     = "P1M"
      pricing_mode = "default"
    }
  ]
}

resource "ovh_network_defense_vac_policy" "policy" {
  resource_name = ovh_network_defense_vac_resource.vac.id
  target_spec = {
    name        = "testacc-vac-policy"
    profile_id  = "%s"
    sensitivity = "MEDIUM"
    targets = [
      { prefix = { cidr = "%s" } }
    ]
  }
}

data "ovh_network_defense_vac_resource" "vac" {
  resource_name = ovh_network_defense_vac_resource.vac.id
}

data "ovh_network_defense_vac_profile" "profile" {
  profile_id = ovh_network_defense_vac_policy.policy.target_spec.profile_id
}

data "ovh_network_defense_vac_policy" "policy" {
  resource_name = ovh_network_defense_vac_policy.policy.resource_name
  policy_id     = ovh_network_defense_vac_policy.policy.id
}
`, name, os.Getenv("OVH_VAC_PROFILE"), os.Getenv("OVH_VAC_PREFIX"))
}

func TestAccResourceNetworkDefenseVacResource_basic(t *testing.T) {
	const res = "ovh_network_defense_vac_resource.vac"
	const policy = "ovh_network_defense_vac_policy.policy"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckOrderNetworkDefenseVac(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkDefenseVacResourceConfig("testacc-vac"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(res, "id", regexp.MustCompile(`^[0-9a-f-]{36}$`)),
					resource.TestCheckResourceAttr(res, "name", "testacc-vac"),
					resource.TestCheckResourceAttr(res, "resource_status", "READY"),
					resource.TestCheckResourceAttrSet(res, "order.order_id"),
					resource.TestCheckResourceAttrPair("data.ovh_network_defense_vac_resource.vac", "id", res, "id"),
					resource.TestCheckResourceAttrSet("data.ovh_network_defense_vac_resource.vac", "iam.urn"),
					resource.TestCheckResourceAttrSet("data.ovh_network_defense_vac_profile.profile", "name"),
					resource.TestCheckResourceAttr(policy, "resource_status", "READY"),
					resource.TestCheckResourceAttr(policy, "current_state.sensitivity", "MEDIUM"),
					resource.TestCheckResourceAttr(policy, "current_state.targets.0.prefix.cidr", os.Getenv("OVH_VAC_PREFIX")),
					resource.TestCheckResourceAttrPair("data.ovh_network_defense_vac_policy.policy", "id", policy, "id"),
				),
			},
			{
				Config: testAccNetworkDefenseVacResourceConfig("testacc-vac-renamed"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(res, "name", "testacc-vac-renamed"),
					resource.TestCheckResourceAttr(res, "resource_status", "READY"),
				),
			},
			{
				ResourceName:            res,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"plan", "plan_option", "ovh_subsidiary", "order"},
			},
			{
				ResourceName:      policy,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources[policy]
					return rs.Primary.Attributes["resource_name"] + "/" + rs.Primary.Attributes["id"], nil
				},
				ImportStateVerifyIdentifierAttribute: "id",
				ImportStateVerifyIgnore:              []string{"force"},
			},
		},
	})
}
