package commands

import (
	"backend/structs"
	"fmt"
	"os"
	"strings"
)

// Copy realiza la copia de archivos o carpetas dentro de ExtreamFS
func Copy(params map[string]string) string {
	if usuarioActual == nil {
		return "Error: No hay una sesión activa."
	}

	rawPath, ok := params["-path"]
	if !ok || strings.TrimSpace(rawPath) == "" {
		return "Error: parámetro -path es obligatorio."
	}

	rawDestino, ok := params["-destino"]
	if !ok || strings.TrimSpace(rawDestino) == "" {
		return "Error: parámetro -destino es obligatorio."
	}

	rutaOrigen := unquoteValue(strings.TrimSpace(rawPath))
	rutaDestino := unquoteValue(strings.TrimSpace(rawDestino))

	if !strings.HasPrefix(rutaOrigen, "/") {
		return "Error: -path debe ser ruta absoluta."
	}
	if !strings.HasPrefix(rutaDestino, "/") {
		return "Error: -destino debe ser ruta absoluta."
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

	inodoOrigen, inodoOrigenIdx, err := structs.BuscarInodoPorRuta_(disk, sb, rutaOrigen)
	if err != nil {
		return fmt.Sprintf("Error: ruta de origen no existe: %v", err)
	}
	if !Permisos(&inodoOrigen, permRead) {
		return "Error: no tiene permisos de lectura sobre el recurso origen."
	}

	inodoDestino, inodoDestinoIdx, err := structs.BuscarInodoPorRuta_(disk, sb, rutaDestino)
	if err != nil {
		return fmt.Sprintf("Error: ruta de destino no existe: %v", err)
	}
	if !structs.EsCarpeta(inodoDestino) {
		return "Error: el destino debe ser una carpeta."
	}
	if !Permisos(&inodoDestino, permWrite) {
		return "Error: no tiene permisos de escritura sobre la carpeta destino."
	}

	nombreOrigen := obtenerNombreDeRuta(rutaOrigen)
	if existeNombreEnDirectorio(disk, sb, &inodoDestino, nombreOrigen, -1) {
		return fmt.Sprintf("Error: ya existe un recurso con el nombre '%s' en el destino.", nombreOrigen)
	}

	var skipped []string
	nuevoIdx, err := copiarRecursivo(disk, sb, inodoOrigenIdx, inodoDestinoIdx, nombreOrigen, &skipped)
	if err != nil {
		return fmt.Sprintf("Error al copiar: %v", err)
	}

	if err := writeSuperBlock(disk, sb, int64(pm.Partition.Part_start)); err != nil {
		return "Error al actualizar superbloque: " + err.Error()
	}

	if sb.S_filesystem_type == 3 {
		if err := RegistrarOperacionJournal(disk, sb, "copy", rutaOrigen, rutaDestino); err != nil {
			fmt.Printf("Advertencia: no se pudo registrar en journal: %v\n", err)
		}
	}

	resultado := fmt.Sprintf("Copia realizada correctamente. Nuevo recurso creado con índice %d.", nuevoIdx)
	if len(skipped) > 0 {
		resultado += fmt.Sprintf("\n\nArchivos omitidos por falta de permisos:\n- %s", strings.Join(skipped, "\n- "))
	}

	return resultado
}

func copiarRecursivo(f *os.File, sb *structs.SuperBloque, origenIdx, padreDestinoIdx int32, nombre string, skipped *[]string) (int32, error) {
	inodoOrigen, err := ReadInode(f, sb, origenIdx)
	if err != nil {
		return -1, err
	}
	if !Permisos(&inodoOrigen, permRead) {
		*skipped = append(*skipped, nombre)
		return -1, fmt.Errorf("sin permisos de lectura")
	}

	nuevoIdx, err := allocInode(f, sb)
	if err != nil {
		return -1, err
	}

	nuevoInodo := inodoOrigen
	nuevoInodo.I_uid = int32(usuarioActual.UID)
	nuevoInodo.I_gid = int32(usuarioActual.GID)

	t := fecha17()
	copy(nuevoInodo.I_ctime[:], t)
	copy(nuevoInodo.I_mtime[:], t)

	// Inicializar bloques
	for i := range nuevoInodo.I_block {
		nuevoInodo.I_block[i] = -1
	}

	if structs.EsCarpeta(inodoOrigen) {
		nuevoBlkIdx, err := allocBlock(f, sb)
		if err != nil {
			freeInode(f, sb, nuevoIdx)
			return -1, err
		}

		var dirBlock structs.BCarpeta
		copy(dirBlock.B_content[0].B_name[:], ".")
		dirBlock.B_content[0].B_inodo = nuevoIdx
		copy(dirBlock.B_content[1].B_name[:], "..")
		dirBlock.B_content[1].B_inodo = padreDestinoIdx
		dirBlock.B_content[2].B_inodo = -1
		dirBlock.B_content[3].B_inodo = -1

		nuevoInodo.I_block[0] = nuevoBlkIdx
		nuevoInodo.I_s = 0

		if err := writeDirBlock(f, sb, nuevoBlkIdx, &dirBlock); err != nil {
			freeBlock(f, sb, nuevoBlkIdx)
			freeInode(f, sb, nuevoIdx)
			return -1, err
		}

		// Copiar contenido recursivamente
		for _, blk := range inodoOrigen.I_block {
			if blk == -1 {
				break
			}
			dirOrigen, err := ReadDirBlock(f, sb, blk)
			if err != nil {
				continue
			}
			for _, entry := range dirOrigen.B_content {
				if entry.B_inodo == -1 {
					continue
				}
				nombreEntry := strings.TrimRight(string(entry.B_name[:]), "\x00")
				if nombreEntry == "." || nombreEntry == ".." {
					continue
				}
				copiarRecursivo(f, sb, entry.B_inodo, nuevoIdx, nombreEntry, skipped)
			}
		}

	} else {
		data := []byte{}
		// Bloques directos
		for i := 0; i < structs.DIRECT_BLOCKS; i++ {
			if inodoOrigen.I_block[i] == -1 {
				break
			}
			bloqueOrigen, ok := structs.LeerBloqueArchivo(f, sb, inodoOrigen.I_block[i])
			if !ok {
				continue
			}
			nuevoBlkIdx, err := allocBlock(f, sb)
			if err != nil {
				for j := 0; j < i; j++ {
					if nuevoInodo.I_block[j] != -1 {
						freeBlock(f, sb, nuevoInodo.I_block[j])
					}
				}
				freeInode(f, sb, nuevoIdx)
				return -1, err
			}
			// Convertir slice a [64]byte
			var arr [64]byte
			copy(arr[:], bloqueOrigen.B_content[:])
			bloque := structs.BArchivo{B_content: arr}

			if err := structs.EscribirBloqueArchivo(f, sb, nuevoBlkIdx, &bloque); err != nil {
				freeBlock(f, sb, nuevoBlkIdx)
				continue
			}
			nuevoInodo.I_block[i] = nuevoBlkIdx
			data = append(data, bloque.B_content[:]...)
		}
		nuevoInodo.I_s = int32(len(data))
	}

	// Escribir inodo
	if err := writeInode(f, sb, nuevoIdx, &nuevoInodo); err != nil {
		for _, blk := range nuevoInodo.I_block {
			if blk != -1 {
				freeBlock(f, sb, blk)
			}
		}
		freeInode(f, sb, nuevoIdx)
		return -1, err
	}

	// Agregar entrada en directorio padre
	if err := agregarEntradaEnDirectorio(f, sb, padreDestinoIdx, nuevoIdx, nombre); err != nil {
		eliminarInodoRecursivo(f, sb, nuevoIdx)
		return -1, err
	}

	return nuevoIdx, nil
}

func agregarEntradaEnDirectorio(f *os.File, sb *structs.SuperBloque, padreIdx, childIdx int32, nombre string) error {
	inodoPadre, err := ReadInode(f, sb, padreIdx)
	if err != nil {
		return err
	}

	if len(nombre) > 12 {
		return fmt.Errorf("nombre demasiado largo (máximo 12 caracteres)")
	}

	// Revisar bloques existentes
	for i, blk := range inodoPadre.I_block {
		if blk == -1 {
			// Bloque vacío: asignar uno nuevo
			nuevoBlk, err := allocBlock(f, sb)
			if err != nil {
				return fmt.Errorf("no se pudo asignar bloque para directorio: %v", err)
			}

			var dir structs.BCarpeta
			// Inicializar entradas a -1
			for j := range dir.B_content {
				dir.B_content[j].B_inodo = -1
			}
			inodoPadre.I_block[i] = nuevoBlk

			if i == 0 {
				// Primer bloque de directorio: colocar "." y ".."
				copy(dir.B_content[0].B_name[:], ".")
				dir.B_content[0].B_inodo = padreIdx
				copy(dir.B_content[1].B_name[:], "..")
				dir.B_content[1].B_inodo = padreIdx
			}

			if err := writeDirBlock(f, sb, nuevoBlk, &dir); err != nil {
				freeBlock(f, sb, nuevoBlk)
				return err
			}

			// Actualizar inodo
			if err := writeInode(f, sb, padreIdx, &inodoPadre); err != nil {
				return err
			}

			blk = nuevoBlk // Continuar con el bloque recién creado
		}

		dir, err := ReadDirBlock(f, sb, blk)
		if err != nil {
			continue
		}

		// Buscar espacio libre en este bloque
		for i := range dir.B_content {
			if dir.B_content[i].B_inodo == -1 {
				copy(dir.B_content[i].B_name[:], nombre)
				dir.B_content[i].B_inodo = childIdx
				return writeDirBlock(f, sb, blk, &dir)
			}
		}
	}

	return fmt.Errorf("no hay espacio en el directorio padre ni se pudo asignar bloque")
}


func obtenerNombreDeRuta(ruta string) string {
	parts := strings.Split(strings.Trim(ruta, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}
