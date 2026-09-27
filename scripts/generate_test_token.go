//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"acilkan.backend/config"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx := context.Background()

	// Get test user ID from command line or use default
	testUID := "test-user-123"
	if len(os.Args) > 1 {
		testUID = os.Args[1]
	}

	// Create custom token for testing
	token, err := cfg.FirebaseAuth.CustomToken(ctx, testUID)
	if err != nil {
		log.Fatalf("Failed to create custom token: %v", err)
	}

	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║              Firebase Test Token Generated                     ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("User ID: %s\n", testUID)
	fmt.Println()
	fmt.Println("Custom Token (use this for testing):")
	fmt.Println(token)
	fmt.Println()
	fmt.Println("⚠️  NOT: Bu bir custom token. ID token'a çevirmek için:")
	fmt.Println("1. Firebase REST API kullanın:")
	fmt.Printf("   curl -X POST 'https://identitytoolkit.googleapis.com/v1/accounts:signInWithCustomToken?key=YOUR_API_KEY' \\\n")
	fmt.Printf("     -H 'Content-Type: application/json' \\\n")
	fmt.Printf("     -d '{\"token\":\"%s\",\"returnSecureToken\":true}'\n", token)
	fmt.Println()
	fmt.Println("2. Response'daki 'idToken' değerini kullanın")
	fmt.Println()
	fmt.Println("📝 Kullanım örneği:")
	fmt.Printf("   export TOKEN=\"<id_token_from_response>\"\n")
	fmt.Printf("   curl -H \"Authorization: Bearer $TOKEN\" http://localhost:8080/api/v1/users/me\n")
	fmt.Println()
}
