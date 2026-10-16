package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"worklog/internal/types"
)

// mainProject é o projeto de emprego fixo (vazio no campo Project).
// Todas as regras de meta (8h/dia, 40h/semana, cap de sessão) valem só pra ele.
const mainProject = "meioemensagem"

// detectProject identifica o projeto pelo git remote do diretório atual.
// Ex.: repo thiagogaspar/movidesk.git → "movidesk".
// Sem repo git (ou remote ausente) → meioemensagem (projeto fixo).
func detectProject() string {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return mainProject
	}
	url := strings.TrimSpace(string(out))
	url = strings.TrimSuffix(url, ".git")
	url = strings.TrimSuffix(url, "/")
	base := filepath.Base(url)
	if base == "" || base == "." || base == "/" {
		return mainProject
	}
	// Repos do projeto fixo → normaliza para meioemensagem.
	if base == "meioemensagem" || base == "www.meioemensagem.com.br" || base == "local.meioemensagem.com.br" || base == "theme_meioemensagem" {
		return mainProject
	}
	return base
}

// workRoot retorna ~/Work (raiz dos projetos).
func workRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/home/thiago/Work"
	}
	return filepath.Join(home, "Work")
}

// isMainProject informa se o projeto é o fixo (meioemensagem).
func isMainProject(p string) bool {
	return p == "" || p == mainProject
}

// projectLabel retorna o rótulo legível do projeto (vazio = meioemensagem).
func projectLabel(p string) string {
	if isMainProject(p) {
		return mainProject
	}
	return p
}

// projectEntryTotal soma as horas de entradas de um projeto.
func projectEntryTotal(entries []types.Entry, project string) float64 {
	var t float64
	for _, e := range entries {
		if entryIsProject(e, project) {
			t += float64(e.Minutes) / 60
		}
	}
	return t
}

// entryIsProject verifica se a entrada pertence ao projeto dado.
func entryIsProject(e types.Entry, project string) bool {
	if isMainProject(project) {
		return isMainProject(e.Project)
	}
	return e.Project == project
}

// mainEntries retorna apenas entradas do projeto fixo.
func mainEntries(entries []types.Entry) []types.Entry {
	var r []types.Entry
	for _, e := range entries {
		if isMainProject(e.Project) {
			r = append(r, e)
		}
	}
	return r
}
