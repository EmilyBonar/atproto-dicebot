package main

import (
	dicebot "atproto-dicebot/src"
	"atproto-dicebot/utils"

	"context"
	"os"

	cliutil "github.com/bluesky-social/indigo/cmd/gosky/util"

	"github.com/bluesky-social/indigo/xrpc"
	"github.com/k0kubun/pp/v3"
	"golang.org/x/exp/slog"
)

func main() {
	ctx := context.Background()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))

	xrpcc := &xrpc.Client{
		Client: cliutil.NewHttpClient(),
		Host:   "https://bsky.social",
	}

	auth, err := utils.LoadAuthInfo(ctx, xrpcc)
	if err != nil {
		slog.Error("error on cliutils.LoadAuthInfo", "error", err)
		panic(err)
	}
	xrpcc.Auth = auth

	err = utils.CheckTokenExpired(ctx, xrpcc)
	if err != nil {
		slog.Error("error on cliutils.CheckTokenExpired", "error", err)
		panic(err)
	}

	defer func() {
		err := utils.DeleteSession(ctx, xrpcc)
		if err != nil {
			slog.Error("error raised by com.atproto.server.deleteSession", "error", err)
		}
	}()
	respList, err := dicebot.ProcessNotifications(ctx, xrpcc)
	if err != nil {
		slog.Error("error on dicebot.ProcessNotifications", "error", err)
		panic(err)
	}

	pp.Println(respList)
}
