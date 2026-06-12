package grading

import (
	"fmt"
	"sort"
	"testing"
)

func TestComponent_Percent(t *testing.T) {
	tests := []struct {
		name string
		c    Component
		want float64
	}{
		{"half", Component{Score: 50, MaxScore: 100}, 0.5},
		{"full", Component{Score: 100, MaxScore: 100}, 1},
		{"zero score", Component{Score: 0, MaxScore: 100}, 0},
		{"zero max guards division", Component{Score: 10, MaxScore: 0}, 0},
		{"negative max guards division", Component{Score: 10, MaxScore: -5}, 0},
		{"over 100 percent", Component{Score: 150, MaxScore: 100}, 1.5},
		{"quarter", Component{Score: 25, MaxScore: 100}, 0.25},
		{"odd ratio", Component{Score: 3, MaxScore: 4}, 0.75},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.c.Percent(); got != tt.want {
				t.Errorf("Percent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRound2(t *testing.T) {
	tests := []struct {
		name string
		in   float64
		want float64
	}{
		{"integer unchanged", 82, 82},
		{"rounds up", 66.666, 66.67},
		{"rounds down", 66.664, 66.66},
		{"exact two decimals", 12.34, 12.34},
		{"zero", 0, 0},
		{"hundred", 100, 100},
		{"half rounds up", 50.005, 50.01},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := round2(tt.in); got != tt.want {
				t.Errorf("round2(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestAverage(t *testing.T) {
	tests := []struct {
		name       string
		components []Component
		want       float64
	}{
		{"empty is zero", nil, 0},
		{"single full", []Component{{Score: 100, MaxScore: 100}}, 100},
		{"two halves", []Component{{Score: 40, MaxScore: 100}, {Score: 60, MaxScore: 100}}, 50},
		{"all zero", []Component{{Score: 0, MaxScore: 100}, {Score: 0, MaxScore: 100}}, 0},
		{"mixed", []Component{{Score: 80, MaxScore: 100}, {Score: 100, MaxScore: 100}}, 90},
		{"max zero ignored as zero percent", []Component{{Score: 10, MaxScore: 0}, {Score: 100, MaxScore: 100}}, 50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := average(tt.components); got != tt.want {
				t.Errorf("average() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNames(t *testing.T) {
	got := Names()
	sort.Strings(got)
	want := []string{StrategyLetter, StrategyPassFail, StrategyWeighted}
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("Names() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Names()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestGet_Table(t *testing.T) {
	tests := []struct {
		name     string
		strategy string
		wantErr  bool
	}{
		{"weighted", StrategyWeighted, false},
		{"passfail", StrategyPassFail, false},
		{"letter", StrategyLetter, false},
		{"unknown", "bogus", true},
		{"empty", "", true},
		{"case sensitive", "Weighted", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s, err := Get(tt.strategy)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Get(%q) = nil error, want error", tt.strategy)
				}
				return
			}
			if err != nil {
				t.Fatalf("Get(%q) error = %v", tt.strategy, err)
			}
			if s.Name() != tt.strategy {
				t.Errorf("Name() = %q, want %q", s.Name(), tt.strategy)
			}
		})
	}
}

func TestWeightedAverage_Table(t *testing.T) {
	tests := []struct {
		name       string
		components []Component
		wantFinal  float64
		wantLetter string
		wantPassed bool
	}{
		{"empty", nil, 0, "", false},
		{"single A", []Component{{Score: 95, MaxScore: 100, Weight: 1}}, 95, "A", true},
		{"single boundary 90", []Component{{Score: 90, MaxScore: 100, Weight: 1}}, 90, "A", true},
		{"single B", []Component{{Score: 85, MaxScore: 100, Weight: 1}}, 85, "B", true},
		{"single boundary 60", []Component{{Score: 60, MaxScore: 100, Weight: 1}}, 60, "D", true},
		{"single fail 59", []Component{{Score: 59, MaxScore: 100, Weight: 1}}, 59, "F", false},
		{"two weighted", []Component{{Score: 50, MaxScore: 100, Weight: 0.5}, {Score: 90, MaxScore: 100, Weight: 0.5}}, 70, "C", true},
		{"zero max with weight", []Component{{Score: 10, MaxScore: 0, Weight: 1}}, 0, "F", false},
		{"unweighted fallback", []Component{{Score: 70, MaxScore: 100}, {Score: 90, MaxScore: 100}}, 80, "B", true},
		{"perfect", []Component{{Score: 100, MaxScore: 100, Weight: 2}}, 100, "A", true},
	}
	var s WeightedAverage
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := s.Evaluate(tt.components)
			if got.Final != tt.wantFinal || got.Letter != tt.wantLetter || got.Passed != tt.wantPassed {
				t.Errorf("Evaluate() = {%.2f %q %v}, want {%.2f %q %v}",
					got.Final, got.Letter, got.Passed, tt.wantFinal, tt.wantLetter, tt.wantPassed)
			}
		})
	}
}

func TestLetterGrade_Boundaries(t *testing.T) {
	tests := []struct {
		percent    float64
		wantLetter string
		wantPassed bool
	}{
		{100, "A", true},
		{90, "A", true},
		{89, "B", true},
		{80, "B", true},
		{79, "C", true},
		{70, "C", true},
		{69, "D", true},
		{60, "D", true},
		{59, "F", false},
		{30, "F", false},
		{0, "F", false},
	}
	var s LetterGrade
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%.0f", tt.wantLetter, tt.percent), func(t *testing.T) {
			t.Parallel()
			got := s.Evaluate([]Component{{Score: tt.percent, MaxScore: 100}})
			if got.Letter != tt.wantLetter {
				t.Errorf("Letter for %.0f = %q, want %q", tt.percent, got.Letter, tt.wantLetter)
			}
			if got.Passed != tt.wantPassed {
				t.Errorf("Passed for %.0f = %v, want %v", tt.percent, got.Passed, tt.wantPassed)
			}
		})
	}
}

func TestPassFail_Thresholds(t *testing.T) {
	tests := []struct {
		name       string
		threshold  float64
		score      float64
		wantPassed bool
		wantLetter string
	}{
		{"well above", 60, 90, true, "P"},
		{"just above", 60, 61, true, "P"},
		{"exactly at", 60, 60, true, "P"},
		{"just below", 60, 59, false, "F"},
		{"well below", 60, 10, false, "F"},
		{"zero threshold always passes", 0, 0, true, "P"},
		{"high threshold", 90, 89, false, "F"},
		{"high threshold pass", 90, 90, true, "P"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NewPassFail(tt.threshold).Evaluate([]Component{{Score: tt.score, MaxScore: 100}})
			if got.Passed != tt.wantPassed || got.Letter != tt.wantLetter {
				t.Errorf("Evaluate() = {%v %q}, want {%v %q}", got.Passed, got.Letter, tt.wantPassed, tt.wantLetter)
			}
		})
	}
}

func TestStrategyNames(t *testing.T) {
	tests := []struct {
		strategy Strategy
		want     string
	}{
		{WeightedAverage{}, StrategyWeighted},
		{NewPassFail(60), StrategyPassFail},
		{LetterGrade{}, StrategyLetter},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if tt.strategy.Name() != tt.want {
				t.Errorf("Name() = %q, want %q", tt.strategy.Name(), tt.want)
			}
		})
	}
}
