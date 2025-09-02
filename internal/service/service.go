// пакет сервис реализует слой бизнес логики

package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Asus/L0_DemoServise/internal/entity"
	"github.com/jackc/pgx/v5"
)

type getOrder interface {
	GetOrderByUID(ctx context.Context, in string) (entity.Order, error)
	GetAllOrders(ctx context.Context) ([]entity.Order, error)
	SaveOrder(ctx context.Context, o entity.Order) error
}

type Service struct {
	OrderMap map[string]entity.Order // это реализация нашего Cache, очень удобно использовать map тк достум к данным константа
	OrderTaker getOrder // это интерфейс, в нашем случае реализация этого интерфейса это пакет storage
}

func NewService(storage getOrder) *Service {
	return &Service{
		OrderMap:   make(map[string]entity.Order),
		OrderTaker: storage,
	}
}

// загружаем Cache при запусте сервиса
func (s *Service) LoadCache(ctx context.Context) error {
	orders, err := s.OrderTaker.GetAllOrders(ctx)
	if err != nil {
		return fmt.Errorf("error occured while tryed load cache in service.LoadCache() %w", err)
	}

	for _, ord := range orders {
		s.OrderMap[ord.OrderUID] = ord
	}

	return nil
}

// возвращает Order по UID
func (s *Service) GiveOrderByUID(UID string) (entity.Order, error) {
	ord, isIn := s.OrderMap[UID]

	if isIn {
		return ord, nil
	}

	ord, err := s.OrderTaker.GetOrderByUID(context.Background(), UID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.Order{}, fmt.Errorf("order with UID %s not found", UID)
		}
		return entity.Order{}, fmt.Errorf("error occurred while trying to get order with UID %s: %w", UID, err)
	}
	s.addToCache(ord)
	slog.Info("Order loaded from database to cache", "order_uid", ord.OrderUID)
	return ord, nil
}

// сохраняет Order в БД и в Cache
func (s *Service) SaveOrder(ctx context.Context, o entity.Order) error {
	s.addToCache(o)
	slog.Info("Saving order to database", "order_uid", o.OrderUID)
	if err := s.OrderTaker.SaveOrder(ctx, o); err != nil {
		return fmt.Errorf("error occurred while trying to save order: %w", err)
	}
	return nil
}

// добавляет Order в cache
func (s *Service) addToCache(ord entity.Order) {
	s.OrderMap[ord.OrderUID] = ord
}