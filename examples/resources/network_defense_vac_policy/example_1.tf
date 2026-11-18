data "ovh_network_defense_vac_profile" "prod_webservers" {
  profile_id = "vac-profile-xxxxxxxx"
}

resource "ovh_network_defense_vac_policy" "prod_webservers" {
  resource_name = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"

  target_spec = {
    name        = "prod-webservers"
    enabled     = true
    profile_id  = data.ovh_network_defense_vac_profile.prod_webservers.id
    sensitivity = "MEDIUM"
    targets = [
      { prefix = { cidr = "192.0.2.10/32" } },
      { prefix = { cidr = "198.51.100.0/28" } },
    ]
  }
}
