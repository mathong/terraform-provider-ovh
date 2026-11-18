---
subcategory : "Network Defense (VAC)"
---

# ovh_network_defense_vac_profile (Data Source)

Use this data source to read a VAC protection profile. A profile is a read-only protection configuration for a workload type, for example a DNS server or a web server.

## Example Usage

```terraform
data "ovh_network_defense_vac_profile" "prod_webservers" {
  profile_id = "vac-profile-xxxxxxxx"
}
```

## Argument Reference

- `profile_id` (String) ID of the profile

## Attributes Reference

- `id` (String) Unique identifier of the profile
- `name` (String) Profile name, for example `DNS server`
- `description` (String) Intended use case of the profile
- `sensitivities` (List of String) Sensitivity levels the profile offers. A policy's `sensitivity` must be one of them.
- `state` (String) Lifecycle state of the profile: `ACTIVE`, `BETA`, `DEPRECATED` or `RETIRED`. Only `RETIRED` blocks new policies.
