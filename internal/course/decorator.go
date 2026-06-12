// Package course builds domain.Course values using the Factory pattern and
// extends them with the Decorator pattern. A base course carries the intrinsic
// data; decorators wrap a course to add behaviour (a completion certificate, an
// online-delivery platform) while remaining a domain.Course themselves, so the
// rest of the system treats decorated and plain courses uniformly.
package course

import (
	"strings"

	"coursehub/internal/domain"
)

// base is the concrete, undecorated course. It is unexported: callers obtain a
// domain.Course through the Factory, never by constructing this directly.
type base struct {
	id       string
	code     string
	title    string
	credits  int
	grading  string
	features []string
}

func (b *base) ID() string              { return b.id }
func (b *base) Code() string            { return b.code }
func (b *base) Title() string           { return b.title }
func (b *base) Credits() int            { return b.credits }
func (b *base) GradingStrategy() string { return b.grading }
func (b *base) Features() []string      { return append([]string(nil), b.features...) }
func (b *base) Describe() string {
	return b.code + " — " + b.title
}

// Feature names contributed by decorators. They are also the tokens stored in
// CourseRecord.Features so the factory can rebuild the decorator chain on load.
const (
	FeatureCertified = "certified"
	FeatureOnline    = "online"
)

// CertifiedCourse decorates a course so completing it yields a certificate.
type CertifiedCourse struct {
	domain.Course // embedded wrapped course
}

// Features appends the certified marker to the wrapped course's features.
func (c CertifiedCourse) Features() []string {
	return append(c.Course.Features(), FeatureCertified)
}

// Describe extends the wrapped description.
func (c CertifiedCourse) Describe() string {
	return c.Course.Describe() + " [сертифікований]"
}

// IssuesCertificate reports whether a passing student earns a certificate.
// This is behaviour the bare course does not have.
func (c CertifiedCourse) IssuesCertificate() bool { return true }

// OnlineCourse decorates a course delivered through an online platform.
type OnlineCourse struct {
	domain.Course
	Platform string
}

// Features appends the online marker.
func (o OnlineCourse) Features() []string {
	return append(o.Course.Features(), FeatureOnline)
}

// Describe extends the wrapped description with the platform.
func (o OnlineCourse) Describe() string {
	d := o.Course.Describe() + " [онлайн"
	if o.Platform != "" {
		d += ": " + o.Platform
	}
	return d + "]"
}

// hasFeature reports whether features contains name (case-insensitive).
func hasFeature(features []string, name string) bool {
	for _, f := range features {
		if strings.EqualFold(strings.TrimSpace(f), name) {
			return true
		}
	}
	return false
}
