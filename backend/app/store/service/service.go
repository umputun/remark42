// Package service wraps engine interfaces with common logic unrelated to any particular engine implementation.
// All consumers should be using service.DataStore and not the naked engine!
package service

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/google/uuid"
	bf "github.com/russross/blackfriday/v2"

	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/admin"
	"github.com/umputun/remark42/backend/app/store/engine"
	"github.com/umputun/remark42/backend/app/store/image"
)

// DataStore wraps store.Interface with additional methods
type DataStore struct {
	Engine              engine.Interface
	EditDuration        time.Duration
	AdminStore          admin.Store
	MinCommentSize      int
	MaxCommentSize      int
	MaxVotes            int
	RestrictSameIPVotes struct {
		Enabled  bool
		Duration time.Duration
	}
	PositiveScore          bool
	TitleExtractor         *TitleExtractor
	RestrictedWordsMatcher *RestrictedWordsMatcher
	ImageService           *image.Service
	AdminEdits             bool // allow admin unlimited edits

	// granular locks
	scopedLocks struct {
		sync.Mutex
		sync.Once
		locks map[string]sync.Locker
	}
}

// UserMetaData keeps info about user flags and details
type UserMetaData struct {
	ID      string `json:"id"`
	Blocked struct {
		Status bool      `json:"status"`
		Until  time.Time `json:"until"`
	} `json:"blocked"`
	Verified bool                   `json:"verified"`
	Details  engine.UserDetailEntry `json:"details"`
}

// PostMetaData keeps info about post flags
type PostMetaData struct {
	URL      string `json:"url"`
	ReadOnly bool   `json:"read_only"`
}

const defaultCommentMaxSize = 2048
const maxLastCommentsReply = 5000

// UnlimitedVotes doesn't restrict MaxVotes
const UnlimitedVotes = -1

var nonAdminUser = store.User{}

// ErrRestrictedWordsFound returned in case comment text contains restricted words
var ErrRestrictedWordsFound = fmt.Errorf("comment contains restricted words")

// Create prepares comment and forward to Interface.Create
func (s *DataStore) Create(comment store.Comment) (commentID string, err error) {
	if comment, err = s.prepareNewComment(comment); err != nil {
		return "", fmt.Errorf("failed to prepare comment: %w", err)
	}

	// restricted words are checked on new comments only, an import brings back what the site already had
	if !comment.Imported && s.RestrictedWordsMatcher != nil && s.RestrictedWordsMatcher.Match(comment.Locator.SiteID, comment.Text) {
		return "", ErrRestrictedWordsFound
	}

	func() { // keep input title and set to extracted if missing
		if s.TitleExtractor == nil || comment.PostTitle != "" {
			return
		}
		title, e := s.TitleExtractor.Get(comment.Locator.URL)
		if e != nil {
			log.Printf("[WARN] failed to set title, %v", e)
			return
		}
		comment.PostTitle = title
	}()

	commentID, err = s.Engine.Create(comment)
	s.submitImages(comment)

	if e := s.AdminStore.OnEvent(comment.Locator.SiteID, admin.EvCreate); e != nil {
		log.Printf("[WARN] failed to send create event, %s", e)
	}
	return commentID, err
}

// Find wraps engine's Find call and alter results if needed. User used to alter comments
// in order to differentiate between user's comments vs others comments.
func (s *DataStore) Find(locator store.Locator, sortMethod string, user store.User) ([]store.Comment, error) {
	return s.FindSince(locator, sortMethod, user, time.Time{})
}

// FindSince wraps engine's Find call and alter results if needed. Returns comments after since tx
func (s *DataStore) FindSince(locator store.Locator, sortMethod string, user store.User, since time.Time) ([]store.Comment, error) {
	req := engine.FindRequest{Locator: locator, Sort: sortMethod, Since: since}
	comments, err := s.Engine.Find(req)
	if err != nil {
		return comments, err
	}

	changedSort := false
	flags := s.newUserFlagCache()
	// sets votes controversy for comments added prior to #274
	// also sanitizes locator.URL for comments added prior to #927
	for i, c := range comments {
		if c.Controversy == 0 && len(c.Votes) > 0 {
			c.Controversy = s.controversy(s.upsAndDowns(c))
			if !changedSort && strings.Contains(sortMethod, "controversy") { // trigger sort change
				changedSort = true
			}
		}
		comments[i] = s.alterCommentCached(c, user, flags)
	}

	// resort commits if altered
	if changedSort {
		comments = engine.SortComments(comments, sortMethod)
	}

	return comments, nil
}

// FindRaw returns the post's comments sorted by time as the engine stores them, without the changes
// Find makes for a reader: blocked and verified marks, hidden votes and voter IPs. Used for backups.
func (s *DataStore) FindRaw(locator store.Locator) ([]store.Comment, error) {
	return s.Engine.Find(engine.FindRequest{Locator: locator, Sort: "time"})
}

// Get comment by ID
func (s *DataStore) Get(locator store.Locator, commentID string, user store.User) (store.Comment, error) {
	c, err := s.Engine.Get(engine.GetRequest{Locator: locator, CommentID: commentID})
	if err != nil {
		return store.Comment{}, err
	}
	return s.alterComment(c, user), nil
}

// Put updates comment, mutable parts only
func (s *DataStore) Put(locator store.Locator, comment store.Comment) error {
	comment.Locator = locator
	return s.Engine.Update(comment)
}

// GetUserEmail gets user email
func (s *DataStore) GetUserEmail(siteID, userID string) (string, error) {
	entry, err := s.userDetail(siteID, userID, engine.UserEmail, "")
	return entry.Email, err
}

// SetUserEmail sets user email
func (s *DataStore) SetUserEmail(siteID, userID, value string) (string, error) {
	entry, err := s.userDetail(siteID, userID, engine.UserEmail, value)
	return entry.Email, err
}

// GetUserTelegram gets user telegram
func (s *DataStore) GetUserTelegram(siteID, userID string) (string, error) {
	entry, err := s.userDetail(siteID, userID, engine.UserTelegram, "")
	return entry.Telegram, err
}

// SetUserTelegram sets user telegram
func (s *DataStore) SetUserTelegram(siteID, userID, value string) (string, error) {
	entry, err := s.userDetail(siteID, userID, engine.UserTelegram, value)
	return entry.Telegram, err
}

// userDetail gets the user's detail, or sets it when update is not empty.
// It returns an empty entry when the engine has none or more than one
func (s *DataStore) userDetail(siteID, userID string, detail engine.UserDetail, update string) (engine.UserDetailEntry, error) {
	res, err := s.Engine.UserDetail(engine.UserDetailRequest{
		Detail:  detail,
		Locator: store.Locator{SiteID: siteID},
		UserID:  userID,
		Update:  update,
	})
	if err != nil || len(res) != 1 {
		return engine.UserDetailEntry{}, err
	}
	return res[0], nil
}

// DeleteUserDetail deletes user detail
func (s *DataStore) DeleteUserDetail(siteID, userID string, detail engine.UserDetail) error {
	return s.Engine.Delete(engine.DeleteRequest{
		Locator:    store.Locator{SiteID: siteID},
		UserID:     userID,
		UserDetail: detail,
	})
}

// ResubmitStagingImages retrieves timestamp of the oldest image in staging and
// calls s.submitImages on all comments newer than it
func (s *DataStore) ResubmitStagingImages(sites []string) error {
	info, err := s.ImageService.Info()
	if err != nil {
		return err
	}
	ts := info.FirstStagingImageTS
	if ts.IsZero() {
		return nil
	}
	var errs []error
	for _, site := range sites {
		locator := store.Locator{SiteID: site}
		comments, err := s.FindSince(locator, "time", store.User{}, ts)
		if err != nil {
			errs = append(errs, fmt.Errorf("problem finding comments for site %s: %w", site, err))
		}
		for _, c := range comments {
			s.submitImages(c)
		}
	}
	return errors.Join(errs...)
}

// submitImages initiated delayed commit of all images from the comment uploaded to remark42
func (s *DataStore) submitImages(comment store.Comment) {
	idsFn := func() []string { // get all ids from comment's text
		// this can be called after last edit, we have to retrieve fresh comment
		cc, err := s.Engine.Get(engine.GetRequest{Locator: comment.Locator, CommentID: comment.ID})
		if err != nil {
			log.Printf("[WARN] can't get comment's %s text for image extraction, %v", comment.ID, err)
			return nil
		}
		imgIDs := s.ImageService.ExtractPictures(cc.Text)
		if len(imgIDs) > 0 {
			log.Printf("[DEBUG] image ids extracted from %s - %+v", comment.ID, imgIDs)
		}
		return imgIDs
	}

	var err error
	if comment.Imported {
		err = s.ImageService.Commit(idsFn)
	} else {
		s.ImageService.Submit(idsFn)
	}

	if err != nil {
		log.Printf("[WARN] failed to commit comment's images: %v", err)
	}
}

// prepareNewComment sets new comment fields, hashing and sanitizing data
func (s *DataStore) prepareNewComment(comment store.Comment) (store.Comment, error) {
	// fill ID and time if empty
	if comment.ID == "" {
		comment.ID = uuid.New().String()
	}
	if comment.Timestamp.IsZero() {
		comment.Timestamp = time.Now()
	}
	// reset votes if nothing
	if comment.Votes == nil {
		comment.Votes = make(map[string]bool)
	}
	comment.Sanitize() // clear potentially dangerous js from all parts of comment

	secret, err := s.getSecret(comment.Locator.SiteID)
	if err != nil {
		return store.Comment{}, fmt.Errorf("can't get secret for site %s: %w", comment.Locator.SiteID, err)
	}
	comment.User.HashIP(secret) // replace ip by hash
	return comment, nil
}

// DeleteAll removes all data from site
func (s *DataStore) DeleteAll(siteID string) error {
	req := engine.DeleteRequest{Locator: store.Locator{SiteID: siteID}}
	return s.Engine.Delete(req)
}

// SetPin pin/un-pin comment as special
func (s *DataStore) SetPin(locator store.Locator, commentID string, status bool) error {
	comment, err := s.Engine.Get(engine.GetRequest{Locator: locator, CommentID: commentID})
	if err != nil {
		return err
	}
	comment.Pin = status
	comment.Locator = locator
	return s.Engine.Update(comment)
}

// VoteReq is the request ot make a vote
type VoteReq struct {
	Locator   store.Locator
	CommentID string
	UserID    string
	UserIP    string
	Val       bool
}

// Vote for comment by id and locator
func (s *DataStore) Vote(req VoteReq) (comment store.Comment, err error) {
	cLock := s.getScopedLocks(req.Locator.URL) // get lock for URL scope
	cLock.Lock()                               // prevents race on voting
	defer cLock.Unlock()

	comment, err = s.Engine.Get(engine.GetRequest{Locator: req.Locator, CommentID: req.CommentID})
	if err != nil {
		return comment, err
	}

	if comment.User.ID == req.UserID && req.UserID != "dev" {
		return comment, fmt.Errorf("user %s can not vote for his own comment %s", req.UserID, req.CommentID)
	}

	if comment.Votes == nil {
		comment.Votes = make(map[string]bool)
	}

	v, voted := comment.Votes[req.UserID]
	if voted && v == req.Val { // voted before and same vote (+/-) again. Change allowed, i.e. +, - or -, + is fine
		return comment, fmt.Errorf("user %s already voted for %s", req.UserID, req.CommentID)
	}

	secret, err := s.getSecret(comment.Locator.SiteID)
	if err != nil {
		return store.Comment{}, fmt.Errorf("can't get secret for site %s: %w", comment.Locator.SiteID, err)
	}
	userIPHash := store.HashValue(req.UserIP, secret)
	if s.isSameIPVote(req, userIPHash, comment) {
		return comment, fmt.Errorf("the same ip already voted for %s", req.CommentID)
	}

	maxVotes := s.MaxVotes // 0 value allowed and treated as "no votes allowed"
	if s.MaxVotes < 0 {    // any negative value reset max votes to unlimited
		maxVotes = UnlimitedVotes
	}

	if maxVotes >= 0 && len(comment.Votes) >= maxVotes {
		return comment, fmt.Errorf("maximum number of votes exceeded for comment %s", req.CommentID)
	}

	if s.PositiveScore && comment.Score <= 0 && !req.Val {
		return comment, fmt.Errorf("minimal score reached for comment %s", req.CommentID)
	}

	// drop ip hashes outside the same-ip window, they can't restrict a vote anymore; done with the
	// restriction off as well, so the hashes are not kept forever and turning it on later still works
	now := time.Now()
	for ipHash, ipVote := range comment.VotedIPs {
		if s.sameIPVoteExpired(ipVote, now) {
			delete(comment.VotedIPs, ipHash)
		}
	}

	// add ip hash to voted ip map
	if comment.VotedIPs == nil {
		comment.VotedIPs = map[string]store.VotedIPInfo{}
	}
	comment.VotedIPs[userIPHash] = store.VotedIPInfo{Timestamp: now, Value: req.Val}

	// reset vote if user changed to opposite. Effectively it is "forget about prev votes" to allow "+ - -" or "- + +" corrections
	if voted && v != req.Val {
		delete(comment.Votes, req.UserID)
		delete(comment.VotedIPs, userIPHash)
	}

	// add to voted map if first vote
	if !voted {
		comment.Votes[req.UserID] = req.Val
	}

	// update score
	if req.Val {
		comment.Score++
	} else {
		comment.Score--
	}

	comment.Vote = 0
	if vv, ok := comment.Votes[req.UserID]; ok {
		if vv {
			comment.Vote = 1
		} else {
			comment.Vote = -1
		}
	}

	if e := s.AdminStore.OnEvent(comment.Locator.SiteID, admin.EvVote); e != nil {
		log.Printf("[WARN] failed to send vote event, %s", e)
	}

	comment.Controversy = s.controversy(s.upsAndDowns(comment))
	comment.Locator = req.Locator
	return comment, s.Engine.Update(comment)
}

func (s *DataStore) isSameIPVote(req VoteReq, userIPHash string, comment store.Comment) bool {
	if req.UserIP == "" || !s.RestrictSameIPVotes.Enabled {
		return false
	}

	if v, ipFound := comment.VotedIPs[userIPHash]; ipFound {
		if v.Value != req.Val {
			return false // opposite direction vote allowed
		}
		return !s.sameIPVoteExpired(v, time.Now())
	}
	return false
}

// sameIPVoteExpired reports whether the vote recorded for an ip is outside the same-ip window at the given time.
// Zero duration means the window never ends.
func (s *DataStore) sameIPVoteExpired(v store.VotedIPInfo, now time.Time) bool {
	d := s.RestrictSameIPVotes.Duration
	return d != 0 && !v.Timestamp.Add(d).After(now)
}

// controversy calculates controversial index of votes
// source - https://github.com/reddit-archive/reddit/blob/master/r2/r2/lib/db/_sorts.pyx#L60
func (s *DataStore) controversy(ups, downs int) float64 {
	if downs <= 0 || ups <= 0 {
		return 0
	}

	magnitude := ups + downs
	balance := float64(downs) / float64(ups)
	if ups <= downs {
		balance = float64(ups) / float64(downs)
	}
	return math.Pow(float64(magnitude), balance)
}

// EditRequest contains fields needed for comment update
type EditRequest struct {
	Text    string
	Orig    string
	Summary string
	Delete  bool
	Admin   bool
}

// EditComment to edit text and update Edit info
func (s *DataStore) EditComment(locator store.Locator, commentID string, req EditRequest) (comment store.Comment, err error) {
	editAllowed := func(comment store.Comment) error {
		if req.Admin && s.AdminEdits {
			return nil
		}

		// edit allowed in editDuration window only
		if s.EditDuration > 0 && time.Now().After(comment.Timestamp.Add(s.EditDuration)) {
			return fmt.Errorf("too late to edit %s", commentID)
		}

		// edit rejected on replayed threads
		if s.HasReplies(comment) {
			return fmt.Errorf("parent comment with reply can't be edited, %s", commentID)
		}
		return nil
	}

	if comment, err = s.Engine.Get(engine.GetRequest{Locator: locator, CommentID: commentID}); err != nil {
		return comment, err
	}

	if err = editAllowed(comment); err != nil { //nolint gocritic
		return comment, err
	}

	// a soft-deleted comment keeps its author, who must not be able to write text back into it,
	// admin edits included; a repeated delete request is still accepted
	if comment.Deleted && !req.Delete {
		return comment, fmt.Errorf("deleted comment can't be edited, %s", commentID)
	}

	if req.Delete { // delete request
		if e := s.AdminStore.OnEvent(comment.Locator.SiteID, admin.EvDelete); e != nil {
			log.Printf("[WARN] failed to send delete event, %s", e)
		}
		comment.Deleted = true
		delReq := engine.DeleteRequest{Locator: locator, CommentID: commentID, DeleteMode: store.SoftDelete}
		if e := s.Engine.Delete(delReq); e != nil {
			return comment, e
		}
		s.deleteRemovedPictures(locator, commentID, comment.User.ID, comment.Text, "")
		return comment, nil
	}

	oldText := comment.Text
	comment.Text = req.Text
	comment.Orig = req.Orig
	comment.Edit = &store.Edit{Timestamp: time.Now(), Summary: req.Summary}
	comment.Locator = locator
	comment.Sanitize()

	// match the sanitized text, as Create does, because this is the text to store and show
	if s.RestrictedWordsMatcher != nil && s.RestrictedWordsMatcher.Match(comment.Locator.SiteID, comment.Text) {
		return comment, ErrRestrictedWordsFound
	}

	if e := s.AdminStore.OnEvent(comment.Locator.SiteID, admin.EvUpdate); e != nil {
		log.Printf("[WARN] failed to send update event, %s", e)
	}

	if e := s.Engine.Update(comment); e != nil {
		return comment, e
	}
	s.deleteRemovedPictures(locator, commentID, comment.User.ID, oldText, comment.Text)
	return comment, nil
}

// HasReplies checks if there is any reply to the comment among the comments of its post
func (s *DataStore) HasReplies(comment store.Comment) bool {
	comments, err := s.Engine.Find(engine.FindRequest{Locator: comment.Locator})
	if err != nil {
		log.Printf("[WARN] can't get comments for reply check, %v", err)
		return false
	}
	return slices.ContainsFunc(comments, func(c store.Comment) bool { return c.ParentID == comment.ID && !c.Deleted })
}

// UserReplies returns list of all comments replied to given user. It looks for them among the site's
// last maxLastCommentsReply comments, a number the engine caps (bolt returns at most 1000)
func (s *DataStore) UserReplies(siteID, userID string, limit int, duration time.Duration) ([]store.Comment, string, error) {
	comments, e := s.Last(siteID, maxLastCommentsReply, time.Time{}, nonAdminUser)
	if e != nil {
		return nil, "", fmt.Errorf("can't get last comments: %w", e)
	}
	replies := []store.Comment{}

	// get a comment for given userID in order to retrieve name
	userName := ""
	if cc, err := s.User(siteID, userID, 1, 0, nonAdminUser); err == nil && len(cc) > 0 {
		userName = cc[0].User.Name
	}

	// collect replies
	postIDs := map[string]map[string]bool{} // comment IDs of the posts read by parentMissing, by post URL
	for _, c := range comments {
		if len(replies) >= limit || time.Since(c.Timestamp) > duration {
			break
		}

		if c.ParentID != "" && !c.Deleted && c.User.ID != userID { // not interested in replies to yourself
			var pc store.Comment
			if pc, e = s.Get(c.Locator, c.ParentID, nonAdminUser); e != nil {
				if !s.parentMissing(c, postIDs) {
					return nil, "", fmt.Errorf("can't get parent comment: %w", e)
				}
				log.Printf("[DEBUG] skip reply %s to missing comment %s", c.ID, c.ParentID)
				continue
			}
			if pc.User.ID == userID {
				if userName == "" { // the user's own comments may all be deleted; a deleted one keeps the name
					userName = pc.User.Name
				}
				replies = append(replies, c)
			}
		}
	}

	return replies, userName, nil
}

// parentMissing reports whether the post of the reply was read and has no comment with the reply's ParentID,
// as for a reply stored before parents were checked. Get doesn't tell a missing comment from a failed read.
// A post is read once per postIDs, which keeps the comment IDs of the posts read so far by post URL
func (s *DataStore) parentMissing(reply store.Comment, postIDs map[string]map[string]bool) bool {
	ids, ok := postIDs[reply.Locator.URL]
	if !ok {
		comments, err := s.Engine.Find(engine.FindRequest{Locator: reply.Locator})
		if err != nil {
			return false
		}
		ids = make(map[string]bool, len(comments))
		for _, c := range comments {
			ids[c.ID] = true
		}
		postIDs[reply.Locator.URL] = ids
	}
	return !ids[reply.ParentID]
}

// SetTitle puts title from the locator.URL page and overwrites any existing title
func (s *DataStore) SetTitle(locator store.Locator, commentID string) (comment store.Comment, err error) {
	if s.TitleExtractor == nil {
		return comment, fmt.Errorf("no title extractor")
	}

	comment, err = s.Engine.Get(engine.GetRequest{Locator: locator, CommentID: commentID})
	if err != nil {
		return comment, err
	}

	// set title, overwrite the current one
	title, e := s.TitleExtractor.Get(comment.Locator.URL)
	if e != nil {
		return comment, e
	}
	comment.PostTitle = title
	comment.Locator = locator
	err = s.Engine.Update(comment)
	return comment, err
}

// Counts returns postID+count list for given comments
func (s *DataStore) Counts(siteID string, postIDs []string) ([]store.PostInfo, error) {
	res := []store.PostInfo{}
	for _, p := range postIDs {
		req := engine.FindRequest{Locator: store.Locator{SiteID: siteID, URL: p}}
		if c, err := s.Engine.Count(req); err == nil {
			res = append(res, store.PostInfo{URL: p, Count: c})
		}
	}
	return res, nil
}

// ValidateComment checks if comment size below max and user fields set.
// It also validates the absence of relative links as they are almost never the intention of the commenter,
// usually added by mistakes and only create confusion.
func (s *DataStore) ValidateComment(c *store.Comment) error {
	maxSize := s.MaxCommentSize
	if s.MaxCommentSize <= 0 {
		maxSize = defaultCommentMaxSize
	}
	if c.Orig == "" {
		return fmt.Errorf("empty comment text")
	}
	commentLength := len([]rune(c.Orig))
	if commentLength > maxSize {
		return fmt.Errorf("comment text exceeded max allowed size %d (%d)", maxSize, commentLength)
	}
	if s.MinCommentSize > 0 && commentLength < s.MinCommentSize {
		return fmt.Errorf("comment text is smaller than min allowed size %d (%d)", s.MinCommentSize, commentLength)
	}
	if c.User.ID == "" || c.User.Name == "" {
		return fmt.Errorf("empty user info")
	}

	// for validation purposes it's not important if SmartyPants formatting is disabled or enabled,
	// while for storing the comment that flag is set based on user preference
	mdExt, rend := store.GetMdExtensionsAndRenderer(false)
	parser := bf.New(bf.WithRenderer(rend), bf.WithExtensions(bf.CommonExtensions), bf.WithExtensions(mdExt))
	doc, err := parseMarkdown(parser, c.Orig)
	if err != nil {
		return err
	}
	var wrongLinkError error
	doc.Walk(func(node *bf.Node, _ bool) bf.WalkStatus {
		if len(node.Destination) != 0 &&
			(!strings.HasPrefix(string(node.Destination), "http://") && !strings.HasPrefix(string(node.Destination), "https://") && !strings.HasPrefix(string(node.Destination), "mailto:")) {
			wrongLinkError = fmt.Errorf("links should start with mailto:, http:// or https://")
			return bf.Terminate
		}
		return bf.GoToNext
	})
	return wrongLinkError
}

// parseMarkdown parses the comment text, the parser panics on some text, which is refused then
func parseMarkdown(parser *bf.Markdown, txt string) (doc *bf.Node, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("[WARN] can't parse comment text, %v", rec)
			err = fmt.Errorf("can't parse comment text")
		}
	}()
	return parser.Parse([]byte(txt)), nil
}

// IsAdmin checks if usesID in the list of admins
func (s *DataStore) IsAdmin(siteID, userID string) bool {
	admins, err := s.AdminStore.Admins(siteID)
	if err != nil {
		log.Printf("[WARN] can't get admins for %s, %v", siteID, err)
		return false
	}
	return slices.Contains(admins, userID)
}

// IsReadOnly checks if post read-only
func (s *DataStore) IsReadOnly(locator store.Locator) bool {
	req := engine.FlagRequest{Locator: locator, Flag: engine.ReadOnly}
	ro, err := s.Engine.Flag(req)
	return err == nil && ro
}

// SetReadOnly set/reset read-only flag
func (s *DataStore) SetReadOnly(locator store.Locator, status bool) error {
	roStatus := engine.FlagFalse
	if status {
		roStatus = engine.FlagTrue
	}
	req := engine.FlagRequest{Locator: locator, Flag: engine.ReadOnly, Update: roStatus}
	_, err := s.Engine.Flag(req)
	return err
}

// IsVerified checks if user verified
func (s *DataStore) IsVerified(siteID, userID string) bool {
	req := engine.FlagRequest{Locator: store.Locator{SiteID: siteID}, UserID: userID, Flag: engine.Verified}
	ro, err := s.Engine.Flag(req)
	return err == nil && ro
}

// SetVerified set/reset verified status for user
func (s *DataStore) SetVerified(siteID, userID string, status bool) error {
	roStatus := engine.FlagFalse
	if status {
		roStatus = engine.FlagTrue
	}
	req := engine.FlagRequest{Locator: store.Locator{SiteID: siteID}, UserID: userID, Flag: engine.Verified, Update: roStatus}
	_, err := s.Engine.Flag(req)
	return err
}

// IsBlocked checks if user blocked
func (s *DataStore) IsBlocked(siteID, userID string) bool {
	req := engine.FlagRequest{Locator: store.Locator{SiteID: siteID}, UserID: userID, Flag: engine.Blocked}
	ro, err := s.Engine.Flag(req)
	return err == nil && ro
}

// SetBlock set/reset verified status for user
func (s *DataStore) SetBlock(siteID, userID string, status bool, ttl time.Duration) error {
	roStatus := engine.FlagFalse
	if status {
		roStatus = engine.FlagTrue
	}
	req := engine.FlagRequest{Locator: store.Locator{SiteID: siteID}, UserID: userID,
		Flag: engine.Blocked, Update: roStatus, TTL: ttl}
	_, err := s.Engine.Flag(req)
	return err
}

// BlockedUsers returns list with all blocked users for given siteID
func (s *DataStore) BlockedUsers(siteID string) (res []store.BlockedUser, err error) {
	blocked, e := s.Engine.ListFlags(engine.FlagRequest{Locator: store.Locator{SiteID: siteID}, Flag: engine.Blocked})
	if e != nil {
		return nil, fmt.Errorf("can't get list of blocked users for %s: %w", siteID, e)
	}
	for _, v := range blocked {
		res = append(res, v.(store.BlockedUser))
	}
	return res, nil
}

// Info get post info
func (s *DataStore) Info(locator store.Locator, readonlyAge int) (store.PostInfo, error) {
	req := engine.InfoRequest{Locator: locator, ReadOnlyAge: readonlyAge}
	res, err := s.Engine.Info(req)
	if err != nil {
		return store.PostInfo{}, err
	}
	if len(res) == 0 {
		return store.PostInfo{}, fmt.Errorf("post %+v not found", locator)
	}
	// URL request
	if locator.URL != "" {
		return res[0], nil
	}
	// site-wide request which returned multiple store.PostInfo, so that URL and ReadOnly flags don't make sense
	var info store.PostInfo
	for _, i := range res {
		info.Count += i.Count
		if info.FirstTS.IsZero() || i.FirstTS.Before(info.FirstTS) {
			info.FirstTS = i.FirstTS
		}
		if info.LastTS.IsZero() || i.LastTS.After(info.LastTS) {
			info.LastTS = i.LastTS
		}
	}
	return info, nil

}

// Delete comment by id. The pictures its author uploaded go with it once the engine has deleted it, unless another
// comment still shows them, see deleteRemovedPictures
func (s *DataStore) Delete(locator store.Locator, commentID string, mode store.DeleteMode) error {
	if e := s.AdminStore.OnEvent(locator.SiteID, admin.EvDelete); e != nil {
		log.Printf("[WARN] failed to send delete event, %s", e)
	}
	// get the comment to learn its author and pictures
	comment, err := s.Engine.Get(engine.GetRequest{Locator: locator, CommentID: commentID})
	if err != nil {
		return err
	}
	if err := s.Engine.Delete(engine.DeleteRequest{Locator: locator, CommentID: commentID, DeleteMode: mode}); err != nil {
		return err
	}
	s.deleteRemovedPictures(locator, commentID, comment.User.ID, comment.Text, "")
	return nil
}

// DeleteUser removes all comments from user, along with the pictures used in them
func (s *DataStore) DeleteUser(siteID, userID string, mode store.DeleteMode) error {
	// collect the pictures before the comments are blanked, delete them only once the comments are gone
	imgIDs := s.userPictures(siteID, userID)
	req := engine.DeleteRequest{Locator: store.Locator{SiteID: siteID}, UserID: userID, DeleteMode: mode}
	if err := s.Engine.Delete(req); err != nil {
		return err
	}
	for _, id := range imgIDs {
		if err := s.ImageService.Delete(id); err != nil {
			log.Printf("[WARN] failed to delete image %s on user %s deletion, %v", id, userID, err)
		}
	}
	return nil
}

// ownPictures returns the IDs of the pictures in commentHTML that userID uploaded, saved as userID/<guid>.
// A comment can also show another user's upload or a cached external image, which other comments may
// use on any page, so deleting the comment or its author must leave those alone
func (s *DataStore) ownPictures(userID, commentHTML string) []string {
	res := []string{}
	for _, id := range s.ImageService.ExtractPictures(commentHTML) {
		if strings.HasPrefix(id, userID+"/") {
			res = append(res, id)
		}
	}
	return res
}

// deleteRemovedPictures deletes the pictures authorID uploaded that oldHTML shows and newHTML does not, once the
// edit or delete of comment commentID is stored. A picture another comment on the same page shows stays, and
// so does one any other comment of the author shows, on any page: the address of an upload can be pasted into
// another comment. Other users' uploads and cached external images stay too. A picture still
// waiting in staging is deleted there, and the delayed commit, which reads the comment's stored text, no longer
// looks for it
func (s *DataStore) deleteRemovedPictures(locator store.Locator, commentID, authorID, oldHTML, newHTML string) {
	if s.ImageService == nil {
		return
	}
	keep := s.ImageService.ExtractPictures(newHTML)
	removed := []string{}
	for _, id := range s.ownPictures(authorID, oldHTML) {
		if !slices.Contains(keep, id) && !slices.Contains(removed, id) {
			removed = append(removed, id)
		}
	}
	if len(removed) == 0 {
		return
	}

	comments, err := s.Engine.Find(engine.FindRequest{Locator: locator})
	if err != nil {
		log.Printf("[WARN] can't get comments of %s for image cleanup of comment %s, %v", locator.URL, commentID, err)
		return
	}
	// the comment's stored text counts as well: another edit of it may have been stored in the meantime
	for _, c := range comments {
		keep = append(keep, s.ImageService.ExtractPictures(c.Text)...)
	}
	if !slices.ContainsFunc(removed, func(id string) bool { return !slices.Contains(keep, id) }) {
		return
	}

	err = s.eachUserComment(locator.SiteID, authorID, func(c store.Comment) {
		keep = append(keep, s.ImageService.ExtractPictures(c.Text)...)
	})
	if err != nil {
		log.Printf("[WARN] can't get comments of user %s for image cleanup of comment %s, %v", authorID, commentID, err)
		return
	}
	for _, id := range removed {
		if slices.Contains(keep, id) {
			continue
		}
		if err := s.ImageService.Delete(id); err != nil {
			log.Printf("[WARN] failed to delete image %s removed from comment %s, %v", id, commentID, err)
		}
	}
}

// eachUserComment calls fn for every comment of the user on the site, reading them page by page and keeping
// no page once it is passed on. It stops on a page with nothing new, as an engine may ignore Skip. A user with
// no comments is not an error
func (s *DataStore) eachUserComment(siteID, userID string, fn func(c store.Comment)) error {
	seen := map[string]bool{}
	for skip := 0; ; {
		comments, err := s.Engine.Find(engine.FindRequest{Locator: store.Locator{SiteID: siteID}, UserID: userID, Skip: skip})
		if errors.Is(err, engine.ErrNoUserComments) {
			return nil
		}
		if err != nil {
			return err
		}
		added := 0
		for _, c := range comments {
			key := c.Locator.URL + "|" + c.ID
			if seen[key] {
				continue
			}
			seen[key] = true
			added++
			fn(c)
		}
		if added == 0 {
			return nil
		}
		skip += len(comments)
	}
}

// userPictures returns IDs of the pictures the user uploaded and used in their comments, leaving out
// those other users' comments to the same pages also use, the same check Delete makes for one comment
func (s *DataStore) userPictures(siteID, userID string) []string {
	userImgIDs := []string{}
	pages := map[string]bool{}
	err := s.eachUserComment(siteID, userID, func(c store.Comment) {
		if ids := s.ownPictures(userID, c.Text); len(ids) > 0 {
			userImgIDs = append(userImgIDs, ids...)
			pages[c.Locator.URL] = true
		}
	})
	if err != nil {
		log.Printf("[WARN] can't get comments of user %s for image cleanup, %v", userID, err)
		return nil
	}
	if len(userImgIDs) == 0 {
		return nil
	}

	skipIDs := map[string]bool{} // pictures other users use, then also the ones already listed
	for url := range pages {
		comments, err := s.Engine.Find(engine.FindRequest{Locator: store.Locator{SiteID: siteID, URL: url}})
		if err != nil {
			log.Printf("[WARN] can't get comments of %s for image cleanup of user %s, %v", url, userID, err)
			return nil
		}
		for _, c := range comments {
			if c.User.ID != userID {
				for _, id := range s.ImageService.ExtractPictures(c.Text) {
					skipIDs[id] = true
				}
			}
		}
	}

	res := []string{}
	for _, id := range userImgIDs {
		if !skipIDs[id] {
			skipIDs[id] = true
			res = append(res, id)
		}
	}
	return res
}

// List of commented posts
func (s *DataStore) List(siteID string, limit, skip int) ([]store.PostInfo, error) {
	req := engine.InfoRequest{Locator: store.Locator{SiteID: siteID}, Limit: limit, Skip: skip}
	return s.Engine.Info(req)
}

// Count gets number of comments for the post
func (s *DataStore) Count(locator store.Locator) (int, error) {
	req := engine.FindRequest{Locator: locator}
	return s.Engine.Count(req)
}

// Metas returns metadata for users and posts
func (s *DataStore) Metas(siteID string) (umetas []UserMetaData, pmetas []PostMetaData, err error) {
	umetas = []UserMetaData{}
	pmetas = []PostMetaData{}

	// set posts meta
	posts, err := s.Engine.Info(engine.InfoRequest{Locator: store.Locator{SiteID: siteID}})
	if err != nil {
		return nil, nil, fmt.Errorf("can't get list of posts for %s: %w", siteID, err)
	}

	for _, p := range posts {
		if s.IsReadOnly(store.Locator{SiteID: siteID, URL: p.URL}) {
			pmetas = append(pmetas, PostMetaData{URL: p.URL, ReadOnly: true})
		}
	}

	// set users meta, key is userID
	m := map[string]UserMetaData{}

	// process blocked users
	blocked, err := s.BlockedUsers(siteID)
	if err != nil {
		return nil, nil, fmt.Errorf("can't get list of blocked users for %s: %w", siteID, err)
	}
	for _, b := range blocked {
		val, ok := m[b.ID]
		if !ok {
			val = UserMetaData{ID: b.ID}
		}
		val.Blocked.Status = true
		val.Blocked.Until = b.Until
		m[b.ID] = val
	}

	// process verified users
	verified, err := s.Engine.ListFlags(engine.FlagRequest{Locator: store.Locator{SiteID: siteID}, Flag: engine.Verified})
	if err != nil {
		return nil, nil, fmt.Errorf("can't get list of verified users for %s: %w", siteID, err)
	}
	for _, vi := range verified {
		v := vi.(string)
		val, ok := m[v]
		if !ok {
			val = UserMetaData{ID: v}
		}
		val.Verified = true
		m[v] = val
	}

	// process users details
	usersDetails, err := s.Engine.UserDetail(engine.UserDetailRequest{Locator: store.Locator{SiteID: siteID}, Detail: engine.AllUserDetails})
	if err != nil {
		return nil, nil, fmt.Errorf("can't get user details for %s: %w", siteID, err)
	}
	for _, entry := range usersDetails {
		val, ok := m[entry.UserID]
		if !ok {
			val = UserMetaData{ID: entry.UserID}
		}
		val.Details = entry
		m[entry.UserID] = val
	}

	for _, u := range m {
		umetas = append(umetas, u)
	}
	sort.Slice(umetas, func(i, j int) bool { return umetas[i].ID < umetas[j].ID })

	return umetas, pmetas, nil
}

// SetMetas saves metadata for users and posts
func (s *DataStore) SetMetas(siteID string, umetas []UserMetaData, pmetas []PostMetaData) (err error) {
	var errs []error

	// save posts metas
	for _, pm := range pmetas {
		if pm.ReadOnly {
			errs = append(errs, s.SetReadOnly(store.Locator{SiteID: siteID, URL: pm.URL}, true))
		}
	}

	// save users metas
	for _, um := range umetas {
		// a block that ran out before the restore is skipped: a negative TTL would make it permanent.
		// a block without an end time stays permanent
		if um.Blocked.Status && (um.Blocked.Until.IsZero() || um.Blocked.Until.After(time.Now())) {
			errs = append(errs, s.SetBlock(siteID, um.ID, true, time.Until(um.Blocked.Until)))
		}
		if um.Verified {
			errs = append(errs, s.SetVerified(siteID, um.ID, true))
		}
		// this code doesn't delete user details in case they are not set in import but present in DB already
		if um.Details.Email != "" {
			req := engine.UserDetailRequest{Locator: store.Locator{SiteID: siteID}, UserID: um.ID, Detail: engine.UserEmail, Update: um.Details.Email}
			_, err := s.Engine.UserDetail(req)
			errs = append(errs, err)
		}
		if um.Details.Telegram != "" {
			req := engine.UserDetailRequest{Locator: store.Locator{SiteID: siteID}, UserID: um.ID, Detail: engine.UserTelegram, Update: um.Details.Telegram}
			_, err := s.Engine.UserDetail(req)
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// User gets comment for given userID on siteID
func (s *DataStore) User(siteID, userID string, limit, skip int, user store.User) ([]store.Comment, error) {
	req := engine.FindRequest{Locator: store.Locator{SiteID: siteID}, UserID: userID,
		Limit: limit, Skip: skip, Sort: "-time"}
	comments, err := s.Engine.Find(req)
	if err != nil {
		return comments, err
	}
	return s.alterComments(comments, user), nil
}

// UserCount is comments count by user
func (s *DataStore) UserCount(siteID, userID string) (int, error) {
	req := engine.FindRequest{Locator: store.Locator{SiteID: siteID}, UserID: userID}
	return s.Engine.Count(req)
}

// Last gets last comments for site, cross-post. Limited by count and optional since ts
func (s *DataStore) Last(siteID string, limit int, since time.Time, user store.User) ([]store.Comment, error) {
	req := engine.FindRequest{Locator: store.Locator{SiteID: siteID}, Limit: limit, Since: since, Sort: "-time"}
	comments, err := s.Engine.Find(req)
	if err != nil {
		return comments, err
	}
	return s.alterComments(comments, user), nil
}

// Close store service
func (s *DataStore) Close() error {
	var errs []error
	if s.TitleExtractor != nil {
		errs = append(errs, s.TitleExtractor.Close())
	}
	errs = append(errs, s.Engine.Close())
	return errors.Join(errs...)
}

func (s *DataStore) upsAndDowns(c store.Comment) (ups, downs int) {
	for _, v := range c.Votes {
		if v {
			ups++
			continue
		}
		downs++
	}
	return ups, downs
}

// getScopedLocks pull lock from the map if found or create a new one
func (s *DataStore) getScopedLocks(id string) (lock sync.Locker) {
	s.scopedLocks.Do(func() { s.scopedLocks.locks = map[string]sync.Locker{} })

	s.scopedLocks.Lock()
	lock, ok := s.scopedLocks.locks[id]
	if !ok {
		lock = &sync.Mutex{}
		s.scopedLocks.locks[id] = lock
	}
	s.scopedLocks.Unlock()

	return lock
}

func (s *DataStore) alterComments(cc []store.Comment, user store.User) (res []store.Comment) {
	res = make([]store.Comment, len(cc))
	flags := s.newUserFlagCache()
	for i, c := range cc {
		res[i] = s.alterCommentCached(c, user, flags)
	}
	return res
}

func (s *DataStore) alterComment(c store.Comment, user store.User) (res store.Comment) {
	return s.alterCommentCached(c, user, s.newUserFlagCache())
}

// alterCommentCached is alterComment sharing a userFlagCache so that block/verified
// lookups for a user repeated across a listing hit the engine only once.
func (s *DataStore) alterCommentCached(c store.Comment, user store.User, flags *userFlagCache) (res store.Comment) {
	// mark user blocked
	if flags.blocked(c.Locator.SiteID, c.User.ID) {
		c.User.Blocked = true
	}

	// set verified status retroactively
	if !c.User.Blocked {
		c.User.Verified = flags.verified(c.Locator.SiteID, c.User.ID)
	}

	// hide info from non-admins
	if !flags.admin(c.Locator.SiteID, user) {
		c.User.IP = ""
	}

	c = s.prepVotes(c, user)
	c.Locator.URL = c.SanitizeAsURL(c.Locator.URL) // urls prior to #927
	c.PostTitle = c.SanitizeText(c.PostTitle)
	c.SanitizeEditSummary() // summaries stored before edits cleaned them
	c.MarkLinksUGC()        // links stored before they were marked ugc
	return c
}

// userFlagCache memoises engine block/verified flag lookups by site and user within
// a single listing, avoiding two engine.Flag calls per comment for repeated users.
// It also keeps the viewer's admin status per site, see admin.
type userFlagCache struct {
	s      *DataStore
	flags  map[engine.FlagRequest]bool // engine flag by flag, site and user
	adminM map[string]bool             // viewer's admin status by site, for sites other than user.SiteID
}

func (s *DataStore) newUserFlagCache() *userFlagCache {
	return &userFlagCache{s: s, flags: map[engine.FlagRequest]bool{}, adminM: map[string]bool{}}
}

// admin reports whether the viewer counts as an admin for a comment on siteID. The admin flag of a
// signed-in user was set for the site the token was issued for, user.SiteID, while the open routes
// accept that token for any site the instance serves. On another site the flag holds only if the
// admin store lists the user for that site as well, which with the shared store is always the case.
// A user with no site is the basic-auth admin, who is admin of every site
func (f *userFlagCache) admin(siteID string, user store.User) bool {
	if !user.Admin {
		return false
	}
	if user.SiteID == "" || user.SiteID == siteID {
		return true
	}
	if v, ok := f.adminM[siteID]; ok {
		return v
	}
	v := f.s.IsAdmin(siteID, user.ID)
	f.adminM[siteID] = v
	return v
}

func (f *userFlagCache) blocked(siteID, userID string) bool {
	return f.flag(engine.Blocked, siteID, userID)
}

func (f *userFlagCache) verified(siteID, userID string) bool {
	return f.flag(engine.Verified, siteID, userID)
}

func (f *userFlagCache) flag(flag engine.Flag, siteID, userID string) bool {
	req := engine.FlagRequest{Flag: flag, Locator: store.Locator{SiteID: siteID}, UserID: userID}
	if v, ok := f.flags[req]; ok {
		return v
	}
	v, err := f.s.Engine.Flag(req)
	if err != nil {
		// don't cache on error so a repeated user is retried, matching the
		// pre-refactor per-comment behavior; treat this comment as not flagged
		return false
	}
	f.flags[req] = v
	return v
}

// prepare vote info for client view
func (s *DataStore) prepVotes(c store.Comment, user store.User) store.Comment {
	c.Vote = 0 // default is "none" (not voted)

	if v, ok := c.Votes[user.ID]; ok {
		if v {
			c.Vote = 1
		} else {
			c.Vote = -1
		}
	}

	c.Votes = nil    // hide voters list
	c.VotedIPs = nil // hide voted ips (hashes)
	return c
}

// get secret for given siteID
// Note: siteID ignored for the default admin.Static store
func (s *DataStore) getSecret(siteID string) (secret string, err error) {
	if secret, err = s.AdminStore.Key(siteID); err != nil {
		return "", fmt.Errorf("can't get secret for site %s: %w", siteID, err)
	}

	ok, err := s.AdminStore.Enabled(siteID)
	if err != nil {
		return "", fmt.Errorf("can't check secret enabled for site %s: %w", siteID, err)
	}
	if !ok {
		return "", fmt.Errorf("site %s disabled", siteID)
	}
	return secret, nil
}
