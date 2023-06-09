package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

// UserStatus : defined the status of a Headscale user
type UserStatus uint8

// Defined UserStatus enum values
const (
	UserCreated UserStatus = iota
	UserExists  UserStatus = iota
	UserDeleted UserStatus = iota
	UserUnknown UserStatus = iota
	UserError   UserStatus = iota
)

// UserConfig : struct that defines a Headscale users
type UserConfig struct {
	ID        uint32    `json:"id,string"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

type UserCreationResponse struct {
	User UserConfig `json:"user"`
}

// ListUsers list all exisint users from Headscale controle plane
func (c *Client) ListUsers(ctx context.Context) (users []UserConfig, err error) {
	resp, err := c.get(ctx, "/user", nil)
	defer closeResponseBody(resp)
	if err != nil {
		return users, err
	}
	return checkUsersList(resp)
}

func checkUsersList(response *http.Response) (users []UserConfig, err error) {

	switch response.StatusCode {
	case http.StatusOK:
		respBody, err := io.ReadAll(response.Body)
		if err != nil {
			return []UserConfig{}, err
		}
		err = json.Unmarshal(respBody, &users)
		if err != nil {
			return []UserConfig{}, err
		}
		return users, nil
	}
	return []UserConfig{}, err
}

// GetUser : return a Headscale user and its status
func (c *Client) GetUser(ctx context.Context, name string) (status UserStatus, user UserConfig, err error) {
	resp, err := c.get(ctx, "/user/"+name, nil)
	if err != nil {
		return UserError, UserConfig{}, err
	}
	return checkUserGetStatus(resp)
}

func checkUserGetStatus(response *http.Response) (status UserStatus, user UserConfig, err error) {

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return UserError, UserConfig{}, err
	}
	switch response.StatusCode {
	case http.StatusOK:
		var userCreationResponse UserCreationResponse
		err = json.Unmarshal(body, &userCreationResponse)
		if err != nil {
			return UserError, UserConfig{}, err
		}
		user = userCreationResponse.User
		return UserExists, user, nil
	case http.StatusInternalServerError:
		IsMessageUnauthorized := strings.Contains(string(body), "Unauthorized")
		if IsMessageUnauthorized {
			return UserError, UserConfig{}, ErrUnauthorized
		}
		isMessageUserAlreadyExists := strings.Contains(string(body), "User already exists")
		if isMessageUserAlreadyExists {
			return UserExists, UserConfig{}, nil
		}
	}
	return UserError, UserConfig{}, nil
}

// CreateUser create a new Headscale user and return its status
func (c *Client) CreateUser(ctx context.Context, name string) (status UserStatus, user UserConfig, err error) {

	var requestBody = make(map[string]string)
	requestBody["name"] = name
	resp, err := c.post(ctx, "/user", requestBody)
	defer closeResponseBody(resp)
	if err != nil {
		return UserError, UserConfig{}, err
	}
	return checkUserCreationStatus(resp)
}

func checkUserCreationStatus(response *http.Response) (UserStatus, UserConfig, error) {

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return UserError, UserConfig{}, err
	}
	switch response.StatusCode {
	case http.StatusOK:
		var userCreationResponse UserCreationResponse
		err = json.Unmarshal(body, &userCreationResponse)
		if err != nil {
			return UserError, UserConfig{}, err
		}
		return UserCreated, userCreationResponse.User, nil
	case http.StatusInternalServerError:
		IsMessageUnauthorized := strings.Contains(string(body), "Unauthorized")
		if IsMessageUnauthorized {
			return UserError, UserConfig{}, ErrUnauthorized
		}
		isMessageUserAlreadyExists := strings.Contains(string(body), "User already exists")
		if isMessageUserAlreadyExists {
			return UserExists, UserConfig{}, nil
		}
	}
	return UserError, UserConfig{}, nil
}

// DeleteUser delete a headscale user from the control plance and return deletion status
func (c *Client) DeleteUser(ctx context.Context, name string) (status UserStatus, err error) {
	status = UserUnknown
	resp, err := c.delete(ctx, "/user/"+name)
	defer closeResponseBody(resp)
	if err != nil {
		return UserError, err
	}
	defer closeResponseBody(resp)
	return checkUserDeletionStatus(resp)
}

func checkUserDeletionStatus(response *http.Response) (status UserStatus, err error) {
	switch response.StatusCode {
	case http.StatusOK:
		return UserDeleted, nil
	case http.StatusInternalServerError:
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return UserError, err
		}

		IsMessageUnauthorized := strings.Contains(string(body), "Unauthorized")
		if IsMessageUnauthorized {
			return UserError, ErrUnauthorized
		}

		isMessageUserNotFound := strings.Contains(string(body), "User not found")
		if isMessageUserNotFound {
			return UserUnknown, ErrUserNotFound
		}
	}
	return UserError, nil
}
