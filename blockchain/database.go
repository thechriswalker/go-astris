package blockchain

import (
	"database/sql"
	"errors"

	_ "github.com/mattn/go-sqlite3" // Import go-sqlite3 library
)

// ErrBlockMissing should be returned from Storage.Get for missing blocks
var ErrBlockMissing = errors.New("Block Not Found")

// SQLiteStorage is backed by SQLite
type SQLiteStorage struct {
	db *sql.DB
}

//NewSQLiteStorage does what is says on the tin
func NewSQLiteStorage(path string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite3", path)

	if err != nil {
		return nil, err
	}
	// OK lets see if the table exists.
	// the hash should probably be the primary key...
	stmt, err := db.Prepare(`
		CREATE TABLE IF NOT EXISTS blockchain (
			id BLOB  NOT NULL PRIMARY KEY,
			prev_id BLOB NOT NULL,          -- id of the "previous" block
			chain_id BLOB NOT NULL,         -- the id of the chain for this block (e.g. election Id)
			depth INTEGER NOT NULL,         -- how deep into the chain are we. NB this is *not* unique
			epoch_seconds INTEGER NOT NULL, -- unix timestamp in seconds
			nonce INTEGER NOT NULL,         -- proof of work nonce
			payload_hash BLOB NOT NULL,
			payload BLOB NOT NULL
		);
	`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	if _, err = stmt.Exec(); err != nil {
		return nil, err
	}
	return &SQLiteStorage{db: db}, err
}

func scanBlock(row *sql.Row, blk *BlockHeader) (int, error) {
	if blk == nil {
		panic("scanBlock recieved nil *BlockHeader")
	}
	var depth int
	err := row.Scan(
		&(blk.ID),
		&(blk.PrevID),
		&(blk.ChainID),
		&(blk.EpochSeconds),
		&(blk.Nonce),
		&(blk.PayloadHash),
		&depth,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrBlockMissing
	}
	return depth, err
}

// GetHeader fetches a block by it's hash.
// find and populate the block, or error
// validate the hash over this single block
func (s *SQLiteStorage) GetHeader(hash []byte, blk *BlockHeader) (int, error) {
	stmt, err := s.db.Prepare(`
		SELECT id, prev_id, chain_id, epoch_seconds, nonce, payload_hash, depth
		FROM blocks
		WHERE id = ?
	`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()
	return scanBlock(stmt.QueryRow(hash), blk)
}

// GetPayload fetches the block payload by the block id.
func (s *SQLiteStorage) GetPayload(hash []byte) ([]byte, error) {
	stmt, err := s.db.Prepare(`SELECT payload FROM blocks WHERE id = ?`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(hash)
	if err != nil {
		return nil, err
	}
	if !rows.Next() {
		return nil, ErrBlockMissing
	}

	raw := sql.RawBytes{}
	rows.Scan(&raw)
	// now initialise the buffer with the content we just got
	buf := make([]byte, len(raw))
	copy(buf, raw)
	return buf, nil
}
