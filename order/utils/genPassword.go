package utils

import (
	"crypto/sha512"
	"fmt"
	"github.com/anaskhan96/go-password-encoder"
	"strings"
)

var (
	options = &password.Options{SaltLen: 16, Iterations: 100, KeyLen: 32, HashFunction: sha512.New}
)

func GenMd5(pwd string) string {
	salt, encodedPwd := password.Encode(pwd, options)
	newPassword := fmt.Sprintf("$pbkdf2-sha512$%s$%s", salt, encodedPwd)
	return newPassword

}
func VerifyPassword(genPwd, oldPwd string) bool {
	pwdInfo := strings.Split(genPwd, "$")
	check := password.Verify(oldPwd, pwdInfo[2], pwdInfo[3], options)
	return check
}
