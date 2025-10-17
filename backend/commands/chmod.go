package commands

import (
    "backend/structs"
    "encoding/binary"
    "errors"
    "fmt"
    "strconv"
)

func Chmod(partID, ruta, ugo string, recursivo bool) error {
    if !EsRoot() {
        return errors.New("permiso denegado: solo root puede ejecutar chmod")
    }

    if len(ugo) != 3 {
        return errors.New("el parámetro ugo debe tener 3 dígitos")
    }

    perms := [3]byte{}
    for i := 0; i < 3; i++ {
        val, err := strconv.Atoi(string(ugo[i]))
        if err != nil || val < 0 || val > 7 {
            return fmt.Errorf("permiso inválido en posición %d: %c", i, ugo[i])
        }
        perms[i] = byte(val)
    }

    f, sb, _, err := structs.SuperBloque_ID(partID)
    if err != nil {
        return err
    }
    defer f.Close()

    ino, idx, err := structs.BuscarInodoPorRuta_(f, sb, ruta)
    if err != nil {
        return err
    }

    if !EsRoot() && int(ino.I_uid) != usuarioActual.UID {
        return errors.New("permiso denegado: no es propietario")
    }

    ino.I_perm[0] = perms[0]
    ino.I_perm[1] = perms[1]
    ino.I_perm[2] = perms[2]

    offset := int64(sb.S_inode_start) + int64(idx)*int64(sb.S_inode_s)
    if _, err := f.Seek(offset, 0); err != nil {
        return err
    }
    if err := binary.Write(f, binary.LittleEndian, &ino); err != nil {
        return err
    }


    if sb.S_filesystem_type == 3 {
        contenido := ugo
        if err := RegistrarOperacionJournal(f, sb, sb.S_bm_inode_start, "chmod", ruta, contenido); err != nil {
            fmt.Printf("Advertencia: no se pudo registrar en journal: %v\n", err)
        }
    }

    if recursivo && structs.EsCarpeta(ino) {
        hijos, err := structs.ListaCarpetasFS(partID, ruta)
        if err == nil {
            for _, hijo := range hijos {
                hijoRuta := fmt.Sprintf("%s/%s", ruta, hijo.Nombre)
                _ = Chmod(partID, hijoRuta, ugo, true)
            }
        }
    }

    return nil
}
