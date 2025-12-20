#!/usr/bin/env python3
"""Progress tracking module for harness.

Provides utilities for reading and writing to claude-progress.txt

Security:
- Validates work_dir to prevent path traversal attacks
- Uses safe path joining for file operations
"""

import os
from datetime import datetime
from pathlib import Path
from typing import Optional

try:
    from core.validation import validate_work_dir, safe_join
except ImportError:
    # Fallback validation if module not available
    def validate_work_dir(work_dir):
        if work_dir and os.path.isdir(work_dir):
            return True, None
        return False, "Invalid directory"
    def safe_join(base, *paths):
        return Path(base).joinpath(*paths)

PROGRESS_FILE = "claude-progress.txt"


def get_progress_path(work_dir: Optional[str] = None) -> Optional[Path]:
    """Get path to progress file.

    Args:
        work_dir: Working directory (uses cwd if None)

    Returns:
        Path to progress file, or None if work_dir is invalid
    """
    if work_dir is None:
        work_dir = os.getcwd()

    # Validate working directory
    is_valid, error = validate_work_dir(work_dir)
    if not is_valid:
        return None

    # Use safe path joining
    return safe_join(work_dir, PROGRESS_FILE)


def read_progress(work_dir=None) -> str:
    """Read the entire progress file."""
    path = get_progress_path(work_dir)
    if path is None:
        return ""
    if path.exists():
        return path.read_text()
    return ""


def append_progress(message: str, work_dir=None, include_timestamp: bool = True) -> Optional[str]:
    """Append a message to the progress file.

    Returns:
        The entry written, or None if path validation failed
    """
    path = get_progress_path(work_dir)
    if path is None:
        return None

    if include_timestamp:
        timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        entry = f"[{timestamp}] {message}\n"
    else:
        entry = f"{message}\n"

    with open(path, 'a') as f:
        f.write(entry)

    return entry


def log_session_start(work_dir=None):
    """Log session start."""
    append_progress("=== SESSION STARTED ===", work_dir)


def log_session_end(work_dir=None):
    """Log session end."""
    append_progress("=== SESSION ENDED ===", work_dir)


def log_task_start(task_name, work_dir=None):
    """Log starting a task."""
    append_progress(f"STARTED: {task_name}", work_dir)


def log_task_complete(task_name, work_dir=None):
    """Log completing a task."""
    append_progress(f"COMPLETED: {task_name}", work_dir)


def log_checkpoint(commit_hash, message, work_dir=None):
    """Log a git checkpoint."""
    append_progress(f"CHECKPOINT [{commit_hash[:8]}]: {message}", work_dir)


def log_note(note, work_dir=None):
    """Log a general note."""
    append_progress(f"NOTE: {note}", work_dir)


def log_blocker(blocker, work_dir=None):
    """Log a blocker or issue."""
    append_progress(f"BLOCKER: {blocker}", work_dir)


def initialize_progress_file(project_name=None, work_dir=None) -> bool:
    """Initialize a new progress file.

    Returns:
        True if file was created, False if already exists or validation failed
    """
    path = get_progress_path(work_dir)
    if path is None:
        return False  # Validation failed

    if path.exists():
        return False  # Already exists

    timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    project = project_name or Path(work_dir or os.getcwd()).name

    content = f"""# Claude Agent Progress Log
# Project: {project}
# Created: {timestamp}
#
# This file tracks progress across Claude Code sessions.
# Each session reads this file to understand prior work.
# Update this file as you complete tasks.
#
# Format:
# [timestamp] ACTION: description
#   - STARTED: Beginning a task
#   - COMPLETED: Finishing a task
#   - CHECKPOINT: Git commit created
#   - NOTE: General observations
#   - BLOCKER: Issues preventing progress
#
# ============================================

[{timestamp}] INITIALIZED: Progress tracking enabled for {project}

"""
    path.write_text(content)
    return True
