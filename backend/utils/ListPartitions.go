package utils

import (
	"encoding/binary"
	"log"
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
		name := strings.TrimRight(string(part.Part_name[:]), "\x00")
		if part.Part_status != 0 && name != "" {
			ptype := "Primaria"
			if part.Part_type == 'e' || part.Part_type == 'E' {
				ptype = "Extendida"
			}
			partitions = append(partitions, PartitionInfo{
				Name:   name,
				Type:   ptype,
				Status: "Montada",
				Size:   part.Part_s,
				Start:  part.Part_start,
			})

			// Si es extendida, buscar EBRs (particiones lógicas)
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
					lname := strings.TrimRight(string(ebr.Part_name[:]), "\x00")
					if ebr.Part_mount != 0 && lname != "" {
						partitions = append(partitions, PartitionInfo{
							Name:   lname,
							Type:   "Lógica",
							Status: "Montada",
							Size:   ebr.Part_s,
							Start:  ebr.Part_start,
						})
					}
					if ebr.Part_next <= 0 {
						break
					}
					ebrStart = ebr.Part_next
				}
			}
		}
	}
	log.Println("Partitions found:", partitions)
	return partitions, nil
}