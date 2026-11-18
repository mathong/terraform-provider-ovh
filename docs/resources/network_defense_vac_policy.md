---
subcategory : "Network Defense (VAC)"
---

# ovh_network_defense_vac_policy (Resource)

Creates a policy on a VAC DDoS protection resource. A policy applies a protection profile and a sensitivity level to a list of IP prefixes.

The VAC resource itself is ordered outside Terraform. Use the `ovh_network_defense_vac_resource` data source to read it.

## Example Usage

```terraform
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
```

## Argument Reference

### Required

- `resource_name` (String) ID of the VAC resource that holds the policy
- `target_spec` (Attributes) Protection configuration to apply to the policy (see [below for nested schema](#nestedatt--target_spec))

### Optional

- `force` (Boolean) Overwrite the protection settings that were not set via a policy on the targets this request adds. Without it, the API refuses such a request. The previous settings are lost and cannot be restored. The API does not store `force`, so it is never read back.

<a id="nestedatt--target_spec"></a>
### Nested Schema for `target_spec`

Required:

- `name` (String) Name of the policy. Renaming it does not change the protection being enforced.
- `profile_id` (String) ID of the protection profile, from the profile catalog. See the `ovh_network_defense_vac_profile` data source.
- `sensitivity` (String) Protection sensitivity level. Allowed values: `LOW`, `MEDIUM`, `HIGH`. The profile must offer it: see the `sensitivities` attribute of the `ovh_network_defense_vac_profile` data source.

Optional:

- `enabled` (Boolean) Whether the policy is active. Defaults to `true`. A disabled policy keeps its configuration but does not protect any traffic.
- `targets` (Attributes List) The complete set of targets to protect, at most 1000. Every change replaces the whole set: an empty list removes every target. A prefix can belong to only one policy. (see [below for nested schema](#nestedatt--target_spec--targets))

<a id="nestedatt--target_spec--targets"></a>
### Nested Schema for `target_spec.targets`

Required:

- `prefix` (Attributes) The IP range to protect
  - `cidr` (String) CIDR notation, for example `192.0.2.0/24`

## Attributes Reference

- `id` (String) ID of the policy
- `checksum` (String) Checksum of the current configuration, used to detect concurrent changes
- `created_at` (String) Date and time the policy was created
- `updated_at` (String) Date and time the policy was last updated
- `resource_status` (String) Current status of the policy, for example `CREATING`, `READY`, `DELETING` or `ERROR`
- `error_message` (String) Details of the last error, set when the status is `ERROR`
- `current_state` (Attributes) Configuration currently being enforced
  - `enabled` (Boolean) Whether the policy is active
  - `name` (String) Name of the policy
  - `profile_id` (String) ID of the protection profile
  - `profile_version` (Number) Version of the enforced configuration, increased each time a change is applied
  - `sensitivity` (String) Protection sensitivity level
  - `service_id` (String) ID of the VAC resource the policy belongs to
  - `targets` (Attributes List) Targets currently held by the policy. Also lists removed targets that are still being taken down.
    - `id` (String) ID of the target
    - `type` (String) Type of target (`PREFIX`)
    - `prefix` (Attributes) The IP range, set when the type is `PREFIX`
      - `cidr` (String) CIDR notation
    - `resolved` (Attributes List) The IP prefixes the target currently protects
      - `cidr` (String) CIDR notation
      - `status` (String) Current status of this prefix (`PENDING`, `ACTIVE` or `DELETING`)
- `iam` (Attributes) IAM resource metadata
  - `id` (String) Unique identifier of the resource
  - `urn` (String) Unique resource name used in policies
  - `display_name` (String) Resource display name
  - `state` (String) Resource state
  - `tags` (Map of String) Resource tags. Tags computed internally are prefixed with `ovh:`

## Import

A VAC policy can be imported using the `resource_name` and the policy `id`, separated by "/". E.g.,

```bash
$ terraform import ovh_network_defense_vac_policy.prod_webservers resource_name/policy_id
```

Using an import block (Terraform 1.5 and later):

```terraform
import {
  to = ovh_network_defense_vac_policy.prod_webservers
  id = "<resource_name>/<policy_id>"
}
```
