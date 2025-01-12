package dicebot

import (
	"atproto-dicebot/utils"
	"context"
	"fmt"
	"strings"
	"time"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	lexutil "github.com/bluesky-social/indigo/lex/util"
	"github.com/bluesky-social/indigo/xrpc"
	"go.opentelemetry.io/otel"
	"golang.org/x/exp/slog"
)

type ResponseReplyDice struct {
	Base   *bsky.NotificationListNotifications_Notification
	Input  *comatproto.RepoCreateRecord_Input
	Output *comatproto.RepoCreateRecord_Output
}

func (reply *ResponseReplyDice) isResponse() {}

func replyDice(ctx context.Context, xrpcc *xrpc.Client, nf *bsky.NotificationListNotifications_Notification, dicePool []utils.Dice) (_ Response, err error) {
	ctx, span := otel.Tracer("dicebot").Start(ctx, "replyDice")
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
		span.End()
	}()

	responseText := ""

	for _, dice := range dicePool {
		diceRoll := utils.RollDice(dice)
		rollString := strings.Trim(fmt.Sprint(diceRoll), "[]")
		if len(diceRoll) == 1 {
			responseText += fmt.Sprintf("%dd%d: %v\n", dice.Number, dice.Sides, rollString)
		} else {
			responseText += fmt.Sprintf("%dd%d: %v = %d\n", dice.Number, dice.Sides, rollString, utils.Sum(diceRoll))
		}
	}

	reply := utils.GetReplyRefs(ctx, xrpcc, comatproto.RepoStrongRef{
		Cid: nf.Cid,
		Uri: nf.Uri,
	})

	input := &comatproto.RepoCreateRecord_Input{
		Collection: "app.bsky.feed.post",
		Repo:       xrpcc.Auth.Did,
		Record: &lexutil.LexiconTypeDecoder{
			Val: &bsky.FeedPost{
				Text:      responseText,
				CreatedAt: time.Now().Local().Format(time.RFC3339),
				Reply:     &reply,
			},
		},
	}

	output, err := comatproto.RepoCreateRecord(ctx, xrpcc, input)
	if err != nil {
		slog.Error("error raised by com.atproto.repo.createRecord", "error", err)
		return nil, err
	}

	slog.InfoContext(ctx, "message posted", "uri", output.Uri, "cid", output.Cid)

	resp := &ResponseReplyDice{
		Base:   nf,
		Input:  input,
		Output: output,
	}

	return resp, nil
}
