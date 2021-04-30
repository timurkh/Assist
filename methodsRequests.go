package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"assist/db"
	assist_db "assist/db"

	gorilla_context "github.com/gorilla/context"
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

func (app *App) methodGetUserQueuesAndRequests(w http.ResponseWriter, r *http.Request) error {
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

	userQueues, userRequests, queuesToApprove, requestsToApprove, queuesToHandle, requestsToHandle, err := app.db.GetUserQueuesAndRequests(ctx, userId, userData.UserTags, squadsAdmin, squadsAll)
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
	ctx := r.Context()

	var request assist_db.RequestDetails
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		err = fmt.Errorf("Failed to decode request details from the HTTP request: %w", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	queue, err := app.db.GetRequestQueue(ctx, request.QueueId)
	if err != nil {
		err := fmt.Errorf("Failed to get queue " + request.QueueId + " details")
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	userId, authLevel := app.checkAuthorization(r, "me", queue.SquadId, squadAdmin|squadOwner|squadMember)

	if authLevel == 0 {
		err := fmt.Errorf("Current user is not authorized to to create requests in queue " + request.QueueId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	request.UserId = userId
	userData, err := app.db.GetUserData(ctx, userId)
	request.UserName = userData.DisplayName

	if queue.Approvers == "" {
		request.Status = db.Processing
	}

	requestId, err := app.db.CreateRequest(ctx, &request)
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
			notification = "New request in queue " + request.QueueId + " waiting to be processed"
			memberIds, err = app.db.GetSquadMemberIdsByTag(context.Background(), queue.SquadId, queue.Handlers)
		} else {
			notification = "New request in queue " + request.QueueId + " waiting to be approved"
			memberIds, err = app.db.GetSquadMemberIdsByTag(context.Background(), queue.SquadId, queue.Approvers)
		}

		if err != nil {
			log.Println("Failed to get list of squad " + queue.SquadId + " members, will not be able to create notifications")
		}
		app.ntfs.createNotification(memberIds, "Request "+request.QueueId, notification)
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

func (app *App) methodGetRequests(w http.ResponseWriter, r *http.Request) (err error) {
	ctx := r.Context()

	v := r.URL.Query()
	from := v.Get("from")
	var timeFrom time.Time
	if from != "" {
		timeFrom, err = time.Parse(time.RFC3339, from)
		if err != nil {
			err = fmt.Errorf("Failed to convert from to a time struct: %w", err)
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return err
		}
	}
	status := v.Get("status")

	userId, authLevel := app.checkAuthorization(r, "me", "", myself)

	if authLevel == 0 {
		err := fmt.Errorf("Current user is not authorized to get requests for user " + userId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	var requests interface{}
	if status == "WaitingApprove" || status == "Processing" {
		userData, err := app.db.GetUserData(ctx, userId)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}

		squadsAdmin, err := app.db.GetUserSquads(ctx, userId, "admin")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}

		requests, err = app.db.GetRequestsByTag(ctx, userData.UserTags, squadsAdmin, assist_db.RequestStatusFromString(status), &timeFrom)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}
	} else if status == "User" {
		requests, err = app.db.GetUserRequests(ctx, userId, &timeFrom)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}

	} else {
		err = fmt.Errorf("Status not supported")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(requests)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return err
}

func (app *App) methodSetRequestStatus(w http.ResponseWriter, r *http.Request) error {
	params := mux.Vars(r)
	ctx := r.Context()

	requestId := params["requestId"]

	var requestDetails struct {
		Status assist_db.RequestStatusType `json:"status"`
	}
	err := json.NewDecoder(r.Body).Decode(&requestDetails)
	if err != nil {
		err = fmt.Errorf("Failed to decode request details from the HTTP request: %w", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	// get request details
	request, err := app.db.GetRequest(ctx, requestId)
	if err != nil {
		err = fmt.Errorf("Failed to get request details: %w", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	// get queue
	queue, err := app.db.GetRequestQueue(ctx, request.QueueId)
	if err != nil {
		err = fmt.Errorf("Failed to get request queue details: %w", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	ud := app.sd.getCurrentUserData(r)
	userId := ud.UID

	// check authorization and get
	var memberIds []string
	var notification string
	authorized := false
	switch requestDetails.Status {
	case assist_db.Cancelled:
		authorized = request.UserId == userId
	case assist_db.Declined, assist_db.Processing:
		if queue.Approvers != "" && ud.HasTag(queue.SquadId+"/"+queue.Approvers) {
			authorized = true
		} else {
			// check if user is admin
			status, err := app.db.GetSquadMemberStatus(ctx, userId, queue.SquadId)
			if err == nil && status == assist_db.Admin {
				authorized = true
			}
		}

		if authorized && requestDetails.Status == assist_db.Processing {
			notification = "New request in queue " + request.QueueId + " waiting to be processed"
			memberIds, err = app.db.GetSquadMemberIdsByTag(context.Background(), queue.SquadId, queue.Handlers)
			if err != nil {
				log.Println("Failed to get list of squad " + queue.SquadId + " members, will not be able to create notifications")
			}
		}
	case assist_db.Completed:
		if ud.HasTag(queue.Handlers) {
			authorized = true
		} else if ud.HasTag(queue.Approvers) {
			authorized = true
		} else {
			// check if user is admin
			status, err := app.db.GetSquadMemberStatus(ctx, userId, queue.SquadId)
			if err == nil && status >= assist_db.Admin {
				authorized = true
			}
		}
		if authorized {
			notification = "Request '" + request.QueueId + " : " + request.Details + "' completed"
			memberIds = []string{request.UserId}
		}
	}

	if !authorized {
		err := fmt.Errorf("Current user %v is not authorized to mark request %v as %v", ud.UID, requestId, requestDetails.Status.String())
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	gorilla_context.Set(r, "AuthChecked", true)

	err = app.db.SetRequestStatus(ctx, requestId, requestDetails.Status)
	if err != nil {
		err = fmt.Errorf("Failed to set request "+requestId+" status: %w", err)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	// send notifications
	if notification != "" {
		go func() {
			app.ntfs.createNotification(memberIds, "Request "+request.QueueId, notification)
		}()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	return nil
}
