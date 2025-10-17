package commands

import (
    "encoding/json"
    "backend/structs"
    "strings"
    "time"
)

type JournalEntry struct {
    Operation string `json:"operation"`
    Path      string `json:"path"`
    Content   string `json:"content"`
    Date      string `json:"date"`
}

func Journaling(params map[string]string) string {
    id, ok := params["-id"]
    if !ok || strings.TrimSpace(id) == "" {
        return `{"error":"Falta el parámetro obligatorio -id"}`
    }

    journal, err := ObtenerJournal(id)
    if err != nil {
        return `{"error":"` + strings.ReplaceAll(err.Error(), `"`, `'`) + `"}`
    }

    if len(journal) == 0 {
        return `{"entries":[]}`
    }

    entries := make([]JournalEntry, 0, len(journal))
    for _, entry := range journal {
        op := strings.Trim(string(entry.Operation[:]), "\x00")
        path := strings.Trim(string(entry.Path[:]), "\x00")
        content := strings.Trim(string(entry.Content[:]), "\x00")

        var dateStr string
        if entry.Date > 0 {
            t := int64(entry.Date)
            dateStr = time.Unix(t, 0).Format("2006-01-02 15:04:05")
        }

        entries = append(entries, JournalEntry{
            Operation: op,
            Path:      path,
            Content:   content,
            Date:      dateStr,
        })
    }
    jsonData, _ := json.Marshal(struct {
        Entries []JournalEntry `json:"entries"`
    }{entries})

    return string(jsonData)
}

func ObtenerJournal(id string) ([]structs.Information, error) {
    f, _, particion, err := structs.SuperBloque_ID(id)
    if err != nil {
        return nil, err
    }
    defer f.Close()
    journal, err := structs.LeerJournal(f, particion.Part_start)
    if err != nil {
        return nil, err
    }
    operaciones := make([]structs.Information, 0)
    if journal.Count > 0 {
        for i := int32(0); i < journal.Count && i < int32(len(journal.Content)); i++ {
            operaciones = append(operaciones, journal.Content[i])
        }
    }
    return operaciones, nil
}