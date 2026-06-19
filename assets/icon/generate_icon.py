from pathlib import Path
from PIL import Image, ImageChops, ImageDraw, ImageFilter


ROOT = Path(__file__).resolve().parents[2]
ASSET_DIR = Path(__file__).resolve().parent
SIZE = 1024

BASE_STOPS = [
    (0.0, (122, 92, 255)),
    (0.58, (200, 77, 255)),
    (1.0, (255, 111, 174)),
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
    base = diagonal_gradient(SIZE, BASE_STOPS, 0.7, 0.86)
    base.putalpha(rounded_mask((104, 104, 920, 920), 210))

    glow = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    glow_draw = ImageDraw.Draw(glow)
    glow_draw.ellipse((146, 110, 642, 494), fill=(255, 255, 255, 76))
    glow = glow.filter(ImageFilter.GaussianBlur(42))
    glow.putalpha(ImageChops.multiply(glow.getchannel("A"), rounded_mask((104, 104, 920, 920), 210)))
    base.alpha_composite(glow)

    stroke = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    stroke_draw = ImageDraw.Draw(stroke)
    stroke_draw.rounded_rectangle((126, 126, 898, 898), radius=188, outline=(255, 255, 255, 46), width=2)
    base.alpha_composite(stroke)
    return base


def build_robot_mask():
    mask = Image.new("L", (SIZE, SIZE), 0)
    draw = ImageDraw.Draw(mask)
    draw.rounded_rectangle((350, 350, 674, 690), radius=120, fill=255)
    return mask


def build_robot():
    robot_mask = build_robot_mask()
    final_mask = robot_mask

    shadow_mask = final_mask.filter(ImageFilter.GaussianBlur(22))
    shadow = Image.new("RGBA", (SIZE, SIZE), (71, 31, 139, 0))
    shadow.putalpha(shadow_mask)
    shadow = ImageChops.offset(shadow, 0, 24)

    robot = Image.new("RGBA", (SIZE, SIZE), (255, 255, 255, 0))
    robot.putalpha(final_mask)

    outline = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    outline_draw = ImageDraw.Draw(outline)
    outline_draw.rounded_rectangle((350, 350, 674, 690), radius=120, outline=(244, 223, 255, 255), width=4)
    outline.putalpha(ImageChops.multiply(outline.getchannel("A"), final_mask))

    antennas = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    antennas_draw = ImageDraw.Draw(antennas)
    antennas_draw.rounded_rectangle((416, 296, 450, 382), radius=17, fill=(255, 255, 255, 255))
    antennas_draw.rounded_rectangle((574, 296, 608, 382), radius=17, fill=(255, 255, 255, 255))

    eyes = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    eyes_draw = ImageDraw.Draw(eyes)
    eyes_draw.ellipse((420, 508, 464, 552), fill=(154, 121, 255, 255))
    eyes_draw.ellipse((560, 508, 604, 552), fill=(154, 121, 255, 255))

    mouth = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    mouth_draw = ImageDraw.Draw(mouth)
    mouth_draw.rounded_rectangle((460, 606, 564, 624), radius=9, fill=(176, 141, 255, 255))

    return shadow, robot, outline, antennas, eyes, mouth


def make_icon():
    canvas = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    canvas.alpha_composite(build_base())
    for layer in build_robot():
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
