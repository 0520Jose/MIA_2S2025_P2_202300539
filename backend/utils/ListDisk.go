package utils

import (
    "os"
    "path/filepath"
    "strings"
    "backend/structs"
)

type DiskInfo struct {
    Path              string   `json:"path"`
    Name              string   `json:"name"`
    Size              int32    `json:"size"`    // Capacidad en bytes
    Fit               string   `json:"fit"`     // Fit real del disco
    MountedPartitions []string `json:"mountedPartitions"`
}

func ListDisks(dir string) ([]DiskInfo, error) {
    files, err := os.ReadDir(dir)
    if err != nil {
        return nil, err
    }
    var disks []DiskInfo
    for _, f := range files {
        if !f.IsDir() && strings.HasSuffix(f.Name(), ".mia") {
            path := filepath.Join(dir, f.Name())
            mbr, err := structs.LeerMBR(path)
            if err != nil {
                continue
            }
            // Fit como string
            fit := string(mbr.Dsk_fit)
            // Particiones montadas (nombre de las particiones activas)
            var mounted []string
            for _, p := range mbr.Mbr_partitions {
                if p.Part_status == 1 { // 1 = activa, ajusta si tu sistema usa otro valor
                    pname := strings.TrimRight(string(p.Part_name[:]), "\x00")
                    if pname != "" {
                        mounted = append(mounted, pname)
                    }
                }
            }
            disks = append(disks, DiskInfo{
                Path:              path,
                Name:              f.Name(),
                Size:              mbr.Mbr_tamano,
                Fit:               fit,
                MountedPartitions: mounted,
            })
        }
    }
    return disks, nil
}