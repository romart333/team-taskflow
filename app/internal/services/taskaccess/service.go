// Package taskaccess is the single owner of the "may this actor touch this
// task/team" rule shared by every task-scoped usecase.
package taskaccess

import (
	"context"
	"errors"
	"fmt"

	"team-taskflow/internal/domain"
)

// TaskGetter loads tasks by ID.
type TaskGetter interface {
	// GetByID returns domain.ErrNotFound when the task does not exist.
	GetByID(ctx context.Context, taskID int64) (domain.Task, error)
}

// MemberGetter checks team memberships.
type MemberGetter interface {
	// GetMember returns domain.ErrNotFound when the user is not a member.
	GetMember(ctx context.Context, teamID, userID int64) (domain.TeamMember, error)
}

type Service struct {
	tasks TaskGetter
	teams MemberGetter
}

func New(tasks TaskGetter, teams MemberGetter) *Service {
	return &Service{tasks: tasks, teams: teams}
}

// EnsureTeamMember maps a missing membership to a client-visible permission error.
func (s *Service) EnsureTeamMember(ctx context.Context, teamID, actorID int64) error {
	if _, err := s.teams.GetMember(ctx, teamID, actorID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return fmt.Errorf("checking membership: %w",
				domain.NewPermissionDeniedError("you are not a member of this team"))
		}
		return fmt.Errorf("getting membership: %w", err)
	}
	return nil
}

// LoadTaskForMember loads a task and authorizes the actor as a member of the
// task's team, mapping a missing task to a client-visible not-found error.
func (s *Service) LoadTaskForMember(ctx context.Context, taskID, actorID int64) (domain.Task, error) {
	task, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.Task{}, fmt.Errorf("loading task: %w", domain.NewNotFoundError("task not found"))
		}
		return domain.Task{}, fmt.Errorf("loading task: %w", err)
	}
	if err := s.EnsureTeamMember(ctx, task.TeamID, actorID); err != nil {
		return domain.Task{}, err
	}
	return task, nil
}
