package commands

import (
    "backend/structs"
    "fmt"
    "strings"
)

func Recovery(params map[string]string) string {
    id, ok := params["-id"]
    if !ok || strings.TrimSpace(id) == "" {
        return "Error: Falta el parámetro obligatorio -id"
    }

    f, sb, part, err := structs.SuperBloque_ID(id)
    if err != nil {
        return fmt.Sprintf("Error al abrir sistema: %v", err)
    }
    defer f.Close()

    err = RecuperarEXT3(f, part.Part_start, sb)
    if err != nil {
        return fmt.Sprintf("Error al recuperar sistema: %v", err)
    }
    return fmt.Sprintf("¡Sistema de archivos con id %s recuperado exitosamente!", id)
}