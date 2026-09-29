//go:build ignore

// Bir kullanıcıya yönetici (ADMIN) rolü verir ya da geri alır.
// Yöneticiler kurum doğrulama başvurularını inceler.
//
// Kullanım:
//
//	go run scripts/set_admin.go ornek@eposta.com          # yönetici yap
//	go run scripts/set_admin.go --remove ornek@eposta.com # yöneticiliği kaldır
//
// .env'deki FIREBASE_PROJECT_ID ve GOOGLE_APPLICATION_CREDENTIALS kullanılır.
// Kullanıcı uygulamada en az bir kez giriş yapmış olmalı (profil belgesi o zaman oluşur).
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

func main() {
	_ = godotenv.Load()
	args := os.Args[1:]
	remove := len(args) > 0 && args[0] == "--remove"
	if remove {
		args = args[1:]
	}
	if len(args) != 1 {
		log.Fatal("kullanım: go run scripts/set_admin.go [--remove] <e-posta>")
	}
	email := args[0]

	ctx := context.Background()
	var opts []option.ClientOption
	if cred := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); cred != "" {
		opts = append(opts, option.WithCredentialsFile(cred))
	}
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: os.Getenv("FIREBASE_PROJECT_ID")}, opts...)
	if err != nil {
		log.Fatal(err)
	}
	authClient, err := app.Auth(ctx)
	if err != nil {
		log.Fatal(err)
	}
	db, err := app.Firestore(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	record, err := authClient.GetUserByEmail(ctx, email)
	if err != nil {
		log.Fatalf("%s bulunamadı: %v", email, err)
	}
	doc := db.Collection("users").Doc(record.UID)
	if _, err := doc.Get(ctx); err != nil {
		log.Fatalf("%s için profil yok; önce uygulamada giriş yapmalı: %v", email, err)
	}

	op := firestore.ArrayUnion("ADMIN")
	if remove {
		op = firestore.ArrayRemove("ADMIN")
	}
	if _, err := doc.Update(ctx, []firestore.Update{{Path: "roles", Value: op}}); err != nil {
		log.Fatal(err)
	}
	if remove {
		fmt.Printf("%s artık yönetici değil.\n", email)
	} else {
		fmt.Printf("%s yönetici yapıldı (uid %s). Uygulamada Profil > Kurum başvuruları görünür.\n", email, record.UID)
	}
}
