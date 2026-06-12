package course

import (
	"errors"
	"testing"

	"coursehub/internal/domain"
	"coursehub/internal/grading"
)

func TestHasFeature(t *testing.T) {
	tests := []struct {
		name     string
		features []string
		target   string
		want     bool
	}{
		{"present", []string{"certified"}, "certified", true},
		{"absent", []string{"online"}, "certified", false},
		{"empty list", nil, "certified", false},
		{"case insensitive", []string{"Certified"}, "certified", true},
		{"upper target", []string{"online"}, "ONLINE", true},
		{"trims spaces", []string{"  certified  "}, "certified", true},
		{"multiple has one", []string{"online", "certified"}, "certified", true},
		{"empty target empty entry", []string{""}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := hasFeature(tt.features, tt.target); got != tt.want {
				t.Errorf("hasFeature(%v, %q) = %v, want %v", tt.features, tt.target, got, tt.want)
			}
		})
	}
}

func TestFactory_Build_Combinations(t *testing.T) {
	f := NewFactory()
	tests := []struct {
		name         string
		rec          domain.CourseRecord
		wantErr      error
		wantFeatures []string
		wantOnline   bool
		wantCert     bool
	}{
		{
			name:         "bare standard",
			rec:          domain.CourseRecord{Code: "CS1", Title: "Intro", Grading: grading.StrategyWeighted},
			wantFeatures: nil,
		},
		{
			name:         "online by type",
			rec:          domain.CourseRecord{Code: "CS2", Title: "Web", Type: domain.CourseTypeOnline, Grading: grading.StrategyLetter, Platform: "Moodle"},
			wantFeatures: []string{FeatureOnline},
			wantOnline:   true,
		},
		{
			name:         "online by feature flag",
			rec:          domain.CourseRecord{Code: "CS3", Title: "Web2", Grading: grading.StrategyLetter, Features: []string{FeatureOnline}},
			wantFeatures: []string{FeatureOnline},
			wantOnline:   true,
		},
		{
			name:         "certified only",
			rec:          domain.CourseRecord{Code: "CS4", Title: "Cert", Grading: grading.StrategyPassFail, Features: []string{FeatureCertified}},
			wantFeatures: []string{FeatureCertified},
			wantCert:     true,
		},
		{
			name:         "online and certified stacked",
			rec:          domain.CourseRecord{Code: "CS5", Title: "Both", Type: domain.CourseTypeOnline, Grading: grading.StrategyWeighted, Features: []string{FeatureCertified}, Platform: "edX"},
			wantFeatures: []string{FeatureOnline, FeatureCertified},
			wantOnline:   true,
			wantCert:     true,
		},
		{"missing code", domain.CourseRecord{Title: "T", Grading: grading.StrategyWeighted}, domain.ErrValidation, nil, false, false},
		{"missing title", domain.CourseRecord{Code: "X", Grading: grading.StrategyWeighted}, domain.ErrValidation, nil, false, false},
		{"empty grading", domain.CourseRecord{Code: "X", Title: "T", Grading: ""}, domain.ErrValidation, nil, false, false},
		{"unknown grading", domain.CourseRecord{Code: "X", Title: "T", Grading: "bogus"}, domain.ErrValidation, nil, false, false},
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
			_, isOnline := c.(OnlineCourse)
			_, isCert := c.(CertifiedCourse)
			// Certified wraps online, so when both apply the outer type is Certified.
			if tt.wantCert && !isCert {
				t.Errorf("expected CertifiedCourse outer type, got %T", c)
			}
			if tt.wantOnline && !tt.wantCert && !isOnline {
				t.Errorf("expected OnlineCourse outer type, got %T", c)
			}
		})
	}
}

func TestBaseAccessors(t *testing.T) {
	c, err := NewFactory().Build(domain.CourseRecord{
		ID: "id1", Code: "CS101", Title: "Algorithms", Credits: 5, Grading: grading.StrategyWeighted,
	})
	if err != nil {
		t.Fatal(err)
	}
	if c.ID() != "id1" {
		t.Errorf("ID() = %q", c.ID())
	}
	if c.Code() != "CS101" {
		t.Errorf("Code() = %q", c.Code())
	}
	if c.Title() != "Algorithms" {
		t.Errorf("Title() = %q", c.Title())
	}
	if c.Credits() != 5 {
		t.Errorf("Credits() = %d", c.Credits())
	}
	if c.GradingStrategy() != grading.StrategyWeighted {
		t.Errorf("GradingStrategy() = %q", c.GradingStrategy())
	}
	if c.Describe() != "CS101 — Algorithms" {
		t.Errorf("Describe() = %q", c.Describe())
	}
}

func TestDecorators_Describe_Table(t *testing.T) {
	f := NewFactory()
	tests := []struct {
		name string
		rec  domain.CourseRecord
		want string
	}{
		{
			"online with platform",
			domain.CourseRecord{Code: "C", Title: "T", Grading: grading.StrategyLetter, Type: domain.CourseTypeOnline, Platform: "Moodle"},
			"C — T [онлайн: Moodle]",
		},
		{
			"online without platform",
			domain.CourseRecord{Code: "C", Title: "T", Grading: grading.StrategyLetter, Type: domain.CourseTypeOnline},
			"C — T [онлайн]",
		},
		{
			"certified",
			domain.CourseRecord{Code: "C", Title: "T", Grading: grading.StrategyLetter, Features: []string{FeatureCertified}},
			"C — T [сертифікований]",
		},
		{
			"online and certified",
			domain.CourseRecord{Code: "C", Title: "T", Grading: grading.StrategyLetter, Type: domain.CourseTypeOnline, Platform: "edX", Features: []string{FeatureCertified}},
			"C — T [онлайн: edX] [сертифікований]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c, err := f.Build(tt.rec)
			if err != nil {
				t.Fatal(err)
			}
			if got := c.Describe(); got != tt.want {
				t.Errorf("Describe() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCertifiedCourse_IssuesCertificate(t *testing.T) {
	c, err := NewFactory().Build(domain.CourseRecord{
		Code: "C", Title: "T", Grading: grading.StrategyLetter, Features: []string{FeatureCertified},
	})
	if err != nil {
		t.Fatal(err)
	}
	cert, ok := c.(CertifiedCourse)
	if !ok {
		t.Fatalf("expected CertifiedCourse, got %T", c)
	}
	if !cert.IssuesCertificate() {
		t.Error("IssuesCertificate() = false, want true")
	}
}
