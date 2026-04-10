package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUpsertLocalSkillUsesExternalDirectory(t *testing.T) {
	skillsDir := filepath.Join(t.TempDir(), "external-skills")
	t.Setenv("LOCAL_SKILLS_DIR", skillsDir)
	t.Setenv("DATA_DIR", filepath.Join(t.TempDir(), "data"))

	summary, err := UpsertLocalSkill("coding.md", []byte("follow the repo conventions"))
	if err != nil {
		t.Fatalf("UpsertLocalSkill() error = %v", err)
	}

	if summary.Filename != "coding.md" {
		t.Fatalf("unexpected filename: %s", summary.Filename)
	}

	content, err := os.ReadFile(filepath.Join(skillsDir, "coding.md"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(content) != "follow the repo conventions" {
		t.Fatalf("unexpected content: %q", string(content))
	}
}

func TestUpsertLocalSkillRejectsInvalidInput(t *testing.T) {
	t.Setenv("LOCAL_SKILLS_DIR", t.TempDir())

	if _, err := UpsertLocalSkill("../bad.md", []byte("test")); err != ErrInvalidLocalSkillName {
		t.Fatalf("expected ErrInvalidLocalSkillName, got %v", err)
	}

	if _, err := UpsertLocalSkill("bad.json", []byte("test")); err != ErrInvalidLocalSkillName {
		t.Fatalf("expected ErrInvalidLocalSkillName for extension, got %v", err)
	}

	if _, err := UpsertLocalSkill("empty.md", []byte("   \n")); err != ErrInvalidLocalSkillContent {
		t.Fatalf("expected ErrInvalidLocalSkillContent, got %v", err)
	}
}

func TestDeleteLocalSkill(t *testing.T) {
	skillsDir := filepath.Join(t.TempDir(), "external-skills")
	t.Setenv("LOCAL_SKILLS_DIR", skillsDir)

	if _, err := UpsertLocalSkill("cn.md", []byte("always answer in Chinese")); err != nil {
		t.Fatalf("UpsertLocalSkill() error = %v", err)
	}

	if err := DeleteLocalSkill("cn.md"); err != nil {
		t.Fatalf("DeleteLocalSkill() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(skillsDir, "cn.md")); !os.IsNotExist(err) {
		t.Fatalf("expected file to be removed, stat err = %v", err)
	}

	if err := DeleteLocalSkill("cn.md"); err != ErrLocalSkillNotFound {
		t.Fatalf("expected ErrLocalSkillNotFound, got %v", err)
	}
}
