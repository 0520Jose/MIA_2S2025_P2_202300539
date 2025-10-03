package commands

import (
    "encoding/json"
    "strings"
)

type JournalEntry struct {
    Operation string `json:"operation"`
    Path      string `json:"path"`
    Content   string `json:"content"`
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
        op := strings.Trim(string(entry.I_operation[:]), "\x00")
        path := strings.Trim(string(entry.I_path[:]), "\x00")
        content := strings.Trim(string(entry.I_content[:]), "\x00")
        entries = append(entries, JournalEntry{
            Operation: op,
            Path:      path,
            Content:   content,
        })
    }
    jsonData, _ := json.Marshal(struct {
        Entries []JournalEntry `json:"entries"`
    }{entries})
    return string(jsonData)
}