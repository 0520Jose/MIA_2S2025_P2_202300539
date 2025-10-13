package commands

import (
    "encoding/json"
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
        op := strings.Trim(string(entry.I_operation[:]), "\x00")
        path := strings.Trim(string(entry.I_path[:]), "\x00")
        content := strings.Trim(string(entry.I_content[:]), "\x00")

        var dateStr string
        if entry.I_date > 0 {
            t := int64(entry.I_date)
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
