#!/usr/bin/env python3
"""Input validation utilities for harness plugin.

Provides security-focused validation for:
- File paths (preventing traversal attacks, null bytes, etc.)
- Working directories (ensuring operations stay within bounds)
- Command inputs (basic sanitization)

Security considerations:
- Path traversal prevention (../ attacks)
- Null byte injection prevention
- Symlink resolution for security checks
- Bounds checking to ensure operations stay within work_dir
"""

import os
import re
from pathlib import Path
from typing import Tuple, Optional


# Characters that are dangerous in file paths
DANGEROUS_PATH_CHARS = frozenset(['\x00', '\n', '\r'])

# Patterns that indicate path traversal attempts
PATH_TRAVERSAL_PATTERNS = [
    re.compile(r'\.\.[\\/]'),      # ../ or ..\
    re.compile(r'[\\/]\.\.'),       # /.. or \..
    re.compile(r'^\.\.'),           # Starts with ..
]


def validate_path(
    path: str,
    work_dir: str = None,
    must_exist: bool = False,
    allow_absolute: bool = True
) -> Tuple[bool, Optional[str], Optional[Path]]:
    """
    Validate a file path for security.

    Args:
        path: The path to validate
        work_dir: If provided, ensure path is within this directory
        must_exist: If True, path must exist on filesystem
        allow_absolute: If False, reject absolute paths

    Returns:
        (is_valid, error_message, resolved_path)
        - is_valid: True if path is safe
        - error_message: Description of validation failure (None if valid)
        - resolved_path: The resolved Path object (None if invalid)
    """
    if not path:
        return False, "Path is empty", None

    # Check for dangerous characters (null bytes, newlines)
    for char in DANGEROUS_PATH_CHARS:
        if char in path:
            char_name = repr(char)
            return False, f"Path contains dangerous character: {char_name}", None

    # Check for path traversal patterns
    for pattern in PATH_TRAVERSAL_PATTERNS:
        if pattern.search(path):
            return False, "Path contains traversal pattern (../)", None

    # Check absolute path restriction
    if not allow_absolute and os.path.isabs(path):
        return False, "Absolute paths not allowed", None

    try:
        # Convert to Path object
        path_obj = Path(path)

        # If work_dir is specified, ensure path is within bounds
        if work_dir:
            work_path = Path(work_dir).resolve()

            # Resolve the full path (handles relative paths)
            if path_obj.is_absolute():
                resolved = path_obj.resolve()
            else:
                resolved = (work_path / path_obj).resolve()

            # Security check: ensure resolved path is within work_dir
            # Use os.path.commonpath for reliable cross-platform check
            try:
                common = os.path.commonpath([str(work_path), str(resolved)])
                if common != str(work_path):
                    return False, f"Path escapes working directory", None
            except ValueError:
                # Different drives on Windows
                return False, "Path is on different drive than working directory", None
        else:
            resolved = path_obj.resolve() if path_obj.is_absolute() else path_obj

        # Check existence if required
        if must_exist and not resolved.exists():
            return False, f"Path does not exist: {path}", None

        return True, None, resolved

    except (OSError, ValueError) as e:
        return False, f"Invalid path: {str(e)}", None


def validate_work_dir(work_dir: str) -> Tuple[bool, Optional[str]]:
    """
    Validate a working directory.

    Args:
        work_dir: The working directory to validate

    Returns:
        (is_valid, error_message)
    """
    if not work_dir:
        return False, "Working directory is empty"

    # Check for dangerous characters
    for char in DANGEROUS_PATH_CHARS:
        if char in work_dir:
            return False, f"Working directory contains dangerous character"

    try:
        path = Path(work_dir)

        # Must be absolute
        if not path.is_absolute():
            return False, "Working directory must be absolute path"

        # Must exist
        if not path.exists():
            return False, "Working directory does not exist"

        # Must be a directory
        if not path.is_dir():
            return False, "Working directory is not a directory"

        return True, None

    except (OSError, ValueError) as e:
        return False, f"Invalid working directory: {str(e)}"


def safe_join(base: str, *paths: str) -> Optional[Path]:
    """
    Safely join paths, ensuring result stays within base directory.

    Args:
        base: The base directory (must be absolute)
        *paths: Path components to join

    Returns:
        Resolved Path if safe, None if path escapes base directory
    """
    try:
        base_path = Path(base).resolve()

        # Join all paths
        result = base_path
        for p in paths:
            # Validate each component
            if '\x00' in p:
                return None
            result = result / p

        # Resolve and check bounds
        resolved = result.resolve()

        # Ensure result is within base
        try:
            common = os.path.commonpath([str(base_path), str(resolved)])
            if common != str(base_path):
                return None
        except ValueError:
            return None

        return resolved

    except (OSError, ValueError):
        return None


def sanitize_filename(filename: str, max_length: int = 255) -> str:
    """
    Sanitize a filename for safe filesystem use.

    Args:
        filename: The filename to sanitize
        max_length: Maximum allowed length (default 255)

    Returns:
        Sanitized filename
    """
    if not filename:
        return "unnamed"

    # Remove dangerous characters
    sanitized = filename
    for char in DANGEROUS_PATH_CHARS:
        sanitized = sanitized.replace(char, '')

    # Remove path separators
    sanitized = sanitized.replace('/', '_').replace('\\', '_')

    # Remove other potentially problematic characters
    sanitized = re.sub(r'[<>:"|?*]', '_', sanitized)

    # Collapse multiple underscores
    sanitized = re.sub(r'_+', '_', sanitized)

    # Strip leading/trailing whitespace and dots
    sanitized = sanitized.strip('. \t')

    # Truncate if too long
    if len(sanitized) > max_length:
        # Try to preserve extension
        if '.' in sanitized:
            name, ext = sanitized.rsplit('.', 1)
            max_name = max_length - len(ext) - 1
            if max_name > 0:
                sanitized = name[:max_name] + '.' + ext
            else:
                sanitized = sanitized[:max_length]
        else:
            sanitized = sanitized[:max_length]

    # If completely empty after sanitization, use default
    if not sanitized:
        return "unnamed"

    return sanitized


def is_safe_command_char(char: str) -> bool:
    """Check if a character is safe in a command context."""
    # Allow alphanumeric, common punctuation, and path separators
    return char.isalnum() or char in ' ._-/\\'


def validate_session_id(session_id: str) -> Tuple[bool, Optional[str]]:
    """
    Validate a session ID for safe use in filenames and paths.

    Args:
        session_id: The session ID to validate

    Returns:
        (is_valid, error_message)
    """
    if not session_id:
        return False, "Session ID is empty"

    # Check length (reasonable bounds)
    if len(session_id) > 128:
        return False, "Session ID too long"

    # Check for dangerous characters
    for char in DANGEROUS_PATH_CHARS:
        if char in session_id:
            return False, "Session ID contains dangerous character"

    # Check for path traversal
    if '..' in session_id or '/' in session_id or '\\' in session_id:
        return False, "Session ID contains path characters"

    return True, None
