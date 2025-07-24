func CalculateUserTopicAccuracy(db *gorm.DB, userID uuid.UUID) (map[string]float64, error) {
	type Result struct {
		Topic   string
		Total   int64
		Correct int64
	}

	var results []Result
	// Single query with JOIN and aggregation
	if err := db.
		Table("question_attempts").
		Joins("JOIN questions ON questions.id = question_attempts.question_id").
		Select(`
			questions.topic AS topic,
			COUNT(*) AS total,
			SUM(CASE WHEN question_attempts.is_correct THEN 1 ELSE 0 END) AS correct
		`).
		Where("question_attempts.user_id = ?", userID).
		Group("questions.topic").
		Scan(&results).Error; err != nil {
		return nil, err
	}

	// Convert to map of topic -> accuracy percentage
	accuracies := make(map[string]float64)
	for _, r := range results {
		accuracies[r.Topic] = (float64(r.Correct) / float64(r.Total)) * 100
	}

	return accuracies, nil
}
