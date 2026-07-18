package datasource

import (
	"errors"
	"fmt"
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
	err := &PGError{Code: "23505", Message: "duplicate key"}
	got := err.Error()
	want := "pg error 23505: duplicate key"
	if got != want {
		t.Errorf("PGError.Error() = %q, want %q", got, want)
	}
}

func TestPGError_Unwrap(t *testing.T) {
	inner := errors.New("inner error")
	pgErr := &PGError{Code: "23505", Message: "dup", err: inner}
	unwrapped := pgErr.Unwrap()
	if unwrapped != inner {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, inner)
	}
}

func TestIsPGError_Match(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "23505", Message: `duplicate key`}
	mapped := MapPGError(pgErr)
	if !IsPGError(mapped, ErrUniqueViolation) {
		t.Error("IsPGError should match ErrUniqueViolation")
	}
}

func TestIsPGError_NoMatch(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "42P01", Message: `no table`}
	mapped := MapPGError(pgErr)
	if IsPGError(mapped, ErrUniqueViolation) {
		t.Error("IsPGError should not match ErrUniqueViolation for 42P01")
	}
}

func TestIsPGError_NilError(t *testing.T) {
	if IsPGError(nil, ErrUniqueViolation) {
		t.Error("IsPGError(nil) should be false")
	}
}

func TestIsPGError_Wrapped(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "23505", Message: `duplicate key`}
	mapped := MapPGError(pgErr)
	wrapped := fmt.Errorf("wrapped: %w", mapped)
	if !IsPGError(wrapped, ErrUniqueViolation) {
		t.Error("IsPGError should find PGError through error chain")
	}
}

func TestIsPGError_NonPGError(t *testing.T) {
	plain := errors.New("plain error")
	if IsPGError(plain, ErrUniqueViolation) {
		t.Error("IsPGError should be false for non-PGError")
	}
}