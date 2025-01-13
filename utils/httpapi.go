package utils

import (
	"context"
	"fmt"
	"os"
	"runtime"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/xrpc"
	"golang.org/x/exp/slog"
)

func LogError(ctx context.Context, errIn error) (err error) {
	if errIn != nil {
		pc, _, _, ok := runtime.Caller(1)
		details := runtime.FuncForPC(pc)
		var callingFunc string
		if ok && details != nil {
			callingFunc = fmt.Sprintf("called from %s", details.Name())
		}
		slog.ErrorContext(ctx, "error ", callingFunc, slog.Any("error", errIn))
		return fmt.Errorf("error %v msg: %v", callingFunc, errIn)
	}
	return nil
}

func getHandle() string {
	return os.Getenv("ATPROTO_BOT_HANDLE")
}

func getPassword() string {
	return os.Getenv("ATPROTO_BOT_PASSWORD")
}

func LoadAuthInfo(ctx context.Context, xrpcc *xrpc.Client) (*xrpc.AuthInfo, error) {
	auth, err := getAuthSession(ctx, xrpcc)

	if auth == nil {
		auth, err = refreshAuthSession(ctx, xrpcc)
	}

	if auth == nil {
		auth, err = createAuthSession(ctx, xrpcc)
	}

	return auth, err
}

func getAuthSession(ctx context.Context, xrpcc *xrpc.Client) (*xrpc.AuthInfo, error) {
	slog.DebugContext(ctx, "getting session")
	auth, err := comatproto.ServerGetSession(ctx, xrpcc)

	if err != nil {
		return nil, LogError(ctx, fmt.Errorf("failed to get session: %w", err))
	}

	return &xrpc.AuthInfo{
		Did:    auth.Did,
		Handle: auth.Handle,
	}, nil
}

func refreshAuthSession(ctx context.Context, xrpcc *xrpc.Client) (*xrpc.AuthInfo, error) {
	slog.DebugContext(ctx, "refreshing session")
	auth, err := comatproto.ServerRefreshSession(ctx, xrpcc)

	if err != nil {
		return nil, LogError(ctx, fmt.Errorf("failed to refresh session: %w", err))
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
		return nil, LogError(ctx, fmt.Errorf("failed to create session: %w", err))
	}

	return &xrpc.AuthInfo{
		AccessJwt:  auth.AccessJwt,
		Did:        auth.Did,
		Handle:     auth.Handle,
		RefreshJwt: auth.RefreshJwt,
	}, nil
}

// func CheckTokenExpired(ctx context.Context, xrpcc *xrpc.Client) error {
// 	slog.DebugContext(ctx, "check xrpc auth token status")

// 	if xrpcc.Auth == nil {
// 		slog.InfoContext(ctx, "create new session by password")
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
// 			slog.DebugContext(ctx, "accessJwt expired")
// 			xrpcc.Auth.AccessJwt = ""
// 		}
// 	}
// 	if xrpcc.Auth.RefreshJwt != "" {
// 		token, err := jwt.ParseString(xrpcc.Auth.RefreshJwt, jwt.WithVerify(false))
// 		if err != nil && !errors.Is(err, jwt.ErrTokenExpired()) {
// 			return fmt.Errorf("failed to parse jwt: %w", err)
// 		}

// 		if errors.Is(err, jwt.ErrTokenExpired()) || token.Expiration().Before(now) {
// 			slog.DebugContext(ctx, "refreshJwt expired")
// 			xrpcc.Auth.RefreshJwt = ""
// 		}
// 	}

// 	if xrpcc.Auth.AccessJwt == "" && xrpcc.Auth.RefreshJwt == "" {
// 		slog.InfoContext(ctx, "create new session from scratch")
// 		xrpcc.Auth = nil
// 		auth, err := LoadAuthInfo(ctx, xrpcc)
// 		if err != nil {
// 			return err
// 		}
// 		xrpcc.Auth = auth

// 	} else if xrpcc.Auth.AccessJwt == "" {
// 		slog.InfoContext(ctx, "refresh session by refreshJwt")
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
