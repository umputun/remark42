package format

import (
	"sort"
	"strings"

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
}

// MakeTree gets unsorted list of comments and produces Tree
func MakeTree(comments []store.Comment, sortType string) *Tree {
	res := Tree{}

	topComments := res.filter(comments, "")
	res.Nodes = []*Node{}
	for _, rootComment := range topComments {
		node := Node{Comment: rootComment}

		commentsTree := res.proc(comments, &node, rootComment.ID)
		if rootComment.Deleted && len(commentsTree.Replies) == 0 { // skip deleted with no subcomments
			continue
		}
		res.Nodes = append(res.Nodes, commentsTree)
	}

	res.sortNodes(sortType)
	return &res
}

// proc makes tree for one top-level comment recursively
func (t *Tree) proc(comments []store.Comment, node *Node, parentID string) *Node {
	repComments := t.filter(comments, parentID)
	for _, rc := range repComments {
		rnode := &Node{Comment: rc, Replies: []*Node{}}
		node.Replies = append(node.Replies, rnode)
		t.proc(comments, rnode, rc.ID)
	}
	// replies always sorted by time
	sort.Slice(node.Replies, func(i, j int) bool {
		return node.Replies[i].Comment.Timestamp.Before(node.Replies[j].Comment.Timestamp)
	})
	return node
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
func (t *Tree) sortNodes(sortType string) {

	sort.Slice(t.Nodes, func(i, j int) bool {
		switch sortType {
		case "+time", "-time", "time":
			if strings.HasPrefix(sortType, "-") {
				return t.Nodes[i].Comment.Timestamp.After(t.Nodes[j].Comment.Timestamp)
			}
			return t.Nodes[i].Comment.Timestamp.Before(t.Nodes[j].Comment.Timestamp)

		case "+score", "-score", "score":
			if strings.HasPrefix(sortType, "-") {
				return t.Nodes[i].Comment.Score > t.Nodes[j].Comment.Score
			}
			return t.Nodes[i].Comment.Score < t.Nodes[j].Comment.Score

		default:
			return t.Nodes[i].Comment.Timestamp.Before(t.Nodes[j].Comment.Timestamp)
		}
	})
}
