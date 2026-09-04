package db_storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/helpers/retry"
	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
)

type DBStorage struct {
	db *pgxpool.Pool
}

type execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func NewDBStorage(db *pgxpool.Pool) *DBStorage {
	return &DBStorage{db}
}

func execSetGauge(ctx context.Context, e execer, field string, value float64) error {
	_, err := e.Exec(ctx,
		`INSERT INTO metrics (id, mtype, value) VALUES ($1, $2, $3)
         ON CONFLICT (id, mtype) DO UPDATE SET value = EXCLUDED.value`,
		field, models.Gauge, value)
	return err
}

func (dbs *DBStorage) SetGauge(field string, value float64) error {
	return retry.Do(context.TODO(), retry.IsPGRetriable, func() error {
		return execSetGauge(context.TODO(), dbs.db, field, value)
	})
}

func execAddCounter(ctx context.Context, e execer, field string, value int64) error {
	_, err := e.Exec(ctx,
		`INSERT INTO metrics (id, mtype, delta) VALUES ($1, $2, $3)
         ON CONFLICT (id, mtype) DO UPDATE SET delta = metrics.delta + EXCLUDED.delta`,
		field, models.Counter, value)
	return err
}

func (dbs *DBStorage) AddCounter(field string, value int64) error {
	return retry.Do(context.TODO(), retry.IsPGRetriable, func() error {
		return execAddCounter(context.TODO(), dbs.db, field, value)
	})
}

func (dbs *DBStorage) getGaugeOnce(field string) (float64, bool, error) {
	var value float64

	row := dbs.db.QueryRow(context.TODO(), "SELECT value FROM metrics WHERE id = $1 AND mtype = $2", field, models.Gauge)

	err := row.Scan(&value)

	if errors.Is(err, pgx.ErrNoRows) {
		return 0.0, false, nil
	}

	if err != nil {
		return 0.0, false, err
	}

	return value, true, nil
}

func (dbs *DBStorage) GetGauge(field string) (float64, bool, error) {
	var value float64
	var found bool
	err := retry.Do(context.TODO(), retry.IsPGRetriable, func() error {
		v, ok, e := dbs.getGaugeOnce(field)
		value, found = v, ok
		return e
	})
	return value, found, err
}

func (dbs *DBStorage) getCounterOnce(field string) (int64, bool, error) {
	var value int64

	row := dbs.db.QueryRow(context.TODO(), "SELECT delta FROM metrics WHERE id = $1 AND mtype = $2", field, models.Counter)

	err := row.Scan(&value)

	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}

	if err != nil {
		return 0, false, err
	}

	return value, true, nil
}

func (dbs *DBStorage) GetCounter(field string) (int64, bool, error) {
	var value int64
	var found bool
	err := retry.Do(context.TODO(), retry.IsPGRetriable, func() error {
		v, ok, e := dbs.getCounterOnce(field)
		value, found = v, ok
		return e
	})
	return value, found, err
}

func (dbs *DBStorage) getAllGaugesOnce() (storage.GaugeMap, error) {
	gaugeMap := make(storage.GaugeMap)

	rows, err := dbs.db.Query(context.TODO(), "SELECT id, value FROM metrics WHERE mtype = $1", models.Gauge)

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

func (dbs *DBStorage) GetAllGauges() (storage.GaugeMap, error) {
	var gaugeMap storage.GaugeMap
	err := retry.Do(context.TODO(), retry.IsPGRetriable, func() error {
		gm, e := dbs.getAllGaugesOnce()
		gaugeMap = gm
		return e
	})
	return gaugeMap, err
}

func (dbs *DBStorage) getAllCountersOnce() (storage.CounterMap, error) {
	counterMap := make(storage.CounterMap)

	rows, err := dbs.db.Query(context.TODO(), "SELECT id, delta FROM metrics WHERE mtype = $1", models.Counter)

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

func (dbs *DBStorage) GetAllCounters() (storage.CounterMap, error) {
	var counterMap storage.CounterMap
	err := retry.Do(context.TODO(), retry.IsPGRetriable, func() error {
		cm, e := dbs.getAllCountersOnce()
		counterMap = cm
		return e
	})
	return counterMap, err
}

func (dbs *DBStorage) saveBatchOnce(ctx context.Context, metrics []models.Metrics) error {
	tx, err := dbs.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, metric := range metrics {
		if metric.ID == "" {
			return errors.New("metric id should not be empty")
		}

		switch metric.MType {
		case models.Gauge:
			if metric.Value == nil {
				return fmt.Errorf("value for type \"gauge\" is required")
			}
			err = execSetGauge(ctx, tx, metric.ID, *metric.Value)
		case models.Counter:
			if metric.Delta == nil {
				return fmt.Errorf("delta for type \"counter\" is required")
			}
			err = execAddCounter(ctx, tx, metric.ID, *metric.Delta)
		default:
			return fmt.Errorf("unknown metric type: %s", metric.MType)
		}

		if err != nil {
			return fmt.Errorf("error setting metric: %s : %w", metric.ID, err)
		}
	}

	return tx.Commit(ctx)
}

func (dbs *DBStorage) SaveMetricsBatch(ctx context.Context, metrics []models.Metrics) error {
	return retry.Do(ctx, retry.IsPGRetriable, func() error {
		return dbs.saveBatchOnce(ctx, metrics)
	})
}
