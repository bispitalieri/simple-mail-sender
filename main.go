package main

import (
	"fmt"
	"html/template"
	"net/http"
	"net/smtp"
	"os"

	"gopkg.in/yaml.v3"
)

// Config Struttura che rispecchia il file config.yaml
type Config struct {
	SMTP struct {
		Host string `yaml:"host"`
		Port string `yaml:"port"`
	} `yaml:"smtp"`
}

var cfg Config

func main() {
	// Leggi il file config.yaml
	yamlFile, err := os.ReadFile("config.yaml")
	if err != nil {
		fmt.Printf("Errore nella lettura del file di configurazione: %v\n", err)
		os.Exit(1)
	}

	// Decodifica il contenuto del file YAML nella struct cfg
	err = yaml.Unmarshal(yamlFile, &cfg)
	if err != nil {
		fmt.Printf("Errore nel parsing del file YAML: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Configurazione caricata. SMTP Host: %s, Port: %s\n", cfg.SMTP.Host, cfg.SMTP.Port)

	http.HandleFunc("/", handleForm)
	fmt.Println("Server avviato su http://localhost:8080")
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

		// Usa i dati letti dal file YAML
		err := smtp.SendMail(cfg.SMTP.Host+":"+cfg.SMTP.Port, nil, "sender@test.local", []string{to}, msg)
		if err != nil {
			http.Error(w, "Errore invio: "+err.Error(), http.StatusInternalServerError)
			return
		}
		_, err = fmt.Fprintf(w, "<h1>Email inviata con successo!</h1><a href='/'>Invia un'altra</a>")
		if err != nil {
			return
		}
		return
	}

	tmpl := `
	<!DOCTYPE html>
	<html>
	<head><title>Mini Email Sender</title><style>body{font-family:sans-serif;max-width:400px;margin:40px auto;padding:20px;border:1px solid #ccc;border-radius:5px;}input,textarea{width:100%;margin-bottom:10px;padding:8px;box-sizing:border-box;}button{width:100%;padding:10px;background:#007bff;color:white;border:none;border-radius:3px;cursor:pointer;}</style></head>
	<body>
		<h2>Invia Email a Mailpit (via YAML config)</h2>
		<form method="POST">
			<input type="email" name="to" placeholder="A (Destinatario)" required>
			<input type="text" name="subject" placeholder="Oggetto" required>
			<textarea name="body" rows="5" placeholder="Contenuto del messaggio" required></textarea>
			<button type="submit">Invia Email</button>
		</form>
	</body>
	</html>`
	t, _ := template.New("web").Parse(tmpl)
	err := t.Execute(w, nil)
	if err != nil {
		return
	}
}
