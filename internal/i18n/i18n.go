package i18n

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Locale map[string]string

type Manager struct {
	locales        map[string]Locale
	availableLangs []string
	fallbackLang   string
}

func NewManager(localesDir string, fallbackLang string) (*Manager, error) {
	m := &Manager{
		locales:      make(map[string]Locale),
		fallbackLang: fallbackLang,
	}

	entries, err := os.ReadDir(localesDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read locales directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		lang := strings.TrimSuffix(entry.Name(), ".json")
		path := filepath.Join(localesDir, entry.Name())

		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("cannot read locale file %s: %w", path, err)
		}

		var locale Locale
		if err := json.Unmarshal(data, &locale); err != nil {
			return nil, fmt.Errorf("cannot parse locale file %s: %w", path, err)
		}

		m.locales[lang] = locale
		m.availableLangs = append(m.availableLangs, lang)
	}

	sort.Strings(m.availableLangs)

	if len(m.locales) == 0 {
		return nil, fmt.Errorf("no locale files found in %s", localesDir)
	}

	if fallbackLang == "" {
		m.fallbackLang = "it"
	}

	if !m.HasLanguage(m.fallbackLang) {
		if len(m.availableLangs) > 0 {
			m.fallbackLang = m.availableLangs[0]
		}
	}

	return m, nil
}

func (m *Manager) AvailableLanguages() []string {
	return append([]string(nil), m.availableLangs...)
}

func (m *Manager) FallbackLanguage() string {
	return m.fallbackLang
}

func (m *Manager) HasLanguage(lang string) bool {
	_, ok := m.locales[lang]
	return ok
}

func (m *Manager) DetectLanguage(r *http.Request) string {
	lang := strings.TrimSpace(r.URL.Query().Get("lang"))
	if lang != "" && m.HasLanguage(lang) {
		return lang
	}

	if acceptLang := r.Header.Get("Accept-Language"); acceptLang != "" {
		parts := strings.Split(acceptLang, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			lang := strings.Split(part, ";")[0]
			lang = strings.TrimSpace(lang)

			if lang == "*" {
				continue
			}

			langParts := strings.Split(lang, "-")
			if len(langParts) > 0 {
				baseLang := langParts[0]
				if m.HasLanguage(baseLang) {
					return baseLang
				}
			}

			if m.HasLanguage(lang) {
				return lang
			}
		}
	}

	return m.fallbackLang
}

type Translator struct {
	manager  *Manager
	language string
}

func (m *Manager) GetTranslator(lang string) *Translator {
	if !m.HasLanguage(lang) {
		lang = m.fallbackLang
	}
	return &Translator{
		manager:  m,
		language: lang,
	}
}

func (t *Translator) Language() string {
	return t.language
}

func (t *Translator) Translate(key string) string {
	if locale, ok := t.manager.locales[t.language]; ok {
		if val, ok := locale[key]; ok {
			return val
		}
	}

	if t.language != t.manager.fallbackLang {
		if locale, ok := t.manager.locales[t.manager.fallbackLang]; ok {
			if val, ok := locale[key]; ok {
				return val
			}
		}
	}

	return key
}

func (t *Translator) TranslateF(key string, args ...interface{}) string {
	return fmt.Sprintf(t.Translate(key), args...)
}

func (m *Manager) TemplateFuncs(lang string) map[string]interface{} {
	tr := m.GetTranslator(lang)
	return map[string]interface{}{
		"T":        tr.Translate,
		"TF":       tr.TranslateF,
		"langName": m.LangDisplayName,
	}
}

func (m *Manager) LangDisplayName(lang string) string {
	if locale, ok := m.locales[lang]; ok {
		if name, ok := locale["lang_name"]; ok {
			return name
		}
	}
	if locale, ok := m.locales[m.fallbackLang]; ok {
		if name, ok := locale["lang_name"]; ok {
			return name
		}
	}
	return lang
}
