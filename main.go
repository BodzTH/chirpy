package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) metrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(200)
	w.Write([]byte(fmt.Appendf(nil,
		`<html>
  	<body>
    	<h1>Welcome, Chirpy Admin</h1>
    		<p>Chirpy has been visited %d times!</p>
  	</body>
	</html>`,
		cfg.fileserverHits.Load())))
}

func (cfg *apiConfig) reset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type returnError struct {
		Error string `json:"error"`
	}
	errReturn := returnError{
		Error: msg,
	}
	errMsg, _ := json.Marshal(errReturn)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(errMsg)
}

func respondWithJSON(w http.ResponseWriter, code int, payload any) {
	data, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(data)
}

func validateChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}
	if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}
	splittedBody := strings.Split(params.Body, " ")
	badWords := [3]string{"kerfuffle", "sharbert", "fornax"}
	for _, badWord := range badWords {
		for index, word := range splittedBody {
			if badWord == strings.ToLower(string(word)) {
				splittedBody[index] = "****"
			}
		}
	}
	cleanedBody := strings.Join(splittedBody, " ")
	type returnValue struct {
		CleandBody string `json:"cleaned_body"`
	}
	body := returnValue{
		CleandBody: cleanedBody,
	}
	respondWithJSON(w, 200, body)
}

func main() {
	mux := http.NewServeMux()
	s := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	var apiCfg apiConfig
	apiCfg.fileserverHits.Store(0)

	mux.Handle("/app/", http.StripPrefix("/app", apiCfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))

	mux.HandleFunc("GET /admin/metrics", apiCfg.metrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.reset)

	mux.HandleFunc("GET /api/healthz", healthz)
	mux.HandleFunc("POST /api/validate_chirp", validateChirp)

	s.ListenAndServe()
}
