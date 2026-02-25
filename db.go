package main

import (
	"encoding/json"
	"fmt"
	"go.etcd.io/bbolt"
	"time"
)

var (
	bucketSessions = []byte("Sessions")
	bucketVault    = []byte("Vault")
	bucketSkills   = []byte("Skills")
)

type DB struct {
	conn *bbolt.DB
}

func InitDB(path string) (*DB, error) {
	db, err := bbolt.Open(path, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucketSessions)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(bucketVault)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(bucketSkills)
		return err
	})

	if err != nil {
		return nil, err
	}

	return &DB{conn: db}, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

// Session Operations
func (db *DB) SaveSession(s Session) error {
	return db.conn.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketSessions)
		data, err := json.Marshal(s)
		if err != nil {
			return err
		}
		return b.Put([]byte(s.ID), data)
	})
}

func (db *DB) GetSession(id string) (Session, error) {
	var s Session
	err := db.conn.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketSessions)
		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("session not found")
		}
		return json.Unmarshal(data, &s)
	})
	return s, err
}

func (db *DB) DeleteSession(id string) error {
	return db.conn.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketSessions)
		return b.Delete([]byte(id))
	})
}

func (db *DB) ListSessions() ([]Session, error) {
	var sessions []Session
	err := db.conn.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketSessions)
		return b.ForEach(func(k, v []byte) error {
			var s Session
			if err := json.Unmarshal(v, &s); err != nil {
				return err
			}
			sessions = append(sessions, s)
			return nil
		})
	})
	return sessions, err
}

// Vault Operations
func (db *DB) SaveSecret(key, value string) error {
	return db.conn.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketVault)
		return b.Put([]byte(key), []byte(value))
	})
}

func (db *DB) GetSecret(key string) (string, error) {
	var val string
	err := db.conn.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketVault)
		data := b.Get([]byte(key))
		if data == nil {
			return fmt.Errorf("secret not found")
		}
		val = string(data)
		return nil
	})
	return val, err
}

func (db *DB) ListSecrets() (map[string]string, error) {
	secrets := make(map[string]string)
	err := db.conn.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketVault)
		return b.ForEach(func(k, v []byte) error {
			secrets[string(k)] = string(v)
			return nil
		})
	})
	return secrets, err
}

func (db *DB) DeleteSecret(key string) error {
	return db.conn.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketVault)
		return b.Delete([]byte(key))
	})
}

// Skill Operations
func (db *DB) SaveSkill(name, docs string) error {
	return db.conn.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketSkills)
		return b.Put([]byte(name), []byte(docs))
	})
}

func (db *DB) GetSkill(name string) (string, error) {
	var docs string
	err := db.conn.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketSkills)
		data := b.Get([]byte(name))
		if data == nil {
			return fmt.Errorf("skill not found")
		}
		docs = string(data)
		return nil
	})
	return docs, err
}

func (db *DB) ListSkills() ([]string, error) {
	var skills []string
	err := db.conn.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketSkills)
		return b.ForEach(func(k, v []byte) error {
			skills = append(skills, string(k))
			return nil
		})
	})
	return skills, err
}

func (db *DB) DeleteSkill(name string) error {
	return db.conn.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketSkills)
		return b.Delete([]byte(name))
	})
}
