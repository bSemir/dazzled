// Package incidents tests normalization and validation of incident submissions.
package incidents

import (
	"context"
	"testing"

	"dazzled/backend/internal/domain"
)

type testRepository struct {
	incident domain.Incident
}

func (r *testRepository) Create(_ context.Context, incident domain.Incident) (int64, error) {
	r.incident = incident
	return 1, nil
}

func (r *testRepository) Statistics(_ context.Context, period string) (domain.Statistics, error) {
	return domain.Statistics{Period: period}, nil
}

func TestCreateValidatesAndNormalizesIncident(t *testing.T) {
	repository := &testRepository{}
	service := New(repository)
	incident := domain.Incident{VehicleType: " Police car ", Brand: "BMW", Location: " Sarajevo ", Severity: 5}
	if _, err := service.Create(context.Background(), incident); err != nil {
		t.Fatalf("valid incident rejected: %v", err)
	}
	if repository.incident.VehicleType != "Police car" || repository.incident.Location != "Sarajevo" {
		t.Fatalf("incident was not normalized: %+v", repository.incident)
	}
	for _, invalid := range []domain.Incident{
		{VehicleType: "spaceship", Brand: "BMW", Severity: 3},
		{VehicleType: "SUV / crossover", Brand: "BMW", Severity: 6},
		{VehicleType: "SUV / crossover", Brand: "made up", Severity: 1},
		{VehicleType: "SUV / crossover", Brand: "BMW", Location: "1234567890 1234567890 1234567890 1234567890 1234567890 1234567890 1234567890 1234567890 1234567890 12345678901", Severity: 1},
		{VehicleType: "SUV / crossover", Brand: "BMW", Severity: 1, LocalHour: intPointer(24)},
	} {
		if _, err := service.Create(context.Background(), invalid); err == nil {
			t.Errorf("invalid incident accepted: %+v", invalid)
		}
	}
}

func intPointer(value int) *int { return &value }
