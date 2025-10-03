package utils

import (
	"os"
	"path/filepath"
	"strings"
	"backend/structs"
)

type DiskInfo struct {
	Path       string   `json:"path"`
	Name       string   `json:"name"`
	Size       int32    `json:"size"`
	Fit        string   `json:"fit"`
	Partitions []string `json:"partitions"`
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

			fit := string(mbr.Dsk_fit)

			var partitions []string
			for _, p := range mbr.Mbr_partitions {
				name := strings.TrimRight(string(p.Part_name[:]), "\x00")
				if name != "" {
					partitions = append(partitions, name)
				}
			}

			disks = append(disks, DiskInfo{
				Path:       path,
				Name:       f.Name(),
				Size:       mbr.Mbr_tamano,
				Fit:        fit,
				Partitions: partitions,
			})
		}
	}
	return disks, nil
}
