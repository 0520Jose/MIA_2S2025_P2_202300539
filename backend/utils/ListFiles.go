package utils

import (
    "backend/structs"
    "encoding/binary"
    "backend/commands"
    "fmt"
    "os"
    "strings"
)

func ListFilesFromDisk(diskPath, partitionName, ruta string) ([]structs.InfoArchivo, error) {
    f, err := os.Open(diskPath)
    if err != nil {
        return nil, fmt.Errorf("no se pudo abrir el disco: %v", err)
    }
    defer f.Close()

    var mbr structs.MBR
    if err := binary.Read(f, binary.LittleEndian, &mbr); err != nil {
        return nil, fmt.Errorf("error leyendo MBR: %v", err)
    }

    var start int32 = -1
    for _, part := range mbr.Mbr_partitions {
        name := strings.TrimRight(string(part.Part_name[:]), "\x00")
        if name == partitionName {
            start = part.Part_start
            break
        }
    }
    if start < 0 {
        return nil, fmt.Errorf("partición %s no encontrada", partitionName)
    }

    if _, err := f.Seek(int64(start), 0); err != nil {
        return nil, fmt.Errorf("error buscando superbloque: %v", err)
    }
    var sb structs.SuperBloque
    if err := binary.Read(f, binary.LittleEndian, &sb); err != nil {
        return nil, fmt.Errorf("error leyendo superbloque: %v", err)
    }

    inoIdx, err := commands.FindInodeByPath(f, &sb, ruta)
    if err != nil {
        return nil, fmt.Errorf("no se encontró la ruta '%s': %v", ruta, err)
    }

    ino, err := commands.ReadInode(f, &sb, inoIdx)
    if err != nil {
        return nil, fmt.Errorf("no se pudo leer inodo de la ruta: %v", err)
    }

    if len(ino.I_type) == 0 || ino.I_type[0] != 0 {
        return nil, fmt.Errorf("la ruta especificada no es carpeta")
    }

    var archivos []structs.InfoArchivo

    fmt.Printf("I_block del inodo: %+v\n", ino.I_block)

    for _, blk := range ino.I_block {
        if blk < 0 {
            continue
        }

        fmt.Printf("Leyendo bloque de directorio: %d\n", blk)
        bc, err := commands.ReadDirBlock(f, &sb, blk)
        if err != nil {
            fmt.Printf("Error leyendo bloque de directorio %d: %v\n", blk, err)
            continue
        }

        fmt.Printf("Contenido del bloque: %+v\n", bc.B_content)
        for _, content := range bc.B_content {
            name := strings.TrimRight(string(content.B_name[:]), "\x00")
            fmt.Printf("Entry: inodo=%d, nombre=%s\n", content.B_inodo, name)
            if content.B_inodo < 0 || name == "" || name == "." || name == ".." {
                continue
            }

            childIno, err := commands.ReadInode(f, &sb, content.B_inodo)
            if err != nil {
                fmt.Printf("Error leyendo inodo hijo %d: %v\n", content.B_inodo, err)
                continue
            }

            info := structs.InfoArchivo{
                Nombre:       name,
                Propietario:  fmt.Sprintf("%d", childIno.I_uid),
                Grupo:        fmt.Sprintf("%d", childIno.I_gid),
                Size:         childIno.I_s,
                Permisos:     fmt.Sprintf("%o", childIno.I_perm),
                Creacion:     strings.TrimRight(string(childIno.I_ctime[:]), "\x00"),
                Modificacion: strings.TrimRight(string(childIno.I_mtime[:]), "\x00"),
            }
            if len(childIno.I_type) > 0 && childIno.I_type[0] == 0 {
                info.Tipo = "d"
            } else {
                info.Tipo = "f"
            }
            archivos = append(archivos, info)
        }
    }
    fmt.Printf("Archivos en %s: %+v\n", ruta, archivos)
    return archivos, nil
}

func buscarInodoPorRutaRaw(f *os.File, sb *structs.SuperBloque, ruta string) (structs.Inodo, error) {
    if ruta == "/" {
        ino, ok := structs.ObtenerInodo(f, sb, 0)
        if !ok {
            return ino, fmt.Errorf("no se pudo obtener el inodo raíz")
        }
        return ino, nil
    }

    parts := strings.Split(strings.Trim(ruta, "/"), "/")
    fmt.Printf("Partes de la ruta: %+v\n", parts)
    currentIno, ok := structs.ObtenerInodo(f, sb, 0)
    if !ok {
        return currentIno, fmt.Errorf("no se pudo obtener el inodo raíz")
    }

    for _, part := range parts {
        if part == "" {
            continue
        }

        foundIno := int32(-1)
        for i := 0; i < structs.DIRECT_BLOCKS; i++ {
            blockIdx := currentIno.I_block[i]
            if blockIdx < 0 {
                break
            }
            bc, ok := structs.LeerBloqueCarpeta(f, sb, blockIdx)
            if ok {
                for _, content := range bc.B_content {
                    name := strings.TrimRight(string(content.B_name[:]), "\x00")
                    if name == part {
                        foundIno = content.B_inodo
                        break
                    }
                }
            }
            if foundIno >= 0 {
                break
            }
        }

        if foundIno < 0 {
            return currentIno, fmt.Errorf("ruta %s no encontrada", part)
        }

        currentIno, ok = structs.ObtenerInodo(f, sb, int(foundIno))
        if !ok {
            return currentIno, fmt.Errorf("no se pudo obtener el inodo %d", foundIno)
        }
    }

    return currentIno, nil
}