package service

import (
	"context"
	"github.com/Asus/L0_DemoServise/internal/entity"
)

type OrderCache interface {
	GiveOrderByUID(UID string) (entity.Order, error)
	SaveOrder(ctx context.Context, o entity.Order) error
	LoadCache(ctx context.Context) error
}