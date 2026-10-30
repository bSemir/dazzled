package incidents

import (
	"context"
	"fmt"
	"strings"

	inbound "dazzled/backend/api/rest/v1/ports/inbound/incidents"
	"dazzled/backend/internal/domain"
	outbound "dazzled/backend/internal/pkg/ports/outbound/incidents"
)

type Service struct {
	repository outbound.Repository
}

var _ inbound.Service = (*Service)(nil)

func New(repository outbound.Repository) *Service {
	return &Service{repository: repository}
}

var allowedBrands = map[string]struct{}{
	"BMW": {}, "Audi": {}, "Mercedes-Benz": {}, "Tesla": {}, "Ford": {}, "Volkswagen": {},
	"Toyota": {}, "Honda": {}, "Hyundai": {}, "Kia": {}, "Nissan": {}, "Mazda": {}, "Subaru": {},
	"Volvo": {}, "Škoda": {}, "Peugeot": {}, "Renault": {}, "Opel": {}, "Dacia": {}, "Jeep": {},
	"Porsche": {}, "Lexus": {}, "Land Rover": {}, "Other": {}, "Unknown": {},
	"Aftermarket 'xenon' bulbs from a webshop": {},
}

var allowedVehicleTypes = map[string]struct{}{
	"Hatchback": {}, "Sedan": {}, "SUV / crossover": {}, "Coupe": {}, "Wagon / estate": {},
	"Pickup truck": {}, "Van": {}, "Minivan": {}, "Bus": {}, "Motorcycle": {}, "Scooter": {},
	"Bicycle": {}, "Police car": {}, "Other": {},
}

func (s *Service) Create(ctx context.Context, incident domain.Incident) (int64, error) {
	incident.VehicleType = strings.TrimSpace(incident.VehicleType)
	incident.Brand = strings.TrimSpace(incident.Brand)
	incident.Location = strings.TrimSpace(incident.Location)
	if _, ok := allowedVehicleTypes[incident.VehicleType]; !ok {
		return 0, ErrInvalidIncident
	}
	if _, ok := allowedBrands[incident.Brand]; !ok {
		return 0, ErrInvalidIncident
	}
	if len([]rune(incident.Location)) > 100 || incident.Severity < 1 || incident.Severity > 5 || (incident.LocalHour != nil && (*incident.LocalHour < 0 || *incident.LocalHour > 23)) {
		return 0, ErrInvalidIncident
	}
	id, err := s.repository.Create(ctx, incident)
	if err != nil {
		return 0, fmt.Errorf("create incident: %w", err)
	}
	return id, nil
}

func (s *Service) Count(ctx context.Context) (int, error) {
	statistics, err := s.Statistics(ctx, "all")
	if err != nil {
		return 0, fmt.Errorf("count incidents: %w", err)
	}
	return statistics.Total, nil
}

func (s *Service) Statistics(ctx context.Context, period string) (domain.Statistics, error) {
	if period == "" {
		period = "today"
	}
	if period != "today" && period != "week" && period != "all" {
		return domain.Statistics{}, ErrInvalidPeriod
	}
	statistics, err := s.repository.Statistics(ctx, period)
	if err != nil {
		return domain.Statistics{}, fmt.Errorf("get incident statistics: %w", err)
	}
	return statistics, nil
}
