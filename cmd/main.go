package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/danpi/marca_ai_backend/internal/config"
	"github.com/danpi/marca_ai_backend/internal/handlers"
	"github.com/danpi/marca_ai_backend/internal/middleware"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

func main() {
	// Carregar variáveis do .env
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️ Aviso: arquivo .env não encontrado, usando variáveis do sistema")
	}

	r := mux.NewRouter()
	// Configuração CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"}, // frontend
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	// Conectar ao banco PostgreSQL
	config.ConnectDB()

	// Porta do servidor (default: 8080)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Rota raiz
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "🚀 API Marca-Ai rodando com Go e PostgreSQL!")
	}).Methods("GET")

	// Rota de healthcheck
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	}).Methods("GET")

	// Rotas de Usuario (Públicas)
	r.HandleFunc("/cadastro", handlers.RegisterUsuarioHandler).Methods("POST")
	r.HandleFunc("/login", handlers.LoginHandler).Methods("POST")
	r.HandleFunc("/ajogador", handlers.GetArenasJogador).Methods("GET")
	//r.HandleFunc("/cadastrar-arena", handlers.CadastrodeArena).Methods("POST")

	// --- Rotas Protegidas ---
	// O sub-roteador 'authRouter' aplica o middleware de autenticação a todas as suas rotas.
	authRouter := r.PathPrefix("").Subrouter()
	authRouter.Use(middleware.AuthMiddleware)
	//USUARIO
	authRouter.HandleFunc("/Usuario", handlers.GetUserHandler).Methods("GET")
	authRouter.HandleFunc("/editar-perfil", handlers.UpdateUsuarioHandler).Methods("PUT")
	authRouter.HandleFunc("/excluir-conta", handlers.DeleteUsuarioHandler).Methods("DELETE")
	//ARENAS
	authRouter.HandleFunc("/cadastrar-arena", handlers.CadastrodeArena).Methods("POST")
	authRouter.HandleFunc("/excluir-arena", handlers.DeleteArena).Methods("DELETE")
	authRouter.HandleFunc("/editar-arena", handlers.UpdateArena).Methods("PUT")
	authRouter.HandleFunc("/arenas", handlers.GetArenas).Methods("GET")
	//CAMPOS
	authRouter.HandleFunc("/cadastrar-campo", handlers.CadastrodeCampo).Methods("POST")
	authRouter.HandleFunc("/listar-campos", handlers.GetCampos).Methods("GET")
	authRouter.HandleFunc("/editar-campo/{id}", handlers.UpdateCampo).Methods("PUT")
	authRouter.HandleFunc("/excluir-campo/{id}", handlers.DeleteCampo).Methods("DELETE")
	// AGENDAMENTOS
	authRouter.HandleFunc("/cadastrar-agendamento", handlers.AgendarCampo).Methods("POST")
	authRouter.HandleFunc("/agendamentos", handlers.GetAgendamentos).Methods("GET")
	authRouter.HandleFunc("/agendamentos/status", handlers.AtualizarStatusAgendamento).Methods("PUT")
	authRouter.HandleFunc("/editar-agendamento", handlers.EditarAgendamento).Methods("PUT")
	// DASHBOARD
	authRouter.HandleFunc("/dashboard", handlers.GetDashboard).Methods("GET")
	log.Printf("Servidor rodando em http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, c.Handler(r)))
}
