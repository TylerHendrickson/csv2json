package csvctx

import (
	"context"
	"encoding/csv"
	"errors"
)

type Reader struct {
	ctx context.Context
	r   *csv.Reader
}

func New(ctx context.Context, r *csv.Reader) *Reader {
	return &Reader{ctx, r}
}

func (r *Reader) Read() ([]string, error) {
	type result struct {
		rec []string
		err error
	}
	ch := make(chan result, 1)

	go func() {
		rec, err := r.r.Read()
		ch <- result{rec: rec, err: err}
	}()

	select {
	case <-r.ctx.Done():
		return nil, r.ctx.Err()
	case res := <-ch:
		return res.rec, res.err
	}
}

func (r *Reader) FieldPos(field int) (line int, column int) {
	return r.r.FieldPos(field)
}

// IsContextError is a helper for checking whether err is the result of context
// cancellation or timeout.
func IsContextError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
