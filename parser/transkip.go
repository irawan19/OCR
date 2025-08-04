package parser

import (
	"regexp"
	"strings"
)

type Transkrip struct {
	Nama           string `json:"nama"`
	TempatTglLahir string `json:"tempat_tanggal_lahir"`
	Valid          bool   `json:"valid"`
}

func ParseTranskrip(text string) Transkrip {
	text = strings.ToUpper(text)
	text = strings.ReplaceAll(text, "\r\n", " ")
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.Join(strings.Fields(text), " ")

	tempatTgl := extractTempatTglLahir(text)
	nama := extractNamaDekatTTL(text, tempatTgl)

	valid := nama != "" && tempatTgl != ""

	return Transkrip{
		Nama:           nama,
		TempatTglLahir: tempatTgl,
		Valid:          valid,
	}
}

func extractTempatTglLahir(text string) string {
	re := regexp.MustCompile(`([A-Z]{3,})(,| [ -])\s*(\d{1,2})\s*([A-Z]+)\s*(\d{4})`)
	match := re.FindStringSubmatch(text)
	if len(match) >= 6 {
		tempat := match[1]
		tanggal := match[3]
		bulan := match[4]
		tahun := match[5]
		return tempat + ", " + tanggal + " " + bulan + " " + tahun
	}
	return ""
}

func extractNamaDekatTTL(text, ttl string) string {
	if ttl == "" {
		return ""
	}

	idx := strings.Index(text, ttl)
	if idx == -1 {
		return ""
	}

	// Ambil 200 karakter sebelum TTL
	start := idx - 200
	if start < 0 {
		start = 0
	}
	sub := text[start:idx]

	// Ambil kandidat nama (2-4 kata kapital, exclude kata umum institusi)
	re := regexp.MustCompile(`([A-Z]{2,}(?:\s+[A-Z]{2,}){1,3})`)
	candidates := re.FindAllString(sub, -1)

	// Prioritaskan kandidat yang tidak mengandung kata noise
	for i := len(candidates) - 1; i >= 0; i-- {
		nama := candidates[i]
		if !containsNoiseWord(nama) {
			return strings.TrimSpace(nama)
		}
	}

	return ""
}

func containsNoiseWord(s string) bool {
	noise := []string{"UNIVERSITAS", "SEKOLAH", "TEKNIK", "ILMU", "FAKULTAS", "KOMPUTER", "PROGRAM", "STUDI"}
	for _, word := range noise {
		if strings.Contains(s, word) {
			return true
		}
	}
	return false
}
