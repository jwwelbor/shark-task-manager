package runner

import (
	"encoding/json"
	"fmt"
	"strings"
)

// bugSite is an anchored path and line range.
type bugSite struct {
	File        reviewGitPath `json:"file"`
	StartLine   int           `json:"start_line"`
	EndLine     int           `json:"end_line"`
	Observation string        `json:"observation"`
}

// bugDiscovery is a causal site first found by the consolidator.
type bugDiscovery struct {
	bugSite
	Angle string `json:"angle"`
}

// bugSiteInspection is a responsible angle's fresh inspection of a discovered
// site. The runner, not the packet, records which angle was assigned and which
// actually answered.
type bugSiteInspection struct {
	bugSite
	Tree          string `json:"tree"`
	AssignedAngle string `json:"assigned_angle"`
	ResponderName string `json:"responder_angle"`
}

func (s bugSite) valid() bool {
	return s.File != "" && s.StartLine > 0 && s.EndLine >= s.StartLine && strings.TrimSpace(s.Observation) != ""
}

// verifyBugDiscoveries requires a fresh inspection by the responsible angle,
// on the frozen tree and exact range, for every consolidation discovery, and
// that the reconsolidated decision carries the observation that reply delivered.
func verifyBugDiscoveries(decisionData []byte, angles map[string]bugAngle, scope bugCandidateScope) error {
	var packet struct {
		Discoveries []bugDiscovery `json:"discoveries"`
	}
	if err := json.Unmarshal(decisionData, &packet); err != nil {
		return fmt.Errorf("invalid review decisions: %w", err)
	}
	for _, discovery := range packet.Discoveries {
		if !discovery.valid() || strings.Count("abcdef", discovery.Angle) != 1 || len(discovery.Angle) != 1 {
			return fmt.Errorf("consolidation discovery is malformed")
		}
		label := fmt.Sprintf("%s:%d-%d", discovery.File, discovery.StartLine, discovery.EndLine)
		matched, carried := false, false
		for _, probe := range angles[discovery.Angle].SiteInspections {
			if !probe.valid() || probe.File != discovery.File || probe.StartLine != discovery.StartLine || probe.EndLine != discovery.EndLine ||
				probe.Tree != scope.CandidateTree || probe.AssignedAngle != discovery.Angle || probe.ResponderName != discovery.Angle {
				continue
			}
			matched = true
			carried = carried || probe.Observation == discovery.Observation
		}
		if !matched {
			return fmt.Errorf("discovered site %s lacks a fresh inspection by responsible angle %s on the frozen tree", label, discovery.Angle)
		}
		if !carried {
			return fmt.Errorf("reconsolidation for %s does not carry angle %s's follow-up observation", label, discovery.Angle)
		}
	}
	return nil
}
