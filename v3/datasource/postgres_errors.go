package datasource

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

// PGError is a typed PostgreSQL error that consumers can recognise with errors.Is.
type PGError struct {
	Code    string
	Message string
	err     error
}

func (e *PGError) Error() string {
	return fmt.Sprintf("pg error %s: %s", e.Code, e.Message)
}

func (e *PGError) Unwrap() error {
	return e.err
}

func (e *PGError) Is(target error) bool {
	t, ok := target.(*PGError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

var (
	ErrUniqueViolation  = &PGError{Code: "23505", Message: "unique constraint violation"}
	ErrUndefinedTable   = &PGError{Code: "42P01", Message: "undefined table"}
	ErrInvalidTextRep   = &PGError{Code: "22P02", Message: "invalid text representation"}
)

// MapPGError wraps a *pgconn.PgError into a typed *PGError when the code
// matches a known sentinel. All other errors pass through unchanged.
func MapPGError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch pgErr.Code {
	case "23505":
		return &PGError{Code: pgErr.Code, Message: pgErr.Message, err: err}
	case "42P01":
		return &PGError{Code: pgErr.Code, Message: pgErr.Message, err: err}
	case "22P02":
		return &PGError{Code: pgErr.Code, Message: pgErr.Message, err: err}
	default:
		return err
	}
}

// IsPGError reports whether err contains a *PGError matching the given sentinel.
func IsPGError(err error, target *PGError) bool {
	return errors.Is(err, target)
}