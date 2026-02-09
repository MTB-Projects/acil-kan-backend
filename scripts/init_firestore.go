package main

import (
	"context"
	"log"

	"acilkan.backend/config"
	"acilkan.backend/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	if err := logger.InitLogger(cfg.Env); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting Firestore initialization...")

	ctx := context.Background()

	// Test Firestore connection
	logger.Info("Testing Firestore connection...")
	
	// Try to access Firestore
	iter := cfg.FirestoreClient.Collections(ctx)
	collections, err := iter.GetAll()
	if err != nil {
		logger.Fatal("Failed to connect to Firestore", zap.Error(err))
	}

	logger.Info("Successfully connected to Firestore",
		zap.Int("existing_collections", len(collections)),
	)

	if len(collections) > 0 {
		logger.Info("Existing collections:")
		for _, coll := range collections {
			logger.Info("  - " + coll.ID)
		}
	} else {
		logger.Info("No collections found. Collections will be created automatically when first document is added.")
	}

	logger.Info(`
╔════════════════════════════════════════════════════════════════╗
║                  Firestore Setup Instructions                  ║
╚════════════════════════════════════════════════════════════════╝

✅ Firestore connection successful!

📝 Collections will be created automatically when you:
   1. Create your first user (on first login)
   2. Create your first blood request

🔍 Recommended Indexes (create in Firebase Console):

1. Active Blood Requests:
   Collection: blood_requests
   Fields: status (Ascending), expires_at (Ascending)

2. User Requests:
   Collection: blood_requests
   Fields: requester_uid (Ascending), created_at (Descending)

3. Daily Request Count:
   Collection: blood_requests
   Fields: requester_uid (Asc), status (Asc), created_at (Asc)

4. Compatible Donors:
   Collection: users
   Fields: is_donor (Ascending), city (Ascending)

🔐 Security Rules (already set in Firebase Console):
   - Mobile clients cannot access Firestore directly
   - All operations go through backend API

🚀 Next Steps:
   1. Start the API server: make run
   2. Test health endpoint: curl http://localhost:8080/health
   3. Create your first user via mobile app login
   4. Create indexes as queries are used (Firestore will suggest)

`)

	logger.Info("Initialization complete!")
}
