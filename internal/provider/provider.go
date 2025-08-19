package provider

import (
	"context"
	"dockwatch/internal/domain"
)

// Provider defines the interface for volume data providers
type Provider interface {
	ListVolumes(ctx context.Context) ([]domain.Volume, error)
	GetVolumeDetails(ctx context.Context, name string) (*domain.Volume, error)
	RemoveVolume(ctx context.Context, name string) error
	Close() error
}
