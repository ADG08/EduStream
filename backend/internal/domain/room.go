package domain

import "time"

// RoomMode représente l'état d'une salle (ouverte ou verrouillée / salle rouge).
type RoomMode string

const (
	RoomModeOpen   RoomMode = "open"
	RoomModeLocked RoomMode = "locked"
)

// Room représente une salle de classe virtuelle.
type Room struct {
	ID          string
	Mode        RoomMode
	Students    []Student
	Teacher     *Teacher // présent uniquement quand Mode == RoomModeLocked
	LockedUntil int64    // Unix timestamp (secondes), pertinent uniquement quand Mode == RoomModeLocked
}

// IsLocked retourne vrai si la salle est verrouillée (mode == locked ET le timer n'a pas expiré).
func (r *Room) IsLocked() bool {
	if r.Mode != RoomModeLocked {
		return false
	}
	// Vérifier que le timer n'a pas expiré
	return time.Now().Unix() < r.LockedUntil
}

// Lock verrouille la salle avec un professeur et une durée. Retourne ErrRoomLocked si la salle est déjà verrouillée.
// Transition open → locked uniquement si la salle est actuellement open.
func (r *Room) Lock(teacher Teacher, duration time.Duration) error {
	if r.Mode == RoomModeLocked {
		return ErrRoomLocked
	}
	r.Teacher = &teacher
	r.Mode = RoomModeLocked
	r.LockedUntil = time.Now().Add(duration).Unix()
	return nil
}

// Unlock déverrouille la salle : passe en mode open, reset Teacher à nil, et vide la liste des étudiants.
// Le timer est annulé (reset) implicitement en passant en mode open.
func (r *Room) Unlock() {
	r.Mode = RoomModeOpen
	r.Teacher = nil
	r.LockedUntil = 0
	r.Students = nil // Vide la liste des étudiants (fin du cours)
}

// AddStudent ajoute un étudiant à la salle. Retourne ErrRoomLocked si la salle est verrouillée.
func (r *Room) AddStudent(s Student) error {
	if r.IsLocked() {
		return ErrRoomLocked
	}
	for i := range r.Students {
		if r.Students[i].StudentID == s.StudentID {
			r.Students[i].LastSeen = s.LastSeen
			return nil
		}
	}
	r.Students = append(r.Students, s)
	return nil
}

// RemoveStudent retire un étudiant de la salle par son ID.
func (r *Room) RemoveStudent(studentID string) {
	for i := range r.Students {
		if r.Students[i].StudentID == studentID {
			r.Students = append(r.Students[:i], r.Students[i+1:]...)
			return
		}
	}
}

// AddTeacher ajoute un professeur à la salle et passe la salle en mode verrouillé.
// Retourne ErrRoomLocked si la salle est déjà verrouillée (un professeur est déjà présent).
// Utilise Lock() en interne avec une durée par défaut de 10 minutes.
func (r *Room) AddTeacher(t Teacher) error {
	return r.Lock(t, 10*time.Minute)
}

// SetLockedUntil définit l'instant (Unix secondes) jusqu'auquel la salle reste verrouillée.
// À appeler par la couche application après AddTeacher (ex. now + duration_seconds).
// Déprécié : utiliser Lock() directement avec la durée souhaitée.
func (r *Room) SetLockedUntil(until int64) {
	r.LockedUntil = until
}
