package structs

import (
    "encoding/binary"
    "os"
    "io"
)

type Journal struct {
    Count   int32
    Content [50]Information 
}

type Information struct {
    Operation [10]byte
    Path      [128]byte
    Content   [64]byte 
    Date      float32
}

const JournalConstant = 50


func SuperBloque_EXT3_ID(id string) (*os.File, *SuperBloqueEXT3, Partition, error) {
    f, particion, _, err := SistemaArchivos_ID(id)
    if err != nil {
        return nil, nil, particion, err
    }

    sbEXT3 := &SuperBloqueEXT3{}
    if _, err := f.Seek(int64(particion.Part_start), io.SeekStart); err != nil {
        f.Close()
        return nil, nil, particion, err
    }

    if err := binary.Read(f, binary.LittleEndian, &sbEXT3.SuperBloque); err != nil {
        f.Close()
        return nil, nil, particion, err
    }

    if sbEXT3.EsEXT3() {
        sbEXT3.S_journal_start = particion.Part_start + int32(binary.Size(SuperBloque{}))
        sbEXT3.S_journal_size = JournalConstant
    }

    return f, sbEXT3, particion, nil
}

type SuperBloqueEXT3 struct {
    SuperBloque
    S_journal_start int32
    S_journal_size  int32
}

func CalcularEstructurasEXT3(tamanoParticion int64) int {
    sizeofSuperblock := int64(binary.Size(SuperBloque{}))
    sizeofJournaling := int64(JournalConstant)
    sizeofInodo := int64(binary.Size(Inodo{}))
    sizeofBlock := int64(64)

    coeficiente := sizeofJournaling + 1 + 3 + sizeofInodo + 3*sizeofBlock
    
    if coeficiente == 0 {
        return 0
    }
    
    n := float64(tamanoParticion - sizeofSuperblock) / float64(coeficiente)
    
    return int(n)
}

func (sb *SuperBloque) EsEXT3() bool {
    return sb.S_filesystem_type == 3
}

func GetJournalOffset(partStart int32) int64 {
    return int64(partStart) + int64(binary.Size(SuperBloque{}))
}

func LeerJournal(f *os.File, partStart int32) (Journal, error) {
    var journal Journal
    journalOffset := GetJournalOffset(partStart)
    
    if _, err := f.Seek(journalOffset, io.SeekStart); err != nil {
        return journal, err
    }
    
    err := binary.Read(f, binary.LittleEndian, &journal)
    return journal, err
}

func EscribirJournal(f *os.File, partStart int32, journal *Journal) error {
    journalOffset := GetJournalOffset(partStart)
    
    if _, err := f.Seek(journalOffset, io.SeekStart); err != nil {
        return err
    }
    
    return binary.Write(f, binary.LittleEndian, journal)
}