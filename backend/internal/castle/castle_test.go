package castle

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCode(t *testing.T) {
	for _, c := range []string{"a", "the-grey-castle", "castle-2026", "abc"} {
		assert.NoError(t, ValidateCode(c), c)
	}
	for _, c := range []string{"", "-foo", "foo-", "Foo", "a..b", "x/y", "a b"} {
		assert.ErrorIs(t, ValidateCode(c), ErrInvalidCode, c)
	}
}

func TestRegistry_GetOrCreate(t *testing.T) {
	r := NewRegistry()
	c1, err := r.GetOrCreate("grey")
	require.NoError(t, err)
	c2, err := r.GetOrCreate("grey")
	require.NoError(t, err)
	assert.Same(t, c1, c2, "GetOrCreate must be stable for the same code")
}

func TestCastle_SetQuestion_PushesHistory(t *testing.T) {
	c, err := NewRegistry().GetOrCreate("grey")
	require.NoError(t, err)
	q1 := Question{BucketID: "2026-05-11T10:00:00Z", Text: "first", EmittedAt: time.Now()}
	q2 := Question{BucketID: "2026-05-11T11:00:00Z", Text: "second", EmittedAt: time.Now()}
	c.SetQuestion(q1)
	c.SetQuestion(q2)
	cur, hist := c.Snapshot()
	assert.Equal(t, "second", cur.Text)
	require.Len(t, hist, 1)
	assert.Equal(t, "first", hist[0].Text)
}

func TestCastle_SubscribeUnsubscribe(t *testing.T) {
	c, _ := NewRegistry().GetOrCreate("grey")
	c.Subscribe(PushSubscription{Endpoint: "e1"})
	c.Subscribe(PushSubscription{Endpoint: "e1"}) // idempotent
	c.Subscribe(PushSubscription{Endpoint: "e2"})
	assert.Len(t, c.Subs(), 2)
	c.Unsubscribe("e1")
	assert.Len(t, c.Subs(), 1)
}

func TestRegistry_GetReturnsNotFound(t *testing.T) {
	r := NewRegistry()
	_, err := r.Get("unknown")
	assert.ErrorIs(t, err, ErrCastleNotFound)
}
