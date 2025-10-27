package commands

import (
    "backend/structs"
    "encoding/binary"
    "errors"
    "fmt"
    "strings"
)

func Chown(ruta, nuevoUsuario string, recursivo bool) error {
    // Verificar que hay una sesión activa
    if usuarioActual.PartID == "" {
        return errors.New("no hay una sesión activa")
    }
    
    // Solo root puede ejecutar chown
    if !EsRoot() {
        return errors.New("permiso denegado: solo root puede ejecutar chown")
    }
    
    partID := usuarioActual.PartID
    
    f, sb, _, err := structs.SuperBloque_ID(partID)
    if err != nil {
        return err
    }
    defer f.Close()
    
    // Leer archivo de usuarios
    usuarios, err := structs.LeerArchivoDeFS(partID, "/home/users.txt")
    if err != nil {
        return err
    }
    
    // Buscar el UID del nuevo usuario
    uidNuevo := -1
    lines := strings.Split(usuarios, "\n")
    for _, line := range lines {
        campos := strings.Split(line, ",")
        if len(campos) >= 4 && campos[1] == "U" && campos[3] == nuevoUsuario {
            fmt.Sscanf(campos[0], "%d", &uidNuevo)
            break
        }
    }
    
    fmt.Println("UID del nuevo usuario:", uidNuevo)
    if uidNuevo == -1 {
        return fmt.Errorf("usuario %s no existe", nuevoUsuario)
    }
    
    // Buscar el inodo del archivo/carpeta
    ino, idx, err := structs.BuscarInodoPorRuta_(f, sb, ruta)
    if err != nil {
        return err
    }
    
    // Actualizar el propietario
    ino.I_uid = int32(uidNuevo)
    
    // Escribir el inodo actualizado
    offset := int64(sb.S_inode_start) + int64(idx)*int64(sb.S_inode_s)
    if _, err := f.Seek(offset, 0); err != nil {
        return err
    }
    if err := binary.Write(f, binary.LittleEndian, &ino); err != nil {
        return err
    }
    
    // Registrar en journal si es EXT3
    if sb.S_filesystem_type == 3 {
        contenido := nuevoUsuario
        if err := RegistrarOperacionJournal(f, sb, sb.S_bm_inode_start, "chown", ruta, contenido); err != nil {
            fmt.Printf("Advertencia: no se pudo registrar en journal: %v\n", err)
        }
    }
    
    // Aplicar recursivamente si es necesario
    if recursivo && structs.EsCarpeta(ino) {
        hijos, err := structs.ListaCarpetasFS(partID, ruta)
        if err != nil {
            return err
        }
        for _, hijo := range hijos {
            hijoRuta := fmt.Sprintf("%s/%s", ruta, hijo.Nombre)
            // Llamada recursiva con la nueva firma
            _ = Chown(hijoRuta, nuevoUsuario, true)
        }
    }
    
    return nil
}