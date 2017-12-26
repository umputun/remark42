package format

import (
	"sort"
	"strings"

	"github.com/umputun/remark/app/store"
)

// Tree is formatter as comment tree list of comments
type Tree struct {
	Nodes []*Node `json:"comments"`
}

// Node is a comment with optional replies
type Node struct {
	Comment store.Comment `json:"comment"`
	Replies []*Node       `json:"replies,omitempty"`
}

// MakeTree gets unsorted list of comments and produces Tree
func MakeTree(comments []store.Comment, sortType string) (res Tree) {
	res = Tree{}

	repComments := res.filter(comments, func(c store.Comment) bool { return c.ParentID == "" })
	for _, rc := range repComments {
		node := Node{Comment: rc}
		res.Nodes = append(res.Nodes, res.proc(comments, &node, rc.ID))
	}

	// sort result according to sortType
	sort.Slice(res.Nodes, func(i, j int) bool {
		switch sortType {
		case "+time", "-time", "time":
			if strings.HasPrefix(sortType, "-") {
				return res.Nodes[i].Comment.Timestamp.After(res.Nodes[j].Comment.Timestamp)
			}
			return res.Nodes[i].Comment.Timestamp.Before(res.Nodes[j].Comment.Timestamp)

		case "+score", "-score", "score":
			if strings.HasPrefix(sortType, "-") {
				return res.Nodes[i].Comment.Score > res.Nodes[j].Comment.Score
			}
			return res.Nodes[i].Comment.Score < res.Nodes[j].Comment.Score

		default:
			return res.Nodes[i].Comment.Timestamp.Before(res.Nodes[j].Comment.Timestamp)
		}
	})

	return res
}

func (t *Tree) proc(comments []store.Comment, node *Node, parentID string) *Node {
	repComments := t.filter(comments, func(c store.Comment) bool { return c.ParentID == parentID })
	for _, rc := range repComments {
		rnode := &Node{Comment: rc, Replies: []*Node{}}
		node.Replies = append(node.Replies, rnode)

		// replies always sorted by time
		sort.Slice(node.Replies, func(i, j int) bool {
			return node.Replies[i].Comment.Timestamp.Before(node.Replies[j].Comment.Timestamp)
		})
		t.proc(comments, rnode, rc.ID)
	}
	return node
}

func (t *Tree) filter(comments []store.Comment, fn func(c store.Comment) bool) (f []store.Comment) {
	for _, c := range comments {
		if fn(c) {
			f = append(f, c)
		}
	}
	return f
}
