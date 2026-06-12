package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"coursehub/internal/domain"
)

// StudentRepo is a SQLite-backed service.StudentRepository.
type StudentRepo struct {
	db *sql.DB
}

// NewStudentRepo returns a StudentRepo using db.
func NewStudentRepo(db *sql.DB) *StudentRepo { return &StudentRepo{db: db} }

// Create inserts a student, mapping a duplicate e-mail to domain.ErrDuplicate.
func (r *StudentRepo) Create(ctx context.Context, s *domain.Student) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO students (id, name, email, created_at) VALUES (?, ?, ?, ?)`,
		s.ID, s.Name, s.Email, s.CreatedAt.Format(timeLayout))
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("student %q: %w", s.Email, domain.ErrDuplicate)
		}
		return fmt.Errorf("insert student: %w", err)
	}
	return nil
}

// GetByID returns the student with id, or domain.ErrNotFound.
func (r *StudentRepo) GetByID(ctx context.Context, id string) (*domain.Student, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, email, created_at FROM students WHERE id = ?`, id)
	return scanStudent(row)
}

// List returns all students ordered by creation time.
func (r *StudentRepo) List(ctx context.Context) ([]*domain.Student, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, email, created_at FROM students ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("list students: %w", err)
	}
	defer rows.Close()

	var students []*domain.Student
	for rows.Next() {
		s, err := scanStudent(rows)
		if err != nil {
			return nil, err
		}
		students = append(students, s)
	}
	return students, rows.Err()
}

// scanner abstracts *sql.Row and *sql.Rows so scanStudent serves both.
type scanner interface {
	Scan(dest ...any) error
}

func scanStudent(row scanner) (*domain.Student, error) {
	var (
		s         domain.Student
		createdAt string
	)
	if err := row.Scan(&s.ID, &s.Name, &s.Email, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("scan student: %w", err)
	}
	t, err := time.Parse(timeLayout, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	s.CreatedAt = t
	return &s, nil
}
