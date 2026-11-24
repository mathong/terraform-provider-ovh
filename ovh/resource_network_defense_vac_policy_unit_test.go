package ovh

import (
	"encoding/json"
	"strings"
	"testing"

	ovhtypes "github.com/ovh/terraform-provider-ovh/v2/ovh/types"
)

func TestVacPolicyTargetSpecToUpdateSendsEnabled(t *testing.T) {
	spec := VacPolicyTargetSpecValue{Enabled: ovhtypes.NewTfBoolValue(false)}

	body, err := json.Marshal(spec.ToUpdate())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"enabled":false`) {
		t.Errorf("expected enabled=false in the PUT body, got %s", body)
	}
}
