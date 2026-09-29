package service

import (
	"github.com/fasih/tabungan-siswa/internal/repository"
)

type Dependencies struct {
	Repositories repository.RepositorySet
	UOW          repository.UnitOfWorkManager
}

func (d Dependencies) Validate() error {
	if d.UOW == nil {
		return ErrInvalidDependency
	}

	return nil
}
