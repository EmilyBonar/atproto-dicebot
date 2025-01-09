package utils

import (
	"context"
	"fmt"
	"os"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/xrpc"
	"golang.org/x/exp/slog"
)

func getHandle() string {
	return os.Getenv("ATPROTO_BOT_HANDLE")
}

func getPassword() string {
	return os.Getenv("ATPROTO_BOT_PASSWORD")
}

func LoadAuthInfo(ctx context.Context, xrpcc *xrpc.Client) (*xrpc.AuthInfo, error) {
	auth, err := refreshAuthSession(ctx, xrpcc)

	if err != nil {
		auth, err = createAuthSession(ctx, xrpcc)
	}

	return auth, err
}

func refreshAuthSession(ctx context.Context, xrpcc *xrpc.Client) (*xrpc.AuthInfo, error) {
	slog.DebugCtx(ctx, "refreshing session")
	auth, err := comatproto.ServerRefreshSession(ctx, xrpcc)

	if err != nil {
		return nil, fmt.Errorf("failed to refresh session: %w", err)
	}

	return &xrpc.AuthInfo{
		AccessJwt:  auth.AccessJwt,
		Did:        auth.Did,
		Handle:     auth.Handle,
		RefreshJwt: auth.RefreshJwt,
	}, nil
}

func createAuthSession(ctx context.Context, xrpcc *xrpc.Client) (*xrpc.AuthInfo, error) {
	handle := getHandle()
	password := getPassword()

	slog.DebugContext(ctx, "creating session", "handle", handle)
	auth, err := comatproto.ServerCreateSession(ctx, xrpcc, &comatproto.ServerCreateSession_Input{
		Identifier: handle,
		Password:   string(password),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &xrpc.AuthInfo{
		AccessJwt:  auth.AccessJwt,
		Did:        auth.Did,
		Handle:     auth.Handle,
		RefreshJwt: auth.RefreshJwt,
	}, nil
}

// func CheckTokenExpired(ctx context.Context, xrpcc *xrpc.Client) error {
// 	slog.DebugCtx(ctx, "check xrpc auth token status")

// 	if xrpcc.Auth == nil {
// 		slog.InfoCtx(ctx, "create new session by password")
// 		auth, err := LoadAuthInfo(ctx, xrpcc)
// 		if err != nil {
// 			return err
// 		}
// 		xrpcc.Auth = auth
// 		return nil
// 	}

// 	now := time.Now().Add(+1 * time.Minute)
// 	if xrpcc.Auth.AccessJwt != "" {
// 		token, err := jwt.ParseString(xrpcc.Auth.AccessJwt, jwt.WithVerify(false))
// 		if err != nil && !errors.Is(err, jwt.ErrTokenExpired()) {
// 			return fmt.Errorf("failed to parse jwt: %w", err)
// 		}

// 		if errors.Is(err, jwt.ErrTokenExpired()) || token.Expiration().Before(now) {
// 			slog.DebugCtx(ctx, "accessJwt expired")
// 			xrpcc.Auth.AccessJwt = ""
// 		}
// 	}
// 	if xrpcc.Auth.RefreshJwt != "" {
// 		token, err := jwt.ParseString(xrpcc.Auth.RefreshJwt, jwt.WithVerify(false))
// 		if err != nil && !errors.Is(err, jwt.ErrTokenExpired()) {
// 			return fmt.Errorf("failed to parse jwt: %w", err)
// 		}

// 		if errors.Is(err, jwt.ErrTokenExpired()) || token.Expiration().Before(now) {
// 			slog.DebugCtx(ctx, "refreshJwt expired")
// 			xrpcc.Auth.RefreshJwt = ""
// 		}
// 	}

// 	if xrpcc.Auth.AccessJwt == "" && xrpcc.Auth.RefreshJwt == "" {
// 		slog.InfoCtx(ctx, "create new session from scratch")
// 		xrpcc.Auth = nil
// 		auth, err := LoadAuthInfo(ctx, xrpcc)
// 		if err != nil {
// 			return err
// 		}
// 		xrpcc.Auth = auth

// 	} else if xrpcc.Auth.AccessJwt == "" {
// 		slog.InfoCtx(ctx, "refresh session by refreshJwt")
// 		xrpcc.Auth.AccessJwt = xrpcc.Auth.RefreshJwt
// 		xrpcc.Auth.RefreshJwt = ""

// 		resp, err := comatproto.ServerRefreshSession(ctx, xrpcc)
// 		if err != nil {
// 			return fmt.Errorf("failed to refresh session: %w", err)
// 		}

// 		xrpcc.Auth = &xrpc.AuthInfo{
// 			AccessJwt:  resp.AccessJwt,
// 			RefreshJwt: resp.RefreshJwt,
// 			Handle:     resp.Handle,
// 			Did:        resp.Did,
// 		}
// 	}

// 	return nil
// }
