package grading

import "testing"

func TestWeightedAverage_Evaluate(t *testing.T) {
	tests := []struct {
		name       string
		components []Component
		wantFinal  float64
		wantLetter string
		wantPassed bool
	}{
		{
			name:       "empty",
			components: nil,
			wantFinal:  0,
			wantLetter: "",
			wantPassed: false,
		},
		{
			name: "weighted toward strong exam",
			components: []Component{
				{Name: "lab", Score: 50, MaxScore: 100, Weight: 0.2},
				{Name: "exam", Score: 90, MaxScore: 100, Weight: 0.8},
			},
			wantFinal:  82, // 0.2*50 + 0.8*90
			wantLetter: "B",
			wantPassed: true,
		},
		{
			name: "no weights falls back to plain average",
			components: []Component{
				{Name: "a", Score: 40, MaxScore: 100},
				{Name: "b", Score: 60, MaxScore: 100},
			},
			wantFinal:  50,
			wantLetter: "F",
			wantPassed: false,
		},
	}

	var s WeightedAverage
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := s.Evaluate(tt.components)
			if got.Final != tt.wantFinal {
				t.Errorf("Final = %v, want %v", got.Final, tt.wantFinal)
			}
			if got.Letter != tt.wantLetter {
				t.Errorf("Letter = %q, want %q", got.Letter, tt.wantLetter)
			}
			if got.Passed != tt.wantPassed {
				t.Errorf("Passed = %v, want %v", got.Passed, tt.wantPassed)
			}
		})
	}
}

func TestPassFail_Evaluate(t *testing.T) {
	tests := []struct {
		name       string
		threshold  float64
		components []Component
		wantPassed bool
		wantLetter string
	}{
		{
			name:       "above threshold",
			threshold:  60,
			components: []Component{{Score: 70, MaxScore: 100}},
			wantPassed: true,
			wantLetter: "P",
		},
		{
			name:       "below threshold",
			threshold:  60,
			components: []Component{{Score: 59, MaxScore: 100}},
			wantPassed: false,
			wantLetter: "F",
		},
		{
			name:       "exactly at threshold passes",
			threshold:  50,
			components: []Component{{Score: 50, MaxScore: 100}},
			wantPassed: true,
			wantLetter: "P",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NewPassFail(tt.threshold).Evaluate(tt.components)
			if got.Passed != tt.wantPassed {
				t.Errorf("Passed = %v, want %v", got.Passed, tt.wantPassed)
			}
			if got.Letter != tt.wantLetter {
				t.Errorf("Letter = %q, want %q", got.Letter, tt.wantLetter)
			}
		})
	}
}

func TestLetterGrade_Evaluate(t *testing.T) {
	tests := []struct {
		name       string
		percent    float64
		wantLetter string
	}{
		{"A", 95, "A"},
		{"B", 85, "B"},
		{"C", 75, "C"},
		{"D", 65, "D"},
		{"F", 40, "F"},
		{"boundary 90 is A", 90, "A"},
		{"boundary 60 is D", 60, "D"},
	}

	var s LetterGrade
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := s.Evaluate([]Component{{Score: tt.percent, MaxScore: 100}})
			if got.Letter != tt.wantLetter {
				t.Errorf("Letter = %q, want %q", got.Letter, tt.wantLetter)
			}
		})
	}
}

func TestGet(t *testing.T) {
	for _, name := range []string{StrategyWeighted, StrategyPassFail, StrategyLetter} {
		s, err := Get(name)
		if err != nil {
			t.Fatalf("Get(%q) returned error: %v", name, err)
		}
		if s.Name() != name {
			t.Errorf("Get(%q).Name() = %q", name, s.Name())
		}
	}

	if _, err := Get("nonexistent"); err == nil {
		t.Error("Get(nonexistent) expected error, got nil")
	}
}
