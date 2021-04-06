package db

import (
	"context"
	"fmt"
	"log"

	"sync"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

type QueueInfo struct {
	SquadId                string `json:"squadId"`
	Approvers              string `json:"approvers"`
	Handlers               string `json:"handlers"`
	RequestsWaitingApprove int    `json:"requestsWaitingApprove"`
	RequestsProcessing     int    `json:"requestsProcessing"`
}

type QueueRecord struct {
	ID string `json:"id"`
	QueueInfo
}

type RequestStatusType int

const (
	WaitingApprove RequestStatusType = iota
	Processing
)

func (db *FirestoreDB) CreateRequestsQueue(ctx context.Context, queueId string, qi *QueueInfo) (err error) {

	if db.dev {
		log.Println("Creating queue " + qi.SquadId + "/" + queueId)
	}

	batch := db.Client.Batch()

	queueRef := db.RequestQueues.Doc(queueId)
	batch.Set(queueRef, qi)

	if qi.Approvers != "" {
		approvers := qi.SquadId + "/" + qi.Approvers
		batch.Update(queueRef, []firestore.Update{
			{Path: "ApproversPath", Value: approvers},
		})
	}
	if qi.Handlers != "" {
		handlers := qi.SquadId + "/" + qi.Handlers
		batch.Update(queueRef, []firestore.Update{
			{Path: "HandlersPath", Value: handlers},
		})
	}

	_, err = batch.Commit(ctx)
	if err != nil {
		return err
	}

	return err
}

func (db *FirestoreDB) getQueuesFromQuery(ctx context.Context, query firestore.Query) ([]*QueueRecord, error) {
	queues := make([]*QueueRecord, 0)
	iter := query.Documents(ctx)

	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to get queues: %w", err)
		}

		qi := &QueueRecord{}
		doc.DataTo(qi)
		if err != nil {
			return nil, fmt.Errorf("Failed to get queue info: %w", err)
		}
		qi.ID = doc.Ref.ID

		queues = append(queues, qi)
	}

	return queues, nil
}

func (db *FirestoreDB) GetRequestQueues(ctx context.Context, squadId string) ([]*QueueRecord, error) {

	if db.dev {
		log.Println("Getting request queues for squad " + squadId)
	}

	return db.getQueuesFromQuery(ctx, db.RequestQueues.Where("SquadId", "==", squadId))

}

func (db *FirestoreDB) GetUserRequestQueues(ctx context.Context, userTags []string) (queuesToApprove []*QueueRecord, queuesToHandle []*QueueRecord, err error) {

	if db.dev {
		log.Printf("Getting request queues for user tags %v", userTags)
	}

	var errs [2]error
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		queuesToApprove, errs[0] = db.getQueuesFromQuery(ctx, db.RequestQueues.Where("ApproversPath", "in", userTags))
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		queuesToHandle, errs[1] = db.getQueuesFromQuery(ctx, db.RequestQueues.Where("HandlersPath", "in", userTags))
		wg.Done()
	}()

	wg.Wait()

	for _, e := range errs {
		if e != nil {
			return nil, nil, e
		}
	}

	return queuesToApprove, queuesToHandle, nil
}
