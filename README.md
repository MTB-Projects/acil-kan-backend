# Emergency Blood Donation Backend API

Production-ready backend API for emergency blood donation mobile application built with Go, Firebase, and Firestore.

## 🏗️ Architecture

### Core Principles
- **Firebase Authentication**: Identity provider only - all business logic in backend
- **Backend-Driven**: All validation, authorization, and data control on server
- **Security First**: Never trust the client, production-level security
- **Scalable Design**: Optimized for correctness, safety, and scalability

### Tech Stack
- **Language**: Go 1.24
- **Framework**: Fiber (Express-like web framework)
- **Authentication**: Firebase Authentication
- **Database**: Firestore
- **Notifications**: Firebase Cloud Messaging (FCM)
- **Logging**: Uber Zap (structured logging)

## 📁 Project Structure

```
acil-kan-backend/
├── cmd/
│   └── api/
│       └── main.go              # Application entry point
├── config/
│   └── config.go                # Configuration management
├── internal/
│   ├── auth/
│   │   └── middleware.go        # Firebase token verification
│   ├── handler/
│   │   ├── user_handler.go      # User endpoints
│   │   └── request_handler.go   # Blood request endpoints
│   ├── model/
│   │   ├── user.go              # User domain model
│   │   └── blood_request.go     # Blood request model
│   ├── repository/
│   │   ├── user_repo.go         # User data access
│   │   └── blood_request_repo.go # Blood request data access
│   ├── router/
│   │   └── router.go            # API route definitions
│   ├── service/
│   │   ├── user_service.go      # User business logic
│   │   └── donation_service.go  # Donation business logic
│   └── notification/
│       └── fcm_client.go        # FCM notification client
├── pkg/
│   ├── errors/
│   │   └── errors.go            # Error codes and handling
│   ├── logger/
│   │   └── logger.go            # Structured logging
│   └── response/
│       └── response.go          # Standard API responses
├── .env.example                 # Environment variables template
├── .gitignore
├── go.mod
├── go.sum
├── Makefile                     # Build and run commands
└── README.md
```

## 🚀 Getting Started

### Prerequisites
- Go 1.24+
- Firebase project with:
  - Firebase Authentication enabled
  - Firestore database created
  - Service account key (JSON file)

### Installation

1. **Clone the repository**
```bash
git clone <repository-url>
cd acil-kan-backend
```

2. **Install dependencies**
```bash
go mod download
```

3. **Configure environment variables**
```bash
cp .env.example .env
```

Edit `.env` with your Firebase credentials:
```env
PORT=8080
ENV=development
FIREBASE_PROJECT_ID=your-project-id
GOOGLE_APPLICATION_CREDENTIALS=./serviceAccountKey.json
MAX_REQUESTS_PER_USER_PER_HOUR=5
```

4. **Add Firebase service account key**
- Download service account key from Firebase Console
- Save as `serviceAccountKey.json` in project root

### Running the Application

**Development mode:**
```bash
make run
```

**With live reload (requires Air):**
```bash
make watch
```

**Build only:**
```bash
make build
```

## 📡 API Endpoints

### Health Check
```
GET /health
```
Returns API health status (no authentication required)

### Public Endpoints

#### Get Active Blood Requests
```
GET /api/v1/public/requests
```
Returns all active blood donation requests

### Protected Endpoints (Require Authentication)

All protected endpoints require `Authorization: Bearer <firebase-token>` header.

#### User Management

**Get User Profile**
```
GET /api/v1/users/me
```
Returns or creates user profile on first login

**Update User Profile**
```
PUT /api/v1/users/me
Content-Type: application/json

{
  "full_name": "John Doe",
  "phone_number": "+905551234567",
  "blood_type": "A+",
  "city": "Istanbul",
  "is_donor": true
}
```

**Update FCM Token**
```
POST /api/v1/users/fcm-token
Content-Type: application/json

{
  "token": "fcm-device-token"
}
```

#### Blood Requests

**Create Blood Request**
```
POST /api/v1/requests
Content-Type: application/json

{
  "patient_name": "Jane Doe",
  "blood_type": "A+",
  "city": "Istanbul",
  "hospital_name": "City Hospital",
  "hospital_address": "123 Main St",
  "contact_phone": "+905551234567",
  "units_needed": 2,
  "description": "Urgent need for surgery"
}
```

**Get My Requests**
```
GET /api/v1/requests/my
```

**Cancel Request**
```
DELETE /api/v1/requests/:id
```

## 🔒 Security Features

- **Firebase Token Verification**: All protected endpoints verify Firebase ID tokens
- **Rate Limiting**: Maximum 3 blood requests per user per day
- **Authorization Checks**: Users can only modify their own requests
- **Input Validation**: Comprehensive validation on all inputs
- **Structured Logging**: All actions logged with user context
- **Error Handling**: Standardized error codes and messages

## 📊 Error Codes

Errors follow a structured format:

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable message"
  },
  "timestamp": "2026-02-09T17:00:00Z"
}
```

### Error Code Categories
- `AUTH_1xxx`: Authentication/Authorization errors
- `USER_2xxx`: User-related errors
- `REQUEST_3xxx`: Blood request errors
- `VALIDATION_4xxx`: Validation errors
- `DB_5xxx`: Database errors
- `NOTIFICATION_6xxx`: Notification errors
- `GENERAL_9xxx`: General errors

## 🎯 Business Rules

### User Management
- User profile created automatically on first login
- Profile must be complete (city, phone) before creating requests
- FCM token stored for push notifications

### Blood Requests
- Maximum 3 active requests per user per day
- Requests auto-expire after 48 hours
- Only requester can cancel their own requests
- Compatible donors notified automatically

### Notifications
- Backend determines notification recipients
- Blood type compatibility checked
- Location-based matching (same city)
- Donor availability verified (can donate every 90 days)

## 🧪 Testing

```bash
make test
```

## 📝 Logging

Application uses structured logging with Zap:
- Request/response logging with trace IDs
- User action tracking
- Database operation monitoring
- Notification delivery tracking
- Error tracking with context

## 🛠️ Development

### Code Organization
- **Handlers**: HTTP request/response handling
- **Services**: Business logic
- **Repositories**: Data access layer
- **Models**: Domain entities
- **Middleware**: Cross-cutting concerns

### Adding New Endpoints
1. Define route in `internal/router/router.go`
2. Create handler in `internal/handler/`
3. Implement business logic in `internal/service/`
4. Add repository methods if needed
5. Update error codes in `pkg/errors/errors.go`

## 📦 Deployment

The application is designed for cloud deployment:
- Docker-ready
- Environment-based configuration
- Graceful shutdown support
- Health check endpoint for load balancers

## 📄 License

[Your License Here]

## 👥 Contributors

[Your Team Here]