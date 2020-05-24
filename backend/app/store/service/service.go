// Package service wraps engine interfaces with common logic unrelated to any particular engine implementation.
// All consumers should be using service.DataStore and not the naked engine!
package service

import (
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-pkgz/lcw"
	log "github.com/go-pkgz/lgr"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

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

	// granular locks
	scopedLocks struct {
		sync.Mutex
		sync.Once
		locks map[string]sync.Locker
	}

	repliesCache struct {
		lcw.LoadingCache
		once sync.Once
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
	Details  engine.UserDetailEntry `json:"details,omitempty"`
}

// PostMetaData keeps info about post flags
type PostMetaData struct {
	URL      string `json:"url"`
	ReadOnly bool   `json:"read_only"`
}

const defaultCommentMaxSize = 2000
const maxLastCommentsReply = 5000

// UnlimitedVotes doesn't restrict MaxVotes
const UnlimitedVotes = -1

var nonAdminUser = store.User{}

// ErrRestrictedWordsFound returned in case comment text contains restricted words
var ErrRestrictedWordsFound = errors.New("comment contains restricted words")

// Create prepares comment and forward to Interface.Create
func (s *DataStore) Create(comment store.Comment) (commentID string, err error) {

	if comment, err = s.prepareNewComment(comment); err != nil {
		return "", errors.Wrap(err, "failed to prepare comment")
	}

	if s.RestrictedWordsMatcher != nil && s.RestrictedWordsMatcher.Match(comment.Locator.SiteID, comment.Text) {
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

	s.submitImages(comment.Locator, comment.ID)
	if e := s.AdminStore.OnEvent(comment.Locator.SiteID, admin.EvCreate); e != nil {
		log.Printf("[WARN] failed to send create event, %s", e)
	}

	return s.Engine.Create(comment)
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
	// set votes controversy for comments added prior to #274
	for i, c := range comments {
		if c.Controversy == 0 && len(c.Votes) > 0 {
			c.Controversy = s.controversy(s.upsAndDowns(c))
			if !changedSort && strings.Contains(sortMethod, "controversy") { // trigger sort change
				changedSort = true
			}
		}
		comments[i] = s.alterComment(c, user)
	}

	// resort commits if altered
	if changedSort {
		comments = engine.SortComments(comments, sortMethod)
	}

	return comments, nil
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
	res, err := s.Engine.UserDetail(engine.UserDetailRequest{
		Detail:  engine.UserEmail,
		Locator: store.Locator{SiteID: siteID},
		UserID:  userID,
	})
	if err != nil {
		return "", err
	}
	if len(res) == 1 {
		return res[0].Email, nil
	}
	return "", nil
}

// SetUserEmail sets user email
func (s *DataStore) SetUserEmail(siteID, userID, value string) (string, error) {
	res, err := s.Engine.UserDetail(engine.UserDetailRequest{
		Detail:  engine.UserEmail,
		Locator: store.Locator{SiteID: siteID},
		UserID:  userID,
		Update:  value,
	})
	if err != nil {
		return "", err
	}
	if len(res) == 1 {
		return res[0].Email, nil
	}
	return "", nil
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
	result := new(multierror.Error)
	for _, site := range sites {
		locator := store.Locator{SiteID: site}
		comments, err := s.FindSince(locator, "time", store.User{}, ts)
		result = multierror.Append(result, errors.Wrapf(err, "problem finding comments for site %s", site))
		for _, c := range comments {
			s.submitImages(c.Locator, c.ID)
		}
	}
	return result.ErrorOrNil()
}

// submitImages initiated delayed commit of all images from the comment uploaded to remark42
func (s *DataStore) submitImages(locator store.Locator, commentID string) {

	s.ImageService.Submit(func() []string { // get all ids from comment's text
		// this can be called after last edit, we have to retrieve fresh comment
		cc, err := s.Engine.Get(engine.GetRequest{Locator: locator, CommentID: commentID})
		if err != nil {
			log.Printf("[WARN] can't get comment's %s text for image extraction, %v", commentID, err)
			return nil
		}
		imgIds, err := s.ImageService.ExtractPictures(cc.Text)
		if err != nil {
			log.Printf("[WARN] can't get extract pictures from %s, %v", commentID, err)
			return nil
		}
		if len(imgIds) > 0 {
			log.Printf("[DEBUG] image ids extracted from %s - %+v", commentID, imgIds)
		}
		return imgIds
	})
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
		return store.Comment{}, errors.Wrapf(err, "can't get secret for site %s", comment.Locator.SiteID)
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
		return comment, errors.Errorf("user %s can not vote for his own comment %s", req.UserID, req.CommentID)
	}

	if comment.Votes == nil {
		comment.Votes = make(map[string]bool)
	}

	v, voted := comment.Votes[req.UserID]
	if voted && v == req.Val {
		return comment, errors.Errorf("user %s already voted for %s", req.UserID, req.CommentID)
	}

	secret, err := s.getSecret(comment.Locator.SiteID)
	if err != nil {
		return store.Comment{}, errors.Wrapf(err, "can't get secret for site %s", comment.Locator.SiteID)
	}
	userIPHash := store.HashValue(req.UserIP, secret)
	if s.isSameIPVote(req, userIPHash, comment) {
		return comment, errors.Errorf("the same ip %s already voted for %s", userIPHash, req.CommentID)
	}

	maxVotes := s.MaxVotes // 0 value allowed and treated as "no comments allowed"
	if s.MaxVotes < 0 {    // any negative value reset max votes to unlimited
		maxVotes = UnlimitedVotes
	}

	if maxVotes >= 0 && len(comment.Votes) >= maxVotes {
		return comment, errors.Errorf("maximum number of votes exceeded for comment %s", req.CommentID)
	}

	if s.PositiveScore && comment.Score <= 0 && !req.Val {
		return comment, errors.Errorf("minimal score reached for comment %s", req.CommentID)
	}

	// reset vote if user changed to opposite
	if voted && v != req.Val {
		delete(comment.Votes, req.UserID)
	}

	// add to voted map if first vote
	if !voted {
		comment.Votes[req.UserID] = req.Val
	}

	// add ip hash to voted ip map
	if comment.VotedIPs == nil {
		comment.VotedIPs = map[string]store.VotedIPInfo{}
	}

	comment.VotedIPs[userIPHash] = store.VotedIPInfo{Timestamp: time.Now(), Value: req.Val}

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
		if s.RestrictSameIPVotes.Duration == 0 || v.Timestamp.Add(s.RestrictSameIPVotes.Duration).After(time.Now()) {
			return true
		}
	}
	return false
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
}

// EditComment to edit text and update Edit info
func (s *DataStore) EditComment(locator store.Locator, commentID string, req EditRequest) (comment store.Comment, err error) {
	comment, err = s.Engine.Get(engine.GetRequest{Locator: locator, CommentID: commentID})
	if err != nil {
		return comment, err
	}

	// edit allowed in editDuration window only
	if s.EditDuration > 0 && time.Now().After(comment.Timestamp.Add(s.EditDuration)) {
		return comment, errors.Errorf("too late to edit %s", commentID)
	}

	if s.HasReplies(comment) {
		return comment, errors.Errorf("parent comment with reply can't be edited, %s", commentID)
	}

	if req.Delete { // delete request
		if e := s.AdminStore.OnEvent(comment.Locator.SiteID, admin.EvDelete); e != nil {
			log.Printf("[WARN] failed to send delete event, %s", e)
		}
		comment.Deleted = true
		delReq := engine.DeleteRequest{Locator: locator, CommentID: commentID, DeleteMode: store.SoftDelete}
		return comment, s.Engine.Delete(delReq)
	}

	if s.RestrictedWordsMatcher != nil && s.RestrictedWordsMatcher.Match(comment.Locator.SiteID, req.Text) {
		return comment, ErrRestrictedWordsFound
	}

	comment.Text = req.Text
	comment.Orig = req.Orig
	comment.Edit = &store.Edit{
		Timestamp: time.Now(),
		Summary:   req.Summary,
	}
	comment.Locator = locator
	comment.Sanitize()

	if e := s.AdminStore.OnEvent(comment.Locator.SiteID, admin.EvUpdate); e != nil {
		log.Printf("[WARN] failed to send update event, %s", e)
	}

	err = s.Engine.Update(comment)
	return comment, err
}

// HasReplies checks if there is any reply to the comments
// Loads last maxLastCommentsReply comments and compare parent id to the comment's id
// Comments with replies cached for 5 minutes
func (s *DataStore) HasReplies(comment store.Comment) bool {

	s.repliesCache.once.Do(func() {
		// default expiration time of 5 minutes and cleanup time of 2.5 minutes
		s.repliesCache.LoadingCache, _ = lcw.NewExpirableCache(lcw.TTL(5 * time.Minute))
	})

	if _, found := s.repliesCache.Peek(comment.ID); found {
		return true
	}

	req := engine.FindRequest{Locator: store.Locator{SiteID: comment.Locator.SiteID}, Limit: maxLastCommentsReply}
	comments, err := s.Engine.Find(req)
	if err != nil {
		log.Printf("[WARN] can't get last comments for reply check, %v", err)
		return false
	}

	for _, c := range comments {
		if c.ParentID != "" && !c.Deleted {
			if c.ParentID == comment.ID {
				// When this code is reached, key "comment.ID" is not in cache.
				// Calling cache.Get on it will put it in cache with 5 minutes TTL.
				// We call it with empty struct as value as we care about keys and not values.
				_, _ = s.repliesCache.Get(comment.ID, func() (lcw.Value, error) { return struct{}{}, nil })
				return true
			}
		}
	}
	return false
}

// UserReplies returns list of all comments replied to given user
func (s *DataStore) UserReplies(siteID, userID string, limit int, duration time.Duration) ([]store.Comment, string, error) {

	comments, e := s.Last(siteID, maxLastCommentsReply, time.Time{}, nonAdminUser)
	if e != nil {
		return nil, "", errors.Wrap(e, "can't get last comments")
	}
	replies := []store.Comment{}

	// get a comment for given userID in order to retrieve name
	userName := ""
	if cc, err := s.User(siteID, userID, 1, 0, nonAdminUser); err == nil && len(cc) > 0 {
		userName = cc[0].User.Name
	}

	// collect replies
	for _, c := range comments {

		if len(replies) > limit || time.Since(c.Timestamp) > duration {
			break
		}

		if c.ParentID != "" && !c.Deleted && c.User.ID != userID { // not interested in replies to yourself
			var pc store.Comment
			if pc, e = s.Get(c.Locator, c.ParentID, nonAdminUser); e != nil {
				return nil, "", errors.Wrap(e, "can't get parent comment")
			}
			if pc.User.ID == userID {
				replies = append(replies, c)
			}
		}
	}

	return replies, userName, nil
}

// SetTitle puts title from the locator.URL page and overwrites any existing title
func (s *DataStore) SetTitle(locator store.Locator, commentID string) (comment store.Comment, err error) {
	if s.TitleExtractor == nil {
		return comment, errors.New("no title extractor")
	}

	comment, err = s.Engine.Get(engine.GetRequest{Locator: locator, CommentID: commentID})
	if err != nil {
		return comment, err
	}

	// set title, overwrite the current one
	title, e := s.TitleExtractor.Get(comment.Locator.URL)
	if e != nil {
		return comment, err
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

// ValidateComment checks if comment size below max and user fields set
func (s *DataStore) ValidateComment(c *store.Comment) error {
	maxSize := s.MaxCommentSize
	if s.MaxCommentSize <= 0 {
		maxSize = defaultCommentMaxSize
	}
	if c.Orig == "" {
		return errors.New("empty comment text")
	}
	if len([]rune(c.Orig)) > maxSize {
		return errors.Errorf("comment text exceeded max allowed size %d (%d)", maxSize, len([]rune(c.Orig)))
	}
	if c.User.ID == "" || c.User.Name == "" {
		return errors.Errorf("empty user info")
	}
	return nil
}

// IsAdmin checks if usesID in the list of admins
func (s *DataStore) IsAdmin(siteID, userID string) bool {
	admins, err := s.AdminStore.Admins(siteID)
	if err != nil {
		log.Printf("[WARN] can't get admins for %s, %v", siteID, err)
		return false
	}
	for _, a := range admins {
		if a == userID {
			return true
		}
	}
	return false
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
		return nil, errors.Wrapf(err, "can't get list of blocked users for %s", siteID)
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
		return store.PostInfo{}, errors.Errorf("post %+v not found", locator)
	}
	return res[0], nil
}

// Delete comment by id
func (s *DataStore) Delete(locator store.Locator, commentID string, mode store.DeleteMode) error {
	if e := s.AdminStore.OnEvent(locator.SiteID, admin.EvDelete); e != nil {
		log.Printf("[WARN] failed to send delete event, %s", e)
	}
	req := engine.DeleteRequest{Locator: locator, CommentID: commentID, DeleteMode: mode}
	return s.Engine.Delete(req)
}

// DeleteUser removes all comments from user
func (s *DataStore) DeleteUser(siteID, userID string, mode store.DeleteMode) error {
	req := engine.DeleteRequest{Locator: store.Locator{SiteID: siteID}, UserID: userID, DeleteMode: mode}
	return s.Engine.Delete(req)
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
		return nil, nil, errors.Wrapf(err, "can't get list of posts for %s", siteID)
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
		return nil, nil, errors.Wrapf(err, "can't get list of blocked users for %s", siteID)
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
		return nil, nil, errors.Wrapf(err, "can't get list of verified users for %s", siteID)
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
		return nil, nil, errors.Wrapf(err, "can't get user details for %s", siteID)
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
	errs := new(multierror.Error)

	// save posts metas
	for _, pm := range pmetas {
		if pm.ReadOnly {
			errs = multierror.Append(errs, s.SetReadOnly(store.Locator{SiteID: siteID, URL: pm.URL}, true))
		}
	}

	// save users metas
	for _, um := range umetas {
		if um.Blocked.Status {
			errs = multierror.Append(errs, s.SetBlock(siteID, um.ID, true, time.Until(um.Blocked.Until)))
		}
		if um.Verified {
			errs = multierror.Append(errs, s.SetVerified(siteID, um.ID, true))
		}
		// this code doesn't delete user details in case they are not set in import but present in DB already
		if um.Details.Email != "" {
			req := engine.UserDetailRequest{Locator: store.Locator{SiteID: siteID}, UserID: um.ID, Detail: engine.UserEmail, Update: um.Details.Email}
			_, err := s.Engine.UserDetail(req)
			errs = multierror.Append(errs, err)
		}
	}

	return errs.ErrorOrNil()
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
	errs := new(multierror.Error)
	if s.repliesCache.LoadingCache != nil {
		errs = multierror.Append(errs, s.repliesCache.LoadingCache.Close())
	}
	if s.TitleExtractor != nil {
		errs = multierror.Append(errs, s.TitleExtractor.Close())
	}
	errs = multierror.Append(errs, s.Engine.Close())
	return errs.ErrorOrNil()
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
	for i, c := range cc {
		res[i] = s.alterComment(c, user)
	}
	return res
}

func (s *DataStore) alterComment(c store.Comment, user store.User) (res store.Comment) {

	blocReq := engine.FlagRequest{Flag: engine.Blocked, Locator: store.Locator{SiteID: c.Locator.SiteID}, UserID: c.User.ID}
	blocked, bErr := s.Engine.Flag(blocReq)

	// mark user blocked
	if bErr == nil && blocked {
		c.User.Blocked = blocked
	}

	// set verified status retroactively
	if !c.User.Blocked {
		verifReq := engine.FlagRequest{Flag: engine.Verified, Locator: store.Locator{SiteID: c.Locator.SiteID}, UserID: c.User.ID}
		c.User.Verified, _ = s.Engine.Flag(verifReq)
	}

	// hide info from non-admins
	if !user.Admin {
		c.User.IP = ""
	}

	c = s.prepVotes(c, user)
	return c
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

	c.Votes = nil // hide voters list
	return c
}

// get secret for given siteID
// Note: secret shared across sites, but some sites can be disabled.
func (s *DataStore) getSecret(siteID string) (secret string, err error) {

	if secret, err = s.AdminStore.Key(); err != nil {
		return "", errors.Wrapf(err, "can't get secret for site %s", siteID)
	}

	ok, err := s.AdminStore.Enabled(siteID)
	if err != nil {
		return "", errors.Wrapf(err, "can't check secret enabled for site %s", siteID)
	}
	if !ok {
		return "", errors.Errorf("site %s disabled", siteID)
	}
	return secret, nil
}
