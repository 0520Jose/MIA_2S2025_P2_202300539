package commands

import (
    "backend/structs"
    "fmt"
    "os"
    "strings"
    "io"
)

func Remove(params map[string]string) string {
    if usuarioActual == nil {
        return "Error: No hay una sesión activa."
    }
    rawPath, ok := params["-path"]
    if !ok || strings.TrimSpace(rawPath) == "" {
        return "Error: parámetro -path es obligatorio."
    }

    pflag := false
    if v, ok := params["-p"]; ok {
        if strings.TrimSpace(v) != "" {
            return "Error: -p no recibe valor."
        }
        pflag = true
    }

    ruta := unquoteValue(strings.TrimSpace(rawPath))
    if !strings.HasPrefix(ruta, "/") {
        return "Error: -path debe ser ruta absoluta."
    }

    partes := splitPathComponents(ruta)
    if len(partes) == 0 {
        return "Error: ruta invalida"
    }

    nombre := partes[len(partes)-1]
    if nombre == "" {
        return "Error: nombre de recurso vacío"
    }

    disk, sb, err := CargarSistemaEXT2(usuarioActual.PartitionID)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    defer disk.Close()

    pm := getMountByID(usuarioActual.PartitionID)
    if pm == nil {
        return "Error: partición no montada."
    }

    padreIno, err := ensureParentDir(disk, sb, partes[:len(partes)-1], pflag)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }

    childIdx, _ := findEntryInDir(disk, sb, padreIno, nombre)
    if childIdx < 0 {
        return "Error: no existe el recurso especificado."
    }

    hijoInodo, err := ReadInode(disk, sb, int32(childIdx))
    if err != nil {
        return "Error al leer inodo hijo: " + err.Error()
    }

    if !Permisos(&hijoInodo, permWrite) {
        return "Error: no tiene permisos de escritura sobre el recurso."
    }

    if hijoInodo.I_type[0] == 0 {
        if !validarPermisosRecursivos(disk, sb, int32(childIdx)) {
            return "Error: no tiene permisos completos sobre el contenido de la carpeta."
        }
    }

    if err := eliminarInodoRecursivo(disk, sb, int32(childIdx)); err != nil {
        return "Error al eliminar recurso: " + err.Error()
    }

    if err := removerEntradaDePadre(disk, sb, padreIno, int32(childIdx)); err != nil {
        return "Error al limpiar entrada en directorio padre: " + err.Error()
    }

    if err := writeSuperBlock(disk, sb, int64(pm.Partition.Part_start)); err != nil {
        return "Error al actualizar superbloque: " + err.Error()
    }

    if sb.S_filesystem_type == 3 {
        RegistrarOperacionJournal(disk, sb, pm.Partition.Part_start, "remove", ruta, "")
    }

    return "Recurso eliminado correctamente."
}

func validarPermisosRecursivos(f *os.File, sb *structs.SuperBloque, inoIdx int32) bool {
    inodo, err := ReadInode(f, sb, inoIdx)
    if err != nil {
        return false
    }
    if !Permisos(&inodo, permWrite) {
        return false
    }
    if inodo.I_type[0] == 1 {
        return true
    }
    for _, blk := range inodo.I_block {
        if blk == -1 {
            continue
        }
        dir, err := ReadDirBlock(f, sb, blk)
        if err != nil {
            return false
        }
        for _, entry := range dir.B_content {
            if entry.B_inodo != -1 {
                nombre := strings.TrimRight(string(entry.B_name[:]), "\x00")
                if nombre == "." || nombre == ".." {
                    continue
                }
                if !validarPermisosRecursivos(f, sb, entry.B_inodo) {
                    return false
                }
            }
        }
    }
    return true
}

func eliminarInodoRecursivo(f *os.File, sb *structs.SuperBloque, inoIdx int32) error {
    inodo, err := ReadInode(f, sb, inoIdx)
    if err != nil {
        return err
    }
    if inodo.I_type[0] == 0 {
        for _, blk := range inodo.I_block {
            if blk == -1 {
                continue
            }
            dir, err := ReadDirBlock(f, sb, blk)
            if err != nil {
                return err
            }
            for _, entry := range dir.B_content {
                if entry.B_inodo != -1 {
                    nombre := strings.TrimRight(string(entry.B_name[:]), "\x00")
                    if nombre == "." || nombre == ".." {
                        continue
                    }
                    if err := eliminarInodoRecursivo(f, sb, entry.B_inodo); err != nil {
                        return err
                    }
                }
            }
            if err := freeBlock(f, sb, blk); err != nil {
                return err
            }
        }
    } else {
        for _, blk := range inodo.I_block {
            if blk != -1 {
                if err := freeBlock(f, sb, blk); err != nil {
                    return err
                }
            }
        }
    }
    if err := freeInode(f, sb, inoIdx); err != nil {
        return err
    }
    return nil
}

func removerEntradaDePadre(f *os.File, sb *structs.SuperBloque, padreIno int32, childIdx int32) error {
    inodoPadre, err := ReadInode(f, sb, padreIno)
    if err != nil {
        return err
    }
    for _, blk := range inodoPadre.I_block {
        if blk == -1 {
            continue
        }
        dir, err := ReadDirBlock(f, sb, blk)
        if err != nil {
            return err
        }
        changed := false
        for i := range dir.B_content {
            if dir.B_content[i].B_inodo == childIdx {
                dir.B_content[i].B_inodo = -1
                for j := range dir.B_content[i].B_name {
                    dir.B_content[i].B_name[j] = 0
                }
                changed = true
            }
        }
        if changed {
            if err := writeDirBlock(f, sb, blk, &dir); err != nil {
                return err
            }
        }
    }
    return nil
}

func freeBlock(f *os.File, sb *structs.SuperBloque, blockIdx int32) error {
    if blockIdx < 0 || blockIdx >= sb.S_blocks_count {
        return fmt.Errorf("índice de bloque inválido: %d", blockIdx)
    }

    bm := structs.GetBitmapBlocks(f, sb)
    if bm == nil {
        return fmt.Errorf("no se pudo leer el bitmap de bloques")
    }

    if bm[blockIdx] == 1 {
        bm[blockIdx] = 0
        sb.S_free_blocks_count++

        if _, err := f.Seek(int64(sb.S_bm_block_start), io.SeekStart); err != nil {
            return err
        }
        if _, err := f.Write(bm); err != nil {
            return err
        }
    }

    return nil
}

func freeInode(f *os.File, sb *structs.SuperBloque, inodeIdx int32) error {
    if inodeIdx < 0 || inodeIdx >= sb.S_inodes_count {
        return fmt.Errorf("índice de inodo inválido: %d", inodeIdx)
    }

    bm := structs.BitMapInodos(f, sb)
    if bm == nil {
        return fmt.Errorf("no se pudo leer el bitmap de inodos")
    }

    if bm[inodeIdx] == 1 {
        bm[inodeIdx] = 0
        sb.S_free_inodes_count++

        if _, err := f.Seek(int64(sb.S_bm_inode_start), io.SeekStart); err != nil {
            return err
        }
        if _, err := f.Write(bm); err != nil {
            return err
        }
    }

    return nil
}