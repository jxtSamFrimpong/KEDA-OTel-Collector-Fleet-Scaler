package config

import "testing"

func TestGetProfile(t *testing.T) {
	fwd, ok := GetProfile(ProfileForward)
	if !ok || fwd.WorkloadKind != "Deployment" || fwd.PreStopSeconds != 15 {
		t.Fatalf("forward=%+v ok=%v", fwd, ok)
	}
	st, ok := GetProfile(ProfileStateful)
	if !ok || st.WorkloadKind != "StatefulSet" || st.PreStopSeconds != 35 {
		t.Fatalf("stateful=%+v ok=%v", st, ok)
	}
	if _, ok := GetProfile("nope"); ok {
		t.Fatal("expected missing profile")
	}
}
