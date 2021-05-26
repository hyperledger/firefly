package e2e

import (
	"github.com/go-resty/resty/v2"
	"github.com/kaleido-io/firefly/pkg/fftypes"
)

var (
	urlGetNamespaces     = "/namespaces"
	urlBroadcastDatatype = "/namespaces/default/broadcast/datatype"
	urlGetData           = "/namespaces/default/data"
)

func GetNamespaces(client *resty.Client) (*resty.Response, error) {
	return client.R().
		SetResult(&[]fftypes.Namespace{}).
		Get(urlGetNamespaces)
}

func BroadcastDatatype(client *resty.Client, name string) (*resty.Response, error) {
	return client.R().
		SetBody(fftypes.Datatype{
			Name:    name,
			Version: "1",
			Value: fftypes.JSONObject{
				"property1": "string",
			},
		}).
		Post(urlBroadcastDatatype)
}

func GetData(client *resty.Client) (*resty.Response, error) {
	return client.R().
		SetResult(&[]fftypes.Data{}).
		Get(urlGetData)
}
