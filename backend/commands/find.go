package commands

import (
	"backend/structs"
	"fmt"
	"os"
	"strings"
)

func Find(params map[string]string) string {
	path, okPath := params["-path"]
	pattern, okName := params["-name"]

	if !okPath || !okName || strings.TrimSpace(path) == "" || strings.TrimSpace(pattern) == "" {
		return "Error: Parámetros -path y -name son obligatorios"
	}

	disk, sb, err := CargarSistemaEXT2(usuarioActual.PartitionID)
	if err != nil {
		return fmt.Sprintf("Error al cargar sistema: %v", err)
	}
	defer disk.Close()

	inode, _, err := structs.BuscarInodoPorRuta_(disk, sb, path)
	if err != nil {
		return fmt.Sprintf("Error: ruta no encontrada: %v", err)
	}

	if !Permisos(&inode, permRead) {
		return fmt.Sprintf("Error: no tiene permisos de lectura sobre %s", path)
	}

	var resultados []string
	buscarRecursivo(disk, sb, inode, strings.TrimRight(path, "/"), pattern, &resultados)

	if len(resultados) == 0 {
		return "No se encontraron coincidencias"
	}

	return strings.Join(resultados, "\n")
}

func buscarRecursivo(f *os.File, sb *structs.SuperBloque, inode structs.Inodo, rutaActual, patron string, resultados *[]string) {
	if !Permisos(&inode, permRead) {
		return
	}

	if structs.EsCarpeta(inode) {
		for _, idx := range inode.I_block {
			if idx < 0 {
				continue
			}
			bloque, ok := structs.LeerBloqueCarpeta(f, sb, idx)
			if !ok {
				continue
			}
			for _, entry := range bloque.B_content {
				nombre := strings.TrimRight(string(entry.B_name[:]), "\x00")
				if nombre == "" || nombre == "." || nombre == ".." {
					continue
				}

				hijo, err := ReadInode(f, sb, entry.B_inodo)
				if err != nil {
					continue
				}

				nuevaRuta := strings.TrimRight(rutaActual, "/") + "/" + nombre

				if wildcardMatch(patron, nombre) {
					*resultados = append(*resultados, nuevaRuta)
				}

				buscarRecursivo(f, sb, hijo, nuevaRuta, patron, resultados)
			}
		}
	} else {
		nombre := obtenerNombreArchivo(rutaActual)
		if wildcardMatch(patron, nombre) {
			*resultados = append(*resultados, rutaActual)
		}
	}
}

func wildcardMatch(pattern, str string) bool {
	p, s := 0, 0
	star, match := -1, 0

	for s < len(str) {
		if p < len(pattern) && (pattern[p] == str[s] || pattern[p] == '?') {
			p++
			s++
		} else if p < len(pattern) && pattern[p] == '*' {
			star = p
			match = s
			p++
		} else if star != -1 {
			p = star + 1
			match++
			s = match
		} else {
			return false
		}
	}

	for p < len(pattern) && pattern[p] == '*' {
		p++
	}

	return p == len(pattern)
}

func obtenerNombreArchivo(ruta string) string {
	parts := strings.Split(strings.Trim(ruta, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}
