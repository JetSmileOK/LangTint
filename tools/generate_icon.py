"""Generate deterministic LangTint PNG/ICO assets using only Python stdlib."""
from __future__ import annotations
from pathlib import Path
import struct
import zlib

ROOT = Path(__file__).resolve().parents[1]
ASSETS = ROOT / 'assets'

BG = (24, 28, 36, 255)
BLUE = (183, 233, 255, 255)  # product tint #B7E9FF
DARK = (12, 17, 24, 255)
WHITE = (245, 250, 252, 255)


def inside_round_rect(x, y, n, margin, radius):
    if margin + radius <= x < n - margin - radius and margin <= y < n - margin:
        return True
    if margin <= x < n - margin and margin + radius <= y < n - margin - radius:
        return True
    cx = margin + radius if x < n / 2 else n - margin - radius
    cy = margin + radius if y < n / 2 else n - margin - radius
    return (x - cx) ** 2 + (y - cy) ** 2 <= radius ** 2


def point_in_poly(x, y, pts):
    inside = False
    j = len(pts) - 1
    for i, (xi, yi) in enumerate(pts):
        xj, yj = pts[j]
        if ((yi > y) != (yj > y)) and x < (xj - xi) * (y - yi) / (yj - yi) + xi:
            inside = not inside
        j = i
    return inside


def render(n: int) -> bytes:
    # 4x supersampling gives clean small Windows icon edges without dependencies.
    ss = 4
    N = n * ss
    margin = N * 0.075
    radius = N * 0.18
    pixels = bytearray(n * n * 4)

    # Cursor polygon normalized to the canvas. A dark larger polygon gives outline.
    arrow = [(0.31, 0.20), (0.31, 0.69), (0.43, 0.58), (0.52, 0.79),
             (0.62, 0.74), (0.53, 0.54), (0.70, 0.53)]
    center = (0.48, 0.50)
    def scale_poly(poly, factor):
        cx, cy = center
        return [((cx + (px-cx)*factor)*N, (cy + (py-cy)*factor)*N) for px, py in poly]
    arrow_outer = scale_poly(arrow, 1.10)
    arrow_inner = scale_poly(arrow, 1.00)

    for py in range(n):
        for px in range(n):
            acc = [0, 0, 0, 0]
            for sy in range(ss):
                for sx in range(ss):
                    x = px * ss + sx + 0.5
                    y = py * ss + sy + 0.5
                    rgba = (0, 0, 0, 0)
                    if inside_round_rect(x, y, N, margin, radius):
                        rgba = BG
                        # Ambient taskbar cue at bottom.
                        if y >= N * 0.77:
                            rgba = BLUE
                        # Cursor dark outline then blue fill.
                        if point_in_poly(x, y, arrow_outer):
                            rgba = DARK
                        if point_in_poly(x, y, arrow_inner):
                            rgba = BLUE
                        # Tiny highlight helps the cursor remain legible at 16px.
                        if N * 0.315 <= x <= N * 0.34 and N * 0.24 <= y <= N * 0.54:
                            rgba = WHITE
                    for k in range(4):
                        acc[k] += rgba[k]
            off = (py*n + px)*4
            count = ss*ss
            pixels[off:off+4] = bytes(v//count for v in acc)
    return bytes(pixels)


def png_bytes(n: int) -> bytes:
    rgba = render(n)
    raw = b''.join(b'\x00' + rgba[y*n*4:(y+1)*n*4] for y in range(n))
    def chunk(kind, data):
        return struct.pack('>I', len(data)) + kind + data + struct.pack('>I', zlib.crc32(kind+data)&0xffffffff)
    return (b'\x89PNG\r\n\x1a\n' +
            chunk(b'IHDR', struct.pack('>IIBBBBB', n,n,8,6,0,0,0)) +
            chunk(b'IDAT', zlib.compress(raw, 9)) + chunk(b'IEND', b''))


def write_ico(path: Path) -> None:
    sizes = [16, 20, 24, 32, 40, 48, 64, 128, 256]
    images = [(n, png_bytes(n)) for n in sizes]
    header = struct.pack('<HHH', 0, 1, len(images))
    offset = 6 + 16*len(images)
    entries = []
    payload = []
    for n, data in images:
        w = 0 if n == 256 else n
        h = 0 if n == 256 else n
        entries.append(struct.pack('<BBBBHHII', w,h,0,0,1,32,len(data),offset))
        payload.append(data)
        offset += len(data)
    path.write_bytes(header + b''.join(entries) + b''.join(payload))


def main():
    ASSETS.mkdir(parents=True, exist_ok=True)
    (ASSETS/'LangTint-256.png').write_bytes(png_bytes(256))
    write_ico(ASSETS/'LangTint.ico')
    print('generated', ASSETS/'LangTint.ico')

if __name__ == '__main__':
    main()
