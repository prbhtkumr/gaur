#!/usr/bin/env python3
"""Generate public/og.png (1200x630): the terminal hero with a large title overlay.

Run after `npx astro build`:

    npx astro build && python3 scripts/make-og.py

Capture is done with headless Chromium against dist/, so the card always matches
what the site actually renders (reduced-motion static state, transitions off).
"""

import re
import socket
import subprocess
import sys
import time
from pathlib import Path

from PIL import Image, ImageDraw, ImageFilter, ImageFont

ROOT = Path(__file__).resolve().parent.parent
DIST = ROOT / "dist"
OUT = ROOT / "public" / "og.png"

W, H = 1200, 630

def rgb(value):
    """'#rrggbb' -> (r, g, b)."""
    if isinstance(value, tuple):
        return value
    return tuple(int(value[i:i + 2], 16) for i in (1, 3, 5))


def ink(font, text):
    """True glyph ink box. font.getbbox() reports the advance box, not ink."""
    return font.getmask(text).getbbox()


# Matches .crt-glow in src/pages/index.astro: tight core, quick falloff, faint
# spill. PIL's GaussianBlur radius is sigma; CSS text-shadow blur-radius is 2*sigma.
CRT_GLOW = ((2, 0.45), (6, 0.22), (14, 0.10))


def crt_glow(canvas, xy, text, font, color, layers):
    """Composite a layered phosphor bloom behind `text`, then the glyphs.

    The halo is built from a blurred *alpha* channel over a flat colour so
    transparent pixels can't bleed black into the bloom.
    """
    tone = rgb(color)
    glyphs = Image.new("RGBA", canvas.size, (0, 0, 0, 0))
    ImageDraw.Draw(glyphs).text(xy, text, font=font, fill=(*tone, 255))

    out = canvas.convert("RGBA")
    for radius, alpha in layers:
        halo = Image.new("RGBA", canvas.size, (*tone, 255))
        halo.putalpha(
            glyphs.getchannel("A").filter(ImageFilter.GaussianBlur(radius))
            .point(lambda v: round(v * alpha))
        )
        out = Image.alpha_composite(out, halo)
    return Image.alpha_composite(out, glyphs).convert("RGB")
VIEW_W, VIEW_H = 1440, 900

FONT_DIR = Path("/usr/local/share/fonts/c")
FONT_MONO_BOLD = FONT_DIR / "CaskaydiaCoveNerdFontMono_Bold.ttf"
FONT_MONO_REG = FONT_DIR / "CaskaydiaCoveNerdFontMono_Regular.ttf"

BASE = "#1e1e2e"
CRUST = "#11111b"
CRUST_rgb = tuple(int(CRUST[i:i + 2], 16) for i in (1, 3, 5))
MAUVE = "#cba6f7"
MUTED = "#a6adc8"
HIGHLIGHT = "#74c7ec"
GREEN = "#a6e3a1"
# Blended over CRUST to match the hero's /70 and /90 alpha utilities.
CURSOR = (147, 121, 181)   # bg-primary/70
GREEN_90 = (151, 206, 147)  # text-green/90

CHROMIUM = "/usr/bin/chromium"
FLAGS = [
    "--headless=new",
    "--no-sandbox",
    "--disable-dev-shm-usage",
    "--force-prefers-reduced-motion",
    "--hide-scrollbars",
]

INJECTED_STYLE = (
    "<style>*,*::before,*::after{transition:none!important;animation:none!important}"
    "html{scroll-behavior:auto!important}</style>"
)

RECT_PROBE = """<pre id="__rect"></pre><script>
window.addEventListener('load',function(){setTimeout(function(){
  var el=document.getElementById('terminal-container');
  var out;
  if(!el){out='ABSENT';}
  else{var r=el.getBoundingClientRect();
    out=[Math.round(r.x),Math.round(r.y),Math.round(r.width),Math.round(r.height)].join(',');}
  document.getElementById('__rect').textContent='\\n'+out;
},2500);});
</script>"""


def die(msg: str) -> None:
    print(f"error: {msg}", file=sys.stderr)
    sys.exit(1)


def free_port() -> int:
    with socket.socket() as s:
        s.bind(("127.0.0.1", 0))
        return s.getsockname()[1]


def make_capture_html() -> None:
    """Write dist/__og_capture.html = built home page with transitions disabled."""
    src = DIST / "index.html"
    if not src.exists():
        die(f"{src} not found - run `npx astro build` first")
    html = src.read_text()
    if "</head>" not in html:
        die("built index.html has no </head>")
    html = html.replace("</head>", INJECTED_STYLE + "</head>", 1)
    html = html.replace("</body>", RECT_PROBE + "</body>", 1)
    (DIST / "__og_capture.html").write_text(html)


def serve() -> tuple[subprocess.Popen, int]:
    port = free_port()
    proc = subprocess.Popen(
        [sys.executable, "-m", "http.server", str(port), "--directory", str(DIST)],
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
    time.sleep(1.0)
    return proc, port


def run_chromium(
    url: str, port: int, extra: list[str], out: Path | None, view_h: int = VIEW_H
) -> str:
    cmd = [
        CHROMIUM,
        *FLAGS,
        f"--window-size={VIEW_W},{view_h}",
        *extra,
    ]
    if out is None:
        cmd += ["--virtual-time-budget=7000", "--dump-dom"]
    else:
        cmd.append(f"--screenshot={out}")
    cmd.append(url)
    res = subprocess.run(cmd, capture_output=True, text=True, timeout=120)
    if out is None and "error" in res.stderr.lower() and not res.stdout:
        die(f"chromium failed: {res.stderr.strip()[:400]}")
    return res.stdout


def main() -> None:
    if not (DIST / "index.html").exists():
        die("dist/ not built - run `npx astro build` first")
    if not FONT_MONO_BOLD.exists() or not FONT_MONO_REG.exists():
        die(f"CaskaydiaCove Nerd Font not found in {FONT_DIR}")

    make_capture_html()
    proc, port = serve()
    url = f"http://127.0.0.1:{port}/__og_capture.html"
    shot = Path("/tmp/og_hero.png")

    try:
        # The hero layout moves (e.g. the install bar sits above the terminal),
        # so size the capture window to whatever the terminal actually needs.
        view_h = VIEW_H
        for _ in range(4):
            dom = run_chromium(url, port, [], None, view_h=view_h)
            m = re.search(r'<pre id="__rect">(.*?)</pre>', dom, re.S)
            raw = m.group(1) if m else ""
            if "ABSENT" in raw or not re.match(r"^\s*\d+,\s*\d+,\s*\d+,\s*\d+", raw):
                die(f"could not measure terminal-container (got {raw!r})")
            x, y, w, h = (int(v) for v in raw.strip().split(","))
            if w < 400 or h < 300:
                die(f"terminal-container measured implausibly small: {w}x{h}")
            if y + h + 40 <= view_h:
                break
            view_h = y + h + 40
        else:
            die("could not find a window height that fits the terminal")

        run_chromium(
            url, port, ["--virtual-time-budget=7000"], shot, view_h=view_h
        )
        if not shot.exists():
            die("chromium produced no screenshot")
    finally:
        proc.terminate()
        try:
            proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            proc.kill()
        (DIST / "__og_capture.html").unlink(missing_ok=True)

    compose(shot, (x, y, w, h))
    shot.unlink(missing_ok=True)
    print(f"wrote {OUT} ({W}x{H})")


def compose(shot: Path, rect: tuple[int, int, int, int]) -> None:
    x, y, w, h = rect
    hero = Image.open(shot).convert("RGB")
    if hero.width < x + w or hero.height < y + h:
        die(f"terminal crop {rect} outside screenshot {hero.size}")
    term = hero.crop((x, y, x + w, y + h))

    # Solid crust background — the left column must stay perfectly legible.
    canvas = Image.new("RGB", (W, H), CRUST)

    # Whole terminal, right-aligned. Fit inside a box so nothing is cropped away.
    box_w, box_h = 700, 470
    scale = min(box_w / term.width, box_h / term.height)
    term = term.resize(
        (round(term.width * scale), round(term.height * scale)), Image.LANCZOS
    )
    tx = W - 64 - term.width
    ty = (H - term.height) // 2
    radius = 10

    # Soft drop shadow so the window separates from the background.
    # Same rounded silhouette as the terminal itself.
    shadow = Image.new("RGB", term.size, "#000000")
    shadow_mask = Image.new("L", term.size, 0)
    ImageDraw.Draw(shadow_mask).rounded_rectangle(
        [0, 0, term.width - 1, term.height - 1], radius=radius, fill=90
    )
    canvas.paste(shadow, (tx + 8, ty + 10), shadow_mask)

    # Rounded corners + 1px mauve-tinted frame.
    mask = Image.new("L", term.size, 0)
    ImageDraw.Draw(mask).rounded_rectangle([0, 0, term.width - 1, term.height - 1], radius=radius, fill=255)
    canvas.paste(term, (tx, ty), mask)
    frame = ImageDraw.Draw(canvas)
    frame.rounded_rectangle(
        [tx - 1, ty - 1, tx + term.width, ty + term.height],
        radius=radius + 1,
        outline="#45475a",
        width=1,
    )

    f_h1_b = ImageFont.truetype(str(FONT_MONO_BOLD), 70)
    f_h1_r = ImageFont.truetype(str(FONT_MONO_REG), 70)
    f_body = ImageFont.truetype(str(FONT_MONO_REG), 27)
    f_url = ImageFont.truetype(str(FONT_MONO_BOLD), 25)
    f_cmd_b = ImageFont.truetype(str(FONT_MONO_BOLD), 23)
    f_cmd_r = ImageFont.truetype(str(FONT_MONO_REG), 23)

    LEFT = 64

    # Headline mirrors the hero h1 exactly: `$` is font-normal, `gaur` is
    # font-bold with tracking-tight (-0.025em), the parts are spaced by gap-3,
    # and `gaur` carries the .crt-glow phosphor bloom. The gaps are the hero's
    # *ink-to-ink* gaps measured at 48px (24 and 21) scaled to this 70px face.
    TRK = -0.025 * 70
    GAP1, GAP2 = round(24 * 70 / 48), round(21 * 70 / 48)
    base, top = LEFT - 4, 132

    dollar_r = base + ink(f_h1_r, "$")[2] - 1
    gx = dollar_r + GAP1 - ink(f_h1_b, "g")[0]
    starts, x = [], gx
    for glyph in "gaur":
        starts.append(x)
        x += f_h1_b.getlength(glyph) + TRK
    last_r = starts[-1] + ink(f_h1_b, "r")[2]

    draw = ImageDraw.Draw(canvas)
    draw.text((base, top), "$", font=f_h1_r, fill=GREEN)
    draw.text((gx, top), "gaur", font=f_h1_b, fill=MAUVE)
    draw.rectangle([LEFT, 96, LEFT + 96, 102], fill=MAUVE)

    # Cursor: centre it on the rendered ink band, measured *before* the bloom
    # is composited so the halo can't skew the scan.
    cur_w, cur_h = 20, 58
    cx = round(last_r + GAP2)
    band = [
        yy
        for yy in range(top - 20, top + 110)
        if any(
            canvas.getpixel((xx, yy)) != CRUST_rgb
            for xx in range(LEFT - 8, round(last_r) + 4)
        )
    ]
    cy = round((band[0] + band[-1] + 1) / 2 - cur_h / 2)

    canvas = crt_glow(canvas, (gx, top), "gaur", f_h1_b, MAUVE, CRT_GLOW)
    draw = ImageDraw.Draw(canvas)
    draw.rectangle([cx, cy, cx + cur_w, cy + cur_h], fill=CURSOR)

    for i, line in enumerate(("terminal-native", "package manager", "for arch linux")):
        draw.text((LEFT, 262 + i * 37), line, font=f_body, fill=MUTED)

    # Destination above, install command below — the hero's tagline -> CTA order.
    draw.rectangle([LEFT, 418, LEFT + 16, 444], fill=HIGHLIGHT)
    draw.text((LEFT + 32, 418), "gaur.prbhtkumr.xyz", font=f_url, fill=HIGHLIGHT)

    draw.text((LEFT, 494), "$", font=f_cmd_b, fill=GREEN)
    draw.text((LEFT + f_cmd_b.getlength("$ "), 494), "paru -S gaur-bin",
              font=f_cmd_r, fill=GREEN_90)

    OUT.parent.mkdir(parents=True, exist_ok=True)
    canvas.save(OUT, "PNG", optimize=True)


if __name__ == "__main__":
    main()
