package http

import (
	"strings"

	"github.com/fasih/tabungan-siswa/internal/domain"
)

func interfaceStatusLabel(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "ACTIVE":
		return "Aktif"
	case "INACTIVE":
		return "Nonaktif"
	case "OPEN":
		return "Terbuka"
	case "CLOSING":
		return "Proses penutupan"
	case "CLOSED":
		return "Ditutup"
	case "PENDING":
		return "Menunggu"
	case "APPROVED":
		return "Disetujui"
	case "REJECTED":
		return "Ditolak"
	case "CANCELLED":
		return "Dibatalkan"
	case "SETTLED":
		return "Selesai"
	case "COMPLETED":
		return "Selesai"
	case "SUPERSEDED":
		return "Digantikan"
	case "GRADUATED":
		return "Lulus"
	case "LEFT":
		return "Keluar"
	default:
		return value
	}
}

func interfaceSettlementTypeLabel(value string) string {
	switch value {
	case string(domain.SettlementRegularYearEnd):
		return "Penutupan tahun ajaran"
	case string(domain.SettlementStudentLeaving):
		return "Siswa pindah sekolah"
	default:
		return interfaceStatusLabel(value)
	}
}

func interfaceSettlementMovementLabel(value string) string {
	switch value {
	case string(domain.SettlementCashHandover):
		return "Penyerahan uang"
	default:
		return "Pengembalian uang"
	}
}

func localizedStatusMessage(status int) string {
	switch status {
	case 400:
		return "Permintaan tidak valid. Periksa kembali data yang dikirim."
	case 401:
		return "Silakan masuk kembali untuk melanjutkan."
	case 403:
		return "Anda tidak memiliki izin untuk mengakses halaman ini."
	case 404:
		return "Halaman atau data tidak ditemukan."
	case 405:
		return "Tindakan ini tidak didukung."
	case 409:
		return "Data berubah atau bertentangan dengan kondisi saat ini. Muat ulang halaman lalu coba lagi."
	case 413:
		return "Permintaan terlalu besar. Kurangi ukuran data lalu coba lagi."
	default:
		return "Terjadi kesalahan pada server. Silakan coba lagi."
	}
}
