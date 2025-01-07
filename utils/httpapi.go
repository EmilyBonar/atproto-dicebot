package utils

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	dicebot "github.com/EmilyBonar/atproto-dicebot"
	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"golang.org/x/exp/slog"
)

func getHandle() string {
	return os.Getenv("ATPROTO_BOT_HANDLE")
}

func getPassword() string {
	return os.Getenv("ATPROTO_BOT_PASSWORD")
}

func LoadAuthInfo(ctx context.Context, xrpcc *xrpc.Client) (*xrpc.AuthInfo, error) {
	handle := getHandle()
	password := getPassword()

	slog.DebugCtx(ctx, "create session", "handle", handle)

	auth, err := comatproto.ServerCreateSession(ctx, xrpcc, &comatproto.ServerCreateSession_Input{
		Identifier: handle,
		Password:   string(password),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &xrpc.AuthInfo{
		AccessJwt:  auth.AccessJwt,
		RefreshJwt: auth.RefreshJwt,
		Handle:     auth.Handle,
		Did:        auth.Did,
	}, nil
}

func CheckTokenExpired(ctx context.Context, xrpcc *xrpc.Client) error {
	slog.DebugCtx(ctx, "check xrpc auth token status")

	if xrpcc.Auth == nil {
		slog.InfoCtx(ctx, "create new session by password")
		auth, err := LoadAuthInfo(ctx, xrpcc)
		if err != nil {
			return err
		}
		xrpcc.Auth = auth
		return nil
	}

	now := time.Now().Add(+1 * time.Minute)
	if xrpcc.Auth.AccessJwt != "" {
		token, err := jwt.ParseString(xrpcc.Auth.AccessJwt, jwt.WithVerify(false))
		if err != nil && !errors.Is(err, jwt.ErrTokenExpired()) {
			return fmt.Errorf("faield to parse jwt: %w", err)
		}

		if errors.Is(err, jwt.ErrTokenExpired()) || token.Expiration().Before(now) {
			slog.DebugCtx(ctx, "accessJwt expired")
			xrpcc.Auth.AccessJwt = ""
		}
	}
	if xrpcc.Auth.RefreshJwt != "" {
		token, err := jwt.ParseString(xrpcc.Auth.RefreshJwt, jwt.WithVerify(false))
		if err != nil && !errors.Is(err, jwt.ErrTokenExpired()) {
			return fmt.Errorf("faield to parse jwt: %w", err)
		}

		if errors.Is(err, jwt.ErrTokenExpired()) || token.Expiration().Before(now) {
			slog.DebugCtx(ctx, "refreshJwt expired")
			xrpcc.Auth.RefreshJwt = ""
		}
	}

	if xrpcc.Auth.AccessJwt == "" && xrpcc.Auth.RefreshJwt == "" {
		slog.InfoCtx(ctx, "create new session from scratch")
		xrpcc.Auth = nil
		auth, err := LoadAuthInfo(ctx, xrpcc)
		if err != nil {
			return err
		}
		xrpcc.Auth = auth

	} else if xrpcc.Auth.AccessJwt == "" {
		slog.InfoCtx(ctx, "refresh session by refreshJwt")
		xrpcc.Auth.AccessJwt = xrpcc.Auth.RefreshJwt
		xrpcc.Auth.RefreshJwt = ""

		resp, err := comatproto.ServerRefreshSession(ctx, xrpcc)
		if err != nil {
			return fmt.Errorf("failed to refresh session: %w", err)
		}

		xrpcc.Auth = &xrpc.AuthInfo{
			AccessJwt:  resp.AccessJwt,
			RefreshJwt: resp.RefreshJwt,
			Handle:     resp.Handle,
			Did:        resp.Did,
		}
	}

	return nil
}

func New(xrpcc *xrpc.Client) (*Handler, error) {
	if xrpcc.Auth == nil {
		return nil, errors.New("xrpc client doesn't have auth info")
	}

	return &Handler{xrpcc: xrpcc}, nil
}

type Handler struct {
	xrpcc *xrpc.Client
}

func (h *Handler) Serve(mux *http.ServeMux) {
	mux.HandleFunc("/api/processNotifications", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		err := CheckTokenExpired(ctx, h.xrpcc)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			slog.ErrorCtx(ctx, "error on cliutils.CheckTokenExpired", "error", err)
			return
		}

		err = h.ProcessNotifications(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			slog.ErrorCtx(ctx, "failed to process notifications", "error", err)
			return
		}
	})
}

func (h *Handler) ProcessNotifications(ctx context.Context) error {
	respList, err := dicebot.ProcessNotifications(ctx, h.xrpcc)
	if err != nil {
		return err
	}

	slog.InfoCtx(ctx, "processed message", "count", len(respList))

	return nil
}

func DeleteSession(ctx context.Context, xrpcc *xrpc.Client) error {
	slog.DebugCtx(ctx, "delete xrpc server session")

	err := CheckTokenExpired(ctx, xrpcc)

	if err != nil {
		slog.Error("error on cliutils.CheckTokenExpired", "error", err)
		panic(err)
	}

	err = comatproto.ServerDeleteSession(ctx, xrpcc)
	return err
}
