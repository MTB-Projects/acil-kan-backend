package repository

import (
	"context"
	"time"

	"acilkan.backend/internal/model"
	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

type BloodRequestRepository struct {
	db *firestore.Client
}

func NewBloodRequestRepository(db *firestore.Client) *BloodRequestRepository {
	return &BloodRequestRepository{db: db}
}

// Create creates a new blood request
func (r *BloodRequestRepository) Create(ctx context.Context, request *model.BloodRequest) error {
	request.CreatedAt = time.Now()
	request.UpdatedAt = time.Now()

	_, err := r.db.Collection("blood_requests").Doc(request.ID).Set(ctx, request)
	return err
}

// GetByID retrieves a blood request by ID
func (r *BloodRequestRepository) GetByID(ctx context.Context, id string) (*model.BloodRequest, error) {
	doc, err := r.db.Collection("blood_requests").Doc(id).Get(ctx)
	if err != nil {
		return nil, err
	}

	var request model.BloodRequest
	if err := doc.DataTo(&request); err != nil {
		return nil, err
	}

	return &request, nil
}

// Update updates an existing blood request
func (r *BloodRequestRepository) Update(ctx context.Context, request *model.BloodRequest) error {
	request.UpdatedAt = time.Now()

	_, err := r.db.Collection("blood_requests").Doc(request.ID).Set(ctx, request)
	return err
}

// GetActiveRequests retrieves all active blood requests
func (r *BloodRequestRepository) GetActiveRequests(ctx context.Context) ([]*model.BloodRequest, error) {
	iter := r.db.Collection("blood_requests").
		Where("status", "==", string(model.RequestStatusActive)).
		Where("expires_at", ">", time.Now()).
		OrderBy("expires_at", firestore.Asc).
		Documents(ctx)

	var requests []*model.BloodRequest
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var request model.BloodRequest
		if err := doc.DataTo(&request); err != nil {
			continue
		}

		requests = append(requests, &request)
	}

	return requests, nil
}

// RequestFilters represents filter options for blood requests
type RequestFilters struct {
	City      string
	District  string
	BloodType string
}

// GetActiveRequestsWithFilters retrieves active blood requests with optional filters
func (r *BloodRequestRepository) GetActiveRequestsWithFilters(ctx context.Context, filters RequestFilters) ([]*model.BloodRequest, error) {
	query := r.db.Collection("blood_requests").
		Where("status", "==", string(model.RequestStatusActive)).
		Where("expires_at", ">", time.Now())

	// Apply filters
	if filters.City != "" {
		query = query.Where("city", "==", filters.City)
	}
	if filters.District != "" {
		query = query.Where("district", "==", filters.District)
	}
	if filters.BloodType != "" {
		query = query.Where("blood_type", "==", filters.BloodType)
	}

	// Order by expiration time (most urgent first)
	query = query.OrderBy("expires_at", firestore.Asc)

	iter := query.Documents(ctx)

	var requests []*model.BloodRequest
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var request model.BloodRequest
		if err := doc.DataTo(&request); err != nil {
			continue
		}

		requests = append(requests, &request)
	}

	return requests, nil
}

// GetUserRequests retrieves all requests created by a specific user
func (r *BloodRequestRepository) GetUserRequests(ctx context.Context, uid string) ([]*model.BloodRequest, error) {
	iter := r.db.Collection("blood_requests").
		Where("requester_uid", "==", uid).
		OrderBy("created_at", firestore.Desc).
		Documents(ctx)

	var requests []*model.BloodRequest
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var request model.BloodRequest
		if err := doc.DataTo(&request); err != nil {
			continue
		}

		requests = append(requests, &request)
	}

	return requests, nil
}

// CountUserRequestsSince counts how many requests (in any status) a user has created since the given time
func (r *BloodRequestRepository) CountUserRequestsSince(ctx context.Context, uid string, since time.Time) (int, error) {
	iter := r.db.Collection("blood_requests").
		Where("requester_uid", "==", uid).
		Where("created_at", ">=", since).
		Documents(ctx)

	count := 0
	for {
		_, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}
		count++
	}

	return count, nil
}

// AddNotifiedUser adds a user UID to the list of notified users
func (r *BloodRequestRepository) AddNotifiedUser(ctx context.Context, requestID, userUID string) error {
	_, err := r.db.Collection("blood_requests").Doc(requestID).Update(ctx, []firestore.Update{
		{Path: "notified_user_uids", Value: firestore.ArrayUnion(userUID)},
	})
	return err
}
