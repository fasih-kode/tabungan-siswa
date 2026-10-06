package service

import (
	"context"
	"errors"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
)

type authenticationService struct {
	deps Dependencies
}

var _ AuthenticationService = (*authenticationService)(nil)

func NewAuthenticationService(deps Dependencies) (*authenticationService, error) {
	if err := deps.Validate(); err != nil {
		return nil, err
	}

	if deps.Repositories.Users == nil ||
		deps.Repositories.Sessions == nil ||
		deps.PasswordHasher == nil {
		return nil, ErrInvalidDependency
	}

	return &authenticationService{
		deps: deps,
	}, nil
}

func (s *authenticationService) Authenticate(
	ctx context.Context,
	input AuthenticateInput,
) (AuthenticateOutput, error) {
	if input.Username == "" || input.Password == "" {
		return AuthenticateOutput{}, ErrInvalidCredentials
	}

	user, err := s.deps.Repositories.Users.GetByUsername(
		ctx,
		input.Username,
	)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return AuthenticateOutput{}, ErrInvalidCredentials
		}

		return AuthenticateOutput{}, err
	}

	if err := s.deps.PasswordHasher.Compare(
		user.PasswordHash,
		input.Password,
	); err != nil {
		return AuthenticateOutput{}, ErrInvalidCredentials
	}

	actor := Actor{
		UserID: user.ID,
		Role:   user.Role,
	}
	if err := actor.Validate(); err != nil {
		return AuthenticateOutput{}, ErrInvalidCredentials
	}

	return AuthenticateOutput{
		Actor: actor,
	}, nil
}

func (s *authenticationService) Logout(
	ctx context.Context,
	input LogoutInput,
) error {
	if input.SessionTokenHash == "" {
		return domain.ErrInvalidSessionTokenHash
	}

	session, err := s.deps.Repositories.Sessions.GetByTokenHash(
		ctx,
		input.SessionTokenHash,
	)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil
		}
		return err
	}

	now := time.Now()

	if !session.IsActive(now) {
		return nil
	}

	if err := s.deps.Repositories.Sessions.Revoke(
		ctx,
		session.ID,
		now,
	); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil
		}
		return err
	}

	return nil
}
