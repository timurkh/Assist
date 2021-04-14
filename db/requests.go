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

func (db *FirestoreDB) DeleteRequestsQueue(ctx context.Context, queueId string) (err error) {
	_, err = db.RequestQueues.Doc(queueId).Delete(ctx)

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

func (db *FirestoreDB) getQueuesToApproveAndHandleIds(ctx context.Context, userTags []string, squadsAdmin []string) (map[string]int, map[string]int, error) {

	queuesToApprove := make(map[string]int, 0)
	queuesToHandle := make(map[string]int, 0)

	var errs [4]error
	var wg sync.WaitGroup

	// get ids of queues that this user should approve
	wg.Add(1)
	go func() {
		iter := db.RequestQueues.Where("ApproversPath", "in", userTags).Select().Documents(ctx)
		defer iter.Stop()
		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				errs[0] = fmt.Errorf("Failed to get queues: %w", err)
				return
			}
			queuesToApprove[doc.Ref.ID] = -1
		}
		wg.Done()
	}()

	// get ids of queues that this user should approve
	wg.Add(1)
	go func() {
		iter := db.RequestQueues.Where("HandlersPath", "in", userTags).Select().Documents(ctx)
		defer iter.Stop()
		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				errs[1] = fmt.Errorf("Failed to get queues: %w", err)
				return
			}
			queuesToHandle[doc.Ref.ID] = -1
		}
		wg.Done()
	}()

	// get ids of queues from squads that this user is administrating
	wg.Add(1)
	go func() {
		iter := db.RequestQueues.Where("squadId", "in", squadsAdmin).Select().Documents(ctx)
		defer iter.Stop()
		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				errs[1] = fmt.Errorf("Failed to get queues: %w", err)
				return
			}
			queuesToApprove[doc.Ref.ID] = -1
			queuesToHandle[doc.Ref.ID] = -1
		}
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

func (db *FirestoreDB) GetQueuesToApproveAndHandle(ctx context.Context, userTags []string, squadsAdmin []string) (map[string]int, map[string]int, error) {

	if db.dev {
		log.Printf("Getting request queues for user tags %v", userTags)
	}

	// get maps with queue ids and -1 as value
	queuesToApprove, queuesToHandle, err := db.getQueuesToApproveAndHandleIds(ctx, userTags, squadsAdmin)

	if err != nil {
		return nil, nil, err
	}

	//now get amount of requests to approve and to handle
	for k := range queuesToApprove {
		doc, err := db.RequestQueues.Doc(k).Get(ctx)
		if err != nil {
			return nil, nil, err
		}

		data := doc.Data()
		queuesToApprove[k] = int(data["RequestsWaitingApprove"].(int64))

		if _, found := queuesToHandle[k]; found {
			queuesToHandle[k] = int(data["RequestsBeingProcessed"].(int64))
		}
	}
	for k, v := range queuesToHandle {

		if v == -1 {
			doc, err := db.RequestQueues.Doc(k).Get(ctx)
			if err != nil {
				return nil, nil, err
			}
			data := doc.Data()
			queuesToHandle[k] = int(data["RequestsBeingProcessed"].(int64))
		}
	}

	return queuesToApprove, queuesToHandle, nil
}

func (db *FirestoreDB) GetUserRequests(ctx context.Context, userTags []string, squadsAdmin []string, squadsAll []string) (userQueues []string, userRequests []string, queuesToApprove []string, requestsToApprove []string, queuesToHandle []string, requestsToHandle []string, err error) {

	return nil, nil, nil, nil, nil, nil, nil
}
