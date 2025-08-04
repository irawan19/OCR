from paddleocr import PaddleOCR
import sys
import os
import numpy as np
import difflib
import paddle # Import library PaddlePaddle

# ---
# Verifikasi dukungan GPU di PaddlePaddle
# ---
try:
    if paddle.fluid.is_compiled_with_cuda():
        print("[INFO] PaddlePaddle terinstal dengan dukungan CUDA (GPU).")
    else:
        print("[INFO] PaddlePaddle terinstal tanpa dukungan CUDA (CPU).")
except Exception as e:
    print(f"[ERROR] Gagal memeriksa dukungan CUDA: {e}")
    print("[INFO] Pastikan PaddlePaddle dan CUDA Toolkit terinstal dengan benar.")
# ---

ocr = PaddleOCR(use_angle_cls=True, lang='latin', use_gpu=True)

# Label-label yang umum dalam dokumen KTP/KK
KNOWN_LABELS = [
    "PROVINSI", "KABUPATEN", "NIK", "NAMA", "TEMPAT/TGL LAHIR",
    "JENIS KELAMIN", "GOL. DARAH", "ALAMAT", "RT/RW", "KEL/DESA",
    "KECAMATAN", "AGAMA", "STATUS PERKAWINAN", "PEKERJAAN",
    "KEWARGANEGARAAN", "BERLAKU HINGGA"
]

def fuzzy_label(text):
    text = text.strip().upper()
    match = difflib.get_close_matches(text, KNOWN_LABELS, n=1, cutoff=0.6)
    return match[0] if match else text

def extract_text(image_path):
    if not os.path.exists(image_path):
        raise FileNotFoundError(f"File '{image_path}' tidak ditemukan.")

    result = ocr.ocr(image_path, cls=True)
    if not result or not result[0]:
        return "Teks tidak terdeteksi."

    ocr_lines = []
    for line in result[0]:
        box, (text, conf) = line
        if not text.strip():
            continue
        box = np.array(box, dtype=np.float32)
        x_center = np.mean(box[:, 0])
        y_center = np.mean(box[:, 1])
        width = box[1, 0] - box[0, 0]
        ocr_lines.append({'text': text.strip(), 'x': x_center, 'y': y_center, 'width': width, 'box': box})

    ocr_lines.sort(key=lambda x: x['y'])

    image_width = max(line['box'][1, 0] for line in ocr_lines)
    center_x = image_width / 2
    avg_width = np.mean([l['width'] for l in ocr_lines])

    header, left, right = [], [], []

    for line in ocr_lines:
        if line['y'] < ocr_lines[0]['y'] + 50 and line['width'] > image_width * 0.5:
            header.append(line)
        elif line['x'] < center_x - (avg_width / 2):
            left.append(line)
        elif line['x'] > center_x + (avg_width / 2):
            right.append(line)
        else:
            left.append(line)

    left.sort(key=lambda x: x['y'])
    right.sort(key=lambda x: x['y'])

    combined_text = [line['text'] for line in header]

    right_idx = 0
    for left_line in left:
        label = fuzzy_label(left_line['text'])
        match_found = False
        while right_idx < len(right):
            right_line = right[right_idx]
            if abs(left_line['y'] - right_line['y']) < 20:
                combined_text.append(f"{label}: {right_line['text']}")
                right_idx += 1
                match_found = True
                break
            elif right_line['y'] > left_line['y']:
                break
            right_idx += 1
        if not match_found:
            combined_text.append(label)

    for i in range(right_idx, len(right)):
        combined_text.append(right[i]['text'])

    return "\n".join(combined_text).strip()

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Penggunaan: python extractor.py path/to/image.jpg")
        sys.exit(1)

    image_path = sys.argv[1]

    try:
        text = extract_text(image_path)
        print(text)
    except Exception as e:
        print(f"[ERROR] {e}")
        sys.exit(1)