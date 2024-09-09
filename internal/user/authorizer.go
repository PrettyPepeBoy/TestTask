package user

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/spf13/viper"
	"io"
	"os"
	"testTask/internal/cast"
)

var (
	ErrInvalidToken = errors.New("invalid token")
)

type userInfo struct {
	User  string `json:"user"`
	Token string `json:"token"`
}

type Authorizer struct {
	users            []userInfo
	userDataFileName string
}

func NewAuthorizer() (*Authorizer, error) {
	auth := &Authorizer{
		users:            make([]userInfo, 0),
		userDataFileName: viper.GetString("authorize.file-location"),
	}

	return auth, auth.readUsersData()
}

func (auth *Authorizer) readUsersData() error {
	file, err := os.OpenFile(auth.userDataFileName, os.O_RDONLY, 0777)
	if err != nil {
		return err
	}

	rawResp, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	err = json.Unmarshal(rawResp, &auth.users)
	if err != nil {
		return err
	}

	return nil
}

func (auth *Authorizer) saveUsersData() error {
	rawByte, err := json.Marshal(auth.users)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(auth.userDataFileName, os.O_WRONLY, 0777)
	if err != nil {
		return err
	}

	_, err = file.Write(rawByte)
	if err != nil {
		return err
	}

	return nil
}

func (auth *Authorizer) Verify(token string) (string, error) {
	t, err := hex.DecodeString(token)
	if err != nil {
		return "", err
	}

	for _, u := range auth.users {
		if bytes.Equal(cast.StringToByteArray(u.Token), t) {
			return u.User, nil
		}
	}

	return "", ErrInvalidToken
}
