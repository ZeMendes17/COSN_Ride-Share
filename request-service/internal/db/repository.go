package db

import (
	"context"
	"database/sql"
	"errors"
	"request-service/pkg/model"

	_ "github.com/lib/pq"
)

// Repository defines the contract for data access operations.
type Repository interface {
	Save(ctx context.Context, req model.CarRequest) (model.CarRequest, error)
	GetByID(ctx context.Context, id string) (model.CarRequest, error)
	GetAll(ctx context.Context, statusFilter model.RequestStatus, userID string) ([]model.CarRequest, error)
	GetAllPending(ctx context.Context, userID string) ([]model.CarRequest, error)
	Update(ctx context.Context, req model.CarRequest) (model.CarRequest, error)
	DeleteByID(ctx context.Context, id string) error
	Ping() error
	Close() error
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(dbPath string) (*PostgresRepository, error) {
	db, err := sql.Open("postgres", dbPath)
	if err != nil {
		return nil, err
	}

	// Basic check
	if err := db.Ping(); err != nil {
		return nil, err
	}
	repo := &PostgresRepository{db: db}
	if err := repo.migrate(); err != nil {
		return nil, err
	}

	return repo, nil
}

func (r *PostgresRepository) Close() error {
	return r.db.Close()
}

func (r *PostgresRepository) migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS requests (
		id TEXT PRIMARY KEY,
		passenger_id TEXT,
		origin_lat REAL,
		origin_lon REAL,
		dest_lat REAL,
		dest_lon REAL,
		desired_time TIMESTAMP,
		passengers INTEGER,
		pref_smoking BOOLEAN,
		pref_pets BOOLEAN,
		pref_music BOOLEAN,
		status TEXT,
		created_at TIMESTAMP,
		updated_at TIMESTAMP
	);
	`
	_, err := r.db.Exec(query)
	return err
}

func (r *PostgresRepository) Save(ctx context.Context, req model.CarRequest) (model.CarRequest, error) {
	query := `
		INSERT INTO requests (
			id, passenger_id, 
			origin_lat, origin_lon, 
			dest_lat, dest_lon, 
			desired_time, passengers, 
			pref_smoking, pref_pets, pref_music, 
			status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err := r.db.ExecContext(ctx, query,
		req.ID, req.PassengerID,
		req.Origin.Lat, req.Origin.Lon,
		req.Destination.Lat, req.Destination.Lon,
		req.DesiredTime, req.Passengers,
		req.Preferences.Smoking, req.Preferences.Pets, req.Preferences.Music,
		req.Status, req.CreatedAt, req.UpdatedAt,
	)
	if err != nil {
		return model.CarRequest{}, err
	}
	return req, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (model.CarRequest, error) {
	query := `SELECT * FROM requests WHERE id = $1`
	return r.scanRow(r.db.QueryRowContext(ctx, query, id))
}

func (r *PostgresRepository) scanRow(row *sql.Row) (model.CarRequest, error) {
	var req model.CarRequest
	err := row.Scan(
		&req.ID, &req.PassengerID,
		&req.Origin.Lat, &req.Origin.Lon,
		&req.Destination.Lat, &req.Destination.Lon,
		&req.DesiredTime, &req.Passengers,
		&req.Preferences.Smoking, &req.Preferences.Pets, &req.Preferences.Music,
		&req.Status, &req.CreatedAt, &req.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.CarRequest{}, model.ErrNotFound
		}
		return model.CarRequest{}, err
	}
	return req, nil
}

func (r *PostgresRepository) GetAll(ctx context.Context, statusFilter model.RequestStatus, userID string) ([]model.CarRequest, error) {
	var rows *sql.Rows
	var err error

	if statusFilter != "" && userID != "" {
		query := `SELECT * FROM requests WHERE status = $1 AND passenger_id = $2`
		rows, err = r.db.QueryContext(ctx, query, statusFilter, userID)
	} else if userID != "" {
		query := `SELECT * FROM requests WHERE passenger_id = $1`
		rows, err = r.db.QueryContext(ctx, query, userID)
	} else if statusFilter != "" {
		query := `SELECT * FROM requests WHERE status = $1`
		rows, err = r.db.QueryContext(ctx, query, statusFilter)
	} else {
		query := `SELECT * FROM requests`
		rows, err = r.db.QueryContext(ctx, query)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanRows(rows)
}

func (r *PostgresRepository) GetAllPending(ctx context.Context, userID string) ([]model.CarRequest, error) {
	var rows *sql.Rows
	var err error

	if userID == "" {
		query := `SELECT * FROM requests WHERE status = 'Pending'`
		rows, err = r.db.QueryContext(ctx, query)
	} else {
		query := `SELECT * FROM requests WHERE status = 'Pending' AND passenger_id = $1`
		rows, err = r.db.QueryContext(ctx, query, userID)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanRows(rows)
}

func (r *PostgresRepository) scanRows(rows *sql.Rows) ([]model.CarRequest, error) {
	var results []model.CarRequest
	for rows.Next() {
		var req model.CarRequest
		err := rows.Scan(
			&req.ID, &req.PassengerID,
			&req.Origin.Lat, &req.Origin.Lon,
			&req.Destination.Lat, &req.Destination.Lon,
			&req.DesiredTime, &req.Passengers,
			&req.Preferences.Smoking, &req.Preferences.Pets, &req.Preferences.Music,
			&req.Status, &req.CreatedAt, &req.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, req)
	}
	return results, nil
}

func (r *PostgresRepository) Update(ctx context.Context, req model.CarRequest) (model.CarRequest, error) {
	query := `
		UPDATE requests SET
			origin_lat=$1, origin_lon=$2, 
			dest_lat=$3, dest_lon=$4, 
			desired_time=$5, passengers=$6, 
			pref_smoking=$7, pref_pets=$8, pref_music=$9, 
			status=$10, updated_at=$11
		WHERE id=$12
	`
	result, err := r.db.ExecContext(ctx, query,
		req.Origin.Lat, req.Origin.Lon,
		req.Destination.Lat, req.Destination.Lon,
		req.DesiredTime, req.Passengers,
		req.Preferences.Smoking, req.Preferences.Pets, req.Preferences.Music,
		req.Status, req.UpdatedAt,
		req.ID,
	)
	if err != nil {
		return model.CarRequest{}, err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return model.CarRequest{}, err
	}
	if rows == 0 {
		return model.CarRequest{}, model.ErrNotFound
	}

	return req, nil
}

func (r *PostgresRepository) DeleteByID(ctx context.Context, id string) error {
	query := "DELETE FROM requests WHERE id = $1"
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) Ping() error { return r.db.Ping() }
