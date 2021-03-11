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
	updater           *AsyncUpdater
	userDataCache     sync.Map
	memberStatusCache sync.Map
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

	return &FirestoreDB{
		dev:     dev,
		Client:  dbClient,
		Squads:  dbClient.Collection(testPrefix + "squads"),
		Users:   dbClient.Collection(testPrefix + "squads").Doc(ALL_USERS_SQUAD).Collection("members"),
		Events:  dbClient.Collection(testPrefix + "events"),
		updater: initAsyncUpdater(),
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

func (db *FirestoreDB) propagateChangedGroupInfo(docRef *firestore.DocumentRef, collection string, id string, fields ...string) {
	ctx := context.Background()
	doc, err := docRef.Get(ctx)
	if err != nil {
		log.Printf("Failed to get object %v: %v", id, err)
	}

	vals := make([]interface{}, len(fields))
	for i, field := range fields {
		vals[i] = doc.Data()[field]
	}

	iter := docRef.Collection(collection).Where("Replicant", "!=", true).Documents(ctx)
	defer iter.Stop()
	for {
		docMember, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error while getting group %v member: %v", id, err.Error())
			break
		}

		userId := docMember.Ref.ID

		log.Println("\t\t" + userId)

		doc := db.Users.Doc(userId).Collection(USER_SQUADS).Doc(id)
		for i, field := range fields {
			db.updater.dispatchCommand(doc, field, vals[i])
		}
	}
}

func (db *FirestoreDB) GetFilteredDocuments(ctx context.Context, collection *firestore.CollectionRef, from string, filter *map[string]string) (*firestore.DocumentIterator, error) {
	query := collection.OrderBy("Timestamp", firestore.Asc)
	if from != "" {
		log.Printf("\tstarting from %v\n", from)
		timeFrom, err := time.Parse(time.RFC3339, from)
		if err != nil {
			return nil, fmt.Errorf("Failed to convert from to a time struct: %w", err)
		}

		query = query.StartAfter(timeFrom)
	}

	if filter != nil {
		f := *filter
		if f["Keys"] != "" {
			if db.dev {
				log.Printf("\tapplying filter by keys %v\n", f["Keys"])
			}
			query = query.Where("Keys", "array-contains-any", strings.Fields(strings.ToLower(f["Keys"])))
		}

		if f["Status"] != "" {
			if s := statusFromString(f["Status"]); s != -1 {
				if db.dev {
					log.Printf("\tapplying filter by status %v\n", f["Status"])
				}
				query = query.Where("Status", "==", s)
			}
		}

		if f["Tag"] != "" {
			if db.dev {
				log.Printf("\tapplying filter by tag %v\n", f["Tag"])
			}
			query = query.Where("Tags", "array-contains-any", strings.Fields(f["Tag"]))
		}
	}

	return query.Limit(numRecords).Documents(ctx), nil
}

func (db *FirestoreDB) DeleteGroup(ctx context.Context, groupType string, groupCollection *firestore.CollectionRef, membersCollection string, groupId string) error {

	log.Println("Deleting " + groupType + " " + groupId)

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
			log.Printf("Deleting %v %v from user %v", groupType, groupId, userId)

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
