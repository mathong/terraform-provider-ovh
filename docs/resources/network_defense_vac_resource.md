---
subcategory : "Network Defense (VAC)"
---

# ovh_network_defense_vac_resource (Resource)

Orders a VAC DDoS protection resource. A VAC resource groups protection policies, managed with `ovh_network_defense_vac_policy`.

## Important

-> **NOTE** The VAC plan is free. This resource is still created through the OVHcloud order workflow and deleted through the termination workflow. If your consumer key has restricted access rules, it must allow the routes listed in [Consumer key access rules for resources that place orders](../guides/order_consumer_key_access_rules), in addition to `/v2/networkDefense/*`.

-> **NOTE** VAC can be ordered in the EU and CA regions only.

## Example Usage

```terraform
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
```

## Argument Reference

### Required

- `ovh_subsidiary` (String) OVHcloud subsidiary
- `plan` (Attributes List) Product plan to order. Use `plan_code = "vac"`, `duration = "P1M"` and `pricing_mode = "default"`
  - `plan_code` (String) Plan code
  - `duration` (String) Service duration
  - `pricing_mode` (String) Pricing model identifier
  - `configuration` (Attributes List) Configuration to apply to the plan. VAC needs none

### Optional

- `name` (String) Name you chose for the resource
- `plan_option` (Attributes List) Product plan options. VAC has none

## Attributes Reference

- `id` (String) Unique identifier of the resource
- `resource_status` (String) Current status of the resource: `READY` or `DELETING`
- `created_at` (String) Date and time the resource was created
- `updated_at` (String) Date and time the resource was last updated
- `iam` (Attributes) IAM resource metadata
  - `id` (String) Unique identifier of the resource
  - `urn` (String) Unique resource name used in policies
  - `display_name` (String) Resource display name
  - `state` (String) Resource state
  - `tags` (Map of String) Resource tags. Tags computed internally are prefixed with `ovh:`
- `order` (Attributes) Details about the order used to create the resource

## Import

An existing VAC resource can be imported using its `id`:

```bash
$ terraform import ovh_network_defense_vac_resource.vac xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

The API does not return `ovh_subsidiary` and `plan`. Both force a new order when they change, so an imported resource whose configuration sets them plans a replacement. Ignore them after an import:

```terraform
resource "ovh_network_defense_vac_resource" "vac" {
  # ...
  lifecycle {
    ignore_changes = [ovh_subsidiary, plan, plan_option]
  }
}
```
