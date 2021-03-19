package db

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
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
}

type EventRecord struct {
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
		log.Printf("Creating event '%+v'", event)

		doc, _, err := db.Events.Add(ctx, event)

		if err != nil {
			return "", err
		}

		event.Status = EventOwner
		db.AddEventRecordToParticipant(ctx, event.OwnerId, doc.ID, event)

		return doc.ID, nil
	}

	return "", fmt.Errorf("Failed to create event, not enough details provided: %+v", event)
}

func (db *FirestoreDB) GetEvent(ctx context.Context, ID string) (*EventInfo, error) {

	log.Println("Getting details for event " + ID)

	v, found := db.eventDataCache.Load(ID)
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

		db.eventDataCache.Store(ID, s)
		return s, nil
	}
}

func (db *FirestoreDB) GetUserEventsMap(ctx context.Context, userId string) (map[string]*EventInfo, error) {
	events := make(map[string]*EventInfo, 0)

	var iter *firestore.DocumentIterator
	iter = db.Users.Doc(userId).Collection(USER_EVENTS).Documents(ctx)

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

func (db *FirestoreDB) GetUserEvents(ctx context.Context, userId string, limit int) ([]*EventInfo, error) {
	events := make([]*EventInfo, 0)

	var iter *firestore.DocumentIterator
	iter = db.Users.Doc(userId).Collection(USER_EVENTS).OrderBy("Date", firestore.Asc).Limit(limit).Documents(ctx)

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

func (db *FirestoreDB) GetEvents(ctx context.Context, squads []string, userId string) (events []*EventRecord, err error) {

	var myEvents map[string]*EventInfo
	if userId != "" {
		myEvents, err = db.GetUserEventsMap(ctx, userId)
		if err != nil {
			return nil, err
		}
	}

	events = make([]*EventRecord, 0)
	iter := db.Events.Where("SquadId", "in", squads).OrderBy("Date", firestore.Asc).Documents(ctx)

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

		if myEvents != nil {
			if myEvent, ok := myEvents[e.ID]; ok {
				e.Status = myEvent.Status
			}
		}

		events = append(events, e)
	}

	return events, nil
}

func (db *FirestoreDB) RegisterParticipants(ctx context.Context, userIds []string, eventId string, eventInfo *EventInfo, status ParticipantStatusType) error {
	log.Println("Registering users " + strings.Join(userIds, ", ") + " for event " + eventId)

	errs := make([]error, len(userIds))
	var wg sync.WaitGroup

	for i, uid := range userIds {

		wg.Add(1)

		go func(userId string) {

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

				errs[i] = db.AddParticipantRecordToEvent(ctx, eventId, userId, participant)
				if errs[i] != nil {
					return
				}

				eventInfo.Status = status

				errs[i] = db.AddEventRecordToParticipant(ctx, userId, eventId, eventInfo)
				if errs[i] != nil {
					return
				}
			} else {
				errs[i] = fmt.Errorf("User %v is already registered for event %v with status %v", userId, eventId, st.String())
			}

		}(uid)
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

func (db *FirestoreDB) AddParticipantRecordToEvent(ctx context.Context, eventId string, userId string, userInfo *ParticipantInfo) error {

	log.Println("Adding participant " + userId + " to event " + eventId)
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

func (db *FirestoreDB) AddEventRecordToParticipant(ctx context.Context, userId string, eventId string, eventInfo *EventInfo) error {

	doc := db.Users.Doc(userId).Collection(USER_EVENTS).Doc(eventId)

	_, err := doc.Set(ctx, eventInfo)
	if err != nil {
		return fmt.Errorf("Failed to add event to user "+userId+": %w", err)
	}

	return nil
}

func (db *FirestoreDB) propagateChangedEventInfo(eventId string, fields ...string) {
	db.propagateChangedGroupInfo(db.Events.Doc(eventId), USER_EVENTS, eventId, fields...)
}

func (db *FirestoreDB) GetParticipants(ctx context.Context, eventId string, from *time.Time, filter *map[string]string) ([]*ParticipantRecord, error) {

	log.Printf("Getting participants of the event %v (from %v, filter %v)\n", eventId, from, filter)

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

	go db.propagateChangedEventInfo(eventId, oldStatus.String(), status.String())

	return nil
}

func (db *FirestoreDB) DeleteParticipant(ctx context.Context, userId string, eventId string) error {
	log.Println("Removing user " + userId + " from event " + eventId)

	err := db.DeleteEventRecordFromParticipant(ctx, userId, eventId)
	if err != nil {
		return err
	}

	err = db.DeleteParticipantRecordFromEvent(ctx, eventId, userId)
	if err != nil {
		return err
	}
	return nil
}

func (db *FirestoreDB) DeleteEventRecordFromParticipant(ctx context.Context, userId string, eventId string) error {

	doc := db.Users.Doc(userId).Collection(USER_EVENTS).Doc(eventId)

	_, err := doc.Delete(ctx)
	if err != nil {
		return fmt.Errorf("Failed to remove event from user "+userId+": %w", err)
	}

	return nil
}

func (db *FirestoreDB) DeleteParticipantRecordFromEvent(ctx context.Context, eventId string, userId string) error {

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
	return db.DeleteGroup(ctx, "event", db.Events, USER_EVENTS, eventId)
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

	log.Printf("Getting candidates to participate in the event %v (filter %v)\n", eventId, filter)

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
