package service

import (
	"context"
	"testing"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/google/uuid"
)

func TestActorContextRoundTrip(t *testing.T) {
	actor := Actor{
		UserID: uuid.New(),
		Role:   domain.RoleAdmin,
	}

	ctx := WithActor(context.Background(), actor)

	got, ok := ActorFromContext(ctx)
	if !ok {
		t.Fatal("ActorFromContext() ok = false, want true")
	}

	if got != actor {
		t.Fatalf("ActorFromContext() = %+v, want %+v", got, actor)
	}
}

func TestActorFromContextMissing(t *testing.T) {
	_, ok := ActorFromContext(context.Background())
	if ok {
		t.Fatal("ActorFromContext() ok = true, want false")
	}
}

func TestActorFromContextRejectsInvalidActor(t *testing.T) {
	ctx := WithActor(context.Background(), Actor{})

	_, ok := ActorFromContext(ctx)
	if ok {
		t.Fatal("ActorFromContext() ok = true, want false")
	}
}
