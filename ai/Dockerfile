# Gunakan image Python resmi sebagai base image
FROM python:3.10-slim-bullseye

# Atur direktori kerja di dalam container
WORKDIR /app

# Menginstal dependensi sistem untuk Tesseract OCR dan Poppler
# Tesseract-OCR: untuk melakukan OCR
# tesseract-ocr-ind: data bahasa Indonesia untuk Tesseract
# poppler-utils: untuk mengonversi PDF ke gambar (dibutuhkan oleh pdf2image)
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        tesseract-ocr \
        tesseract-ocr-ind \
        poppler-utils \
        libgl1-mesa-glx \
        libsm6 \
        libxext6 \
        libxrender1 \
    && rm -rf /var/lib/apt/lists/*

# Salin file requirements.txt untuk menginstal dependensi Python terlebih dahulu
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Salin seluruh kode sumber aplikasi AI Anda
COPY . .

# Ekspos port yang akan digunakan aplikasi AI (misalnya, 5000 untuk Flask)
EXPOSE 5000

# Perintah default untuk menjalankan aplikasi saat container dimulai
# Gunakan Gunicorn atau uWSGI untuk produksi, tetapi Flask built-in server cukup untuk dev
CMD ["flask", "run", "--host=0.0.0.0", "--port=5000"]
# Atau jika Anda menggunakan Gunicorn untuk produksi:
# CMD ["gunicorn", "-w", "4", "-b", "0.0.0.0:5000", "main:app"]