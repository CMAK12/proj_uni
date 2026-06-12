package course

import (
	"errors"
	"testing"

	"coursehub/internal/domain"
	"coursehub/internal/grading"
)

func TestFactory_Build(t *testing.T) {
	f := NewFactory()

	tests := []struct {
		name         string
		rec          domain.CourseRecord
		wantErr      error
		wantFeatures []string
		wantCert     bool
	}{
		{
			name: "standard course has no decorator features",
			rec: domain.CourseRecord{
				Code: "CS101", Title: "Algorithms", Credits: 5,
				Type: domain.CourseTypeStandard, Grading: grading.StrategyWeighted,
			},
			wantFeatures: nil,
		},
		{
			name: "online certified course stacks both decorators",
			rec: domain.CourseRecord{
				Code: "CS102", Title: "Web", Credits: 4,
				Type:     domain.CourseTypeOnline,
				Grading:  grading.StrategyLetter,
				Features: []string{FeatureCertified},
				Platform: "Moodle",
			},
			wantFeatures: []string{FeatureOnline, FeatureCertified},
			wantCert:     true,
		},
		{
			name:    "missing title fails validation",
			rec:     domain.CourseRecord{Code: "X", Grading: grading.StrategyWeighted},
			wantErr: domain.ErrValidation,
		},
		{
			name:    "unknown grading strategy fails validation",
			rec:     domain.CourseRecord{Code: "X", Title: "T", Grading: "bogus"},
			wantErr: domain.ErrValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c, err := f.Build(tt.rec)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !equalFeatures(c.Features(), tt.wantFeatures) {
				t.Errorf("Features() = %v, want %v", c.Features(), tt.wantFeatures)
			}
			if _, ok := c.(CertifiedCourse); ok != tt.wantCert {
				t.Errorf("CertifiedCourse type assertion = %v, want %v", ok, tt.wantCert)
			}
		})
	}
}

func TestDecorators_Describe(t *testing.T) {
	f := NewFactory()
	c, err := f.Build(domain.CourseRecord{
		Code: "CS102", Title: "Web", Grading: grading.StrategyLetter,
		Type: domain.CourseTypeOnline, Features: []string{FeatureCertified}, Platform: "Moodle",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := c.Describe()
	want := "CS102 — Web [онлайн: Moodle] [сертифікований]"
	if got != want {
		t.Errorf("Describe() = %q, want %q", got, want)
	}
}

func equalFeatures(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
