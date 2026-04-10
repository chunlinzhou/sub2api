package service

import (
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type LocalSkillSummary struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Filename  string    `json:"filename"`
	Size      int64     `json:"size"`
	UpdatedAt time.Time `json:"updated_at"`
}

func LocalSkillsDir() string {
	return filepath.Join(localSkillsDataDir(), "skills")
}

func ListLocalSkillSummaries() ([]LocalSkillSummary, error) {
	dir := LocalSkillsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []LocalSkillSummary{}, nil
		}
		return nil, err
	}

	skills := make([]LocalSkillSummary, 0, len(entries))
	for _, entry := range entries {
		if !isAllowedLocalSkillEntry(entry) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		filename := entry.Name()
		skills = append(skills, LocalSkillSummary{
			ID:        filename,
			Name:      strings.TrimSuffix(filename, filepath.Ext(filename)),
			Filename:  filename,
			Size:      info.Size(),
			UpdatedAt: info.ModTime(),
		})
	}

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Filename < skills[j].Filename
	})
	return skills, nil
}

func CompileLocalSkillText(skillIDs []string) string {
	if len(skillIDs) == 0 {
		return ""
	}

	dir := LocalSkillsDir()
	parts := make([]string, 0, len(skillIDs))
	for _, rawID := range skillIDs {
		id, ok := normalizeLocalSkillID(rawID)
		if !ok {
			slog.Warn("skip invalid local skill id", "skill_id", rawID)
			continue
		}
		path := filepath.Join(dir, id)
		content, err := os.ReadFile(path)
		if err != nil {
			slog.Warn("skip unreadable local skill", "skill_id", id, "path", path, "error", err)
			continue
		}
		text := strings.TrimSpace(string(content))
		if text == "" {
			continue
		}
		parts = append(parts, "[Skill: "+strings.TrimSuffix(id, filepath.Ext(id))+"]\n"+text)
	}

	return strings.Join(parts, "\n\n")
}

func isAllowedLocalSkillEntry(entry os.DirEntry) bool {
	if entry.IsDir() {
		return false
	}
	name := entry.Name()
	if strings.HasPrefix(name, ".") {
		return false
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".md", ".txt":
		return true
	default:
		return false
	}
}

func normalizeLocalSkillID(raw string) (string, bool) {
	id := strings.TrimSpace(raw)
	if id == "" {
		return "", false
	}
	if filepath.Base(id) != id || strings.ContainsAny(id, `/\`) {
		return "", false
	}
	switch strings.ToLower(filepath.Ext(id)) {
	case ".md", ".txt":
		return id, true
	default:
		return "", false
	}
}

func localSkillsDataDir() string {
	if dir := strings.TrimSpace(os.Getenv("DATA_DIR")); dir != "" {
		return dir
	}

	const dockerDataDir = "/app/data"
	if info, err := os.Stat(dockerDataDir); err == nil && info.IsDir() {
		testFile := filepath.Join(dockerDataDir, ".write_test")
		if f, err := os.Create(testFile); err == nil {
			_ = f.Close()
			_ = os.Remove(testFile)
			return dockerDataDir
		}
	}

	return "."
}
