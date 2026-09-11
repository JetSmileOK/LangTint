from pathlib import Path

root = Path(__file__).resolve().parents[1]
src = root / "source"
fragments = root / "tools" / "internal" / "source_fragments"

for name in ("app_windows.go", "winapi_windows.go"):
    parts = sorted(fragments.glob(f"{name}.part*"))
    if not parts:
        raise SystemExit(f"Missing fragments for {name}")
    data = b"".join(p.read_bytes() for p in parts)
    (src / name).write_bytes(data)
    print(f"assembled {name}: {len(data)} bytes from {len(parts)} parts")
