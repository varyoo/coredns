package dnstap

import (
	"context"
	"testing"
)

func TestDnstapContext(t *testing.T) {
	ctx := tapContext{context.TODO(), Dnstap{}}
	tapper := TapperFromContext(ctx)

	if tapper == nil {
		t.Fatal("can't get tapper")
	}
}
