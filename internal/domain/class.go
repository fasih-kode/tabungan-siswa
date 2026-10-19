package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type Class struct {
	ID             uuid.UUID
	Name           string
	Level          int
	AcademicYearID uuid.UUID
	EffectiveDate  time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NewClass(name string, level int) (Class, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Class{}, ErrEmptyName
	}

	if level < 1 || level > 12 {
		return Class{}, ErrInvalidClassLevel
	}

	now := time.Now()

	return Class{
		ID:        uuid.New(),
		Name:      name,
		Level:     level,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}
