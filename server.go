package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("sqlite3", "./cotacoes.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS cotacoes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		valor TEXT,
		data TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/cotacao", cotacaoHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func cotacaoHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 200*time.Millisecond)
	defer cancel()

	client := http.Client{
		Timeout: 200 * time.Millisecond,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		log.Println("Erro ao criar requisição:", err)
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Erro ao fazer requisição:", err)
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Erro ao ler resposta:", err)
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	var cotacao map[string]interface{}
	err = json.Unmarshal(body, &cotacao)
	if err != nil {
		log.Println("Erro ao decodificar JSON:", err)
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	valor, ok := cotacao["bid"].(string)
	if !ok {
		log.Println("Valor da cotação não encontrado.")
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("INSERT INTO cotacoes (valor) VALUES (?)", valor)
	if err != nil {
		log.Println("Erro ao inserir no banco:", err)
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	result := map[string]string{"valor": valor}
	jsonResult, err := json.Marshal(result)
	if err != nil {
		log.Println("Erro ao codificar JSON:", err)
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResult)
}
