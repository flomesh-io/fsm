package context

import (
	"context"
	"errors"
)

func ToControllerContext(ctx context.Context) (*ControllerContext, error) {
	if ctx == nil {
		return nil, errors.New("missing Context")
	}

	parentCtx := ctx.Value(&ControllerCtxKey)
	if parentCtx == nil {
		return nil, errors.New("missing Controller Context")
	}

	if cc, ok := parentCtx.(*ControllerContext); ok {
		return cc, nil
	}

	return nil, errors.New("invalid Controller Context")
}
