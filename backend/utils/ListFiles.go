package utils

import (
    "backend/structs"
    "encoding/binary"
    "fmt"
    "io"
    "os"
    "strings"
)

type FileNode struct {
    Name        string `json:"name"`
    Type        string `json:"type"` // "file" o "folder"
    Size        int32  `json:"size"`
    Permissions string `json:"permissions"`
    Owner       string `json:"owner"`
    Group       string `json:"group"`
    Created     string `json:"created"`
    Modified    string `json:"modified"`
}

type PartitionFiles struct {
    PartitionName string     `json:"partition"`
    Files         []FileNode `json:"files"`
}

// Lista archivos/carpetas de todas las particiones (primarias y lógicas) del disco
func ListFiles(diskPath, internalPath string) ([]PartitionFiles, error) {
    mbr, err := structs.LeerMBR(diskPath)
    if err != nil {
        return nil, fmt.Errorf("no se pudo leer el MBR: %v", err)
    }

    var results []PartitionFiles

    // Recorre todas las particiones primarias y extendidas
    for _, part := range mbr.Mbr_partitions {
        name := strings.Trim(string(part.Part_name[:]), "\x00")
        if part.Part_s <= 0 || name == "" {
            continue
        }
        // Intenta listar archivos en la partición primaria (si tiene FS)
        files, err := listFilesFromPartition(diskPath, part.Part_start, name, internalPath)
        if err == nil && len(files) > 0 {
            results = append(results, PartitionFiles{
                PartitionName: name,
                Files:         files,
            })
        }
        // Si es extendida, busca lógicas
        if part.Part_type == 'e' || part.Part_type == 'E' {
            ebrPos := int64(part.Part_start)
            for {
                f, err := os.Open(diskPath)
                if err != nil {
                    break
                }
                var ebr structs.EBR
                if _, err := f.Seek(ebrPos, 0); err != nil {
                    f.Close()
                    break
                }
                if err := binary.Read(f, binary.LittleEndian, &ebr); err != nil {
                    f.Close()
                    break
                }
                f.Close()
                lname := strings.Trim(string(ebr.Part_name[:]), "\x00")
                if ebr.Part_s > 0 && lname != "" {
                    files, err := listFilesFromPartition(diskPath, ebr.Part_start, lname, internalPath)
                    if err == nil && len(files) > 0 {
                        results = append(results, PartitionFiles{
                            PartitionName: lname,
                            Files:         files,
                        })
                    }
                }
                if ebr.Part_next <= 0 {
                    break
                }
                ebrPos = int64(ebr.Part_next)
            }
        }
    }
    return results, nil
}

// Usa tus propias funciones y structs, solo cambia el punto de entrada
func listFilesFromPartition(diskPath string, partStart int32, partitionName, internalPath string) ([]FileNode, error) {
    // Lee el superbloque de la partición
    f, err := os.Open(diskPath)
    if err != nil {
        return nil, err
    }
    defer f.Close()
    sb := &structs.SuperBloque{}
    if _, err := f.Seek(int64(partStart), io.SeekStart); err != nil {
        return nil, err
    }
    if err := binary.Read(f, binary.LittleEndian, sb); err != nil {
        return nil, err
    }

    // Crea un mount temporal solo para usar ListaCarpetasFS
    tempMount := structs.PartitionMount{
        Id:        "TEMP_" + partitionName,
        Path:      diskPath,
        Partition: structs.Partition{Part_start: partStart, Part_name: [16]byte{}},
    }
    copy(tempMount.Partition.Part_name[:], []byte(partitionName))
    structs.Particiones_Montadas = append(structs.Particiones_Montadas, tempMount)
    filesInfo, err := structs.ListaCarpetasFS(tempMount.Id, internalPath)
    // Limpia el mount temporal
    structs.Particiones_Montadas = structs.Particiones_Montadas[:len(structs.Particiones_Montadas)-1]
    if err != nil {
        return nil, err
    }

    var files []FileNode
    for _, a := range filesInfo {
        tipo := "file"
        if a.Tipo == "d" {
            tipo = "folder"
        }
        files = append(files, FileNode{
            Name:        a.Nombre,
            Type:        tipo,
            Size:        a.Size,
            Permissions: a.Permisos,
            Owner:       a.Propietario,
            Group:       a.Grupo,
            Created:     a.Creacion,
            Modified:    a.Modificacion,
        })
    }
    return files, nil
}