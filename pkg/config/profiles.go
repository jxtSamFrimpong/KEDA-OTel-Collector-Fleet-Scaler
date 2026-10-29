package config

import "time"

// FleetProfile names match docs/fleet-profiles.md.
const (
	ProfileForward  = "forward"
	ProfileStateful = "stateful"
)

// ProfilePreset captures orchestration defaults for a fleet profile.
type ProfilePreset struct {
	Name                           string
	WorkloadKind                   string // Deployment | StatefulSet
	PreStopSeconds                 int
	TerminationGracePeriodSeconds  int
	CooldownPeriodSeconds          int
	ScaleUpStabilizationSeconds    int
	ScaleDownStabilizationSeconds  int
	ScaleUpPodsPerPeriod           int
	ScaleUpPeriodSeconds           int
	ScaleDownPodsPerPeriod         int
	ScaleDownPeriodSeconds         int
	RecommendedMinReplicas         int
}

// ProfilePresets returns built-in forward/stateful presets.
func ProfilePresets() map[string]ProfilePreset {
	return map[string]ProfilePreset{
		ProfileForward: {
			Name:                          ProfileForward,
			WorkloadKind:                  "Deployment",
			PreStopSeconds:                15,
			TerminationGracePeriodSeconds: 30,
			CooldownPeriodSeconds:         60,
			ScaleUpStabilizationSeconds:   30,
			ScaleDownStabilizationSeconds: 60,
			ScaleUpPodsPerPeriod:          2,
			ScaleUpPeriodSeconds:          60,
			ScaleDownPodsPerPeriod:        1,
			ScaleDownPeriodSeconds:        60,
			RecommendedMinReplicas:        2,
		},
		ProfileStateful: {
			Name:                          ProfileStateful,
			WorkloadKind:                  "StatefulSet",
			PreStopSeconds:                35,
			TerminationGracePeriodSeconds: 60,
			CooldownPeriodSeconds:         300,
			ScaleUpStabilizationSeconds:   60,
			ScaleDownStabilizationSeconds: 300,
			ScaleUpPodsPerPeriod:          1,
			ScaleUpPeriodSeconds:          120,
			ScaleDownPodsPerPeriod:        1,
			ScaleDownPeriodSeconds:        180,
			RecommendedMinReplicas:        2,
		},
	}
}

// GetProfile returns a preset or false if unknown.
func GetProfile(name string) (ProfilePreset, bool) {
	p, ok := ProfilePresets()[name]
	return p, ok
}

// DrainDuration is the preStop sleep for the profile.
func (p ProfilePreset) DrainDuration() time.Duration {
	return time.Duration(p.PreStopSeconds) * time.Second
}
