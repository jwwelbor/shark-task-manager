package runner

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// B02: a causal site first discovered at consolidation is accepted only with a
// fresh inspection by the responsible angle, on the frozen tree and exact range,
// whose observation the reconsolidated decision carries.
func TestBugDiscoveriesNeedResponsibleAngleInspection(t *testing.T) {
	scope := bugCandidateScope{CandidateTree: "tree"}
	site := bugSite{File: "internal/foo.go", StartLine: 10, EndLine: 12, Observation: "callback skips registration in the error branch"}
	inspection := bugSiteInspection{bugSite: site, Tree: "tree", AssignedAngle: "a", ResponderName: "a"}
	decisions := func(discovery bugDiscovery) []byte {
		return mustJSON(t, map[string]any{"candidate_identity": bugReviewIdentity{}, "findings": []any{}, "discoveries": []bugDiscovery{discovery}})
	}
	good := bugDiscovery{bugSite: site, Angle: "a"}
	angles := func(in ...bugSiteInspection) map[string]bugAngle {
		return map[string]bugAngle{"a": {SiteInspections: in}, "e": {}}
	}
	require.NoError(t, verifyBugDiscoveries(decisions(good), angles(inspection), scope))
	require.NoError(t, verifyBugDiscoveries(mustJSON(t, map[string]any{"findings": []any{}}), angles(), scope), "no discovery, no obligation")

	for name, trial := range map[string]struct {
		discovery bugDiscovery
		angles    map[string]bugAngle
		want      string
	}{
		"withheld reply":        {good, angles(), "lacks a fresh inspection by responsible angle a"},
		"neighboring range":     {good, angles(func() bugSiteInspection { i := inspection; i.StartLine, i.EndLine = 20, 22; return i }()), "lacks a fresh inspection"},
		"different tree":        {good, angles(func() bugSiteInspection { i := inspection; i.Tree = "other"; return i }()), "frozen tree"},
		"different file":        {good, angles(func() bugSiteInspection { i := inspection; i.File = "internal/bar.go"; return i }()), "lacks a fresh inspection"},
		"answered by angle e":   {good, angles(func() bugSiteInspection { i := inspection; i.ResponderName = "e"; return i }()), "lacks a fresh inspection"},
		"reply under wrong ang": {bugDiscovery{bugSite: site, Angle: "a"}, map[string]bugAngle{"e": {SiteInspections: []bugSiteInspection{inspection}}}, "lacks a fresh inspection"},
		"observation removed":   {bugDiscovery{bugSite: bugSite{File: site.File, StartLine: 10, EndLine: 12, Observation: " "}, Angle: "a"}, angles(inspection), "malformed"},
		"observation replaced":  {bugDiscovery{bugSite: bugSite{File: site.File, StartLine: 10, EndLine: 12, Observation: "a different plausible observation"}, Angle: "a"}, angles(inspection), "does not carry angle a's follow-up observation"},
		"unknown angle":         {bugDiscovery{bugSite: site, Angle: "z"}, angles(inspection), "malformed"},
	} {
		t.Run(name, func(t *testing.T) {
			require.ErrorContains(t, verifyBugDiscoveries(decisions(trial.discovery), trial.angles, scope), trial.want)
		})
	}
}
