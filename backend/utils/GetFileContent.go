package utils

import (
	"backend/structs"
	"fmt"
	"os"
	"strings"
)

func GetFileContent(diskPath, partitionName, filePath string) (string, error) {
	f, err := os.Open(diskPath)
	if err != nil {
		return "", fmt.Errorf("no se pudo abrir el disco: %v", err)
	}
	defer f.Close()

	var mbr structs.MBR
	if err := structs.ReadBinaryStruct(f, &mbr); err != nil {
		return "", fmt.Errorf("error leyendo MBR: %v", err)
	}

	var start int32 = -1
	for _, part := range mbr.Mbr_partitions {
		name := strings.TrimRight(string(part.Part_name[:]), "\x00")
		if name == partitionName {
			start = part.Part_start
			break
		}
	}
	if start < 0 {
		return "", fmt.Errorf("partición %s no encontrada", partitionName)
	}

	if _, err := f.Seek(int64(start), 0); err != nil {
		return "", fmt.Errorf("error buscando superbloque: %v", err)
	}
	var sb structs.SuperBloque
	if err := structs.ReadBinaryStruct(f, &sb); err != nil {
		return "", fmt.Errorf("error leyendo superbloque: %v", err)
	}

	ino, err := buscarInodoPorRutaRaw(f, &sb, filePath)
	if err != nil {
		return "", err
	}
	if structs.EsCarpeta(ino) {
		return "", fmt.Errorf("la ruta especificada es una carpeta, no un archivo")
	}

	var content strings.Builder
	bytesLeidos := int32(0)
	tamanoArchivo := ino.I_s

	for i := 0; i < structs.DIRECT_BLOCKS && bytesLeidos < tamanoArchivo; i++ {
		blockIdx := ino.I_block[i]
		if blockIdx == -1 {
			continue
		}
		ba, ok := structs.LeerBloqueArchivo(f, &sb, blockIdx)
		if !ok {
			continue
		}
		toRead := int32(len(ba.B_content))
		if bytesLeidos+toRead > tamanoArchivo {
			toRead = tamanoArchivo - bytesLeidos
		}
		content.Write(ba.B_content[:toRead])
		bytesLeidos += toRead
	}

	if bytesLeidos < tamanoArchivo && ino.I_block[structs.INDIRECT_SIMPLE] != -1 {
		siBlockIdx := ino.I_block[structs.INDIRECT_SIMPLE]
		bp, ok := structs.ObtenerBloqueApuntadores(f, &sb, int(siBlockIdx))
		if ok {
			for _, blockIdx := range bp.B_pointers {
				if blockIdx == -1 || bytesLeidos >= tamanoArchivo {
					continue
				}
				ba, ok := structs.LeerBloqueArchivo(f, &sb, blockIdx)
				if !ok {
					continue
				}
				toRead := int32(len(ba.B_content))
				if bytesLeidos+toRead > tamanoArchivo {
					toRead = tamanoArchivo - bytesLeidos
				}
				content.Write(ba.B_content[:toRead])
				bytesLeidos += toRead
			}
		}
	}

	return strings.TrimRight(content.String(), "\x00"), nil
}
