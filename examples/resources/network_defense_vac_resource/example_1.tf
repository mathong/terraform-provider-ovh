data "ovh_me" "myaccount" {}

resource "ovh_network_defense_vac_resource" "vac" {
  name           = "my-vac"
  ovh_subsidiary = data.ovh_me.myaccount.ovh_subsidiary
  plan = [
    {
      plan_code    = "vac"
      duration     = "P1M"
      pricing_mode = "default"
    }
  ]
}
