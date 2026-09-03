package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"worklog/internal/holidays"
	"worklog/internal/types"
)

const configDir = ".config/worklog"
const storeFile = "week.json"
const backupDirName = "backups"
const maxBackups = 60

func storePath() (string, error) {
	return joinHome(configDir, storeFile)
}

func archivePath(yearWeek string) (string, error) {
	return joinHome(configDir, "archive", yearWeek+".json")
}

func archiveDir() (string, error) {
	return joinHome(configDir, "archive")
}

func joinHome(parts ...string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	all := append([]string{home}, parts...)
	dir := filepath.Join(all...)
	// mkdir all parents
	if err := os.MkdirAll(filepath.Dir(dir), 0755); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}
	return dir, nil
}

func Load() (*types.Store, error) {
	p, err := storePath()
	if err != nil {
		return nil, err
	}

	s := &types.Store{
		Week: types.Week{
			YearWeek: currentYearWeek(),
			Planned:  holidays.PlannedHoursForWeek(time.Now()),
		},
	}

	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, fmt.Errorf("reading store: %w", err)
	}

	if err := json.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("parsing store: %w", err)
	}

	// Se mudou de semana, arquiva a anterior e começa nova
	if s.Week.YearWeek != currentYearWeek() {
		archiveOld(s)
		active := s.Active // preserva sessão ativa cross-week
		s.Week = types.Week{
			YearWeek: currentYearWeek(),
			Planned:  holidays.PlannedHoursForWeek(time.Now()),
		}
		s.Active = active
	}

	return s, nil
}

func archiveOld(s *types.Store) {
	if len(s.Week.Days) == 0 {
		return // semana vazia, nao arquiva
	}
	ap, err := archivePath(s.Week.YearWeek)
	if err != nil {
		return
	}
	data, _ := json.MarshalIndent(s.Week, "", "  ")
	os.WriteFile(ap, data, 0644)
}

// LoadArchive carrega o arquivo de uma semana anterior.
func LoadArchive(yearWeek string) (*types.Week, error) {
	ap, err := archivePath(yearWeek)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(ap)
	if err != nil {
		return nil, fmt.Errorf("arquivo '%s' nao encontrado", yearWeek)
	}
	var w types.Week
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	return &w, nil
}

// ListArchive lista as semanas arquivadas.
func ListArchive() ([]string, error) {
	ad, err := archiveDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(ad)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var weeks []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			weeks = append(weeks, e.Name()[:len(e.Name())-5])
		}
	}
	return weeks, nil
}

func Save(s *types.Store) error {
	// Backup rotativo antes de sobrescrever (proteção contra perda de dados)
	backupCurrent()

	p, err := storePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling: %w", err)
	}

	if err := os.WriteFile(p, data, 0644); err != nil {
		return fmt.Errorf("writing: %w", err)
	}
	return nil
}

// backupCurrent copia o week.json atual para backups/ com timestamp e poda os antigos.
// Erros são intencionalmente silenciados — backup nunca deve bloquear o save.
func backupCurrent() {
	p, err := storePath()
	if err != nil {
		return
	}

	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return // primeiro save, nada a preservar
		}
		return
	}

	bd, err := backupDir()
	if err != nil {
		return
	}

	name := fmt.Sprintf("week-%s.json", time.Now().Format("20060102-150405"))
	if err := os.WriteFile(filepath.Join(bd, name), data, 0644); err != nil {
		return
	}

	pruneBackups(bd)
}

func backupDir() (string, error) {
	bd, err := joinHome(configDir, backupDirName)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(bd, 0755); err != nil {
		return "", fmt.Errorf("mkdir backups: %w", err)
	}
	return bd, nil
}

// pruneBackups mantém apenas os maxBackups mais recentes, ordenados por nome (timestamp).
func pruneBackups(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			names = append(names, e.Name())
		}
	}

	if len(names) <= maxBackups {
		return
	}

	sort.Strings(names)
	for _, n := range names[:len(names)-maxBackups] {
		os.Remove(filepath.Join(dir, n))
	}
}

// ListBackups lista os backups disponíveis, do mais recente ao mais antigo.
func ListBackups() ([]string, error) {
	bd, err := backupDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(bd)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			names = append(names, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	return names, nil
}

// RestoreBackup copia um backup (nome do arquivo em backups/) sobre o week.json atual.
func RestoreBackup(name string) error {
	bd, err := backupDir()
	if err != nil {
		return err
	}
	src := filepath.Join(bd, name)
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("backup '%s' não encontrado", name)
	}
	// Valida que é JSON válido antes de sobrescrever
	var s types.Store
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("backup '%s' corrompido: %w", name, err)
	}
	p, err := storePath()
	if err != nil {
		return err
	}
	if err := os.WriteFile(p, data, 0644); err != nil {
		return fmt.Errorf("writing: %w", err)
	}
	return nil
}

// SaveArchive salva uma Week no arquivo de archive.
func SaveArchive(w *types.Week) error {
	if w == nil || w.YearWeek == "" {
		return fmt.Errorf("week inválida para archive")
	}
	ap, err := archivePath(w.YearWeek)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling: %w", err)
	}
	return os.WriteFile(ap, data, 0644)
}

// WeekOf retorna o string YYYY-Www para uma data.
func WeekOf(t time.Time) string {
	y, w := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", y, w)
}

func currentYearWeek() string {
	y, w := time.Now().ISOWeek()
	return fmt.Sprintf("%d-W%02d", y, w)
}

func TodayKey() string {
	return time.Now().Format("2006-01-02")
}

// FindOrCreateDay returns the index of today in the week's days.
func FindOrCreateDay(s *types.Store) int {
	today := TodayKey()
	for i, d := range s.Week.Days {
		if d.Date == today {
			return i
		}
	}
	// cria dia
	s.Week.Days = append(s.Week.Days, types.Day{Date: today})
	return len(s.Week.Days) - 1
}

// FindOrCreateDayAt returns the index of the day containing t, creating it if
// needed. Used by stop to land the entry on the day the session started.
func FindOrCreateDayAt(s *types.Store, t time.Time) int {
	key := t.Format("2006-01-02")
	for i, d := range s.Week.Days {
		if d.Date == key {
			return i
		}
	}
	s.Week.Days = append(s.Week.Days, types.Day{Date: key})
	return len(s.Week.Days) - 1
}
