package datasource

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestMapPGError_UniqueViolation(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "23505", Message: `duplicate key value violates unique constraint "pk_name"`}
	mapped := MapPGError(pgErr)
	if !errors.Is(mapped, ErrUniqueViolation) {
		t.Errorf("expected ErrUniqueViolation, got %T: %v", mapped, mapped)
	}
}

func TestMapPGError_UndefinedTable(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "42P01", Message: `relation "foo" does not exist`}
	mapped := MapPGError(pgErr)
	if !errors.Is(mapped, ErrUndefinedTable) {
		t.Errorf("expected ErrUndefinedTable, got %T: %v", mapped, mapped)
	}
}

func TestMapPGError_InvalidTextRepresentation(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "22P02", Message: `invalid input syntax for type vector`}
	mapped := MapPGError(pgErr)
	if !errors.Is(mapped, ErrInvalidTextRep) {
		t.Errorf("expected ErrInvalidTextRep, got %T: %v", mapped, mapped)
	}
}

func TestMapPGError_NonPGError(t *testing.T) {
	err := errors.New("some other error")
	mapped := MapPGError(err)
	if mapped != err {
		t.Errorf("expected original error, got %v", mapped)
	}
}

func TestMapPGError_Nil(t *testing.T) {
	mapped := MapPGError(nil)
	if mapped != nil {
		t.Errorf("expected nil, got %v", mapped)
	}
}

func TestMapPGError_UnknownCode(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "01000", Message: "warning"}
	mapped := MapPGError(pgErr)
	if mapped != pgErr {
		t.Errorf("expected original pgErr for unknown code, got %v", mapped)
	}
}

func TestMapPGError_WrapsOriginal(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "23505", Message: `duplicate key`}
	mapped := MapPGError(pgErr)
	var inner *pgconn.PgError
	if !errors.As(mapped, &inner) {
		t.Fatal("expected MapPGError to wrap original *pgconn.PgError")
	}
	if inner.Code != "23505" {
		t.Errorf("inner Code = %q, want 23505", inner.Code)
	}
}

func TestPGError_Error(t *testing.T) {
	e := &PGError{Code: "23505", Message: "duplicate key"}
	got := e.Error()
	if got != "pg error 23505: duplicate key" {
		t.Errorf("Error() = %q, want %q", got, "pg error 23505: duplicate key")
	}
}

func TestPGError_Unwrap(t *testing.T) {
	inner := errors.New("original")
	e := &PGError{Code: "23505", Message: "dup", err: inner}
	if e.Unwrap() != inner {
		t.Error("Unwrap() should return the original error")
	}
}

func TestPGError_Is_Match(t *testing.T) {
	e := &PGError{Code: "23505", Message: "dup"}
	if !e.Is(ErrUniqueViolation) {
		t.Error("Is(ErrUniqueViolation) should be true for code 23505")
	}
}

func TestPGError_Is_CodeMismatch(t *testing.T) {
	e := &PGError{Code: "23505", Message: "dup"}
	target := &PGError{Code: "42P01", Message: "other"}
	if e.Is(target) {
		t.Error("Is(PGError with different code) should be false")
	}
}

func TestPGError_Is_NonPGErrorTarget(t *testing.T) {
	e := &PGError{Code: "23505", Message: "dup"}
	target := errors.New("not a PGError")
	if e.Is(target) {
		t.Error("Is(non-PGError target) should be false")
	}
}

func TestIsPGError_Match(t *testing.T) {
	e := &PGError{Code: "23505", Message: "dup"}
	if !IsPGError(e, ErrUniqueViolation) {
		t.Error("IsPGError should match ErrUniqueViolation for code 23505")
	}
}

func TestIsPGError_NoMatch(t *testing.T) {
	e := &PGError{Code: "42P01", Message: "other"}
	if IsPGError(e, ErrUniqueViolation) {
		t.Error("IsPGError should not match ErrUniqueViolation for code 42P01")
	}
}

func TestIsPGError_NilError(t *testing.T) {
	if IsPGError(nil, ErrUniqueViolation) {
		t.Error("IsPGError(nil, ...) should be false")
	}
}

func TestIsPGError_NonPGErrorType(t *testing.T) {
	if IsPGError(errors.New("plain error"), ErrUniqueViolation) {
		t.Error("IsPGError(plain error, ...) should be false")
	}
}