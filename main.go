package main

import (
	"html/template"
	"log"
	"net/http"
	"net/smtp"
	"os"

	"simple-mail-server/internal/i18n"

	"gopkg.in/yaml.v3"
)

var logger = log.New(os.Stdout, "", log.LstdFlags)

type Config struct {
	SMTP struct {
		Host     string `yaml:"host"`
		Port     string `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		From     string `yaml:"from"`
	} `yaml:"smtp"`
}

var cfg Config
var tmpl *template.Template
var i18nMgr *i18n.Manager

func main() {
	mgr, err := i18n.NewManager("locales", "it")
	if err != nil {
		logger.Printf("Errore nel caricamento dei file di lingua: %v", err)
		os.Exit(1)
	}
	i18nMgr = mgr
	logger.Printf("Lingue disponibili: %v", i18nMgr.AvailableLanguages())

	yamlFile, err := os.ReadFile("config.yaml")
	if err != nil {
		logger.Println(i18nMgr.GetTranslator("it").TranslateF("config_load_error", err))
		os.Exit(1)
	}

	err = yaml.Unmarshal(yamlFile, &cfg)
	if err != nil {
		logger.Println(i18nMgr.GetTranslator("it").TranslateF("config_parse_error", err))
		os.Exit(1)
	}

	logger.Println(i18nMgr.GetTranslator("it").TranslateF("config_loaded", cfg.SMTP.Host, cfg.SMTP.Port))

	funcMap := template.FuncMap{
		"T":        func(key string) string { return key },
		"TF":       func(key string, args ...interface{}) string { return key },
		"langName": func(lang string) string { return lang },
	}

	tmpl = template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/*.html"))

	http.HandleFunc("/", handleForm)
	logger.Println(i18nMgr.GetTranslator("it").Translate("server_started"))

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		return
	}
}

func handleForm(w http.ResponseWriter, r *http.Request) {
	lang := i18nMgr.DetectLanguage(r)

	t, err := tmpl.Clone()
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	t.Funcs(i18nMgr.TemplateFuncs(lang))

	if r.Method == http.MethodPost {
		to := r.FormValue("to")
		subject := r.FormValue("subject")
		body := r.FormValue("body")

		msg := []byte("To: " + to + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"\r\n" +
			body + "\r\n")

		addr := cfg.SMTP.Host + ":" + cfg.SMTP.Port
		from := cfg.SMTP.From
		if from == "" {
			from = "sender@test.local"
		}

		var auth smtp.Auth
		if cfg.SMTP.Username != "" && cfg.SMTP.Password != "" {
			auth = smtp.PlainAuth("", cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.Host)
		}

		err := smtp.SendMail(addr, auth, from, []string{to}, msg)
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			t.ExecuteTemplate(w, "error.html", map[string]interface{}{
				"Error":          err.Error(),
				"Lang":           lang,
				"AvailableLangs": i18nMgr.AvailableLanguages(),
			})
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		t.ExecuteTemplate(w, "success.html", map[string]interface{}{
			"Lang":           lang,
			"AvailableLangs": i18nMgr.AvailableLanguages(),
		})
		return
	}

	err = t.ExecuteTemplate(w, "form.html", map[string]interface{}{
		"Lang":           lang,
		"AvailableLangs": i18nMgr.AvailableLanguages(),
	})
	if err != nil {
		return
	}
}
