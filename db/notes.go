package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

type Note struct {
	Title     string    `json:"title"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
}

type NoteRecord struct {
	ID string `json:"id"`
	Note
}

func (db *FirestoreDB) CreateNote(ctx context.Context, squadId string, note *Note) (id string, err error) {

	log.Printf("Creating note '%+v' in squad '%v'", note, squadId)

	dbNotes := db.Squads.Doc(squadId).Collection("notes")
	doc, _, err := dbNotes.Add(ctx, map[string]interface{}{
		"Title":     note.Title,
		"Timestamp": firestore.ServerTimestamp,
		"Text":      note.Text,
	})

	if err != nil {
		return "", err
	}

	return doc.ID, nil
}

func (db *FirestoreDB) GetNotes(ctx context.Context, squadId string) (notes []*NoteRecord, err error) {

	log.Printf("Getting notes for squad '%v'", squadId)

	notes = make([]*NoteRecord, 0)

	iter := db.Squads.Doc(squadId).Collection("notes").OrderBy("Timestamp", firestore.Desc).Documents(ctx)

	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to get squad notes: %w", err)
		}

		note := &Note{}
		err = doc.DataTo(note)
		if err != nil {
			return nil, fmt.Errorf("Failed to get squad notes: %w", err)
		}
		nr := &NoteRecord{
			ID:   doc.Ref.ID,
			Note: *note,
		}
		notes = append(notes, nr)
	}

	return notes, nil
}

func (db *FirestoreDB) DeleteNote(ctx context.Context, squadId string, noteId string) error {

	log.Println("Deleting note " + noteId + " from squad " + squadId)

	docSquad := db.Squads.Doc(squadId)
	docNote := docSquad.Collection("notes").Doc(noteId)
	err := db.deleteDocRecurse(ctx, docNote)
	if err != nil {
		return fmt.Errorf("Error while deleting note "+noteId+" from squad "+squadId+": %w", err)
	}

	return nil
}

func (db *FirestoreDB) UpdateNote(ctx context.Context, squadId string, noteId string, note *Note) error {

	log.Printf("Updating note '%v' in squad '%v'", noteId, squadId)

	dbNote := db.Squads.Doc(squadId).Collection("notes").Doc(noteId)
	_, err := dbNote.Set(ctx, map[string]interface{}{
		"Title":     note.Title,
		"Timestamp": firestore.ServerTimestamp,
		"Text":      note.Text,
	}, firestore.MergeAll)

	return err
}
