package http

import "testing"

func TestInterfaceStatusLabelUsesBahasaIndonesia(t *testing.T) {
	cases := map[string]string{
		"ACTIVE":     "Aktif",
		"INACTIVE":   "Nonaktif",
		"OPEN":       "Terbuka",
		"CLOSING":    "Proses penutupan",
		"CLOSED":     "Ditutup",
		"PENDING":    "Menunggu",
		"APPROVED":   "Disetujui",
		"REJECTED":   "Ditolak",
		"CANCELLED":  "Dibatalkan",
		"SETTLED":    "Selesai",
		"COMPLETED":  "Selesai",
		"SUPERSEDED": "Digantikan",
		"GRADUATED":  "Lulus",
		"LEFT":       "Keluar",
	}
	for input, want := range cases {
		t.Run(input, func(t *testing.T) {
			if got := interfaceStatusLabel(input); got != want {
				t.Errorf("interfaceStatusLabel(%q) = %q, want %q", input, got, want)
			}
		})
	}
}

func TestInterfaceSettlementLabelsAreUnderstandable(t *testing.T) {
	if got := interfaceSettlementTypeLabel("STUDENT_LEAVING"); got != "Siswa pindah sekolah" {
		t.Errorf("interfaceSettlementTypeLabel(STUDENT_LEAVING) = %q", got)
	}
	if got := interfaceSettlementMovementLabel("HANDOVER"); got != "Penyerahan uang" {
		t.Errorf("interfaceSettlementMovementLabel(HANDOVER) = %q", got)
	}
}
