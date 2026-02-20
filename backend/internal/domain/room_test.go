package domain

import (
	"errors"
	"testing"
)

func TestRoom_AddStudent_WhenLocked_ReturnsError(t *testing.T) {
	room := &Room{
		ID:   "room-1",
		Mode: RoomModeLocked,
		Teacher: &Teacher{
			TeacherID: "teacher-001",
			JoinedAt:  1708455600,
		},
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
}
