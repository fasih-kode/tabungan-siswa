package service_test

import (
	"errors"
	"testing"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/service"
)

func TestServiceErrorsAreIdentifiable(t *testing.T) {
	errs := []struct {
		name string
		err  error
	}{
		{
			name: "invalid actor",
			err:  service.ErrInvalidActor,
		},
		{
			name: "forbidden",
			err:  service.ErrForbidden,
		},
		{
			name: "scope violation",
			err:  service.ErrScopeViolation,
		},
	}

	for _, tc := range errs {
		t.Run(tc.name, func(t *testing.T) {
			if !errors.Is(tc.err, tc.err) {
				t.Fatalf("errors.Is(%v, %v) = false", tc.err, tc.err)
			}
		})
	}
}

func TestServiceErrorsRemainDistinct(t *testing.T) {
	errs := []error{
		service.ErrInvalidActor,
		service.ErrForbidden,
		service.ErrScopeViolation,
	}

	for i := range errs {
		for j := range errs {
			if i == j {
				continue
			}

			if errors.Is(errs[i], errs[j]) {
				t.Fatalf(
					"errors.Is(%v, %v) = true; service errors must remain distinct",
					errs[i],
					errs[j],
				)
			}
		}
	}
}

func TestServiceErrorsRemainDistinctFromDomainAndRepositoryErrors(t *testing.T) {
	serviceErrors := []error{
		service.ErrInvalidActor,
		service.ErrForbidden,
		service.ErrScopeViolation,
	}

	otherErrors := []error{
		domain.ErrInvalidID,
		domain.ErrInvalidTransition,
		repository.ErrNotFound,
		repository.ErrConflict,
	}

	for _, serviceErr := range serviceErrors {
		for _, otherErr := range otherErrors {
			if errors.Is(serviceErr, otherErr) {
				t.Fatalf(
					"errors.Is(%v, %v) = true; service and lower-layer errors must remain distinct",
					serviceErr,
					otherErr,
				)
			}
		}
	}
}
