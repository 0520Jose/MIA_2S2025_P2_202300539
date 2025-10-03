package commands

import (
    "backend/structs"
    "errors"
    "fmt"
	"strings"
	"encoding/binary"
)

func Chown(partID, ruta, nuevoUsuario string, recursivo bool) error {
    f, sb, _, err := structs.SuperBloque_ID(partID)
    if err != nil {
        return err
    }
    defer f.Close()

    usuarios, err := structs.LeerArchivoDeFS(partID, "/home/users.txt")
    if err != nil {
        return err
    }

    uidNuevo := -1
    var usuarioActual structs.Usuario
    lines := strings.Split(usuarios, "\n")
    for _, line := range lines {
        campos := strings.Split(line, ",")
        if len(campos) >= 4 && campos[1] == "U" && campos[2] == nuevoUsuario {
            fmt.Sscanf(campos[0], "%d", &uidNuevo)
            usuarioActual = structs.Usuario{
                UID: uidNuevo,
                Nombre: campos[2],
                Grupo: campos[1],
            }
            break
        }
    }
    fmt.Println("UID del nuevo usuario:", uidNuevo)

    if uidNuevo == -1 {
        return fmt.Errorf("usuario %s no existe", nuevoUsuario)
    }
    ino, idx, err := structs.BuscarInodoPorRuta_(f, sb, ruta)
    if err != nil {
        return err
    }

    if !EsRoot() && int(ino.I_uid) != usuarioActual.UID {
        return errors.New("permiso denegado: no es propietario")
    }

    ino.I_uid = int32(uidNuevo)

    offset := int64(sb.S_inode_start) + int64(idx)*int64(sb.S_inode_s)
    if _, err := f.Seek(offset, 0); err != nil {
        return err
    }
    if err := binary.Write(f, binary.LittleEndian, &ino); err != nil {
        return err
    }

    if recursivo && structs.EsCarpeta(ino) {
        hijos, err := structs.ListaCarpetasFS(partID, ruta)
        if err != nil {
            return err
        }
        for _, hijo := range hijos {
            hijoRuta := fmt.Sprintf("%s/%s", ruta, hijo.Nombre)
            Chown(partID, hijoRuta, nuevoUsuario, true)
        }
    }

    return nil
}
