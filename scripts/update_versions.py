#!/usr/bin/env python3
"""Update MageBox versions.yaml files with new Magento/MageOS releases.

Updates both:
  - internal/lib/config/versions.yaml  (embedded in binary)
  - lib/templates/config/versions.yaml (user-overridable templates)

Usage:
  python3 scripts/update_versions.py --magento 2.4.8-p5
  python3 scripts/update_versions.py --mageos 2.2.1
  python3 scripts/update_versions.py --mageos 2.2.1 --base 2.4.8
  python3 scripts/update_versions.py --magento 2.4.8-p5 --mageos 2.2.1
"""

import argparse
import re
import sys
from pathlib import Path

try:
    from ruamel.yaml import YAML
    from ruamel.yaml.comments import CommentedMap, CommentedSeq
    from ruamel.yaml.scalarstring import DoubleQuotedScalarString as DQ
except ImportError:
    print("ERROR: ruamel.yaml is required. Install with: pip install ruamel.yaml", file=sys.stderr)
    sys.exit(1)

REPO_ROOT = Path(__file__).parent.parent

VERSION_FILES = [
    REPO_ROOT / "internal" / "lib" / "config" / "versions.yaml",
    REPO_ROOT / "lib" / "templates" / "config" / "versions.yaml",
]

# PHP compatibility keyed by (major, minor, patch) of Magento version.
# Patch 0 is the base release (e.g. 2.4.8 → (2,4,8)).
MAGENTO_PHP_MAP = {
    (2, 4, 8): ["8.4", "8.3", "8.2"],
    (2, 4, 7): ["8.3", "8.2"],
    (2, 4, 6): ["8.2", "8.1"],
    (2, 4, 5): ["8.1", "8.0"],
}


# ---------------------------------------------------------------------------
# Version helpers
# ---------------------------------------------------------------------------

def parse_magento_version(v: str) -> tuple[int, int, int, int]:
    """Parse '2.4.8-p3' → (2, 4, 8, 3) or '2.4.8' → (2, 4, 8, 0)."""
    m = re.fullmatch(r"(\d+)\.(\d+)\.(\d+)(?:-p(\d+))?", v)
    if not m:
        raise ValueError(f"Unrecognised Magento version format: {v!r}")
    return int(m.group(1)), int(m.group(2)), int(m.group(3)), int(m.group(4) or 0)


def parse_mageos_version(v: str) -> tuple[int, int, int]:
    """Parse '2.2.1' → (2, 2, 1)."""
    m = re.fullmatch(r"(\d+)\.(\d+)\.(\d+)", v)
    if not m:
        raise ValueError(f"Unrecognised MageOS version format: {v!r}")
    return int(m.group(1)), int(m.group(2)), int(m.group(3))


def flow_list(items: list) -> CommentedSeq:
    """Return a ruamel CommentedSeq rendered as a YAML flow sequence [a, b, c]."""
    seq = CommentedSeq([DQ(i) for i in items])
    seq.fa.set_flow_style()
    return seq


def is_newer(a: str, b: str, distribution: str) -> bool:
    """Return True if version *a* is strictly newer than *b*."""
    if distribution == "magento":
        return parse_magento_version(a) > parse_magento_version(b)
    return parse_mageos_version(a) > parse_mageos_version(b)


# ---------------------------------------------------------------------------
# PHP / base-version lookup
# ---------------------------------------------------------------------------

def magento_php_versions(new_ver: str, existing: list) -> list[str]:
    """
    Return the PHP versions compatible with *new_ver*.

    Strategy (in order):
    1. Copy from the most-recent existing entry with the same major.minor.patch.
    2. Use the static lookup table.
    3. Fall back to the most-recent entry's PHP list.
    4. Hard fallback: latest supported PHP trio.
    """
    maj, minor, patch, _ = parse_magento_version(new_ver)

    for entry in existing:
        e_maj, e_minor, e_patch, _ = parse_magento_version(entry["version"])
        if (e_maj, e_minor, e_patch) == (maj, minor, patch):
            return list(entry["php"])

    key = (maj, minor, patch)
    if key in MAGENTO_PHP_MAP:
        return MAGENTO_PHP_MAP[key]

    if existing:
        return list(existing[0]["php"])

    return ["8.4", "8.3", "8.2"]


def mageos_php_and_base(new_ver: str, existing: list, override_base: str | None) -> tuple[list[str], str]:
    """
    Return (php_versions, base_magento_version) for a new MageOS release.

    Strategy for base version (in order):
    1. Use --base CLI argument if provided.
    2. Copy base from an existing entry with the same major.minor.
    3. Copy base from any existing entry (newest first).
    4. Fall back to the hardcoded latest known Magento version.
    """
    maj, minor, _ = parse_mageos_version(new_ver)

    # Walk existing entries looking for same major.minor
    for entry in existing:
        e_maj, e_minor, _ = parse_mageos_version(entry["version"])
        if (e_maj, e_minor) == (maj, minor):
            base = override_base or entry["base"]
            return list(entry["php"]), base

    # No same-minor entry; use override or first existing base
    if override_base:
        base = override_base
    elif existing:
        base = existing[0]["base"]
    else:
        base = "2.4.8"

    # Derive PHP versions from the base Magento minor
    try:
        b_maj, b_minor, b_patch, _ = parse_magento_version(base)
        php = MAGENTO_PHP_MAP.get((b_maj, b_minor, b_patch))
        if php is None and existing:
            php = list(existing[0]["php"])
        php = php or ["8.4", "8.3", "8.2"]
    except ValueError:
        php = ["8.4", "8.3", "8.2"]

    return php, base


# ---------------------------------------------------------------------------
# YAML update logic
# ---------------------------------------------------------------------------

def build_magento_entry(version: str, php_versions: list, is_latest: bool) -> CommentedMap:
    label = f"Magento {version} (Latest)" if is_latest else f"Magento {version}"
    entry = CommentedMap()
    entry["version"] = DQ(version)
    entry["name"] = DQ(label)
    entry["php"] = flow_list(php_versions)
    if is_latest:
        entry["default"] = True
    return entry


def build_mageos_entry(version: str, php_versions: list, base: str, is_latest: bool) -> CommentedMap:
    label = f"MageOS {version} (Latest)" if is_latest else f"MageOS {version}"
    entry = CommentedMap()
    entry["version"] = DQ(version)
    entry["name"] = DQ(label)
    entry["php"] = flow_list(php_versions)
    if is_latest:
        entry["default"] = True
    entry["base"] = DQ(base)
    return entry


def strip_latest_from_entry(entry: CommentedMap) -> None:
    """Remove '(Latest)' suffix and unset default:true on an existing entry."""
    name: str = entry.get("name", "")
    entry["name"] = DQ(str(name).replace(" (Latest)", ""))
    if "default" in entry:
        del entry["default"]


def version_already_known(version: str, existing: list) -> bool:
    return any(e["version"] == version for e in existing)


def add_magento_version(data: CommentedMap, new_ver: str) -> bool:
    """
    Insert *new_ver* into the magento.versions list.  Returns True if added.
    """
    versions: list = data["magento"]["versions"]

    if version_already_known(new_ver, versions):
        print(f"  [magento] {new_ver} already present – skipping.")
        return False

    php = magento_php_versions(new_ver, versions)

    # Determine whether the new version is the overall latest
    is_latest = all(
        not is_newer(e["version"], new_ver, "magento") for e in versions
    )

    new_entry = build_magento_entry(new_ver, php, is_latest)

    if is_latest:
        # Demote the current top entry
        if versions:
            strip_latest_from_entry(versions[0])
        versions.insert(0, new_entry)
    else:
        # Insert in sorted position
        idx = 0
        while idx < len(versions) and is_newer(versions[idx]["version"], new_ver, "magento"):
            idx += 1
        versions.insert(idx, new_entry)

    print(f"  [magento] Added {new_ver}  php={php}  latest={is_latest}")
    return True


def add_mageos_version(data: CommentedMap, new_ver: str, override_base: str | None) -> bool:
    """
    Insert *new_ver* into the mageos.versions list.  Returns True if added.
    """
    versions: list = data["mageos"]["versions"]

    if version_already_known(new_ver, versions):
        print(f"  [mageos] {new_ver} already present – skipping.")
        return False

    php, base = mageos_php_and_base(new_ver, versions, override_base)

    is_latest = all(
        not is_newer(e["version"], new_ver, "mageos") for e in versions
    )

    new_entry = build_mageos_entry(new_ver, php, base, is_latest)

    if is_latest:
        if versions:
            strip_latest_from_entry(versions[0])
        versions.insert(0, new_entry)
    else:
        idx = 0
        while idx < len(versions) and is_newer(versions[idx]["version"], new_ver, "mageos"):
            idx += 1
        versions.insert(idx, new_entry)

    print(f"  [mageos]  Added {new_ver}  php={php}  base={base}  latest={is_latest}")
    return True


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def update_file(path: Path, magento_ver: str | None, mageos_ver: str | None, base: str | None) -> bool:
    yaml = YAML()
    yaml.preserve_quotes = True
    yaml.width = 120
    yaml.indent(mapping=2, sequence=4, offset=2)

    with path.open() as fh:
        data = yaml.load(fh)

    changed = False

    if magento_ver:
        changed |= add_magento_version(data, magento_ver)

    if mageos_ver:
        changed |= add_mageos_version(data, mageos_ver, base)

    if changed:
        with path.open("w") as fh:
            yaml.dump(data, fh)
        print(f"  Wrote {path}")

    return changed


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Add new Magento / MageOS versions to the MageBox version registry."
    )
    parser.add_argument("--magento", metavar="VERSION", help="New Magento Open Source version (e.g. 2.4.8-p5)")
    parser.add_argument("--mageos",  metavar="VERSION", help="New MageOS version (e.g. 2.2.1)")
    parser.add_argument(
        "--base",
        metavar="MAGENTO_VERSION",
        help="Base Magento version for the MageOS release (e.g. 2.4.8). "
             "Inferred automatically when not provided.",
    )
    args = parser.parse_args()

    if not args.magento and not args.mageos:
        parser.print_help()
        sys.exit(1)

    if args.base and not args.mageos:
        parser.error("--base only applies when --mageos is also given")

    any_changed = False
    for path in VERSION_FILES:
        if not path.exists():
            print(f"WARNING: {path} not found – skipping.", file=sys.stderr)
            continue
        print(f"\nProcessing {path}:")
        any_changed |= update_file(path, args.magento, args.mageos, args.base)

    if any_changed:
        print("\nDone – version files updated.")
    else:
        print("\nNo changes made.")
        sys.exit(0)


if __name__ == "__main__":
    main()
