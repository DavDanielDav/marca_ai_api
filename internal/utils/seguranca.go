package utils

import "golang.org/x/crypto/bcrypt"

// HashPassword recebe uma senha em texto puro e gera um hash seguro com bcrypt.
// Esse hash deve ser armazenado no banco em vez da senha original.
func HashSenha(Senha string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(Senha), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash compara uma senha digitada com o hash armazenado no banco.
// Retorna true se a senha for válida, false caso contrário.
func CheckSenhaHash(Senha, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(Senha))
	return err == nil
}
