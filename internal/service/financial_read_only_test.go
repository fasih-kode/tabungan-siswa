package service

import (
	"context"
	"errors"
	"testing"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/google/uuid"
)

func TestAdminFinancialWritesForbidden(t *testing.T) {
	actor := Actor{UserID: uuid.New(), Role: domain.RoleAdmin}
	ctx := context.Background()
	// Empty dependencies ensure authorization happens before accessing any data.
	cases := map[string]func() error{
		"create account": func() error {
			_, err := (&savingsAccountService{}).Create(ctx, CreateSavingsAccountInput{Actor: actor})
			return err
		},
		"deposit": func() error { _, err := (&transactionService{}).Deposit(ctx, DepositInput{Actor: actor}); return err },
		"withdraw": func() error {
			_, err := (&transactionService{}).Withdrawal(ctx, WithdrawalInput{Actor: actor})
			return err
		},
		"edit": func() error {
			_, err := (&transactionService{}).Edit(ctx, EditTransactionInput{Actor: actor})
			return err
		},
		"cancel": func() error {
			_, err := (&transactionService{}).Cancel(ctx, CancelTransactionInput{Actor: actor})
			return err
		},
		"settle year": func() error {
			_, err := (&settlementService{}).SettleYearEnd(ctx, SettleYearEndInput{Actor: actor})
			return err
		},
		"settle leaving": func() error {
			_, err := (&settlementService{}).SettleStudentLeaving(ctx, SettleStudentLeavingInput{Actor: actor})
			return err
		},
		"reopen": func() error {
			_, err := (&settlementService{}).Reopen(ctx, ReopenSettlementInput{Actor: actor})
			return err
		},
	}
	for name, call := range cases {
		t.Run(name, func(t *testing.T) {
			if err := call(); !errors.Is(err, ErrForbidden) {
				t.Fatalf("got %v, want forbidden", err)
			}
		})
	}
}

// Reopening used to be admin-only. Read-only supervision withdraws that
// permission without granting it implicitly to teachers or students.
func TestSettlementReopenForbiddenForAllRoles(t *testing.T) {
	for _, role := range []domain.UserRole{domain.RoleAdmin, domain.RoleWaliKelas, domain.RoleSiswa} {
		t.Run(string(role), func(t *testing.T) {
			_, err := (&settlementService{}).Reopen(context.Background(), ReopenSettlementInput{
				Actor: Actor{UserID: uuid.New(), Role: role}, SettlementID: uuid.New(),
			})
			if !errors.Is(err, ErrForbidden) {
				t.Fatalf("Reopen() = %v, want forbidden", err)
			}
		})
	}
}
