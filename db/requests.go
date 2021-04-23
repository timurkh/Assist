package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"sync"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

type QueueInfo struct {
	SquadId        string `json:"squadId"`
	Approvers      string `json:"approvers"`
	Handlers       string `json:"handlers"`
	WaitingApprove int    `json:"waitingApprove"`
	Processing     int    `json:"processing"`
}

type QueueRecord struct {
	ID string `json:"id"`
	QueueInfo
}

const REQUESTS = "requests"

type RequestStatusType int

const (
	WaitingApprove RequestStatusType = iota
	Processing
	Completed
	Declined
)

func (s RequestStatusType) String() string {
	texts := []string{
		"WaitingApprove",
		"Processing",
		"Completed",
		"Declined",
	}

	return texts[s]
}

type RequestDetails struct {
	Details  string            `json:"details"`
	Status   RequestStatusType `json:"status"`
	QueueId  string            `json:"queueId"`
	Time     *time.Time        `json:"time"`
	UserId   string            `json:"userId"`
	UserName string            `json:"userName"`
}

type RequestRecord struct {
	RequestId string `json:"requestId"`
	RequestDetails
}

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

func (db *FirestoreDB) GetRequestQueue(ctx context.Context, queueId string) (queueInfo *QueueInfo, err error) {
	doc, err := db.RequestQueues.Doc(queueId).Get(ctx)
	if err != nil {
		return nil, err
	}

	q := &QueueInfo{}
	doc.DataTo(q)

	return q, nil
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

	var errs [3]error
	var wg sync.WaitGroup

	// get ids of queues that this user should approve
	var mxApprove, mxHandle sync.Mutex
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
			mxApprove.Lock()
			queuesToApprove[doc.Ref.ID] = -1
			mxApprove.Unlock()
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
			mxApprove.Lock()
			queuesToHandle[doc.Ref.ID] = -1
			mxApprove.Unlock()
		}
		wg.Done()
	}()

	// get ids of queues from squads that this user is administrating
	wg.Add(1)
	go func() {
		iter := db.RequestQueues.Where("SquadId", "in", squadsAdmin).Select().Documents(ctx)
		defer iter.Stop()
		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				errs[2] = fmt.Errorf("Failed to get queues: %w", err)
				return
			}
			mxApprove.Lock()
			queuesToApprove[doc.Ref.ID] = -1
			mxApprove.Unlock()

			mxHandle.Lock()
			queuesToHandle[doc.Ref.ID] = -1
			mxHandle.Unlock()
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
		val, _ := data["WaitingApprove"].(int64)
		queuesToApprove[k] = int(val)

		if _, found := queuesToHandle[k]; found {
			val, _ = data["Processing"].(int64)
			queuesToHandle[k] = int(val)
		}
	}
	for k, v := range queuesToHandle {

		if v == -1 {
			doc, err := db.RequestQueues.Doc(k).Get(ctx)
			if err != nil {
				return nil, nil, err
			}
			data := doc.Data()
			val, _ := data["Processing"].(int64)
			queuesToHandle[k] = int(val)
		}
	}

	return queuesToApprove, queuesToHandle, nil
}

func (db *FirestoreDB) getRequestsFromQuery(ctx context.Context, query firestore.Query) (requests []RequestRecord, err error) {
	requests = make([]RequestRecord, 0)
	iter := query.Limit(10).Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to get queues: %w", err)
		}
		r := RequestRecord{}
		doc.DataTo(&r)
		r.RequestId = doc.Ref.ID
		requests = append(requests, r)
	}
	return requests, nil
}

func (db *FirestoreDB) GetUserQueuesAndRequests(ctx context.Context, userId string, userTags []string, squadsAdmin []string, squadsAll []string) (userQueues []string, userRequests []RequestRecord, queuesToApprove []string, requestsToApprove []RequestRecord, queuesToHandle []string, requestsToHandle []RequestRecord, err error) {

	// get maps with queue ids and -1 as value
	queuesToApproveMap, queuesToHandleMap, err := db.getQueuesToApproveAndHandleIds(ctx, userTags, squadsAdmin)

	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	queuesToApprove = make([]string, len(queuesToApproveMap))
	i := 0
	for k := range queuesToApproveMap {
		queuesToApprove[i] = k
		i++
	}
	if len(queuesToApprove) > 0 {
		requestsToApprove, err = db.getRequestsFromQuery(ctx, db.Requests.Where("QueueId", "in", queuesToApprove).Where("Status", "==", WaitingApprove).OrderBy("Time", firestore.Desc))
		if err != nil {
			return nil, nil, nil, nil, nil, nil, err
		}
	}

	queuesToHandle = make([]string, len(queuesToHandleMap))
	i = 0
	for k := range queuesToHandleMap {
		queuesToHandle[i] = k
		i++
	}
	if len(queuesToHandle) > 0 {
		requestsToHandle, err = db.getRequestsFromQuery(ctx, db.Requests.Where("QueueId", "in", queuesToHandle).Where("Status", "==", Processing).OrderBy("Time", firestore.Desc))
		if err != nil {
			return nil, nil, nil, nil, nil, nil, err
		}
	}

	// get queues from user squads (where she might file requests)
	userQueues = make([]string, 0)
	iter := db.RequestQueues.Where("SquadId", "in", squadsAll).Select().Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, nil, nil, nil, nil, nil, fmt.Errorf("Failed to get queues: %w", err)
		}
		userQueues = append(userQueues, doc.Ref.ID)
	}

	// get user requests
	userRequests, err = db.getRequestsFromQuery(ctx, db.Requests.Where("UserId", "==", userId).OrderBy("Time", firestore.Desc))
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	return userQueues, userRequests, queuesToApprove, requestsToApprove, queuesToHandle, requestsToHandle, nil
}

func (db *FirestoreDB) CreateRequest(ctx context.Context, request *RequestDetails) (requestId string, err error) {
	if db.dev {
		log.Printf("Creating request %+v\n", request)
	}

	queueDoc := db.RequestQueues.Doc(request.QueueId)
	newRequestDoc := db.Requests.NewDoc()

	batch := db.Client.Batch()

	batch.Set(newRequestDoc, request)
	batch.Set(newRequestDoc, map[string]interface{}{
		"Time": firestore.ServerTimestamp,
	}, firestore.MergeAll)
	batch.Update(queueDoc, []firestore.Update{
		{Path: request.Status.String(), Value: firestore.Increment(1)},
	})

	_, err = batch.Commit(ctx)
	if err != nil {
		return "", err
	}

	return newRequestDoc.ID, nil
}

func (db *FirestoreDB) GetUserRequests(ctx context.Context, userId string, from *time.Time) (requests []RequestRecord, err error) {

	query := db.Requests.Where("UserId", "==", userId).OrderBy("Time", firestore.Desc)
	if from != nil {
		query = query.StartAfter(from)
	}
	requests, err = db.getRequestsFromQuery(ctx, query)
	if err != nil {
		return nil, err
	}

	return requests, nil
}

func (db *FirestoreDB) GetRequestsByTag(ctx context.Context, tags []string, status string, from *time.Time) (requests []RequestRecord, err error) {

	var query firestore.Query
	if status == "WaitingApprove" {
		query = db.Requests.Where("Approvers", "in", tags).Where("Status", "==", WaitingApprove).OrderBy("Time", firestore.Desc)
	} else {
		query = db.Requests.Where("Handlers", "in", tags).Where("Status", "==", Processing).OrderBy("Time", firestore.Desc)
	}

	if from != nil {
		query = query.StartAfter(from)
	}
	requests, err = db.getRequestsFromQuery(ctx, query)
	if err != nil {
		return nil, err
	}

	return requests, nil
}
