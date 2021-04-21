package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"assist/db"
	assist_db "assist/db"

	"github.com/gorilla/mux"
)

func (app *App) methodCreateQueue(w http.ResponseWriter, r *http.Request) error {
	params := mux.Vars(r)
	ctx := r.Context()

	squadId := params["squadId"]

	_, authLevel := app.checkAuthorization(r, "", squadId, squadAdmin|squadOwner)

	if authLevel == 0 {
		err := fmt.Errorf("Current user is not authorized to to add queues to squad " + squadId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	var qr assist_db.QueueRecord
	err := json.NewDecoder(r.Body).Decode(&qr)
	if err != nil {
		err = fmt.Errorf("Failed to decode requests queue from the HTTP request: %w", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	qr.SquadId = squadId

	err = app.db.CreateRequestsQueue(ctx, qr.ID, &qr.QueueInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	return nil
}

func (app *App) methodDeleteQueue(w http.ResponseWriter, r *http.Request) error {
	params := mux.Vars(r)
	ctx := r.Context()

	squadId := params["squadId"]
	queueId := params["queueId"]

	_, authLevel := app.checkAuthorization(r, "", squadId, squadAdmin|squadOwner)

	if authLevel == 0 {
		err := fmt.Errorf("Current user is not authorized to delete queues in squad " + squadId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	queue, err := app.db.GetRequestQueue(ctx, queueId)
	if err != nil || queue.SquadId != squadId {
		err := fmt.Errorf("There is no queue " + queueId + " in squad " + squadId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	err = app.db.DeleteRequestsQueue(ctx, queueId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	return nil
}

func (app *App) methodGetSquadQueues(w http.ResponseWriter, r *http.Request) error {
	params := mux.Vars(r)
	ctx := r.Context()

	squadId := params["squadId"]

	_, authLevel := app.checkAuthorization(r, "", squadId, squadMember|squadAdmin|squadOwner)

	if authLevel == 0 {
		err := fmt.Errorf("Current user is not authorized to get squad " + squadId + " queues")
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	queues, err := app.db.GetRequestQueues(ctx, squadId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(queues)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return err
}

func (app *App) methodGetUserQueues(w http.ResponseWriter, r *http.Request) error {
	params := mux.Vars(r)
	ctx := r.Context()

	userId := params["userId"]

	userId, authLevel := app.checkAuthorization(r, userId, "", myself)

	if authLevel == 0 {
		err := fmt.Errorf("Current user is not authorized to get queues for user " + userId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	userData, err := app.db.GetUserData(ctx, userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	squadsAll, err := app.db.GetUserSquads(ctx, userId, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	squadsAdmin, err := app.db.GetUserSquads(ctx, userId, "admin")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	userQueues, userRequests, queuesToApprove, requestsToApprove, queuesToHandle, requestsToHandle, err := app.db.GetUserRequests(ctx, userData.UserTags, squadsAdmin, squadsAll)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(struct {
		UserQueueus       interface{} `json:"userQueues"`
		UserRequests      interface{} `json:"userRequests"`
		QueuesToApprove   interface{} `json:"queuesToApprove"`
		RequestsToApprove interface{} `json:"requestsToApprove"`
		QueuesToHandle    interface{} `json:"queuesToHandle"`
		RequestsToHandle  interface{} `json:"requestsToHandle"`
	}{userQueues, userRequests, queuesToApprove, requestsToApprove, queuesToHandle, requestsToHandle})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return err
}

func (app *App) methodCreateRequest(w http.ResponseWriter, r *http.Request) error {
	params := mux.Vars(r)
	ctx := r.Context()

	queueId := params["queueId"]

	queue, err := app.db.GetRequestQueue(ctx, queueId)
	if err != nil {
		err := fmt.Errorf("Failed to get queue " + queueId + " details")
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	userId, authLevel := app.checkAuthorization(r, "me", queue.SquadId, squadAdmin|squadOwner|squadMember)

	if authLevel == 0 {
		err := fmt.Errorf("Current user is not authorized to to create requests in queue " + queueId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	var request assist_db.RequestDetails
	err = json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		err = fmt.Errorf("Failed to decode request details from the HTTP request: %w", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	if queue.Approvers == "" {
		request.Status = db.Processing
	}

	requestId, err := app.db.CreateRequest(ctx, &request, queueId, userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	// notify squad members about new event
	go func() {
		var memberIds []string
		var notification string
		var err error
		if request.Status == db.Processing {
			notification = "New request in queue " + queueId + " waiting to be processed"
			memberIds, err = app.db.GetSquadMemberIdsByTag(context.Background(), queue.SquadId, queue.Handlers)
		} else {
			notification = "New request in queue " + queueId + " waiting to be approved"
			memberIds, err = app.db.GetSquadMemberIdsByTag(context.Background(), queue.SquadId, queue.Approvers)
		}

		if err != nil {
			log.Println("Failed to get list of squad " + queue.SquadId + " members, will not be able to create notifications")
		}
		app.ntfs.createNotification(memberIds, "Request "+queueId, notification)
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(struct {
		RequestId string `json:"requestId"`
		Status    int    `json:"status"`
	}{requestId, int(request.Status)})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return nil
}
