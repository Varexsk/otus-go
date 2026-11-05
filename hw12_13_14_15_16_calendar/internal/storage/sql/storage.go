package sqlstorage

import "context"

type Storage struct { // TODO
}

func New(dsn string) (*Storage, error) {
	// TODO: Initialize the SQL storage with the provided DSN
	return &Storage{}, nil
}

func (s *Storage) Connect(ctx context.Context) error {
	// TODO
	return nil
}

func (s *Storage) Close(ctx context.Context) error {
	// TODO
	return nil
}
