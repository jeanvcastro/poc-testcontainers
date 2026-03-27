package entity

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// JSON é um tipo customizado para lidar com JSONB do Postgres.
// Este é um dos pontos que causa incompatibilidade com SQLite,
// já que SQLite não tem tipo nativo JSONB.
type JSON map[string]interface{}

func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan JSON: value is not []byte")
	}

	return json.Unmarshal(bytes, j)
}
