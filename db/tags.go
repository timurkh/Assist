package db

import (
	"context"
	"fmt"
	"log"

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
