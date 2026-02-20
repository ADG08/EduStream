package domain

import "errors"

var (
	// ErrRoomLocked indique que la salle est verrouillée : aucune nouvelle entrée (étudiant ou professeur) n'est autorisée.
	ErrRoomLocked = errors.New("room is locked: no new entries allowed")
)
