package pipeline

import (
	"fmt"
	"regexp"
	"strings"
)

type SrtEntry struct {
	Index int
	Start string
	End   string
	Text  string
}

func ParseSRT(content string) []SrtEntry {
	if content == "" {
		return nil
	}

	re := regexp.MustCompile(`\n\s*\n`)
	blocks := re.Split(strings.TrimSpace(content), -1)

	var entries []SrtEntry
	for _, block := range blocks {
		lines := strings.Split(strings.TrimSpace(block), "\n")
		if len(lines) < 3 {
			continue
		}

		var index int
		fmt.Sscanf(strings.TrimSpace(lines[0]), "%d", &index)

		timeLine := strings.TrimSpace(lines[1])
		parts := strings.Split(timeLine, " --> ")
		if len(parts) != 2 {
			continue
		}

		text := strings.Join(lines[2:], "\n")

		entries = append(entries, SrtEntry{
			Index: index,
			Start: strings.TrimSpace(parts[0]),
			End:   strings.TrimSpace(parts[1]),
			Text:  text,
		})
	}

	return entries
}

func BuildSRT(entries []SrtEntry) string {
	var sb strings.Builder
	for i, entry := range entries {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(fmt.Sprintf("%d\n%s --> %s\n%s", entry.Index, entry.Start, entry.End, entry.Text))
	}
	return sb.String()
}
