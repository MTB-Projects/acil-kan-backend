#!/bin/bash

# Firestore Index Creation Script
# Bu script Firebase CLI kullanarak index'leri otomatik oluşturur

echo "🔥 Firebase Firestore Index Setup"
echo "=================================="
echo ""

# Firebase CLI kurulu mu kontrol et
if ! command -v firebase &> /dev/null
then
    echo "❌ Firebase CLI bulunamadı!"
    echo "📦 Kurulum için: npm install -g firebase-tools"
    echo "🔐 Login için: firebase login"
    exit 1
fi

echo "✅ Firebase CLI bulundu"
echo ""

# firebase.json dosyası oluştur (eğer yoksa)
if [ ! -f "firebase.json" ]; then
    cat > firebase.json << 'EOF'
{
  "firestore": {
    "rules": "firestore.rules",
    "indexes": "firestore.indexes.json"
  }
}
EOF
    echo "📋 firebase.json oluşturuldu"
else
    echo "📋 firebase.json zaten mevcut"
fi
echo ""

# firestore.indexes.json dosyası oluştur
cat > firestore.indexes.json << 'EOF'
{
  "indexes": [
    {
      "collectionGroup": "blood_requests",
      "queryScope": "COLLECTION",
      "fields": [
        {
          "fieldPath": "status",
          "order": "ASCENDING"
        },
        {
          "fieldPath": "expires_at",
          "order": "ASCENDING"
        }
      ]
    },
    {
      "collectionGroup": "blood_requests",
      "queryScope": "COLLECTION",
      "fields": [
        {
          "fieldPath": "requester_uid",
          "order": "ASCENDING"
        },
        {
          "fieldPath": "created_at",
          "order": "DESCENDING"
        }
      ]
    },
    {
      "collectionGroup": "blood_requests",
      "queryScope": "COLLECTION",
      "fields": [
        {
          "fieldPath": "requester_uid",
          "order": "ASCENDING"
        },
        {
          "fieldPath": "status",
          "order": "ASCENDING"
        },
        {
          "fieldPath": "created_at",
          "order": "ASCENDING"
        }
      ]
    },
    {
      "collectionGroup": "users",
      "queryScope": "COLLECTION",
      "fields": [
        {
          "fieldPath": "is_donor",
          "order": "ASCENDING"
        },
        {
          "fieldPath": "city",
          "order": "ASCENDING"
        }
      ]
    }
  ],
  "fieldOverrides": []
}
EOF

echo "📝 firestore.indexes.json oluşturuldu"
echo ""

# firestore.rules dosyası oluştur
cat > firestore.rules << 'EOF'
rules_version = '2';
service cloud.firestore {
  match /databases/{database}/documents {
    // Backend-only access
    // Mobile clients cannot access Firestore directly
    // All operations must go through the Go backend API
    match /{document=**} {
      allow read, write: if false;
    }
  }
}
EOF

echo "🔐 firestore.rules oluşturuldu"
echo ""

echo "📤 Firebase'e deploy ediliyor..."
echo ""

# Firebase login kontrolü
echo "🔐 Firebase login durumu kontrol ediliyor..."
if ! firebase projects:list &> /dev/null; then
    echo "❌ Firebase'e giriş yapılmamış!"
    echo ""
    echo "Lütfen önce Firebase'e giriş yapın:"
    echo "   firebase login"
    echo ""
    exit 1
fi

echo "✅ Firebase login başarılı"
echo ""

# Mevcut projeleri listele
echo "📋 Erişilebilir Firebase projeleri:"
firebase projects:list
echo ""

# .env dosyasından project ID'yi oku
if [ -f ".env" ]; then
    PROJECT_ID=$(grep FIREBASE_PROJECT_ID .env | cut -d '=' -f2 | tr -d ' "'"'"'')
    if [ ! -z "$PROJECT_ID" ]; then
        echo "💡 .env dosyasında bulunan proje: $PROJECT_ID"
        echo ""
    fi
fi

read -p "Firebase project ID'nizi girin (varsayılan: $PROJECT_ID): " INPUT_PROJECT_ID

# Kullanıcı input vermişse onu kullan, yoksa .env'den okunanı kullan
if [ ! -z "$INPUT_PROJECT_ID" ]; then
    PROJECT_ID=$INPUT_PROJECT_ID
fi

if [ -z "$PROJECT_ID" ]; then
    echo "❌ Proje ID'si belirtilmedi!"
    exit 1
fi

echo ""
echo "🎯 Proje seçiliyor: $PROJECT_ID"
if ! firebase use $PROJECT_ID; then
    echo ""
    echo "❌ Proje seçilemedi! Lütfen yukarıdaki listeden geçerli bir proje ID'si girin."
    exit 1
fi

echo ""
echo "🚀 Index'ler ve rules deploy ediliyor..."
if ! firebase deploy --only firestore; then
    echo ""
    echo "❌ Deploy başarısız oldu!"
    exit 1
fi

echo ""
echo "✅ Firestore setup tamamlandı!"
echo ""
echo "🔍 Index'leri kontrol etmek için:"
echo "   https://console.firebase.google.com/project/$PROJECT_ID/firestore/indexes"
echo ""
