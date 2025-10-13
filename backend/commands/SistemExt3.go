package commands

import (
    "backend/structs"
    "encoding/binary"
    "fmt"
    "io"
    "os"
    "strings"
    "time"
)

func RecuperarEXT3(f *os.File, partStart int32, sb *SuperBloque) error {
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
        case "write", "create":
            if err := EscribirArchivoFS("diskID", ruta, contenido); err != nil {
                return fmt.Errorf("error al recuperar archivo/carpeta: %v", err)
            }
        case "delete":
            fmt.Printf("Recuperación de eliminación no implementada para: %s\n", ruta)
        case "modify":
            if err := EscribirArchivoFS("diskID", ruta, contenido); err != nil {
                return fmt.Errorf("error al recuperar modificación: %v", err)
            }
        default:
            return fmt.Errorf("operación no soportada: %s", operacion)
        }
    }

    return nil
}

func SimularPerdidaEXT3(f *os.File, sb *SuperBloque, partStart int32) error {
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

func EscribirArchivoFS(id, ruta, contenido string) error {
    f, sb, _, err := structs.SuperBloque_ID(id)
    if err != nil {
        return err
    }
    defer f.Close()

    ino, _, err := structs.BuscarInodoPorRuta_(f, sb, ruta)
    if err != nil {
        return err
    }

    if structs.EsCarpeta(ino) {
        return fmt.Errorf("la ruta %s es una carpeta, no un archivo", ruta)
    }

    data := []byte(contenido)
    totalBloques := (len(data) + 63) / 64

    for i := 0; i < totalBloques && i < 12; i++ {
        var block structs.BArchivo
        copy(block.B_content[:], data[i*64:min((i+1)*64, len(data))])
        if err := structs.EscribirBloqueArchivo(f, sb, ino.I_block[i], &block); err != nil {
            return err
        }
    }

    return nil
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

func RegistrarOperacion(f *os.File, partStart int32, operacion, ruta, contenido string) error {
    journal, err := LeerJournal(f, partStart)
    if err != nil {
        journal = Journal{Count: 0}
    }

    copy(journal.Content.Operation[:], operacion)
    copy(journal.Content.Path[:], ruta)
    copy(journal.Content.Content[:], contenido)
    journal.Content.Date = float32(time.Now().Unix())
    
    journal.Count++

    return EscribirJournal(f, partStart, &journal)
}

func ObtenerOperacionesJournal(f *os.File, partStart int32) ([]Information, error) {
    journal, err := LeerJournal(f, partStart)
    if err != nil {
        return nil, err
    }

    operaciones := make([]Information, 0)
    if journal.Count > 0 {
        operaciones = append(operaciones, journal.Content)
    }

    return operaciones, nil
}

func RegistrarJournaling(id, operacion, contenido string) error {
    f, sb, particion, err := structs.SuperBloque_ID(id)
    if err != nil {
        return err
    }
    defer f.Close()

    // Solo registrar si es EXT3
    if sb.S_filesystem_type != 3 {
        return nil // No es error, simplemente no registrar
    }

    journalOffset := int64(particion.Part_start) + int64(binary.Size(structs.SuperBloque{}))
    
    var journal structs.Journal
    if _, err := f.Seek(journalOffset, 0); err != nil {
        return err
    }

    // Leer journal actual
    if err := binary.Read(f, binary.LittleEndian, &journal); err != nil {
        journal.Count = 0 // Inicializar si no existe
    }

    // Actualizar journal
    copy(journal.Content.Operation[:], operacion)
    copy(journal.Content.Path[:], "")
    copy(journal.Content.Content[:], contenido)
    journal.Content.Date = float32(time.Now().Unix())
    journal.Count++

    // Escribir journal actualizado
    if _, err := f.Seek(journalOffset, 0); err != nil {
        return err
    }
    
    return binary.Write(f, binary.LittleEndian, &journal)
}