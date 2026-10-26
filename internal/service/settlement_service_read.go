package service

import (
	"context"
	"errors"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

func (s *settlementService) Get(
	ctx context.Context,
	input GetSettlementInput,
) (GetSettlementOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return GetSettlementOutput{}, err
	}

	if input.SettlementID == uuid.Nil {
		return GetSettlementOutput{}, domain.ErrInvalidID
	}

	settlement, err := s.deps.Repositories.Settlements.GetByID(ctx, input.SettlementID)
	if err != nil {
		return GetSettlementOutput{}, err
	}

	account, err := s.deps.Repositories.SavingsAccounts.GetByID(ctx, settlement.SavingsAccountID)
	if err != nil {
		return GetSettlementOutput{}, err
	}

	if err := s.authorizeAccount(ctx, s.deps.Repositories, input.Actor, account); err != nil {
		return GetSettlementOutput{}, err
	}

	movements, err := s.deps.Repositories.Settlements.ListCashMovements(ctx, settlement.ID)
	if err != nil {
		return GetSettlementOutput{}, err
	}
	history, err := s.deps.Repositories.Settlements.ListBySavingsAccount(ctx, account.ID, repository.ListOptions{Limit: 1000})
	if err != nil {
		return GetSettlementOutput{}, err
	}
	student, err := s.deps.Repositories.Students.GetByID(ctx, account.StudentID)
	if err != nil {
		return GetSettlementOutput{}, err
	}
	var request *domain.SettlementReopenRequest
	if latest, e := s.deps.Repositories.Settlements.GetLatestReopenRequest(ctx, settlement.ID); e == nil {
		request = &latest
	} else if !errors.Is(e, repository.ErrNotFound) {
		return GetSettlementOutput{}, e
	}
	actorNames := make(map[uuid.UUID]string)
	if users := s.deps.Repositories.Users; users != nil {
		actorIDs := map[uuid.UUID]struct{}{settlement.ExecutedBy: {}}
		for _, item := range history {
			actorIDs[item.ExecutedBy] = struct{}{}
		}
		for _, movement := range movements {
			actorIDs[movement.ConfirmedBy] = struct{}{}
		}
		if request != nil {
			actorIDs[request.RequestedBy] = struct{}{}
			if request.DecidedBy != nil {
				actorIDs[*request.DecidedBy] = struct{}{}
			}
		}
		for actorID := range actorIDs {
			user, e := users.GetByID(ctx, actorID)
			if e != nil {
				return GetSettlementOutput{}, e
			}
			actorNames[actorID] = user.Username
		}
	}
	return GetSettlementOutput{Settlement: &settlement, History: history, Movements: movements, Student: &student, ReopenRequest: request, Account: &account, ActorNames: actorNames}, nil
}

func (s *settlementService) List(
	ctx context.Context,
	input ListSettlementsInput,
) (ListSettlementsOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return ListSettlementsOutput{}, err
	}

	if input.SavingsAccountID == uuid.Nil {
		return ListSettlementsOutput{}, domain.ErrInvalidID
	}

	account, err := s.deps.Repositories.SavingsAccounts.GetByID(ctx, input.SavingsAccountID)
	if err != nil {
		return ListSettlementsOutput{}, err
	}

	if err := s.authorizeAccount(ctx, s.deps.Repositories, input.Actor, account); err != nil {
		return ListSettlementsOutput{}, err
	}

	settlements, err := s.deps.Repositories.Settlements.ListBySavingsAccount(
		ctx,
		account.ID,
		repository.ListOptions{
			Limit:  input.Options.Limit,
			Offset: input.Options.Offset,
		},
	)
	if err != nil {
		return ListSettlementsOutput{}, err
	}

	return ListSettlementsOutput{Settlements: settlementPointers(settlements)}, nil
}

func settlementPointers(settlements []domain.SavingsSettlement) []*domain.SavingsSettlement {
	result := make([]*domain.SavingsSettlement, len(settlements))
	for i := range settlements {
		settlement := settlements[i]
		result[i] = &settlement
	}
	return result
}
