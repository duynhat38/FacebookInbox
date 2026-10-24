package facebook

import (
	"context"
	"crypto/rand"
	"math/big"
	"net/http"
	"sync"
)

// MessengerSender sends Messenger messages via browser cookie + MQTT/WebSocket.
// Token extraction results are cached per cookie. A send is driven through a
// per-page messengerSession (see session.go) that reuses one WebSocket pair for
// all steps of a page; MessengerSender itself only holds the cross-page token
// cache and hands sessions their tokens via getTokens.
type MessengerSender struct {
	cache sync.Map // map[cookie string] messengerTokens
}

func NewMessengerSender() *MessengerSender { return &MessengerSender{} }

func (s *MessengerSender) getTokens(ctx context.Context, client *http.Client, cookie string) (messengerTokens, error) {
	if v, ok := s.cache.Load(cookie); ok {
		return v.(messengerTokens), nil
	}
	tok, err := extractTokens(ctx, client, cookie)
	if err != nil {
		return messengerTokens{}, err
	}
	s.cache.Store(cookie, tok)
	return tok, nil
}

// ── Random helpers ────────────────────────────────────────────────────────────

func msgrRandInt64(min, max int64) int64 {
	n, err := rand.Int(rand.Reader, big.NewInt(max-min+1))
	if err != nil {
		return min
	}
	return min + n.Int64()
}

func msgrRandInt(min, max int) int {
	return int(msgrRandInt64(int64(min), int64(max)))
}
