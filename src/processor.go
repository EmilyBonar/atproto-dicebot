package dicebot

import (
	"atproto-dicebot/utils"
	"context"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/xrpc"
	"go.opentelemetry.io/otel"
	"golang.org/x/exp/slog"
)

type Response interface {
	isResponse()
}

func ProcessNotifications(ctx context.Context, xrpcc *xrpc.Client) (_ []Response, err error) {
	ctx, span := otel.Tracer("dicebot").Start(ctx, "ProcessNotifications")
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
		span.End()
	}()

	now := time.Now()

	unreadResp, err := bsky.NotificationGetUnreadCount(ctx, xrpcc, false, "")
	if err != nil {
		slog.ErrorContext(ctx, "error raised by app.bsky.notification.getUnreadCount", "error", err)
		return nil, err
	}

	slog.DebugContext(ctx, "check unread count", "count", unreadResp.Count)

	respList := make([]Response, 0)
	limit := int64(20)
	var cursor string
OUTER:
	for {
		resp, err := bsky.NotificationListNotifications(ctx, xrpcc, cursor, limit, false, "")
		if err != nil {
			slog.ErrorContext(ctx, "error raised by app.bsky.notification.listNotifications", "error", err)
			return nil, err
		}

		slog.DebugContext(ctx, "response about app.bsky.notification.listNotifications", "cursor", resp.Cursor, "length", len(resp.Notifications))

		for idx, nf := range resp.Notifications {
			slog.DebugContext(
				ctx,
				"notification",
				"index", idx,
				"reason", nf.Reason,
				"author", nf.Author.Handle,
				"cid", nf.Cid,
				"isRead", nf.IsRead,
			)

			switch v := nf.Record.Val.(type) {
			case *bsky.FeedPost:
				slog.DebugContext(ctx, "feed post", "author", nf.Author.Did, "text", v.Text)

				// Commenting out so that replies that don't explicitly mention still get answers
				// if !utils.DoesMentionMe(ctx, xrpcc.Auth, v) {
				// 	slog.DebugContext(ctx, "this post doesn't mention me")
				// 	continue
				// }

				threadResp, err := bsky.FeedGetPostThread(ctx, xrpcc, 10, 10, nf.Uri)
				if err != nil {
					slog.Error("error raised by app.bsky.feed.getPostThread", "error", err)
					return nil, err
				}

				if utils.HasAlreadyReplied(ctx, xrpcc.Auth, threadResp) {
					slog.DebugContext(ctx, "found newest replied post", "cid", nf.Cid)
					break OUTER
				}

				if dicePool := utils.ParseDice(ctx, xrpcc.Auth, v); len(dicePool) > 0 {
					resp, err := replyDice(ctx, xrpcc, nf, dicePool)
					if err != nil {
						return nil, err
					}

					respList = append(respList, resp)
				} else {
					slog.DebugContext(ctx, "no dice requests found", "text", v.Text)
				}

			case *bsky.FeedRepost:
				slog.DebugContext(ctx, "feed repost", "subjectCid", v.Subject.Cid, "subjectUri", v.Subject.Uri)
			case *bsky.FeedLike:
				slog.DebugContext(ctx, "feed like", "subjectCid", v.Subject.Cid, "subjectUri", v.Subject.Uri)
			case *bsky.GraphFollow:
				slog.DebugContext(ctx, "graph follow", "subject", v.Subject)
			default:
				slog.WarnContext(ctx, "unknown record type", "type", fmt.Sprintf("%T", v))
			}
		}

		if resp.Cursor != nil && *resp.Cursor != "" {
			cursor = *resp.Cursor
			continue
		}

		break
	}

	slog.InfoContext(ctx, "reply count", "count", len(respList))

	if unreadResp.Count != 0 {
		err = bsky.NotificationUpdateSeen(ctx, xrpcc, &bsky.NotificationUpdateSeen_Input{
			SeenAt: now.Format(time.RFC3339Nano),
		})
		if err != nil {
			slog.ErrorContext(ctx, "error raised by app.bsky.notification.updateSeen", "error", err)
			return nil, err
		}

		slog.DebugContext(ctx, "update notification seen", "now", now)
	}

	return respList, nil
}
