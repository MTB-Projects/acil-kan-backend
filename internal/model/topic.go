package model

import "time"

type Topic struct {
	Name        string    `firestore:"name" json:"name"`
	City        string    `firestore:"city" json:"city"`
	District    string    `firestore:"district" json:"district"`
	Subscribers []User    `firestore:"subscribers" json:"subscribers"`
	CreatedAt   time.Time `firestore:"created_at" json:"created_at"`
	UpdatedAt   time.Time `firestore:"updated_at" json:"updated_at"`
}
