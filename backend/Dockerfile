# Gunakan image Go resmi sebagai base image untuk membangun aplikasi
FROM golang:1.22-alpine AS builder

# Atur direktori kerja di dalam container
WORKDIR /app

# Salin go.mod dan go.sum untuk mengunduh dependensi terlebih dahulu
# Ini memanfaatkan layer caching Docker; jika dependensi tidak berubah, langkah ini akan dilewati
COPY go.mod go.sum ./
RUN go mod download

# Salin seluruh kode sumber aplikasi ke dalam direktori kerja
COPY . .

# Build aplikasi Go
# -ldflags="-s -w" mengurangi ukuran binary dan menghilangkan informasi debugging
# -o ocr-backend menentukan nama executable output
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix nocgo -ldflags="-s -w" -o ocr-backend .

# Gunakan image Alpine yang lebih kecil untuk runtime
FROM alpine:latest

# Instal sertifikat CA untuk permintaan HTTPS (penting untuk beberapa operasi jaringan)
RUN apk --no-cache add ca-certificates

# Atur direktori kerja
WORKDIR /root/

# Salin executable yang sudah dibangun dari fase builder
COPY --from=builder /app/ocr-backend .

# Salin folder docs (untuk Swagger UI)
COPY --from=builder /app/docs ./docs

# Salin file .env ke dalam container
# Penting: Di lingkungan produksi, Anda mungkin ingin mengelola variabel lingkungan melalui orkestrator (e.g., Kubernetes, Docker Compose env_file)
# Namun, untuk dev/test, menyalinnya langsung tidak masalah asalkan file .env tidak berisi rahasia yang terhardcode.
COPY --from=builder /app/.env ./.env

# Ekspos port yang akan digunakan aplikasi (misalnya, 8080 untuk Gin)
EXPOSE 8080

# Perintah default untuk menjalankan aplikasi saat container dimulai
CMD ["./ocr-backend"]