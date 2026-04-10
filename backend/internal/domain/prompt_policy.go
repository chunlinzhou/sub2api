package domain

import (
	"path/filepath"
	"strings"
)

const (
	PromptPolicyModePrepend        = "prepend"
	PromptPolicyModeAppend         = "append"
	PromptPolicyModeReplaceIfEmpty = "replace_if_empty"
	PromptPolicyEnforcementSoft    = "soft"
	PromptPolicyEnforcementStrict  = "strict"
)

// PromptPolicy defines the group-level prompt template injection policy.
type PromptPolicy struct {
	Enabled              bool     `json:"enabled"`
	Mode                 string   `json:"mode,omitempty"`
	SkillIDs             []string `json:"skill_ids,omitempty"`
	SkillEnforcementMode string   `json:"skill_enforcement_mode,omitempty"`

	AnthropicSystemMessage  string `json:"anthropic_system_message,omitempty"`
	OpenAIInstructions      string `json:"openai_instructions,omitempty"`
	GeminiSystemInstruction string `json:"gemini_system_instruction,omitempty"`
	Notes                   string `json:"notes,omitempty"`
}

func NormalizePromptPolicy(policy PromptPolicy) PromptPolicy {
	policy.Mode = normalizePromptPolicyMode(policy.Mode)
	policy.SkillIDs = normalizePromptPolicySkillIDs(policy.SkillIDs)
	policy.SkillEnforcementMode = normalizePromptPolicyEnforcementMode(policy.SkillEnforcementMode)
	policy.AnthropicSystemMessage = strings.TrimSpace(policy.AnthropicSystemMessage)
	policy.OpenAIInstructions = strings.TrimSpace(policy.OpenAIInstructions)
	policy.GeminiSystemInstruction = strings.TrimSpace(policy.GeminiSystemInstruction)
	policy.Notes = strings.TrimSpace(policy.Notes)
	return policy
}

func normalizePromptPolicyMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case PromptPolicyModeAppend:
		return PromptPolicyModeAppend
	case PromptPolicyModeReplaceIfEmpty:
		return PromptPolicyModeReplaceIfEmpty
	default:
		return PromptPolicyModePrepend
	}
}

func normalizePromptPolicyEnforcementMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case PromptPolicyEnforcementStrict:
		return PromptPolicyEnforcementStrict
	default:
		return PromptPolicyEnforcementSoft
	}
}

func normalizePromptPolicySkillIDs(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if filepath.Base(id) != id || strings.ContainsAny(id, `/\`) {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
