//go:build ignore

// firestore.indexes.json dosyasındaki composite index'leri Firestore'da oluşturur.
// Firebase CLI gerektirmez; servis hesabı anahtarıyla çalışır.
//
// Kullanım: go run scripts/deploy_indexes.go
// (.env'deki FIREBASE_PROJECT_ID ve GOOGLE_APPLICATION_CREDENTIALS kullanılır)
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	admin "cloud.google.com/go/firestore/apiv1/admin"
	"cloud.google.com/go/firestore/apiv1/admin/adminpb"
	"github.com/joho/godotenv"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type indexFile struct {
	Indexes []struct {
		CollectionGroup string `json:"collectionGroup"`
		QueryScope      string `json:"queryScope"`
		Fields          []struct {
			FieldPath string `json:"fieldPath"`
			Order     string `json:"order"`
		} `json:"fields"`
	} `json:"indexes"`
}

func main() {
	_ = godotenv.Load()
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	if projectID == "" {
		log.Fatal("FIREBASE_PROJECT_ID gerekli")
	}

	raw, err := os.ReadFile("firestore.indexes.json")
	if err != nil {
		log.Fatal(err)
	}
	var file indexFile
	if err := json.Unmarshal(raw, &file); err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	var opts []option.ClientOption
	if cred := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); cred != "" {
		opts = append(opts, option.WithCredentialsFile(cred))
	}
	client, err := admin.NewFirestoreAdminClient(ctx, opts...)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	for _, idx := range file.Indexes {
		parent := fmt.Sprintf("projects/%s/databases/(default)/collectionGroups/%s", projectID, idx.CollectionGroup)

		want := &adminpb.Index{QueryScope: adminpb.Index_COLLECTION}
		var desc []string
		for _, f := range idx.Fields {
			order := adminpb.Index_IndexField_ASCENDING
			if f.Order == "DESCENDING" {
				order = adminpb.Index_IndexField_DESCENDING
			}
			want.Fields = append(want.Fields, &adminpb.Index_IndexField{
				FieldPath: f.FieldPath,
				ValueMode: &adminpb.Index_IndexField_Order_{Order: order},
			})
			desc = append(desc, f.FieldPath+" "+f.Order[:3])
		}
		label := idx.CollectionGroup + " (" + strings.Join(desc, ", ") + ")"

		exists, err := indexExists(ctx, client, parent, want)
		if err != nil {
			log.Fatalf("%s: listelenemedi: %v", label, err)
		}
		if exists {
			fmt.Println("✓ zaten var:", label)
			continue
		}

		if _, err := client.CreateIndex(ctx, &adminpb.CreateIndexRequest{Parent: parent, Index: want}); err != nil {
			log.Fatalf("%s: oluşturulamadı: %v", label, err)
		}
		fmt.Println("+ oluşturuluyor:", label)
	}
	fmt.Println("Bitti. Yeni index'lerin hazır olması birkaç dakika sürebilir.")
}

func indexExists(ctx context.Context, client *admin.FirestoreAdminClient, parent string, want *adminpb.Index) (bool, error) {
	it := client.ListIndexes(ctx, &adminpb.ListIndexesRequest{Parent: parent})
	for {
		idx, err := it.Next()
		if err == iterator.Done {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if sameFields(idx, want) {
			return true, nil
		}
	}
}

// Firestore mevcut index'lere otomatik __name__ alanı ekler; karşılaştırmada yok sayılır
func sameFields(have, want *adminpb.Index) bool {
	var fields []*adminpb.Index_IndexField
	for _, f := range have.Fields {
		if f.FieldPath != "__name__" {
			fields = append(fields, f)
		}
	}
	if have.QueryScope != want.QueryScope || len(fields) != len(want.Fields) {
		return false
	}
	for i, f := range fields {
		if f.FieldPath != want.Fields[i].FieldPath || f.GetOrder() != want.Fields[i].GetOrder() {
			return false
		}
	}
	return true
}
