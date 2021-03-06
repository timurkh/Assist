package db

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/patrickmn/go-cache"
	"google.golang.org/api/iterator"
)

const USER_EVENTS = "participant_events"

type ParticipantStatusType int

const (
	NotGoing ParticipantStatusType = iota
	Applied
	Going
	Attended
	NoShow
	EventOwner
)

var ParticipantStatusTypes = []ParticipantStatusType{
	NotGoing,
	Applied,
	Going,
	Attended,
	NoShow,
	EventOwner,
}

func (s ParticipantStatusType) String() string {
	texts := []string{
		"Not going",
		"Applied",
		"Going",
		"Attended",
		"NoShow",
		"Owner",
	}

	return texts[s]
}

func eventStatusFromString(s string) int {
	for _, t := range ParticipantStatusTypes {
		if t.String() == s {
			return int(t)
		}
	}
	return -1
}

type EventInfo struct {
	Date     *time.Time            `json:"date"`
	TimeFrom string                `json:"timeFrom"`
	TimeTo   string                `json:"timeTo"`
	Text     string                `json:"text"`
	SquadId  string                `json:"squadId"`
	OwnerId  string                `json:"ownerId"`
	Status   ParticipantStatusType `json:"status"`
	Archived bool                  `json:"archived"`
}

type EventRecord struct {
	ID string `json:"id"`
	EventInfo
}

type EventCountersRecord struct {
	ID string `json:"id"`
	EventInfo
	Going    int `json:"going"`
	Applied  int `json:"applied"`
	Attended int `json:"attended"`
	NoShow   int `json:"no-show"`
}

type ParticipantInfo struct {
	UserInfo
	Replicant bool                  `json:"replicant"`
	Tags      []string              `json:"tags"`
	Timestamp interface{}           `json:"timestamp"`
	Status    ParticipantStatusType `json:"status"`
}

type ParticipantRecord struct {
	ID string `json:"id"`
	ParticipantInfo
}

func (db *FirestoreDB) CreateEvent(ctx context.Context, event *EventInfo) (id string, err error) {

	if event.Text != "" && event.Date != nil && event.SquadId != "" {
		if db.dev {
			log.Printf("Creating event '%+v'", event)
		}

		var date time.Time

		// let's ensure all dates are unique
		year, month, day := event.Date.Date()
		startOfTheDay := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		endOfTheDay := time.Date(year, month, day, 24, 0, 0, 0, time.UTC)
		lastDocIter := db.Events.OrderBy("Date", firestore.Desc).Where("Date", ">=", startOfTheDay).Where("Date", "<", endOfTheDay).Limit(1).Documents(ctx)
		defer lastDocIter.Stop()
		lastDoc, err := lastDocIter.Next()
		if err != nil {
			if err != iterator.Done {
				return "", err
			}
			date = time.Date(year, month, day, 7, 0, rand.Intn(60), 0, time.UTC)
		} else {
			ei := EventInfo{}
			lastDoc.DataTo(&ei)
			date = ei.Date.Add(time.Duration(rand.Intn(60)) * time.Second)
		}

		event.Date = &date

		log.Printf("%+v\n", event)
		doc, _, err := db.Events.Add(ctx, event)

		if err != nil {
			return "", err
		}

		go func() {
			event.Status = EventOwner
			db.addEventRecordToParticipant(context.Background(), event.OwnerId, doc.ID, event)
		}()

		return doc.ID, nil
	}

	return "", fmt.Errorf("Failed to create event, not enough details provided: %+v", event)
}

func (db *FirestoreDB) GetEvent(ctx context.Context, ID string) (*EventInfo, error) {

	if db.dev {
		log.Println("Getting details for event " + ID)
	}

	v, found := db.eventDataCache.Get(ID)
	if found {
		return v.(*EventInfo), nil
	} else {
		doc, err := db.Events.Doc(ID).Get(ctx)
		if err != nil {
			return nil, fmt.Errorf("Failed to get event "+ID+": %w", err)
		}

		s := &EventInfo{}
		err = doc.DataTo(s)
		if err != nil {
			return nil, fmt.Errorf("Failed to get event "+ID+": %w", err)
		}

		db.eventDataCache.Set(ID, s, cache.DefaultExpiration)
		return s, nil
	}
}

func getToday() *time.Time {
	year, month, day := time.Now().UTC().Date()
	date := time.Date(year, month, day, 5, 0, 0, 0, time.UTC)
	return &date
}

func (db *FirestoreDB) GetUserEventsMap(ctx context.Context, squads []string, userId string) (map[string]*EventInfo, error) {
	events := make(map[string]*EventInfo, 0)

	iter := db.Users.Doc(userId).Collection(USER_EVENTS).Where("Archived", "==", false).Where("SquadId", "in", squads).Documents(ctx)

	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to get user events: %w", err)
		}

		e := &EventInfo{}
		err = doc.DataTo(e)
		if err != nil {
			return nil, fmt.Errorf("Failed to get user squads: %w", err)
		}

		events[doc.Ref.ID] = e

	}

	return events, nil
}

func (db *FirestoreDB) GetUserEventsCount(ctx context.Context, userId string) (int, error) {
	iter := db.Users.Doc(userId).Collection(USER_EVENTS).Where("Archived", "==", false).Select().Documents(ctx)

	defer iter.Stop()
	count := 0
	for {
		_, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("Failed to get amount of user %s events: %w", userId, err)
		}
		count++

	}

	return count, nil
}

func (db *FirestoreDB) GetUserEvents(ctx context.Context, userId string, limit int) ([]*EventInfo, error) {
	events := make([]*EventInfo, 0)

	query := db.Users.Doc(userId).Collection(USER_EVENTS).Where("Archived", "==", false).OrderBy("Date", firestore.Asc)
	if limit != 0 {
		query = query.Limit(limit)
	}
	iter := query.Documents(ctx)

	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to get user events: %w", err)
		}

		e := &EventInfo{}
		err = doc.DataTo(e)
		if err != nil {
			return nil, fmt.Errorf("Failed to get user squads: %w", err)
		}

		events = append(events, e)

	}

	return events, nil
}

func (db *FirestoreDB) ArchiveOldEvents(ctx context.Context) error {

	batch := db.Client.Batch()

	iter := db.Events.Where("Archived", "==", false).Where("Date", "<", getToday()).Select().Documents(ctx)

	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return fmt.Errorf("Failed to get events: %w", err)
		}

		log.Printf("Marking event %v as archived\b", doc.Ref.ID)
		batch.Update(doc.Ref, []firestore.Update{
			{Path: "Archived", Value: true},
		})

	}

	_, err := batch.Commit(ctx)

	return err
}

func (db *FirestoreDB) GetEvents(ctx context.Context, squads []string, userId string) (events []*EventCountersRecord, err error) {
	var myEvents map[string]*EventInfo
	if userId != "" {
		myEvents, err = db.GetUserEventsMap(ctx, squads, userId)
		if err != nil {
			return nil, err
		}

		// later I will use this field to identify entries that should be archived
		for _, v := range myEvents {
			v.Archived = true
		}
	}

	events = make([]*EventCountersRecord, 0)
	query := db.Events.Where("SquadId", "in", squads).Where("Archived", "==", false)
	query = query.OrderBy("Date", firestore.Asc)
	iter := query.Documents(ctx)

	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("Failed to get events: %w", err)
		}

		e := &EventCountersRecord{}
		err = doc.DataTo(e)
		if err != nil {
			return nil, fmt.Errorf("Failed to get events: %w", err)
		}

		e.ID = doc.Ref.ID

		if myEvents != nil {
			if myEvent, ok := myEvents[e.ID]; ok {
				e.Status = myEvent.Status
				myEvent.Archived = false
			}
		}

		events = append(events, e)
	}

	// Now mark archived events that user has but do not exist in the global list
	batch := db.Client.Batch()
	haveSomethingToArchive := false

	for k, v := range myEvents {
		if v.Archived == true {
			batch.Update(db.Users.Doc(userId).Collection(USER_EVENTS).Doc(k), []firestore.Update{
				{Path: "Archived", Value: true},
			})
			haveSomethingToArchive = true
		}
	}

	if haveSomethingToArchive {
		_, err = batch.Commit(ctx)
		if err != nil {
			return nil, fmt.Errorf("Failed to mark archived user events: %w", err)
		}
	}

	return events, nil
}

func (db *FirestoreDB) GetEventsByStatus(ctx context.Context, squads []string, userId string, status string) (events []*EventCountersRecord, err error) {

	events = make([]*EventCountersRecord, 0)
	query := db.Events.Where("SquadId", "in", squads).Where("Archived", "==", false)
	query = query.Where(status, ">", 0).OrderBy(status, firestore.Desc)
	iter := query.Documents(ctx)

	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("Failed to get events: %w", err)
		}

		e := &EventCountersRecord{}
		err = doc.DataTo(e)
		if err != nil {
			return nil, fmt.Errorf("Failed to get events: %w", err)
		}

		e.ID = doc.Ref.ID

		events = append(events, e)
	}

	return events, nil
}

func (db *FirestoreDB) GetArchivedEvents(ctx context.Context, userId string, from *time.Time, filter *map[string]string) (events []*EventRecord, err error) {

	if db.dev {
		log.Printf("Getting archived events for user %v (from %+v, filter %+v)\n", userId, from, filter)
	}
	events = make([]*EventRecord, 0)
	query := db.Users.Doc(userId).Collection(USER_EVENTS).Where("Archived", "==", true).OrderBy("Date", firestore.Desc)
	if from != nil {
		query = query.Where("Date", "<", from)
	}
	query = db.AddFilterWhere(query, filter, eventStatusFromString)

	iter := query.Limit(numRecords).Documents(ctx)

	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("Failed to get events: %w", err)
		}

		e := &EventRecord{}
		err = doc.DataTo(e)
		if err != nil {
			return nil, fmt.Errorf("Failed to get events: %w", err)
		}

		e.ID = doc.Ref.ID

		events = append(events, e)
	}

	return events, nil
}

func (db *FirestoreDB) RegisterParticipants(ctx context.Context, userIds []string, eventId string, eventInfo *EventInfo, status ParticipantStatusType) error {
	if db.dev {
		log.Println("Registering users " + strings.Join(userIds, ", ") + " for event " + eventId)
	}

	eventInfo.Status = status

	errs := make([]error, len(userIds))
	var wg sync.WaitGroup

	for i, uid := range userIds {

		wg.Add(1)

		go func(i int, userId string) {

			defer wg.Done()

			// if I cared about conistency more, I would need
			// some distributed lock here to keep status numbers
			// consistent; now it is possilbe to send multiple
			// requests to add (or remove) participant from event
			// and counters would be screwed, but I dot think
			// this is important
			if st, err := db.GetParticipantStatus(ctx, userId, eventId); err != nil {

				var userInfo *SquadUserInfo
				userInfo, errs[i] = db.GetSquadMember(ctx, eventInfo.SquadId, userId)
				if errs[i] != nil {
					return
				}

				participant := &ParticipantInfo{
					UserInfo:  userInfo.UserInfo,
					Replicant: userInfo.Replicant,
					Tags:      userInfo.Tags,
					Status:    status,
				}

				errs[i] = db.addParticipantRecordToEvent(ctx, eventId, userId, participant)
				if errs[i] != nil {
					return
				}

				if !userInfo.Replicant {
					errs[i] = db.addEventRecordToParticipant(ctx, userId, eventId, eventInfo)
					if errs[i] != nil {
						return
					}
				}
			} else {
				errs[i] = fmt.Errorf("User %v is already registered for event %v with status %v", userId, eventId, st.String())
			}

		}(i, uid)
	}

	wg.Wait()

	var combinedError error
	for _, err := range errs {
		if err != nil {
			if combinedError == nil {
				combinedError = fmt.Errorf("Failed to register one or more participants for event:")
			}
			combinedError = fmt.Errorf("%s\n\t%w", combinedError, err)
		}
	}

	return combinedError
}

func (db *FirestoreDB) addParticipantRecordToEvent(ctx context.Context, eventId string, userId string, userInfo *ParticipantInfo) error {

	if db.dev {
		log.Println("Adding participant " + userId + " to event " + eventId)
	}

	batch := db.Client.Batch()

	docEvent := db.Events.Doc(eventId)
	docParticipant := docEvent.Collection(MEMBERS).Doc(userId)
	batch.Set(docParticipant, userInfo)

	path := userInfo.Status.String()

	batch.Set(docParticipant, map[string]interface{}{
		"Timestamp": firestore.ServerTimestamp,
	}, firestore.MergeAll)
	batch.Update(docParticipant, []firestore.Update{
		{Path: "Keys", Value: userInfo.Keys()},
	})
	batch.Update(docEvent, []firestore.Update{
		{Path: path, Value: firestore.Increment(1)},
	})

	_, err := batch.Commit(ctx)
	if err != nil {
		return fmt.Errorf("Failed to add participant "+userId+" to event "+eventId+": %w", err)
	}

	return nil
}

func (db *FirestoreDB) addEventRecordToParticipant(ctx context.Context, userId string, eventId string, eventInfo *EventInfo) error {

	doc := db.Users.Doc(userId).Collection(USER_EVENTS).Doc(eventId)

	_, err := doc.Set(ctx, eventInfo)
	if err != nil {
		return fmt.Errorf("Failed to add event to user "+userId+": %w", err)
	}

	return nil
}

func (db *FirestoreDB) GetParticipants(ctx context.Context, eventId string, from *time.Time, filter *map[string]string) ([]*ParticipantRecord, error) {

	if db.dev {
		log.Printf("Getting participants of the event %v (from %v, filter %v)\n", eventId, from, filter)
	}

	participants := make([]*ParticipantRecord, 0)

	iter := db.GetFilteredQuery(db.Events.Doc(eventId).Collection(MEMBERS), from, filter, eventStatusFromString).Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to get event participants: %w", err)
		}

		p := &ParticipantInfo{}
		err = doc.DataTo(p)
		if err != nil {
			return nil, fmt.Errorf("Failed to get event participants: %w", err)
		}
		pr := &ParticipantRecord{
			ID:              doc.Ref.ID,
			ParticipantInfo: *p,
		}
		participants = append(participants, pr)
	}

	return participants, nil
}

func (db *FirestoreDB) GetParticipantStatus(ctx context.Context, userId string, eventId string) (ParticipantStatusType, error) {
	doc, err := db.Events.Doc(eventId).Collection(MEMBERS).Doc(userId).Get(ctx)
	if err != nil {
		return 0, fmt.Errorf("Failed to get event "+eventId+": %w", err)
	}
	status, ok := doc.Data()["Status"]
	if ok {
		return ParticipantStatusType(status.(int64)), nil
	} else {
		return 0, fmt.Errorf("Failed to get event " + eventId + " status")
	}
}

func (db *FirestoreDB) SetParticipantStatus(ctx context.Context, userId string, eventId string, status ParticipantStatusType) error {
	oldStatus, err := db.GetParticipantStatus(ctx, userId, eventId)
	if err != nil {
		return err
	}

	batch := db.Client.Batch()

	docParticipantEvent := db.Users.Doc(userId).Collection(USER_EVENTS).Doc(eventId)
	batch.Update(docParticipantEvent, []firestore.Update{{Path: "Status", Value: status}})

	docEventParticipant := db.Events.Doc(eventId).Collection(MEMBERS).Doc(userId)
	batch.Update(docEventParticipant, []firestore.Update{{Path: "Status", Value: status}})

	docEvent := db.Events.Doc(eventId)
	batch.Update(docEvent, []firestore.Update{
		{Path: oldStatus.String(), Value: firestore.Increment(-1)},
	})
	batch.Update(docEvent, []firestore.Update{
		{Path: status.String(), Value: firestore.Increment(1)},
	})

	_, err = batch.Commit(ctx)
	if err != nil {
		return fmt.Errorf("Failed to change user "+userId+" status: %w", err)
	}

	return nil
}

func (db *FirestoreDB) DeleteParticipant(ctx context.Context, userId string, eventId string) error {
	if db.dev {
		log.Println("Removing user " + userId + " from event " + eventId)
	}

	err := db.deleteEventRecordFromParticipant(ctx, userId, eventId)
	if err != nil {
		return err
	}

	err = db.deleteParticipantRecordFromEvent(ctx, eventId, userId)
	if err != nil {
		return err
	}
	return nil
}

func (db *FirestoreDB) deleteEventRecordFromParticipant(ctx context.Context, userId string, eventId string) error {

	doc := db.Users.Doc(userId).Collection(USER_EVENTS).Doc(eventId)

	_, err := doc.Delete(ctx)
	if err != nil {
		return fmt.Errorf("Failed to remove event from user "+userId+": %w", err)
	}

	return nil
}

func (db *FirestoreDB) deleteParticipantRecordFromEvent(ctx context.Context, eventId string, userId string) error {

	status, err := db.GetParticipantStatus(ctx, userId, eventId)
	if err != nil {
		return err
	}

	batch := db.Client.Batch()
	docEvent := db.Events.Doc(eventId)
	docParticipant := docEvent.Collection(MEMBERS).Doc(userId)
	batch.Delete(docParticipant)
	batch.Update(docEvent, []firestore.Update{
		{Path: status.String(), Value: firestore.Increment(-1)},
	})

	_, err = batch.Commit(ctx)
	if err != nil {
		return fmt.Errorf("Failed to delete user %v from event %v: %w", userId, eventId, err)
	}

	return nil
}

func (db *FirestoreDB) DeleteEvent(ctx context.Context, eventId string) error {
	db.eventDataCache.Delete(eventId)
	return db.deleteGroup(ctx, "event", db.Events, USER_EVENTS, eventId, nil)
}

func (db *FirestoreDB) processIdsTail(candidateIds []string, numCandidates int, idCandidate string, iterCandidates *firestore.DocumentIterator) []string {

	for len(candidateIds) < numCandidates {
		candidateIds = append(candidateIds, idCandidate)
		docCandidate, err := iterCandidates.Next()
		if err != nil {
			break
		}
		idCandidate = docCandidate.Ref.ID
	}

	return candidateIds
}

func (db *FirestoreDB) GetCandidates(ctx context.Context, squadId string, eventId string, from string, filter *map[string]string) ([]*SquadUserInfoRecord, error) {

	if db.dev {
		log.Printf("Getting candidates to participate in the event %v (filter %v)\n", eventId, filter)
	}

	candidateIds := make([]string, 0, numRecords)

	iterCandidates := db.GetFilteredIDsQuery(db.Squads.Doc(squadId).Collection(MEMBERS), from, filter).Select().Documents(ctx)
	defer iterCandidates.Stop()

	iterParticipants := db.GetFilteredIDsQuery(db.Events.Doc(eventId).Collection(MEMBERS), from, filter).Select().Documents(ctx)
	defer iterParticipants.Stop()

	// walk through both lists
	// i intentionally do not check for iter.Next() errors, because otherwise go code is too cumbersome
	idCandidate := "-"
	idParticipant := "-"
	for len(candidateIds) < numRecords {
		if idCandidate == idParticipant { // move both iterators forward
			docCandidate, err := iterCandidates.Next()
			if err != nil {
				break
			}
			idCandidate = docCandidate.Ref.ID

			docParticipant, err := iterParticipants.Next()
			if err != nil { // no participants, thus fill candidateIds array and exit cycle
				candidateIds = db.processIdsTail(candidateIds, numRecords, idCandidate, iterCandidates)
				break
			}
			idParticipant = docParticipant.Ref.ID
			continue
		} else if idCandidate < idParticipant { // save current id and move candidate pointer forward
			candidateIds = append(candidateIds, idCandidate)
			docCandidate, err := iterCandidates.Next()
			if err != nil {
				break
			}
			idCandidate = docCandidate.Ref.ID
		} else { // participant smaller than candidate, this should not normally happen if data is consistent

			docParticipant, err := iterParticipants.Next() //anyway, movel left pointer
			if err != nil {                                // no participants, thus fill candidateIds array and exit cycle
				candidateIds = db.processIdsTail(candidateIds, numRecords, idCandidate, iterCandidates)
				break
			}
			idParticipant = docParticipant.Ref.ID
		}
	}

	// get candidates info
	candidates := make([]*SquadUserInfoRecord, len(candidateIds))

	for i := 0; i < len(candidateIds); i++ {
		doc, err := db.Squads.Doc(squadId).Collection(MEMBERS).Doc(candidateIds[i]).Get(ctx)
		if err != nil {
			return nil, fmt.Errorf("Failed to get event participants: %w", err)
		}
		s := &SquadUserInfo{}
		err = doc.DataTo(s)
		if err != nil {
			return nil, fmt.Errorf("Failed to get event participants: %w", err)
		}
		candidates[i] = &SquadUserInfoRecord{
			ID:            doc.Ref.ID,
			SquadUserInfo: *s,
		}
	}

	return candidates, nil
}
