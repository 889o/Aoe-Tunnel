package rule

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	bolt "go.etcd.io/bbolt"

	"aoe-tunnel/pkg/signal"
)

const rulesBucket = "rules"

type Store struct {
	db    *bolt.DB
	mu    sync.RWMutex
	rules map[uint]signal.PortRule
}

func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("rule store: create dir: %w", err)
	}
	path := filepath.Join(dir, "rules.db")
	bdb, err := bolt.Open(path, 0o600, &bolt.Options{Timeout: 3 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("rule store: open bbolt: %w", err)
	}
	if err := bdb.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(rulesBucket))
		return err
	}); err != nil {
		_ = bdb.Close()
		return nil, fmt.Errorf("rule store: create bucket: %w", err)
	}

	s := &Store{db: bdb, rules: make(map[uint]signal.PortRule)}
	if err := s.load(); err != nil {
		_ = bdb.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	return s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(rulesBucket))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			var r signal.PortRule
			if err := json.Unmarshal(v, &r); err != nil {
				return fmt.Errorf("rule store: decode rule %s: %w", k, err)
			}
			s.rules[r.RuleID] = r
			return nil
		})
	})
}

func (s *Store) Apply(r signal.PortRule) (prev *signal.PortRule, changed bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cur, ok := s.rules[r.RuleID]; ok {
		if r.Revision <= cur.Revision {
			return &cur, false, nil
		}
		prev = &cur
	}

	raw, err := json.Marshal(r)
	if err != nil {
		return nil, false, err
	}
	if err := s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte(rulesBucket)).Put([]byte(fmt.Sprintf("%d", r.RuleID)), raw)
	}); err != nil {
		return nil, false, err
	}
	s.rules[r.RuleID] = r
	return prev, true, nil
}

func (s *Store) Remove(id uint) (removed *signal.PortRule, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cur, ok := s.rules[id]
	if !ok {
		return nil, nil
	}
	if err := s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte(rulesBucket)).Delete([]byte(fmt.Sprintf("%d", id)))
	}); err != nil {
		return nil, err
	}
	delete(s.rules, id)
	return &cur, nil
}

func (s *Store) ReplaceAll(rules []signal.PortRule) (added []signal.PortRule, removed []signal.PortRule, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	incoming := make(map[uint]signal.PortRule, len(rules))
	for _, r := range rules {
		incoming[r.RuleID] = r
	}

	for id, cur := range s.rules {
		next, ok := incoming[id]
		if !ok {
			removed = append(removed, cur)
			continue
		}
		// 全量快照以内容为准：出口实例增删可能沿用同一 revision，仅比较版本号会漏掉地址变更。
		if ruleChanged(cur, next) {
			added = append(added, next)
		}
	}
	for id, next := range incoming {
		if _, ok := s.rules[id]; !ok {
			added = append(added, next)
		}
	}

	if err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(rulesBucket))
		for _, id := range removedIDs(removed) {
			if err := b.Delete([]byte(fmt.Sprintf("%d", id))); err != nil {
				return err
			}
		}
		for _, r := range added {
			raw, err := json.Marshal(r)
			if err != nil {
				return err
			}
			if err := b.Put([]byte(fmt.Sprintf("%d", r.RuleID)), raw); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, nil, err
	}

	for _, id := range removedIDs(removed) {
		delete(s.rules, id)
	}
	for _, r := range added {
		s.rules[r.RuleID] = r
	}
	return added, removed, nil
}

// ruleChanged 以序列化结果判断内容是否变化，避免 revision 未递增时漏应用。
func ruleChanged(cur, next signal.PortRule) bool {
	a, errA := json.Marshal(cur)
	b, errB := json.Marshal(next)
	if errA != nil || errB != nil {
		return true
	}
	return !bytes.Equal(a, b)
}

func removedIDs(rules []signal.PortRule) []uint {
	ids := make([]uint, 0, len(rules))
	for _, r := range rules {
		ids = append(ids, r.RuleID)
	}
	return ids
}

func (s *Store) Get(id uint) (signal.PortRule, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.rules[id]
	return r, ok
}

func (s *Store) Snapshot() []signal.PortRule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]signal.PortRule, 0, len(s.rules))
	for _, r := range s.rules {
		out = append(out, r)
	}
	return out
}

func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.rules)
}

func (s *Store) Close() error {
	return s.db.Close()
}
