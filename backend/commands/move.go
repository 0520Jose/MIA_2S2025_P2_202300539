package commands

import (
	"backend/structs"
	"fmt"
	"strings"
	"os"
)

func Move(params map[string]string) string {
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

    if !strings.HasPrefix(rutaOrigen, "/") || !strings.HasPrefix(rutaDestino, "/") {
        return "Error: las rutas deben ser absolutas."
    }

    disk, sb, err := CargarSistemaEXT2(usuarioActual.PartitionID)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    defer disk.Close()

    inodoOrigen, inodoOrigenIdx, err := structs.BuscarInodoPorRuta_(disk, sb, rutaOrigen)
    if err != nil {
        return fmt.Sprintf("Error: ruta de origen no existe: %v", err)
    }
    if !Permisos(&inodoOrigen, permWrite) {
        return "Error: no tiene permisos de escritura sobre el recurso origen."
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

    padreOrigenIdx, _, err := buscarPadreYNombre(disk, sb, rutaOrigen)
    if err != nil {
        return "Error al buscar padre del origen: " + err.Error()
    }
    if err := quitarEntradaEnDirectorio(disk, sb, padreOrigenIdx, inodoOrigenIdx); err != nil {
        return "Error al quitar referencia en el directorio origen: " + err.Error()
    }
    if err := agregarEntradaEnDirectorio(disk, sb, inodoDestinoIdx, inodoOrigenIdx, nombreOrigen); err != nil {
        _ = agregarEntradaEnDirectorio(disk, sb, padreOrigenIdx, inodoOrigenIdx, nombreOrigen)
        return "Error al agregar referencia en el destino: " + err.Error()
    }

    if structs.EsCarpeta(inodoOrigen) {
        if err := actualizarPadreCarpeta(disk, sb, inodoOrigen, inodoDestinoIdx); err != nil {
            return "Advertencia: no se pudo actualizar el campo '..' de la carpeta movida"
        }
    }

    return "Movimiento realizado correctamente."
}

func buscarPadreYNombre(f *os.File, sb *structs.SuperBloque, ruta string) (int32, string, error) {
    ruta = strings.TrimRight(ruta, "/")
    if ruta == "" || ruta == "/" {
        return -1, "", fmt.Errorf("la ruta es raíz")
    }
    partes := strings.Split(ruta, "/")
    nombre := partes[len(partes)-1]
    padreRuta := "/" + strings.Join(partes[:len(partes)-1], "/")
    if padreRuta == "" {
        padreRuta = "/"
    }
    _, padreIdx, err := structs.BuscarInodoPorRuta_(f, sb, padreRuta)
    if err != nil {
        return -1, "", err
    }
    return padreIdx, nombre, nil
}

func quitarEntradaEnDirectorio(f *os.File, sb *structs.SuperBloque, padreIdx, childIdx int32) error {
    inodoPadre, err := ReadInode(f, sb, padreIdx)
    if err != nil {
        return err
    }
    for _, blk := range inodoPadre.I_block {
        if blk == -1 {
            break
        }
        dir, err := ReadDirBlock(f, sb, blk)
        if err != nil {
            continue
        }
        for i := range dir.B_content {
            if dir.B_content[i].B_inodo == childIdx {
                dir.B_content[i].B_inodo = -1
                for j := range dir.B_content[i].B_name {
                    dir.B_content[i].B_name[j] = 0
                }
                return writeDirBlock(f, sb, blk, &dir)
            }
        }
    }
    return fmt.Errorf("no se encontró la entrada a eliminar")
}

func actualizarPadreCarpeta(f *os.File, sb *structs.SuperBloque, inodo structs.Inodo, nuevoPadreIdx int32) error {
    if inodo.I_block[0] == -1 {
        return fmt.Errorf("la carpeta no tiene bloque de directorio")
    }
    dir, err := ReadDirBlock(f, sb, inodo.I_block[0])
    if err != nil {
        return err
    }
    dir.B_content[1].B_inodo = nuevoPadreIdx
    return writeDirBlock(f, sb, inodo.I_block[0], &dir)
}