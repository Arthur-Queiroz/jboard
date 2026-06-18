package api

import (
	"fmt"
	"strings"
	"time"
)

// required devolve erro se s for vazio após trim. Usado nos handlers pra
// devolver 400 com mensagem específica do campo faltante.
func required(field, s string) error {
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("%s é obrigatório", field)
	}
	return nil
}

// futureTime devolve erro se t for zero ou estiver no passado. Lembretes só
// fazem sentido agendados pra um momento futuro.
func futureTime(field string, t time.Time) error {
	if t.IsZero() {
		return fmt.Errorf("%s é obrigatório", field)
	}
	if !t.After(time.Now()) {
		return fmt.Errorf("%s deve estar no futuro", field)
	}
	return nil
}
