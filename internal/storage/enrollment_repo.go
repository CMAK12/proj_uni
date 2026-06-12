package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"coursehub/internal/domain"
)

// EnrollmentRepo is a SQLite-backed service.EnrollmentRepository. It also
// satisfies progress.Store via SetProgress.
type EnrollmentRepo struct {
	db *sql.DB
}

// NewEnrollmentRepo returns an EnrollmentRepo using db.
func NewEnrollmentRepo(db *sql.DB) *EnrollmentRepo { return &EnrollmentRepo{db: db} }

// Create inserts a new enrollment.
func (r *EnrollmentRepo) Create(ctx context.Context, e *domain.Enrollment) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO enrollments
		 (id, student_id, course_id, status, final_grade, letter, passed, progress, enrolled_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.StudentID, e.CourseID, string(e.Status), e.FinalGrade, e.Letter,
		boolToInt(e.Passed), e.Progress, e.EnrolledAt.Format(timeLayout))
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("enrollment %s/%s: %w", e.StudentID, e.CourseID, domain.ErrAlreadyEnrolled)
		}
		return fmt.Errorf("insert enrollment: %w", err)
	}
	return nil
}

// GetByID loads an enrollment and its assessments.
func (r *EnrollmentRepo) GetByID(ctx context.Context, id string) (*domain.Enrollment, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, student_id, course_id, status, final_grade, letter, passed, progress, enrolled_at
		 FROM enrollments WHERE id = ?`, id)
	e, err := scanEnrollment(row)
	if err != nil {
		return nil, err
	}
	if e.Assessments, err = r.assessments(ctx, e.ID); err != nil {
		return nil, err
	}
	return e, nil
}

// ListByStudent returns all enrollments for a student with their assessments.
func (r *EnrollmentRepo) ListByStudent(ctx context.Context, studentID string) ([]*domain.Enrollment, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, student_id, course_id, status, final_grade, letter, passed, progress, enrolled_at
		 FROM enrollments WHERE student_id = ? ORDER BY enrolled_at`, studentID)
	if err != nil {
		return nil, fmt.Errorf("list enrollments: %w", err)
	}
	defer rows.Close()

	var enrollments []*domain.Enrollment
	for rows.Next() {
		e, err := scanEnrollment(rows)
		if err != nil {
			return nil, err
		}
		enrollments = append(enrollments, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Load assessments after the rows are closed to avoid nested queries on the
	// single-connection pool.
	for _, e := range enrollments {
		if e.Assessments, err = r.assessments(ctx, e.ID); err != nil {
			return nil, err
		}
	}
	return enrollments, nil
}

// Exists reports whether the student is already enrolled in the course.
func (r *EnrollmentRepo) Exists(ctx context.Context, studentID, courseID string) (bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM enrollments WHERE student_id = ? AND course_id = ?`,
		studentID, courseID).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("check enrollment: %w", err)
	}
	return n > 0, nil
}

// Update saves grade fields, status and the full assessment set in a single
// transaction (assessments are replaced wholesale).
func (r *EnrollmentRepo) Update(ctx context.Context, e *domain.Enrollment) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin update: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx,
		`UPDATE enrollments
		 SET status = ?, final_grade = ?, letter = ?, passed = ?, progress = ?
		 WHERE id = ?`,
		string(e.Status), e.FinalGrade, e.Letter, boolToInt(e.Passed), e.Progress, e.ID)
	if err != nil {
		return fmt.Errorf("update enrollment: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrNotFound
	}

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM assessments WHERE enrollment_id = ?`, e.ID); err != nil {
		return fmt.Errorf("clear assessments: %w", err)
	}
	for _, a := range e.Assessments {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO assessments (id, enrollment_id, name, score, max_score, weight)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			a.ID, e.ID, a.Name, a.Score, a.MaxScore, a.Weight); err != nil {
			return fmt.Errorf("insert assessment: %w", err)
		}
	}
	return tx.Commit()
}

// SetProgress updates only the progress column. It implements progress.Store.
func (r *EnrollmentRepo) SetProgress(ctx context.Context, enrollmentID string, progress float64) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE enrollments SET progress = ? WHERE id = ?`, progress, enrollmentID)
	if err != nil {
		return fmt.Errorf("set progress: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *EnrollmentRepo) assessments(ctx context.Context, enrollmentID string) ([]domain.Assessment, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, score, max_score, weight FROM assessments WHERE enrollment_id = ? ORDER BY name`,
		enrollmentID)
	if err != nil {
		return nil, fmt.Errorf("list assessments: %w", err)
	}
	defer rows.Close()

	var out []domain.Assessment
	for rows.Next() {
		var a domain.Assessment
		if err := rows.Scan(&a.ID, &a.Name, &a.Score, &a.MaxScore, &a.Weight); err != nil {
			return nil, fmt.Errorf("scan assessment: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func scanEnrollment(row scanner) (*domain.Enrollment, error) {
	var (
		e          domain.Enrollment
		status     string
		passed     int
		enrolledAt string
	)
	if err := row.Scan(&e.ID, &e.StudentID, &e.CourseID, &status, &e.FinalGrade,
		&e.Letter, &passed, &e.Progress, &enrolledAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("scan enrollment: %w", err)
	}
	e.Status = domain.EnrollmentStatus(status)
	e.Passed = passed != 0
	t, err := time.Parse(timeLayout, enrolledAt)
	if err != nil {
		return nil, fmt.Errorf("parse enrolled_at: %w", err)
	}
	e.EnrolledAt = t
	return &e, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
