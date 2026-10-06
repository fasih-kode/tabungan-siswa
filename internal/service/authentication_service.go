package service

import (
	"context"
	"errors"

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

	if deps.Repositories.Users == nil || deps.PasswordHasher == nil {
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
