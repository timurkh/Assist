package db

import (
	"context"
	"fmt"
	"log"
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
)

func (s ParticipantStatusType) String() string {
	texts := []string{
		"Not going",
		"Applied",
		"Going",
		"Attended",
		"NoShow",
	}

	return texts[s]
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
	Status    ParticipantStatusType `json:"status"`
	Timestamp interface{}           `json:"timestamp"`
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

		return doc.ID, nil
	}

	return "", fmt.Errorf("Failed to create event, not enough details provided: %+v", event)
}

func (db *FirestoreDB) GetEvent(ctx context.Context, ID string) (*EventInfo, error) {

	log.Println("Getting details for event " + ID)

	doc, err := db.Events.Doc(ID).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to get event "+ID+": %w", err)
	}

	s := &EventInfo{}
	err = doc.DataTo(s)
	if err != nil {
		return nil, fmt.Errorf("Failed to get event "+ID+": %w", err)
	}

	return s, nil
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

func (db *FirestoreDB) RegisterParticipant(ctx context.Context, userId string, eventId string, status ParticipantStatusType) error {
	log.Println("Registering user " + userId + " for event " + eventId)

	userInfo, err := db.GetUser(ctx, userId)
	if err != nil {
		return err
	}

	participant := &ParticipantInfo{
		UserInfo: *userInfo,
		Status:   status,
	}

	err = db.AddParticipantRecordToEvent(ctx, eventId, userId, participant)
	if err != nil {
		return err
	}

	eventInfo, err := db.GetEvent(ctx, eventId)
	if err != nil {
		return err
	}

	eventInfo.Status = status

	err = db.AddEventRecordToParticipant(ctx, userId, eventId, eventInfo)
	if err != nil {
		return err
	}

	return nil
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

	go db.propagateChangedEventInfo(eventId, path)

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
	db.propagateChangedGroupInfo(db.Events.Doc(eventId), MEMBERS, eventId, fields...)
}

func (db *FirestoreDB) GetParticipants(ctx context.Context, eventId string, from string, filter *map[string]string) ([]*ParticipantRecord, error) {

	log.Printf("Getting participants of the event %v\n", eventId)

	participants := make([]*ParticipantRecord, 0)

	iter, err := db.GetFilteredDocuments(ctx, db.Events.Doc(eventId).Collection(MEMBERS), from, filter)
	if err != nil {
		return nil, err
	}
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

	go db.propagateChangedEventInfo(eventId, status.String())

	return nil
}

func (db *FirestoreDB) DeleteEvent(ctx context.Context, eventId string) error {
	return db.DeleteGroup(ctx, "event", db.Events, USER_EVENTS, eventId)
}
