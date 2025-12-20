#!/usr/bin/env python3
"""PreToolUse hook for FIC verification gates.

This hook enforces FIC verification gates (research → planning → implementation)
for file modification operations (Edit, Write).

Gate behavior by strictness mode:
- relaxed: No validation, all operations allowed
- standard: Warn on gate violations, allow operation
- strict: Block operations that violate gates
"""

import os
import sys
import json
from pathlib import Path

# Use shared imports module for consistent fallbacks
try:
    from shared_imports import (
        load_config, is_relaxed_mode, is_harness_initialized,
        Gate, GateAction, check_gate, format_gate_message,
        FIC_GATES_AVAILABLE
    )
except ImportError:
    # Fallback: Add plugin root and try direct imports
    PLUGIN_ROOT = os.environ.get('CLAUDE_PLUGIN_ROOT', '')
    if PLUGIN_ROOT:
        sys.path.insert(0, PLUGIN_ROOT)

    try:
        from core.config import load_config, is_relaxed_mode, is_harness_initialized
    except ImportError:
        def load_config(work_dir=None):
            return {"fic_enabled": True}
        def is_relaxed_mode(work_dir=None):
            return False
        def is_harness_initialized(work_dir=None):
            return False

    try:
        from core.verification_gates import Gate, GateAction, check_gate, format_gate_message
        FIC_GATES_AVAILABLE = True
    except ImportError:
        FIC_GATES_AVAILABLE = False
        class Gate:
            ALLOW_EDIT = "allow_edit"
            ALLOW_WRITE = "allow_write"
        class GateAction:
            ALLOW = "allow"
            WARN = "warn"
            BLOCK = "block"
        def check_gate(gate, work_dir=None, **kwargs):
            class Result:
                action = GateAction.ALLOW
                reason = ""
                suggestions = []
            return Result()
        def format_gate_message(result):
            return ""


def check_fic_gates(tool_name: str, tool_input: dict, work_dir: str, config: dict) -> tuple:
    """
    Check FIC verification gates for Edit/Write operations.

    Returns: (action, message)
    - action: 'allow', 'warn', 'block'
    - message: Message to display (None if allow)
    """
    if not FIC_GATES_AVAILABLE:
        return 'allow', None

    if not config.get('fic_enabled', True):
        return 'allow', None

    # Only check gates for file modifications
    if tool_name not in ['Edit', 'Write']:
        return 'allow', None

    file_path = tool_input.get('file_path', '')

    # Determine which gate to check
    if tool_name == 'Edit':
        gate = Gate.ALLOW_EDIT
    else:
        gate = Gate.ALLOW_WRITE

    # Check the gate
    result = check_gate(gate, work_dir, file_path=file_path)

    if result.action == GateAction.BLOCK:
        return 'block', format_gate_message(result)
    elif result.action == GateAction.WARN:
        return 'warn', format_gate_message(result)
    else:
        return 'allow', None


def main():
    """Main entry point for PreToolUse hook."""
    try:
        # Read input from stdin
        # Handle empty or invalid stdin gracefully
        try:
            stdin_content = sys.stdin.read()
            input_data = json.loads(stdin_content) if stdin_content.strip() else {}
        except (json.JSONDecodeError, ValueError):
            input_data = {}

        tool_name = input_data.get('tool_name', '')
        tool_input = input_data.get('tool_input', {})
        work_dir = os.environ.get('CLAUDE_WORKING_DIRECTORY', os.getcwd())

        # Check if harness is initialized
        if not is_harness_initialized(work_dir):
            print(json.dumps({}))
            sys.exit(0)

        # Load config
        config = load_config(work_dir)

        # Skip all validation in relaxed mode
        if is_relaxed_mode(work_dir):
            print(json.dumps({}))
            sys.exit(0)

        result = {}
        messages = []

        # ========================================
        # FIC Verification Gates
        # ========================================
        fic_action, fic_message = check_fic_gates(tool_name, tool_input, work_dir, config)

        if fic_action == 'block':
            # FIC gate blocks the operation
            result['hookSpecificOutput'] = {
                'permissionDecision': 'deny'
            }
            messages.append(fic_message)
            messages.append("\n[FIC Gate: Operation blocked. Complete prior phase first.]")
            result['systemMessage'] = '\n'.join(messages)
            print(json.dumps(result))
            sys.exit(0)
        elif fic_action == 'warn' and fic_message:
            messages.append(fic_message)

        # Output result
        if messages:
            result['systemMessage'] = '\n'.join(messages)

        print(json.dumps(result))

    except Exception as e:
        # Non-blocking error handling
        error_msg = {"systemMessage": f"[Harness] PreToolUse hook error: {str(e)}"}
        print(json.dumps(error_msg))

    finally:
        sys.exit(0)


if __name__ == '__main__':
    main()
