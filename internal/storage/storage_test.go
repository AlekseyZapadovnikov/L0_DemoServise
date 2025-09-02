package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/Asus/L0_DemoServise/internal/entity"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"
)

// эта функция тестирует GetOrderByUID, перед её запуском нужно добавить в БД объект "../../cmd/helpCMD/model.json"
func TestGetOrderByUID(t *testing.T) {
	mock, err := pgxmock.NewPool() // создаём виртуальное подключение к БД
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	s := &Storage{pool: mock}

	// --- Тестовые данные ---
	var testOrder entity.Order
	jsonData, err := os.ReadFile("../../cmd/helpCMD/model.json") // Твой model.json в корне
	if err != nil {
		slog.Info("failed to read testData json", "error", err)
		return
	}

	if err := json.Unmarshal(jsonData, &testOrder); err != nil {
		fmt.Println("failed to unmarshal testData json", "error", err)
		return
	}

	testOrderUID := testOrder.OrderUID

	// Определяем колонки, которые возвращает наш SQL-запрос.
	cols := []string{
		"order_uid", "track_number", "entry", "locale", "internal_signature", "customer_id",
		"delivery_service", "shardkey", "sm_id", "date_created", "oof_shard",
		"name", "phone", "zip", "city", "address", "region", "email",
		"request_id", "currency", "provider", "amount", "payment_dt", "bank",
		"delivery_cost", "goods_total", "custom_fee",
		"rid", "chrt_id", "item_track_number", "price", "item_name",
		"sale", "size", "total_price", "nm_id", "brand", "status",
	}

	// --- Тестовые сценарии ---
	testCases := []struct {
		name          string
		orderUID      string
		mockSetup     func()
		expectedOrder entity.Order
		expectedErr   error
	}{
		{
			name:     "Success: TestOrder with one item found",
			orderUID: testOrderUID,
			mockSetup: func() {
				// Так как в заказе один товар, JOIN вернет одну строку.
				rows := pgxmock.NewRows(cols).
					AddRow(
						testOrder.OrderUID, testOrder.TrackNumber, testOrder.Entry, testOrder.Locale, testOrder.InternalSignature, testOrder.CustomerID,
						testOrder.DeliveryService, testOrder.ShardKey, testOrder.SmID, testOrder.DateCreated, testOrder.OofShard,
						testOrder.Delivery.Name, testOrder.Delivery.Phone, testOrder.Delivery.Zip, testOrder.Delivery.City, testOrder.Delivery.Address, testOrder.Delivery.Region, testOrder.Delivery.Email,
						testOrder.Payment.RequestID, testOrder.Payment.Currency, testOrder.Payment.Provider, testOrder.Payment.Amount, testOrder.Payment.PaymentDt, testOrder.Payment.Bank,
						testOrder.Payment.DeliveryCost, testOrder.Payment.GoodsTotal, testOrder.Payment.CustomFee,
						testOrder.Items[0].Rid, testOrder.Items[0].ChrtID, testOrder.Items[0].TrackNumber, testOrder.Items[0].Price, testOrder.Items[0].Name,
						testOrder.Items[0].Sale, testOrder.Items[0].Size, testOrder.Items[0].TotalPrice, testOrder.Items[0].NmID, testOrder.Items[0].Brand, testOrder.Items[0].Status,
					)

				mock.ExpectQuery(`SELECT .* FROM orders o .* WHERE o.order_uid = \$1`).
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
				mock.ExpectQuery(`SELECT .* FROM orders o .* WHERE o.order_uid = \$1`).
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
				mock.ExpectQuery(`SELECT .* FROM orders o .* WHERE o.order_uid = \$1`).
					WithArgs(testOrderUID).
					WillReturnError(fmt.Errorf("something went wrong"))
			},
			expectedOrder: entity.Order{},
			expectedErr:   fmt.Errorf("failed to query order: something went wrong"),
		},
	}

	// Запуск тестов
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockSetup()
			order, err := s.GetOrderByUID(context.Background(), tc.orderUID)

			// Проверка результатов
			if tc.expectedErr != nil {
				if err == nil {
					t.Errorf("expected error '%v', but got nil", tc.expectedErr)
					return
				}
				if err.Error() != tc.expectedErr.Error() {
					t.Errorf("expected error '%v', but got '%v'", tc.expectedErr, err)
					return
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(order.Items) == 0 {
				order.Items = nil
			}
			if len(tc.expectedOrder.Items) == 0 {
				tc.expectedOrder.Items = nil
			}

			// Пропускаем сравнение времени, т.к. оно может отличаться на наносекунды в реальном коде,
			// но для мока оно должно быть идентичным. Для надежности все же обнуляем.
			order.DateCreated = time.Time{}
			tc.expectedOrder.DateCreated = time.Time{}

			// Сравниваем структуры заказов
			if !reflect.DeepEqual(order, tc.expectedOrder) {
				// Пытаемся красиво сериализовать в JSON для читаемого вывода в тестах.
				gotJSON, gErr := json.MarshalIndent(order, "", "  ")
				expJSON, eErr := json.MarshalIndent(tc.expectedOrder, "", "  ")

				if gErr != nil || eErr != nil {
					// Если по какой-то причине сериализация не удалась, падаём назад на %+v.
					t.Errorf("expected order %+v, but got %+v (also failed to marshal to json: gotErr=%v, expErr=%v)", tc.expectedOrder, order, gErr, eErr)
				} else {
					t.Errorf("order mismatch:\n\nexpected:\n%s\n\ngot:\n%s\n", string(expJSON), string(gotJSON))
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}

		})
	}
}
