package permission

import "testing"

func TestAllowDecision(t *testing.T) {
	d := Allow("looks safe")
	if !d.Allowed() {
		t.Error("expected Allow decision to be allowed")
	}
	if d.Reason() != "looks safe" {
		t.Errorf("unexpected reason: %q", d.Reason())
	}
}

func TestDenyDecision(t *testing.T) {
	d := Deny("not permitted")
	if d.Allowed() {
		t.Error("expected Deny decision to not be allowed")
	}
	if d.Reason() != "not permitted" {
		t.Errorf("unexpected reason: %q", d.Reason())
	}
}

func TestPermissionModes(t *testing.T) {
	modes := []Mode{
		ModeDefault,
		ModeAcceptEdits,
		ModePlan,
		ModeBypassPermissions,
		ModeDontAsk,
		ModeAuto,
	}
	for _, m := range modes {
		if m == "" {
			t.Error("mode must not be empty string")
		}
	}
}

func TestCanUseToolFunc(t *testing.T) {
	var fn CanUseToolFunc = func(toolName string, input map[string]any) (Decision, error) {
		if toolName == "bash" {
			return Deny("bash not allowed"), nil
		}
		return Allow(""), nil
	}

	d, err := fn("bash", nil)
	if err != nil {
		t.Fatal(err)
	}
	if d.Allowed() {
		t.Error("expected bash to be denied")
	}

	d, err = fn("read", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !d.Allowed() {
		t.Error("expected read to be allowed")
	}
}
