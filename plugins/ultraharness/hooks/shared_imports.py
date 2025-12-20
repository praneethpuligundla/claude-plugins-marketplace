#!/usr/bin/env python3
"""Shared imports and fallbacks for all hooks.

This module centralizes the import logic and fallback definitions
that are common across all hooks. Using this module:
1. Reduces code duplication (~200 lines saved)
2. Ensures consistent fallback behavior
3. Makes maintenance easier

Usage in hooks:
    from shared_imports import (
        load_config, is_strict_mode, is_relaxed_mode, is_harness_initialized,
        classify_change, should_auto_log, should_suggest_checkpoint, ChangeLevel,
        append_progress, load_features, get_next_features,
        validate_session_id, validate_work_dir,
        load_context_state, save_context_state, add_context_entry, get_context_summary,
        InformationClass, FIC_AVAILABLE, CONTEXT_INTELLIGENCE_AVAILABLE,
        ArtifactType, get_latest_artifact, load_artifact,
        Gate, GateAction, check_gate, format_gate_message, FIC_GATES_AVAILABLE,
        TestResult, run_tests, get_test_summary_string
    )
"""

import os
import sys
from pathlib import Path

# Add plugin root to path for imports
PLUGIN_ROOT = os.environ.get('CLAUDE_PLUGIN_ROOT', '')
if PLUGIN_ROOT:
    sys.path.insert(0, PLUGIN_ROOT)


# ============================================
# Config Module
# ============================================
try:
    from core.config import (
        load_config,
        is_strict_mode,
        is_relaxed_mode,
        is_standard_mode,
        is_harness_initialized,
        get_setting,
        set_setting,
        get_config_path,
        clear_config_cache
    )
    CONFIG_AVAILABLE = True
except ImportError:
    CONFIG_AVAILABLE = False

    def load_config(work_dir=None):
        return {
            "strictness": "standard",
            "auto_progress_logging": True,
            "auto_checkpoint_suggestions": True,
            "fic_enabled": True,
            "fic_context_tracking": True,
            "fic_auto_delegate_research": True,
            "feature_enforcement": True,
            "init_script_execution": True,
            "baseline_tests_on_startup": True
        }

    def is_strict_mode(work_dir=None):
        return False

    def is_relaxed_mode(work_dir=None):
        return False

    def is_standard_mode(work_dir=None):
        return True

    def is_harness_initialized(work_dir=None):
        if work_dir is None:
            work_dir = os.environ.get('CLAUDE_WORKING_DIRECTORY', os.getcwd())
        marker_path = Path(work_dir) / '.claude' / '.claude-harness-initialized'
        return marker_path.exists()

    def get_setting(key, work_dir=None):
        return None

    def set_setting(key, value, work_dir=None):
        return False

    def get_config_path(work_dir=None):
        if work_dir is None:
            work_dir = os.environ.get('CLAUDE_WORKING_DIRECTORY', os.getcwd())
        return Path(work_dir) / '.claude' / 'claude-harness.json'

    def clear_config_cache(work_dir=None):
        pass


# ============================================
# Validation Module
# ============================================
try:
    from core.validation import (
        validate_path,
        validate_work_dir,
        validate_session_id,
        safe_join,
        sanitize_filename
    )
    VALIDATION_AVAILABLE = True
except ImportError:
    VALIDATION_AVAILABLE = False

    def validate_path(path, work_dir=None, must_exist=False, allow_absolute=True):
        if path and isinstance(path, str):
            return True, None, Path(path)
        return False, "Invalid path", None

    def validate_work_dir(work_dir):
        if work_dir and os.path.isdir(work_dir):
            return True, None
        return False, "Invalid directory"

    def validate_session_id(session_id):
        if session_id and len(session_id) <= 128 and '..' not in session_id:
            return True, None
        return False, "Invalid session ID"

    def safe_join(base, *paths):
        return Path(base).joinpath(*paths)

    def sanitize_filename(filename, max_length=255):
        return filename[:max_length] if filename else "unnamed"


# ============================================
# Change Detector Module
# ============================================
try:
    from core.change_detector import (
        classify_change,
        should_auto_log,
        should_suggest_checkpoint,
        ChangeLevel
    )
    CHANGE_DETECTOR_AVAILABLE = True
except ImportError:
    CHANGE_DETECTOR_AVAILABLE = False

    class ChangeLevel:
        TRIVIAL = "trivial"
        SIGNIFICANT = "significant"
        MAJOR = "major"

    def classify_change(tool_name, tool_input, tool_result=None):
        return (ChangeLevel.TRIVIAL, "fallback")

    def should_auto_log(level):
        return False

    def should_suggest_checkpoint(level):
        return False


# ============================================
# Progress Module
# ============================================
try:
    from core.progress import (
        append_progress,
        read_progress,
        log_session_start,
        log_session_end,
        log_checkpoint,
        initialize_progress_file
    )
    PROGRESS_AVAILABLE = True
except ImportError:
    PROGRESS_AVAILABLE = False

    def append_progress(msg, work_dir=None, include_timestamp=True):
        pass

    def read_progress(work_dir=None):
        return ""

    def log_session_start(work_dir=None):
        pass

    def log_session_end(work_dir=None):
        pass

    def log_checkpoint(commit_hash, message, work_dir=None):
        pass

    def initialize_progress_file(project_name=None, work_dir=None):
        return False


# ============================================
# Features Module
# ============================================
try:
    from core.features import (
        load_features,
        get_next_features,
        save_features,
        get_feature_by_id
    )
    FEATURES_AVAILABLE = True
except ImportError:
    FEATURES_AVAILABLE = False

    def load_features(work_dir=None):
        return {"features": []}

    def get_next_features(count=5, work_dir=None):
        return []

    def save_features(features_data, work_dir=None):
        return False

    def get_feature_by_id(feature_id, work_dir=None):
        return None


# ============================================
# Context Intelligence Module
# ============================================
try:
    from core.context_intelligence import (
        load_context_state,
        save_context_state,
        add_context_entry,
        get_context_summary,
        extract_essential_context,
        InformationClass,
        estimate_tokens
    )
    CONTEXT_INTELLIGENCE_AVAILABLE = True
    FIC_AVAILABLE = True
except ImportError:
    CONTEXT_INTELLIGENCE_AVAILABLE = False
    FIC_AVAILABLE = False

    class InformationClass:
        ESSENTIAL = "essential"
        HELPFUL = "helpful"
        NOISE = "noise"

    def load_context_state(session_id, work_dir=None):
        return None

    def save_context_state(state, work_dir=None):
        return False

    def add_context_entry(state, tool_name, tool_input, tool_result):
        return state, None

    def get_context_summary(state):
        return ""

    def extract_essential_context(state):
        return {}

    def estimate_tokens(content, content_type=None):
        return int(len(content) * 0.25)


# ============================================
# Artifacts Module
# ============================================
try:
    from core.artifacts import (
        ArtifactType,
        get_latest_artifact,
        load_artifact,
        save_artifact
    )
    ARTIFACTS_AVAILABLE = True
except ImportError:
    ARTIFACTS_AVAILABLE = False

    class ArtifactType:
        RESEARCH = "research"
        PLAN = "plan"
        IMPLEMENTATION = "implementation"

    def get_latest_artifact(artifact_type, work_dir=None):
        return None

    def load_artifact(artifact_type, artifact_id, work_dir=None):
        return None

    def save_artifact(artifact, work_dir=None):
        return False


# ============================================
# Verification Gates Module
# ============================================
try:
    from core.verification_gates import (
        Gate,
        GateAction,
        check_gate,
        format_gate_message
    )
    FIC_GATES_AVAILABLE = True
except ImportError:
    FIC_GATES_AVAILABLE = False

    class Gate:
        ALLOW_EDIT = "allow_edit"
        ALLOW_WRITE = "allow_write"
        ALLOW_BASH = "allow_bash"
        RESEARCH_COMPLETE = "research_complete"
        PLAN_VALIDATED = "plan_validated"

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


# ============================================
# Test Runner Module
# ============================================
try:
    from core.test_runner import (
        run_tests,
        get_test_summary_string,
        TestResult,
        detect_project_type
    )
    TEST_RUNNER_AVAILABLE = True
except ImportError:
    TEST_RUNNER_AVAILABLE = False

    class TestResult:
        NOT_RUN = "not_run"
        PASSED = "passed"
        FAILED = "failed"
        SKIPPED = "skipped"
        ERROR = "error"

    def run_tests(work_dir, timeout=120, config=None):
        class Summary:
            result = TestResult.NOT_RUN
            raw_output = "Test runner not available"
            total = 0
            passed = 0
            failed = 0
            skipped = 0
            errors = 0
        return Summary()

    def get_test_summary_string(summary):
        return "Tests not run"

    def detect_project_type(work_dir):
        return None


# ============================================
# Browser Automation Module
# ============================================
try:
    from core.browser_automation import (
        take_screenshot,
        verify_element,
        detect_browser_tool,
        BrowserResult
    )
    BROWSER_AVAILABLE = True
except ImportError:
    BROWSER_AVAILABLE = False

    class BrowserResult:
        def __init__(self, success=False, error=None):
            self.success = success
            self.error = error

    def take_screenshot(*args, **kwargs):
        return BrowserResult(success=False, error="Browser automation not available")

    def verify_element(*args, **kwargs):
        return BrowserResult(success=False, error="Browser automation not available")

    def detect_browser_tool(work_dir):
        return None


# ============================================
# Utility Functions
# ============================================
def get_working_directory():
    """Get the current working directory from environment or fallback to cwd."""
    return os.environ.get('CLAUDE_WORKING_DIRECTORY', os.getcwd())


def safe_json_loads(content):
    """Safely parse JSON, returning empty dict on failure."""
    try:
        return json.loads(content) if content and content.strip() else {}
    except (json.JSONDecodeError, ValueError, TypeError):
        return {}


# Need to import json for safe_json_loads
import json
