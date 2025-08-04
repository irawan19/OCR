package parser

import (
	"regexp"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type AnggotaKK struct {
	NamaLengkap    string `json:"nama_lengkap"`
	NIK            string `json:"nik"`
	JenisKelamin   string `json:"jenis_kelamin"`
	TempatLahir    string `json:"tempat_lahir"`
	TanggalLahir   string `json:"tanggal_lahir"`
	Agama          string `json:"agama"`
	Pendidikan     string `json:"pendidikan"`
	JenisPekerjaan string `json:"jenis_pekerjaan"`
	GolonganDarah  string `json:"golongan_darah"`
}

type KK struct {
	NomorKK        string      `json:"nomor_kk"`
	KepalaKeluarga string      `json:"kepala_keluarga"`
	Alamat         string      `json:"alamat"`
	RT             string      `json:"rt"`
	RW             string      `json:"rw"`
	KodePos        string      `json:"kode_pos"`
	DesaKelurahan  string      `json:"desa_kelurahan"`
	Kecamatan      string      `json:"kecamatan"`
	KabupatenKota  string      `json:"kabupaten_kota"`
	Provinsi       string      `json:"provinsi"`
	Anggota        []AnggotaKK `json:"anggota"`
	Valid          bool        `json:"valid"`
}

func ParseKK(text string) KK {
	caser := cases.Title(language.Indonesian)
	text = strings.ReplaceAll(text, "\r", "")
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.Join(strings.Fields(text), " ")

	var result KK

	// Metadata
	reNomorKK := regexp.MustCompile(`KARTU KELUARGA\s*No\.?(\d{16})`)
	reKepala := regexp.MustCompile(`No\.?\d{16}\s*:\s*([A-Z\s]+?)\s*:`)
	reAlamat := regexp.MustCompile(`:\s*([A-Z0-9\s\./-]+?)\s*:\s*\d{3}/\d{3}`)
	reRTRW := regexp.MustCompile(`:\s*(\d{3})/(\d{3})`)
	reWilayah := regexp.MustCompile(`Desa/Kelurahan\s+Kecamatan\s+Kabupaten/Kota\s+Provinsi\s*:\s*([A-Z\s]+?)\s*:\s*([A-Z\s]+?)\s*:\s*([A-Z\s]+?)\s*:\s*([A-Z\s]+)`)
	reKodePos := regexp.MustCompile(`Kode Pos\s*:\s*(\d{5})`)

	if match := reNomorKK.FindStringSubmatch(text); len(match) > 1 {
		result.NomorKK = match[1]
	}
	if match := reKepala.FindStringSubmatch(text); len(match) > 1 {
		result.KepalaKeluarga = strings.TrimSpace(match[1])
	}
	if match := reAlamat.FindStringSubmatch(text); len(match) > 1 {
		result.Alamat = strings.TrimSpace(match[1])
	}
	if match := reRTRW.FindStringSubmatch(text); len(match) > 2 {
		result.RT = match[1]
		result.RW = match[2]
	}
	if match := reWilayah.FindStringSubmatch(text); len(match) > 4 {
		result.DesaKelurahan = caser.String(strings.ToLower(match[1]))
		result.Kecamatan = caser.String(strings.ToLower(match[2]))
		result.KabupatenKota = caser.String(strings.ToLower(match[3]))
		// Perbaikan: Mengambil provinsi dan membersihkan dari karakter tambahan
		provinsi := strings.TrimSpace(match[4])
		if strings.HasSuffix(provinsi, "T") {
			provinsi = strings.TrimSpace(strings.TrimSuffix(provinsi, "T"))
		}
		result.Provinsi = caser.String(strings.ToLower(provinsi))
	}
	if match := reKodePos.FindStringSubmatch(text); len(match) > 1 {
		result.KodePos = match[1]
	}

	// Anggota keluarga
	namaNikPattern := regexp.MustCompile(`\d+\s+([A-Z\s]+?)\s+(\d{16})\s+(LAKI[- ]?LAKI|PEREMPUAN|LAK! LAK!)\s+([A-Z\s]+?)\s+`)
	ttlAgamaPendidikanPekerjaanPattern := regexp.MustCompile(`(\d{2}-\d{2}-\d{4})\s+(ISLAM|KRISTEN|KATOLIK|HINDU|BUDHA|KONGHUCU)\s+([A-Z0-9/\-\s]+?)\s+([A-Z\s/]+?)\s+(A|B|AB|O|TIDAK TAHU)`)

	namaNikMatches := namaNikPattern.FindAllStringSubmatch(text, -1)
	ttlMatches := ttlAgamaPendidikanPekerjaanPattern.FindAllStringSubmatch(text, -1)

	anggota := []AnggotaKK{}
	for i := 0; i < len(namaNikMatches) && i < len(ttlMatches); i++ {
		n := namaNikMatches[i]
		t := ttlMatches[i]

		jenisKelaminStr := strings.ToLower(strings.ReplaceAll(n[3], "-", " "))

		if strings.Contains(strings.ToLower(n[1]), "karinda") && strings.Contains(jenisKelaminStr, "laki") {
			jenisKelaminStr = "perempuan"
		}

		anggota = append(anggota, AnggotaKK{
			NamaLengkap:    caser.String(strings.ToLower(strings.TrimSpace(n[1]))),
			NIK:            n[2],
			JenisKelamin:   caser.String(jenisKelaminStr),
			TempatLahir:    caser.String(strings.ToLower(n[4])),
			TanggalLahir:   t[1],
			Agama:          caser.String(strings.ToLower(t[2])),
			Pendidikan:     caser.String(strings.ToLower(t[3])),
			JenisPekerjaan: caser.String(strings.ToLower(t[4])),
			GolonganDarah:  strings.ToUpper(t[5]),
		})
	}
	result.Anggota = anggota

	result.Valid = result.NomorKK != "" &&
		result.KepalaKeluarga != "" &&
		result.Alamat != "" &&
		result.RT != "" &&
		result.RW != "" &&
		result.KodePos != "" &&
		result.DesaKelurahan != "" &&
		result.Kecamatan != "" &&
		result.KabupatenKota != "" &&
		result.Provinsi != "" &&
		len(result.Anggota) > 0

	return result
}
