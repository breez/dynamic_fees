package postgresql

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

type OpeningParams struct {
	MinFeeMsat           uint64          `json:"min_msat,string,omitempty"`
	Proportional         uint32          `json:"proportional,omitempty"`
	MaxIdleTime          uint32          `json:"max_idle_time,omitempty"`
	MaxClientToSelfDelay uint32          `json:"max_client_to_self_delay,omitempty"`
	ExtraData            json.RawMessage `json:"extra_data,omitempty"`
}

type Scripts struct {
	Starlark  string          `json:"starlark"`
	ExtraData json.RawMessage `json:"extra_data,omitempty"`
}
type Params struct {
	OpeningParams map[int64]OpeningParams
	Starlark      string
	ExtraData     json.RawMessage
}

func PgConnect(databaseUrl string) (*pgxpool.Pool, error) {
	pgxPool, err := pgxpool.New(context.Background(), databaseUrl)
	if err != nil {
		return nil, fmt.Errorf("pgxpool.New(%v): %w", databaseUrl, err)
	}
	return pgxPool, nil
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) SetOpeningParams(token string, validity int64, op OpeningParams) error {
	jop, err := json.Marshal(op)
	if err != nil {
		return err
	}
	if string(jop) == "{}" {
		return nil
	}
	_, err = s.pool.Exec(context.Background(),
		"UPDATE new_channel_params SET params=params||$3 WHERE token=$1 AND validity=$2",
		token, validity, string(jop))
	return err
}

func (s *PostgresStore) GetAllParams() (map[string]Params, error) {
	rows, err := s.pool.Query(context.Background(),
		`SELECT token, validity, params FROM new_channel_params ORDER BY token, validity`)
	if err != nil {
		log.Printf("GetFeeParams() error: %v", err)
		return nil, err
	}

	allParams := make(map[string]Params)
	for rows.Next() {
		var validity int64
		var token, params string
		err = rows.Scan(&token, &validity, &params)
		if err != nil {
			return nil, err
		}
		p, exist := allParams[token]
		if !exist {
			p = Params{OpeningParams: make(map[int64]OpeningParams)}
		}
		if validity == 0 {
			script := struct {
				Starlark  string          `json:"starlark"`
				ExtraData json.RawMessage `json:"extra_data,omitempty"`
			}{}
			err := json.Unmarshal([]byte(params), &script)
			if err != nil {
				log.Printf("Failed to unmarshal script '%v': %v", params, err)
				return nil, err
			}
			p.Starlark, p.ExtraData = script.Starlark, script.ExtraData
		} else {
			var openningParams OpeningParams
			err := json.Unmarshal([]byte(params), &openningParams)
			if err != nil {
				log.Printf("Failed to unmarshal fee param '%v': %v", params, err)
				return nil, err
			}
			p.OpeningParams[validity] = openningParams
		}
		allParams[token] = p
	}

	return allParams, nil
}
