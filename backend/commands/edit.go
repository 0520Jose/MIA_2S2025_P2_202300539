package commands

import (
    "backend/structs"
    "fmt"
    "io"
    "os"
    "strings"
)

func Edit(params map[string]string) string {
    if usuarioActual == nil {
        return "Error: No hay una sesión activa."
    }

    rawPath, ok := params["-path"]
    if !ok || strings.TrimSpace(rawPath) == "" {
        return "Error: parámetro -path es obligatorio."
    }
    ruta := unquoteValue(strings.TrimSpace(rawPath))
    if !strings.HasPrefix(ruta, "/") {
        return "Error: -path debe ser ruta absoluta."
    }

    rawCont, ok := params["-contenido"]
    if !ok {
        return "Error: parámetro -contenido es obligatorio."
    }
    contenidoArg := unquoteValue(strings.TrimSpace(rawCont))

    disk, sb, err := CargarSistemaEXT2(usuarioActual.PartitionID)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    defer disk.Close()

    pm := getMountByID(usuarioActual.PartitionID)
    if pm == nil {
        return "Error: partición no montada."
    }

    inoIdx, err := FindInodeByPath(disk, sb, ruta)
    if err != nil {
        return "Error: " + err.Error()
    }

    inodo, err := ReadInode(disk, sb, inoIdx)
    if err != nil {
        return "Error al leer inodo: " + err.Error()
    }

    if inodo.I_type[0] != 1 {
        return "Error: la ruta no corresponde a un archivo."
    }

    if !Permisos(&inodo, permWrite) {
        return "Error: permiso denegado sobre el archivo."
    }

    var contenido string
    if strings.HasPrefix(contenidoArg, "/") {
        dataStr, err := structs.LeerArchivoDeFS(usuarioActual.PartitionID, contenidoArg)
        if err != nil {
            // Si no se encuentra en EXT2, intenta leer desde el sistema de archivos del SO
            fileData, fileErr := os.ReadFile(contenidoArg)
            if fileErr != nil {
                return "Error al leer archivo fuente: " + fileErr.Error()
            }
            contenido = string(fileData)
        } else {
            contenido = dataStr
        }
    } else {
        contenido = contenidoArg
    }

    if err := liberarBloquesArchivo(disk, sb, &inodo); err != nil {
        return "Error al limpiar bloques previos: " + err.Error()
    }

    data := []byte(contenido)
    inodo.I_s = int32(len(data))
    t := fecha17()
    copy(inodo.I_mtime[:], t)
    copy(inodo.I_ctime[:], t)

    for i := range inodo.I_block {
        inodo.I_block[i] = -1
    }

    if err := asignarBloquesArchivo(disk, sb, &inodo, data, pm.Partition.Part_start, ruta); err != nil {
        return "Error al asignar bloques: " + err.Error()
    }

    if err := writeInode(disk, sb, inoIdx, &inodo); err != nil {
        return "Error al escribir inodo actualizado: " + err.Error()
    }

    if err := writeSuperBlock(disk, sb, int64(pm.Partition.Part_start)); err != nil {
        return "Error al actualizar superbloque: " + err.Error()
    }

    if sb.S_filesystem_type == 3 {
        partStart := pm.Partition.Part_start
        RegistrarOperacionJournal(disk, sb, partStart, "edit", ruta, contenido)
    }

    return "Archivo editado correctamente."
}

func liberarBloquesArchivo(f *os.File, sb *structs.SuperBloque, inodo *structs.Inodo) error {
    for i := 0; i < DIRECT_BLOCKS; i++ {
        if inodo.I_block[i] != -1 {
            if err := liberarBloque(f, sb, inodo.I_block[i]); err != nil {
                return fmt.Errorf("error liberando bloque %d: %v", inodo.I_block[i], err)
            }
            inodo.I_block[i] = -1
        }
    }

    if inodo.I_block[INDIRECT_SIMPLE] != -1 {
        ptrBlock, err := readPointerBlock(f, sb, inodo.I_block[INDIRECT_SIMPLE])
        if err == nil {
            for _, ptr := range ptrBlock.B_pointers {
                if ptr != -1 {
                    if err := liberarBloque(f, sb, ptr); err != nil {
                        return err
                    }
                }
            }
        }
        if err := liberarBloque(f, sb, inodo.I_block[INDIRECT_SIMPLE]); err != nil {
            return err
        }
        inodo.I_block[INDIRECT_SIMPLE] = -1
    }

    return nil
}

func liberarBloque(f *os.File, sb *structs.SuperBloque, blockIdx int32) error {
    if blockIdx < 0 {
        return nil
    }

    offset := int64(sb.S_bm_block_start) + int64(blockIdx)
    if _, err := f.Seek(offset, io.SeekStart); err != nil {
        return err
    }

    if _, err := f.Write([]byte{0}); err != nil {
        return err
    }

    sb.S_free_blocks_count++
    return nil
}