package bootstrap

import (
	"fmt"
	"strings"

	t "github.com/scott-walker/mememory/internal/types"
)

// Format renders a list of memories as Markdown for session initialization.
// When project is non-empty, each memory is prefixed with its scope label
// (e.g. [global], [project/match]) so the agent can distinguish provenance.
func Format(project string, memories []t.Memory) string {
	if len(memories) == 0 {
		return ""
	}

	groups := map[t.MemoryType][]t.Memory{}
	for _, m := range memories {
		groups[m.Type] = append(groups[m.Type], m)
	}

	typeOrder := []struct {
		typ   t.MemoryType
		label string
	}{
		{t.TypeRule, "Rules"},
		{t.TypeFeedback, "Feedback"},
		{t.TypeFact, "Facts"},
		{t.TypeDecision, "Decisions"},
		{t.TypeContext, "Context"},
	}

	var b strings.Builder

	b.WriteString("# Memory Bootstrap\n\n")
	b.WriteString("The following memories were loaded from persistent storage. Apply rules and feedback to your behavior.\n\n")
	if project != "" {
		fmt.Fprintf(&b, "Project: %s\n\n", project)
	}

	for _, to := range typeOrder {
		mems := groups[to.typ]
		if len(mems) == 0 {
			continue
		}

		fmt.Fprintf(&b, "## %s\n\n", to.label)
		for _, m := range mems {
			if project != "" {
				fmt.Fprintf(&b, "- [%s] %s\n", scopeLabel(m), m.Content)
			} else {
				fmt.Fprintf(&b, "- %s\n", m.Content)
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}

func scopeLabel(m t.Memory) string {
	s := string(m.Scope)
	if m.Project != "" {
		s += "/" + m.Project
	}
	if m.Persona != "" {
		s += "/" + m.Persona
	}
	return s
}
