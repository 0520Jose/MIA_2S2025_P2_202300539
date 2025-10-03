package commands

import (
	"backend/structs"
	"backend/ext3_structs"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

func SimularPerdida(id string) error {
	f, sb, _, err := structs.SuperBloque_ID(id)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Seek(int64(sb.S_bm_inode_start), io.SeekStart); err != nil {
		return err
	}
	f.Write(make([]byte, sb.S_inodes_count))

	if _, err := f.Seek(int64(sb.S_bm_block_start), io.SeekStart); err != nil {
		return err
	}
	f.Write(make([]byte, sb.S_blocks_count))

	if _, err := f.Seek(int64(sb.S_inode_start), io.SeekStart); err != nil {
		return err
	}
	f.Write(make([]byte, int(sb.S_inode_s)*int(sb.S_inodes_count)))

	if _, err := f.Seek(int64(sb.S_block_start), io.SeekStart); err != nil {
		return err
	}
	f.Write(make([]byte, int(sb.S_block_s)*int(sb.S_blocks_count)))

	return nil
}

func RecuperarSistema(id string) error {
	f, sb, _, err := structs.SuperBloque_ID(id)
	if err != nil {
		return err
	}
	defer f.Close()

	var j ext3_structs.Journal
	journalOffset := int64(sb.S_block_start) + int64(sb.S_blocks_count*sb.S_block_s)
	if _, err := f.Seek(journalOffset, io.SeekStart); err != nil {
		return err
	}

	if err := binary.Read(f, binary.LittleEndian, &j); err != nil {
		return err
	}

	for i := 0; i < int(j.J_count); i++ {
		op := strings.Trim(string(j.J_content[i].I_operation[:]), "\x00")
		path := strings.Trim(string(j.J_content[i].I_path[:]), "\x00")
		content := strings.Trim(string(j.J_content[i].I_content[:]), "\x00")

		switch op {
		case "chown":
			if err := Chown(id, path, content, true); err != nil {
				fmt.Printf("Error aplicando chown a %s: %v\n", path, err)
			}
		case "chmod":
			if err := Chmod(id, path, content, true); err != nil {
				fmt.Printf("Error aplicando chmod a %s: %v\n", path, err)
			}
		case "write":
			if err := EscribirArchivoFS(id, path, content); err != nil {
				fmt.Printf("Error escribiendo archivo %s: %v\n", path, err)
			}
		default:
			fmt.Printf("Operación desconocida: %s\n", op)
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

func ObtenerJournal(id string) ([]ext3_structs.Information, error) {
	f, sb, _, err := structs.SuperBloque_ID(id)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var j ext3_structs.Journal
	journalOffset := int64(sb.S_block_start) + int64(sb.S_blocks_count*sb.S_block_s)
	if _, err := f.Seek(journalOffset, io.SeekStart); err != nil {
		return nil, err
	}
	if err := binary.Read(f, binary.LittleEndian, &j); err != nil {
		return nil, err
	}

	return j.J_content[:j.J_count], nil
}

func RegistrarOperacionJournal(f *os.File, sb *structs.SuperBloque, op, path, content string) error {
	var j ext3_structs.Journal
	journalOffset := int64(sb.S_block_start) + int64(sb.S_blocks_count*sb.S_block_s)

	if _, err := f.Seek(journalOffset, io.SeekStart); err != nil {
		return err
	}
	if err := binary.Read(f, binary.LittleEndian, &j); err != nil {
		j.J_count = 0
	}

	if j.J_count >= 64 {
		j.J_count = 0
	}

	var info ext3_structs.Information
	copy(info.I_operation[:], []byte(op))
	copy(info.I_path[:], []byte(path))
	copy(info.I_content[:], []byte(content))
	info.I_date = float32(time.Now().Unix())

	j.J_content[j.J_count] = info
	j.J_count++

	if _, err := f.Seek(journalOffset, io.SeekStart); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, &j); err != nil {
		return err
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
