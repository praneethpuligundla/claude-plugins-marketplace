package protocol

import (
	"encoding/json"
	"testing"
)

func TestHookInputGetters(t *testing.T) {
	t.Run("GetFilePath", func(t *testing.T) {
		tests := []struct {
			name  string
			input HookInput
			want  string
		}{
			{
				name:  "nil tool input",
				input: HookInput{ToolInput: nil},
				want:  "",
			},
			{
				name:  "no file_path key",
				input: HookInput{ToolInput: map[string]interface{}{"other": "value"}},
				want:  "",
			},
			{
				name:  "file_path not string",
				input: HookInput{ToolInput: map[string]interface{}{"file_path": 123}},
				want:  "",
			},
			{
				name:  "valid file_path",
				input: HookInput{ToolInput: map[string]interface{}{"file_path": "/path/to/file.txt"}},
				want:  "/path/to/file.txt",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.input.GetFilePath(); got != tt.want {
					t.Errorf("GetFilePath() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("GetCommand", func(t *testing.T) {
		tests := []struct {
			name  string
			input HookInput
			want  string
		}{
			{
				name:  "nil tool input",
				input: HookInput{ToolInput: nil},
				want:  "",
			},
			{
				name:  "no command key",
				input: HookInput{ToolInput: map[string]interface{}{"other": "value"}},
				want:  "",
			},
			{
				name:  "valid command",
				input: HookInput{ToolInput: map[string]interface{}{"command": "ls -la"}},
				want:  "ls -la",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.input.GetCommand(); got != tt.want {
					t.Errorf("GetCommand() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("GetPrompt", func(t *testing.T) {
		tests := []struct {
			name  string
			input HookInput
			want  string
		}{
			{
				name:  "top-level prompt field",
				input: HookInput{Prompt: "research this topic"},
				want:  "research this topic",
			},
			{
				name:  "prompt in tool_input (fallback)",
				input: HookInput{ToolInput: map[string]interface{}{"prompt": "legacy prompt"}},
				want:  "legacy prompt",
			},
			{
				name:  "top-level takes precedence",
				input: HookInput{Prompt: "top-level", ToolInput: map[string]interface{}{"prompt": "in-tool"}},
				want:  "top-level",
			},
			{
				name:  "no prompt anywhere",
				input: HookInput{ToolInput: map[string]interface{}{"other": "value"}},
				want:  "",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.input.GetPrompt(); got != tt.want {
					t.Errorf("GetPrompt() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("GetSubagentType", func(t *testing.T) {
		tests := []struct {
			name  string
			input HookInput
			want  string
		}{
			{
				name:  "nil tool input",
				input: HookInput{ToolInput: nil},
				want:  "",
			},
			{
				name:  "valid subagent_type",
				input: HookInput{ToolInput: map[string]interface{}{"subagent_type": "fic-researcher"}},
				want:  "fic-researcher",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.input.GetSubagentType(); got != tt.want {
					t.Errorf("GetSubagentType() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("GetDescription", func(t *testing.T) {
		input := HookInput{ToolInput: map[string]interface{}{"description": "task description"}}
		if got := input.GetDescription(); got != "task description" {
			t.Errorf("GetDescription() = %v, want 'task description'", got)
		}
	})

	t.Run("GetOutput", func(t *testing.T) {
		input := HookInput{ToolInput: map[string]interface{}{"output": "agent output"}}
		if got := input.GetOutput(); got != "agent output" {
			t.Errorf("GetOutput() = %v, want 'agent output'", got)
		}
	})

	t.Run("GetStopReason", func(t *testing.T) {
		tests := []struct {
			name  string
			input HookInput
			want  string
		}{
			{
				name:  "stopReason field",
				input: HookInput{ToolInput: map[string]interface{}{"stopReason": "end_turn"}},
				want:  "end_turn",
			},
			{
				name:  "reason field fallback",
				input: HookInput{ToolInput: map[string]interface{}{"reason": "stop_sequence"}},
				want:  "stop_sequence",
			},
			{
				name:  "stopReason takes precedence",
				input: HookInput{ToolInput: map[string]interface{}{"stopReason": "end_turn", "reason": "other"}},
				want:  "end_turn",
			},
			{
				name:  "no stop reason",
				input: HookInput{ToolInput: map[string]interface{}{}},
				want:  "",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.input.GetStopReason(); got != tt.want {
					t.Errorf("GetStopReason() = %v, want %v", got, tt.want)
				}
			})
		}
	})
}

func TestHookOutputJSON(t *testing.T) {
	t.Run("basic output marshaling", func(t *testing.T) {
		output := &HookOutput{
			SystemMessage: "Test message",
		}

		data, err := json.Marshal(output)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if msg, ok := parsed["systemMessage"].(string); !ok || msg != "Test message" {
			t.Errorf("systemMessage = %v, want 'Test message'", parsed["systemMessage"])
		}
	})

	t.Run("permission deny output", func(t *testing.T) {
		output := &HookOutput{
			SystemMessage: "Access denied",
			HookSpecificOutput: &HookSpecificOutput{
				PermissionDecision: PermissionDeny,
			},
		}

		data, err := json.Marshal(output)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		hookOutput, ok := parsed["hookSpecificOutput"].(map[string]interface{})
		if !ok {
			t.Fatal("hookSpecificOutput not found or wrong type")
		}

		if decision := hookOutput["permissionDecision"]; decision != PermissionDeny {
			t.Errorf("permissionDecision = %v, want %v", decision, PermissionDeny)
		}
	})

	t.Run("empty output omits fields", func(t *testing.T) {
		output := &HookOutput{}

		data, err := json.Marshal(output)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		// Should be just "{}"
		if string(data) != "{}" {
			t.Errorf("Empty output = %s, want {}", string(data))
		}
	})
}

func TestHookInputJSONParsing(t *testing.T) {
	t.Run("parse full input", func(t *testing.T) {
		jsonData := `{
			"session_id": "test-session-123",
			"tool_name": "Edit",
			"tool_input": {
				"file_path": "/path/to/file.txt",
				"old_string": "foo",
				"new_string": "bar"
			}
		}`

		var input HookInput
		if err := json.Unmarshal([]byte(jsonData), &input); err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		if input.SessionID != "test-session-123" {
			t.Errorf("SessionID = %v, want 'test-session-123'", input.SessionID)
		}
		if input.ToolName != "Edit" {
			t.Errorf("ToolName = %v, want 'Edit'", input.ToolName)
		}
		if input.GetFilePath() != "/path/to/file.txt" {
			t.Errorf("GetFilePath() = %v, want '/path/to/file.txt'", input.GetFilePath())
		}
	})

	t.Run("parse UserPromptSubmit format", func(t *testing.T) {
		jsonData := `{
			"session_id": "session-456",
			"prompt": "What is the architecture of this codebase?"
		}`

		var input HookInput
		if err := json.Unmarshal([]byte(jsonData), &input); err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		if input.GetPrompt() != "What is the architecture of this codebase?" {
			t.Errorf("GetPrompt() = %v, want prompt", input.GetPrompt())
		}
	})

	t.Run("parse empty input", func(t *testing.T) {
		jsonData := `{}`

		var input HookInput
		if err := json.Unmarshal([]byte(jsonData), &input); err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		if input.SessionID != "" {
			t.Errorf("SessionID = %v, want empty", input.SessionID)
		}
		if input.GetFilePath() != "" {
			t.Errorf("GetFilePath() = %v, want empty", input.GetFilePath())
		}
	})
}

func TestMaxInputSize(t *testing.T) {
	// Verify the constant is set to a reasonable limit
	if MaxInputSize != 10*1024*1024 {
		t.Errorf("MaxInputSize = %d, want 10MB (10485760)", MaxInputSize)
	}
}

func TestPermissionDecisionConstants(t *testing.T) {
	if PermissionAllow != "allow" {
		t.Errorf("PermissionAllow = %v, want 'allow'", PermissionAllow)
	}
	if PermissionDeny != "deny" {
		t.Errorf("PermissionDeny = %v, want 'deny'", PermissionDeny)
	}
}
