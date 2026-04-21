package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/danpi/marca_ai_backend/internal/config"
	"github.com/danpi/marca_ai_backend/internal/handlers"
	"github.com/danpi/marca_ai_backend/internal/middleware"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func defaultAllowedOrigins() []string {
	return []string{
		"http://localhost:*",
		"https://*.onrender.com",
		"http://10.0.50.7:5173",
		"https://*.brs.devtunnels.ms",
		"https://frontend-marcaai.onrender.com",
		"https://www.arenas.marcaai.tec.br",
		"https://arenas.marcaai.tec.br",
	}
}

func normalizeOrigins(origins []string) []string {
	normalized := make([]string, 0, len(origins))
	seen := make(map[string]struct{}, len(origins))

	for _, origin := range origins {
		origin = strings.TrimRight(strings.TrimSpace(origin), "/")
		if origin == "" {
			continue
		}

		if _, exists := seen[origin]; exists {
			continue
		}

		seen[origin] = struct{}{}
		normalized = append(normalized, origin)
	}

	return normalized
}

func getAllowedOrigins() []string {
	origins := append([]string{}, defaultAllowedOrigins()...)
	originsFromEnv := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if originsFromEnv == "" {
		return normalizeOrigins(origins)
	}

	parts := strings.Split(originsFromEnv, ",")
	origins = append(origins, parts...)
	origins = normalizeOrigins(origins)

	if len(origins) == 0 {
		return defaultAllowedOrigins()
	}

	return origins
}

func main() {
	config.LoadEnv()

	r := mux.NewRouter().StrictSlash(true)
	r.Use(middleware.DenySensitivePathsMiddleware)
	allowedOrigins := getAllowedOrigins()
	log.Printf("CORS allowed origins: %v", allowedOrigins)

	c := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	config.ConnectDB()
	config.EnsureEmailCodesTable()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "API Marca-Ai running with Go and PostgreSQL")
	}).Methods("GET")

	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	}).Methods("GET")

	r.HandleFunc("/cadastro", handlers.RegisterUsuarioHandler).Methods("POST")
	r.HandleFunc("/cadastro/confirmar-codigo", handlers.ConfirmSignupCode).Methods("POST")
	r.HandleFunc("/cadastro/reenviar-codigo", handlers.ResendSignupCode).Methods("POST")
	r.HandleFunc("/login", handlers.LoginHandler).Methods("POST")
	r.HandleFunc("/auth/google", handlers.GoogleAuthHandler).Methods("POST")
	r.HandleFunc("/auth/refresh", handlers.RefreshTokenHandler).Methods("POST")
	r.HandleFunc("/refresh-token", handlers.RefreshTokenHandler).Methods("POST")
	r.HandleFunc("/forgot-password/send-code", handlers.SendForgotPasswordCode).Methods("POST")
	r.HandleFunc("/forgot-password/verify-code", handlers.VerifyForgotPasswordCode).Methods("POST")
	r.HandleFunc("/forgot-password/reset-password", handlers.ResetForgotPassword).Methods("POST")
	r.HandleFunc("/arenas", handlers.GetArenasJogador).Methods("GET")
	r.HandleFunc("/arenas/{id}", handlers.GetArenaJogadorPorID).Methods("GET")
	r.HandleFunc("/horarios-disponiveis", handlers.GetHorariosDisponiveisCampo).Methods("GET")
	r.HandleFunc("/integracao/agendamentos", handlers.CriarPedidoAgendamentoJogador).Methods("POST")

	authRouter := r.PathPrefix("").Subrouter()
	authRouter.Use(middleware.AuthMiddleware)
	authRouter.Use(middleware.SingleRequestPerUserMiddleware)
	authRouter.HandleFunc("/Usuario", handlers.GetUserHandler).Methods("GET")
	authRouter.HandleFunc("/editar-perfil", handlers.UpdateUsuarioHandler).Methods("PUT")
	authRouter.HandleFunc("/excluir-conta", handlers.DeleteUsuarioHandler).Methods("DELETE")
	authRouter.HandleFunc("/cadastrar-arena", handlers.CadastrodeArena).Methods("POST")
	authRouter.HandleFunc("/excluir-arena", handlers.DeleteArena).Methods("DELETE")
	authRouter.HandleFunc("/editar-arena", handlers.UpdateArena).Methods("PUT")
	authRouter.HandleFunc("/listararenas", handlers.GetArenas).Methods("GET")
	authRouter.HandleFunc("/cadastrar-campo", handlers.CadastrodeCampo).Methods("POST")
	authRouter.HandleFunc("/listar-campos", handlers.GetCampos).Methods("GET")
	authRouter.HandleFunc("/editar-campo", handlers.UpdateCampo).Methods("PUT")
	authRouter.HandleFunc("/editar-campo/{id}", handlers.UpdateCampo).Methods("PUT")
	authRouter.HandleFunc("/manutencao", handlers.AtualizarManutencaoCampo).Methods("PUT")
	authRouter.HandleFunc("/manutencao/{id}", handlers.AtualizarManutencaoCampo).Methods("PUT")
	authRouter.HandleFunc("/excluir-campo/{id}", handlers.DeleteCampo).Methods("DELETE")
	authRouter.HandleFunc("/cadastrar-agendamento", handlers.AgendarCampo).Methods("POST")
	authRouter.HandleFunc("/agendamentos", handlers.GetAgendamentos).Methods("GET")
	authRouter.HandleFunc("/pedidos", handlers.GetPedidos).Methods("GET")
	authRouter.HandleFunc("/pedidos/{id}/aceitar", handlers.AceitarPedido).Methods("PUT")
	authRouter.HandleFunc("/pedidos/{id}/cancelar", handlers.CancelarPedido).Methods("PUT")
	authRouter.HandleFunc("/agendamentos/status", handlers.AtualizarStatusAgendamento).Methods("PUT")
	authRouter.HandleFunc("/editar-agendamento", handlers.EditarAgendamento).Methods("PUT")
	authRouter.HandleFunc("/agendamentos/{id}/iniciar-cronometro", handlers.IniciarCronometroAgendamento).Methods("POST")
	authRouter.HandleFunc("/agendamentos/{id}/encerrar-cronometro", handlers.EncerrarCronometroAgendamento).Methods("POST")
	authRouter.HandleFunc("/agendamentos/{id}/pagamentos", handlers.GetPagamentosAgendamento).Methods("GET")
	authRouter.HandleFunc("/agendamentos/{id}/pagamentos/parcial", handlers.RegistrarPagamentoParcialAgendamento).Methods("POST")
	authRouter.HandleFunc("/agendamentos/{id}/pagamentos/total", handlers.RegistrarPagamentoTotalAgendamento).Methods("POST")
	authRouter.HandleFunc("/agendamentos/{id}/concluir", handlers.ConcluirAgendamento).Methods("POST")
	authRouter.HandleFunc("/dashboard", handlers.GetDashboard).Methods("GET")

	log.Printf("Server running at http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, c.Handler(r)))
}
