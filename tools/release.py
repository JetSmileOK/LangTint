"""Validate and package a release; never run the Windows installer in CI."""
from __future__ import annotations
import argparse
import hashlib
import json
from pathlib import Path
import re
import shutil
import struct
import zipfile

ROOT = Path(__file__).resolve().parents[1]


def digest(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def config(root: Path = ROOT) -> dict:
    value = json.loads((root / 'packaging/release.json').read_text(encoding='utf-8'))
    if not re.fullmatch(r'[0-9]+\.[0-9]+\.[0-9]+', value['version']):
        raise ValueError('Version must be MAJOR.MINOR.PATCH')
    if not re.fullmatch(r'[A-Za-z][A-Za-z0-9_.-]*\.exe', value['exe']):
        raise ValueError('Unsafe executable name')
    if not re.fullmatch(r'[0-9]+\.[0-9]+\.[0-9]+', value['go_version']):
        raise ValueError('Pin an exact Go toolchain version')
    if value['build_id'] != (root / 'BUILD_ID.txt').read_text().strip():
        raise ValueError('Build ID mismatch')
    return value


def verify_source(root: Path = ROOT) -> None:
    expected = json.loads((root / 'packaging/source-sha256.json').read_text())
    actual = {p.name for p in (root / 'source').iterdir()
              if p.suffix == '.go' or p.name == 'go.mod'}
    if set(expected) != actual:
        raise ValueError('Source file set differs from the reviewed snapshot')
    for name, sha in expected.items():
        p = root / 'source' / name
        if p.is_symlink() or digest(p) != sha:
            raise ValueError('Source integrity mismatch: ' + name)


def assemble(root: Path = ROOT) -> None:
    for name in ('app_windows.go', 'winapi_windows.go'):
        parts = sorted((root / 'source_fragments').glob(name + '.part*'))
        if not parts or any(p.is_symlink() for p in parts):
            raise ValueError('Missing or unsafe source fragments: ' + name)
        if [p.name for p in parts] != [f'{name}.part{i:02}' for i in range(1, len(parts)+1)]:
            raise ValueError('Source fragments are not contiguous: ' + name)
        (root / 'source' / name).write_bytes(b''.join(p.read_bytes() for p in parts))
    verify_source(root)


def check_pe(path: Path) -> None:
    b = path.read_bytes()
    if len(b) < 256 or b[:2] != b'MZ':
        raise ValueError('Not a Windows PE file')
    p = struct.unpack_from('<I', b, 0x3c)[0]
    if p + 96 > len(b) or b[p:p+4] != b'PE\0\0':
        raise ValueError('Invalid PE header')
    if struct.unpack_from('<H', b, p+4)[0] != 0x8664:
        raise ValueError('Expected Windows x64 machine type')
    if struct.unpack_from('<H', b, p+24)[0] != 0x20b:
        raise ValueError('Expected PE32+')
    if struct.unpack_from('<H', b, p+24+68)[0] != 2:
        raise ValueError('Expected Windows GUI subsystem')


def stage(binary: Path, output: Path, commit: str, root: Path = ROOT) -> None:
    c = config(root)
    verify_source(root)
    if not re.fullmatch(r'[0-9a-f]{40}', commit):
        raise ValueError('A full Git commit SHA is required')
    if output.exists():
        raise ValueError('Staging directory must not already exist')
    check_pe(binary)
    output.mkdir(parents=True)
    shutil.copyfile(binary, output / c['exe'])
    names = [c['exe']+'.manifest', 'RUN_ALL_AND_INSTALL.cmd', 'STATUS.cmd',
             'UNINSTALL.cmd', 'LICENSE', 'README.md', 'README.ru.md',
             'PRIVACY.md', 'CODE_SIGNING_POLICY.md', 'THIRD_PARTY_NOTICES.md', 'BUILD_ID.txt']
    for name in names:
        p = root / name
        if p.is_symlink() or not p.is_file():
            raise ValueError('Missing or unsafe package member: ' + name)
        shutil.copyfile(p, output / name)
    info = {'version': c['version'], 'build_id': c['build_id'], 'commit': commit,
            'go_version': c['go_version'], 'signing': 'unsigned',
            'unsigned_exe_sha256': digest(binary),
            'source_manifest_sha256': digest(root/'packaging/source-sha256.json')}
    (output/'RELEASE.json').write_text(json.dumps(info, indent=2)+'\n', encoding='utf-8')


def package(stage_dir: Path, output: Path, root: Path = ROOT) -> Path:
    c = config(root)
    info = json.loads((stage_dir/'RELEASE.json').read_text(encoding='utf-8-sig'))
    if info['version'] != c['version'] or info['build_id'] != c['build_id']:
        raise ValueError('Staged version does not match source')
    if info.get('signing') not in ('unsigned', 'signed'):
        raise ValueError('Unknown signing state')
    exe = stage_dir/c['exe']
    check_pe(exe)
    expected = {c['exe'], c['exe']+'.manifest', 'RUN_ALL_AND_INSTALL.cmd', 'STATUS.cmd',
                'UNINSTALL.cmd', 'LICENSE', 'README.md', 'README.ru.md', 'PRIVACY.md',
                'CODE_SIGNING_POLICY.md', 'THIRD_PARTY_NOTICES.md', 'BUILD_ID.txt', 'RELEASE.json'}
    if info['signing'] == 'signed':
        expected.add('SIGNATURE.json')
        proof = json.loads((stage_dir/'SIGNATURE.json').read_text(encoding='utf-8-sig'))
        if proof.get('status') != 'Valid' or proof.get('sha256','').lower() != digest(exe):
            raise ValueError('Signature validation record missing or stale')
    elif digest(exe) != info['unsigned_exe_sha256']:
        raise ValueError('Unsigned executable changed after staging')
    if {p.name for p in stage_dir.iterdir()} != expected:
        raise ValueError('Unapproved package contents')
    if any(not p.is_file() or p.is_symlink() for p in stage_dir.iterdir()):
        raise ValueError('Package must contain regular files only')
    output.mkdir(parents=True, exist_ok=True)
    name = f"LangTint-v{c['version']}-Windows10-x64-{info['signing']}.zip"
    target = output/name
    if target.exists():
        raise ValueError('Refusing to overwrite an existing release archive')
    hashes = ''.join(f'{digest(p)}  {p.name}\n' for p in sorted(stage_dir.iterdir()))
    with zipfile.ZipFile(target, 'w', zipfile.ZIP_DEFLATED, compresslevel=9) as z:
        for p in sorted(stage_dir.iterdir()):
            zi = zipfile.ZipInfo(p.name, (1980,1,1,0,0,0))
            zi.compress_type = zipfile.ZIP_DEFLATED
            zi.external_attr = 0o100644 << 16
            z.writestr(zi, p.read_bytes())
        zi = zipfile.ZipInfo('SHA256SUMS.txt', (1980,1,1,0,0,0))
        zi.compress_type = zipfile.ZIP_DEFLATED
        zi.external_attr = 0o100644 << 16
        z.writestr(zi, hashes.encode('utf-8'))
    (output/'SHA256SUMS.txt').write_text(f'{digest(target)}  {name}\n', encoding='utf-8')
    return target


if __name__ == '__main__':
    a = argparse.ArgumentParser(description=__doc__)
    s = a.add_subparsers(dest='cmd', required=True)
    s.add_parser('assemble')
    s.add_parser('verify-source')
    s.add_parser('config').add_argument('key', choices=['version','exe','build_id','go_version'])
    p = s.add_parser('stage')
    p.add_argument('--binary', type=Path, required=True)
    p.add_argument('--output', type=Path, required=True)
    p.add_argument('--commit', required=True)
    p = s.add_parser('package')
    p.add_argument('--stage', type=Path, required=True)
    p.add_argument('--output', type=Path, required=True)
    args = a.parse_args()
    if args.cmd == 'assemble': assemble()
    elif args.cmd == 'verify-source': verify_source()
    elif args.cmd == 'config': print(config()[args.key])
    elif args.cmd == 'stage': stage(args.binary, args.output, args.commit)
    elif args.cmd == 'package': print(package(args.stage, args.output))
