package repository

import (
	"context"
	"time"

	"acilkan.backend/internal/model"
	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

const institutionApplications = "institution_applications"

type InstitutionRepository struct {
	db *firestore.Client
}

func NewInstitutionRepository(db *firestore.Client) *InstitutionRepository {
	return &InstitutionRepository{db: db}
}

// Create stores a new application
func (r *InstitutionRepository) Create(ctx context.Context, a *model.InstitutionApplication) error {
	a.CreatedAt = time.Now()
	_, err := r.db.Collection(institutionApplications).Doc(a.ID).Set(ctx, a)
	return err
}

// GetByID retrieves an application
func (r *InstitutionRepository) GetByID(ctx context.Context, id string) (*model.InstitutionApplication, error) {
	doc, err := r.db.Collection(institutionApplications).Doc(id).Get(ctx)
	if err != nil {
		return nil, err
	}
	var a model.InstitutionApplication
	if err := doc.DataTo(&a); err != nil {
		return nil, err
	}
	return &a, nil
}

// Update overwrites an application
func (r *InstitutionRepository) Update(ctx context.Context, a *model.InstitutionApplication) error {
	_, err := r.db.Collection(institutionApplications).Doc(a.ID).Set(ctx, a)
	return err
}

// LatestForUser returns the user's most recent application, or nil if there is none
func (r *InstitutionRepository) LatestForUser(ctx context.Context, uid string) (*model.InstitutionApplication, error) {
	list, err := r.list(ctx, r.db.Collection(institutionApplications).
		Where("uid", "==", uid).
		OrderBy("created_at", firestore.Desc).
		Limit(1))
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return list[0], nil
}

// ListByStatus returns applications in a status, newest first
func (r *InstitutionRepository) ListByStatus(ctx context.Context, status model.ApplicationStatus, limit int) ([]*model.InstitutionApplication, error) {
	return r.list(ctx, r.db.Collection(institutionApplications).
		Where("status", "==", string(status)).
		OrderBy("created_at", firestore.Desc).
		Limit(limit))
}

func (r *InstitutionRepository) list(ctx context.Context, q firestore.Query) ([]*model.InstitutionApplication, error) {
	iter := q.Documents(ctx)
	defer iter.Stop()
	var out []*model.InstitutionApplication
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var a model.InstitutionApplication
		if err := doc.DataTo(&a); err != nil {
			continue
		}
		out = append(out, &a)
	}
	return out, nil
}
