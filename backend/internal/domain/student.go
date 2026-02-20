package domain

// Student représente un étudiant présent dans une salle.
type Student struct {
	StudentID string
	LastSeen  int64 // Unix timestamp (secondes) de la dernière activité
}
