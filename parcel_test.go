package main

import (
	"database/sql"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

var (
	// randSource источник псевдо случайных чисел.
	// Для повышения уникальности в качестве seed
	// используется текущее время в unix формате (в виде числа)
	randSource = rand.NewSource(time.Now().UnixNano())
	// randRange использует randSource для генерации случайных чисел
	randRange = rand.New(randSource)
)

// getTestParcel возвращает тестовую посылку
func getTestParcel() Parcel {
	return Parcel{
		Client:    1000,
		Status:    ParcelStatusRegistered,
		Address:   "test",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", "tracker.db")
	require.NoError(t, err)
	require.NoError(t, db.Ping())

	t.Cleanup(func() { _ = db.Close() })
	return db
}

// TestAddGetDelete проверяет добавление, получение и удаление посылки
func TestAddGetDelete(t *testing.T) {
	// prepare
	db := openTestDB(t)
	store := NewParcelStore(db)
	parcel := getTestParcel()

	// add
	id, err := store.Add(parcel)
	require.NoError(t, err)
	require.NotZero(t, id)

	// get
	got, err := store.Get(id)
	require.NoError(t, err)

	parcel.Number = id
	require.Equal(t, parcel, got)

	// delete
	require.NoError(t, store.Delete(id))

	_, err = store.Get(id)
	require.Error(t, err)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// TestSetAddress проверяет обновление адреса
func TestSetAddress(t *testing.T) {
	// prepare
	db := openTestDB(t)
	store := NewParcelStore(db)

	parcel := getTestParcel()

	// add
	id, err := store.Add(parcel)
	require.NoError(t, err)
	require.NotZero(t, id)

	t.Cleanup(func() {
		_ = store.Delete(id)
	})

	// set address
	newAddress := "new test address"
	require.NoError(t, store.SetAddress(id, newAddress))

	// check
	got, err := store.Get(id)
	require.NoError(t, err)

	parcel.Number = id
	parcel.Address = newAddress
	require.Equal(t, parcel, got)
}

// TestSetStatus проверяет обновление статуса
func TestSetStatus(t *testing.T) {
	// prepare
	db := openTestDB(t)
	store := NewParcelStore(db)

	parcel := getTestParcel()

	// add
	id, err := store.Add(parcel)
	require.NoError(t, err)
	require.NotZero(t, id)

	t.Cleanup(func() {
		// чтобы можно было удалить — вернём статус registered
		_ = store.SetStatus(id, ParcelStatusRegistered)
		_ = store.Delete(id)
	})

	// set status
	require.NoError(t, store.SetStatus(id, ParcelStatusSent))

	// check
	got, err := store.Get(id)
	require.NoError(t, err)

	parcel.Number = id
	parcel.Status = ParcelStatusSent
	require.Equal(t, parcel, got)
}

// TestGetByClient проверяет получение посылок по идентификатору клиента
func TestGetByClient(t *testing.T) {
	// prepare
	db := openTestDB(t)
	store := NewParcelStore(db)

	parcels := []Parcel{
		getTestParcel(),
		getTestParcel(),
		getTestParcel(),
	}
	parcelMap := map[int]Parcel{}

	// задаём всем посылкам один и тот же идентификатор клиента
	client := randRange.Intn(10_000_000)
	for i := range parcels {
		parcels[i].Client = client
	}

	// add
	ids := make([]int, 0, len(parcels))
	for i := 0; i < len(parcels); i++ {
		id, err := store.Add(parcels[i])
		require.NoError(t, err)
		require.NotZero(t, id)

		// обновляем идентификатор добавленной у посылки
		parcels[i].Number = id

		// сохраняем добавленную посылку в map
		parcelMap[id] = parcels[i]
		ids = append(ids, id)
	}

	t.Cleanup(func() {
		for _, id := range ids {
			_ = store.Delete(id)
		}
	})

	// get by client
	storedParcels, err := store.GetByClient(client)
	require.NoError(t, err)
	require.Len(t, storedParcels, len(parcels))

	// check
	for _, p := range storedParcels {
		want, ok := parcelMap[p.Number]
		require.True(t, ok)

		require.Equal(t, want, p)
	}
}
