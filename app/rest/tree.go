package rest

import (
	"sort"
	"strings"
	"time"

	"github.com/umputun/remark/app/store"
)

// Tree is formatter making tree from list of comments
type Tree struct {
	Nodes []*Node `json:"comments"`
}

// Node is a comment with optional replies
type Node struct {
	Comment store.Comment `json:"comment"`
	Replies []*Node       `json:"replies,omitempty"`
	ts      time.Time
}

// recurData wraps all fileds used in recursive processing as intermediate results
type recurData struct {
	ts      time.Time
	visible bool
}

// MakeTree gets unsorted list of comments and produces Tree
func MakeTree(comments []store.Comment, sortType string) *Tree {
	res := Tree{}

	topComments := res.filter(comments, "")
	res.Nodes = []*Node{}
	for _, rootComment := range topComments {
		node := Node{Comment: rootComment}

		rd := recurData{}
		commentsTree, t := res.proc(comments, &node, &rd, rootComment.ID)
		// skip deleted with no sub-comments ar all sub-comments deleted
		if rootComment.Deleted && (len(commentsTree.Replies) == 0 || !rd.visible) {
			continue
		}
		commentsTree.ts = t
		res.Nodes = append(res.Nodes, commentsTree)
	}

	res.sortNodes(sortType)
	return &res
}

// proc makes tree for one top-level comment recursively
func (t *Tree) proc(comments []store.Comment, node *Node, rd *recurData, parentID string) (*Node, time.Time) {

	if rd.ts.IsZero() {
		rd.ts = node.Comment.Timestamp
	}

	repComments := t.filter(comments, parentID)
	for _, rc := range repComments {
		if rc.Timestamp.After(rd.ts) {
			rd.ts = rc.Timestamp
		}
		if !rc.Deleted {
			rd.visible = true
		}
		rnode := &Node{Comment: rc, Replies: []*Node{}}
		node.Replies = append(node.Replies, rnode)
		t.proc(comments, rnode, rd, rc.ID)
	}
	// replies always sorted by time
	sort.Slice(node.Replies, func(i, j int) bool {
		return node.Replies[i].Comment.Timestamp.Before(node.Replies[j].Comment.Timestamp)
	})
	return node, rd.ts
}

// filter returns comments for parentID
func (t *Tree) filter(comments []store.Comment, parentID string) (f []store.Comment) {

	for _, c := range comments {
		if c.ParentID == parentID {
			f = append(f, c)
		}
	}
	return f
}

// sort list of nodes, i.e. top-level comments
// time sort uses ts from latest reply
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
				return t.Nodes[i].ts.After(t.Nodes[j].ts)
			}
			return t.Nodes[i].ts.Before(t.Nodes[j].ts)

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

		default:
			return t.Nodes[i].Comment.Timestamp.Before(t.Nodes[j].Comment.Timestamp)
		}
	})
}
