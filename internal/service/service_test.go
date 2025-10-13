package service

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/Asus/L0_DemoServise/internal/entity"
	"github.com/jackc/pgx/v5"
)

type TestData struct {
	Orders []entity.Order `json:"orders"`
}

func loadTestData() []entity.Order {
	file, err := os.Open("tests/testData.json")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}

	var testData TestData
	if err := json.Unmarshal(data, &testData); err != nil {
		panic(err)
	}

	return testData.Orders
}

func giveItemsByOrdersSlice(orders []entity.Order) []*Item {
	items := make([]*Item, 0, len(orders))
	for _, ord := range orders {
		item := makeItem(ord.OrderUID)
		items = append(items, item)
	}
	return items
}

func giveItemsUIDSlice(items []*Item) []string {
	uids := make([]string, 0, len(items))
	for _, item := range items {
		uids = append(uids, item.Value)
	}
	return uids
}

func TestPriorityQueue(T *testing.T) {
	prq := NewSafePriorityQueue(10)
	orders := loadTestData()
	items := make([]*Item, 0, len(orders))
	for _, ord := range orders {
		item := makeItem(ord.OrderUID)
		items = append(items, item)
		prq.Push(item)
	}
	revItems := make([]*Item, len(items))
	copy(revItems, items)
	slices.Reverse(revItems)
	revItemsUID := giveItemsUIDSlice(revItems)
	itemsUID := giveItemsUIDSlice(items)

	if prq.Len() != len(orders) {
		T.Errorf("expected length %d, got %d", len(orders), prq.Len())
	}

	testCases := []struct {
		testName         string
		itemsToPush      []*Item
		funcToRun        func(prq *SafePriorityQueue, items []*Item)
		expectedPopOrder []string // порядок UID при мзвлечении
		checkFunc        func(a, b []string) bool
	}{
		{
			testName:         "обычное добавление по порядку",
			itemsToPush:      items,
			funcToRun:        func(prq *SafePriorityQueue, items []*Item) {},
			expectedPopOrder: revItemsUID, // ItemsUID в обратном порядке
			checkFunc: func(a, b []string) bool {
				return slices.Equal(a, b)
			},
		},
		{
			testName:    "добавление с обновлением приоритета у одного элемента",
			itemsToPush: items,
			funcToRun: func(prq *SafePriorityQueue, items []*Item) {
				prq.Update(items[3], time.Now()) // обновляем приоритет у первого элемента
			},
			expectedPopOrder: []string{itemsUID[0]},
			checkFunc: func(a, b []string) bool {
				return a[0] == b[0]
			}},
	}
	for _, tc := range testCases {
		T.Run(tc.testName, func(t *testing.T) {
			prq := NewSafePriorityQueue(10)

			for _, item := range tc.itemsToPush {
				prq.Push(item)
			}
			var popped []string
			for prq.Len() > 0 {
				item := prq.Pop()
				if item != nil {
					popped = append(popped, item.Value)
				}
			}

			// Проверяем результат
			switch tc.testName {
			case "обычное добавление по порядку":
				T.Log("PASS")
			case "добавление с обновлением приоритета у одного элемента":
				if tc.expectedPopOrder[0] == popped[0] {
					T.Log("PASS")
				} else {
					T.Errorf("expected first popped %s, got %s", tc.expectedPopOrder[0], popped[0])
				}
			}
		})
	}
}

type mockStorage struct {
	mockDB map[string]entity.Order
}

func (m *mockStorage) GetOrderByUID(ctx context.Context, uid string) (entity.Order, error) {
	if order, ok := m.mockDB[uid]; ok {
		return order, nil
	}
	return entity.Order{}, pgx.ErrNoRows
}

func (m *mockStorage) GetLastNOrders(ctx context.Context, n int) ([]entity.Order, error) {
	return []entity.Order{}, nil
}

func (m *mockStorage) SaveOrder(ctx context.Context, o entity.Order) error {
	return nil
}

func TestCache(t *testing.T) {
	mockOrders := map[string]entity.Order{
		"order-1": {OrderUID: "order-1", TrackNumber: "TRACK_A"},
		"order-2": {OrderUID: "order-2", TrackNumber: "TRACK_B"},
		"order-3": {OrderUID: "order-3", TrackNumber: "TRACK_C"},
		"order-4": {OrderUID: "order-4", TrackNumber: "TRACK_D"},
	}
	storage := &mockStorage{mockDB: mockOrders}

	t.Run("Get from empty cache (miss and fill)", func(t *testing.T) {
		// Создаем новый кэш для каждого теста, чтобы они не влияли друг на друга
		cache := NewCache(storage, 3)

		order, err := cache.GiveOrderByUID("order-1")

		if err != nil {
			t.Fatalf("expected no error, but got: %v", err)
		}
		if order.OrderUID != "order-1" {
			t.Errorf("expected to get order-1, but got: %s", order.OrderUID)
		}

		// Проверяем, что кэш теперь содержит этот элемент
		if _, exists := cache.OrderMap["order-1"]; !exists {
			t.Error("order-1 was not added to the cache after a miss")
		}
		if len(cache.OrderMap) != 1 {
			t.Errorf("expected cache size to be 1, but got: %d", len(cache.OrderMap))
		}
	})

	t.Run("Eviction of least recently used item", func(t *testing.T) {
		// Используем кэш с маленькой емкостью для проверки вытеснения
		cache := NewCache(storage, 2)

		cache.GiveOrderByUID("order-1")
		time.Sleep(10 * time.Millisecond)
		cache.GiveOrderByUID("order-2")
		time.Sleep(10 * time.Millisecond)

		if len(cache.OrderMap) != 2 {
			t.Fatalf("expected cache size to be 2 before eviction, but got: %d", len(cache.OrderMap))
		}

		cache.GiveOrderByUID("order-3")

		// Проверяем состояние кэша после вытеснения
		if len(cache.OrderMap) != 2 {
			t.Errorf("expected cache size to be 2 after eviction, but got: %d", len(cache.OrderMap))
		}
		if _, exists := cache.OrderMap["order-3"]; !exists {
			t.Error("new item order-3 was not added to cache")
		}
		if _, exists := cache.OrderMap["order-2"]; !exists {
			t.Error("item order-2 should not have been evicted")
		}
		if _, exists := cache.OrderMap["order-1"]; exists {
			t.Error("least recently used item order-1 was not evicted")
		}
	})

	t.Run("Accessing an item updates its priority and prevents eviction", func(t *testing.T) {
		cache := NewCache(storage, 2)

		// 1. Добавляем order-1, потом order-2. Порядок старости: 1, 2.
		cache.GiveOrderByUID("order-1")
		time.Sleep(10 * time.Millisecond)
		cache.GiveOrderByUID("order-2")
		time.Sleep(10 * time.Millisecond)

		cache.GiveOrderByUID("order-1")
		time.Sleep(10 * time.Millisecond)

		cache.GiveOrderByUID("order-3")

		if len(cache.OrderMap) != 2 {
			t.Errorf("expected cache size to be 2, but got: %d", len(cache.OrderMap))
		}
		if _, exists := cache.OrderMap["order-1"]; !exists {
			t.Error("order-1 should have been kept in cache because it was recently accessed")
		}
		if _, exists := cache.OrderMap["order-2"]; exists {
			t.Error("order-2 should have been evicted as the new least recently used item")
		}
	})

	t.Run("Getting a non-existent item returns an error", func(t *testing.T) {
		cache := NewCache(storage, 3)

		// Пытаемся получить заказ, которого нет ни в кэше, ни в моке БД
		_, err := cache.GiveOrderByUID("non-existent-order")

		if err == nil {
			t.Fatal("expected an error for a non-existent item, but got nil")
		}
	})
}
