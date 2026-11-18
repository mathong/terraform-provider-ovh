package ovh

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceNetworkDefenseVacResourceAndProfile_basic(t *testing.T) {
	config := fmt.Sprintf(`
data "ovh_network_defense_vac_resource" "res" {
  resource_name = "%s"
}

data "ovh_network_defense_vac_profile" "profile" {
  profile_id = "%s"
}
`, os.Getenv("OVH_VAC_RESOURCE"), os.Getenv("OVH_VAC_PROFILE"))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckNetworkDefenseVac(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.ovh_network_defense_vac_resource.res", "id", os.Getenv("OVH_VAC_RESOURCE")),
					resource.TestCheckResourceAttrSet("data.ovh_network_defense_vac_resource.res", "iam.urn"),
					resource.TestCheckResourceAttr("data.ovh_network_defense_vac_profile.profile", "id", os.Getenv("OVH_VAC_PROFILE")),
					resource.TestCheckResourceAttrSet("data.ovh_network_defense_vac_profile.profile", "name"),
				),
			},
		},
	})
}
