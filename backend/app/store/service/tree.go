package service

import (
	"sort"
	"time"

	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/engine"
)

// Tree is formatter making tree from the list of comments
type Tree struct {
	Nodes []*Node `json:"comments"`

	countLeft          int
	lastLimitedComment string
}

// Node is a comment with optional replies
type Node struct {
	Comment    store.Comment `json:"comment"`
	Replies    []*Node       `json:"replies,omitempty"`
	tsModified time.Time
}

// recurData wraps all fields used in recursive processing as intermediate results
type recurData struct {
	tsModified time.Time
	visible    bool
}

// MakeTree gets unsorted list of comments and produces Tree
func MakeTree(comments []store.Comment, sortType string, limit int, offsetID string, user store.User, premoderationActive bool) *Tree {
	if len(comments) == 0 {
		return &Tree{}
	}

	res := Tree{}

	topComments := res.filter(comments, func(c store.Comment) bool { return c.ParentID == "" })

	res.Nodes = []*Node{}
	for _, rootComment := range topComments {

		// mask text
		// todo validate if this can use a function from the premoderator
		authorOrAdmin := user.Admin || rootComment.User.ID == user.ID
		if premoderationActive && rootComment.PendingApproval {
			if !authorOrAdmin {
				rootComment.Text = ""
				rootComment.Orig = ""
			}
		}

		node := Node{Comment: rootComment}

		rd := recurData{visible: true}
		commentsTree := res.proc(comments, &node, &rd, rootComment.ID, user, premoderationActive)

		// skip deleted with no sub-comments and all sub-comments deleted and unapproved comments, if premodaration is active
		hide := rootComment.Deleted
		if premoderationActive && !authorOrAdmin {
			hide = hide || rootComment.PendingApproval
		}

		if (hide && len(commentsTree.Replies) == 0) || !rd.visible {
			continue
		}

		res.Nodes = append(res.Nodes, commentsTree)
	}

	res.sortNodes(sortType)
	res.limit(limit, offsetID)
	return &res
}

// CountLeft returns number of comments left after limit, 0 if no limit was set
func (t *Tree) CountLeft() int {
	return t.countLeft
}

// LastComment returns ID of the last comment in the tree after limit, empty string if no limit was set
func (t *Tree) LastComment() string {
	return t.lastLimitedComment
}

// proc makes tree for one top-level comment recursively
func (t *Tree) proc(comments []store.Comment, node *Node, rd *recurData, parentID string, user store.User, premoderationActive bool) (result *Node) {
	if rd.tsModified.IsZero() {
		rd.tsModified = node.Comment.Timestamp
	}

	repComments := t.filter(comments, func(comment store.Comment) bool { return comment.ParentID == parentID })
	for _, rc := range repComments {

		// mask text
		authorOrAdmin := user.Admin || rc.User.ID == user.ID
		if premoderationActive && rc.PendingApproval {
			if !authorOrAdmin {
				rc.Text = ""
				rc.Orig = ""
			}
		}

		if !rc.Timestamp.IsZero() && rc.Timestamp.After(rd.tsModified) && !rc.Deleted {
			rd.tsModified = rc.Timestamp
		}
		if !rc.Deleted {
			rd.visible = true // indicates top-level should be visible
		}
		rnode := &Node{Comment: rc, Replies: []*Node{}}

		node.Replies = append(node.Replies, rnode)

		t.proc(comments, rnode, rd, rc.ID, user, premoderationActive)

		shouldHide := false
		// if the comment is not approved AND the user is not admin AND it's a leaf, hide it
		if premoderationActive {
			shouldHide = !authorOrAdmin && len(rnode.Replies) == 0 && rc.PendingApproval
		}

		if shouldHide || !rd.visible || (len(rnode.Replies) == 0 && rc.Deleted) { // clean all-deleted subtree
			node.Replies = node.Replies[:len(node.Replies)-1]
		}
	}
	// replies always sorted by time
	sort.Slice(node.Replies, func(i, j int) bool {
		return node.Replies[i].Comment.Timestamp.Before(node.Replies[j].Comment.Timestamp)
	})
	node.tsModified = rd.tsModified
	return node
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
// active sort uses tsModified from latest reply
func (t *Tree) sortNodes(sortType string) {
	active := sortType == "+active" || sortType == "-active" || sortType == "active"
	sort.Slice(t.Nodes, func(i, j int) bool {
		a, b := t.Nodes[i], t.Nodes[j]
		if !active {
			return engine.LessComments(&a.Comment, &b.Comment, sortType)
		}
		if sortType == "-active" {
			return a.tsModified.After(b.tsModified)
		}
		return a.tsModified.Before(b.tsModified)
	})
}

// limit limits number of comments in tree and sets countLeft and lastLimitedComment,
// starting with comment next after offsetID.
//
// If offsetID is empty or invalid, it starts from the beginning. If limit is 0, it doesn't limit anything.
//
// Limit is applied to top-level comments only, so top-level comments only returned with all replies,
// and lastLimitedComment is set to the last top-level comment and not last reply in it.
//
// In case limit is less than the number of replies to first comment after given offset, that first comment is
// returned completely with all replies.
func (t *Tree) limit(limit int, offsetID string) {
	if offsetID == "" && limit <= 0 {
		return
	}

	start := 0
	if offsetID != "" {
		for i, n := range t.Nodes {
			if n.Comment.ID == offsetID {
				start = i + 1
				break
			}
		}
	}

	if start == len(t.Nodes) { // if the start index is beyond the available nodes, clear the nodes
		t.Nodes = []*Node{}
		return
	}

	t.Nodes = t.Nodes[start:]

	// if there is only offset and no limit, there are no comments left and no point in returning
	// the last comment ID as there are no comments beyond it.
	if limit <= 0 {
		return
	}

	// traverse and limit the number of top-level nodes, including their replies
	limitedNodes := []*Node{}
	commentsCount := 0

	for _, node := range t.Nodes {
		repliesCount := countReplies(node) + 1 // count this node and its replies

		// if the limit is already reached or exceeded, calculate countLeft and move to the next node
		if commentsCount >= limit {
			t.countLeft += repliesCount
			continue
		}

		// stop once adding this subtree would exceed the limit, as long as we already have a node;
		// a subtree that fits exactly is still included, and the first node is always returned in full.
		if commentsCount+repliesCount > limit && len(limitedNodes) > 0 {
			t.countLeft += repliesCount
			commentsCount = limit // adjust commentsCount to stop checking limit for the next nodes
			continue
		}

		// add the node and its replies to the list
		limitedNodes = append(limitedNodes, node)
		commentsCount += repliesCount
	}

	t.lastLimitedComment = limitedNodes[len(limitedNodes)-1].Comment.ID
	t.Nodes = limitedNodes
}

// countReplies counts the total number of replies recursively for a given node.
func countReplies(node *Node) int {
	count := 0
	for _, reply := range node.Replies {
		count++                      // count the reply itself
		count += countReplies(reply) // recursively count its replies
	}
	return count
}
