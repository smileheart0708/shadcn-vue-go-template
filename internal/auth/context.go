package auth

import "context"

type contextKey struct{}

var actorContextKey contextKey

func WithActor(ctx context.Context, actor Actor) context.Context {
	return context.WithValue(ctx, actorContextKey, actor)
}

func ActorFromContext(ctx context.Context) (Actor, bool) {
	actor, ok := ctx.Value(actorContextKey).(Actor)
	return actor, ok
}
