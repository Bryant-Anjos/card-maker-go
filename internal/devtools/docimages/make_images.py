#!/usr/bin/env python3
"""Renders the illustrations used by README.md and docs/ with the real tool.

Nothing here is drawn by hand: the cards come from `cardmaker build` with the
default theme and the sample data in testdata/showcase, then are composed into
sheets. When the theme or the samples change, run `make docs-images` (it also
refreshes docs/images/SOURCE_HASH, which a test checks).

Requirements: Python 3 with Pillow, a Chromium browser (see CARDMAKER_CHROME)
and DejaVu Sans (apt: fonts-dejavu-core) for the labels of the anatomy image.
"""
import os
import shutil
import subprocess
import sys
import tempfile

from PIL import Image, ImageDraw, ImageFont

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", ".."))
OUT = os.path.join(ROOT, "docs", "images")
THEME = os.path.join(ROOT, "themes", "default")

CARD_W, CARD_H = 750, 1050  # theme size, scale 1
BG = (11, 16, 26)
GAP = 40
JPEG_QUALITY = 88

# Layout of the default theme in card pixels, used to crop the title band
# (name, divider and element icon) of standard cards.
BAND_TOP, BAND_BOTTOM = 700, 812

FONT_CANDIDATES = [
    "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
    "/usr/share/fonts/dejavu/DejaVuSans.ttf",
    "/Library/Fonts/DejaVuSans.ttf",
]


def build(cards_file, out_dir, scale=1):
    """Runs the real CLI for one sample set and returns the rendered cards dir."""
    subprocess.run(
        ["go", "run", "./cmd/cardmaker", "build",
         "-cards", os.path.join(ROOT, "testdata", "showcase", cards_file),
         "-theme", THEME, "-out", out_dir, "-scales", str(scale), "-formats", "png"],
        cwd=ROOT, check=True, stdout=subprocess.DEVNULL)
    return os.path.join(out_dir, "pt-BR")


def open_card(directory, card_id):
    return Image.open(os.path.join(directory, "cards", card_id + ".png")).convert("RGBA")


def sheet(width, height):
    return Image.new("RGBA", (width, height), BG + (255,))


def save_jpeg(image, name, max_width):
    image = image.convert("RGB")
    if image.width > max_width:
        image = image.resize((max_width, round(image.height * max_width / image.width)), Image.LANCZOS)
    path = os.path.join(OUT, name)
    image.save(path, quality=JPEG_QUALITY, optimize=True)
    print("  %-26s %5d x %-5d %4d KB" % (name, image.width, image.height, os.path.getsize(path) // 1024))


def row(images, columns):
    """Lays images out in rows of `columns`, centered, on the dark background."""
    rows = [images[i:i + columns] for i in range(0, len(images), columns)]
    width = columns * CARD_W + (columns + 1) * GAP
    canvas = sheet(width, sum(max(i.height for i in r) for r in rows) + (len(rows) + 1) * GAP)
    y = GAP
    for r in rows:
        x = GAP + (columns - len(r)) * (CARD_W + GAP) // 2
        for image in r:
            canvas.alpha_composite(image, (x, y))
            x += CARD_W + GAP
        y += max(i.height for i in r) + GAP
    return canvas


def band(card):
    """Title band of a standard card: name, divider and element icon."""
    background = Image.new("RGBA", card.size, BG + (255,))
    background.alpha_composite(card)
    return background.crop((14, BAND_TOP, CARD_W - 14, BAND_BOTTOM))


def hero_rarities(tmp):
    directory = build("cards.json", tmp)
    ids = ["vulpin", "cervallis", "lunaris", "aureon", "drakarion"]
    save_jpeg(row([open_card(directory, i) for i in ids], 5), "raridades.jpg", 2400)
    return directory


def fullart(tmp, standard_dir):
    full_dir = build("fullart.json", os.path.join(tmp, "fullart"))
    ids = ["vulpin", "cervallis", "lunaris", "aureon", "drakarion"]
    images = [open_card(standard_dir, i) for i in ids] + [open_card(full_dir, i + "-fullart") for i in ids]
    save_jpeg(row(images, 5), "fullart.jpg", 2400)


def elements(tmp):
    directory = build("elements.json", os.path.join(tmp, "elements"))
    ids = ["beast", "fire", "water", "nature", "wind", "light", "dark", "electric", "earth", "ice", "mystic"]
    bands = [band(open_card(directory, "elemento-" + i)) for i in ids]
    bw, bh = bands[0].size
    columns = 4
    rows = -(-len(bands) // columns)  # ceiling division
    canvas = sheet(columns * bw + (columns + 1) * GAP, rows * bh + (rows + 1) * GAP)
    for n, b in enumerate(bands):
        canvas.alpha_composite(b, (GAP + (n % columns) * (bw + GAP), GAP + (n // columns) * (bh + GAP)))
    save_jpeg(canvas, "elementos.jpg", 2000)


def long_names(tmp):
    directory = build("names.json", os.path.join(tmp, "names"))
    standard = [band(open_card(directory, "nome-%d" % i)) for i in range(1, 6)]
    # full art keeps its texts over the artwork, so crop the same region without the dark background
    full = []
    for i in range(1, 6):
        card = open_card(directory, "nome-%d-fullart" % i)
        flat = Image.new("RGBA", card.size, BG + (255,))
        flat.alpha_composite(card)
        full.append(flat.crop((14, 688, CARD_W - 14, 800)))
    bw, bh = standard[0].size
    canvas = sheet(2 * bw + 3 * GAP, 5 * bh + 6 * GAP)
    for n in range(5):
        canvas.alpha_composite(standard[n], (GAP, GAP + n * (bh + GAP)))
        canvas.alpha_composite(full[n], (2 * GAP + bw, GAP + n * (bh + GAP)))
    save_jpeg(canvas, "nomes-longos.jpg", 1800)


def front_and_back(directory):
    front = open_card(directory, "lunaris")
    back = Image.open(os.path.join(directory, "back.png")).convert("RGBA")
    save_jpeg(row([front, back], 2), "frente-e-verso.jpg", 1600)


def load_font(size):
    for path in FONT_CANDIDATES:
        if os.path.exists(path):
            return ImageFont.truetype(path, size)
    sys.exit("DejaVu Sans not found; install fonts-dejavu-core (see the docstring).")


# Numbered callouts of the anatomy image: (label, point on the card, side).
# The points are positions inside the default theme's 750x1050 canvas, chosen
# on empty spots next to the elements so lines never cross text; if the theme
# layout changes, move them (and look at the result).
CALLOUTS = [
    ("Ilustração", (190, 330), "left"),
    ("Moldura: cor e ornamentos\nvariam com a raridade", (24, 520), "left"),
    ("Nome (encolhe se for longo)", (60, 752), "left"),
    ("Descrição (até 3 linhas)", (60, 842), "left"),
    ("Número da coleção", (690, 98), "right"),
    ("Ícone do elemento", (668, 772), "right"),
    ("Gema da raridade", (375, 1030), "right"),
]


def anatomy(directory):
    scale = 2  # draw at 2x for smooth lines and text, then downscale
    margin = 560
    front = open_card(directory, "lunaris")
    canvas = sheet((CARD_W + 2 * margin) * scale, (CARD_H + 80) * scale)
    canvas.alpha_composite(front.resize((CARD_W * scale, CARD_H * scale), Image.LANCZOS), (margin * scale, 40 * scale))
    draw = ImageDraw.Draw(canvas)
    font = load_font(15 * scale)
    badge_font = load_font(15 * scale)
    accent = (245, 192, 74)

    left_y = right_y = 0
    for number, (label, (px, py), side) in enumerate(CALLOUTS, start=1):
        x, y = (margin + px) * scale, (40 + py) * scale
        # label column position: stacked from the top, aligned to the target height
        if side == "left":
            tx = 24 * scale
            ty = max(y - 20 * scale, left_y)
            left_y = ty + 90 * scale
            anchor_x = (margin - 18) * scale
        else:
            tx = (margin + CARD_W + 40) * scale
            ty = max(y - 20 * scale, right_y)
            right_y = ty + 90 * scale
            anchor_x = (margin + CARD_W + 18) * scale
        draw.line([(x, y), (anchor_x, ty + 20 * scale)], fill=accent, width=2 * scale)
        draw.ellipse([x - 7 * scale, y - 7 * scale, x + 7 * scale, y + 7 * scale], fill=accent, outline=(11, 16, 26), width=2 * scale)
        r = 15 * scale
        cx = tx + r if side == "left" else tx + r
        draw.ellipse([cx - r, ty + 20 * scale - r, cx + r, ty + 20 * scale + r], fill=accent)
        draw.text((cx, ty + 20 * scale), str(number), fill=(11, 16, 26), font=badge_font, anchor="mm")
        draw.multiline_text((cx + r + 10 * scale, ty + 20 * scale), label, fill=(232, 237, 246), font=font, anchor="lm", spacing=6 * scale)
    save_jpeg(canvas.resize((canvas.width // scale, canvas.height // scale), Image.LANCZOS), "anatomia.jpg", 1900)


def main():
    os.makedirs(OUT, exist_ok=True)
    tmp = tempfile.mkdtemp(prefix="cardmaker-docimages-")
    try:
        print("rendering with the real cardmaker (default theme)...")
        standard = hero_rarities(tmp)
        fullart(tmp, standard)
        elements(tmp)
        long_names(tmp)
        front_and_back(standard)
        anatomy(standard)
    finally:
        shutil.rmtree(tmp, ignore_errors=True)


if __name__ == "__main__":
    main()
