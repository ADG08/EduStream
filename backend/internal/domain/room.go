package domain

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

// AddStudent ajoute un étudiant à la salle. Retourne ErrRoomLocked si la salle est verrouillée.
func (r *Room) AddStudent(s Student) error {
	if r.Mode == RoomModeLocked {
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

// AddTeacher ajoute un professeur à la salle et passe la salle en mode verrouillé.
// Retourne ErrRoomLocked si la salle est déjà verrouillée (un professeur est déjà présent).
func (r *Room) AddTeacher(t Teacher) error {
	if r.Mode == RoomModeLocked {
		return ErrRoomLocked
	}
	r.Teacher = &t
	r.Mode = RoomModeLocked
	return nil
}

// SetLockedUntil définit l'instant (Unix secondes) jusqu'auquel la salle reste verrouillée.
// À appeler par la couche application après AddTeacher (ex. now + duration_seconds).
func (r *Room) SetLockedUntil(until int64) {
	r.LockedUntil = until
}
