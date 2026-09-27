package config

import (
	"context"
	"log"
	"os"
	"strconv"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

type Config struct {
	Port               string
	Env                string
	FirebaseProjectID  string
	CORSAllowOrigins   string
	RateLimitPerMinute int
	FirebaseApp        *firebase.App
	FirebaseAuth       *auth.Client
	FirestoreClient    *firestore.Client
}

var AppConfig *Config

// LoadConfig loads environment variables and initializes Firebase
func LoadConfig() (*Config, error) {
	// Load .env file if exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	port := getEnv("PORT", "8080")
	env := getEnv("ENV", "development")
	projectID := getEnv("FIREBASE_PROJECT_ID", "")
	credPath := getEnv("GOOGLE_APPLICATION_CREDENTIALS", "")

	if projectID == "" {
		log.Fatal("FIREBASE_PROJECT_ID is required")
	}

	ctx := context.Background()

	// Initialize Firebase Admin SDK. Without a credentials file, Application
	// Default Credentials are used (Cloud Run, GKE, gcloud auth ...).
	var opts []option.ClientOption
	if credPath != "" {
		opts = append(opts, option.WithCredentialsFile(credPath))
	}

	app, err := firebase.NewApp(ctx, &firebase.Config{
		ProjectID: projectID,
	}, opts...)
	if err != nil {
		return nil, err
	}

	// Initialize Auth client
	authClient, err := app.Auth(ctx)
	if err != nil {
		return nil, err
	}

	// Initialize Firestore client
	firestoreClient, err := app.Firestore(ctx)
	if err != nil {
		return nil, err
	}

	config := &Config{
		Port:               port,
		Env:                env,
		FirebaseProjectID:  projectID,
		CORSAllowOrigins:   getEnv("CORS_ALLOW_ORIGINS", "*"),
		RateLimitPerMinute: getEnvInt("RATE_LIMIT_PER_MINUTE", 60),
		FirebaseApp:        app,
		FirebaseAuth:       authClient,
		FirestoreClient:    firestoreClient,
	}

	AppConfig = config
	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if n, err := strconv.Atoi(os.Getenv(key)); err == nil && n > 0 {
		return n
	}
	return defaultValue
}
