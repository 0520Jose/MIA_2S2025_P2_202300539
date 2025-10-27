package commands

import (
    "backend/structs"
    "fmt"
    "strings"
)

var Recuperado int32 = 0

func Recovery(params map[string]string) string {
    id, ok := params["-id"]
    if !ok || strings.TrimSpace(id) == "" {
        return "Error: Falta el parámetro obligatorio -id"
    }

    usuarioActual = &UserSession{
        Username:    "root",
        PartitionID: id,
        Group:       "root",
        UID:         1,
        GID:         1,
        PartID:    id,
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
    Recuperado = 1
    return fmt.Sprintf("¡Sistema de archivos con id %s recuperado exitosamente!", id)
}