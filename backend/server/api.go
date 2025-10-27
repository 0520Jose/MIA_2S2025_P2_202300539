package server

import (
    "backend/commands"
    "backend/utils"
    "encoding/json"
    "log"
    "net/http"
    "strings"
)

type CommandRequest struct {
    Comando string `json:"comando"`
}

func executeHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

    if r.Method == "OPTIONS" {
        w.WriteHeader(http.StatusOK)
        return
    }

    var req CommandRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Error de decodificación del cuerpo de la solicitud", http.StatusBadRequest)
        return
    }

    var fullOutput strings.Builder
    lines := strings.Split(req.Comando, "\n")
    for _, raw := range lines {
        line := strings.TrimRight(raw, "\r")
        trimmed := strings.TrimSpace(line)

        if trimmed == "" {
            fullOutput.WriteString("\n")
            continue
        }
        
        if strings.HasPrefix(trimmed, "#") {
            fullOutput.WriteString(line + "\n")
            continue
        }

        var command string
        var comment string
        
        if commentIndex := strings.Index(line, "#"); commentIndex != -1 {
            command = strings.TrimSpace(line[:commentIndex])
            comment = line[commentIndex:]
        } else {
            command = trimmed
        }

        if command != "" {
            output := commands.ExecuteCommand(command)
            if output != "" {
                fullOutput.WriteString(output + "\n")
            }
        }

        if comment != "" {
            fullOutput.WriteString(comment + "\n")
        }
    }

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(map[string]string{"salida": fullOutput.String()})
}

func IniciarAPIServer(port string) {
    http.HandleFunc("/execute", executeHandler)
    http.HandleFunc("/api/disks", ListDisksHandler)
    http.HandleFunc("/api/partitions", ListPartitionsHandler)
    http.HandleFunc("/api/files", ListFilesHandler)
    http.HandleFunc("/api/file-content", FileContentHandler)

    log.Println("Iniciando el servidor API en el puerto", port)
    if err := http.ListenAndServe(port, nil); err != nil {
        log.Fatalf("Error al iniciar el servidor: %v", err)
    }
}

func ListDisksHandler(w http.ResponseWriter, r *http.Request) {
    disks, err := utils.ListDisks("/home/ubuntu/Resultado/Calificacion_MIA/Discos")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    json.NewEncoder(w).Encode(disks)
}

func ListPartitionsHandler(w http.ResponseWriter, r *http.Request) {
    diskPath := r.URL.Query().Get("path")
    partitions, err := utils.ListPartitions(diskPath)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    json.NewEncoder(w).Encode(partitions)
}

func ListFilesHandler(w http.ResponseWriter, r *http.Request) {
    diskPath := r.URL.Query().Get("disk")
    partitionName := r.URL.Query().Get("partition")
    path := r.URL.Query().Get("path")

    if diskPath == "" || partitionName == "" || path == "" {
        http.Error(w, "Faltan parámetros (disk, partition, path)", http.StatusBadRequest)
        return
    }

    files, err := utils.ListFilesFromDisk(diskPath, partitionName, path)
    if err != nil {
        http.Error(w, "Error listando archivos: "+err.Error(), http.StatusInternalServerError)
        return
    }

    type FileNode struct {
        Name        string `json:"name"`
        Type        string `json:"type"`
        Size        int32  `json:"size"`
        Permissions string `json:"permissions"`
    }

    var nodes []FileNode
    for _, f := range files {
        t := "f"
        if f.Tipo == "d" {
            t = "d"
        }
        nodes = append(nodes, FileNode{
            Name:        f.Nombre,
            Type:        t,
            Size:        f.Size,
            Permissions: f.Permisos,
        })
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(nodes)
}

func FileContentHandler(w http.ResponseWriter, r *http.Request) {
    disk := r.URL.Query().Get("disk")
    partition := r.URL.Query().Get("partition")
    path := r.URL.Query().Get("path")

    if disk == "" || partition == "" || path == "" {
        http.Error(w, "Faltan parámetros", http.StatusBadRequest)
        return
    }

    content, err := utils.GetFileContent(disk, partition, path)
    if err != nil {
        json.NewEncoder(w).Encode(map[string]string{"content": "Error: " + err.Error()})
        return
    }

    json.NewEncoder(w).Encode(map[string]string{"content": content})
}