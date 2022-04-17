package service

import (
	"sort"
	"strings"
	"time"

	"github.com/umputun/remark42/backend/app/store"
)

// Tree is formatter making tree from the list of comments
type Tree struct {
	Nodes []*Node        `json:"comments"`
	Info  store.PostInfo `json:"info,omitempty"`
}

// Node is a comment with optional replies
type Node struct {
	Comment    store.Comment `json:"comment"`
	Replies    []*Node       `json:"replies,omitempty"`
	tsModified time.Time
	tsCreated  time.Time
}

// recurData wraps all fields used in recursive processing as intermediate results
type recurData struct {
	tsModified time.Time
	tsCreated  time.Time
	visible    bool
}

// MakeTree gets unsorted list of comments and produces Tree
// It will make store.PostInfo by itself and will mark Info.ReadOnly based on passed readOnlyAge
// Tree maker is local and has no access to the data store. By this reason it has to make Info and won't be able
// to handle store's read-only status. This status should be set by caller.
func MakeTree(comments []store.Comment, sortType string, readOnlyAge int) *Tree {
	if len(comments) == 0 {
		return &Tree{}
	}

	res := Tree{
		Info: store.PostInfo{
			URL:     comments[0].Locator.URL,
			FirstTS: comments[0].Timestamp,
			LastTS:  comments[0].Timestamp,
		},
	}
	res.Info.Count = len(res.filter(comments, func(c store.Comment) bool { return !c.Deleted }))

	topComments := res.filter(comments, func(c store.Comment) bool { return c.ParentID == "" })

	res.Nodes = []*Node{}
	for _, rootComment := range topComments {
		node := Node{Comment: rootComment}

		rd := recurData{}
		commentsTree, tsModified, tsCreated := res.proc(comments, &node, &rd, rootComment.ID)
		// skip deleted with no sub-comments ar all sub-comments deleted
		if rootComment.Deleted && (len(commentsTree.Replies) == 0 || !rd.visible) {
			continue
		}

		commentsTree.tsModified, commentsTree.tsCreated = tsModified, tsCreated
		if commentsTree.tsCreated.Before(res.Info.FirstTS) {
			res.Info.FirstTS = commentsTree.tsCreated
		}
		if commentsTree.tsModified.After(res.Info.LastTS) {
			res.Info.LastTS = commentsTree.tsModified
		}

		res.Info.ReadOnly = readOnlyAge > 0 && !res.Info.FirstTS.IsZero() &&
			res.Info.FirstTS.AddDate(0, 0, readOnlyAge).Before(time.Now())

		res.Nodes = append(res.Nodes, commentsTree)
	}

	res.sortNodes(sortType)
	return &res
}

// proc makes tree for one top-level comment recursively
func (t *Tree) proc(comments []store.Comment, node *Node, rd *recurData, parentID string) (result *Node, modified, created time.Time) {
	if rd.tsModified.IsZero() || rd.tsCreated.IsZero() {
		rd.tsModified, rd.tsCreated = node.Comment.Timestamp, node.Comment.Timestamp
	}

	repComments := t.filter(comments, func(comment store.Comment) bool { return comment.ParentID == parentID })
	for _, rc := range repComments {
		if !rc.Timestamp.IsZero() && rc.Timestamp.After(rd.tsModified) && !rc.Deleted {
			rd.tsModified = rc.Timestamp
		}
		if !rc.Timestamp.IsZero() && rc.Timestamp.Before(rd.tsCreated) && !rc.Deleted {
			rd.tsCreated = rc.Timestamp
		}
		if !rc.Deleted {
			rd.visible = true // indicates top-level should be visible
		}
		rnode := &Node{Comment: rc, Replies: []*Node{}}
		node.Replies = append(node.Replies, rnode)
		t.proc(comments, rnode, rd, rc.ID)
		if !rd.visible || (len(rnode.Replies) == 0 && rc.Deleted) { // clean all-deleted subtree
			node.Replies = node.Replies[:len(node.Replies)-1]
		}
	}
	// replies always sorted by time
	sort.Slice(node.Replies, func(i, j int) bool {
		return node.Replies[i].Comment.Timestamp.Before(node.Replies[j].Comment.Timestamp)
	})
	return node, rd.tsModified, rd.tsCreated
}

// filter returns comments for parentID
func (t *Tree) filter(comments []store.Comment, fn func(comment store.Comment) bool) []store.Comment {
	f := []store.Comment{}
	for _, c := range comments {
		if fn(c) {
			f = append(f, c)
		}
	}
	return f
}

// sort list of nodes, i.e. top-level comments
// time sort uses tsModified from latest reply
func (t *Tree) sortNodes(sortType string) {
	sort.Slice(t.Nodes, func(i, j int) bool {
		switch sortType {
		case "+time", "-time", "time":
			if strings.HasPrefix(sortType, "-") {
				return t.Nodes[i].Comment.Timestamp.After(t.Nodes[j].Comment.Timestamp)
			}
			return t.Nodes[i].Comment.Timestamp.Before(t.Nodes[j].Comment.Timestamp)

		case "+active", "-active", "active":
			if strings.HasPrefix(sortType, "-") {
				return t.Nodes[i].tsModified.After(t.Nodes[j].tsModified)
			}
			return t.Nodes[i].tsModified.Before(t.Nodes[j].tsModified)

		case "+score", "-score", "score":
			if strings.HasPrefix(sortType, "-") {
				if t.Nodes[i].Comment.Score == t.Nodes[j].Comment.Score {
					return t.Nodes[i].Comment.Timestamp.Before(t.Nodes[j].Comment.Timestamp)
				}
				return t.Nodes[i].Comment.Score > t.Nodes[j].Comment.Score
			}
			if t.Nodes[i].Comment.Score == t.Nodes[j].Comment.Score {
				return t.Nodes[i].Comment.Timestamp.Before(t.Nodes[j].Comment.Timestamp)
			}
			return t.Nodes[i].Comment.Score < t.Nodes[j].Comment.Score

		case "+controversy", "-controversy", "controversy":
			if strings.HasPrefix(sortType, "-") {
				if t.Nodes[i].Comment.Controversy == t.Nodes[j].Comment.Controversy {
					return t.Nodes[i].Comment.Timestamp.Before(t.Nodes[j].Comment.Timestamp)
				}
				return t.Nodes[i].Comment.Controversy > t.Nodes[j].Comment.Controversy
			}
			if t.Nodes[i].Comment.Controversy == t.Nodes[j].Comment.Controversy {
				return t.Nodes[i].Comment.Timestamp.Before(t.Nodes[j].Comment.Timestamp)
			}
			return t.Nodes[i].Comment.Controversy < t.Nodes[j].Comment.Controversy

		default:
			return t.Nodes[i].Comment.Timestamp.Before(t.Nodes[j].Comment.Timestamp)
		}
	})
}
