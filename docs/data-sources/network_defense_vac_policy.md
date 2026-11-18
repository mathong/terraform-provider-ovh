---
subcategory : "Network Defense (VAC)"
---

# ovh_network_defense_vac_policy (Data Source)

Use this data source to read a policy of a VAC DDoS protection resource.

## Example Usage

```terraform
data "ovh_network_defense_vac_policy" "prod_webservers" {
  resource_name = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  policy_id     = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}
```

## Argument Reference

- `resource_name` (String) ID of the VAC resource that holds the policy
- `policy_id` (String) ID of the policy

## Attributes Reference

- `id` (String) ID of the policy
- `checksum` (String) Checksum of the current configuration, used to detect concurrent changes
- `created_at` (String) Date and time the policy was created
- `updated_at` (String) Date and time the policy was last updated
- `resource_status` (String) Current status of the policy, for example `CREATING`, `READY`, `DELETING` or `ERROR`
- `error_message` (String) Details of the last error, set when the status is `ERROR`
- `target_spec` (Attributes) Configuration you requested. It differs from `current_state` while a change is being applied.
  - `enabled` (Boolean) Whether the policy is active
  - `name` (String) Name of the policy
  - `profile_id` (String) ID of the protection profile
  - `sensitivity` (String) Protection sensitivity level
  - `targets` (Attributes List) Targets the policy protects
    - `prefix` (Attributes) The IP range
      - `cidr` (String) CIDR notation
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
