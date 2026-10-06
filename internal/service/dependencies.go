package service

import (
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/security"
)

type Dependencies struct {
	Repositories   repository.RepositorySet
	UOW            repository.UnitOfWorkManager
	PasswordHasher security.PasswordHasher
}

func (d Dependencies) Validate() error {
	if d.UOW == nil {
		return ErrInvalidDependency
	}

	return nil
}
