package accountpolicies

import (
	"context"
	"errors"
)

type Policies struct {
	PublicRegistrationEnabled         bool `json:"publicRegistrationEnabled"`
	SelfServiceAccountDeletionEnabled bool `json:"selfServiceAccountDeletionEnabled"`
}

type UpdateInput struct {
	PublicRegistrationEnabled         *bool `json:"publicRegistrationEnabled"`
	SelfServiceAccountDeletionEnabled *bool `json:"selfServiceAccountDeletionEnabled"`
}

type PublicAuthConfig struct {
	RegistrationEnabled bool `json:"registrationEnabled"`
}

// Repository is the persistence seam for the singleton account policies.
// Implementations apply partial updates atomically so independent changes do
// not overwrite each other.
type Repository interface {
	GetPolicies(ctx context.Context) (Policies, error)
	UpdatePolicies(ctx context.Context, input UpdateInput) (Policies, error)
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Get(ctx context.Context) (Policies, error) {
	repository, err := s.requireRepository()
	if err != nil {
		return Policies{}, err
	}
	return repository.GetPolicies(ctx)
}

func (s *Service) Update(ctx context.Context, input UpdateInput) (Policies, error) {
	repository, err := s.requireRepository()
	if err != nil {
		return Policies{}, err
	}
	return repository.UpdatePolicies(ctx, input)
}

func (s *Service) PublicAuthConfig(ctx context.Context) (PublicAuthConfig, error) {
	policies, err := s.Get(ctx)
	if err != nil {
		return PublicAuthConfig{}, err
	}

	return PublicAuthConfig{
		RegistrationEnabled: policies.PublicRegistrationEnabled,
	}, nil
}

func (s *Service) requireRepository() (Repository, error) {
	if s == nil || s.repository == nil {
		return nil, errors.New("accountpolicies: nil service")
	}
	return s.repository, nil
}
