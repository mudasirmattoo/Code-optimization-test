package main

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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

// CalculateUserTopicAccuracy returns a map[topic]accuracy%
// using a single SQL query that joins attempts with questions
// and aggregates the results.
func CalculateUserTopicAccuracy(db *gorm.DB, userID uuid.UUID) (map[string]float64, error) {
	type Result struct {
		Topic    string
		Total    int
		Correct  int
		Accuracy float64
	}

	var results []Result

	err := db.
		Model(&QuestionAttempt{}).
		Select(`
			questions.topic                                                AS topic,
			COUNT(*)                                                       AS total,
			SUM(CASE WHEN question_attempts.is_correct THEN 1 ELSE 0 END)  AS correct,
			ROUND(
				CAST(SUM(CASE WHEN question_attempts.is_correct THEN 1 ELSE 0 END) AS REAL) * 100.0 / COUNT(*),
				2
			)                                                              AS accuracy
		`).
		Joins("JOIN questions ON questions.id = question_attempts.question_id").
		Where("question_attempts.user_id = ?", userID).
		Group("questions.topic").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	accuracies := make(map[string]float64, len(results))
	for _, r := range results {
		accuracies[r.Topic] = r.Accuracy
	}
	return accuracies, nil
}

func main() {
	db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	db.AutoMigrate(&Question{}, &QuestionAttempt{})

	userID := uuid.New()
	questions := []Question{
		{ID: 1, Topic: "Algebra"}, {ID: 2, Topic: "Calculus"}, {ID: 3, Topic: "Algebra"},
	}
	db.Create(&questions)
	attempts := []QuestionAttempt{
		{UserID: userID, QuestionID: 1, IsCorrect: true},
		{UserID: userID, QuestionID: 1, IsCorrect: false},
		{UserID: userID, QuestionID: 2, IsCorrect: true},
		{UserID: userID, QuestionID: 3, IsCorrect: true},
	}
	db.Create(&attempts)

	accuracies, _ := CalculateUserTopicAccuracy(db, userID)
	fmt.Printf("Accuracies by Topic: %v\\n", accuracies)
}
