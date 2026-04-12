package config

import (
	"errors"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func LoadEnv() {
	for _, candidate := range []string{".env", "backend/.env"} {
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			if err := godotenv.Load(candidate); err != nil {
				log.Printf("Aviso: falha ao carregar %s: %v. Usando variaveis do sistema.", candidate, err)
				return
			}

			log.Printf("Arquivo de ambiente carregado de %s", candidate)
			return
		}

		if err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Printf("Aviso: nao foi possivel verificar %s: %v", candidate, err)
		}
	}

	log.Println("Aviso: .env nao encontrado, usando variaveis do sistema")
}
