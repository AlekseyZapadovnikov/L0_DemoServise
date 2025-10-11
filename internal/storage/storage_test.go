// file: internal/storage/storage_test.go

package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/Asus/L0_DemoServise/internal/entity"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"
)

var (
	// Выносим колонки в глобальную переменную, чтобы не дублировать
	cols = []string{
		"order_uid", "track_number", "entry", "locale", "internal_signature", "customer_id",
		"delivery_service", "shardkey", "sm_id", "date_created", "oof_shard",
		"name", "phone", "zip", "city", "address", "region", "email",
		"request_id", "currency", "provider", "amount", "payment_dt", "bank",
		"delivery_cost", "goods_total", "custom_fee",
		"rid", "chrt_id", "item_track_number", "price", "item_name",
		"sale", "size", "total_price", "nm_id", "brand", "status",
	}
)

// ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ

// loadTemplateOrder загружает базовый заказ-шаблон из JSON файла.
func loadTemplateOrder() (entity.Order, error) {
	jsonData, err := os.ReadFile("../../cmd/helpCMD/model.json")
	if err != nil {
		return entity.Order{}, fmt.Errorf("failed to read model.json: %w", err)
	}

	var testOrder entity.Order
	if err := json.Unmarshal(jsonData, &testOrder); err != nil {
		return entity.Order{}, fmt.Errorf("failed to unmarshal model.json: %w", err)
	}
	return testOrder, nil
}

// generateTestOrder создаёт уникальный заказ на основе шаблона и индекса.
func generateTestOrder(templOrder entity.Order, i int) entity.Order {
	order := templOrder
	// Глубокое копирование слайса, чтобы тесты не влияли друг на друга
	order.Items = make([]entity.Item, len(templOrder.Items))
	copy(order.Items, templOrder.Items)

	// Делаем заказ уникальным
	order.OrderUID = fmt.Sprintf("test-uid-%d", i)
	order.TrackNumber = fmt.Sprintf("TRACK-%d", i)
	if len(order.Items) > 0 {
		order.Items[0].Rid = fmt.Sprintf("rid-%d", i) // Делаем уникальным и товар
	}
	return order
}

// orderToRow конвертирует заказ и один его товар в срез []any для pgxmock.
func orderToRow(order entity.Order, itemIndex int) []any {
	item := entity.Item{} // На случай, если у заказа нет товаров
	if len(order.Items) > itemIndex {
		item = order.Items[itemIndex]
	}
	return []any{
		order.OrderUID, order.TrackNumber, order.Entry, order.Locale, order.InternalSignature, order.CustomerID,
		order.DeliveryService, order.ShardKey, order.SmID, order.DateCreated, order.OofShard,
		order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip, order.Delivery.City, order.Delivery.Address, order.Delivery.Region, order.Delivery.Email,
		order.Payment.RequestID, order.Payment.Currency, order.Payment.Provider, order.Payment.Amount, order.Payment.PaymentDt, order.Payment.Bank,
		order.Payment.DeliveryCost, order.Payment.GoodsTotal, order.Payment.CustomFee,
		item.Rid, item.ChrtID, item.TrackNumber, item.Price, item.Name,
		item.Sale, item.Size, item.TotalPrice, item.NmID, item.Brand, item.Status,
	}
}



// ТЕСТЫ
func TestGetOrderByUID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	s := Storage{pool: mock}

	testOrder, err := loadTemplateOrder()
	if err != nil {
		t.Fatalf("failed to load template order: %v", err)
	}
	testOrderUID := testOrder.OrderUID

	testCases := []struct {
		name          string
		orderUID      string
		mockSetup     func()
		expectedOrder entity.Order
		expectedErr   error
	}{
		{
			name:     "Success: Order with one item found",
			orderUID: testOrderUID,
			mockSetup: func() {
				rows := pgxmock.NewRows(cols).AddRow(orderToRow(testOrder, 0)...)
				mock.ExpectQuery(`SELECT .* WHERE o.order_uid = \$1`).
					WithArgs(testOrderUID).
					WillReturnRows(rows)
			},
			expectedOrder: testOrder,
			expectedErr:   nil,
		},
		{
			name:     "Failure: Order not found",
			orderUID: "nonexistent-uid",
			mockSetup: func() {
				rows := pgxmock.NewRows(cols)
				mock.ExpectQuery(`SELECT .* WHERE o.order_uid = \$1`).
					WithArgs("nonexistent-uid").
					WillReturnRows(rows)
			},
			expectedOrder: entity.Order{},
			expectedErr:   pgx.ErrNoRows,
		},
		{
			name:     "Failure: Database error",
			orderUID: testOrderUID,
			mockSetup: func() {
				mock.ExpectQuery(`SELECT .* WHERE o.order_uid = \$1`).
					WithArgs(testOrderUID).
					WillReturnError(fmt.Errorf("something went wrong"))
			},
			expectedOrder: entity.Order{},
			expectedErr:   fmt.Errorf("failed to query order: something went wrong"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockSetup()
			order, err := s.GetOrderByUID(context.Background(), tc.orderUID)

			if !reflect.DeepEqual(err, tc.expectedErr) {
				// Проверяем текст ошибки, если DeepEqual не сработал (из-за разных типов)
				if tc.expectedErr != nil && err != nil && tc.expectedErr.Error() == err.Error() {
					// Ошибки совпадают, всё ок
				} else {
					t.Errorf("expected error '%v', but got '%v'", tc.expectedErr, err)
				}
			}

			if tc.expectedErr == nil {
				// Сравниваем структуры только если ошибки не было
				if !reflect.DeepEqual(order, tc.expectedOrder) {
					gotJSON, _ := json.MarshalIndent(order, "", "  ")
					expJSON, _ := json.MarshalIndent(tc.expectedOrder, "", "  ")
					t.Errorf("order mismatch:\n\nexpected:\n%s\n\ngot:\n%s\n", string(expJSON), string(gotJSON))
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestGetLastNOrders(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	s := Storage{pool: mock}

	baseOrder, err := loadTemplateOrder()
	if err != nil {
		t.Fatalf("failed to load template order: %v", err)
	}

	// Готовим тестовые данные
	order1 := generateTestOrder(baseOrder, 1)
	order2 := generateTestOrder(baseOrder, 2)

	orderWithTwoItems := generateTestOrder(baseOrder, 3)
	item2 := orderWithTwoItems.Items[0] // Копируем первый товар
	item2.Rid = "rid-3-item-2"           // Делаем его уникальным
	item2.Name = "Второй товар"
	orderWithTwoItems.Items = append(orderWithTwoItems.Items, item2)

	testCases := []struct {
		name           string
		n              int
		mockSetup      func()
		expectedOrders []entity.Order
		expectedErr    error
	}{
		{
			name: "Успех: Получение 2 заказов",
			n:    2,
			mockSetup: func() {
				rows := pgxmock.NewRows(cols).
					AddRow(orderToRow(order1, 0)...).
					AddRow(orderToRow(order2, 0)...)

				mock.ExpectQuery(`SELECT .* LIMIT \$1`).
					WithArgs(2).
					WillReturnRows(rows)
			},
			expectedOrders: []entity.Order{order1, order2},
			expectedErr:    nil,
		},
		{
			name: "Успех: Получение 1 заказа с 2 товарами",
			n:    1,
			mockSetup: func() {
				// JOIN вернет две строки для одного заказа
				rows := pgxmock.NewRows(cols).
					AddRow(orderToRow(orderWithTwoItems, 0)...). // Строка для первого товара
					AddRow(orderToRow(orderWithTwoItems, 1)...)  // Строка для второго товара

				mock.ExpectQuery(`SELECT .* LIMIT \$1`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedOrders: []entity.Order{orderWithTwoItems}, // Ожидаем один "собранный" заказ
			expectedErr:    nil,
		},
		{
			name: "Успех: В БД меньше заказов чем запрошено",
			n:    5,
			mockSetup: func() {
				// БД вернет только один заказ, хотя просили 5
				rows := pgxmock.NewRows(cols).AddRow(orderToRow(order1, 0)...)
				mock.ExpectQuery(`SELECT .* LIMIT \$1`).
					WithArgs(5).
					WillReturnRows(rows)
			},
			expectedOrders: []entity.Order{order1},
			expectedErr:    nil,
		},
		{
			name: "Ошибка: Ошибка базы данных",
			n:    3,
			mockSetup: func() {
				mock.ExpectQuery(`SELECT .* LIMIT \$1`).
					WithArgs(3).
					WillReturnError(fmt.Errorf("db connection failed"))
			},
			expectedOrders: nil,
			expectedErr:    fmt.Errorf("failed to query all orders: db connection failed"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockSetup()
			orders, err := s.GetLastNOrders(context.Background(), tc.n)

			if !reflect.DeepEqual(err, tc.expectedErr) {
				if tc.expectedErr != nil && err != nil && tc.expectedErr.Error() == err.Error() {
				} else {
					t.Errorf("expected error '%v', but got '%v'", tc.expectedErr, err)
				}
			}

			if tc.expectedErr == nil {
				// Для надежного сравнения: если слайсы пустые, приводим их к nil
				if len(orders) == 0 {
					orders = nil
				}
				if len(tc.expectedOrders) == 0 {
					tc.expectedOrders = nil
				}
				if !reflect.DeepEqual(orders, tc.expectedOrders) {
					gotJSON, _ := json.MarshalIndent(orders, "", "  ")
					expJSON, _ := json.MarshalIndent(tc.expectedOrders, "", "  ")
					t.Errorf("orders mismatch:\n\nexpected:\n%s\n\ngot:\n%s\n", string(expJSON), string(gotJSON))
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}