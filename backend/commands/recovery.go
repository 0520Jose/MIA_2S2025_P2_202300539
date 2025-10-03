package commands

import (
    "fmt"
    "strings"
)

func Recovery(params map[string]string) string {
    id, ok := params["-id"]
    if !ok || strings.TrimSpace(id) == "" {
        return "Error: Falta el parámetro obligatorio -id"
    }
    err := RecuperarSistema(id)
    if err != nil {
        return fmt.Sprintf("Error al recuperar sistema: %v", err)
    }
    return fmt.Sprintf("¡Sistema de archivos con id %s recuperado exitosamente!", id)
}