package storage

import (
	"encoding/json"
)

type KV struct {
	Key   string `json:"key" redis:"key"`
	Value string `json:"value" redis:"value"`
}

func (bbi *KV) MarshalBinary() ([]byte, error) {
	return json.Marshal(bbi)
}

func (bbi *KV) UnmarshalBinary(data []byte) error {
	if err := json.Unmarshal(data, &bbi); err != nil {
		return err
	}

	return nil
}
