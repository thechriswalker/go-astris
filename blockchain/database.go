package blockchain

import (
	"database/sql"
	"errors"
	"sync"

	_ "github.com/mattn/go-sqlite3" // Import go-sqlite3 library
)

// ErrBlockMissing should be returned from Storage.Get for missing blocks
var ErrBlockMissing = errors.New("Block Not Found")

// SQLiteStorage is backed by SQLite
type SQLiteStorage struct {
	db *sql.DB
}

var _ Storage = (*SQLiteStorage)(nil)

//NewSQLiteStorage does what is says on the tin
func NewSQLiteStorage(path string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite3", path)

	if err != nil {
		return nil, err
	}
	// OK lets see if the table exists.
	// the hash should probably be the primary key...
	stmt, err := db.Prepare(`
		CREATE TABLE IF NOT EXISTS blocks (
			id BLOB  NOT NULL PRIMARY KEY,
			prev_id BLOB NOT NULL,          -- id of the "previous" block
			depth INTEGER NOT NULL,         -- how deep into the chain are we
			epoch_seconds INTEGER NOT NULL, -- unix timestamp in seconds
			proof INTEGER NOT NULL,         -- proof of work
			payload_hint INTEGER NOT NULL,  -- type of the payload
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

// will return nil if the chain is empty.
func (s *SQLiteStorage) Head() (BlockID, error) {
	// NOTE the tie breakers for our query. depth/epoch_seconds _could_ be
	// duplicates (DAG remember), but the id is unique. so this will always
	// return 0 rows for an empty db or exactly 1 row for a populated DB
	stmt, err := s.db.Prepare(`
		SELECT id FROM blocks ORDER BY depth DESC, epoch_seconds DESC, id ASC LIMIT 1
	`)
	if err != nil {
		// just panic. This one should not fail.
		return ZeroId, err
	}
	defer stmt.Close()
	row := stmt.QueryRow()
	var b BlockID
	tmp := getB()
	err = row.Scan(&tmp)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// that is fine.
			return ZeroId, nil
		}
		return ZeroId, err
	}
	copy(b[:], tmp)
	putB(tmp)
	return b, nil

}

var idBufPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, len(BlockID{}))
	},
}

func getB() []byte {
	return idBufPool.Get().([]byte)
}
func putB(b []byte) {
	idBufPool.Put(b)
}

func scanBlock(row *sql.Row, blk *BlockHeader) (*BlockHeader, error) {
	if blk == nil {
		panic("scanBlock recieved nil *BlockHeader")
	}

	id, prev, ph := getB(), getB(), getB()

	err := row.Scan(
		&id,
		&prev,
		&(blk.EpochSeconds),
		&(blk.Proof),
		&ph,
		&(blk.PayloadHint),
		&(blk.Depth),
	)
	if errors.Is(err, sql.ErrNoRows) {
		return blk, ErrBlockMissing
	}
	copy(blk.ID[:], id)
	copy(blk.PrevID[:], prev)
	copy(blk.PayloadHash[:], ph)
	putB(id)
	putB(prev)
	putB(ph)

	return blk, err
}

// GetHeader fetches a block by it's hash.
// find and populate the block, or error
// validate the hash over this single block
func (s *SQLiteStorage) Header(hash BlockID, blk *BlockHeader) (*BlockHeader, error) {
	stmt, err := s.db.Prepare(`
		SELECT id, prev_id, epoch_seconds, proof, payload_hash, payload_hint, depth
		FROM blocks
		WHERE id = ?
	`)
	if err != nil {
		return blk, err
	}
	defer stmt.Close()
	return scanBlock(stmt.QueryRow(hash[:]), blk)
}

// GetPayload fetches the block payload by the block id.
func (s *SQLiteStorage) Payload(hash BlockID, b []byte) ([]byte, error) {
	stmt, err := s.db.Prepare(`SELECT payload FROM blocks WHERE id = ?`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(hash[:])
	defer rows.Close()
	if err != nil {
		return nil, err
	}
	if !rows.Next() {
		return nil, ErrBlockMissing
	}

	raw := sql.RawBytes{}
	rows.Scan(&raw)
	// now initialise the buffer with the content we just got
	// which might be bigger or smaller than the buffer we were
	// given
	var buf []byte
	if b == nil {
		buf = make([]byte, len(raw))
		copy(buf, raw)
		//fmt.Println("new buffer:", buf)
	} else if len(raw) <= len(b) {
		// raw will fit in b
		copy(b, raw)
		buf = b[0:len(raw)]
		//fmt.Println("reuse:", buf)
	} else {
		// raw is bigger than b
		// copy as much as we can.
		n := copy(b, raw)
		// append the rest
		buf = append(b, raw[n:]...)
		//fmt.Println("extend:", buf)
	}
	return buf, nil
}

func (s *SQLiteStorage) Write(blk *Block) error {
	// we don't validate, we just write.
	stmt, err := s.db.Prepare(`
		INSERT INTO blocks (id, prev_id, epoch_seconds, proof, payload_hash, payload_hint, depth, payload)
					VALUES (?,  ?,       ?,             ?,     ?,            ?,            ?,     ?);
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(
		blk.Header.ID[:],
		blk.Header.PrevID[:],
		blk.Header.EpochSeconds,
		blk.Header.Proof,
		blk.Header.PayloadHash[:],
		blk.Header.PayloadHint,
		blk.Header.Depth,
		blk.Payload,
	)
	return err
}
