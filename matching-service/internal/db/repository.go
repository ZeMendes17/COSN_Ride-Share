package db

import (
	"context"
	"database/sql"
	"matching-service/pkg/model"

	_ "github.com/lib/pq"
)

type Repository interface {
	SaveMatch(ctx context.Context, match model.Match) error
	GetMatchesByRequestID(ctx context.Context, requestID string) ([]model.Match, error)
	GetMatchByID(ctx context.Context, matchID string) (model.Match, error)
	GetMatchByOfferIDRequestID(ctx context.Context, offerID, requestID string) (model.Match, error)
	UpdateMatchStatus(ctx context.Context, matchID, status string) error
	ClearPendingMatchesForOffer(ctx context.Context, offerID string) error
	SaveOffer(ctx context.Context, offer model.Offer) error
	GetAllOffers(ctx context.Context) ([]model.Offer, error)

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
	repo := &PostgresRepository{db: db}
	if err := repo.migrate(); err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *PostgresRepository) Close() error { return r.db.Close() }

func (r *PostgresRepository) Ping() error { return r.db.Ping() }

func (r *PostgresRepository) migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS matches (
			match_id TEXT PRIMARY KEY,
			request_id TEXT,
			offer_id TEXT,
			driver_id TEXT,
			passenger_id TEXT,
			pickup_lat REAL,
			pickup_lon REAL,
			est_pickup_time TIMESTAMP,
			ranking_score REAL,
			status TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS offers (
			offer_id TEXT PRIMARY KEY,
			driver_id TEXT,
			driver_name TEXT,
			origin_lat REAL,
			origin_lon REAL,
			dest_lat REAL,
			dest_lon REAL,
			available_seats INTEGER,
			dept_time_min TIMESTAMP,
			dept_time_max TIMESTAMP
		);`,
	}

	for _, q := range queries {
		if _, err := r.db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

func (r *PostgresRepository) SaveMatch(ctx context.Context, m model.Match) error {
	query := `INSERT INTO matches VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := r.db.ExecContext(ctx, query,
		m.MatchID, m.RequestID, m.OfferID, m.DriverID, m.PassengerID,
		m.PickupLocation.Lat, m.PickupLocation.Lon, m.EstimatedPickupTime,
		m.RankingScore, m.Status,
	)
	return err
}

func (r *PostgresRepository) GetMatchesByRequestID(ctx context.Context, requestID string) ([]model.Match, error) {
	query := `SELECT * FROM matches WHERE request_id = $1`
	rows, err := r.db.QueryContext(ctx, query, requestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanMatches(rows)
}

func (r *PostgresRepository) GetMatchByID(ctx context.Context, matchID string) (model.Match, error) {
	query := `SELECT * FROM matches WHERE match_id = $1`
	rows, err := r.db.QueryContext(ctx, query, matchID)
	if err != nil {
		return model.Match{}, err
	}
	defer rows.Close()
	matches, err := r.scanMatches(rows)
	if err != nil {
		return model.Match{}, err
	}
	if len(matches) == 0 {
		return model.Match{}, model.ErrMatchNotFound
	}
	return matches[0], nil
}

func (r *PostgresRepository) UpdateMatchStatus(ctx context.Context, matchID, status string) error {
	query := `UPDATE matches SET status = $1 WHERE match_id = $2`
	_, err := r.db.ExecContext(ctx, query, status, matchID)
	return err
}

func (r *PostgresRepository) ClearPendingMatchesForOffer(ctx context.Context, offerID string) error {
	query := `DELETE FROM matches WHERE offer_id = $1 AND status = 'Created'`
	_, err := r.db.ExecContext(ctx, query, offerID)
	return err
}

func (r *PostgresRepository) SaveOffer(ctx context.Context, o model.Offer) error {
	query := `INSERT INTO offers (offer_id, driver_id, driver_name, origin_lat, origin_lon, dest_lat, dest_lon, available_seats, dept_time_min, dept_time_max) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) 
		ON CONFLICT (offer_id) DO UPDATE SET 
			driver_id = EXCLUDED.driver_id,
			driver_name = EXCLUDED.driver_name,
			origin_lat = EXCLUDED.origin_lat,
			origin_lon = EXCLUDED.origin_lon,
			dest_lat = EXCLUDED.dest_lat,
			dest_lon = EXCLUDED.dest_lon,
			available_seats = EXCLUDED.available_seats,
			dept_time_min = EXCLUDED.dept_time_min,
			dept_time_max = EXCLUDED.dept_time_max`
	_, err := r.db.ExecContext(ctx, query,
		o.OfferID, o.DriverID, o.DriverName,
		o.Origin.Lat, o.Origin.Lon,
		o.Destination.Lat, o.Destination.Lon,
		o.AvailableSeats,
		o.DepartureTimeMin, o.DepartureTimeMax,
	)
	return err
}

func (r *PostgresRepository) GetAllOffers(ctx context.Context) ([]model.Offer, error) {
	query := `SELECT * FROM offers`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var offers []model.Offer
	for rows.Next() {
		var o model.Offer
		err := rows.Scan(
			&o.OfferID, &o.DriverID, &o.DriverName,
			&o.Origin.Lat, &o.Origin.Lon,
			&o.Destination.Lat, &o.Destination.Lon,
			&o.AvailableSeats,
			&o.DepartureTimeMin, &o.DepartureTimeMax,
		)
		if err != nil {
			return nil, err
		}
		offers = append(offers, o)
	}
	return offers, nil
}

func (r *PostgresRepository) GetMatchByOfferIDRequestID(ctx context.Context, offerID, requestID string) (model.Match, error) {
	query := `SELECT * FROM matches WHERE offer_id = $1 AND request_id = $2`
	rows, err := r.db.QueryContext(ctx, query, offerID, requestID)
	if err != nil {
		return model.Match{}, err
	}
	defer rows.Close()
	matches, err := r.scanMatches(rows)
	if err != nil {
		return model.Match{}, err
	}
	if len(matches) == 0 {
		return model.Match{}, model.ErrMatchNotFound
	}
	return matches[0], nil
}

// Helpers
func (r *PostgresRepository) scanMatches(rows *sql.Rows) ([]model.Match, error) {
	var matches []model.Match
	for rows.Next() {
		var m model.Match
		err := rows.Scan(
			&m.MatchID, &m.RequestID, &m.OfferID, &m.DriverID, &m.PassengerID,
			&m.PickupLocation.Lat, &m.PickupLocation.Lon, &m.EstimatedPickupTime,
			&m.RankingScore, &m.Status,
		)
		if err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, nil
}
