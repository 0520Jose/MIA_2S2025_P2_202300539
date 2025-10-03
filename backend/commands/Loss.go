package commands

import (
    "fmt"
    "strings"
)

func Loss(params map[string]string) string {
    id, ok := params["-id"]
    if !ok || strings.TrimSpace(id) == "" {
        return "Error: Falta el parámetro obligatorio -id"
    }
    err := SimularPerdida(id)
    if err != nil {
        return fmt.Sprintf("Error al simular pérdida: %v", err)
    }
    return fmt.Sprintf("¡Sistema de archivos con id %s ha sido 'perdido' exitosamente!", id)
}