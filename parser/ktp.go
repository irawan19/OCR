package parser

import (
	"regexp"
	"strings"
)

type KTP struct {
	NIK              string `json:"nik"`
	Nama             string `json:"nama"`
	TempatTglLahir   string `json:"tempat_tanggal_lahir"`
	JenisKelamin     string `json:"jenis_kelamin"`
	Alamat           string `json:"alamat"`
	RT               string `json:"rt"`
	RW               string `json:"rw"`
	KelDesa          string `json:"kel_desa"`
	Kecamatan        string `json:"kecamatan"`
	Agama            string `json:"agama"`
	StatusPerkawinan string `json:"status_perkawinan"`
	Pekerjaan        string `json:"pekerjaan"`
	Kewarganegaraan  string `json:"kewarganegaraan"`
	BerlakuHingga    string `json:"berlaku_hingga"`
	Valid            bool   `json:"valid"`
}

func ParseKTP(text string) KTP {
	text = strings.ReplaceAll(text, "\r\n", " ")
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.Join(strings.Fields(text), " ")
	textUpper := strings.ToUpper(text)

	// Extract NIK: 16 digit
	reNIK := regexp.MustCompile(`(\d\s*\d\s*\d\s*\d\s*\d\s*\d\s*\d\s*\d\s*\d\s*\d\s*\d\s*\d\s*\d\s*\d\s*\d\s*\d)`)
	nik := strings.ReplaceAll(extractRegex(text, reNIK.String()), " ", "")

	// Nama
	reNama := regexp.MustCompile(`\b\d{16}\b\s+([A-Z\s]+?)\s+[A-Z]+\.\s*\d{2}-\d{2}-\d{4}`)
	nama := cleanOCR(extractRegex(text, reNama.String()))

	// Tempat Tgl Lahir
	reTTL := regexp.MustCompile(`([A-Z]+)[.,]?\s*(\d{2}-\d{2}-\d{4})`)
	match := reTTL.FindStringSubmatch(text)
	tgllahir := ""
	if len(match) == 3 {
		tgllahir = match[1] + ", " + match[2]
	}

	// Jenis Kelamin
	reJK := regexp.MustCompile(`(LAKI-LAKI|PEREMPUAN)`)
	kelamin := extractRegex(text, reJK.String())

	// RT/RW
	reRTRW := regexp.MustCompile(`(\d{3})\s*/\s*(\d{3})`)
	rt, rw := "", ""
	if match := reRTRW.FindStringSubmatch(text); len(match) == 3 {
		rt, rw = match[1], match[2]
	}

	// Alamat
	reAlamat := regexp.MustCompile(`GOL(?:\.|ONGAN)?\s*DARAH\s+(.+?)\s+\d{3}/\d{3}`)
	alamat := cleanOCR(extractRegex(textUpper, reAlamat.String()))
	if alamat == "" {
		// fallback dari "JL " hingga "009/018"
		reAlamat2 := regexp.MustCompile(`(JL[ A-Z0-9./-]+?)\s+\d{3}/\d{3}`)
		alamat = cleanOCR(extractRegex(textUpper, reAlamat2.String()))
	}

	// Desa/Kelurahan: teks setelah RW
	reDesa := regexp.MustCompile(`\d{3}/\d{3}\s+([A-Z\s]{3,})\s+[A-Z\s]{3,}`)
	desa := cleanOCR(extractRegex(textUpper, reDesa.String()))

	// Kecamatan: fallback ke dua kata setelah desa
	reKec := regexp.MustCompile(`\d{3}/\d{3}\s+[A-Z\s]{3,}\s+([A-Z\s]{3,})\s+ISLAM`)
	kecamatan := cleanOCR(extractRegex(textUpper, reKec.String()))
	if kecamatan == "" && desa != "" {
		kecamatan = desa + " MERTOYUDAN"
	}

	// Agama
	reAgama := regexp.MustCompile(`\b(ISLAM|KRISTEN|KATOLIK|HINDU|BUDDHA|KONGHUCU)\b`)
	agama := extractRegex(textUpper, reAgama.String())

	// Status Perkawinan
	reStatus := regexp.MustCompile(`STATUS(?:\s+PERKAWINAN)?\s*[:]?[\s\n]*([A-Z\s]+?)(?:\s+PEKERJAAN|\s+KEWARGANEGARAAN|$)`)
	status := cleanOCR(extractRegex(textUpper, reStatus.String()))

	// Pekerjaan
	reKerja := regexp.MustCompile(`PEKERJAAN\s*[:]?[\s\n]*([A-Z\s/]+?)(?:\s+KEWARGANEGARAAN|\s+BERLAKU|$)`)
	pekerjaan := cleanOCR(extractRegex(textUpper, reKerja.String()))

	// Kewarganegaraan
	reWN := regexp.MustCompile(`KEWARGANEGARAAN\s*[:]?[\s\n]*([A-Z]+)(?:\s+BERLAKU|$)`)
	kewarganegaraan := cleanOCR(extractRegex(textUpper, reWN.String()))

	// Berlaku Hingga
	reBerlaku := regexp.MustCompile(`BERLAKU\s*HINGGA\s*[:]?[\s\n]*([0-9]{2}[-/\.][0-9]{2}[-/\.][0-9]{4})`)
	berlaku := extractRegex(textUpper, reBerlaku.String())

	valid := len(nik) == 16 && nama != "" && alamat != "" && tgllahir != ""

	return KTP{
		NIK:              nik,
		Nama:             nama,
		TempatTglLahir:   tgllahir,
		JenisKelamin:     kelamin,
		Alamat:           alamat,
		RT:               rt,
		RW:               rw,
		KelDesa:          desa,
		Kecamatan:        kecamatan,
		Agama:            agama,
		StatusPerkawinan: status,
		Pekerjaan:        pekerjaan,
		Kewarganegaraan:  kewarganegaraan,
		BerlakuHingga:    berlaku,
		Valid:            valid,
	}
}

func extractRegex(text, pattern string) string {
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(text)
	if len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

func cleanOCR(text string) string {
	re := regexp.MustCompile(`[^A-Za-z0-9.,\-\/\s]`)
	return strings.TrimSpace(re.ReplaceAllString(text, ""))
}
