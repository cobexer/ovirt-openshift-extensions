package internal

import (
	"errors"
	"fmt"
	"github.com/ovirt/ovirt-flexdriver/internal"
	"gopkg.in/gcfg.v1"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLoadConf(t *testing.T) {
	api := Api{}
	err := gcfg.ReadFileInto(&api, driverConfig)
	if err != nil {
		t.Fatal(err)
	}
	// sanity check the config is loaded
	if api.Connection.Url == "" {
		t.Fatal("empty connection url")
	}
}

func TestInit(t *testing.T) {
	r, err := InitDriver()
	t.Log(r, err)

}

// TestAuthenticateWithUnexpiredToken makes sure we reuse the auth token
func TestAuthenticateWithUnexpiredToken(t *testing.T) {
	api := prepareApi(tokenHandlerFunc(10000000))
	err := api.authenticate()
	if err != nil {
		t.Fatalf("failed authentication %s", err)
	}
}

func TestFetchToken(t *testing.T) {
	// create test server with handler
	api := prepareApi(tokenHandlerFunc(200))

	err := api.authenticate()

	if err != nil {
		t.Fatalf("failed authentication %s", err)
	}

	if api.Token.ExpireIn != 200 {
		t.Fatalf("token expiration expected: 200, got: %v", api.Token.ExpireIn)
	}

	if api.Token.ExpirationTime.Before(time.Now()) {
		t.Fatalf("token should expire only withing 200 sec, but expires on %s", api.Token.ExpirationTime)
	}
}

func TestAttach(t *testing.T) {
	expectedId := 1234
	api := prepareApi(func(w http.ResponseWriter, request *http.Request) {
		fmt.Fprintf(w, `{ "id": "%v", "bootable": "true", "pass_discard": "true", "interface": "ide", "active":"true"}`, expectedId)
	})

	response, e := api.Attach(
		internal.AttachRequest{"data", "vm_disk_1", "ext4", "rw", ""},
		"host1")

	if e != nil {
		t.Fatal(e)
	}

	if response.Status != internal.Success {
		t.Error(errors.New("response failure"))
	}

	if response.Device != "/dev/disk/by-id/virtio"+string(expectedId) {
		t.Error(errors.New("different device paths"))
	}

	t.Log(response)
}

func prepareApi(handler http.HandlerFunc) Api {
	ts := httptest.NewServer(handler)
	api := getApi(http.DefaultClient)
	api.Connection.Url = ts.URL
	api.Connection.Insecure = true
	driverConfig = "/dev/null"
	return api
}

func tokenHandlerFunc(expireIn int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{ "access_token": "1234567890", "expires_in": "%v", "token_type": "Bearer"}`, expireIn)
	}
}

func getApi(client *http.Client) Api {
	return Api{
		Connection{},
		*client,
		Token{"token_123", 0, "Bearer", time.Now()},
	}
}
