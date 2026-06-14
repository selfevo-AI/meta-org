package metaorg

import "context"

type Repository interface {
	Overview(ctx context.Context) (Overview, error)
	Inbox(ctx context.Context, filter InboxFilter) ([]InboxItem, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetOverview(ctx context.Context) (*Overview, error) {
	overview, err := s.repo.Overview(ctx)
	if err != nil {
		return nil, err
	}
	return &overview, nil
}

func (s *Service) GetInbox(ctx context.Context, filter InboxFilter) ([]InboxItem, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	return s.repo.Inbox(ctx, filter)
}
