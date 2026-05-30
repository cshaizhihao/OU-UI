package deploy

import (
	"strings"
	"testing"
)

func TestRenderUnitEnablesIPAccounting(t *testing.T) {
	unit, err := (RuntimeManager{RuntimeName: "xray", CommandArgs: []string{"run", "-config", "{configPath}"}}).renderUnit("/usr/bin/xray", runtimeLayout{
		ConfigDir:   "/var/lib/ou-ui-agent/runtimes/xray/active",
		ConfigPath:  "/var/lib/ou-ui-agent/runtimes/xray/active/nod_a.json",
		ServiceName: "ou-ui-xray-nod_a",
	})
	if err != nil {
		t.Fatalf("render unit: %v", err)
	}
	if !strings.Contains(unit, "IPAccounting=true") {
		t.Fatalf("managed runtime unit must enable systemd IP accounting:\n%s", unit)
	}
}
