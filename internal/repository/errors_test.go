package repository_test

import (
	"errors"
	"testing"

	"github.com/fasih/tabungan-siswa/internal/repository"
)

func TestRepositoryErrors(t *testing.T) {
	if !errors.Is(repository.ErrNotFound, repository.ErrNotFound) {
		t.Fatal("ErrNotFound must be identifiable with errors.Is")
	}

	if !errors.Is(repository.ErrConflict, repository.ErrConflict) {
		t.Fatal("ErrConflict must be identifiable with errors.Is")
	}

	if errors.Is(repository.ErrNotFound, repository.ErrConflict) {
		t.Fatal("ErrNotFound and ErrConflict must remain distinct")
	}
}
