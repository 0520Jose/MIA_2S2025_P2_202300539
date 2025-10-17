package commands

import (
    "backend/structs"
    "fmt"
    "io"
    "os"
    "strings"
    "time"
    "encoding/binary"
	"path/filepath"
)

func RecuperarEXT3(f *os.File, partStart int32, sb *structs.SuperBloque) error {
    if !sb.EsEXT3() {
        return fmt.Errorf("no es un sistema EXT3")
    }

    operaciones, err := ObtenerOperacionesJournal(f, partStart)
    if err != nil {
        return err
    }

    for _, op := range operaciones {
        operacion := strings.Trim(string(op.Operation[:]), "\x00")
        ruta := strings.Trim(string(op.Path[:]), "\x00")
        contenido := strings.Trim(string(op.Content[:]), "\x00")

        fmt.Printf("Recuperando operación: %s en %s\n", operacion, ruta)
        switch operacion {
        case "MKFS":
            fmt.Println("Operación MKFS detectada (solo informativo, se ignora en recuperación)")
        case "mkdir":
            if err := CrearDirectorioFSConPartStart(f, sb, partStart, ruta); err != nil {
                fmt.Printf("Error al recuperar mkdir %s: %v\n", ruta, err)
            }
        case "mkfile":
            if err := EscribirArchivoFSConPartStart(f, sb, partStart, ruta, contenido); err != nil {
                fmt.Printf("Error al recuperar mkfile %s: %v\n", ruta, err)
            }
        case "write", "create", "modify":
            if err := EscribirArchivoFSConPartStart(f, sb, partStart, ruta, contenido); err != nil {
                fmt.Printf("Error al recuperar %s %s: %v\n", operacion, ruta, err)
            }
        case "delete":
            fmt.Printf("Recuperación de eliminación no implementada para: %s\n", ruta)
        default:
            fmt.Printf("Operación no soportada en recuperación: %s\n", operacion)
        }
    }

    return nil
}

func VerificarYCrearRaiz(f *os.File, sb *structs.SuperBloque, partStart int32) error {
	bmInodos := make([]byte, sb.S_inodes_count)
	if _, err := f.Seek(int64(sb.S_bm_inode_start), 0); err != nil {
		return fmt.Errorf("error al leer bitmap inodos: %v", err)
	}
	if _, err := f.Read(bmInodos); err != nil {
		return fmt.Errorf("error al leer bitmap inodos: %v", err)
	}

	if bmInodos[0] == 1 {
		fmt.Println("  Inodo raíz ya existe")
		return nil
	}

	fmt.Println("  Creando inodo raíz (0)...")

	bmBloques := make([]byte, sb.S_blocks_count)
	if _, err := f.Seek(int64(sb.S_bm_block_start), 0); err != nil {
		return fmt.Errorf("error al leer bitmap bloques: %v", err)
	}
	if _, err := f.Read(bmBloques); err != nil {
		return fmt.Errorf("error al leer bitmap bloques: %v", err)
	}

	bloqueRaiz := int32(0)
	if bmBloques[0] == 1 {
		for i := int32(0); i < sb.S_blocks_count; i++ {
			if bmBloques[i] == 0 {
				bloqueRaiz = i
				break
			}
		}
	}

	fmt.Printf("  Usando bloque %d para directorio raíz\n", bloqueRaiz)

	var inoRaiz structs.Inodo
	inoRaiz.I_uid = 1
	inoRaiz.I_gid = 1
	inoRaiz.I_s = 0
	ahora := time.Now().Format("2006-01-02 15:04")
	copy(inoRaiz.I_atime[:], ahora)
	copy(inoRaiz.I_ctime[:], ahora)
	copy(inoRaiz.I_mtime[:], ahora)
	for i := range inoRaiz.I_block {
		inoRaiz.I_block[i] = -1
	}
	inoRaiz.I_type[0] = 0 
	copy(inoRaiz.I_perm[:], "664")
	inoRaiz.I_block[0] = bloqueRaiz

	fmt.Printf("  Asignando bloque %d al inodo raíz\n", bloqueRaiz)

	var bcRaiz structs.BCarpeta
	bcRaiz.B_content[0].B_inodo = 0
	copy(bcRaiz.B_content[0].B_name[:], ".")
	bcRaiz.B_content[1].B_inodo = 0
	copy(bcRaiz.B_content[1].B_name[:], "..")
	for i := 2; i < 4; i++ {
		bcRaiz.B_content[i].B_inodo = -1
	}

	offsetBloque := int64(sb.S_block_start) + int64(bloqueRaiz)*64
	if _, err := f.Seek(offsetBloque, 0); err != nil {
		return fmt.Errorf("error al posicionar bloque raíz: %v", err)
	}
	if err := binary.Write(f, binary.LittleEndian, &bcRaiz); err != nil {
		return fmt.Errorf("error al escribir bloque raíz: %v", err)
	}

	offsetBitmapBloque := int64(sb.S_bm_block_start) + int64(bloqueRaiz)
	if _, err := f.Seek(offsetBitmapBloque, 0); err != nil {
		return err
	}
	if _, err := f.Write([]byte{1}); err != nil {
		return err
	}

	offsetInodo := int64(sb.S_inode_start)
	if _, err := f.Seek(offsetInodo, 0); err != nil {
		return fmt.Errorf("error al posicionar inodo raíz: %v", err)
	}
	if err := binary.Write(f, binary.LittleEndian, &inoRaiz); err != nil {
		return fmt.Errorf("error al escribir inodo raíz: %v", err)
	}

	offsetBitmapInodo := int64(sb.S_bm_inode_start)
	if _, err := f.Seek(offsetBitmapInodo, 0); err != nil {
		return err
	}
	if _, err := f.Write([]byte{1}); err != nil {
		return err
	}

	if sb.S_first_ino == 0 {
		sb.S_first_ino = 1
	}
	if sb.S_free_inodes_count == sb.S_inodes_count {
		sb.S_free_inodes_count--
	}
	if sb.S_free_blocks_count == sb.S_blocks_count {
		sb.S_free_blocks_count--
	}

	if _, err := f.Seek(int64(partStart), 0); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, sb); err != nil {
		return fmt.Errorf("error al actualizar superbloque: %v", err)
	}

	if err := f.Sync(); err != nil {
		fmt.Printf("Advertencia: no se pudo sincronizar: %v\n", err)
	}

	fmt.Println("  Inodo raíz creado exitosamente")
	
	if _, err := f.Seek(int64(partStart), 0); err != nil {
		return err
	}
	if err := binary.Read(f, binary.LittleEndian, sb); err != nil {
		return fmt.Errorf("error al releer superbloque: %v", err)
	}
	
	_, ok := structs.ObtenerInodo(f, sb, 0)
	if !ok {
		return fmt.Errorf("no se pudo verificar el inodo raíz después de crearlo")
	}
	fmt.Println("  Verificación: inodo raíz es legible")
	
	return nil
}

func CrearDirectorioFSConPartStart(f *os.File, sb *structs.SuperBloque, partStart int32, ruta string) error {
	ruta = strings.TrimSpace(ruta)
	if ruta == "" || ruta == "/" {
		return nil
	}

	if err := VerificarYCrearRaiz(f, sb, partStart); err != nil {
		return fmt.Errorf("error al verificar raíz: %v", err)
	}

	partes := strings.Split(strings.Trim(ruta, "/"), "/")
	rutaActual := "/"

	fmt.Printf("  Creando jerarquía para: %s (partes: %v)\n", ruta, partes)

	for _, parte := range partes {
		if parte == "" {
			continue
		}

		rutaActual = filepath.Join(rutaActual, parte)

		fmt.Printf("  Procesando: %s\n", rutaActual)

		_, _, err := structs.BuscarInodoPorRuta_(f, sb, rutaActual)
		if err == nil {
			fmt.Printf("  Directorio ya existe: %s\n", rutaActual)
			continue
		}

		fmt.Printf("  Directorio no existe, creando: %s\n", rutaActual)

		rutaPadre := filepath.Dir(rutaActual)
		fmt.Printf("  Buscando padre: %s\n", rutaPadre)
		inodoPadre, inodoPadreNum, err := structs.BuscarInodoPorRuta_(f, sb, rutaPadre)
		if err != nil {
			return fmt.Errorf("no se pudo encontrar directorio padre %s: %v", rutaPadre, err)
		}

		if !structs.EsCarpeta(inodoPadre) {
			return fmt.Errorf("%s no es un directorio", rutaPadre)
		}

		nuevoInodo := int32(sb.S_first_ino)
		sb.S_first_ino++
		sb.S_free_inodes_count--

		var ino structs.Inodo
		ino.I_uid = 1
		ino.I_gid = 1
		ino.I_s = 0
		ahora := time.Now().Format("2006-01-02 15:04")
		copy(ino.I_atime[:], ahora)
		copy(ino.I_ctime[:], ahora)
		copy(ino.I_mtime[:], ahora)
		for i := range ino.I_block {
			ino.I_block[i] = -1
		}
		ino.I_type[0] = 0
		copy(ino.I_perm[:], "664")

		nuevoBloque := int32(sb.S_first_blo)
		sb.S_first_blo++
		sb.S_free_blocks_count--
		ino.I_block[0] = nuevoBloque

		var bloqueCarpeta structs.BCarpeta
		bloqueCarpeta.B_content[0].B_inodo = nuevoInodo
		copy(bloqueCarpeta.B_content[0].B_name[:], ".")
		bloqueCarpeta.B_content[1].B_inodo = inodoPadreNum
		copy(bloqueCarpeta.B_content[1].B_name[:], "..")
		for i := 2; i < 4; i++ {
			bloqueCarpeta.B_content[i].B_inodo = -1
		}

		offset := int64(sb.S_block_start) + int64(nuevoBloque)*64
		if _, err := f.Seek(offset, 0); err != nil {
			return fmt.Errorf("error al posicionar bloque carpeta: %v", err)
		}
		if err := binary.Write(f, binary.LittleEndian, &bloqueCarpeta); err != nil {
			return fmt.Errorf("error al escribir bloque carpeta: %v", err)
		}

		offsetBitmapBloque := int64(sb.S_bm_block_start) + int64(nuevoBloque)
		if _, err := f.Seek(offsetBitmapBloque, 0); err != nil {
			return err
		}
		if _, err := f.Write([]byte{1}); err != nil {
			return err
		}

		offsetInodo := int64(sb.S_inode_start) + int64(nuevoInodo)*int64(sb.S_inode_s)
		if _, err := f.Seek(offsetInodo, 0); err != nil {
			return fmt.Errorf("error al posicionar inodo: %v", err)
		}
		if err := binary.Write(f, binary.LittleEndian, &ino); err != nil {
			return fmt.Errorf("error al escribir inodo: %v", err)
		}

		offsetBitmapInodo := int64(sb.S_bm_inode_start) + int64(nuevoInodo)
		if _, err := f.Seek(offsetBitmapInodo, 0); err != nil {
			return err
		}
		if _, err := f.Write([]byte{1}); err != nil {
			return err
		}

		if err := agregarEntradaDirectorio(f, sb, &inodoPadre, inodoPadreNum, parte, nuevoInodo); err != nil {
			return fmt.Errorf("error al agregar entrada al directorio padre: %v", err)
		}

		if _, err := f.Seek(int64(partStart), 0); err != nil {
			return err
		}
		if err := binary.Write(f, binary.LittleEndian, sb); err != nil {
			return fmt.Errorf("error al actualizar superbloque: %v", err)
		}

		fmt.Printf("  Directorio creado: %s (inodo: %d)\n", rutaActual, nuevoInodo)
	}

	return nil
}

func agregarEntradaDirectorio(f *os.File, sb *structs.SuperBloque, inodoPadre *structs.Inodo, inodoPadreNum int32, nombre string, inodoHijo int32) error {
	for i := 0; i < 12; i++ {
		blockIdx := inodoPadre.I_block[i]
		if blockIdx < 0 {
			nuevoBloque := int32(sb.S_first_blo)
			sb.S_first_blo++
			sb.S_free_blocks_count--
			inodoPadre.I_block[i] = nuevoBloque

			var bc structs.BCarpeta
			bc.B_content[0].B_inodo = inodoHijo
			copy(bc.B_content[0].B_name[:], nombre)
			for j := 1; j < 4; j++ {
				bc.B_content[j].B_inodo = -1
			}

			offset := int64(sb.S_block_start) + int64(nuevoBloque)*64
			if _, err := f.Seek(offset, 0); err != nil {
				return err
			}
			if err := binary.Write(f, binary.LittleEndian, &bc); err != nil {
				return err
			}

			offsetBitmap := int64(sb.S_bm_block_start) + int64(nuevoBloque)
			if _, err := f.Seek(offsetBitmap, 0); err != nil {
				return err
			}
			if _, err := f.Write([]byte{1}); err != nil {
				return err
			}

			offsetInodo := int64(sb.S_inode_start) + int64(inodoPadreNum)*int64(sb.S_inode_s)
			if _, err := f.Seek(offsetInodo, 0); err != nil {
				return err
			}
			if err := binary.Write(f, binary.LittleEndian, inodoPadre); err != nil {
				return err
			}

			return nil
		}

		bc, ok := structs.LeerBloqueCarpeta(f, sb, blockIdx)
		if !ok {
			continue
		}

		for j := 0; j < 4; j++ {
			if bc.B_content[j].B_inodo < 0 {
				bc.B_content[j].B_inodo = inodoHijo
				copy(bc.B_content[j].B_name[:], nombre)

				offset := int64(sb.S_block_start) + int64(blockIdx)*64
				if _, err := f.Seek(offset, 0); err != nil {
					return err
				}
				if err := binary.Write(f, binary.LittleEndian, &bc); err != nil {
					return err
				}
				return nil
			}
		}
	}

	return fmt.Errorf("no hay espacio en el directorio padre")
}

func EscribirArchivoFSConPartStart(f *os.File, sb *structs.SuperBloque, partStart int32, ruta, contenido string) error {
	ruta = strings.TrimSpace(ruta)
	if ruta == "" || ruta == "/" {
		return fmt.Errorf("ruta inválida para archivo")
	}

	if err := VerificarYCrearRaiz(f, sb, partStart); err != nil {
		return fmt.Errorf("error al verificar raíz: %v", err)
	}

	rutaPadre := filepath.Dir(ruta)
	if rutaPadre != "/" && rutaPadre != "." {
		if err := CrearDirectorioFSConPartStart(f, sb, partStart, rutaPadre); err != nil {
			return fmt.Errorf("error al crear directorios padre: %v", err)
		}
	}

	ino, inodoNum, err := structs.BuscarInodoPorRuta_(f, sb, ruta)
	if err != nil {
		return crearNuevoArchivo(f, sb, partStart, ruta, contenido)
	}

	if structs.EsCarpeta(ino) {
		return fmt.Errorf("la ruta %s es una carpeta, no un archivo", ruta)
	}

	return escribirContenidoArchivo(f, sb, partStart, ino, inodoNum, ruta, contenido)
}

func crearNuevoArchivo(f *os.File, sb *structs.SuperBloque, partStart int32, ruta, contenido string) error {
	rutaPadre := filepath.Dir(ruta)
	nombreArchivo := filepath.Base(ruta)

	inodoPadre, inodoPadreNum, err := structs.BuscarInodoPorRuta_(f, sb, rutaPadre)
	if err != nil {
		return fmt.Errorf("no se pudo encontrar directorio padre %s: %v", rutaPadre, err)
	}

	if !structs.EsCarpeta(inodoPadre) {
		return fmt.Errorf("%s no es un directorio", rutaPadre)
	}

	nuevoInodo := int32(sb.S_first_ino)
	sb.S_first_ino++
	sb.S_free_inodes_count--

	var ino structs.Inodo
	ino.I_uid = 1
	ino.I_gid = 1
	ino.I_s = int32(len(contenido))
	ahora := time.Now().Format("2006-01-02 15:04")
	copy(ino.I_atime[:], ahora)
	copy(ino.I_ctime[:], ahora)
	copy(ino.I_mtime[:], ahora)
	for i := range ino.I_block {
		ino.I_block[i] = -1
	}
	ino.I_type[0] = 1
	copy(ino.I_perm[:], "664")

	data := []byte(contenido)
	totalBloques := (len(data) + 63) / 64
	if totalBloques > 12 {
		totalBloques = 12
	}

	for i := 0; i < totalBloques; i++ {
		nuevoBloque := int32(sb.S_first_blo)
		sb.S_first_blo++
		sb.S_free_blocks_count--
		ino.I_block[i] = nuevoBloque

		var block structs.BArchivo
		inicio := i * 64
		fin := min((i+1)*64, len(data))
		copy(block.B_content[:], data[inicio:fin])

		if err := structs.EscribirBloqueArchivo(f, sb, partStart, nuevoBloque, &block, "write", ruta); err != nil {
			return fmt.Errorf("error al escribir bloque %d: %v", i, err)
		}

		offsetBitmap := int64(sb.S_bm_block_start) + int64(nuevoBloque)
		if _, err := f.Seek(offsetBitmap, 0); err != nil {
			return err
		}
		if _, err := f.Write([]byte{1}); err != nil {
			return err
		}
	}

	offsetInodo := int64(sb.S_inode_start) + int64(nuevoInodo)*int64(sb.S_inode_s)
	if _, err := f.Seek(offsetInodo, 0); err != nil {
		return fmt.Errorf("error al posicionar inodo: %v", err)
	}
	if err := binary.Write(f, binary.LittleEndian, &ino); err != nil {
		return fmt.Errorf("error al escribir inodo: %v", err)
	}

	offsetBitmapInodo := int64(sb.S_bm_inode_start) + int64(nuevoInodo)
	if _, err := f.Seek(offsetBitmapInodo, 0); err != nil {
		return err
	}
	if _, err := f.Write([]byte{1}); err != nil {
		return err
	}

	if err := agregarEntradaDirectorio(f, sb, &inodoPadre, inodoPadreNum, nombreArchivo, nuevoInodo); err != nil {
		return fmt.Errorf("error al agregar entrada al directorio padre: %v", err)
	}

	if _, err := f.Seek(int64(partStart), 0); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, sb); err != nil {
		return fmt.Errorf("error al actualizar superbloque: %v", err)
	}

	fmt.Printf("  Archivo creado: %s (inodo: %d, tamaño: %d bytes)\n", ruta, nuevoInodo, len(contenido))
	return nil
}

func escribirContenidoArchivo(f *os.File, sb *structs.SuperBloque, partStart int32, ino structs.Inodo, inodoNum int32, ruta, contenido string) error {
	data := []byte(contenido)
	ino.I_s = int32(len(data))
	ahora := time.Now().Format("2006-01-02 15:04")
	copy(ino.I_mtime[:], ahora)

	totalBloques := (len(data) + 63) / 64
	if totalBloques > 12 {
		totalBloques = 12
	}

	for i := 0; i < totalBloques; i++ {
		blockIdx := ino.I_block[i]
		
		if blockIdx == -1 {
			blockIdx = int32(sb.S_first_blo)
			sb.S_first_blo++
			sb.S_free_blocks_count--
			ino.I_block[i] = blockIdx

			offsetBitmap := int64(sb.S_bm_block_start) + int64(blockIdx)
			if _, err := f.Seek(offsetBitmap, 0); err != nil {
				return err
			}
			if _, err := f.Write([]byte{1}); err != nil {
				return err
			}
		}

		var block structs.BArchivo
		inicio := i * 64
		fin := min((i+1)*64, len(data))
		copy(block.B_content[:], data[inicio:fin])

		if err := structs.EscribirBloqueArchivo(f, sb, partStart, blockIdx, &block, "write", ruta); err != nil {
			return fmt.Errorf("error al escribir bloque %d: %v", i, err)
		}
	}

	offsetInodo := int64(sb.S_inode_start) + int64(inodoNum)*int64(sb.S_inode_s)
	if _, err := f.Seek(offsetInodo, 0); err != nil {
		return fmt.Errorf("error al actualizar inodo: %v", err)
	}
	if err := binary.Write(f, binary.LittleEndian, &ino); err != nil {
		return fmt.Errorf("error al escribir inodo actualizado: %v", err)
	}

	if _, err := f.Seek(int64(partStart), 0); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, sb); err != nil {
		return fmt.Errorf("error al actualizar superbloque: %v", err)
	}

	fmt.Printf("  Archivo modificado: %s (tamaño: %d bytes)\n", ruta, len(contenido))
	return nil
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