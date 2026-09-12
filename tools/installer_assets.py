"""Deep validation and deterministic negative testing for LangTint installer assets."""
from __future__ import annotations
from pathlib import Path
import argparse
import binascii
import struct
import zlib

ROOT = Path(__file__).resolve().parents[1]
ASSETS = ROOT / 'assets'
PNG_SIG = b'\x89PNG\r\n\x1a\n'
EXPECTED_ICO_SIZES = [16, 20, 24, 32, 40, 48, 64, 128, 256]


class ValidationError(ValueError):
    pass


def _png_chunks(data: bytes):
    if not data.startswith(PNG_SIG):
        raise ValidationError('PNG signature')
    pos = 8
    out = []
    saw_iend = False
    while pos < len(data):
        if pos + 12 > len(data):
            raise ValidationError('PNG truncated chunk header')
        n = struct.unpack('>I', data[pos:pos+4])[0]
        typ = data[pos+4:pos+8]
        end = pos + 12 + n
        if n > 100_000_000 or end > len(data):
            raise ValidationError('PNG truncated chunk')
        payload = data[pos+8:pos+8+n]
        crc = struct.unpack('>I', data[pos+8+n:end])[0]
        calc = binascii.crc32(typ + payload) & 0xffffffff
        if crc != calc:
            raise ValidationError('PNG CRC')
        out.append((typ, payload, pos, end))
        pos = end
        if typ == b'IEND':
            saw_iend = True
            break
    if not saw_iend:
        raise ValidationError('PNG missing IEND')
    if pos != len(data):
        raise ValidationError('PNG trailing data')
    return out


def validate_png(data: bytes, expected_size: int | None = None) -> None:
    chunks = _png_chunks(data)
    if not chunks or chunks[0][0] != b'IHDR':
        raise ValidationError('PNG IHDR first')
    if sum(1 for x in chunks if x[0] == b'IHDR') != 1:
        raise ValidationError('PNG IHDR count')
    if chunks[-1][0] != b'IEND':
        raise ValidationError('PNG IEND last')
    ih = chunks[0][1]
    if len(ih) != 13:
        raise ValidationError('PNG IHDR length')
    w, h, depth, color, compression, filtering, interlace = struct.unpack('>IIBBBBB', ih)
    if expected_size is not None and (w, h) != (expected_size, expected_size):
        raise ValidationError('PNG dimensions')
    if w <= 0 or h <= 0 or w > 4096 or h > 4096:
        raise ValidationError('PNG dimension range')
    if (depth, color, compression, filtering, interlace) != (8, 6, 0, 0, 0):
        raise ValidationError('PNG format')
    idats = [x[1] for x in chunks if x[0] == b'IDAT']
    if not idats:
        raise ValidationError('PNG IDAT missing')
    compressed = b''.join(idats)
    dec = zlib.decompressobj()
    try:
        raw = dec.decompress(compressed) + dec.flush()
    except zlib.error as exc:
        raise ValidationError('PNG zlib stream') from exc
    if not dec.eof or dec.unused_data or dec.unconsumed_tail:
        raise ValidationError('PNG zlib boundaries')
    stride = w * 4
    if len(raw) != (stride + 1) * h:
        raise ValidationError('PNG raw length')
    for y in range(h):
        if raw[y * (stride + 1)] > 4:
            raise ValidationError('PNG filter byte')


def validate_bmp(data: bytes, expected_size: int | None = None) -> None:
    if len(data) < 54:
        raise ValidationError('BMP short')
    sig, size, reserved1, reserved2, offset = struct.unpack('<2sIHHI', data[:14])
    if sig != b'BM':
        raise ValidationError('BMP signature')
    if size != len(data):
        raise ValidationError('BMP file size')
    if (reserved1, reserved2) != (0, 0):
        raise ValidationError('BMP reserved')
    if offset < 54 or offset > len(data):
        raise ValidationError('BMP offset')
    header, w, h, planes, bpp, compression, image_size, xppm, yppm, used, important = struct.unpack('<IIIHHIIIIII', data[14:54])
    if header != 40:
        raise ValidationError('BMP DIB')
    if expected_size is not None and (w, h) != (expected_size, expected_size):
        raise ValidationError('BMP dimensions')
    if planes != 1 or bpp != 24 or compression != 0:
        raise ValidationError('BMP format')
    if (xppm, yppm) != (2835, 2835) or used != 0 or important != 0:
        raise ValidationError('BMP metadata')
    stride = ((w * 3 + 3) // 4) * 4
    expected = stride * h
    if image_size not in (0, expected):
        raise ValidationError('BMP image size')
    if offset + expected != len(data):
        raise ValidationError('BMP payload size')


def validate_ico(data: bytes) -> None:
    if len(data) < 6:
        raise ValidationError('ICO short')
    reserved, typ, count = struct.unpack('<HHH', data[:6])
    if (reserved, typ) != (0, 1):
        raise ValidationError('ICO header')
    if count != len(EXPECTED_ICO_SIZES):
        raise ValidationError('ICO image count')
    if len(data) < 6 + 16 * count:
        raise ValidationError('ICO directory short')
    entries = []
    for i in range(count):
        e = data[6+16*i:22+16*i]
        w, h, colors, reserved_byte, planes, bpp, size, offset = struct.unpack('<BBBBHHII', e)
        w = 256 if w == 0 else w
        h = 256 if h == 0 else h
        if colors != 0 or reserved_byte != 0 or planes != 1 or bpp != 32:
            raise ValidationError('ICO directory format')
        if (w, h) != (EXPECTED_ICO_SIZES[i], EXPECTED_ICO_SIZES[i]):
            raise ValidationError('ICO dimensions')
        if offset < 6 + 16 * count or size <= 0 or offset + size > len(data):
            raise ValidationError('ICO bounds')
        entries.append((offset, size, w))
    previous = 6 + 16 * count
    for offset, size, width in entries:
        if offset != previous:
            raise ValidationError('ICO packing')
        validate_png(data[offset:offset+size], width)
        previous = offset + size
    if previous != len(data):
        raise ValidationError('ICO trailing data')


def validate_repository_assets(root: Path = ASSETS) -> None:
    validate_png((root / 'LangTint-256.png').read_bytes(), 256)
    validate_bmp((root / 'LangTint-WizardLight.bmp').read_bytes(), 64)
    validate_bmp((root / 'LangTint-WizardDark.bmp').read_bytes(), 64)
    validate_ico((root / 'LangTint.ico').read_bytes())


def validate_installer_image_policy(root: Path = ROOT) -> None:
    iss = (root / 'packaging/installer/LangTint.iss').read_text(encoding='utf-8')
    required = [
        r'WizardSmallImageFile=..\..\assets\LangTint-WizardLight.bmp',
        r'WizardSmallImageFileDynamicDark=..\..\assets\LangTint-WizardDark.bmp',
    ]
    for token in required:
        if token not in iss:
            raise ValidationError('installer image policy missing: ' + token)
    for line in iss.splitlines():
        if line.startswith('WizardSmallImageFile') and '.png' in line.lower():
            raise ValidationError('installer wizard must not decode PNG at startup')


def scan_installer_exe(path: Path) -> int:
    data = path.read_bytes()
    offsets = []
    start = 0
    while True:
        pos = data.find(PNG_SIG, start)
        if pos < 0:
            break
        offsets.append(pos)
        start = pos + 1
    if len(offsets) < len(EXPECTED_ICO_SIZES):
        raise ValidationError(f'compiled Setup contains only {len(offsets)} embedded PNG signatures')
    validated = 0
    for pos in offsets:
        cursor = pos + 8
        while True:
            if cursor + 12 > len(data):
                raise ValidationError('embedded PNG truncated')
            n = struct.unpack('>I', data[cursor:cursor+4])[0]
            typ = data[cursor+4:cursor+8]
            cursor += 12 + n
            if cursor > len(data):
                raise ValidationError('embedded PNG bounds')
            if typ == b'IEND':
                break
        validate_png(data[pos:cursor])
        validated += 1
    return validated


def _must_fail(func, *args) -> None:
    try:
        func(*args)
    except Exception:
        return
    raise AssertionError('invalid mutation unexpectedly accepted')


def _png_rebuild_with_idat(data: bytes, new_idat: bytes) -> bytes:
    chunks = _png_chunks(data)
    out = bytearray(PNG_SIG)
    replaced = False
    for typ, payload, _, _ in chunks:
        if typ == b'IDAT':
            if replaced:
                continue
            payload = new_idat
            replaced = True
        out += struct.pack('>I', len(payload)) + typ + payload + struct.pack('>I', binascii.crc32(typ+payload) & 0xffffffff)
    return bytes(out)


def run_negative_matrix(root: Path = ASSETS) -> int:
    light = (root / 'LangTint-WizardLight.bmp').read_bytes()
    dark = (root / 'LangTint-WizardDark.bmp').read_bytes()
    ico = (root / 'LangTint.ico').read_bytes()
    validate_bmp(light, 64)
    validate_bmp(dark, 64)
    validate_ico(ico)
    last = ico[6+16*8:22+16*8]
    _, _, _, _, _, _, png_size, png_offset = struct.unpack('<BBBBHHII', last)
    png = ico[png_offset:png_offset+png_size]
    validate_png(png, 256)
    cases = 0
    for n in range(1, 65):
        _must_fail(validate_png, png[:-n], 256); cases += 1
    positions = list(range(0, min(64, len(png)))) + list(range(64, len(png)-12, max(1, (len(png)-76)//64)))[:64]
    for pos in positions[:128]:
        m = bytearray(png); m[pos] ^= 1
        _must_fail(validate_png, bytes(m), 256); cases += 1
    compressed = b''.join(x[1] for x in _png_chunks(png) if x[0] == b'IDAT')
    for k in range(32):
        z = bytearray(compressed)
        pos = 2 + (k * 83) % (len(z) - 6)
        z[pos] ^= 1 + (k % 251)
        _must_fail(validate_png, _png_rebuild_with_idat(png, bytes(z)), 256); cases += 1
    raw = zlib.decompress(compressed)
    stride = 256 * 4 + 1
    for k in range(32):
        r = bytearray(raw)
        r[((k * 7) % 256) * stride] = 5 + (k % 251)
        _must_fail(validate_png, _png_rebuild_with_idat(png, zlib.compress(bytes(r), 9)), 256); cases += 1
    for k in range(32):
        _must_fail(validate_png, _png_rebuild_with_idat(png, zlib.compress(raw[:-(k+1)], 9)), 256); cases += 1
    for extra in range(1, 33):
        _must_fail(validate_png, png + b'X' * extra, 256); cases += 1
    for n in range(1, 33):
        _must_fail(validate_bmp, light[:-n], 64); cases += 1
    for pos in range(54):
        m = bytearray(light); m[pos] ^= 1
        _must_fail(validate_bmp, bytes(m), 64); cases += 1
    for n in range(1, 65):
        _must_fail(validate_ico, ico[:-n]); cases += 1
    for pos in range(6, 6 + 16 * 3):
        m = bytearray(ico); m[pos] ^= 1
        _must_fail(validate_ico, bytes(m)); cases += 1
    for i in range(9):
        e = ico[6+16*i:22+16*i]
        _, _, _, _, _, _, size, offset = struct.unpack('<BBBBHHII', e)
        m = bytearray(ico); m[offset + min(size-1, 20)] ^= 1
        _must_fail(validate_ico, bytes(m)); cases += 1
    for i in range(32):
        pos = (6 + 16 * 9) + ((i * 211) % (len(ico) - (6 + 16 * 9)))
        m = bytearray(ico); m[pos] ^= 1
        _must_fail(validate_ico, bytes(m)); cases += 1
    _must_fail(validate_png, png[:-13] + png[-12:], 256); cases += 1
    return cases


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--matrix', action='store_true')
    parser.add_argument('--scan-exe', type=Path)
    args = parser.parse_args()
    validate_repository_assets()
    validate_installer_image_policy()
    print('INSTALLER_ASSETS=PASS')
    if args.matrix:
        count = run_negative_matrix()
        print(f'NEGATIVE_ASSET_CASES={count}')
    if args.scan_exe:
        count = scan_installer_exe(args.scan_exe)
        print(f'EMBEDDED_PNGS_VALIDATED={count}')


if __name__ == '__main__':
    main()
