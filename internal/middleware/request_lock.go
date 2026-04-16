package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

const UserEmailKey contextKey = "userEmail"

type requestGate struct {
	slot chan struct{}
	refs int
}

var requestGates = struct {
	mu    sync.Mutex
	byKey map[string]*requestGate
}{
	byKey: make(map[string]*requestGate),
}

func SingleRequestPerUserMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, ok := requestKeyFromContext(r.Context())
		if !ok {
			http.Error(w, "Nao foi possivel identificar o usuario da requisicao", http.StatusInternalServerError)
			return
		}

		release, err := acquireRequestSlot(r.Context(), key)
		if err != nil {
			return
		}
		defer release()

		next.ServeHTTP(w, r)
	})
}

func AcquireEmailRequestSlot(ctx context.Context, email string) (func(), error) {
	normalized := normalizeEmail(email)
	if normalized == "" {
		return func() {}, nil
	}

	return acquireRequestSlot(ctx, emailRequestKey(normalized))
}

func requestKeyFromContext(ctx context.Context) (string, bool) {
	if email, ok := ctx.Value(UserEmailKey).(string); ok {
		normalized := normalizeEmail(email)
		if normalized != "" {
			return emailRequestKey(normalized), true
		}
	}

	if userID, ok := ctx.Value(UserIDKey).(int); ok && userID > 0 {
		return fmt.Sprintf("user:%d", userID), true
	}

	return "", false
}

func acquireRequestSlot(ctx context.Context, key string) (func(), error) {
	gate := retainRequestGate(key)

	select {
	case gate.slot <- struct{}{}:
		return func() {
			<-gate.slot
			releaseRequestGate(key)
		}, nil
	case <-ctx.Done():
		releaseRequestGate(key)
		return nil, ctx.Err()
	}
}

func retainRequestGate(key string) *requestGate {
	requestGates.mu.Lock()
	defer requestGates.mu.Unlock()

	gate, exists := requestGates.byKey[key]
	if !exists {
		gate = &requestGate{
			slot: make(chan struct{}, 1),
		}
		requestGates.byKey[key] = gate
	}

	gate.refs++
	return gate
}

func releaseRequestGate(key string) {
	requestGates.mu.Lock()
	defer requestGates.mu.Unlock()

	gate, exists := requestGates.byKey[key]
	if !exists {
		return
	}

	gate.refs--
	if gate.refs == 0 {
		delete(requestGates.byKey, key)
	}
}

func emailRequestKey(email string) string {
	return "email:" + email
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
