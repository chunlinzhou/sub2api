package service

import (
	"encoding/json"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/domain"
)

type PromptPolicyTarget string

const (
	PromptPolicyTargetAnthropic             PromptPolicyTarget = "anthropic"
	PromptPolicyTargetOpenAIResponses       PromptPolicyTarget = "openai_responses"
	PromptPolicyTargetOpenAIChatCompletions PromptPolicyTarget = "openai_chat_completions"
	PromptPolicyTargetGemini                PromptPolicyTarget = "gemini"
)

func ApplyGroupPromptPolicy(body []byte, group *Group, target PromptPolicyTarget) ([]byte, error) {
	if group == nil {
		return body, nil
	}
	policy := domain.NormalizePromptPolicy(group.PromptPolicy)
	if !policy.Enabled {
		return body, nil
	}

	text := promptPolicyTextForTarget(policy, target)
	if text == "" {
		return body, nil
	}

	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return body, err
	}
	root, ok := payload.(map[string]any)
	if !ok {
		return body, nil
	}

	effectiveMode := policy.Mode
	if policy.SkillEnforcementMode == domain.PromptPolicyEnforcementStrict {
		applyStrictPromptPolicy(root, target)
		if target == PromptPolicyTargetOpenAIChatCompletions {
			effectiveMode = domain.PromptPolicyModePrepend
		}
	}

	changed := false
	switch target {
	case PromptPolicyTargetAnthropic:
		changed = applyAnthropicPromptPolicy(root, effectiveMode, text)
	case PromptPolicyTargetOpenAIResponses:
		changed = applyOpenAIResponsesPromptPolicy(root, effectiveMode, text)
	case PromptPolicyTargetOpenAIChatCompletions:
		changed = applyOpenAIChatCompletionsPromptPolicy(root, effectiveMode, text)
	case PromptPolicyTargetGemini:
		changed = applyGeminiPromptPolicy(root, effectiveMode, text)
	}
	if !changed {
		return body, nil
	}
	return json.Marshal(root)
}

func promptPolicyTextForTarget(policy domain.PromptPolicy, target PromptPolicyTarget) string {
	parts := make([]string, 0, 2)
	if skillText := CompileLocalSkillText(policy.SkillIDs); skillText != "" {
		parts = append(parts, skillText)
	}
	switch target {
	case PromptPolicyTargetAnthropic:
		if policy.AnthropicSystemMessage != "" {
			parts = append(parts, policy.AnthropicSystemMessage)
		}
	case PromptPolicyTargetOpenAIResponses, PromptPolicyTargetOpenAIChatCompletions:
		if policy.OpenAIInstructions != "" {
			parts = append(parts, policy.OpenAIInstructions)
		}
	case PromptPolicyTargetGemini:
		if policy.GeminiSystemInstruction != "" {
			parts = append(parts, policy.GeminiSystemInstruction)
		}
	}
	return strings.Join(parts, "\n\n")
}

func applyStrictPromptPolicy(root map[string]any, target PromptPolicyTarget) {
	switch target {
	case PromptPolicyTargetAnthropic:
		delete(root, "system")
	case PromptPolicyTargetOpenAIResponses:
		delete(root, "instructions")
	case PromptPolicyTargetOpenAIChatCompletions:
		root["messages"] = removeOpenAIChatSystemAndDeveloperMessages(root["messages"])
	case PromptPolicyTargetGemini:
		delete(root, "systemInstruction")
	}
}

func removeOpenAIChatSystemAndDeveloperMessages(raw any) []any {
	messages, ok := raw.([]any)
	if !ok {
		return nil
	}
	filtered := make([]any, 0, len(messages))
	for _, item := range messages {
		msg, ok := item.(map[string]any)
		if !ok {
			filtered = append(filtered, item)
			continue
		}
		role, _ := msg["role"].(string)
		if role == "system" || role == "developer" {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func applyAnthropicPromptPolicy(root map[string]any, mode string, text string) bool {
	current, exists := root["system"]
	if mode == domain.PromptPolicyModeReplaceIfEmpty {
		if anthropicSystemHasContent(current) {
			return false
		}
		root["system"] = text
		return true
	}

	switch v := current.(type) {
	case nil:
		root["system"] = text
		return true
	case string:
		root["system"] = mergePromptText(v, text, mode)
		return root["system"] != v
	case []any:
		block := map[string]any{"type": "text", "text": text}
		if mode == domain.PromptPolicyModeAppend {
			root["system"] = append(v, block)
		} else {
			root["system"] = append([]any{block}, v...)
		}
		return true
	default:
		if !exists {
			root["system"] = text
			return true
		}
		return false
	}
}

func anthropicSystemHasContent(system any) bool {
	switch v := system.(type) {
	case string:
		return strings.TrimSpace(v) != ""
	case []any:
		for _, item := range v {
			block, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if text, ok := block["text"].(string); ok && strings.TrimSpace(text) != "" {
				return true
			}
		}
	}
	return false
}

func applyOpenAIResponsesPromptPolicy(root map[string]any, mode string, text string) bool {
	current, _ := root["instructions"].(string)
	if mode == domain.PromptPolicyModeReplaceIfEmpty {
		if strings.TrimSpace(current) != "" {
			return false
		}
		root["instructions"] = text
		return true
	}
	root["instructions"] = mergePromptText(current, text, mode)
	return root["instructions"] != current
}

func applyOpenAIChatCompletionsPromptPolicy(root map[string]any, mode string, text string) bool {
	current, ok := root["messages"].([]any)
	if !ok {
		current = nil
	}
	if mode == domain.PromptPolicyModeReplaceIfEmpty && openAIChatMessagesHaveSystemOrDeveloper(current) {
		return false
	}
	msg := map[string]any{
		"role":    "developer",
		"content": text,
	}
	if mode == domain.PromptPolicyModeAppend {
		root["messages"] = append(current, msg)
	} else {
		root["messages"] = append([]any{msg}, current...)
	}
	return true
}

func openAIChatMessagesHaveSystemOrDeveloper(messages []any) bool {
	for _, item := range messages {
		msg, ok := item.(map[string]any)
		if !ok {
			continue
		}
		role, _ := msg["role"].(string)
		if role != "system" && role != "developer" {
			continue
		}
		if contentHasText(msg["content"]) {
			return true
		}
	}
	return false
}

func contentHasText(content any) bool {
	switch v := content.(type) {
	case string:
		return strings.TrimSpace(v) != ""
	case []any:
		for _, item := range v {
			part, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if text, ok := part["text"].(string); ok && strings.TrimSpace(text) != "" {
				return true
			}
		}
	}
	return false
}

func applyGeminiPromptPolicy(root map[string]any, mode string, text string) bool {
	currentInstruction, _ := root["systemInstruction"].(map[string]any)
	currentParts, _ := currentInstruction["parts"].([]any)
	if mode == domain.PromptPolicyModeReplaceIfEmpty {
		if geminiSystemInstructionHasContent(currentParts) {
			return false
		}
		root["systemInstruction"] = map[string]any{
			"parts": []any{map[string]any{"text": text}},
		}
		return true
	}

	part := map[string]any{"text": text}
	if len(currentParts) == 0 {
		root["systemInstruction"] = map[string]any{
			"parts": []any{part},
		}
		return true
	}
	if mode == domain.PromptPolicyModeAppend {
		currentInstruction["parts"] = append(currentParts, part)
	} else {
		currentInstruction["parts"] = append([]any{part}, currentParts...)
	}
	root["systemInstruction"] = currentInstruction
	return true
}

func geminiSystemInstructionHasContent(parts []any) bool {
	for _, item := range parts {
		part, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if text, ok := part["text"].(string); ok && strings.TrimSpace(text) != "" {
			return true
		}
	}
	return false
}

func mergePromptText(current string, text string, mode string) string {
	current = strings.TrimSpace(current)
	text = strings.TrimSpace(text)
	if current == "" {
		return text
	}
	if text == "" {
		return current
	}
	if mode == domain.PromptPolicyModeAppend {
		return current + "\n\n" + text
	}
	return text + "\n\n" + current
}
