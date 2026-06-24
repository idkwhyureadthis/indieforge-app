"""Crop the IndieForge logo to a tight, transparent-background PNG.

Usage:
    python frontend/scripts/crop_logo.py <input> [output]

Defaults: input  = frontend/public/logo-src.png
          output = frontend/public/logo.png

- Makes near-white pixels transparent.
- Trims surrounding empty space.
"""
import sys
from pathlib import Path

try:
    from PIL import Image
except ImportError:
    sys.exit("Pillow is required: pip install pillow")

ROOT = Path(__file__).resolve().parents[1]
src = Path(sys.argv[1]) if len(sys.argv) > 1 else ROOT / "public" / "logo-src.png"
out = Path(sys.argv[2]) if len(sys.argv) > 2 else ROOT / "public" / "logo.png"

if not src.exists():
    sys.exit(f"Input not found: {src}\nSave the logo there first.")

img = Image.open(src).convert("RGBA")
px = img.load()
w, h = img.size

# White (and near-white) -> transparent.
THRESH = 240
for y in range(h):
    for x in range(w):
        r, g, b, a = px[x, y]
        if r >= THRESH and g >= THRESH and b >= THRESH:
            px[x, y] = (r, g, b, 0)

# Trim to the bounding box of the remaining (opaque) content.
bbox = img.getbbox()
if bbox:
    pad = 8
    l, t, r, b = bbox
    l, t = max(0, l - pad), max(0, t - pad)
    r, b = min(w, r + pad), min(h, b + pad)
    img = img.crop((l, t, r, b))

out.parent.mkdir(parents=True, exist_ok=True)
img.save(out)
print(f"Saved {out} ({img.size[0]}x{img.size[1]})")
