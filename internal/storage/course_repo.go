package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"coursehub/internal/domain"
)

// CourseRepo is a SQLite-backed service.CourseRepository. Courses are stored as
// flat records; the service rebuilds the decorated domain.Course via the factory.
type CourseRepo struct {
	db *sql.DB
}

// NewCourseRepo returns a CourseRepo using db.
func NewCourseRepo(db *sql.DB) *CourseRepo { return &CourseRepo{db: db} }

// Create inserts a course record, mapping a duplicate code to ErrDuplicate.
func (r *CourseRepo) Create(ctx context.Context, rec domain.CourseRecord) error {
	features, err := json.Marshal(rec.Features)
	if err != nil {
		return fmt.Errorf("encode features: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO courses (id, code, title, credits, type, grading, features, platform, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rec.ID, rec.Code, rec.Title, rec.Credits, string(rec.Type), rec.Grading,
		string(features), rec.Platform, rec.CreatedAt.Format(timeLayout))
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("course %q: %w", rec.Code, domain.ErrDuplicate)
		}
		return fmt.Errorf("insert course: %w", err)
	}
	return nil
}

// GetByID returns the course record with id, or domain.ErrNotFound.
func (r *CourseRepo) GetByID(ctx context.Context, id string) (domain.CourseRecord, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, code, title, credits, type, grading, features, platform, created_at
		 FROM courses WHERE id = ?`, id)
	return scanCourse(row)
}

// List returns all course records ordered by code.
func (r *CourseRepo) List(ctx context.Context) ([]domain.CourseRecord, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, code, title, credits, type, grading, features, platform, created_at
		 FROM courses ORDER BY code`)
	if err != nil {
		return nil, fmt.Errorf("list courses: %w", err)
	}
	defer rows.Close()

	var recs []domain.CourseRecord
	for rows.Next() {
		rec, err := scanCourse(rows)
		if err != nil {
			return nil, err
		}
		recs = append(recs, rec)
	}
	return recs, rows.Err()
}

func scanCourse(row scanner) (domain.CourseRecord, error) {
	var (
		rec       domain.CourseRecord
		typ       string
		features  string
		createdAt string
	)
	if err := row.Scan(&rec.ID, &rec.Code, &rec.Title, &rec.Credits, &typ,
		&rec.Grading, &features, &rec.Platform, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.CourseRecord{}, domain.ErrNotFound
		}
		return domain.CourseRecord{}, fmt.Errorf("scan course: %w", err)
	}
	rec.Type = domain.CourseType(typ)
	if err := json.Unmarshal([]byte(features), &rec.Features); err != nil {
		return domain.CourseRecord{}, fmt.Errorf("decode features: %w", err)
	}
	t, err := time.Parse(timeLayout, createdAt)
	if err != nil {
		return domain.CourseRecord{}, fmt.Errorf("parse created_at: %w", err)
	}
	rec.CreatedAt = t
	return rec, nil
}
