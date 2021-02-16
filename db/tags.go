package db

import (
	"context"
	"fmt"
	"log"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

type TagInfo struct {
	Values []string `json:"values"`
}

type Tag struct {
	Name string `json:"name"`
	TagInfo
}

func (db *FirestoreDB) CreateTag(ctx context.Context, squadId string, tag *Tag) (err error) {

	log.Printf("Creating tag '%+v' in squad '%v'", tag, squadId)

	_, err = db.Squads.Doc(squadId).Collection("tags").Doc(tag.Name).Create(ctx, tag.TagInfo)
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

		ti := &TagInfo{}
		err = doc.DataTo(ti)
		if err != nil {
			return nil, fmt.Errorf("Failed to get squad tags: %w", err)
		}
		t := &Tag{
			Name:    doc.Ref.ID,
			TagInfo: *ti,
		}
		tags = append(tags, t)
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

	tags, ok := doc.Data()["tags"]

	if ok {
		return tags.([]interface{}), nil
	} else {
		return nil, nil
	}

}

func (db *FirestoreDB) SetSquadMemberTag(ctx context.Context, userId string, squadId string, tagName string, tagValue string) ([]interface{}, error) {

	tagNew := tagName
	if tagValue != "" {
		tagNew = tagName + ":" + tagValue
	}

	log.Println("Setting tag " + tagName + " to user " + userId + " from squad " + squadId)

	tags, err := db.GetSquadMemberTags(ctx, userId, squadId)
	if err != nil {
		return nil, err
	}

	tagFound := false

	for i, tag := range tags {
		name := strings.Split(tag.(string), ":")[0]
		if name == tagName {
			tags[i] = tagNew
			tagFound = true
		}
	}

	if !tagFound {
		tags = append(tags, tagNew)
	}

	err = db.SetSquadMemberTags(ctx, userId, squadId, &tags)

	if err != nil {
		return nil, err
	}

	return tags, nil
}

func (db *FirestoreDB) SetSquadMemberTags(ctx context.Context, userId string, squadId string, tags *[]interface{}) error {

	_, err := db.Squads.Doc(squadId).Collection("members").Doc(userId).Update(ctx, []firestore.Update{
		{Path: "tags", Value: tags},
	})

	if err != nil {
		log.Println("Failed to update tags for  user "+userId+" from squad "+squadId+": %v", err)
		return err
	}

	return nil
}
