package db

import (
	"context"
	"fmt"
	"log"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

type Tag struct {
	Name   string           `json:"name"`
	Values map[string]int64 `json:"values"`
}

func (db *FirestoreDB) CreateTag(ctx context.Context, squadId string, tag *Tag) (err error) {

	if db.dev {
		log.Printf("Creating tag '%+v' in squad '%v'", tag, squadId)
	}

	_, err = db.Squads.Doc(squadId).Collection("tags").Doc(tag.Name).Set(ctx, tag.Values)
	return err
}

func (db *FirestoreDB) GetTags(ctx context.Context, squadId string) (tags []*Tag, err error) {

	log.Printf("Getting tags for squad '%v'", squadId)

	tags = make([]*Tag, 0)

	iter := db.Squads.Doc(squadId).Collection("tags").Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to get squad tags: %w", err)
		}

		values := make(map[string]int64)
		values_ := doc.Data()
		for k, v := range values_ {
			values[k] = v.(int64)
		}
		tag := &Tag{
			Name:   doc.Ref.ID,
			Values: values,
		}
		tags = append(tags, tag)
	}

	return tags, nil
}

func (db *FirestoreDB) DeleteTag(ctx context.Context, squadId string, tagName string) error {

	log.Println("Deleting tag " + tagName + " from squad " + squadId)

	docSquad := db.Squads.Doc(squadId)
	docTag := docSquad.Collection("tags").Doc(tagName)

	//TODO delete tag from all members

	//delete squad itself
	err := db.deleteDocRecurse(ctx, docTag)
	if err != nil {
		return fmt.Errorf("Error while deleting tag "+tagName+" from squad "+squadId+": %w", err)
	}

	return nil
}

func (db *FirestoreDB) GetSquadMemberTags(ctx context.Context, userId string, squadId string) ([]interface{}, error) {

	doc, err := db.Squads.Doc(squadId).Collection("members").Doc(userId).Get(ctx)

	if err != nil {
		return nil, fmt.Errorf("Failed to get squad "+squadId+" member "+userId+": %w", err)
	}

	tags, ok := doc.Data()["Tags"]

	if ok && tags != nil {
		return tags.([]interface{}), nil
	} else {
		return nil, nil
	}

}

func (db *FirestoreDB) SetSquadMemberTag(ctx context.Context, userId string, squadId string, tagName string, tagValue string) ([]interface{}, error) {

	tagNew := tagName
	if tagValue != "" {
		tagNew = tagName + "/" + tagValue
	}

	if db.dev {
		log.Println("Setting tag " + tagName + " to user " + userId + " from squad " + squadId)
	}

	tags, err := db.GetSquadMemberTags(ctx, userId, squadId)
	if err != nil {
		return nil, err
	}

	tagFound := false
	tagOldValue := ""

	for i, tag := range tags {
		s := strings.Split(tag.(string), "/")
		name := s[0]
		if name == tagName {
			tagFound = true
			if len(s) == 2 {
				tagOldValue = s[1]
				tags[i] = tagNew
			}
			break
		}
	}

	if !tagFound {
		tags = append(tags, tagNew)
	}

	if !tagFound || tagOldValue != tagValue {

		batch := db.Client.Batch()
		db.SetSquadMemberTags(batch, userId, squadId, &tags)
		db.SetUserTags(batch, userId, squadId, &tags)
		db.UpdateTagCounter(batch, squadId, tagName, tagValue, 1)

		if tagFound {
			db.UpdateTagCounter(batch, squadId, tagName, tagOldValue, -1)
		}
		_, err := batch.Commit(ctx)
		if err != nil {
			fmt.Errorf("Failed to update tags for  user %v from squad %v to %+v: %w", userId, squadId, tags, err)
			return nil, err
		}
	}

	return tags, nil
}

func (db *FirestoreDB) DeleteSquadMemberTag(ctx context.Context, userId string, squadId string, tagName string, tagValue string) ([]interface{}, error) {

	tag := tagName
	if tagValue != "" {
		tag = tag + "/" + tagValue
	}
	log.Println("Deleting tag " + tag + " from user " + userId + " from squad " + squadId)

	tags, err := db.GetSquadMemberTags(ctx, userId, squadId)
	if err != nil {
		return nil, err
	}

	tagFound := false

	for i, t := range tags {
		if t == tag {
			tagFound = true
			if i == len(tags)-1 {
				tags = tags[:i]
			} else if i > 0 {
				tags = append(tags[:i], tags[i+1:]...)
			} else {
				tags = tags[1:]
			}
			break
		}
	}

	batch := db.Client.Batch()
	db.SetSquadMemberTags(batch, userId, squadId, &tags)

	if tagFound {
		db.UpdateTagCounter(batch, squadId, tagName, tagValue, -1)
	}

	_, err = batch.Commit(ctx)
	if err != nil {
		fmt.Errorf("Failed to update tags for  user %v from squad %v to %+v: %w", userId, squadId, tags, err)
		return nil, err
	}

	return tags, nil
}

func (db *FirestoreDB) SetSquadMemberTags(batch *firestore.WriteBatch, userId string, squadId string, tags *[]interface{}) {

	batch.Update(db.Squads.Doc(squadId).Collection("members").Doc(userId), []firestore.Update{
		{Path: "Tags", Value: tags},
	})
}

func (db *FirestoreDB) SetUserTags(batch *firestore.WriteBatch, userId string, squadId string, tags *[]interface{}) {

	squadTags := make([]string, len(*tags))
	for i, v := range *tags {
		squadTags[i] = squadId + "/" + v.(string)
	}

	docUser := db.Users.Doc(userId)

	batch.Update(docUser, []firestore.Update{
		{Path: "UserTags", Value: squadTags},
	})

	db.userDataCache.Delete(userId)
}

func (db *FirestoreDB) UpdateTagCounter(batch *firestore.WriteBatch, squadId string, tag string, value string, inc int) {

	if value == "" {
		value = "_"
	}

	batch.Update(db.Squads.Doc(squadId).Collection("tags").Doc(tag), []firestore.Update{
		{Path: value, Value: firestore.Increment(inc)},
	})

}
