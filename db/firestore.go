package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/patrickmn/go-cache"
	"google.golang.org/api/iterator"
)

const ALL_USERS_SQUAD = "All Users"
const MEMBERS = "members"

type FirestoreDB struct {
	dev               bool
	Client            *firestore.Client
	Squads            *firestore.CollectionRef
	Users             *firestore.CollectionRef
	Events            *firestore.CollectionRef
	RequestQueues     *firestore.CollectionRef
	updater           *AsyncUpdater
	userDataCache     *cache.Cache
	memberStatusCache *cache.Cache
	eventDataCache    *cache.Cache
}

var testPrefix string = ""
var numRecords int = 10

func SetTestPrefix(prefix string) {
	testPrefix = prefix
}

func TimeTrack(name string, start time.Time) {
	elapsed := time.Since(start)

	log.Println(fmt.Sprintf("%s took %s", name, elapsed))
}

// init firestore
func NewFirestoreDB(fireapp *firebase.App, dev bool) (*FirestoreDB, error) {
	ctx := context.Background()

	if testPrefix == "" {
		prefix := os.Getenv("TEST_PREFIX")
		if prefix != "" {
			SetTestPrefix(prefix)
		}
	}

	dbClient, err := fireapp.Firestore(ctx)
	if err != nil {
		return nil, err
	}

	err = dbClient.RunTransaction(ctx, func(ctx context.Context, t *firestore.Transaction) error {
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("Could not connect to database: %v", err)
	}

	uc := cache.New(24*time.Hour, 10*time.Minute)
	sc := cache.New(24*time.Hour, 10*time.Minute)
	ec := cache.New(24*time.Hour, 10*time.Minute)

	return &FirestoreDB{
		dev:               dev,
		Client:            dbClient,
		Squads:            dbClient.Collection(testPrefix + "squads"),
		Users:             dbClient.Collection(testPrefix + "squads").Doc(ALL_USERS_SQUAD).Collection("members"),
		Events:            dbClient.Collection(testPrefix + "events"),
		RequestQueues:     dbClient.Collection(testPrefix + "queues"),
		updater:           initAsyncUpdater(),
		userDataCache:     uc,
		memberStatusCache: sc,
		eventDataCache:    ec,
	}, nil
}

func (db *FirestoreDB) updateDocProperty(ctx context.Context, doc *firestore.DocumentRef, field string, val interface{}) error {
	_, err := doc.Update(ctx, []firestore.Update{
		{
			Path:  field,
			Value: val,
		},
	})
	return err
}

func (db *FirestoreDB) deleteDocRecurse(ctx context.Context, doc *firestore.DocumentRef) error {
	iterCollections := doc.Collections(ctx)
	for {
		collRef, err := iterCollections.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("Failed to iterate through collections: %w", err)
		}
		db.DeleteCollectionRecurse(ctx, collRef)
	}

	_, err := doc.Delete(ctx)
	if err != nil {
		return fmt.Errorf("Failed to delete doc: %w", err)
	}
	return nil
}

func (db *FirestoreDB) DeleteCollectionRecurse(ctx context.Context, collection *firestore.CollectionRef) error {
	iter := collection.Documents(ctx)

	var wg sync.WaitGroup

	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Failed to iterate through docs: %v", err)
		}

		wg.Add(1)
		go func() {

			defer wg.Done()
			err = db.deleteDocRecurse(ctx, doc.Ref)
			if err != nil {
				log.Printf("Failed to delete co recourse: %v", err)
			}
		}()
	}

	wg.Wait()
	return nil
}

func (db *FirestoreDB) AddFilterWhere(query firestore.Query, filter *map[string]string, statusFromStringFunc func(string) int) firestore.Query {
	if filter != nil {
		f := *filter
		if f["Keys"] != "" {
			if db.dev {
				log.Printf("\tapplying filter by keys %v\n", f["Keys"])
			}
			query = query.Where("Keys", "array-contains-any", strings.Fields(strings.ToLower(f["Keys"])))
		}

		if f["Tag"] != "" {
			if db.dev {
				log.Printf("\tapplying filter by tag %v\n", f["Tag"])
			}
			query = query.Where("Tags", "array-contains-any", strings.Fields(f["Tag"]))
		}

		if f["Status"] != "" && statusFromStringFunc != nil {
			if s := statusFromStringFunc(f["Status"]); s != -1 {
				if db.dev {
					log.Printf("\tapplying filter by status %v\n", f["Status"])
				}
				query = query.Where("Status", "==", s)
			}
		}
	}

	return query
}

func (db *FirestoreDB) GetFilteredQuery(collection *firestore.CollectionRef, from *time.Time, filter *map[string]string, statusFromStringFunc func(string) int) *firestore.Query {
	query := collection.OrderBy("Timestamp", firestore.Asc)
	if from != nil {
		query = query.StartAfter(from)
	}

	query = db.AddFilterWhere(query, filter, statusFromString)
	query = query.Limit(numRecords)

	return &query
}

func (db *FirestoreDB) GetFilteredIDsQuery(collection *firestore.CollectionRef, from string, filter *map[string]string) *firestore.Query {
	query := collection.OrderBy(firestore.DocumentID, firestore.Asc)
	if from != "" {
		query = query.StartAfter(from)
	}

	query = db.AddFilterWhere(query, filter, statusFromString)

	return &query
}

func (db *FirestoreDB) DeleteGroup(ctx context.Context, groupType string, groupCollection *firestore.CollectionRef, membersCollection string, groupId string) error {

	if db.dev {
		log.Println("Deleting " + groupType + " " + groupId)
	}

	docGroup := groupCollection.Doc(groupId)

	//delete this event from all participants
	go func() {
		iter := docGroup.Collection(MEMBERS).Documents(ctx)
		defer iter.Stop()
		for {
			docUser, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("Error while iterating through %v %v participants: %v", groupType, groupId, err.Error())
				break
			}

			userId := docUser.Ref.ID
			if db.dev {
				log.Printf("Deleting %v %v from user %v", groupType, groupId, userId)
			}

			db.Users.Doc(userId).Collection(membersCollection).Doc(groupId).Delete(ctx)
		}
	}()

	//delete squad itself
	err := db.deleteDocRecurse(ctx, docGroup)
	if err != nil {
		return fmt.Errorf("Error while deleting %v %v: %w", groupType, groupId, err)
	}

	return nil
}
