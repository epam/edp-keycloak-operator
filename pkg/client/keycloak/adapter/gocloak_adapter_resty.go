package adapter

import (
	"github.com/go-resty/resty/v2"
)

func (a GoCloakAdapter) startRestyRequest() *resty.Request {
	return a.client.RestyClient().R().
		SetAuthToken(a.token.AccessToken).
		SetHeader("Content-Type", "application/json")
}
