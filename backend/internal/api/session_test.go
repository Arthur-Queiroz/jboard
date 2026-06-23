package api

import (
	"testing"
	"time"
)

func TestSession_RoundtripValido(t *testing.T) {
	now := time.Now()
	v := mintSession("segredo", now)
	if !validSession("segredo", v, now) {
		t.Fatal("sessão recém-criada deveria ser válida")
	}
}

func TestSession_SegredoErrado(t *testing.T) {
	v := mintSession("segredo", time.Now())
	if validSession("outro", v, time.Now()) {
		t.Fatal("sessão não deveria validar com outro segredo")
	}
}

func TestSession_Expirada(t *testing.T) {
	past := time.Now().Add(-2 * sessionTTL) // criada no passado → já expirou
	v := mintSession("segredo", past)
	if validSession("segredo", v, time.Now()) {
		t.Fatal("sessão expirada não deveria validar")
	}
}

func TestSession_Adulterada(t *testing.T) {
	for _, bad := range []string{"", "semponto", "a.b", mintSession("segredo", time.Now()) + "x"} {
		if validSession("segredo", bad, time.Now()) {
			t.Fatalf("valor adulterado não deveria validar: %q", bad)
		}
	}
}
