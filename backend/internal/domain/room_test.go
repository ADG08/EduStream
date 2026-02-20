package domain

import (
	"errors"
	"testing"
	"time"
)

func TestRoom_AddStudent_WhenLocked_ReturnsError(t *testing.T) {
	room := &Room{
		ID:   "room-1",
		Mode: RoomModeLocked,
		Teacher: &Teacher{
			TeacherID: "teacher-001",
			JoinedAt:  1708455600,
		},
		LockedUntil: time.Now().Add(10 * time.Minute).Unix(), // Timer actif
	}
	student := Student{
		StudentID: "student-abc",
		LastSeen:  1708455620,
	}

	err := room.AddStudent(student)

	if err == nil {
		t.Fatal("expected error when adding student to locked room, got nil")
	}
	if !errors.Is(err, ErrRoomLocked) {
		t.Errorf("expected ErrRoomLocked, got %v", err)
	}
	if len(room.Students) != 0 {
		t.Errorf("student must not be added when room is locked, got %d students", len(room.Students))
	}
}

func TestRoom_AddTeacher_WhenLocked_ReturnsError(t *testing.T) {
	room := &Room{
		ID:   "room-1",
		Mode: RoomModeLocked,
		Teacher: &Teacher{
			TeacherID: "teacher-001",
			JoinedAt:  1708455600,
		},
	}
	otherTeacher := Teacher{
		TeacherID: "teacher-002",
		JoinedAt:  1708455700,
	}

	err := room.AddTeacher(otherTeacher)

	if err == nil {
		t.Fatal("expected error when adding teacher to already locked room, got nil")
	}
	if !errors.Is(err, ErrRoomLocked) {
		t.Errorf("expected ErrRoomLocked, got %v", err)
	}
	if room.Teacher == nil || room.Teacher.TeacherID != "teacher-001" {
		t.Error("existing teacher must not be replaced when room is already locked")
	}
}

func TestRoom_AddStudent_WhenOpen_Succeeds(t *testing.T) {
	room := &Room{
		ID:   "room-1",
		Mode: RoomModeOpen,
	}
	student := Student{
		StudentID: "student-abc",
		LastSeen:  1708455620,
	}

	err := room.AddStudent(student)

	if err != nil {
		t.Fatalf("unexpected error when adding student to open room: %v", err)
	}
	if len(room.Students) != 1 || room.Students[0].StudentID != "student-abc" {
		t.Errorf("student not added correctly: %+v", room.Students)
	}
}

func TestRoom_AddTeacher_WhenOpen_SucceedsAndLocksRoom(t *testing.T) {
	room := &Room{
		ID:   "room-1",
		Mode: RoomModeOpen,
	}
	teacher := Teacher{
		TeacherID: "teacher-001",
		JoinedAt:  1708455600,
	}

	err := room.AddTeacher(teacher)

	if err != nil {
		t.Fatalf("unexpected error when adding teacher to open room: %v", err)
	}
	if room.Mode != RoomModeLocked {
		t.Errorf("room mode should be locked after adding teacher, got %s", room.Mode)
	}
	if room.Teacher == nil || room.Teacher.TeacherID != "teacher-001" || room.Teacher.JoinedAt != 1708455600 {
		t.Errorf("teacher not set correctly: %+v", room.Teacher)
	}
	if room.LockedUntil == 0 {
		t.Error("LockedUntil should be set after AddTeacher")
	}
}

// Tests pour Lock()

func TestRoom_Lock_WhenOpen_Succeeds(t *testing.T) {
	room := &Room{
		ID:   "room-1",
		Mode: RoomModeOpen,
	}
	teacher := Teacher{
		TeacherID: "teacher-001",
		JoinedAt:  1708455600,
	}
	duration := 10 * time.Minute

	err := room.Lock(teacher, duration)

	if err != nil {
		t.Fatalf("unexpected error when locking open room: %v", err)
	}
	if room.Mode != RoomModeLocked {
		t.Errorf("room mode should be locked, got %s", room.Mode)
	}
	if room.Teacher == nil || room.Teacher.TeacherID != "teacher-001" {
		t.Errorf("teacher not set correctly: %+v", room.Teacher)
	}
	expectedUntil := time.Now().Add(duration).Unix()
	// Tolérance de 2 secondes pour les tests
	if room.LockedUntil < expectedUntil-2 || room.LockedUntil > expectedUntil+2 {
		t.Errorf("LockedUntil should be approximately %d, got %d", expectedUntil, room.LockedUntil)
	}
}

func TestRoom_Lock_WhenAlreadyLocked_ReturnsError(t *testing.T) {
	room := &Room{
		ID:   "room-1",
		Mode: RoomModeLocked,
		Teacher: &Teacher{
			TeacherID: "teacher-001",
			JoinedAt:  1708455600,
		},
		LockedUntil: time.Now().Add(10 * time.Minute).Unix(),
	}
	otherTeacher := Teacher{
		TeacherID: "teacher-002",
		JoinedAt:  1708455700,
	}

	err := room.Lock(otherTeacher, 10*time.Minute)

	if err == nil {
		t.Fatal("expected error when locking already locked room, got nil")
	}
	if !errors.Is(err, ErrRoomLocked) {
		t.Errorf("expected ErrRoomLocked, got %v", err)
	}
	if room.Teacher == nil || room.Teacher.TeacherID != "teacher-001" {
		t.Error("existing teacher must not be replaced when room is already locked")
	}
}

// Tests pour Unlock()

func TestRoom_Unlock_WhenLocked_UnlocksAndClearsStudents(t *testing.T) {
	room := &Room{
		ID:   "room-1",
		Mode: RoomModeLocked,
		Teacher: &Teacher{
			TeacherID: "teacher-001",
			JoinedAt:  1708455600,
		},
		LockedUntil: time.Now().Add(10 * time.Minute).Unix(),
		Students: []Student{
			{StudentID: "student-1", LastSeen: 1708455600},
			{StudentID: "student-2", LastSeen: 1708455610},
		},
	}

	room.Unlock()

	if room.Mode != RoomModeOpen {
		t.Errorf("room mode should be open after unlock, got %s", room.Mode)
	}
	if room.Teacher != nil {
		t.Error("teacher should be nil after unlock")
	}
	if room.LockedUntil != 0 {
		t.Errorf("LockedUntil should be reset to 0, got %d", room.LockedUntil)
	}
	if len(room.Students) != 0 {
		t.Errorf("students list should be empty after unlock, got %d students", len(room.Students))
	}
}

func TestRoom_Unlock_WhenOpen_IsIdempotent(t *testing.T) {
	room := &Room{
		ID:   "room-1",
		Mode: RoomModeOpen,
		Students: []Student{
			{StudentID: "student-1", LastSeen: 1708455600},
		},
	}

	room.Unlock()

	if room.Mode != RoomModeOpen {
		t.Errorf("room mode should remain open, got %s", room.Mode)
	}
	if room.Teacher != nil {
		t.Error("teacher should remain nil")
	}
	if len(room.Students) != 0 {
		t.Errorf("students list should be empty after unlock, got %d students", len(room.Students))
	}
}

// Tests pour IsLocked()

func TestRoom_IsLocked_WhenOpen_ReturnsFalse(t *testing.T) {
	room := &Room{
		ID:   "room-1",
		Mode: RoomModeOpen,
	}

	if room.IsLocked() {
		t.Error("IsLocked should return false when room is open")
	}
}

func TestRoom_IsLocked_WhenLockedAndTimerNotExpired_ReturnsTrue(t *testing.T) {
	room := &Room{
		ID:   "room-1",
		Mode: RoomModeLocked,
		Teacher: &Teacher{
			TeacherID: "teacher-001",
			JoinedAt:  1708455600,
		},
		LockedUntil: time.Now().Add(10 * time.Minute).Unix(),
	}

	if !room.IsLocked() {
		t.Error("IsLocked should return true when room is locked and timer not expired")
	}
}

func TestRoom_IsLocked_WhenLockedButTimerExpired_ReturnsFalse(t *testing.T) {
	room := &Room{
		ID:   "room-1",
		Mode: RoomModeLocked,
		Teacher: &Teacher{
			TeacherID: "teacher-001",
			JoinedAt:  1708455600,
		},
		LockedUntil: time.Now().Add(-1 * time.Minute).Unix(), // Expiré il y a 1 minute
	}

	if room.IsLocked() {
		t.Error("IsLocked should return false when timer has expired, even if mode is locked")
	}
}

// Tests pour RemoveStudent()

func TestRoom_RemoveStudent_RemovesStudentFromList(t *testing.T) {
	room := &Room{
		ID:   "room-1",
		Mode: RoomModeOpen,
		Students: []Student{
			{StudentID: "student-1", LastSeen: 1708455600},
			{StudentID: "student-2", LastSeen: 1708455610},
			{StudentID: "student-3", LastSeen: 1708455620},
		},
	}

	room.RemoveStudent("student-2")

	if len(room.Students) != 2 {
		t.Fatalf("expected 2 students after removal, got %d", len(room.Students))
	}
	if room.Students[0].StudentID != "student-1" || room.Students[1].StudentID != "student-3" {
		t.Errorf("wrong students remaining: %+v", room.Students)
	}
}

func TestRoom_RemoveStudent_WhenStudentNotPresent_DoesNothing(t *testing.T) {
	room := &Room{
		ID:   "room-1",
		Mode: RoomModeOpen,
		Students: []Student{
			{StudentID: "student-1", LastSeen: 1708455600},
		},
	}

	room.RemoveStudent("student-nonexistent")

	if len(room.Students) != 1 {
		t.Errorf("expected 1 student after removal attempt, got %d", len(room.Students))
	}
}

// Tests d'intégration : transitions complètes

func TestRoom_Transition_OpenToLockedToOpen(t *testing.T) {
	room := &Room{
		ID:   "room-1",
		Mode: RoomModeOpen,
		Students: []Student{
			{StudentID: "student-1", LastSeen: 1708455600},
		},
	}
	teacher := Teacher{
		TeacherID: "teacher-001",
		JoinedAt:  1708455600,
	}

	// Transition open → locked
	err := room.Lock(teacher, 10*time.Minute)
	if err != nil {
		t.Fatalf("failed to lock room: %v", err)
	}
	if !room.IsLocked() {
		t.Error("room should be locked after Lock()")
	}

	// Tentative d'ajout d'étudiant en mode locked (doit échouer)
	student2 := Student{StudentID: "student-2", LastSeen: 1708455610}
	err = room.AddStudent(student2)
	if err == nil {
		t.Error("AddStudent should fail when room is locked")
	}
	if len(room.Students) != 1 {
		t.Errorf("student should not be added, expected 1 student, got %d", len(room.Students))
	}

	// Transition locked → open
	room.Unlock()
	if room.Mode != RoomModeOpen {
		t.Error("room should be open after Unlock()")
	}
	if room.IsLocked() {
		t.Error("IsLocked should return false after Unlock()")
	}
	if len(room.Students) != 0 {
		t.Errorf("students should be cleared after Unlock(), got %d students", len(room.Students))
	}

	// Après unlock, on peut à nouveau ajouter des étudiants
	err = room.AddStudent(student2)
	if err != nil {
		t.Fatalf("should be able to add student after unlock: %v", err)
	}
	if len(room.Students) != 1 {
		t.Errorf("expected 1 student after adding, got %d", len(room.Students))
	}
}
