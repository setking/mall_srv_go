package main

import (
	"crypto/sha512"
	"fmt"
	"strings"

	"github.com/anaskhan96/go-password-encoder"
)

// Md5加密方法
//
//	func GenMD5(code string) string {
//		MD5 := md5.New()
//		_, err := io.WriteString(MD5, code)
//		if err != nil {
//			return ""
//		}
//		return hex.EncodeToString(MD5.Sum(nil))
//	}
func GenMD5() {
	// Using the default options
	//salt, encodedPwd := password.Encode("generic password", nil)
	//check := password.Verify("generic password", salt, encodedPwd, nil)
	//fmt.Println(check) // true

	// Using custom options
	options := &password.Options{SaltLen: 16, Iterations: 100, KeyLen: 32, HashFunction: sha512.New}
	salt, encodedPwd := password.Encode("generic password", options)
	newPassword := fmt.Sprintf("$pbkdf2-sha512$%s$%s", salt, encodedPwd)

	pwdInfo := strings.Split(newPassword, "$")
	fmt.Println(pwdInfo)
	check := password.Verify("generic password", pwdInfo[2], pwdInfo[3], options)
	fmt.Println(check) // true
}
