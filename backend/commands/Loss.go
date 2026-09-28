package commands

import (
    "fmt"
    "strings"
    "backend/structs"
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

func SimularPerdida(id string) error {
    f, sb, particion, err := structs.SuperBloque_ID(id)
    if err != nil {
        return err
    }
    defer f.Close()
    return SimularPerdidaEXT3(f, sb, particion.Part_start)
}