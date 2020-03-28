package events

import (
	"fmt"
	log "github.com/golang/glog"
	"github.com/n1try/kitsquid/app/config"
	"github.com/n1try/kitsquid/app/util"
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
	reviewsCache   *cache.Cache
	bookmarksCache *cache.Cache
	miscCache      *cache.Cache
)

func InitStore(store *bolthold.Store) {
	cfg = config.Get()
	db = store

	eventsCache = cache.New(cfg.CacheDuration("events", 30*time.Minute), cfg.CacheDuration("events", 30*time.Minute)*2)
	miscCache = cache.New(cfg.CacheDuration("misc", 30*time.Minute), cfg.CacheDuration("misc", 30*time.Minute)*2)
	bookmarksCache = cache.New(cfg.CacheDuration("bookmarks", 30*time.Minute), cfg.CacheDuration("bookmarks", 30*time.Minute)*2)
	reviewsCache = cache.New(cfg.CacheDuration("reviews", 30*time.Minute), cfg.CacheDuration("reviews", 30*time.Minute)*2)

	setup()
	if !cfg.QuickStart {
		Reindex()
	}
}

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

func FlushCaches() {
	log.Infoln("flushing events caches")
	eventsCache.Flush()
	miscCache.Flush()
	bookmarksCache.Flush()
}

func setup() {}

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

func GetAll() ([]*Event, error) {
	return Find(nil)
}

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

func GetFaculties() ([]string, error) {
	return GetCategoriesAtIndex(config.FacultyIdx)
}

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

func MustGetCurrentSemester() string {
	if s, err := GetCurrentSemester(); err == nil {
		return s
	}
	return ""
}

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

	err := db.Find(&foundReviews, q)
	if err == nil {
		reviewsCache.SetDefault(cacheKey, foundReviews)
	}
	return foundReviews, err
}

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

func GetAllReviews() ([]*Review, error) {
	return FindReviews(&ReviewQuery{})
}

func GetReviewAverages(eventId string) (map[string]float32, error) {
	cacheKey := fmt.Sprintf("avg:%s", eventId)
	if avg, ok := reviewsCache.Get(cacheKey); ok {
		return avg.(map[string]float32), nil
	}

	result := make(map[string]float32)
	count := make(map[string]float32)

	reviews, err := FindReviews(&ReviewQuery{
		EventIdEq: eventId,
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
			count[k] += 1
		}
	}

	for k, v := range result {
		result[k] = float32(math.Round(float64(v/count[k])*10) / 10)
	}

	reviewsCache.SetDefault(cacheKey, result)
	return result, nil
}

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

func GetBookmark(key uint64) (*Bookmark, error) {
	var bookmark Bookmark
	if err := db.Get(key, &bookmark); err != nil {
		return nil, err
	}
	return &bookmark, nil
}

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

func FindBookmark(userId, entityId string) (*Bookmark, error) {
	cacheKey := fmt.Sprintf("find:%s:%s", userId, entityId)
	if l, ok := bookmarksCache.Get(cacheKey); ok {
		return l.(*Bookmark), nil
	}

	var bookmark Bookmark
	if err := db.FindOne(&bookmark, bolthold.
		Where("UserId").
		Eq(userId).
		Index("UserId").
		And("EntityId").
		Eq(entityId).
		Index("EntityId")); err != nil {
		return nil, err
	}

	bookmarksCache.SetDefault(cacheKey, &bookmark)
	return &bookmark, nil
}

func FindBookmarks(userId string) ([]*Bookmark, error) {
	cacheKey := fmt.Sprintf("find:%s", userId)
	if l, ok := bookmarksCache.Get(cacheKey); ok {
		return l.([]*Bookmark), nil
	}

	var bookmarks []*Bookmark
	if err := db.Find(&bookmarks, bolthold.
		Where("UserId").
		Eq(userId).
		Index("UserId")); err != nil {
		return nil, err
	}

	bookmarksCache.SetDefault(cacheKey, bookmarks)
	return bookmarks, nil
}

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

func InsertBookmark(bookmark *Bookmark) error {
	bookmarksCache.Flush()
	return db.Insert(bolthold.NextSequence(), bookmark)
}

func DeleteBookmark(bookmark *Bookmark) error {
	bookmarksCache.Flush()
	return db.Delete(bookmark.Id, bookmark)
}
