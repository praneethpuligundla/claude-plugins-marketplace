#!/usr/bin/env python3
"""PreToolUse hook for enforcing one-feature-at-a-time discipline.

Validates that file edits relate to the current in-progress feature.
Non-blocking in standard mode, blocking in strict mode.
"""

import os
import sys
import json
from pathlib import Path

# Add plugin root to path for imports
PLUGIN_ROOT = os.environ.get('CLAUDE_PLUGIN_ROOT', '')
if PLUGIN_ROOT:
    sys.path.insert(0, PLUGIN_ROOT)

try:
    from core.config import load_config, is_strict_mode, is_relaxed_mode, is_harness_initialized
    from core.features import load_features, get_next_features
except ImportError:
    # Fallback if imports fail
    def load_config(work_dir=None):
        return {"feature_enforcement": True}
    def is_strict_mode(work_dir=None):
        return False
    def is_relaxed_mode(work_dir=None):
        return False
    def is_harness_initialized(work_dir=None):
        return False
    def load_features(work_dir=None):
        return {"features": []}
    def get_next_features(count=5, work_dir=None):
        return []


def get_current_feature(work_dir: str) -> dict:
    """Get the currently in-progress feature."""
    try:
        features_data = load_features(work_dir)
        for feature in features_data.get('features', []):
            if feature.get('status') == 'in_progress':
                return feature
    except Exception:
        pass
    return None


def has_features_defined(work_dir: str) -> bool:
    """Check if any features are defined."""
    try:
        features_data = load_features(work_dir)
        return len(features_data.get('features', [])) > 0
    except Exception:
        return False


def validate_feature_focus(tool_name: str, tool_input: dict, work_dir: str) -> tuple:
    """
    Validate that work is focused on the current feature.

    Returns: (is_valid, message)
    """
    # Only validate file operations
    if tool_name not in ['Edit', 'Write']:
        return True, None

    # Check if features are defined
    if not has_features_defined(work_dir):
        return True, None  # No features defined, allow all

    current = get_current_feature(work_dir)

    if not current:
        # No feature in progress - prompt to select one
        try:
            next_features = get_next_features(3, work_dir)
            if next_features:
                feature_list = '\n'.join([
                    f"  {f['id']}. {f['name']}"
                    for f in next_features
                ])
                return False, (
                    f"[Harness] No feature currently in progress.\n"
                    f"Consider starting one before making changes:\n{feature_list}\n"
                    f"Use `/harness:feature start <id>` to begin."
                )
        except Exception:
            pass

        return False, (
            "[Harness] No feature currently in progress. "
            "Use `/harness:feature start <id>` to begin working on a feature."
        )

    # Feature is in progress, allow the operation
    return True, None


def main():
    """Main entry point for PreToolUse hook."""
    try:
        # Read input from stdin
        input_data = json.load(sys.stdin)

        tool_name = input_data.get('tool_name', '')
        tool_input = input_data.get('tool_input', {})
        work_dir = os.environ.get('CLAUDE_WORKING_DIRECTORY', os.getcwd())

        # Check if harness is initialized
        if not is_harness_initialized(work_dir):
            print(json.dumps({}))
            sys.exit(0)

        # Load config
        config = load_config(work_dir)

        # Skip validation in relaxed mode or if feature enforcement is disabled
        if is_relaxed_mode(work_dir) or not config.get('feature_enforcement', True):
            print(json.dumps({}))
            sys.exit(0)

        # Validate feature focus
        is_valid, message = validate_feature_focus(tool_name, tool_input, work_dir)

        result = {}

        if not is_valid and message:
            if is_strict_mode(work_dir):
                # Block the operation in strict mode
                result['hookSpecificOutput'] = {
                    'permissionDecision': 'deny'
                }
                result['systemMessage'] = message + "\n\n[Strict mode: Operation blocked until a feature is started]"
            else:
                # Just warn in standard mode
                result['systemMessage'] = message

        print(json.dumps(result))

    except Exception as e:
        # Non-blocking error handling
        error_msg = {"systemMessage": f"[Harness] PreToolUse hook error: {str(e)}"}
        print(json.dumps(error_msg))

    finally:
        sys.exit(0)


if __name__ == '__main__':
    main()
