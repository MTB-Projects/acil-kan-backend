package repository

import (
	"context"
	"time"

	"acilkan.backend/internal/model"
	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

type UserRepository struct {
	db *firestore.Client
}

func NewUserRepository(db *firestore.Client) *UserRepository {
	return &UserRepository{db: db}
}

// GetByUID retrieves a user by Firebase UID
func (r *UserRepository) GetByUID(ctx context.Context, uid string) (*model.User, error) {
	doc, err := r.db.Collection("users").Doc(uid).Get(ctx)
	if err != nil {
		return nil, err
	}

	var user model.User
	if err := doc.DataTo(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

// Create creates a new user profile
func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	_, err := r.db.Collection("users").Doc(user.UID).Set(ctx, user)
	return err
}

// Update updates an existing user profile
func (r *UserRepository) Update(ctx context.Context, user *model.User) error {
	user.UpdatedAt = time.Now()

	_, err := r.db.Collection("users").Doc(user.UID).Set(ctx, user)
	return err
}

// UpdateFCMToken updates only the FCM token for a user
func (r *UserRepository) UpdateFCMToken(ctx context.Context, uid, token string) error {
	_, err := r.db.Collection("users").Doc(uid).Update(ctx, []firestore.Update{
		{Path: "fcm_token", Value: token},
		{Path: "updated_at", Value: time.Now()},
	})
	return err
}

// FindCompatibleDonors finds users who can donate to a specific blood type in a city
func (r *UserRepository) FindCompatibleDonors(ctx context.Context, bloodType model.BloodType, city string) ([]*model.User, error) {
	// Get all donors in the city
	iter := r.db.Collection("users").
		Where("is_donor", "==", true).
		Where("city", "==", city).
		Documents(ctx)

	var donors []*model.User
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var user model.User
		if err := doc.DataTo(&user); err != nil {
			continue
		}

		// Check if this donor is compatible and can donate
		if user.IsCompatibleDonor(bloodType) && user.CanDonate() {
			donors = append(donors, &user)
		}
	}

	return donors, nil
}
