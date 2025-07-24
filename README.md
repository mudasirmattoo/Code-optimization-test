# Code-optimization-test
**Common Prompt along with above code:**

“The attached Go function is inefficient. It's slow because it makes a separate database query for every single item inside a loop. Please rewrite it to be highly efficient. The new version should use a single, smart database query to perform all the calculations and return just the final statistics.”

**Kimi K2’s Response:**

Here’s a fully-optimized version that collapses everything into **one SQL query** and returns the per-topic accuracy directly from the database, eliminating every iteration-side lookup.

```go
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
		Topic      string
		Total      int
		Correct    int
		Accuracy   float64
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

```

Key points

1. One query joins `question_attempts` and `questions`.
2. SQL aggregates (`COUNT`, `SUM`) compute total and correct counts per topic in the database.
3. Accuracy is calculated directly in SQL with `ROUND(..., 2)` for two-decimal precision.
4. Go code only maps the returned rows into the final `map[string]float64`.

**Qwen3-235B’s Response:**

You're absolutely right — the original `CalculateUserTopicAccuracy` function is inefficient because it performs **N+1 queries**: one to get the attempts, and then one `db.First()` query **per attempt** to fetch the associated `Question` just to get the `Topic`.

We can optimize this dramatically using a **single SQL query** with a **JOIN**, leveraging GORM's ability to perform aggregations and group results.

Here’s the **highly efficient version** of `CalculateUserTopicAccuracy` that uses **only one database query**:

```go
func CalculateUserTopicAccuracy(db *gorm.DB, userID uuid.UUID) (map[string]float64, error) {
	type Result struct {
		Topic    string
		Total    int64
		Correct  int64
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

```

---

### ✅ Key Improvements:

- **Single Query**: Uses `JOIN` between `question_attempts` and `questions` to get topic info directly.
- **Aggregation in DB**: Uses `COUNT` and `SUM(CASE ...)` to compute totals and correct counts **per topic**.
- **Grouping**: Groups by `topic` so we get one row per topic.
- **No Loop Queries**: Eliminates all `db.First()` calls inside loops.
- **Efficient Memory Use**: Only loads final aggregated data into Go.

---

### 📌 Notes:

- Assumes your table names are pluralized as `"question_attempts"` and `"questions"` (GORM default).
- If you've customized table names via `TableName()` method, ensure they match.
- The `Scan(&results)` is used because we're selecting aggregated fields, not full models.

---

### Example Output (with your test data):

```go
Accuracies by Topic: map[Algebra:66.66666666666666 Calculus:100]

```

Which is correct:

- **Algebra**: 2 attempts, 1 correct → 50%? Wait — let's double-check your test data:

```go
attempts := []QuestionAttempt{
    {UserID: userID, QuestionID: 1, IsCorrect: true},    // Algebra
    {UserID: userID, QuestionID: 1, IsCorrect: false},   // Algebra
    {UserID: userID, QuestionID: 2, IsCorrect: true},    // Calculus
    {UserID: userID, QuestionID: 3, IsCorrect: true},    // Algebra
}

```

Wait — QuestionID 1 and 3 are **both Algebra** → 3 Algebra attempts:

- Correct: Q1 (true), Q1 (false), Q3 (true) → 2 correct
- So Algebra: 2/3 ≈ 66.67%
- Calculus: 1/1 = 100%

✅ Matches expected.

---

### Final Thoughts

This optimized version scales well even with thousands of attempts, as all processing is done in the database engine efficiently

**Grok-4’s Response:**

Certainly! To improve the efficiency, you should avoid executing a separate query inside the loop. Instead, you can perform a join between `QuestionAttempt` and `Question` and use SQL aggregation functions to count total attempts and correct attempts per topic in a single query.

Here's a more efficient version of your `CalculateUserTopicAccuracy` function:

```go
func CalculateUserTopicAccuracy(db *gorm.DB, userID uuid.UUID) (map[string]float64, error) {
	// Define a temporary struct to hold the aggregation results
	type Result struct {
		Topic       string
		Total       int64
		Correct     int64
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

```

### Key improvements:

- Uses a single SQL query with `JOIN`, `GROUP BY`, and aggregate functions (`COUNT`, `SUM`) to gather all data at once.
- Eliminates the per-attempt queries, reducing database round-trips significantly.
- Handles all calculations efficiently within the database engine.

This approach should enhance performance especially with larger datasets.

**Takeaways from the test:**

- All three were able to identify the inefficiencies and solve them.
- Kimi K2 used the approach of performing all the calculations within SQL `select` statement, making the Go code simpler.
- I personally found Qwen3’s reponse pretty simple, concise and understandable for a Go developer like me. It was *super fast*.
