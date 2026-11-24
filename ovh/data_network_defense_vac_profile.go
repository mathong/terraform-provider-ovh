package ovh

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

var _ datasource.DataSourceWithConfigure = (*networkDefenseVacProfileDataSource)(nil)

func NewNetworkDefenseVacProfileDataSource() datasource.DataSource {
	return &networkDefenseVacProfileDataSource{}
}

type networkDefenseVacProfileDataSource struct {
	config *Config
}

func (d *networkDefenseVacProfileDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_defense_vac_profile"
}

func (d *networkDefenseVacProfileDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *Config, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.config = config
}

func (d *networkDefenseVacProfileDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = NetworkDefenseVacProfileDataSourceSchema(ctx)
}

func (d *networkDefenseVacProfileDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data NetworkDefenseVacProfileModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read API call logic
	endpoint := "/v2/networkDefense/vac/reference/profile/" + url.PathEscape(data.ProfileId.ValueString()) + ""

	if err := d.config.OVHClient.GetWithContext(ctx, endpoint, &data); err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error calling Get %s", endpoint),
			err.Error(),
		)
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
