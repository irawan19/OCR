package parser

import (
	"regexp"
	"strings"
)

type Ijazah struct {
	Nama           string `json:"nama"`
	TempatTglLahir string `json:"tempat_tanggal_lahir"`
	TanggalLulus   string `json:"tanggal_lulus"`
	Valid          bool   `json:"valid"`
}

func ParseIjazah(text string) Ijazah {
	text = normalizeTextIjazah(text)

	nama := extractNamaIjazah(text)
	tempatTgl := extractTempatTanggalLahir(text)
	tanggalLulus := extractTanggalLulus(text)

	valid := nama != "" && tempatTgl != "" && tanggalLulus != ""

	return Ijazah{
		Nama:           nama,
		TempatTglLahir: tempatTgl,
		TanggalLulus:   tanggalLulus,
		Valid:          valid,
	}
}

func normalizeTextIjazah(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.Join(strings.Fields(text), " ")
	return text
}

func extractNamaIjazah(text string) string {
	re := regexp.MustCompile(`(?i)Diberikan\s+Kepada\s*[:\-]?\s*([A-Za-z\s]+)`)
	match := re.FindStringSubmatch(text)
	if len(match) > 1 {
		nama := strings.TrimSpace(match[1])
		stopWords := []string{"Tempat", "Tanggal", "Lahir"}
		for _, stop := range stopWords {
			if idx := strings.Index(nama, stop); idx != -1 {
				nama = strings.TrimSpace(nama[:idx])
				break
			}
		}
		return nama
	}

	re2 := regexp.MustCompile(`(?i)DIBERIKAN\s+KEPADA\s+([A-Za-z\s]+)`)
	match2 := re2.FindStringSubmatch(text)
	if len(match2) > 1 {
		nama := strings.TrimSpace(match2[1])
		stopWords := []string{"Tempat", "Tanggal", "Lahir"}
		for _, stop := range stopWords {
			if idx := strings.Index(nama, stop); idx != -1 {
				nama = strings.TrimSpace(nama[:idx])
				break
			}
		}
		return nama
	}

	return ""
}

func extractTempatTanggalLahir(text string) string {
	re := regexp.MustCompile(`(?i)Tempat.*?Lahir\s*[:\-]?\s*([A-Z][a-z]+,\s*\d{1,2}\s+[A-Z][a-z]+(?:\s+\d{4})?)`)
	match := re.FindStringSubmatch(text)
	if len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

func extractTanggalLulus(text string) string {
	reKalimat := regexp.MustCompile(`(?i)([^\.]{0,50}(lulus|dinyatakan lulus)[^\.]{0,100})`)
	kalimatLulus := reKalimat.FindString(text)

	reTanggal := regexp.MustCompile(`(?i)(\d{1,2}\s+(Januari|Februari|Maret|April|Mei|Juni|Juli|Agustus|September|Oktober|November|Desember)\s+\d{4})`)
	match := reTanggal.FindString(kalimatLulus)

	if match != "" {
		return strings.TrimSpace(match)
	}

	allDates := reTanggal.FindAllString(text, -1)
	if len(allDates) > 0 {
		return strings.TrimSpace(allDates[len(allDates)-1])
	}

	return ""
}
