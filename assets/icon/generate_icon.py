from pathlib import Path
from PIL import Image, ImageChops, ImageDraw, ImageFilter


ROOT = Path(__file__).resolve().parents[2]
ASSET_DIR = Path(__file__).resolve().parent
SIZE = 1024

BASE_STOPS = [
    (0.0, (47, 107, 255)),
    (0.56, (29, 174, 255)),
    (1.0, (35, 215, 164)),
]

CORE_STOPS = [
    (0.0, (217, 246, 255)),
    (0.48, (255, 255, 255)),
    (1.0, (204, 255, 241)),
]


def lerp(a, b, t):
    return a + (b - a) * t


def mix_color(a, b, t):
    return tuple(round(lerp(a[i], b[i], t)) for i in range(3))


def sample_stops(stops, t):
    t = max(0.0, min(1.0, t))
    for i in range(len(stops) - 1):
        left_t, left_color = stops[i]
        right_t, right_color = stops[i + 1]
        if t <= right_t:
            local = 0.0 if right_t == left_t else (t - left_t) / (right_t - left_t)
            return mix_color(left_color, right_color, local)
    return stops[-1][1]


def diagonal_gradient(size, stops, weight_x, weight_y):
    img = Image.new("RGBA", (size, size))
    pixels = img.load()
    denom = weight_x * (size - 1) + weight_y * (size - 1)
    for y in range(size):
        for x in range(size):
            t = (weight_x * x + weight_y * y) / denom
            pixels[x, y] = (*sample_stops(stops, t), 255)
    return img


def rounded_mask(bounds, radius):
    mask = Image.new("L", (SIZE, SIZE), 0)
    draw = ImageDraw.Draw(mask)
    draw.rounded_rectangle(bounds, radius=radius, fill=255)
    return mask


def build_base():
    base = diagonal_gradient(SIZE, BASE_STOPS, 0.62, 0.92)
    base.putalpha(rounded_mask((96, 96, 928, 928), 220))

    gloss = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    gloss_draw = ImageDraw.Draw(gloss)
    gloss_draw.ellipse((136, 102, 742, 594), fill=(255, 255, 255, 72))
    gloss = gloss.filter(ImageFilter.GaussianBlur(42))
    gloss.putalpha(ImageChops.multiply(gloss.getchannel("A"), rounded_mask((96, 96, 928, 928), 220)))
    base.alpha_composite(gloss)

    stroke = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    stroke_draw = ImageDraw.Draw(stroke)
    stroke_draw.rounded_rectangle((118, 118, 906, 906), radius=198, outline=(255, 255, 255, 52), width=2)
    base.alpha_composite(stroke)
    return base


def build_core_mask():
    mask = Image.new("L", (SIZE, SIZE), 0)
    draw = ImageDraw.Draw(mask)
    draw.rounded_rectangle((252, 396, 772, 812), radius=156, fill=255)
    draw.rounded_rectangle((338, 212, 686, 566), radius=174, fill=255)
    draw.rounded_rectangle((418, 292, 606, 492), radius=96, fill=0)
    draw.rectangle((418, 492, 606, 566), fill=0)
    return mask


def build_code_cut():
    cut = Image.new("L", (SIZE, SIZE), 0)
    draw = ImageDraw.Draw(cut)
    draw.polygon([(430, 506), (576, 596), (430, 686), (492, 686), (646, 596), (492, 506)], fill=255)
    draw.rounded_rectangle((570, 648, 730, 706), radius=29, fill=255)
    return cut


def build_core():
    core = diagonal_gradient(SIZE, CORE_STOPS, 0.5, 0.8)
    core_mask = build_core_mask()
    cut_mask = build_code_cut()
    final_mask = ImageChops.subtract(core_mask, cut_mask)
    core.putalpha(final_mask)

    shadow_mask = final_mask.filter(ImageFilter.GaussianBlur(22))
    shadow = Image.new("RGBA", (SIZE, SIZE), (9, 34, 83, 0))
    shadow.putalpha(shadow_mask)
    shadow = ImageChops.offset(shadow, 0, 20)

    accent_bar = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    accent_draw = ImageDraw.Draw(accent_bar)
    accent_draw.rounded_rectangle((290, 482, 734, 538), radius=28, fill=(255, 255, 255, 30))
    accent_bar.putalpha(ImageChops.multiply(accent_bar.getchannel("A"), final_mask))

    outline = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    outline_draw = ImageDraw.Draw(outline)
    outline_draw.rounded_rectangle((252, 396, 772, 812), radius=156, outline=(255, 255, 255, 52), width=4)
    outline_draw.rounded_rectangle((338, 212, 686, 566), radius=174, outline=(255, 255, 255, 44), width=4)
    outline.putalpha(ImageChops.multiply(outline.getchannel("A"), final_mask))

    footer = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    footer_draw = ImageDraw.Draw(footer)
    footer_draw.rounded_rectangle((242, 750, 782, 834), radius=58, fill=(12, 61, 143, 26))
    footer.putalpha(ImageChops.multiply(footer.getchannel("A"), final_mask))

    return shadow, core, accent_bar, outline, footer


def make_icon():
    canvas = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    canvas.alpha_composite(build_base())
    for layer in build_core():
        canvas.alpha_composite(layer)
    return canvas


def save_outputs(icon):
    build_dir = ROOT / "build"
    windows_dir = build_dir / "windows"

    build_dir.mkdir(parents=True, exist_ok=True)
    windows_dir.mkdir(parents=True, exist_ok=True)

    png_1024 = build_dir / "appicon-1024.png"
    png_default = build_dir / "appicon.png"
    ico_root = ROOT / "icon.ico"
    ico_windows = windows_dir / "icon.ico"

    icon.save(png_1024)
    icon.save(png_default)
    icon.save(
        ico_root,
        format="ICO",
        sizes=[(256, 256), (128, 128), (64, 64), (48, 48), (32, 32), (16, 16)],
    )
    icon.save(
        ico_windows,
        format="ICO",
        sizes=[(256, 256), (128, 128), (64, 64), (48, 48), (32, 32), (16, 16)],
    )


if __name__ == "__main__":
    save_outputs(make_icon())
