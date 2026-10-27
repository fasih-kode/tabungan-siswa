package service

import (
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/google/uuid"
)

func TestCalendarDateComparison(t *testing.T) {
	date := time.Date(2026, time.October, 27, 0, 0, 0, 0, time.UTC)
	if calendarDateAfter(date, date) || calendarDateAfter(date, date.AddDate(0, 0, 1)) {
		t.Fatal("tanggal yang sama atau lebih awal dianggap melewati batas")
	}
	if !calendarDateAfter(date.AddDate(0, 0, 1), date) {
		t.Fatal("tanggal kalender setelah batas tidak terdeteksi")
	}
	if !calendarDateWithin(date, date, date.AddDate(0, 0, 1)) || calendarDateWithin(date.AddDate(0, 0, 1), date, date.AddDate(0, 0, 1)) {
		t.Fatal("batas tahun ajaran harus inklusif di awal dan eksklusif di akhir")
	}
}

func TestReadinessHandoverNet(t *testing.T) {
	settlementID, actorID := uuid.New(), uuid.New()
	confirmedAt := time.Now().UTC()
	makeMovement := func(kind domain.SettlementCashMovementKind, amount domain.Money) domain.SettlementCashMovement {
		return domain.SettlementCashMovement{ID: uuid.New(), SavingsSettlementID: settlementID, Kind: kind, Amount: amount, ConfirmedBy: actorID, ConfirmedAt: confirmedAt}
	}

	tests := []struct {
		name      string
		movements []domain.SettlementCashMovement
		wantNet   domain.Money
		wantValid bool
	}{
		{name: "kosong", wantValid: true},
		{name: "handover", movements: []domain.SettlementCashMovement{makeMovement(domain.SettlementCashHandover, 100)}, wantNet: 100, wantValid: true},
		{name: "dikurangi pengembalian", movements: []domain.SettlementCashMovement{makeMovement(domain.SettlementCashHandover, 100), makeMovement(domain.SettlementCashReturn, 20)}, wantNet: 80, wantValid: true},
		{name: "ID settlement salah", movements: []domain.SettlementCashMovement{{ID: uuid.New(), SavingsSettlementID: uuid.New(), Kind: domain.SettlementCashHandover, Amount: 1, ConfirmedBy: actorID, ConfirmedAt: confirmedAt}}, wantValid: false},
		{name: "jenis tidak dikenal", movements: []domain.SettlementCashMovement{makeMovement("UNKNOWN", 1)}, wantValid: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotNet, gotValid := readinessHandoverNet(settlementID, tc.movements)
			if gotNet != tc.wantNet || gotValid != tc.wantValid {
				t.Fatalf("readinessHandoverNet()=(%d,%t), want (%d,%t)", gotNet, gotValid, tc.wantNet, tc.wantValid)
			}
		})
	}
}

func TestStudentLeavingReadinessBlockersSortDeterministically(t *testing.T) {
	output := StudentLeavingReadinessOutput{}
	accountB, accountA := uuid.MustParse("00000000-0000-0000-0000-000000000002"), uuid.MustParse("00000000-0000-0000-0000-000000000001")
	record := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	output.addBlocker(LeavingBlockerLateRequestPending, &accountA, &record, "a")
	output.addBlocker(LeavingBlockerAccountOpen, &accountB, nil, "b")
	output.addBlocker(LeavingBlockerAccountOpenYearClosed, &accountA, nil, "c")
	if len(output.Blockers) != 3 || output.Blockers[0].AccountID == nil || *output.Blockers[0].AccountID != accountA ||
		output.Blockers[0].Code != LeavingBlockerAccountOpenYearClosed || output.Blockers[1].Code != LeavingBlockerLateRequestPending ||
		output.Blockers[2].AccountID == nil || *output.Blockers[2].AccountID != accountB {
		t.Fatalf("blocker order=%+v", output.Blockers)
	}
}
