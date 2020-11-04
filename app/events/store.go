package events

import (
	"fmt"
	log "github.com/golang/glog"
	"github.com/muety/kitsquid/app/config"
	"github.com/muety/kitsquid/app/util"
	"github.com/patrickmn/go-cache"
	"github.com/timshannon/bolthold"
	"go.etcd.io/bbolt"
	"math"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	db             *bolthold.Store
	cfg            *config.Config
	eventsCache    *cache.Cache
	bookmarksCache *cache.Cache
	miscCache      *cache.Cache
	reviewsCache   *cache.Cache
	commentsCache  *cache.Cache
)

func initStore(store *bolthold.Store) {
	cfg = config.Get()
	db = store

	eventsCache = cache.New(cfg.CacheDuration("events", 30*time.Minute), cfg.CacheDuration("events", 30*time.Minute)*2)
	miscCache = cache.New(cfg.CacheDuration("misc", 30*time.Minute), cfg.CacheDuration("misc", 30*time.Minute)*2)
	bookmarksCache = cache.New(cfg.CacheDuration("bookmarks", 30*time.Minute), cfg.CacheDuration("bookmarks", 30*time.Minute)*2)
	reviewsCache = cache.New(cfg.CacheDuration("reviews", 30*time.Minute), cfg.CacheDuration("reviews", 30*time.Minute)*2)
	commentsCache = cache.New(cfg.CacheDuration("comments", 30*time.Minute), cfg.CacheDuration("comments", 30*time.Minute)*2)

	setup()
	if !cfg.QuickStart {
		Reindex()
	}
}

/*
Reindex rebuilds all indices in the events bucket
*/
func Reindex() {
	r := func(name string) {
		if r := recover(); r != nil {
			log.Errorf("failed to reindex %s store\n", name)
		}
	}

	for _, t := range []interface{}{&Event{}, &Bookmark{}} {
		tn := reflect.TypeOf(t).String()
		log.Infof("reindexing %s", tn)
		func() {
			defer r(tn)
			db.ReIndex(t, nil)
		}()
	}
}

/*
FlushCaches invalidates all event-related caches
*/
func FlushCaches() {
	log.Infoln("flushing events caches")
	eventsCache.Flush()
	miscCache.Flush()
	bookmarksCache.Flush()
	reviewsCache.Flush()
	commentsCache.Flush()
}

func setup() {}

/*
Get retrieves an event by its Id
*/
func Get(id string) (*Event, error) {
	cacheKey := fmt.Sprintf("get:%s", id)
	if l, ok := eventsCache.Get(cacheKey); ok {
		return l.(*Event), nil
	}

	var event Event
	if err := db.Get(id, &event); err != nil {
		return nil, err
	}

	eventsCache.SetDefault(cacheKey, &event)
	return &event, nil
}

/*
GetAll returns all events
*/
func GetAll() ([]*Event, error) {
	return Find(nil)
}

/*
Find retrieves a list of events matching the given query
*/
func Find(query *EventQuery) ([]*Event, error) {
	cacheKey := fmt.Sprintf("find:%v", query)
	if ee, ok := eventsCache.Get(cacheKey); ok {
		return ee.([]*Event), nil
	}

	var foundEvents []*Event

	q := bolthold.Where("Id").Not().Eq("").Index("Id")

	if query != nil {
		if query.NameLike != "" {
			if re, err := regexp.Compile("(?i)" + query.NameLike); err == nil {
				q.And("Name").RegExp(re)
			}
		}
		if query.TypeEq != "" {
			q.And("Type").Eq(query.TypeEq).Index("Type")
		}
		if query.LecturerIdEq != "" {
			q.And("Lecturers").MatchFunc(func(ra *bolthold.RecordAccess) (bool, error) {
				if field := ra.Field(); field != nil {
					for _, item := range field.([]*Lecturer) {
						if item.Gguid == query.LecturerIdEq {
							return true, nil
						}
					}
				}
				return false, nil
			})
		}
		if query.SemesterEq != "" {
			q.And("Semesters").Contains(query.SemesterEq).Index("Semesters")
		}
		if len(query.CategoryIn) > 0 {
			// Isn't there a better solution?!
			cats := make([]interface{}, 0)
			for _, c := range query.CategoryIn {
				if c != "" {
					cats = append(cats, c)
				}
			}
			if len(cats) > 0 {
				q.And("Categories").ContainsAny(cats...).Index("Categories")
			}
		}
		if query.Skip > 0 {
			q.Skip(query.Skip)
		}
		if query.Limit > 0 {
			q.Limit(query.Limit)
		}
		if len(query.SortFields) > 0 {
			q.SortBy(query.SortFields...)
		}
	}

	err := db.Find(&foundEvents, q)
	if err == nil {
		eventsCache.SetDefault(cacheKey, foundEvents)
	}
	return foundEvents, err
}

/*
Count returns the total number of events
*/
func Count() int {
	cacheKey := "count"
	if c, ok := eventsCache.Get(cacheKey); ok {
		return c.(int)
	}

	all, err := GetAll()
	if err != nil {
		return -1
	}

	eventsCache.SetDefault(cacheKey, len(all))
	return len(all)
}

/*
Insert adds a new event or updates an existing one
*/
func Insert(event *Event, upsert bool, overwrite bool) error {
	f := db.Insert

	if upsert {
		if existing, err := Get(event.Id); err == nil {
			f = db.Upsert
			if !overwrite {
				updateEvent(event, existing)
			}
		}
	}

	eventsCache.Flush()
	miscCache.Flush()
	return f(event.Id, event)
}

/*
InsertMulti adds a new events or updates an existing ones
*/
func InsertMulti(events []*Event, upsert bool, overwrite bool) error {
	f := db.TxInsert
	if upsert {
		f = db.TxUpsert
	}

	tx, err := db.Bolt().Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, e := range events {
		if upsert {
			if existing, err := Get(e.Id); err == nil {
				if !overwrite {
					updateEvent(e, existing)
				}
			}
		}

		if err := f(tx, e.Id, e); err != nil {
			log.Errorf("failed to insert event %s", e.Id)
			tx.Rollback()
		}
	}

	eventsCache.Flush()
	miscCache.Flush()
	return tx.Commit()
}

/*
Delete removes an event by its Id
*/
func Delete(key string) error {
	defer FlushCaches()

	return db.Bolt().Update(func(tx *bbolt.Tx) error {
		if err := db.TxDelete(tx, key, &Event{}); err != nil {
			return err
		}
		if err := db.TxDeleteMatching(tx, &Bookmark{}, bolthold.Where("EntityId").Eq(key)); err != nil {
			return err
		}
		return nil
	})
}

// Inplace
func updateEvent(newEvent, existingEvent *Event) {
	// Keep rating
	newEvent.Rating = existingEvent.Rating
	newEvent.InverseRating = existingEvent.InverseRating

	// Keep semesters
	for _, c := range existingEvent.Semesters {
		if !util.ContainsString(c, newEvent.Semesters) {
			newEvent.Semesters = append(newEvent.Semesters, c)
		}
	}
}

/*
GetFaculties retrieves all faculties
*/
func GetFaculties() ([]string, error) {
	return GetCategoriesAtIndex(config.FacultyIdx)
}

/*
CountFaculties counts the number of faculties
*/
func CountFaculties() int {
	cacheKey := "count:faculties"
	if c, ok := miscCache.Get(cacheKey); ok {
		return c.(int)
	}

	all, err := GetFaculties()
	if err != nil {
		return -1
	}

	miscCache.SetDefault(cacheKey, len(all))
	return len(all)
}

/*
GetCategories retrieves all categories
*/
func GetCategories() ([]string, error) {
	cacheKey := "get:categories"
	if fl, ok := miscCache.Get(cacheKey); ok {
		return fl.([]string), nil
	}

	categories := make([]string, 0)

	for i := 0; ; i++ {
		batch, err := GetCategoriesAtIndex(i)
		if err != nil || len(batch) == 0 {
			break
		}
		categories = append(categories, batch...)
	}

	miscCache.SetDefault(cacheKey, categories)
	return categories, nil
}

/*
GetCategoriesAtIndex ?
*/
func GetCategoriesAtIndex(index int) ([]string, error) {
	cacheKey := fmt.Sprintf("get:categories:index:%d", index)
	if fl, ok := miscCache.Get(cacheKey); ok {
		return fl.([]string), nil
	}

	categoryMap := make(map[string]bool)
	events, err := GetAll()
	if err != nil {
		return []string{}, err
	}

	for _, l := range events {
		if len(l.Categories) > index {
			if _, ok := categoryMap[l.Categories[index]]; !ok {
				categoryMap[l.Categories[index]] = true
			}
		}
	}

	var i int
	categories := make([]string, len(categoryMap))
	for k := range categoryMap {
		categories[i] = k
		i++
	}

	miscCache.SetDefault(cacheKey, categories)
	return categories, nil
}

/*
CountCategories counts the number of categories
*/
func CountCategories() int {
	cacheKey := "count:categories"
	if c, ok := miscCache.Get(cacheKey); ok {
		return c.(int)
	}

	all, err := GetCategories()
	if err != nil {
		return -1
	}

	miscCache.SetDefault(cacheKey, len(all))
	return len(all)
}

/*
GetTypes retrieves all types of events
*/
func GetTypes() ([]string, error) {
	cacheKey := "get:types"
	if fl, ok := miscCache.Get(cacheKey); ok {
		return fl.([]string), nil
	}

	typeMap := make(map[string]bool)
	events, err := GetAll()
	if err != nil {
		return []string{}, err
	}

	for _, l := range events {
		if _, ok := typeMap[l.Type]; !ok {
			typeMap[l.Type] = true
		}
	}

	var i int
	types := make([]string, len(typeMap))
	for k := range typeMap {
		types[i] = k
		i++
	}

	sort.Sort(sort.StringSlice(types))

	miscCache.SetDefault(cacheKey, types)
	return types, nil
}

/*
GetLecturers retrieves all lecturers
*/
func GetLecturers() ([]*Lecturer, error) {
	cacheKey := "get:lecturers"
	if fl, ok := miscCache.Get(cacheKey); ok {
		return fl.([]*Lecturer), nil
	}

	lecturerMap := make(map[string]*Lecturer)
	events, err := GetAll()
	if err != nil {
		return []*Lecturer{}, err
	}

	for _, l := range events {
		for _, le := range l.Lecturers {
			if _, ok := lecturerMap[le.Gguid]; !ok {
				lecturerMap[le.Gguid] = le
			}
		}
	}

	var i int
	lecturers := make([]*Lecturer, len(lecturerMap))
	for _, v := range lecturerMap {
		lecturers[i] = v
		i++
	}

	sort.Slice(lecturers, func(i, j int) bool {
		return strings.Compare(lecturers[i].Name, lecturers[j].Name) < 0
	})

	miscCache.SetDefault(cacheKey, lecturers)
	return lecturers, nil
}

/*
GetSemesters retrieves all semesters
*/
func GetSemesters() (Semesters, error) {
	cacheKey := "get:semesters"
	if fl, ok := miscCache.Get(cacheKey); ok {
		return fl.(Semesters), nil
	}

	semesterMap := make(map[string]bool)
	events, err := GetAll()
	if err != nil {
		return Semesters{}, err
	}

	for _, l := range events {
		for _, s := range l.Semesters {
			if _, ok := semesterMap[s]; !ok {
				semesterMap[s] = true
			}
		}
	}

	semesters := Semesters{}
	for k := range semesterMap {
		semesters = append(semesters, k)
	}
	sort.Sort(sort.Reverse(semesters))

	miscCache.SetDefault(cacheKey, semesters)
	return semesters, nil
}

/*
GetCurrentSemester returns the current semester
*/
func GetCurrentSemester() (curr string, err error) {
	cacheKey := "get:semester:current"
	if s, ok := miscCache.Get(cacheKey); ok {
		return s.(string), nil
	}

	allSemesters, err := GetSemesters()
	if err == nil && len(allSemesters) > 0 {
		curr = allSemesters[0]
		miscCache.SetDefault(cacheKey, curr)
	}

	return curr, err
}

/*
MustGetCurrentSemester returns the current semester
*/
func MustGetCurrentSemester() string {
	if s, err := GetCurrentSemester(); err == nil {
		return s
	}
	return ""
}

/*
GetReview retrieves a review by its Id
*/
func GetReview(id string) (*Review, error) {
	cacheKey := fmt.Sprintf("get:%s", id)
	if c, ok := reviewsCache.Get(cacheKey); ok {
		return c.(*Review), nil
	}

	var review Review

	if err := db.Get(id, &review); err != nil {
		return &review, err
	}

	reviewsCache.SetDefault(cacheKey, &review)
	return &review, nil
}

/*
FindReviews retrieves a list of reviews matching the given query
*/
func FindReviews(query *ReviewQuery) ([]*Review, error) {
	cacheKey := fmt.Sprintf("find:%v", query)
	if rr, ok := reviewsCache.Get(cacheKey); ok {
		return rr.([]*Review), nil
	}

	var foundReviews []*Review

	q := bolthold.
		Where("Id").Not().Eq("").Index("Id")

	if query != nil {
		if query.EventIdEq != "" {
			q.And("EventId").Eq(query.EventIdEq).Index("EventId")
		}
		if query.UserIdEq != "" {
			q.And("UserId").Eq(query.UserIdEq).Index("UserId")
		}
	}

	err := db.Find(&foundReviews, q.SortBy("CreatedAt").Reverse())
	if err == nil {
		reviewsCache.SetDefault(cacheKey, foundReviews)
	}
	return foundReviews, err
}

/*
FindCountReviews returns the amount of reviews matching the given query
*/
func FindCountReviews(query *ReviewQuery) int {
	cacheKey := fmt.Sprintf("count:%v", query)
	if c, ok := reviewsCache.Get(cacheKey); ok {
		return c.(int)
	}
	if all, err := FindReviews(query); err == nil {
		reviewsCache.SetDefault(cacheKey, len(all))
		return len(all)
	}
	return -1
}

/*
GetAllReviews retrieves all reviews
*/
func GetAllReviews() ([]*Review, error) {
	return FindReviews(&ReviewQuery{})
}

/*
GetReviewAverages aggregates the average ratings of all reviews of a given event
*/
func GetReviewAverages(eventID string) (map[string]float32, error) {
	cacheKey := fmt.Sprintf("avg:%s", eventID)
	if avg, ok := reviewsCache.Get(cacheKey); ok {
		return avg.(map[string]float32), nil
	}

	result := make(map[string]float32)
	count := make(map[string]float32)

	reviews, err := FindReviews(&ReviewQuery{
		EventIdEq: eventID,
	})

	if err != nil {
		return result, err
	}

	for _, r := range reviews {
		for k, v := range r.Ratings {
			if _, ok := result[k]; !ok {
				result[k] = 0
			}
			if _, ok := count[k]; !ok {
				count[k] = 0
			}
			result[k] += float32(v)
			count[k]++
		}
	}

	for k, v := range result {
		result[k] = float32(math.Round(float64(v/count[k])*10) / 10)
	}

	reviewsCache.SetDefault(cacheKey, result)
	return result, nil
}

/*
InsertReview adds a new review or updates an existing one
*/
func InsertReview(review *Review, upsert bool) error {
	review.Id = fmt.Sprintf("%s:%s", review.UserId, review.EventId)

	f := db.Insert
	if upsert {
		if existing, err := GetReview(review.Id); err == nil {
			f = db.Upsert
			updateReview(review, existing)
		}
	}

	reviewsCache.Flush()
	return f(review.Id, review)
}

/*
DeleteReview removes a review by its Id
*/
func DeleteReview(key string) error {
	defer reviewsCache.Flush()
	return db.Delete(key, &Review{})
}

func updateReview(newReview, existingReview *Review) {
	for k, v := range existingReview.Ratings {
		if _, ok := newReview.Ratings[k]; !ok {
			newReview.Ratings[k] = v
		}
	}
}

/*
CountReviews the total number of reviews
*/
func CountReviews() int {
	cacheKey := "count"
	if c, ok := reviewsCache.Get(cacheKey); ok {
		return c.(int)
	}

	all, err := GetAllReviews()
	if err != nil {
		return -1
	}

	reviewsCache.SetDefault(cacheKey, len(all))
	return len(all)
}

/*
GetComment retrieves a comment by its Id
*/
func GetComment(id string) (*Comment, error) {
	cacheKey := fmt.Sprintf("get:%s", id)
	if c, ok := commentsCache.Get(cacheKey); ok {
		return c.(*Comment), nil
	}

	var comment Comment

	if err := db.Get(id, &comment); err != nil {
		return &comment, err
	}

	commentsCache.SetDefault(cacheKey, &comment)
	return &comment, nil
}

/*
FindComments retrieves a list of comments matching the given query
*/
func FindComments(query *CommentQuery) ([]*Comment, error) {
	cacheKey := fmt.Sprintf("find:%v", query)
	if cc, ok := commentsCache.Get(cacheKey); ok {
		return cc.([]*Comment), nil
	}

	var foundComments []*Comment

	q := bolthold.
		Where("Id").Not().Eq("").Index("Id").
		And("Active").Eq(query.ActiveEq).Index("Active")

	if query != nil {
		if query.EventIdEq != "" {
			q.And("EventId").Eq(query.EventIdEq).Index("EventId")
		}
		if query.UserIdEq != "" {
			q.And("UserId").Eq(query.UserIdEq).Index("UserId")
		}
		if query.Skip > 0 {
			q.Skip(query.Skip)
		}
		if query.Limit > 0 {
			q.Limit(query.Limit)
		}
	}

	err := db.Find(&foundComments, q.SortBy("CreatedAt"))
	if err == nil {
		commentsCache.SetDefault(cacheKey, foundComments)
	}
	return foundComments, err
}

/*
GetAllComments retrieves all available comments
*/
func GetAllComments() ([]*Comment, error) {
	inactive, err := FindComments(&CommentQuery{
		ActiveEq: false,
	})
	if err != nil {
		return []*Comment{}, err
	}
	active, err := FindComments(&CommentQuery{
		ActiveEq: true,
	})

	all := make([]*Comment, len(active)+len(inactive))
	for i, c := range inactive {
		all[i] = c
	}

	for i, c := range active {
		all[i+len(inactive)] = c
	}

	return all, nil
}

/*
CountComments returns the total number of comments
*/
func CountComments() int {
	cacheKey := "count"
	if c, ok := commentsCache.Get(cacheKey); ok {
		return c.(int)
	}

	all, err := GetAllComments()
	if err != nil {
		return -1
	}

	commentsCache.SetDefault(cacheKey, len(all))
	return len(all)
}

/*
InsertComment adds a new comment or updates an existing one
*/
func InsertComment(comment *Comment, upsert bool) error {
	commentsCache.Flush()

	f := db.Insert
	if upsert {
		f = db.Upsert
	}

	return f(comment.Id, comment)
}

/*
DeleteComment removes a comment by its Id
*/
func DeleteComment(id string) error {
	commentsCache.Flush()
	return db.Delete(id, &Comment{})
}

/*
GetMaxCommentIndexByEvent returns the index of the last inserted comment for the given event
*/
func GetMaxCommentIndexByEvent(eventID string) (count uint8, err error) {
	cacheKey := fmt.Sprintf("max:index:%s", eventID)
	if e, ok := commentsCache.Get(cacheKey); ok {
		return e.(uint8), err
	}

	q := bolthold.
		Where("EventId").Eq(eventID).Index("EventId")

	var result Comment

	aggResult, err := db.FindAggregate(&result, q)
	if err == nil && len(aggResult) == 1 && aggResult[0].Count() > 0 {
		aggResult[0].Max("Index", &result)
		count = result.Index
		commentsCache.SetDefault(cacheKey, count)
	}

	return count, err
}

/*
GetBookmark retrieves a bookmark by its Id
*/
func GetBookmark(key uint64) (*Bookmark, error) {
	var bookmark Bookmark
	if err := db.Get(key, &bookmark); err != nil {
		return nil, err
	}
	return &bookmark, nil
}

/*
GetAllBookmarks retrieves all bookmarks
*/
func GetAllBookmarks() ([]*Bookmark, error) {
	cacheKey := "get:bookmark:all"
	if bb, ok := eventsCache.Get(cacheKey); ok {
		return bb.([]*Bookmark), nil
	}

	var bookmarks []*Bookmark
	if err := db.Find(&bookmarks, &bolthold.Query{}); err != nil {
		return bookmarks, err
	}

	bookmarksCache.SetDefault(cacheKey, bookmarks)
	return bookmarks, nil
}

/*
FindBookmark retrieves the bookmarks for the given user and event
*/
func FindBookmark(userID, entityID string) (*Bookmark, error) {
	cacheKey := fmt.Sprintf("find:%s:%s", userID, entityID)
	if l, ok := bookmarksCache.Get(cacheKey); ok {
		return l.(*Bookmark), nil
	}

	var bookmark Bookmark
	if err := db.FindOne(&bookmark, bolthold.
		Where("UserId").
		Eq(userID).
		Index("UserId").
		And("EntityId").
		Eq(entityID).
		Index("EntityId")); err != nil {
		return nil, err
	}

	bookmarksCache.SetDefault(cacheKey, &bookmark)
	return &bookmark, nil
}

/*
FindBookmarks retrieves a list of bookmarks for the given user
*/
func FindBookmarks(userID string) ([]*Bookmark, error) {
	cacheKey := fmt.Sprintf("find:%s", userID)
	if l, ok := bookmarksCache.Get(cacheKey); ok {
		return l.([]*Bookmark), nil
	}

	var bookmarks []*Bookmark
	if err := db.Find(&bookmarks, bolthold.
		Where("UserId").
		Eq(userID).
		Index("UserId")); err != nil {
		return nil, err
	}

	bookmarksCache.SetDefault(cacheKey, bookmarks)
	return bookmarks, nil
}

/*
CountBookmarks counts the number of bookmarks
*/
func CountBookmarks() int {
	cacheKey := "count"
	if c, ok := bookmarksCache.Get(cacheKey); ok {
		return c.(int)
	}

	all, err := GetAllBookmarks()
	if err != nil {
		return -1
	}

	bookmarksCache.SetDefault(cacheKey, len(all))
	return len(all)
}

/*
InsertBookmark adds a new bookmark or updates an existing one
*/
func InsertBookmark(bookmark *Bookmark) error {
	bookmarksCache.Flush()
	return db.Insert(bolthold.NextSequence(), bookmark)
}

/*
DeleteBookmark removes a user by its Id
*/
func DeleteBookmark(bookmark *Bookmark) error {
	bookmarksCache.Flush()
	return db.Delete(bookmark.Id, bookmark)
}
