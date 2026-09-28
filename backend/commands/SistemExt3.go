package commands

import (
    "backend/structs"
    "encoding/binary"
    "fmt"
    "io"
    "os"
    "strings"
    "strconv"
    "time"
)

func RecuperarEXT3(f *os.File, partStart int32, sb *structs.SuperBloque) error {
    if !sb.EsEXT3() {
        return fmt.Errorf("no es un sistema EXT3")
    }

    if err := recrearEstructuraBaseEXT3(f, sb, partStart); err != nil {
        return fmt.Errorf("error al recrear estructura base: %v", err)
    }

    operaciones, err := ObtenerOperacionesJournal(f, partStart)
    if err != nil {
        return err
    }

    operacionesProcesadas := make(map[string]bool)

    for _, op := range operaciones {
        operacion := strings.Trim(string(op.Operation[:]), "\x00")
        ruta := strings.Trim(string(op.Path[:]), "\x00")
        contenido := strings.Trim(string(op.Content[:]), "\x00")

        if !strings.HasPrefix(ruta, "/") && ruta != "" {
            ruta = "/" + ruta
        }

        clave := fmt.Sprintf("%s:%s", operacion, ruta)
        if operacionesProcesadas[clave] {
            fmt.Printf("Saltando operación duplicada: %s en %s\n", operacion, ruta)
            continue
        }
        operacionesProcesadas[clave] = true

        fmt.Printf("Recuperando operación: %s en %s\n", operacion, ruta)
        
        switch operacion {
        case "MKFS":
            fmt.Println("Operación MKFS detectada (ya procesada)")
        
        case "mkdir":
            if err := recuperarMkdir(f, sb, partStart, ruta); err != nil {
                fmt.Printf("Error al recuperar mkdir %s: %v\n", ruta, err)
            }
        
        case "mkfile":
            if err := recuperarMkfile(f, sb, partStart, ruta, contenido); err != nil {
                fmt.Printf("Error al recuperar mkfile %s: %v\n", ruta, err)
            }
        
        case "write", "create", "modify":
            if err := recuperarWrite(f, sb, partStart, ruta, contenido); err != nil {
                fmt.Printf("Error al recuperar %s %s: %v\n", operacion, ruta, err)
            }
        
        case "remove", "delete":
            if err := recuperarRemove(f, sb, partStart, ruta, contenido); err != nil {
                fmt.Printf("Error al recuperar remove %s: %v\n", ruta, err)
            }
        
        case "edit":
            if err := recuperarEdit(f, sb, partStart, ruta, contenido); err != nil {
                fmt.Printf("Error al recuperar edit %s: %v\n", ruta, err)
            }
        
        case "rename":
            if err := recuperarRename(f, sb, partStart, ruta, contenido); err != nil {
                fmt.Printf("Error al recuperar rename %s: %v\n", ruta, err)
            }
        
        case "copy":
            if err := recuperarCopy(f, sb, partStart, ruta, contenido); err != nil {
                fmt.Printf("Error al recuperar copy %s: %v\n", ruta, err)
            }
        
        case "move":
            if err := recuperarMove(f, sb, partStart, ruta, contenido); err != nil {
                fmt.Printf("Error al recuperar move %s: %v\n", ruta, err)
            }
        
        case "chown":
            if err := recuperarChown(f, sb, partStart, ruta, contenido); err != nil {
                fmt.Printf("Error al recuperar chown %s: %v\n", ruta, err)
            }
        
        case "chmod":
            if err := recuperarChmod(f, sb, partStart, ruta, contenido); err != nil {
                fmt.Printf("Error al recuperar chmod %s: %v\n", ruta, err)
            }
        case "mkgrp":
            if err := recuperarMkgrp(f, sb, partStart, ruta, contenido); err != nil {
                fmt.Printf("Error al recuperar mkgrp %s: %v\n", contenido, err)
            }

        case "rmgrp":
            if err := recuperarRmgrp(f, sb, partStart, ruta, contenido); err != nil {
                fmt.Printf("Error al recuperar rmgrp %s: %v\n", contenido, err)
            }

        case "mkusr":
            if err := recuperarMkusr(f, sb, partStart, ruta, contenido); err != nil {
                fmt.Printf("Error al recuperar mkusr %s: %v\n", contenido, err)
            }

        case "rmusr":
            if err := recuperarRmusr(f, sb, partStart, ruta, contenido); err != nil {
                fmt.Printf("Error al recuperar rmusr %s: %v\n", contenido, err)
            }

        case "chgrp":
            if err := recuperarChgrp(f, sb, partStart, ruta, contenido); err != nil {
                fmt.Printf("Error al recuperar chgrp %s: %v\n", contenido, err)
            }
        default:
            fmt.Printf("Operación no soportada en recuperación: %s\n", operacion)
        }
    }

    fmt.Println("Recuperación EXT3 completada exitosamente")
    return nil
}

func recrearEstructuraBaseEXT3(f *os.File, sb *structs.SuperBloque, partStart int32) error {
    fmt.Println("Recreando estructura base del filesystem EXT3...")
    
    if err := inicializarBitmapInodos(f, sb); err != nil {
        return err
    }
    
    if err := inicializarBitmapBloques(f, sb); err != nil {
        return err
    }
    
    if err := crearInodoRaiz(f, sb); err != nil {
        return err
    }
    
    if err := crearBloqueDirectorioRaiz(f, sb); err != nil {
        return err
    }
    
    if err := crearInodoHome(f, sb); err != nil {
        return err
    }
    
    if err := crearBloqueDirectorioHome(f, sb); err != nil {
        return err
    }
    
    if err := crearInodoUsersTxt(f, sb); err != nil {
        return err
    }
    
    if err := crearBloqueUsersTxt(f, sb); err != nil {
        return err
    }
    
    fmt.Println("Estructura base recreada exitosamente")
    return nil
}

func inicializarBitmapInodos(f *os.File, sb *structs.SuperBloque) error {
    if _, err := f.Seek(int64(sb.S_bm_inode_start), 0); err != nil {
        return err
    }
    
    bitmap := make([]byte, sb.S_inodes_count)
    if sb.S_inodes_count >= 1 {
        bitmap[0] = 1
    }
    if sb.S_inodes_count >= 2 {
        bitmap[1] = 1
    }
    if sb.S_inodes_count >= 3 {
        bitmap[2] = 1
    }
    
    if _, err := f.Write(bitmap); err != nil {
        return err
    }
    
    return nil
}

func inicializarBitmapBloques(f *os.File, sb *structs.SuperBloque) error {
    if _, err := f.Seek(int64(sb.S_bm_block_start), 0); err != nil {
        return err
    }
    
    bitmap := make([]byte, sb.S_blocks_count)
    if sb.S_blocks_count >= 1 {
        bitmap[0] = 1
    }
    if sb.S_blocks_count >= 2 {
        bitmap[1] = 1
    }
    if sb.S_blocks_count >= 3 {
        bitmap[2] = 1
    }
    
    if _, err := f.Write(bitmap); err != nil {
        return err
    }
    
    return nil
}

func crearInodoRaiz(f *os.File, sb *structs.SuperBloque) error {
    var ino structs.Inodo
    ino.I_uid = 1
    ino.I_gid = 1
    ino.I_s = int32(binary.Size(structs.BCarpeta{}))
    t := fecha17()
    copy(ino.I_atime[:], t)
    copy(ino.I_ctime[:], t)
    copy(ino.I_mtime[:], t)
    
    for i := range ino.I_block {
        ino.I_block[i] = -1
    }
    ino.I_block[0] = 0
    
    ino.I_type[0] = 0
    copy(ino.I_perm[:], "755")
    
    offset := int64(sb.S_inode_start) + 0*int64(sb.S_inode_s)
    if _, err := f.Seek(offset, 0); err != nil {
        return err
    }
    
    return binary.Write(f, binary.LittleEndian, &ino)
}

func crearBloqueDirectorioRaiz(f *os.File, sb *structs.SuperBloque) error {
    var dir structs.BCarpeta
    
    for i := range dir.B_content {
        dir.B_content[i].B_inodo = -1
        for j := range dir.B_content[i].B_name {
            dir.B_content[i].B_name[j] = 0
        }
    }
    
    dir.B_content[0].B_inodo = 0
    copy(dir.B_content[0].B_name[:], ".")
    
    dir.B_content[1].B_inodo = 0
    copy(dir.B_content[1].B_name[:], "..")
    
    dir.B_content[2].B_inodo = 1
    copy(dir.B_content[2].B_name[:], "home")
    
    offset := int64(sb.S_block_start) + 0*int64(sb.S_block_s)
    if _, err := f.Seek(offset, 0); err != nil {
        return err
    }
    
    return binary.Write(f, binary.LittleEndian, &dir)
}

func crearInodoHome(f *os.File, sb *structs.SuperBloque) error {
    var ino structs.Inodo
    ino.I_uid = 1
    ino.I_gid = 1
    ino.I_s = int32(binary.Size(structs.BCarpeta{}))
    t := fecha17()
    copy(ino.I_atime[:], t)
    copy(ino.I_ctime[:], t)
    copy(ino.I_mtime[:], t)
    
    for i := range ino.I_block {
        ino.I_block[i] = -1
    }
    ino.I_block[0] = 1
    
    ino.I_type[0] = 0
    copy(ino.I_perm[:], "755")
    
    offset := int64(sb.S_inode_start) + 1*int64(sb.S_inode_s)
    if _, err := f.Seek(offset, 0); err != nil {
        return err
    }
    
    return binary.Write(f, binary.LittleEndian, &ino)
}

func crearBloqueDirectorioHome(f *os.File, sb *structs.SuperBloque) error {
    var dir structs.BCarpeta
    
    for i := range dir.B_content {
        dir.B_content[i].B_inodo = -1
        for j := range dir.B_content[i].B_name {
            dir.B_content[i].B_name[j] = 0
        }
    }
    
    dir.B_content[0].B_inodo = 1
    copy(dir.B_content[0].B_name[:], ".")
    
    dir.B_content[1].B_inodo = 0
    copy(dir.B_content[1].B_name[:], "..")
    
    dir.B_content[2].B_inodo = 2
    copy(dir.B_content[2].B_name[:], "users.txt")
    
    offset := int64(sb.S_block_start) + 1*int64(sb.S_block_s)
    if _, err := f.Seek(offset, 0); err != nil {
        return err
    }
    
    return binary.Write(f, binary.LittleEndian, &dir)
}

func crearInodoUsersTxt(f *os.File, sb *structs.SuperBloque) error {
    contenido := "1,G,root\n1,U,root,root,123\n"
    
    var ino structs.Inodo
    ino.I_uid = 1
    ino.I_gid = 1
    ino.I_s = int32(len(contenido))
    t := fecha17()
    copy(ino.I_atime[:], t)
    copy(ino.I_ctime[:], t)
    copy(ino.I_mtime[:], t)
    
    for i := range ino.I_block {
        ino.I_block[i] = -1
    }
    ino.I_block[0] = 2
    
    ino.I_type[0] = 1
    copy(ino.I_perm[:], "644")
    
    offset := int64(sb.S_inode_start) + 2*int64(sb.S_inode_s)
    if _, err := f.Seek(offset, 0); err != nil {
        return err
    }
    
    return binary.Write(f, binary.LittleEndian, &ino)
}

func crearBloqueUsersTxt(f *os.File, sb *structs.SuperBloque) error {
    contenido := "1,G,root\n1,U,root,root,123\n"
    
    var archivo structs.BArchivo
    copy(archivo.B_content[:], []byte(contenido))
    
    offset := int64(sb.S_block_start) + 2*int64(sb.S_block_s)
    if _, err := f.Seek(offset, 0); err != nil {
        return err
    }
    
    return binary.Write(f, binary.LittleEndian, &archivo)
}

func crearDirectorioRecuperacion(f *os.File, sb *structs.SuperBloque, parentIno int32, name string) (int32, error) {
    idxIno, err := allocInode(f, sb)
    if err != nil {
        return -1, err
    }
    idxBlk, err := allocBlock(f, sb)
    if err != nil {
        return -1, err
    }

    var ino structs.Inodo
    ino.I_uid = 1
    ino.I_gid = 1
    ino.I_s = 0
    t := fecha17()
    copy(ino.I_atime[:], t)
    copy(ino.I_ctime[:], t)
    copy(ino.I_mtime[:], t)
    for i := range ino.I_block {
        ino.I_block[i] = -1
    }
    ino.I_block[0] = idxBlk
    ino.I_type[0] = 0
    copy(ino.I_perm[:], "755")
    if err := writeInode(f, sb, idxIno, &ino); err != nil {
        return -1, err
    }

    var dir structs.BCarpeta
    for i := range dir.B_content {
        dir.B_content[i].B_inodo = -1
        for j := range dir.B_content[i].B_name {
            dir.B_content[i].B_name[j] = 0
        }
    }
    dir.B_content[0].B_inodo = idxIno
    copy(dir.B_content[0].B_name[:], ".")
    dir.B_content[1].B_inodo = parentIno
    copy(dir.B_content[1].B_name[:], "..")
    if err := writeDirBlock(f, sb, idxBlk, &dir); err != nil {
        return -1, err
    }

    if err := addDirEntry(f, sb, parentIno, name, idxIno); err != nil {
        return -1, err
    }
    return idxIno, nil
}

func recuperarMkdir(f *os.File, sb *structs.SuperBloque, partStart int32, ruta string) error {
    if ruta == "" || ruta == "/" {
        return nil
    }

    partes := splitPathComponents(ruta)
    if len(partes) == 0 {
        return nil
    }

    currentIno := int32(0)

    for _, parte := range partes {
        childIno, err := findEntryInDir(f, sb, currentIno, parte)
        if err != nil {
            childInoInt32, err := crearDirectorioRecuperacion(f, sb, currentIno, parte)
            if err != nil {
                return fmt.Errorf("error al crear directorio %s: %v", parte, err)
            }
            childIno = int(childInoInt32)
            fmt.Printf("  Directorio creado: %s (inodo: %d)\n", parte, childInoInt32)
        }
        currentIno = int32(childIno)
    }

    return nil
}

func recuperarMkfile(f *os.File, sb *structs.SuperBloque, partStart int32, ruta, contenido string) error {
    return recuperarWrite(f, sb, partStart, ruta, contenido)
}

func recuperarWrite(f *os.File, sb *structs.SuperBloque, partStart int32, ruta, contenido string) error {
    if ruta == "" || ruta == "/" {
        return fmt.Errorf("ruta inválida para archivo")
    }

    partes := splitPathComponents(ruta)
    if len(partes) == 0 {
        return fmt.Errorf("ruta inválida")
    }

    dirParts := partes[:len(partes)-1]
    currentIno := int32(0)

    for _, parte := range dirParts {
        childIno, err := findEntryInDir(f, sb, currentIno, parte)
        if err != nil {
            childInoInt32, err := crearDirectorioRecuperacion(f, sb, currentIno, parte)
            if err != nil {
                return fmt.Errorf("error al crear directorio padre %s: %v", parte, err)
            }
            childIno = int(childInoInt32)
        }
        currentIno = int32(childIno)
    }

    nombreArchivo := partes[len(partes)-1]
    
    existingIno, err := findEntryInDir(f, sb, currentIno, nombreArchivo)
    if err == nil {
        inodo, err := ReadInode(f, sb, int32(existingIno))
        if err != nil {
            return fmt.Errorf("error al leer inodo existente: %v", err)
        }
        
        if inodo.I_type[0] != 1 {
            return fmt.Errorf("la ruta existe pero no es un archivo")
        }

        return actualizarContenidoArchivo(f, sb, int32(existingIno), contenido)
    }

    return crearArchivoConContenido(f, sb, currentIno, nombreArchivo, contenido)
}

func recuperarRemove(f *os.File, sb *structs.SuperBloque, partStart int32, ruta, contenido string) error {
    fmt.Printf("Recuperando remove: %s\n", ruta)
    
    inodoIdx, err := FindInodeByPath(f, sb, ruta)
    if err != nil {
        fmt.Printf("  Archivo no encontrado para eliminar: %s\n", ruta)
        return nil
    }
    
    inodoPadre, nombreArchivo, err := buscarPadreYNombre(f, sb, ruta)
    if err != nil {
        return fmt.Errorf("error al buscar padre: %v", err)
    }
    
    if err := eliminarEntradaDirectorio(f, sb, inodoPadre, nombreArchivo); err != nil {
        return fmt.Errorf("error al eliminar entrada del directorio: %v", err)
    }
    
    if err := liberarInodoYBloques(f, sb, inodoIdx); err != nil {
        return fmt.Errorf("error al liberar inodo y bloques: %v", err)
    }
    
    fmt.Printf("  Archivo eliminado: %s (inodo: %d)\n", ruta, inodoIdx)
    return nil
}

func eliminarEntradaDirectorio(f *os.File, sb *structs.SuperBloque, inodoPadre int32, nombre string) error {
    inodo, err := ReadInode(f, sb, inodoPadre)
    if err != nil {
        return err
    }
    
    for i := 0; i < len(inodo.I_block); i++ {
        if inodo.I_block[i] == -1 {
            continue
        }
        
        dir, err := ReadDirBlock(f, sb, inodo.I_block[i])
        if err != nil {
            continue
        }
        
        for j := range dir.B_content {
            entryNombre := strings.TrimRight(string(dir.B_content[j].B_name[:]), "\x00")
            if entryNombre == nombre {
                dir.B_content[j].B_inodo = -1
                for k := range dir.B_content[j].B_name {
                    dir.B_content[j].B_name[k] = 0
                }
                
                offset := int64(sb.S_block_start) + int64(inodo.I_block[i])*int64(sb.S_block_s)
                if _, err := f.Seek(offset, 0); err != nil {
                    return err
                }
                return binary.Write(f, binary.LittleEndian, &dir)
            }
        }
    }
    
    return fmt.Errorf("entrada no encontrada en directorio")
}

func liberarInodoYBloques(f *os.File, sb *structs.SuperBloque, inodoIdx int32) error {
    inodo, err := ReadInode(f, sb, inodoIdx)
    if err != nil {
        return err
    }
    
    for i := 0; i < len(inodo.I_block); i++ {
        if inodo.I_block[i] != -1 {
            if err := freeBlock(f, sb, inodo.I_block[i]); err != nil {
                return err
            }
            inodo.I_block[i] = -1
        }
    }
    
    if err := freeInode(f, sb, inodoIdx); err != nil {
        return err
    }
    
    return nil
}

func recuperarEdit(f *os.File, sb *structs.SuperBloque, partStart int32, ruta, contenido string) error {
    fmt.Printf("Recuperando edit: %s\n", ruta)
    
    if strings.HasPrefix(contenido, "/") {
        contenidoLeido, err := structs.LeerArchivoDeFSFromFile(f, sb, partStart, contenido)
        if err != nil {
            fmt.Printf("  No se pudo leer archivo fuente %s: %v\n", contenido, err)
            return recuperarWrite(f, sb, partStart, ruta, "[contenido no disponible]")
        }
        return recuperarWrite(f, sb, partStart, ruta, contenidoLeido)
    }
    
    return recuperarWrite(f, sb, partStart, ruta, contenido)
}

func recuperarRename(f *os.File, sb *structs.SuperBloque, partStart int32, rutaVieja, nuevoNombre string) error {
    fmt.Printf("Recuperando rename: %s -> %s\n", rutaVieja, nuevoNombre)
    
    inodoIdx, err := FindInodeByPath(f, sb, rutaVieja)
    if err != nil {
        partes := splitPathComponents(rutaVieja)
        if len(partes) == 0 {
            return fmt.Errorf("ruta inválida")
        }
        
        partes[len(partes)-1] = nuevoNombre
        rutaNueva := "/" + strings.Join(partes, "/")
        
        fmt.Printf("Recurso original no encontrado, creando nuevo: %s\n", rutaNueva)
        if strings.Contains(rutaNueva, ".") {
            return recuperarWrite(f, sb, partStart, rutaNueva, "[contenido renombrado]")
        }
        return recuperarMkdir(f, sb, partStart, rutaNueva)
    }
    
    inodoPadre, _, err := buscarPadreYNombre(f, sb, rutaVieja)
    if err != nil {
        return err
    }
    
    return actualizarNombreEnPadre(f, sb, inodoPadre, inodoIdx, nuevoNombre)
}

func recuperarCopy(f *os.File, sb *structs.SuperBloque, partStart int32, destino, origen string) error {
    fmt.Printf("Recuperando copy: %s -> %s\n", origen, destino)
    
    inodoDestino, _, err := structs.BuscarInodoPorRuta_(f, sb, destino)
    if err == nil && structs.EsCarpeta(inodoDestino) {
        nombreOrigen := obtenerNombreRuta(origen)
        destino = destino + "/" + nombreOrigen
    }
    
    inodoOrigen, _, err := structs.BuscarInodoPorRuta_(f, sb, origen)
    if err != nil {
        fmt.Printf("  Origen no encontrado: %s\n", origen)
        return nil
    }
    
    if structs.EsCarpeta(inodoOrigen) {
        return copiarRecursivoRecuperacion(f, sb, partStart, origen, destino)
    } else {
        contenido, err := leerContenidoArchivo(f, sb, inodoOrigen)
        if err != nil {
            return err
        }
        return recuperarWrite(f, sb, partStart, destino, string(contenido))
    }
}

func recuperarMove(f *os.File, sb *structs.SuperBloque, partStart int32, origen, destino string) error {
    fmt.Printf("Recuperando move: %s -> %s\n", origen, destino)
    
    inodoDestino, _, err := structs.BuscarInodoPorRuta_(f, sb, destino)
    if err == nil && structs.EsCarpeta(inodoDestino) {
        nombreArchivo := obtenerNombreRuta(origen)
        destino = destino + "/" + nombreArchivo
    }
    
    err = recuperarCopy(f, sb, partStart, destino, origen)
    if err != nil {
        return err
    }
    
    if err := recuperarRemove(f, sb, partStart, origen, ""); err != nil {
        fmt.Printf("  No se pudo eliminar origen después del move: %v\n", err)
    }
    
    fmt.Printf("  Move completado: %s -> %s\n", origen, destino)
    return nil
}

func recuperarChown(f *os.File, sb *structs.SuperBloque, partStart int32, ruta, usuario string) error {
    fmt.Printf("Recuperando chown: %s -> %s\n", ruta, usuario)
    
    inodo, idx, err := structs.BuscarInodoPorRuta_(f, sb, ruta)
    if err != nil {
        fmt.Printf("  Ruta no encontrada: %s\n", ruta)
        return nil
    }
    
    usuarios, err := structs.LeerArchivoDeFSFromFile(f, sb, partStart, "/home/users.txt")
    if err != nil {
        fmt.Printf("  No se pudo leer users.txt: %v\n", err)
        return nil
    }
    
    uidNuevo := int32(1)
    lines := strings.Split(usuarios, "\n")
    for _, line := range lines {
        campos := strings.Split(line, ",")
        if len(campos) >= 4 && campos[1] == "U" && campos[3] == usuario {
            if uid, err := strconv.Atoi(campos[0]); err == nil {
                uidNuevo = int32(uid)
                break
            }
        }
    }
    
    inodo.I_uid = uidNuevo
    
    offset := int64(sb.S_inode_start) + int64(idx)*int64(sb.S_inode_s)
    if _, err := f.Seek(offset, 0); err != nil {
        return err
    }
    if err := binary.Write(f, binary.LittleEndian, &inodo); err != nil {
        return err
    }
    
    fmt.Printf("  Propietario actualizado: %s -> UID:%d\n", ruta, uidNuevo)
    return nil
}

func recuperarChmod(f *os.File, sb *structs.SuperBloque, partStart int32, ruta, permisos string) error {
    fmt.Printf("Recuperando chmod: %s -> %s\n", ruta, permisos)
    
    if len(permisos) != 3 {
        fmt.Printf("  Permisos inválidos: %s\n", permisos)
        return nil
    }
    
    inodo, idx, err := structs.BuscarInodoPorRuta_(f, sb, ruta)
    if err != nil {
        fmt.Printf("  Ruta no encontrada: %s\n", ruta)
        return nil
    }
    
    perms := [3]byte{}
    for i := 0; i < 3; i++ {
        val, err := strconv.Atoi(string(permisos[i]))
        if err != nil || val < 0 || val > 7 {
            fmt.Printf("  Permiso inválido: %c\n", permisos[i])
            return nil
        }
        perms[i] = byte(val)
    }
    
    inodo.I_perm[0] = perms[0]
    inodo.I_perm[1] = perms[1]
    inodo.I_perm[2] = perms[2]
    
    offset := int64(sb.S_inode_start) + int64(idx)*int64(sb.S_inode_s)
    if _, err := f.Seek(offset, 0); err != nil {
        return err
    }
    if err := binary.Write(f, binary.LittleEndian, &inodo); err != nil {
        return err
    }
    
    fmt.Printf("  Permisos actualizados: %s -> %s\n", ruta, permisos)
    return nil
}

func crearArchivoConContenido(f *os.File, sb *structs.SuperBloque, dirIno int32, nombre, contenido string) error {
    idxIno, err := allocInode(f, sb)
    if err != nil {
        return err
    }

    var ino structs.Inodo
    ino.I_uid = 1
    ino.I_gid = 1
    ino.I_s = int32(len(contenido))
    t := fecha17()
    copy(ino.I_atime[:], t)
    copy(ino.I_ctime[:], t)
    copy(ino.I_mtime[:], t)
    for i := range ino.I_block {
        ino.I_block[i] = -1
    }
    ino.I_type[0] = 1
    copy(ino.I_perm[:], "644")

    data := []byte(contenido)
    totalBloques := (len(data) + 63) / 64
    if totalBloques > 12 {
        totalBloques = 12
    }

    for i := 0; i < totalBloques; i++ {
        idxBlk, err := allocBlock(f, sb)
        if err != nil {
            return err
        }
        ino.I_block[i] = idxBlk

        var block structs.BArchivo
        inicio := i * 64
        fin := min((i+1)*64, len(data))
        copy(block.B_content[:], data[inicio:fin])

        offset := int64(sb.S_block_start) + int64(idxBlk)*64
        if _, err := f.Seek(offset, io.SeekStart); err != nil {
            return err
        }
        if err := binary.Write(f, binary.LittleEndian, &block); err != nil {
            return err
        }
    }

    if err := writeInode(f, sb, idxIno, &ino); err != nil {
        return err
    }

    if err := addDirEntry(f, sb, dirIno, nombre, idxIno); err != nil {
        return err
    }

    fmt.Printf("  Archivo creado: %s (inodo: %d, tamaño: %d bytes)\n", nombre, idxIno, len(contenido))
    return nil
}

func actualizarContenidoArchivo(f *os.File, sb *structs.SuperBloque, inodoIdx int32, contenido string) error {
    inodo, err := ReadInode(f, sb, inodoIdx)
    if err != nil {
        return err
    }

    for i := 0; i < 12; i++ {
        if inodo.I_block[i] != -1 {
            if err := freeBlock(f, sb, inodo.I_block[i]); err != nil {
                return err
            }
            inodo.I_block[i] = -1
        }
    }

    data := []byte(contenido)
    inodo.I_s = int32(len(data))
    t := fecha17()
    copy(inodo.I_mtime[:], t)

    totalBloques := (len(data) + 63) / 64
    if totalBloques > 12 {
        totalBloques = 12
    }

    for i := 0; i < totalBloques; i++ {
        idxBlk, err := allocBlock(f, sb)
        if err != nil {
            return err
        }
        inodo.I_block[i] = idxBlk

        var block structs.BArchivo
        inicio := i * 64
        fin := min((i+1)*64, len(data))
        copy(block.B_content[:], data[inicio:fin])

        offset := int64(sb.S_block_start) + int64(idxBlk)*64
        if _, err := f.Seek(offset, io.SeekStart); err != nil {
            return err
        }
        if err := binary.Write(f, binary.LittleEndian, &block); err != nil {
            return err
        }
    }

    if err := writeInode(f, sb, inodoIdx, &inodo); err != nil {
        return err
    }

    fmt.Printf("  Archivo actualizado: (inodo: %d, tamaño: %d bytes)\n", inodoIdx, len(contenido))
    return nil
}

func copiarRecursivoRecuperacion(f *os.File, sb *structs.SuperBloque, partStart int32, origen, destino string) error {
    inodoOrigen, _, err := structs.BuscarInodoPorRuta_(f, sb, origen)
    if err != nil {
        return err
    }
    
    if !structs.EsCarpeta(inodoOrigen) {
        contenido, err := leerContenidoArchivo(f, sb, inodoOrigen)
        if err != nil {
            return err
        }
        return recuperarWrite(f, sb, partStart, destino, string(contenido))
    }

    if err := recuperarMkdir(f, sb, partStart, destino); err != nil {
        return err
    }

    for i := 0; i < len(inodoOrigen.I_block); i++ {
        if inodoOrigen.I_block[i] == -1 {
            continue
        }
        
        dir, err := ReadDirBlock(f, sb, inodoOrigen.I_block[i])
        if err != nil {
            continue
        }
        
        for _, entry := range dir.B_content {
            if entry.B_inodo == -1 {
                continue
            }
            
            nombreEntry := strings.TrimRight(string(entry.B_name[:]), "\x00")
            if nombreEntry == "." || nombreEntry == ".." {
                continue
            }
            
            subOrigen := origen + "/" + nombreEntry
            subDestino := destino + "/" + nombreEntry
            
            if err := copiarRecursivoRecuperacion(f, sb, partStart, subOrigen, subDestino); err != nil {
                fmt.Printf("Error copiando %s: %v\n", subOrigen, err)
            }
        }
    }
    
    return nil
}

func obtenerNombreRuta(ruta string) string {
    ruta = strings.Trim(ruta, "/")
    if ruta == "" {
        return ""
    }
    partes := strings.Split(ruta, "/")
    return partes[len(partes)-1]
}

func leerContenidoArchivo(f *os.File, sb *structs.SuperBloque, inodo structs.Inodo) ([]byte, error) {
    var contenido []byte
    
    for i := 0; i < structs.DIRECT_BLOCKS; i++ {
        if inodo.I_block[i] == -1 {
            break
        }
        
        bloque, ok := structs.LeerBloqueArchivo(f, sb, inodo.I_block[i])
        if !ok {
            continue
        }
        
        contenido = append(contenido, bloque.B_content[:]...)
    }
    
    return contenido, nil
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

func SimularPerdidaEXT3(f *os.File, sb *structs.SuperBloque, partStart int32) error {
    areas := []struct {
        start int32
        size  int32
    }{
        {sb.S_bm_inode_start, sb.S_inodes_count},
        {sb.S_bm_block_start, sb.S_blocks_count},
        {sb.S_inode_start, sb.S_inode_s * sb.S_inodes_count},
        {sb.S_block_start, sb.S_block_s * sb.S_blocks_count},
    }

    for _, area := range areas {
        if _, err := f.Seek(int64(area.start), io.SeekStart); err != nil {
            return err
        }
        zeros := make([]byte, area.size)
        if _, err := f.Write(zeros); err != nil {
            return err
        }
    }

    return nil
}

func RegistrarOperacion(f *os.File, partStart int32, operacion, ruta, contenido string) error {
    journal, err := structs.LeerJournal(f, partStart)
    if err != nil {
        journal = structs.Journal{Count: 0}
    }

    if journal.Count >= int32(len(journal.Content)) {
        journal.Count = 0
    }

    idx := journal.Count
    copy(journal.Content[idx].Operation[:], operacion)
    copy(journal.Content[idx].Path[:], ruta)
    copy(journal.Content[idx].Content[:], contenido)
    journal.Content[idx].Date = float32(time.Now().Unix())
    journal.Count++

    return structs.EscribirJournal(f, partStart, &journal)
}

func RegistrarOperacionJournal(f *os.File, sb *structs.SuperBloque, partStart int32, operacion, ruta, contenido string) error {
    return RegistrarOperacion(f, partStart, operacion, ruta, contenido)
}

func ObtenerOperacionesJournal(f *os.File, partStart int32) ([]structs.Information, error) {
    journal, err := structs.LeerJournal(f, partStart)
    if err != nil {
        return nil, err
    }

    operaciones := make([]structs.Information, 0)
    for i := int32(0); i < journal.Count && i < int32(len(journal.Content)); i++ {
        operaciones = append(operaciones, journal.Content[i])
    }

    return operaciones, nil
}

func RegistrarJournaling(id, operacion, contenido string) error {
    f, sb, particion, err := structs.SuperBloque_ID(id)
    if err != nil {
        return err
    }
    defer f.Close()

    if sb.S_filesystem_type != 3 {
        return nil
    }

    partStart := particion.Part_start
    journal, err := structs.LeerJournal(f, partStart)
    if err != nil {
        journal = structs.Journal{Count: 0}
    }

    if journal.Count >= int32(len(journal.Content)) {
        journal.Count = 0
    }

    idx := journal.Count
    copy(journal.Content[idx].Operation[:], operacion)
    copy(journal.Content[idx].Path[:], "")
    copy(journal.Content[idx].Content[:], contenido)
    journal.Content[idx].Date = float32(time.Now().Unix())
    journal.Count++

    return structs.EscribirJournal(f, partStart, &journal)
}

func recuperarMkgrp(f *os.File, sb *structs.SuperBloque, partStart int32, ruta, contenido string) error {
    fmt.Printf("Recuperando mkgrp: %s\n", contenido)
    
    nombreGrupo := contenido
    
    contenidoActual, err := structs.LeerArchivoDeFSFromFile(f, sb, partStart, "/home/users.txt")
    if err != nil {
        return fmt.Errorf("no se pudo leer users.txt: %v", err)
    }
    
    if grupoExisteEnRecuperacion(contenidoActual, nombreGrupo) {
        fmt.Printf("  Grupo ya existe: %s\n", nombreGrupo)
        return nil
    }
    
    nuevoID := obtenerSiguienteIDGrupoRecuperacion(contenidoActual)
    nuevaLinea := fmt.Sprintf("%d,G,%s\n", nuevoID, nombreGrupo)
    
    nuevoContenido := asegurarNuevaLineaFinalRecuperacion(contenidoActual) + nuevaLinea
    
    return actualizarUsersTxt(f, sb, partStart, nuevoContenido)
}

func recuperarRmgrp(f *os.File, sb *structs.SuperBloque, partStart int32, ruta, contenido string) error {
    fmt.Printf("Recuperando rmgrp: %s\n", contenido)
    
    nombreGrupo := contenido
    
    contenidoActual, err := structs.LeerArchivoDeFSFromFile(f, sb, partStart, "/home/users.txt")
    if err != nil {
        return fmt.Errorf("no se pudo leer users.txt: %v", err)
    }
    
    lineas := strings.Split(contenidoActual, "\n")
    encontrado := false
    
    for i, linea := range lineas {
        l := strings.TrimSpace(linea)
        if l == "" {
            continue
        }
        campos := strings.Split(l, ",")
        if len(campos) < 3 {
            continue
        }
        id := strings.TrimSpace(campos[0])
        tipo := strings.TrimSpace(campos[1])
        nombre := strings.TrimSpace(campos[2])
        
        if tipo == "G" && nombre == nombreGrupo && id != "0" {
            campos[0] = "0"
            lineas[i] = strings.Join(campos, ",")
            encontrado = true
            break
        }
    }
    
    if !encontrado {
        fmt.Printf("  Grupo no encontrado: %s\n", nombreGrupo)
        return nil
    }
    
    nuevoContenido := strings.Join(lineas, "\n")
    if !strings.HasSuffix(nuevoContenido, "\n") {
        nuevoContenido += "\n"
    }
    
    return actualizarUsersTxt(f, sb, partStart, nuevoContenido)
}

func recuperarMkusr(f *os.File, sb *structs.SuperBloque, partStart int32, ruta, contenido string) error {
    fmt.Printf("Recuperando mkusr: %s\n", contenido)
    
    partes := strings.Split(contenido, ",")
    if len(partes) < 3 {
        return fmt.Errorf("contenido inválido para mkusr: %s", contenido)
    }
    
    grupo := strings.TrimSpace(partes[0])
    usuario := strings.TrimSpace(partes[1])
    password := strings.TrimSpace(partes[2])
    
    contenidoActual, err := structs.LeerArchivoDeFSFromFile(f, sb, partStart, "/home/users.txt")
    if err != nil {
        return fmt.Errorf("no se pudo leer users.txt: %v", err)
    }
    
    if !grupoExisteActivoRecuperacion(contenidoActual, grupo) {
        return fmt.Errorf("el grupo '%s' no existe", grupo)
    }
    
    if usuarioExisteActivoRecuperacion(contenidoActual, usuario) {
        fmt.Printf("  Usuario ya existe: %s\n", usuario)
        return nil
    }
    
    nuevoID := obtenerSiguienteIDUsuarioRecuperacion(contenidoActual)
    nuevaLinea := fmt.Sprintf("%d,U,%s,%s,%s\n", nuevoID, grupo, usuario, password)
    
    nuevoContenido := asegurarNuevaLineaFinalRecuperacion(contenidoActual) + nuevaLinea
    
    return actualizarUsersTxt(f, sb, partStart, nuevoContenido)
}

func recuperarRmusr(f *os.File, sb *structs.SuperBloque, partStart int32, ruta, contenido string) error {
    fmt.Printf("Recuperando rmusr: %s\n", contenido)
    
    nombreUsuario := contenido
    
    contenidoActual, err := structs.LeerArchivoDeFSFromFile(f, sb, partStart, "/home/users.txt")
    if err != nil {
        return fmt.Errorf("no se pudo leer users.txt: %v", err)
    }
    
    lineas := strings.Split(contenidoActual, "\n")
    encontrado := false
    
    for i, linea := range lineas {
        l := strings.TrimSpace(linea)
        if l == "" {
            continue
        }
        campos := strings.Split(l, ",")
        if len(campos) < 5 {
            continue
        }
        id := strings.TrimSpace(campos[0])
        tipo := strings.TrimSpace(campos[1])
        usuario := strings.TrimSpace(campos[3])
        
        if tipo == "U" && usuario == nombreUsuario && id != "0" {
            campos[0] = "0"
            lineas[i] = strings.Join(campos, ",")
            encontrado = true
            break
        }
    }
    
    if !encontrado {
        fmt.Printf("  Usuario no encontrado: %s\n", nombreUsuario)
        return nil
    }
    
    nuevoContenido := strings.Join(lineas, "\n")
    if !strings.HasSuffix(nuevoContenido, "\n") {
        nuevoContenido += "\n"
    }
    
    return actualizarUsersTxt(f, sb, partStart, nuevoContenido)
}

func recuperarChgrp(f *os.File, sb *structs.SuperBloque, partStart int32, ruta, contenido string) error {
    fmt.Printf("Recuperando chgrp: %s\n", contenido)
    
    partes := strings.Split(contenido, "->")
    if len(partes) < 2 {
        return fmt.Errorf("contenido inválido para chgrp: %s", contenido)
    }
    
    usuario := strings.TrimSpace(partes[0])
    nuevoGrupo := strings.TrimSpace(partes[1])
    
    contenidoActual, err := structs.LeerArchivoDeFSFromFile(f, sb, partStart, "/home/users.txt")
    if err != nil {
        return fmt.Errorf("no se pudo leer users.txt: %v", err)
    }
    
    if !grupoExisteActivoRecuperacion(contenidoActual, nuevoGrupo) {
        return fmt.Errorf("el grupo '%s' no existe", nuevoGrupo)
    }
    
    lineas := strings.Split(contenidoActual, "\n")
    actualizado := false
    
    for i, linea := range lineas {
        l := strings.TrimSpace(linea)
        if l == "" {
            continue
        }
        campos := strings.Split(l, ",")
        if len(campos) < 5 {
            continue
        }
        id := strings.TrimSpace(campos[0])
        tipo := strings.TrimSpace(campos[1])
        usuarioActual := strings.TrimSpace(campos[3])
        password := strings.TrimSpace(campos[4])
        
        if tipo == "U" && usuarioActual == usuario && id != "0" {
            campos[2] = nuevoGrupo
            lineas[i] = strings.Join([]string{id, "U", nuevoGrupo, usuarioActual, password}, ",")
            actualizado = true
            break
        }
    }
    
    if !actualizado {
        return fmt.Errorf("no se pudo actualizar el usuario '%s'", usuario)
    }
    
    nuevoContenido := strings.Join(lineas, "\n")
    if !strings.HasSuffix(nuevoContenido, "\n") {
        nuevoContenido += "\n"
    }
    
    return actualizarUsersTxt(f, sb, partStart, nuevoContenido)
}

// Funciones auxiliares para la recuperación
func grupoExisteEnRecuperacion(contenido, grupo string) bool {
    lineas := strings.Split(contenido, "\n")
    for _, linea := range lineas {
        linea = strings.TrimSpace(linea)
        if linea == "" {
            continue
        }
        campos := strings.Split(linea, ",")
        if len(campos) >= 3 {
            id := strings.TrimSpace(campos[0])
            tipo := strings.TrimSpace(campos[1])
            nombre := strings.TrimSpace(campos[2])
            if tipo == "G" && nombre == grupo && id != "0" {
                return true
            }
        }
    }
    return false
}

func grupoExisteActivoRecuperacion(contenido, grupo string) bool {
    lineas := strings.Split(contenido, "\n")
    for _, l := range lineas {
        l = strings.TrimSpace(l)
        if l == "" {
            continue
        }
        p := strings.Split(l, ",")
        if len(p) < 3 {
            continue
        }
        id := strings.TrimSpace(p[0])
        tipo := strings.TrimSpace(p[1])
        nombre := strings.TrimSpace(p[2])
        if tipo == "G" && nombre == grupo && id != "0" {
            return true
        }
    }
    return false
}

func usuarioExisteActivoRecuperacion(contenido, usuario string) bool {
    lineas := strings.Split(contenido, "\n")
    for _, l := range lineas {
        l = strings.TrimSpace(l)
        if l == "" {
            continue
        }
        p := strings.Split(l, ",")
        if len(p) < 5 {
            continue
        }
        id := strings.TrimSpace(p[0])
        tipo := strings.TrimSpace(p[1])
        nombreUsuario := strings.TrimSpace(p[3])
        if tipo == "U" && nombreUsuario == usuario && id != "0" {
            return true
        }
    }
    return false
}

func obtenerSiguienteIDGrupoRecuperacion(contenido string) int {
    maxID := 0
    for _, l := range strings.Split(contenido, "\n") {
        l = strings.TrimSpace(l)
        if l == "" {
            continue
        }
        p := strings.Split(l, ",")
        if len(p) < 2 {
            continue
        }
        if strings.TrimSpace(p[1]) != "G" {
            continue
        }
        idStr := strings.TrimSpace(p[0])
        if idStr == "0" {
            continue
        }
        if id, err := strconv.Atoi(idStr); err == nil && id > maxID {
            maxID = id
        }
    }
    return maxID + 1
}

func obtenerSiguienteIDUsuarioRecuperacion(contenido string) int {
    maxID := 0
    for _, l := range strings.Split(contenido, "\n") {
        l = strings.TrimSpace(l)
        if l == "" {
            continue
        }
        p := strings.Split(l, ",")
        if len(p) < 5 {
            continue
        }
        idStr := strings.TrimSpace(p[0])
        tipo := strings.TrimSpace(p[1])
        if tipo != "U" || idStr == "0" {
            continue
        }
        if id, err := strconv.Atoi(idStr); err == nil && id > maxID {
            maxID = id
        }
    }
    return maxID + 1
}

func asegurarNuevaLineaFinalRecuperacion(s string) string {
    if s == "" {
        return ""
    }
    if strings.HasSuffix(s, "\n") {
        return s
    }
    return s + "\n"
}

func actualizarUsersTxt(f *os.File, sb *structs.SuperBloque, partStart int32, contenido string) error {
    return recuperarWrite(f, sb, partStart, "/home/users.txt", contenido)
}