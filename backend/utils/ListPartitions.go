package utils

import (
	"encoding/binary"
	"os"
	"strings"
	"backend/structs"
)

type PartitionInfo struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status string `json:"status"`
	Size   int32  `json:"size"`
	Start  int32  `json:"start"`
	Fit    string `json:"fit"`
}

func ListPartitions(diskPath string) ([]PartitionInfo, error) {
	f, err := os.Open(diskPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var mbr structs.MBR
	if err := binary.Read(f, binary.LittleEndian, &mbr); err != nil {
		return nil, err
	}

	var partitions []PartitionInfo

	for _, part := range mbr.Mbr_partitions {
		if part.Part_status == 0 {
			continue
		}

		name := strings.TrimRight(string(part.Part_name[:]), "\x00")
		if name == "" {
			continue
		}

		ptype := "Primaria"
		if part.Part_type == 'e' || part.Part_type == 'E' {
			ptype = "Extendida"
		}

		fit := string(part.Part_fit)
		
		status := "No montada"
		if estaMontada(name, diskPath) {
			status = "Montada"
		}
		
		partitions = append(partitions, PartitionInfo{
			Name:   name,
			Type:   ptype,
			Status: status,
			Size:   part.Part_s,
			Start:  part.Part_start,
			Fit:    fit,
		})

		if part.Part_type == 'e' || part.Part_type == 'E' {
			ebrStart := part.Part_start
			for {
				var ebr structs.EBR
				if _, err := f.Seek(int64(ebrStart), 0); err != nil {
					break
				}
				if err := binary.Read(f, binary.LittleEndian, &ebr); err != nil {
					break
				}

				if ebr.Part_mount == 0 {
					if ebr.Part_next <= 0 {
						break
					}
					ebrStart = ebr.Part_next
					continue
				}

				lname := strings.TrimRight(string(ebr.Part_name[:]), "\x00")
				if lname != "" {
					lfit := string(ebr.Part_fit)
					
					lstatus := "No montada"

					name := strings.TrimRight(string(part.Part_name[:]), "\x00")
					if estaMontada(name, diskPath) {
						lstatus = "Montada"
					}

					partitions = append(partitions, PartitionInfo{
						Name:   lname,
						Type:   "Lógica",
						Status: lstatus,
						Size:   ebr.Part_s,
						Start:  ebr.Part_start,
						Fit:    lfit,
					})
				}

				if ebr.Part_next <= 0 {
					break
				}
				ebrStart = ebr.Part_next
			}
		}
	}

	return partitions, nil
}

func estaMontada(partitionName string, diskPath string) bool {
    for _, pm := range structs.Particiones_Montadas {
        pmName := strings.TrimRight(string(pm.Partition.Part_name[:]), "\x00")
        if pmName == partitionName {
            return true
        }
    }
    return false
}