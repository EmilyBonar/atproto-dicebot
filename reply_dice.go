package dicebot

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	lexutil "github.com/bluesky-social/indigo/lex/util"
	"github.com/bluesky-social/indigo/xrpc"
	"go.opentelemetry.io/otel"
	"golang.org/x/exp/slog"
)

type ResponseReplyDID struct {
	Base   *bsky.NotificationListNotifications_Notification
	Input  *comatproto.RepoCreateRecord_Input
	Output *comatproto.RepoCreateRecord_Output
}

func (reply *ResponseReplyDID) isResponse() {}

type ResponseReplyDice struct {
	Base   *bsky.NotificationListNotifications_Notification
	Input  *comatproto.RepoCreateRecord_Input
	Output *comatproto.RepoCreateRecord_Output
}

func (reply *ResponseReplyDice) isResponse() {}

func parseDice(_ context.Context, me *xrpc.AuthInfo, feedPost *bsky.FeedPost) []Dice {
	s := feedPost.Text
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "@"+me.Handle)
	s = strings.TrimSpace(s)

	var diceMatcher = regexp.MustCompile(`(\d*)d(\d+)`)
	diceStrs := diceMatcher.FindAllString(s, -1)

	var dicePool []Dice
	for _, diceStr := range diceStrs {
		captureGroups := diceMatcher.FindStringSubmatch(diceStr)
		if len(captureGroups) == 2 {
			sides, _ := strconv.Atoi(captureGroups[1])

			dicePool = append(dicePool, Dice{number: 1, sides: sides})
		} else if len(captureGroups) == 3 {
			number, _ := strconv.Atoi(captureGroups[1])
			sides, _ := strconv.Atoi(captureGroups[2])

			dicePool = append(dicePool, Dice{number: number, sides: sides})

		}
	}

	return dicePool
}

func replyDice(ctx context.Context, xrpcc *xrpc.Client, nf *bsky.NotificationListNotifications_Notification, dicePool []Dice) (_ Response, err error) {
	ctx, span := otel.Tracer("dicebot").Start(ctx, "replyDice")
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
		span.End()
	}()

	responseText := ""

	for _, dice := range dicePool {
		diceRoll := rollDice(dice)
		rollString := strings.Trim(fmt.Sprint(diceRoll), "[]")
		if len(diceRoll) == 1 {
			responseText += fmt.Sprintf("%dd%d: %v\n", dice.number, dice.sides, rollString)
		} else {
			responseText += fmt.Sprintf("%dd%d: %v = %d\n", dice.number, dice.sides, rollString, sum(diceRoll))
		}
	}

	input := &comatproto.RepoCreateRecord_Input{
		Collection: "app.bsky.feed.post",
		Repo:       xrpcc.Auth.Did,
		Record: &lexutil.LexiconTypeDecoder{
			Val: &bsky.FeedPost{
				Text:      responseText,
				CreatedAt: time.Now().Local().Format(time.RFC3339),
				Reply: &bsky.FeedPost_ReplyRef{
					Parent: &comatproto.RepoStrongRef{
						Cid: nf.Cid,
						Uri: nf.Uri,
					},
					Root: &comatproto.RepoStrongRef{
						Cid: nf.Cid,
						Uri: nf.Uri,
					},
				},
			},
		},
	}

	output, err := comatproto.RepoCreateRecord(ctx, xrpcc, input)
	if err != nil {
		slog.Error("error raised by com.atproto.repo.createRecord", "error", err)
		return nil, err
	}

	slog.InfoCtx(ctx, "message posted", "uri", output.Uri, "cid", output.Cid)

	resp := &ResponseReplyDID{
		Base:   nf,
		Input:  input,
		Output: output,
	}

	return resp, nil
}
