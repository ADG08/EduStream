package domain

// Role représente le rôle d'un client (étudiant ou professeur).
type Role string

const (
	RoleStudent Role = "student"
	RoleTeacher Role = "teacher"
)
