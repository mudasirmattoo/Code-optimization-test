package main

import (
	"[github.com/google/uuid](http://github.com/google/uuid)"
	"[gorm.io/gorm](http://gorm.io/gorm)"
)

type Question struct {
	ID    uint   `gorm:"primaryKey"`
	Topic string `gorm:"size:100;index"`
}

type QuestionAttempt struct {
	ID         uint      `gorm:"primaryKey"`
	UserID     uuid.UUID `gorm:"type:uuid;not null;index"`
	QuestionID uint      `gorm:"not null;index"`
	Question   Question  `gorm:"foreignKey:QuestionID"`
	IsCorrect  bool
}

func CalculateUserTopicAccuracy(db *gorm.DB, userID uuid.UUID) (map[string]float64, error) {
	var attempts []QuestionAttempt
	if err := db.Where("user_id = ?", userID).Find(&attempts).Error; err != nil {
		return nil, err
	}
	topicStats := make(map[string]map[string]int)

	for _, attempt := range attempts {
		var question Question
		// Inefficient: Triggers a separate DB query for each attempt in the loop.
		db.First(&question, attempt.QuestionID)

		if _, ok := topicStats[question.Topic]; !ok {
			topicStats[question.Topic] = map[string]int{"total": 0, "correct": 0}
		}

		topicStats[question.Topic]["total"]++
		if attempt.IsCorrect {
			topicStats[question.Topic]["correct"]++
		}
	}

	accuracies := make(map[string]float64)
	for topic, stats := range topicStats {
		if stats["total"] > 0 {
			accuracies[topic] = (float64(stats["correct"]) / float64(stats["total"])) * 100
		}
	}

	return accuracies, nil
}
