package datasource

import (
	"context"
	"strings"
	"testing"
)

// Quien usa el connector necesita a veces ejecutar DDL que no sale de un modelo: el caso
// que lo motivo es crear el ESQUEMA (namespace) donde luego viven las tablas. Sin esto, el
// pool es privado y la aplicacion tiene que abrir una segunda conexion por su cuenta,
// duplicando configuracion de TLS y credenciales.
//
// Sin conexion, ExecContext debe fallar con un mensaje que se entienda, no con un panico
// por desreferenciar el pool nulo.
func TestExecContext_SinConexionDaErrorClaro(t *testing.T) {
	c := NewPostgresConnector("postgres://nadie@127.0.0.1:1/x", "public")

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("ExecContext hizo panic sin conexion: %v", r)
		}
	}()

	err := c.ExecContext(context.Background(), "SELECT 1")
	if err == nil {
		t.Fatal("ExecContext devolvio nil sin conexion")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "conect") &&
		!strings.Contains(strings.ToLower(err.Error()), "connect") {
		t.Errorf("el error no explica que falta la conexion: %v", err)
	}
}
