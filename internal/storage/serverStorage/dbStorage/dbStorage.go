package dbStorage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger"
	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
	"go.uber.org/zap"
)

type DBStorage struct {
	db *sql.DB
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

func NewDBStorage(db *sql.DB) *DBStorage {
	return &DBStorage{db}
}

func execSetGauge(ctx context.Context, e execer, field string, value float64) error {
	_, err := e.ExecContext(ctx,
		`INSERT INTO metrics (id, mtype, value) VALUES ($1, $2, $3)
         ON CONFLICT (id, mtype) DO UPDATE SET value = EXCLUDED.value`,
		field, models.Gauge, value)
	return err
}

func (dbs *DBStorage) SetGauge(field string, value float64) error {
	return execSetGauge(context.Background(), dbs.db, field, value)
}

func execAddCounter(ctx context.Context, e execer, field string, value int64) error {
	_, err := e.ExecContext(ctx,
		`INSERT INTO metrics (id, mtype, delta) VALUES ($1, $2, $3)
         ON CONFLICT (id, mtype) DO UPDATE SET delta = metrics.delta + EXCLUDED.delta`,
		field, models.Counter, value)
	return err
}

func (dbs *DBStorage) AddCounter(field string, value int64) error {
	return execAddCounter(context.Background(), dbs.db, field, value)
}

func (dbs *DBStorage) GetGauge(field string) (float64, bool, error) {
	var value float64

	row := dbs.db.QueryRowContext(context.Background(), "SELECT value FROM metrics WHERE id = $1 AND mtype = $2", field, models.Gauge)

	err := row.Scan(&value)

	if errors.Is(err, sql.ErrNoRows) {
		return 0.0, false, nil
	}

	if err != nil {
		return 0.0, false, err
	}

	return value, true, nil
}

func (dbs *DBStorage) GetCounter(field string) (int64, bool, error) {
	var value int64

	row := dbs.db.QueryRowContext(context.Background(), "SELECT delta FROM metrics WHERE id = $1 AND mtype = $2", field, models.Counter)

	err := row.Scan(&value)

	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}

	if err != nil {
		return 0, false, err
	}

	return value, true, nil
}

func (dbs *DBStorage) GetAllGauges() (storage.GaugeMap, error) {
	gaugeMap := make(storage.GaugeMap)

	rows, err := dbs.db.QueryContext(context.Background(), "SELECT id, value FROM metrics WHERE mtype = $1", models.Gauge)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var field string
		var value float64

		err = rows.Scan(&field, &value)
		if err != nil {
			return nil, err
		}

		gaugeMap[field] = value
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return gaugeMap, nil
}

func (dbs *DBStorage) GetAllCounters() (storage.CounterMap, error) {
	counterMap := make(storage.CounterMap)

	rows, err := dbs.db.QueryContext(context.Background(), "SELECT id, delta FROM metrics WHERE mtype = $1", models.Counter)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var field string
		var value int64

		err = rows.Scan(&field, &value)
		if err != nil {
			return nil, err
		}

		counterMap[field] = value
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return counterMap, nil
}

func (dbs *DBStorage) SaveMetricsBatch(ctx context.Context, metrics []models.Metrics) error {
	tx, err := dbs.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, metric := range metrics {
		if metric.ID == "" {
			return errors.New("metric id should not be empty")
		}

		switch metric.MType {
		case models.Gauge:
			if metric.Value == nil {
				logger.Log.Error("Value is omitted", zap.String("type", metric.MType))
				return fmt.Errorf("value for type \"gauge\" is required")
			}
			err = execSetGauge(ctx, tx, metric.ID, *metric.Value)
		case models.Counter:
			if metric.Delta == nil {
				logger.Log.Error("Delta is omitted", zap.String("type", metric.MType))
				return fmt.Errorf("delta for type \"counter\" is required")
			}
			err = execAddCounter(ctx, tx, metric.ID, *metric.Delta)
		default:
			return fmt.Errorf("unknown metric type: %s", metric.MType)
		}

		if err != nil {
			logger.Log.Error("Error setting metric", zap.Error(err))
			return fmt.Errorf("error setting metric: %s", metric.ID)
		}
	}

	return tx.Commit()
}
