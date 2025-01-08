package utils

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	comagnos "github.com/bluesky-social/indigo/api/agnostic"
	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	atprotoutil "github.com/bluesky-social/indigo/util"
	"github.com/bluesky-social/indigo/xrpc"
)

func DoesMentionMe(_ context.Context, me *xrpc.AuthInfo, post *bsky.FeedPost) bool {
	if me.Did != "" {
		for _, facet := range post.Facets {
			for _, f := range facet.Features {
				if v := f.RichtextFacet_Mention; v != nil {
					if me.Did == v.Did {
						return true
					}
				}
			}
		}
	}
	if me.Handle != "" {
		if strings.Contains(post.Text, me.Handle) {
			return true
		}
	}

	return false
}

func HasAlreadyReplied(_ context.Context, me *xrpc.AuthInfo, thread *bsky.FeedGetPostThread_Output) bool {
	if thread.Thread == nil {
		return false
	}
	if thread.Thread.FeedDefs_ThreadViewPost == nil {
		return false
	}
	for _, reply := range thread.Thread.FeedDefs_ThreadViewPost.Replies {
		if reply.FeedDefs_ThreadViewPost == nil {
			continue
		}
		if reply.FeedDefs_ThreadViewPost.Post.Author.Did == me.Did {
			return true
		}
	}

	return false
}

// Resolve the parent record and copy whatever the root reply reference there is
// If none exists, then the parent record was a top-level post, so that parent reference can be reused as the root value
func GetReplyRefs(ctx context.Context, xrpcc *xrpc.Client, parent comatproto.RepoStrongRef) bsky.FeedPost_ReplyRef {
	parsedUri, err := atprotoutil.ParseAtUri(parent.Uri)
	if err != nil {
		slog.Error("error on parsing uri", "error", err)
		panic(err)
	}
	resp, err := comagnos.RepoGetRecord(ctx, xrpcc, parent.Cid, "app.bsky.feed.post", xrpcc.Auth.Did, parsedUri.Rkey)
	if err != nil {
		slog.Error("error on getting parent record", "error", err)
		panic(err)
	}

	var parentRecord struct {
		reply *bsky.FeedPost_ReplyRef
	}
	err = json.Unmarshal(*resp.Value, parentRecord)
	if err != nil {
		slog.Error("error on unmarshalling parent record", "error", err)
		panic(err)
	}

	var root comatproto.RepoStrongRef

	parentReply := parentRecord.reply
	if parentReply != nil {
		root = *parentReply.Root
	} else {
		// The parent record is a top-level post, so it is also the root
		root = parent
	}

	return bsky.FeedPost_ReplyRef{
		Root:   &root,
		Parent: &parent,
	}
}
