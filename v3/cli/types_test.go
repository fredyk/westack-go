package cli

import "testing"

func TestRequired_applyTo(t *testing.T) {
	f := Field{Name: "Email", Type: TypeString}
	Required{}.applyTo(&f)
	if len(f.Options) != 1 {
		t.Fatalf("expected 1 option, got %d", len(f.Options))
	}
	if _, ok := f.Options[0].(Required); !ok {
		t.Errorf("expected option to be Required, got %T", f.Options[0])
	}
}

func TestPrimaryKey_applyTo(t *testing.T) {
	f := Field{Name: "ID", Type: TypeInt}
	PrimaryKey{}.applyTo(&f)
	if len(f.Options) != 1 {
		t.Fatalf("expected 1 option, got %d", len(f.Options))
	}
	if _, ok := f.Options[0].(PrimaryKey); !ok {
		t.Errorf("expected option to be PrimaryKey, got %T", f.Options[0])
	}
}

func TestVectorDim_applyTo(t *testing.T) {
	f := Field{Name: "Embedding", Type: TypeVector}
	VectorDim(768).applyTo(&f)
	if len(f.Options) != 1 {
		t.Fatalf("expected 1 option, got %d", len(f.Options))
	}
	if vd, ok := f.Options[0].(VectorDim); !ok || int(vd) != 768 {
		t.Errorf("expected VectorDim(768), got %v", f.Options)
	}
}

func TestVectorDim_applyTo_zero(t *testing.T) {
	f := Field{Name: "Embedding", Type: TypeVector}
	VectorDim(0).applyTo(&f)
	if len(f.Options) != 1 {
		t.Fatalf("expected 1 option, got %d", len(f.Options))
	}
}

func TestFieldOpts(t *testing.T) {
	// Verify FieldRequired and FieldPrimaryKey are usable
	f1 := Field{Name: "Email", Type: TypeString}
	FieldRequired.applyTo(&f1)
	if len(f1.Options) != 1 {
		t.Fatalf("FieldRequired: expected 1 option, got %d", len(f1.Options))
	}

	f2 := Field{Name: "ID", Type: TypeInt}
	FieldPrimaryKey.applyTo(&f2)
	if len(f2.Options) != 1 {
		t.Fatalf("FieldPrimaryKey: expected 1 option, got %d", len(f2.Options))
	}
}
