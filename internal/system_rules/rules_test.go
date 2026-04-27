package system_rules

import (
	"reflect"
	"testing"
)

func TestSelect_DeterministicWithSeed(t *testing.T) {
	a := Select(42)
	b := Select(42)
	if !reflect.DeepEqual(a, b) {
		t.Errorf("same seed should produce same Selected\n a=%+v\n b=%+v", a, b)
	}
}

func TestSelect_AllMetaRulesPresent(t *testing.T) {
	sel := Select(1)
	if len(sel.MetaRules) != len(MetaRules) {
		t.Fatalf("Selected has %d rules, want %d", len(sel.MetaRules), len(MetaRules))
	}

	got := make(map[string]bool, len(sel.MetaRules))
	for _, r := range sel.MetaRules {
		got[r.ID] = true
	}
	for _, want := range MetaRules {
		if !got[want.ID] {
			t.Errorf("Selected missing meta-rule with ID %q", want.ID)
		}
	}
}

func TestSelect_FrameOpenFromVariants(t *testing.T) {
	sel := Select(7)
	if !contains(FrameOpenVariants, sel.FrameOpen) {
		t.Errorf("FrameOpen %q not found in FrameOpenVariants", sel.FrameOpen)
	}
}

func TestSelect_FrameCloseFromVariants(t *testing.T) {
	sel := Select(7)
	if !contains(FrameCloseVariants, sel.FrameClose) {
		t.Errorf("FrameClose %q not found in FrameCloseVariants", sel.FrameClose)
	}
}

func TestSelect_MetaRuleTextFromOwnVariants(t *testing.T) {
	sel := Select(7)
	byID := make(map[string]MetaRule, len(MetaRules))
	for _, r := range MetaRules {
		byID[r.ID] = r
	}
	for _, sr := range sel.MetaRules {
		rule, ok := byID[sr.ID]
		if !ok {
			t.Errorf("Selected has unknown rule ID %q", sr.ID)
			continue
		}
		if !contains(rule.Variants, sr.Text) {
			t.Errorf("rule %q: text %q not in its own variants", sr.ID, sr.Text)
		}
	}
}

func TestSelect_DifferentSeedsProduceDifferentSelections(t *testing.T) {
	// Statistical sanity check: across many seeds, we should see at least
	// some variation in each rotation slot. If every Select is identical,
	// rotation is broken.
	first := Select(0)
	allSame := true
	for seed := int64(1); seed < 100; seed++ {
		next := Select(seed)
		if !reflect.DeepEqual(first, next) {
			allSame = false
			break
		}
	}
	if allSame {
		t.Error("Select produced identical output for 100 different seeds — rotation appears broken")
	}
}

func TestVariants_NoEmptyStrings(t *testing.T) {
	for i, v := range FrameOpenVariants {
		if v == "" {
			t.Errorf("FrameOpenVariants[%d] is empty", i)
		}
	}
	for i, v := range FrameCloseVariants {
		if v == "" {
			t.Errorf("FrameCloseVariants[%d] is empty", i)
		}
	}
	for _, rule := range MetaRules {
		if rule.ID == "" {
			t.Error("MetaRule has empty ID")
		}
		if len(rule.Variants) == 0 {
			t.Errorf("MetaRule %q has zero variants", rule.ID)
		}
		for i, v := range rule.Variants {
			if v == "" {
				t.Errorf("MetaRule %q variant %d is empty", rule.ID, i)
			}
		}
	}
}

func TestMetaRules_UniqueIDs(t *testing.T) {
	seen := make(map[string]bool, len(MetaRules))
	for _, rule := range MetaRules {
		if seen[rule.ID] {
			t.Errorf("duplicate MetaRule ID: %q", rule.ID)
		}
		seen[rule.ID] = true
	}
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
