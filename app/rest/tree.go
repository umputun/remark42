package rest

import (
	"sort"
	"strings"
	"time"

	"github.com/umputun/remark/app/store"
)

// Tree is formatter making tree from list of comments
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
func MakeTree(comments []store.Comment, sortType string, readOnlyAge int) *Tree {
	if len(comments) == 0 {
		return &Tree{}
	}

	res := Tree{
		Info: store.PostInfo{
			URL:     comments[0].Locator.URL,
			Count:   len(comments), // TODO: includes deleted?
			FirstTS: comments[0].Timestamp,
			LastTS:  comments[0].Timestamp,
		},
	}

	topComments := res.filter(comments, "")
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
func (t *Tree) proc(comments []store.Comment, node *Node, rd *recurData, parentID string) (*Node, time.Time, time.Time) {

	if rd.tsModified.IsZero() || rd.tsCreated.IsZero() {
		rd.tsModified, rd.tsCreated = node.Comment.Timestamp, node.Comment.Timestamp
	}

	repComments := t.filter(comments, parentID)
	for _, rc := range repComments {
		if rc.Timestamp.After(rd.tsModified) {
			rd.tsModified = rc.Timestamp
		}
		if rc.Timestamp.Before(rd.tsCreated) {
			rd.tsCreated = rc.Timestamp
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
	return node, rd.tsModified, rd.tsCreated
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

		default:
			return t.Nodes[i].Comment.Timestamp.Before(t.Nodes[j].Comment.Timestamp)
		}
	})
}
