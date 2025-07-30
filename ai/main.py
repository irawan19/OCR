from flask import Flask, request, jsonify
import os
import io
from PIL import Image
import json
import logging
import re

from paddleocr import PaddleOCR
import numpy as np
import cv2
from pdf2image import convert_from_bytes

app = Flask(__name__)
logging.basicConfig(level=logging.DEBUG, format='%(asctime)s - %(levelname)s - %(message)s')

# --- Utility Functions ---
def normalize_text(text):
    text = text.replace('\r', '')
    text = re.sub(r'\s+', ' ', text)
    return text.strip()

# --- Parsing KK ---
def parse_kk_structured(text: str) -> dict:
    lines = [line.strip() for line in text.splitlines() if line.strip()]
    data = {
        "nomor_kk": "",
        "nama_kepala_keluarga": "",
        "alamat": "",
        "rt_rw": "",
        "desa_kelurahan": "",
        "kecamatan": "",
        "kabupaten_kota": "",
        "provinsi": "",
        "kode_pos": "",
        "anggota_keluarga": []
    }

    for i, line in enumerate(lines):
        # Nomor KK
        if not data["nomor_kk"]:
            match = re.search(r'(?:No(?:\.|:)?\s*)(\d{16})', line)
            if match:
                data["nomor_kk"] = match.group(1)

        # RT/RW
        if not data["rt_rw"]:
            match = re.search(r'(\d{1,3}/\d{1,3})', line)
            if match:
                data["rt_rw"] = match.group(1)

        # Kode Pos
        if not data["kode_pos"]:
            match = re.search(r'Kode Pos\s*(\d{5})', line)
            if match:
                data["kode_pos"] = match.group(1)
            else:
                match = re.search(r'(\d{5})$', line)
                if match:
                    data["kode_pos"] = match.group(1)

        # Alamat
        if "alamat" in line.lower() and not data["alamat"] and i > 0:
            data["alamat"] = lines[i-1]

        # Kecamatan
        if "kecamatan" in line.lower() and not data["kecamatan"] and i > 0:
            data["kecamatan"] = lines[i-1]

        # Kabupaten/Kota
        if ("kabupaten" in line.lower() or "kota" in line.lower()) and not data["kabupaten_kota"] and i > 0:
            data["kabupaten_kota"] = lines[i-1]

        # Provinsi
        if "provinsi" in line.lower() and not data["provinsi"] and i > 0:
            data["provinsi"] = lines[i-1]

        # Desa/Kelurahan
        if ("desa" in line.lower() or "kelurahan" in line.lower()) and not data["desa_kelurahan"] and i > 0:
            data["desa_kelurahan"] = lines[i-1]

        # Nama Kepala Keluarga
        if "nama kepala keluarga" in line.lower() and not data["nama_kepala_keluarga"]:
            for j in range(i-2, i):
                if j >= 0 and len(lines[j]) > 3:
                    data["nama_kepala_keluarga"] = lines[j]
                    break

    # Anggota keluarga
    anggota = re.findall(r'(\d{16})\s+(LAKI-LAKI|PEREMPUAN)\s+(\d{2}-\d{2}-\d{4})\s+([A-Z\s\.]+)', text)
    for match in anggota:
        data["anggota_keluarga"].append({
            "nik": match[0],
            "jenis_kelamin": match[1],
            "tanggal_lahir": match[2],
            "nama": match[3].strip()
        })

    return data

# --- OCR Engine ---
class OCRModel:
    def __init__(self):
        self.ocr_engine = PaddleOCR(use_angle_cls=True, lang="id", use_gpu=False, show_log=False)
        logging.info("PaddleOCR engine initialized successfully.")

    def _preprocess_image(self, image_np):
        if image_np is None or image_np.size == 0:
            logging.warning("Input image for preprocessing is empty or None.")
            return image_np
        gray = cv2.cvtColor(image_np, cv2.COLOR_BGR2GRAY)
        denoised = cv2.medianBlur(gray, 3)
        thresh = cv2.adaptiveThreshold(denoised, 255, cv2.ADAPTIVE_THRESH_GAUSSIAN_C, 
                                       cv2.THRESH_BINARY, 15, 5)
        return thresh

    def extract_text(self, file_bytes, mime_type):
        text_data = ""
        images_to_process = []

        if "pdf" in mime_type:
            pil_images = convert_from_bytes(file_bytes)
            for image_pil in pil_images:
                img_np = cv2.cvtColor(np.array(image_pil), cv2.COLOR_RGB2BGR)
                processed_img_np = self._preprocess_image(img_np)
                images_to_process.append(processed_img_np)
        elif "image" in mime_type:
            image_pil = Image.open(io.BytesIO(file_bytes))
            img_np = cv2.cvtColor(np.array(image_pil), cv2.COLOR_RGB2BGR)
            processed_img_np = self._preprocess_image(img_np)
            images_to_process.append(processed_img_np)
        else:
            raise ValueError("Unsupported file type for OCR.")

        for img_np in images_to_process:
            result = self.ocr_engine.ocr(img_np, cls=True)
            if result and result[0]:
                sorted_lines = sorted(result[0], key=lambda item: (item[0][0][1], item[0][0][0]))
                for line in sorted_lines:
                    text_data += line[1][0] + "\n"

        return text_data.strip()

    def detect_doc_type(self, text):
        lower = text.lower()
        if "kartu keluarga" in lower:
            return "kk"
        elif "nik" in lower and "nama" in lower:
            return "ktp"
        elif "ijazah" in lower:
            return "ijazah"
        elif "transkrip" in lower or "ipk" in lower:
            return "transkrip"
        return "unknown"

    def parse_doc(self, text, doc_type):
        text = normalize_text(text)
        if doc_type == "ktp":
            return {
                "ktp_nik": re.search(r'NIK\s*[:\.]?\s*(\d{16})', text).group(1) if re.search(r'NIK\s*[:\.]?\s*(\d{16})', text) else "",
                "ktp_nama": re.search(r'Nama\s*[:\.]?\s*([A-Za-z\s]+)', text).group(1).strip() if re.search(r'Nama\s*[:\.]?\s*([A-Za-z\s]+)', text) else ""
            }
        elif doc_type == "kk":
            return parse_kk_structured(text)
        elif doc_type == "ijazah":
            return {
                "nama_lengkap": re.search(r'(?:Nama Lengkap|Nama)\s*[:\.]?\s*([A-Za-z\s]+)', text).group(1).strip() if re.search(r'(?:Nama Lengkap|Nama)\s*[:\.]?\s*([A-Za-z\s]+)', text) else "",
                "nim": re.search(r'NIM\s*[:\.]?\s*([A-Za-z0-9]+)', text).group(1) if re.search(r'NIM\s*[:\.]?\s*([A-Za-z0-9]+)', text) else "",
                "nomor_ijazah": re.search(r'Nomor Ijazah\s*[:\.]?\s*([A-Z0-9/\.\-]+)', text).group(1) if re.search(r'Nomor Ijazah\s*[:\.]?\s*([A-Z0-9/\.\-]+)', text) else ""
            }
        elif doc_type == "transkrip":
            match = re.search(r'IPK\s*[:=]?\s*(\d[\.,]\d{2})', text)
            return {"ipk": float(match.group(1).replace(",", ".")) if match else 0.0}
        return {}

ocr_model = OCRModel()

@app.route('/process-document', methods=['POST'])
def process_document():
    if 'document' not in request.files:
        return jsonify({"error": "No document uploaded"}), 400

    file = request.files['document']
    file_bytes = file.read()
    mime_type = file.content_type
    filename = file.filename

    try:
        raw_text = ocr_model.extract_text(file_bytes, mime_type)
        doc_type_hint = request.form.get('doc_type', '').lower()
        doc_type = doc_type_hint or ocr_model.detect_doc_type(raw_text)
        structured = ocr_model.parse_doc(raw_text, doc_type)

        return jsonify({
            "text_raw": raw_text,
            "doc_type": doc_type,
            "extracted_data": structured,
            "file_info": {
                "filename": filename,
                "mime_type": mime_type,
                "size": len(file_bytes)
            }
        }), 200
    except Exception as e:
        logging.exception("OCR processing failed")
        return jsonify({"error": str(e)}), 500

@app.route('/validate-scholarship', methods=['POST'])
def validate_scholarship():
    data = request.get_json()
    app_data = data.get("application_data")
    extracted = data.get("extracted_data")

    is_valid = bool(app_data and extracted)
    notes = "Validasi sederhana selesai. Harap implementasi penuh." if is_valid else "Data tidak lengkap."

    return jsonify({
        "is_valid": is_valid,
        "notes": notes
    }), 200

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5000, debug=True)