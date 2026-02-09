# Değişkenler
APP_NAME=emergency-blood-api
CMD_PATH=./cmd/api/main.go
BUILD_PATH=./build/${APP_NAME}

# .env dosyasını yükle (eğer varsa)
ifneq ("$(wildcard .env)","")
    include .env
    export
endif

.PHONY: help build run watch clean test migrate-up migrate-down

help: ## Mevcut komutları listeler
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Uygulamayı derler
	@echo "Derleniyor..."
	@go build -o ${BUILD_PATH} ${CMD_PATH}

run: build ## Uygulamayı derler ve çalıştırır
	@${BUILD_PATH}

watch: ## Air (live reload) ile uygulamayı sıcak modda çalıştırır
	@air

test: ## Testleri çalıştırır
	@go test -v ./internal/...

clean: ## Binary dosyasını siler
	@rm -f ${BUILD_PATH}
	@echo "Temizlendi."

deps: ## Bağımlılıkları indirir ve düzenler
	@go mod tidy
	@go mod download

# Firestore İşlemleri
init-firestore: ## Firestore bağlantısını test eder ve bilgi verir
	@echo "Firestore bağlantısı test ediliyor..."
	@go run scripts/init_firestore.go

setup-firestore: ## Firestore index'lerini ve rules'ları deploy eder (Firebase CLI gerekli)
	@echo "Firestore setup başlatılıyor..."
	@./scripts/setup_firestore.sh

# Test İşlemleri
generate-token: ## Test için Firebase token oluşturur (kullanım: make generate-token UID=test-user-123)
	@echo "Test token oluşturuluyor..."
	@go run scripts/generate_test_token.go $(UID)

test-health: ## Health endpoint'ini test eder
	@echo "Testing health endpoint..."
	@curl -s http://localhost:8080/health | jq '.' || curl http://localhost:8080/health

test-public: ## Public endpoint'leri test eder
	@echo "Testing public blood requests endpoint..."
	@curl -s http://localhost:8080/api/v1/public/requests | jq '.' || curl http://localhost:8080/api/v1/public/requests


# Veritabanı Migration İşlemleri (Eğer 'golang-migrate' kullanıyorsan)
migrate-up: ## Veritabanı tablolarını oluşturur
	migrate -path scripts/migrations -database "$(DB_URL)" up

migrate-down: ## Veritabanı tablolarını geri alır
	migrate -path scripts/migrations -database "$(DB_URL)" down

docker-up: ## Docker konteynerlarını başlatır
	docker-compose up -d

docker-down: ## Docker konteynerlarını durdurur
	docker-compose down

# Proje başlangıç kurulumu (Klasör yapısını ve boş dosyaları oluşturur)
init:
	@echo "Proje klasör yapısı oluşturuluyor..."
	mkdir -p cmd/api config internal/auth internal/handler internal/repository internal/service internal/model internal/notification pkg/utils scripts/migrations

	@echo "Dosyalar oluşturuluyor..."
	touch cmd/api/main.go
	touch config/config.go
	touch internal/auth/middleware.go
	touch internal/handler/user_handler.go
	touch internal/handler/request_handler.go
	touch internal/repository/user_repo.go
	touch internal/repository/blood_request_repo.go
	touch internal/service/user_service.go
	touch internal/service/donation_service.go
	touch internal/model/user.go
	touch internal/model/blood_request.go
	touch internal/notification/fcm_client.go
	touch pkg/utils/validator.go
	touch .env
	touch .gitignore

	@echo "Başarıyla tamamlandı. Klasör yapısı hazır."