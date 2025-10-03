package commands

import (
	"backend/structs"
	"fmt"
	"strings"
	"os"
)

func Rename(params map[string]string) string {
	if usuarioActual == nil {
		return "Error: No hay una sesión activa."
	}

	rawPath, ok := params["-path"]
	if !ok || strings.TrimSpace(rawPath) == "" {
		return "Error: parámetro -path es obligatorio."
	}

	pflag := false
	if v, ok := params["-p"]; ok {
		if strings.TrimSpace(v) != "" {
			return "Error: -p no recibe valor."
		}
		pflag = true
	}

	rawName, ok := params["-name"]
	if !ok || strings.TrimSpace(rawName) == "" {
		return "Error: parámetro -name es obligatorio."
	}

	ruta := unquoteValue(strings.TrimSpace(rawPath))
	nuevoNombre := unquoteValue(strings.TrimSpace(rawName))

	if !strings.HasPrefix(ruta, "/") {
		return "Error: -path debe ser ruta absoluta."
	}
	if strings.Contains(nuevoNombre, "/") {
		return "Error: el nombre no puede contener barras '/'."
	}
	if len(nuevoNombre) > 12 {
		return "Error: el nombre no puede exceder 12 caracteres."
	}

	partes := splitPathComponents(ruta)
	if len(partes) == 0 {
		return "Error: ruta invalida"
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

	padreIdx, err := ensureParentDir(disk, sb, partes[:len(partes)-1], pflag)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	if padreIdx == 0 && len(partes) == 1 {
		return "Error: no se puede renombrar el directorio raíz."
	}

	childIdx, _ := findEntryInDir(disk, sb, padreIdx, partes[len(partes)-1])
	if childIdx < 0 {
		return "Error: no existe el recurso a renombrar."
	}

	inodo, err := ReadInode(disk, sb, int32(childIdx))
	if err != nil {
		return "Error al leer inodo: " + err.Error()
	}

	if !Permisos(&inodo, permWrite) {
		return "Error: no tiene permisos de escritura sobre este recurso."
	}

	inodoPadre, err := ReadInode(disk, sb, padreIdx)
	if err != nil {
		return "Error al leer directorio padre: " + err.Error()
	}

	if existeNombreEnDirectorio(disk, sb, &inodoPadre, nuevoNombre, int32(childIdx)) {
		return fmt.Sprintf("Error: ya existe un archivo o carpeta con el nombre '%s'.", nuevoNombre)
	}

	if err := actualizarNombreEnPadre(disk, sb, padreIdx, int32(childIdx), nuevoNombre); err != nil {
		return "Error al actualizar nombre: " + err.Error()
	}

	if err := writeSuperBlock(disk, sb, int64(pm.Partition.Part_start)); err != nil {
		return "Error al actualizar superbloque: " + err.Error()
	}

	if sb.S_filesystem_type == 3 {
		if err := RegistrarOperacionJournal(disk, sb, "rename", ruta, nuevoNombre); err != nil {
			fmt.Printf("Advertencia: no se pudo registrar en journal: %v\n", err)
		}
	}

	return fmt.Sprintf("Recurso renombrado correctamente a '%s'.", nuevoNombre)
}


func existeNombreEnDirectorio(f *os.File, sb *structs.SuperBloque, inodoPadre *structs.Inodo, nombre string, excluirIdx int32) bool {
	for _, blk := range inodoPadre.I_block {
		if blk == -1 {
			continue
		}

		dir, err := ReadDirBlock(f, sb, blk)
		if err != nil {
			continue
		}

		for _, entry := range dir.B_content {
			if entry.B_inodo == -1 || entry.B_inodo == excluirIdx {
				continue
			}

			nombreEntry := strings.TrimRight(string(entry.B_name[:]), "\x00")
			if nombreEntry == nombre {
				return true
			}
		}
	}
	return false
}

func actualizarNombreEnPadre(f *os.File, sb *structs.SuperBloque, padreIdx int32, childIdx int32, nuevoNombre string) error {
	inodoPadre, err := ReadInode(f, sb, padreIdx)
	if err != nil {
		return err
	}

	if len(nuevoNombre) > 12 {
		return fmt.Errorf("el nombre no puede exceder 12 caracteres")
	}

	for _, blk := range inodoPadre.I_block {
		if blk == -1 {
			continue
		}

		dir, err := ReadDirBlock(f, sb, blk)
		if err != nil {
			return err
		}

		changed := false
		for i := range dir.B_content {
			if dir.B_content[i].B_inodo == childIdx {
				for j := range dir.B_content[i].B_name {
					dir.B_content[i].B_name[j] = 0
				}
				copy(dir.B_content[i].B_name[:], nuevoNombre)
				changed = true
				break
			}
		}

		if changed {
			if err := writeDirBlock(f, sb, blk, &dir); err != nil {
				return err
			}
			return nil
		}
	}

	return fmt.Errorf("no se encontró la entrada en el directorio padre")
}