package main

import (
	"html/template"
	"log"
	"net/http"
	"net/smtp"
	"os"

	"gopkg.in/yaml.v3"
)

var logger = log.New(os.Stdout, "", log.LstdFlags)

// Config Struttura che rispecchia il file config.yaml
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

func main() {
	tmpl = template.Must(template.ParseGlob("templates/*.html"))

	// Leggi il file config.yaml
	yamlFile, err := os.ReadFile("config.yaml")
	if err != nil {
		logger.Printf("Errore nella lettura del file di configurazione: %v\n", err)
		os.Exit(1)
	}

	// Decodifica il contenuto del file YAML nella struct cfg
	err = yaml.Unmarshal(yamlFile, &cfg)
	if err != nil {
		logger.Printf("Errore nel parsing del file YAML: %v\n", err)
		os.Exit(1)
	}

	logger.Printf("Configurazione caricata. SMTP Host: %s, Port: %s\n", cfg.SMTP.Host, cfg.SMTP.Port)

	http.HandleFunc("/", handleForm)
	logger.Println("Server avviato su http://localhost:8080")

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		return
	}
}

func handleForm(w http.ResponseWriter, r *http.Request) {
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
			tmpl.ExecuteTemplate(w, "error.html", map[string]string{"Error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl.ExecuteTemplate(w, "success.html", nil)
		return
	}

	err := tmpl.ExecuteTemplate(w, "form.html", nil)
	if err != nil {
		return
	}
}
