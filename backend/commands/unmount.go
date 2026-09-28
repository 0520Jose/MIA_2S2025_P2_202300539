package commands

import (
	"backend/structs"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
)

func Unmount(params map[string]string) string {
	id, ok := params["-id"]
	if !ok || strings.TrimSpace(id) == "" {
		return "Error: parámetro -id es obligatorio\n"
	}
	id = strings.TrimSpace(id)

	var pm *structs.PartitionMount
	var idx int = -1
	for i := range particionesMontadas {
		if particionesMontadas[i].Id == id {
			pm = &particionesMontadas[i]
			idx = i
			break
		}
	}
	if pm == nil {
		return fmt.Sprintf("Error: no existe una partición montada con id '%s'\n", id)
	}

	file, err := os.OpenFile(pm.Path, os.O_RDWR, 0666)
	if err != nil {
		return fmt.Sprintf("Error al abrir disco: %v\n", err)
	}
	defer file.Close()

	mbr, err := structs.LeerMBR(pm.Path)
	if err != nil {
		return fmt.Sprintf("Error leyendo MBR: %v\n", err)
	}

	found := false
	for i := 0; i < 4; i++ {
		if mbr.Mbr_partitions[i].Part_start == pm.Partition.Part_start &&
			mbr.Mbr_partitions[i].Part_s == pm.Partition.Part_s {
			mbr.Mbr_partitions[i].Part_correlative = 0
			found = true
			break
		}
	}

	if !found && pm.Partition.Part_type == 'L' {
		extIdx := indiceExtendida(&mbr)
		if extIdx != -1 {
			var ebr structs.EBR
			pos := int64(mbr.Mbr_partitions[extIdx].Part_start)
			for {
				if _, err := file.Seek(pos, 0); err != nil {
					break
				}
				if err := binary.Read(file, binary.LittleEndian, &ebr); err != nil {
					break
				}
				if ebr.Part_start == pm.Partition.Part_start &&
					ebr.Part_s == pm.Partition.Part_s {
					ebr.Part_correlative = 0
					if _, err := file.Seek(pos, 0); err == nil {
						_ = binary.Write(file, binary.LittleEndian, &ebr)
					}
					found = true
					break
				}
				if ebr.Part_next == -1 || ebr.Part_next == 0 {
					break
				}
				pos = int64(ebr.Part_next)
			}
		}
	}

	if found {
		if err := escribirMBR(file, &mbr); err != nil {
			return fmt.Sprintf("Error escribiendo MBR: %v\n", err)
		}
		particionesMontadas = append(particionesMontadas[:idx], particionesMontadas[idx+1:]...)
		structs.Particiones_Montadas = particionesMontadas
		return fmt.Sprintf("Partición con id '%s' desmontada correctamente\n", id)
	}

	return fmt.Sprintf("Error: partición con id '%s' no encontrada en el disco\n", id)
}
