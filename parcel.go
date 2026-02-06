package main

import (
	"database/sql"
)

type ParcelStore struct {
	db *sql.DB
}

func NewParcelStore(db *sql.DB) ParcelStore {
	return ParcelStore{db: db}
}

func (s ParcelStore) Add(p Parcel) (int, error) {
	res, err := s.db.Exec(
		`INSERT INTO parcel (client, status, address, created_at)
		 VALUES (?, ?, ?, ?)`,
		p.Client, p.Status, p.Address, p.CreatedAt,
	)
	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (s ParcelStore) Get(number int) (Parcel, error) {
	row := s.db.QueryRow(
		`SELECT number, client, status, address, created_at
		 FROM parcel
		 WHERE number = ?`,
		number,
	)

	p := Parcel{}
	if err := row.Scan(&p.Number, &p.Client, &p.Status, &p.Address, &p.CreatedAt); err != nil {
		return Parcel{}, err // sql.ErrNoRows если не найдено
	}

	return p, nil
}

func (s ParcelStore) GetByClient(client int) ([]Parcel, error) {
	rows, err := s.db.Query(
		`SELECT number, client, status, address, created_at
		 FROM parcel
		 WHERE client = ?
		 ORDER BY number`,
		client,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []Parcel
	for rows.Next() {
		p := Parcel{}
		if err := rows.Scan(&p.Number, &p.Client, &p.Status, &p.Address, &p.CreatedAt); err != nil {
			return nil, err
		}
		res = append(res, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return res, nil
}

func (s ParcelStore) SetStatus(number int, status string) error {
	res, err := s.db.Exec(
		`UPDATE parcel
		 SET status = ?
		 WHERE number = ?`,
		status, number,
	)
	if err != nil {
		return err
	}

	aff, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if aff == 0 {
		return sql.ErrNoRows // посылки с таким номером нет
	}

	return nil
}

func (s ParcelStore) SetAddress(number int, address string) error {
	// менять адрес можно только если статус registered
	res, err := s.db.Exec(
		`UPDATE parcel
		 SET address = ?
		 WHERE number = ? AND status = ?`,
		address, number, ParcelStatusRegistered,
	)
	if err != nil {
		return err
	}

	aff, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if aff == 0 {
		// если посылки нет — ошибка, если статус не registered — просто ничего не делаем (nil)
		exists, err := s.exists(number)
		if err != nil {
			return err
		}
		if !exists {
			return sql.ErrNoRows
		}
		return nil
	}

	return nil
}

func (s ParcelStore) Delete(number int) error {
	// удалять можно только если статус registered
	res, err := s.db.Exec(
		`DELETE FROM parcel
		 WHERE number = ? AND status = ?`,
		number, ParcelStatusRegistered,
	)
	if err != nil {
		return err
	}

	aff, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if aff == 0 {
		// если посылки нет — ошибка, если статус не registered — просто ничего не делаем (nil)
		exists, err := s.exists(number)
		if err != nil {
			return err
		}
		if !exists {
			return sql.ErrNoRows
		}
		return nil
	}

	return nil
}

func (s ParcelStore) exists(number int) (bool, error) {
	row := s.db.QueryRow(`SELECT 1 FROM parcel WHERE number = ?`, number)
	var one int
	err := row.Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
