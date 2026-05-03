package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"gin-quickstart/internal/repository"
)

type EventQueueService interface {
	Enqueue(ctx context.Context, event *QueuedEvent) error
	ProcessEvents(ctx context.Context)
	GetQueueLength() int64
	GetProcessedCount() int64
}

type QueuedEvent struct {
	UserID    string                 `json:"user_id"`
	EventType string                 `json:"event_type"`
	SongID    string                 `json:"song_id,omitempty"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp time.Time              `json:"timestamp"`
}

type EventQueueServiceImpl struct {
	redisClient   *redis.Client
	eventRepo     repository.EventRepository
	subRepo       repository.SubscriptionRepository
	musicRepo     repository.MusicRepository
	queueName     string
	processedCount int64
}

func NewEventQueueService(redisClient *redis.Client, eventRepo repository.EventRepository,
	subRepo repository.SubscriptionRepository, musicRepo repository.MusicRepository) EventQueueService {

	return &EventQueueServiceImpl{
		redisClient: redisClient,
		eventRepo:   eventRepo,
		subRepo:     subRepo,
		musicRepo:   musicRepo,
		queueName:   "event_queue",
	}
}

func (s *EventQueueServiceImpl) Enqueue(ctx context.Context, event *QueuedEvent) error {
	event.Timestamp = time.Now()

	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	return s.redisClient.LPush(ctx, s.queueName, eventBytes).Err()
}

func (s *EventQueueServiceImpl) ProcessEvents(ctx context.Context) {
	log.Println("Starting event queue processor...")

	if s.redisClient == nil {
		log.Println("Event queue processor aborted: Redis client is nil")
		return
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping event queue processor...")
			return
		default:
			// Get event from queue
			result := s.redisClient.BRPop(ctx, 0, s.queueName)
			if result.Err() != nil {
				log.Printf("Error getting event from queue: %v", result.Err())
				continue
			}

			eventBytes := result.Val()[1]
			var event QueuedEvent
			if err := json.Unmarshal([]byte(eventBytes), &event); err != nil {
				log.Printf("Failed to unmarshal event: %v", err)
				continue
			}

			// Process the event
			if err := s.processEvent(ctx, &event); err != nil {
				log.Printf("Failed to process event: %v", err)
				// Could implement retry logic here
				continue
			}

			s.processedCount++

			log.Printf("Successfully processed event: %s for user %s", event.EventType, event.UserID)
		}
	}
}

func (s *EventQueueServiceImpl) processEvent(ctx context.Context, event *QueuedEvent) error {
	// Ensure subscription exists
	if err := s.subRepo.Create(ctx, event.UserID); err != nil {
		return fmt.Errorf("failed to ensure subscription: %w", err)
	}

	// Increment version in subscriptions table and get the NEW version atomically.
	// This ensures we always have a unique version for the (aggregate_id, version) constraint.
	newVersion, err := s.subRepo.IncrementVersion(ctx, event.UserID)
	if err != nil {
		return fmt.Errorf("failed to increment version: %w", err)
	}

	// Create event record
	eventRecord := &repository.Event{
		AggregateID: event.UserID,
		UserID:      event.UserID,
		EventType:   event.EventType,
		Version:     newVersion,
		Payload:     event.Payload,
	}

	if err := s.eventRepo.Create(ctx, eventRecord); err != nil {
		return fmt.Errorf("failed to create event: %w", err)
	}

	// Update subscription projection based on event type
	switch event.EventType {
	case "PlanUpgraded":
		price := 199
		if p, ok := event.Payload["price"].(float64); ok {
			price = int(p)
		}
		periodEnd := time.Now().AddDate(0, 1, 0) // +1 month
		if err := s.subRepo.Update(ctx, event.UserID, "premium", "active", price, &periodEnd, false); err != nil {
			return fmt.Errorf("failed to update subscription: %w", err)
		}
	case "PlanDowngraded":
		// Pro-rata billing: calculate days used out of 30-day billing cycle.
		sub, _ := s.subRepo.GetByUserID(ctx, event.UserID)
		var proRataCharge int
		if sub != nil && sub.Plan == "premium" {
			total := sub.CurrentPeriodEnd.Sub(sub.StartedAt)
			used := time.Since(sub.StartedAt)
			if total > 0 && used > 0 {
				fraction := used.Seconds() / total.Seconds()
				if fraction > 1 { fraction = 1 }
				proRataCharge = int(fraction * float64(sub.Price))
			}
		}
		// Store charged amount in event payload for audit
		eventRecord.Payload["pro_rata_charge"] = proRataCharge
		
		// Immediate Downgrade: Change to free plan immediately and collect pro-rata revenue
		if err := s.subRepo.Update(ctx, event.UserID, "free", "active", proRataCharge, nil, false); err != nil {
			return fmt.Errorf("failed to downgrade subscription: %w", err)
		}
		log.Printf("Downgraded user %s to free. Pro-rata charge: ₹%d", event.UserID, proRataCharge)
	case "SubscriptionPaused":
		sub, err := s.subRepo.GetByUserID(ctx, event.UserID)
		if err != nil {
			return fmt.Errorf("failed to get subscription for pause: %w", err)
		}
		// Calculate exact days remaining
		daysRemaining := int(time.Until(sub.CurrentPeriodEnd).Hours() / 24)
		if daysRemaining < 0 {
			daysRemaining = 0
		}
		eventRecord.Payload["days_remaining"] = daysRemaining
		if err := s.subRepo.PauseSubscription(ctx, event.UserID, daysRemaining); err != nil {
			return fmt.Errorf("failed to pause subscription: %w", err)
		}
		log.Printf("Paused subscription for user %s with %d days remaining", event.UserID, daysRemaining)
	case "SubscriptionResumed":
		sub, err := s.subRepo.GetByUserID(ctx, event.UserID)
		if err != nil {
			return fmt.Errorf("failed to get subscription for resume: %w", err)
		}
		if sub.PauseDaysRemaining == nil {
			return fmt.Errorf("subscription is not paused")
		}
		newPeriodEnd := time.Now().AddDate(0, 0, *sub.PauseDaysRemaining)
		eventRecord.Payload["days_remaining"] = *sub.PauseDaysRemaining
		eventRecord.Payload["new_period_end"] = newPeriodEnd.Format(time.RFC3339)
		if err := s.subRepo.ResumeSubscription(ctx, event.UserID, newPeriodEnd); err != nil {
			return fmt.Errorf("failed to resume subscription: %w", err)
		}
		log.Printf("Resumed subscription for user %s, new end: %s", event.UserID, newPeriodEnd.Format("2006-01-02"))
	case "SongPlayed", "PLAY_TRACK", "SongPaused", "SongNext", "SongPrevious":
		if event.SongID != "" {
			if err := s.musicRepo.RecordPlay(ctx, event.SongID, event.UserID, event.Payload); err != nil {
				log.Printf("Failed to record play/pause event %s: %v", event.EventType, err)
			}
		}
	case "SongLiked":
		if event.SongID != "" {
			var liked repository.LikedSong
			liked.ID = event.SongID
			if t, ok := event.Payload["title"].(string); ok { liked.Title = t }
			if a, ok := event.Payload["artist"].(string); ok { liked.Artist = a }
			if al, ok := event.Payload["album"].(string); ok { liked.Album = al }
			if d, ok := event.Payload["duration"].(string); ok { liked.Duration = d }
			if aw, ok := event.Payload["artwork"].(string); ok { liked.Artwork = aw }
			
			if err := s.musicRepo.LikeSong(ctx, event.UserID, event.SongID, liked); err != nil {
				log.Printf("Failed to record like: %v", err)
			}
		}
	case "SongUnliked":
		if event.SongID != "" {
			if err := s.musicRepo.UnlikeSong(ctx, event.UserID, event.SongID); err != nil {
				log.Printf("Failed to record unlike: %v", err)
			}
		}
	}

	log.Printf("Processed event: %s for user %s", event.EventType, event.UserID)
	return nil
}

func (s *EventQueueServiceImpl) GetQueueLength() int64 {
	ctx := context.Background()
	length, err := s.redisClient.LLen(ctx, s.queueName).Result()
	if err != nil {
		log.Printf("Failed to get queue length: %v", err)
		return 0
	}
	return length
}

func (s *EventQueueServiceImpl) GetProcessedCount() int64 {
	return s.processedCount
}