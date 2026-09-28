package commands

import (
	"backend/structs"
	"encoding/binary"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

func Fdisk(params map[string]string) string {
	path, okPath := params["-path"]
	name, okName := params["-name"]
	sizeStr, okSize := params["-size"]

	if !okPath {
		return "Error: parámetro obligatorio faltante (-path)"
	}
	var err error
	path, err = LimpiarRuta(path)
	if err != nil {
		return fmt.Sprintf("Error en la ruta: %v", err)
	}

	unit := "K"
	if v, ok := params["-unit"]; ok {
		unit = strings.ToUpper(v)
		if unit != "B" && unit != "K" && unit != "M" {
			return "Error: unidad inválida (use B, K o M)"
		}
	}

	ptype := "P"
	if v, ok := params["-type"]; ok {
		ptype = strings.ToUpper(v)
		if ptype != "P" && ptype != "E" && ptype != "L" {
			return "Error: type inválido (use P, E o L)"
		}
	}

	fit := "WF"
	if v, ok := params["-fit"]; ok {
		fit = strings.ToUpper(v)
		if fit != "FF" && fit != "BF" && fit != "WF" {
			return "Error: fit inválido (use FF, BF o WF)"
		}
	}

	deleteType, hasDelete := params["-delete"]
	addStr, hasAdd := params["-add"]

	file, err := os.OpenFile(path, os.O_RDWR, 0666)
	if err != nil {
		return fmt.Sprintf("Error abriendo disco: %v", err)
	}
	defer file.Close()

	mbr, err := structs.LeerMBR(path)
	if err != nil {
		return fmt.Sprintf("Error leyendo MBR: %v", err)
	}

	if hasDelete {
		if !okName {
			return "Error: -name es obligatorio para -delete"
		}
		dt := strings.ToLower(deleteType)
		if dt != "fast" && dt != "full" {
			return "Error: valor inválido para -delete (usar fast o full)"
		}
		return eliminarParticion(file, &mbr, name, dt)
	}

	if hasAdd {
		if !okName {
			return "Error: -name es obligatorio para -add"
		}
		addVal, err := strconv.ParseInt(addStr, 10, 64)
		if err != nil {
			return "Error: -add debe ser un entero (positivo o negativo)"
		}
		var addBytes int64
		switch unit {
		case "B":
			addBytes = addVal
		case "K":
			addBytes = addVal * 1024
		case "M":
			addBytes = addVal * 1024 * 1024
		}
		return redimensionarParticion(file, &mbr, name, addBytes)
	}

	if !okName || !okSize {
		return "Error: parámetros obligatorios faltantes (-path, -name, -size)"
	}
	sizeVal, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil || sizeVal <= 0 {
		return "Error: -size debe ser un entero positivo"
	}
	var reqBytes int64
	switch unit {
	case "B":
		reqBytes = sizeVal
	case "K":
		reqBytes = sizeVal * 1024
	case "M":
		reqBytes = sizeVal * 1024 * 1024
	}

	if nombreDuplicado(file, &mbr, name) {
		return "Error: el nombre de partición ya existe en el disco"
	}

	if ptype == "L" {
		extIdx := indiceExtendida(&mbr)
		if extIdx == -1 {
			return "Error: no existe partición extendida para crear una lógica"
		}
		return crearLogica(file, &mbr.Mbr_partitions[extIdx], name, reqBytes, fit)
	}

	if ptype == "E" && existeExtendida(&mbr) {
		return "Error: ya existe una partición extendida en el disco"
	}

	slot := primerSlotLibre(&mbr)
	if slot == -1 {
		return "Error: límite de 4 particiones primarias/extendidas alcanzado"
	}

	start, err := ubicarPrimariaExtendida(&mbr, reqBytes, fit)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	var fitCh byte
	switch fit {
	case "FF":
		fitCh = 'F'
	case "BF":
		fitCh = 'B'
	case "WF":
		fitCh = 'W'
	}

	mbr.Mbr_partitions[slot].Part_status = 1
	mbr.Mbr_partitions[slot].Part_type = ptype[0]
	mbr.Mbr_partitions[slot].Part_fit = fitCh
	mbr.Mbr_partitions[slot].Part_start = int32(start)
	mbr.Mbr_partitions[slot].Part_s = int32(reqBytes)
	mbr.Mbr_partitions[slot].Part_correlative = -1

	for i := range mbr.Mbr_partitions[slot].Part_name {
		mbr.Mbr_partitions[slot].Part_name[i] = 0
	}
	copy(mbr.Mbr_partitions[slot].Part_name[:], name)

	if err := escribirMBR(file, &mbr); err != nil {
		return fmt.Sprintf("Error escribiendo MBR: %v", err)
	}

	if ptype == "E" {
		var ebr structs.EBR
		ebr.Part_mount = 0
		ebr.Part_fit = 0
		ebr.Part_start = 0
		ebr.Part_s = 0
		ebr.Part_next = -1
		for i := range ebr.Part_name {
			ebr.Part_name[i] = 0
		}
		if _, err := file.Seek(int64(mbr.Mbr_partitions[slot].Part_start), 0); err != nil {
			return fmt.Sprintf("Error posicionando EBR: %v", err)
		}
		if err := binary.Write(file, binary.LittleEndian, &ebr); err != nil {
			return fmt.Sprintf("Error inicializando EBR: %v", err)
		}
	}

	return fmt.Sprintf("Partición creada: %s", name)
}

func escribirMBR(file *os.File, mbr *structs.MBR) error {
	if _, err := file.Seek(0, 0); err != nil {
		return err
	}
	return binary.Write(file, binary.LittleEndian, mbr)
}

func primerSlotLibre(mbr *structs.MBR) int {
	for i := 0; i < 4; i++ {
		if mbr.Mbr_partitions[i].Part_status == 0 {
			return i
		}
	}
	return -1
}

func existeExtendida(mbr *structs.MBR) bool {
	for i := 0; i < 4; i++ {
		if mbr.Mbr_partitions[i].Part_status == 1 && mbr.Mbr_partitions[i].Part_type == 'E' {
			return true
		}
	}
	return false
}

func indiceExtendida(mbr *structs.MBR) int {
	for i := 0; i < 4; i++ {
		if mbr.Mbr_partitions[i].Part_status == 1 && mbr.Mbr_partitions[i].Part_type == 'E' {
			return i
		}
	}
	return -1
}

func nombreDuplicado(file *os.File, mbr *structs.MBR, name string) bool {
	for i := 0; i < 4; i++ {
		p := mbr.Mbr_partitions[i]
		if p.Part_status == 1 && bytesToString(p.Part_name[:]) == name {
			return true
		}
		if p.Part_status == 1 && p.Part_type == 'E' {
			if nombreDuplicadoLogicas(file, int64(p.Part_start), name) {
				return true
			}
		}
	}
	return false
}

func nombreDuplicadoLogicas(file *os.File, extStart int64, name string) bool {
	var ebr structs.EBR
	pos := extStart
	if _, err := file.Seek(pos, 0); err != nil {
		return false
	}
	if err := binary.Read(file, binary.LittleEndian, &ebr); err != nil {
		return false
	}

	if ebr.Part_mount == 0 && (ebr.Part_next == -1 || ebr.Part_next == 0) && ebr.Part_s == 0 {
		return false
	}

	for {
		if ebr.Part_mount == 1 {
			if bytesToString(ebr.Part_name[:]) == name {
				return true
			}
		}
		if ebr.Part_next == -1 || ebr.Part_next == 0 {
			break
		}
		pos = int64(ebr.Part_next)
		if _, err := file.Seek(pos, 0); err != nil {
			break
		}
		if err := binary.Read(file, binary.LittleEndian, &ebr); err != nil {
			break
		}
	}
	return false
}

func ubicarPrimariaExtendida(mbr *structs.MBR, req int64, fit string) (int64, error) {
	type seg struct{ start, size int64 }
	var ocupadas []seg
	for i := 0; i < 4; i++ {
		p := mbr.Mbr_partitions[i]
		if p.Part_status == 1 {
			ocupadas = append(ocupadas, seg{int64(p.Part_start), int64(p.Part_s)})
		}
	}
	sort.Slice(ocupadas, func(i, j int) bool { return ocupadas[i].start < ocupadas[j].start })
	header := int64(binary.Size(*mbr))

	var gaps []seg
	cur := header
	for _, s := range ocupadas {
		if s.start > cur {
			gap := seg{cur, s.start - cur}
			gaps = append(gaps, gap)
		}
		cur = s.start + s.size
	}
	if int64(mbr.Mbr_tamano) > cur {
		gap := seg{cur, int64(mbr.Mbr_tamano) - cur}
		gaps = append(gaps, gap)
	}

	if len(gaps) == 0 {
		return 0, fmt.Errorf("No hay espacio disponible")
	}

	switch fit {
	case "FF":
		for _, g := range gaps {
			if g.size >= req {
				return g.start, nil
			}
		}
	case "BF":
		bestIdx := -1
		for i, g := range gaps {
			if g.size >= req {
				if bestIdx == -1 || g.size < gaps[bestIdx].size {
					bestIdx = i
				}
			}
		}
		if bestIdx != -1 {
			return gaps[bestIdx].start, nil
		}
	case "WF":
		worstIdx := -1
		for i, g := range gaps {
			if g.size >= req {
				if worstIdx == -1 || g.size > gaps[worstIdx].size {
					worstIdx = i
				}
			}
		}
		if worstIdx != -1 {
			return gaps[worstIdx].start, nil
		}
	}
	return 0, fmt.Errorf("No hay un segmento libre con tamaño suficiente")
}

func crearLogica(file *os.File, ext *structs.Partition, name string, reqBytes int64, fit string) string {

	var ebr structs.EBR
	ebrSize := int64(binary.Size(ebr))
	extStart := int64(ext.Part_start)
	extEnd := extStart + int64(ext.Part_s)

	if _, err := file.Seek(extStart, 0); err != nil {
		return fmt.Sprintf("Error posicionando EBR: %v", err)
	}
	if err := binary.Read(file, binary.LittleEndian, &ebr); err != nil {
		return fmt.Sprintf("Error leyendo EBR: %v", err)
	}

	if ebr.Part_next == 0 {
		ebr.Part_next = -1
	}

	if ebr.Part_mount == 0 && ebr.Part_next == -1 && ebr.Part_s == 0 && ebr.Part_start == 0 {
		firstDataStart := extStart + ebrSize
		if firstDataStart+reqBytes > extEnd {
			return "Error: no hay espacio en la partición extendida"
		}

		ebr.Part_mount = 1
		ebr.Part_fit = fit[0]
		ebr.Part_start = int32(firstDataStart)
		ebr.Part_s = int32(reqBytes)
		ebr.Part_next = -1

		for i := range ebr.Part_name {
			ebr.Part_name[i] = 0
		}
		copy(ebr.Part_name[:], name)

		if _, err := file.Seek(extStart, 0); err != nil {
			return fmt.Sprintf("Error posicionando EBR: %v", err)
		}
		if err := binary.Write(file, binary.LittleEndian, &ebr); err != nil {
			return fmt.Sprintf("Error escribiendo EBR: %v", err)
		}
		return fmt.Sprintf("Partición lógica creada: %s", name)
	}

	pos := extStart
	ebrCount := 0

	for {
		ebrCount++

		if ebr.Part_next == -1 || ebr.Part_next == 0 {

			currentDataEnd := int64(ebr.Part_start) + int64(ebr.Part_s)
			newEBRPos := currentDataEnd

			if newEBRPos < extStart {
				return "Error: cálculo de posición de EBR inválido (underflow)"
			}
			if newEBRPos+ebrSize+reqBytes > extEnd {
				return "Error: no hay espacio en la partición extendida"
			}

			ebr.Part_next = int32(newEBRPos)
			if _, err := file.Seek(pos, 0); err != nil {
				return fmt.Sprintf("Error posicionando EBR previo: %v", err)
			}
			if err := binary.Write(file, binary.LittleEndian, &ebr); err != nil {
				return fmt.Sprintf("Error actualizando EBR previo: %v", err)
			}

			var newEBR structs.EBR
			newEBR.Part_mount = 1
			newEBR.Part_fit = fit[0]
			newEBR.Part_start = int32(newEBRPos + ebrSize)
			newEBR.Part_s = int32(reqBytes)
			newEBR.Part_next = -1
			for i := range newEBR.Part_name {
				newEBR.Part_name[i] = 0
			}
			copy(newEBR.Part_name[:], name)

			if _, err := file.Seek(newEBRPos, 0); err != nil {
				return fmt.Sprintf("Error posicionando nuevo EBR: %v", err)
			}
			if err := binary.Write(file, binary.LittleEndian, &newEBR); err != nil {
				return fmt.Sprintf("Error escribiendo nuevo EBR: %v", err)
			}
			return fmt.Sprintf("Partición lógica creada: %s", name)
		}

		pos = int64(ebr.Part_next)
		if _, err := file.Seek(pos, 0); err != nil {
			return fmt.Sprintf("Error recorriendo EBR: %v", err)
		}
		if err := binary.Read(file, binary.LittleEndian, &ebr); err != nil {
			return fmt.Sprintf("Error leyendo EBR: %v", err)
		}
		if ebr.Part_next == 0 {
			ebr.Part_next = -1
		}
	}
}

func bytesToString(b []byte) string {
	n := 0
	for i := range b {
		if b[i] == 0 {
			break
		}
		n++
	}
	return string(b[:n])
}

func eliminarParticion(file *os.File, mbr *structs.MBR, name string, deleteType string) string {
	for i := 0; i < 4; i++ {
		p := mbr.Mbr_partitions[i]
		if p.Part_status == 1 && bytesToString(p.Part_name[:]) == name {
			if deleteType == "fast" {
				mbr.Mbr_partitions[i].Part_status = 0
				if err := escribirMBR(file, mbr); err != nil {
					return fmt.Sprintf("Error escribiendo MBR: %v", err)
				}
				if p.Part_type == 'E' {
					if err := desactivarLogicasFast(file, int64(p.Part_start)); err != nil {
						return fmt.Sprintf("Partición extendida eliminada (fast), pero ocurrió un error limpiando lógicas: %v", err)
					}
				}
				return fmt.Sprintf("Partición %s eliminada (fast)", name)
			} else {
				start := int64(p.Part_start)
				size := int64(p.Part_s)
				mbr.Mbr_partitions[i].Part_status = 0
				for j := range mbr.Mbr_partitions[i].Part_name {
					mbr.Mbr_partitions[i].Part_name[j] = 0
				}
				if err := escribirMBR(file, mbr); err != nil {
					return fmt.Sprintf("Error escribiendo MBR: %v", err)
				}
				if err := sobrescribirCeros(file, start, size); err != nil {
					return fmt.Sprintf("Partición eliminada (full) pero error al limpiar: %v", err)
				}
				return fmt.Sprintf("Partición %s eliminada (full) y espacio rellenado con zeros", name)
			}
		}
		if p.Part_status == 1 && p.Part_type == 'E' {
			found, res := buscarLogicaPorNombre(file, int64(p.Part_start), name)
			if found {
				ebrPos := res.pos
				ebr := res.ebr
				if deleteType == "fast" {
					ebr.Part_mount = 0
					for j := range ebr.Part_name {
						ebr.Part_name[j] = 0
					}
					if _, err := file.Seek(ebrPos, 0); err != nil {
						return fmt.Sprintf("Error actualizando EBR: %v", err)
					}
					if err := binary.Write(file, binary.LittleEndian, &ebr); err != nil {
						return fmt.Sprintf("Error escribiendo EBR: %v", err)
					}
					return fmt.Sprintf("Partición lógica %s eliminada (fast)", name)
				} else {
					start := int64(ebr.Part_start)
					size := int64(ebr.Part_s)
					ebr.Part_mount = 0
					for j := range ebr.Part_name {
						ebr.Part_name[j] = 0
					}
					if _, err := file.Seek(ebrPos, 0); err != nil {
						return fmt.Sprintf("Error actualizando EBR: %v", err)
					}
					if err := binary.Write(file, binary.LittleEndian, &ebr); err != nil {
						return fmt.Sprintf("Error escribiendo EBR: %v", err)
					}
					if err := sobrescribirCeros(file, start, size); err != nil {
						return fmt.Sprintf("Partición lógica eliminada (full) pero error limpiando espacio: %v", err)
					}
					return fmt.Sprintf("Partición lógica %s eliminada (full) y espacio rellenado con zeros", name)
				}
			}
		}
	}
	return "Error: partición con ese nombre no encontrada"
}

func desactivarLogicasFast(file *os.File, extStart int64) error {
	var ebr structs.EBR
	pos := extStart
	if _, err := file.Seek(pos, 0); err != nil {
		return err
	}
	if err := binary.Read(file, binary.LittleEndian, &ebr); err != nil {
		return err
	}
	if ebr.Part_mount == 0 && (ebr.Part_next == -1 || ebr.Part_next == 0) && ebr.Part_s == 0 {
		return nil
	}
	for {
		if ebr.Part_mount == 1 {
			ebr.Part_mount = 0
			for j := range ebr.Part_name {
				ebr.Part_name[j] = 0
			}
			if _, err := file.Seek(pos, 0); err != nil {
				return err
			}
			if err := binary.Write(file, binary.LittleEndian, &ebr); err != nil {
				return err
			}
		}
		if ebr.Part_next == -1 || ebr.Part_next == 0 {
			break
		}
		pos = int64(ebr.Part_next)
		if _, err := file.Seek(pos, 0); err != nil {
			break
		}
		if err := binary.Read(file, binary.LittleEndian, &ebr); err != nil {
			break
		}
	}
	return nil
}

func sobrescribirCeros(file *os.File, start int64, size int64) error {
	if size <= 0 {
		return nil
	}
	if _, err := file.Seek(start, 0); err != nil {
		return err
	}
	chunk := int64(1024 * 1024)
	zeroBuf := make([]byte, chunk)
	toWrite := size
	for toWrite > 0 {
		if toWrite < chunk {
			zeroBuf = make([]byte, toWrite)
		}
		if _, err := file.Write(zeroBuf); err != nil {
			return err
		}
		toWrite -= int64(len(zeroBuf))
	}
	return nil
}

type ebrSearchResult struct {
	pos int64
	ebr structs.EBR
}

func buscarLogicaPorNombre(file *os.File, extStart int64, name string) (bool, ebrSearchResult) {
	var ebr structs.EBR
	pos := extStart
	if _, err := file.Seek(pos, 0); err != nil {
		return false, ebrSearchResult{}
	}
	if err := binary.Read(file, binary.LittleEndian, &ebr); err != nil {
		return false, ebrSearchResult{}
	}
	if ebr.Part_next == 0 {
		ebr.Part_next = -1
	}
	if ebr.Part_mount == 0 && (ebr.Part_next == -1 || ebr.Part_next == 0) && ebr.Part_s == 0 {
		return false, ebrSearchResult{}
	}
	for {
		if ebr.Part_mount == 1 && bytesToString(ebr.Part_name[:]) == name {
			return true, ebrSearchResult{pos: pos, ebr: ebr}
		}
		if ebr.Part_next == -1 || ebr.Part_next == 0 {
			break
		}
		pos = int64(ebr.Part_next)
		if _, err := file.Seek(pos, 0); err != nil {
			break
		}
		if err := binary.Read(file, binary.LittleEndian, &ebr); err != nil {
			break
		}
	}
	return false, ebrSearchResult{}
}

func redimensionarParticion(file *os.File, mbr *structs.MBR, name string, addBytes int64) string {
	for i := 0; i < 4; i++ {
		p := mbr.Mbr_partitions[i]
		if p.Part_status == 1 && bytesToString(p.Part_name[:]) == name {
			origSize := int64(p.Part_s)
			newSize := origSize + addBytes
			if newSize <= 0 {
				return "Error: la nueva partición debe ser mayor a 0 bytes"
			}
			if addBytes > 0 {
				curEnd := int64(p.Part_start) + origSize
				nextStart := int64(mbr.Mbr_tamano)
				for j := 0; j < 4; j++ {
					if j == i {
						continue
					}
					other := mbr.Mbr_partitions[j]
					if other.Part_status == 1 {
						s := int64(other.Part_start)
						if s > int64(p.Part_start) && s < nextStart {
							nextStart = s
						}
					}
				}
				available := nextStart - curEnd
				if available < addBytes {
					return "Error: no hay espacio contiguo suficiente después de la partición para expandirla"
				}
				mbr.Mbr_partitions[i].Part_s = int32(newSize)
				if err := escribirMBR(file, mbr); err != nil {
					return fmt.Sprintf("Error escribiendo MBR: %v", err)
				}
				return fmt.Sprintf("Partición %s agrandada en %d bytes", name, addBytes)
			} else {
				mbr.Mbr_partitions[i].Part_s = int32(newSize)
				if err := escribirMBR(file, mbr); err != nil {
					return fmt.Sprintf("Error escribiendo MBR: %v", err)
				}
				return fmt.Sprintf("Partición %s reducida en %d bytes", name, -addBytes)
			}
		}
		if p.Part_status == 1 && p.Part_type == 'E' {
			found, res := buscarLogicaPorNombre(file, int64(p.Part_start), name)
			if found {
				ebrPos := res.pos
				ebr := res.ebr
				origSize := int64(ebr.Part_s)
				newSize := origSize + addBytes
				if newSize <= 0 {
					return "Error: la nueva partición lógica debe ser mayor a 0 bytes"
				}
				extStart := int64(p.Part_start)
				extEnd := extStart + int64(p.Part_s)
				curEnd := int64(ebr.Part_start) + origSize
				var nextEbrPos int64 = extEnd
				var tmp structs.EBR
				pos := extStart
				if _, err := file.Seek(pos, 0); err == nil {
					if err := binary.Read(file, binary.LittleEndian, &tmp); err == nil {
						if tmp.Part_next == 0 {
							tmp.Part_next = -1
						}
						for {
							if tmp.Part_mount == 1 && int64(tmp.Part_start) > int64(ebr.Part_start) {
								if int64(tmp.Part_start) < nextEbrPos {
									nextEbrPos = int64(pos)
								}
							}
							if tmp.Part_next == -1 || tmp.Part_next == 0 {
								break
							}
							pos = int64(tmp.Part_next)
							if _, err := file.Seek(pos, 0); err != nil {
								break
							}
							if err := binary.Read(file, binary.LittleEndian, &tmp); err != nil {
								break
							}
						}
					}
				}
				var nextLogicStart int64 = extEnd
				pos = extStart
				if _, err := file.Seek(pos, 0); err == nil {
					if err := binary.Read(file, binary.LittleEndian, &tmp); err == nil {
						if tmp.Part_next == 0 {
							tmp.Part_next = -1
						}
						for {
							if tmp.Part_mount == 1 && int64(tmp.Part_start) > int64(ebr.Part_start) {
								if int64(tmp.Part_start) < nextLogicStart {
									nextLogicStart = int64(tmp.Part_start)
								}
							}
							if tmp.Part_next == -1 || tmp.Part_next == 0 {
								break
							}
							pos = int64(tmp.Part_next)
							if _, err := file.Seek(pos, 0); err != nil {
								break
							}
							if err := binary.Read(file, binary.LittleEndian, &tmp); err != nil {
								break
							}
						}
					}
				}
				if addBytes > 0 {
					available := nextLogicStart - curEnd
					if available < addBytes {
						return "Error: no hay espacio contiguo suficiente dentro de la partición extendida para expandir la lógica"
					}
					ebr.Part_s = int32(newSize)
					if _, err := file.Seek(ebrPos, 0); err != nil {
						return fmt.Sprintf("Error posicionando EBR para escritura: %v", err)
					}
					if err := binary.Write(file, binary.LittleEndian, &ebr); err != nil {
						return fmt.Sprintf("Error escribiendo EBR: %v", err)
					}
					return fmt.Sprintf("Partición lógica %s agrandada en %d bytes", name, addBytes)
				} else {
					ebr.Part_s = int32(newSize)
					if _, err := file.Seek(ebrPos, 0); err != nil {
						return fmt.Sprintf("Error posicionando EBR para escritura: %v", err)
					}
					if err := binary.Write(file, binary.LittleEndian, &ebr); err != nil {
						return fmt.Sprintf("Error escribiendo EBR: %v", err)
					}
					return fmt.Sprintf("Partición lógica %s reducida en %d bytes", name, -addBytes)
				}
			}
		}
	}
	return "Error: partición con ese nombre no encontrada para redimensionar"
}

func timeNowString() string {
	return time.Now().Format("2006-01-02 15:04:05")
}
