package domain

// Teacher représente un professeur présent dans une salle.
type Teacher struct {
	TeacherID string
	JoinedAt  int64 // Unix timestamp (secondes)
}
