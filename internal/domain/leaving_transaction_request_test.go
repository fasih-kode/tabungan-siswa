package domain_test

import (
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/google/uuid"
)

func TestLeavingTransactionRequestStateIsOneWay(t *testing.T) {
	date := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	request, err := domain.NewLeavingTransactionRequest(uuid.New(), uuid.New(), domain.TransactionDeposit, 10000, date, "terlambat", "KWT-1", uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	adminID := uuid.New()
	if err := request.Decide(adminID, true, "bukti cocok"); err != nil {
		t.Fatal(err)
	}
	if err := request.Decide(adminID, false, "ubah keputusan"); err == nil {
		t.Fatal("second decision was accepted")
	}
	if err := request.Complete(); err != nil {
		t.Fatal(err)
	}
	if err := request.Complete(); err == nil {
		t.Fatal("second completion was accepted")
	}
}

func TestLeavingTransactionRequestRequiresVerifiableReferenceAndReason(t *testing.T) {
	date := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct{ name, reason, evidence string }{{"reason", "", "KWT-1"}, {"evidence", "alasan", ""}} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := domain.NewLeavingTransactionRequest(uuid.New(), uuid.New(), domain.TransactionDeposit, 10000, date, tc.reason, tc.evidence, uuid.New()); err == nil {
				t.Fatal("expected invalid request")
			}
		})
	}
}
