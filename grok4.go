func CalculateUserTopicAccuracy(db *gorm.DB, userID uuid.UUID) (map[string]float64, error) {
	// Define a temporary struct to hold the aggregation results
	type Result struct {
		Topic   string
		Total   int64
		Correct int64
	}

	var results []Result

	// Perform a join between QuestionAttempt and Question, grouping by Topic
	err := db.Table("question_attempts").
		Select("questions.topic, COUNT(*) as total, SUM(CASE WHEN question_attempts.is_correct THEN 1 ELSE 0 END) as correct").
		Joins("JOIN questions ON questions.id = question_attempts.question_id").
		Where("question_attempts.user_id = ?", userID).
		Group("questions.topic").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	accuracies := make(map[string]float64)
	for _, res := range results {
		if res.Total > 0 {
			accuracies[res.Topic] = (float64(res.Correct) / float64(res.Total)) * 100
		}
	}

	return accuracies, nil
}
