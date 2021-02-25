package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"google.golang.org/api/iterator"
)

const ALL_USERS_SQUAD = "All Users"
const USER_SQUADS = "member_squads"

type FirestoreDB struct {
	Client  *firestore.Client
	Squads  *firestore.CollectionRef
	Users   *firestore.CollectionRef
	updater *AsyncUpdater
}

var testPrefix string = ""

func SetTestPrefix(prefix string) {
	testPrefix = prefix
}

// init firestore
func NewFirestoreDB(fireapp *firebase.App) (*FirestoreDB, error) {
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
		dbClient,
		dbClient.Collection(testPrefix + "squads"),
		dbClient.Collection(testPrefix + "squads").Doc(ALL_USERS_SQUAD).Collection("members"),
		initAsyncUpdater(),
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
