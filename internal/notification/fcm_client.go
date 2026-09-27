package notification

import (
	"context"
	"fmt"
	"strings"

	"acilkan.backend/internal/model"
	"acilkan.backend/pkg/logger"
	"firebase.google.com/go/v4/messaging"
	"go.uber.org/zap"
)

type FCMClient struct {
	client *messaging.Client
}

func NewFCMClient(client *messaging.Client) *FCMClient {
	return &FCMClient{client: client}
}

var turkishReplacer = strings.NewReplacer(
	"ç", "c", "Ç", "c",
	"ğ", "g", "Ğ", "g",
	"ı", "i", "I", "i", "İ", "i",
	"ö", "o", "Ö", "o",
	"ş", "s", "Ş", "s",
	"ü", "u", "Ü", "u",
)

// Slug converts a place name into a lowercase ASCII token that is valid in an
// FCM topic name ([a-zA-Z0-9-_.~%]+). "İstanbul" and "istanbul" map to the same slug.
func Slug(s string) string {
	s = turkishReplacer.Replace(strings.TrimSpace(s))
	s = strings.ToLower(s)

	var b strings.Builder
	lastUnderscore := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastUnderscore = false
		} else if !lastUnderscore && b.Len() > 0 {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	return strings.TrimSuffix(b.String(), "_")
}

func bloodTypeToken(bloodType model.BloodType) string {
	t := strings.ReplaceAll(string(bloodType), "+", "pos")
	return strings.ToLower(strings.ReplaceAll(t, "-", "neg"))
}

// GenerateTopicName returns the city-level topic for donors of a blood type.
// Example: ("A+", "İstanbul") -> "blood_apos_istanbul"
func GenerateTopicName(bloodType model.BloodType, city string) string {
	return fmt.Sprintf("blood_%s_%s", bloodTypeToken(bloodType), Slug(city))
}

// CityTopicName returns the topic every donor in a city is subscribed to.
func CityTopicName(city string) string {
	return fmt.Sprintf("city_%s", Slug(city))
}

func donorTopics(bloodType model.BloodType, city string) []string {
	return []string{CityTopicName(city), GenerateTopicName(bloodType, city)}
}

// SubscribeToTopics subscribes a donor's FCM token to the city topic and
// their own blood type topic in that city.
func (f *FCMClient) SubscribeToTopics(ctx context.Context, fcmToken string, bloodType model.BloodType, city string) error {
	if fcmToken == "" {
		return fmt.Errorf("FCM token is required")
	}

	for _, topic := range donorTopics(bloodType, city) {
		if _, err := f.client.SubscribeToTopic(ctx, []string{fcmToken}, topic); err != nil {
			logger.Error("Failed to subscribe to topic", zap.String("topic", topic), zap.Error(err))
			return err
		}
		logger.Info("Subscribed to topic", zap.String("topic", topic))
	}
	return nil
}

// UnsubscribeFromTopics removes a token from the topics SubscribeToTopics added
func (f *FCMClient) UnsubscribeFromTopics(ctx context.Context, fcmToken string, bloodType model.BloodType, city string) error {
	if fcmToken == "" {
		return fmt.Errorf("FCM token is required")
	}

	for _, topic := range donorTopics(bloodType, city) {
		if _, err := f.client.UnsubscribeFromTopic(ctx, []string{fcmToken}, topic); err != nil {
			logger.Error("Failed to unsubscribe from topic", zap.String("topic", topic), zap.Error(err))
			return err
		}
		logger.Info("Unsubscribed from topic", zap.String("topic", topic))
	}
	return nil
}

// RecipientTopics returns the topics that should receive a request:
// the city-level topic of every donor blood type compatible with the request.
func RecipientTopics(request *model.BloodRequest) []string {
	var topics []string
	for _, donorType := range model.CompatibleDonorTypes(request.BloodType) {
		topics = append(topics, GenerateTopicName(donorType, request.City))
	}
	return topics
}

// SendBloodRequestNotificationToTopic notifies every compatible donor in the request's city.
// Topic payloads can be read by any subscriber, so no personal data is included;
// the app fetches contact details from the authenticated API.
func (f *FCMClient) SendBloodRequestNotificationToTopic(ctx context.Context, request *model.BloodRequest) error {
	var messages []*messaging.Message
	for _, topic := range RecipientTopics(request) {
		messages = append(messages, &messaging.Message{
			Topic: topic,
			Notification: &messaging.Notification{
				Title: fmt.Sprintf("Acil %s Kan İhtiyacı!", request.BloodType),
				Body:  fmt.Sprintf("%s - %s hastanesinde kan ihtiyacı var", request.City, request.HospitalName),
			},
			Data: map[string]string{
				"type":          "blood_request",
				"request_id":    request.ID,
				"blood_type":    string(request.BloodType),
				"city":          request.City,
				"hospital_name": request.HospitalName,
			},
			Android: &messaging.AndroidConfig{
				Priority: "high",
			},
			APNS: &messaging.APNSConfig{
				Headers: map[string]string{
					"apns-priority": "10",
				},
			},
		})
	}

	resp, err := f.client.SendEach(ctx, messages)
	if err != nil {
		logger.Error("Failed to send topic notifications", zap.String("request_id", request.ID), zap.Error(err))
		return err
	}

	for i, r := range resp.Responses {
		if !r.Success {
			logger.Error("Topic notification failed",
				zap.String("topic", messages[i].Topic),
				zap.String("request_id", request.ID),
				zap.Error(r.Error),
			)
		}
	}
	if resp.FailureCount == len(messages) {
		return fmt.Errorf("all %d topic notifications failed", len(messages))
	}

	logger.Info("Blood request notifications sent",
		zap.String("request_id", request.ID),
		zap.Int("topics", len(messages)),
		zap.Int("failed", resp.FailureCount),
	)
	return nil
}
