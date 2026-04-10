package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/tidwall/gjson"
)

func TestApplyGroupPromptPolicyAnthropicPrepend(t *testing.T) {
	group := &Group{
		PromptPolicy: domain.PromptPolicy{
			Enabled:                true,
			Mode:                   domain.PromptPolicyModePrepend,
			AnthropicSystemMessage: "team policy",
		},
	}

	body, err := ApplyGroupPromptPolicy([]byte(`{"system":"original","messages":[]}`), group, PromptPolicyTargetAnthropic)
	if err != nil {
		t.Fatalf("ApplyGroupPromptPolicy() error = %v", err)
	}
	if got := gjson.GetBytes(body, "system").String(); got != "team policy\n\noriginal" {
		t.Fatalf("unexpected anthropic system: %q", got)
	}
}

func TestApplyGroupPromptPolicyResponsesReplaceIfEmpty(t *testing.T) {
	group := &Group{
		PromptPolicy: domain.PromptPolicy{
			Enabled:            true,
			Mode:               domain.PromptPolicyModeReplaceIfEmpty,
			OpenAIInstructions: "team instructions",
		},
	}

	body, err := ApplyGroupPromptPolicy([]byte(`{"model":"gpt-5","input":[{"type":"input_text","text":"hi"}]}`), group, PromptPolicyTargetOpenAIResponses)
	if err != nil {
		t.Fatalf("ApplyGroupPromptPolicy() error = %v", err)
	}
	if got := gjson.GetBytes(body, "instructions").String(); got != "team instructions" {
		t.Fatalf("unexpected instructions: %q", got)
	}
}

func TestApplyGroupPromptPolicyChatCompletionsPrependDeveloper(t *testing.T) {
	group := &Group{
		PromptPolicy: domain.PromptPolicy{
			Enabled:            true,
			Mode:               domain.PromptPolicyModePrepend,
			OpenAIInstructions: "team instructions",
		},
	}

	body, err := ApplyGroupPromptPolicy([]byte(`{"model":"gpt-5","messages":[{"role":"user","content":"hi"}]}`), group, PromptPolicyTargetOpenAIChatCompletions)
	if err != nil {
		t.Fatalf("ApplyGroupPromptPolicy() error = %v", err)
	}
	if got := gjson.GetBytes(body, "messages.0.role").String(); got != "developer" {
		t.Fatalf("unexpected role: %q", got)
	}
	if got := gjson.GetBytes(body, "messages.0.content").String(); got != "team instructions" {
		t.Fatalf("unexpected content: %q", got)
	}
}

func TestApplyGroupPromptPolicyGeminiPrepend(t *testing.T) {
	group := &Group{
		PromptPolicy: domain.PromptPolicy{
			Enabled:                 true,
			Mode:                    domain.PromptPolicyModePrepend,
			GeminiSystemInstruction: "team system",
		},
	}

	body, err := ApplyGroupPromptPolicy([]byte(`{"contents":[{"role":"user","parts":[{"text":"hi"}]}]}`), group, PromptPolicyTargetGemini)
	if err != nil {
		t.Fatalf("ApplyGroupPromptPolicy() error = %v", err)
	}
	if got := gjson.GetBytes(body, "systemInstruction.parts.0.text").String(); got != "team system" {
		t.Fatalf("unexpected gemini system text: %q", got)
	}
}

func TestApplyGroupPromptPolicyChatCompletionsStrictRemovesUserDeveloper(t *testing.T) {
	group := &Group{
		PromptPolicy: domain.PromptPolicy{
			Enabled:              true,
			Mode:                 domain.PromptPolicyModeAppend,
			SkillEnforcementMode: domain.PromptPolicyEnforcementStrict,
			OpenAIInstructions:   "team instructions",
		},
	}

	body, err := ApplyGroupPromptPolicy([]byte(`{"model":"gpt-5","messages":[{"role":"system","content":"user system"},{"role":"user","content":"hi"},{"role":"developer","content":"user developer"}]}`), group, PromptPolicyTargetOpenAIChatCompletions)
	if err != nil {
		t.Fatalf("ApplyGroupPromptPolicy() error = %v", err)
	}
	if got := gjson.GetBytes(body, "messages.#").Int(); got != 2 {
		t.Fatalf("unexpected messages count: %d", got)
	}
	if got := gjson.GetBytes(body, "messages.0.role").String(); got != "developer" {
		t.Fatalf("unexpected first role: %q", got)
	}
	if got := gjson.GetBytes(body, "messages.1.role").String(); got != "user" {
		t.Fatalf("unexpected second role: %q", got)
	}
}

func TestApplyGroupPromptPolicyIncludesLocalSkills(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)
	skillsDir := filepath.Join(dataDir, "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillsDir, "cn.md"), []byte("always answer in Chinese"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	group := &Group{
		PromptPolicy: domain.PromptPolicy{
			Enabled:            true,
			Mode:               domain.PromptPolicyModePrepend,
			SkillIDs:           []string{"cn.md"},
			OpenAIInstructions: "follow coding style",
		},
	}

	body, err := ApplyGroupPromptPolicy([]byte(`{"model":"gpt-5","input":[{"type":"input_text","text":"hi"}]}`), group, PromptPolicyTargetOpenAIResponses)
	if err != nil {
		t.Fatalf("ApplyGroupPromptPolicy() error = %v", err)
	}
	got := gjson.GetBytes(body, "instructions").String()
	want := "[Skill: cn]\nalways answer in Chinese\n\nfollow coding style"
	if got != want {
		t.Fatalf("unexpected instructions: %q", got)
	}
}
