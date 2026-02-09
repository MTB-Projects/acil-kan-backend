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

// GenerateTopicName generates a topic name based on blood type, city, and optional district
// Format: blood_<BLOOD_TYPE>_<CITY>_<DISTRICT> (if district provided)
//         blood_<BLOOD_TYPE>_<CITY> (if no district)
// Example: blood_APlus_Istanbul_Kadikoy, blood_APlus_Istanbul
func GenerateTopicName(bloodType model.BloodType, city string, district ...string) string {
	// Replace + with Plus for topic name (Firebase topic names can't have +)
	cleanBloodType := strings.ReplaceAll(string(bloodType), "+", "Plus")
	cleanBloodType = strings.ReplaceAll(cleanBloodType, "-", "Minus")
	cleanCity := strings.ReplaceAll(city, " ", "_")
	
	topicName := fmt.Sprintf("blood_%s_%s", cleanBloodType, cleanCity)
	
	// Add district if provided
	if len(district) > 0 && district[0] != "" {
		cleanDistrict := strings.ReplaceAll(district[0], " ", "_")
		topicName = fmt.Sprintf("%s_%s", topicName, cleanDistrict)
	}
	
	return topicName
}

// SubscribeToTopics subscribes a user's FCM token to relevant topics
// Uses hierarchical topic structure for better targeting:
// 1. Specific: blood_APlus_Istanbul_Kadikoy (district-level)
// 2. City-wide: blood_APlus_Istanbul (city-level for same blood type)
// 3. General: city_Istanbul (all blood requests in city)
func (f *FCMClient) SubscribeToTopics(ctx context.Context, fcmToken string, bloodType model.BloodType, city, district string) error {
	if fcmToken == "" {
		return fmt.Errorf("FCM token is required")
	}

	topics := []string{
		fmt.Sprintf("city_%s", strings.ReplaceAll(city, " ", "_")), // General city topic
		GenerateTopicName(bloodType, city),                          // City-wide blood type topic
	}
	
	// Add district-specific topic if district is provided
	if district != "" {
		topics = append(topics, GenerateTopicName(bloodType, city, district))
	}

	for _, topic := range topics {
		_, err := f.client.SubscribeToTopic(ctx, []string{fcmToken}, topic)
		if err != nil {
			logger.Error("Failed to subscribe to topic",
				zap.String("topic", topic),
				zap.Error(err),
			)
			return err
		}
		logger.Info("Subscribed to topic",
			zap.String("topic", topic),
		)
	}

	return nil
}

// UnsubscribeFromTopics unsubscribes a user from all topics
func (f *FCMClient) UnsubscribeFromTopics(ctx context.Context, fcmToken string, bloodType model.BloodType, city, district string) error {
	if fcmToken == "" {
		return fmt.Errorf("FCM token is required")
	}

	topics := []string{
		fmt.Sprintf("city_%s", strings.ReplaceAll(city, " ", "_")),
		GenerateTopicName(bloodType, city),
	}
	
	// Add district-specific topic if district is provided
	if district != "" {
		topics = append(topics, GenerateTopicName(bloodType, city, district))
	}

	for _, topic := range topics {
		_, err := f.client.UnsubscribeFromTopic(ctx, []string{fcmToken}, topic)
		if err != nil {
			logger.Error("Failed to unsubscribe from topic",
				zap.String("topic", topic),
				zap.Error(err),
			)
			return err
		}
		logger.Info("Unsubscribed from topic",
			zap.String("topic", topic),
		)
	}

	return nil
}

// SendBloodRequestNotificationToTopic sends notification to a topic
// This is much more scalable than sending to individual tokens
// Sends to the most specific topic available (district-level if district is set, otherwise city-level)
func (f *FCMClient) SendBloodRequestNotificationToTopic(ctx context.Context, request *model.BloodRequest) error {
	// Use district-specific topic if available, otherwise city-wide topic
	var topic string
	if request.District != "" {
		topic = GenerateTopicName(request.BloodType, request.City, request.District)
	} else {
		topic = GenerateTopicName(request.BloodType, request.City)
	}

	message := &messaging.Message{
		Topic: topic,
		Notification: &messaging.Notification{
			Title: fmt.Sprintf("Acil %s Kan İhtiyacı!", request.BloodType),
			Body:  fmt.Sprintf("%s - %s hastanesinde kan ihtiyacı var", request.City, request.HospitalName),
		},
		Data: map[string]string{
			"type":             "blood_request",
			"request_id":       request.ID,
			"blood_type":       string(request.BloodType),
			"city":             request.City,
			"hospital_name":    request.HospitalName,
			"hospital_address": request.HospitalAddress,
			"contact_phone":    request.ContactPhone,
		},
		Android: &messaging.AndroidConfig{
			Priority: "high",
		},
		APNS: &messaging.APNSConfig{
			Headers: map[string]string{
				"apns-priority": "10",
			},
		},
	}

	_, err := f.client.Send(ctx, message)
	if err != nil {
		logger.Error("Failed to send topic notification",
			zap.String("topic", topic),
			zap.Error(err),
		)
		return err
	}

	logger.Info("Blood request notification sent to topic",
		zap.String("topic", topic),
		zap.String("request_id", request.ID),
	)

	return nil
}

// SendBloodRequestNotification sends a notification about a new blood request to individual token
// Deprecated: Use SendBloodRequestNotificationToTopic for better scalability
func (f *FCMClient) SendBloodRequestNotification(ctx context.Context, fcmToken string, request *model.BloodRequest) error {
	message := &messaging.Message{
		Token: fcmToken,
		Notification: &messaging.Notification{
			Title: fmt.Sprintf("Acil %s Kan İhtiyacı!", request.BloodType),
			Body:  fmt.Sprintf("%s - %s hastanesinde kan ihtiyacı var", request.City, request.HospitalName),
		},
		Data: map[string]string{
			"type":             "blood_request",
			"request_id":       request.ID,
			"blood_type":       string(request.BloodType),
			"city":             request.City,
			"hospital_name":    request.HospitalName,
			"hospital_address": request.HospitalAddress,
			"contact_phone":    request.ContactPhone,
		},
		Android: &messaging.AndroidConfig{
			Priority: "high",
		},
		APNS: &messaging.APNSConfig{
			Headers: map[string]string{
				"apns-priority": "10",
			},
		},
	}

	_, err := f.client.Send(ctx, message)
	return err
}
